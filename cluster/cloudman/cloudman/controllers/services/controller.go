// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package svccontroller

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnss"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	octeliumC octeliumc.ClientInterface,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
	}
}

func (c *Controller) OnAdd(ctx context.Context, svc *corev1.Service) error {
	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}
	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, c.octeliumC)
	if err != nil {
		return err
	}

	if svc.Spec.IsPublic {
		if err := dnss.SetServiceCNAME(ctx, c.octeliumC, svc, cc, provider); err != nil {
			return err
		}
	}

	if err := doSetCert(ctx, c.octeliumC, svc); err != nil {
		return err
	}

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Service) error {
	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}
	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, c.octeliumC)
	if err != nil {
		return err
	}

	if new.Spec.IsPublic && !old.Spec.IsPublic {
		if err := dnss.SetServiceCNAME(ctx, c.octeliumC, new, cc, provider); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, svc *corev1.Service) error {
	if !needsCert(svc) {
		return nil
	}

	if _, err := c.octeliumC.EnterpriseC().DeleteCertificate(ctx, &rmetav1.DeleteOptions{
		Name: fmt.Sprintf("svc-%s-%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name),
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func needsCert(svc *corev1.Service) bool {
	return svc.Status.ManagedService != nil && svc.Status.ManagedService.HasSubdomain
}

func doSetCert(ctx context.Context, octeliumC octeliumc.ClientInterface, svc *corev1.Service) error {
	if !needsCert(svc) {
		return nil
	}

	if _, err := octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
		Name: fmt.Sprintf("svc-%s-%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name),
	}); err == nil {
		return nil
	} else if !grpcerr.IsNotFound(err) {
		return err
	}

	iss, err := cloudmanutils.GetDefaultCertificateIssuer(ctx, octeliumC)
	if err != nil {
		return err
	}

	cc, err := octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	mode := func() enterprisev1.Certificate_Spec_Mode {
		if cc.Spec.Certificate != nil && cc.Spec.Certificate.DefaultMode != enterprisev1.Certificate_Spec_MODE_UNSET {
			return cc.Spec.Certificate.DefaultMode
		}
		return enterprisev1.Certificate_Spec_MANUAL
	}()

	if _, err := octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("svc-%s-%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name),
		},
		Spec: &enterprisev1.Certificate_Spec{
			Mode: mode,
		},
		Status: &enterprisev1.Certificate_Status{
			Issuance: func() *enterprisev1.Certificate_Status_Issuance {
				switch mode {
				case enterprisev1.Certificate_Spec_MANAGED:
					return &enterprisev1.Certificate_Status_Issuance{
						CreatedAt: pbutils.Now(),
						State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
					}
				default:
					return nil
				}

			}(),
			ServiceRef:           umetav1.GetObjectReference(svc),
			NamespaceRef:         svc.Status.NamespaceRef,
			CertificateIssuerRef: umetav1.GetObjectReference(iss),
		},
	}); err != nil {
		return err
	}

	return nil
}
