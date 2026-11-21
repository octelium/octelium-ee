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

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
)

const (
	listPageSize            = 1000
	maxListPages            = 10
	defaultListItemsPerPage = 200
	maxListItemsPerPage     = 1000
)

func listAllRequests(ctx context.Context, octeliumC octeliumc.ClientInterface,
	filters ...*rmetav1.ListOptions_Filter) ([]*accessv1.Request, error) {
	var ret []*accessv1.Request

	for page := uint32(0); page < maxListPages; page++ {
		itemList, err := octeliumC.AccessC().ListRequest(ctx, &rmetav1.ListOptions{
			Filters:      filters,
			Paginate:     true,
			Page:         page,
			ItemsPerPage: listPageSize,
			OrderBy: []*rmetav1.ListOptions_OrderBy{
				{
					Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
					Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, itemList.Items...)

		if itemList.ListResponseMeta == nil || !itemList.ListResponseMeta.HasMore {
			break
		}
	}

	return ret, nil
}

func mergeRequests(lists ...[]*accessv1.Request) []*accessv1.Request {
	seen := map[string]struct{}{}
	var ret []*accessv1.Request

	for _, list := range lists {
		for _, itm := range list {
			if _, ok := seen[itm.Metadata.Uid]; ok {
				continue
			}
			seen[itm.Metadata.Uid] = struct{}{}
			ret = append(ret, itm)
		}
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Metadata.CreatedAt.AsTime().After(ret[j].Metadata.CreatedAt.AsTime())
	})

	return ret
}

func paginateRequests(items []*accessv1.Request, common *metav1.CommonListOptions) *accessv1.RequestList {
	var page, itemsPerPage uint32
	if common != nil {
		page = common.Page
		itemsPerPage = common.ItemsPerPage
	}

	if itemsPerPage == 0 {
		itemsPerPage = defaultListItemsPerPage
	} else if itemsPerPage > maxListItemsPerPage {
		itemsPerPage = maxListItemsPerPage
	}

	if orderBy := common.GetOrderBy(); orderBy != nil {
		isDesc := orderBy.Mode == metav1.CommonListOptions_OrderBy_DESC

		switch orderBy.Type {
		case metav1.CommonListOptions_OrderBy_CREATED_AT:
			if !isDesc {
				sort.SliceStable(items, func(i, j int) bool {
					return items[i].Metadata.CreatedAt.AsTime().Before(items[j].Metadata.CreatedAt.AsTime())
				})
			}
		case metav1.CommonListOptions_OrderBy_NAME:
			sort.SliceStable(items, func(i, j int) bool {
				if isDesc {
					return items[i].Metadata.Name > items[j].Metadata.Name
				}
				return items[i].Metadata.Name < items[j].Metadata.Name
			})
		}
	}

	total := uint32(len(items))
	start := page * itemsPerPage
	end := start + itemsPerPage

	ret := &accessv1.RequestList{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       "RequestList",
		Items:      []*accessv1.Request{},
		ListResponseMeta: &metav1.ListResponseMeta{
			Page:         page,
			ItemsPerPage: itemsPerPage,
			TotalCount:   total,
			HasMore:      end < total,
		},
	}

	if start >= total {
		return ret
	}

	if end > total {
		end = total
	}

	ret.Items = items[start:end]

	return ret
}

func matchesRequestQuery(req *accessv1.Request, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	candidates := []string{
		req.GetMetadata().GetName(),
		req.GetSpec().GetJustification(),
		req.GetSpec().GetTicket(),
		req.GetSpec().GetResource().GetCatalog().GetCatalogRef().GetName(),
		req.GetSpec().GetResource().GetCatalog().GetAccessLevel(),
		req.GetStatus().GetResolvedScope().GetCatalogDisplayName(),
		req.GetStatus().GetRequesterInfo().GetDisplayName(),
		req.GetStatus().GetRequesterInfo().GetEmail(),
		req.GetStatus().GetSubjectInfo().GetDisplayName(),
		req.GetStatus().GetSubjectInfo().GetEmail(),
		req.GetStatus().GetUserRef().GetName(),
		req.GetSpec().GetSubject().GetUserRef().GetName(),
	}

	for _, ref := range req.GetStatus().GetResolvedScope().GetServiceRefs() {
		candidates = append(candidates, ref.Name)
	}

	for _, itm := range candidates {
		if itm != "" && strings.Contains(strings.ToLower(itm), query) {
			return true
		}
	}

	return false
}

func requestIncludesService(req *accessv1.Request, ref *metav1.ObjectReference) bool {
	if ref == nil || (ref.Uid == "" && ref.Name == "") {
		return true
	}

	name := vutils.GetServiceFullNameFromName(ref.Name)

	for _, itm := range req.GetStatus().GetResolvedScope().GetServiceRefs() {
		if (ref.Uid != "" && itm.Uid == ref.Uid) ||
			(ref.Uid == "" && itm.Name == name) {
			return true
		}
	}

	if _, ns, ok := strings.Cut(name, "."); ok {
		for _, itm := range req.GetStatus().GetResolvedScope().GetNamespaceRefs() {
			if itm.Name == ns {
				return true
			}
		}
	}

	return false
}
