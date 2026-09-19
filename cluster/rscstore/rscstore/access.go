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
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
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

func (s *Server) doSummaryAccessRequest(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetRequestSummaryResponse, error) {

	ret := &vaccessv1.GetRequestSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindRequest, common)
	if err != nil {
		return nil, err
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

func (s *Server) doSummaryAccessReview(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetReviewSummaryResponse, error) {

	ret := &vaccessv1.GetReviewSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindReview, common)
	if err != nil {
		return nil, err
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

func (s *Server) doSummaryAccessIntegrationIdentity(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetIntegrationIdentitySummaryResponse, error) {

	ret := &vaccessv1.GetIntegrationIdentitySummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindIntegrationIdentity, common)
	if err != nil {
		return nil, err
	}

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

func (s *Server) doSummaryAccessIntegrationBinding(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vaccessv1.GetIntegrationBindingSummaryResponse, error) {

	ret := &vaccessv1.GetIntegrationBindingSummaryResponse{}
	filters, err := getSummaryFilters(uaccessv1.API, uaccessv1.Version, uaccessv1.KindIntegrationBinding, common)
	if err != nil {
		return nil, err
	}

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
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessRequest)
}

func (s *Server) getSummaryAccessReview(ctx context.Context, req *vaccessv1.GetReviewSummaryRequest) (*vaccessv1.GetReviewSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessReview)
}

func (s *Server) getSummaryAccessSecret(ctx context.Context, req *vaccessv1.GetSecretSummaryRequest) (*vaccessv1.GetSecretSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessSecret)
}

func (s *Server) getSummaryAccessIntegration(ctx context.Context, req *vaccessv1.GetIntegrationSummaryRequest) (*vaccessv1.GetIntegrationSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessIntegration)
}

func (s *Server) getSummaryAccessIntegrationIdentity(ctx context.Context, req *vaccessv1.GetIntegrationIdentitySummaryRequest) (*vaccessv1.GetIntegrationIdentitySummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessIntegrationIdentity)
}

func (s *Server) getSummaryAccessIntegrationBinding(ctx context.Context, req *vaccessv1.GetIntegrationBindingSummaryRequest) (*vaccessv1.GetIntegrationBindingSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryAccessIntegrationBinding)
}
