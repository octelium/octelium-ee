// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package reviews

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxRequestStateHistory = 100

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	return &Controller{
		octeliumC: octeliumC,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.Review) error {
	return c.reconcile(ctx, itm, false)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.Review) error {
	force := new.Spec.Decision != old.Spec.Decision ||
		new.Spec.Justification != old.Spec.Justification

	return c.reconcile(ctx, new, force)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.Review) error {
	return nil
}

func (c *Controller) reconcile(ctx context.Context, rev *accessv1.Review, force bool) error {
	if rev.Status.RequestRef == nil || rev.Status.RequestRef.Uid == "" {
		return errors.Errorf("review %q has no requestRef", rev.Metadata.Name)
	}

	if rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
		return errors.Errorf("review %q has no userRef", rev.Metadata.Name)
	}

	if rev.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
		return nil
	}

	req, err := c.getRequest(ctx, rev.Status.RequestRef)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}

	if req.Status.State == nil ||
		req.Status.State.Status != accessv1.Request_Status_State_PENDING {
		return nil
	}

	next := pbutils.Clone(req).(*accessv1.Request)

	if next.Status.Rule == nil {
		return errors.Errorf("pending request %q has no matched policy rule", next.Metadata.Name)
	}

	if next.Status.Rule.Action == nil || next.Status.Rule.Action.GetReview() == nil {
		return errors.Errorf("pending request %q has no review action", next.Metadata.Name)
	}

	if next.Status.Review == nil {
		next.Status.Review = &accessv1.Request_Status_Review{
			CurrentStep:          0,
			CurrentStepStartedAt: pbutils.Now(),
		}
	}

	actionReview := next.Status.Rule.Action.GetReview()

	if len(actionReview.Steps) == 0 {
		return errors.Errorf("pending request %q has no review steps", next.Metadata.Name)
	}

	if next.Status.Review.CurrentStep < 0 ||
		int(next.Status.Review.CurrentStep) >= len(actionReview.Steps) {
		return errors.Errorf("pending request %q has invalid current review step %d",
			next.Metadata.Name, next.Status.Review.CurrentStep)
	}

	if rev.Status.StepIndex != next.Status.Review.CurrentStep {
		return nil
	}

	step := actionReview.Steps[int(next.Status.Review.CurrentStep)]

	isReviewer, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, step, next)
	if err != nil {
		return err
	}
	if !isReviewer {
		return nil
	}

	alreadyApplied, appliedStepIndex := findAppliedReviewStep(next.Status.Review, rev)
	if alreadyApplied {
		if appliedStepIndex != next.Status.Review.CurrentStep {
			return nil
		}

		if !force {
			return nil
		}
	}

	if !alreadyApplied {
		next.Status.Review.LastSteps = append(next.Status.Review.LastSteps, &accessv1.Request_Status_Review_Step{
			ReviewRef: umetav1.GetObjectReference(rev),
			SetAt:     reviewSetAt(rev),
			StepIndex: rev.Status.StepIndex,
		})
	}

	switch rev.Spec.Decision {
	case accessv1.Review_Spec_DECISION_REJECT:
		c.setState(next, accessv1.Request_Status_State_REJECTED)

	case accessv1.Review_Spec_DECISION_APPROVE:
		if err := c.applyApproval(ctx, next, actionReview, rev); err != nil {
			return err
		}

	default:
		return errors.Errorf("review %q has invalid decision", rev.Metadata.Name)
	}

	if pbutils.IsEqual(req, next) {
		return nil
	}

	_, err = c.octeliumC.AccessC().UpdateRequest(ctx, next)
	return err
}

func (c *Controller) applyApproval(
	ctx context.Context,
	req *accessv1.Request,
	actionReview *accessv1.Policy_Spec_Rule_Action_Review,
	curr *accessv1.Review,
) error {
	currentStepReviews, err := c.getCurrentStepReviews(ctx, req, curr)
	if err != nil {
		return err
	}

	step := actionReview.Steps[int(req.Status.Review.CurrentStep)]

	approved, err := c.isStepApproved(ctx, req, step, currentStepReviews)
	if err != nil {
		return err
	}

	if !approved {
		return nil
	}

	c.gotoNextStepOrApprove(req, actionReview)
	return nil
}

func (c *Controller) gotoNextStepOrApprove(
	req *accessv1.Request,
	actionReview *accessv1.Policy_Spec_Rule_Action_Review,
) {
	if int(req.Status.Review.CurrentStep)+1 >= len(actionReview.Steps) {
		c.setState(req, accessv1.Request_Status_State_APPROVED)
		return
	}

	req.Status.Review.CurrentStep++
	req.Status.Review.CurrentStepStartedAt = pbutils.Now()
}

func (c *Controller) getCurrentStepReviews(
	ctx context.Context,
	req *accessv1.Request,
	curr *accessv1.Review,
) ([]*accessv1.Review, error) {
	if req.Status.Review == nil {
		return nil, nil
	}

	currentStep := req.Status.Review.CurrentStep

	revList, err := c.octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, err
	}

	ret := []*accessv1.Review{}
	hasCurr := false

	for _, rev := range revList.Items {
		if rev == nil || rev.Status.StepIndex != currentStep {
			continue
		}

		if curr != nil && curr.Metadata.Uid == rev.Metadata.Uid {
			ret = append(ret, curr)
			hasCurr = true
			continue
		}

		ret = append(ret, rev)
	}

	if curr != nil && !hasCurr && curr.Status.StepIndex == currentStep {
		ret = append(ret, curr)
	}

	return ret, nil
}

func (c *Controller) isStepApproved(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	reviews []*accessv1.Review,
) (bool, error) {
	switch step.ApprovalRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ANY:
		return c.isStepApprovedAny(ctx, req, step, reviews)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
		return c.isStepApprovedAll(ctx, req, step, reviews)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT:
		return c.isStepApprovedCount(ctx, req, step, reviews)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_APPROVAL_REQUIREMENT_UNSET:
		return false, errors.Errorf("approval requirement must be set")

	default:
		return false, errors.Errorf("invalid approval requirement")
	}
}

func (c *Controller) isStepApprovedAny(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	reviews []*accessv1.Review,
) (bool, error) {
	for _, rev := range reviews {
		if rev.Spec.Decision != accessv1.Review_Spec_DECISION_APPROVE {
			continue
		}

		ok, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, step, req)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (c *Controller) isStepApprovedAll(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	reviews []*accessv1.Review,
) (bool, error) {
	if len(step.Reviewers) == 0 {
		return false, nil
	}

	usrs, err := accesscmd.GetStepReviewerUsers(ctx, c.octeliumC, step, req)
	if err != nil {
		return false, err
	}

	if len(usrs) == 0 {
		return false, nil
	}

	approvals := map[string]struct{}{}

	for _, rev := range reviews {
		if rev.Spec.Decision != accessv1.Review_Spec_DECISION_APPROVE {
			continue
		}

		if rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
			continue
		}

		approvals[rev.Status.UserRef.Uid] = struct{}{}
	}

	for _, usr := range usrs {
		if _, ok := approvals[usr.Metadata.Uid]; !ok {
			return false, nil
		}
	}

	return true, nil
}

func (c *Controller) isEligibleReviewer(
	ctx context.Context,
	userRef *metav1.ObjectReference,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	req *accessv1.Request,
) (bool, error) {
	usr, err := accesscmd.GetUser(ctx, c.octeliumC, userRef)
	if err != nil {
		return false, err
	}

	if !accesscmd.IsUserEligible(usr) {
		return false, nil
	}

	if !accesscmd.IsSeparationOfDutiesSatisfied(step, usr, req) {
		return false, nil
	}

	return accesscmd.UserMatchesAnyReviewer(ctx, c.octeliumC, userRef, step.Reviewers)
}

func (c *Controller) isStepApprovedCount(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	reviews []*accessv1.Review,
) (bool, error) {
	if step.ApprovalCount == 0 {
		return false, errors.Errorf("approvalCount must be greater than zero")
	}

	approvals := map[string]struct{}{}

	for _, rev := range reviews {
		if rev.Spec.Decision != accessv1.Review_Spec_DECISION_APPROVE {
			continue
		}

		if rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
			continue
		}

		ok, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, step, req)
		if err != nil {
			return false, err
		}

		if !ok {
			continue
		}

		approvals[rev.Status.UserRef.Uid] = struct{}{}
	}

	return uint32(len(approvals)) >= step.ApprovalCount, nil
}

func (c *Controller) getRequest(ctx context.Context, ref *metav1.ObjectReference) (*accessv1.Request, error) {
	req, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: ref.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return req, nil
}

func (c *Controller) setState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
	now := pbutils.Now()

	if req.Status.State != nil && req.Status.State.Status == status {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		prevState := pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State)

		req.Status.LastStates = append(
			[]*accessv1.Request_Status_State{prevState},
			req.Status.LastStates...,
		)

		if len(req.Status.LastStates) > maxRequestStateHistory {
			req.Status.LastStates = req.Status.LastStates[:maxRequestStateHistory]
		}
	}

	req.Status.State = &accessv1.Request_Status_State{
		CreatedAt: now,
		Status:    status,
	}

	switch status {
	case accessv1.Request_Status_State_APPROVED,
		accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		if req.Status.ApprovalEndAt == nil {
			req.Status.ApprovalEndAt = now
		}
	}
}

func findAppliedReviewStep(reqReview *accessv1.Request_Status_Review, rev *accessv1.Review) (bool, int32) {
	for _, step := range reqReview.LastSteps {
		if step.ReviewRef != nil &&
			step.ReviewRef.Uid != "" &&
			step.ReviewRef.Uid == rev.Metadata.Uid {
			return true, step.StepIndex
		}
	}

	return false, 0
}

func reviewSetAt(rev *accessv1.Review) *timestamppb.Timestamp {
	if rev.Status.SetAt != nil {
		return rev.Status.SetAt
	}

	return pbutils.Now()
}
