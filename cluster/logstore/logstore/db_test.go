package logstore

import (
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func restoreLogStoreTestDatabase(t *testing.T, database string, withWAL bool) {
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

func TestOpenLogStoreDBRecreatesADuckDBV155Database(t *testing.T) {
	for _, name := range []string{"Checkpointed", "WAL"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "store.db")
			restoreLogStoreTestDatabase(t, database, name == "WAL")
			before, err := os.Stat(database)
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(database+".tmp", 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(database+".tmp", "spill"), []byte("old spill"), 0o600))
			keep := filepath.Join(filepath.Dir(database), "keep.db")
			require.NoError(t, os.WriteFile(keep, []byte("keep"), 0o600))

			dsn := database + "?threads=1&memory_limit=64MB&storage_compatibility_version=v1.5.5"
			db, err := openLogStoreDB(ctx, dsn)
			require.NoError(t, err)
			srv := &Server{db: db, dbPath: database}
			t.Cleanup(func() { _ = srv.db.Close() })
			after, err := os.Stat(database)
			require.NoError(t, err)
			assert.Less(t, after.Size(), before.Size())
			assert.NoFileExists(t, database+".wal")
			assert.NoDirExists(t, database+".tmp")
			require.NoError(t, srv.initDB(ctx))
			for _, table := range getLogTables() {
				assert.Zero(t, getTableCount(t, srv, table), table)
			}

			now := time.Now().UTC().Truncate(time.Microsecond)
			logs := []struct {
				table  string
				insert func([]byte) error
				item   proto.Message
			}{
				{
					table: "access_logs", insert: srv.insertAccessLog,
					item: newAccessLog(&accessLogOptions{
						CreatedAt: now, Status: corev1.AccessLog_Entry_Common_ALLOWED,
						UserRef: testRef("upgraded-user"),
					}),
				},
				{table: "component_logs", insert: srv.insertComponentLog,
					item: newComponentLog(now, corev1.ComponentLog_Entry_INFO, "upgraded")},
				{table: "authentication_logs", insert: srv.insertAuthenticationLog,
					item: newAuthenticationLog(&authenticationLogOptions{CreatedAt: now})},
				{table: "audit_logs", insert: srv.insertAuditLog,
					item: newAuditLog(&auditLogOptions{CreatedAt: now, Operation: "CreateUser"})},
			}
			for _, log := range logs {
				require.NoError(t, log.insert(marshalLog(t, log.item)))
			}

			for i := 0; i < 3; i++ {
				var storageVersion, threads, memoryLimit string
				require.NoError(t, srv.db.QueryRow(`SELECT tags['storage_version'] FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&storageVersion))
				assert.Equal(t, "v2.0.0+", storageVersion)
				require.NoError(t, srv.db.QueryRow(`SELECT current_setting('threads'), current_setting('memory_limit')`).Scan(&threads, &memoryLimit))
				assert.Equal(t, "1", threads)
				assert.Equal(t, "61.0 MiB", memoryLimit)
				for _, log := range logs {
					assert.Equal(t, 1, getTableCount(t, srv, log.table), log.table)
					var stored map[string]any
					require.NoError(t, srv.db.QueryRow(`SELECT rsc FROM `+log.table).Scan(&stored))
					item := log.item.ProtoReflect().Type().New().Interface()
					require.NoError(t, pbutils.UnmarshalFromMap(stored, item))
					assert.True(t, proto.Equal(log.item, item), log.table)
				}
				resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
					UserRef: testRef("upgraded-user"),
					Common:  &vmetav1.CommonListOptions{ItemsPerPage: 50},
				})
				require.NoError(t, err)
				require.Len(t, resp.Items, 1)
				assert.True(t, proto.Equal(logs[0].item, resp.Items[0]))
				if i < 2 {
					require.NoError(t, srv.db.Close())
					srv.db, err = openLogStoreDB(ctx, dsn)
					require.NoError(t, err)
					require.NoError(t, srv.initDB(ctx))
				}
			}
			data, err := os.ReadFile(keep)
			require.NoError(t, err)
			assert.Equal(t, []byte("keep"), data)
		})
	}
}

func TestOpenLogStoreDBKeepsAnExistingV2Database(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("duckdb", database+"?storage_compatibility_version=v2.0.0")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE access_logs (rsc JSON)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO access_logs VALUES ('{"preserve":true}')`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = openLogStoreDB(context.Background(), database)
	require.NoError(t, err)
	defer db.Close()
	srv := &Server{db: db}
	require.NoError(t, srv.initDB(context.Background()))
	var preserved bool
	require.NoError(t, db.QueryRow(`SELECT CAST(json_extract(rsc, '$.preserve') AS BOOLEAN) FROM access_logs`).Scan(&preserved))
	assert.True(t, preserved)
}

func TestOpenLogStoreDBKeepsAV2WAL(t *testing.T) {
	if database := os.Getenv("OCTELIUM_LOGSTORE_TEST_WAL"); database != "" {
		db, err := openLogStoreDB(context.Background(), database+"?threads=1")
		require.NoError(t, err)
		srv := &Server{db: db}
		require.NoError(t, srv.initDB(context.Background()))
		_, err = db.Exec(`SET wal_autocheckpoint='1GB'`)
		require.NoError(t, err)
		_, err = db.Exec(`CHECKPOINT`)
		require.NoError(t, err)
		require.NoError(t, srv.insertComponentLog(marshalLog(t,
			newComponentLog(time.Now().UTC(), corev1.ComponentLog_Entry_INFO, "preserved WAL"))))
		os.Exit(0)
	}

	database := filepath.Join(t.TempDir(), "store.db")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOpenLogStoreDBKeepsAV2WAL$")
	cmd.Env = append(os.Environ(), "OCTELIUM_LOGSTORE_TEST_WAL="+database)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.FileExists(t, database+".wal")

	db, err := openLogStoreDB(context.Background(), database+"?threads=1")
	require.NoError(t, err)
	defer db.Close()
	srv := &Server{db: db}
	require.NoError(t, srv.initDB(context.Background()))
	assert.Equal(t, 1, getTableCount(t, srv, "component_logs"))
	var stored map[string]any
	require.NoError(t, db.QueryRow(`SELECT rsc FROM component_logs`).Scan(&stored))
	item := &corev1.ComponentLog{}
	require.NoError(t, pbutils.UnmarshalFromMap(stored, item))
	assert.Equal(t, corev1.ComponentLog_Entry_INFO, item.Entry.Level)
	assert.Equal(t, "preserved WAL", item.Entry.Message)
}

func TestOpenLogStoreDBKeepsUnrelatedStorage(t *testing.T) {
	for _, name := range []string{"OtherStore", "Mixed", "WrongColumn", "OtherSchema", "EmptySchema", "View"} {
		t.Run(name, func(t *testing.T) {
			database := filepath.Join(t.TempDir(), "store.db")
			legacyDSN := database + "?storage_compatibility_version=v1.5.5"
			db, err := sql.Open("duckdb", legacyDSN)
			require.NoError(t, err)
			defer db.Close()
			queries := []string{}
			switch name {
			case "OtherStore":
				queries = append(queries, `CREATE TABLE resources (uid VARCHAR)`)
			case "Mixed":
				queries = append(queries, `CREATE TABLE access_logs (rsc JSON)`, `CREATE TABLE resources (uid VARCHAR)`)
			case "WrongColumn":
				queries = append(queries, `CREATE TABLE access_logs (rsc VARCHAR)`)
			case "OtherSchema":
				queries = append(queries, `CREATE SCHEMA other`, `CREATE TABLE other.resources (uid VARCHAR)`)
			case "EmptySchema":
				queries = append(queries, `CREATE TABLE access_logs (rsc JSON)`, `CREATE SCHEMA other`)
			case "View":
				queries = append(queries, `CREATE VIEW access_logs AS SELECT '{}'::JSON AS rsc`)
			}
			for _, query := range queries {
				_, err = db.Exec(query)
				require.NoError(t, err)
			}
			require.NoError(t, db.Close())
			before, err := os.ReadFile(database)
			require.NoError(t, err)

			opened, err := openLogStoreDB(context.Background(), database)
			require.ErrorContains(t, err, "does not contain logstore storage")
			assert.Nil(t, opened)
			after, err := os.ReadFile(database)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestOpenLogStoreDBKeepsUnreadableStorage(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	data := []byte("unrecognized database")
	wal := []byte("unrecognized WAL")
	require.NoError(t, os.WriteFile(database, data, 0o600))
	require.NoError(t, os.WriteFile(database+".wal", wal, 0o600))

	db, err := openLogStoreDB(context.Background(), database)
	require.Error(t, err)
	assert.Nil(t, db)
	stored, err := os.ReadFile(database)
	require.NoError(t, err)
	assert.Equal(t, data, stored)
	stored, err = os.ReadFile(database + ".wal")
	require.NoError(t, err)
	assert.Equal(t, wal, stored)
}

func TestOpenLogStoreDBKeepsAnOpenDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("duckdb", database+"?storage_compatibility_version=v1.5.5")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE access_logs (rsc JSON)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO access_logs VALUES ('{"preserve":true}')`)
	require.NoError(t, err)

	opened, err := openLogStoreDB(context.Background(), database)
	if opened != nil {
		defer opened.Close()
	}
	require.Error(t, err)
	assert.Nil(t, opened)
	assert.FileExists(t, database)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM access_logs`).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestOpenLogStoreDBKeepsStorageOnCancellation(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	restoreLogStoreTestDatabase(t, database, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := openLogStoreDB(ctx, database)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, db)

	db, err = sql.Open("duckdb", database+"?storage_compatibility_version=v1.5.5")
	require.NoError(t, err)
	defer db.Close()
	for _, table := range getLogTables() {
		var count int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
		assert.Equal(t, 1, count, table)
	}
}

func TestOpenLogStoreDBRecreatesAnEmptyOldDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("duckdb", database+"?storage_compatibility_version=v1.5.5")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = openLogStoreDB(context.Background(), database)
	require.NoError(t, err)
	defer db.Close()
	var storageVersion string
	require.NoError(t, db.QueryRow(`SELECT tags['storage_version'] FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&storageVersion))
	assert.Equal(t, "v2.0.0+", storageVersion)
}

func TestOpenLogStoreDBInMemory(t *testing.T) {
	for _, dsn := range []string{"", ":memory:"} {
		t.Run(dsn, func(t *testing.T) {
			db, err := openLogStoreDB(context.Background(), dsn)
			require.NoError(t, err)
			defer db.Close()
			srv := &Server{db: db}
			require.NoError(t, srv.initDB(context.Background()))
			for _, table := range getLogTables() {
				assert.Zero(t, getTableCount(t, srv, table), table)
			}
		})
	}
}

func TestRemoveLogStoreDatabaseRemovesOnlyTheDatabaseFiles(t *testing.T) {
	dir := t.TempDir()
	database := filepath.Join(dir, "store.db")
	for _, path := range []string{database, database + ".wal", filepath.Join(dir, "keep.db")} {
		require.NoError(t, os.WriteFile(path, []byte("db"), 0o600))
	}
	require.NoError(t, os.MkdirAll(database+".tmp", 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(database+".tmp", "spill"), []byte("tmp"), 0o600))
	require.NoError(t, removeLogStoreDatabase(database))
	assert.NoFileExists(t, database)
	assert.NoFileExists(t, database+".wal")
	assert.NoDirExists(t, database+".tmp")
	assert.FileExists(t, filepath.Join(dir, "keep.db"))
	require.NoError(t, removeLogStoreDatabase(database))
	require.NoError(t, removeLogStoreDatabase(""))
}
