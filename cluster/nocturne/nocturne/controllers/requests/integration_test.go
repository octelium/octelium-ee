// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package requests

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func createIntegration(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	capabilities ...accessv1.Integration_Status_Capability) *accessv1.Integration {
	item, err := octeliumC.AccessC().CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:           utilrand.GetRandomStringCanonical(24),
			Type:         accessv1.Integration_Status_SLACK,
			Capabilities: capabilities,
		},
	})
	assert.Nil(t, err, "%+v", err)
	return item
}

func createIntegrationTarget(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	integration *accessv1.Integration) *accessv1.IntegrationTarget {
	item, err := octeliumC.AccessC().CreateIntegrationTarget(ctx, &accessv1.IntegrationTarget{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.IntegrationTarget_Spec{
			IntegrationRef: umetav1.GetObjectReference(integration),
			Type: &accessv1.IntegrationTarget_Spec_Slack_{
				Slack: &accessv1.IntegrationTarget_Spec_Slack{
					ChannelID: "C12345678",
				},
			},
		},
		Status: &accessv1.IntegrationTarget_Status{
			Type: accessv1.Integration_Status_SLACK,
		},
	})
	assert.Nil(t, err, "%+v", err)
	return item
}

func createReviewerUser(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return usr
}

func listBindings(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) []*accessv1.IntegrationBinding {
	itemList, err := octeliumC.AccessC().ListIntegrationBinding(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.requestRef.uid", req.Metadata.Uid),
		},
	})
	assert.Nil(t, err, "%+v", err)
	return itemList.Items
}

func TestRequestEnsuresIntegrationBindings(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	integration := createIntegration(t, ctx, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)
	target := createIntegrationTarget(t, ctx, octeliumC, integration)

	reviewer := createReviewerUser(t, ctx, octeliumC)
	svc := createService(t, ctx, octeliumC)

	step := anyStep(userReviewer(umetav1.GetObjectReference(reviewer)))
	step.Name = "first"
	step.Surfaces = []*accessv1.Policy_Spec_Rule_Surface{
		{
			InteractionMode: accessv1.Policy_Spec_Rule_Surface_INTERACTIVE,
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				Type: &accessv1.Policy_Spec_Rule_Surface_Destination_TargetRef{
					TargetRef: umetav1.GetObjectReference(target),
				},
			},
		},
	}

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(step),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			MaxAccessDuration: durationHours(2),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.EffectiveDuration)
	assert.Equal(t, uint32(7200), reqG.Status.EffectiveDuration.GetSeconds())

	bindings := listBindings(t, ctx, octeliumC, reqG)
	assert.Equal(t, 1, len(bindings))

	binding := bindings[0]
	assert.Equal(t, accessintg.BindingName(reqG.Metadata.Uid,
		accessv1.IntegrationBinding_Spec_REVIEW_SURFACE, 0, 0, target.Metadata.Uid),
		binding.Metadata.Name)
	assert.Equal(t, "first", binding.Spec.StepName)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE, binding.Spec.InteractionMode)
	assert.NotEmpty(t, binding.Status.DesiredRevision)

	assert.Equal(t, 1, len(listBindings(t, ctx, octeliumC, reqG)))
}

func TestRequestUpdatesBindingDesiredRevision(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	integration := createIntegration(t, ctx, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)
	target := createIntegrationTarget(t, ctx, octeliumC, integration)

	reviewer := createReviewerUser(t, ctx, octeliumC)
	svc := createService(t, ctx, octeliumC)

	step := anyStep(userReviewer(umetav1.GetObjectReference(reviewer)))
	step.Name = "first"
	step.Surfaces = []*accessv1.Policy_Spec_Rule_Surface{
		{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				Type: &accessv1.Policy_Spec_Rule_Surface_Destination_TargetRef{
					TargetRef: umetav1.GetObjectReference(target),
				},
			},
		},
	}

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(step),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	before := listBindings(t, ctx, octeliumC, reqG)[0].Status.DesiredRevision

	reqG.Status.State.Status = accessv1.Request_Status_State_REJECTED
	reqG, err := octeliumC.AccessC().UpdateRequest(ctx, reqG)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ctrl.OnAdd(ctx, reqG))

	after := listBindings(t, ctx, octeliumC, reqG)[0].Status.DesiredRevision
	assert.NotEqual(t, before, after)
}

func TestRequestDeletesIntegrationBindings(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	integration := createIntegration(t, ctx, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)
	target := createIntegrationTarget(t, ctx, octeliumC, integration)

	reviewer := createReviewerUser(t, ctx, octeliumC)
	svc := createService(t, ctx, octeliumC)

	step := anyStep(userReviewer(umetav1.GetObjectReference(reviewer)))
	step.Name = "first"
	step.Surfaces = []*accessv1.Policy_Spec_Rule_Surface{
		{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				Type: &accessv1.Policy_Spec_Rule_Surface_Destination_TargetRef{
					TargetRef: umetav1.GetObjectReference(target),
				},
			},
		},
	}

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(step),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, 1, len(listBindings(t, ctx, octeliumC, reqG)))

	assert.Nil(t, ctrl.OnDelete(ctx, reqG))
	assert.Equal(t, 0, len(listBindings(t, ctx, octeliumC, reqG)))
}

func TestRequestWithoutSurfacesCreatesNoBindings(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reviewer := createReviewerUser(t, ctx, octeliumC)
	svc := createService(t, ctx, octeliumC)

	step := anyStep(userReviewer(umetav1.GetObjectReference(reviewer)))
	step.Name = "first"

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(step),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, 0, len(listBindings(t, ctx, octeliumC, reqG)))
}
