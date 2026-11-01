// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const collectorRemoteServiceName = "collector.octelium"

const (
	collectorRemoteLogsService    = "opentelemetry.proto.collector.logs.v1.LogsService"
	collectorRemoteMetricsService = "opentelemetry.proto.collector.metrics.v1.MetricsService"
)

func withCollectorRemoteToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, "x-octelium-auth", token)
}

func newCollectorRemoteLogs(t *testing.T, componentUID string) plogotlp.ExportRequest {
	t.Helper()

	body, err := pbutils.MarshalJSON(&corev1.ComponentLog{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindComponentLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Now(),
		},
		Entry: &corev1.ComponentLog_Entry{
			Level:   corev1.ComponentLog_Entry_INFO,
			Message: "e2e remote collector log",
			Time:    pbutils.Now(),
			Component: &corev1.ComponentLog_Entry_Component{
				Namespace: "octeliumee",
				Type:      "e2e-collector-remote",
				Uid:       componentUID,
			},
		},
	}, false)
	require.NoError(t, err)

	ret := plogotlp.NewExportRequest()
	ret.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(string(body))
	return ret
}

func newCollectorRemoteMetrics(name string) pmetricotlp.ExportRequest {
	md := pmetric.NewMetrics()
	metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName(name)
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().UTC()))
	dp.SetIntValue(1)
	return pmetricotlp.NewExportRequestFromMetrics(md)
}

func testCollectorRemoteService(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	t.Run("ServiceIsInstalled", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		svc, err := h.CoreC().GetService(ctx, &metav1.GetOptions{Name: collectorRemoteServiceName})
		require.NoError(t, err)
		assert.True(t, svc.Metadata.IsSystem)
		assert.True(t, svc.Metadata.IsUserHidden)
		assert.True(t, svc.Spec.IsPublic)
		assert.False(t, svc.Spec.IsAnonymous)
		assert.Equal(t, corev1.Service_Spec_GRPC, svc.Spec.Mode)
	})

	conn, err := grpc.NewClient(fmt.Sprintf("%s.%s:443", collectorRemoteServiceName, h.Domain),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	logsC := plogotlp.NewGRPCClient(conn)
	metricsC := pmetricotlp.NewGRPCClient(conn)

	t.Run("AnonymousRequestsAreRejected", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		_, err := logsC.Export(ctx, newCollectorRemoteLogs(t, vutils.UUIDv4()))
		require.Error(t, err)
		assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, status.Code(err))

		_, err = metricsC.Export(ctx, newCollectorRemoteMetrics(h.Name()))
		require.Error(t, err)
		assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, status.Code(err))
	})

	t.Run("OtherUsersAreDenied", func(t *testing.T) {
		usr := h.CreateUser(t, &corev1.User{
			Spec: &corev1.User_Spec{
				Type: corev1.User_Spec_WORKLOAD,
				Authorization: &corev1.User_Spec_Authorization{
					InlinePolicies: []*corev1.InlinePolicy{
						{
							Spec: &corev1.Policy_Spec{
								Rules: []*corev1.Policy_Spec_Rule{
									{
										Effect:   corev1.Policy_Spec_Rule_ALLOW,
										Priority: -4,
										Condition: &corev1.Condition{
											Type: &corev1.Condition_MatchAny{
												MatchAny: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
		token := h.AccessToken(t, usr)

		h.Within(t, "a broadly authorized User to be denied", harness.DecisionBudget, func(ctx context.Context) error {
			_, err := logsC.Export(withCollectorRemoteToken(ctx, token), newCollectorRemoteLogs(t, vutils.UUIDv4()))
			if status.Code(err) != codes.PermissionDenied {
				return errors.Errorf("got %v, want PermissionDenied", err)
			}
			_, err = metricsC.Export(withCollectorRemoteToken(ctx, token), newCollectorRemoteMetrics(h.Name()))
			if status.Code(err) != codes.PermissionDenied {
				return errors.Errorf("got %v, want PermissionDenied", err)
			}
			return nil
		})
	})

	t.Run("OcteliumUserCredentialsAreDenied", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		usr, err := h.CoreC().GetUser(ctx, &metav1.GetOptions{Name: "octelium"})
		require.NoError(t, err)

		for i, typ := range []corev1.Credential_Spec_Type{
			corev1.Credential_Spec_AUTH_TOKEN,
			corev1.Credential_Spec_OAUTH2,
			corev1.Credential_Spec_ACCESS_TOKEN,
		} {
			cred, err := h.CoreC().CreateCredential(ctx, &corev1.Credential{
				Metadata: &metav1.Metadata{Name: fmt.Sprintf("%s-%d", h.Name(), i)},
				Spec: &corev1.Credential_Spec{
					User: usr.Metadata.Name,
					Type: typ,
				},
			})
			if err == nil {
				t.Cleanup(func() {
					h.CoreC().DeleteCredential(context.Background(), &metav1.DeleteOptions{Uid: cred.Metadata.Uid})
				})
			}
			require.Error(t, err, "%s", typ)
			assert.Equal(t, codes.PermissionDenied, status.Code(err), "%s: %+v", typ, err)
		}

		creds, err := h.CoreC().ListCredential(ctx, &corev1.ListCredentialOptions{
			UserRef: umetav1.GetObjectReference(usr),
		})
		require.NoError(t, err)
		assert.Empty(t, creds.Items)
	})
}
