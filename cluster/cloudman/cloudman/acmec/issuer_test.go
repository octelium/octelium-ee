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
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCertificateIssuer(t *testing.T) {
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

	c := NewController(fakeC.OcteliumC)
	t.Cleanup(c.shutdown)

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)
	cc.Status.Domain = fmt.Sprintf("%s.octelium.org", utilrand.GetRandomStringCanonical(8))
	cc, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)

	newIssuer := func(acme *enterprisev1.CertificateIssuer_Spec_ACME) *enterprisev1.CertificateIssuer {
		iss, err := fakeC.OcteliumC.EnterpriseC().CreateCertificateIssuer(ctx,
			&enterprisev1.CertificateIssuer{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Spec: &enterprisev1.CertificateIssuer_Spec{
					Type: &enterprisev1.CertificateIssuer_Spec_Acme{
						Acme: acme,
					},
				},
				Status: &enterprisev1.CertificateIssuer_Status{
					State: enterprisev1.CertificateIssuer_Status_NOT_READY,
				},
			})
		assert.Nil(t, err, "%+v", err)

		return iss
	}

	getIssuer := func(uid string) *enterprisev1.CertificateIssuer {
		iss, err := fakeC.OcteliumC.EnterpriseC().GetCertificateIssuer(ctx,
			&rmetav1.GetOptions{Uid: uid})
		assert.Nil(t, err)
		return iss
	}

	t.Run("NotAnACMEIssuer", func(t *testing.T) {
		iss, err := fakeC.OcteliumC.EnterpriseC().CreateCertificateIssuer(ctx,
			&enterprisev1.CertificateIssuer{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Spec:   &enterprisev1.CertificateIssuer_Spec{},
				Status: &enterprisev1.CertificateIssuer_Status{},
			})
		assert.Nil(t, err)

		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_STATE_UNKNOWN,
			getIssuer(iss.Metadata.Uid).Status.State)
	})

	t.Run("NonExistentIssuer", func(t *testing.T) {
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, vutils.UUIDv4()))
	})

	t.Run("InvalidDirectoryURL", func(t *testing.T) {
		iss := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{
			Server: "ftp://acme.example.com/directory",
		})

		err := c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid)
		assert.NotNil(t, err)
		assert.True(t, getRetryDelay(1, err) >= 10*time.Minute)

		assert.Equal(t, enterprisev1.CertificateIssuer_Status_NOT_READY,
			getIssuer(iss.Metadata.Uid).Status.State)
	})

	iss := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{
		Email: fmt.Sprintf("contact@%s", cc.Status.Domain),
		Solver: &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
			Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{
				Dns: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_DNS{},
			},
		},
	})

	err = c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid)
	assert.Nil(t, err, "%+v", err)

	iss = getIssuer(iss.Metadata.Uid)
	assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, iss.Status.State)
	assert.NotNil(t, iss.Status.GetAcme())
	assert.NotNil(t, iss.Status.GetAcme().SecretRef)

	sec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Uid: iss.Status.GetAcme().SecretRef.Uid,
	})
	assert.Nil(t, err)
	assert.Equal(t, uenterprisev1.ToCertificateIssuer(iss).GetACMEAccountSecretName(),
		sec.Metadata.Name)
	assert.True(t, sec.Metadata.IsSystem)

	dataMap := sec.Data.GetDataMap().GetMap()
	assert.Equal(t, getCADirURL(iss), string(dataMap[acmeSecretDirectoryURLKey]))

	privateKeyPEM := dataMap[acmeSecretPrivateKeyKey]
	_, err = utils_cert.ParsePrivateKeyPEM(privateKeyPEM)
	assert.Nil(t, err)

	{
		var account Account
		assert.Nil(t, json.Unmarshal(dataMap[acmeSecretAccountKey], &account))
		assert.NotNil(t, account.Registration)
		assert.Equal(t, fmt.Sprintf("contact@%s", cc.Status.Domain), account.Email)
	}

	t.Run("AlreadyReadyIsANoOp", func(t *testing.T) {
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, iss.Metadata.ResourceVersion, cur.Metadata.ResourceVersion)
	})

	t.Run("LoadedAccount", func(t *testing.T) {
		loaded, err := c.loadACMEAccount(ctx, iss, "contact@example.com")
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, loaded)
		assert.Equal(t, "contact@example.com", loaded.account.Email)
		assert.NotNil(t, loaded.account.GetPrivateKey())
		assert.NotNil(t, loaded.account.GetRegistration())
		assert.Equal(t, getCADirURL(iss), loaded.directoryURL)

		account, err := c.getACMEAccount(ctx, iss)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, account.GetRegistration())

		acmeC, err := c.newLegoClient(ctx, iss)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, acmeC)
	})

	t.Run("ForceKeepsTheAccountKey", func(t *testing.T) {
		assert.Nil(t, c.SetCertificateIssuer(ctx, iss, true))
		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, cur.Status.State)
		assert.False(t, c.hasIssuerForce(cur.Metadata.Uid))

		curSec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: cur.Status.GetAcme().SecretRef.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, string(privateKeyPEM),
			string(curSec.Data.GetDataMap().GetMap()[acmeSecretPrivateKeyKey]))
	})

	t.Run("ACorruptAccountIsReRegistered", func(t *testing.T) {
		curSec, err := fakeC.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: iss.Status.GetAcme().SecretRef.Uid,
		})
		assert.Nil(t, err)

		curSec.Data.GetDataMap().Map[acmeSecretAccountKey] = []byte("not-an-account")
		_, err = fakeC.OcteliumC.EnterpriseC().UpdateSecret(ctx, curSec)
		assert.Nil(t, err)

		loaded, err := c.loadACMEAccount(ctx, iss, "contact@example.com")
		assert.Nil(t, err)
		assert.Nil(t, loaded)

		assert.Nil(t, c.reconcileCertificateIssuer(ctx, iss.Metadata.Uid))

		cur := getIssuer(iss.Metadata.Uid)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, cur.Status.State)

		account, err := c.getACMEAccount(ctx, cur)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, account.GetRegistration())
	})

	t.Run("EmailFallsBackToTheClusterDomain", func(t *testing.T) {
		other := newIssuer(&enterprisev1.CertificateIssuer_Spec_ACME{})

		email, err := c.getIssuerEmail(ctx, other)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("contact@%s", cc.Status.Domain), email)
	})

	t.Run("SetIssuerState", func(t *testing.T) {
		cur, err := c.setIssuerState(ctx, iss.Metadata.Uid,
			enterprisev1.CertificateIssuer_Status_NOT_READY, nil)
		assert.Nil(t, err)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_NOT_READY, cur.Status.State)
		assert.NotNil(t, cur.Status.GetAcme())

		_, err = c.setIssuerState(ctx, vutils.UUIDv4(),
			enterprisev1.CertificateIssuer_Status_NOT_READY, nil)
		assert.NotNil(t, err)
	})
}
