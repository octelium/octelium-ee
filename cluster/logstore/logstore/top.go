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
	"github.com/octelium/octelium/cluster/common/apivalidation"
)

func (s *Server) listAccessLogTopUser(ctx context.Context, req *visibilityv1.ListAccessLogTopUserRequest) (*visibilityv1.ListAccessLogTopUserResponse, error) {

	ret := &visibilityv1.ListAccessLogTopUserResponse{}

	var filters []exp.Expression
	var err error

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "access_logs", 10, "entry.common.userRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAccessLogTopService(ctx context.Context, req *visibilityv1.ListAccessLogTopServiceRequest) (*visibilityv1.ListAccessLogTopServiceResponse, error) {

	ret := &visibilityv1.ListAccessLogTopServiceResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "access_logs", 10, "entry.common.serviceRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAccessLogTopPolicy(ctx context.Context, req *visibilityv1.ListAccessLogTopPolicyRequest) (*visibilityv1.ListAccessLogTopPolicyResponse, error) {

	ret := &visibilityv1.ListAccessLogTopPolicyResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "access_logs", 10, "entry.common.reason.details.policyMatch.policy.policyRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAccessLogTopSession(ctx context.Context, req *visibilityv1.ListAccessLogTopSessionRequest) (*visibilityv1.ListAccessLogTopSessionResponse, error) {

	ret := &visibilityv1.ListAccessLogTopSessionResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "access_logs", 10, "entry.common.sessionRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAuthenticationLogTopUser(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopUserRequest) (*visibilityv1.ListAuthenticationLogTopUserResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopUserResponse{}

	var filters []exp.Expression
	var err error

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "authentication_logs", 10, "entry.userRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAuthenticationLogTopCredential(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopCredentialRequest) (*visibilityv1.ListAuthenticationLogTopCredentialResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopCredentialResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "authentication_logs", 10, "entry.credentialRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

func (s *Server) listAuthenticationLogTopIdentityProvider(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopIdentityProviderRequest) (*visibilityv1.ListAuthenticationLogTopIdentityProviderResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopIdentityProviderResponse{}

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

	if req.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Gte(req.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).Lte(req.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	res, err := s.getTop(ctx, "authentication_logs", 10, "entry.identityProviderRef", filters)
	if err != nil {
		return nil, err
	}

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
	}

	return ret, nil
}

type getTopResult struct {
	items []*getTopResultItem
}

type getTopResultItem struct {
	UID   string
	Count int
}

func (s *Server) getTop(ctx context.Context, table string, n int, field string, filters []exp.Expression) (*getTopResult, error) {
	dialect := goqu.Dialect("postgres")

	ds := dialect.From(table).
		Select(
			goqu.L(fmt.Sprintf("json_extract_string(rsc, '$.%s.uid')", field)).As("uid"),
			goqu.L("COUNT(*)").As("count"),
		).
		Where(goqu.L(fmt.Sprintf("json_extract_string(rsc, '$.%s.uid') IS NOT NULL", field))).
		Where(filters...).
		GroupBy(goqu.L(fmt.Sprintf("json_extract_string(rsc, '$.%s.uid')", field))).
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

	return ret, rows.Err()
}
