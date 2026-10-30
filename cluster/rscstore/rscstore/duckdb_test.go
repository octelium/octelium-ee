// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func restoreRscStoreTestDatabase(t *testing.T, database string, withWAL bool) {
	t.Helper()

	suffixes := []string{""}
	if withWAL {
		suffixes = append(suffixes, ".wal")
	}
	for _, suffix := range suffixes {
		file, err := os.Open("testdata/duckdb-v1.5.5.db" + suffix + ".gz")
		require.NoError(t, err)
		defer file.Close()
		reader, err := gzip.NewReader(file)
		require.NoError(t, err)
		defer reader.Close()
		dst, err := os.OpenFile(database+suffix, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		require.NoError(t, err)
		_, err = io.Copy(dst, reader)
		require.NoError(t, dst.Close())
		require.NoError(t, err)
	}
}

func TestOpenRscStoreDBRecreatesADuckDBV155Database(t *testing.T) {
	for _, name := range []string{"Checkpointed", "WAL"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "store.db")
			restoreRscStoreTestDatabase(t, database, name == "WAL")
			before, err := os.Stat(database)
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(database+".tmp", 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(database+".tmp", "spill"), []byte("old spill"), 0o600))

			db, err := openRscStoreDB(ctx, database+"?threads=1")
			require.NoError(t, err)
			srv := &Server{db: db}
			t.Cleanup(func() { _ = srv.db.Close() })
			after, err := os.Stat(database)
			require.NoError(t, err)
			assert.Less(t, after.Size(), before.Size())
			assert.NoFileExists(t, database+".wal")
			assert.NoDirExists(t, database+".tmp")
			require.NoError(t, srv.initDB(ctx))
			for _, table := range []string{"resources", "versioned_resources"} {
				var count int
				require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
				assert.Zero(t, count, table)
			}
			var legacySchemas int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = 'fts_main_resources'`).Scan(&legacySchemas))
			assert.Zero(t, legacySchemas)

			usr := &corev1.User{
				ApiVersion: ucorev1.APIVersion,
				Kind:       ucorev1.KindUser,
				Metadata:   newRscStoreMetadata("rebuilt-user", time.Now().UTC()),
				Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			}
			require.NoError(t, srv.upsertResource(ctx, usr))

			for i := 0; i < 2; i++ {
				require.NoError(t, srv.db.Close())
				db, err := openRscStoreDB(ctx, database+"?threads=1")
				require.NoError(t, err)
				srv.db = db
				require.NoError(t, srv.initDB(ctx))
				var storageVersion, threads string
				require.NoError(t, db.QueryRow(`SELECT tags['storage_version'] FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&storageVersion))
				assert.Equal(t, "v2.0.0+", storageVersion)
				require.NoError(t, db.QueryRow(`SELECT current_setting('threads')`).Scan(&threads))
				assert.Equal(t, "1", threads)
				var stored map[string]any
				require.NoError(t, db.QueryRow(`SELECT rsc FROM resources WHERE uid = ?`, usr.Metadata.Uid).Scan(&stored))
				res := &corev1.User{}
				require.NoError(t, pbutils.UnmarshalFromMap(stored, res))
				assert.True(t, proto.Equal(usr, res))
				var name, userType string
				require.NoError(t, db.QueryRow(`SELECT name, spec_type FROM resources WHERE uid = ?`, usr.Metadata.Uid).Scan(&name, &userType))
				assert.Equal(t, "rebuilt-user", name)
				assert.Equal(t, "HUMAN", userType)
			}
		})
	}
}

func TestOpenRscStoreDBKeepsUnrelatedStorage(t *testing.T) {
	for _, name := range []string{"OtherStore", "Mixed"} {
		t.Run(name, func(t *testing.T) {
			database := filepath.Join(t.TempDir(), "store.db")
			legacyDSN := database + "?storage_compatibility_version=v1.5.5"
			db, err := sql.Open("duckdb", legacyDSN)
			require.NoError(t, err)
			defer db.Close()
			_, err = db.Exec(`CREATE TABLE access_logs (rsc JSON)`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO access_logs VALUES ('{"preserve":true}')`)
			require.NoError(t, err)
			if name == "Mixed" {
				_, err = db.Exec(`CREATE TABLE resources (api VARCHAR, version VARCHAR, kind VARCHAR, uid VARCHAR, resource_version VARCHAR, rsc JSON)`)
				require.NoError(t, err)
			}
			require.NoError(t, db.Close())

			opened, err := openRscStoreDB(context.Background(), database)
			require.ErrorContains(t, err, "does not contain rscstore storage")
			assert.Nil(t, opened)
			db, err = sql.Open("duckdb", legacyDSN)
			require.NoError(t, err)
			defer db.Close()
			var count int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM access_logs`).Scan(&count))
			assert.Equal(t, 1, count)
		})
	}
}

func TestOpenRscStoreDBKeepsUnreadableStorage(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	data := []byte("unrecognized database")
	require.NoError(t, os.WriteFile(database, data, 0o600))
	db, err := openRscStoreDB(context.Background(), database)
	require.Error(t, err)
	assert.Nil(t, db)
	stored, err := os.ReadFile(database)
	require.NoError(t, err)
	assert.Equal(t, data, stored)
}

func TestOpenRscStoreDBKeepsAnOpenDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("duckdb", database+"?storage_compatibility_version=v1.5.5")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE resources (api VARCHAR, version VARCHAR, kind VARCHAR, uid VARCHAR, resource_version VARCHAR, rsc JSON)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO resources (uid) VALUES ('keep')`)
	require.NoError(t, err)

	opened, err := openRscStoreDB(context.Background(), database)
	if opened != nil {
		defer opened.Close()
	}
	require.Error(t, err)
	assert.Nil(t, opened)
	assert.FileExists(t, database)
	var uid string
	require.NoError(t, db.QueryRow(`SELECT uid FROM resources`).Scan(&uid))
	assert.Equal(t, "keep", uid)
}

func TestOpenRscStoreDBRecreatesAnEmptyOldDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("duckdb", database+"?storage_compatibility_version=v1.5.5")
	require.NoError(t, err)
	require.NoError(t, db.Close())
	db, err = openRscStoreDB(context.Background(), database)
	require.NoError(t, err)
	defer db.Close()
	var storageVersion string
	require.NoError(t, db.QueryRow(`SELECT tags['storage_version'] FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&storageVersion))
	assert.Equal(t, "v2.0.0+", storageVersion)
}

func TestRemoveRscStoreDatabaseRemovesOnlyTheDatabaseFiles(t *testing.T) {
	dir := t.TempDir()
	database := filepath.Join(dir, "store.db")
	for _, path := range []string{database, database + ".wal", filepath.Join(dir, "keep.db")} {
		require.NoError(t, os.WriteFile(path, []byte("db"), 0o600))
	}
	require.NoError(t, os.MkdirAll(database+".tmp", 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(database+".tmp", "spill"), []byte("tmp"), 0o600))
	require.NoError(t, removeRscStoreDatabase(database))
	assert.NoFileExists(t, database)
	assert.NoFileExists(t, database+".wal")
	assert.NoDirExists(t, database+".tmp")
	assert.FileExists(t, filepath.Join(dir, "keep.db"))
	require.NoError(t, removeRscStoreDatabase(database))
	require.NoError(t, removeRscStoreDatabase(""))
}

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
