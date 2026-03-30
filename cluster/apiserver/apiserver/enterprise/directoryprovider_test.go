// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestDirectoryProvider(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		dp, err := srv.CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.DirectoryProvider_Spec{
				Type: &enterprisev1.DirectoryProvider_Spec_Scim{
					Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
				},
			},
		})
		assert.Nil(t, err)

		usr, err := srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dp.Status.UserRef.Uid,
		})

		assert.Nil(t, err)

		assert.Nil(t, dp.Status.SessionRef)

		info := &enterprisev1.UserExtInfo{}
		err = pbutils.StructToMessage(usr.Status.Ext[ovutils.ExtInfoKeyEnterprise], info)
		assert.Nil(t, err)
		assert.Equal(t, info.DirectoryProviderRef.Uid, dp.Metadata.Uid)

		{
			resp, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err)

			jwkCtl, err := jwkctl.NewJWKController(ctx, srv.octeliumC)
			assert.Nil(t, err)

			tkn, err := jwkCtl.VerifyAccessToken(resp.GetBearer().AccessToken)
			assert.Nil(t, err)

			dp, err = srv.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
				Uid: dp.Metadata.Uid,
			})
			assert.Nil(t, err)

			assert.NotNil(t, dp.Status.SessionRef)

			sess, err := srv.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
				Uid: dp.Status.SessionRef.Uid,
			})
			assert.Nil(t, err)

			assert.Equal(t, sess.Status.Authentication.TokenID, tkn.TokenID)

			usr, err := srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
				Uid: sess.Status.UserRef.Uid,
			})
			assert.Nil(t, err)

			info := &enterprisev1.UserExtInfo{}
			err = pbutils.StructToMessage(usr.Status.Ext[ovutils.ExtInfoKeyEnterprise], info)
			assert.Nil(t, err)
			assert.Equal(t, info.DirectoryProviderRef.Uid, dp.Metadata.Uid)
		}

		{
			resp, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err)

			jwkCtl, err := jwkctl.NewJWKController(ctx, srv.octeliumC)
			assert.Nil(t, err)

			tkn, err := jwkCtl.VerifyAccessToken(resp.GetBearer().AccessToken)
			assert.Nil(t, err)

			sess, err := srv.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
				Uid: dp.Status.SessionRef.Uid,
			})
			assert.Nil(t, err)

			srv.octeliumC.CoreC().DeleteSession(ctx, &rmetav1.DeleteOptions{
				Uid: sess.Metadata.Uid,
			})
			assert.Nil(t, err)

			assert.Equal(t, sess.Status.Authentication.TokenID, tkn.TokenID)

			resp, err = srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err)
		}
	}
}

/*
 func TestUpgradeCluster(t *testing.T) {

	 tst, err := tests.Initialize(nil)
	 assert.Nil(t, err)
	 t.Cleanup(func() {
		 tst.Destroy()
	 })



	 srv := NewServer(tst.C.OcteliumC)

	 u, err := tests.NewUser(srv.octeliumC, admin.NewServer(&admin.Opts{
		 OcteliumC:  tst.C.OcteliumC,
		 IsEmbedded: true,
	 }), nil, nil)
	 assert.Nil(t, err)


		 _, err = srv.UpgradeCluster(u.Ctx(), &clusterv1.UpgradeClusterRequest{})
		 assert.Nil(t, err)

 }
*/
