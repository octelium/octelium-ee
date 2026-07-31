// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	mockRawMetricRetention      = 48 * time.Hour
	mockMinimumQueryStep        = time.Second
	mockMaxSeriesPerQuery       = 200
	mockDefaultSeriesPerQuery   = 50
	mockMaxPointsPerSeries      = 5000
	mockDefaultPointsPerSeries  = 1000
	mockMaxTotalPoints          = 50000
	mockDefaultTotalPoints      = 10000
	mockMaxGroupByAttributes    = 8
	mockMaxFilters              = 32
	mockMaxFilterValues         = 128
	mockMaximumSourceSeries     = 20000
	mockMaximumRawHistogramRows = 100000
	mockMaximumRawNumberRows    = 2000000

	mockMaxDataPointsPerExport          = 50000
	mockMaxEffectiveAttributesPerSeries = 64
	mockMaxAttributeKeyBytes            = 256
	mockMaxAttributeValueBytes          = 4096
	mockMaxSeriesLabelsBytes            = 64 << 10
	mockMaxHistogramBuckets             = 512
	mockMaxQueuedExports                = 32
	mockMaxQueuedDataPoints             = 200000
	mockMaxQueuedEstimatedBytes         = 128 << 20
	mockMaxGRPCMessageBytes             = 8 << 20
	mockMaximumFutureSkew               = 10 * time.Minute
	mockAcceptedPastWindow              = mockRawMetricRetention + time.Hour
)

type tstMetricsService struct {
	vmetricsv1.UnimplementedMetricsServiceServer
}

type mockMetricInfo struct {
	kind            vmetricsv1.MetricDescriptor_Kind
	numberValueType vmetricsv1.MetricDescriptor_NumberValueType
	unit            string
	description     string
	temporality     vmetricsv1.MetricDescriptor_Temporality
	attributeKeys   []string
	base            float64
	amp             float64
}

type mockSeries struct {
	id       string
	labels   []*vmetricsv1.Attribute
	quantile *float64
	phase    float64
}

var mockComponentKeys = []string{
	"octelium.component.type",
	"octelium.component.namespace",
	"octelium.component.name",
}

var mockVigilKeys = []string{
	"octelium.component.type",
	"octelium.component.namespace",
	"octelium.component.name",
	"octelium.vigil.svc.name",
	"octelium.vigil.svc.namespace.name",
	"octelium.vigil.svc.region.name",
	"octelium.vigil.svc.mode",
	"http.method",
}

var mockMetricMeta = map[string]mockMetricInfo{
	"req.total": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "requests",
		description:     "Total number of requests",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockVigilKeys,
		base:            220,
		amp:             0.6,
	},
	"req.active": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "requests",
		description:     "Number of active requests",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockVigilKeys,
		base:            36,
		amp:             0.5,
	},
	"req.duration": {
		kind:            vmetricsv1.MetricDescriptor_HISTOGRAM,
		numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		unit:            "ms",
		description:     "Request duration in milliseconds",
		temporality:     vmetricsv1.MetricDescriptor_DELTA,
		attributeKeys:   mockVigilKeys,
		base:            220,
		amp:             0.6,
	},
	"authorization.req.total": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "requests",
		description:     "Total number of authorization requests",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            180,
		amp:             0.6,
	},
	"authorization.req.active": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "requests",
		description:     "Number of active authorization requests",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            22,
		amp:             0.5,
	},
	"authorization.req.duration": {
		kind:            vmetricsv1.MetricDescriptor_HISTOGRAM,
		numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		unit:            "ms",
		description:     "Authorization request duration in milliseconds",
		temporality:     vmetricsv1.MetricDescriptor_DELTA,
		attributeKeys:   mockComponentKeys,
		base:            180,
		amp:             0.55,
	},
	"process.goroutines": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "goroutines",
		description:     "Number of goroutines",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            600,
		amp:             0.3,
	},
	"process.uptime": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "seconds",
		description:     "Process uptime in seconds",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            1,
	},
	"process.gomaxprocs": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		description:     "Current GOMAXPROCS setting",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            8,
	},
	"process.mem.heap_alloc": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Bytes of allocated heap objects",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            180e6,
		amp:             0.3,
	},
	"process.mem.total": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Total bytes of memory mapped by the Go runtime",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            320e6,
		amp:             0.15,
	},
	"process.mem.heap_released": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Heap memory returned to the operating system",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            90e6,
		amp:             0.2,
	},
	"process.mem.stacks": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Memory used by goroutine stacks",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            24e6,
		amp:             0.2,
	},
	"process.mem.heap_objects": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		description:     "Number of live heap objects",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            1.2e6,
		amp:             0.25,
	},
	"process.mem.alloc_bytes": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Cumulative bytes allocated for heap objects",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            5e6,
		amp:             0.5,
	},
	"process.mem.alloc_objects": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		description:     "Cumulative count of heap objects allocated",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            60000,
		amp:             0.5,
	},
	"process.gc.cycles": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		description:     "Completed GC cycles",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            0.06,
		amp:             0.4,
	},
	"process.gc.heap_goal": {
		kind:            vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_INT64,
		unit:            "bytes",
		description:     "Target heap size for the next GC cycle",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            360e6,
		amp:             0.15,
	},
	"process.cpu.gc_seconds": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		unit:            "seconds",
		description:     "Cumulative CPU time spent in garbage collection",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            0.02,
		amp:             0.4,
	},
	"process.cpu.seconds": {
		kind:            vmetricsv1.MetricDescriptor_COUNTER,
		numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		unit:            "seconds",
		description:     "Cumulative CPU time consumed by the process",
		temporality:     vmetricsv1.MetricDescriptor_CUMULATIVE,
		attributeKeys:   mockComponentKeys,
		base:            0.45,
		amp:             0.4,
	},
}

var (
	mockHistBounds = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500}
	mockHistWFast  = []float64{0.10, 0.16, 0.26, 0.24, 0.13, 0.06, 0.03, 0.012, 0.006, 0.002}
	mockHistWSlow  = []float64{0.05, 0.09, 0.18, 0.22, 0.18, 0.12, 0.08, 0.04, 0.025, 0.015}
)

func (s *tstMetricsService) QueryMetrics(
	ctx context.Context,
	req *vmetricsv1.QueryMetricsRequest,
) (*vmetricsv1.QueryMetricsResponse, error) {
	_ = ctx

	if req == nil || req.Metric == nil || req.Metric.Selector == nil {
		return nil, status.Error(codes.InvalidArgument, "metric selector must be set")
	}
	if req.TimeRange == nil || req.TimeRange.From == nil || req.TimeRange.To == nil {
		return nil, status.Error(codes.InvalidArgument, "timeRange.from and timeRange.to must be set")
	}
	if req.Operation == nil || req.Operation.Type == nil {
		return nil, status.Error(codes.InvalidArgument, "operation must be set")
	}

	name, info, err := mockResolveMetric(req.Metric)
	if err != nil {
		return nil, err
	}
	if req.Metric.Kind != vmetricsv1.MetricDescriptor_KIND_UNSET && req.Metric.Kind != info.kind {
		return nil, status.Error(codes.InvalidArgument, "metric kind does not match descriptor")
	}
	if err := mockValidateOperation(info, req.Operation); err != nil {
		return nil, err
	}

	from, to := mockTimeRange(req.TimeRange)
	if !to.After(from) {
		return nil, status.Error(codes.InvalidArgument, "timeRange.to must be after timeRange.from")
	}
	if to.Sub(from) > mockRawMetricRetention {
		return nil, status.Error(codes.InvalidArgument, "query time range exceeds available raw retention")
	}

	isRaw := req.Operation.GetCounter() != nil &&
		req.Operation.GetCounter().Function == vmetricsv1.CounterOperation_RAW
	step := time.Duration(0)
	if isRaw {
		if req.Step != nil {
			return nil, status.Error(codes.InvalidArgument, "step must be unset for counter RAW queries")
		}
	} else {
		step = mockDurationToTime(req.Step)
		if step <= 0 {
			return nil, status.Error(codes.InvalidArgument, "step must be set for bucketed queries")
		}
		if step < mockMinimumQueryStep {
			return nil, status.Error(codes.InvalidArgument, "step is too small")
		}
	}

	aggregation := req.SeriesAggregation
	if aggregation == vmetricsv1.QueryMetricsRequest_SERIES_AGGREGATION_UNSET {
		switch {
		case len(req.GroupBy) == 0:
			aggregation = vmetricsv1.QueryMetricsRequest_NONE
		case info.kind == vmetricsv1.MetricDescriptor_HISTOGRAM:
			aggregation = vmetricsv1.QueryMetricsRequest_MERGE
		default:
			aggregation = vmetricsv1.QueryMetricsRequest_SUM
		}
	}
	if aggregation == vmetricsv1.QueryMetricsRequest_NONE && len(req.GroupBy) > 0 {
		return nil, status.Error(codes.InvalidArgument, "groupBy must be empty for SeriesAggregation.NONE")
	}
	if info.kind == vmetricsv1.MetricDescriptor_HISTOGRAM {
		if aggregation != vmetricsv1.QueryMetricsRequest_NONE &&
			aggregation != vmetricsv1.QueryMetricsRequest_MERGE {
			return nil, status.Error(codes.InvalidArgument, "histogram queries require NONE or MERGE series aggregation")
		}
	} else if aggregation == vmetricsv1.QueryMetricsRequest_MERGE {
		return nil, status.Error(codes.InvalidArgument, "MERGE is only valid for histogram metrics")
	}

	snapshot := time.Now().UTC()

	descriptor := mockDescriptor(name, info)
	allSeries := mockBuildSeries(req, descriptor, info, aggregation)
	sort.Slice(allSeries, func(i, j int) bool {
		return allSeries[i].id < allSeries[j].id
	})

	totalSeries := uint32(len(allSeries))
	limitSeries := int(req.LimitSeries)
	if limitSeries <= 0 {
		limitSeries = mockDefaultSeriesPerQuery
	}
	if limitSeries > mockMaxSeriesPerQuery {
		limitSeries = mockMaxSeriesPerQuery
	}

	seriesTruncated := false
	if len(allSeries) > limitSeries {
		behavior := req.LimitBehavior
		if behavior == vmetricsv1.QueryMetricsRequest_LIMIT_BEHAVIOR_UNSET {
			behavior = vmetricsv1.QueryMetricsRequest_ERROR
		}

		switch behavior {
		case vmetricsv1.QueryMetricsRequest_ERROR:
			return nil, status.Error(
				codes.ResourceExhausted,
				"metric query exceeds the output-series limit",
			)

		case vmetricsv1.QueryMetricsRequest_TRUNCATE:
			allSeries = allSeries[:limitSeries]
			seriesTruncated = true

		default:
			return nil, status.Error(codes.InvalidArgument, "invalid limit behavior")
		}
	}
	page := allSeries

	generationStep := step
	if isRaw {
		generationStep = time.Minute
	}
	pointCount := mockPointCount(from, to, generationStep, false)
	if pointCount > mockMaxPointsPerSeries {
		return nil, status.Error(codes.InvalidArgument, "query produces too many points")
	}

	limitPoints := int(req.LimitPointsPerSeries)
	if limitPoints <= 0 {
		limitPoints = mockDefaultPointsPerSeries
	}
	if limitPoints > mockMaxPointsPerSeries {
		limitPoints = mockMaxPointsPerSeries
	}

	limitTotal := int(req.LimitTotalPoints)
	if limitTotal <= 0 {
		limitTotal = mockDefaultTotalPoints
	}
	if limitTotal > mockMaxTotalPoints {
		limitTotal = mockMaxTotalPoints
	}

	limitBehavior := req.LimitBehavior
	if limitBehavior == vmetricsv1.QueryMetricsRequest_LIMIT_BEHAVIOR_UNSET {
		limitBehavior = vmetricsv1.QueryMetricsRequest_ERROR
	}

	response := &vmetricsv1.QueryMetricsResponse{
		SourceDescriptor: descriptor,
		Operation:        req.Operation,
		Series:           make([]*vmetricsv1.TimeSeries, 0, len(page)),
		Result:           mockResultDescriptor(info, req.Operation),
		Truncation:       &vmetricsv1.TruncationInfo{},
		SnapshotTime:     pbutils.Timestamp(snapshot),
	}
	if !isRaw {
		response.Step = mockDurationPB(step)
	}
	if req.IncludeTotalSeries {
		response.TotalSeries = &totalSeries
	}
	if seriesTruncated {
		response.Truncation.SeriesTruncated = true
		response.Truncation.Reasons = append(
			response.Truncation.Reasons,
			vmetricsv1.TruncationInfo_SERVER_LIMIT,
		)
	}

	for _, item := range page {
		var output *vmetricsv1.TimeSeries

		histOp := req.Operation.GetHistogram()
		switch {
		case histOp != nil && histOp.Function == vmetricsv1.HistogramOperation_BUCKETS:
			output = &vmetricsv1.TimeSeries{
				Id:     item.id,
				Labels: item.labels,
				Points: &vmetricsv1.TimeSeries_Histogram{
					Histogram: s.genHistogramSeries(info, item.phase, from, generationStep, pointCount),
				},
			}

		case histOp != nil && histOp.Function == vmetricsv1.HistogramOperation_QUANTILE:
			output = &vmetricsv1.TimeSeries{
				Id:       item.id,
				Labels:   item.labels,
				Quantile: item.quantile,
				Points: &vmetricsv1.TimeSeries_Number{
					Number: s.genQuantileSeries(info, item.phase, from, generationStep, pointCount, *item.quantile),
				},
			}

		default:
			output = &vmetricsv1.TimeSeries{
				Id:     item.id,
				Labels: item.labels,
				Points: &vmetricsv1.TimeSeries_Number{
					Number: s.genNumberSeries(info, req.Operation, item.phase, from, generationStep, pointCount),
				},
			}
		}

		response.Series = append(response.Series, output)
	}

	if err := mockEnforcePointLimits(response, limitPoints, limitTotal, limitBehavior); err != nil {
		return nil, err
	}

	response.Truncation.ReturnedSeries = uint32(len(response.Series))
	response.Truncation.ReturnedPoints = uint32(mockCountResponsePoints(response.Series))
	response.Truncation.Reasons = mockUniqueTruncationReasons(response.Truncation.Reasons)

	return response, nil
}

func (s *tstMetricsService) ListMetricDescriptors(
	ctx context.Context,
	req *vmetricsv1.ListMetricDescriptorsRequest,
) (*vmetricsv1.ListMetricDescriptorsResponse, error) {
	_ = ctx

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	kindSet := map[vmetricsv1.MetricDescriptor_Kind]struct{}{}
	for _, kind := range req.Kinds {
		if kind == vmetricsv1.MetricDescriptor_KIND_UNSET {
			return nil, status.Error(
				codes.InvalidArgument,
				"descriptor kind filter cannot contain KIND_UNSET",
			)
		}
		if _, ok := kindSet[kind]; ok {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"duplicate descriptor kind filter: %s",
				kind.String(),
			)
		}
		kindSet[kind] = struct{}{}
	}

	items := make([]*vmetricsv1.MetricDescriptor, 0, len(mockMetricMeta))
	namePrefix := strings.TrimSpace(req.NamePrefix)
	for name, info := range mockMetricMeta {
		if namePrefix != "" && !strings.HasPrefix(name, namePrefix) {
			continue
		}
		if len(kindSet) > 0 {
			if _, ok := kindSet[info.kind]; !ok {
				continue
			}
		}
		if !mockMetricMatchesComponent(name, req.Component) {
			continue
		}

		items = append(items, mockDescriptor(name, info))
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Id < items[j].Id
		}
		return items[i].Name < items[j].Name
	})

	return &vmetricsv1.ListMetricDescriptorsResponse{
		Items: items,
	}, nil
}

func (s *tstMetricsService) ListMetricCatalog(
	ctx context.Context,
	req *vmetricsv1.ListMetricCatalogRequest,
) (*vmetricsv1.ListMetricCatalogResponse, error) {
	_ = ctx

	var component *vmetricsv1.ComponentSelector
	if req != nil {
		component = req.Component
	}

	groupByType := []string{"octelium.component.type", "octelium.component.namespace"}

	items := []*vmetricsv1.MetricCatalogItem{
		mockCatalogItem(
			"memory_total",
			"Memory usage",
			"Total memory used by the component",
			"process.mem.total",
			mockGaugeOp(vmetricsv1.GaugeOperation_LAST),
			groupByType,
			"bytes",
			vmetricsv1.QueryMetricsRequest_SUM,
		),
		mockCatalogItem(
			"heap_alloc",
			"Heap allocated",
			"Bytes of live heap objects",
			"process.mem.heap_alloc",
			mockGaugeOp(vmetricsv1.GaugeOperation_LAST),
			groupByType,
			"bytes",
			vmetricsv1.QueryMetricsRequest_SUM,
		),
		mockCatalogItem(
			"goroutines",
			"Goroutines",
			"Number of goroutines",
			"process.goroutines",
			mockGaugeOp(vmetricsv1.GaugeOperation_LAST),
			groupByType,
			"goroutines",
			vmetricsv1.QueryMetricsRequest_SUM,
		),
		mockCatalogItem(
			"cpu_usage",
			"CPU usage",
			"CPU cores consumed",
			"process.cpu.seconds",
			mockCounterOp(vmetricsv1.CounterOperation_RATE),
			groupByType,
			"cores",
			vmetricsv1.QueryMetricsRequest_SUM,
		),
		mockCatalogItem(
			"gc_rate",
			"GC rate",
			"Completed GC cycles per second",
			"process.gc.cycles",
			mockCounterOp(vmetricsv1.CounterOperation_RATE),
			groupByType,
			"cycles/s",
			vmetricsv1.QueryMetricsRequest_SUM,
		),
	}

	if mockComponentMatches(component, "vigil", "octovigil", "rscserver", "portal", "authserver") {
		items = append(items,
			mockCatalogItem(
				"requests_rate",
				"Requests rate",
				"Per-second request rate",
				"req.total",
				mockCounterOp(vmetricsv1.CounterOperation_RATE),
				groupByType,
				"requests/s",
				vmetricsv1.QueryMetricsRequest_SUM,
			),
			mockCatalogItem(
				"active_requests",
				"Active requests",
				"Current active requests",
				"req.active",
				mockGaugeOp(vmetricsv1.GaugeOperation_LAST),
				groupByType,
				"requests",
				vmetricsv1.QueryMetricsRequest_SUM,
			),
			mockCatalogItem(
				"request_p95_latency",
				"Request p95 latency",
				"p95 request duration",
				"req.duration",
				mockHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.95),
				groupByType,
				"ms",
				vmetricsv1.QueryMetricsRequest_MERGE,
			),
		)
	}

	if mockComponentMatches(component, "octovigil") {
		items = append(items,
			mockCatalogItem(
				"authorization_requests_rate",
				"Authorization rate",
				"Per-second authorization request rate",
				"authorization.req.total",
				mockCounterOp(vmetricsv1.CounterOperation_RATE),
				groupByType,
				"requests/s",
				vmetricsv1.QueryMetricsRequest_SUM,
			),
		)
	}

	return &vmetricsv1.ListMetricCatalogResponse{Items: items}, nil
}

func (s *tstMetricsService) GetMetricsCapabilities(
	ctx context.Context,
	req *vmetricsv1.GetMetricsCapabilitiesRequest,
) (*vmetricsv1.GetMetricsCapabilitiesResponse, error) {
	_ = ctx
	_ = req

	return &vmetricsv1.GetMetricsCapabilitiesResponse{
		QueryLimits: &vmetricsv1.QueryLimits{
			MaximumTimeRange:         mockDurationPB(mockRawMetricRetention),
			MinimumStep:              mockDurationPB(mockMinimumQueryStep),
			MaximumSeries:            mockMaxSeriesPerQuery,
			MaximumPointsPerSeries:   mockMaxPointsPerSeries,
			MaximumTotalPoints:       mockMaxTotalPoints,
			MaximumGroupByAttributes: mockMaxGroupByAttributes,
			MaximumFilters:           mockMaxFilters,
			MaximumFilterValues:      mockMaxFilterValues,
			MaximumSourceSeries:      mockMaximumSourceSeries,
			MaximumRawHistogramRows:  mockMaximumRawHistogramRows,
			MaximumRawNumberRows:     mockMaximumRawNumberRows,
		},
		RetentionTiers: []*vmetricsv1.MetricRetentionTier{
			{
				Name:      "raw",
				Retention: mockDurationPB(mockRawMetricRetention),
				Raw:       true,
			},
		},
		ServerTime: pbutils.Now(),
		MetricKinds: []*vmetricsv1.MetricKindCapability{
			{
				Kind: vmetricsv1.MetricDescriptor_COUNTER,
				CounterFunctions: []vmetricsv1.CounterOperation_Function{
					vmetricsv1.CounterOperation_RAW,
					vmetricsv1.CounterOperation_RATE,
					vmetricsv1.CounterOperation_INCREASE,
				},
				SeriesAggregations: mockNumberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
				GaugeFunctions:     mockGaugeFunctions(),
				SeriesAggregations: mockNumberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_GAUGE,
				GaugeFunctions:     mockGaugeFunctions(),
				SeriesAggregations: mockNumberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_HISTOGRAM,
				HistogramFunctions: mockHistogramFunctions(),
				SeriesAggregations: []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
					vmetricsv1.QueryMetricsRequest_NONE,
					vmetricsv1.QueryMetricsRequest_MERGE,
				},
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM,
				HistogramFunctions: mockHistogramFunctions(),
				SeriesAggregations: []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
					vmetricsv1.QueryMetricsRequest_NONE,
					vmetricsv1.QueryMetricsRequest_MERGE,
				},
			},
		},
		IngestionLimits: &vmetricsv1.MetricIngestionLimits{
			MaximumDataPointsPerExport:          mockMaxDataPointsPerExport,
			MaximumEffectiveAttributesPerSeries: mockMaxEffectiveAttributesPerSeries,
			MaximumAttributeKeyBytes:            mockMaxAttributeKeyBytes,
			MaximumAttributeValueBytes:          mockMaxAttributeValueBytes,
			MaximumSeriesLabelsBytes:            mockMaxSeriesLabelsBytes,
			MaximumHistogramBuckets:             mockMaxHistogramBuckets,
			MaximumQueuedExports:                mockMaxQueuedExports,
			MaximumQueuedDataPoints:             mockMaxQueuedDataPoints,
			MaximumQueuedEstimatedBytes:         mockMaxQueuedEstimatedBytes,
			MaximumGRPCMessageBytes:             mockMaxGRPCMessageBytes,
			MaximumFutureSkew:                   mockDurationPB(mockMaximumFutureSkew),
			AcceptedPastWindow:                  mockDurationPB(mockAcceptedPastWindow),
		},
	}, nil
}

func (s *tstMetricsService) genNumberSeries(
	info mockMetricInfo,
	op *vmetricsv1.QueryOperation,
	phase float64,
	from time.Time,
	step time.Duration,
	n int,
) *vmetricsv1.NumberPointSeries {
	stepSec := step.Seconds()
	if stepSec <= 0 {
		stepSec = 1
	}

	points := &vmetricsv1.NumberPointSeries{}
	rawInt := op.GetCounter() != nil &&
		op.GetCounter().Function == vmetricsv1.CounterOperation_RAW &&
		info.numberValueType == vmetricsv1.MetricDescriptor_INT64

	cumulative := info.base * 1000

	for i := 0; i < n; i++ {
		timestamp := from.Add(time.Duration(i+1) * step)
		level := mockLevel(info.base, info.amp, phase, timestamp)
		var value float64

		switch {
		case op.GetCounter() != nil:
			switch op.GetCounter().Function {
			case vmetricsv1.CounterOperation_RATE:
				value = level
			case vmetricsv1.CounterOperation_INCREASE:
				value = level * stepSec
			case vmetricsv1.CounterOperation_RAW:
				cumulative += level * stepSec
				value = cumulative
			default:
				value = level
			}

		case op.GetGauge() != nil:
			switch op.GetGauge().Function {
			case vmetricsv1.GaugeOperation_MIN:
				value = level * 0.82
			case vmetricsv1.GaugeOperation_MAX:
				value = level * 1.22
			case vmetricsv1.GaugeOperation_SUM:
				value = level * 4
			default:
				value = level
			}

		case op.GetHistogram() != nil:
			bounds, counts, sum := mockHistogram(info, phase, timestamp, stepSec)
			switch op.GetHistogram().Function {
			case vmetricsv1.HistogramOperation_COUNT:
				value = float64(mockSumCounts(counts))
			case vmetricsv1.HistogramOperation_SUM:
				value = sum
			case vmetricsv1.HistogramOperation_AVG:
				if count := mockSumCounts(counts); count > 0 {
					value = sum / float64(count)
				}
			case vmetricsv1.HistogramOperation_MIN:
				value = bounds[0] * 0.5
			case vmetricsv1.HistogramOperation_MAX:
				value = bounds[len(bounds)-1] * 1.3
			default:
				value = mockQuantile(bounds, counts, 0.5)
			}

		default:
			value = level
		}

		points.Points = append(points.Points, mockNumberPoint(timestamp, value, rawInt))
	}

	return points
}

func (s *tstMetricsService) genQuantileSeries(
	info mockMetricInfo,
	phase float64,
	from time.Time,
	step time.Duration,
	n int,
	quantile float64,
) *vmetricsv1.NumberPointSeries {
	stepSec := step.Seconds()
	points := &vmetricsv1.NumberPointSeries{}

	for i := 0; i < n; i++ {
		timestamp := from.Add(time.Duration(i+1) * step)
		bounds, counts, _ := mockHistogram(info, phase, timestamp, stepSec)
		points.Points = append(points.Points,
			mockNumberPoint(timestamp, mockQuantile(bounds, counts, quantile), false))
	}

	return points
}

func (s *tstMetricsService) genHistogramSeries(
	info mockMetricInfo,
	phase float64,
	from time.Time,
	step time.Duration,
	n int,
) *vmetricsv1.HistogramPointSeries {
	stepSec := step.Seconds()
	points := &vmetricsv1.HistogramPointSeries{}

	for i := 0; i < n; i++ {
		timestamp := from.Add(time.Duration(i+1) * step)
		bounds, counts, sum := mockHistogram(info, phase, timestamp, stepSec)
		minimum := bounds[0] * 0.5
		maximum := bounds[len(bounds)-1] * 1.3
		sumValue := sum

		points.Points = append(points.Points, &vmetricsv1.HistogramPoint{
			Timestamp: pbutils.Timestamp(timestamp),
			Sum:       &sumValue,
			Count:     mockSumCounts(counts),
			Min:       &minimum,
			Max:       &maximum,
			Buckets:   mockCumulativeBuckets(bounds, counts),
		})
	}

	return points
}

func mockResolveMetric(selector *vmetricsv1.MetricSelector) (string, mockMetricInfo, error) {
	name := strings.TrimSpace(selector.GetName())
	if name != "" {
		info, ok := mockMetricMeta[name]
		if !ok {
			return "", mockMetricInfo{}, status.Errorf(codes.NotFound, "metric not found: %s", name)
		}
		return name, info, nil
	}

	descriptorID := strings.TrimSpace(selector.GetDescriptorID())
	for metricName, info := range mockMetricMeta {
		if mockDescriptorID(metricName, info) == descriptorID {
			return metricName, info, nil
		}
	}

	return "", mockMetricInfo{}, status.Error(codes.NotFound, "metric descriptor not found")
}

func mockValidateOperation(info mockMetricInfo, operation *vmetricsv1.QueryOperation) error {
	switch info.kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		if operation.GetCounter() == nil {
			return status.Error(codes.InvalidArgument, "counter metric requires counter operation")
		}
	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_GAUGE:
		if operation.GetGauge() == nil {
			return status.Error(codes.InvalidArgument, "gauge metric requires gauge operation")
		}
	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		if operation.GetHistogram() == nil {
			return status.Error(codes.InvalidArgument, "histogram metric requires histogram operation")
		}
	default:
		return status.Error(codes.InvalidArgument, "unsupported metric kind")
	}

	return nil
}

func mockBuildSeries(
	req *vmetricsv1.QueryMetricsRequest,
	descriptor *vmetricsv1.MetricDescriptor,
	info mockMetricInfo,
	aggregation vmetricsv1.QueryMetricsRequest_SeriesAggregation,
) []mockSeries {
	labelSets := mockOutputLabelSets(req, info, aggregation)
	histogramOperation := req.Operation.GetHistogram()

	var ret []mockSeries
	for _, labels := range labelSets {
		if histogramOperation != nil &&
			histogramOperation.Function == vmetricsv1.HistogramOperation_QUANTILE {
			quantiles := append([]float64(nil), histogramOperation.Quantiles...)
			if len(quantiles) == 0 {
				quantiles = []float64{0.5, 0.9, 0.99}
			}
			sort.Float64s(quantiles)

			for _, quantile := range quantiles {
				q := quantile
				id := mockOutputSeriesID(descriptor.Id, labels, &q)
				ret = append(ret, mockSeries{
					id:       id,
					labels:   labels,
					quantile: &q,
					phase:    mockPhase(id),
				})
			}
			continue
		}

		id := mockOutputSeriesID(descriptor.Id, labels, nil)
		ret = append(ret, mockSeries{
			id:     id,
			labels: labels,
			phase:  mockPhase(id),
		})
	}

	return ret
}

func mockOutputLabelSets(
	req *vmetricsv1.QueryMetricsRequest,
	info mockMetricInfo,
	aggregation vmetricsv1.QueryMetricsRequest_SeriesAggregation,
) [][]*vmetricsv1.Attribute {
	if aggregation == vmetricsv1.QueryMetricsRequest_NONE {
		return mockSourceLabelSets(req, info)
	}
	if len(req.GroupBy) == 0 {
		return [][]*vmetricsv1.Attribute{nil}
	}

	keys := append([]string(nil), req.GroupBy...)
	return mockCartesianLabelSets(req, info, keys, 64)
}

func mockSourceLabelSets(
	req *vmetricsv1.QueryMetricsRequest,
	info mockMetricInfo,
) [][]*vmetricsv1.Attribute {
	types := mockValuesForKey(req, info, "octelium.component.type")
	if len(types) == 0 {
		return nil
	}

	ret := make([][]*vmetricsv1.Attribute, 0, len(types))
	for idx, componentType := range types {
		name := fmt.Sprintf("%s-%04x", componentType, idx+1)
		if req.Component != nil && req.Component.Name != "" {
			name = req.Component.Name
		}

		labels := []*vmetricsv1.Attribute{
			mockStringAttribute("octelium.component.type", componentType),
			mockStringAttribute("octelium.component.namespace", mockComponentNamespace(req)),
			mockStringAttribute("octelium.component.name", name),
		}
		mockSortAttrs(labels)
		ret = append(ret, labels)
	}

	return ret
}

func mockCartesianLabelSets(
	req *vmetricsv1.QueryMetricsRequest,
	info mockMetricInfo,
	keys []string,
	limit int,
) [][]*vmetricsv1.Attribute {
	ret := [][]*vmetricsv1.Attribute{{}}

	for _, key := range keys {
		values := mockValuesForKey(req, info, key)
		if len(values) == 0 {
			return nil
		}

		next := make([][]*vmetricsv1.Attribute, 0)
		for _, current := range ret {
			for _, value := range values {
				labels := append([]*vmetricsv1.Attribute{}, current...)
				labels = append(labels, mockStringAttribute(key, value))
				next = append(next, labels)
				if len(next) >= limit {
					break
				}
			}
			if len(next) >= limit {
				break
			}
		}
		ret = next
	}

	for _, labels := range ret {
		mockSortAttrs(labels)
	}

	return ret
}

func mockValuesForKey(
	req *vmetricsv1.QueryMetricsRequest,
	info mockMetricInfo,
	key string,
) []string {
	var values []string

	switch key {
	case "octelium.component.type":
		if req.Component != nil && req.Component.Type != "" {
			values = []string{req.Component.Type}
		} else {
			values = mockMetricComponentTypes(info)
		}

	case "octelium.component.namespace":
		values = []string{mockComponentNamespace(req)}

	case "octelium.component.name":
		if req.Component != nil && req.Component.Name != "" {
			values = []string{req.Component.Name}
		} else {
			values = []string{"vigil-7d9f", "vigil-4a2b", "octovigil-66c1"}
		}

	case "octelium.vigil.svc.name":
		values = []string{"api", "web", "db-proxy"}
	case "octelium.vigil.svc.namespace.name":
		values = []string{"default", "production"}
	case "octelium.vigil.svc.region.name":
		values = []string{"default", "edge"}
	case "octelium.vigil.svc.mode":
		values = []string{"HTTP", "TCP"}
	case "http.method":
		values = []string{"GET", "POST", "PUT", "DELETE"}
	default:
		values = []string{"series-a", "series-b", "series-c"}
	}

	return mockApplyStringFilter(values, key, req.Filters)
}

func mockApplyStringFilter(
	values []string,
	key string,
	filters []*vmetricsv1.AttributeFilter,
) []string {
	for _, filter := range filters {
		if filter == nil || filter.Key != key {
			continue
		}

		switch filter.Operator {
		case vmetricsv1.AttributeFilter_EQ:
			target, ok := mockAttributeStringValue(filter.Value)
			if !ok {
				return nil
			}
			values = mockKeepStrings(values, func(value string) bool {
				return value == target
			})

		case vmetricsv1.AttributeFilter_NOT_EQ:
			target, ok := mockAttributeStringValue(filter.Value)
			if !ok {
				return values
			}
			values = mockKeepStrings(values, func(value string) bool {
				return value != target
			})

		case vmetricsv1.AttributeFilter_IN:
			allowed := map[string]struct{}{}
			for _, value := range filter.Values {
				if stringValue, ok := mockAttributeStringValue(value); ok {
					allowed[stringValue] = struct{}{}
				}
			}
			values = mockKeepStrings(values, func(value string) bool {
				_, ok := allowed[value]
				return ok
			})

		case vmetricsv1.AttributeFilter_NOT_EXISTS:
			return nil
		}
	}

	return values
}

func mockKeepStrings(values []string, fn func(string) bool) []string {
	ret := make([]string, 0, len(values))
	for _, value := range values {
		if fn(value) {
			ret = append(ret, value)
		}
	}
	return ret
}

func mockLabelsMatchFilters(
	labels []*vmetricsv1.Attribute,
	filters []*vmetricsv1.AttributeFilter,
) bool {
	byKey := map[string]*vmetricsv1.AttributeValue{}
	for _, label := range labels {
		byKey[label.Key] = label.Value
	}

	for _, filter := range filters {
		if filter == nil {
			return false
		}

		value, exists := byKey[filter.Key]
		switch filter.Operator {
		case vmetricsv1.AttributeFilter_EQ:
			if !exists || mockAttributeValueKey(value) != mockAttributeValueKey(filter.Value) {
				return false
			}
		case vmetricsv1.AttributeFilter_NOT_EQ:
			if !exists || mockAttributeValueKey(value) == mockAttributeValueKey(filter.Value) {
				return false
			}
		case vmetricsv1.AttributeFilter_EXISTS:
			if !exists {
				return false
			}
		case vmetricsv1.AttributeFilter_NOT_EXISTS:
			if exists {
				return false
			}
		case vmetricsv1.AttributeFilter_IN:
			if !exists {
				return false
			}
			found := false
			for _, candidate := range filter.Values {
				if mockAttributeValueKey(value) == mockAttributeValueKey(candidate) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

func mockMetricComponentTypes(info mockMetricInfo) []string {
	switch {
	case strings.Contains(strings.ToLower(info.description), "authorization"):
		return []string{"octovigil"}
	case len(info.attributeKeys) == len(mockVigilKeys):
		return []string{"vigil", "octovigil", "rscserver", "portal", "authserver"}
	default:
		return []string{"vigil", "octovigil", "rscserver", "portal", "authserver", "gateway"}
	}
}

func mockComponentNamespace(req *vmetricsv1.QueryMetricsRequest) string {
	if req.Component != nil && req.Component.Namespace != "" {
		return req.Component.Namespace
	}
	return "octelium"
}

func mockMetricMatchesComponent(name string, component *vmetricsv1.ComponentSelector) bool {
	if component == nil || component.Type == "" {
		return true
	}

	info, ok := mockMetricMeta[name]
	if !ok {
		return false
	}
	for _, componentType := range mockMetricComponentTypes(info) {
		if component.Type == componentType {
			return true
		}
	}
	return false
}

func mockDescriptor(name string, info mockMetricInfo) *vmetricsv1.MetricDescriptor {
	ret := &vmetricsv1.MetricDescriptor{
		Id:              mockDescriptorID(name, info),
		Name:            name,
		Kind:            info.kind,
		NumberValueType: info.numberValueType,
		Unit:            info.unit,
		Description:     info.description,
		Temporality:     info.temporality,
		InstrumentationScope: &vmetricsv1.InstrumentationScope{
			Name:    "octelium.mockapiserver",
			Version: "1",
		},
	}

	for _, key := range info.attributeKeys {
		ret.Attributes = append(ret.Attributes, mockAttributeDescriptor(key))
	}

	if info.kind == vmetricsv1.MetricDescriptor_HISTOGRAM {
		ret.Histogram = &vmetricsv1.MetricDescriptor_ExplicitHistogram{
			ExplicitHistogram: &vmetricsv1.ExplicitHistogramDescriptor{
				Bounds:         append([]float64(nil), mockHistBounds...),
				MergeSupported: true,
			},
		}
	}

	return ret
}

func mockAttributeDescriptor(key string) *vmetricsv1.MetricAttributeDescriptor {
	estimate := uint64(len(mockGroupValues(key)))
	source := vmetricsv1.MetricAttributeDescriptor_DATA_POINT
	if strings.HasPrefix(key, "octelium.component.") {
		source = vmetricsv1.MetricAttributeDescriptor_RESOURCE
	}

	return &vmetricsv1.MetricAttributeDescriptor{
		Key:                     key,
		ValueKind:               vmetricsv1.AttributeValue_STRING,
		Sources:                 []vmetricsv1.MetricAttributeDescriptor_Source{source},
		Filterable:              true,
		Groupable:               true,
		EstimatedDistinctValues: &estimate,
		EstimateAsOf:            pbutils.Now(),
		EstimateLookback:        mockDurationPB(mockRawMetricRetention),
	}
}

func mockDescriptorID(name string, info mockMetricInfo) string {
	var input bytes.Buffer
	mockWriteLengthPrefixedString(&input, name)
	_ = binary.Write(&input, binary.BigEndian, uint32(info.kind))
	_ = binary.Write(&input, binary.BigEndian, uint32(info.numberValueType))
	mockWriteLengthPrefixedString(&input, info.unit)
	_ = binary.Write(&input, binary.BigEndian, uint32(info.temporality))
	mockWriteLengthPrefixedString(&input, "octelium.mockapiserver")
	mockWriteLengthPrefixedString(&input, "1")
	mockWriteLengthPrefixedString(&input, "")

	bounds := []float64(nil)
	if info.kind == vmetricsv1.MetricDescriptor_HISTOGRAM {
		bounds = mockHistBounds
	}
	_ = binary.Write(&input, binary.BigEndian, uint32(len(bounds)))
	for _, bound := range bounds {
		_ = binary.Write(&input, binary.BigEndian, math.Float64bits(bound))
	}

	sum := sha256.Sum256(input.Bytes())
	return "v1:sha256:" + hex.EncodeToString(sum[:])
}

func mockWriteLengthPrefixedString(buffer *bytes.Buffer, value string) {
	_ = binary.Write(buffer, binary.BigEndian, uint64(len([]byte(value))))
	_, _ = buffer.WriteString(value)
}

func mockOutputSeriesID(
	descriptorID string,
	labels []*vmetricsv1.Attribute,
	quantile *float64,
) string {
	var builder strings.Builder
	builder.WriteString(descriptorID)
	builder.WriteByte(0)
	for _, label := range labels {
		builder.WriteString(label.Key)
		builder.WriteByte('=')
		builder.WriteString(mockAttributeValueKey(label.Value))
		builder.WriteByte(0)
	}
	if quantile != nil {
		builder.WriteString("quantile=")
		builder.WriteString(strconv.FormatUint(math.Float64bits(*quantile), 16))
	}
	return mockContentID(builder.String())
}

func mockContentID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "v1:sha256:" + hex.EncodeToString(sum[:])
}

func mockStringAttribute(key, value string) *vmetricsv1.Attribute {
	return &vmetricsv1.Attribute{
		Key: key,
		Value: &vmetricsv1.AttributeValue{
			Value: &vmetricsv1.AttributeValue_StringValue{StringValue: value},
		},
	}
}

func mockAttributeStringValue(value *vmetricsv1.AttributeValue) (string, bool) {
	if value == nil {
		return "", false
	}
	switch typed := value.Value.(type) {
	case *vmetricsv1.AttributeValue_StringValue:
		return typed.StringValue, true
	default:
		return "", false
	}
}

func mockAttributeValueKey(value *vmetricsv1.AttributeValue) string {
	if value == nil {
		return ""
	}

	switch typed := value.Value.(type) {
	case *vmetricsv1.AttributeValue_StringValue:
		return "STRING:" + typed.StringValue
	case *vmetricsv1.AttributeValue_BoolValue:
		return "BOOL:" + strconv.FormatBool(typed.BoolValue)
	case *vmetricsv1.AttributeValue_IntValue:
		return "INT64:" + strconv.FormatInt(typed.IntValue, 10)
	case *vmetricsv1.AttributeValue_DoubleValue:
		return "DOUBLE:" + strconv.FormatUint(math.Float64bits(typed.DoubleValue), 16)
	default:
		return ""
	}
}

func mockResultDescriptor(
	info mockMetricInfo,
	operation *vmetricsv1.QueryOperation,
) *vmetricsv1.QueryResultDescriptor {
	ret := &vmetricsv1.QueryResultDescriptor{
		PointKind:       vmetricsv1.QueryResultDescriptor_NUMBER,
		NumberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		Unit:            info.unit,
	}

	if counter := operation.GetCounter(); counter != nil {
		if counter.Function == vmetricsv1.CounterOperation_RAW {
			ret.NumberValueType = info.numberValueType
		}
		if counter.Function == vmetricsv1.CounterOperation_RATE {
			switch info.unit {
			case "seconds":
				ret.Unit = "cores"
			case "":
				ret.Unit = "1/s"
			default:
				ret.Unit = info.unit + "/s"
			}
		}
		return ret
	}

	if histogram := operation.GetHistogram(); histogram != nil {
		if histogram.Function == vmetricsv1.HistogramOperation_BUCKETS {
			ret.PointKind = vmetricsv1.QueryResultDescriptor_HISTOGRAM
			ret.NumberValueType = vmetricsv1.MetricDescriptor_NUMBER_VALUE_TYPE_UNSET
		}
		if histogram.Function == vmetricsv1.HistogramOperation_COUNT {
			ret.Unit = "observations"
		}
	}

	return ret
}

func mockPointCount(from, to time.Time, step time.Duration, raw bool) int {
	if raw {
		step = time.Minute
	}
	if step <= 0 {
		step = time.Minute
	}

	count := int((to.Sub(from) + step - 1) / step)
	if count < 1 {
		return 1
	}
	return count
}

func mockEnforcePointLimits(
	response *vmetricsv1.QueryMetricsResponse,
	perSeriesLimit int,
	totalLimit int,
	behavior vmetricsv1.QueryMetricsRequest_LimitBehavior,
) error {
	pointsWouldTruncate := false
	for _, series := range response.Series {
		if mockSeriesPointCount(series) > perSeriesLimit {
			pointsWouldTruncate = true
			break
		}
	}

	if pointsWouldTruncate && behavior != vmetricsv1.QueryMetricsRequest_TRUNCATE {
		return status.Error(codes.ResourceExhausted, "metric response exceeds the per-series point limit")
	}

	if pointsWouldTruncate {
		for _, series := range response.Series {
			mockTrimSeriesToNewest(series, perSeriesLimit)
		}
		response.Truncation.PointsTruncated = true
		response.Truncation.Reasons = append(response.Truncation.Reasons,
			vmetricsv1.TruncationInfo_POINTS_PER_SERIES_LIMIT)
	}

	total := mockCountResponsePoints(response.Series)
	if total <= totalLimit {
		return nil
	}
	if behavior != vmetricsv1.QueryMetricsRequest_TRUNCATE {
		return status.Error(codes.ResourceExhausted, "metric response exceeds the total point limit")
	}

	remaining := totalLimit
	for _, series := range response.Series {
		count := mockSeriesPointCount(series)
		if remaining <= 0 {
			mockTrimSeriesToNewest(series, 0)
			continue
		}
		if count > remaining {
			mockTrimSeriesToNewest(series, remaining)
			remaining = 0
			continue
		}
		remaining -= count
	}

	response.Truncation.PointsTruncated = true
	response.Truncation.Reasons = append(response.Truncation.Reasons,
		vmetricsv1.TruncationInfo_TOTAL_POINTS_LIMIT)

	return nil
}

func mockSeriesPointCount(series *vmetricsv1.TimeSeries) int {
	switch {
	case series.GetNumber() != nil:
		return len(series.GetNumber().Points)
	case series.GetHistogram() != nil:
		return len(series.GetHistogram().Points)
	case series.GetExponentialHistogram() != nil:
		return len(series.GetExponentialHistogram().Points)
	default:
		return 0
	}
}

func mockCountResponsePoints(series []*vmetricsv1.TimeSeries) int {
	total := 0
	for _, item := range series {
		total += mockSeriesPointCount(item)
	}
	return total
}

func mockTrimSeriesToNewest(series *vmetricsv1.TimeSeries, limit int) {
	if limit < 0 {
		limit = 0
	}

	switch {
	case series.GetNumber() != nil:
		points := series.GetNumber().Points
		if len(points) > limit {
			series.GetNumber().Points = points[len(points)-limit:]
			if limit == 0 {
				series.GetNumber().Points = nil
			}
		}

	case series.GetHistogram() != nil:
		points := series.GetHistogram().Points
		if len(points) > limit {
			series.GetHistogram().Points = points[len(points)-limit:]
			if limit == 0 {
				series.GetHistogram().Points = nil
			}
		}

	case series.GetExponentialHistogram() != nil:
		points := series.GetExponentialHistogram().Points
		if len(points) > limit {
			series.GetExponentialHistogram().Points = points[len(points)-limit:]
			if limit == 0 {
				series.GetExponentialHistogram().Points = nil
			}
		}
	}
}

func mockUniqueTruncationReasons(
	reasons []vmetricsv1.TruncationInfo_Reason,
) []vmetricsv1.TruncationInfo_Reason {
	seen := map[vmetricsv1.TruncationInfo_Reason]struct{}{}
	ret := make([]vmetricsv1.TruncationInfo_Reason, 0, len(reasons))
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		ret = append(ret, reason)
	}
	return ret
}

func mockCatalogItem(
	id string,
	displayName string,
	description string,
	metricName string,
	operation *vmetricsv1.QueryOperation,
	groupBy []string,
	unit string,
	aggregation vmetricsv1.QueryMetricsRequest_SeriesAggregation,
) *vmetricsv1.MetricCatalogItem {
	info := mockMetricMeta[metricName]
	return &vmetricsv1.MetricCatalogItem{
		Id:          id,
		DisplayName: displayName,
		Description: description,
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: metricName},
			Kind:     info.kind,
		},
		DefaultOperation:         operation,
		DefaultGroupBy:           append([]string(nil), groupBy...),
		Unit:                     unit,
		DefaultSeriesAggregation: aggregation,
		DefaultStep:              mockDurationPB(time.Minute),
	}
}

func mockComponentMatches(component *vmetricsv1.ComponentSelector, types ...string) bool {
	if component == nil || component.Type == "" {
		return true
	}
	for _, componentType := range types {
		if component.Type == componentType {
			return true
		}
	}
	return false
}

func mockGaugeOp(fn vmetricsv1.GaugeOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{
		Type: &vmetricsv1.QueryOperation_Gauge{
			Gauge: &vmetricsv1.GaugeOperation{Function: fn},
		},
	}
}

func mockCounterOp(fn vmetricsv1.CounterOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{
		Type: &vmetricsv1.QueryOperation_Counter{
			Counter: &vmetricsv1.CounterOperation{Function: fn},
		},
	}
}

func mockHistogramOp(
	fn vmetricsv1.HistogramOperation_Function,
	quantiles ...float64,
) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{
		Type: &vmetricsv1.QueryOperation_Histogram{
			Histogram: &vmetricsv1.HistogramOperation{
				Function:  fn,
				Quantiles: append([]float64(nil), quantiles...),
			},
		},
	}
}

func mockNumberSeriesAggregations() []vmetricsv1.QueryMetricsRequest_SeriesAggregation {
	return []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
		vmetricsv1.QueryMetricsRequest_NONE,
		vmetricsv1.QueryMetricsRequest_SUM,
		vmetricsv1.QueryMetricsRequest_AVG,
		vmetricsv1.QueryMetricsRequest_MIN,
		vmetricsv1.QueryMetricsRequest_MAX,
		vmetricsv1.QueryMetricsRequest_LAST,
	}
}

func mockGaugeFunctions() []vmetricsv1.GaugeOperation_Function {
	return []vmetricsv1.GaugeOperation_Function{
		vmetricsv1.GaugeOperation_LAST,
		vmetricsv1.GaugeOperation_AVG,
		vmetricsv1.GaugeOperation_MIN,
		vmetricsv1.GaugeOperation_MAX,
		vmetricsv1.GaugeOperation_SUM,
	}
}

func mockHistogramFunctions() []vmetricsv1.HistogramOperation_Function {
	return []vmetricsv1.HistogramOperation_Function{
		vmetricsv1.HistogramOperation_BUCKETS,
		vmetricsv1.HistogramOperation_QUANTILE,
		vmetricsv1.HistogramOperation_AVG,
		vmetricsv1.HistogramOperation_COUNT,
		vmetricsv1.HistogramOperation_SUM,
		vmetricsv1.HistogramOperation_MIN,
		vmetricsv1.HistogramOperation_MAX,
	}
}

func mockNumberPoint(timestamp time.Time, value float64, asInt bool) *vmetricsv1.NumberPoint {
	if asInt {
		return &vmetricsv1.NumberPoint{
			Timestamp: pbutils.Timestamp(timestamp),
			Value: &vmetricsv1.NumberPoint_AsInt{
				AsInt: int64(value),
			},
		}
	}
	return &vmetricsv1.NumberPoint{
		Timestamp: pbutils.Timestamp(timestamp),
		Value: &vmetricsv1.NumberPoint_AsDouble{
			AsDouble: value,
		},
	}
}

func mockTimeRange(timeRange *vmetricsv1.TimeRange) (time.Time, time.Time) {
	now := time.Now().UTC()
	if timeRange == nil || timeRange.From == nil || timeRange.To == nil {
		return now.Add(-6 * time.Hour), now
	}
	return timeRange.From.AsTime().UTC(), timeRange.To.AsTime().UTC()
}

func mockDurationToTime(duration *metav1.Duration) time.Duration {
	if duration == nil {
		return 0
	}

	switch {
	case duration.GetMilliseconds() > 0:
		return time.Duration(duration.GetMilliseconds()) * time.Millisecond
	case duration.GetSeconds() > 0:
		return time.Duration(duration.GetSeconds()) * time.Second
	case duration.GetMinutes() > 0:
		return time.Duration(duration.GetMinutes()) * time.Minute
	case duration.GetHours() > 0:
		return time.Duration(duration.GetHours()) * time.Hour
	case duration.GetDays() > 0:
		return time.Duration(duration.GetDays()) * 24 * time.Hour
	case duration.GetWeeks() > 0:
		return time.Duration(duration.GetWeeks()) * 7 * 24 * time.Hour
	default:
		return 0
	}
}

func mockDurationPB(duration time.Duration) *metav1.Duration {
	if duration <= 0 {
		return nil
	}
	if duration%time.Hour == 0 {
		return &metav1.Duration{
			Type: &metav1.Duration_Hours{Hours: uint32(duration / time.Hour)},
		}
	}
	if duration%time.Minute == 0 {
		return &metav1.Duration{
			Type: &metav1.Duration_Minutes{Minutes: uint32(duration / time.Minute)},
		}
	}
	if duration%time.Second == 0 {
		return &metav1.Duration{
			Type: &metav1.Duration_Seconds{Seconds: uint32(duration / time.Second)},
		}
	}
	return &metav1.Duration{
		Type: &metav1.Duration_Milliseconds{Milliseconds: uint32(duration / time.Millisecond)},
	}
}

func mockSortAttrs(attributes []*vmetricsv1.Attribute) {
	sort.Slice(attributes, func(i, j int) bool {
		if attributes[i].Key != attributes[j].Key {
			return attributes[i].Key < attributes[j].Key
		}
		return mockAttributeValueKey(attributes[i].Value) <
			mockAttributeValueKey(attributes[j].Value)
	})
}

func mockGroupValues(key string) []string {
	switch key {
	case "octelium.component.type":
		return []string{"vigil", "octovigil", "rscserver", "portal", "authserver", "gateway"}
	case "octelium.component.namespace":
		return []string{"octelium"}
	case "octelium.component.name":
		return []string{"vigil-7d9f", "vigil-4a2b", "octovigil-66c1"}
	case "octelium.vigil.svc.name":
		return []string{"api", "web", "db-proxy"}
	case "octelium.vigil.svc.namespace.name":
		return []string{"default", "production"}
	case "octelium.vigil.svc.region.name":
		return []string{"default", "edge"}
	case "octelium.vigil.svc.mode":
		return []string{"HTTP", "TCP"}
	case "http.method":
		return []string{"GET", "POST", "PUT", "DELETE"}
	default:
		return []string{"series-a", "series-b", "series-c"}
	}
}

func mockHistogram(
	info mockMetricInfo,
	phase float64,
	timestamp time.Time,
	stepSeconds float64,
) ([]float64, []uint64, float64) {
	total := int64(mockLevel(info.base, info.amp, phase, timestamp) * stepSeconds)
	if total < 1 {
		total = 1
	}

	distributionFraction := mockLatFraction(timestamp, phase)
	counts := make([]uint64, len(mockHistBounds)+1)
	var assigned int64

	for idx := range counts {
		weight := mockHistWFast[idx]*(1-distributionFraction) +
			mockHistWSlow[idx]*distributionFraction
		count := int64(float64(total) * weight)
		counts[idx] = uint64(count)
		assigned += count
	}
	if assigned < total {
		counts[3] += uint64(total - assigned)
	}

	var sum float64
	for idx, count := range counts {
		var midpoint float64
		switch {
		case idx == 0:
			midpoint = mockHistBounds[0] / 2
		case idx < len(mockHistBounds):
			midpoint = (mockHistBounds[idx-1] + mockHistBounds[idx]) / 2
		default:
			midpoint = mockHistBounds[len(mockHistBounds)-1] * 1.4
		}
		sum += midpoint * float64(count)
	}

	return mockHistBounds, counts, sum
}

func mockQuantile(bounds []float64, counts []uint64, quantile float64) float64 {
	total := mockSumCounts(counts)
	if total == 0 {
		return 0
	}

	target := quantile * float64(total)
	var cumulative uint64
	var previousUpperBound float64

	for idx, count := range counts {
		cumulative += count
		if float64(cumulative) >= target {
			if idx >= len(bounds) {
				return previousUpperBound
			}
			if count == 0 {
				return bounds[idx]
			}
			position := (target - float64(cumulative-count)) / float64(count)
			return previousUpperBound + position*(bounds[idx]-previousUpperBound)
		}
		if idx < len(bounds) {
			previousUpperBound = bounds[idx]
		}
	}

	return previousUpperBound
}

func mockCumulativeBuckets(
	bounds []float64,
	counts []uint64,
) []*vmetricsv1.HistogramBucket {
	ret := make([]*vmetricsv1.HistogramBucket, 0, len(counts))
	var cumulative uint64

	for idx, count := range counts {
		cumulative += count
		if idx < len(bounds) {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Le:    bounds[idx],
				Count: cumulative,
			})
		} else {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Count: cumulative,
				IsInf: true,
			})
		}
	}

	return ret
}

func mockSumCounts(counts []uint64) uint64 {
	var total uint64
	for _, count := range counts {
		total += count
	}
	return total
}

func mockLevel(base, amplitude, phase float64, timestamp time.Time) float64 {
	hourFraction := float64(timestamp.Hour()) +
		float64(timestamp.Minute())/60.0 +
		float64(timestamp.Second())/3600.0

	diurnal := 1 +
		amplitude*0.6*math.Sin((hourFraction/24)*2*math.Pi+phase) +
		amplitude*0.22*math.Sin((hourFraction/6)*2*math.Pi+phase*1.7) +
		amplitude*0.08*math.Sin((hourFraction/1.5)*2*math.Pi+phase*0.5)

	jitter := 1 + 0.03*math.Sin(float64(timestamp.Unix())/89.0+phase*3)
	value := base * diurnal * jitter
	if value < 0 {
		return 0
	}
	return value
}

func mockLatFraction(timestamp time.Time, phase float64) float64 {
	hourFraction := float64(timestamp.Hour()) + float64(timestamp.Minute())/60.0
	return 0.5 + 0.5*math.Sin((hourFraction/24)*2*math.Pi+phase)
}

func mockPhase(seed string) float64 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return float64(hash.Sum32()%1000) / 1000.0 * 2 * math.Pi
}
