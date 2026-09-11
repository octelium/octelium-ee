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
	"net/url"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accessintg/registry"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
)

const (
	maxIntegrationStringLen = 512
	maxIntegrationURLLen    = 2048
)

type secretOwner interface {
	GetFromSecret() string
}

func (s *ServerMain) CreateIntegration(ctx context.Context,
	req *accessv1.Integration) (*accessv1.Integration, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetIntegration(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The Integration %s already exists", req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.validateIntegration(ctx, req); err != nil {
		return nil, err
	}

	id, err := s.generateIntegrationID(ctx)
	if err != nil {
		return nil, err
	}

	item := &accessv1.Integration{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status: &accessv1.Integration_Status{
			Id:           id,
			Type:         registry.TypeOf(req),
			Capabilities: registry.CapabilitiesOf(req),
			Synchronization: &accessv1.Integration_Status_Synchronization{
				CreatedAt: pbutils.Now(),
				State:     accessv1.Integration_Status_Synchronization_SYNC_REQUESTED,
			},
		},
	}

	item, err = s.octeliumC.AccessC().CreateIntegration(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) GetIntegration(ctx context.Context,
	req *metav1.GetOptions) (*accessv1.Integration, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegration(ctx,
		apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListIntegration(ctx context.Context,
	req *accessv1.ListIntegrationOptions) (*accessv1.IntegrationList, error) {
	itemList, err := s.octeliumC.AccessC().ListIntegration(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) UpdateIntegration(ctx context.Context,
	req *accessv1.Integration) (*accessv1.Integration, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateIntegration(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegration(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if registry.TypeOf(req) != item.Status.Type {
		return nil, grpcutils.InvalidArg("The Integration type cannot be changed")
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)

	if !pbutils.IsEqual(item.Spec, req.Spec) {
		item.Spec = req.Spec
		item.Status.Capabilities = registry.CapabilitiesOf(req)
		item.Status.Synchronization = &accessv1.Integration_Status_Synchronization{
			CreatedAt: pbutils.Now(),
			State:     accessv1.Integration_Status_Synchronization_SYNC_REQUESTED,
		}
	}

	item, err = s.octeliumC.AccessC().UpdateIntegration(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) DeleteIntegration(ctx context.Context,
	req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetIntegration(ctx,
		apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if _, err := s.octeliumC.AccessC().DeleteIntegration(ctx,
		apivalidation.ObjectToRDeleteOptions(item)); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) SynchronizeIntegration(ctx context.Context,
	req *accessv1.SynchronizeIntegrationRequest) (*accessv1.SynchronizeIntegrationResponse, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	if err := apivalidation.CheckObjectRef(req.IntegrationRef,
		&apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetIntegration(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.IntegrationRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if item.Status.Synchronization != nil &&
		item.Status.Synchronization.State == accessv1.Integration_Status_Synchronization_SYNCING {
		return nil, grpcutils.InvalidArg("The Integration is already SYNCING")
	}

	item.Status.Synchronization = &accessv1.Integration_Status_Synchronization{
		CreatedAt: pbutils.Now(),
		State:     accessv1.Integration_Status_Synchronization_SYNC_REQUESTED,
	}

	if _, err := s.octeliumC.AccessC().UpdateIntegration(ctx, item); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &accessv1.SynchronizeIntegrationResponse{}, nil
}

func (s *ServerMain) validateIntegration(ctx context.Context, req *accessv1.Integration) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	switch req.Spec.Type.(type) {
	case *accessv1.Integration_Spec_Slack_:
		typ := req.Spec.GetSlack()

		if err := s.validateSecretOwner(ctx, typ.GetBotToken()); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.GetSigningSecret()); err != nil {
			return err
		}
		if err := validateIntegrationStr(typ.GetTeamID(), false, "Slack teamID"); err != nil {
			return err
		}
		if typ.GetBaseURL() != "" {
			if err := validateIntegrationURL(typ.GetBaseURL()); err != nil {
				return err
			}
		}

	case *accessv1.Integration_Spec_Jira_:
		typ := req.Spec.GetJira()

		if err := validateIntegrationURL(typ.GetUrl()); err != nil {
			return err
		}
		if typ.GetEmail() == "" {
			return grpcutils.InvalidArg("The Jira email is required")
		}
		if err := apivalidation.CheckEmail(typ.GetEmail()); err != nil {
			return grpcutils.InvalidArg("Invalid email: %s", typ.GetEmail())
		}
		if err := s.validateSecretOwner(ctx, typ.GetApiToken()); err != nil {
			return err
		}
		if typ.GetWebhookSecret() != nil {
			if err := s.validateSecretOwner(ctx, typ.GetWebhookSecret()); err != nil {
				return err
			}
		}

	case *accessv1.Integration_Spec_Webhook_:
		typ := req.Spec.GetWebhook()

		if err := validateIntegrationURL(typ.GetUrl()); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.GetSigningSecret()); err != nil {
			return err
		}
		if typ.GetInboundSecret() != nil {
			if err := s.validateSecretOwner(ctx, typ.GetInboundSecret()); err != nil {
				return err
			}
		}

	default:
		return grpcutils.InvalidArg("You must set a type for the Integration")
	}

	return nil
}

func (s *ServerMain) validateSecretOwner(ctx context.Context, secOwner secretOwner) error {
	if secOwner == nil {
		return grpcutils.InvalidArg("You must set fromSecret")
	}
	if secOwner.GetFromSecret() == "" {
		return grpcutils.InvalidArg("Empty Secret name")
	}
	if err := apivalidation.ValidateName(secOwner.GetFromSecret(), 0, 0); err != nil {
		return grpcutils.InvalidArg("Invalid Secret name: %s", secOwner.GetFromSecret())
	}

	if _, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: secOwner.GetFromSecret(),
	}); err != nil {
		if grpcerr.IsNotFound(err) {
			return grpcutils.InvalidArg("The Secret %s is not found", secOwner.GetFromSecret())
		}
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *ServerMain) generateIntegrationID(ctx context.Context) (string, error) {
	for i := 0; i < 32; i++ {
		id := utilrand.GetRandomStringCanonical(24)

		itemList, err := s.octeliumC.AccessC().ListIntegration(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.id", id),
			},
		})
		if err != nil {
			return "", grpcutils.InternalWithErr(err)
		}

		if len(itemList.Items) == 0 {
			return id, nil
		}
	}

	return "", grpcutils.Internal("Could not generate a unique Integration ID")
}

func validateIntegrationStr(arg string, required bool, name string) error {
	if strings.TrimSpace(arg) == "" {
		if required {
			return grpcutils.InvalidArg("%s is required", name)
		}
		return nil
	}

	if len(arg) > maxIntegrationStringLen {
		return grpcutils.InvalidArg("%s is too long", name)
	}

	if err := apivalidation.ValidateGenASCII(arg); err != nil {
		return grpcutils.InvalidArg("%s", err.Error())
	}

	return nil
}

func validateIntegrationURL(arg string) error {
	if strings.TrimSpace(arg) == "" {
		return grpcutils.InvalidArg("The URL is required")
	}

	if len(arg) > maxIntegrationURLLen {
		return grpcutils.InvalidArg("The URL is too long")
	}

	u, err := url.Parse(arg)
	if err != nil || u.Host == "" {
		return grpcutils.InvalidArg("Invalid URL")
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return grpcutils.InvalidArg("The URL scheme must be http or https")
	}

	return nil
}
