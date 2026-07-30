// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	attributeSourceResource  = 1 << 0
	attributeSourceScope     = 1 << 1
	attributeSourceDataPoint = 1 << 2
)

type descriptorRecord struct {
	id                  string
	name                string
	kind                vmetricsv1.MetricDescriptor_Kind
	numberValueType     vmetricsv1.MetricDescriptor_NumberValueType
	unit                string
	description         string
	temporality         vmetricsv1.MetricDescriptor_Temporality
	scopeName           string
	scopeVersion        string
	scopeSchemaURL      string
	explicitBounds      []float64
	expMinimumScale     *int32
	expMaximumScale     *int32
	expMinimumThreshold *float64
	expMaximumThreshold *float64
}

type storedAttribute struct {
	Key         string   `json:"key"`
	Kind        string   `json:"kind"`
	StringValue string   `json:"stringValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	IntValue    *int64   `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	SourceMask  int      `json:"sourceMask"`
}

type seriesRecord struct {
	id                 string
	descriptorID       string
	labels             []storedAttribute
	labelsJSON         string
	labelsKey          string
	componentType      string
	componentNamespace string
	componentName      string
}

type attributeKeyRecord struct {
	descriptorID string
	key          string
	valueKind    string
	sourceMask   int
}

type numberPointRecord struct {
	pointID        string
	timestamp      time.Time
	ingestedAt     time.Time
	startTimestamp *time.Time
	seriesID       string
	intValue       *int64
	doubleValue    *float64
}

type histogramPointRecord struct {
	pointID        string
	timestamp      time.Time
	ingestedAt     time.Time
	startTimestamp *time.Time
	seriesID       string
	count          uint64
	hasSum         bool
	sum            *float64
	min            *float64
	max            *float64
	bucketCounts   []uint64
}

type exponentialHistogramPointRecord struct {
	pointID        string
	timestamp      time.Time
	ingestedAt     time.Time
	startTimestamp *time.Time
	seriesID       string
	count          uint64
	hasSum         bool
	sum            *float64
	min            *float64
	max            *float64
	scale          int32
	zeroCount      uint64
	zeroThreshold  float64
	positiveOffset int32
	positiveCounts []uint64
	negativeOffset int32
	negativeCounts []uint64
}

type metricWriteBatch struct {
	descriptors map[string]*descriptorRecord
	series      map[string]*seriesRecord
	attributes  map[string]*attributeKeyRecord

	numberPoints      []numberPointRecord
	histogramPoints   []histogramPointRecord
	exponentialPoints []exponentialHistogramPointRecord
}

func newMetricWriteBatch() *metricWriteBatch {
	return &metricWriteBatch{
		descriptors: map[string]*descriptorRecord{},
		series:      map[string]*seriesRecord{},
		attributes:  map[string]*attributeKeyRecord{},
	}
}

func (b *metricWriteBatch) merge(other *metricWriteBatch) error {
	for key, val := range other.attributes {
		if current := b.attributes[key]; current != nil && current.valueKind != val.valueKind {
			return fmt.Errorf("metric attribute %s changed value kind from %s to %s", val.key,
				current.valueKind, val.valueKind)
		}
	}

	for key, val := range other.descriptors {
		if current := b.descriptors[key]; current != nil {
			mergeDescriptorRecord(current, val)
		} else {
			b.descriptors[key] = val
		}
	}
	for key, val := range other.series {
		b.series[key] = val
	}
	for key, val := range other.attributes {
		if current := b.attributes[key]; current != nil {
			current.sourceMask |= val.sourceMask
		} else {
			b.attributes[key] = val
		}
	}

	b.numberPoints = append(b.numberPoints, other.numberPoints...)
	b.histogramPoints = append(b.histogramPoints, other.histogramPoints...)
	b.exponentialPoints = append(b.exponentialPoints, other.exponentialPoints...)
	return nil
}

func (b *metricWriteBatch) normalize() {
	b.numberPoints = deduplicateNumberPoints(b.numberPoints)
	b.histogramPoints = deduplicateHistogramPoints(b.histogramPoints)
	b.exponentialPoints = deduplicateExponentialHistogramPoints(b.exponentialPoints)
}

func deduplicateNumberPoints(points []numberPointRecord) []numberPointRecord {
	byID := make(map[string]numberPointRecord, len(points))
	for _, point := range points {
		if current, ok := byID[point.pointID]; !ok || point.ingestedAt.Before(current.ingestedAt) {
			byID[point.pointID] = point
		}
	}

	ret := make([]numberPointRecord, 0, len(byID))
	for _, point := range byID {
		ret = append(ret, point)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].seriesID != ret[j].seriesID {
			return ret[i].seriesID < ret[j].seriesID
		}
		if !ret[i].timestamp.Equal(ret[j].timestamp) {
			return ret[i].timestamp.Before(ret[j].timestamp)
		}
		return ret[i].pointID < ret[j].pointID
	})
	return ret
}

func deduplicateHistogramPoints(points []histogramPointRecord) []histogramPointRecord {
	byID := make(map[string]histogramPointRecord, len(points))
	for _, point := range points {
		if current, ok := byID[point.pointID]; !ok || point.ingestedAt.Before(current.ingestedAt) {
			byID[point.pointID] = point
		}
	}

	ret := make([]histogramPointRecord, 0, len(byID))
	for _, point := range byID {
		ret = append(ret, point)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].seriesID != ret[j].seriesID {
			return ret[i].seriesID < ret[j].seriesID
		}
		if !ret[i].timestamp.Equal(ret[j].timestamp) {
			return ret[i].timestamp.Before(ret[j].timestamp)
		}
		return ret[i].pointID < ret[j].pointID
	})
	return ret
}

func deduplicateExponentialHistogramPoints(points []exponentialHistogramPointRecord) []exponentialHistogramPointRecord {
	byID := make(map[string]exponentialHistogramPointRecord, len(points))
	for _, point := range points {
		if current, ok := byID[point.pointID]; !ok || point.ingestedAt.Before(current.ingestedAt) {
			byID[point.pointID] = point
		}
	}

	ret := make([]exponentialHistogramPointRecord, 0, len(byID))
	for _, point := range byID {
		ret = append(ret, point)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].seriesID != ret[j].seriesID {
			return ret[i].seriesID < ret[j].seriesID
		}
		if !ret[i].timestamp.Equal(ret[j].timestamp) {
			return ret[i].timestamp.Before(ret[j].timestamp)
		}
		return ret[i].pointID < ret[j].pointID
	})
	return ret
}

func mergeDescriptorRecord(current, incoming *descriptorRecord) {
	if incoming.description != "" {
		current.description = incoming.description
	}
	mergeInt32Range(&current.expMinimumScale, &current.expMaximumScale,
		incoming.expMinimumScale, incoming.expMaximumScale)
	mergeFloat64Range(&current.expMinimumThreshold, &current.expMaximumThreshold,
		incoming.expMinimumThreshold, incoming.expMaximumThreshold)
}

func mergeInt32Range(currentMin, currentMax **int32, incomingMin, incomingMax *int32) {
	if incomingMin != nil && (*currentMin == nil || *incomingMin < **currentMin) {
		value := *incomingMin
		*currentMin = &value
	}
	if incomingMax != nil && (*currentMax == nil || *incomingMax > **currentMax) {
		value := *incomingMax
		*currentMax = &value
	}
}

func mergeFloat64Range(currentMin, currentMax **float64, incomingMin, incomingMax *float64) {
	if incomingMin != nil && (*currentMin == nil || *incomingMin < **currentMin) {
		value := *incomingMin
		*currentMin = &value
	}
	if incomingMax != nil && (*currentMax == nil || *incomingMax > **currentMax) {
		value := *incomingMax
		*currentMax = &value
	}
}

func descriptorID(record *descriptorRecord) string {
	h := sha256.New()
	writeHashString(h, record.name)
	writeHashUint32(h, uint32(record.kind))
	writeHashUint32(h, uint32(record.numberValueType))
	writeHashString(h, record.unit)
	writeHashUint32(h, uint32(record.temporality))
	writeHashString(h, record.scopeName)
	writeHashString(h, record.scopeVersion)
	writeHashString(h, record.scopeSchemaURL)
	writeHashUint32(h, uint32(len(record.explicitBounds)))
	for _, bound := range record.explicitBounds {
		writeHashUint64(h, math.Float64bits(bound))
	}
	return "v1:sha256:" + hex.EncodeToString(h.Sum(nil))
}

func seriesID(descriptorID, labelsKey string) string {
	h := sha256.New()
	writeHashString(h, descriptorID)
	writeHashString(h, labelsKey)
	return "v1:sha256:" + hex.EncodeToString(h.Sum(nil))
}

func outputSeriesID(descriptorID, labelsKey string, quantile *float64) string {
	h := sha256.New()
	writeHashString(h, descriptorID)
	writeHashString(h, labelsKey)
	if quantile == nil {
		writeHashInt64(h, 0)
	} else {
		writeHashInt64(h, 1)
		writeHashUint64(h, math.Float64bits(*quantile))
	}
	return "v1:sha256:" + hex.EncodeToString(h.Sum(nil))
}

func pointID(kind, seriesID string, timestamp time.Time, startTimestamp *time.Time) string {
	h := sha256.New()
	writeHashString(h, kind)
	writeHashString(h, seriesID)
	writeHashInt64(h, timestamp.UTC().UnixNano())
	if startTimestamp == nil {
		writeHashInt64(h, 0)
	} else {
		writeHashInt64(h, 1)
		writeHashInt64(h, startTimestamp.UTC().UnixNano())
	}
	return "v1:sha256:" + hex.EncodeToString(h.Sum(nil))
}

func writeHashString(h interface{ Write([]byte) (int, error) }, val string) {
	writeHashUint64(h, uint64(len(val)))
	_, _ = h.Write([]byte(val))
}

func writeHashUint32(h interface{ Write([]byte) (int, error) }, val uint32) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, val)
	_, _ = h.Write(buf)
}

func writeHashInt64(h interface{ Write([]byte) (int, error) }, val int64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(val))
	_, _ = h.Write(buf)
}

func writeHashUint64(h interface{ Write([]byte) (int, error) }, val uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	_, _ = h.Write(buf)
}

func attributesFromMaps(resource, scope, point pcommon.Map) ([]storedAttribute, error) {
	values := map[string]storedAttribute{}

	if err := addAttributes(values, resource, attributeSourceResource); err != nil {
		return nil, err
	}
	if err := addAttributes(values, scope, attributeSourceScope); err != nil {
		return nil, err
	}
	if err := addAttributes(values, point, attributeSourceDataPoint); err != nil {
		return nil, err
	}

	if len(values) > maxAttributesPerPoint {
		return nil, fmt.Errorf("metric series has too many effective attributes")
	}

	ret := make([]storedAttribute, 0, len(values))
	for _, val := range values {
		ret = append(ret, val)
	}
	sortStoredAttributes(ret)

	return ret, nil
}

func addAttributes(ret map[string]storedAttribute, attrs pcommon.Map, sourceMask int) error {
	var retErr error
	attrs.Range(func(key string, value pcommon.Value) bool {
		item, err := storedAttributeFromValue(key, value, sourceMask)
		if err != nil {
			retErr = err
			return false
		}

		if existing, ok := ret[key]; ok {
			item.SourceMask |= existing.SourceMask
		}
		ret[key] = item
		return true
	})

	return retErr
}

func storedAttributeFromValue(key string, value pcommon.Value, sourceMask int) (storedAttribute, error) {
	ret := storedAttribute{
		Key:        key,
		SourceMask: sourceMask,
	}

	switch value.Type() {
	case pcommon.ValueTypeStr:
		ret.Kind = "STRING"
		ret.StringValue = value.Str()

	case pcommon.ValueTypeBool:
		ret.Kind = "BOOL"
		val := value.Bool()
		ret.BoolValue = &val

	case pcommon.ValueTypeInt:
		ret.Kind = "INT64"
		val := value.Int()
		ret.IntValue = &val

	case pcommon.ValueTypeDouble:
		val := value.Double()
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return storedAttribute{}, fmt.Errorf("metric attribute %s contains a non-finite double", key)
		}
		ret.Kind = "DOUBLE"
		ret.DoubleValue = &val

	default:
		return storedAttribute{}, fmt.Errorf("unsupported metric attribute value type for key %s: %s", key, value.Type().String())
	}

	return ret, nil
}

func sortStoredAttributes(attrs []storedAttribute) {
	sort.Slice(attrs, func(i, j int) bool {
		if attrs[i].Key != attrs[j].Key {
			return attrs[i].Key < attrs[j].Key
		}
		if attrs[i].Kind != attrs[j].Kind {
			return attrs[i].Kind < attrs[j].Kind
		}
		return attrs[i].canonicalValue() < attrs[j].canonicalValue()
	})
}

func storedAttributesJSON(attrs []storedAttribute) (string, error) {
	data, err := json.Marshal(attrs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeStoredAttributes(raw string) ([]storedAttribute, error) {
	var ret []storedAttribute
	if strings.TrimSpace(raw) == "" {
		return ret, nil
	}
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return nil, err
	}
	sortStoredAttributes(ret)
	return ret, nil
}

func storedAttributesKey(attrs []storedAttribute) string {
	var buf bytes.Buffer
	for _, attr := range attrs {
		writeLengthPrefixedString(&buf, attr.Key)
		writeLengthPrefixedString(&buf, attr.Kind)
		writeLengthPrefixedString(&buf, attr.canonicalValue())
	}
	return buf.String()
}

func writeLengthPrefixedString(buf *bytes.Buffer, val string) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(val)))
	_, _ = buf.WriteString(val)
}

func (a storedAttribute) canonicalValue() string {
	switch a.Kind {
	case "STRING":
		return a.StringValue
	case "BOOL":
		if a.BoolValue == nil {
			return "false"
		}
		return strconv.FormatBool(*a.BoolValue)
	case "INT64":
		if a.IntValue == nil {
			return "0"
		}
		return strconv.FormatInt(*a.IntValue, 10)
	case "DOUBLE":
		if a.DoubleValue == nil {
			return "0"
		}
		return strconv.FormatUint(math.Float64bits(*a.DoubleValue), 16)
	default:
		return ""
	}
}

func (a storedAttribute) valueKey() string {
	return a.Kind + ":" + a.canonicalValue()
}

func pickStoredAttributes(attrs []storedAttribute, keys []string) []storedAttribute {
	if len(keys) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		allowed[key] = struct{}{}
	}

	ret := make([]storedAttribute, 0, len(keys))
	for _, attr := range attrs {
		if _, ok := allowed[attr.Key]; ok {
			ret = append(ret, attr)
		}
	}
	sortStoredAttributes(ret)
	return ret
}

func findStoredAttribute(attrs []storedAttribute, key string) *storedAttribute {
	for i := range attrs {
		if attrs[i].Key == key {
			return &attrs[i]
		}
	}
	return nil
}

func componentValues(attrs []storedAttribute) (string, string, string) {
	getString := func(key string) string {
		attr := findStoredAttribute(attrs, key)
		if attr == nil || attr.Kind != "STRING" {
			return ""
		}
		return attr.StringValue
	}

	return getString("octelium.component.type"),
		getString("octelium.component.namespace"),
		getString("octelium.component.name")
}

func attributeRecordKey(descriptorID, key string) string {
	return descriptorID + "\x00" + key
}

func marshalUint64Slice(values []uint64) string {
	data, _ := json.Marshal(values)
	return string(data)
}

func marshalFloat64Slice(values []float64) string {
	data, _ := json.Marshal(values)
	return string(data)
}
