// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func TestAccessLogQueriesComprehensive(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	region, err := ts.srv.octeliumC.CoreC().GetRegion(ts.ctx, &rmetav1.GetOptions{Name: vutils.GetMyRegionName()})
	assert.Nil(t, err, "%+v", err)

	user1 := createTestUser(t, ts)
	user2 := createTestUser(t, ts)
	service1 := createTestService(t, ts)
	service2 := createTestService(t, ts)

	userRef1 := umetav1.GetObjectReference(user1)
	userRef2 := umetav1.GetObjectReference(user2)
	serviceRef1 := umetav1.GetObjectReference(service1)
	serviceRef2 := umetav1.GetObjectReference(service2)
	regionRef := umetav1.GetObjectReference(region)
	namespaceRef := randomObjectReference()
	policyRef := randomObjectReference()
	deviceRef := randomObjectReference()
	sessionRef := randomObjectReference()

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)

	for idx := range 120 {
		opts := &accessLogOptions{
			CreatedAt:    base.Add(time.Duration(idx) * time.Second),
			Status:       corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:      userRef1,
			DeviceRef:    deviceRef,
			SessionRef:   sessionRef,
			ServiceRef:   serviceRef1,
			NamespaceRef: namespaceRef,
			RegionRef:    regionRef,
			PolicyRef:    policyRef,
		}

		if idx >= 75 {
			opts.UserRef = userRef2
		}
		if idx%3 == 0 {
			opts.ServiceRef = serviceRef2
		}
		if idx%4 == 0 {
			opts.Status = corev1.AccessLog_Entry_Common_DENIED
		}

		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(opts)))
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(120), resp.ListResponseMeta.TotalCount)
		assert.Equal(t, uint32(defaultItemsPerPage), resp.ListResponseMeta.ItemsPerPage)
		assert.Equal(t, defaultItemsPerPage, len(resp.Items))
		assert.True(t, resp.ListResponseMeta.HasMore)
		assert.True(t, resp.Items[0].Metadata.CreatedAt.AsTime().After(resp.Items[len(resp.Items)-1].Metadata.CreatedAt.AsTime()))
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{
				Page:         2,
				ItemsPerPage: 50,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 20, len(resp.Items))
		assert.Equal(t, uint32(120), resp.ListResponseMeta.TotalCount)
		assert.False(t, resp.ListResponseMeta.HasMore)
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
				OrderBy: &vmetav1.CommonListOptions_OrderBy{
					Mode: vmetav1.CommonListOptions_OrderBy_ASC,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 120, len(resp.Items))
		assert.True(t, resp.Items[0].Metadata.CreatedAt.AsTime().Before(resp.Items[len(resp.Items)-1].Metadata.CreatedAt.AsTime()))
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: &metav1.ObjectReference{Uid: userRef1.Uid},
			Common:  &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(75), resp.ListResponseMeta.TotalCount)
		assert.Equal(t, 75, len(resp.Items))
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			ServiceRef: &metav1.ObjectReference{
				Uid:  serviceRef1.Uid,
				Name: serviceRef1.Name,
			},
			Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(80), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Status: corev1.AccessLog_Entry_Common_DENIED,
			Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(30), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			From:   pbutils.Timestamp(base.Add(60 * time.Second)),
			To:     pbutils.Timestamp(base.Add(89 * time.Second)),
			Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(30), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 120, int(resp.TotalNumber))
		assert.Equal(t, 90, int(resp.TotalAllowed))
		assert.Equal(t, 30, int(resp.TotalDenied))
		assert.Equal(t, 2, int(resp.TotalUser))
		assert.Equal(t, 1, int(resp.TotalSession))
		assert.Equal(t, 1, int(resp.TotalDevice))
		assert.Equal(t, 2, int(resp.TotalService))
		assert.Equal(t, 1, int(resp.TotalNamespace))
		assert.Equal(t, 1, int(resp.TotalMatchPolicy))
	}

	{
		resp, err := ts.srv.listAccessLogTopUser(ts.ctx, &visibilityv1.ListAccessLogTopUserRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, resp.Items)
		assert.Equal(t, user1.Metadata.Uid, resp.Items[0].User.Metadata.Uid)
		assert.Equal(t, int32(75), resp.Items[0].Count)
	}

	{
		resp, err := ts.srv.listAccessLogTopService(ts.ctx, &visibilityv1.ListAccessLogTopServiceRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, resp.Items)
		assert.Equal(t, service1.Metadata.Uid, resp.Items[0].Service.Metadata.Uid)
		assert.Equal(t, int32(80), resp.Items[0].Count)
	}

	{
		resp, err := ts.srv.getTop(ts.ctx, "access_logs", 10, "entry.common.reason.details.policyMatch.policy.policyRef", nil)
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.items, 1)
		assert.Equal(t, policyRef.Uid, resp.items[0].UID)
		assert.Equal(t, 120, resp.items[0].Count)
	}

	{
		_, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{Page: 100001},
		})
		assert.NotNil(t, err)
	}
}

func TestCleanupByAgeAndMaximumCount(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	oldAccessMax := maxDBAccessLogs
	oldAuthenticationMax := maxDBAuthenticationLogs
	oldAuditMax := maxDBAuditLogs
	oldComponentMax := maxDBComponentLogs
	t.Cleanup(func() {
		maxDBAccessLogs = oldAccessMax
		maxDBAuthenticationLogs = oldAuthenticationMax
		maxDBAuditLogs = oldAuditMax
		maxDBComponentLogs = oldComponentMax
	})

	maxDBAccessLogs = 25
	maxDBAuthenticationLogs = 20
	maxDBAuditLogs = 15
	maxDBComponentLogs = 10

	now := time.Now().UTC().Truncate(time.Second)
	old := now.Add(-ts.srv.cleanupDuration - time.Hour)

	for idx := range 40 {
		createdAt := now.Add(time.Duration(idx) * time.Second)
		if idx < 5 {
			createdAt = old.Add(time.Duration(idx) * time.Second)
		}

		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "authentication_logs", marshalLog(t, newAuthenticationLog(&authenticationLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t, newAuditLog(&auditLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t, newComponentLog(createdAt, corev1.ComponentLog_Entry_INFO, "test")))
	}

	err := ts.srv.doCleanup(ts.ctx)
	assert.Nil(t, err, "%+v", err)

	assert.Equal(t, maxDBAccessLogs, getTableCount(t, ts.srv, "access_logs"))
	assert.Equal(t, maxDBAuthenticationLogs, getTableCount(t, ts.srv, "authentication_logs"))
	assert.Equal(t, maxDBAuditLogs, getTableCount(t, ts.srv, "audit_logs"))
	assert.Equal(t, maxDBComponentLogs, getTableCount(t, ts.srv, "component_logs"))
}

func TestAuthenticationLogQueriesComprehensive(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	user1 := createTestUser(t, ts)
	user2 := createTestUser(t, ts)
	userRef1 := umetav1.GetObjectReference(user1)
	userRef2 := umetav1.GetObjectReference(user2)
	deviceRef := randomObjectReference()
	sessionRef := randomObjectReference()
	credentialRef1 := randomObjectReference()
	credentialRef2 := randomObjectReference()
	identityProviderRef1 := randomObjectReference()
	identityProviderRef2 := randomObjectReference()

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	for idx := range 60 {
		opts := &authenticationLogOptions{
			CreatedAt:           base.Add(time.Duration(idx) * time.Second),
			UserRef:             userRef1,
			DeviceRef:           deviceRef,
			SessionRef:          sessionRef,
			Type:                corev1.Session_Status_Authentication_Info_CREDENTIAL,
			AAL:                 corev1.Session_Status_Authentication_Info_AAL2,
			AuthenticationIndex: 0,
			CredentialRef:       credentialRef1,
		}

		if idx >= 40 {
			opts.UserRef = userRef2
			opts.CredentialRef = credentialRef2
		}
		if idx%3 == 0 {
			opts.Type = corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER
			opts.CredentialRef = nil
			opts.IdentityProviderRef = identityProviderRef1
			if idx >= 40 {
				opts.IdentityProviderRef = identityProviderRef2
			}
		}
		if idx%4 == 0 {
			opts.AuthenticationIndex = 1
		}

		insertLogJSON(t, ts.srv, "authentication_logs", marshalLog(t, newAuthenticationLog(opts)))
	}

	{
		resp, err := ts.srv.listAuthenticationLog(ts.ctx, &visibilityv1.ListAuthenticationLogRequest{
			Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 60)
		assert.Equal(t, uint32(60), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.listAuthenticationLog(ts.ctx, &visibilityv1.ListAuthenticationLogRequest{
			UserRef: &metav1.ObjectReference{Uid: userRef1.Uid},
			Common:  &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(40), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.getSummaryAuthenticationLog(ts.ctx, &visibilityv1.GetAuthenticationLogSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 60, int(resp.TotalNumber))
		assert.Equal(t, 40, int(resp.TotalCredential))
		assert.Equal(t, 20, int(resp.TotalIdentityProvider))
		assert.Equal(t, 60, int(resp.TotalAAL2))
		assert.Equal(t, 15, int(resp.TotalReauthentication))
		assert.Equal(t, 2, int(resp.TotalUser))
		assert.Equal(t, 1, int(resp.TotalSession))
	}

	{
		resp, err := ts.srv.listAuthenticationLogTopUser(ts.ctx, &visibilityv1.ListAuthenticationLogTopUserRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, resp.Items)
		assert.Equal(t, user1.Metadata.Uid, resp.Items[0].User.Metadata.Uid)
		assert.Equal(t, int32(40), resp.Items[0].Count)
	}

	{
		resp, err := ts.srv.getTop(ts.ctx, "authentication_logs", 10, "entry.authentication.info.credential.credentialRef", nil)
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.items, 2)
		assert.Equal(t, credentialRef1.Uid, resp.items[0].UID)
	}

	{
		resp, err := ts.srv.getTop(ts.ctx, "authentication_logs", 10, "entry.authentication.info.identityProvider.identityProviderRef", nil)
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.items, 2)
		assert.Equal(t, identityProviderRef1.Uid, resp.items[0].UID)
	}
}

func TestAuditAndComponentLogQueriesComprehensive(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	user1 := createTestUser(t, ts)
	user2 := createTestUser(t, ts)
	userRef1 := umetav1.GetObjectReference(user1)
	userRef2 := umetav1.GetObjectReference(user2)
	deviceRef := randomObjectReference()
	sessionRef := randomObjectReference()
	resourceRef1 := randomObjectReference()
	resourceRef2 := randomObjectReference()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	for idx := range 30 {
		opts := &auditLogOptions{
			CreatedAt:   base.Add(time.Duration(idx) * time.Second),
			UserRef:     userRef1,
			DeviceRef:   deviceRef,
			SessionRef:  sessionRef,
			ResourceRef: resourceRef1,
			Operation:   "CREATE",
		}

		if idx >= 20 {
			opts.UserRef = userRef2
			opts.ResourceRef = resourceRef2
			opts.Operation = "DELETE"
		}

		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t, newAuditLog(opts)))
	}

	levels := []corev1.ComponentLog_Entry_Level{
		corev1.ComponentLog_Entry_DEBUG,
		corev1.ComponentLog_Entry_INFO,
		corev1.ComponentLog_Entry_WARN,
		corev1.ComponentLog_Entry_ERROR,
		corev1.ComponentLog_Entry_PANIC,
		corev1.ComponentLog_Entry_FATAL,
	}

	for idx, level := range levels {
		for count := 0; count <= idx; count++ {
			insertLogJSON(t, ts.srv, "component_logs", marshalLog(t, newComponentLog(base.Add(time.Duration(idx*10+count)*time.Second), level, level.String())))
		}
	}

	{
		resp, err := ts.srv.listAuditLog(ts.ctx, &visibilityv1.ListAuditLogRequest{
			ResourceRef: resourceRef1,
			Common:      &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 20)
		assert.Equal(t, uint32(20), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.getSummaryAuditLog(ts.ctx, &visibilityv1.GetAuditLogSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 30, int(resp.TotalNumber))
		assert.Equal(t, 2, int(resp.TotalResource))
		assert.Equal(t, 2, int(resp.TotalUser))
		assert.Equal(t, 1, int(resp.TotalSession))
		assert.Equal(t, 1, int(resp.TotalDevice))
	}

	{
		resp, err := ts.srv.listAuditLogTopUser(ts.ctx, &visibilityv1.ListAuditLogTopUserRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, resp.Items)
		assert.Equal(t, user1.Metadata.Uid, resp.Items[0].User.Metadata.Uid)
		assert.Equal(t, int32(20), resp.Items[0].Count)
	}

	{
		resp, err := ts.srv.listComponentLog(ts.ctx, &visibilityv1.ListComponentLogRequest{
			Level:  corev1.ComponentLog_Entry_ERROR,
			Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 4)
		assert.Equal(t, uint32(4), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := ts.srv.getSummaryComponentLog(ts.ctx, &visibilityv1.GetComponentLogSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 21, int(resp.TotalNumber))
		assert.Equal(t, 1, int(resp.TotalDebug))
		assert.Equal(t, 2, int(resp.TotalInfo))
		assert.Equal(t, 3, int(resp.TotalWarn))
		assert.Equal(t, 4, int(resp.TotalError))
		assert.Equal(t, 5, int(resp.TotalPanic))
		assert.Equal(t, 6, int(resp.TotalFatal))
	}
}
