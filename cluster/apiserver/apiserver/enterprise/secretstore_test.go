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

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestSecretStore(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	item := tstCreateSecretStore(ctx, t, srv, tstGenSecretStore(tstSecretStoreKubernetesSpec()))

	{
		ret, err := srv.GetSecretStore(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)

		ret, err = srv.GetSecretStore(ctx, &metav1.GetOptions{Name: item.Metadata.Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
	}

	{
		_, err := srv.GetSecretStore(ctx, &metav1.GetOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetSecretStore(ctx, &metav1.GetOptions{Name: utilrand.GetRandomStringCanonical(8)})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		item.Status = &enterprisev1.SecretStore_Status{
			Type:  enterprisev1.SecretStore_Status_KUBERNETES,
			State: enterprisev1.SecretStore_Status_OK,
			Synchronization: &enterprisev1.SecretStore_Status_Synchronization{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
			},
			LastSynchronizations: []*enterprisev1.SecretStore_Status_Synchronization{
				{
					CreatedAt:   pbutils.Now(),
					CompletedAt: pbutils.Now(),
					State:       enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
				},
			},
		}
		item, err = srv.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)

		arg := tstCloneSecretStore(item)
		arg.Spec = tstSecretStoreAWSKMSSpec()
		arg.Status = &enterprisev1.SecretStore_Status{}

		updated, err := srv.UpdateSecretStore(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_TYPE_AWS_KMS, updated.Status.Type)
		assert.Equal(t, enterprisev1.SecretStore_Status_OK, updated.Status.State)
		assert.NotNil(t, updated.Status.Synchronization)
		assert.Len(t, updated.Status.LastSynchronizations, 1)
		assert.Equal(t, "alias/octelium", updated.Spec.GetAwsKeyManagementService().KeyID)
		item = updated
	}

	{
		item.Spec = tstSecretStoreAzureKeyVaultSpec()
		updated, err := srv.UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT, updated.Status.Type)
		assert.Equal(t, "https://vault.example.com", updated.Spec.GetAzureKeyVault().VaultURL)
		item = updated
	}

	{
		item.Spec = tstSecretStoreGCPKMSSpec()
		updated, err := srv.UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_TYPE_GCP_KMS, updated.Status.Type)
		assert.Equal(t, "octelium", updated.Spec.GetGoogleCloudKeyManagementService().Project)
		item = updated
	}

	{
		item.Spec = tstSecretStoreHashicorpVaultSpec()
		updated, err := srv.UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT, updated.Status.Type)
		assert.Equal(t, "octelium", updated.Spec.GetHashicorpVault().Key)
		item = updated
	}

	{
		item.Spec = tstSecretStoreKubernetesSpec()
		updated, err := srv.UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_KUBERNETES, updated.Status.Type)
		assert.NotNil(t, updated.Spec.GetKubernetes())
		item = updated
	}

	{
		itemList, err := srv.ListSecretStore(ctx, nil)
		assert.Nil(t, err, "%+v", err)
		assert.True(t, len(itemList.Items) > 0)
	}

	{
		itemList, err := srv.ListSecretStore(ctx, &enterprisev1.ListSecretStoreOptions{})
		assert.Nil(t, err, "%+v", err)
		found := false
		for _, i := range itemList.Items {
			if i.Metadata.Uid == item.Metadata.Uid {
				found = true
			}
		}
		assert.True(t, found)
	}
}

func TestUpdateSecretStore(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	item := tstCreateSecretStore(ctx, t, srv, tstGenSecretStore(tstSecretStoreKubernetesSpec()))

	{
		_, err := srv.UpdateSecretStore(ctx, &enterprisev1.SecretStore{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpdateSecretStore(ctx, &enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec:     tstSecretStoreKubernetesSpec(),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		arg := tstCloneSecretStore(item)
		arg.Spec = nil
		_, err := srv.UpdateSecretStore(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		arg := tstCloneSecretStore(item)
		arg.Spec = &enterprisev1.SecretStore_Spec{}
		_, err := srv.UpdateSecretStore(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		arg := tstCloneSecretStore(item)
		arg.Spec = tstSecretStoreAWSKMSSpec()
		arg.Spec.GetAwsKeyManagementService().KeyID = ""
		_, err := srv.UpdateSecretStore(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		arg := tstCloneSecretStore(item)
		arg.Spec = tstSecretStoreAWSKMSSpec()
		updated, err := srv.UpdateSecretStore(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.SecretStore_Status_TYPE_AWS_KMS, updated.Status.Type)
	}
}

func TestSynchronizeSecretStore(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	item := tstCreateSecretStore(ctx, t, srv, tstGenSecretStore(tstSecretStoreKubernetesSpec()))

	{
		_, err := srv.SynchronizeSecretStore(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SynchronizeSecretStore(ctx, &enterprisev1.SynchronizeSecretStoreRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SynchronizeSecretStore(ctx, &enterprisev1.SynchronizeSecretStoreRequest{
			SecretStoreRef: &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.SynchronizeSecretStore(ctx, &enterprisev1.SynchronizeSecretStoreRequest{
			SecretStoreRef: umetav1.GetObjectReference(item),
		})
		assert.Nil(t, err, "%+v", err)

		ret, err := srv.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, ret.Status.Synchronization)
		assert.NotNil(t, ret.Status.Synchronization.CreatedAt)
		assert.Equal(t, enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED, ret.Status.Synchronization.State)
		item = ret
	}

	{
		item.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			CreatedAt: pbutils.Now(),
			State:     enterprisev1.SecretStore_Status_Synchronization_SYNCING,
		}
		_, err := srv.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SynchronizeSecretStore(ctx, &enterprisev1.SynchronizeSecretStoreRequest{
			SecretStoreRef: umetav1.GetObjectReference(item),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{

		item, err := srv.octeliumC.EnterpriseC().GetSecretStore(ctx, apivalidation.ObjectToRGetOptions(item))
		assert.Nil(t, err)
		item.Status.Synchronization = nil
		_, err = srv.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SynchronizeSecretStore(ctx, &enterprisev1.SynchronizeSecretStoreRequest{
			SecretStoreRef: umetav1.GetObjectReference(item),
		})
		assert.Nil(t, err, "%+v", err)
	}
}

func TestValidateSecretStore(t *testing.T) {
	invalids := []*enterprisev1.SecretStore{
		nil,
		{},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		},
		{
			Spec: tstSecretStoreKubernetesSpec(),
		},
		{
			Metadata: &metav1.Metadata{Name: strings.Repeat("a", 256)},
			Spec:     tstSecretStoreKubernetesSpec(),
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec:     &enterprisev1.SecretStore_Spec{},
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_AzureKeyVault_{},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_{},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_AwsKeyManagementService{},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_Kubernetes_{},
			},
		},
	}

	for _, invalid := range invalids {
		err := validateSecretStore(invalid)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	valids := []*enterprisev1.SecretStore{
		tstGenSecretStore(tstSecretStoreAzureKeyVaultSpec()),
		tstGenSecretStore(tstSecretStoreHashicorpVaultSpec()),
		tstGenSecretStore(tstSecretStoreGCPKMSSpec()),
		tstGenSecretStore(tstSecretStoreAWSKMSSpec()),
		tstGenSecretStore(tstSecretStoreKubernetesSpec()),
	}

	for _, valid := range valids {
		err := validateSecretStore(valid)
		assert.Nil(t, err, "%+v", err)
	}
}

func TestValidateSecretStoreSpec(t *testing.T) {
	longValue := strings.Repeat("a", maxSecretStoreStringBytes+1)

	invalids := []*enterprisev1.SecretStore_Spec{
		nil,
		{},
		tstSecretStoreAzureKeyVaultSpecWith("", "tenant-id", "https://vault.example.com", "octelium"),
		tstSecretStoreAzureKeyVaultSpecWith("client-id", "", "https://vault.example.com", "octelium"),
		tstSecretStoreAzureKeyVaultSpecWith("client-id", "tenant-id", "vault.example.com", "octelium"),
		tstSecretStoreAzureKeyVaultSpecWith("client-id", "tenant-id", "ftp://vault.example.com", "octelium"),
		tstSecretStoreAzureKeyVaultSpecWith("client-id", "tenant-id", "https://vault.example.com", ""),
		tstSecretStoreAzureKeyVaultSpecWith(longValue, "tenant-id", "https://vault.example.com", "octelium"),
		tstSecretStoreHashicorpVaultSpecWith("", "octelium", "octelium"),
		tstSecretStoreHashicorpVaultSpecWith("vault.example.com", "octelium", "octelium"),
		tstSecretStoreHashicorpVaultSpecWith("https://vault.example.com", "", "octelium"),
		tstSecretStoreHashicorpVaultSpecWith("https://vault.example.com", "octelium", ""),
		tstSecretStoreHashicorpVaultSpecWith("https://vault.example.com", longValue, "octelium"),
		tstSecretStoreGCPKMSSpecWith("", "global", "ring", "key"),
		tstSecretStoreGCPKMSSpecWith("project", "", "ring", "key"),
		tstSecretStoreGCPKMSSpecWith("project", "global", "", "key"),
		tstSecretStoreGCPKMSSpecWith("project", "global", "ring", ""),
		tstSecretStoreGCPKMSSpecWith("project", "global", "ring", longValue),
		tstSecretStoreAWSKMSSpecWith("", "us-east-1", ""),
		tstSecretStoreAWSKMSSpecWith("alias/octelium", "", ""),
		tstSecretStoreAWSKMSSpecWith("alias/octelium", "us-east-1", "bad-role"),
		tstSecretStoreAWSKMSSpecWith("alias/octelium", "us-east-1", longValue),
	}

	for _, invalid := range invalids {
		err := validateSecretStoreSpec(invalid)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	valids := []*enterprisev1.SecretStore_Spec{
		tstSecretStoreAzureKeyVaultSpec(),
		tstSecretStoreHashicorpVaultSpec(),
		tstSecretStoreGCPKMSSpec(),
		tstSecretStoreAWSKMSSpec(),
		tstSecretStoreAWSKMSSpecWith("alias/octelium", "us-east-1", ""),
		tstSecretStoreKubernetesSpec(),
	}

	for _, valid := range valids {
		err := validateSecretStoreSpec(valid)
		assert.Nil(t, err, "%+v", err)
	}
}

func TestSecretStoreStatusType(t *testing.T) {
	tcs := []struct {
		spec *enterprisev1.SecretStore_Spec
		typ  enterprisev1.SecretStore_Status_Type
	}{
		{
			spec: tstSecretStoreAzureKeyVaultSpec(),
			typ:  enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT,
		},
		{
			spec: tstSecretStoreHashicorpVaultSpec(),
			typ:  enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT,
		},
		{
			spec: tstSecretStoreGCPKMSSpec(),
			typ:  enterprisev1.SecretStore_Status_TYPE_GCP_KMS,
		},
		{
			spec: tstSecretStoreAWSKMSSpec(),
			typ:  enterprisev1.SecretStore_Status_TYPE_AWS_KMS,
		},
		{
			spec: tstSecretStoreKubernetesSpec(),
			typ:  enterprisev1.SecretStore_Status_KUBERNETES,
		},
	}

	for _, tc := range tcs {
		typ, err := getSecretStoreStatusType(tc.spec)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, tc.typ, typ)
	}

	_, err := getSecretStoreStatusType(&enterprisev1.SecretStore_Spec{})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func tstCreateSecretStore(ctx context.Context, t *testing.T, srv *Server, item *enterprisev1.SecretStore) *enterprisev1.SecretStore {
	ret, err := srv.octeliumC.EnterpriseC().CreateSecretStore(ctx, item)
	assert.Nil(t, err, "%+v", err)
	return ret
}

func tstGenSecretStore(spec *enterprisev1.SecretStore_Spec) *enterprisev1.SecretStore {
	return &enterprisev1.SecretStore{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: spec,
		Status: &enterprisev1.SecretStore_Status{
			State: enterprisev1.SecretStore_Status_OK,
		},
	}
}

func tstCloneSecretStore(arg *enterprisev1.SecretStore) *enterprisev1.SecretStore {
	ret := &enterprisev1.SecretStore{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec: arg.Spec,
	}
	if arg.Status != nil {
		ret.Status = &enterprisev1.SecretStore_Status{
			Type:                 arg.Status.Type,
			State:                arg.Status.State,
			Synchronization:      arg.Status.Synchronization,
			LastSynchronizations: arg.Status.LastSynchronizations,
		}
	}
	return ret
}

func tstSecretStoreAzureKeyVaultSpec() *enterprisev1.SecretStore_Spec {
	return tstSecretStoreAzureKeyVaultSpecWith("client-id", "tenant-id", "https://vault.example.com", "octelium")
}

func tstSecretStoreAzureKeyVaultSpecWith(clientID string, tenantID string, vaultURL string, key string) *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_AzureKeyVault_{
			AzureKeyVault: &enterprisev1.SecretStore_Spec_AzureKeyVault{
				ClientID: clientID,
				TenantID: tenantID,
				VaultURL: vaultURL,
				Key:      key,
			},
		},
	}
}

func tstSecretStoreHashicorpVaultSpec() *enterprisev1.SecretStore_Spec {
	return tstSecretStoreHashicorpVaultSpecWith("https://vault.example.com", "octelium", "octelium")
}

func tstSecretStoreHashicorpVaultSpecWith(address string, role string, key string) *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{
			HashicorpVault: &enterprisev1.SecretStore_Spec_HashicorpVault{
				Address: address,
				Role:    role,
				Key:     key,
			},
		},
	}
}

func tstSecretStoreGCPKMSSpec() *enterprisev1.SecretStore_Spec {
	return tstSecretStoreGCPKMSSpecWith("octelium", "global", "octelium", "octelium")
}

func tstSecretStoreGCPKMSSpecWith(project string, location string, keyRing string, key string) *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_{
			GoogleCloudKeyManagementService: &enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService{
				Project:  project,
				Location: location,
				KeyRing:  keyRing,
				Key:      key,
			},
		},
	}
}

func tstSecretStoreAWSKMSSpec() *enterprisev1.SecretStore_Spec {
	return tstSecretStoreAWSKMSSpecWith("alias/octelium", "us-east-1", "arn:aws:iam::123456789012:role/octelium")
}

func tstSecretStoreAWSKMSSpecWith(keyID string, region string, roleARN string) *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_AwsKeyManagementService{
			AwsKeyManagementService: &enterprisev1.SecretStore_Spec_AWSKeyManagementService{
				KeyID:   keyID,
				Region:  region,
				RoleARN: roleARN,
			},
		},
	}
}

func tstSecretStoreKubernetesSpec() *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
			Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
		},
	}
}
