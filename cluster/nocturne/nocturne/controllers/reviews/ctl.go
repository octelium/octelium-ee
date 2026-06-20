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

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
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
			CurrentStep: 0,
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

	alreadyApplied := hasReview(next.Status.Review, rev)

	if alreadyApplied && !force {
		return nil
	}

	if alreadyApplied && force && rev.Spec.Decision != accessv1.Review_Spec_DECISION_REJECT {
		return nil
	}

	if !alreadyApplied {
		next.Status.Review.LastSteps = append(next.Status.Review.LastSteps, &accessv1.Request_Status_Review_Step{
			ReviewRef: umetav1.GetObjectReference(rev),
			SetAt:     reviewSetAt(rev),
		})
	}

	switch rev.Spec.Decision {
	case accessv1.Review_Spec_DECISION_REJECT:
		c.setState(next, accessv1.Request_Status_State_REJECTED)

	case accessv1.Review_Spec_DECISION_APPROVE:
		if err := c.applyApproval(next, actionReview); err != nil {
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
	req *accessv1.Request,
	actionReview *accessv1.Policy_Spec_Rule_Action_Review,
) error {
	step := actionReview.Steps[req.Status.Review.CurrentStep]

	switch step.OnApproval {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_APPROVE:
		c.setState(req, accessv1.Request_Status_State_APPROVED)
		return nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_GOTO_NEXT_STEP:
		c.gotoNextStepOrApprove(req, actionReview)
		return nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_UNSET:
		switch actionReview.ApprovalMode {
		case accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_FIRST:
			c.setState(req, accessv1.Request_Status_State_APPROVED)
			return nil

		case accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_ALL_STEPS:
			c.gotoNextStepOrApprove(req, actionReview)
			return nil

		case accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_UNSET:
			c.gotoNextStepOrApprove(req, actionReview)
			return nil

		default:
			return errors.Errorf("request %q has invalid approval mode", req.Metadata.Name)
		}

	default:
		return errors.Errorf("request %q has invalid onApproval mode", req.Metadata.Name)
	}
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
		req.Status.LastStates = append(req.Status.LastStates,
			pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State))
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

func hasReview(reqReview *accessv1.Request_Status_Review, rev *accessv1.Review) bool {
	for _, step := range reqReview.LastSteps {
		if step.ReviewRef != nil &&
			step.ReviewRef.Uid != "" &&
			step.ReviewRef.Uid == rev.Metadata.Uid {
			return true
		}
	}

	return false
}

func reviewSetAt(rev *accessv1.Review) *timestamppb.Timestamp {
	if rev.Status.SetAt != nil {
		return rev.Status.SetAt
	}

	return pbutils.Now()
}
