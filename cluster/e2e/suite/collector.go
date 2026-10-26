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

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const settleWindow = 20 * time.Second

const driveEvery = 20

const driveBatch = 5

func testCollectorExportOTLP(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	sec := h.CreateEnterpriseSecret(t, sink.Token)
	exp := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, sec.Metadata.Name))

	h.SetCollectorPipelines(t,
		logsPipeline(h.Name(), exp.Metadata.Name),
		metricsPipeline(h.Name(), exp.Metadata.Name))

	restartsBefore := h.EnterpriseRestarts(t, "collector")

	driver := newTrafficDriver(t, h)
	driver.drive(t)

	t.Run("LogsReachTheSink", func(t *testing.T) {
		sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	})

	t.Run("MetricsReachTheSink", func(t *testing.T) {
		sink.WaitMetrics(t, "resourceMetrics", eeharness.IngestionBudget)
	})

	t.Run("InternalPipelineIsUnaffected", func(t *testing.T) {
		waitAccessLogGrows(t, driver, 1)
	})

	t.Run("DisablingStopsTheExport", func(t *testing.T) {
		exp.Spec.IsDisabled = true
		h.UpdateCollectorExporter(t, exp)

		time.Sleep(eeharness.PropagationBudget)
		sink.Truncate(t)

		driver.drive(t)
		sink.MustStayEmpty(t, "logs.json", settleWindow)

		waitAccessLogGrows(t, driver, 1)
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

	newTrafficDriver(t, h).drive(t)

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

	driver := newTrafficDriver(t, h)
	driver.drive(t)

	t.Run("TheHealthyExporterStillDelivers", func(t *testing.T) {
		sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	})

	t.Run("TheInternalPipelineStillDelivers", func(t *testing.T) {
		waitAccessLogGrows(t, driver, 1)
	})

	t.Run("TheCollectorDoesNotCrashLoop", func(t *testing.T) {
		assert.Equal(t, restartsBefore, h.EnterpriseRestarts(t, "collector"))
	})
}

func testCollectorLiveUpdate(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	exp := h.CreateCollectorExporter(t,
		otlpExporter("octelium-e2e-blackhole.e2e.svc:4317", ""))
	h.SetCollectorPipelines(t, logsPipeline(h.Name(), exp.Metadata.Name))

	restartsBefore := h.EnterpriseRestarts(t, "collector")
	driver := newTrafficDriver(t, h)
	driver.drive(t)
	sink.MustStayEmpty(t, "logs.json", settleWindow)

	exp.Spec.GetOtlp().Endpoint = sink.GRPCEndpoint
	h.UpdateCollectorExporter(t, exp)
	driver.drive(t)
	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)

	assert.Equal(t, restartsBefore, h.EnterpriseRestarts(t, "collector"))
}

func testCollectorOutageRecovery(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	exp := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, ""))
	h.SetCollectorPipelines(t, logsPipeline(h.Name(), exp.Metadata.Name))
	driver := newTrafficDriver(t, h)
	driver.drive(t)
	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	sink.Truncate(t)

	restartsBefore := h.EnterpriseRestarts(t, "collector")
	restore := sink.Stop(t)
	driver.drive(t)
	restore()

	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	assert.Equal(t, restartsBefore, h.EnterpriseRestarts(t, "collector"))
}

func testCollectorSignalRouting(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapOTLPSink)

	sink := h.Sink(t)
	sink.Truncate(t)

	exp := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, ""))
	h.SetCollectorPipelines(t, logsPipeline(h.Name(), exp.Metadata.Name))
	driver := newTrafficDriver(t, h)
	driver.drive(t)
	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)

	sink.Truncate(t)
	driver.drive(t)
	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	sink.MustStayEmpty(t, "metrics.json", settleWindow)
}

type trafficDriver struct {
	h *eeharness.H
	c *resty.Client
}

func newTrafficDriver(t *testing.T, h *eeharness.H) *trafficDriver {
	t.Helper()

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})

	return &trafficDriver{
		h: h,
		c: h.HTTPPublicToken("demo-nginx", h.AccessToken(t, usr)).SetRetryCount(0),
	}
}

func (d *trafficDriver) drive(t *testing.T) {
	t.Helper()

	for range driveBatch {
		d.h.WaitAllowed(t, d.c)
	}
}

func (d *trafficDriver) driveContext(ctx context.Context) error {
	for range driveBatch {
		got, err := d.h.StatusOf(ctx, d.c, "/")
		if err != nil {
			return err
		}
		if got != http.StatusOK {
			return errUnexpectedStatus(got, http.StatusOK)
		}
	}

	return nil
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

func componentLogCount(ctx context.Context, h *eeharness.H) (uint32, error) {
	res, err := h.ComponentLogC().ListComponentLog(ctx, &visibilityv1.ListComponentLogRequest{})
	if err != nil {
		return 0, err
	}
	if res.ListResponseMeta == nil {
		return 0, errors.Errorf("the list response has no metadata")
	}

	return res.ListResponseMeta.TotalCount, nil
}

func waitAccessLogGrows(t *testing.T, d *trafficDriver, by uint32) {
	t.Helper()

	h := d.h

	ctx, cancel := h.Ctx(t)
	before, err := accessLogCount(ctx, h)
	componentBefore, componentErr := componentLogCount(ctx, h)
	cancel()
	require.Nil(t, err)
	require.Nil(t, componentErr)

	var attempts int
	var driven int
	last := before

	h.Eventually(t, "the access log count to grow", eeharness.TimeoutBudget,
		func(ctx context.Context) error {
			cur, err := accessLogCount(ctx, h)
			if err != nil {
				return errors.Wrapf(err,
					"the access log count was last %d, want at least %d after %d driven requests",
					last, before+by, driven)
			}

			last = cur

			if cur >= before+by {
				return nil
			}

			if attempts%driveEvery == 0 {
				if err := d.driveContext(ctx); err != nil {
					return errors.Wrap(err, "could not generate access log traffic")
				}
				driven += driveBatch
			}
			attempts++

			componentCur, err := componentLogCount(ctx, h)
			if err != nil {
				return errors.Wrapf(err,
					"the access log count is %d, want at least %d after %d driven requests",
					cur, before+by, driven)
			}

			return errors.Errorf(
				"the access log count is %d, want at least %d after %d driven requests, "+
					"while the component log count is %d, was %d",
				cur, before+by, driven, componentCur, componentBefore)
		})
}

func testCollectorExporterAPI(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	ctx, cancel := h.Ctx(t)
	defer cancel()

	sec := h.CreateEnterpriseSecret(t, "exporter-api-credential")
	exp := h.CreateCollectorExporter(t, otlpExporter("collector.example.com:4317", sec.Metadata.Name))

	t.Run("NameAndUIDResolveTheSameExporter", func(t *testing.T) {
		byName, err := h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Name: exp.Metadata.Name})
		require.Nil(t, err)
		byUID, err := h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Uid: exp.Metadata.Uid})
		require.Nil(t, err)
		assert.True(t, proto.Equal(byName, byUID))
		assert.True(t, proto.Equal(exp.Spec, byUID.Spec))
	})

	t.Run("DuplicateCreationPreservesTheExporter", func(t *testing.T) {
		duplicate := otlpExporter("other.example.com:4317", "")
		duplicate.Metadata = &metav1.Metadata{Name: exp.Metadata.Name}
		_, err := h.EnterpriseC().CreateCollectorExporter(ctx, duplicate)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		got, err := h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Uid: exp.Metadata.Uid})
		require.Nil(t, err)
		assert.True(t, proto.Equal(exp.Spec, got.Spec))
	})

	t.Run("PaginationIncludesEveryExporterOnce", func(t *testing.T) {
		wanted := map[string]bool{exp.Metadata.Uid: true}
		for range 3 {
			item := h.CreateCollectorExporter(t, otlpExporter("collector.example.com:4317", ""))
			wanted[item.Metadata.Uid] = true
		}
		seen := make(map[string]bool)
		for page := uint32(0); ; page++ {
			res, err := h.EnterpriseC().ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{
				Common: &metav1.CommonListOptions{
					Page: page, ItemsPerPage: 2,
					OrderBy: &metav1.CommonListOptions_OrderBy{Type: metav1.CommonListOptions_OrderBy_NAME, Mode: metav1.CommonListOptions_OrderBy_ASC},
				},
			})
			require.Nil(t, err)
			assert.LessOrEqual(t, len(res.Items), 2)
			for _, item := range res.Items {
				assert.False(t, seen[item.Metadata.Uid], "duplicate exporter %s", item.Metadata.Name)
				seen[item.Metadata.Uid] = true
			}
			if len(res.Items) < 2 {
				break
			}
		}
		for uid := range wanted {
			assert.True(t, seen[uid], "missing exporter %s", uid)
		}
	})

	t.Run("RejectedUpdatesPreserveTheStoredSpec", func(t *testing.T) {
		for _, mutate := range []func(*enterprisev1.CollectorExporter){
			func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "invalid" },
			func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:0" },
			func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:-1" },
			func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:65536" },
			func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Compression = 99 },
			func(arg *enterprisev1.CollectorExporter) {
				arg.Spec.GetOtlp().Timeout = &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 307445735}}
			},
			func(arg *enterprisev1.CollectorExporter) {
				arg.Spec.GetOtlp().Headers = []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\r\nInjected: true"}}
			},
			func(arg *enterprisev1.CollectorExporter) {
				arg.Spec.GetOtlp().Auth.GetBearer().Type = &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{FromSecret: h.Name()}
			},
		} {
			arg := proto.Clone(exp).(*enterprisev1.CollectorExporter)
			arg.Metadata.Description = "must not persist"
			mutate(arg)
			_, err := h.EnterpriseC().UpdateCollectorExporter(ctx, arg)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
			got, err := h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Uid: exp.Metadata.Uid})
			require.Nil(t, err)
			assert.True(t, proto.Equal(exp.Spec, got.Spec))
			assert.Equal(t, exp.Metadata.Description, got.Metadata.Description)
		}
	})

	t.Run("AnExporterCanChangeProtocol", func(t *testing.T) {
		arg := proto.Clone(exp).(*enterprisev1.CollectorExporter)
		arg.Spec = otlpHTTPExporter("https://collector.example.com", sec.Metadata.Name).Spec
		arg.Metadata.Description = "HTTP exporter"
		updated, err := h.EnterpriseC().UpdateCollectorExporter(ctx, arg)
		require.Nil(t, err)
		assert.Equal(t, exp.Metadata.Uid, updated.Metadata.Uid)
		assert.Equal(t, "HTTP exporter", updated.Metadata.Description)
		got, err := h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Uid: exp.Metadata.Uid})
		require.Nil(t, err)
		assert.True(t, proto.Equal(arg.Spec, got.Spec))
	})

	t.Run("RecreationDoesNotReuseTheUID", func(t *testing.T) {
		_, err := h.EnterpriseC().DeleteCollectorExporter(ctx, &metav1.DeleteOptions{Uid: exp.Metadata.Uid})
		require.Nil(t, err)
		_, err = h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Name: exp.Metadata.Name})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		replacement := otlpExporter("collector.example.com:4317", "")
		replacement.Metadata = &metav1.Metadata{Name: exp.Metadata.Name}
		recreated := h.CreateCollectorExporter(t, replacement)
		assert.NotEqual(t, exp.Metadata.Uid, recreated.Metadata.Uid)
		_, err = h.EnterpriseC().DeleteCollectorExporter(ctx, &metav1.DeleteOptions{Uid: exp.Metadata.Uid})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		_, err = h.EnterpriseC().GetCollectorExporter(ctx, &metav1.GetOptions{Uid: recreated.Metadata.Uid})
		assert.Nil(t, err)
	})
}
