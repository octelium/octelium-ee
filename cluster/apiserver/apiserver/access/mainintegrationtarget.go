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
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerMain) CreateIntegrationTarget(ctx context.Context,
	req *accessv1.IntegrationTarget) (*accessv1.IntegrationTarget, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetIntegrationTarget(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The IntegrationTarget %s already exists",
			req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	integration, err := s.validateIntegrationTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	item := &accessv1.IntegrationTarget{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status: &accessv1.IntegrationTarget_Status{
			Type: integration.Status.Type,
		},
	}

	item, err = s.octeliumC.AccessC().CreateIntegrationTarget(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) GetIntegrationTarget(ctx context.Context,
	req *metav1.GetOptions) (*accessv1.IntegrationTarget, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegrationTarget(ctx,
		apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListIntegrationTarget(ctx context.Context,
	req *accessv1.ListIntegrationTargetOptions) (*accessv1.IntegrationTargetList, error) {
	if req == nil {
		req = &accessv1.ListIntegrationTargetOptions{}
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

	itemList, err := s.octeliumC.AccessC().ListIntegrationTarget(ctx,
		urscsrv.GetPublicListOptions(req, filters...))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) UpdateIntegrationTarget(ctx context.Context,
	req *accessv1.IntegrationTarget) (*accessv1.IntegrationTarget, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	integration, err := s.validateIntegrationTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegrationTarget(ctx,
		apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec
	item.Status.Type = integration.Status.Type

	item, err = s.octeliumC.AccessC().UpdateIntegrationTarget(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) DeleteIntegrationTarget(ctx context.Context,
	req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetIntegrationTarget(ctx,
		apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if _, err := s.octeliumC.AccessC().DeleteIntegrationTarget(ctx,
		apivalidation.ObjectToRDeleteOptions(item)); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) validateIntegrationTarget(ctx context.Context,
	req *accessv1.IntegrationTarget) (*accessv1.Integration, error) {
	if req.Spec == nil {
		return nil, grpcutils.InvalidArg("Nil Spec")
	}

	integration, err := s.getIntegrationRef(ctx, req.Spec.IntegrationRef)
	if err != nil {
		return nil, err
	}

	switch req.Spec.Type.(type) {
	case *accessv1.IntegrationTarget_Spec_Slack_:
		if integration.Status.Type != accessv1.Integration_Status_SLACK {
			return nil, grpcutils.InvalidArg(
				"The Integration %s is not a Slack Integration", integration.Metadata.Name)
		}

		typ := req.Spec.GetSlack()

		if err := validateIntegrationStr(typ.GetChannelID(), true, "The Slack channelID"); err != nil {
			return nil, err
		}
		if err := validateIntegrationStr(typ.GetMentionUserGroupID(), false,
			"The Slack mentionUserGroupID"); err != nil {
			return nil, err
		}

	case *accessv1.IntegrationTarget_Spec_Jira_:
		if integration.Status.Type != accessv1.Integration_Status_JIRA {
			return nil, grpcutils.InvalidArg(
				"The Integration %s is not a Jira Integration", integration.Metadata.Name)
		}

		typ := req.Spec.GetJira()

		if err := validateIntegrationStr(typ.GetProjectKey(), true, "The Jira projectKey"); err != nil {
			return nil, err
		}
		if err := validateIntegrationStr(typ.GetIssueTypeName(), false,
			"The Jira issueTypeName"); err != nil {
			return nil, err
		}
		if err := validateIntegrationStr(typ.GetApproveStatus(), false,
			"The Jira approveStatus"); err != nil {
			return nil, err
		}
		if err := validateIntegrationStr(typ.GetRejectStatus(), false,
			"The Jira rejectStatus"); err != nil {
			return nil, err
		}

	case *accessv1.IntegrationTarget_Spec_Webhook_:
		if integration.Status.Type != accessv1.Integration_Status_WEBHOOK {
			return nil, grpcutils.InvalidArg(
				"The Integration %s is not a Webhook Integration", integration.Metadata.Name)
		}

		typ := req.Spec.GetWebhook()

		if typ.GetUrl() != "" {
			if err := validateIntegrationURL(typ.GetUrl()); err != nil {
				return nil, err
			}
		}
		if err := validateIntegrationStr(typ.GetName(), false, "The webhook name"); err != nil {
			return nil, err
		}

	default:
		return nil, grpcutils.InvalidArg("You must set a type for the IntegrationTarget")
	}

	return integration, nil
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
