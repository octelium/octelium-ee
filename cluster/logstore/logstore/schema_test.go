// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func insertLegacyLogJSON(t *testing.T, srv *Server, table string, data []byte) {
	t.Helper()

	_, err := srv.db.Exec(fmt.Sprintf("INSERT INTO %s (rsc) VALUES ($1)", table), string(data))
	assert.Nil(t, err, "%+v", err)
}

func getTestLogColumn(t *testing.T, srv *Server, table, column string) []sql.NullString {
	t.Helper()

	rows, err := srv.db.Query(fmt.Sprintf(`SELECT CAST(%s AS VARCHAR) FROM %s ORDER BY rowid`, column, table))
	assert.Nil(t, err, "%+v", err)
	defer rows.Close()

	ret := []sql.NullString{}
	for rows.Next() {
		var value sql.NullString
		assert.Nil(t, rows.Scan(&value))
		ret = append(ret, value)
	}

	return ret
}

func testRef(uid string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       "User",
		Uid:        testRefUID(uid),
		Name:       "name-" + uid,
	}
}

func testRefUID(uid string) string {
	sum := 0
	for _, c := range uid {
		sum += int(c)
	}
	return fmt.Sprintf("%08x-1111-4111-8111-%012d", sum, len(uid)*1000+sum)
}

func TestGetLogTableColumns(t *testing.T) {
	for _, table := range getLogTables() {
		columns := getLogTableColumns(table)
		assert.NotEmpty(t, columns)
		assert.Equal(t, colCreatedAt, columns[0].name)
		assert.Equal(t, "TIMESTAMP", columns[0].kind)
	}

	assert.Len(t, getLogTableColumns("access_logs"), 9)
	assert.Len(t, getLogTableColumns("component_logs"), 2)
	assert.Len(t, getLogTableColumns("authentication_logs"), 1)
	assert.Len(t, getLogTableColumns("audit_logs"), 1)
}

func TestGetLogInsertQuery(t *testing.T) {
	query, err := getLogInsertQuery(logTableAccess)
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, query, "INSERT INTO access_logs")
	assert.Contains(t, query, colUserUID)
	assert.Contains(t, query, "$.entry.common.reason.details.policyMatch.policy.policyRef.uid")

	query, err = getLogInsertQuery(logTableAudit)
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, query, "INSERT INTO audit_logs")
	assert.NotContains(t, query, colUserUID)

	_, err = getLogInsertQuery(logTableUnknown)
	assert.NotNil(t, err)
}

func TestInsertPopulatesLogColumns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	log := newAccessLog(&accessLogOptions{
		CreatedAt:    createdAt,
		Status:       corev1.AccessLog_Entry_Common_DENIED,
		UserRef:      testRef("usr-1"),
		DeviceRef:    testRef("dev-1"),
		SessionRef:   testRef("ses-1"),
		ServiceRef:   testRef("svc-1"),
		NamespaceRef: testRef("ns-1"),
		RegionRef:    testRef("rgn-1"),
		PolicyRef:    testRef("pol-1"),
	})

	assert.Nil(t, ts.srv.insertAccessLog(marshalLog(t, log)))

	for column, expected := range map[string]string{
		colStatus:       "DENIED",
		colUserUID:      testRefUID("usr-1"),
		colDeviceUID:    testRefUID("dev-1"),
		colSessionUID:   testRefUID("ses-1"),
		colServiceUID:   testRefUID("svc-1"),
		colNamespaceUID: testRefUID("ns-1"),
		colRegionUID:    testRefUID("rgn-1"),
		colPolicyUID:    testRefUID("pol-1"),
	} {
		values := getTestLogColumn(t, ts.srv, "access_logs", column)
		assert.Len(t, values, 1, column)
		assert.Equal(t, expected, values[0].String, column)
	}

	var stored time.Time
	assert.Nil(t, ts.srv.db.QueryRow(`SELECT created_at FROM access_logs`).Scan(&stored))
	assert.Equal(t, createdAt, stored.UTC())
}

func TestInsertPopulatesComponentLogLevel(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	assert.Nil(t, ts.srv.insertComponentLog(marshalLog(t,
		newComponentLog(time.Now().UTC(), corev1.ComponentLog_Entry_ERROR, "boom"))))

	values := getTestLogColumn(t, ts.srv, "component_logs", colLevel)
	assert.Len(t, values, 1)
	assert.Equal(t, "ERROR", values[0].String)
}

func TestInsertWithMissingFieldsLeavesNullColumns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	log := newAccessLog(&accessLogOptions{CreatedAt: time.Now().UTC()})
	assert.Nil(t, ts.srv.insertAccessLog(marshalLog(t, log)))

	values := getTestLogColumn(t, ts.srv, "access_logs", colUserUID)
	assert.Len(t, values, 1)
	assert.False(t, values[0].Valid)

	values = getTestLogColumn(t, ts.srv, "access_logs", colCreatedAt)
	assert.True(t, values[0].Valid)
}

func TestBackfillLogColumns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	for idx := range 25 {
		insertLegacyLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt: now.Add(time.Duration(idx) * time.Second),
			UserRef:   testRef(fmt.Sprintf("usr-%d", idx%3)),
		})))
	}

	values := getTestLogColumn(t, ts.srv, "access_logs", colCreatedAt)
	assert.Len(t, values, 25)
	assert.False(t, values[0].Valid)

	assert.Nil(t, ts.srv.initLogColumns(ts.ctx))

	values = getTestLogColumn(t, ts.srv, "access_logs", colCreatedAt)
	assert.Len(t, values, 25)
	for _, value := range values {
		assert.True(t, value.Valid)
	}

	values = getTestLogColumn(t, ts.srv, "access_logs", colUserUID)
	assert.Equal(t, testRefUID("usr-0"), values[0].String)
	assert.Equal(t, testRefUID("usr-1"), values[1].String)
}

func TestBackfillLogColumnsIsIdempotentAndTerminates(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	insertLegacyLogJSON(t, ts.srv, "access_logs",
		marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: time.Now().UTC()})))

	_, err := ts.srv.db.Exec(`INSERT INTO access_logs (rsc) VALUES ('{"kind": "AccessLog"}')`)
	assert.Nil(t, err, "%+v", err)

	for range 3 {
		assert.Nil(t, ts.srv.initLogColumns(ts.ctx))
	}

	values := getTestLogColumn(t, ts.srv, "access_logs", colCreatedAt)
	assert.Len(t, values, 2)
	assert.True(t, values[0].Valid)
	assert.False(t, values[1].Valid)
}

func TestBackfilledLogsRemainQueryable(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	for idx := range 10 {
		insertLegacyLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt: now.Add(-time.Duration(idx) * time.Minute),
			Status:    corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:   testRef("usr-1"),
		})))
	}

	assert.Nil(t, ts.srv.initLogColumns(ts.ctx))

	resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		UserRef: &metav1.ObjectReference{Uid: testRefUID("usr-1")},
		Common:  &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 10)
	assert.Equal(t, uint32(10), resp.ListResponseMeta.TotalCount)

	assert.False(t, resp.Items[0].Metadata.CreatedAt.AsTime().
		Before(resp.Items[1].Metadata.CreatedAt.AsTime()))

	summary, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
		From: pbutils.Timestamp(now.Add(-time.Hour)),
		To:   pbutils.Timestamp(now.Add(time.Hour)),
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, uint64(10), summary.TotalNumber)
	assert.Equal(t, uint64(10), summary.TotalAllowed)
	assert.Equal(t, uint64(1), summary.TotalUser)
}

func TestListAccessLogFiltersByIndexedColumns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	for idx := range 12 {
		status := corev1.AccessLog_Entry_Common_ALLOWED
		if idx%3 == 0 {
			status = corev1.AccessLog_Entry_Common_DENIED
		}
		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt:  now.Add(time.Duration(idx) * time.Minute),
			Status:     status,
			UserRef:    testRef(fmt.Sprintf("usr-%d", idx%2)),
			ServiceRef: testRef("svc-1"),
		})))
	}

	resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		Status: corev1.AccessLog_Entry_Common_DENIED,
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 4)

	resp, err = ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		UserRef: &metav1.ObjectReference{Uid: testRefUID("usr-1")},
		Common:  &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 6)

	resp, err = ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		UserRef: &metav1.ObjectReference{Name: "name-usr-1"},
		Common:  &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 6)

	resp, err = ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		From:   pbutils.Timestamp(now.Add(6 * time.Minute)),
		Common: &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 6)
}

func TestUpgradeFromLegacyLogSchema(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	for _, table := range getLogTables() {
		_, err := ts.srv.db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table))
		assert.Nil(t, err, "%+v", err)

		_, err = ts.srv.db.Exec(fmt.Sprintf(`CREATE TABLE %s (rsc JSON)`, table))
		assert.Nil(t, err, "%+v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	for idx := range 20 {
		_, err := ts.srv.db.Exec(`INSERT INTO access_logs VALUES ($1)`,
			string(marshalLog(t, newAccessLog(&accessLogOptions{
				CreatedAt: now.Add(-time.Duration(idx) * time.Minute),
				Status:    corev1.AccessLog_Entry_Common_ALLOWED,
				UserRef:   testRef("usr-1"),
			}))))
		assert.Nil(t, err, "%+v", err)
	}

	_, err := ts.srv.db.Exec(`INSERT INTO component_logs VALUES ($1)`,
		string(marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_INFO, "legacy"))))
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ts.srv.initDB(ts.ctx))

	resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		UserRef: &metav1.ObjectReference{Uid: testRefUID("usr-1")},
		Common:  &vmetav1.CommonListOptions{ItemsPerPage: 50},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 20)

	summary, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
		From: pbutils.Timestamp(now.Add(-time.Hour)),
		To:   pbutils.Timestamp(now.Add(time.Hour)),
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, uint64(20), summary.TotalNumber)

	values := getTestLogColumn(t, ts.srv, "component_logs", colLevel)
	assert.Len(t, values, 1)
	assert.Equal(t, "INFO", values[0].String)

	assert.Nil(t, ts.srv.insertAccessLog(marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: now}))))
	assert.Nil(t, ts.srv.initDB(ts.ctx))
	assert.Equal(t, 21, getTableCount(t, ts.srv, "access_logs"))
}
