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
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) getSummaryAccessPolicy(ctx context.Context, req *vaccessv1.GetPolicySummaryRequest) (*vaccessv1.GetPolicySummaryResponse, error) {

	ret := &vaccessv1.GetPolicySummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uaccessv1.KindPolicy))
		filters = append(filters, goqu.L(`api`).Eq(uaccessv1.API))
		filters = append(filters, goqu.L(`version`).Eq(uaccessv1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.rules')), 0) AS count_rules`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> json_extract_string(x, '$.effect') = 'DENY'
				))
			), 0) AS count_rules_deny`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> json_extract_string(x, '$.effect') = 'REVIEW'
				))
			), 0) AS count_rules_review`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> json_extract_string(x, '$.effect') = 'AUTO_APPROVE'
				))
			), 0) AS count_rules_auto_approve`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> json_extract(x, '$.authorization') IS NOT NULL
				))
			), 0) AS count_rules_authorization`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> json_extract(x, '$.authorization.maxAccessDuration') IS NOT NULL
				))
			), 0) AS count_rules_max_access_duration`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> COALESCE(json_array_length(x, '$.action.review.steps'), 0)
				))
			), 0) AS count_review_steps`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					x -> list_sum(list_transform(
						COALESCE(CAST(json_extract(x, '$.action.review.steps') AS JSON[]), []),
						y -> COALESCE(json_array_length(y, '$.reviewers'), 0)
					))
				))
			), 0) AS count_reviewers`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDisabled,
			&ret.TotalRule,
			&ret.TotalRuleDeny, &ret.TotalRuleReview, &ret.TotalRuleAutoApprove,
			&ret.TotalRuleAuthorization, &ret.TotalRuleMaxAccessDuration,
			&ret.TotalReviewStep, &ret.TotalReviewer)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryAccessCatalog(ctx context.Context, req *vaccessv1.GetCatalogSummaryRequest) (*vaccessv1.GetCatalogSummaryResponse, error) {

	ret := &vaccessv1.GetCatalogSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uaccessv1.KindCatalog))
		filters = append(filters, goqu.L(`api`).Eq(uaccessv1.API))
		filters = append(filters, goqu.L(`version`).Eq(uaccessv1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.resourceCollection.service.services')), 0) AS count_service`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.resourceCollection.service.namespaces')), 0) AS count_namespace`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber, &ret.TotalService, &ret.TotalNamespace)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryAccessRequest(ctx context.Context, req *vaccessv1.GetRequestSummaryRequest) (*vaccessv1.GetRequestSummaryResponse, error) {

	ret := &vaccessv1.GetRequestSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uaccessv1.KindRequest))
		filters = append(filters, goqu.L(`api`).Eq(uaccessv1.API))
		filters = append(filters, goqu.L(`version`).Eq(uaccessv1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'PENDING') AS count_pending`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'APPROVED') AS count_approved`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'REJECTED') AS count_rejected`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'REVOKED') AS count_revoked`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'EXPIRED') AS count_expired`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state.status' = 'CANCELLED') AS count_cancelled`),
			goqu.L(`COUNT(*) FILTER (WHERE (rsc->>'$.status.state.status' = 'APPROVED') AND ((json_extract(rsc, '$.status.accessEndsAt') IS NULL) OR (rsc->>'$.status.accessEndsAt' > ?))) AS count_active`,
				nowStr),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.spec.subject.userRef.uid')) AS count_subject_user`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.spec.resource.serviceRef.uid')) AS count_service`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.spec.resource.catalog.catalogRef.uid')) AS count_catalog`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.policyRef.uid')) AS count_policy`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'VERY_LOW') AS count_urgency_very_low`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'LOW') AS count_urgency_low`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'NORMAL') AS count_urgency_normal`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'HIGH') AS count_urgency_high`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'VERY_HIGH') AS count_urgency_very_high`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.urgency' = 'HIGHEST') AS count_urgency_highest`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.deadline') IS NOT NULL) AS count_with_deadline`),
			goqu.L(`COUNT(*) FILTER (WHERE (rsc->>'$.spec.deadline') < ?) AS count_deadline_passed`, nowStr),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalPending, &ret.TotalApproved, &ret.TotalRejected, &ret.TotalRevoked,
			&ret.TotalExpired, &ret.TotalCancelled, &ret.TotalActive,
			&ret.TotalUser, &ret.TotalSubjectUser, &ret.TotalService, &ret.TotalCatalog, &ret.TotalPolicy,
			&ret.TotalUrgencyVeryLow, &ret.TotalUrgencyLow, &ret.TotalUrgencyNormal,
			&ret.TotalUrgencyHigh, &ret.TotalUrgencyVeryHigh, &ret.TotalUrgencyHighest,
			&ret.TotalWithDeadline, &ret.TotalDeadlinePassed)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryAccessReview(ctx context.Context, req *vaccessv1.GetReviewSummaryRequest) (*vaccessv1.GetReviewSummaryResponse, error) {

	ret := &vaccessv1.GetReviewSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uaccessv1.KindReview))
		filters = append(filters, goqu.L(`api`).Eq(uaccessv1.API))
		filters = append(filters, goqu.L(`version`).Eq(uaccessv1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE (json_extract(rsc, '$.spec.decision') IS NULL) OR (rsc->>'$.spec.decision' = 'DECISION_UNSET')) AS count_pending`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.decision' = 'DECISION_APPROVE') AS count_approved`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.decision' = 'DECISION_REJECT') AS count_rejected`),
			goqu.L(`COUNT(*) FILTER (WHERE json_array_length(rsc, '$.status.lastRevisions') > 0) AS count_revised`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.requestRef.uid')) AS count_request`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalPending, &ret.TotalApproved, &ret.TotalRejected, &ret.TotalRevised,
			&ret.TotalUser, &ret.TotalRequest)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}
