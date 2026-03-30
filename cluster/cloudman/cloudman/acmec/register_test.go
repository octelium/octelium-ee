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
	"fmt"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestServer(t *testing.T) {

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)
	cc.Status.Domain = fmt.Sprintf("%s.octelium.org", utilrand.GetRandomStringCanonical(8))
	cc, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)

	iss, err := fakeC.OcteliumC.EnterpriseC().CreateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.CertificateIssuer_Spec{
			Type: &enterprisev1.CertificateIssuer_Spec_Acme{
				Acme: &enterprisev1.CertificateIssuer_Spec_ACME{
					Solver: &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
						Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{
							Dns: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_DNS{},
						},
					},
				},
			},
		},
		Status: &enterprisev1.CertificateIssuer_Status{
			State: enterprisev1.CertificateIssuer_Status_NOT_READY,
		},
	})
	assert.Nil(t, err)
	err = RegisterAccount(ctx, fakeC.OcteliumC, iss, false)
	assert.Nil(t, err)

	iss, err = fakeC.OcteliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
		Uid: iss.Metadata.Uid,
	})
	assert.Nil(t, err)

	assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, iss.Status.State)

	sec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Uid: iss.Status.GetAcme().SecretRef.Uid,
	})
	assert.Nil(t, err)

	_, err = utils_cert.ParsePrivateKeyPEM(sec.Data.GetDataMap().Map["privateKey"])
	assert.Nil(t, err)

	// do again
	err = RegisterAccount(ctx, fakeC.OcteliumC, iss, false)
	assert.Nil(t, err)
}
