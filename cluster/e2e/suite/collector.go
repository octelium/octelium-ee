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
	"net/http"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const settleWindow = 20 * time.Second

const driveEvery = 20

func testCollectorExportOTLP(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomStringCanonical(40))
	exp := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, sec.Metadata.Name))

	h.SetCollectorPipelines(t,
		logsPipeline(h.Name(), exp.Metadata.Name),
		metricsPipeline(h.Name(), exp.Metadata.Name))

	restartsBefore := h.EnterpriseRestarts(t, "collector")

	driveTraffic(t, h)

	t.Run("LogsReachTheSink", func(t *testing.T) {
		sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	})

	t.Run("MetricsReachTheSink", func(t *testing.T) {
		sink.WaitMetrics(t, "resourceMetrics", eeharness.IngestionBudget)
	})

	t.Run("InternalPipelineIsUnaffected", func(t *testing.T) {
		waitAccessLogGrows(t, h, 1)
	})

	t.Run("DisablingStopsTheExport", func(t *testing.T) {
		exp.Spec.IsDisabled = true
		h.UpdateCollectorExporter(t, exp)

		time.Sleep(eeharness.PropagationBudget)
		sink.Truncate(t)

		driveTraffic(t, h)
		sink.MustStayEmpty(t, "logs.json", settleWindow)

		waitAccessLogGrows(t, h, 1)
	})

	t.Run("TheCollectorHasNotRestarted", func(t *testing.T) {
		assert.Equal(t, restartsBefore, h.EnterpriseRestarts(t, "collector"))
	})
}

func testCollectorExportOTLPHTTP(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomStringCanonical(40))
	exp := h.CreateCollectorExporter(t,
		otlpHTTPExporter(sink.HTTPEndpoint, sec.Metadata.Name))

	h.SetCollectorPipelines(t, logsPipeline(h.Name(), exp.Metadata.Name))

	driveTraffic(t, h)

	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
}

func testCollectorExportUnreachable(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	good := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, ""))
	bad := h.CreateCollectorExporter(t,
		otlpExporter("octelium-e2e-blackhole.e2e.svc:4317", ""))

	h.SetCollectorPipelines(t,
		logsPipeline(h.Name(), good.Metadata.Name, bad.Metadata.Name))

	restartsBefore := h.EnterpriseRestarts(t, "collector")

	driveTraffic(t, h)

	t.Run("TheHealthyExporterStillDelivers", func(t *testing.T) {
		sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	})

	t.Run("TheInternalPipelineStillDelivers", func(t *testing.T) {
		waitAccessLogGrows(t, h, 1)
	})

	t.Run("TheCollectorDoesNotCrashLoop", func(t *testing.T) {
		assert.Equal(t, restartsBefore, h.EnterpriseRestarts(t, "collector"))
	})
}

func driveTraffic(t *testing.T, h *eeharness.H) {
	t.Helper()

	for range 5 {
		h.GetStatus(t, h.HTTPPublic("demo-nginx"), "/", http.StatusUnauthorized)
	}
}

func accessLogCount(ctx context.Context, h *eeharness.H) (uint32, error) {
	res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
	if err != nil {
		return 0, err
	}
	if res.ListResponseMeta == nil {
		return 0, errors.Errorf("the list response has no metadata")
	}

	return res.ListResponseMeta.TotalCount, nil
}

func waitAccessLogGrows(t *testing.T, h *eeharness.H, by uint32) {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	before, err := accessLogCount(ctx, h)
	cancel()
	require.Nil(t, err)

	var attempts int

	h.Eventually(t, "the access log count to grow", eeharness.IngestionBudget,
		func(ctx context.Context) error {
			cur, err := accessLogCount(ctx, h)
			if err != nil {
				return err
			}
			if cur >= before+by {
				return nil
			}

			if attempts%driveEvery == 0 {
				driveTraffic(t, h)
			}
			attempts++

			return errors.Errorf("the access log count is %d, want at least %d", cur, before+by)
		})
}
