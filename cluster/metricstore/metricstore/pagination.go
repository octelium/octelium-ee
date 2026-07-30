// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const seriesPageTokenTTL = 5 * time.Minute

type pageTokenPayload struct {
	Version     uint32 `json:"version"`
	Kind        string `json:"kind"`
	Fingerprint string `json:"fingerprint"`
	Cursor      string `json:"cursor"`
	SnapshotNS  int64  `json:"snapshotNS"`
	ExpiresNS   int64  `json:"expiresNS"`
}

func newPageTokenPayload(kind, fingerprint, cursor string, snapshot time.Time) *pageTokenPayload {
	snapshot = normalizeMetricTime(snapshot)

	return &pageTokenPayload{
		Kind:        kind,
		Fingerprint: fingerprint,
		Cursor:      cursor,
		SnapshotNS:  snapshot.UnixNano(),
		ExpiresNS:   snapshot.Add(seriesPageTokenTTL).UnixNano(),
	}
}

func (s *Server) encodePageToken(payload *pageTokenPayload) (string, error) {
	if payload == nil {
		return "", status.Error(codes.Internal, "nil page token payload")
	}

	payload.Version = 1
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, s.pageTokenKey)
	_, _ = mac.Write(data)
	signature := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *Server) decodePageToken(raw, kind, fingerprint string) (*pageTokenPayload, error) {
	parts := splitPageToken(raw)
	if len(parts) != 2 {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}

	mac := hmac.New(sha256.New, s.pageTokenKey)
	_, _ = mac.Write(data)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, status.Error(codes.InvalidArgument, "invalid page token signature")
	}

	payload := &pageTokenPayload{}
	if err := json.Unmarshal(data, payload); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
	if payload.Version != 1 || payload.Kind != kind || payload.Fingerprint != fingerprint {
		return nil, status.Error(codes.InvalidArgument, "page token does not match request")
	}
	if payload.SnapshotNS <= 0 || payload.ExpiresNS <= payload.SnapshotNS ||
		payload.ExpiresNS-payload.SnapshotNS != seriesPageTokenTTL.Nanoseconds() {
		return nil, status.Error(codes.InvalidArgument, "invalid page token lifetime")
	}
	now := time.Now().UTC().UnixNano()
	if payload.SnapshotNS > now+maximumFutureSkew.Nanoseconds() {
		return nil, status.Error(codes.InvalidArgument, "invalid page token snapshot")
	}
	if now > payload.ExpiresNS {
		return nil, status.Error(codes.FailedPrecondition, "page token has expired")
	}

	return payload, nil
}

func splitPageToken(raw string) []string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '.' {
			return []string{raw[:i], raw[i+1:]}
		}
	}
	return nil
}

func queryRequestFingerprint(req *vmetricsv1.QueryMetricsRequest) (string, error) {
	cloned := proto.Clone(req).(*vmetricsv1.QueryMetricsRequest)
	cloned.SeriesPageToken = ""
	cloned.SeriesPageSize = 0

	if cloned.Metric != nil {
		switch selector := cloned.Metric.Selector.(type) {
		case *vmetricsv1.MetricSelector_Name:
			selector.Name = strings.TrimSpace(selector.Name)
		case *vmetricsv1.MetricSelector_DescriptorID:
			selector.DescriptorID = strings.TrimSpace(selector.DescriptorID)
		}
	}
	normalizeComponentForFingerprint(cloned.Component)
	for i := range cloned.GroupBy {
		cloned.GroupBy[i] = strings.TrimSpace(cloned.GroupBy[i])
	}
	sort.Strings(cloned.GroupBy)

	for _, filter := range cloned.Filters {
		if filter == nil {
			continue
		}
		filter.Key = strings.TrimSpace(filter.Key)
		sort.Slice(filter.Values, func(i, j int) bool {
			return protoAttributeValueKey(filter.Values[i]) < protoAttributeValueKey(filter.Values[j])
		})
	}
	sort.Slice(cloned.Filters, func(i, j int) bool {
		if cloned.Filters[i] == nil {
			return cloned.Filters[j] != nil
		}
		if cloned.Filters[j] == nil {
			return false
		}
		return cloned.Filters[i].Key < cloned.Filters[j].Key
	})

	if histogram := cloned.Operation.GetHistogram(); histogram != nil {
		sort.Float64s(histogram.Quantiles)
	}

	return deterministicProtoFingerprint(cloned)
}

func descriptorRequestFingerprint(req *vmetricsv1.ListMetricDescriptorsRequest) (string, error) {
	cloned := proto.Clone(req).(*vmetricsv1.ListMetricDescriptorsRequest)
	cloned.PageToken = ""
	cloned.Limit = 0
	cloned.NamePrefix = strings.TrimSpace(cloned.NamePrefix)
	normalizeComponentForFingerprint(cloned.Component)
	sort.Slice(cloned.Kinds, func(i, j int) bool {
		return cloned.Kinds[i] < cloned.Kinds[j]
	})
	return deterministicProtoFingerprint(cloned)
}

func normalizeComponentForFingerprint(component *vmetricsv1.ComponentSelector) {
	if component == nil {
		return
	}
	component.Type = strings.TrimSpace(component.Type)
	component.Namespace = strings.TrimSpace(component.Namespace)
	component.Name = strings.TrimSpace(component.Name)
}

func deterministicProtoFingerprint(message proto.Message) (string, error) {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
