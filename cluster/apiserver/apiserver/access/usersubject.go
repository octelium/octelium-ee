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
	"strings"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

const minSubjectUserQueryLen = 2
const maxSubjectUserItemsPerPage = 50

func (s *ServerUser) ListSubjectUser(ctx context.Context, req *accessv1.ListSubjectUserOptions) (*accessv1.SubjectUserList, error) {
	if _, err := userctx.GetUserCtx(ctx); err != nil {
		return nil, err
	}

	query := strings.TrimSpace(req.GetQuery())
	if len(query) < minSubjectUserQueryLen {
		return nil, grpcutils.InvalidArg("Query must be at least %d characters", minSubjectUserQueryLen)
	}

	common := req.GetCommon()
	if common == nil {
		common = &metav1.CommonListOptions{}
	}

	itemsPerPage := common.ItemsPerPage
	if itemsPerPage == 0 || itemsPerPage > maxSubjectUserItemsPerPage {
		itemsPerPage = maxSubjectUserItemsPerPage
	}

	itemList, err := s.rscStoreC.ListSubjectUser(ctx, &caccessv1.ListSubjectUserRequest{
		Query:        query,
		Page:         common.Page,
		ItemsPerPage: itemsPerPage,
	})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	ret := &accessv1.SubjectUserList{
		ApiVersion:       uaccessv1.APIVersion,
		Kind:             "SubjectUserList",
		Items:            []*accessv1.SubjectUser{},
		ListResponseMeta: itemList.ListResponseMeta,
	}

	for _, item := range itemList.Items {
		ret.Items = append(ret.Items, toSubjectUser(item))
	}

	return ret, nil
}

func (s *ServerUser) GetSubjectUser(ctx context.Context, req *accessv1.GetSubjectUserRequest) (*accessv1.SubjectUser, error) {
	if _, err := userctx.GetUserCtx(ctx); err != nil {
		return nil, err
	}

	if req == nil || req.UserRef == nil {
		return nil, grpcutils.InvalidArg("UserRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(req.UserRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if item.Metadata.IsSystemHidden {
		return nil, grpcutils.NotFound("User not found")
	}

	return toSubjectUser(item), nil
}

func toSubjectUser(item *corev1.User) *accessv1.SubjectUser {
	return &accessv1.SubjectUser{
		UserRef:     umetav1.GetObjectReference(item),
		DisplayName: item.Metadata.DisplayName,
		PicURL:      item.Metadata.PicURL,
		Email:       item.Spec.Email,
		Type:        item.Spec.Type,
	}
}
