// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package k8s

import (
	"context"
	"fmt"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	db, err := postgresutils.NewDB()
	assert.Nil(t, err)

	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	ss, err := fakeC.OcteliumC.EnterpriseC().CreateSecretStore(ctx, &enterprisev1.SecretStore{
		Metadata: &metav1.Metadata{
			Name:           fmt.Sprintf("sys-k8s-%s", vutils.GetMyRegionName()),
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec: &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		},
		Status: &enterprisev1.SecretStore_Status{
			State: enterprisev1.SecretStore_Status_OK,
			Type:  enterprisev1.SecretStore_Status_KUBERNETES,
		},
	})
	assert.Nil(t, err)

	store, err := NewStore(ctx, &stores.StoreOpts{
		SecretStore: ss,
	})
	assert.Nil(t, err)

	err = store.Initialize(ctx)
	assert.Nil(t, err)

	val, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err)

	ciphertext, err := store.Encrypt(ctx, "", val)
	assert.Nil(t, err)

	plaintext, err := store.Decrypt(ctx, "", ciphertext)
	assert.Nil(t, err)
	assert.Equal(t, val, plaintext)
	{
		p2, err := store.Decrypt(ctx, "", ciphertext)
		assert.Nil(t, err)
		assert.Equal(t, plaintext, p2)
	}
}
