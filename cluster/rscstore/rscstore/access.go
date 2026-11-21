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
	"regexp"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) doSummaryAccessPolicy(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetPolicySummaryResponse, error) {

	ret := &vaccessv1.GetPolicySummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindPolicy, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.rules')), 0) AS count_rules`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : json_extract_string(x, '$.effect') = 'DENY'
				))
			), 0) AS count_rules_deny`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : json_extract_string(x, '$.effect') = 'REVIEW'
				))
			), 0) AS count_rules_review`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : json_extract_string(x, '$.effect') = 'AUTO_APPROVE'
				))
			), 0) AS count_rules_auto_approve`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : COALESCE(json_array_length(x, '$.action.review.steps'), 0)
				))
			), 0) AS count_review_steps`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : list_sum(list_transform(
						COALESCE(CAST(json_extract(x, '$.action.review.steps') AS JSON[]), []),
						lambda y : COALESCE(json_array_length(y, '$.reviewers'), 0)
					))
				))
			), 0) AS count_reviewers`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : COALESCE(json_array_length(x, '$.notifications'), 0) > 0
				))
			), 0) AS count_rules_notification`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : list_sum(list_transform(
						COALESCE(CAST(json_extract(x, '$.action.review.steps') AS JSON[]), []),
						lambda y : COALESCE(json_array_length(y, '$.surfaces'), 0)
					))
				))
			), 0) AS count_review_surfaces`),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : list_sum(list_transform(
						COALESCE(CAST(json_extract(x, '$.action.review.steps') AS JSON[]), []),
						lambda y : len(list_filter(
							COALESCE(CAST(json_extract(y, '$.surfaces') AS JSON[]), []),
							lambda z : json_extract_string(z, '$.interactionMode') = 'INTERACTIVE'
						))
					))
				))
			), 0) AS count_interactive_surfaces`),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : json_extract(x, '$.maxDuration') IS NOT NULL
				))
			), 0) AS count_rules_max_duration`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE len(COALESCE(%s, [])) > 0) AS count_with_issues`,
				colStatusIssueTypes)),
			goqu.L(`COALESCE(SUM(
				list_sum(list_transform(
					CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
					lambda x : len(list_filter(
						COALESCE(CAST(json_extract(x, '$.action.review.steps') AS JSON[]), []),
						lambda y : json_extract(y, '$.escalation') IS NOT NULL
					))
				))
			), 0) AS count_review_escalations`),
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
			&ret.TotalReviewStep, &ret.TotalReviewer,
			&ret.TotalRuleNotification, &ret.TotalReviewSurface, &ret.TotalInteractiveSurface,
			&ret.TotalRuleMaxDuration, &ret.TotalWithIssues, &ret.TotalReviewEscalation)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessCatalog(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetCatalogSummaryResponse, error) {

	ret := &vaccessv1.GetCatalogSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindCatalog, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.resourceCollection.service.services')), 0) AS count_service`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.resourceCollection.service.namespaces')), 0) AS count_namespace`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.accessLevels')), 0) AS count_access_levels`),
			goqu.L(`COALESCE(SUM(len(list_filter(
				COALESCE(CAST(json_extract(rsc, '$.spec.accessLevels') AS JSON[]), []),
				lambda x : json_extract(x, '$.full') IS NOT NULL
			))), 0) AS count_full_access_levels`),
			goqu.L(`COALESCE(SUM(len(list_filter(
				COALESCE(CAST(json_extract(rsc, '$.spec.accessLevels') AS JSON[]), []),
				lambda x : json_extract(x, '$.restricted') IS NOT NULL
			))), 0) AS count_restricted_access_levels`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(COALESCE(%s, []), 'OVERRIDES_GUARDRAILS')) AS count_overrides_guardrails`,
				colStatusIssueTypes)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE len(COALESCE(%s, [])) > 0) AS count_with_issues`,
				colStatusIssueTypes)),
			goqu.L(`COUNT(*) FILTER (WHERE COALESCE(json_array_length(rsc, '$.spec.audience.users'), 0) > 0
				OR COALESCE(json_array_length(rsc, '$.spec.audience.groups'), 0) > 0
				OR COALESCE(json_extract_string(rsc, '$.spec.audience.match'), '') != '') AS count_restricted_audience`),
			goqu.L(`COUNT(*) FILTER (WHERE COALESCE(json_array_length(rsc, '$.spec.owners.users'), 0) > 0
				OR COALESCE(json_array_length(rsc, '$.spec.owners.groups'), 0) > 0) AS count_with_owners`),
			goqu.L(`COUNT(*) FILTER (WHERE COALESCE(json_array_length(rsc, '$.spec.audience.onBehalf.users'), 0) > 0
				OR COALESCE(json_array_length(rsc, '$.spec.audience.onBehalf.groups'), 0) > 0) AS count_on_behalf`),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalService, &ret.TotalNamespace,
			&ret.TotalDisabled, &ret.TotalAccessLevel,
			&ret.TotalFullAccessLevel, &ret.TotalRestrictedAccessLevel,
			&ret.TotalOverridesGuardrails, &ret.TotalWithIssues,
			&ret.TotalRestrictedAudience, &ret.TotalWithOwners, &ret.TotalOnBehalf)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

type requestSummaryOpts struct {
	refFilters     []exp.Expression
	staleAfter     time.Duration
	expiringWithin time.Duration
}

func (s *Server) doSummaryAccessRequest(ctx context.Context, common *vmetav1.CommonSummaryOptions,
	o *requestSummaryOpts) (*vaccessv1.GetRequestSummaryResponse, error) {

	ret := &vaccessv1.GetRequestSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindRequest, common)
	if err != nil {
		return nil, err
	}
	filters = append(filters, o.refFilters...)

	now := time.Now().UTC()
	decided := getRequestDecidedExpr()
	decisionSeconds := fmt.Sprintf(`(epoch(%s) - epoch(%s))`, colStatusApprovalEndAt, colStatusApprovalStartAt)

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
			goqu.L(fmt.Sprintf(`COALESCE(len(list_distinct(flatten(list(COALESCE(%s, []))))), 0) AS count_service`,
				colStatusScopeServiceUIDs)),
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
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'AUTO_APPROVE') AS count_auto_approved`, colStatusRuleEffect)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REVIEW') AS count_reviewed`, colStatusRuleEffect)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DENY') AS count_denied`, colStatusRuleEffect)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_unmatched`, getRequestUnmatchedExpr())),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_on_behalf`, getRequestOnBehalfExpr())),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'INTEGRATION') AS count_from_integration`, colStatusOriginType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_expiring_soon`, getRequestExpiringSoonExpr()),
				now, now.Add(o.expiringWithin)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_pending_stale`, getRequestPendingStaleExpr()),
				now.Add(-o.staleAfter)),
			goqu.L(fmt.Sprintf(`MIN(%s) FILTER (WHERE %s = 'PENDING') AS oldest_pending_at`,
				colStatusApprovalStartAt, colStatusStateStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_decided`, decided)),
			goqu.L(fmt.Sprintf(`COALESCE(AVG(%s) FILTER (WHERE %s), 0) AS decision_avg`, decisionSeconds, decided)),
			goqu.L(fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.5) FILTER (WHERE %s), 0) AS decision_p50`, decisionSeconds, decided)),
			goqu.L(fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.9) FILTER (WHERE %s), 0) AS decision_p90`, decisionSeconds, decided)),
			goqu.L(fmt.Sprintf(`COALESCE(MAX(%s) FILTER (WHERE %s), 0) AS decision_max`, decisionSeconds, decided)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING' AND %s < ?) AS count_pending_deadline_passed`,
				colStatusStateStatus, colSpecDeadline), now),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_granted`, getRequestGrantedExpr())),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PROVISIONING_FAILED') AS count_provisioning_failed`,
				colStatusStateReason)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'EVALUATION_FAILED') AS count_evaluation_failed`,
				colStatusStateReason)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_unroutable`, getRequestUnroutableExpr())),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s) AS count_awaiting_requester`,
				getRequestAwaitingRequesterExpr())),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'APPROVED' AND %s = true AND ((%s IS NULL) OR (%s > ?))) AS count_overrides_guardrails`,
				colStatusStateStatus, colStatusOverridesGuardrails, colStatusAccessEndsAt, colStatusAccessEndsAt), now),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'APPROVED' AND %s = 'SCHEDULED') AS count_scheduled`,
				colStatusStateStatus, colStatusGrantState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ENDED_EARLY') AS count_ended_early`,
				colStatusStateReason)),
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
		var oldestPendingAt sql.NullTime
		decisionTime := &vaccessv1.DecisionTime{}

		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalPending, &ret.TotalApproved, &ret.TotalRejected, &ret.TotalRevoked,
			&ret.TotalExpired, &ret.TotalCancelled, &ret.TotalActive,
			&ret.TotalUser, &ret.TotalSubjectUser, &ret.TotalService, &ret.TotalCatalog, &ret.TotalPolicy,
			&ret.TotalUrgencyVeryLow, &ret.TotalUrgencyLow, &ret.TotalUrgencyNormal,
			&ret.TotalUrgencyHigh, &ret.TotalUrgencyVeryHigh, &ret.TotalUrgencyHighest,
			&ret.TotalWithDeadline, &ret.TotalDeadlinePassed,
			&ret.TotalAutoApproved, &ret.TotalReviewed, &ret.TotalDenied, &ret.TotalUnmatched,
			&ret.TotalOnBehalf, &ret.TotalFromIntegration,
			&ret.TotalExpiringSoon, &ret.TotalPendingStale,
			&oldestPendingAt,
			&decisionTime.Count, &decisionTime.AverageSeconds,
			&decisionTime.P50Seconds, &decisionTime.P90Seconds, &decisionTime.MaxSeconds,
			&ret.TotalPendingDeadlinePassed, &ret.TotalGranted,
			&ret.TotalProvisioningFailed, &ret.TotalEvaluationFailed,
			&ret.TotalUnroutable, &ret.TotalAwaitingRequester,
			&ret.TotalOverridesGuardrails, &ret.TotalScheduled, &ret.TotalEndedEarly)
		if err != nil {
			return nil, err
		}

		if oldestPendingAt.Valid {
			ret.OldestPendingAt = timestamppb.New(oldestPendingAt.Time)
		}

		if decisionTime.Count > 0 {
			ret.DecisionTime = decisionTime
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	stepDecisionTimes, err := s.getStepDecisionTimes(ctx, filters)
	if err != nil {
		return nil, err
	}
	ret.StepDecisionTimes = stepDecisionTimes

	return ret, nil
}

func (s *Server) getStepDecisionTimes(ctx context.Context,
	filters []exp.Expression) ([]*vaccessv1.StepDecisionTime, error) {
	inner := goqu.From("resources").Where(filters...).
		Select(goqu.L(`unnest(COALESCE(CAST(json_extract(rsc, '$.status.review.steps') AS JSON[]), [])) AS step`))

	seconds := `(epoch(TRY_CAST(json_extract_string(step, '$.endedAt') AS TIMESTAMP)) - epoch(TRY_CAST(json_extract_string(step, '$.startedAt') AS TIMESTAMP)))`
	satisfied := `json_extract_string(step, '$.outcome') = 'SATISFIED'`

	ds := goqu.From(inner.As("steps")).
		Select(
			goqu.L(`COALESCE(json_extract_string(step, '$.name'), '')`).As("name"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s)`, satisfied)),
			goqu.L(fmt.Sprintf(`COALESCE(AVG(%s) FILTER (WHERE %s), 0)`, seconds, satisfied)),
			goqu.L(fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.5) FILTER (WHERE %s), 0)`, seconds, satisfied)),
			goqu.L(fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.9) FILTER (WHERE %s), 0)`, seconds, satisfied)),
			goqu.L(fmt.Sprintf(`COALESCE(MAX(%s) FILTER (WHERE %s), 0)`, seconds, satisfied)),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract_string(step, '$.outcome') = 'TIMED_OUT')`),
			goqu.L(`COUNT(*) FILTER (WHERE COALESCE(TRY_CAST(json_extract_string(step, '$.isEscalated') AS BOOLEAN), false))`),
		).
		GroupBy(goqu.L("name")).
		Order(goqu.L("name").Asc())

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var ret []*vaccessv1.StepDecisionTime
	for rows.Next() {
		itm := &vaccessv1.StepDecisionTime{
			DecisionTime: &vaccessv1.DecisionTime{},
		}

		if err := rows.Scan(&itm.StepName, &itm.DecisionTime.Count, &itm.DecisionTime.AverageSeconds,
			&itm.DecisionTime.P50Seconds, &itm.DecisionTime.P90Seconds, &itm.DecisionTime.MaxSeconds,
			&itm.TotalTimedOut, &itm.TotalEscalated); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		if itm.DecisionTime.Count == 0 {
			itm.DecisionTime = nil
		}

		ret = append(ret, itm)
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessReview(ctx context.Context, common *vmetav1.CommonSummaryOptions,
	refFilters []exp.Expression) (*vaccessv1.GetReviewSummaryResponse, error) {

	ret := &vaccessv1.GetReviewSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindReview, common)
	if err != nil {
		return nil, err
	}
	filters = append(filters, refFilters...)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE (%s IS NULL) OR (%s = 'DECISION_UNSET')) AS count_pending`, colSpecDecision, colSpecDecision)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DECISION_APPROVE') AS count_approved`, colSpecDecision)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DECISION_REJECT') AS count_rejected`, colSpecDecision)),
			goqu.L(`COUNT(*) FILTER (WHERE json_array_length(rsc, '$.status.lastRevisions') > 0) AS count_revised`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_request`, colRequestUID)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'INTEGRATION') AS count_from_integration`, colStatusOriginType)),
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
			&ret.TotalUser, &ret.TotalRequest, &ret.TotalFromIntegration)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessSecret(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetSecretSummaryResponse, error) {

	ret := &vaccessv1.GetSecretSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindSecret, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
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
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessIntegration(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetIntegrationSummaryResponse, error) {

	ret := &vaccessv1.GetIntegrationSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindIntegration, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SLACK') AS count_slack`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'JIRA') AS count_jira`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'WEBHOOK') AS count_webhook`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'READY') AS count_ready`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DEGRADED') AS count_degraded`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ERROR') AS count_error`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SYNCING') AS count_synchronizing`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS') AS count_sync_success`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILED') AS count_sync_failed`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'NOTIFICATION')) AS count_notification`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'DIRECT_USER_DELIVERY')) AS count_direct_user_delivery`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'INTERACTIVE_REVIEW')) AS count_interactive_review`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'REQUEST_CREATION')) AS count_request_creation`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'IDENTITY_RESOLUTION')) AS count_identity_resolution`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'PRESENTATION_UPDATE')) AS count_presentation_update`, colStatusCapabilities)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_external_tenant`, colStatusExternalTenantID)),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalDisabled,
			&ret.TotalSlack, &ret.TotalJira, &ret.TotalWebhook,
			&ret.TotalReady, &ret.TotalDegraded, &ret.TotalError,
			&ret.TotalSynchronizing, &ret.TotalSynchronizationSuccess, &ret.TotalSynchronizationFailed,
			&ret.TotalNotification, &ret.TotalDirectUserDelivery, &ret.TotalInteractiveReview,
			&ret.TotalRequestCreation, &ret.TotalIdentityResolution, &ret.TotalPresentationUpdate,
			&ret.TotalExternalTenant)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessIntegrationIdentity(ctx context.Context, common *vmetav1.CommonSummaryOptions,
	refFilters []exp.Expression) (*vaccessv1.GetIntegrationIdentitySummaryResponse, error) {

	ret := &vaccessv1.GetIntegrationIdentitySummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindIntegrationIdentity, common)
	if err != nil {
		return nil, err
	}
	filters = append(filters, refFilters...)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'EMAIL_DISCOVERY') AS count_email_discovery`, colStatusSource)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_integration`, colIntegrationUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalEmailDiscovery,
			&ret.TotalIntegration, &ret.TotalUser)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryAccessIntegrationBinding(ctx context.Context, common *vmetav1.CommonSummaryOptions,
	refFilters []exp.Expression) (*vaccessv1.GetIntegrationBindingSummaryResponse, error) {

	ret := &vaccessv1.GetIntegrationBindingSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindIntegrationBinding, common)
	if err != nil {
		return nil, err
	}
	filters = append(filters, refFilters...)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING') AS count_pending`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'READY') AS count_ready`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DEGRADED') AS count_degraded`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'CLOSED') AS count_closed`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REVIEW_SURFACE') AS count_review_surface`, colStatusPurpose)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'NOTIFICATION') AS count_notification`, colStatusPurpose)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SHARED') AS count_shared`, colStatusAudience)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REVIEWERS') AS count_reviewers`, colStatusAudience)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REQUESTER') AS count_requester`, colStatusAudience)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUBJECT') AS count_subject`, colStatusAudience)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'INTERACTIVE') AS count_interactive`, colStatusInteractionMode)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE COALESCE(%s, '') != COALESCE(%s, '')) AS count_out_of_date`, colStatusDesiredRevision, colStatusAppliedRevision)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s > 0) AS count_failing`, colStatusAttempts)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_integration`, colIntegrationUID)),
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
			&ret.TotalPending, &ret.TotalReady, &ret.TotalDegraded, &ret.TotalClosed,
			&ret.TotalReviewSurface, &ret.TotalNotification,
			&ret.TotalShared, &ret.TotalReviewers, &ret.TotalRequester, &ret.TotalSubject,
			&ret.TotalInteractive,
			&ret.TotalOutOfDate, &ret.TotalFailing,
			&ret.TotalIntegration, &ret.TotalRequest)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryAccessPolicy(ctx context.Context, req *vaccessv1.GetPolicySummaryRequest) (*vaccessv1.GetPolicySummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessPolicy)
}

func (s *Server) getSummaryAccessCatalog(ctx context.Context, req *vaccessv1.GetCatalogSummaryRequest) (*vaccessv1.GetCatalogSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessCatalog)
}

func (s *Server) getSummaryAccessRequest(ctx context.Context, req *vaccessv1.GetRequestSummaryRequest) (*vaccessv1.GetRequestSummaryResponse, error) {
	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.GetPolicyRef(), pth: "status.policyRef"},
		{ref: req.GetCatalogRef(), pth: "spec.resource.catalog.catalogRef"},
		{ref: req.GetUserRef(), pth: "status.userRef"},
		{ref: req.GetSubjectUserRef(), pth: "spec.subject.userRef"},
	})
	if err != nil {
		return nil, err
	}

	refFilters, err = appendRequestScopeServiceFilter(refFilters, req.GetServiceRef())
	if err != nil {
		return nil, err
	}

	o := &requestSummaryOpts{
		refFilters:     refFilters,
		staleAfter:     getDurationOr(req.GetStaleAfter(), requestPendingStaleDuration),
		expiringWithin: getDurationOr(req.GetExpiringWithin(), requestExpiringSoonDuration),
	}

	return withSummaryComparison(ctx, req.GetCommon(),
		func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetRequestSummaryResponse, error) {
			return s.doSummaryAccessRequest(ctx, common, o)
		})
}

func (s *Server) getSummaryAccessReview(ctx context.Context, req *vaccessv1.GetReviewSummaryRequest) (*vaccessv1.GetReviewSummaryResponse, error) {
	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.GetUserRef(), pth: "status.userRef"},
		{ref: req.GetRequestRef(), pth: "status.requestRef"},
	})
	if err != nil {
		return nil, err
	}

	return withSummaryComparison(ctx, req.GetCommon(),
		func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetReviewSummaryResponse, error) {
			return s.doSummaryAccessReview(ctx, common, refFilters)
		})
}

func (s *Server) getSummaryAccessSecret(ctx context.Context, req *vaccessv1.GetSecretSummaryRequest) (*vaccessv1.GetSecretSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessSecret)
}

func (s *Server) getSummaryAccessIntegration(ctx context.Context, req *vaccessv1.GetIntegrationSummaryRequest) (*vaccessv1.GetIntegrationSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessIntegration)
}

func (s *Server) getSummaryAccessIntegrationIdentity(ctx context.Context, req *vaccessv1.GetIntegrationIdentitySummaryRequest) (*vaccessv1.GetIntegrationIdentitySummaryResponse, error) {
	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.GetIntegrationRef(), pth: "status.integrationRef"},
	})
	if err != nil {
		return nil, err
	}

	return withSummaryComparison(ctx, req.GetCommon(),
		func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetIntegrationIdentitySummaryResponse, error) {
			return s.doSummaryAccessIntegrationIdentity(ctx, common, refFilters)
		})
}

func (s *Server) getSummaryAccessIntegrationBinding(ctx context.Context, req *vaccessv1.GetIntegrationBindingSummaryRequest) (*vaccessv1.GetIntegrationBindingSummaryResponse, error) {
	refFilters, err := getAccessRefFilters([]*accessRefFilter{
		{ref: req.GetIntegrationRef(), pth: "status.integrationRef"},
		{ref: req.GetRequestRef(), pth: "status.requestRef"},
	})
	if err != nil {
		return nil, err
	}

	return withSummaryComparison(ctx, req.GetCommon(),
		func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetIntegrationBindingSummaryResponse, error) {
			return s.doSummaryAccessIntegrationBinding(ctx, common, refFilters)
		})
}

type accessRefFilter struct {
	ref  *metav1.ObjectReference
	pth  string
	opts *apivalidation.CheckGetOptionsOpts
}

func getAccessRefFilters(items []*accessRefFilter) ([]exp.Expression, error) {
	var ret []exp.Expression
	var err error

	for _, itm := range items {
		ret, err = appendRefFilter(ret, itm.ref, itm.opts, itm.pth)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func serviceRefCheckOpts() *apivalidation.CheckGetOptionsOpts {
	return &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 1,
	}
}

const requestExpiringSoonDuration = time.Hour
const requestPendingStaleDuration = 24 * time.Hour

func getRequestDecidedExpr() string {
	return fmt.Sprintf(`(%s = 'REVIEW' AND %s IS NOT NULL AND %s IS NOT NULL AND (%s IN ('APPROVED', 'REJECTED') OR list_has_any(COALESCE(TRY_CAST(json_extract_string(rsc, '$.status.lastStates[*].status') AS VARCHAR[]), []), ['APPROVED', 'REJECTED'])))`,
		colStatusRuleEffect, colStatusApprovalStartAt, colStatusApprovalEndAt, colStatusStateStatus)
}

func getRequestGrantedExpr() string {
	return fmt.Sprintf(`(%s = 'APPROVED' OR list_contains(COALESCE(TRY_CAST(json_extract_string(rsc, '$.status.lastStates[*].status') AS VARCHAR[]), []), 'APPROVED'))`,
		colStatusStateStatus)
}

func getRequestUnmatchedExpr() string {
	return fmt.Sprintf(`(%s = 'REJECTED' AND %s IS NULL)`, colStatusStateStatus, colStatusRuleEffect)
}

func getRequestOnBehalfExpr() string {
	return fmt.Sprintf(`(%s IS NOT NULL AND %s IS NOT NULL AND %s != %s)`,
		colSpecSubjectUserUID, colUserUID, colSpecSubjectUserUID, colUserUID)
}

func getRequestExpiringSoonExpr() string {
	return fmt.Sprintf(`(%s = 'APPROVED' AND %s > ? AND %s <= ?)`,
		colStatusStateStatus, colStatusAccessEndsAt, colStatusAccessEndsAt)
}

func getRequestUnroutableExpr() string {
	return fmt.Sprintf(`(%s = 'PENDING' AND COALESCE(TRY_CAST(json_extract_string(rsc, '$.status.review.steps[#-1].isUnroutable') AS BOOLEAN), false) AND COALESCE(json_extract_string(rsc, '$.status.review.steps[#-1].outcome'), 'OUTCOME_UNSET') = 'OUTCOME_UNSET')`,
		colStatusStateStatus)
}

func getRequestAwaitingRequesterExpr() string {
	return fmt.Sprintf(`(%s = 'PENDING' AND json_extract(rsc, '$.status.review.informationRequests[#-1].requestedAt') IS NOT NULL AND json_extract(rsc, '$.status.review.informationRequests[#-1].respondedAt') IS NULL)`,
		colStatusStateStatus)
}

func appendRequestScopeServiceFilter(filters []exp.Expression, ref *metav1.ObjectReference) ([]exp.Expression, error) {
	if ref == nil {
		return filters, nil
	}

	if err := apivalidation.CheckObjectRef(ref, serviceRefCheckOpts()); err != nil {
		return nil, err
	}

	if ref.Uid != "" {
		filters = append(filters, goqu.L(fmt.Sprintf(`(list_contains(COALESCE(%s, []), ?) OR %s = ?)`,
			colStatusScopeServiceUIDs, colSpecServiceUID), ref.Uid, ref.Uid))
	}

	if ref.Name != "" {
		name := vutils.GetServiceFullNameFromName(ref.Name)
		filters = append(filters, goqu.L(fmt.Sprintf(`(list_contains(COALESCE(%s, []), ?) OR %s = ?)`,
			colStatusScopeServiceNames, colSpecServiceName), name, name))
	}

	return filters, nil
}

func appendRequestScopeNamespaceFilter(filters []exp.Expression, ref *metav1.ObjectReference) ([]exp.Expression, error) {
	if ref == nil {
		return filters, nil
	}

	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	if ref.Name != "" {
		return append(filters, goqu.L(fmt.Sprintf(`(list_contains(COALESCE(%s, []), ?) OR len(list_filter(COALESCE(%s, []), lambda x : suffix(x, ?))) > 0)`,
			colStatusScopeNamespaceNames, colStatusScopeServiceNames), ref.Name, "."+ref.Name)), nil
	}

	return append(filters, goqu.L(fmt.Sprintf(`list_contains(COALESCE(%s, []), ?)`,
		colStatusScopeNamespaceUIDs), ref.Uid)), nil
}

func getCatalogOwnerFilter(ref *metav1.ObjectReference) (exp.Expression, error) {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	userFilter := goqu.L(`uid`).Eq(ref.Uid)
	if ref.Uid == "" {
		userFilter = goqu.L(colName).Eq(ref.Name)
	}

	userDS := goqu.From("resources").Where(
		goqu.L(`kind`).Eq(ucorev1.KindUser),
		goqu.L(`api`).Eq(ucorev1.API),
		userFilter,
	).Limit(1)

	namesSQL, _, err := userDS.Select(goqu.L(colName)).ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	groupsSQL, _, err := userDS.Select(goqu.L(fmt.Sprintf(`COALESCE(%s, [])`, colSpecGroups))).ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return goqu.L(fmt.Sprintf(`(list_contains(COALESCE(TRY_CAST(json_extract_string(rsc, '$.spec.owners.users') AS VARCHAR[]), []), (%s)) OR list_has_any(COALESCE(TRY_CAST(json_extract_string(rsc, '$.spec.owners.groups') AS VARCHAR[]), []), (%s)))`,
		namesSQL, groupsSQL)), nil
}

func getDurationOr(d *metav1.Duration, fallback time.Duration) time.Duration {
	if ret := umetav1.ToDuration(d).ToGo(); ret > 0 {
		return ret
	}

	return fallback
}

func getRequestPendingStaleExpr() string {
	return fmt.Sprintf(`(%s = 'PENDING' AND %s < ?)`, colStatusStateStatus, colStatusApprovalStartAt)
}

func getPolicyRuleConditionRefFilter(ref *metav1.ObjectReference, parent, field string) exp.Expression {
	key, val := "uid", ref.Uid
	if val == "" {
		key, val = "name", ref.Name
	}

	pattern := fmt.Sprintf(`"%s"\s*:\s*\{\s*"%s"\s*:\s*\{[^{}]*"%s"\s*:\s*"%s"`,
		parent, field, key, regexp.QuoteMeta(val))

	return goqu.L(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), lambda x : regexp_matches(CAST(json_extract(x, '$.condition') AS VARCHAR), ?))) > 0`,
		pattern)
}
