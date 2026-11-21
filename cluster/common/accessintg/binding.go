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
	Name   string
	Status *accessv1.IntegrationBinding_Status
}

func BindingName(requestUID string, status *accessv1.IntegrationBinding_Status, surfaceIndex int) string {
	return resourceNameFromKey("b", requestUID, status.Purpose.String(),
		fmt.Sprintf("%d", status.SpecRevision), fmt.Sprintf("%d", status.StepIndex),
		fmt.Sprintf("%d", surfaceIndex), status.GetUserRef().GetUid(), status.GetIntegrationRef().GetUid(),
		status.Audience.String(), status.InteractionMode.String())
}

func IsBindingCurrent(binding *accessv1.IntegrationBinding, req *accessv1.Request) bool {
	if !accesscmd.IsPending(req) {
		return false
	}

	switch binding.Status.Purpose {
	case accessv1.IntegrationBinding_Status_REVIEW_SURFACE:
		step, stepIndex, err := accesscmd.CurrentStep(req)
		if err != nil || stepIndex != binding.Status.StepIndex ||
			binding.Status.SpecRevision != req.Status.SpecRevision {
			return false
		}

		return hasBindingSurface(binding, step.Surfaces)

	case accessv1.IntegrationBinding_Status_NOTIFICATION:
		if req.Status.Rule == nil {
			return true
		}

		return hasBindingSurface(binding, req.Status.Rule.Notifications)

	default:
		return false
	}
}

func hasBindingSurface(binding *accessv1.IntegrationBinding, surfaces []*accessv1.Policy_Spec_Rule_Surface) bool {
	for _, surface := range surfaces {
		if surface == nil || surface.Destination == nil || surface.Destination.IntegrationRef == nil {
			continue
		}

		interactionMode := surface.InteractionMode
		if binding.Status.Purpose == accessv1.IntegrationBinding_Status_NOTIFICATION {
			interactionMode = accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY
		}

		if interactionMode != binding.Status.InteractionMode ||
			surface.Destination.Audience != binding.Status.Audience {
			continue
		}

		ref := surface.Destination.IntegrationRef
		bindingRef := binding.Status.GetIntegrationRef()

		if (ref.Uid != "" && ref.Uid == bindingRef.GetUid()) ||
			(ref.Uid == "" && ref.Name != "" && ref.Name == bindingRef.GetName()) {
			return true
		}
	}

	return false
}

func DesiredBindings(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) ([]*DesiredBinding, error) {
	if req.Status.Rule == nil {
		return nil, nil
	}

	ret := []*DesiredBinding{}

	for idx, surface := range req.Status.Rule.Notifications {
		items, err := desiredBindingsOf(ctx, octeliumC, req, surface,
			accessv1.IntegrationBinding_Status_NOTIFICATION, nil, 0, idx)
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
			accessv1.IntegrationBinding_Status_REVIEW_SURFACE, step, stepIndex, idx)
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
	purpose accessv1.IntegrationBinding_Status_Purpose,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	stepIndex int32, surfaceIndex int) ([]*DesiredBinding, error) {
	if surface == nil || surface.Destination == nil {
		return nil, nil
	}

	interactionMode := surface.InteractionMode
	if purpose == accessv1.IntegrationBinding_Status_NOTIFICATION {
		interactionMode = accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY
	}

	stepName := ""
	if step != nil {
		stepName = accesscmd.StepName(step, stepIndex)
	}

	integration, err := GetIntegration(ctx, octeliumC, surface.Destination.IntegrationRef)
	if err != nil {
		return nil, err
	}

	var specRevision uint32
	if purpose == accessv1.IntegrationBinding_Status_REVIEW_SURFACE {
		specRevision = req.Status.SpecRevision
	}

	newBinding := func(userRef *metav1.ObjectReference) *DesiredBinding {
		status := &accessv1.IntegrationBinding_Status{
			State:           accessv1.IntegrationBinding_Status_PENDING,
			IntegrationRef:  umetav1.GetObjectReference(integration),
			RequestRef:      umetav1.GetObjectReference(req),
			UserRef:         userRef,
			Audience:        surface.Destination.Audience,
			StepIndex:       stepIndex,
			StepName:        stepName,
			Purpose:         purpose,
			InteractionMode: interactionMode,
			SpecRevision:    specRevision,
		}

		return &DesiredBinding{
			Name:   BindingName(req.Metadata.Uid, status, surfaceIndex),
			Status: status,
		}
	}

	directBinding := func(userRef *metav1.ObjectReference) ([]*DesiredBinding, error) {
		if userRef == nil {
			return nil, nil
		}

		if !isIntegrationUsable(integration, accessv1.Integration_Status_DIRECT_USER_DELIVERY) {
			return nil, nil
		}

		return []*DesiredBinding{newBinding(userRef)}, nil
	}

	switch surface.Destination.Audience {
	case accessv1.Policy_Spec_Rule_Surface_Destination_SHARED:
		if !isIntegrationUsable(integration, accessv1.Integration_Status_NOTIFICATION) {
			return nil, nil
		}

		return []*DesiredBinding{newBinding(nil)}, nil

	case accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS:
		if purpose != accessv1.IntegrationBinding_Status_REVIEW_SURFACE || step == nil {
			return nil, nil
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
			ret = append(ret, newBinding(umetav1.GetObjectReference(usr)))
		}

		return ret, nil

	case accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER:
		return directBinding(cloneRef(req.Status.UserRef))

	case accessv1.Policy_Spec_Rule_Surface_Destination_SUBJECT:
		return directBinding(cloneRef(accesscmd.GetSubjectUserRef(req)))

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
