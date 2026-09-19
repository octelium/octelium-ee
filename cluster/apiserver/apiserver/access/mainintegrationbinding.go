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

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
)

func (s *ServerMain) GetIntegrationBinding(ctx context.Context,
	req *metav1.GetOptions) (*accessv1.IntegrationBinding, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegrationBinding(ctx,
		apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListIntegrationBinding(ctx context.Context,
	req *accessv1.ListIntegrationBindingOptions) (*accessv1.IntegrationBindingList, error) {
	if req == nil {
		req = &accessv1.ListIntegrationBindingOptions{}
	}

	var filters []*rmetav1.ListOptions_Filter

	if req.IntegrationRef != nil {
		integration, err := s.getIntegrationRef(ctx, req.IntegrationRef)
		if err != nil {
			return nil, err
		}
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.integrationRef.uid", integration.Metadata.Uid))
	}

	if req.RequestRef != nil {
		if err := apivalidation.CheckObjectRef(req.RequestRef,
			&apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		item, err := s.octeliumC.AccessC().GetRequest(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
		if err != nil {
			return nil, serr.K8sNotFoundOrInternalWithErr(err)
		}

		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", item.Metadata.Uid))
	}

	itemList, err := s.octeliumC.AccessC().ListIntegrationBinding(ctx,
		urscsrv.GetPublicListOptions(req, filters...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}
