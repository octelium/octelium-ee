// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestDescriptorIDIdentityFields(t *testing.T) {
	base := &descriptorRecord{
		name:            "req.duration",
		kind:            vmetricsv1.MetricDescriptor_HISTOGRAM,
		numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		unit:            "ms",
		temporality:     vmetricsv1.MetricDescriptor_DELTA,
		scopeName:       "octelium",
		scopeVersion:    "1.0.0",
		scopeSchemaURL:  "https://opentelemetry.io/schemas/1.30.0",
		explicitBounds:  []float64{10, 100, 1000},
	}

	id := descriptorID(base)
	base.description = "Changed description"
	assert.Equal(t, id, descriptorID(base))

	base.explicitBounds = []float64{10, 100, 500}
	assert.NotEqual(t, id, descriptorID(base))
}

func TestAttributesFromMapsPrecedence(t *testing.T) {
	resource := pcommon.NewMap()
	resource.PutStr("shared", "resource")
	resource.PutStr("resource-only", "value")

	scope := pcommon.NewMap()
	scope.PutStr("shared", "scope")
	scope.PutInt("scope-only", 10)

	point := pcommon.NewMap()
	point.PutStr("shared", "point")
	point.PutBool("point-only", true)

	attributes, err := attributesFromMaps(resource, scope, point)
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, attributes, 4)

	shared := findStoredAttribute(attributes, "shared")
	assert.NotNil(t, shared)
	assert.Equal(t, "point", shared.StringValue)
	assert.Equal(t, attributeSourceResource|attributeSourceScope|attributeSourceDataPoint, shared.SourceMask)
}

func TestMetricWriteBatchDeduplicatesPoints(t *testing.T) {
	now := normalizeMetricTime(time.Now())
	seriesID := "series"
	pointID := pointID("NUMBER", seriesID, now, nil)

	batch := newMetricWriteBatch()
	batch.numberPoints = []numberPointRecord{
		{pointID: pointID, seriesID: seriesID, timestamp: now, ingestedAt: now},
		{pointID: pointID, seriesID: seriesID, timestamp: now, ingestedAt: now.Add(time.Second)},
	}
	batch.normalize()

	assert.Len(t, batch.numberPoints, 1)
	assert.Equal(t, now, batch.numberPoints[0].ingestedAt)
}

func TestDuckDBTimeNormalization(t *testing.T) {
	value := time.Date(2026, time.July, 30, 12, 0, 0, 123456789, time.FixedZone("test", 2*60*60))

	ret := normalizeMetricTime(value)
	assert.Equal(t, time.UTC, ret.Location())
	assert.Equal(t, 123456789, ret.Nanosecond())
	assert.Equal(t, value.UnixNano(), ret.UnixNano())
}
