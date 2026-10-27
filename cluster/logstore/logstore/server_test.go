// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type testServer struct {
	ctx     context.Context
	srv     *Server
	coreSrv *admin.Server
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	srv, err := newServer(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		_ = tst.Destroy()
		return nil
	}

	err = srv.initDB(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		_ = srv.db.Close()
		_ = tst.Destroy()
		return nil
	}

	t.Cleanup(func() {
		assert.Nil(t, srv.db.Close())
		tst.Destroy()
	})

	return &testServer{
		ctx: ctx,
		srv: srv,
		coreSrv: admin.NewServer(&admin.Opts{
			IsEmbedded: true,
			OcteliumC:  srv.octeliumC,
		}),
	}
}

func marshalLog(t *testing.T, msg proto.Message) []byte {
	t.Helper()

	ret, err := pbutils.MarshalJSON(msg, false)
	assert.Nil(t, err, "%+v", err)
	return ret
}

func insertLogJSON(t *testing.T, srv *Server, table string, data []byte) {
	t.Helper()

	_, err := srv.db.Exec(getLogTableInsertQuery(table), string(data))
	assert.Nil(t, err, "%+v", err)
}

func getTableCount(t *testing.T, srv *Server, table string) int {
	t.Helper()

	var ret int
	err := srv.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&ret)
	assert.Nil(t, err, "%+v", err)
	return ret
}

type accessLogOptions struct {
	CreatedAt     time.Time
	Status        corev1.AccessLog_Entry_Common_Status
	UserRef       *metav1.ObjectReference
	DeviceRef     *metav1.ObjectReference
	SessionRef    *metav1.ObjectReference
	ServiceRef    *metav1.ObjectReference
	NamespaceRef  *metav1.ObjectReference
	RegionRef     *metav1.ObjectReference
	PolicyRef     *metav1.ObjectReference
	ConnectionID  string
	SessionID     string
	Sequence      int64
	SSHType       corev1.AccessLog_Entry_Info_SSH_Type
	RecordingType corev1.AccessLog_Entry_Info_SSH_SessionRecording_Type
	RecordingData []byte
}

func newAccessLog(opts *accessLogOptions) *corev1.AccessLog {
	if opts == nil {
		opts = &accessLogOptions{}
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}

	common := &corev1.AccessLog_Entry_Common{
		StartedAt:    pbutils.Timestamp(opts.CreatedAt.Add(-time.Second)),
		EndedAt:      pbutils.Timestamp(opts.CreatedAt),
		Status:       opts.Status,
		UserRef:      opts.UserRef,
		DeviceRef:    opts.DeviceRef,
		SessionRef:   opts.SessionRef,
		ServiceRef:   opts.ServiceRef,
		NamespaceRef: opts.NamespaceRef,
		RegionRef:    opts.RegionRef,
		ConnectionID: opts.ConnectionID,
		SessionID:    opts.SessionID,
		Sequence:     opts.Sequence,
	}

	if common.Status == corev1.AccessLog_Entry_Common_STATUS_UNSET {
		common.Status = corev1.AccessLog_Entry_Common_ALLOWED
	}

	if opts.PolicyRef != nil {
		common.Reason = &corev1.AccessLog_Entry_Common_Reason{
			Type: corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH,
			Details: &corev1.AccessLog_Entry_Common_Reason_Details{
				Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_{
					PolicyMatch: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch{
						Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_Policy_{
							Policy: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_Policy{
								PolicyRef: opts.PolicyRef,
							},
						},
					},
				},
			},
		}
	}

	ret := &corev1.AccessLog{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindAccessLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(opts.CreatedAt),
		},
		Entry: &corev1.AccessLog_Entry{
			Common: common,
		},
	}

	if opts.SSHType != corev1.AccessLog_Entry_Info_SSH_TYPE_UNKNOWN {
		ssh := &corev1.AccessLog_Entry_Info_SSH{
			Type: opts.SSHType,
		}

		if opts.SSHType == corev1.AccessLog_Entry_Info_SSH_SESSION_RECORDING {
			ssh.Details = &corev1.AccessLog_Entry_Info_SSH_SessionRecording_{
				SessionRecording: &corev1.AccessLog_Entry_Info_SSH_SessionRecording{
					Type: opts.RecordingType,
					Data: append([]byte(nil), opts.RecordingData...),
				},
			}
		}

		ret.Entry.Info = &corev1.AccessLog_Entry_Info{
			Type: &corev1.AccessLog_Entry_Info_Ssh{
				Ssh: ssh,
			},
		}
	}

	return ret
}

type authenticationLogOptions struct {
	CreatedAt           time.Time
	UserRef             *metav1.ObjectReference
	DeviceRef           *metav1.ObjectReference
	SessionRef          *metav1.ObjectReference
	Type                corev1.Session_Status_Authentication_Info_Type
	AAL                 corev1.Session_Status_Authentication_Info_AAL
	AuthenticationIndex uint32
	CredentialRef       *metav1.ObjectReference
	IdentityProviderRef *metav1.ObjectReference
	Status              enterprisev1.AuthenticationLog_Entry_Status
	FailureReason       enterprisev1.AuthenticationLog_Entry_Failure_Reason
}

func newAuthenticationLog(opts *authenticationLogOptions) *enterprisev1.AuthenticationLog {
	if opts == nil {
		opts = &authenticationLogOptions{}
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}

	info := &corev1.Session_Status_Authentication_Info{
		Type: opts.Type,
		Aal:  opts.AAL,
	}

	if opts.CredentialRef != nil {
		info.Details = &corev1.Session_Status_Authentication_Info_Credential_{
			Credential: &corev1.Session_Status_Authentication_Info_Credential{
				CredentialRef: opts.CredentialRef,
			},
		}
	}

	if opts.IdentityProviderRef != nil {
		info.Details = &corev1.Session_Status_Authentication_Info_IdentityProvider_{
			IdentityProvider: &corev1.Session_Status_Authentication_Info_IdentityProvider{
				IdentityProviderRef: opts.IdentityProviderRef,
			},
		}
	}

	return &enterprisev1.AuthenticationLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuthenticationLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(opts.CreatedAt),
		},
		Entry: &enterprisev1.AuthenticationLog_Entry{
			UserRef:             opts.UserRef,
			DeviceRef:           opts.DeviceRef,
			SessionRef:          opts.SessionRef,
			Authentication:      &corev1.Session_Status_Authentication{Info: info},
			AuthenticationIndex: opts.AuthenticationIndex,
			Status:              opts.Status,
			Failure: func() *enterprisev1.AuthenticationLog_Entry_Failure {
				if opts.Status != enterprisev1.AuthenticationLog_Entry_FAILURE {
					return nil
				}
				return &enterprisev1.AuthenticationLog_Entry_Failure{
					Reason: opts.FailureReason,
				}
			}(),
		},
	}
}

type auditLogOptions struct {
	CreatedAt   time.Time
	UserRef     *metav1.ObjectReference
	DeviceRef   *metav1.ObjectReference
	SessionRef  *metav1.ObjectReference
	ResourceRef *metav1.ObjectReference
	Operation   string
}

func newAuditLog(opts *auditLogOptions) *enterprisev1.AuditLog {
	if opts == nil {
		opts = &auditLogOptions{}
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}

	return &enterprisev1.AuditLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuditLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(opts.CreatedAt),
		},
		Entry: &enterprisev1.AuditLog_Entry{
			UserRef:     opts.UserRef,
			DeviceRef:   opts.DeviceRef,
			SessionRef:  opts.SessionRef,
			ResourceRef: opts.ResourceRef,
			Operation:   opts.Operation,
		},
	}
}

func newComponentLog(createdAt time.Time, level corev1.ComponentLog_Entry_Level, message string) *corev1.ComponentLog {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return &corev1.ComponentLog{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindComponentLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(createdAt),
		},
		Entry: &corev1.ComponentLog_Entry{
			Level:   level,
			Message: message,
			Time:    pbutils.Timestamp(createdAt),
		},
	}
}

func createTestUser(t *testing.T, ts *testServer) *corev1.User {
	t.Helper()

	ret, err := ts.coreSrv.CreateUser(ts.ctx, tests.GenUser(nil))
	assert.Nil(t, err, "%+v", err)
	return ret
}

func createTestService(t *testing.T, ts *testServer) *corev1.Service {
	t.Helper()

	ret, err := ts.coreSrv.CreateService(ts.ctx, tests.GenService(""))
	assert.Nil(t, err, "%+v", err)
	return ret
}

func randomObjectReference() *metav1.ObjectReference {
	return &metav1.ObjectReference{
		Uid:  vutils.UUIDv4(),
		Name: fmt.Sprintf("rsc-%s", utilrand.GetRandomStringLowercase(6)),
	}
}

func TestServer(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	assert.Nil(t, err)

	coreSrv := admin.NewServer(&admin.Opts{
		IsEmbedded: true,
		OcteliumC:  srv.octeliumC,
	})

	usr, err := coreSrv.CreateUser(ctx, tests.GenUser(nil))
	assert.Nil(t, err)

	for range 1000 {
		accessLog := &corev1.AccessLog{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindAccessLog,
			Metadata: &metav1.LogMetadata{
				Id:        vutils.GenerateLogID(),
				CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 100) * int(time.Minute)))),
			},

			Entry: &corev1.AccessLog_Entry{

				Common: &corev1.AccessLog_Entry_Common{
					StartedAt: pbutils.Timestamp(pbutils.Now().
						AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 10000)) * time.Second)),
					EndedAt:   pbutils.Now(),
					Status:    corev1.AccessLog_Entry_Common_ALLOWED,
					RegionRef: umetav1.GetObjectReference(rgn),
					UserRef:   umetav1.GetObjectReference(usr),
				},
			},
		}

		jsn, err := pbutils.MarshalJSON(accessLog, false)
		assert.Nil(t, err)
		err = srv.insertAccessLog(jsn)
		assert.Nil(t, err)

		err = srv.listAccessLogTop(ctx)
		assert.Nil(t, err)
	}

	resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
	assert.Nil(t, err, "%+v", err)

	assert.True(t, (resp.ListResponseMeta.TotalCount) == 1000)

	{
		res, err := srv.getSummaryAccessLog(ctx, &visibilityv1.GetAccessLogSummaryRequest{})
		assert.Nil(t, err)

		assert.Equal(t, 1000, int(res.TotalNumber))
	}

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(),
			},
		})
		assert.Nil(t, err)

		assert.True(t, len(resp.Items) == 0)
	}

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			RegionRef: umetav1.GetObjectReference(rgn),
		})
		assert.Nil(t, err, "%+v", err)

		assert.True(t, (resp.ListResponseMeta.TotalCount) == 1000)
	}

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			RegionRef: &metav1.ObjectReference{
				Uid: rgn.Metadata.Uid,
			},
		})
		assert.Nil(t, err)

		assert.True(t, (resp.ListResponseMeta.TotalCount) == 1000)

	}

	{

		timeAt := pbutils.Timestamp(pbutils.Now().AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(100, 200) * int(time.Second))))
		respFrom, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			From: timeAt,
		})
		assert.Nil(t, err)

		respTo, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			To: timeAt,
		})
		assert.Nil(t, err)

		assert.True(t, respFrom.ListResponseMeta.TotalCount+respTo.ListResponseMeta.TotalCount == 1000)

	}

	/*
			{

				const sqlT = `
		WITH block_info AS (
		    SELECT CAST(block_size AS BIGINT) AS block_size
		    FROM pragma_database_size()
		),
		-- Step 2: Aggregate storage_info for the table
		table_storage AS (
		    SELECT
		        COUNT(DISTINCT block_id) AS used_blocks
		    FROM pragma_storage_info('access_logs')
		    WHERE persistent = TRUE AND block_id IS NOT NULL
		)
		-- Step 3: Compute total size in GiB
		SELECT
		    (block_size * used_blocks) / 1024 / 1024 / 1024 AS table_size_gb
		FROM block_info, table_storage;
				`
				_, err := srv.db.Exec(`PRAGMA storage_info('access_logs')`)
				assert.Nil(t, err)
				rows, err := srv.db.QueryContext(ctx, sqlT)
				assert.Nil(t, err)
				for rows.Next() {
					var a1 int

					rows.Scan(&a1)
				}
			}
	*/
	/*
		err = srv.getOP(ctx)
		assert.Nil(t, err, "OP ERR %+v", err)

		db := srv.db

		usr := tests.GenUserHuman([]string{"g1", "g2"})
		usrJSON, err := pbutils.MarshalJSON(usr, false)
		assert.Nil(t, err)
		fmt.Printf("=== %s\n", string(usrJSON))
		_, err = db.Exec(fmt.Sprintf(`INSERT INTO resources VALUES ('%s')`, string(usrJSON)))

		rsc := make([]any, 3)
		row := db.QueryRow(`SELECT rsc->'$.spec.groups[*]' as groups FROM resources WHERE json_contains(rsc->'$.spec.groups', '"g1"') AND starts_with(rsc->>'$.metadata.name', 'usr')`)
		err = row.Scan(&rsc)

	*/
}

func TestSSHSession(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	assert.Nil(t, err)

	for range 10 {
		id := vutils.GenerateLogID()
		for range 10 {
			sessionID := fmt.Sprintf("%s-%s", id, utilrand.GetRandomStringLowercase(6))

			{
				accessLog := &corev1.AccessLog{
					ApiVersion: ucorev1.APIVersion,
					Kind:       ucorev1.KindAccessLog,
					Metadata: &metav1.LogMetadata{
						Id: id,

						CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 100) * int(time.Minute)))),
					},

					Entry: &corev1.AccessLog_Entry{

						Common: &corev1.AccessLog_Entry_Common{
							StartedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-3 * time.Second)),
							EndedAt:   pbutils.Now(),
							Status:    corev1.AccessLog_Entry_Common_ALLOWED,
							RegionRef: umetav1.GetObjectReference(rgn),
							SessionID: sessionID,
						},

						Info: &corev1.AccessLog_Entry_Info{
							Type: &corev1.AccessLog_Entry_Info_Ssh{
								Ssh: &corev1.AccessLog_Entry_Info_SSH{
									Type: corev1.AccessLog_Entry_Info_SSH_SESSION_START,
								},
							},
						},
					},
				}

				jsn, err := pbutils.MarshalJSON(accessLog, false)
				assert.Nil(t, err)
				err = srv.insertAccessLog(jsn)
				assert.Nil(t, err)
			}

			for range 10 {
				accessLog := &corev1.AccessLog{
					ApiVersion: ucorev1.APIVersion,
					Kind:       ucorev1.KindAccessLog,
					Metadata: &metav1.LogMetadata{
						Id: id,

						CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 100) * int(time.Second)))),
					},

					Entry: &corev1.AccessLog_Entry{

						Common: &corev1.AccessLog_Entry_Common{
							StartedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-3 * time.Second)),
							EndedAt:   pbutils.Now(),
							Status:    corev1.AccessLog_Entry_Common_ALLOWED,
							RegionRef: umetav1.GetObjectReference(rgn),
							SessionID: sessionID,
						},
						Info: &corev1.AccessLog_Entry_Info{
							Type: &corev1.AccessLog_Entry_Info_Ssh{
								Ssh: &corev1.AccessLog_Entry_Info_SSH{
									Type: corev1.AccessLog_Entry_Info_SSH_SESSION_RECORDING,
									Details: &corev1.AccessLog_Entry_Info_SSH_SessionRecording_{
										SessionRecording: &corev1.AccessLog_Entry_Info_SSH_SessionRecording{
											Data: []byte(utilrand.GetRandomString(32)),
										},
									},
								},
							},
						},
					},
				}

				jsn, err := pbutils.MarshalJSON(accessLog, false)
				assert.Nil(t, err)
				err = srv.insertAccessLog(jsn)
				assert.Nil(t, err)
			}

			{
				accessLog := &corev1.AccessLog{
					ApiVersion: ucorev1.APIVersion,
					Kind:       ucorev1.KindAccessLog,
					Metadata: &metav1.LogMetadata{
						Id: id,

						CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 100) * int(time.Second)))),
					},

					Entry: &corev1.AccessLog_Entry{

						Common: &corev1.AccessLog_Entry_Common{
							StartedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-3 * time.Second)),
							EndedAt:   pbutils.Now(),
							Status:    corev1.AccessLog_Entry_Common_ALLOWED,
							RegionRef: umetav1.GetObjectReference(rgn),
							SessionID: sessionID,
						},

						Info: &corev1.AccessLog_Entry_Info{
							Type: &corev1.AccessLog_Entry_Info_Ssh{
								Ssh: &corev1.AccessLog_Entry_Info_SSH{
									Type: corev1.AccessLog_Entry_Info_SSH_SESSION_END,
								},
							},
						},
					},
				}

				jsn, err := pbutils.MarshalJSON(accessLog, false)
				assert.Nil(t, err)
				err = srv.insertAccessLog(jsn)
				assert.Nil(t, err)
			}

		}
	}

	resp, err := srv.listSSHSession(ctx, &visibilityv1.ListSSHSessionRequest{})
	assert.Nil(t, err)

	assert.True(t, len(resp.Items) > 0)

	{
		res, err := srv.getSSHSession(ctx, &visibilityv1.GetSSHSessionRequest{
			Id: resp.Items[0].Id,
		})
		assert.Nil(t, err)

		assert.True(t, pbutils.IsEqual(res, resp.Items[0]))
	}

	{
		resp, err := srv.listSSHSessionRecording(ctx, &visibilityv1.ListSSHSessionRecordingRequest{
			SessionID: resp.Items[0].Id,
		})
		assert.Nil(t, err)

		assert.True(t, len(resp.Items) == 10)

		for _, r := range resp.Items {
			//jsn, err := pbutils.MarshalJSON(r, false)
			assert.Nil(t, err)

			zap.L().Debug("-", zap.Any("rec", r), zap.String("g", r.Timestamp.AsTime().Format(time.TimeOnly)))
		}
	}
}

func TestAuditLog(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	{
		resp, err := srv.listAuditLog(ctx, &visibilityv1.ListAuditLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		for _ = range 1000 {
			lgJSON, err := pbutils.MarshalJSON(&enterprisev1.AuditLog{
				Metadata: &metav1.LogMetadata{
					Id:        vutils.GenerateLogID(),
					CreatedAt: pbutils.Now(),
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAuditLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1000, len(resp.Items))
	}
}

func TestAuthenticationLog(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	{
		resp, err := srv.listAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		userRef := &metav1.ObjectReference{
			Uid: vutils.UUIDv4(),
		}

		for range 1000 {
			lgJSON, err := pbutils.MarshalJSON(&enterprisev1.AuthenticationLog{
				Metadata: &metav1.LogMetadata{
					Id:        vutils.GenerateLogID(),
					CreatedAt: pbutils.Now(),
				},
				Entry: &enterprisev1.AuthenticationLog_Entry{
					UserRef: userRef,
					Authentication: &corev1.Session_Status_Authentication{
						Info: &corev1.Session_Status_Authentication_Info{
							Type: func() corev1.Session_Status_Authentication_Info_Type {
								return corev1.Session_Status_Authentication_Info_Type(utilrand.GetRandomRangeMath(0, 6))
							}(),
							Aal: corev1.Session_Status_Authentication_Info_AAL(utilrand.GetRandomRangeMath(0, 3)),
						},
					},
					AuthenticationIndex: 1,
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAuthenticationLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1000, len(resp.Items))

		{
			res, err := srv.getSummaryAuthenticationLog(ctx, &visibilityv1.GetAuthenticationLogSummaryRequest{})
			assert.Nil(t, err, "%+v", err)

			assert.Equal(t, 1000, int(res.TotalNumber))
			assert.Equal(t, 1, int(res.TotalUser))
			assert.Equal(t, 1000, int(res.TotalReauthentication))
		}
	}
}

func TestCleanup(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.doCleanup(ctx)
	assert.Nil(t, err)

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		for _ = range 1000 {
			lgJSON, err := pbutils.MarshalJSON(&corev1.AccessLog{
				Metadata: &metav1.LogMetadata{
					Id:        vutils.GenerateLogID(),
					CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-1 * (srv.cleanupDuration + time.Hour))),
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAccessLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1000, len(resp.Items))

		{
			res, err := srv.getSummaryAccessLog(ctx, &visibilityv1.GetAccessLogSummaryRequest{})
			assert.Nil(t, err)

			assert.Equal(t, 1000, int(res.TotalNumber))
		}
	}

	{
		resp, err := srv.listAuditLog(ctx, &visibilityv1.ListAuditLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		for _ = range 1000 {
			lgJSON, err := pbutils.MarshalJSON(&enterprisev1.AuditLog{
				Metadata: &metav1.LogMetadata{
					Id:        vutils.GenerateLogID(),
					CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-1 * (srv.cleanupDuration + time.Hour))),
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAuditLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1000, len(resp.Items))
	}

	{
		resp, err := srv.listAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		for _ = range 1000 {
			lgJSON, err := pbutils.MarshalJSON(&enterprisev1.AuthenticationLog{
				Metadata: &metav1.LogMetadata{
					Id:        vutils.GenerateLogID(),
					CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-1 * (srv.cleanupDuration + time.Hour))),
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAuthenticationLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1000, len(resp.Items))
	}

	err = srv.doCleanup(ctx)
	assert.Nil(t, err)

	{
		resp, err := srv.listAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))
	}
}

func TestCleanupLatestN(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	maxDBAccessLogs = 1000

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.doCleanup(ctx)
	assert.Nil(t, err)

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))

		for range maxDBAccessLogs + 1000 {
			lgJSON, err := pbutils.MarshalJSON(&corev1.AccessLog{
				Metadata: &metav1.LogMetadata{
					Id: vutils.GenerateLogID(),
					CreatedAt: pbutils.Timestamp(pbutils.Now().AsTime().Add(-1 *
						time.Duration(utilrand.GetRandomRangeMath(10, 100)*int(time.Second)))),
				},
			}, false)
			assert.Nil(t, err)
			err = srv.insertAccessLog(lgJSON)
			assert.Nil(t, err)
		}

		resp, err = srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, maxDBAccessLogs+1000, int(resp.ListResponseMeta.TotalCount))
	}

	err = srv.doCleanup(ctx)
	assert.Nil(t, err)

	// TODO: fix this test
	/*
		{
			resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
				Common: &vmetav1.CommonListOptions{
					ItemsPerPage: 1000,
				},
			})
			assert.Nil(t, err)
			assert.Equal(t, maxDBAccessLogs, int(resp.ListResponseMeta.TotalCount))
		}
	*/
}

func TestServerDataPoint(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	assert.Nil(t, err)

	coreSrv := admin.NewServer(&admin.Opts{
		IsEmbedded: true,
		OcteliumC:  srv.octeliumC,
	})

	usr, err := coreSrv.CreateUser(ctx, tests.GenUser(nil))
	assert.Nil(t, err)

	for range 5000 {
		accessLog := &corev1.AccessLog{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindAccessLog,
			Metadata: &metav1.LogMetadata{
				Id: vutils.GenerateLogID(),
				CreatedAt: pbutils.Timestamp(pbutils.Now().
					AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 12000) * int(time.Second)))),
			},

			Entry: &corev1.AccessLog_Entry{

				Common: &corev1.AccessLog_Entry_Common{
					StartedAt: pbutils.Timestamp(pbutils.Now().
						AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 120)) * time.Second)),
					EndedAt:   pbutils.Now(),
					Status:    corev1.AccessLog_Entry_Common_ALLOWED,
					RegionRef: umetav1.GetObjectReference(rgn),
					UserRef:   umetav1.GetObjectReference(usr),
				},
			},
		}

		jsn, err := pbutils.MarshalJSON(accessLog, false)
		assert.Nil(t, err)
		err = srv.insertAccessLog(jsn)
		assert.Nil(t, err)

	}

	{
		/*

			dataPoints, err := srv.doGetDatapoints(ctx, "access_logs", "rsc", LogQueryOptions{
				From:     time.Now().Add(-1 * time.Hour),
				To:       time.Now(),
				Interval: "1 minute",
			})
			assert.Nil(t, err)

			for _, itm := range dataPoints {
				if itm.count > 0 {
				}

			}
		*/

		{
			_, err := srv.getDataPoints(ctx, "access_logs",
				time.Now().Add(-2*time.Hour), time.Now(),
				&intervalDataPoint{
					Value: 30,
					Unit:  "minute",
				}, nil)
			assert.Nil(t, err)
		}

		{
			_, err := srv.getAccessLogDataPoint(ctx, &visibilityv1.GetAccessLogDataPointRequest{
				From: pbutils.Timestamp(time.Now().Add(-2 * time.Hour)),
				Interval: &metav1.Duration{
					Type: &metav1.Duration_Minutes{
						Minutes: 30,
					},
				},
			})
			assert.Nil(t, err)

		}
	}

}

func TestTop(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	assert.Nil(t, err)

	coreSrv := admin.NewServer(&admin.Opts{
		IsEmbedded: true,
		OcteliumC:  srv.octeliumC,
	})

	var usrList []*corev1.User

	var svcList []*corev1.Service

	for range 100 {
		usr, err := coreSrv.CreateUser(ctx, tests.GenUser(nil))
		assert.Nil(t, err)
		usrList = append(usrList, usr)
	}

	for range 100 {
		svc, err := coreSrv.CreateService(ctx, tests.GenService(""))
		assert.Nil(t, err)
		svcList = append(svcList, svc)
	}

	for range 5000 {
		accessLog := &corev1.AccessLog{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindAccessLog,
			Metadata: &metav1.LogMetadata{
				Id: vutils.GenerateLogID(),
				CreatedAt: pbutils.Timestamp(pbutils.Now().
					AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 120) * int(time.Second)))),
			},

			Entry: &corev1.AccessLog_Entry{

				Common: &corev1.AccessLog_Entry_Common{
					StartedAt: pbutils.Timestamp(pbutils.Now().
						AsTime().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 120)) * time.Second)),
					EndedAt:    pbutils.Now(),
					Status:     corev1.AccessLog_Entry_Common_ALLOWED,
					RegionRef:  umetav1.GetObjectReference(rgn),
					UserRef:    umetav1.GetObjectReference(usrList[utilrand.GetRandomRangeMath(0, len(usrList)-1)]),
					ServiceRef: umetav1.GetObjectReference(usrList[utilrand.GetRandomRangeMath(0, len(svcList)-1)]),
				},
			},
		}

		jsn, err := pbutils.MarshalJSON(accessLog, false)
		assert.Nil(t, err)
		err = srv.insertAccessLog(jsn)
		assert.Nil(t, err)

	}

	{
		res, err := srv.getTop(ctx, "access_logs", 10, colUserUID, "entry.common.userRef", nil)
		assert.Nil(t, err)

		assert.True(t, len(res.items) > 0)
	}

}

func TestLimitQueries(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	readInfo := &grpc.UnaryServerInfo{
		FullMethod: "/octelium.api.main.visibility.v1.AccessLogService/ListAccessLog",
	}
	ingestInfo := &grpc.UnaryServerInfo{
		FullMethod: otlpLogsServicePrefix + "Export",
	}

	noop := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	{
		release := make(chan struct{})
		entered := make(chan struct{}, maxConcurrentQueries)
		exited := make(chan struct{}, maxConcurrentQueries)

		for range maxConcurrentQueries {
			go func() {
				defer func() { exited <- struct{}{} }()
				ts.srv.limitQueries(ts.ctx, nil, readInfo,
					func(ctx context.Context, req any) (any, error) {
						entered <- struct{}{}
						<-release
						return nil, nil
					})
			}()
		}

		for range maxConcurrentQueries {
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				assert.Fail(t, "Query did not start")
			}
		}

		ctx, cancel := context.WithTimeout(ts.ctx, 200*time.Millisecond)
		_, err := ts.srv.limitQueries(ctx, nil, readInfo, noop)
		cancel()
		assert.NotNil(t, err)
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))

		ingestDone := make(chan struct{})
		go func() {
			defer close(ingestDone)
			_, err := ts.srv.limitQueries(ts.ctx, nil, ingestInfo, noop)
			assert.Nil(t, err, "%+v", err)
		}()

		select {
		case <-ingestDone:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "Ingest queued behind reads")
		}

		close(release)

		for range maxConcurrentQueries {
			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				assert.Fail(t, "Query did not exit")
			}
		}
	}

	{
		for range maxConcurrentQueries * 2 {
			_, err := ts.srv.limitQueries(ts.ctx, nil, readInfo, noop)
			assert.Nil(t, err, "%+v", err)
		}
	}

	{
		_, err := ts.srv.limitQueries(ts.ctx, nil, readInfo,
			func(ctx context.Context, req any) (any, error) {
				deadline, ok := ctx.Deadline()
				assert.True(t, ok)
				assert.True(t, time.Until(deadline) <= queryTimeout)
				return nil, nil
			})
		assert.Nil(t, err, "%+v", err)
	}
}

func newLogStoreRPCClient(t *testing.T, ts *testServer) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.UnaryInterceptor(ts.srv.limitQueries))
	processor, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	plogotlp.RegisterGRPCServer(srv, processor)
	visibilityv1.RegisterAccessLogServiceServer(srv, &srvAccessLog{s: ts.srv})
	visibilityv1.RegisterAuthenticationLogServiceServer(srv, &srvAuthenticationLog{s: ts.srv})
	visibilityv1.RegisterAuditLogServiceServer(srv, &srvAuditLog{s: ts.srv})
	visibilityv1.RegisterComponentLogServiceServer(srv, &srvComponentLog{s: ts.srv})
	vllmv1.RegisterLLMServiceServer(srv, &srvLLM{s: ts.srv})
	go func() { _ = srv.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///logstore-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	require.Nil(t, err)
	t.Cleanup(func() {
		assert.Nil(t, conn.Close())
		srv.Stop()
		assert.Nil(t, listener.Close())
	})
	return conn
}

func logStoreRPCMethods() []protoreflect.MethodDescriptor {
	var ret []protoreflect.MethodDescriptor
	for _, service := range []protoreflect.ServiceDescriptor{
		visibilityv1.File_visibilityv1_proto.Services().ByName("AccessLogService"),
		visibilityv1.File_visibilityv1_proto.Services().ByName("AuthenticationLogService"),
		visibilityv1.File_visibilityv1_proto.Services().ByName("AuditLogService"),
		visibilityv1.File_visibilityv1_proto.Services().ByName("ComponentLogService"),
		vllmv1.File_vllmv1_proto.Services().ByName("LLMService"),
	} {
		for i := range service.Methods().Len() {
			ret = append(ret, service.Methods().Get(i))
		}
	}
	return ret
}

func newLogStoreRPCRequest(method protoreflect.MethodDescriptor) *dynamicpb.Message {
	ret := dynamicpb.NewMessage(method.Input())
	if string(method.Name()) == "ListTopDimension" {
		ret.Set(ret.Descriptor().Fields().ByName("dimension"), protoreflect.ValueOfEnum(protoreflect.EnumNumber(vllmv1.Dimension_STATUS)))
	}
	return ret
}

func logStoreRPCName(method protoreflect.MethodDescriptor) string {
	return "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
}

func TestLogStoreRPCDefaultsAndFailures(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	conn := newLogStoreRPCClient(t, ts)
	for _, method := range logStoreRPCMethods() {
		t.Run(string(method.FullName()), func(t *testing.T) {
			res := dynamicpb.NewMessage(method.Output())
			err := conn.Invoke(ts.ctx, logStoreRPCName(method), newLogStoreRPCRequest(method), res)
			if string(method.Name()) == "GetSSHSession" {
				assert.Equal(t, codes.NotFound, status.Code(err))
				return
			}
			require.Nil(t, err)
			fields := res.Descriptor().Fields()
			if field := fields.ByName("items"); field != nil {
				assert.Zero(t, res.Get(field).List().Len())
			}
			for _, name := range []protoreflect.Name{"totalCount", "totalNumber", "totalOther"} {
				if field := fields.ByName(name); field != nil {
					assert.EqualValues(t, 0, res.Get(field).Interface())
				}
			}
			if field := fields.ByName("previous"); field != nil {
				assert.False(t, res.Has(field))
			}
		})
	}
	t.Run("DatabaseUnavailable", func(t *testing.T) {
		require.Nil(t, ts.srv.db.Close())
		for _, method := range logStoreRPCMethods() {
			res := dynamicpb.NewMessage(method.Output())
			err := conn.Invoke(ts.ctx, logStoreRPCName(method), newLogStoreRPCRequest(method), res)
			assert.NotNil(t, err, "%s", method.FullName())
		}
	})
}

func TestLogStoreRPCReferenceValidation(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	conn := newLogStoreRPCClient(t, ts)
	for _, method := range logStoreRPCMethods() {
		request := newLogStoreRPCRequest(method)
		container := request.ProtoReflect()
		if field := request.Descriptor().Fields().ByName("filter"); field != nil {
			container = request.Mutable(field).Message()
		}
		fields := container.Descriptor().Fields()
		for i := range fields.Len() {
			field := fields.Get(i)
			if field.Name() == "matchInlinePolicyRef" || (method.Name() == "GetAuditLogSummary" && field.Name() == "resourceRef") {
				continue
			}
			if field.Kind() != protoreflect.MessageKind || field.Message().FullName() != "octelium.api.main.meta.v1.ObjectReference" {
				continue
			}
			t.Run(string(method.FullName())+"/"+string(field.Name()), func(t *testing.T) {
				for _, ref := range []*metav1.ObjectReference{{}, {Uid: "invalid-uid"}} {
					container.Set(field, protoreflect.ValueOfMessage(ref.ProtoReflect()))
					res := dynamicpb.NewMessage(method.Output())
					err := conn.Invoke(ts.ctx, logStoreRPCName(method), request, res)
					assert.Equal(t, codes.InvalidArgument, status.Code(err))
				}
				container.Clear(field)
			})
		}
	}
}

func TestLogStoreRPCIngestionVisibility(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	conn := newLogStoreRPCClient(t, ts)
	now := time.Now().UTC()
	access := newAccessLog(&accessLogOptions{CreatedAt: now})
	authentication := newAuthenticationLog(&authenticationLogOptions{CreatedAt: now})
	audit := newAuditLog(&auditLogOptions{CreatedAt: now})
	component := newComponentLog(now, corev1.ComponentLog_Entry_INFO, "rpc-ingestion")
	_, err := plogotlp.NewGRPCClient(conn).Export(ts.ctx, newExportRequest(
		marshalLog(t, access), marshalLog(t, authentication), marshalLog(t, audit), marshalLog(t, component),
	))
	require.Nil(t, err)
	for _, method := range logStoreRPCMethods() {
		var want proto.Message
		switch method.Name() {
		case "ListAccessLog":
			if method.Parent().Name() == "AccessLogService" {
				want = access
			}
		case "ListAuthenticationLog":
			want = authentication
		case "ListAuditLog":
			want = audit
		case "ListComponentLog":
			want = component
		}
		if want == nil {
			continue
		}
		res := dynamicpb.NewMessage(method.Output())
		require.Nil(t, conn.Invoke(ts.ctx, logStoreRPCName(method), newLogStoreRPCRequest(method), res))
		items := res.Get(res.Descriptor().Fields().ByName("items")).List()
		require.Equal(t, 1, items.Len())
		got := want.ProtoReflect().Type().New().Interface()
		data, err := proto.Marshal(items.Get(0).Message().Interface())
		require.Nil(t, err)
		require.Nil(t, proto.Unmarshal(data, got))
		assert.True(t, proto.Equal(want, got))
	}
}

func TestLogQueryLimiterReleasesSlots(t *testing.T) {
	srv := &Server{querySem: make(chan struct{}, 1)}
	info := &grpc.UnaryServerInfo{FullMethod: "/visibility/Test"}
	wantErr := status.Error(codes.Internal, "handler failed")
	before := time.Now()
	_, err := srv.limitQueries(context.Background(), "request", info, func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, before.Add(queryTimeout), deadline, time.Second)
		assert.Equal(t, "request", req)
		return nil, wantErr
	})
	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, srv.querySem)
	assert.Panics(t, func() {
		_, _ = srv.limitQueries(context.Background(), nil, info, func(context.Context, any) (any, error) { panic("handler") })
	})
	assert.Empty(t, srv.querySem)
}

func TestLogStoreCleanupLoopCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); (&Server{}).startCleanupLoop(ctx) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Cleanup loop did not stop")
	}
}

func TestLogStoreClosedDatabaseStartup(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	require.Nil(t, ts.srv.db.Close())
	assert.NotNil(t, ts.srv.Run(ts.ctx))
}

func TestLogRPCComponentSelectorBoundaries(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	conn := newLogStoreRPCClient(t, ts)
	for _, method := range logStoreRPCMethods() {
		if method.Parent().Name() != "ComponentLogService" {
			continue
		}
		request := newLogStoreRPCRequest(method)
		componentField := request.Descriptor().Fields().ByName("component")
		if componentField == nil {
			continue
		}
		component := request.Mutable(componentField).Message()
		for _, name := range []protoreflect.Name{"type", "namespace", "uid"} {
			field := component.Descriptor().Fields().ByName(name)
			for _, length := range []int{120, 121} {
				component.Set(field, protoreflect.ValueOfString(strings.Repeat("x", length)))
				res := dynamicpb.NewMessage(method.Output())
				err := conn.Invoke(ts.ctx, logStoreRPCName(method), request, res)
				if length == 120 {
					assert.Nil(t, err)
				} else {
					assert.Equal(t, codes.InvalidArgument, status.Code(err))
				}
			}
			component.Clear(field)
		}
		levelsField := request.Descriptor().Fields().ByName("levels")
		if levelsField == nil {
			continue
		}
		levels := request.Mutable(levelsField).List()
		for range maxComponentLevels + 1 {
			levels.Append(protoreflect.ValueOfEnum(0))
		}
		err := conn.Invoke(ts.ctx, logStoreRPCName(method), request, dynamicpb.NewMessage(method.Output()))
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestLogStoreListenerBindFailure(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	addr := tstAddr
	if ovutils.IsMockMode() {
		addr = "localhost:40001"
	}
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, listener.Close()) })
	ctx, cancel := context.WithCancel(ts.ctx)
	defer cancel()

	err = ts.srv.Run(ctx)
	require.Error(t, err)
	var netErr *net.OpError
	require.ErrorAs(t, err, &netErr)
	assert.Equal(t, "listen", netErr.Op)
}

func TestLogStoreGRPCLifecycle(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	addr := tstAddr
	if ovutils.IsMockMode() {
		addr = "localhost:40001"
	}

	for range 2 {
		ctx, cancel := context.WithCancel(ts.ctx)
		t.Cleanup(cancel)
		require.NoError(t, ts.srv.Run(ctx))
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, conn.Close()) })
		rpcCtx, rpcCancel := context.WithTimeout(ts.ctx, 5*time.Second)
		_, err = plogotlp.NewGRPCClient(conn).Export(rpcCtx, newExportRequest(
			marshalLog(t, newComponentLog(time.Now().UTC(), corev1.ComponentLog_Entry_INFO, "listener")),
		))
		rpcCancel()
		require.NoError(t, err)
		cancel()

		require.Eventually(t, func() bool {
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return false
			}
			assert.NoError(t, listener.Close())
			return true
		}, 3*time.Second, 10*time.Millisecond)
	}
	assert.Equal(t, 2, getTableCount(t, ts.srv, "component_logs"))
}
