// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func (s *ServerUser) ListCatalog(ctx context.Context, req *accessv1.ListUserCatalogOptions) (*accessv1.UserCatalogList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	catalogs, err := s.listRequestableCatalogs(ctx, i.User)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.GetQuery()))

	var filtered []*accessv1.Catalog
	for _, catalog := range catalogs {
		if query != "" && !catalogMatchesQuery(catalog, query) {
			continue
		}
		filtered = append(filtered, catalog)
	}

	viewers, err := s.getCatalogViewers(ctx, i.User)
	if err != nil {
		return nil, err
	}

	var page, itemsPerPage uint32
	if common := req.GetCommon(); common != nil {
		page = common.Page
		itemsPerPage = common.ItemsPerPage
	}
	if itemsPerPage == 0 {
		itemsPerPage = defaultListItemsPerPage
	} else if itemsPerPage > maxListItemsPerPage {
		itemsPerPage = maxListItemsPerPage
	}

	total := uint32(len(filtered))
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > total {
		end = total
	}

	ret := &accessv1.UserCatalogList{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       "UserCatalogList",
		Items:      []*accessv1.UserCatalog{},
		ListResponseMeta: &metav1.ListResponseMeta{
			Page:         page,
			ItemsPerPage: itemsPerPage,
			TotalCount:   total,
			HasMore:      start+itemsPerPage < total,
		},
	}

	for idx := start; idx < end; idx++ {
		catalog := filtered[idx]

		viewer := viewers[catalog.Metadata.Uid]
		if viewer == nil {
			viewer = &accessv1.UserCatalog_Viewer{}
		}
		viewer.CanRequestOnBehalf = accesscmd.CanRequestOnBehalf(catalog, i.User)

		itm, err := toUserCatalog(ctx, s.octeliumC, catalog, viewer)
		if err != nil {
			return nil, err
		}

		ret.Items = append(ret.Items, itm)
	}

	return ret, nil
}

func (s *ServerUser) GetCatalog(ctx context.Context, req *metav1.GetOptions) (*accessv1.UserCatalog, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	catalog, err := s.getRequestableCatalog(ctx, i.User, &metav1.ObjectReference{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, err
	}

	viewers, err := s.getCatalogViewers(ctx, i.User)
	if err != nil {
		return nil, err
	}

	viewer := viewers[catalog.Metadata.Uid]
	if viewer == nil {
		viewer = &accessv1.UserCatalog_Viewer{}
	}
	viewer.CanRequestOnBehalf = accesscmd.CanRequestOnBehalf(catalog, i.User)

	return toUserCatalog(ctx, s.octeliumC, catalog, viewer)
}

func (s *ServerUser) ListCatalogService(ctx context.Context,
	req *accessv1.ListUserCatalogServiceOptions) (*userv1.ServiceList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req.CatalogRef == nil {
		return nil, grpcutils.InvalidArg("CatalogRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.CatalogRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	catalog, err := s.getRequestableCatalog(ctx, i.User, req.CatalogRef)
	if err != nil {
		return nil, err
	}

	cc, err := accesscmd.GetClusterConfig(ctx, s.octeliumC)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	scope, problems, err := accesscmd.ResolveScope(ctx, s.octeliumC, &accesscmd.ResolveScopeOpts{
		Catalog:         catalog,
		DefaultPriority: accesscmd.GetDefaultAccessPriority(cc),
	})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	var items []*corev1.Service
	if len(problems) == 0 && scope != nil {
		items = scope.Services
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Metadata.Name < items[j].Metadata.Name
	})

	var page, itemsPerPage uint32
	if common := req.GetCommon(); common != nil {
		page = common.Page
		itemsPerPage = common.ItemsPerPage
	}
	if itemsPerPage == 0 {
		itemsPerPage = defaultListItemsPerPage
	} else if itemsPerPage > maxListItemsPerPage {
		itemsPerPage = maxListItemsPerPage
	}

	total := uint32(len(items))
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > total {
		end = total
	}

	ret := &userv1.ServiceList{
		ApiVersion: "user/v1",
		Kind:       "ServiceList",
		Items:      []*userv1.Service{},
		ListResponseMeta: &metav1.ListResponseMeta{
			Page:         page,
			ItemsPerPage: itemsPerPage,
			TotalCount:   total,
			HasMore:      start+itemsPerPage < total,
		},
	}

	for idx := start; idx < end; idx++ {
		ret.Items = append(ret.Items, toUserCatalogService(items[idx]))
	}

	return ret, nil
}

func (s *ServerUser) listRequestableCatalogs(ctx context.Context, usr *corev1.User) ([]*accessv1.Catalog, error) {
	celEngine, err := s.cel.get(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	catalogList, err := s.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldBooleanFalse("metadata.isSystemHidden"),
		},
		OrderBy: []*rmetav1.ListOptions_OrderBy{
			{
				Type: rmetav1.ListOptions_OrderBy_TYPE_NAME,
				Mode: rmetav1.ListOptions_OrderBy_MODE_ASC,
			},
		},
	})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	var ret []*accessv1.Catalog

	for _, catalog := range catalogList.Items {
		if catalog.GetSpec().GetIsDisabled() || len(catalog.GetSpec().GetAccessLevels()) == 0 {
			continue
		}

		ok, err := accesscmd.IsInAudience(ctx, celEngine, catalog, usr)
		if err != nil {
			ok = false
		}

		if !ok && !accesscmd.CanRequestOnBehalf(catalog, usr) {
			continue
		}

		ret = append(ret, catalog)
	}

	return ret, nil
}

func (s *ServerUser) getRequestableCatalog(ctx context.Context,
	usr *corev1.User, ref *metav1.ObjectReference) (*accessv1.Catalog, error) {
	catalog, err := accesscmd.GetCatalog(ctx, s.octeliumC, ref)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if catalog == nil || catalog.Metadata.IsSystemHidden {
		return nil, grpcutils.NotFound("The Catalog does not exist")
	}

	celEngine, err := s.cel.get(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ok, err := accesscmd.IsInAudience(ctx, celEngine, catalog, usr)
	if err != nil {
		ok = false
	}

	if !ok && !accesscmd.CanRequestOnBehalf(catalog, usr) {
		return nil, grpcutils.NotFound("The Catalog does not exist")
	}

	return catalog, nil
}

func (s *ServerUser) getCatalogViewers(ctx context.Context,
	usr *corev1.User) (map[string]*accessv1.UserCatalog_Viewer, error) {
	itemList, err := listAllRequests(ctx, s.octeliumC,
		urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", usr.Metadata.Uid))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	now := time.Now()
	ret := map[string]*accessv1.UserCatalog_Viewer{}

	for _, itm := range itemList {
		ref := itm.Spec.GetResource().GetCatalog().GetCatalogRef()
		if ref == nil || ref.Uid == "" {
			continue
		}

		viewer := ret[ref.Uid]
		if viewer == nil {
			viewer = &accessv1.UserCatalog_Viewer{}
			ret[ref.Uid] = viewer
		}

		switch {
		case accesscmd.IsPending(itm) && viewer.PendingRequestRef == nil:
			viewer.PendingRequestRef = umetav1.GetObjectReference(itm)
		case accesscmd.IsAccessActive(itm, now) &&
			(viewer.AccessEndsAt == nil || itm.Status.AccessEndsAt.AsTime().After(viewer.AccessEndsAt.AsTime())):
			viewer.ActiveRequestRef = umetav1.GetObjectReference(itm)
			viewer.AccessEndsAt = itm.Status.AccessEndsAt
		}
	}

	return ret, nil
}

func catalogMatchesQuery(catalog *accessv1.Catalog, query string) bool {
	candidates := []string{
		catalog.Metadata.Name,
		catalog.Metadata.DisplayName,
		catalog.Metadata.Description,
	}

	if coll := catalog.GetSpec().GetResourceCollection().GetService(); coll != nil {
		candidates = append(candidates, coll.Services...)
		candidates = append(candidates, coll.Namespaces...)
	}

	for _, itm := range candidates {
		if itm != "" && strings.Contains(strings.ToLower(itm), query) {
			return true
		}
	}

	return false
}

func toUserCatalogService(svc *corev1.Service) *userv1.Service {
	return &userv1.Service{
		ApiVersion: "user/v1",
		Kind:       "Service",
		Metadata:   pbutils.Clone(svc.Metadata).(*metav1.Metadata),
		Spec: &userv1.Service_Spec{
			Type: userv1.Service_Spec_Type(svc.Spec.Mode),
			Port: uint32(ucorev1.ToService(svc).RealPort()),
		},
		Status: &userv1.Service_Status{
			Namespace: func() string {
				if svc.Status.NamespaceRef != nil {
					return svc.Status.NamespaceRef.Name
				}
				return ""
			}(),
		},
	}
}
