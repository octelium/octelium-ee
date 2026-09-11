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
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerMain) CreateIntegrationIdentity(ctx context.Context,
	req *accessv1.IntegrationIdentity) (*accessv1.IntegrationIdentity, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetIntegrationIdentity(ctx,
		apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The IntegrationIdentity %s already exists",
			req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.validateIntegrationIdentity(ctx, req, ""); err != nil {
		return nil, err
	}

	item := &accessv1.IntegrationIdentity{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status: &accessv1.IntegrationIdentity_Status{
			Source:     accessv1.IntegrationIdentity_Status_MANUAL,
			VerifiedAt: pbutils.Now(),
		},
	}

	item, err = s.octeliumC.AccessC().CreateIntegrationIdentity(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

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
			urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", integration.Metadata.Uid))
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
			urscsrv.FilterFieldEQValStr("spec.userRef.uid", usr.Metadata.Uid))
	}

	itemList, err := s.octeliumC.AccessC().ListIntegrationIdentity(ctx,
		urscsrv.GetPublicListOptions(req, filters...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) UpdateIntegrationIdentity(ctx context.Context,
	req *accessv1.IntegrationIdentity) (*accessv1.IntegrationIdentity, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegrationIdentity(ctx,
		apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := s.validateIntegrationIdentity(ctx, req, item.Metadata.Uid); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)

	if !pbutils.IsEqual(item.Spec, req.Spec) {
		item.Spec = req.Spec
		item.Status.Source = accessv1.IntegrationIdentity_Status_MANUAL
		item.Status.VerifiedAt = pbutils.Now()
	}

	item, err = s.octeliumC.AccessC().UpdateIntegrationIdentity(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) DeleteIntegrationIdentity(ctx context.Context,
	req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetIntegrationIdentity(ctx,
		apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if _, err := s.octeliumC.AccessC().DeleteIntegrationIdentity(ctx,
		apivalidation.ObjectToRDeleteOptions(item)); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
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

func (s *ServerMain) validateIntegrationIdentity(ctx context.Context,
	req *accessv1.IntegrationIdentity, selfUID string) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	integration, err := s.getIntegrationRef(ctx, req.Spec.IntegrationRef)
	if err != nil {
		return err
	}

	if err := apivalidation.CheckObjectRef(req.Spec.UserRef,
		&apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}

	usr, err := s.octeliumC.CoreC().GetUser(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.Spec.UserRef))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return grpcutils.InvalidArg("The User does not exist")
		}
		return grpcutils.InternalWithErr(err)
	}

	if err := validateIntegrationStr(req.Spec.ExternalID, true, "The externalID"); err != nil {
		return err
	}

	if err := s.checkIntegrationIdentityUnique(ctx, selfUID,
		urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", integration.Metadata.Uid),
		urscsrv.FilterFieldEQValStr("spec.externalID", req.Spec.ExternalID)); err != nil {
		return grpcutils.AlreadyExists(
			"The external actor %s is already linked within the Integration %s",
			req.Spec.ExternalID, integration.Metadata.Name)
	}

	if err := s.checkIntegrationIdentityUnique(ctx, selfUID,
		urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", integration.Metadata.Uid),
		urscsrv.FilterFieldEQValStr("spec.userRef.uid", usr.Metadata.Uid)); err != nil {
		return grpcutils.AlreadyExists(
			"The User %s is already linked within the Integration %s",
			usr.Metadata.Name, integration.Metadata.Name)
	}

	return nil
}

func (s *ServerMain) checkIntegrationIdentityUnique(ctx context.Context, selfUID string,
	filters ...*rmetav1.ListOptions_Filter) error {
	itemList, err := s.octeliumC.AccessC().ListIntegrationIdentity(ctx, &rmetav1.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	for _, item := range itemList.Items {
		if item.Metadata.Uid != selfUID {
			return grpcutils.AlreadyExists("Duplicate IntegrationIdentity")
		}
	}

	return nil
}
