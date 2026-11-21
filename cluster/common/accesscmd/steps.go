// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accesscmd

import (
	"context"
	"sort"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

const MaxStepEligibleReviewers = 100

type Reviewer = accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer

func GetReviewAction(req *accessv1.Request) *accessv1.Policy_Spec_Rule_Action_Review {
	if req == nil || req.Status == nil || req.Status.Rule == nil {
		return nil
	}

	return req.Status.Rule.GetAction().GetReview()
}

func GetStepState(req *accessv1.Request, index int32) *accessv1.Request_Status_Review_StepState {
	if req.Status.Review == nil {
		return nil
	}

	for i := len(req.Status.Review.Steps) - 1; i >= 0; i-- {
		itm := req.Status.Review.Steps[i]
		if itm != nil && itm.Index == index {
			return itm
		}
	}

	return nil
}

func CurrentStepState(req *accessv1.Request) *accessv1.Request_Status_Review_StepState {
	if req == nil || req.Status == nil || req.Status.Review == nil {
		return nil
	}

	itm := GetStepState(req, req.Status.Review.CurrentStep)
	if itm == nil || itm.Outcome != accessv1.Request_Status_Review_StepState_OUTCOME_UNSET {
		return nil
	}

	return itm
}

func EndCurrentStep(req *accessv1.Request, outcome accessv1.Request_Status_Review_StepState_Outcome) {
	itm := CurrentStepState(req)
	if itm == nil {
		return
	}

	itm.Outcome = outcome
	itm.EndedAt = pbutils.Now()
}

type StartReviewStepOpts struct {
	OcteliumC     octeliumc.ClientInterface
	Request       *accessv1.Request
	Index         int32
	ClusterConfig *accessv1.ClusterConfig
}

func StartReviewStep(ctx context.Context, o *StartReviewStepOpts) error {
	req := o.Request
	review := GetReviewAction(req)
	if review == nil || o.Index < 0 || int(o.Index) >= len(review.Steps) {
		return errNilArg("review step")
	}

	step := review.Steps[o.Index]
	now := pbutils.Now()

	if req.Status.Review == nil {
		req.Status.Review = &accessv1.Request_Status_Review{}
	}

	req.Status.Review.CurrentStep = o.Index
	req.Status.Review.CurrentStepStartedAt = now

	stepState := &accessv1.Request_Status_Review_StepState{
		Name:      StepName(step, o.Index),
		Index:     o.Index,
		StartedAt: now,
	}
	req.Status.Review.Steps = append(req.Status.Review.Steps, stepState)

	return RefreshStepReviewers(ctx, o.OcteliumC, req, o.ClusterConfig)
}

func ShouldEscalateStep(req *accessv1.Request, now time.Time) bool {
	review := GetReviewAction(req)
	stepState := CurrentStepState(req)
	if review == nil || stepState == nil || stepState.IsEscalated {
		return false
	}

	step := review.Steps[req.Status.Review.CurrentStep]
	if step.Escalation == nil || len(step.Escalation.Reviewers) == 0 {
		return false
	}

	after := umetav1.ToDuration(step.Escalation.After).ToGo()
	if after <= 0 || stepState.StartedAt == nil {
		return false
	}

	return !stepState.StartedAt.AsTime().Add(after).After(now)
}

func EscalateStep(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	stepState := CurrentStepState(req)
	if stepState == nil || stepState.IsEscalated {
		return nil
	}

	stepState.IsEscalated = true
	stepState.EscalatedAt = pbutils.Now()

	return RefreshStepReviewers(ctx, octeliumC, req, cc)
}

func RefreshStepReviewers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	review := GetReviewAction(req)
	stepState := CurrentStepState(req)
	if review == nil || stepState == nil {
		return nil
	}

	step := review.Steps[req.Status.Review.CurrentStep]

	usrs, err := GetStepEligibleUsers(ctx, octeliumC, req, step, stepState, nil)
	if err != nil {
		return err
	}

	hasEscalation := step.Escalation != nil && len(step.Escalation.Reviewers) > 0

	if len(usrs) == 0 && hasEscalation && !stepState.IsEscalated {
		stepState.IsEscalated = true
		stepState.EscalatedAt = pbutils.Now()

		usrs, err = GetStepEligibleUsers(ctx, octeliumC, req, step, stepState, nil)
		if err != nil {
			return err
		}
	}

	if len(usrs) == 0 && cc != nil {
		usrs, err = GetStepEligibleUsers(ctx, octeliumC, req, step, stepState,
			cc.GetSpec().GetReview().GetFallbackReviewers())
		if err != nil {
			return err
		}
	}

	stepState.IsUnroutable = len(usrs) == 0
	stepState.TotalEligibleReviewer = uint32(len(usrs))
	stepState.EligibleReviewers = nil

	for i, usr := range usrs {
		if i >= MaxStepEligibleReviewers {
			break
		}
		stepState.EligibleReviewers = append(stepState.EligibleReviewers, objectRef(usr))
	}

	stepState.RequiredApprovals = GetRequiredApprovals(step, stepState.TotalEligibleReviewer)

	return nil
}

func GetRequiredApprovals(step *accessv1.Policy_Spec_Rule_Action_Review_Step, totalEligible uint32) uint32 {
	switch step.ApprovalRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
		if totalEligible == 0 {
			return 1
		}
		return totalEligible
	case accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT:
		return step.ApprovalCount
	default:
		return 1
	}
}

func GetStepEligibleUsers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	stepState *accessv1.Request_Status_Review_StepState,
	extraReviewers []*Reviewer) ([]*corev1.User, error) {
	reviewers := GetStepReviewers(step, stepState)
	reviewers = append(reviewers, extraReviewers...)

	catalog, err := getRequestCatalog(ctx, octeliumC, req)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var ret []*corev1.User

	addUser := func(usr *corev1.User) {
		if usr == nil || !IsUserEligible(usr) {
			return
		}

		if !IsSeparationOfDutiesSatisfied(step, usr, req) {
			return
		}

		if _, ok := seen[usr.Metadata.Uid]; ok {
			return
		}
		seen[usr.Metadata.Uid] = struct{}{}

		ret = append(ret, usr)
	}

	for _, reviewer := range reviewers {
		usrs, err := resolveReviewerUsers(ctx, octeliumC, reviewer, catalog)
		if err != nil {
			return nil, err
		}

		for _, usr := range usrs {
			addUser(usr)
		}
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Metadata.Name < ret[j].Metadata.Name
	})

	return ret, nil
}

func GetStepReviewers(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	stepState *accessv1.Request_Status_Review_StepState) []*Reviewer {
	ret := append([]*Reviewer{}, step.Reviewers...)

	if stepState != nil && stepState.IsEscalated && step.Escalation != nil {
		ret = append(ret, step.Escalation.Reviewers...)
	}

	return ret
}

func resolveReviewerUsers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	reviewer *Reviewer, catalog *accessv1.Catalog) ([]*corev1.User, error) {
	if reviewer == nil {
		return nil, nil
	}

	switch reviewer.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
		usr, err := GetUser(ctx, octeliumC, reviewer.GetUser().GetUserRef())
		if err != nil {
			return nil, err
		}
		if usr == nil {
			return nil, nil
		}

		return []*corev1.User{usr}, nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
		grp, err := GetGroup(ctx, octeliumC, reviewer.GetGroup().GetGroupRef())
		if err != nil {
			return nil, err
		}
		if grp == nil {
			return nil, nil
		}

		return listGroupUsers(ctx, octeliumC, grp.Metadata.Name)

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_:
		owners := catalog.GetSpec().GetOwners()
		if owners == nil {
			return nil, nil
		}

		var ret []*corev1.User

		for _, name := range owners.Users {
			usr, err := GetUser(ctx, octeliumC, &metav1.ObjectReference{
				Name: name,
			})
			if err != nil {
				return nil, err
			}
			if usr != nil {
				ret = append(ret, usr)
			}
		}

		for _, name := range owners.Groups {
			usrs, err := listGroupUsers(ctx, octeliumC, name)
			if err != nil {
				return nil, err
			}
			ret = append(ret, usrs...)
		}

		return ret, nil

	default:
		return nil, nil
	}
}

func listGroupUsers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	groupName string) ([]*corev1.User, error) {
	usrList, err := octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldIncludesValStr("spec.groups", groupName),
		},
	})
	if err != nil {
		return nil, err
	}

	return usrList.Items, nil
}

func getRequestCatalog(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) (*accessv1.Catalog, error) {
	if scope := req.Status.GetResolvedScope(); scope != nil && scope.CatalogRef != nil {
		return GetCatalog(ctx, octeliumC, scope.CatalogRef)
	}

	return GetCatalog(ctx, octeliumC, req.Spec.GetResource().GetCatalog().GetCatalogRef())
}

func IsStepTimedOut(req *accessv1.Request, now time.Time) bool {
	review := GetReviewAction(req)
	if review == nil || req.Status.Review == nil {
		return false
	}

	idx := req.Status.Review.CurrentStep
	if idx < 0 || int(idx) >= len(review.Steps) {
		return false
	}

	step := review.Steps[idx]
	timeout := umetav1.ToDuration(step.GetTimeout()).ToGo()
	if timeout <= 0 {
		return false
	}

	startedAt := req.Status.Review.CurrentStepStartedAt
	if startedAt == nil && idx == 0 {
		startedAt = req.Status.ApprovalStartAt
	}
	if startedAt == nil || !startedAt.IsValid() {
		return false
	}

	return !startedAt.AsTime().Add(timeout).After(now)
}

func IsDeadlinePassed(req *accessv1.Request, now time.Time) bool {
	deadline := req.GetSpec().GetDeadline()
	return deadline != nil && deadline.IsValid() && !deadline.AsTime().After(now)
}
