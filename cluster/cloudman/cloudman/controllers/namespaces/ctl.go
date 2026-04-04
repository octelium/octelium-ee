// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package nscontroller

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
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

func (c *Controller) OnAdd(ctx context.Context, ns *corev1.Namespace) error {
	zap.S().Debugf("New Namespace: %s", ns.Metadata.Name)
	if err := doSetCert(ctx, c.octeliumC, ns); err != nil {
		return err
	}
	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Namespace) error {

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, ns *corev1.Namespace) error {

	if _, err := c.octeliumC.EnterpriseC().DeleteCertificate(ctx, &rmetav1.DeleteOptions{
		Name: fmt.Sprintf("ns-%s", ns.Metadata.Name),
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func doSetCert(ctx context.Context, octeliumC octeliumc.ClientInterface, ns *corev1.Namespace) error {

	if _, err := octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
		Name: fmt.Sprintf("ns-%s", ns.Metadata.Name),
	}); err == nil {
		return nil
	} else if !grpcerr.IsNotFound(err) {
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

	var certificateIssuerRef *metav1.ObjectReference
	switch mode {
	case enterprisev1.Certificate_Spec_MANAGED:
		iss, err := cloudmanutils.GetDefaultCertificateIssuer(ctx, octeliumC)
		if err != nil {
			return err
		}
		certificateIssuerRef = umetav1.GetObjectReference(iss)
	}

	if _, err := octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("ns-%s", ns.Metadata.Name),
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
			NamespaceRef:         umetav1.GetObjectReference(ns),
			CertificateIssuerRef: certificateIssuerRef,
		},
	}); err != nil {
		return err
	}

	return nil
}
