// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestInsertResource(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	srvCore := &srvCore{
		s: srv,
	}

	{
		rsc := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.User_Spec{
				Type:  corev1.User_Spec_HUMAN,
				Email: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringAlphabetLC(8)),
			},
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)

		{
			rows, err := srv.db.QueryContext(ctx,
				fmt.Sprintf(`SELECT rsc FROM resources WHERE uid = '%s'`, rsc.Metadata.Uid),
			)
			assert.Nil(t, err)

			for rows.Next() {

				rscMap := make(map[string]any)

				err = rows.Scan(&rscMap)
				assert.Nil(t, err)
				res := &corev1.User{}

				err = pbutils.UnmarshalFromMap(rscMap, res)
				assert.Nil(t, err)

				assert.True(t, pbutils.IsEqual(rsc, res))
			}
		}

		rsc.Metadata.ResourceVersion = vutils.UUIDv7()

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)

		{
			rows, err := srv.db.QueryContext(ctx,
				fmt.Sprintf(`SELECT rsc FROM resources WHERE uid = '%s'`, rsc.Metadata.Uid),
			)
			assert.Nil(t, err)

			for rows.Next() {

				rscMap := make(map[string]any)

				err = rows.Scan(&rscMap)
				assert.Nil(t, err)
				res := &corev1.User{}

				err = pbutils.UnmarshalFromMap(rscMap, res)
				assert.Nil(t, err)

				assert.True(t, pbutils.IsEqual(rsc, res))
			}
		}

		{
			res, err := srvCore.ListUser(ctx, &vcorev1.ListUserOptions{})
			assert.Nil(t, err)
			assert.True(t, len(res.Items) == 1)
			assert.True(t, pbutils.IsEqual(rsc, res.Items[0]))
		}
		err = srv.removeResource(ctx, rsc)
		assert.Nil(t, err)

		{
			rows, err := srv.db.QueryContext(ctx,
				fmt.Sprintf(`SELECT COUNT(*) FROM resources WHERE uid = '%s'`, rsc.Metadata.Uid),
			)
			assert.Nil(t, err)

			for rows.Next() {

				var count int

				err = rows.Scan(&count)
				assert.Nil(t, err)

				assert.True(t, count == 0)
			}
		}

		err = srv.removeResource(ctx, rsc)
		assert.Nil(t, err)

		{
			rows, err := srv.db.QueryContext(ctx,
				fmt.Sprintf(`SELECT COUNT(*) FROM resources WHERE uid = '%s'`, rsc.Metadata.Uid),
			)
			assert.Nil(t, err)

			for rows.Next() {

				var count int

				err = rows.Scan(&count)
				assert.Nil(t, err)

				assert.True(t, count == 0)
			}
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)

		{
			rows, err := srv.db.QueryContext(ctx,
				fmt.Sprintf(`SELECT COUNT(*) FROM resources WHERE uid = '%s'`, rsc.Metadata.Uid),
			)
			assert.Nil(t, err)

			for rows.Next() {

				var count int

				err = rows.Scan(&count)
				assert.Nil(t, err)

				assert.True(t, count == 0)
			}
		}
	}

}

func TestInsertResourceRollbackAndRetry(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	usr := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("atomic-upsert", time.Now()),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN, Email: "original@example.com"},
		Status:     &corev1.User_Status{},
	}
	usr.Metadata.Description = "original-search-value"
	require.NoError(t, env.srv.insertResource(env.ctx, usr))
	_, err := env.srv.db.ExecContext(env.ctx, `ALTER TABLE resources ALTER COLUMN name SET NOT NULL`)
	require.NoError(t, err)
	updated := proto.Clone(usr).(*corev1.User)
	updated.Metadata.Name = ""
	updated.Metadata.Description = "replacement-search-value"
	updated.Metadata.ResourceVersion = vutils.UUIDv7()
	updated.Spec.Email = "replacement@example.com"
	require.Error(t, env.srv.insertResource(env.ctx, updated))

	client := &srvCore{s: env.srv}
	for _, tt := range []struct {
		query string
		count int
	}{
		{query: "original-search-value", count: 1},
		{query: "replacement-search-value"},
	} {
		res, err := client.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{Query: tt.query},
		})
		require.NoError(t, err)
		require.Len(t, res.Items, tt.count)
		if tt.count != 0 {
			assert.True(t, proto.Equal(usr, res.Items[0]))
		}
	}
	updated.Metadata.Name = usr.Metadata.Name
	require.NoError(t, env.srv.insertResource(env.ctx, updated))
	res, err := client.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{Query: "replacement-search-value"},
	})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.True(t, proto.Equal(updated, res.Items[0]))
	assert.Equal(t, 1, getRscStoreResourceCount(t, env, ucorev1.KindUser))
}

func TestResourceRemovalAndTombstoneExpiry(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	for _, item := range []umetav1.ResourceObjectI{nil, &corev1.User{}, &corev1.User{Metadata: &metav1.Metadata{}}} {
		require.NoError(t, env.srv.removeResource(env.ctx, item))
		assert.False(t, env.srv.isRemovedResource(item))
	}
	usr := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("removed-user", time.Now()),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
	}
	require.NoError(t, env.srv.insertResource(env.ctx, usr))
	for range 2 {
		require.NoError(t, env.srv.removeResource(env.ctx, usr))
		assert.Zero(t, getRscStoreResourceCount(t, env, ucorev1.KindUser))
		assert.True(t, env.srv.isRemovedResource(usr))
	}
	_, expiry, ok := env.srv.removedUIDs.GetWithExpiration(usr.Metadata.Uid)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(removedResourceTTL), expiry, time.Second)
	require.NoError(t, env.srv.insertResource(env.ctx, usr))
	assert.Zero(t, getRscStoreResourceCount(t, env, ucorev1.KindUser))

	env.srv.removedUIDs.Set(usr.Metadata.Uid, struct{}{}, time.Nanosecond)
	require.Eventually(t, func() bool { return !env.srv.isRemovedResource(usr) }, time.Second, time.Millisecond)
	usr.Metadata.ResourceVersion = vutils.UUIDv7()
	require.NoError(t, env.srv.insertResource(env.ctx, usr))
	assert.Equal(t, 1, getRscStoreResourceCount(t, env, ucorev1.KindUser))
	assert.Equal(t, usr.Metadata.ResourceVersion, getStoredRscStoreResourceVersion(t, env, usr.Metadata.Uid))
}
