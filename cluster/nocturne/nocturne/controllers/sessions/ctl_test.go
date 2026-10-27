// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package sessioncontroller

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rpubsubv1"
	"github.com/octelium/octelium/cluster/common/authevents"
	"github.com/octelium/octelium/cluster/common/redisutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"google.golang.org/grpc"
)

type fakeLogClient struct {
	plogotlp.GRPCClient

	mu         sync.Mutex
	logs       []*enterprisev1.AuthenticationLog
	severities []plog.SeverityNumber
	err        error
}

func (c *fakeLogClient) Export(ctx context.Context,
	req plogotlp.ExportRequest, opts ...grpc.CallOption) (plogotlp.ExportResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return plogotlp.NewExportResponse(), c.err
	}

	resourceLogs := req.Logs().ResourceLogs()
	for i := range resourceLogs.Len() {
		scopeLogs := resourceLogs.At(i).ScopeLogs()
		for j := range scopeLogs.Len() {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := range logRecords.Len() {
				lg := &enterprisev1.AuthenticationLog{}
				if err := pbutils.UnmarshalFromMap(logRecords.At(k).Body().Map().AsRaw(), lg); err != nil {
					return plogotlp.NewExportResponse(), err
				}
				c.logs = append(c.logs, lg)
				c.severities = append(c.severities, logRecords.At(k).SeverityNumber())
			}
		}
	}

	return plogotlp.NewExportResponse(), nil
}

func (c *fakeLogClient) getLogs() []*enterprisev1.AuthenticationLog {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]*enterprisev1.AuthenticationLog{}, c.logs...)
}

type fakePubSub struct {
	rpubsubv1.MainServiceClient

	mu    sync.Mutex
	acked [][]byte
}

func (c *fakePubSub) Acknowledge(ctx context.Context,
	req *rpubsubv1.AcknowledgeRequest, opts ...grpc.CallOption) (*rpubsubv1.AcknowledgeResponse, error) {
	c.mu.Lock()
	c.acked = append(c.acked, req.Ids...)
	c.mu.Unlock()

	return c.MainServiceClient.Acknowledge(ctx, req, opts...)
}

func (c *fakePubSub) getAcked() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([][]byte{}, c.acked...)
}

type fakeOcteliumC struct {
	octeliumc.ClientInterface
	pubSubC *fakePubSub
}

func (c *fakeOcteliumC) PubSubC() rpubsubv1.MainServiceClient {
	return c.pubSubC
}

func objRef(kind string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       kind,
		Name:       utilrand.GetRandomStringCanonical(8),
		Uid:        utilrand.GetRandomStringCanonical(16),
	}
}

func newFailureMessage(t *testing.T, failure *authevents.Failure) *rpubsubv1.Message {
	msg, err := failure.ToMessage()
	assert.Nil(t, err)
	return msg
}

func TestGetFailureAuthenticationLog(t *testing.T) {
	failure := &authevents.Failure{
		Reason:     authevents.ReasonRevokedCredential,
		CreatedAt:  time.Now().Add(-1 * time.Minute),
		SessionRef: objRef("Session"),
		UserRef:    objRef("User"),
		DeviceRef:  objRef("Device"),
		Info: &corev1.Session_Status_Authentication_Info{
			Type: corev1.Session_Status_Authentication_Info_REFRESH_TOKEN,
			Downstream: &corev1.Session_Status_Authentication_Info_Downstream{
				IpAddress: "203.0.113.7",
			},
		},
	}

	lg := getFailureAuthenticationLog(failure)
	assert.Equal(t, uenterprisev1.APIVersion, lg.ApiVersion)
	assert.Equal(t, uenterprisev1.KindAuthenticationLog, lg.Kind)
	assert.NotEmpty(t, lg.Metadata.Id)
	assert.True(t, failure.CreatedAt.Equal(lg.Metadata.CreatedAt.AsTime()))
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_FAILURE, lg.Entry.Status)
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_Failure_REVOKED_CREDENTIAL, lg.Entry.Failure.Reason)
	assert.True(t, pbutils.IsEqual(failure.SessionRef, lg.Entry.SessionRef))
	assert.True(t, pbutils.IsEqual(failure.UserRef, lg.Entry.UserRef))
	assert.True(t, pbutils.IsEqual(failure.DeviceRef, lg.Entry.DeviceRef))
	assert.True(t, pbutils.IsEqual(failure.Info, lg.Entry.Authentication.Info))
	assert.True(t, failure.CreatedAt.Equal(lg.Entry.Authentication.SetAt.AsTime()))
	assert.Equal(t, "", lg.Entry.Authentication.TokenID)
	assert.Equal(t, uint32(0), lg.Entry.AuthenticationIndex)

	for reason, expected := range map[authevents.Reason]enterprisev1.AuthenticationLog_Entry_Failure_Reason{
		authevents.ReasonInvalidCredential:      enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL,
		authevents.ReasonExpiredCredential:      enterprisev1.AuthenticationLog_Entry_Failure_EXPIRED_CREDENTIAL,
		authevents.ReasonRevokedCredential:      enterprisev1.AuthenticationLog_Entry_Failure_REVOKED_CREDENTIAL,
		authevents.ReasonUserNotFound:           enterprisev1.AuthenticationLog_Entry_Failure_USER_NOT_FOUND,
		authevents.ReasonUserDeactivated:        enterprisev1.AuthenticationLog_Entry_Failure_USER_DEACTIVATED,
		authevents.ReasonUserLocked:             enterprisev1.AuthenticationLog_Entry_Failure_USER_LOCKED,
		authevents.ReasonDeniedByRule:           enterprisev1.AuthenticationLog_Entry_Failure_DENIED_BY_RULE,
		authevents.ReasonNotAllowed:             enterprisev1.AuthenticationLog_Entry_Failure_NOT_ALLOWED,
		authevents.ReasonLimitExceeded:          enterprisev1.AuthenticationLog_Entry_Failure_LIMIT_EXCEEDED,
		authevents.ReasonIdentityProviderError:  enterprisev1.AuthenticationLog_Entry_Failure_IDENTITY_PROVIDER_ERROR,
		authevents.ReasonInternalError:          enterprisev1.AuthenticationLog_Entry_Failure_INTERNAL_ERROR,
		authevents.Reason(""):                   enterprisev1.AuthenticationLog_Entry_Failure_REASON_UNSET,
		authevents.Reason("SOME_FUTURE_REASON"): enterprisev1.AuthenticationLog_Entry_Failure_REASON_UNSET,
	} {
		lg := getFailureAuthenticationLog(&authevents.Failure{
			Reason:    reason,
			CreatedAt: time.Now(),
		})
		assert.Equal(t, expected, lg.Entry.Failure.Reason, string(reason))
		assert.Nil(t, lg.Entry.UserRef)
	}
}

func TestDoAuthenticationLog(t *testing.T) {
	ctx := context.Background()
	client := &fakeLogClient{}
	ctrl := &Controller{
		client: client,
	}

	sess := &corev1.Session{
		ApiVersion: "core/v1",
		Kind:       "Session",
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
			Uid:  utilrand.GetRandomStringCanonical(16),
		},
		Status: &corev1.Session_Status{
			UserRef:              objRef("User"),
			TotalAuthentications: 2,
			Authentication: &corev1.Session_Status_Authentication{
				SetAt: pbutils.Now(),
				Info: &corev1.Session_Status_Authentication_Info{
					Type: corev1.Session_Status_Authentication_Info_REFRESH_TOKEN,
				},
			},
		},
	}

	assert.Nil(t, ctrl.doAuthenticationLog(ctx, sess))

	logs := client.getLogs()
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_SUCCESS, logs[0].Entry.Status)
	assert.Nil(t, logs[0].Entry.Failure)
	assert.Equal(t, uint32(1), logs[0].Entry.AuthenticationIndex)
	assert.Equal(t, sess.Metadata.Uid, logs[0].Entry.SessionRef.Uid)
	assert.Equal(t, plog.SeverityNumberInfo, client.severities[0])
}

func TestRunFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	assert.Nil(t, redisutils.NewClient().Del(ctx, fmt.Sprintf("octelium:pubsub:%s",
		vutils.Sha256SumHex([]byte(authevents.TopicFailure)))).Err())

	pubSubC := &fakePubSub{
		MainServiceClient: tst.C.OcteliumC.PubSubC(),
	}
	octeliumC := &fakeOcteliumC{
		ClientInterface: tst.C.OcteliumC,
		pubSubC:         pubSubC,
	}

	{
		client := &fakeLogClient{
			err: errors.New("logstore is unavailable"),
		}
		ctrl := &Controller{
			octeliumC: octeliumC,
			client:    client,
		}

		err := ctrl.handleFailures(ctx, []*rpubsubv1.Message{
			newFailureMessage(t, &authevents.Failure{
				Reason:    authevents.ReasonInvalidCredential,
				CreatedAt: time.Now(),
			}),
		})
		assert.NotNil(t, err)
		assert.Equal(t, 0, len(pubSubC.getAcked()))
	}

	client := &fakeLogClient{}
	ctrl := &Controller{
		octeliumC: octeliumC,
		client:    client,
	}

	sessRef := objRef("Session")
	res, err := tst.C.OcteliumC.PubSubC().Publish(ctx, &rpubsubv1.PublishRequest{
		Topic: []byte(authevents.TopicFailure),
		Messages: []*rpubsubv1.Message{
			newFailureMessage(t, &authevents.Failure{
				Reason:     authevents.ReasonExpiredCredential,
				CreatedAt:  time.Now(),
				SessionRef: sessRef,
				Info: &corev1.Session_Status_Authentication_Info{
					Type: corev1.Session_Status_Authentication_Info_REFRESH_TOKEN,
				},
			}),
			{
				Type: &rpubsubv1.Message_Data{
					Data: []byte("invalid"),
				},
			},
			newFailureMessage(t, &authevents.Failure{
				Reason:    authevents.ReasonUserNotFound,
				CreatedAt: time.Now(),
			}),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res.Ids))

	ctrl.RunFailures(ctx)

	assert.Eventually(t, func() bool {
		return len(client.getLogs()) == 2 && len(pubSubC.getAcked()) == 3
	}, 30*time.Second, 100*time.Millisecond)

	logs := client.getLogs()
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_FAILURE, logs[0].Entry.Status)
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_Failure_EXPIRED_CREDENTIAL, logs[0].Entry.Failure.Reason)
	assert.Equal(t, sessRef.Uid, logs[0].Entry.SessionRef.Uid)
	assert.Equal(t, corev1.Session_Status_Authentication_Info_REFRESH_TOKEN, logs[0].Entry.Authentication.Info.Type)
	assert.Equal(t, enterprisev1.AuthenticationLog_Entry_Failure_USER_NOT_FOUND, logs[1].Entry.Failure.Reason)
	assert.Equal(t, plog.SeverityNumberWarn, client.severities[0])
	assert.ElementsMatch(t, res.Ids, pubSubC.getAcked())

	{
		res, err := tst.C.OcteliumC.PubSubC().Publish(ctx, &rpubsubv1.PublishRequest{
			Topic: []byte(authevents.TopicFailure),
			Messages: []*rpubsubv1.Message{
				newFailureMessage(t, &authevents.Failure{
					Reason:    authevents.ReasonLimitExceeded,
					CreatedAt: time.Now(),
				}),
			},
			IfExists: true,
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(res.Ids))

		assert.Eventually(t, func() bool {
			return len(client.getLogs()) == 3
		}, 30*time.Second, 100*time.Millisecond)
		assert.Equal(t, enterprisev1.AuthenticationLog_Entry_Failure_LIMIT_EXCEEDED,
			client.getLogs()[2].Entry.Failure.Reason)
	}
}

func TestSessionEventLoggingAndReplayGuards(t *testing.T) {
	logs := &fakeLogClient{}
	ctl := &Controller{client: logs}
	sess := &corev1.Session{
		Metadata: &metav1.Metadata{Uid: vutils.UUIDv4(), Name: "session", CreatedAt: pbutils.Now()},
		Status:   &corev1.Session_Status{TotalAuthentications: 1, Authentication: &corev1.Session_Status_Authentication{SetAt: pbutils.Now()}},
	}
	require.NoError(t, ctl.OnAdd(t.Context(), sess))
	require.Len(t, logs.getLogs(), 1)
	assert.Zero(t, logs.getLogs()[0].Entry.AuthenticationIndex)
	old := pbutils.Clone(sess).(*corev1.Session)
	sess.Metadata.UpdatedAt = pbutils.Now()
	require.NoError(t, ctl.OnAdd(t.Context(), sess))
	require.NoError(t, ctl.OnUpdate(t.Context(), sess, old))
	assert.Len(t, logs.getLogs(), 1)
	sess.Status.TotalAuthentications = 3
	sess.Status.Authentication.SetAt = pbutils.Timestamp(time.Now().Add(time.Second))
	require.NoError(t, ctl.OnUpdate(t.Context(), sess, old))
	require.Len(t, logs.getLogs(), 2)
	assert.EqualValues(t, 2, logs.getLogs()[1].Entry.AuthenticationIndex)
	old.Metadata.CreatedAt = pbutils.Timestamp(time.Now().Add(-2 * time.Minute))
	require.NoError(t, ctl.OnAdd(t.Context(), old))
	require.NoError(t, ctl.OnDelete(t.Context(), old))
	assert.Len(t, logs.getLogs(), 2)
}

type ackFailureClient struct {
	rpubsubv1.MainServiceClient
	err error
}

func (c *ackFailureClient) Acknowledge(context.Context, *rpubsubv1.AcknowledgeRequest, ...grpc.CallOption) (*rpubsubv1.AcknowledgeResponse, error) {
	return &rpubsubv1.AcknowledgeResponse{}, c.err
}

func TestFailureBatchAcknowledgementErrorsAndPoisonMessages(t *testing.T) {
	for _, invalidOnly := range []bool{false, true} {
		failure := errors.New("acknowledgement unavailable")
		pubsub := &fakePubSub{MainServiceClient: &ackFailureClient{err: failure}}
		logs := &fakeLogClient{}
		ctl := &Controller{octeliumC: &fakeOcteliumC{pubSubC: pubsub}, client: logs}
		msgs := []*rpubsubv1.Message{{Id: []byte("invalid"), Type: &rpubsubv1.Message_Data{Data: []byte("invalid")}}}
		if !invalidOnly {
			msg := newFailureMessage(t, &authevents.Failure{CreatedAt: time.Now(), Reason: authevents.ReasonInvalidCredential})
			msg.Id = []byte("valid")
			msgs = append(msgs, msg)
		}
		assert.ErrorIs(t, ctl.handleFailures(t.Context(), msgs), failure)
		assert.Len(t, pubsub.getAcked(), len(msgs))
		if invalidOnly {
			assert.Empty(t, logs.getLogs())
		} else {
			assert.Len(t, logs.getLogs(), 1)
		}
	}
}

func TestFailureSubscriptionStopsOnCanceledContext(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	ctl := &Controller{octeliumC: tst.C.OcteliumC, client: &fakeLogClient{}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.Error(t, ctl.doRunFailures(ctx))
	done := make(chan struct{})
	go func() { defer close(done); ctl.runFailures(ctx) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("failure subscription did not stop")
	}
}
