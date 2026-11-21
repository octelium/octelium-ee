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

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

const minSubjectUserQueryLen = 2
const maxSubjectUserItemsPerPage = 50

func (s *ServerUser) ListSubjectUser(ctx context.Context, req *accessv1.ListSubjectUserOptions) (*accessv1.SubjectUserList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	query := strings.TrimSpace(req.GetQuery())
	if len(query) < minSubjectUserQueryLen {
		return nil, grpcutils.InvalidArg("Query must be at least %d characters", minSubjectUserQueryLen)
	}

	if err := s.checkCanRequestOnBehalf(ctx, i.User); err != nil {
		return nil, err
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
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
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

	if item.Metadata.Uid != i.User.Metadata.Uid {
		if err := s.checkCanRequestOnBehalf(ctx, i.User); err != nil {
			return nil, err
		}
	}

	return toSubjectUser(item), nil
}

func (s *ServerUser) checkCanRequestOnBehalf(ctx context.Context, usr *corev1.User) error {
	catalogList, err := s.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return serr.InternalWithErr(err)
	}

	for _, catalog := range catalogList.Items {
		if !catalog.GetSpec().GetIsDisabled() && accesscmd.CanRequestOnBehalf(catalog, usr) {
			return nil
		}
	}

	return grpcutils.Unauthorized("You are not allowed to request access on behalf of others")
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
