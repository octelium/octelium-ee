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
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReconcileReviewStepOpts struct {
	OcteliumC     octeliumc.ClientInterface
	Request       *accessv1.Request
	ClusterConfig *accessv1.ClusterConfig
}

func ReconcileReviewStep(ctx context.Context, o *ReconcileReviewStepOpts) error {
	req := o.Request
	if !IsPending(req) || req.Status.Review == nil {
		return nil
	}

	step, stepIndex, err := CurrentStep(req)
	if err != nil {
		return nil
	}

	stepState := CurrentStepState(req)
	if stepState == nil || stepState.SpecRevision != req.Status.SpecRevision {
		return nil
	}

	now := time.Now()
	if IsDeadlinePassed(req, now) || IsStepTimedOut(req, now) {
		return nil
	}

	cc := o.ClusterConfig
	if cc == nil {
		cc, err = GetClusterConfig(ctx, o.OcteliumC)
		if err != nil {
			return err
		}
	}

	usrs, err := refreshStepReviewers(ctx, o.OcteliumC, req, cc)
	if err != nil {
		return err
	}

	reviews, err := getCountedStepReviews(ctx, o.OcteliumC, req, stepIndex)
	if err != nil {
		return err
	}

	syncStepDecisions(req, step, stepIndex, reviews)

	var approvals []*accessv1.Review

	for _, rev := range reviews {
		if rev.Spec.Decision == accessv1.Review_Spec_DECISION_REJECT {
			EndCurrentStep(req, accessv1.Request_Status_Review_StepState_REJECTED)
			SetRequestState(req, &StateChange{
				Status:   accessv1.Request_Status_State_REJECTED,
				Reason:   accessv1.Request_Status_State_REJECTED_BY_REVIEWER,
				Message:  rev.Spec.Justification,
				ActorRef: rev.Status.UserRef,
			})
			return nil
		}

		approvals = append(approvals, rev)
	}

	stepState.Approvals = uint32(len(approvals))

	approved, err := isStepApproved(step, usrs, approvals)
	if err != nil {
		return err
	}

	if !approved || IsAwaitingRequester(req) {
		return nil
	}

	EndCurrentStep(req, accessv1.Request_Status_Review_StepState_SATISFIED)

	if int(stepIndex)+1 < len(GetReviewAction(req).Steps) {
		return StartReviewStep(ctx, &StartReviewStepOpts{
			OcteliumC:     o.OcteliumC,
			Request:       req,
			Index:         stepIndex + 1,
			ClusterConfig: cc,
		})
	}

	applyApprovedDuration(req)
	SetRequestState(req, &StateChange{
		Status:   accessv1.Request_Status_State_APPROVED,
		Reason:   accessv1.Request_Status_State_APPROVED_BY_REVIEWERS,
		ActorRef: approvals[len(approvals)-1].Status.UserRef,
	})

	return nil
}

func getCountedStepReviews(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, stepIndex int32) ([]*accessv1.Review, error) {
	revList, err := octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, err
	}

	byUser := map[string]*accessv1.Review{}

	for _, rev := range revList.Items {
		if rev == nil || rev.Spec == nil || rev.Status == nil ||
			rev.Status.StepIndex != stepIndex ||
			rev.Status.SpecRevision != req.Status.SpecRevision ||
			rev.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET ||
			rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
			continue
		}

		if prev, ok := byUser[rev.Status.UserRef.Uid]; ok &&
			!reviewSetAt(rev).AsTime().After(reviewSetAt(prev).AsTime()) {
			continue
		}

		byUser[rev.Status.UserRef.Uid] = rev
	}

	var ret []*accessv1.Review

	for _, rev := range byUser {
		usr, err := GetUser(ctx, octeliumC, rev.Status.UserRef)
		if err != nil {
			return nil, err
		}

		if !IsUserEligible(usr) {
			continue
		}

		ok, err := CanReview(ctx, octeliumC, usr, req)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		ret = append(ret, rev)
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return reviewSetAt(ret[i]).AsTime().Before(reviewSetAt(ret[j]).AsTime())
	})

	return ret, nil
}

func syncStepDecisions(req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step, stepIndex int32, reviews []*accessv1.Review) {
	counted := map[string]*accessv1.Review{}
	for _, rev := range reviews {
		counted[rev.Metadata.Uid] = rev
	}

	applied := map[string]struct{}{}
	var lastSteps []*accessv1.Request_Status_Review_Step

	for _, itm := range req.Status.Review.LastSteps {
		var rev *accessv1.Review
		if itm.ReviewRef != nil {
			rev = counted[itm.ReviewRef.Uid]
		}

		if rev == nil {
			if itm.StepIndex == stepIndex && itm.SpecRevision == req.Status.SpecRevision {
				continue
			}

			lastSteps = append(lastSteps, itm)
			continue
		}

		if _, ok := applied[rev.Metadata.Uid]; ok {
			continue
		}
		applied[rev.Metadata.Uid] = struct{}{}

		lastSteps = append(lastSteps, newAppliedStep(step, stepIndex, rev))
	}

	for _, rev := range reviews {
		if _, ok := applied[rev.Metadata.Uid]; ok {
			continue
		}

		lastSteps = append(lastSteps, newAppliedStep(step, stepIndex, rev))
	}

	req.Status.Review.LastSteps = lastSteps
}

func newAppliedStep(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	stepIndex int32, rev *accessv1.Review) *accessv1.Request_Status_Review_Step {
	return &accessv1.Request_Status_Review_Step{
		ReviewRef:     umetav1.GetObjectReference(rev),
		SetAt:         reviewSetAt(rev),
		StepIndex:     stepIndex,
		Decision:      rev.Spec.Decision,
		UserRef:       pbutils.Clone(rev.Status.UserRef).(*metav1.ObjectReference),
		StepName:      StepName(step, stepIndex),
		Justification: rev.Spec.Justification,
		Duration:      rev.Spec.Duration,
		SpecRevision:  rev.Status.SpecRevision,
	}
}

func isStepApproved(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	usrs []*corev1.User, approvals []*accessv1.Review) (bool, error) {
	switch step.ApprovalRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ANY:
		return len(approvals) > 0, nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
		if len(usrs) == 0 || len(approvals) == 0 {
			return false, nil
		}

		approvers := map[string]struct{}{}
		for _, rev := range approvals {
			approvers[rev.Status.UserRef.Uid] = struct{}{}
		}

		for _, usr := range usrs {
			if _, ok := approvers[usr.Metadata.Uid]; !ok {
				return false, nil
			}
		}

		return true, nil

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
		req.Status.EffectiveDuration = DurationFromGo(effective)
	}
}

func reviewSetAt(rev *accessv1.Review) *timestamppb.Timestamp {
	if rev.Status.SetAt != nil {
		return rev.Status.SetAt
	}

	return pbutils.Now()
}
