// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessconfig

import (
	"context"
	"fmt"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newControllerTest(t *testing.T) (context.Context, *Controller, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	celEngine, err := celengine.New(ctx, nil)
	require.NoError(t, err)

	return ctx, NewController(tst.C.OcteliumC, celEngine), tst.C.OcteliumC
}

func createService(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	md *metav1.Metadata) *corev1.Service {
	if md == nil {
		md = &metav1.Metadata{}
	}
	md.Name = fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8))

	svc, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: md,
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{},
	})
	require.NoError(t, err)
	return svc
}

func createUser(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status:   &corev1.User_Status{},
	})
	require.NoError(t, err)
	return usr
}

func fullLevel(name string, priority *int32) *accessv1.Catalog_Spec_AccessLevel {
	return &accessv1.Catalog_Spec_AccessLevel{
		Name: name,
		Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
			Full: &accessv1.Catalog_Spec_AccessLevel_Full{
				Priority: priority,
			},
		},
	}
}

func issueTypes(issues []*accessv1.ConfigurationIssue) []accessv1.ConfigurationIssue_Type {
	var ret []accessv1.ConfigurationIssue_Type
	for _, itm := range issues {
		ret = append(ret, itm.Type)
	}
	return ret
}

func userReviewer(usr *corev1.User) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
			User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
				UserRef: umetav1.GetObjectReference(usr),
			},
		},
	}
}

func reviewRule(name string, priority int32, cond *accessv1.Policy_Spec_Rule_Condition,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      name,
		Priority:  priority,
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: cond,
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{step},
				},
			},
		},
	}
}

func TestCatalogStatus(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC, nil)
	hidden := createService(t, ctx, octeliumC, &metav1.Metadata{IsSystem: true, IsUserHidden: true})
	priority := int32(-2)

	catalog, err := octeliumC.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services:   []string{svc.Metadata.Name, hidden.Metadata.Name, "missing.default"},
					Namespaces: []string{"missing"},
				},
			},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{
				fullLevel("read", nil),
				fullLevel("break-glass", &priority),
			},
			Audience: &accessv1.Catalog_Spec_Audience{
				Users: []string{"nobody"},
				Match: "ctx.user.(",
			},
			Owners: &accessv1.Catalog_Spec_Owners{
				Groups: []string{"missing-group"},
			},
		},
		Status: &accessv1.Catalog_Status{},
	})
	require.NoError(t, err)

	require.NoError(t, ctrl.OnCatalogAdd(ctx, catalog))

	catalogG, err := octeliumC.AccessC().GetCatalog(ctx, &rmetav1.GetOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)

	assert.Equal(t, uint32(1), catalogG.Status.TotalService)
	assert.NotNil(t, catalogG.Status.CheckedAt)

	types := issueTypes(catalogG.Status.Issues)
	assert.Contains(t, types, accessv1.ConfigurationIssue_MISSING_REFERENCE)
	assert.Contains(t, types, accessv1.ConfigurationIssue_SYSTEM_RESOURCE_EXCLUDED)
	assert.Contains(t, types, accessv1.ConfigurationIssue_OVERRIDES_GUARDRAILS)
	assert.Contains(t, types, accessv1.ConfigurationIssue_INVALID_EXPRESSION)
	assert.NotContains(t, types, accessv1.ConfigurationIssue_EMPTY_SCOPE)

	for _, issue := range catalogG.Status.Issues {
		if issue.Type == accessv1.ConfigurationIssue_OVERRIDES_GUARDRAILS {
			assert.Equal(t, "break-glass", issue.AccessLevel)
		}
	}

	require.NoError(t, ctrl.OnCatalogUpdate(ctx, catalogG, catalogG))
	catalogA, err := octeliumC.AccessC().GetCatalog(ctx, &rmetav1.GetOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, catalogG.Metadata.ResourceVersion, catalogA.Metadata.ResourceVersion)

	catalogA.Spec.AccessLevels = nil
	catalogA, err = octeliumC.AccessC().UpdateCatalog(ctx, catalogA)
	require.NoError(t, err)
	require.NoError(t, ctrl.OnCatalogUpdate(ctx, catalogA, catalogG))

	catalogA, err = octeliumC.AccessC().GetCatalog(ctx, &rmetav1.GetOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)
	assert.Contains(t, issueTypes(catalogA.Status.Issues), accessv1.ConfigurationIssue_NO_ACCESS_LEVEL)

	empty, err := octeliumC.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services: []string{hidden.Metadata.Name},
				},
			},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{fullLevel("read", nil)},
		},
		Status: &accessv1.Catalog_Status{},
	})
	require.NoError(t, err)

	status, err := ctrl.GetCatalogStatus(ctx, empty)
	require.NoError(t, err)
	assert.Contains(t, issueTypes(status.Issues), accessv1.ConfigurationIssue_EMPTY_SCOPE)
	assert.Equal(t, uint32(0), status.TotalService)
}

func TestPolicyStatus(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reviewer := createUser(t, ctx, octeliumC)
	disabled := createUser(t, ctx, octeliumC)
	disabled.Spec.IsDisabled = true
	disabled, err := octeliumC.CoreC().UpdateUser(ctx, disabled)
	require.NoError(t, err)

	matchAny := &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true},
	}

	countStep := &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{userReviewer(reviewer)},
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
		ApprovalCount:       2,
	}

	unroutableStep := &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{userReviewer(disabled)},
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}

	pol, err := octeliumC.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{Name: "a-policy"},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				reviewRule("count", -2, &accessv1.Policy_Spec_Rule_Condition{
					Type: &accessv1.Policy_Spec_Rule_Condition_Match{Match: "ctx.request.("},
				}, countStep),
				reviewRule("unroutable", -1, &accessv1.Policy_Spec_Rule_Condition{
					Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
						Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
							Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{
								CatalogRef: &metav1.ObjectReference{Name: "missing", Uid: "00000000-0000-0000-0000-000000000000"},
							},
						},
					},
				}, unroutableStep),
				{
					Name:      "catch-all",
					Effect:    accessv1.Policy_Spec_Rule_DENY,
					Condition: matchAny,
				},
				{
					Name:      "never",
					Priority:  2,
					Effect:    accessv1.Policy_Spec_Rule_DENY,
					Condition: matchAny,
				},
			},
		},
		Status: &accessv1.Policy_Status{},
	})
	require.NoError(t, err)

	require.NoError(t, ctrl.OnPolicyAdd(ctx, pol))

	polG, err := octeliumC.AccessC().GetPolicy(ctx, &rmetav1.GetOptions{Uid: pol.Metadata.Uid})
	require.NoError(t, err)

	byRule := map[string][]accessv1.ConfigurationIssue_Type{}
	for _, issue := range polG.Status.Issues {
		byRule[issue.RuleName] = append(byRule[issue.RuleName], issue.Type)
	}

	assert.Contains(t, byRule["count"], accessv1.ConfigurationIssue_INVALID_EXPRESSION)
	assert.Contains(t, byRule["count"], accessv1.ConfigurationIssue_IMPOSSIBLE_APPROVAL_COUNT)
	assert.Contains(t, byRule["unroutable"], accessv1.ConfigurationIssue_MISSING_REFERENCE)
	assert.Contains(t, byRule["unroutable"], accessv1.ConfigurationIssue_NO_ELIGIBLE_REVIEWER)
	assert.Empty(t, byRule["catch-all"])
	assert.Equal(t, []accessv1.ConfigurationIssue_Type{accessv1.ConfigurationIssue_SHADOWED_RULE}, byRule["never"])

	require.NoError(t, ctrl.OnPolicyUpdate(ctx, polG, polG))
	polA, err := octeliumC.AccessC().GetPolicy(ctx, &rmetav1.GetOptions{Uid: pol.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, polG.Metadata.ResourceVersion, polA.Metadata.ResourceVersion)

	require.NoError(t, ctrl.ResyncAll(ctx))
	polA, err = octeliumC.AccessC().GetPolicy(ctx, &rmetav1.GetOptions{Uid: pol.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, polG.Metadata.ResourceVersion, polA.Metadata.ResourceVersion)
}
