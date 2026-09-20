// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestMemoryLimitedServer(t *testing.T, memoryLimit string) *srvMetric {
	t.Helper()

	database := filepath.Join(t.TempDir(), "metricstore.db")

	values := url.Values{}
	values.Set("access_mode", "read_write")
	values.Set("memory_limit", memoryLimit)
	values.Set("preserve_insertion_order", "false")
	values.Set("threads", "2")

	s := &Server{
		dbConfig: &metricStoreDBConfig{dsn: database + "?" + values.Encode(), database: database},
	}

	db, err := openMetricStoreDB(s.dbConfig)
	require.NoError(t, err)
	s.setDatabase(db)
	t.Cleanup(func() {
		if db := s.database(); db != nil {
			_ = db.Close()
		}
	})

	require.NoError(t, s.initDB(context.Background()))

	return &srvMetric{s: s}
}

func newTestMetadataMetrics(descriptors, seriesPerDescriptor int,
	attributes map[string]any) pmetric.Metrics {
	now := time.Now().UTC()

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	putTestAttributes(rm.Resource().Attributes(), map[string]any{
		"octelium.component.type":      "vigil",
		"octelium.component.namespace": "octelium",
		"octelium.component.uid":       "octelium-vigil-1",
	})
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("default")

	for d := 0; d < descriptors; d++ {
		metric := sm.Metrics().AppendEmpty()
		metric.SetName(fmt.Sprintf("octelium.test.metric_%02d", d))
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

		for i := 0; i < seriesPerDescriptor; i++ {
			point := sum.DataPoints().AppendEmpty()
			point.SetStartTimestamp(pcommon.NewTimestampFromTime(now.Add(-time.Hour)))
			point.SetTimestamp(pcommon.NewTimestampFromTime(now))
			point.SetIntValue(int64(i))

			if attributes != nil {
				putTestAttributes(point.Attributes(), attributes)
				continue
			}

			putTestAttributes(point.Attributes(), map[string]any{
				"service": fmt.Sprintf("svc-%d", i%13),
				"method":  fmt.Sprintf("m-%d", i%5),
				"status":  fmt.Sprintf("s-%d", i%3),
			})
		}
	}

	return metrics
}

func storeTestMetricsErr(t *testing.T, srv *srvMetric, metrics pmetric.Metrics) error {
	t.Helper()

	batch, err := buildMetricWriteBatch(metrics)
	require.NoError(t, err)
	batch.normalize()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	return srv.storeMetricWriteBatch(ctx, batch)
}

func TestUpsertMetricMetadataStoresEveryTable(t *testing.T) {
	srv := newTestSrvMetric(t)

	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(3, 4, nil)))

	assert.Equal(t, int64(3), testTableCount(t, srv.s, "metric_descriptors"))
	assert.Equal(t, int64(12), testTableCount(t, srv.s, "metric_series"))
	assert.Equal(t, int64(12), testTableCount(t, srv.s, "metric_number_points"))

	var seriesAttributes int
	require.NoError(t, srv.s.database().QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM metric_series_attributes WHERE key = 'service'`).Scan(&seriesAttributes))
	assert.Equal(t, 12, seriesAttributes)

	var attributeKeys int
	require.NoError(t, srv.s.database().QueryRowContext(context.Background(),
		`SELECT COUNT(DISTINCT key) FROM metric_attribute_keys`).Scan(&attributeKeys))
	assert.Equal(t, 6, attributeKeys)

	var labels string
	require.NoError(t, srv.s.database().QueryRowContext(context.Background(),
		`SELECT CAST(labels AS VARCHAR) FROM metric_series LIMIT 1`).Scan(&labels))
	assert.Contains(t, labels, "service")

	for _, table := range []string{
		"metric_descriptors_staging", "metric_series_staging",
		"metric_series_attributes_staging", "metric_attribute_keys_staging",
	} {
		assert.Zero(t, testTableCount(t, srv.s, table), table)
	}
}

func TestUpsertMetricMetadataIsIdempotent(t *testing.T) {
	srv := newTestSrvMetric(t)
	ctx := context.Background()

	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(2, 3, nil)))

	var createdAt int64
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT min(created_at) FROM metric_series`).Scan(&createdAt))

	var firstSeenAt int64
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT min(first_seen_at) FROM metric_attribute_keys`).Scan(&firstSeenAt))

	descriptors := testTableCount(t, srv.s, "metric_descriptors")
	series := testTableCount(t, srv.s, "metric_series")
	seriesAttributes := testTableCount(t, srv.s, "metric_series_attributes")
	attributeKeys := testTableCount(t, srv.s, "metric_attribute_keys")

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(2, 3, nil)))

	assert.Equal(t, descriptors, testTableCount(t, srv.s, "metric_descriptors"))
	assert.Equal(t, series, testTableCount(t, srv.s, "metric_series"))
	assert.Equal(t, seriesAttributes, testTableCount(t, srv.s, "metric_series_attributes"))
	assert.Equal(t, attributeKeys, testTableCount(t, srv.s, "metric_attribute_keys"))

	var createdAtAfter int64
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT min(created_at) FROM metric_series`).Scan(&createdAtAfter))
	assert.Equal(t, createdAt, createdAtAfter)

	var firstSeenAtAfter int64
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT min(first_seen_at) FROM metric_attribute_keys`).Scan(&firstSeenAtAfter))
	assert.Equal(t, firstSeenAt, firstSeenAtAfter)

	var updatedAt int64
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT min(updated_at) FROM metric_series`).Scan(&updatedAt))
	assert.Greater(t, updatedAt, createdAt)
}

func TestUpsertMetricMetadataMergesTheAttributeSourceMask(t *testing.T) {
	srv := newTestSrvMetric(t)
	ctx := context.Background()

	resourceOnly := newTestMetadataMetrics(1, 1, map[string]any{})
	putTestAttributes(resourceOnly.ResourceMetrics().At(0).Resource().Attributes(),
		map[string]any{"shared": "value"})
	require.NoError(t, storeTestMetricsErr(t, srv, resourceOnly))

	var mask int
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT source_mask FROM metric_attribute_keys WHERE key = 'shared'`).Scan(&mask))
	assert.Equal(t, attributeSourceResource, mask)

	pointOnly := newTestMetadataMetrics(1, 1, map[string]any{"shared": "value"})
	require.NoError(t, storeTestMetricsErr(t, srv, pointOnly))

	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT source_mask FROM metric_attribute_keys WHERE key = 'shared'`).Scan(&mask))
	assert.Equal(t, attributeSourceResource|attributeSourceDataPoint, mask)

	var seriesMask int
	require.NoError(t, srv.s.database().QueryRowContext(ctx, `
SELECT max(source_mask) FROM metric_series_attributes WHERE key = 'shared'`).Scan(&seriesMask))
	assert.NotZero(t, seriesMask)
}

func TestUpsertMetricMetadataRejectsChangedAttributeValueKind(t *testing.T) {
	srv := newTestSrvMetric(t)

	require.NoError(t, storeTestMetricsErr(t, srv,
		newTestMetadataMetrics(1, 1, map[string]any{"mixed": "text"})))

	err := storeTestMetricsErr(t, srv,
		newTestMetadataMetrics(1, 1, map[string]any{"mixed": int64(7)}))
	require.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "changed value kind")

	assert.Zero(t, testTableCount(t, srv.s, "metric_attribute_keys_staging"))

	var kind string
	require.NoError(t, srv.s.database().QueryRowContext(context.Background(),
		`SELECT value_kind FROM metric_attribute_keys WHERE key = 'mixed'`).Scan(&kind))
	assert.Equal(t, "STRING", kind)
}

func TestUpsertMetricMetadataKeepsTheDescriptorDescription(t *testing.T) {
	srv := newTestSrvMetric(t)
	ctx := context.Background()

	described := newTestMetadataMetrics(1, 1, nil)
	described.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).
		SetDescription("the original description")
	require.NoError(t, storeTestMetricsErr(t, srv, described))

	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(1, 1, nil)))

	var description string
	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT description FROM metric_descriptors`).Scan(&description))
	assert.Equal(t, "the original description", description)

	updated := newTestMetadataMetrics(1, 1, nil)
	updated.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).
		SetDescription("a newer description")
	require.NoError(t, storeTestMetricsErr(t, srv, updated))

	require.NoError(t, srv.s.database().QueryRowContext(ctx,
		`SELECT description FROM metric_descriptors`).Scan(&description))
	assert.Equal(t, "a newer description", description)
}

func TestStoreMetricWriteBatchRollsBackOnFailure(t *testing.T) {
	srv := newTestSrvMetric(t)

	require.NoError(t, storeTestMetricsErr(t, srv,
		newTestMetadataMetrics(1, 1, map[string]any{"mixed": "text"})))

	descriptors := testTableCount(t, srv.s, "metric_descriptors")
	points := testTableCount(t, srv.s, "metric_number_points")

	conflicting := newTestMetadataMetrics(2, 2, map[string]any{"mixed": int64(7)})
	require.NotNil(t, storeTestMetricsErr(t, srv, conflicting))

	assert.Equal(t, descriptors, testTableCount(t, srv.s, "metric_descriptors"))
	assert.Equal(t, points, testTableCount(t, srv.s, "metric_number_points"))
	assert.Zero(t, testTableCount(t, srv.s, "metric_series_staging"))
	assert.Zero(t, testTableCount(t, srv.s, "metric_number_staging"))
}

func TestStoreLargeMetadataBatchWithinTheMemoryLimit(t *testing.T) {
	srv := newTestMemoryLimitedServer(t, "256MB")

	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(20, 20, nil)))

	assert.Equal(t, int64(20), testTableCount(t, srv.s, "metric_descriptors"))
	assert.Equal(t, int64(400), testTableCount(t, srv.s, "metric_series"))
	assert.Equal(t, int64(400), testTableCount(t, srv.s, "metric_number_points"))
	assert.Equal(t, int64(2400), testTableCount(t, srv.s, "metric_series_attributes"))

	require.NoError(t, storeTestMetricsErr(t, srv, newTestMetadataMetrics(20, 20, nil)))
	assert.Equal(t, int64(400), testTableCount(t, srv.s, "metric_series"))
}
