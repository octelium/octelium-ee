// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"slices"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type issuanceAttempt struct {
	createdAt time.Time
	startedAt time.Time
}

func (c *Controller) reconcileCertificate(ctx context.Context, uid string) (err error) {
	lockCtx, unlock, err := c.acquireLock(ctx, "certificate:"+uid)
	if err != nil {
		return err
	}
	defer unlock()

	crt, err := c.prepareCertificate(lockCtx, uid)
	if err != nil || crt == nil {
		return err
	}

	iss, err := c.getReadyIssuer(lockCtx, crt.Status.CertificateIssuerRef)
	if err != nil {
		return err
	}

	domains, err := c.getCertificateDomains(lockCtx, crt)
	if err != nil {
		return err
	}

	acmeC, err := c.newLegoClient(lockCtx, iss)
	if err != nil {
		return err
	}

	provider, err := c.getProvider(lockCtx)
	if err != nil {
		return err
	}

	if err := acmeC.Challenge.SetDNS01Provider(provider); err != nil {
		return err
	}

	crt, attempt, err := c.claimCertificate(lockCtx, crt)
	if err != nil || crt == nil {
		return err
	}

	defer func() {
		if err != nil {
			c.markAttemptFailed(ctx, uid, attempt, err)
		}
	}()

	zap.L().Info("Starting the ACME Certificate issuance",
		zap.String("uid", uid),
		zap.String("name", crt.Metadata.Name),
		zap.Strings("domains", domains))

	rsc, err := acmeC.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Could not obtain the Certificate from the ACME server")
	}

	if err := c.persistIssuedCertificate(lockCtx,
		crt, attempt, domains, rsc.Certificate, rsc.PrivateKey); err != nil {
		return err
	}

	zap.L().Info("Successfully issued the ACME Certificate",
		zap.String("uid", uid),
		zap.String("name", crt.Metadata.Name))

	return nil
}

func (c *Controller) prepareCertificate(ctx context.Context, uid string) (
	*enterprisev1.Certificate, error) {

	crt, err := c.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
		Uid: uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if !isACMECertificate(crt) {
		return nil, nil
	}

	issuance := crt.Status.Issuance
	now := time.Now()

	switch {
	case issuance == nil:
	case issuance.State == enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED:
		return crt, nil
	case issuance.State == enterprisev1.Certificate_Status_Issuance_ISSUING:
		if !isStaleIssuance(issuance, now) {
			return nil, nil
		}
	case issuance.State == enterprisev1.Certificate_Status_Issuance_SUCCESS:
		if !needsRenewal(issuance, now) {
			return nil, nil
		}
	case issuance.State == enterprisev1.Certificate_Status_Issuance_FAILED:
	default:
		return nil, nil
	}

	return c.requestIssuance(ctx, crt)
}

func (c *Controller) requestIssuance(ctx context.Context, crt *enterprisev1.Certificate) (
	*enterprisev1.Certificate, error) {

	uid := crt.Metadata.Uid

	for i := range statusUpdateRetries {
		updated, err := certutils.DoIssueCertificate(ctx, c.octeliumC, crt)
		if err == nil {
			return updated, nil
		}

		if i == statusUpdateRetries-1 || isCanceled(err) {
			return nil, err
		}

		if err := sleepContext(ctx, statusUpdateInterval); err != nil {
			return nil, err
		}

		crt, err = c.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: uid,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}

		if !isACMECertificate(crt) {
			return nil, nil
		}

		if isIssuanceRequested(crt) {
			return crt, nil
		}
	}

	return nil, &deferredError{
		err:   errors.Errorf("Could not request the Certificate issuance: %s", uid),
		delay: minRetryDelay,
	}
}

func (c *Controller) claimCertificate(ctx context.Context, crt *enterprisev1.Certificate) (
	*enterprisev1.Certificate, *issuanceAttempt, error) {

	uid := crt.Metadata.Uid

	for range statusUpdateRetries {
		if !isIssuanceRequested(crt) {
			return nil, nil, nil
		}

		crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_ISSUING
		crt.Status.Issuance.IssuanceStartedAt = pbutils.Now()
		crt.Status.Issuance.IssuanceCompletedAt = nil

		updated, err := c.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt)
		if err == nil {
			return updated, getIssuanceAttempt(updated), nil
		}

		if !grpcerr.IsResourceChanged(err) {
			return nil, nil, err
		}

		if err := sleepContext(ctx, statusUpdateInterval); err != nil {
			return nil, nil, err
		}

		crt, err = c.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: uid,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				return nil, nil, nil
			}
			return nil, nil, err
		}
	}

	return nil, nil, &deferredError{
		err:   errors.Errorf("Could not claim the Certificate issuance: %s", uid),
		delay: minRetryDelay,
	}
}

func (c *Controller) markAttemptFailed(ctx context.Context,
	uid string, attempt *issuanceAttempt, cause error) {

	zap.L().Warn("The ACME Certificate issuance has failed",
		zap.String("uid", uid), zap.Error(cause))

	updateCtx, cancel := c.detachedContext(ctx)
	defer cancel()

	if err := c.updateCertificateStatus(updateCtx, uid, attempt,
		func(crt *enterprisev1.Certificate) {
			crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_FAILED
			crt.Status.Issuance.IssuanceCompletedAt = pbutils.Now()
			crt.Status.FailedIssuances = crt.Status.FailedIssuances + 1
		}); err != nil {
		zap.L().Warn("Could not set the Certificate issuance as failed",
			zap.String("uid", uid), zap.Error(err))
	}
}

func (c *Controller) persistIssuedCertificate(ctx context.Context,
	crt *enterprisev1.Certificate, attempt *issuanceAttempt,
	domains []string, certPEM, privateKeyPEM []byte) error {

	info, x509Crt, err := validateIssuedCertificate(certPEM, privateKeyPEM, domains)
	if err != nil {
		return &permanentError{err: err}
	}

	sec, err := c.upsertCertificateSecret(ctx, crt, certPEM, privateKeyPEM)
	if err != nil {
		return err
	}

	return c.updateCertificateStatus(ctx, crt.Metadata.Uid, attempt,
		func(crt *enterprisev1.Certificate) {
			crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_SUCCESS
			crt.Status.Issuance.IssuanceCompletedAt = pbutils.Now()
			crt.Status.Issuance.ExpiresAt = pbutils.Timestamp(x509Crt.NotAfter)
			crt.Status.SuccessfulIssuances = crt.Status.SuccessfulIssuances + 1
			crt.Status.SecretRef = umetav1.GetObjectReference(sec)
			crt.Status.Info = info
		})
}

func (c *Controller) updateCertificateStatus(ctx context.Context,
	uid string, attempt *issuanceAttempt, fn func(*enterprisev1.Certificate)) error {

	for range statusUpdateRetries {
		crt, err := c.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: uid,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				return nil
			}
			return err
		}

		if !isSameAttempt(crt, attempt) {
			zap.L().Debug("The Certificate issuance attempt was superseded",
				zap.String("uid", uid))
			return nil
		}

		fn(crt)

		if _, err := c.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt); err == nil {
			return nil
		} else if !grpcerr.IsResourceChanged(err) {
			return err
		}

		if err := sleepContext(ctx, statusUpdateInterval); err != nil {
			return err
		}
	}

	return errors.Errorf("Could not update the Certificate status: %s", uid)
}

func (c *Controller) upsertCertificateSecret(ctx context.Context,
	crt *enterprisev1.Certificate, certPEM, privateKeyPEM []byte) (*corev1.Secret, error) {

	name := uenterprisev1.ToCertificate(crt).GetSecretName()

	for range statusUpdateRetries {
		sec, err := c.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: name,
		})
		if err != nil {
			if !grpcerr.IsNotFound(err) {
				return nil, err
			}

			sec = &corev1.Secret{
				Metadata: &metav1.Metadata{
					Name:           name,
					IsSystem:       true,
					IsUserHidden:   true,
					IsSystemHidden: true,
					SystemLabels: map[string]string{
						"octelium-cert": "true",
					},
				},
				Spec:   &corev1.Secret_Spec{},
				Status: &corev1.Secret_Status{},
			}

			ucorev1.ToSecret(sec).SetCertificate(string(certPEM), string(privateKeyPEM))

			sec, err = c.octeliumC.CoreC().CreateSecret(ctx, sec)
			if err == nil {
				return sec, nil
			}
			if !grpcerr.AlreadyExists(err) {
				return nil, err
			}
		} else {
			if sec.Metadata.SystemLabels == nil {
				sec.Metadata.SystemLabels = make(map[string]string)
			}
			sec.Metadata.SystemLabels["octelium-cert"] = "true"
			sec.Metadata.IsSystem = true
			sec.Metadata.IsUserHidden = true
			sec.Metadata.IsSystemHidden = true

			ucorev1.ToSecret(sec).SetCertificate(string(certPEM), string(privateKeyPEM))

			sec, err = c.octeliumC.CoreC().UpdateSecret(ctx, sec)
			if err == nil {
				return sec, nil
			}
			if !grpcerr.IsResourceChanged(err) {
				return nil, err
			}
		}

		if err := sleepContext(ctx, statusUpdateInterval); err != nil {
			return nil, err
		}
	}

	return nil, errors.Errorf("Could not set the Certificate Secret: %s", name)
}

func (c *Controller) getReadyIssuer(ctx context.Context, ref *metav1.ObjectReference) (
	*enterprisev1.CertificateIssuer, error) {

	iss, err := c.octeliumC.EnterpriseC().GetCertificateIssuer(ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, &permanentError{
				err: errors.Errorf("The CertificateIssuer does not exist"),
			}
		}
		return nil, err
	}

	if iss.Spec.GetAcme() == nil {
		return nil, &permanentError{
			err: errors.Errorf("The CertificateIssuer type is not ACME"),
		}
	}

	if iss.Status.State != enterprisev1.CertificateIssuer_Status_READY ||
		iss.Status.GetAcme() == nil || iss.Status.GetAcme().SecretRef == nil {

		c.enqueue(workKey{
			typ: workTypeCertificateIssuer,
			uid: iss.Metadata.Uid,
		}, false)

		return nil, &deferredError{
			err: errors.Errorf("The CertificateIssuer %s is not ready yet",
				iss.Metadata.Name),
			delay: minRetryDelay,
		}
	}

	return iss, nil
}

func (c *Controller) getCertificateDomains(ctx context.Context,
	crt *enterprisev1.Certificate) ([]string, error) {

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	domain := normalizeDomain(cc.Status.Domain)
	if !isValidDNSName(domain, false) {
		return nil, &permanentError{
			err: errors.Errorf("Invalid Cluster domain: %s", cc.Status.Domain),
		}
	}

	var domains []string

	switch {
	case crt.Status.ServiceRef != nil:
		svc, err := c.octeliumC.CoreC().GetService(ctx,
			apivalidation.ObjectReferenceToRGetOptions(crt.Status.ServiceRef))
		if err != nil {
			if grpcerr.IsNotFound(err) {
				return nil, &permanentError{
					err: errors.Errorf("The Certificate Service does not exist"),
				}
			}
			return nil, err
		}

		if svc.Status == nil {
			return nil, &permanentError{
				err: errors.Errorf("The Certificate Service has no status"),
			}
		}

		publicFQDN := joinDomain(svc.Status.PrimaryHostname, domain)
		privateFQDN := joinDomain(svc.Status.PrimaryHostname, "local."+domain)

		domains = []string{publicFQDN, privateFQDN}

		if svc.Status.ManagedService != nil && svc.Status.ManagedService.HasSubdomain {
			domains = append(domains, "*."+publicFQDN, "*."+privateFQDN)
		}

	case crt.Status.NamespaceRef != nil:
		ns := normalizeDomain(crt.Status.NamespaceRef.Name)
		if !isValidDNSLabel(ns) {
			return nil, &permanentError{
				err: errors.Errorf("Invalid Namespace name: %s",
					crt.Status.NamespaceRef.Name),
			}
		}

		domains = []string{
			"*." + joinDomain(ns, domain),
			"*." + joinDomain(ns, "local."+domain),
		}

		if ns == "default" {
			domains = append([]string{domain}, domains...)
			domains = append(domains, "*."+domain, "*.local."+domain)
		}

	default:
		return nil, &permanentError{
			err: errors.Errorf("Could not find the domains of the Certificate: %s",
				crt.Metadata.Name),
		}
	}

	domains = uniqueDomains(domains)

	for _, itm := range domains {
		if !isValidDNSName(itm, true) {
			return nil, &permanentError{
				err: errors.Errorf("Invalid Certificate domain: %s", itm),
			}
		}
	}

	return domains, nil
}

func validateIssuedCertificate(certPEM, privateKeyPEM []byte, domains []string) (
	*enterprisev1.Certificate_Status_Info, *x509.Certificate, error) {

	if _, err := tls.X509KeyPair(certPEM, privateKeyPEM); err != nil {
		return nil, nil, errors.Errorf(
			"The issued Certificate does not match its private key: %+v", err)
	}

	x509Crt, err := utils_cert.ParsePEMCertificate(string(certPEM))
	if err != nil {
		return nil, nil, errors.Errorf("Could not parse the issued Certificate: %+v", err)
	}

	now := time.Now()
	if !x509Crt.NotAfter.After(now) {
		return nil, nil, errors.Errorf("The issued Certificate is already expired")
	}
	if x509Crt.NotBefore.After(now.Add(10 * time.Minute)) {
		return nil, nil, errors.Errorf("The issued Certificate is not valid yet")
	}

	for _, domain := range domains {
		if strings.HasPrefix(domain, "*.") {
			if !slices.ContainsFunc(x509Crt.DNSNames, func(arg string) bool {
				return strings.EqualFold(arg, domain)
			}) {
				return nil, nil, errors.Errorf(
					"The issued Certificate does not cover the domain: %s", domain)
			}
			continue
		}

		if err := x509Crt.VerifyHostname(domain); err != nil {
			return nil, nil, errors.Errorf(
				"The issued Certificate does not cover the domain %s: %+v", domain, err)
		}
	}

	info, err := certutils.GetInfo(string(certPEM), string(privateKeyPEM))
	if err != nil {
		return nil, nil, errors.Errorf("Could not get the issued Certificate info: %+v", err)
	}

	return info, x509Crt, nil
}

func isACMECertificate(crt *enterprisev1.Certificate) bool {
	return crt != nil && crt.Metadata != nil &&
		crt.Spec != nil && crt.Status != nil &&
		crt.Spec.Mode == enterprisev1.Certificate_Spec_MANAGED &&
		crt.Status.CertificateIssuerRef != nil
}

func isIssuanceRequested(crt *enterprisev1.Certificate) bool {
	return crt != nil && crt.Status != nil && crt.Status.Issuance != nil &&
		crt.Status.Issuance.State ==
			enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED
}

func isStaleIssuance(issuance *enterprisev1.Certificate_Status_Issuance, now time.Time) bool {
	return issuance.IssuanceStartedAt == nil ||
		now.After(issuance.IssuanceStartedAt.AsTime().Add(staleIssuanceAfter))
}

func needsRenewal(issuance *enterprisev1.Certificate_Status_Issuance, now time.Time) bool {
	return issuance.ExpiresAt == nil ||
		now.Add(renewBefore).After(issuance.ExpiresAt.AsTime())
}

func needsReconcile(crt *enterprisev1.Certificate, now time.Time) bool {
	if !isACMECertificate(crt) {
		return false
	}

	issuance := crt.Status.Issuance
	if issuance == nil {
		return true
	}

	switch issuance.State {
	case enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
		enterprisev1.Certificate_Status_Issuance_FAILED:
		return true
	case enterprisev1.Certificate_Status_Issuance_ISSUING:
		return isStaleIssuance(issuance, now)
	case enterprisev1.Certificate_Status_Issuance_SUCCESS:
		return needsRenewal(issuance, now)
	default:
		return true
	}
}

func getIssuanceAttempt(crt *enterprisev1.Certificate) *issuanceAttempt {
	return &issuanceAttempt{
		createdAt: crt.Status.Issuance.CreatedAt.AsTime(),
		startedAt: crt.Status.Issuance.IssuanceStartedAt.AsTime(),
	}
}

func isSameAttempt(crt *enterprisev1.Certificate, attempt *issuanceAttempt) bool {
	if attempt == nil || crt == nil || crt.Status == nil ||
		crt.Status.Issuance == nil ||
		crt.Status.Issuance.State != enterprisev1.Certificate_Status_Issuance_ISSUING {
		return false
	}

	return crt.Status.Issuance.CreatedAt.AsTime().Equal(attempt.createdAt) &&
		crt.Status.Issuance.IssuanceStartedAt.AsTime().Equal(attempt.startedAt)
}
