// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func getTestResourceColumn(t *testing.T, name string) *resourceColumn {
	t.Helper()

	for _, column := range getResourceColumns() {
		if column.name == name {
			return column
		}
	}

	t.Fatalf("Could not find the column: %s", name)
	return nil
}

func getTestColumnValue(t *testing.T, env *rscStoreTestEnv, uid, column string) any {
	t.Helper()

	var ret any
	err := env.srv.db.QueryRowContext(env.ctx,
		fmt.Sprintf(`SELECT %s FROM resources WHERE uid = ?`, column), uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func getTestColumnList(t *testing.T, env *rscStoreTestEnv, uid, column string) []string {
	t.Helper()

	var ret sql.NullString
	err := env.srv.db.QueryRowContext(env.ctx,
		fmt.Sprintf(`SELECT list_aggregate(%s, 'string_agg', ',') FROM resources WHERE uid = ?`, column),
		uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	if !ret.Valid || ret.String == "" {
		return nil
	}

	return strings.Split(ret.String, ",")
}

func insertLegacyRscStoreResource(t *testing.T, env *rscStoreTestEnv, rsc map[string]any) {
	t.Helper()

	metadata, ok := rsc["metadata"].(map[string]any)
	assert.True(t, ok)
	if !ok {
		return
	}

	uid, _ := metadata["uid"].(string)
	resourceVersion, _ := metadata["resourceVersion"].(string)
	kind, _ := rsc["kind"].(string)

	rscJSON, err := json.Marshal(rsc)
	assert.Nil(t, err, "%+v", err)

	rscStr, err := env.srv.getRSCStr(rscJSON)
	assert.Nil(t, err, "%+v", err)

	_, err = env.srv.db.ExecContext(env.ctx, `
INSERT INTO resources
	(api, version, kind, uid, resource_version, rsc, rsc_str)
VALUES
	(?, ?, ?, ?, ?, ?, ?)`,
		ucorev1.API, ucorev1.Version, kind, uid, resourceVersion, string(rscJSON), rscStr)
	assert.Nil(t, err, "%+v", err)
}

func newTestSession(name string, createdAt time.Time, userUID, deviceUID string) *corev1.Session {
	return &corev1.Session{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindSession,
		Metadata: func() *metav1.Metadata {
			ret := newRscStoreMetadata(name, createdAt)
			ret.Tags = []string{"tag-one", "tag-two"}
			return ret
		}(),
		Spec: &corev1.Session_Spec{
			State: corev1.Session_Spec_ACTIVE,
		},
		Status: &corev1.Session_Status{
			Type:        corev1.Session_Status_CLIENT,
			IsConnected: true,
			UserRef: &metav1.ObjectReference{
				Uid: userUID, Name: "session-user", ApiVersion: ucorev1.APIVersion, Kind: ucorev1.KindUser,
			},
			DeviceRef: &metav1.ObjectReference{
				Uid: deviceUID, Name: "session-device", ApiVersion: ucorev1.APIVersion, Kind: ucorev1.KindDevice,
			},
		},
	}
}

func TestGetResourceColumns(t *testing.T) {
	names := make(map[string]struct{})
	paths := make(map[string]struct{})

	reserved := []string{"api", "version", "kind", "uid", "resource_version", "rsc", "rsc_str"}

	for _, column := range getResourceColumns() {
		assert.NotEmpty(t, column.name)
		assert.NotEmpty(t, column.path)
		assert.True(t, strings.HasPrefix(column.path, "$."), column.path)

		_, ok := names[column.name]
		assert.False(t, ok, "duplicate column name: %s", column.name)
		names[column.name] = struct{}{}

		for _, name := range reserved {
			assert.NotEqual(t, name, column.name)
		}

		switch column.kind {
		case kindVarchar, kindTimestamp, kindBoolean, kindBigint, kindVarcharList:
			_, ok := paths[column.path]
			assert.False(t, ok, "duplicate column path: %s", column.path)
			paths[column.path] = struct{}{}
		case kindKeys:
		default:
			t.Fatalf("Invalid column kind: %s", column.kind)
		}
	}

	assert.NotEmpty(t, names)
}

func TestGetResourceColumnByPath(t *testing.T) {
	name, ok := getResourceColumnNameByPath("$.metadata.createdAt")
	assert.True(t, ok)
	assert.Equal(t, colCreatedAt, name)

	name, ok = getResourceColumnNameByPath("$.status.userRef.uid")
	assert.True(t, ok)
	assert.Equal(t, colUserUID, name)

	_, ok = getResourceColumnNameByPath("$.spec.doesNotExist")
	assert.False(t, ok)

	_, ok = getResourceColumnByPath("$.spec")
	assert.False(t, ok)
}

func TestGetRefFilterExpr(t *testing.T) {
	assert.Equal(t, colUserUID, getRefFilterExpr("status.userRef", "uid"))
	assert.Equal(t, colUserName, getRefFilterExpr("status.userRef", "name"))
	assert.Equal(t, `rsc->>'$.status.doesNotExistRef.uid'`,
		getRefFilterExpr("status.doesNotExistRef", "uid"))
}

func TestInsertPopulatesResourceColumns(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	userUID := vutils.UUIDv4()
	deviceUID := vutils.UUIDv4()
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	sess := newTestSession("indexed-session", createdAt, userUID, deviceUID)
	insertRscStoreObject(t, env, sess)

	uid := sess.Metadata.Uid

	assert.Equal(t, createdAt, getTestColumnValue(t, env, uid, colCreatedAt))
	assert.Equal(t, "indexed-session", getTestColumnValue(t, env, uid, colName))
	assert.Equal(t, "CLIENT", getTestColumnValue(t, env, uid, colStatusType))
	assert.Equal(t, "ACTIVE", getTestColumnValue(t, env, uid, colSpecState))
	assert.Equal(t, true, getTestColumnValue(t, env, uid, colStatusIsConnected))
	assert.Equal(t, userUID, getTestColumnValue(t, env, uid, colUserUID))
	assert.Equal(t, "session-user", getTestColumnValue(t, env, uid, colUserName))
	assert.Equal(t, deviceUID, getTestColumnValue(t, env, uid, colDeviceUID))
	assert.Equal(t, "session-device", getTestColumnValue(t, env, uid, colDeviceName))

	assert.Equal(t, []string{"tag-one", "tag-two"}, getTestColumnList(t, env, uid, colTags))
	assert.Contains(t, getTestColumnList(t, env, uid, colSpecKeys), "state")
	assert.Contains(t, getTestColumnList(t, env, uid, colStatusKeys), "userRef")
	assert.Contains(t, getTestColumnList(t, env, uid, colStatusKeys), "isConnected")
}

func TestInsertWithMissingFieldsLeavesNullColumns(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	user := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("sparse-user", time.Now()),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status:     &corev1.User_Status{},
	}
	insertRscStoreObject(t, env, user)

	uid := user.Metadata.Uid

	assert.Equal(t, "HUMAN", getTestColumnValue(t, env, uid, colSpecType))
	assert.Nil(t, getTestColumnValue(t, env, uid, colUserUID))
	assert.Nil(t, getTestColumnValue(t, env, uid, colStatusIsConnected))
	assert.Nil(t, getTestColumnValue(t, env, uid, colStatusIssuanceExpiry))
	assert.Nil(t, getTestColumnValue(t, env, uid, colStatusManagedDevices))
	assert.Nil(t, getTestColumnList(t, env, uid, colTags))
	assert.NotNil(t, getTestColumnValue(t, env, uid, colCreatedAt))
}

func TestUpsertReplacesTheExistingResource(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	sess := newTestSession("replaced-session", time.Now(), vutils.UUIDv4(), vutils.UUIDv4())
	insertRscStoreObject(t, env, sess)

	sess.Metadata.ResourceVersion = vutils.UUIDv7()
	sess.Status.Type = corev1.Session_Status_CLIENTLESS
	sess.Status.IsConnected = false
	sess.Metadata.Tags = []string{"tag-three"}
	insertRscStoreObject(t, env, sess)

	uid := sess.Metadata.Uid

	var count int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx,
		`SELECT COUNT(*) FROM resources WHERE uid = ?`, uid).Scan(&count))
	assert.Equal(t, 1, count)

	assert.Equal(t, "CLIENTLESS", getTestColumnValue(t, env, uid, colStatusType))
	assert.Nil(t, getTestColumnValue(t, env, uid, colStatusIsConnected))
	assert.Equal(t, []string{"tag-three"}, getTestColumnList(t, env, uid, colTags))
	assert.Equal(t, sess.Metadata.ResourceVersion, getStoredRscStoreResourceVersion(t, env, uid))
}

func TestBackfillProducesTheSameValuesAsInsert(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	sess := newTestSession("backfilled-session", time.Now(), vutils.UUIDv4(), vutils.UUIDv4())
	insertRscStoreObject(t, env, sess)

	cert := &corev1.Device{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindDevice,
		Metadata:   newRscStoreMetadata("backfilled-device", time.Now()),
		Spec:       &corev1.Device_Spec{State: corev1.Device_Spec_ACTIVE},
		Status: &corev1.Device_Status{
			OsType: corev1.Device_Status_LINUX,
			UserRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(), Name: "device-user",
			},
		},
	}
	insertRscStoreObject(t, env, cert)

	columns := getResourceColumns()

	names := make([]string, 0, len(columns))
	projections := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.name)
		projections = append(projections,
			fmt.Sprintf(`COALESCE(CAST(%s AS VARCHAR), '<null>')`, column.name))
	}

	readAll := func() map[string]string {
		ret := make(map[string]string)

		rows, err := env.srv.db.QueryContext(env.ctx, fmt.Sprintf(
			`SELECT uid, concat_ws('|', %s) FROM resources ORDER BY uid`,
			strings.Join(projections, ", ")))
		assert.Nil(t, err, "%+v", err)
		defer rows.Close()

		for rows.Next() {
			var uid, values string
			assert.Nil(t, rows.Scan(&uid, &values))
			ret[uid] = values
		}
		assert.Nil(t, rows.Err())

		return ret
	}

	inserted := readAll()
	assert.Len(t, inserted, 2)

	var resets []string
	for _, name := range names {
		resets = append(resets, fmt.Sprintf("%s = NULL", name))
	}
	_, err := env.srv.db.ExecContext(env.ctx,
		fmt.Sprintf(`UPDATE resources SET %s`, strings.Join(resets, ", ")))
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, env.srv.backfillResourceColumns(env.ctx, columns))

	assert.Equal(t, inserted, readAll())
}

func TestBackfillResourceColumnsIsIdempotentAndTerminates(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	for i := 0; i < 5; i++ {
		insertLegacyRscStoreResource(t, env, newRawRscStoreResource(ucorev1.KindUser,
			fmt.Sprintf("legacy-user-%d", i), false,
			map[string]any{"type": "HUMAN"}, map[string]any{}))
	}

	columns := getResourceColumns()

	assert.Nil(t, env.srv.backfillResourceColumns(env.ctx, columns))

	var pending int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM resources WHERE %s IS NULL`, colCreatedAt)).Scan(&pending))
	assert.Zero(t, pending)

	var typed int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM resources WHERE %s = 'HUMAN'`, colSpecType)).Scan(&typed))
	assert.Equal(t, 5, typed)

	assert.Nil(t, env.srv.backfillResourceColumns(env.ctx, columns))
	assert.Nil(t, env.srv.initResourceColumns(env.ctx))

	var total int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx,
		`SELECT COUNT(*) FROM resources`).Scan(&total))
	assert.Equal(t, 5, total)
}

func TestBackfillHandlesResourcesWithoutCreatedAt(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	rsc := newRawRscStoreResource(ucorev1.KindUser, "no-created-at", false,
		map[string]any{"type": "WORKLOAD"}, map[string]any{})
	metadata := rsc["metadata"].(map[string]any)
	delete(metadata, "createdAt")

	insertLegacyRscStoreResource(t, env, rsc)

	assert.Nil(t, env.srv.backfillResourceColumns(env.ctx, getResourceColumns()))

	uid := metadata["uid"].(string)
	assert.Nil(t, getTestColumnValue(t, env, uid, colCreatedAt))
	assert.Equal(t, "WORKLOAD", getTestColumnValue(t, env, uid, colSpecType))
}

func TestUpgradeFromLegacyResourceSchema(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	_, err := env.srv.db.ExecContext(env.ctx, `DROP TABLE resources`)
	assert.Nil(t, err, "%+v", err)

	_, err = env.srv.db.ExecContext(env.ctx,
		`CREATE TABLE resources (api VARCHAR, version VARCHAR, kind VARCHAR, uid VARCHAR, resource_version VARCHAR, rsc JSON, rsc_str VARCHAR, PRIMARY KEY (uid))`)
	assert.Nil(t, err, "%+v", err)

	for i := 0; i < 12; i++ {
		userType := "HUMAN"
		if i%3 == 0 {
			userType = "WORKLOAD"
		}

		insertLegacyRscStoreResource(t, env, newRawRscStoreResource(ucorev1.KindUser,
			fmt.Sprintf("legacy-user-%02d", i), false,
			map[string]any{"type": userType, "isDisabled": i == 1},
			map[string]any{}))
	}

	insertLegacyRscStoreResource(t, env, newRawRscStoreResource(ucorev1.KindUser,
		"legacy-hidden", true, map[string]any{"type": "HUMAN"}, map[string]any{}))

	assert.Nil(t, env.srv.initDB(env.ctx))

	srvCore := &srvCore{s: env.srv}

	resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 12)
	assert.Equal(t, uint32(12), resp.ListResponseMeta.TotalCount)

	summary, err := srvCore.GetUserSummary(env.ctx, &vcorev1.GetUserSummaryRequest{})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, uint32(12), summary.TotalNumber)
	assert.Equal(t, uint32(8), summary.TotalHuman)
	assert.Equal(t, uint32(4), summary.TotalWorkload)
	assert.Equal(t, uint32(1), summary.TotalDisabled)

	user := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("post-upgrade-user", time.Now()),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status:     &corev1.User_Status{},
	}
	insertRscStoreObject(t, env, user)

	assert.Nil(t, env.srv.initDB(env.ctx))

	resp, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 13)
}

func TestInitDBDropsTheLegacyFTSIndex(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	insertRscStoreObject(t, env, newTestSession("fts-session", time.Now(),
		vutils.UUIDv4(), vutils.UUIDv4()))

	if _, err := env.srv.db.ExecContext(env.ctx,
		`PRAGMA create_fts_index('resources', 'uid', 'rsc_str')`); err != nil {
		t.Skipf("The FTS extension is unavailable: %+v", err)
	}

	var before int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx,
		`SELECT COUNT(*) FROM duckdb_tables() WHERE schema_name = 'fts_main_resources'`).Scan(&before))
	assert.NotZero(t, before)

	assert.Nil(t, env.srv.initDB(env.ctx))

	var after int
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx,
		`SELECT COUNT(*) FROM duckdb_tables() WHERE schema_name = 'fts_main_resources'`).Scan(&after))
	assert.Zero(t, after)

	env.srv.dropLegacyFTSIndex(env.ctx)
}

func TestResourceColumnSourceExpr(t *testing.T) {
	assert.Equal(t, `json_extract_string(rsc, '$.metadata.name')`,
		getTestResourceColumn(t, colName).sourceExpr("rsc"))
	assert.Equal(t, `TRY_CAST(json_extract_string(rsc, '$.metadata.createdAt') AS TIMESTAMP)`,
		getTestResourceColumn(t, colCreatedAt).sourceExpr("rsc"))
	assert.Equal(t, `TRY_CAST(json_extract_string(rsc, '$.metadata.tags') AS VARCHAR[])`,
		getTestResourceColumn(t, colTags).sourceExpr("rsc"))
	assert.Equal(t, `json_keys(rsc, '$.spec')`,
		getTestResourceColumn(t, colSpecKeys).sourceExpr("rsc"))

	assert.Equal(t, `v[3]`, getTestResourceColumn(t, colName).scalarExpr("v", 3))
	assert.Equal(t, `TRY_CAST(v[1] AS TIMESTAMP)`,
		getTestResourceColumn(t, colCreatedAt).scalarExpr("v", 1))

	assert.Equal(t, kindVarcharList, getTestResourceColumn(t, colSpecKeys).sqlType())
	assert.Equal(t, kindTimestamp, getTestResourceColumn(t, colCreatedAt).sqlType())
	assert.False(t, getTestResourceColumn(t, colSpecKeys).isScalar())
	assert.False(t, getTestResourceColumn(t, colTags).isScalar())
	assert.True(t, getTestResourceColumn(t, colCreatedAt).isScalar())
}

func TestGetResourceInsertQueryCoversEveryColumn(t *testing.T) {
	query := getResourceInsertQuery()

	for _, column := range getResourceColumns() {
		assert.Contains(t, query, column.name)
	}

	assert.Contains(t, query, "$6")
	assert.Len(t, getResourceInsertArgs("core", "v1", "User", "uid", "rv", "{}", ""), 7)
	assert.NotContains(t, query, "ON CONFLICT")
}

func TestToResourceListSupportsEveryResourceKind(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	var kinds []resourceKind
	kinds = append(kinds, coreResourceKinds()...)
	kinds = append(kinds, enterpriseResourceKinds()...)
	kinds = append(kinds, accessResourceKinds()...)

	assert.NotEmpty(t, kinds)

	for _, rk := range kinds {
		listMeta := &metav1.ListResponseMeta{Page: 1, ItemsPerPage: 10, TotalCount: 42, HasMore: true}

		ret, err := env.srv.toResourceList(nil, listMeta, rk.api, rk.version, rk.kind)
		assert.Nil(t, err, "%s/%s %s: %+v", rk.api, rk.version, rk.kind, err)
		if err != nil {
			continue
		}

		msg := ret.ProtoReflect()
		fields := msg.Descriptor().Fields()

		assert.Equal(t, fmt.Sprintf("%s/%s", rk.api, rk.version),
			msg.Get(fields.ByName("apiVersion")).String(), rk.kind)
		assert.Equal(t, fmt.Sprintf("%sList", rk.kind),
			msg.Get(fields.ByName("kind")).String())
		assert.Zero(t, msg.Get(fields.ByName("items")).List().Len())

		meta := msg.Get(fields.ByName("listResponseMeta")).Message()
		metaFields := meta.Descriptor().Fields()
		assert.Equal(t, uint64(42), meta.Get(metaFields.ByName("totalCount")).Uint())
		assert.True(t, meta.Get(metaFields.ByName("hasMore")).Bool())
	}

	assert.NotEmpty(t, uenterprisev1.API)
	assert.NotEmpty(t, uaccessv1.API)
}

func TestToResourceListCarriesTheItems(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	sess := newTestSession("carried-session", time.Now(), vutils.UUIDv4(), vutils.UUIDv4())

	ret, err := env.srv.toResourceList([]umetav1.ResourceObjectI{sess},
		&metav1.ListResponseMeta{ItemsPerPage: 10, TotalCount: 1},
		ucorev1.API, ucorev1.Version, ucorev1.KindSession)
	assert.Nil(t, err, "%+v", err)

	lst, ok := ret.(*corev1.SessionList)
	assert.True(t, ok)
	assert.Len(t, lst.Items, 1)
	assert.True(t, pbutils.IsEqual(sess, lst.Items[0]))
	assert.Equal(t, ucorev1.APIVersion, lst.ApiVersion)
	assert.Equal(t, "SessionList", lst.Kind)
	assert.Equal(t, uint32(1), lst.ListResponseMeta.TotalCount)
}

func TestGetOrderByExpr(t *testing.T) {
	for _, tst := range []struct {
		orderBy *vmetav1.CommonListOptions_OrderBy
		sql     string
	}{
		{orderBy: nil, sql: `created_at DESC`},
		{orderBy: &vmetav1.CommonListOptions_OrderBy{}, sql: `created_at DESC`},
		{orderBy: &vmetav1.CommonListOptions_OrderBy{
			Type: vmetav1.CommonListOptions_OrderBy_CREATED_AT}, sql: `created_at ASC`},
		{orderBy: &vmetav1.CommonListOptions_OrderBy{
			Type: vmetav1.CommonListOptions_OrderBy_CREATED_AT,
			Mode: vmetav1.CommonListOptions_OrderBy_DESC}, sql: `created_at DESC`},
		{orderBy: &vmetav1.CommonListOptions_OrderBy{
			Type: vmetav1.CommonListOptions_OrderBy_NAME}, sql: `name ASC`},
		{orderBy: &vmetav1.CommonListOptions_OrderBy{
			Type: vmetav1.CommonListOptions_OrderBy_NAME,
			Mode: vmetav1.CommonListOptions_OrderBy_DESC}, sql: `name DESC`},
	} {
		sqln, _, err := goqu.From("resources").Order(getOrderByExpr(tst.orderBy)).ToSQL()
		assert.Nil(t, err, "%+v", err)
		assert.Contains(t, sqln, tst.sql)
	}
}

func TestListRanksTheQueryMatchesByName(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	newUser := func(name, displayName string) *corev1.User {
		ret := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata:   newRscStoreMetadata(name, time.Now()),
			Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			Status:     &corev1.User_Status{},
		}
		ret.Metadata.DisplayName = displayName
		return ret
	}

	insertRscStoreObject(t, env, newUser("unrelated-one", "zeta"))
	insertRscStoreObject(t, env, newUser("contains-zeta-inside", "nothing"))
	insertRscStoreObject(t, env, newUser("zeta-prefixed", "nothing"))
	insertRscStoreObject(t, env, newUser("zeta", "nothing"))

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{Query: "zeta", ItemsPerPage: 100},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 4)
	assert.Equal(t, uint32(4), resp.ListResponseMeta.TotalCount)

	assert.Equal(t, "zeta", resp.Items[0].Metadata.Name)
	assert.Equal(t, "unrelated-one", resp.Items[1].Metadata.Name)
	assert.Equal(t, "zeta-prefixed", resp.Items[2].Metadata.Name)
	assert.Equal(t, "contains-zeta-inside", resp.Items[3].Metadata.Name)
}

func TestListReportsTheTotalCountAndHasMore(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	for i := 0; i < 25; i++ {
		insertRscStoreObject(t, env, &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata:   newRscStoreMetadata(fmt.Sprintf("counted-user-%02d", i), time.Now()),
			Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			Status:     &corev1.User_Status{},
		})
	}

	srvCore := &srvCore{s: env.srv}

	resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 10},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 10)
	assert.Equal(t, uint32(25), resp.ListResponseMeta.TotalCount)
	assert.True(t, resp.ListResponseMeta.HasMore)

	resp, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 10, Page: 2},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 5)
	assert.Equal(t, uint32(25), resp.ListResponseMeta.TotalCount)
	assert.False(t, resp.ListResponseMeta.HasMore)

	resp, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 25)
	assert.Equal(t, uint32(25), resp.ListResponseMeta.TotalCount)
	assert.False(t, resp.ListResponseMeta.HasMore)

	_, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 10, Page: 9},
	})
	assert.NotNil(t, err)
}

func TestListFiltersByCreatedAtRangeAndTag(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	base := time.Now().UTC().Add(-10 * time.Hour)

	for i := 0; i < 5; i++ {
		user := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata:   newRscStoreMetadata(fmt.Sprintf("ranged-user-%d", i), base.Add(time.Duration(i)*time.Hour)),
			Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			Status:     &corev1.User_Status{},
		}
		if i%2 == 0 {
			user.Metadata.Tags = []string{"even"}
		}
		insertRscStoreObject(t, env, user)
	}

	srvCore := &srvCore{s: env.srv}

	resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			ItemsPerPage: 100,
			From:         pbutils.Timestamp(base.Add(time.Hour)),
			To:           pbutils.Timestamp(base.Add(3 * time.Hour)),
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 3)
	assert.Equal(t, uint32(3), resp.ListResponseMeta.TotalCount)

	resp, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100, Tag: "even"},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 3)

	resp, err = srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100, Tag: "odd"},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Empty(t, resp.Items)
	assert.Zero(t, resp.ListResponseMeta.TotalCount)
}

func TestLegacyInsertStillWorksAfterTheMigration(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	insertRscStoreObject(t, env, newTestSession("migrated-session", time.Now(),
		vutils.UUIDv4(), vutils.UUIDv4()))

	rsc := newRawRscStoreResource(ucorev1.KindUser, "rolled-back-user", false,
		map[string]any{"type": "HUMAN"}, map[string]any{})

	insertLegacyRscStoreResource(t, env, rsc)

	uid := rsc["metadata"].(map[string]any)["uid"].(string)

	assert.Nil(t, getTestColumnValue(t, env, uid, colCreatedAt))
	assert.Nil(t, getTestColumnValue(t, env, uid, colSpecType))

	var legacyType string
	assert.Nil(t, env.srv.db.QueryRowContext(env.ctx,
		`SELECT rsc->>'$.spec.type' FROM resources WHERE uid = ?`, uid).Scan(&legacyType))
	assert.Equal(t, "HUMAN", legacyType)

	assert.Nil(t, env.srv.initResourceColumns(env.ctx))

	assert.Equal(t, "HUMAN", getTestColumnValue(t, env, uid, colSpecType))

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 1)
}
