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
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/raccessv1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestMainPolicy(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	mainSrv := NewServerMain(tst.C.OcteliumC)

	pol, err := mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	polG, err := mainSrv.GetPolicy(ctx, &metav1.GetOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, pol.Metadata.Uid, polG.Metadata.Uid)

	polList, err := mainSrv.ListPolicy(ctx, &accessv1.ListPolicyOptions{})
	assert.Nil(t, err)
	assert.True(t, len(polList.Items) > 0)

	found := false
	for _, item := range polList.Items {
		if item.Metadata.Uid == pol.Metadata.Uid {
			found = true
		}
	}
	assert.True(t, found)

	newName := utilrand.GetRandomStringCanonical(6)
	polG.Spec.Rules[0].Name = newName
	polU, err := mainSrv.UpdatePolicy(ctx, polG)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, newName, polU.Spec.Rules[0].Name)

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: pol.Metadata.Name,
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	dupName := utilrand.GetRandomStringCanonical(6)
	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   dupName,
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
				{
					Name:   dupName,
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
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
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:   utilrand.GetRandomStringCanonical(6),
					Effect: accessv1.Policy_Spec_Rule_DENY,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
							MatchAny: true,
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies: []string{utilrand.GetRandomStringCanonical(6)},
					},
				},
			},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	_, err = mainSrv.DeletePolicy(ctx, &metav1.DeleteOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.GetPolicy(ctx, &metav1.GetOptions{
		Uid: pol.Metadata.Uid,
	})
	assert.NotNil(t, err)
}

func TestMainPolicyMatch(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	mainSrv := NewServerMain(tst.C.OcteliumC)

	usr := tstCreateUser(ctx, t, tst.C.OcteliumC, "")

	newPolicy := func(exp string) *accessv1.Policy {
		return &accessv1.Policy{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.Policy_Spec{
				Rules: []*accessv1.Policy_Spec_Rule{
					{
						Name:     utilrand.GetRandomStringCanonical(6),
						Effect:   accessv1.Policy_Spec_Rule_AUTO_APPROVE,
						Priority: -1,
						Condition: &accessv1.Policy_Spec_Rule_Condition{
							Type: &accessv1.Policy_Spec_Rule_Condition_All_{
								All: &accessv1.Policy_Spec_Rule_Condition_All{
									Of: []*accessv1.Policy_Spec_Rule_Condition{
										{
											Type: &accessv1.Policy_Spec_Rule_Condition_UserRef{
												UserRef: umetav1.GetObjectReference(usr),
											},
										},
										{
											Type: &accessv1.Policy_Spec_Rule_Condition_Match{
												Match: exp,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	pol, err := mainSrv.CreatePolicy(ctx, newPolicy(`time.isWeekdayInTZ(now(), 'Africa/Cairo')`))
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.CreatePolicy(ctx, newPolicy(`ctx.user.metadata.name == "usr1" && ctx.request.spec.urgency == "NORMAL"`))
	assert.Nil(t, err, "%+v", err)

	for _, exp := range []string{
		`time.isWeekdayInTZ(now(), `,
		`time.isWeekdayInTimezone(now(), 'Africa/Cairo')`,
		`'Africa/Cairo'`,
	} {
		_, err = mainSrv.CreatePolicy(ctx, newPolicy(exp))
		assert.NotNil(t, err, "%s", exp)
		assert.True(t, grpcerr.IsInvalidArg(err), "%s: %+v", exp, err)
	}

	pol.Spec.Rules[0].Condition.GetAll().Of[1].Type = &accessv1.Policy_Spec_Rule_Condition_Match{
		Match: `now(`,
	}
	_, err = mainSrv.UpdatePolicy(ctx, pol)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

type policyValidationClient struct {
	octeliumc.ClientInterface
	coreC   *policyValidationCoreClient
	accessC *policyValidationAccessClient
}

func (c *policyValidationClient) CoreC() rcorev1.ResourceServiceClient {
	return c.coreC
}

func (c *policyValidationClient) AccessC() raccessv1.ResourceServiceClient {
	return c.accessC
}

type policyValidationCoreClient struct {
	rcorev1.ResourceServiceClient
	err error
}

func (c *policyValidationCoreClient) GetUser(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*corev1.User, error) {
	return nil, c.err
}

func (c *policyValidationCoreClient) GetGroup(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*corev1.Group, error) {
	return nil, c.err
}

func (c *policyValidationCoreClient) GetService(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*corev1.Service, error) {
	return nil, c.err
}

type policyValidationAccessClient struct {
	raccessv1.ResourceServiceClient
	err error
}

func (c *policyValidationAccessClient) GetCatalog(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*accessv1.Catalog, error) {
	return nil, c.err
}

func newPolicyValidationServer(err error) *ServerMain {
	return NewServerMain(&policyValidationClient{
		coreC:   &policyValidationCoreClient{err: err},
		accessC: &policyValidationAccessClient{err: err},
	})
}

func TestAccessPolicyConditionReferences(t *testing.T) {
	ref := &metav1.ObjectReference{Name: "resource", Uid: "c995d925-cee9-4157-aada-b8f171374539"}
	for _, tt := range []struct {
		name string
		cond *accessv1.Policy_Spec_Rule_Condition
	}{
		{name: "User", cond: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_UserRef{UserRef: ref}}},
		{name: "SubjectUser", cond: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_UserRef{UserRef: ref}}}}},
		{name: "SubjectGroup", cond: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{GroupRef: ref}}}}},
		{name: "Service", cond: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: ref}}}}},
		{name: "Catalog", cond: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{CatalogRef: ref}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, newPolicyValidationServer(nil).validatePolicyCondition(t.Context(), tt.cond, 0))
			err := newPolicyValidationServer(status.Error(codes.NotFound, "not found")).validatePolicyCondition(t.Context(), tt.cond, 0)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
			err = newPolicyValidationServer(status.Error(codes.Unavailable, "unavailable")).validatePolicyCondition(t.Context(), tt.cond, 0)
			assert.True(t, grpcerr.IsInternal(err), "%+v", err)
		})
	}
}

func TestAccessPolicyConditionStructure(t *testing.T) {
	srv := newPolicyValidationServer(nil)
	match := &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true}}
	for _, cond := range []*accessv1.Policy_Spec_Rule_Condition{
		nil,
		{},
		{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_UserRef{}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_All_{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_All_{All: &accessv1.Policy_Spec_Rule_Condition_All{}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_All_{All: &accessv1.Policy_Spec_Rule_Condition_All{Of: []*accessv1.Policy_Spec_Rule_Condition{match, nil}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{Any: &accessv1.Policy_Spec_Rule_Condition_Any{}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{Any: &accessv1.Policy_Spec_Rule_Condition_Any{Of: []*accessv1.Policy_Spec_Rule_Condition{match, nil}}}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_UserRef{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Match{}},
		{Type: &accessv1.Policy_Spec_Rule_Condition_Match{Match: "ctx."}},
	} {
		err := srv.validatePolicyCondition(t.Context(), cond, 0)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v: %+v", cond, err)
	}
	assert.Nil(t, srv.validatePolicyCondition(t.Context(), &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Match{Match: "true"}}, 0))
	cond := match
	for idx := 0; idx < 16; idx++ {
		if idx%2 == 0 {
			cond = &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_All_{All: &accessv1.Policy_Spec_Rule_Condition_All{Of: []*accessv1.Policy_Spec_Rule_Condition{cond}}}}
		} else {
			cond = &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{Any: &accessv1.Policy_Spec_Rule_Condition_Any{Of: []*accessv1.Policy_Spec_Rule_Condition{cond}}}}
		}
	}
	assert.Nil(t, srv.validatePolicyCondition(t.Context(), cond, 0))
	cond = &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_All_{All: &accessv1.Policy_Spec_Rule_Condition_All{Of: []*accessv1.Policy_Spec_Rule_Condition{cond}}}}
	assert.True(t, grpcerr.IsInvalidArg(srv.validatePolicyCondition(t.Context(), cond, 0)))
}

func TestAccessPolicyReviewValidation(t *testing.T) {
	ref := &metav1.ObjectReference{Name: "reviewer", Uid: "c995d925-cee9-4157-aada-b8f171374539"}
	valid := &accessv1.Policy_Spec_Rule_Action_Review{Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{{
		Name: "approval",
		Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
				User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: ref},
			},
		}},
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}}}
	srv := newPolicyValidationServer(nil)
	assert.Nil(t, srv.validatePolicyReview(t.Context(), "rule", valid))
	assert.True(t, grpcerr.IsInvalidArg(srv.validatePolicyReview(t.Context(), "rule", nil)))

	for _, tt := range []struct {
		name   string
		mutate func(*accessv1.Policy_Spec_Rule_Action_Review)
	}{
		{name: "NoSteps", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps = nil }},
		{name: "NilStep", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0] = nil }},
		{name: "NoName", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Name = "" }},
		{name: "InvalidName", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Name = "Invalid Name" }},
		{name: "DuplicateName", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) {
			arg.Steps = append(arg.Steps, proto.Clone(arg.Steps[0]).(*accessv1.Policy_Spec_Rule_Action_Review_Step))
		}},
		{name: "NoReviewers", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Reviewers = nil }},
		{name: "NilReviewer", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Reviewers[0] = nil }},
		{name: "ReviewerType", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Reviewers[0].Type = nil }},
		{name: "UserRef", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Reviewers[0].GetUser().UserRef = nil }},
		{name: "GroupRef", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) {
			arg.Steps[0].Reviewers[0].Type = &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{}}
		}},
		{name: "OnTimeout", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].OnTimeout = 99 }},
		{name: "AnyWithCount", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].ApprovalCount = 1 }},
		{name: "CountWithoutCount", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) {
			arg.Steps[0].ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT
		}},
		{name: "ApprovalRequirement", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].ApprovalRequirement = 99 }},
		{name: "Timeout", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) { arg.Steps[0].Timeout = &metav1.Duration{} }},
		{name: "Surface", mutate: func(arg *accessv1.Policy_Spec_Rule_Action_Review) {
			arg.Steps[0].Surfaces = []*accessv1.Policy_Spec_Rule_Surface{nil}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			arg := proto.Clone(valid).(*accessv1.Policy_Spec_Rule_Action_Review)
			tt.mutate(arg)
			err := srv.validatePolicyReview(t.Context(), "rule", arg)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		})
	}
	for _, group := range []bool{false, true} {
		arg := proto.Clone(valid).(*accessv1.Policy_Spec_Rule_Action_Review)
		if group {
			arg.Steps[0].Reviewers[0].Type = &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{GroupRef: ref}}
		}
		assert.Nil(t, srv.validatePolicyReview(t.Context(), "rule", arg))
		err := newPolicyValidationServer(status.Error(codes.NotFound, "not found")).validatePolicyReview(t.Context(), "rule", arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		err = newPolicyValidationServer(status.Error(codes.Unavailable, "unavailable")).validatePolicyReview(t.Context(), "rule", arg)
		assert.True(t, grpcerr.IsInternal(err), "%+v", err)
		arg.Steps[0].ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT
		arg.Steps[0].ApprovalCount = 2
		arg.Steps[0].Timeout = &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}}
		assert.Nil(t, srv.validatePolicyReview(t.Context(), "rule", arg))
	}
}
