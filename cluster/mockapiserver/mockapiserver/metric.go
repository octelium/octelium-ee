// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
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

type tstMetricsService struct {
	vmetricsv1.UnimplementedMetricsServiceServer
}

type mockMetricInfo struct {
	kind          vmetricsv1.MetricDescriptor_Kind
	valueType     vmetricsv1.MetricDescriptor_ValueType
	unit          string
	description   string
	temporality   vmetricsv1.MetricDescriptor_Temporality
	attributeKeys []string
	base          float64
	amp           float64
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
	"req.total":                  {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "requests", "Total number of requests", vmetricsv1.MetricDescriptor_CUMULATIVE, mockVigilKeys, 220, 0.6},
	"req.active":                 {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "requests", "Number of active requests", vmetricsv1.MetricDescriptor_CUMULATIVE, mockVigilKeys, 36, 0.5},
	"req.duration":               {vmetricsv1.MetricDescriptor_HISTOGRAM, vmetricsv1.MetricDescriptor_DOUBLE, "ms", "Request duration in milliseconds", vmetricsv1.MetricDescriptor_DELTA, mockVigilKeys, 220, 0.6},
	"authorization.req.total":    {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "requests", "Total number of authorization requests", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 180, 0.6},
	"authorization.req.active":   {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "requests", "Number of active authorization requests", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 22, 0.5},
	"authorization.req.duration": {vmetricsv1.MetricDescriptor_HISTOGRAM, vmetricsv1.MetricDescriptor_DOUBLE, "ms", "Authorization request duration in milliseconds", vmetricsv1.MetricDescriptor_DELTA, mockComponentKeys, 180, 0.55},
	"process.goroutines":         {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "", "Number of goroutines", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 600, 0.3},
	"process.uptime":             {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "seconds", "Process uptime in seconds", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 1, 0.0},
	"process.gomaxprocs":         {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "", "Current GOMAXPROCS setting", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 8, 0.0},
	"process.mem.heap_alloc":     {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Bytes of allocated heap objects", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 180e6, 0.3},
	"process.mem.total":          {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Total bytes of memory mapped by the Go runtime", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 320e6, 0.15},
	"process.mem.heap_released":  {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Heap memory returned to the operating system", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 90e6, 0.2},
	"process.mem.stacks":         {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Memory used by goroutine stacks", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 24e6, 0.2},
	"process.mem.heap_objects":   {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "", "Number of live heap objects", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 1.2e6, 0.25},
	"process.mem.alloc_bytes":    {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Cumulative bytes allocated for heap objects", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 5e6, 0.5},
	"process.mem.alloc_objects":  {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "", "Cumulative count of heap objects allocated", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 60000, 0.5},
	"process.gc.cycles":          {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_INT64, "", "Completed GC cycles", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 0.06, 0.4},
	"process.gc.heap_goal":       {vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_INT64, "bytes", "Target heap size for the next GC cycle", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 360e6, 0.15},
	"process.cpu.gc_seconds":     {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_DOUBLE, "seconds", "Cumulative CPU time spent in garbage collection", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 0.02, 0.4},
	"process.cpu.seconds":        {vmetricsv1.MetricDescriptor_COUNTER, vmetricsv1.MetricDescriptor_DOUBLE, "seconds", "Cumulative CPU time consumed by the process", vmetricsv1.MetricDescriptor_CUMULATIVE, mockComponentKeys, 0.45, 0.4},
}

func (s *tstMetricsService) QueryMetrics(ctx context.Context, req *vmetricsv1.QueryMetricsRequest) (*vmetricsv1.QueryMetricsResponse, error) {
	if req == nil || req.Metric == nil || strings.TrimSpace(req.Metric.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "metric.name must be set")
	}

	name := strings.TrimSpace(req.Metric.Name)
	info := mockInfoFor(name)

	from, to := mockTimeRange(req.TimeRange)
	step := mockStep(req.Step, from, to)

	n := int(to.Sub(from) / step)
	if n < 1 {
		n = 1
	}
	if n > 500 {
		n = 500
		step = to.Sub(from) / time.Duration(n)
	}

	op := req.Operation

	type mockGroup struct {
		labels []*vmetricsv1.Attribute
		phase  float64
	}

	var groups []mockGroup
	if len(req.GroupBy) > 0 {
		key := req.GroupBy[0]
		for _, val := range mockGroupValues(key) {
			labels := []*vmetricsv1.Attribute{{Key: key, Value: val}}
			for _, extra := range req.GroupBy[1:] {
				labels = append(labels, &vmetricsv1.Attribute{Key: extra, Value: mockGroupValues(extra)[0]})
			}
			mockSortAttrs(labels)
			groups = append(groups, mockGroup{labels: labels, phase: mockPhase(name + "|" + val)})
		}
	} else {
		groups = []mockGroup{{labels: nil, phase: mockPhase(name)}}
	}

	truncated := false
	if limit := int(req.LimitSeries); limit > 0 && len(groups) > limit {
		groups = groups[:limit]
		truncated = true
	}

	histOp := (*vmetricsv1.HistogramOperation)(nil)
	if op != nil {
		histOp = op.GetHistogram()
	}

	var series []*vmetricsv1.TimeSeries

	for _, g := range groups {
		switch {
		case histOp != nil && histOp.Function == vmetricsv1.HistogramOperation_BUCKETS:
			series = append(series, &vmetricsv1.TimeSeries{
				Labels: g.labels,
				Points: &vmetricsv1.TimeSeries_Histogram{Histogram: s.genHistogramSeries(info, g.phase, from, step, n)},
			})

		case histOp != nil && histOp.Function == vmetricsv1.HistogramOperation_QUANTILE:
			quantiles := histOp.Quantiles
			if len(quantiles) == 0 {
				quantiles = []float64{0.5, 0.9, 0.99}
			}
			for _, q := range quantiles {
				labels := append([]*vmetricsv1.Attribute{}, g.labels...)
				labels = append(labels, &vmetricsv1.Attribute{Key: "quantile", Value: mockFormatQuantile(q)})
				mockSortAttrs(labels)
				series = append(series, &vmetricsv1.TimeSeries{
					Labels: labels,
					Points: &vmetricsv1.TimeSeries_Number{Number: s.genQuantileSeries(info, g.phase, from, step, n, q)},
				})
			}

		default:
			series = append(series, &vmetricsv1.TimeSeries{
				Labels: g.labels,
				Points: &vmetricsv1.TimeSeries_Number{Number: s.genNumberSeries(info, op, g.phase, from, step, n)},
			})
		}
	}

	return &vmetricsv1.QueryMetricsResponse{
		Descriptor_: mockDescriptor(name, info),
		Operation:   op,
		Step:        mockDurationPB(step),
		Series:      series,
		Truncated:   truncated,
		TotalSeries: uint32(len(series)),
	}, nil
}

func (s *tstMetricsService) ListMetricDescriptors(ctx context.Context, req *vmetricsv1.ListMetricDescriptorsRequest) (*vmetricsv1.ListMetricDescriptorsResponse, error) {
	limit := 1000
	var prefix, pageToken string
	if req != nil {
		if req.Limit > 0 && int(req.Limit) < limit {
			limit = int(req.Limit)
		}
		prefix = req.NamePrefix
		pageToken = req.PageToken
	}

	names := make([]string, 0, len(mockMetricMeta))
	for n := range mockMetricMeta {
		names = append(names, n)
	}
	sort.Strings(names)

	var items []*vmetricsv1.MetricDescriptor
	for _, n := range names {
		if prefix != "" && !strings.HasPrefix(n, prefix) {
			continue
		}
		if pageToken != "" && n <= pageToken {
			continue
		}
		items = append(items, mockDescriptor(n, mockMetricMeta[n]))
		if len(items) > limit {
			break
		}
	}

	resp := &vmetricsv1.ListMetricDescriptorsResponse{}
	if len(items) > limit {
		items = items[:limit]
		resp.NextPageToken = items[len(items)-1].Name
	}
	resp.Items = items

	return resp, nil
}

func (s *tstMetricsService) ListMetricCatalog(ctx context.Context, req *vmetricsv1.ListMetricCatalogRequest) (*vmetricsv1.ListMetricCatalogResponse, error) {
	var c *vmetricsv1.ComponentSelector
	if req != nil {
		c = req.Component
	}

	groupByType := []string{"octelium.component.type", "octelium.component.namespace"}

	items := []*vmetricsv1.MetricCatalogItem{
		mockCatalogItem("memory_total", "Memory usage", "Total memory used by the component", "process.mem.total", mockGaugeOp(vmetricsv1.GaugeOperation_LAST), groupByType, "bytes"),
		mockCatalogItem("heap_alloc", "Heap allocated", "Bytes of live heap objects", "process.mem.heap_alloc", mockGaugeOp(vmetricsv1.GaugeOperation_LAST), groupByType, "bytes"),
		mockCatalogItem("goroutines", "Goroutines", "Number of goroutines", "process.goroutines", mockGaugeOp(vmetricsv1.GaugeOperation_LAST), groupByType, ""),
		mockCatalogItem("cpu_usage", "CPU usage", "CPU cores consumed", "process.cpu.seconds", mockCounterOp(vmetricsv1.CounterOperation_RATE), groupByType, "cores"),
		mockCatalogItem("gc_rate", "GC rate", "Completed GC cycles per second", "process.gc.cycles", mockCounterOp(vmetricsv1.CounterOperation_RATE), groupByType, "cycles/s"),
	}

	if mockComponentMatches(c, "vigil", "octovigil", "rscserver", "portal", "authserver") {
		items = append(items,
			mockCatalogItem("requests_rate", "Requests rate", "Per-second request rate", "req.total", mockCounterOp(vmetricsv1.CounterOperation_RATE), groupByType, "requests/s"),
			mockCatalogItem("active_requests", "Active requests", "Current active requests", "req.active", mockGaugeOp(vmetricsv1.GaugeOperation_LAST), groupByType, "requests"),
			mockCatalogItem("request_p95_latency", "Request p95 latency", "p95 request duration", "req.duration", mockHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.95), groupByType, "ms"),
		)
	}

	if mockComponentMatches(c, "octovigil") {
		items = append(items,
			mockCatalogItem("authorization_requests_rate", "Authorization rate", "Per-second authorization request rate", "authorization.req.total", mockCounterOp(vmetricsv1.CounterOperation_RATE), groupByType, "requests/s"),
		)
	}

	return &vmetricsv1.ListMetricCatalogResponse{Items: items}, nil
}

func (s *tstMetricsService) genNumberSeries(info mockMetricInfo, op *vmetricsv1.QueryOperation, phase float64, from time.Time, step time.Duration, n int) *vmetricsv1.NumberPointSeries {
	stepSec := step.Seconds()
	pts := &vmetricsv1.NumberPointSeries{}

	rawInt := op != nil && op.GetCounter() != nil &&
		op.GetCounter().Function == vmetricsv1.CounterOperation_RAW &&
		info.valueType == vmetricsv1.MetricDescriptor_INT64

	cum := info.base * 1000

	for i := 0; i < n; i++ {
		ts := from.Add(time.Duration(i+1) * step)
		level := mockLevel(info.base, info.amp, phase, ts)
		var v float64

		switch {
		case op != nil && op.GetCounter() != nil:
			switch op.GetCounter().Function {
			case vmetricsv1.CounterOperation_RATE:
				v = level
			case vmetricsv1.CounterOperation_INCREASE:
				v = level * stepSec
			case vmetricsv1.CounterOperation_RAW:
				cum += level * stepSec
				v = cum
			default:
				v = level
			}

		case op != nil && op.GetGauge() != nil:
			switch op.GetGauge().Function {
			case vmetricsv1.GaugeOperation_MIN:
				v = level * 0.82
			case vmetricsv1.GaugeOperation_MAX:
				v = level * 1.22
			case vmetricsv1.GaugeOperation_SUM:
				v = level * 4
			default:
				v = level
			}

		case op != nil && op.GetHistogram() != nil:
			bounds, counts, sum := mockHistogram(info, phase, ts, stepSec)
			switch op.GetHistogram().Function {
			case vmetricsv1.HistogramOperation_COUNT:
				v = float64(mockSumCounts(counts))
			case vmetricsv1.HistogramOperation_SUM:
				v = sum
			case vmetricsv1.HistogramOperation_AVG:
				if c := mockSumCounts(counts); c > 0 {
					v = sum / float64(c)
				}
			case vmetricsv1.HistogramOperation_MIN:
				v = bounds[0] * 0.5
			case vmetricsv1.HistogramOperation_MAX:
				v = bounds[len(bounds)-1] * 1.3
			default:
				v = mockQuantile(bounds, counts, 0.5)
			}

		default:
			v = level
		}

		pts.Points = append(pts.Points, mockNumberPoint(ts, v, rawInt))
	}

	return pts
}

func (s *tstMetricsService) genQuantileSeries(info mockMetricInfo, phase float64, from time.Time, step time.Duration, n int, q float64) *vmetricsv1.NumberPointSeries {
	stepSec := step.Seconds()
	pts := &vmetricsv1.NumberPointSeries{}

	for i := 0; i < n; i++ {
		ts := from.Add(time.Duration(i+1) * step)
		bounds, counts, _ := mockHistogram(info, phase, ts, stepSec)
		pts.Points = append(pts.Points, mockNumberPoint(ts, mockQuantile(bounds, counts, q), false))
	}

	return pts
}

func (s *tstMetricsService) genHistogramSeries(info mockMetricInfo, phase float64, from time.Time, step time.Duration, n int) *vmetricsv1.HistogramPointSeries {
	stepSec := step.Seconds()
	pts := &vmetricsv1.HistogramPointSeries{}

	for i := 0; i < n; i++ {
		ts := from.Add(time.Duration(i+1) * step)
		bounds, counts, sum := mockHistogram(info, phase, ts, stepSec)
		pts.Points = append(pts.Points, &vmetricsv1.HistogramPoint{
			Timestamp: pbutils.Timestamp(ts),
			Sum:       sum,
			Count:     mockSumCounts(counts),
			Buckets:   mockCumulativeBuckets(bounds, counts),
		})
	}

	return pts
}

var (
	mockHistBounds = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500}
	mockHistWFast  = []float64{0.10, 0.16, 0.26, 0.24, 0.13, 0.06, 0.03, 0.012, 0.006, 0.002}
	mockHistWSlow  = []float64{0.05, 0.09, 0.18, 0.22, 0.18, 0.12, 0.08, 0.04, 0.025, 0.015}
)

func mockHistogram(info mockMetricInfo, phase float64, t time.Time, stepSec float64) ([]float64, []uint64, float64) {
	total := int64(mockLevel(info.base, info.amp, phase, t) * stepSec)
	if total < 1 {
		total = 1
	}

	df := mockLatFraction(t, phase)

	counts := make([]uint64, len(mockHistBounds)+1)
	var assigned int64
	for i := range counts {
		w := mockHistWFast[i]*(1-df) + mockHistWSlow[i]*df
		c := int64(float64(total) * w)
		counts[i] = uint64(c)
		assigned += c
	}
	if assigned < total {
		counts[3] += uint64(total - assigned)
	}

	var sum float64
	for i, c := range counts {
		var mid float64
		switch {
		case i == 0:
			mid = mockHistBounds[0] / 2
		case i < len(mockHistBounds):
			mid = (mockHistBounds[i-1] + mockHistBounds[i]) / 2
		default:
			mid = mockHistBounds[len(mockHistBounds)-1] * 1.4
		}
		sum += mid * float64(c)
	}

	return mockHistBounds, counts, sum
}

func mockQuantile(bounds []float64, counts []uint64, q float64) float64 {
	total := mockSumCounts(counts)
	if total == 0 {
		return 0
	}

	target := q * float64(total)

	var cum uint64
	var prevLe float64
	for i, c := range counts {
		cum += c
		if float64(cum) >= target {
			if i >= len(bounds) {
				return prevLe
			}
			if c == 0 {
				return bounds[i]
			}
			pos := (target - float64(cum-c)) / float64(c)
			return prevLe + pos*(bounds[i]-prevLe)
		}
		if i < len(bounds) {
			prevLe = bounds[i]
		}
	}

	return prevLe
}

func mockCumulativeBuckets(bounds []float64, counts []uint64) []*vmetricsv1.HistogramBucket {
	ret := make([]*vmetricsv1.HistogramBucket, 0, len(counts))
	var cumulative uint64

	for i, c := range counts {
		cumulative += c
		if i < len(bounds) {
			ret = append(ret, &vmetricsv1.HistogramBucket{Le: bounds[i], Count: cumulative})
		} else {
			ret = append(ret, &vmetricsv1.HistogramBucket{Count: cumulative, IsInf: true})
		}
	}

	return ret
}

func mockSumCounts(counts []uint64) uint64 {
	var total uint64
	for _, c := range counts {
		total += c
	}
	return total
}

func mockLevel(base, amp, phase float64, t time.Time) float64 {
	hf := float64(t.Hour()) + float64(t.Minute())/60.0 + float64(t.Second())/3600.0
	d := 1 +
		amp*0.6*math.Sin((hf/24)*2*math.Pi+phase) +
		amp*0.22*math.Sin((hf/6)*2*math.Pi+phase*1.7) +
		amp*0.08*math.Sin((hf/1.5)*2*math.Pi+phase*0.5)
	j := 1 + 0.03*math.Sin(float64(t.Unix())/89.0+phase*3)
	v := base * d * j
	if v < 0 {
		v = 0
	}
	return v
}

func mockLatFraction(t time.Time, phase float64) float64 {
	hf := float64(t.Hour()) + float64(t.Minute())/60.0
	return 0.5 + 0.5*math.Sin((hf/24)*2*math.Pi+phase)
}

func mockDiurnalCount(t time.Time, base, amp float64) int64 {
	v := mockLevel(base, amp, 0.7, t)
	if v < 1 {
		v = 1
	}
	return int64(v)
}

func mockGroupValues(key string) []string {
	switch key {
	case "octelium.component.type":
		return []string{"vigil", "octovigil", "rscserver", "portal"}
	case "octelium.component.namespace":
		return []string{"octelium"}
	case "octelium.component.name":
		return []string{"vigil-7d9f", "vigil-4a2b", "octovigil-66c1"}
	case "octelium.vigil.svc.name":
		return []string{"api", "web", "db-proxy"}
	case "octelium.vigil.svc.mode":
		return []string{"HTTP", "TCP"}
	case "http.method":
		return []string{"GET", "POST", "PUT", "DELETE"}
	default:
		return []string{"series-a", "series-b", "series-c"}
	}
}

func mockInfoFor(name string) mockMetricInfo {
	if info, ok := mockMetricMeta[name]; ok {
		return info
	}
	return mockMetricInfo{
		kind:          vmetricsv1.MetricDescriptor_GAUGE,
		valueType:     vmetricsv1.MetricDescriptor_DOUBLE,
		temporality:   vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET,
		attributeKeys: mockComponentKeys,
		base:          50,
		amp:           0.4,
	}
}

func mockDescriptor(name string, info mockMetricInfo) *vmetricsv1.MetricDescriptor {
	return &vmetricsv1.MetricDescriptor{
		Name:          name,
		Kind:          info.kind,
		ValueType:     info.valueType,
		Unit:          info.unit,
		Description:   info.description,
		AttributeKeys: info.attributeKeys,
		Temporality:   info.temporality,
	}
}

func mockCatalogItem(id, displayName, description, metricName string, op *vmetricsv1.QueryOperation, groupBy []string, unit string) *vmetricsv1.MetricCatalogItem {
	info := mockInfoFor(metricName)
	return &vmetricsv1.MetricCatalogItem{
		Id:               id,
		DisplayName:      displayName,
		Description:      description,
		Metric:           &vmetricsv1.MetricSelector{Name: metricName, Kind: info.kind},
		DefaultOperation: op,
		DefaultGroupBy:   groupBy,
		Unit:             unit,
	}
}

func mockComponentMatches(c *vmetricsv1.ComponentSelector, types ...string) bool {
	if c == nil || c.Type == "" {
		return true
	}
	for _, t := range types {
		if c.Type == t {
			return true
		}
	}
	return false
}

func mockGaugeOp(fn vmetricsv1.GaugeOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{Function: fn}}}
}

func mockCounterOp(fn vmetricsv1.CounterOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{Function: fn}}}
}

func mockHistogramOp(fn vmetricsv1.HistogramOperation_Function, quantiles ...float64) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Histogram{Histogram: &vmetricsv1.HistogramOperation{Function: fn, Quantiles: quantiles}}}
}

func mockNumberPoint(t time.Time, v float64, asInt bool) *vmetricsv1.NumberPoint {
	if asInt {
		return &vmetricsv1.NumberPoint{Timestamp: pbutils.Timestamp(t), Value: &vmetricsv1.NumberPoint_AsInt{AsInt: int64(v)}}
	}
	return &vmetricsv1.NumberPoint{Timestamp: pbutils.Timestamp(t), Value: &vmetricsv1.NumberPoint_AsDouble{AsDouble: v}}
}

func mockTimeRange(tr *vmetricsv1.TimeRange) (time.Time, time.Time) {
	now := time.Now().UTC()
	if tr == nil || tr.From == nil || tr.To == nil {
		return now.Add(-6 * time.Hour), now
	}
	from := tr.From.AsTime().UTC()
	to := tr.To.AsTime().UTC()
	if !to.After(from) {
		return now.Add(-6 * time.Hour), now
	}
	return from, to
}

func mockStep(d *metav1.Duration, from, to time.Time) time.Duration {
	if step := mockDurationToTime(d); step > 0 {
		return step
	}
	step := to.Sub(from) / 120
	if step < time.Minute {
		step = time.Minute
	}
	return step
}

func mockDurationToTime(d *metav1.Duration) time.Duration {
	if d == nil {
		return 0
	}
	switch {
	case d.GetMilliseconds() > 0:
		return time.Duration(d.GetMilliseconds()) * time.Millisecond
	case d.GetSeconds() > 0:
		return time.Duration(d.GetSeconds()) * time.Second
	case d.GetMinutes() > 0:
		return time.Duration(d.GetMinutes()) * time.Minute
	case d.GetHours() > 0:
		return time.Duration(d.GetHours()) * time.Hour
	case d.GetDays() > 0:
		return time.Duration(d.GetDays()) * 24 * time.Hour
	case d.GetWeeks() > 0:
		return time.Duration(d.GetWeeks()) * 7 * 24 * time.Hour
	}
	return 0
}

func mockDurationPB(d time.Duration) *metav1.Duration {
	if d%time.Second == 0 {
		return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: uint32(d / time.Second)}}
	}
	return &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: uint32(d / time.Millisecond)}}
}

func mockSortAttrs(a []*vmetricsv1.Attribute) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Key == a[j].Key {
			return a[i].Value < a[j].Value
		}
		return a[i].Key < a[j].Key
	})
}

func mockFormatQuantile(q float64) string {
	return strconv.FormatFloat(q, 'f', -1, 64)
}

func mockPhase(seed string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return float64(h.Sum32()%1000) / 1000.0 * 2 * math.Pi
}
