// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ingress

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

func (s *Server) setReviewDecision(ctx context.Context, integration *accessv1.Integration,
	provider accessintg.Provider,
	action *accessintg.Action) (*accessintg.ActionResult, error) {
	if !accessintg.HasCapability(integration, accessv1.Integration_Status_INTERACTIVE_REVIEW) {
		return nil, errors.Errorf("The Integration %s cannot submit the decisions of its actors",
			integration.Metadata.Name)
	}

	binding, err := s.getActionBinding(ctx, integration, action)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return &accessintg.ActionResult{
			Message: "This Request is no longer presented here.",
		}, nil
	}

	if binding.Spec.Purpose != accessv1.IntegrationBinding_Spec_REVIEW_SURFACE ||
		binding.Spec.InteractionMode != accessv1.Policy_Spec_Rule_Surface_INTERACTIVE {
		return nil, errors.Errorf("The IntegrationBinding %s does not accept any decision",
			binding.Metadata.Name)
	}

	if hasEventID(binding, action.ExternalEventID) {
		return &accessintg.ActionResult{
			IsAccepted: true,
			Message:    "This decision has already been recorded.",
		}, nil
	}

	req, err := s.getRequest(ctx, binding.Spec.RequestRef)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return &accessintg.ActionResult{
			Message: "This access Request no longer exists.",
		}, nil
	}

	resolution, err := s.resolveActor(ctx, integration, provider, action.ExternalActorID)
	if err != nil {
		return nil, err
	}
	if !resolution.IsResolved {
		return &accessintg.ActionResult{
			Message: fmt.Sprintf(
				"Octelium could not recognize you as one of its Users. %s", resolution.Detail),
		}, nil
	}

	decision := action.Decision

	if resolver, ok := provider.(accessintg.DecisionResolver); ok {
		target, err := accessintg.GetIntegrationTarget(ctx, s.octeliumC, binding.Spec.TargetRef)
		if err != nil {
			return nil, err
		}

		decision, err = resolver.ResolveDecision(ctx, action, target)
		if err != nil {
			return nil, err
		}

		if decision == accessv1.Review_Spec_DECISION_UNSET {
			return &accessintg.ActionResult{
				Message: "This change does not map to any Octelium decision.",
			}, nil
		}
	}

	review, err := accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:        s.octeliumC,
		Reviewer:         resolution.User,
		Request:          req,
		Decision:         decision,
		Justification:    action.Justification,
		ExpectedStepName: binding.Spec.StepName,
		Origin: &accessv1.Origin{
			Type:            accessv1.Origin_INTEGRATION,
			IntegrationRef:  umetav1.GetObjectReference(integration),
			ExternalActorID: action.ExternalActorID,
			ExternalEventID: action.ExternalEventID,
		},
	})
	if err != nil {
		if isActorError(err) {
			return &accessintg.ActionResult{
				Message: errMessage(err),
			}, nil
		}

		return nil, err
	}

	s.appendEventID(ctx, binding, action.ExternalEventID)

	return &accessintg.ActionResult{
		IsAccepted:  true,
		Message:     fmt.Sprintf("Your %s decision was recorded.", decisionLabel(decision)),
		RequestName: req.Metadata.Name,
		PortalURL:   s.portalURL(fmt.Sprintf("reviewer/reviews/%s", review.Metadata.Name)),
	}, nil
}

func (s *Server) createRequest(ctx context.Context, integration *accessv1.Integration,
	provider accessintg.Provider,
	action *accessintg.Action) (*accessintg.ActionResult, error) {
	if !accessintg.HasCapability(integration, accessv1.Integration_Status_REQUEST_CREATION) {
		return nil, errors.Errorf("The Integration %s cannot create access Requests",
			integration.Metadata.Name)
	}

	resolution, err := s.resolveActor(ctx, integration, provider, action.ExternalActorID)
	if err != nil {
		return nil, err
	}
	if !resolution.IsResolved {
		return &accessintg.ActionResult{
			Message: fmt.Sprintf(
				"Octelium could not recognize you as one of its Users. %s", resolution.Detail),
		}, nil
	}

	resource, err := accessintg.ResolveResourceQuery(ctx, s.octeliumC, action.ResourceQuery)
	if err != nil {
		if grpcerr.IsInvalidArg(err) {
			return &accessintg.ActionResult{
				Message: errMessage(err),
			}, nil
		}
		return nil, err
	}

	spec := &accessv1.Request_Spec{
		Resource:      resource,
		Urgency:       action.Urgency,
		Justification: action.Justification,
	}

	if action.Duration > 0 {
		spec.Duration = &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: uint32(action.Duration.Seconds()),
			},
		}
	}

	req, err := accesscmd.CreateRequest(ctx, &accesscmd.CreateRequestOpts{
		OcteliumC: s.octeliumC,
		Requester: resolution.User,
		Spec:      spec,
		Origin: &accessv1.Origin{
			Type:            accessv1.Origin_INTEGRATION,
			IntegrationRef:  umetav1.GetObjectReference(integration),
			ExternalActorID: action.ExternalActorID,
			ExternalEventID: action.ExternalEventID,
		},
	})
	if err != nil {
		if isActorError(err) {
			return &accessintg.ActionResult{
				Message: errMessage(err),
			}, nil
		}

		return nil, err
	}

	return &accessintg.ActionResult{
		IsAccepted:  true,
		Message:     fmt.Sprintf("The access Request %s was created.", req.Metadata.Name),
		RequestName: req.Metadata.Name,
		PortalURL:   s.portalURL(fmt.Sprintf("user/requests/%s", req.Metadata.Name)),
	}, nil
}

func (s *Server) resolveActor(ctx context.Context, integration *accessv1.Integration,
	provider accessintg.Provider, externalActorID string) (*accessintg.Resolution, error) {
	opts := &accessintg.ResolveOpts{
		OcteliumC:   s.octeliumC,
		Integration: integration,
	}

	if resolver, ok := provider.(accessintg.IdentityResolver); ok {
		opts.Resolver = resolver
	}

	return accessintg.ResolveUserFromExternalID(ctx, opts, externalActorID)
}

func (s *Server) getActionBinding(ctx context.Context, integration *accessv1.Integration,
	action *accessintg.Action) (*accessv1.IntegrationBinding, error) {
	var binding *accessv1.IntegrationBinding

	switch {
	case action.BindingName != "":
		item, err := s.octeliumC.AccessC().GetIntegrationBinding(ctx, &rmetav1.GetOptions{
			Name: action.BindingName,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				return nil, nil
			}
			return nil, grpcutils.InternalWithErr(err)
		}
		binding = item

	case action.ExternalObjectID != "":
		itemList, err := s.octeliumC.AccessC().ListIntegrationBinding(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", integration.Metadata.Uid),
				urscsrv.FilterFieldEQValStr("status.externalID", action.ExternalObjectID),
			},
		})
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		if len(itemList.Items) != 1 {
			return nil, nil
		}
		binding = itemList.Items[0]

	default:
		return nil, errors.Errorf("The inbound action refers to no external object")
	}

	if binding.Spec.IntegrationRef == nil ||
		binding.Spec.IntegrationRef.Uid != integration.Metadata.Uid {
		return nil, errors.Errorf("The IntegrationBinding %s belongs to another Integration",
			binding.Metadata.Name)
	}

	return binding, nil
}

func (s *Server) getRequest(ctx context.Context,
	ref *metav1.ObjectReference) (*accessv1.Request, error) {
	if ref == nil {
		return nil, nil
	}

	req, err := s.octeliumC.AccessC().GetRequest(ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return req, nil
}

func (s *Server) appendEventID(ctx context.Context,
	binding *accessv1.IntegrationBinding, eventID string) {
	if eventID == "" {
		return
	}

	next := pbutils.Clone(binding).(*accessv1.IntegrationBinding)

	next.Status.LastExternalEventIDs = append(
		[]string{eventID}, next.Status.LastExternalEventIDs...)

	if len(next.Status.LastExternalEventIDs) > accessintg.MaxBindingEventIDs {
		next.Status.LastExternalEventIDs =
			next.Status.LastExternalEventIDs[:accessintg.MaxBindingEventIDs]
	}

	if _, err := s.octeliumC.AccessC().UpdateIntegrationBinding(ctx, next); err != nil {
		zap.L().Warn("Could not record the inbound event ID of an IntegrationBinding",
			zap.String("binding", binding.Metadata.Name), zap.Error(err))
	}
}

func hasEventID(binding *accessv1.IntegrationBinding, eventID string) bool {
	if eventID == "" {
		return false
	}

	for _, itm := range binding.Status.LastExternalEventIDs {
		if itm == eventID {
			return true
		}
	}

	return false
}

func isActorError(err error) bool {
	return grpcerr.IsUnauthorized(err) || grpcerr.IsInvalidArg(err) || grpcerr.IsNotFound(err)
}

func errMessage(err error) string {
	return status.Convert(err).Message()
}

func decisionLabel(decision accessv1.Review_Spec_Decision) string {
	switch decision {
	case accessv1.Review_Spec_DECISION_APPROVE:
		return "APPROVE"
	case accessv1.Review_Spec_DECISION_REJECT:
		return "REJECT"
	default:
		return "WITHDRAW"
	}
}
