// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watcher

import (
	"context"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/requests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxRequestStateHistory = 100

type Watcher struct {
	octeliumC octeliumc.ClientInterface
}

func InitWatcher(octeliumC octeliumc.ClientInterface) *Watcher {
	return &Watcher{
		octeliumC: octeliumC,
	}
}

func (w *Watcher) runRequests(ctx context.Context) {
	zap.L().Debug("Starting access Request watcher")

	doRun := func() error {
		reqList, err := w.listRequests(ctx)
		if err != nil {
			return err
		}

		for _, req := range reqList {
			if err := w.doCheckRequest(ctx, req); err != nil {
				zap.L().Warn("Could not doCheckRequest", zap.Error(err), zap.Any("req", req))
			}
		}

		return nil
	}

	tickerCh := time.NewTicker(5 * time.Minute)
	defer tickerCh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh.C:
			if err := doRun(); err != nil {
				zap.L().Error("Could not run access Request watcher doFn", zap.Error(err))
			}
		}
	}
}

func (w *Watcher) listRequests(ctx context.Context) ([]*accessv1.Request, error) {
	var ret []*accessv1.Request
	var page uint32

	for {
		itmList, err := w.octeliumC.AccessC().ListRequest(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: 500,
			Page:         page,
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, itmList.Items...)

		if itmList.ListResponseMeta == nil || !itmList.ListResponseMeta.HasMore {
			return ret, nil
		}

		page = page + 1
	}
}

func (w *Watcher) runPolicyTriggers(ctx context.Context) {
	zap.L().Debug("Starting access Request PolicyTrigger watcher")

	doRun := func() error {
		ptList, err := w.listPolicyTriggers(ctx)
		if err != nil {
			return err
		}

		for _, pt := range ptList {
			if err := w.doCheckPolicyTrigger(ctx, pt); err != nil {
				zap.L().Warn("Could not doCheckPolicyTrigger", zap.Error(err), zap.Any("policyTrigger", pt))
			}
		}

		return nil
	}

	tickerCh := time.NewTicker(5 * time.Minute)
	defer tickerCh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh.C:
			if err := doRun(); err != nil {
				zap.L().Error("Could not run access Request PolicyTrigger watcher doFn", zap.Error(err))
			}
		}
	}
}

func (w *Watcher) listPolicyTriggers(ctx context.Context) ([]*corev1.PolicyTrigger, error) {
	var ret []*corev1.PolicyTrigger
	var page uint32

	for {
		itmList, err := w.octeliumC.CoreC().ListPolicyTrigger(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: 500,
			Page:         page,
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, itmList.Items...)

		if itmList.ListResponseMeta == nil || !itmList.ListResponseMeta.HasMore {
			return ret, nil
		}

		page = page + 1
	}
}

func (w *Watcher) doCheckPolicyTrigger(ctx context.Context, pt *corev1.PolicyTrigger) error {
	if pt == nil || pt.Metadata == nil || pt.Status == nil {
		return nil
	}

	if !strings.HasPrefix(pt.Metadata.Name, requests.PolicyTriggerNamePrefix) {
		return nil
	}

	isOrphan, err := w.isOrphanPolicyTrigger(ctx, pt)
	if err != nil {
		return err
	}
	if !isOrphan {
		return nil
	}

	zap.L().Debug("Deleting orphan access Request PolicyTrigger", zap.Any("policyTrigger", pt))

	_, err = w.octeliumC.CoreC().DeletePolicyTrigger(ctx, &rmetav1.DeleteOptions{
		Uid: pt.Metadata.Uid,
	})
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func (w *Watcher) isOrphanPolicyTrigger(ctx context.Context, pt *corev1.PolicyTrigger) (bool, error) {
	if isPolicyTriggerExpired(pt) {
		return true, nil
	}

	if pt.Status.OwnerRef == nil || pt.Status.OwnerRef.Uid == "" {
		return true, nil
	}

	req, err := w.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: pt.Status.OwnerRef.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if req.Status.State == nil {
		return false, nil
	}

	switch req.Status.State.Status {
	case accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		return true, nil
	}

	return false, nil
}

func isPolicyTriggerExpired(pt *corev1.PolicyTrigger) bool {
	if pt.Status.PreCondition == nil {
		return false
	}

	all := pt.Status.PreCondition.GetAll()
	if all == nil {
		return false
	}

	now := time.Now()

	for _, itm := range all.Of {
		if itm == nil {
			continue
		}

		notAfter := itm.GetNotAfter()
		if notAfter == nil || !notAfter.IsValid() {
			continue
		}

		if !notAfter.AsTime().After(now) {
			return true
		}
	}

	return false
}

func (w *Watcher) doCheckRequest(ctx context.Context, req *accessv1.Request) error {
	if req == nil || req.Status == nil || req.Status.State == nil {
		return nil
	}

	switch req.Status.State.Status {
	case accessv1.Request_Status_State_PENDING:
		return w.doCheckPendingRequest(ctx, req)

	case accessv1.Request_Status_State_APPROVED:
		return w.doCheckApprovedRequest(ctx, req)

	default:
		return nil
	}
}

func (w *Watcher) doCheckPendingRequest(ctx context.Context, req *accessv1.Request) error {
	next := pbutils.Clone(req).(*accessv1.Request)
	now := pbutils.Now()

	if next.Spec != nil &&
		next.Spec.Deadline != nil &&
		next.Spec.Deadline.IsValid() &&
		!next.Spec.Deadline.AsTime().After(now.AsTime()) {
		zap.L().Debug("Expiring pending access Request because deadline passed", zap.Any("req", req))
		setRequestState(next, accessv1.Request_Status_State_EXPIRED)
		return w.updateRequestIfChanged(ctx, req, next)
	}

	if next.Status.Rule == nil ||
		next.Status.Rule.Action == nil ||
		next.Status.Rule.Action.GetReview() == nil ||
		next.Status.Review == nil {
		return nil
	}

	review := next.Status.Rule.Action.GetReview()
	if len(review.Steps) == 0 {
		return nil
	}

	if next.Status.Review.CurrentStep < 0 ||
		int(next.Status.Review.CurrentStep) >= len(review.Steps) {
		return nil
	}

	step := review.Steps[int(next.Status.Review.CurrentStep)]
	if step == nil || step.Timeout == nil {
		return nil
	}

	timeout := umetav1.ToDuration(step.Timeout).ToGo()
	if timeout <= 0 {
		return nil
	}

	startedAt := getCurrentReviewStepStartedAt(next)
	if startedAt == nil || !startedAt.IsValid() {
		next.Status.Review.CurrentStepStartedAt = now
		return w.updateRequestIfChanged(ctx, req, next)
	}

	if startedAt.AsTime().Add(timeout).After(now.AsTime()) {
		return nil
	}

	zap.L().Debug("Access Request review step timed out",
		zap.Any("req", req),
		zap.Int32("step", next.Status.Review.CurrentStep))

	switch step.OnTimeout {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP:
		gotoNextReviewStepOrExpire(next, review)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT:
		setRequestState(next, accessv1.Request_Status_State_REJECTED)

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_UNSET:
		setRequestState(next, accessv1.Request_Status_State_EXPIRED)

	default:
		setRequestState(next, accessv1.Request_Status_State_EXPIRED)
	}

	return w.updateRequestIfChanged(ctx, req, next)
}

func (w *Watcher) doCheckApprovedRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Status.AccessEndsAt == nil || !req.Status.AccessEndsAt.IsValid() {
		return nil
	}

	if req.Status.AccessEndsAt.AsTime().After(time.Now()) {
		return nil
	}

	zap.L().Debug("Expiring approved access Request because accessEndsAt passed", zap.Any("req", req))

	next := pbutils.Clone(req).(*accessv1.Request)
	setRequestState(next, accessv1.Request_Status_State_EXPIRED)

	return w.updateRequestIfChanged(ctx, req, next)
}

func (w *Watcher) updateRequestIfChanged(ctx context.Context, oldReq, newReq *accessv1.Request) error {
	if pbutils.IsEqual(oldReq, newReq) {
		return nil
	}

	_, err := w.octeliumC.AccessC().UpdateRequest(ctx, newReq)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func getCurrentReviewStepStartedAt(req *accessv1.Request) *timestamppb.Timestamp {
	if req.Status.Review.CurrentStepStartedAt != nil {
		return req.Status.Review.CurrentStepStartedAt
	}

	if req.Status.Review.CurrentStep == 0 {
		return req.Status.ApprovalStartAt
	}

	return nil
}

func gotoNextReviewStepOrExpire(
	req *accessv1.Request,
	review *accessv1.Policy_Spec_Rule_Action_Review,
) {
	if int(req.Status.Review.CurrentStep)+1 >= len(review.Steps) {
		setRequestState(req, accessv1.Request_Status_State_EXPIRED)
		return
	}

	req.Status.Review.CurrentStep++
	req.Status.Review.CurrentStepStartedAt = pbutils.Now()
}

func setRequestState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
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

func (w *Watcher) Run(ctx context.Context) {
	go w.runRequests(ctx)
	go w.runPolicyTriggers(ctx)
}
