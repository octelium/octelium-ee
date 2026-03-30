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
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
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
	"go.uber.org/zap"
)

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

	assert.True(t, len(resp.Items) == 100)

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

	{
		resp, err := srv.listAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, maxDBAccessLogs, int(resp.ListResponseMeta.TotalCount))
	}
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
		res, err := srv.getTop(ctx, "access_logs", 10, "entry.common.userRef", nil)
		assert.Nil(t, err)

		assert.True(t, len(res.items) > 0)
	}

}
