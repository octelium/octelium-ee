// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	accessDataPointMaxBuckets    = 1000
	accessDataPointAutoBuckets   = 120
	accessDataPointDefaultSeries = 8
	accessDataPointMaxSeries     = 25
	accessDataPointDefaultWindow = 24 * time.Hour

	accessTopDefaultLimit = 10
	accessTopMaxLimit     = 100
)

var accessDataPointAutoIntervals = []time.Duration{
	time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	time.Hour,
	3 * time.Hour,
	6 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
	7 * 24 * time.Hour,
}

func getAccessWindow(from, to *timestamppb.Timestamp) (time.Time, time.Time, error) {
	if from != nil {
		if err := from.CheckValid(); err != nil {
			return time.Time{}, time.Time{}, grpcutils.InvalidArg("Invalid from timestamp: %s", err.Error())
		}
	}

	if to != nil {
		if err := to.CheckValid(); err != nil {
			return time.Time{}, time.Time{}, grpcutils.InvalidArg("Invalid to timestamp: %s", err.Error())
		}
	}

	end := time.Now().UTC()
	if to != nil {
		end = to.AsTime().UTC()
	}

	start := end.Add(-accessDataPointDefaultWindow)
	if from != nil {
		start = from.AsTime().UTC()
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, grpcutils.InvalidArg("The from timestamp must not be after the to timestamp")
	}

	return start, end, nil
}

func getAccessDataPointInterval(from, to time.Time, interval *metav1.Duration) (time.Duration, error) {
	if interval != nil && interval.Type != nil {
		d := umetav1.ToDuration(interval).ToGo()
		if d < time.Second {
			return 0, grpcutils.InvalidArg("Invalid interval")
		}

		if to.Sub(from)/d > accessDataPointMaxBuckets {
			return 0, grpcutils.InvalidArg(
				"The time range and the interval produce more than %d buckets", accessDataPointMaxBuckets)
		}

		return d.Truncate(time.Second), nil
	}

	for _, d := range accessDataPointAutoIntervals {
		if to.Sub(from)/d <= accessDataPointAutoBuckets {
			return d, nil
		}
	}

	d := accessDataPointAutoIntervals[len(accessDataPointAutoIntervals)-1]
	if to.Sub(from)/d > accessDataPointMaxBuckets {
		return 0, grpcutils.InvalidArg("The time range is too large")
	}

	return d, nil
}

func getAccessDataPointBuckets(from, to time.Time, d time.Duration) []int64 {
	secs := int64(d / time.Second)

	var ret []int64
	for cur := (from.Unix() / secs) * secs; cur <= to.Unix(); cur += secs {
		ret = append(ret, cur)
	}

	return ret
}

type accessDataPointGroup struct {
	key  string
	name string
}

func getRequestDataPointGroup(groupBy vaccessv1.GetRequestDataPointRequest_GroupBy) (*accessDataPointGroup, error) {
	switch groupBy {
	case vaccessv1.GetRequestDataPointRequest_GROUP_BY_UNSET:
		return &accessDataPointGroup{key: `''`, name: `''`}, nil
	case vaccessv1.GetRequestDataPointRequest_STATE:
		return &accessDataPointGroup{
			key:  fmt.Sprintf(`COALESCE(%s, '%s')`, colStatusStateStatus, accessv1.Request_Status_State_STATUS_UNKNOWN.String()),
			name: `''`,
		}, nil
	case vaccessv1.GetRequestDataPointRequest_URGENCY:
		return &accessDataPointGroup{
			key:  fmt.Sprintf(`COALESCE(%s, '%s')`, colSpecUrgency, accessv1.Request_Spec_URGENCY_UNSET.String()),
			name: `''`,
		}, nil
	case vaccessv1.GetRequestDataPointRequest_EFFECT:
		return &accessDataPointGroup{
			key:  fmt.Sprintf(`COALESCE(%s, '%s')`, colStatusRuleEffect, accessv1.Policy_Spec_Rule_EFFECT_UNSET.String()),
			name: `''`,
		}, nil
	case vaccessv1.GetRequestDataPointRequest_ORIGIN:
		return &accessDataPointGroup{
			key:  fmt.Sprintf(`COALESCE(%s, '%s')`, colStatusOriginType, accessv1.Origin_TYPE_UNSET.String()),
			name: `''`,
		}, nil
	case vaccessv1.GetRequestDataPointRequest_POLICY:
		return &accessDataPointGroup{
			key:  fmt.Sprintf(`COALESCE(%s, '')`, colPolicyUID),
			name: fmt.Sprintf(`COALESCE(%s, '')`, colPolicyName),
		}, nil
	default:
		return nil, grpcutils.InvalidArg("Invalid groupBy")
	}
}

type accessDataPointRow struct {
	bucket int64
	key    string
	name   string
	count  int64
}

func (s *Server) getRequestDataPoint(ctx context.Context,
	req *vaccessv1.GetRequestDataPointRequest) (*vaccessv1.GetRequestDataPointResponse, error) {

	from, to, err := getAccessWindow(req.From, req.To)
	if err != nil {
		return nil, err
	}

	interval, err := getAccessDataPointInterval(from, to, req.Interval)
	if err != nil {
		return nil, err
	}

	group, err := getRequestDataPointGroup(req.GroupBy)
	if err != nil {
		return nil, err
	}

	limitSeries := int(req.LimitSeries)
	switch {
	case limitSeries == 0:
		limitSeries = accessDataPointDefaultSeries
	case limitSeries > accessDataPointMaxSeries:
		limitSeries = accessDataPointMaxSeries
	}

	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindRequest,
		&vmetav1.CommonSummaryOptions{
			From: timestamppb.New(from),
			To:   timestamppb.New(to),
		})
	if err != nil {
		return nil, err
	}

	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.PolicyRef, pth: "status.policyRef"},
		{ref: req.CatalogRef, pth: "spec.resource.catalog.catalogRef"},
		{ref: req.ServiceRef, pth: "spec.resource.serviceRef", opts: serviceRefCheckOpts()},
		{ref: req.UserRef, pth: "status.userRef"},
	})
	if err != nil {
		return nil, err
	}
	filters = append(filters, refFilters...)

	secs := int64(interval / time.Second)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(fmt.Sprintf(`CAST(floor(epoch(%s) / %d) AS BIGINT) * %d`, colCreatedAt, secs, secs)).As("bucket"),
			goqu.L(group.key).As("key"),
			goqu.L(fmt.Sprintf(`any_value(%s)`, group.name)).As("name"),
			goqu.L(`COUNT(*)`).As("count"),
		).
		GroupBy(goqu.L("bucket"), goqu.L("key"))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var items []*accessDataPointRow
	for rows.Next() {
		itm := &accessDataPointRow{}
		if err := rows.Scan(&itm.bucket, &itm.key, &itm.name, &itm.count); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}
		items = append(items, itm)
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	buckets := getAccessDataPointBuckets(from, to, interval)

	ret := &vaccessv1.GetRequestDataPointResponse{
		Datapoints: toRequestDataPoints(buckets, items, nil),
	}

	if req.GroupBy == vaccessv1.GetRequestDataPointRequest_GROUP_BY_UNSET {
		return ret, nil
	}

	type seriesTotal struct {
		key   string
		name  string
		total int64
	}

	totals := make(map[string]*seriesTotal)
	for _, itm := range items {
		if itm.key == "" {
			continue
		}
		if _, ok := totals[itm.key]; !ok {
			totals[itm.key] = &seriesTotal{key: itm.key, name: itm.name}
		}
		totals[itm.key].total += itm.count
	}

	var series []*seriesTotal
	for _, itm := range totals {
		series = append(series, itm)
	}

	sort.Slice(series, func(i, j int) bool {
		if series[i].total != series[j].total {
			return series[i].total > series[j].total
		}
		return series[i].key < series[j].key
	})

	if len(series) > limitSeries {
		series = series[:limitSeries]
	}

	for _, itm := range series {
		key := itm.key
		ret.Series = append(ret.Series, &vaccessv1.GetRequestDataPointResponse_Series{
			Key:         itm.key,
			DisplayName: itm.name,
			Total:       itm.total,
			Datapoints:  toRequestDataPoints(buckets, items, &key),
		})
	}

	return ret, nil
}

func toRequestDataPoints(buckets []int64, items []*accessDataPointRow,
	key *string) []*vaccessv1.GetRequestDataPointResponse_DataPoint {

	counts := make(map[int64]int64)
	for _, itm := range items {
		if key != nil && itm.key != *key {
			continue
		}
		counts[itm.bucket] += itm.count
	}

	ret := make([]*vaccessv1.GetRequestDataPointResponse_DataPoint, 0, len(buckets))
	for _, bucket := range buckets {
		ret = append(ret, &vaccessv1.GetRequestDataPointResponse_DataPoint{
			Count:     counts[bucket],
			Timestamp: timestamppb.New(time.Unix(bucket, 0).UTC()),
		})
	}

	return ret
}

type accessTopDimension struct {
	key  string
	name string
	kind string
}

func getRequestTopDimension(dim vaccessv1.ListRequestTopRequest_Dimension) (*accessTopDimension, error) {
	switch dim {
	case vaccessv1.ListRequestTopRequest_REQUESTER:
		return &accessTopDimension{key: colUserUID, name: colUserName, kind: `'User'`}, nil
	case vaccessv1.ListRequestTopRequest_SUBJECT:
		return &accessTopDimension{key: colSpecSubjectUserUID, name: colSpecSubjectUserName, kind: `'User'`}, nil
	case vaccessv1.ListRequestTopRequest_RESOURCE:
		return &accessTopDimension{
			key:  fmt.Sprintf(`COALESCE(%s, %s)`, colSpecServiceUID, colSpecCatalogUID),
			name: fmt.Sprintf(`COALESCE(%s, %s)`, colSpecServiceName, colSpecCatalogName),
			kind: fmt.Sprintf(`CASE WHEN %s IS NOT NULL THEN 'Service' ELSE 'Catalog' END`, colSpecServiceUID),
		}, nil
	case vaccessv1.ListRequestTopRequest_POLICY:
		return &accessTopDimension{key: colPolicyUID, name: colPolicyName, kind: `'Policy'`}, nil
	case vaccessv1.ListRequestTopRequest_DIMENSION_UNSET:
		return nil, grpcutils.InvalidArg("The dimension must be set")
	default:
		return nil, grpcutils.InvalidArg("Invalid dimension")
	}
}

func getReviewTopDimension(dim vaccessv1.ListReviewTopRequest_Dimension) (*accessTopDimension, error) {
	switch dim {
	case vaccessv1.ListReviewTopRequest_REVIEWER:
		return &accessTopDimension{key: colUserUID, name: colUserName, kind: `'User'`}, nil
	case vaccessv1.ListReviewTopRequest_DIMENSION_UNSET:
		return nil, grpcutils.InvalidArg("The dimension must be set")
	default:
		return nil, grpcutils.InvalidArg("Invalid dimension")
	}
}

func getAccessTopLimit(limit uint32) uint32 {
	switch {
	case limit == 0:
		return accessTopDefaultLimit
	case limit > accessTopMaxLimit:
		return accessTopMaxLimit
	default:
		return limit
	}
}

func (s *Server) listRequestTop(ctx context.Context,
	req *vaccessv1.ListRequestTopRequest) (*vaccessv1.ListTopResponse, error) {

	dim, err := getRequestTopDimension(req.Dimension)
	if err != nil {
		return nil, err
	}

	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindRequest,
		&vmetav1.CommonSummaryOptions{
			From: req.From,
			To:   req.To,
		})
	if err != nil {
		return nil, err
	}

	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.PolicyRef, pth: "status.policyRef"},
		{ref: req.CatalogRef, pth: "spec.resource.catalog.catalogRef"},
		{ref: req.ServiceRef, pth: "spec.resource.serviceRef", opts: serviceRefCheckOpts()},
	})
	if err != nil {
		return nil, err
	}
	filters = append(filters, refFilters...)

	if req.State != accessv1.Request_Status_State_STATUS_UNKNOWN {
		filters = append(filters, goqu.L(colStatusStateStatus).Eq(req.State.String()))
	}

	return s.listAccessTop(ctx, filters, dim, getAccessTopLimit(req.Limit))
}

func (s *Server) listReviewTop(ctx context.Context,
	req *vaccessv1.ListReviewTopRequest) (*vaccessv1.ListTopResponse, error) {

	dim, err := getReviewTopDimension(req.Dimension)
	if err != nil {
		return nil, err
	}

	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindReview,
		&vmetav1.CommonSummaryOptions{
			From: req.From,
			To:   req.To,
		})
	if err != nil {
		return nil, err
	}

	if req.Decision != accessv1.Review_Spec_DECISION_UNSET {
		filters = append(filters, goqu.L(colSpecDecision).Eq(req.Decision.String()))
	}

	return s.listAccessTop(ctx, filters, dim, getAccessTopLimit(req.Limit))
}

func (s *Server) listAccessTop(ctx context.Context, filters []exp.Expression,
	dim *accessTopDimension, limit uint32) (*vaccessv1.ListTopResponse, error) {

	filters = append(filters, goqu.L(fmt.Sprintf(`(%s IS NOT NULL AND %s != '')`, dim.key, dim.key)))

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(dim.key).As("key"),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(%s), '')`, dim.name)).As("name"),
			goqu.L(fmt.Sprintf(`any_value(%s)`, dim.kind)).As("kind"),
			goqu.L(`COUNT(*)`).As("count"),
		).
		GroupBy(goqu.L("key")).
		Order(goqu.L("count").Desc(), goqu.L("key").Asc()).
		Limit(uint(limit))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	ret := &vaccessv1.ListTopResponse{}
	var kinds []string
	var sum uint64

	for rows.Next() {
		var uid, name, kind string
		var count int64
		if err := rows.Scan(&uid, &name, &kind, &count); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, &vaccessv1.ListTopResponse_Item{
			Ref: &metav1.ObjectReference{
				ApiVersion: getAccessTopAPIVersion(kind),
				Kind:       kind,
				Uid:        uid,
				Name:       name,
			},
			Count: uint64(count),
		})
		kinds = append(kinds, kind)
		sum += uint64(count)
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	{
		ds := goqu.From("resources").Where(filters...).
			Select(
				goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, dim.key)),
				goqu.L(`COUNT(*)`),
			)

		sqln, sqlargs, err := ds.ToSQL()
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		var totalCount, total int64
		if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&totalCount, &total); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.TotalCount = uint64(totalCount)
		if uint64(total) > sum {
			ret.TotalOther = uint64(total) - sum
		}
	}

	if err := s.setAccessTopDisplayInfo(ctx, ret.Items, kinds); err != nil {
		return nil, err
	}

	return ret, nil
}

func getAccessTopAPIVersion(kind string) string {
	switch kind {
	case ucorev1.KindUser, ucorev1.KindService:
		return ucorev1.APIVersion
	default:
		return uaccessv1.APIVersion
	}
}

func (s *Server) setAccessTopDisplayInfo(ctx context.Context,
	items []*vaccessv1.ListTopResponse_Item, kinds []string) error {

	uidsByKind := make(map[string][]string)
	for i, itm := range items {
		uidsByKind[kinds[i]] = append(uidsByKind[kinds[i]], itm.Ref.Uid)
	}

	type displayInfo struct {
		displayName string
		picURL      string
	}

	infos := make(map[string]*displayInfo)

	for kind, uids := range uidsByKind {
		api := uaccessv1.API
		if getAccessTopAPIVersion(kind) == ucorev1.APIVersion {
			api = ucorev1.API
		}

		ds := goqu.From("resources").
			Where(
				goqu.L(`api`).Eq(api),
				goqu.L(`kind`).Eq(kind),
				goqu.L(`uid`).In(uids),
			).
			Select(
				goqu.L(`uid`),
				goqu.L(fmt.Sprintf(`COALESCE(%s, '')`, colDisplayName)),
				goqu.L(`COALESCE(json_extract_string(rsc, '$.metadata.picURL'), '')`),
			)

		sqln, sqlargs, err := ds.ToSQL()
		if err != nil {
			return grpcutils.InternalWithErr(err)
		}

		rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
		if err != nil {
			return grpcutils.InternalWithErr(err)
		}

		for rows.Next() {
			var uid string
			info := &displayInfo{}
			if err := rows.Scan(&uid, &info.displayName, &info.picURL); err != nil {
				rows.Close()
				return grpcutils.InternalWithErr(err)
			}
			infos[uid] = info
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return grpcutils.InternalWithErr(err)
		}
		rows.Close()
	}

	for _, itm := range items {
		if info, ok := infos[itm.Ref.Uid]; ok {
			itm.DisplayName = info.displayName
			itm.PicURL = info.picURL
		}
	}

	return nil
}
