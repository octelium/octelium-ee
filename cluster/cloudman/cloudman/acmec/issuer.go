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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	acmeSecretAccountKey      = "account"
	acmeSecretPrivateKeyKey   = "privateKey"
	acmeSecretDirectoryURLKey = "directoryURL"
)

type acmeAccount struct {
	account       *Account
	privateKeyPEM []byte
	directoryURL  string
}

func (c *Controller) reconcileCertificateIssuer(ctx context.Context, uid string) error {
	lockCtx, unlock, err := c.acquireLock(ctx, "issuer:"+uid)
	if err != nil {
		return err
	}
	defer unlock()

	iss, err := c.octeliumC.EnterpriseC().GetCertificateIssuer(lockCtx, &rmetav1.GetOptions{
		Uid: uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			c.clearIssuerForce(uid)
			return nil
		}
		return err
	}

	if iss.Spec.GetAcme() == nil {
		c.clearIssuerForce(uid)
		return nil
	}

	directoryURL := getCADirURL(iss)
	if err := validateDirectoryURL(directoryURL); err != nil {
		c.setIssuerNotReady(ctx, uid)
		return &permanentError{err: err}
	}

	email, err := c.getIssuerEmail(lockCtx, iss)
	if err != nil {
		return err
	}

	loaded, err := c.loadACMEAccount(lockCtx, iss, email)
	if err != nil {
		c.setIssuerNotReady(ctx, uid)
		return err
	}

	directoryChanged := loaded != nil &&
		loaded.directoryURL != "" && loaded.directoryURL != directoryURL

	isReady := !directoryChanged && loaded != nil &&
		loaded.account.Registration != nil &&
		iss.Status.State == enterprisev1.CertificateIssuer_Status_READY &&
		iss.Status.GetAcme() != nil && iss.Status.GetAcme().SecretRef != nil

	if isReady && !c.hasIssuerForce(uid) {
		return nil
	}

	zap.L().Debug("Starting reconciling the ACME account",
		zap.String("uid", uid), zap.String("name", iss.Metadata.Name))

	if iss, err = c.setIssuerState(lockCtx, uid,
		enterprisev1.CertificateIssuer_Status_PREPARING, nil); err != nil {
		return err
	}

	account, privateKeyPEM := func() (*Account, []byte) {
		if loaded != nil && loaded.account.key != nil {
			if directoryChanged {
				loaded.account.Registration = nil
			}
			return loaded.account, loaded.privateKeyPEM
		}
		return nil, nil
	}()

	if account == nil {
		k, err := utils_cert.GenerateECDSA()
		if err != nil {
			c.setIssuerNotReady(ctx, uid)
			return err
		}

		keyPEM, err := k.GetPrivateKeyPEM()
		if err != nil {
			c.setIssuerNotReady(ctx, uid)
			return err
		}

		account = &Account{
			Email: email,
			key:   k.PrivateKey,
		}
		privateKeyPEM = []byte(keyPEM)
	}

	acmeC, err := c.newLegoClientWithAccount(lockCtx, account, directoryURL)
	if err != nil {
		c.setIssuerNotReady(ctx, uid)
		return err
	}

	rsc, err := doRegisterACMEAccount(acmeC, account)
	if err != nil {
		c.setIssuerNotReady(ctx, uid)
		return errors.Wrap(err, "Could not register the ACME account")
	}

	account.Registration = rsc

	sec, err := c.upsertACMEAccountSecret(lockCtx, iss, account, privateKeyPEM, directoryURL)
	if err != nil {
		c.setIssuerNotReady(ctx, uid)
		return err
	}

	if iss, err = c.setIssuerState(lockCtx, uid,
		enterprisev1.CertificateIssuer_Status_READY,
		umetav1.GetObjectReference(sec)); err != nil {
		return err
	}

	c.clearIssuerForce(uid)

	zap.L().Info("Successfully registered the ACME account",
		zap.String("uid", uid), zap.String("name", iss.Metadata.Name))

	if err := c.enqueueIssuerCertificates(lockCtx, iss, directoryChanged); err != nil {
		zap.L().Warn("Could not enqueue the CertificateIssuer Certificates",
			zap.String("uid", uid), zap.Error(err))
	}

	return nil
}

func doRegisterACMEAccount(acmeC *lego.Client, account *Account) (
	*registration.Resource, error) {

	opts := registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	}

	if account.Registration != nil {
		rsc, err := acmeC.Registration.QueryRegistration()
		if err == nil {
			account.Registration = rsc
			return acmeC.Registration.UpdateRegistration(opts)
		}
		if !isAccountDoesNotExist(err) {
			return nil, err
		}
		account.Registration = nil
	}

	rsc, err := acmeC.Registration.ResolveAccountByKey()
	if err == nil {
		account.Registration = rsc
		return acmeC.Registration.UpdateRegistration(opts)
	}
	if !isAccountDoesNotExist(err) {
		return nil, err
	}

	return acmeC.Registration.Register(opts)
}

func (c *Controller) getIssuerEmail(ctx context.Context,
	iss *enterprisev1.CertificateIssuer) (string, error) {

	if email := strings.TrimSpace(iss.Spec.GetAcme().Email); email != "" {
		return email, nil
	}

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return "", err
	}

	domain := normalizeDomain(cc.Status.Domain)
	if !isValidDNSName(domain, false) {
		return "", &permanentError{
			err: errors.Errorf("Invalid Cluster domain: %s", cc.Status.Domain),
		}
	}

	return fmt.Sprintf("contact@%s", domain), nil
}

func (c *Controller) loadACMEAccount(ctx context.Context,
	iss *enterprisev1.CertificateIssuer, email string) (*acmeAccount, error) {

	sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: uenterprisev1.ToCertificateIssuer(iss).GetACMEAccountSecretName(),
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	dataMap := sec.Data.GetDataMap().GetMap()
	if len(dataMap[acmeSecretAccountKey]) == 0 || len(dataMap[acmeSecretPrivateKeyKey]) == 0 {
		zap.L().Warn("The ACME account Secret is incomplete. Registering a new account...",
			zap.String("secret", sec.Metadata.Name))
		return nil, nil
	}

	var account Account
	if err := json.Unmarshal(dataMap[acmeSecretAccountKey], &account); err != nil {
		zap.L().Warn("Could not unmarshal the ACME account. Registering a new account...",
			zap.String("secret", sec.Metadata.Name), zap.Error(err))
		return nil, nil
	}

	privateKey, err := utils_cert.ParsePrivateKeyPEM(dataMap[acmeSecretPrivateKeyKey])
	if err != nil {
		zap.L().Warn("Could not parse the ACME account key. Registering a new account...",
			zap.String("secret", sec.Metadata.Name), zap.Error(err))
		return nil, nil
	}

	account.Email = email
	account.key = privateKey

	return &acmeAccount{
		account:       &account,
		privateKeyPEM: dataMap[acmeSecretPrivateKeyKey],
		directoryURL:  string(dataMap[acmeSecretDirectoryURLKey]),
	}, nil
}

func (c *Controller) getACMEAccount(ctx context.Context,
	iss *enterprisev1.CertificateIssuer) (*Account, error) {

	sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Uid: iss.Status.GetAcme().SecretRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	dataMap := sec.Data.GetDataMap().GetMap()

	var account Account
	if err := json.Unmarshal(dataMap[acmeSecretAccountKey], &account); err != nil {
		return nil, &permanentError{
			err: errors.Errorf("Could not unmarshal the ACME account: %+v", err),
		}
	}

	if account.Registration == nil {
		return nil, &deferredError{
			err:   errors.Errorf("The ACME account is not registered yet"),
			delay: minRetryDelay,
		}
	}

	privateKey, err := utils_cert.ParsePrivateKeyPEM(dataMap[acmeSecretPrivateKeyKey])
	if err != nil {
		return nil, &permanentError{
			err: errors.Errorf("Could not parse the ACME account key: %+v", err),
		}
	}

	account.key = privateKey

	return &account, nil
}

func (c *Controller) newLegoClient(ctx context.Context,
	iss *enterprisev1.CertificateIssuer) (*lego.Client, error) {

	account, err := c.getACMEAccount(ctx, iss)
	if err != nil {
		return nil, err
	}

	return c.newLegoClientWithAccount(ctx, account, getCADirURL(iss))
}

func (c *Controller) newLegoClientWithAccount(ctx context.Context,
	account *Account, directoryURL string) (*lego.Client, error) {

	if err := validateDirectoryURL(directoryURL); err != nil {
		return nil, &permanentError{err: err}
	}

	cfg := lego.NewConfig(account)
	cfg.CADirURL = directoryURL
	cfg.UserAgent = "octelium-cloudman"
	cfg.HTTPClient = newACMEHTTPClient(ctx)
	cfg.Certificate.Timeout = acmeOrderTimeout

	return lego.NewClient(cfg)
}

func (c *Controller) upsertACMEAccountSecret(ctx context.Context,
	iss *enterprisev1.CertificateIssuer, account *Account,
	privateKeyPEM []byte, directoryURL string) (*enterprisev1.Secret, error) {

	accountBytes, err := json.Marshal(account)
	if err != nil {
		return nil, err
	}

	name := uenterprisev1.ToCertificateIssuer(iss).GetACMEAccountSecretName()

	for range statusUpdateRetries {
		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: name,
		})
		if err != nil {
			if !grpcerr.IsNotFound(err) {
				return nil, err
			}

			sec, err = c.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
				Metadata: &metav1.Metadata{
					Name:           name,
					IsSystem:       true,
					IsUserHidden:   true,
					IsSystemHidden: true,
				},
				Spec:   &enterprisev1.Secret_Spec{},
				Status: &enterprisev1.Secret_Status{},
				Data: &enterprisev1.Secret_Data{
					Type: &enterprisev1.Secret_Data_DataMap_{
						DataMap: &enterprisev1.Secret_Data_DataMap{
							Map: map[string][]byte{
								acmeSecretAccountKey:      accountBytes,
								acmeSecretPrivateKeyKey:   privateKeyPEM,
								acmeSecretDirectoryURLKey: []byte(directoryURL),
							},
						},
					},
				},
			})
			if err == nil {
				return sec, nil
			}
			if !grpcerr.AlreadyExists(err) {
				return nil, err
			}
		} else {
			if sec.Data.GetDataMap().GetMap() == nil {
				sec.Data = &enterprisev1.Secret_Data{
					Type: &enterprisev1.Secret_Data_DataMap_{
						DataMap: &enterprisev1.Secret_Data_DataMap{
							Map: make(map[string][]byte),
						},
					},
				}
			}

			sec.Metadata.IsSystem = true
			sec.Metadata.IsUserHidden = true
			sec.Metadata.IsSystemHidden = true

			sec.Data.GetDataMap().Map[acmeSecretAccountKey] = accountBytes
			sec.Data.GetDataMap().Map[acmeSecretPrivateKeyKey] = privateKeyPEM
			sec.Data.GetDataMap().Map[acmeSecretDirectoryURLKey] = []byte(directoryURL)

			sec, err = c.octeliumC.EnterpriseC().UpdateSecret(ctx, sec)
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

	return nil, errors.Errorf("Could not set the ACME account Secret: %s", name)
}

func (c *Controller) setIssuerState(ctx context.Context, uid string,
	state enterprisev1.CertificateIssuer_Status_State,
	secretRef *metav1.ObjectReference) (*enterprisev1.CertificateIssuer, error) {

	for range statusUpdateRetries {
		iss, err := c.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Uid: uid,
		})
		if err != nil {
			return nil, err
		}

		iss.Status.State = state
		if secretRef != nil {
			iss.Status.Type = &enterprisev1.CertificateIssuer_Status_Acme{
				Acme: &enterprisev1.CertificateIssuer_Status_ACME{
					SecretRef: secretRef,
				},
			}
		}

		updated, err := c.octeliumC.EnterpriseC().UpdateCertificateIssuer(ctx, iss)
		if err == nil {
			return updated, nil
		}
		if !grpcerr.IsResourceChanged(err) {
			return nil, err
		}

		if err := sleepContext(ctx, statusUpdateInterval); err != nil {
			return nil, err
		}
	}

	return nil, errors.Errorf("Could not update the CertificateIssuer state: %s", uid)
}

func (c *Controller) setIssuerNotReady(ctx context.Context, uid string) {
	updateCtx, cancel := c.detachedContext(ctx)
	defer cancel()

	if _, err := c.setIssuerState(updateCtx, uid,
		enterprisev1.CertificateIssuer_Status_NOT_READY, nil); err != nil &&
		!grpcerr.IsNotFound(err) {
		zap.L().Warn("Could not set the CertificateIssuer as not ready",
			zap.String("uid", uid), zap.Error(err))
	}
}

func (c *Controller) enqueueIssuerCertificates(ctx context.Context,
	iss *enterprisev1.CertificateIssuer, reissue bool) error {

	return c.listCertificates(ctx, []*rmetav1.ListOptions_Filter{
		urscsrv.FilterFieldEQValStr("status.certificateIssuerRef.uid", iss.Metadata.Uid),
	}, func(crt *enterprisev1.Certificate) error {
		if !isACMECertificate(crt) {
			return nil
		}

		if reissue && !isIssuanceRequested(crt) {
			if _, err := c.requestIssuance(ctx, crt); err != nil {
				zap.L().Warn("Could not request the Certificate issuance",
					zap.String("uid", crt.Metadata.Uid), zap.Error(err))
			}
		}

		c.enqueue(workKey{
			typ: workTypeCertificate,
			uid: crt.Metadata.Uid,
		}, true)

		return nil
	})
}
