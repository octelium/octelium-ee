// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLicenseCredential(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	{
		_, err := GetLicenseCredential(ctx, tst.C.OcteliumC)
		unavailable := &LicenseUnavailableError{}
		require.ErrorAs(t, err, &unavailable)
		assert.Equal(t, "The Cluster does not have a License", unavailable.Reason)
	}

	{
		jwtStr, lic := tstValidLicense(t, signer)
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr,
		})
		assert.Nil(t, err, "%+v", err)

		cred, err := GetLicenseCredential(ctx, tst.C.OcteliumC)
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, jwtStr, cred.JWT)
		assert.Equal(t, lic.Uid, cred.License.Uid)

		clusterDomain, err := srv.getLicenseClusterDomain(ctx)
		assert.Nil(t, err)
		assert.Equal(t, clusterDomain, cred.ClusterDomain)
	}

	{
		cc, err := tst.C.OcteliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		require.Nil(t, err)
		sec, err := tst.C.OcteliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: cc.Status.LicenseInfo.LicenseSecretRef.Uid,
		})
		require.Nil(t, err)

		now := time.Now().UTC().Truncate(time.Second)
		expired := tstLicenseSpec()
		expired.IssuedAt = pbutils.Timestamp(now.Add(-72 * time.Hour))
		expired.NotBefore = pbutils.Timestamp(now.Add(-72 * time.Hour))
		expired.NotAfter = pbutils.Timestamp(now.Add(-24 * time.Hour))
		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: tstSignLicense(t, signer, tstLicenseClaims(expired), expired),
			},
		}
		sec, err = tst.C.OcteliumC.EnterpriseC().UpdateSecret(ctx, sec)
		require.Nil(t, err)

		_, err = GetLicenseCredential(ctx, tst.C.OcteliumC)
		unavailable := &LicenseUnavailableError{}
		require.ErrorAs(t, err, &unavailable)
		assert.Equal(t, "The License of the Cluster has expired", unavailable.Reason)

		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: "not-a-license",
			},
		}
		_, err = tst.C.OcteliumC.EnterpriseC().UpdateSecret(ctx, sec)
		require.Nil(t, err)

		_, err = GetLicenseCredential(ctx, tst.C.OcteliumC)
		require.ErrorAs(t, err, &unavailable)
		assert.Equal(t, "The License of the Cluster is not valid", unavailable.Reason)
	}

	{
		_, err := srv.DeleteLicense(ctx, &enterprisev1.DeleteLicenseRequest{})
		assert.Nil(t, err, "%+v", err)

		_, err = GetLicenseCredential(ctx, tst.C.OcteliumC)
		unavailable := &LicenseUnavailableError{}
		require.ErrorAs(t, err, &unavailable)
	}
}
