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
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) insertAccessLog(accessLogJSON []byte) error {
	if len(accessLogJSON) < 1 {
		return nil
	}
	if len(accessLogJSON) > 25000 {
		zap.L().Warn("accessLog is too big. Skipping....", zap.String("accessLog", string(accessLogJSON)))
		return nil
	}

	_, err := s.db.Exec(`INSERT INTO access_logs VALUES ($1)`, string(accessLogJSON))

	return err
}

func (s *Server) insertComponentLog(logJSON []byte) error {
	_, err := s.db.Exec(`INSERT INTO component_logs VALUES ($1)`, string(logJSON))

	return err
}

func (s *Server) insertAuthenticationLog(logJSON []byte) error {
	_, err := s.db.Exec(`INSERT INTO authentication_logs VALUES ($1)`, string(logJSON))

	return err
}

func (s *Server) insertAuditLog(logJSON []byte) error {
	_, err := s.db.Exec(`INSERT INTO audit_logs VALUES ($1)`, string(logJSON))

	return err
}

const defaultItemsPerPage = 50
const maxItemsPerPage = 1000

func (s *Server) listAccessLog(ctx context.Context, req *visibilityv1.ListAccessLogRequest) (*visibilityv1.ListAccessLogResponse, error) {
	ret := &visibilityv1.ListAccessLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}
	var err error

	if req.Common == nil {
		req.Common = &vmetav1.CommonListOptions{}
	}

	var filters []exp.Expression

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
	/*
		if req.UserRef != nil {
			if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}

			if req.UserRef.Uid != "" {
				filters = append(filters, goqu.L(`rsc->>'$.entry.common.userRef.uid'`).Eq(req.UserRef.Uid))
			}

		}

		if req.DeviceRef != nil {
			if err := apivalidation.CheckObjectRef(req.DeviceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			if err := apivalidation.CheckObjectRef(req.SessionRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}

		if req.ServiceRef != nil {
			if err := apivalidation.CheckObjectRef(req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
				ParentsMust: 1,
			}); err != nil {
				return nil, err
			}
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.serviceRef.uid'`).Eq(req.ServiceRef.Uid))
		}

		if req.NamespaceRef != nil {
			if err := apivalidation.CheckObjectRef(req.NamespaceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.namespaceRef.uid'`).Eq(req.NamespaceRef.Uid))
		}

		if req.RegionRef != nil {
			if err := apivalidation.CheckObjectRef(req.RegionRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.regionRef.uid'`).Eq(req.RegionRef.Uid))
		}

		if req.PolicyRef != nil {
			if err := apivalidation.CheckObjectRef(req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
				ParentsMax: 8,
			}); err != nil {
				return nil, err
			}
			filters = append(filters,
				goqu.L(`rsc->>'$.entry.common.reason.details.policyMatch.policy.policyRef.uid'`).Eq(req.PolicyRef.Uid))
		}

		if req.MatchInlinePolicyRef != nil {
			if err := apivalidation.CheckObjectRef(req.MatchInlinePolicyRef, &apivalidation.CheckGetOptionsOpts{
				ParentsMax: 8,
			}); err != nil {
				return nil, err
			}
			filters = append(filters,
				goqu.L(`rsc->>'$.entry.common.reason.details.policyMatch.inlinePolicy.resourceRef.uid'`).
					Eq(req.MatchInlinePolicyRef.Uid))
		}
	*/

	if req.Status != corev1.AccessLog_Entry_Common_STATUS_UNSET {
		switch req.Status {
		case corev1.AccessLog_Entry_Common_ALLOWED:
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.status' = 'ALLOWED'`))
		case corev1.AccessLog_Entry_Common_DENIED:
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.status' = 'DENIED'`))
		}
	}

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("access_logs").Where(filters...).Select(
		goqu.L(`count(*) OVER() AS full_count`), "rsc")

	listMeta := ret.ListResponseMeta
	{
		limit := req.Common.ItemsPerPage
		if req.Common.Page > 100000 {
			return nil, grpcutils.InvalidArg(("Page number is too high"))
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Common.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Common.Page
	}

	{
		switch {
		case req.Common.OrderBy != nil && req.Common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_ASC:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Asc())
		default:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Desc())
		}

	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListAccessLogResponse{
				ListResponseMeta: &metav1.ListResponseMeta{},
			}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rsc := make(map[string]any)
		var count int

		err := rows.Scan(&count, &rsc)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.ListResponseMeta.TotalCount = uint32(count)

		accessLog := &corev1.AccessLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, accessLog)
	}

	if ret.ListResponseMeta.TotalCount > (ret.ListResponseMeta.Page+1)*ret.ListResponseMeta.ItemsPerPage {
		ret.ListResponseMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) listSSHSession(ctx context.Context, req *visibilityv1.ListSSHSessionRequest) (*visibilityv1.ListSSHSessionResponse, error) {
	ret := &visibilityv1.ListSSHSessionResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}

	if req.Common == nil {
		req.Common = &vmetav1.CommonListOptions{}
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

	/*
		if req.UserRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.userRef.uid'`).Eq(req.UserRef.Uid))
		}

		if req.DeviceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}

		if req.ServiceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.serviceRef.uid'`).Eq(req.ServiceRef.Uid))
		}

		if req.NamespaceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.namespaceRef.uid'`).Eq(req.NamespaceRef.Uid))
		}
	*/

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	{
		filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionID'`).IsNotNull())
	}

	ds := goqu.From("access_logs").
		Where(filters...).
		Select(
			goqu.L(`CAST(json_extract_string(rsc, '$.entry.common.sessionID') AS VARCHAR)`).As(`session_id`),
			goqu.L(`first(rsc) AS rsc_f`),
			// goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_RECORDING') AS count`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.sessionID')) AS count`),
			goqu.L(`min(rsc->>'$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_START') AS created_at`),
			goqu.L(`max(rsc->>'$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_END') AS ended_at`),
			// goqu.L(`json_extract(rsc, '$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_START') AS created_at`),
			// goqu.L(`json_extract(rsc, '$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_END') AS ended_at`),
			// goqu.L(`json_extract(rsc, '$.entry.info.ssh.type') AS session_type`),
		).GroupBy(goqu.L(`session_id`))

	listMeta := ret.ListResponseMeta
	{
		limit := req.Common.ItemsPerPage
		if req.Common.Page > 100000 {
			return nil, grpcutils.InvalidArg(("Page number is too high"))
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Common.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Common.Page
	}

	{
		ds = ds.Order(goqu.L(`created_at`).Desc())
	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.Query(sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListSSHSessionResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID string
		rsc := make(map[string]any)
		var count int
		var createdAt *string
		var endedAt *string
		// var sessionType string

		err := rows.Scan(&sessionID, &rsc, &count, &createdAt, &endedAt)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		accessLog := &corev1.AccessLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, &visibilityv1.SSHSession{
			Id: sessionID,
			// StartedAt: pbutils.Timestamp(vutils.MustParseTime(createdAt)),

			State: func() visibilityv1.SSHSession_State {
				if endedAt != nil {
					return visibilityv1.SSHSession_COMPLETED
				}
				return visibilityv1.SSHSession_ONGOING
			}(),
			StartedAt: func() *timestamppb.Timestamp {
				if createdAt == nil {
					return nil
				}
				return pbutils.Timestamp(vutils.MustParseTime(*createdAt))
			}(),
			EndedAt: func() *timestamppb.Timestamp {
				if endedAt == nil {
					return nil
				}
				return pbutils.Timestamp(vutils.MustParseTime(*endedAt))
			}(),
			SessionRef: accessLog.Entry.Common.SessionRef,
			DeviceRef:  accessLog.Entry.Common.DeviceRef,
			UserRef:    accessLog.Entry.Common.UserRef,

			ServiceRef:   accessLog.Entry.Common.ServiceRef,
			NamespaceRef: accessLog.Entry.Common.NamespaceRef,
		})

		listMeta.TotalCount = uint32(count)
	}

	if len(ret.Items) == 0 && listMeta.Page > 0 {
		return nil, grpcutils.NotFound("Not Items found for that page")
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) listSSHSessionRecording(ctx context.Context, req *visibilityv1.ListSSHSessionRecordingRequest) (*visibilityv1.ListSSHSessionRecordingResponse, error) {

	ret := &visibilityv1.ListSSHSessionRecordingResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}

	var filters []exp.Expression

	/*
		var from *timestamppb.Timestamp
		var to *timestamppb.Timestamp

		switch {
		case req.From == nil && req.To == nil:
			to = pbutils.Now()
			from = pbutils.Timestamp(to.AsTime().UTC().Add(-5 * time.Minute))
		case req.From != nil && req.To == nil:
			from = req.From
			to = pbutils.Timestamp(from.AsTime().UTC().Add(5 * time.Minute))
		case req.From == nil && req.To != nil:
			to = req.To
			from = pbutils.Timestamp(from.AsTime().UTC().Add(-5 * time.Minute))
		default:
			if req.From.AsTime().UTC().After(req.To.AsTime().UTC()) {
				return nil, grpcutils.InvalidArg("Invalid timing")
			}
		}
	*/

	/*
		{
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionID'`).Eq(req.SessionID),
				goqu.L(`rsc->>'$.entry.info.ssh.type'`).Eq(`SESSION_RECORDING`))
		}

		{
			filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(from.AsTime().UTC().Format(time.RFC3339Nano)))
		}
	*/

	/*
		if req.To != nil {
			filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(to.AsTime().UTC().Format(time.RFC3339Nano)))
		}
	*/

	{
		filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionID'`).Eq(req.SessionID),
			goqu.L(`rsc->>'$.entry.info.ssh.type'`).Eq(`SESSION_RECORDING`))
	}

	ds := goqu.From("access_logs").
		Where(filters...).
		Select(
			goqu.L(`rsc`),
			goqu.L(`count(*) OVER() AS full_count`),
		)

	{
		limit := 1000
		offset := int(req.Page) * limit
		ds = ds.Offset(uint(offset)).Limit(uint(limit))
	}

	listMeta := ret.ListResponseMeta
	{

		if req.Page > 10000 {
			return nil, grpcutils.InvalidArg("Page number is too high")
		}

		limit := 1000
		offset := int(req.Page) * limit
		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = uint32(limit)
		listMeta.Page = req.Page
	}

	{
		ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Asc())
	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.Query(sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListSSHSessionRecordingResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		rsc := make(map[string]any)

		err := rows.Scan(&rsc, &count)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		accessLog := &corev1.AccessLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		if accessLog.Entry == nil || accessLog.Entry.Info == nil ||
			accessLog.Entry.Info.GetSsh() == nil ||
			accessLog.Entry.Info.GetSsh().GetSessionRecording() == nil ||
			accessLog.Entry.Info.GetSsh().GetSessionRecording().Data == nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListSSHSessionRecordingResponse_Recording{
			Timestamp: accessLog.Metadata.CreatedAt,
			Data:      accessLog.Entry.Info.GetSsh().GetSessionRecording().Data,
		})
	}

	listMeta.TotalCount = uint32(count)

	if len(ret.Items) == 0 && listMeta.Page > 0 {
		return nil, grpcutils.NotFound("Not Items found for that page")
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) getSummaryAccessLog(ctx context.Context, req *visibilityv1.GetAccessLogSummaryRequest) (*visibilityv1.GetAccessLogSummaryResponse, error) {
	ret := &visibilityv1.GetAccessLogSummaryResponse{}

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

	/*
		if req.UserRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.userRef.uid'`).Eq(req.UserRef.Uid))
		}

		if req.DeviceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}

		if req.ServiceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.serviceRef.uid'`).Eq(req.ServiceRef.Uid))
		}

		if req.NamespaceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.namespaceRef.uid'`).Eq(req.NamespaceRef.Uid))
		}

		if req.RegionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.common.regionRef.uid'`).Eq(req.RegionRef.Uid))
		}

		if req.PolicyRef != nil {
			if err := apivalidation.CheckObjectRef(req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
				ParentsMax: 8,
			}); err != nil {
				return nil, err
			}
			filters = append(filters,
				goqu.L(`rsc->>'$.entry.common.reason.details.policyMatch.policy.policyRef.uid'`).Eq(req.PolicyRef.Uid))
		}
	*/

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("access_logs").Where(filters...).Select(
		goqu.L(`COUNT(*) AS count_total`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.common.status') = 'ALLOWED') AS count_allowed`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.common.status') = 'DENIED') AS count_denied`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.userRef.uid')) AS count_user`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.sessionRef.uid')) AS count_session`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.deviceRef.uid')) AS count_device`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.serviceRef.uid')) AS count_service`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.namespaceRef.uid')) AS count_namespace`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.reason.details.policyMatch.policy.policyRef.uid')) AS count_match_policy`),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.GetAccessLogSummaryResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalAllowed, &ret.TotalDenied,
			&ret.TotalUser, &ret.TotalSession, &ret.TotalDevice,
			&ret.TotalService, &ret.TotalNamespace, &ret.TotalMatchPolicy)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	return ret, nil
}

func (s *Server) getSummaryAuthenticationLog(ctx context.Context, req *visibilityv1.GetAuthenticationLogSummaryRequest) (*visibilityv1.GetAuthenticationLogSummaryResponse, error) {
	ret := &visibilityv1.GetAuthenticationLogSummaryResponse{}

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

	/*
		if req.UserRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.userRef.uid'`).Eq(req.UserRef.Uid))
		}

		if req.DeviceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}

		if req.IdentityProviderRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.authentication.info.identityProvider.identityProviderRef.uid'`).Eq(req.IdentityProviderRef.Uid))
		}
	*/

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("authentication_logs").Where(filters...).Select(
		goqu.L(`COUNT(*) AS count_total`),

		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.type') = 'REFRESH_TOKEN') AS count_refresh_token`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.type') = 'AUTHENTICATOR') AS count_authenticator`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.type') = 'IDENTITY_PROVIDER') AS count_identity_provider`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.type') = 'CREDENTIAL') AS count_credential`),

		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.aal') = 'AAL1') AS count_aal1`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.aal') = 'AAL2') AS count_aal2`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.aal') = 'AAL3') AS count_aal3`),

		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.authenticator.type') = 'FIDO') AS count_authenticator_fido`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.authenticator.type') = 'TOTP') AS count_authenticator_totp`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.authenticator.type') = 'TPM') AS count_authenticator_tpm`),

		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.authenticator.mode') = 'PASSKEY') AS count_authenticator_passkey`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.authentication.info.authenticator.mode') = 'MFA') AS count_authenticator_mfa`),

		goqu.L(`COUNT(*) FILTER (WHERE (rsc ->> '$.entry.authenticationIndex')::INTEGER > 0) AS count_reauth`),

		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.userRef.uid')) AS count_user`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.sessionRef.uid')) AS count_session`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.identityProviderRef.uid')) AS count_identity_provider`),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.GetAuthenticationLogSummaryResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalRefreshToken, &ret.TotalAuthenticator, &ret.TotalIdentityProvider, &ret.TotalCredential,
			&ret.TotalAAL1, &ret.TotalAAL2, &ret.TotalAAL3,
			&ret.TotalAuthenticatorFIDO, &ret.TotalAuthenticatorTOTP, &ret.TotalAuthenticatorTPM,
			&ret.TotalAuthenticatorPasskey, &ret.TotalAuthenticatorMFA, &ret.TotalReauthentication,
			&ret.TotalUser, &ret.TotalSession, &ret.TotalIdentityProvider)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

	}

	return ret, nil
}

func (s *Server) listAuthenticationLog(ctx context.Context, req *visibilityv1.ListAuthenticationLogRequest) (*visibilityv1.ListAuthenticationLogResponse, error) {
	ret := &visibilityv1.ListAuthenticationLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}
	if req.Common == nil {
		req.Common = &vmetav1.CommonListOptions{}
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

	/*
		if req.UserRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.userRef.uid'`).Eq(req.UserRef.Uid))
		}

		if req.DeviceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}
	*/

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("authentication_logs").Where(filters...).Select(
		goqu.L(`count(*) OVER() AS full_count`), "rsc")

	listMeta := ret.ListResponseMeta
	{
		limit := req.Common.ItemsPerPage
		if req.Common.Page > 100000 {
			return nil, grpcutils.InvalidArg(("Page number is too high"))
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Common.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Common.Page
	}

	{
		switch {
		case req.Common.OrderBy != nil && req.Common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_ASC:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Asc())
		default:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Desc())
		}

	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListAuthenticationLogResponse{
				ListResponseMeta: &metav1.ListResponseMeta{},
			}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rsc := make(map[string]any)
		var count int

		err := rows.Scan(&count, &rsc)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.ListResponseMeta.TotalCount = uint32(count)

		accessLog := &enterprisev1.AuthenticationLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, accessLog)
	}

	if ret.ListResponseMeta.TotalCount > (ret.ListResponseMeta.Page+1)*ret.ListResponseMeta.ItemsPerPage {
		ret.ListResponseMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) listAuditLog(ctx context.Context, req *visibilityv1.ListAuditLogRequest) (*visibilityv1.ListAuditLogResponse, error) {
	ret := &visibilityv1.ListAuditLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}

	if req.Common == nil {
		req.Common = &vmetav1.CommonListOptions{}
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
	filters, err = appendRefFilter(filters, req.ResourceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, "entry.resourceRef")
	if err != nil {
		return nil, err
	}

	/*
		if req.ResourceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.resourceRef.uid'`).Eq(req.ResourceRef.Uid))
		}

		if req.UserRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.userRef.uid'`).Eq(req.UserRef.Uid))
		}

		if req.DeviceRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}

		if req.SessionRef != nil {
			filters = append(filters, goqu.L(`rsc->>'$.entry.sessionRef.uid'`).Eq(req.SessionRef.Uid))
		}
	*/

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("audit_logs").Where(filters...).Select(
		goqu.L(`count(*) OVER() AS full_count`), "rsc")

	listMeta := ret.ListResponseMeta
	{
		limit := req.Common.ItemsPerPage
		if req.Common.Page > 100000 {
			return nil, grpcutils.InvalidArg(("Page number is too high"))
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Common.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Common.Page
	}

	{
		switch {
		case req.Common.OrderBy != nil && req.Common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_ASC:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Asc())
		default:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Desc())
		}

	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListAuditLogResponse{
				ListResponseMeta: &metav1.ListResponseMeta{},
			}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rsc := make(map[string]any)
		var count int

		err := rows.Scan(&count, &rsc)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.ListResponseMeta.TotalCount = uint32(count)

		auditLog := &enterprisev1.AuditLog{}
		if err := pbutils.UnmarshalFromMap(rsc, auditLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, auditLog)
	}

	if ret.ListResponseMeta.TotalCount > (ret.ListResponseMeta.Page+1)*ret.ListResponseMeta.ItemsPerPage {
		ret.ListResponseMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) listAccessLogTop(ctx context.Context) error {

	rows, err := s.db.QueryContext(ctx, `
-- Try with a subquery to ensure proper aggregation
WITH extracted_data AS (
    SELECT json_extract_string(rsc, '$.entry.common.userRef.uid') as user_id
    FROM access_logs
    WHERE json_extract_string(rsc, '$.entry.common.userRef.uid') IS NOT NULL
)
SELECT 
    user_id,
    COUNT(*) as frequency
FROM extracted_data
GROUP BY user_id
ORDER BY frequency DESC
LIMIT 5;
`)

	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var count int

		if err := rows.Scan(&id, &count); err != nil {
			return err
		}

	}
	defer rows.Close()

	return nil
}

func (s *Server) listComponentLog(ctx context.Context, req *visibilityv1.ListComponentLogRequest) (*visibilityv1.ListComponentLogResponse, error) {
	ret := &visibilityv1.ListComponentLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}
	if req.Common == nil {
		req.Common = &vmetav1.CommonListOptions{}
	}

	var filters []exp.Expression

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.Level != corev1.ComponentLog_Entry_LEVEL_UNSET {
		filters = append(filters, goqu.L(`rsc->>'$.entry.level'`).Eq(req.Level.String()))
	}

	ds := goqu.From("component_logs").Where(filters...).Select(
		goqu.L(`count(*) OVER() AS full_count`), "rsc")

	listMeta := ret.ListResponseMeta
	{
		limit := req.Common.ItemsPerPage
		if req.Common.Page > 100000 {
			return nil, grpcutils.InvalidArg(("Page number is too high"))
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Common.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Common.Page
	}

	{
		switch {
		case req.Common.OrderBy != nil && req.Common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_ASC:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Asc())
		default:
			ds = ds.Order(goqu.L(`rsc->>'$.metadata.createdAt'`).Desc())
		}

	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.ListComponentLogResponse{
				ListResponseMeta: &metav1.ListResponseMeta{},
			}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rsc := make(map[string]any)
		var count int

		err := rows.Scan(&count, &rsc)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.ListResponseMeta.TotalCount = uint32(count)

		accessLog := &corev1.ComponentLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, accessLog)
	}

	if ret.ListResponseMeta.TotalCount > (ret.ListResponseMeta.Page+1)*ret.ListResponseMeta.ItemsPerPage {
		ret.ListResponseMeta.HasMore = true
	}

	return ret, nil
}

func (s *Server) getSSHSession(ctx context.Context, req *visibilityv1.GetSSHSessionRequest) (*visibilityv1.SSHSession, error) {
	ret := &visibilityv1.SSHSession{}

	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`rsc->>'$.entry.common.sessionID'`).Eq(req.Id))
	}

	ds := goqu.From("access_logs").
		Where(filters...).
		Select(
			goqu.L(`CAST(json_extract_string(rsc, '$.entry.common.sessionID') AS VARCHAR)`).As(`session_id`),
			goqu.L(`first(rsc) AS rsc_f`),
			// goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_RECORDING') AS count`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.common.sessionID')) AS count`),
			goqu.L(`min(rsc->>'$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_START') AS created_at`),
			goqu.L(`max(rsc->>'$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_END') AS ended_at`),
			// goqu.L(`json_extract(rsc, '$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_START') AS created_at`),
			// goqu.L(`json_extract(rsc, '$.metadata.createdAt') FILTER (WHERE rsc->>'$.entry.info.ssh.type' = 'SESSION_END') AS ended_at`),
			// goqu.L(`json_extract(rsc, '$.entry.info.ssh.type') AS session_type`),
		).GroupBy(goqu.L(`session_id`))

	{
		ds = ds.Order(goqu.L(`created_at`).Desc())
	}

	ds = ds.Limit(1)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var sessionID string
	rsc := make(map[string]any)
	var count int
	var createdAt *string
	var endedAt *string

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&sessionID, &rsc, &count, &createdAt, &endedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grpcutils.NotFound("No SSH Sesion with this ID: %s", req.Id)
		}

		return nil, grpcutils.InternalWithErr(err)
	}

	accessLog := &corev1.AccessLog{}
	if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret = &visibilityv1.SSHSession{
		Id: sessionID,

		State: func() visibilityv1.SSHSession_State {
			if endedAt != nil {
				return visibilityv1.SSHSession_COMPLETED
			}
			return visibilityv1.SSHSession_ONGOING
		}(),
		StartedAt: func() *timestamppb.Timestamp {
			if createdAt == nil {
				return nil
			}
			return pbutils.Timestamp(vutils.MustParseTime(*createdAt))
		}(),
		EndedAt: func() *timestamppb.Timestamp {
			if endedAt == nil {
				return nil
			}
			return pbutils.Timestamp(vutils.MustParseTime(*endedAt))
		}(),
		SessionRef: accessLog.Entry.Common.SessionRef,
		DeviceRef:  accessLog.Entry.Common.DeviceRef,
		UserRef:    accessLog.Entry.Common.UserRef,

		ServiceRef:   accessLog.Entry.Common.ServiceRef,
		NamespaceRef: accessLog.Entry.Common.NamespaceRef,
	}

	return ret, nil
}

func appendRefFilter(filters []exp.Expression, ref *metav1.ObjectReference, o *apivalidation.CheckGetOptionsOpts, pth string) ([]exp.Expression, error) {
	if ref == nil {
		return filters, nil
	}
	if o == nil {
		o = &apivalidation.CheckGetOptionsOpts{}
	}

	if err := apivalidation.CheckObjectRef(ref, o); err != nil {
		return nil, err
	}

	if ref.Uid != "" {
		filterName := fmt.Sprintf(`rsc->>'$.%s.uid'`, pth)
		filters = append(filters, goqu.L(filterName).Eq(ref.Uid))
	}

	if ref.Name != "" {
		filterName := fmt.Sprintf(`rsc->>'$.%s.name'`, pth)
		filters = append(filters, goqu.L(filterName).Eq(ref.Name))
	}

	return filters, nil
}

func (s *Server) getSummaryComponentLog(ctx context.Context, req *visibilityv1.GetComponentLogSummaryRequest) (*visibilityv1.GetComponentLogSummaryResponse, error) {
	ret := &visibilityv1.GetComponentLogSummaryResponse{}

	var filters []exp.Expression
	var err error

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("component_logs").Where(filters...).Select(
		goqu.L(`COUNT(*) AS count_total`),

		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'DEBUG') AS count_debug`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'INFO') AS count_info`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'WARNING') AS count_warning`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'ERROR') AS count_error`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'PANIC') AS count_panic`),
		goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(rsc, '$.entry.level') = 'FATAL') AS count_fatal`),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.GetComponentLogSummaryResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDebug, &ret.TotalInfo, &ret.TotalWarn,
			&ret.TotalError, &ret.TotalPanic, &ret.TotalFatal)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

	}

	return ret, nil
}

func (s *Server) getSummaryAuditLog(ctx context.Context, req *visibilityv1.GetAuditLogSummaryRequest) (*visibilityv1.GetAuditLogSummaryResponse, error) {
	ret := &visibilityv1.GetAuditLogSummaryResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	ds := goqu.From("audit_logs").Where(filters...).Select(
		goqu.L(`COUNT(*) AS count_total`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.resourceRef.uid')) AS count_resource`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.userRef.uid')) AS count_user`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.sessionRef.uid')) AS count_session`),
		goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.entry.deviceRef.uid')) AS count_device`),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &visibilityv1.GetAuditLogSummaryResponse{}, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber, &ret.TotalResource,
			&ret.TotalUser, &ret.TotalSession, &ret.TotalDevice)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	return ret, nil
}
