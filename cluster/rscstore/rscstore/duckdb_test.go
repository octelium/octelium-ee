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
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
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
				// `SELECT uid, rsc, score FROM (SELECT *, fts_main_resources.match_bm25(uid, 'User', fields := 'rsc_strr') AS score FROM resources) sq WHERE score IS NOT NULL ORDER BY score DESC`,
				// `with fts as (select *, fts_main_resources.match_bm25(uid, 'user', fields := 'rsc_str') as score FROM resources) SELECT uid, rsc, score FROM fts WHERE score IS NOT NULL ORDER BY score DESC`,
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
				// `SELECT uid, rsc, score FROM (SELECT *, fts_main_resources.match_bm25(uid, 'User', fields := 'rsc_strr') AS score FROM resources) sq WHERE score IS NOT NULL ORDER BY score DESC`,
				// `with fts as (select *, fts_main_resources.match_bm25(uid, 'user', fields := 'rsc_str') as score FROM resources) SELECT uid, rsc, score FROM fts WHERE score IS NOT NULL ORDER BY score DESC`,
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

			err := srv.recreateFTSIndex(ctx)
			assert.Nil(t, err)
		}

		{
			rows, err := srv.db.QueryContext(ctx,
				// `SELECT uid, rsc, score FROM (SELECT *, fts_main_resources.match_bm25(uid, 'User', fields := 'rsc_strr') AS score FROM resources) sq WHERE score IS NOT NULL ORDER BY score DESC`,
				`with fts as (select *, fts_main_resources.match_bm25(uid, 'example', fields := 'rsc_str') as score FROM resources) SELECT uid, rsc, score FROM fts WHERE score is NOT NULL ORDER BY score DESC`,
				// fmt.Sprintf(`SELECT uid, rsc, fts_main_resources.match_bm25(uid, '%s') AS score FROM resources WHERE score IS NOT NULL ORDER BY score DESC`, "human"),
			)
			assert.Nil(t, err)

			for rows.Next() {
				var uid string
				rsc := make(map[string]any)
				var score float64
				err = rows.Scan(&uid, &rsc, &score)
				assert.Nil(t, err)
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
				// `SELECT uid, rsc, score FROM (SELECT *, fts_main_resources.match_bm25(uid, 'User', fields := 'rsc_strr') AS score FROM resources) sq WHERE score IS NOT NULL ORDER BY score DESC`,
				// `with fts as (select *, fts_main_resources.match_bm25(uid, 'user', fields := 'rsc_str') as score FROM resources) SELECT uid, rsc, score FROM fts WHERE score IS NOT NULL ORDER BY score DESC`,
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
				// `SELECT uid, rsc, score FROM (SELECT *, fts_main_resources.match_bm25(uid, 'User', fields := 'rsc_strr') AS score FROM resources) sq WHERE score IS NOT NULL ORDER BY score DESC`,
				// `with fts as (select *, fts_main_resources.match_bm25(uid, 'user', fields := 'rsc_str') as score FROM resources) SELECT uid, rsc, score FROM fts WHERE score IS NOT NULL ORDER BY score DESC`,
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
