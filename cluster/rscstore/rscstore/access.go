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
	"fmt"
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	now := time.Now().UTC()

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING') AS count_pending`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'APPROVED') AS count_approved`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REJECTED') AS count_rejected`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REVOKED') AS count_revoked`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'EXPIRED') AS count_expired`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'CANCELLED') AS count_cancelled`, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE (%s = 'APPROVED') AND ((%s IS NULL) OR (%s > ?))) AS count_active`, colStatusStateStatus, colStatusAccessEndsAt, colStatusAccessEndsAt),
				now),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_subject_user`, colSpecSubjectUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_service`, colSpecServiceUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_catalog`, colSpecCatalogUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_policy`, colPolicyUID)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'VERY_LOW') AS count_urgency_very_low`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'LOW') AS count_urgency_low`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'NORMAL') AS count_urgency_normal`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'HIGH') AS count_urgency_high`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'VERY_HIGH') AS count_urgency_very_high`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'HIGHEST') AS count_urgency_highest`, colSpecUrgency)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'deadline')) AS count_with_deadline`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s < ?) AS count_deadline_passed`, colSpecDeadline), now),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE (%s IS NULL) OR (%s = 'DECISION_UNSET')) AS count_pending`, colSpecDecision, colSpecDecision)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DECISION_APPROVE') AS count_approved`, colSpecDecision)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DECISION_REJECT') AS count_rejected`, colSpecDecision)),
			goqu.L(`COUNT(*) FILTER (WHERE json_array_length(rsc, '$.status.lastRevisions') > 0) AS count_revised`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_request`, colRequestUID)),
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
