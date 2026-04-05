// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestSync(t *testing.T) {
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

	srv, err := newServer(ctx, fakeC.OcteliumC, db)
	assert.Nil(t, err)

	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		assert.False(t, srv.shouldSync(new, old))
	}
	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		new.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{}
		assert.False(t, srv.shouldSync(new, old))
	}
	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		new.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			State: enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
		}
		assert.False(t, srv.shouldSync(new, old))
	}
	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		new.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			State: enterprisev1.SecretStore_Status_Synchronization_FAILED,
		}
		assert.False(t, srv.shouldSync(new, old))
	}
	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		new.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			State: enterprisev1.SecretStore_Status_Synchronization_SYNCING,
		}
		assert.False(t, srv.shouldSync(new, old))
	}
	{
		old := &enterprisev1.SecretStore{
			Status: &enterprisev1.SecretStore_Status{},
		}
		new := pbutils.Clone(old).(*enterprisev1.SecretStore)
		new.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			State: enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED,
		}
		assert.True(t, srv.shouldSync(new, old))
	}

	{
		assert.True(t, len(srv.deks.dekMap) == 0)

		getDEKCiphertext := func(uid string) []byte {
			filters := []exp.Expression{
				goqu.C("uid").Eq(uid),
			}
			ds := goqu.From(dekTable).Where(filters...).Select("key_uid", "key_version", "ciphertext")
			sqln, sqlargs, err := ds.ToSQL()
			assert.Nil(t, err)

			var ciphertext []byte
			var keyUID string
			var keyVersion string

			err = srv.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&keyUID, &keyVersion, &ciphertext)
			assert.Nil(t, err)

			return ciphertext
		}

		sec1 := utilrand.GetRandomBytesMust(32)
		os.Setenv("OCTELIUM_TEST_SS_SECRET", base64.StdEncoding.EncodeToString(sec1))

		err = srv.doCreateDEK(ctx)
		assert.Nil(t, err)

		err = srv.setDEKMap(ctx)
		assert.Nil(t, err)

		assert.True(t, len(srv.deks.dekMap) == 1)

		dek1, err := srv.chooseDEK(ctx)
		assert.Nil(t, err)

		cipher1 := getDEKCiphertext(dek1.uid)

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
				Synchronization: &enterprisev1.SecretStore_Status_Synchronization{
					CreatedAt: pbutils.Now(),
					State:     enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED,
				},
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_KUBERNETES,
			},
		})
		assert.Nil(t, err)

		sec2 := utilrand.GetRandomBytesMust(32)
		os.Setenv("OCTELIUM_TEST_SS_SECRET", base64.StdEncoding.EncodeToString(sec2))

		err = srv.doSync(ctx, ss)
		assert.Nil(t, err)

		ss, err = srv.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
			Uid: ss.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Nil(t, ss.Status.Synchronization)
		assert.True(t, len(ss.Status.LastSynchronizations) == 1)
		assert.Equal(t, enterprisev1.SecretStore_Status_OK, ss.Status.State)

		last := ss.Status.LastSynchronizations[0]
		assert.Equal(t, enterprisev1.SecretStore_Status_Synchronization_SUCCESS, last.State)
		assert.NotNil(t, last.CompletedAt)

		err = srv.setDEKMap(ctx)
		assert.Nil(t, err)

		dek11, err := srv.chooseDEK(ctx)
		assert.Nil(t, err)

		cipher11 := getDEKCiphertext(dek11.uid)

		assert.Equal(t, dek1.key, dek11.key)
		assert.Equal(t, dek1.uid, dek11.uid)

		assert.NotNil(t, cipher1)
		assert.NotNil(t, cipher11)
		assert.NotEqual(t, cipher1, cipher11)
	}
}
