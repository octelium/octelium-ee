// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils"
)

func (s *Server) getDataPointInterval(iv *metav1.Duration) *intervalDataPoint {
	if iv == nil || iv.Type == nil {
		return &intervalDataPoint{
			Unit:  "minute",
			Value: 1,
		}
	}
	switch iv.Type.(type) {
	case *metav1.Duration_Seconds:
		return &intervalDataPoint{
			Unit:  "second",
			Value: int(iv.GetSeconds()),
		}
	case *metav1.Duration_Minutes:
		return &intervalDataPoint{
			Unit:  "minute",
			Value: int(iv.GetMinutes()),
		}
	case *metav1.Duration_Hours:
		return &intervalDataPoint{
			Unit:  "hour",
			Value: int(iv.GetHours()),
		}
	case *metav1.Duration_Days:
		return &intervalDataPoint{
			Unit:  "day",
			Value: int(iv.GetDays()),
		}
	case *metav1.Duration_Weeks:
		return &intervalDataPoint{
			Unit:  "day",
			Value: int(7 * iv.GetDays()),
		}
	case *metav1.Duration_Months:
		return &intervalDataPoint{
			Unit:  "day",
			Value: int(30 * iv.GetDays()),
		}
	}

	return &intervalDataPoint{
		Unit:  "minute",
		Value: 1,
	}
}

func (s *Server) getAccessLogDataPoint(ctx context.Context, req *visibilityv1.GetAccessLogDataPointRequest) (*visibilityv1.GetAccessLogDataPointResponse, error) {

	var from, to time.Time

	if req.From == nil && req.To == nil {
		from = time.Now().Add(-1 * time.Hour)
		to = from.Add(1 * time.Hour)
	}

	if req.From != nil {
		from = req.From.AsTime()
		if req.To == nil {
			to = time.Now()
		}
	}
	if req.To != nil {
		to = req.To.AsTime()
		if req.From == nil {
			from = to.Add(-1 * time.Hour)
		}
	}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	dps, err := s.getDataPoints(ctx, "access_logs", from, to, s.getDataPointInterval(req.Interval), filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret := &visibilityv1.GetAccessLogDataPointResponse{}

	for _, dp := range dps {
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAccessLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(utils.MustParseTime(dp.Timestamp)),
			Count:     dp.Count,
		})
	}

	return ret, nil
}

func (s *Server) getAuthenticationLogDataPoint(ctx context.Context, req *visibilityv1.GetAuthenticationLogDataPointRequest) (*visibilityv1.GetAuthenticationLogDataPointResponse, error) {

	var from, to time.Time

	if req.From == nil && req.To == nil {
		from = time.Now().Add(-1 * time.Hour)
		to = from.Add(1 * time.Hour)
	}

	if req.From != nil {
		from = req.From.AsTime()
		if req.To == nil {
			to = time.Now()
		}
	}
	if req.To != nil {
		to = req.To.AsTime()
		if req.From == nil {
			from = to.Add(-1 * time.Hour)
		}
	}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, "entry.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, "entry.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, "entry.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.IdentityProviderRef, nil, "entry.authentication.info.identityProvider.identityProviderRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.AuthenticatorRef, nil, "entry.authentication.info.authenticator.authenticatorRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.CredentialRef, nil, "entry.authentication.info.credential.credentialRef")
	if err != nil {
		return nil, err
	}

	dps, err := s.getDataPoints(ctx, "authentication_logs", from, to, s.getDataPointInterval(req.Interval), filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret := &visibilityv1.GetAuthenticationLogDataPointResponse{}

	for _, dp := range dps {
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAuthenticationLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(utils.MustParseTime(dp.Timestamp)),
			Count:     dp.Count,
		})
	}

	return ret, nil
}

func (s *Server) getAuditLogDataPoint(ctx context.Context, req *visibilityv1.GetAuditLogDataPointRequest) (*visibilityv1.GetAuditLogDataPointResponse, error) {

	var from, to time.Time

	if req.From == nil && req.To == nil {
		from = time.Now().Add(-1 * time.Hour)
		to = from.Add(1 * time.Hour)
	}

	if req.From != nil {
		from = req.From.AsTime()
		if req.To == nil {
			to = time.Now()
		}
	}
	if req.To != nil {
		to = req.To.AsTime()
		if req.From == nil {
			from = to.Add(-1 * time.Hour)
		}
	}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, "entry.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, "entry.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, "entry.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.ResourceRef, nil, "entry.resourceRef")
	if err != nil {
		return nil, err
	}

	dps, err := s.getDataPoints(ctx, "audit_logs", from, to, s.getDataPointInterval(req.Interval), filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret := &visibilityv1.GetAuditLogDataPointResponse{}

	for _, dp := range dps {
		ret.Datapoints = append(ret.Datapoints, &visibilityv1.GetAuditLogDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(utils.MustParseTime(dp.Timestamp)),
			Count:     dp.Count,
		})
	}

	return ret, nil
}

type DataPoint struct {
	Timestamp string `db:"timestamp" goqu:"skipinsert,skipupdate"`
	Count     int64  `db:"count" goqu:"skipinsert,skipupdate"`
}

type intervalDataPoint struct {
	Value int
	Unit  string
}

func (s *Server) getDataPoints(ctx context.Context,
	tableName string, fromTime, toTime time.Time, interval *intervalDataPoint, filters []exp.Expression) ([]DataPoint, error) {

	if fromTime.After(toTime) {
		return nil, fmt.Errorf("from timestamp must be before to timestamp")
	}

	intervalStr := fmt.Sprintf("%d %s", interval.Value, interval.Unit)

	dialect := goqu.Dialect("postgres")

	query := dialect.From(tableName).
		Select(
			goqu.L(fmt.Sprintf("time_bucket(INTERVAL '%s', CAST(json_extract(rsc, '$.metadata.createdAt') AS TIMESTAMP))", intervalStr)).As("timestamp"),
			goqu.COUNT("*").As("count"),
		).
		Where(filters...).
		Where(
			goqu.L("CAST(json_extract(rsc, '$.metadata.createdAt') AS TIMESTAMP) >= ?", fromTime),
			goqu.L("CAST(json_extract(rsc, '$.metadata.createdAt') AS TIMESTAMP) < ?", toTime),
		).
		GroupBy(goqu.L("timestamp")).
		Order(goqu.L("timestamp").Asc())

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var dataPoints []DataPoint
	for rows.Next() {
		var dp DataPoint
		var ts time.Time

		if err := rows.Scan(&ts, &dp.Count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		dp.Timestamp = ts.Format(time.RFC3339)
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	dataPoints = fillGaps(dataPoints, fromTime, toTime, interval)

	return dataPoints, nil
}

func fillGaps(dataPoints []DataPoint, from, to time.Time, interval *intervalDataPoint) []DataPoint {
	if len(dataPoints) == 0 {
		return generateEmptyDataPoints(from, to, interval)
	}

	pointMap := make(map[string]int64)
	for _, dp := range dataPoints {
		pointMap[dp.Timestamp] = dp.Count
	}

	var result []DataPoint
	current := alignTimeToBucket(from, interval)

	for current.Before(to) || current.Equal(to) {
		timestamp := current.Format(time.RFC3339)
		count := pointMap[timestamp]

		result = append(result, DataPoint{
			Timestamp: timestamp,
			Count:     count,
		})

		current = addInterval(current, interval)

		if current.After(to.Add(24 * time.Hour)) {
			break
		}
	}

	return result
}

func generateEmptyDataPoints(from, to time.Time, interval *intervalDataPoint) []DataPoint {
	var result []DataPoint
	current := alignTimeToBucket(from, interval)

	for current.Before(to) {
		result = append(result, DataPoint{
			Timestamp: current.Format(time.RFC3339),
			Count:     0,
		})
		current = addInterval(current, interval)
	}

	return result
}

func alignTimeToBucket(t time.Time, interval *intervalDataPoint) time.Time {
	switch interval.Unit {
	case "second":
		return t.Truncate(time.Duration(interval.Value) * time.Second)
	case "minute":
		return t.Truncate(time.Duration(interval.Value) * time.Minute)
	case "hour":
		return t.Truncate(time.Duration(interval.Value) * time.Hour)
	case "day":
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

func addInterval(t time.Time, interval *intervalDataPoint) time.Time {
	switch interval.Unit {
	case "second":
		return t.Add(time.Duration(interval.Value) * time.Second)
	case "minute":
		return t.Add(time.Duration(interval.Value) * time.Minute)
	case "hour":
		return t.Add(time.Duration(interval.Value) * time.Hour)
	case "day":
		return t.AddDate(0, 0, interval.Value)
	default:
		return t
	}
}
