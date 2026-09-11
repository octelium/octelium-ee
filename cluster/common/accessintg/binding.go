// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

const MaxBindingEventIDs = 64

type DesiredBinding struct {
	Name string
	Spec *accessv1.IntegrationBinding_Spec
}

func BindingName(requestUID string, purpose accessv1.IntegrationBinding_Spec_Purpose,
	stepIndex int32, surfaceIndex int, recipientUID string) string {
	return resourceNameFromKey("b", requestUID, purpose.String(),
		fmt.Sprintf("%d", stepIndex), fmt.Sprintf("%d", surfaceIndex), recipientUID)
}

func DesiredBindings(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) ([]*DesiredBinding, error) {
	if req.Status.Rule == nil {
		return nil, nil
	}

	ret := []*DesiredBinding{}

	for idx, surface := range req.Status.Rule.Notifications {
		items, err := desiredBindingsOf(ctx, octeliumC, req, surface,
			accessv1.IntegrationBinding_Spec_NOTIFICATION, nil, 0, idx)
		if err != nil {
			return nil, err
		}
		ret = append(ret, items...)
	}

	step, stepIndex, err := accesscmd.CurrentStep(req)
	if err != nil {
		return ret, nil
	}

	for idx, surface := range step.Surfaces {
		items, err := desiredBindingsOf(ctx, octeliumC, req, surface,
			accessv1.IntegrationBinding_Spec_REVIEW_SURFACE, step, stepIndex, idx)
		if err != nil {
			return nil, err
		}
		ret = append(ret, items...)
	}

	return ret, nil
}

func desiredBindingsOf(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request,
	surface *accessv1.Policy_Spec_Rule_Surface,
	purpose accessv1.IntegrationBinding_Spec_Purpose,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	stepIndex int32, surfaceIndex int) ([]*DesiredBinding, error) {
	if surface == nil || surface.Destination == nil {
		return nil, nil
	}

	interactionMode := surface.InteractionMode
	if purpose == accessv1.IntegrationBinding_Spec_NOTIFICATION {
		interactionMode = accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY
	}

	stepName := ""
	if step != nil {
		stepName = accesscmd.StepName(step, stepIndex)
	}

	newBinding := func(integration *accessv1.Integration,
		targetRef *metav1.ObjectReference, userRef *metav1.ObjectReference) *DesiredBinding {
		recipientUID := ""
		switch {
		case targetRef != nil:
			recipientUID = targetRef.Uid
		case userRef != nil:
			recipientUID = userRef.Uid
		}

		return &DesiredBinding{
			Name: BindingName(req.Metadata.Uid, purpose, stepIndex, surfaceIndex, recipientUID),
			Spec: &accessv1.IntegrationBinding_Spec{
				IntegrationRef:  umetav1.GetObjectReference(integration),
				RequestRef:      umetav1.GetObjectReference(req),
				TargetRef:       targetRef,
				UserRef:         userRef,
				StepIndex:       stepIndex,
				StepName:        stepName,
				Purpose:         purpose,
				InteractionMode: interactionMode,
			},
		}
	}

	directBinding := func(integrationRef *metav1.ObjectReference,
		userRef *metav1.ObjectReference) ([]*DesiredBinding, error) {
		if userRef == nil {
			return nil, nil
		}

		integration, err := GetIntegration(ctx, octeliumC, integrationRef)
		if err != nil {
			return nil, err
		}
		if !isIntegrationUsable(integration, accessv1.Integration_Status_DIRECT_USER_DELIVERY) {
			return nil, nil
		}

		return []*DesiredBinding{newBinding(integration, nil, userRef)}, nil
	}

	switch surface.Destination.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Surface_Destination_Reviewers_:
		if purpose != accessv1.IntegrationBinding_Spec_REVIEW_SURFACE || step == nil {
			return nil, nil
		}

		integration, err := GetIntegration(ctx, octeliumC,
			surface.Destination.GetReviewers().GetIntegrationRef())
		if err != nil {
			return nil, err
		}
		if !isIntegrationUsable(integration, accessv1.Integration_Status_DIRECT_USER_DELIVERY) {
			return nil, nil
		}

		usrs, err := accesscmd.GetStepReviewerUsers(ctx, octeliumC, step, req)
		if err != nil {
			return nil, err
		}

		ret := []*DesiredBinding{}
		for _, usr := range usrs {
			ret = append(ret, newBinding(integration, nil, umetav1.GetObjectReference(usr)))
		}

		return ret, nil

	case *accessv1.Policy_Spec_Rule_Surface_Destination_Requester_:
		return directBinding(surface.Destination.GetRequester().GetIntegrationRef(),
			cloneRef(req.Status.UserRef))

	case *accessv1.Policy_Spec_Rule_Surface_Destination_Subject_:
		return directBinding(surface.Destination.GetSubject().GetIntegrationRef(),
			cloneRef(accesscmd.GetSubjectUserRef(req)))

	case *accessv1.Policy_Spec_Rule_Surface_Destination_TargetRef:
		target, err := GetIntegrationTarget(ctx, octeliumC, surface.Destination.GetTargetRef())
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, nil
		}

		integration, err := GetIntegration(ctx, octeliumC, target.Spec.IntegrationRef)
		if err != nil {
			return nil, err
		}
		if !isIntegrationUsable(integration, accessv1.Integration_Status_NOTIFICATION) {
			return nil, nil
		}

		return []*DesiredBinding{
			newBinding(integration, umetav1.GetObjectReference(target), nil),
		}, nil

	default:
		return nil, nil
	}
}

func GetIntegration(ctx context.Context, octeliumC octeliumc.ClientInterface,
	ref *metav1.ObjectReference) (*accessv1.Integration, error) {
	if ref == nil {
		return nil, nil
	}

	item, err := octeliumC.AccessC().GetIntegration(ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}

func GetIntegrationTarget(ctx context.Context, octeliumC octeliumc.ClientInterface,
	ref *metav1.ObjectReference) (*accessv1.IntegrationTarget, error) {
	if ref == nil {
		return nil, nil
	}

	item, err := octeliumC.AccessC().GetIntegrationTarget(ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}

func isIntegrationUsable(itm *accessv1.Integration, capability Capability) bool {
	if itm == nil || itm.Spec == nil || itm.Spec.IsDisabled {
		return false
	}

	return HasCapability(itm, capability)
}

func cloneRef(ref *metav1.ObjectReference) *metav1.ObjectReference {
	if ref == nil {
		return nil
	}

	return &metav1.ObjectReference{
		ApiVersion: ref.ApiVersion,
		Kind:       ref.Kind,
		Name:       ref.Name,
		Uid:        ref.Uid,
	}
}
