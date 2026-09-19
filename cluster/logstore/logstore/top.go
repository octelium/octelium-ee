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

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) listAccessLogTopUser(ctx context.Context, req *visibilityv1.ListAccessLogTopUserRequest) (*visibilityv1.ListAccessLogTopUserResponse, error) {

	ret := &visibilityv1.ListAccessLogTopUserResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, colServiceUID, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, colNamespaceUID, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "access_logs", getTopLimit(req.Limit), colUserUID, "entry.common.userRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopUserResponse_Item{
			User:  itm,
			Count: int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAccessLogTopService(ctx context.Context, req *visibilityv1.ListAccessLogTopServiceRequest) (*visibilityv1.ListAccessLogTopServiceResponse, error) {

	ret := &visibilityv1.ListAccessLogTopServiceResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, colUserUID, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, colDeviceUID, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, colSessionUID, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}

	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "access_logs", getTopLimit(req.Limit), colServiceUID, "entry.common.serviceRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopServiceResponse_Item{
			Service: itm,
			Count:   int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAccessLogTopPolicy(ctx context.Context, req *visibilityv1.ListAccessLogTopPolicyRequest) (*visibilityv1.ListAccessLogTopPolicyResponse, error) {

	ret := &visibilityv1.ListAccessLogTopPolicyResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, colUserUID, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, colDeviceUID, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, colSessionUID, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, colServiceUID, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, colNamespaceUID, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "access_logs", getTopLimit(req.Limit), colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopPolicyResponse_Item{
			Policy: itm,
			Count:  int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAccessLogTopSession(ctx context.Context, req *visibilityv1.ListAccessLogTopSessionRequest) (*visibilityv1.ListAccessLogTopSessionResponse, error) {

	ret := &visibilityv1.ListAccessLogTopSessionResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, colUserUID, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, colDeviceUID, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}

	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, colServiceUID, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, colNamespaceUID, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "access_logs", getTopLimit(req.Limit), colSessionUID, "entry.common.sessionRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAccessLogTopSessionResponse_Item{
			Session: itm,
			Count:   int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAuthenticationLogTopUser(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopUserRequest) (*visibilityv1.ListAuthenticationLogTopUserResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopUserResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.IdentityProviderRef, nil, "", "entry.authentication.info.identityProvider.identityProviderRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.AuthenticatorRef, nil, "", "entry.authentication.info.authenticator.authenticatorRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.CredentialRef, nil, "", "entry.authentication.info.credential.credentialRef")
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "authentication_logs", getTopLimit(req.Limit), "", "entry.userRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAuthenticationLogTopUserResponse_Item{
			User:  itm,
			Count: int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAuthenticationLogTopCredential(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopCredentialRequest) (*visibilityv1.ListAuthenticationLogTopCredentialResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopCredentialResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, "", "entry.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, "", "entry.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, "", "entry.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.IdentityProviderRef, nil, "", "entry.authentication.info.identityProvider.identityProviderRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.AuthenticatorRef, nil, "", "entry.authentication.info.authenticator.authenticatorRef")
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "authentication_logs", getTopLimit(req.Limit), "", "entry.authentication.info.credential.credentialRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetCredential(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAuthenticationLogTopCredentialResponse_Item{
			Credential: itm,
			Count:      int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAuthenticationLogTopIdentityProvider(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopIdentityProviderRequest) (*visibilityv1.ListAuthenticationLogTopIdentityProviderResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopIdentityProviderResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, "", "entry.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, "", "entry.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, "", "entry.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.IdentityProviderRef, nil, "", "entry.authentication.info.identityProvider.identityProviderRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.AuthenticatorRef, nil, "", "entry.authentication.info.authenticator.authenticatorRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.CredentialRef, nil, "", "entry.authentication.info.credential.credentialRef")
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "authentication_logs", getTopLimit(req.Limit), "", "entry.identityProviderRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetIdentityProvider(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAuthenticationLogTopIdentityProviderResponse_Item{
			IdentityProvider: itm,
			Count:            int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAuditLogTopUser(ctx context.Context, req *visibilityv1.ListAuditLogTopUserRequest) (*visibilityv1.ListAuditLogTopUserResponse, error) {

	ret := &visibilityv1.ListAuditLogTopUserResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.ResourceRef, nil, "", "entry.resourceRef")
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "audit_logs", getTopLimit(req.Limit), "", "entry.userRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAuditLogTopUserResponse_Item{
			User:  itm,
			Count: int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

func (s *Server) listAuditLogTopSession(ctx context.Context, req *visibilityv1.ListAuditLogTopSessionRequest) (*visibilityv1.ListAuditLogTopSessionResponse, error) {

	ret := &visibilityv1.ListAuditLogTopSessionResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.ResourceRef, nil, "", "entry.resourceRef")
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	res, err := s.getTop(ctx, "audit_logs", getTopLimit(req.Limit), "", "entry.sessionRef", filters)
	if err != nil {
		return nil, err
	}

	var returned uint64

	for _, item := range res.items {
		itm, err := s.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
			Uid: item.UID,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &visibilityv1.ListAuditLogTopSessionResponse_Item{
			Session: itm,
			Count:   int32(item.Count),
		})

		returned = returned + uint64(item.Count)
	}

	ret.TotalCount = res.totalCount
	ret.TotalOther = res.other(returned)

	return ret, nil
}

type getTopResult struct {
	items []*getTopResultItem
	// totalCount is the number of the distinct keys that matched.
	totalCount uint64
	// totalSum is the sum of the counts of every matching key.
	totalSum uint64
}

// other returns the sum of the counts of the matching keys that the caller did
// not return to the client.
func (r *getTopResult) other(returned uint64) uint64 {
	if r.totalSum <= returned {
		return 0
	}

	return r.totalSum - returned
}

type getTopResultItem struct {
	UID   string
	Count int
}

func (s *Server) getTopTotals(ctx context.Context,
	table, expr string, filters []exp.Expression) (uint64, uint64, error) {

	inner := goqu.Dialect("postgres").From(table).
		Select(goqu.L(expr).As("uid"), goqu.L("COUNT(*)").As("count")).
		Where(goqu.L(fmt.Sprintf("%s IS NOT NULL", expr))).
		Where(filters...).
		GroupBy(goqu.L(expr))

	ds := goqu.Dialect("postgres").From(inner.As("grouped")).
		Select(
			goqu.L(`COUNT(*)`).As("total_count"),
			goqu.L(`COALESCE(SUM(count), 0)`).As("total_sum"),
		)

	query, args, err := ds.ToSQL()
	if err != nil {
		return 0, 0, err
	}

	var totalCount, totalSum uint64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&totalCount, &totalSum); err != nil {
		return 0, 0, err
	}

	return totalCount, totalSum, nil
}

func (s *Server) getTop(ctx context.Context, table string, n int,
	column, field string, filters []exp.Expression) (*getTopResult, error) {
	dialect := goqu.Dialect("postgres")

	expr := fmt.Sprintf("json_extract_string(rsc, '$.%s.uid')", field)
	if column != "" {
		expr = column
	}

	ds := dialect.From(table).
		Select(
			goqu.L(expr).As("uid"),
			goqu.L("COUNT(*)").As("count"),
		).
		Where(goqu.L(fmt.Sprintf("%s IS NOT NULL", expr))).
		Where(filters...).
		GroupBy(goqu.L(expr)).
		Order(goqu.L("count").Desc()).
		Limit(uint(n))

	query, args, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := &getTopResult{}
	for rows.Next() {
		item := &getTopResultItem{}
		if err := rows.Scan(&item.UID, &item.Count); err != nil {
			return nil, err
		}

		ret.items = append(ret.items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ret.totalCount, ret.totalSum, err = s.getTopTotals(ctx, table, expr, filters)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Server) listAccessLogTopDenyReason(ctx context.Context,
	req *visibilityv1.ListAccessLogTopDenyReasonRequest) (*visibilityv1.ListAccessLogTopDenyReasonResponse, error) {

	ret := &visibilityv1.ListAccessLogTopDenyReasonResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, colUserUID, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, colDeviceUID, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, colSessionUID, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, colServiceUID, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, colNamespaceUID, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendTimeFilters(filters, req.From, req.To)

	filters = append(filters, goqu.L(colStatus).Eq(corev1.AccessLog_Entry_Common_DENIED.String()))

	limit := getTopLimit(req.Limit)

	ds := goqu.Dialect("postgres").From("access_logs").
		Select(
			goqu.L(jsonDenyReason).As("reason"),
			goqu.L(`COUNT(*)`).As("count"),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, colPolicyUID)).As("count_policy"),
			goqu.L(fmt.Sprintf(`MIN(%s)`, colPolicyUID)).As("policy_uid"),
		).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, jsonDenyReason))).
		Where(filters...).
		GroupBy(goqu.L("reason")).
		Order(goqu.L("count").Desc()).
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

	var returned uint64

	for rows.Next() {
		var reason string
		var count uint64
		var countPolicy uint64
		var policyUID *string

		if err := rows.Scan(&reason, &count, &countPolicy, &policyUID); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		item := &visibilityv1.ListAccessLogTopDenyReasonResponse_Item{
			Reason: corev1.AccessLog_Entry_Common_Reason_Type(
				corev1.AccessLog_Entry_Common_Reason_Type_value[reason]),
			Count: count,
		}

		if countPolicy == 1 && policyUID != nil && *policyUID != "" {
			item.PolicyRef = &metav1.ObjectReference{
				ApiVersion: "core/v1",
				Kind:       "Policy",
				Uid:        *policyUID,
				Name:       s.resolveDisplayName(ctx, "Policy", *policyUID),
			}
		}

		ret.Items = append(ret.Items, item)
		returned = returned + count
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	totalCount, totalSum, err := s.getTopTotals(ctx, "access_logs", jsonDenyReason, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totalCount
	if totalSum > returned {
		ret.TotalOther = totalSum - returned
	}

	return ret, nil
}

func (s *Server) listComponentLogTopComponent(ctx context.Context,
	req *visibilityv1.ListComponentLogTopComponentRequest) (*visibilityv1.ListComponentLogTopComponentResponse, error) {

	ret := &visibilityv1.ListComponentLogTopComponentResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendComponentFilter(filters, req.Component)
	if err != nil {
		return nil, err
	}

	if req.Level != corev1.ComponentLog_Entry_LEVEL_UNSET {
		filters = append(filters, goqu.L(colLevel).Eq(req.Level.String()))
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	limit := getTopLimit(req.Limit)

	ds := goqu.Dialect("postgres").From("component_logs").
		Select(
			goqu.L(jsonComponentNamespace).As("namespace"),
			goqu.L(jsonComponentType).As("type"),
			goqu.L(`COUNT(*)`).As("count"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'WARN')`, colLevel)).As("count_warn"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ERROR')`, colLevel)).As("count_error"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PANIC')`, colLevel)).As("count_panic"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FATAL')`, colLevel)).As("count_fatal"),
		).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, jsonComponentType))).
		Where(filters...).
		GroupBy(goqu.L("namespace"), goqu.L("type")).
		Order(goqu.L("count").Desc()).
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

	var returned uint64

	for rows.Next() {
		var namespace *string
		var componentType string
		item := &visibilityv1.ListComponentLogTopComponentResponse_Item{}

		if err := rows.Scan(&namespace, &componentType, &item.Count,
			&item.CountWarn, &item.CountError, &item.CountPanic, &item.CountFatal); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		item.Component = &visibilityv1.ComponentSelector{
			Type: componentType,
		}
		if namespace != nil {
			item.Component.Namespace = *namespace
		}

		ret.Items = append(ret.Items, item)
		returned = returned + item.Count
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	totalCount, totalSum, err := s.getTopTotals(ctx, "component_logs", sqlComponentKey, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totalCount
	if totalSum > returned {
		ret.TotalOther = totalSum - returned
	}

	return ret, nil
}
