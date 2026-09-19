// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"testing"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func tstReviewPolicy(reviewer *corev1.User,
	steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy {
	return &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_REVIEW,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Action: &accessv1.Policy_Spec_Rule_Action{
						Type: &accessv1.Policy_Spec_Rule_Action_Review_{
							Review: &accessv1.Policy_Spec_Rule_Action_Review{
								Steps: steps,
							},
						},
					},
				},
			},
		},
	}
}

func tstPolicyStep(name string, reviewer *corev1.User,
	surfaces ...*accessv1.Policy_Spec_Rule_Surface) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Name:                name,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			{
				Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
					User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
						UserRef: umetav1.GetObjectReference(reviewer),
					},
				},
			},
		},
		Surfaces: surfaces,
	}
}

func tstPolicySurface(integration *accessv1.Integration,
	audience accessv1.Policy_Spec_Rule_Surface_Destination_Audience,
	mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
	return &accessv1.Policy_Spec_Rule_Surface{
		InteractionMode: mode,
		Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
			IntegrationRef: umetav1.GetObjectReference(integration),
			Audience:       audience,
		},
	}
}

func TestPolicySurface(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer,
			tstPolicySurface(integration,
				accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
				accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))
	assert.Nil(t, err, "%+v", err)
}

func TestPolicySurfaceSharedWithoutAnyDestination(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegrationOf(ctx, t, srv, octeliumC, "")

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer,
			tstPolicySurface(integration,
				accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
				accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicySurfaceWithoutAnyAudience(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer,
			tstPolicySurface(integration,
				accessv1.Policy_Spec_Rule_Surface_Destination_AUDIENCE_UNSET,
				accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicySurfaceReviewers(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer,
			tstPolicySurface(integration,
				accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
				accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))
	assert.Nil(t, err, "%+v", err)
}

func TestPolicyStepNameIsRequired(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer, tstPolicyStep("", reviewer)))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicyStepNameMustBeUnique(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer), tstPolicyStep("first", reviewer)))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicySurfaceUnknownIntegration(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer, &accessv1.Policy_Spec_Rule_Surface{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				IntegrationRef: &metav1.ObjectReference{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Audience: accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			},
		})))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicyReviewSurfaceRejectsRequesterDestination(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	_, err := srv.CreatePolicy(ctx, tstReviewPolicy(reviewer,
		tstPolicyStep("first", reviewer,
			tstPolicySurface(integration,
				accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
				accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET))))
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicyNotificationSurfaceRejectsReviewersDestination(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	pol := tstReviewPolicy(reviewer, tstPolicyStep("first", reviewer))
	pol.Spec.Rules[0].Notifications = []*accessv1.Policy_Spec_Rule_Surface{
		tstPolicySurface(integration,
			accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
			accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
	}

	_, err := srv.CreatePolicy(ctx, pol)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestPolicyNotificationSurface(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC, "")
	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	pol := tstReviewPolicy(reviewer, tstPolicyStep("first", reviewer))
	pol.Spec.Rules[0].Notifications = []*accessv1.Policy_Spec_Rule_Surface{
		tstPolicySurface(integration,
			accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
			accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
	}

	_, err := srv.CreatePolicy(ctx, pol)
	assert.Nil(t, err, "%+v", err)
}
