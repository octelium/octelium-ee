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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
	client    plogotlp.GRPCClient
}

func getAddr() string {
	if ldflags.IsTest() {
		return "localhost:54321"
	}

	return "octeliumee-logstore.octelium.svc:8080"
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	grpcOpts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.DefaultConfig,
		}),
	}

	{
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(getAddr(), grpcOpts...)
	if err != nil {
		return nil, err
	}

	return &Controller{
		octeliumC: octeliumC,
		client:    plogotlp.NewGRPCClient(conn),
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, sess *corev1.Session) error {

	if time.Now().After(sess.Metadata.CreatedAt.AsTime().Add(1*time.Minute)) || sess.Metadata.UpdatedAt != nil {
		return nil
	}

	return c.doAuthenticationLog(ctx, sess)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Session) error {
	if pbutils.IsEqual(new.Status.Authentication, old.Status.Authentication) {
		return nil
	}

	return c.doAuthenticationLog(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, sess *corev1.Session) error {

	return nil
}

func (c *Controller) doAuthenticationLog(ctx context.Context, sess *corev1.Session) error {
	auth := sess.Status.Authentication
	lg := &enterprisev1.AuthenticationLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuthenticationLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: auth.SetAt,
		},
		Entry: &enterprisev1.AuthenticationLog_Entry{
			SessionRef:     umetav1.GetObjectReference(sess),
			UserRef:        sess.Status.UserRef,
			DeviceRef:      sess.Status.DeviceRef,
			Authentication: auth,
			AuthenticationIndex: func() uint32 {
				if sess.Status.TotalAuthentications < 1 {
					return 0
				}
				return sess.Status.TotalAuthentications - 1
			}(),
		},
	}

	zap.L().Debug("Created authenticationLog", zap.Any("log", lg))

	lgs := plog.NewLogs()
	lgs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	logRecords := lgs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	lr := logRecords.AppendEmpty()

	convertLogRecord(lg, lr)

	_, err := c.client.Export(ctx, plogotlp.NewExportRequestFromLogs(lgs))
	return err
}

func convertLogRecord(in *enterprisev1.AuthenticationLog, ret plog.LogRecord) {
	inMap := pbutils.MustConvertToMap(in)

	ret.SetTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetObservedTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetSeverityNumber(plog.SeverityNumberInfo)
	ret.SetSeverityText(plog.SeverityNumberInfo.String())
	ret.Body().SetEmptyMap().FromRaw(inMap)
}
