// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpexporter

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/octelium/octelium/cluster/common/spiffec"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

type baseExporter struct {
	config  *Config
	set     exporter.Settings
	telset  component.TelemetrySettings

	conn          *grpc.ClientConn
	logExporter   plogotlp.GRPCClient
	metricExporter pmetricotlp.GRPCClient

	metadata    metadata.MD
	callOptions []grpc.CallOption
	userAgent   string
}

func newExporter(cfg *Config, set exporter.Settings) *baseExporter {
	userAgent := fmt.Sprintf("%s/%s (%s/%s)",
		set.BuildInfo.Description,
		set.BuildInfo.Version,
		runtime.GOOS,
		runtime.GOARCH,
	)

	return &baseExporter{
		config:    cfg,
		set:       set,
		telset:    set.TelemetrySettings,
		userAgent: userAgent,
	}
}

func (e *baseExporter) start(ctx context.Context, _ component.Host) error {
	credOpt, err := spiffec.GetGRPCClientCred(ctx, nil)
	if err != nil {
		return err
	}

	dialOpts := []grpc.DialOption{
		credOpt,
		grpc.WithUserAgent(e.userAgent),
	}

	if e.config.MaxCallRecvMsgSizeMiB > 0 {
		dialOpts = append(dialOpts,
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(e.config.MaxCallRecvMsgSizeMiB<<20)),
		)
	}

	if e.config.MaxCallSendMsgSizeMiB > 0 {
		dialOpts = append(dialOpts,
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(e.config.MaxCallSendMsgSizeMiB<<20)),
		)
	}

	if e.config.Compression == "gzip" {
		dialOpts = append(dialOpts,
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")),
		)
	}

	conn, err := grpc.NewClient(e.config.sanitizedEndpoint(), dialOpts...)
	if err != nil {
		return err
	}

	e.conn = conn
	e.logExporter = plogotlp.NewGRPCClient(conn)
	e.metricExporter = pmetricotlp.NewGRPCClient(conn)

	headers := make(map[string]string, len(e.config.Headers))
	for k, v := range e.config.Headers {
		headers[k] = v
	}
	e.metadata = metadata.New(headers)

	e.callOptions = []grpc.CallOption{
		grpc.WaitForReady(e.config.WaitForReady),
	}

	return nil
}

func (e *baseExporter) shutdown(context.Context) error {
	if e.conn != nil {
		return e.conn.Close()
	}

	return nil
}

func (e *baseExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	if e.logExporter == nil {
		return errors.New("octelium otlp logs exporter not started")
	}

	if len(e.metadata) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, e.metadata)
	}

	resp, respErr := e.logExporter.Export(
		ctx,
		plogotlp.NewExportRequestFromLogs(ld),
		e.callOptions...,
	)
	if err := processError(respErr); err != nil {
		return err
	}

	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedLogRecords() != 0 {
		e.telset.Logger.Warn("Partial success response",
			zap.String("message", partialSuccess.ErrorMessage()),
			zap.Int64("dropped_log_records", partialSuccess.RejectedLogRecords()),
		)
	}

	return nil
}

func (e *baseExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	if e.metricExporter == nil {
		return errors.New("octelium otlp metrics exporter not started")
	}

	if len(e.metadata) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, e.metadata)
	}

	resp, respErr := e.metricExporter.Export(
		ctx,
		pmetricotlp.NewExportRequestFromMetrics(md),
		e.callOptions...,
	)
	if err := processError(respErr); err != nil {
		return err
	}

	partialSuccess := resp.PartialSuccess()
	if partialSuccess.ErrorMessage() != "" || partialSuccess.RejectedDataPoints() != 0 {
		e.telset.Logger.Warn("Partial success response",
			zap.String("message", partialSuccess.ErrorMessage()),
			zap.Int64("dropped_data_points", partialSuccess.RejectedDataPoints()),
		)
	}

	return nil
}

func processError(err error) error {
	if err == nil {
		return nil
	}

	st := status.Convert(err)
	if st.Code() == codes.OK {
		return nil
	}

	retryInfo := getRetryInfo(st)

	if !shouldRetry(st.Code(), retryInfo) {
		return consumererror.NewPermanent(err)
	}

	throttleDuration := retryInfo.GetRetryDelay().AsDuration()
	if throttleDuration != 0 {
		return exporterhelper.NewThrottleRetry(err, throttleDuration)
	}

	return err
}

func getRetryInfo(st *status.Status) *errdetails.RetryInfo {
	if st == nil {
		return nil
	}

	for _, detail := range st.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			return t
		}
	}

	return nil
}

func shouldRetry(code codes.Code, retryInfo *errdetails.RetryInfo) bool {
	switch code {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		return true

	case codes.ResourceExhausted:
		return retryInfo != nil

	default:
		return false
	}
}