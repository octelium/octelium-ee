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
	"time"

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
		new.Spec.Justification != old.Spec.Justification ||
		!pbutils.IsEqual(new.Spec.Duration, old.Spec.Duration) ||
		new.Status.SpecRevision != old.Status.SpecRevision

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

	if rev.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET && !force {
		return nil
	}

	rev, err := c.getReview(ctx, rev)
	if err != nil {
		return err
	}
	if rev == nil {
		return nil
	}

	if rev.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
		return c.withdraw(ctx, rev)
	}

	req, err := c.getRequest(ctx, rev.Status.RequestRef)
	if err != nil {
		return err
	}
	if req == nil || !accesscmd.IsPending(req) {
		return nil
	}

	next := pbutils.Clone(req).(*accessv1.Request)

	actionReview := accesscmd.GetReviewAction(next)
	if actionReview == nil {
		return errors.Errorf("pending request %q has no review action", next.Metadata.Name)
	}

	if len(actionReview.Steps) == 0 {
		return errors.Errorf("pending request %q has no review steps", next.Metadata.Name)
	}

	if next.Status.Review == nil {
		next.Status.Review = &accessv1.Request_Status_Review{
			CurrentStepStartedAt: pbutils.Now(),
		}
	}

	if next.Status.Review.CurrentStep < 0 ||
		int(next.Status.Review.CurrentStep) >= len(actionReview.Steps) {
		return errors.Errorf("pending request %q has invalid current review step %d",
			next.Metadata.Name, next.Status.Review.CurrentStep)
	}

	if rev.Status.StepIndex != next.Status.Review.CurrentStep {
		return nil
	}

	if rev.Status.SpecRevision != next.Status.SpecRevision {
		return nil
	}

	if accesscmd.HasStaleAppliedSteps(next) {
		return nil
	}

	now := time.Now()
	if accesscmd.IsDeadlinePassed(next, now) || accesscmd.IsStepTimedOut(next, now) {
		return nil
	}

	step := actionReview.Steps[int(next.Status.Review.CurrentStep)]

	isReviewer, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, next)
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
			ReviewRef:     umetav1.GetObjectReference(rev),
			SetAt:         reviewSetAt(rev),
			StepIndex:     rev.Status.StepIndex,
			Decision:      rev.Spec.Decision,
			UserRef:       pbutils.Clone(rev.Status.UserRef).(*metav1.ObjectReference),
			StepName:      accesscmd.StepName(step, rev.Status.StepIndex),
			Justification: rev.Spec.Justification,
			Duration:      rev.Spec.Duration,
			SpecRevision:  rev.Status.SpecRevision,
		})
	} else {
		updateAppliedReviewStep(next.Status.Review, rev)
	}

	switch rev.Spec.Decision {
	case accessv1.Review_Spec_DECISION_REJECT:
		accesscmd.EndCurrentStep(next, accessv1.Request_Status_Review_StepState_REJECTED)
		accesscmd.SetRequestState(next, &accesscmd.StateChange{
			Status:   accessv1.Request_Status_State_REJECTED,
			Reason:   accessv1.Request_Status_State_REJECTED_BY_REVIEWER,
			Message:  rev.Spec.Justification,
			ActorRef: rev.Status.UserRef,
		})

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

	rejected, err := c.isStepRejected(ctx, req, currentStepReviews)
	if err != nil {
		return err
	}

	if rejected != nil {
		accesscmd.EndCurrentStep(req, accessv1.Request_Status_Review_StepState_REJECTED)
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:   accessv1.Request_Status_State_REJECTED,
			Reason:   accessv1.Request_Status_State_REJECTED_BY_REVIEWER,
			ActorRef: rejected.Status.UserRef,
			Message:  rejected.Spec.Justification,
		})
		return nil
	}

	approvals, err := c.getStepApprovals(ctx, req, currentStepReviews)
	if err != nil {
		return err
	}

	if stepState := accesscmd.CurrentStepState(req); stepState != nil {
		stepState.Approvals = uint32(len(approvals))
	}

	approved, err := c.isStepApproved(ctx, req, step, approvals)
	if err != nil {
		return err
	}

	if !approved {
		return nil
	}

	cc, err := accesscmd.GetClusterConfig(ctx, c.octeliumC)
	if err != nil {
		return err
	}

	return c.gotoNextStepOrApprove(ctx, req, actionReview, curr, cc)
}

func (c *Controller) gotoNextStepOrApprove(
	ctx context.Context,
	req *accessv1.Request,
	actionReview *accessv1.Policy_Spec_Rule_Action_Review,
	curr *accessv1.Review,
	cc *accessv1.ClusterConfig,
) error {
	accesscmd.EndCurrentStep(req, accessv1.Request_Status_Review_StepState_SATISFIED)

	if int(req.Status.Review.CurrentStep)+1 >= len(actionReview.Steps) {
		applyApprovedDuration(req)
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:   accessv1.Request_Status_State_APPROVED,
			Reason:   accessv1.Request_Status_State_APPROVED_BY_REVIEWERS,
			ActorRef: curr.Status.UserRef,
		})
		return nil
	}

	return accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
		OcteliumC:     c.octeliumC,
		Request:       req,
		Index:         req.Status.Review.CurrentStep + 1,
		ClusterConfig: cc,
	})
}

func (c *Controller) withdraw(ctx context.Context, rev *accessv1.Review) error {
	req, err := c.getRequest(ctx, rev.Status.RequestRef)
	if err != nil {
		return err
	}
	if req == nil || !accesscmd.IsPending(req) || req.Status.Review == nil ||
		rev.Status.StepIndex != req.Status.Review.CurrentStep {
		return nil
	}

	next := pbutils.Clone(req).(*accessv1.Request)

	var lastSteps []*accessv1.Request_Status_Review_Step
	for _, itm := range next.Status.Review.LastSteps {
		if itm.StepIndex == rev.Status.StepIndex && itm.ReviewRef != nil &&
			itm.ReviewRef.Uid == rev.Metadata.Uid {
			continue
		}
		lastSteps = append(lastSteps, itm)
	}
	next.Status.Review.LastSteps = lastSteps

	if stepState := accesscmd.CurrentStepState(next); stepState != nil {
		reviews, err := c.getCurrentStepReviews(ctx, next, rev)
		if err != nil {
			return err
		}

		approvals, err := c.getStepApprovals(ctx, next, reviews)
		if err != nil {
			return err
		}

		stepState.Approvals = uint32(len(approvals))
	}

	if pbutils.IsEqual(req, next) {
		return nil
	}

	_, err = c.octeliumC.AccessC().UpdateRequest(ctx, next)
	return err
}

func applyApprovedDuration(req *accessv1.Request) {
	effective := umetav1.ToDuration(req.Status.EffectiveDuration).ToGo()

	for _, itm := range req.Status.Review.LastSteps {
		if itm.Decision != accessv1.Review_Spec_DECISION_APPROVE ||
			itm.SpecRevision != req.Status.SpecRevision ||
			itm.Duration == nil {
			continue
		}

		if d := umetav1.ToDuration(itm.Duration).ToGo(); d > 0 && (effective <= 0 || d < effective) {
			effective = d
		}
	}

	if effective > 0 {
		req.Status.EffectiveDuration = accesscmd.DurationFromGo(effective)
	}
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
		if rev == nil || rev.Status.StepIndex != currentStep ||
			rev.Status.SpecRevision != req.Status.SpecRevision {
			continue
		}

		if curr != nil && curr.Metadata.Uid == rev.Metadata.Uid {
			ret = append(ret, curr)
			hasCurr = true
			continue
		}

		ret = append(ret, rev)
	}

	if curr != nil && !hasCurr && curr.Status.StepIndex == currentStep &&
		curr.Status.SpecRevision == req.Status.SpecRevision {
		ret = append(ret, curr)
	}

	return ret, nil
}

func (c *Controller) isStepRejected(
	ctx context.Context,
	req *accessv1.Request,
	reviews []*accessv1.Review,
) (*accessv1.Review, error) {
	for _, rev := range reviews {
		if rev.Spec.Decision != accessv1.Review_Spec_DECISION_REJECT {
			continue
		}

		ok, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, req)
		if err != nil {
			return nil, err
		}

		if ok {
			return rev, nil
		}
	}

	return nil, nil
}

func (c *Controller) getStepApprovals(
	ctx context.Context,
	req *accessv1.Request,
	reviews []*accessv1.Review,
) (map[string]struct{}, error) {
	ret := map[string]struct{}{}

	for _, rev := range reviews {
		if rev.Spec.Decision != accessv1.Review_Spec_DECISION_APPROVE {
			continue
		}

		if rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
			continue
		}

		ok, err := c.isEligibleReviewer(ctx, rev.Status.UserRef, req)
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		ret[rev.Status.UserRef.Uid] = struct{}{}
	}

	return ret, nil
}

func (c *Controller) isStepApproved(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	approvals map[string]struct{},
) (bool, error) {
	switch step.ApprovalRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ANY:
		return len(approvals) > 0, nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
		return c.isStepApprovedAll(ctx, req, step, approvals)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT:
		if step.ApprovalCount == 0 {
			return false, errors.Errorf("approvalCount must be greater than zero")
		}

		return uint32(len(approvals)) >= step.ApprovalCount, nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_APPROVAL_REQUIREMENT_UNSET:
		return false, errors.Errorf("approval requirement must be set")

	default:
		return false, errors.Errorf("invalid approval requirement")
	}
}

func (c *Controller) isStepApprovedAll(
	ctx context.Context,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	approvals map[string]struct{},
) (bool, error) {
	if len(step.Reviewers) == 0 || len(approvals) == 0 {
		return false, nil
	}

	usrs, err := accesscmd.GetStepReviewerUsers(ctx, c.octeliumC, step, req)
	if err != nil {
		return false, err
	}

	if stepState := accesscmd.CurrentStepState(req); stepState != nil {
		stepState.RequiredApprovals = accesscmd.GetRequiredApprovals(step, uint32(len(usrs)))
	}

	if len(usrs) == 0 {
		return false, nil
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
	req *accessv1.Request,
) (bool, error) {
	usr, err := accesscmd.GetUser(ctx, c.octeliumC, userRef)
	if err != nil {
		return false, err
	}

	if !accesscmd.IsUserEligible(usr) {
		return false, nil
	}

	return accesscmd.CanReview(ctx, c.octeliumC, usr, req)
}

func (c *Controller) getReview(ctx context.Context,
	rev *accessv1.Review) (*accessv1.Review, error) {
	if rev.Metadata == nil || rev.Metadata.Uid == "" {
		return rev, nil
	}

	item, err := c.octeliumC.AccessC().GetReview(ctx, &rmetav1.GetOptions{
		Uid: rev.Metadata.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return item, nil
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

func updateAppliedReviewStep(reqReview *accessv1.Request_Status_Review, rev *accessv1.Review) {
	for _, step := range reqReview.LastSteps {
		if step.ReviewRef != nil &&
			step.ReviewRef.Uid != "" &&
			step.ReviewRef.Uid == rev.Metadata.Uid {
			step.Decision = rev.Spec.Decision
			step.SetAt = reviewSetAt(rev)
			step.Justification = rev.Spec.Justification
			step.Duration = rev.Spec.Duration
			step.SpecRevision = rev.Status.SpecRevision
			return
		}
	}
}

func reviewSetAt(rev *accessv1.Review) *timestamppb.Timestamp {
	if rev.Status.SetAt != nil {
		return rev.Status.SetAt
	}

	return pbutils.Now()
}
