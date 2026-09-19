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
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils"
)

// groupByDimension describes how a datapoint response is split into series.
type groupByDimension struct {
	// expr is the SQL expression whose value identifies a series.
	expr string
	// kind is the resource kind of the expression's value whenever the value
	// is a resource UID. It is used to resolve the series display names.
	kind string
}

func getAccessLogGroupBy(groupBy visibilityv1.GetAccessLogDataPointRequest_GroupBy) *groupByDimension {
	switch groupBy {
	case visibilityv1.GetAccessLogDataPointRequest_STATUS:
		return &groupByDimension{expr: colStatus}
	case visibilityv1.GetAccessLogDataPointRequest_MODE:
		return &groupByDimension{expr: jsonMode}
	case visibilityv1.GetAccessLogDataPointRequest_REASON:
		return &groupByDimension{expr: jsonDenyReason}
	case visibilityv1.GetAccessLogDataPointRequest_SERVICE:
		return &groupByDimension{expr: colServiceUID, kind: "Service"}
	case visibilityv1.GetAccessLogDataPointRequest_USER:
		return &groupByDimension{expr: colUserUID, kind: "User"}
	case visibilityv1.GetAccessLogDataPointRequest_SESSION:
		return &groupByDimension{expr: colSessionUID, kind: "Session"}
	case visibilityv1.GetAccessLogDataPointRequest_DEVICE:
		return &groupByDimension{expr: colDeviceUID, kind: "Device"}
	case visibilityv1.GetAccessLogDataPointRequest_NAMESPACE:
		return &groupByDimension{expr: colNamespaceUID, kind: "Namespace"}
	case visibilityv1.GetAccessLogDataPointRequest_REGION:
		return &groupByDimension{expr: colRegionUID, kind: "Region"}
	case visibilityv1.GetAccessLogDataPointRequest_POLICY:
		return &groupByDimension{expr: colPolicyUID, kind: "Policy"}
	default:
		return nil
	}
}

func getAuthenticationLogGroupBy(
	groupBy visibilityv1.GetAuthenticationLogDataPointRequest_GroupBy) *groupByDimension {
	switch groupBy {
	case visibilityv1.GetAuthenticationLogDataPointRequest_TYPE:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.authentication.info.type')`}
	case visibilityv1.GetAuthenticationLogDataPointRequest_ASSURANCE_LEVEL:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.authentication.info.aal')`}
	case visibilityv1.GetAuthenticationLogDataPointRequest_USER:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.userRef.uid')`, kind: "User"}
	case visibilityv1.GetAuthenticationLogDataPointRequest_SESSION:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.sessionRef.uid')`, kind: "Session"}
	case visibilityv1.GetAuthenticationLogDataPointRequest_DEVICE:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.deviceRef.uid')`, kind: "Device"}
	case visibilityv1.GetAuthenticationLogDataPointRequest_IDENTITY_PROVIDER:
		return &groupByDimension{
			expr: `json_extract_string(rsc, '$.entry.authentication.info.identityProvider.identityProviderRef.uid')`,
			kind: "IdentityProvider",
		}
	case visibilityv1.GetAuthenticationLogDataPointRequest_CREDENTIAL:
		return &groupByDimension{
			expr: `json_extract_string(rsc, '$.entry.authentication.info.credential.credentialRef.uid')`,
			kind: "Credential",
		}
	case visibilityv1.GetAuthenticationLogDataPointRequest_AUTHENTICATOR:
		return &groupByDimension{
			expr: `json_extract_string(rsc, '$.entry.authentication.info.authenticator.authenticatorRef.uid')`,
			kind: "Authenticator",
		}
	default:
		return nil
	}
}

func getAuditLogGroupBy(groupBy visibilityv1.GetAuditLogDataPointRequest_GroupBy) *groupByDimension {
	switch groupBy {
	case visibilityv1.GetAuditLogDataPointRequest_ACTION:
		return &groupByDimension{expr: sqlAuditAction}
	case visibilityv1.GetAuditLogDataPointRequest_RESOURCE_KIND:
		return &groupByDimension{expr: jsonAuditResourceKind}
	case visibilityv1.GetAuditLogDataPointRequest_USER:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.userRef.uid')`, kind: "User"}
	case visibilityv1.GetAuditLogDataPointRequest_SESSION:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.sessionRef.uid')`, kind: "Session"}
	case visibilityv1.GetAuditLogDataPointRequest_DEVICE:
		return &groupByDimension{expr: `json_extract_string(rsc, '$.entry.deviceRef.uid')`, kind: "Device"}
	default:
		return nil
	}
}

func getComponentLogGroupBy(groupBy visibilityv1.GetComponentLogDataPointRequest_GroupBy) *groupByDimension {
	switch groupBy {
	case visibilityv1.GetComponentLogDataPointRequest_LEVEL:
		return &groupByDimension{expr: colLevel}
	case visibilityv1.GetComponentLogDataPointRequest_COMPONENT_TYPE:
		return &groupByDimension{expr: jsonComponentType}
	case visibilityv1.GetComponentLogDataPointRequest_COMPONENT_NAMESPACE:
		return &groupByDimension{expr: jsonComponentNamespace}
	default:
		return nil
	}
}

func (s *Server) resolveDisplayName(ctx context.Context, kind, uid string) string {
	if kind == "" || uid == "" {
		return ""
	}

	opts := &rmetav1.GetOptions{Uid: uid}
	coreC := s.octeliumC.CoreC()

	switch kind {
	case "User":
		if itm, err := coreC.GetUser(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Service":
		if itm, err := coreC.GetService(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Session":
		if itm, err := coreC.GetSession(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Device":
		if itm, err := coreC.GetDevice(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Namespace":
		if itm, err := coreC.GetNamespace(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Region":
		if itm, err := coreC.GetRegion(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Policy":
		if itm, err := coreC.GetPolicy(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "IdentityProvider":
		if itm, err := coreC.GetIdentityProvider(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Credential":
		if itm, err := coreC.GetCredential(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	case "Authenticator":
		if itm, err := coreC.GetAuthenticator(ctx, opts); err == nil {
			return itm.Metadata.Name
		}
	}

	return ""
}

func (s *Server) getTopSeriesKeys(ctx context.Context,
	table string, dimension *groupByDimension, filters []exp.Expression, limit int) ([]string, error) {

	ds := goqu.Dialect("postgres").From(table).
		Select(
			goqu.L(dimension.expr).As("key"),
			goqu.L(`COUNT(*)`).As("count"),
		).
		Where(filters...).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, dimension.expr))).
		GroupBy(goqu.L("key")).
		Order(goqu.L("count").Desc()).
		Limit(uint(limit))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []string
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		if key == "" {
			continue
		}

		ret = append(ret, key)
	}

	return ret, rows.Err()
}

func (s *Server) getDataPointSeries(ctx context.Context, table string,
	dimension *groupByDimension, fromTime, toTime time.Time,
	interval *intervalDataPoint, filters []exp.Expression,
	limit int) ([]*visibilityv1.DataPointSeries, error) {

	if fromTime.After(toTime) {
		return nil, fmt.Errorf("from timestamp must be before to timestamp")
	}

	timeFilters := []exp.Expression{
		goqu.L(fmt.Sprintf("%s >= ?", colCreatedAt), fromTime),
		goqu.L(fmt.Sprintf("%s < ?", colCreatedAt), toTime),
	}

	keys, err := s.getTopSeriesKeys(ctx, table, dimension,
		append(append([]exp.Expression{}, filters...), timeFilters...), limit)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}

	intervalStr := fmt.Sprintf("%d %s", interval.Value, interval.Unit)

	ds := goqu.Dialect("postgres").From(table).
		Select(
			goqu.L(dimension.expr).As("key"),
			goqu.L(fmt.Sprintf("time_bucket(INTERVAL '%s', %s)", intervalStr, colCreatedAt)).As("timestamp"),
			goqu.L(`COUNT(*)`).As("count"),
		).
		Where(filters...).
		Where(timeFilters...).
		Where(goqu.L(dimension.expr).In(keys)).
		GroupBy(goqu.L("key"), goqu.L("timestamp")).
		Order(goqu.L("key").Asc(), goqu.L("timestamp").Asc())

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byKey := make(map[string][]DataPoint)
	for rows.Next() {
		var key string
		var ts time.Time
		var count int64

		if err := rows.Scan(&key, &ts, &count); err != nil {
			return nil, err
		}

		byKey[key] = append(byKey[key], DataPoint{
			Timestamp: ts.Format(time.RFC3339),
			Count:     count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var ret []*visibilityv1.DataPointSeries

	for _, key := range keys {
		series := &visibilityv1.DataPointSeries{
			Key:         key,
			DisplayName: s.resolveDisplayName(ctx, dimension.kind, key),
		}

		for _, dp := range fillGaps(byKey[key], fromTime, toTime, interval) {
			series.Total = series.Total + dp.Count
			series.Datapoints = append(series.Datapoints, &visibilityv1.DataPointSeries_DataPoint{
				Timestamp: pbutils.Timestamp(utils.MustParseTime(dp.Timestamp)),
				Count:     dp.Count,
			})
		}

		ret = append(ret, series)
	}

	return ret, nil
}
