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
	"net"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type auditCaptureServer struct {
	plogotlp.UnimplementedGRPCServer
	itemCh chan *enterprisev1.AuditLog
}

func (s *auditCaptureServer) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	logs := req.Logs()
	for i := range logs.ResourceLogs().Len() {
		scopeLogs := logs.ResourceLogs().At(i).ScopeLogs()
		for j := range scopeLogs.Len() {
			records := scopeLogs.At(j).LogRecords()
			for k := range records.Len() {
				ret := &enterprisev1.AuditLog{}
				if err := pbutils.UnmarshalFromMap(records.At(k).Body().Map().AsRaw(), ret); err != nil {
					return plogotlp.NewExportResponse(), err
				}

				select {
				case s.itemCh <- ret:
				case <-ctx.Done():
					return plogotlp.NewExportResponse(), ctx.Err()
				}
			}
		}
	}

	return plogotlp.NewExportResponse(), nil
}

func setAuditCaptureClient(t *testing.T, env *rscStoreTestEnv) *auditCaptureServer {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcSrv := grpc.NewServer()
	capture := &auditCaptureServer{
		itemCh: make(chan *enterprisev1.AuditLog, 10),
	}
	plogotlp.RegisterGRPCServer(grpcSrv, capture)

	go func() {
		_ = grpcSrv.Serve(listener)
	}()

	conn, err := grpc.NewClient("passthrough:///rscstore-audit-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		grpcSrv.Stop()
		_ = listener.Close()
		return capture
	}

	env.srv.client = plogotlp.NewGRPCClient(conn)

	t.Cleanup(func() {
		assert.Nil(t, conn.Close())
		grpcSrv.Stop()
		assert.Nil(t, listener.Close())
	})

	return capture
}

func insertAuditActorSession(t *testing.T, env *rscStoreTestEnv) (*corev1.Session, *metav1.ObjectReference, *metav1.ObjectReference) {
	t.Helper()

	userRef := &metav1.ObjectReference{Name: "actor-user", Uid: vutils.UUIDv4()}
	deviceRef := &metav1.ObjectReference{Name: "actor-device", Uid: vutils.UUIDv4()}
	session := &corev1.Session{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindSession,
		Metadata:   newRscStoreMetadata("actor-session", time.Now()),
		Spec:       &corev1.Session_Spec{State: corev1.Session_Spec_ACTIVE},
		Status: &corev1.Session_Status{
			Type:      corev1.Session_Status_CLIENT,
			UserRef:   userRef,
			DeviceRef: deviceRef,
		},
	}
	insertRscStoreObject(t, env, session)

	return session, userRef, deviceRef
}

func TestProcessAuditLogExportsEnrichedAuditLog(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	capture := setAuditCaptureClient(t, env)
	session, userRef, deviceRef := insertAuditActorSession(t, env)
	createdAt := time.Now().UTC().Add(-time.Minute)
	updatedAt := time.Now().UTC()

	resource := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "audit-target",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Timestamp(createdAt),
			UpdatedAt:       pbutils.Timestamp(updatedAt),
			ActorRef:        &metav1.ObjectReference{Uid: session.Metadata.Uid},
			ActorOperation:  "octelium.api.main.core.v1.UserService/UpdateUser",
		},
		Spec:   &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status: &corev1.User_Status{},
	}

	err := env.srv.processAuditLog(env.ctx, resource)
	assert.Nil(t, err, "%+v", err)

	select {
	case auditLog := <-capture.itemCh:
		assert.Equal(t, resource.Metadata.Uid, auditLog.Entry.ResourceRef.Uid)
		assert.Equal(t, session.Metadata.Uid, auditLog.Entry.SessionRef.Uid)
		assert.Equal(t, userRef.Uid, auditLog.Entry.UserRef.Uid)
		assert.Equal(t, deviceRef.Uid, auditLog.Entry.DeviceRef.Uid)
		assert.Equal(t, resource.Metadata.ActorOperation, auditLog.Entry.Operation)
		assert.Equal(t, "octelium.api.main.core.v1", auditLog.Entry.Package)
		assert.Equal(t, "UserService", auditLog.Entry.Service)
		assert.Equal(t, "UpdateUser", auditLog.Entry.Method)
		assert.Equal(t, updatedAt.UnixNano(), auditLog.Metadata.CreatedAt.AsTime().UnixNano())
		assert.Equal(t, resource.Metadata.Uid, auditLog.Metadata.TargetRef.Uid)
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Audit log was not exported")
	}
}

func TestProcessAuditLogMalformedOperationDoesNotPanic(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	capture := setAuditCaptureClient(t, env)
	session, _, _ := insertAuditActorSession(t, env)

	resource := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "malformed-operation",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Now(),
			ActorRef:        &metav1.ObjectReference{Uid: session.Metadata.Uid},
			ActorOperation:  "UserService/UpdateUser",
		},
		Spec:   &corev1.User_Spec{},
		Status: &corev1.User_Status{},
	}

	var err error
	if !assert.NotPanics(t, func() {
		err = env.srv.processAuditLog(env.ctx, resource)
	}) {
		return
	}
	assert.Nil(t, err, "%+v", err)

	select {
	case auditLog := <-capture.itemCh:
		assert.Equal(t, "UpdateUser", auditLog.Entry.Method)
		assert.Equal(t, resource.Metadata.ActorOperation, auditLog.Entry.Operation)
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Audit log was not exported")
	}
}

func TestProcessAuditLogSkipsUserAPIResources(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	capture := setAuditCaptureClient(t, env)
	session, _, _ := insertAuditActorSession(t, env)

	resource := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "user-api-resource",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Now(),
			ActorRef:        &metav1.ObjectReference{Uid: session.Metadata.Uid},
			ActorOperation:  "octelium.api.main.user.v1.UserService/UpdateUser",
		},
		Spec:   &corev1.User_Spec{},
		Status: &corev1.User_Status{},
	}

	err := env.srv.processAuditLog(env.ctx, resource)
	assert.Nil(t, err, "%+v", err)

	select {
	case <-capture.itemCh:
		assert.Fail(t, "User API audit log must be skipped")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestProcessAuditLogMissingActorSessionStillExports(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	capture := setAuditCaptureClient(t, env)
	resource := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "missing-actor-session",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Now(),
			ActorRef:        &metav1.ObjectReference{Uid: vutils.UUIDv4()},
			ActorOperation:  "octelium.api.main.core.v1.UserService/UpdateUser",
		},
		Spec:   &corev1.User_Spec{},
		Status: &corev1.User_Status{},
	}

	err := env.srv.processAuditLog(env.ctx, resource)
	assert.Nil(t, err, "%+v", err)

	select {
	case auditLog := <-capture.itemCh:
		assert.Nil(t, auditLog.Entry.UserRef)
		assert.Nil(t, auditLog.Entry.DeviceRef)
		assert.Equal(t, resource.Metadata.ActorRef.Uid, auditLog.Entry.SessionRef.Uid)
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Audit log was not exported")
	}
}

func TestProcessAuditLogWithoutActorDoesNothing(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	capture := setAuditCaptureClient(t, env)

	resource := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("no-actor", time.Now()),
		Spec:       &corev1.User_Spec{},
		Status:     &corev1.User_Status{},
	}

	err := env.srv.processAuditLog(env.ctx, resource)
	assert.Nil(t, err, "%+v", err)

	select {
	case <-capture.itemCh:
		assert.Fail(t, "Audit log without actor must not be exported")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSetAuditLogQueuesOnlyResourcesWithActor(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	withoutActor := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("without-actor", time.Now()),
		Spec:       &corev1.User_Spec{},
		Status:     &corev1.User_Status{},
	}
	env.srv.setAuditLog(env.ctx, withoutActor)
	assert.Zero(t, len(env.srv.auditLogItem))

	withActor := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("with-actor", time.Now()),
		Spec:       &corev1.User_Spec{},
		Status:     &corev1.User_Status{},
	}
	withActor.Metadata.ActorRef = &metav1.ObjectReference{Uid: vutils.UUIDv4()}
	env.srv.setAuditLog(env.ctx, withActor)
	assert.Equal(t, 1, len(env.srv.auditLogItem))
}

func TestConvertLogRecord(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Millisecond)
	auditLog := &enterprisev1.AuditLog{
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(createdAt),
		},
		Entry: &enterprisev1.AuditLog_Entry{
			Operation: "test-operation",
		},
	}

	record := plog.NewLogRecord()
	convertLogRecord(auditLog, record)

	assert.Equal(t, createdAt.UnixNano(), record.Timestamp().AsTime().UnixNano())
	assert.Equal(t, createdAt.UnixNano(), record.ObservedTimestamp().AsTime().UnixNano())
	assert.Equal(t, plog.SeverityNumberInfo, record.SeverityNumber())
	assert.Equal(t, plog.SeverityNumberInfo.String(), record.SeverityText())

	decoded := &enterprisev1.AuditLog{}
	err := pbutils.UnmarshalFromMap(record.Body().Map().AsRaw(), decoded)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, auditLog.Entry.Operation, decoded.Entry.Operation)
}
