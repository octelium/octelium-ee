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
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/registry"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerMain) GetIntegrationIdentity(ctx context.Context,
	req *metav1.GetOptions) (*accessv1.IntegrationIdentity, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegrationIdentity(ctx,
		apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListIntegrationIdentity(ctx context.Context,
	req *accessv1.ListIntegrationIdentityOptions) (*accessv1.IntegrationIdentityList, error) {
	if req == nil {
		req = &accessv1.ListIntegrationIdentityOptions{}
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

	if req.UserRef != nil {
		if err := apivalidation.CheckObjectRef(req.UserRef,
			&apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		usr, err := s.octeliumC.CoreC().GetUser(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.UserRef))
		if err != nil {
			return nil, serr.K8sNotFoundOrInternalWithErr(err)
		}

		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.userRef.uid", usr.Metadata.Uid))
	}

	itemList, err := s.octeliumC.AccessC().ListIntegrationIdentity(ctx,
		urscsrv.GetPublicListOptions(req, filters...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) ResolveIntegrationIdentity(ctx context.Context,
	req *accessv1.ResolveIntegrationIdentityRequest) (*accessv1.ResolveIntegrationIdentityResponse, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	integration, err := s.getIntegrationRef(ctx, req.IntegrationRef)
	if err != nil {
		return nil, err
	}

	opts := &accessintg.ResolveOpts{
		OcteliumC:   s.octeliumC,
		Integration: integration,
	}

	provider, providerErr := registry.New(ctx, s.octeliumC, integration)
	if providerErr == nil {
		defer provider.Close()

		if resolver, ok := provider.(accessintg.IdentityResolver); ok {
			opts.Resolver = resolver
		}
	}

	var resolution *accessintg.Resolution

	switch req.Type.(type) {
	case *accessv1.ResolveIntegrationIdentityRequest_ExternalID:
		resolution, err = accessintg.ResolveUserFromExternalID(ctx, opts, req.GetExternalID())
		if err != nil {
			return nil, err
		}

	case *accessv1.ResolveIntegrationIdentityRequest_UserRef:
		if err := apivalidation.CheckObjectRef(req.GetUserRef(),
			&apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		usr, err := s.octeliumC.CoreC().GetUser(ctx,
			apivalidation.ObjectReferenceToRGetOptions(req.GetUserRef()))
		if err != nil {
			return nil, serr.K8sNotFoundOrInternalWithErr(err)
		}

		resolution, err = accessintg.ResolveExternalIDFromUser(ctx, opts, usr)
		if err != nil {
			return nil, err
		}

	default:
		return nil, grpcutils.InvalidArg("You must set either an externalID or a userRef")
	}

	if !resolution.IsResolved && providerErr != nil {
		resolution.Detail = fmt.Sprintf("%s The Integration provider is unusable: %s",
			resolution.Detail, providerErr.Error())
	}

	ret := &accessv1.ResolveIntegrationIdentityResponse{
		IsResolved: resolution.IsResolved,
		ExternalID: resolution.ExternalID,
		Source:     resolution.Source,
		Detail:     resolution.Detail,
	}

	if resolution.User != nil {
		ret.UserRef = umetav1.GetObjectReference(resolution.User)
	}

	if resolution.Identity != nil {
		ret.IntegrationIdentityRef = umetav1.GetObjectReference(resolution.Identity)
	}

	return ret, nil
}

func (s *ServerMain) getIntegrationRef(ctx context.Context,
	ref *metav1.ObjectReference) (*accessv1.Integration, error) {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegration(ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, grpcutils.InvalidArg("The Integration does not exist")
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}
