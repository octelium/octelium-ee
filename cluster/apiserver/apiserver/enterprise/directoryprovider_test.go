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
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
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
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, dp.Status.Id)
		assert.NotNil(t, dp.Status.UserRef)
		assert.Nil(t, dp.Status.SessionRef)

		usr, err := srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dp.Status.UserRef.Uid,
		})
		assert.Nil(t, err, "%+v", err)

		info := &enterprisev1.UserExtInfo{}
		err = pbutils.StructToMessage(usr.Status.Ext[ovutils.ExtInfoKeyEnterprise], info)
		assert.Nil(t, err)
		assert.Equal(t, dp.Metadata.Uid, info.DirectoryProviderRef.Uid)

		{
			ret, err := srv.GetDirectoryProvider(ctx, &metav1.GetOptions{Uid: dp.Metadata.Uid})
			assert.Nil(t, err, "%+v", err)
			assert.Equal(t, dp.Metadata.Uid, ret.Metadata.Uid)

			ret, err = srv.GetDirectoryProvider(ctx, &metav1.GetOptions{Name: dp.Metadata.Name})
			assert.Nil(t, err, "%+v", err)
			assert.Equal(t, dp.Metadata.Uid, ret.Metadata.Uid)
		}

		{
			itemList, err := srv.ListDirectoryProvider(ctx, nil)
			assert.Nil(t, err, "%+v", err)
			found := false
			for _, item := range itemList.Items {
				if item.Metadata.Uid == dp.Metadata.Uid {
					found = true
				}
			}
			assert.True(t, found)
		}

		{
			_, err = srv.CreateDirectoryProvider(ctx, tstCloneDirectoryProvider(dp))
			assert.NotNil(t, err)
		}

		{
			arg := tstCloneDirectoryProvider(dp)
			arg.Spec.IsDisabled = true
			arg.Status = &enterprisev1.DirectoryProvider_Status{}

			updated, err := srv.UpdateDirectoryProvider(ctx, arg)
			assert.Nil(t, err, "%+v", err)
			assert.True(t, updated.Spec.IsDisabled)
			assert.Equal(t, dp.Status.Id, updated.Status.Id)
			assert.Equal(t, dp.Status.UserRef.Uid, updated.Status.UserRef.Uid)
			dp = updated
		}

		{
			resp, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err, "%+v", err)

			jwkCtl, err := jwkctl.NewJWKController(ctx, srv.octeliumC, nil)
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
			assert.Equal(t, dp.Metadata.Uid, info.DirectoryProviderRef.Uid)
		}

		{
			resp, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err, "%+v", err)

			jwkCtl, err := jwkctl.NewJWKController(ctx, srv.octeliumC, nil)
			assert.Nil(t, err)

			tkn, err := jwkCtl.VerifyAccessToken(resp.GetBearer().AccessToken)
			assert.Nil(t, err)

			sess, err := srv.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
				Uid: dp.Status.SessionRef.Uid,
			})
			assert.Nil(t, err)

			_, err = srv.octeliumC.CoreC().DeleteSession(ctx, &rmetav1.DeleteOptions{
				Uid: sess.Metadata.Uid,
			})
			assert.Nil(t, err)
			assert.Equal(t, sess.Status.Authentication.TokenID, tkn.TokenID)

			resp, err = srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
			assert.Nil(t, err, "%+v", err)
			assert.NotEmpty(t, resp.GetBearer().AccessToken)
		}

		{
			_, err = srv.DeleteDirectoryProvider(ctx, &metav1.DeleteOptions{Uid: dp.Metadata.Uid})
			assert.Nil(t, err, "%+v", err)

			_, err = srv.GetDirectoryProvider(ctx, &metav1.GetOptions{Uid: dp.Metadata.Uid})
			assert.NotNil(t, err)
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		}
	}
}

func TestDirectoryProviderTypes(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)
	secretName := tstCreateDirectoryProviderSecret(ctx, t, srv)

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderGoogleWorkspace(secretName))
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, dp.Status.Id)
		assert.Nil(t, dp.Status.UserRef)
		assert.Equal(t, "customer", dp.Spec.GetGoogleWorkspace().Customer)
		assert.Equal(t, "admin@octelium.com", dp.Spec.GetGoogleWorkspace().ImpersonateSubject)

		_, err = srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
		assert.Nil(t, err, "%+v", err)

		dp, err = srv.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Uid: dp.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, dp.Status.Synchronization)
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED, dp.Status.Synchronization.State)
	}

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderKeycloak(secretName))
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, dp.Status.Id)
		assert.Nil(t, dp.Status.UserRef)
		assert.Equal(t, "https://keycloak.example.com", dp.Spec.GetKeycloak().Url)
		assert.True(t, dp.Spec.GetKeycloak().InsecureSkipVerify)

		_, err = srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
		assert.Nil(t, err, "%+v", err)

		dp, err = srv.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Uid: dp.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, dp.Status.Synchronization)
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED, dp.Status.Synchronization.State)

		_, err = srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
			Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestValidateDirectoryProvider(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)
	secretName := tstCreateDirectoryProviderSecret(ctx, t, srv)

	{
		err := srv.validateDirectoryProvider(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{
				Type: &enterprisev1.DirectoryProvider_Spec_Scim{
					Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{
				IsDisabled: true,
				Type: &enterprisev1.DirectoryProvider_Spec_Scim{
					Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{
				Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().Customer = ""
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().Customer = strings.Repeat("a", maxDirectoryProviderStringBytes+1)
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().ServiceAccount = nil
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().ServiceAccount = tstDirectoryProviderGoogleWorkspaceServiceAccount(utilrand.GetRandomStringCanonical(8))
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().ImpersonateSubject = "not-email"
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().ImpersonateSubject = strings.Repeat("a", maxDirectoryProviderStringBytes+1) + "@octelium.com"
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderGoogleWorkspace(secretName)
		item.Spec.GetGoogleWorkspace().Polling = &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_Polling{}
		err := srv.validateDirectoryProvider(ctx, item)
		assert.Nil(t, err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, tstDirectoryProviderGoogleWorkspace(secretName))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{
				Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Url = ""
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Url = "not-url"
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Url = "ftp://keycloak.example.com"
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Url = "https://" + strings.Repeat("a", maxDirectoryProviderURLBytes+1) + ".example.com"
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Realm = ""
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().ClientID = ""
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().ClientSecret = nil
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().ClientSecret = tstDirectoryProviderKeycloakClientSecret(utilrand.GetRandomStringCanonical(8))
		err := srv.validateDirectoryProvider(ctx, item)
		assert.NotNil(t, err)
	}

	{
		item := tstDirectoryProviderKeycloak(secretName)
		item.Spec.GetKeycloak().Polling = &enterprisev1.DirectoryProvider_Spec_Keycloak_Polling{}
		err := srv.validateDirectoryProvider(ctx, item)
		assert.Nil(t, err)
	}

	{
		err := srv.validateDirectoryProvider(ctx, tstDirectoryProviderKeycloak(secretName))
		assert.Nil(t, err, "%+v", err)
	}
}

func TestGenerateDirectoryProviderCredential(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		_, err := srv.GenerateDirectoryProviderCredential(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{})
		assert.NotNil(t, err)
	}

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
			Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_Mode(1000),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: utilrand.GetRandomStringCanonical(8)},
			Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestSynchronizeDirectoryProvider(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)
	secretName := tstCreateDirectoryProviderSecret(ctx, t, srv)

	{
		_, err := srv.SynchronizeDirectoryProvider(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{})
		assert.NotNil(t, err)
	}

	{
		_, err := srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: utilrand.GetRandomStringCanonical(8)},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		dp, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderKeycloak(secretName))
		assert.Nil(t, err, "%+v", err)

		dp.Status.Synchronization = &enterprisev1.DirectoryProvider_Status_Synchronization{
			CreatedAt: pbutils.Now(),
			State:     enterprisev1.DirectoryProvider_Status_Synchronization_SYNCING,
		}
		dp, err = srv.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, dp)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SynchronizeDirectoryProvider(ctx, &enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestListDirectoryProviderUsersAndGroups(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)
	dp1, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
	assert.Nil(t, err, "%+v", err)
	dp2, err := srv.CreateDirectoryProvider(ctx, tstDirectoryProviderSCIM())
	assert.Nil(t, err, "%+v", err)

	{
		_, err = srv.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, &enterprisev1.DirectoryProviderUser{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Status: &enterprisev1.DirectoryProviderUser_Status{
				DirectoryProviderRef: umetav1.GetObjectReference(dp1),
				UserRef:              &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
			},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, &enterprisev1.DirectoryProviderUser{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Status: &enterprisev1.DirectoryProviderUser_Status{
				DirectoryProviderRef: umetav1.GetObjectReference(dp2),
				UserRef:              &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
			},
		})
		assert.Nil(t, err, "%+v", err)

		itemList, err := srv.ListDirectoryProviderUser(ctx, &enterprisev1.ListDirectoryProviderUserOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp1),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, itemList.Items, 1)
		assert.Equal(t, dp1.Metadata.Uid, itemList.Items[0].Status.DirectoryProviderRef.Uid)

		_, err = srv.ListDirectoryProviderUser(ctx, &enterprisev1.ListDirectoryProviderUserOptions{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: utilrand.GetRandomStringCanonical(8)},
		})
		assert.NotNil(t, err)
	}

	{
		_, err = srv.octeliumC.EnterpriseC().CreateDirectoryProviderGroup(ctx, &enterprisev1.DirectoryProviderGroup{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Status: &enterprisev1.DirectoryProviderGroup_Status{
				DirectoryProviderRef: umetav1.GetObjectReference(dp1),
				GroupRef:             &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
			},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.octeliumC.EnterpriseC().CreateDirectoryProviderGroup(ctx, &enterprisev1.DirectoryProviderGroup{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Status: &enterprisev1.DirectoryProviderGroup_Status{
				DirectoryProviderRef: umetav1.GetObjectReference(dp2),
				GroupRef:             &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
			},
		})
		assert.Nil(t, err, "%+v", err)

		itemList, err := srv.ListDirectoryProviderGroup(ctx, &enterprisev1.ListDirectoryProviderGroupOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp1),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, itemList.Items, 1)
		assert.Equal(t, dp1.Metadata.Uid, itemList.Items[0].Status.DirectoryProviderRef.Uid)

		_, err = srv.ListDirectoryProviderGroup(ctx, &enterprisev1.ListDirectoryProviderGroupOptions{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: utilrand.GetRandomStringCanonical(8)},
		})
		assert.NotNil(t, err)
	}
}

func tstDirectoryProviderSCIM() *enterprisev1.DirectoryProvider {
	return &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	}
}

func tstDirectoryProviderGoogleWorkspace(secretName string) *enterprisev1.DirectoryProvider {
	return &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{
				GoogleWorkspace: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace{
					Customer:           "customer",
					ServiceAccount:     tstDirectoryProviderGoogleWorkspaceServiceAccount(secretName),
					ImpersonateSubject: "admin@octelium.com",
					Polling:            tstDirectoryProviderGoogleWorkspacePolling(),
				},
			},
		},
	}
}

func tstDirectoryProviderKeycloak(secretName string) *enterprisev1.DirectoryProvider {
	return &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_{
				Keycloak: &enterprisev1.DirectoryProvider_Spec_Keycloak{
					Url:                "https://keycloak.example.com",
					Realm:              "master",
					ClientID:           "octelium",
					ClientSecret:       tstDirectoryProviderKeycloakClientSecret(secretName),
					InsecureSkipVerify: true,
					Polling:            tstDirectoryProviderKeycloakPolling(),
				},
			},
		},
	}
}

func tstDirectoryProviderGoogleWorkspaceServiceAccount(secretName string) *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_ServiceAccount {
	return &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_ServiceAccount{
		Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_ServiceAccount_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDirectoryProviderKeycloakClientSecret(secretName string) *enterprisev1.DirectoryProvider_Spec_Keycloak_ClientSecret {
	return &enterprisev1.DirectoryProvider_Spec_Keycloak_ClientSecret{
		Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_ClientSecret_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDirectoryProviderGoogleWorkspacePolling() *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_Polling {
	return &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_Polling{
		Interval: &metav1.Duration{
			Type: &metav1.Duration_Seconds{Seconds: 60},
		},
	}
}

func tstDirectoryProviderKeycloakPolling() *enterprisev1.DirectoryProvider_Spec_Keycloak_Polling {
	return &enterprisev1.DirectoryProvider_Spec_Keycloak_Polling{
		Interval: &metav1.Duration{
			Type: &metav1.Duration_Seconds{Seconds: 60},
		},
	}
}

func tstCreateDirectoryProviderSecret(ctx context.Context, t *testing.T, srv *Server) string {
	sec, err := srv.CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return sec.Metadata.Name
}

func tstCloneDirectoryProvider(arg *enterprisev1.DirectoryProvider) *enterprisev1.DirectoryProvider {
	return &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec: arg.Spec,
		Status: &enterprisev1.DirectoryProvider_Status{
			Id:                   arg.Status.Id,
			UserRef:              arg.Status.UserRef,
			SessionRef:           arg.Status.SessionRef,
			Synchronization:      arg.Status.Synchronization,
			LastSynchronizations: arg.Status.LastSynchronizations,
		},
	}
}
