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
	"os"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/requests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

var accessWatchInterval = durationFromEnv(
	"OCTELIUM_NOCTURNE_ACCESS_WATCH_INTERVAL", 5*time.Minute)

var integrationBindingWatchInterval = durationFromEnv(
	"OCTELIUM_NOCTURNE_INTEGRATION_BINDING_WATCH_INTERVAL", time.Minute)

type BindingReconciler interface {
	Reconcile(ctx context.Context, itm *accessv1.IntegrationBinding) error
}

type RequestReconciler interface {
	Reconcile(ctx context.Context, itm *accessv1.Request) error
}

type AccessConfigReconciler interface {
	ResyncAll(ctx context.Context) error
}

type Opts struct {
	OcteliumC    octeliumc.ClientInterface
	BindingCtl   BindingReconciler
	RequestCtl   RequestReconciler
	AccessConfig AccessConfigReconciler
}

type Watcher struct {
	octeliumC    octeliumc.ClientInterface
	bindingCtl   BindingReconciler
	requestCtl   RequestReconciler
	accessConfig AccessConfigReconciler
}

func InitWatcher(o *Opts) *Watcher {
	return &Watcher{
		octeliumC:    o.OcteliumC,
		bindingCtl:   o.BindingCtl,
		requestCtl:   o.RequestCtl,
		accessConfig: o.AccessConfig,
	}
}

func (w *Watcher) runRequests(ctx context.Context) {
	zap.L().Debug("Starting access Request watcher")

	doRun := func() error {
		reqList, err := w.listRequests(ctx)
		if err != nil {
			return err
		}

		cc, err := accesscmd.GetClusterConfig(ctx, w.octeliumC)
		if err != nil {
			return err
		}

		for _, req := range reqList {
			if err := w.doCheckRequest(ctx, req, cc); err != nil {
				zap.L().Warn("Could not doCheckRequest", zap.Error(err), zap.Any("req", req))
			}
		}

		if w.accessConfig != nil {
			if err := w.accessConfig.ResyncAll(ctx); err != nil {
				zap.L().Warn("Could not resync the access configuration status", zap.Error(err))
			}
		}

		return nil
	}

	tickerCh := time.NewTicker(accessWatchInterval)
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

	tickerCh := time.NewTicker(accessWatchInterval)
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

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	val := os.Getenv(name)
	if val == "" {
		return fallback
	}

	ret, err := time.ParseDuration(val)
	if err != nil || ret <= 0 {
		return fallback
	}

	return ret
}

func (w *Watcher) doCheckRequest(ctx context.Context, req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	if req == nil || req.Status == nil || req.Status.State == nil {
		return nil
	}

	switch req.Status.State.Status {
	case accessv1.Request_Status_State_PENDING:
		return w.doCheckPendingRequest(ctx, req, cc)

	case accessv1.Request_Status_State_APPROVED:
		return w.doCheckApprovedRequest(ctx, req)

	default:
		return nil
	}
}

func (w *Watcher) doCheckPendingRequest(ctx context.Context, req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	next := pbutils.Clone(req).(*accessv1.Request)
	now := time.Now()

	if accesscmd.IsDeadlinePassed(next, now) {
		zap.L().Debug("Expiring pending access Request because deadline passed", zap.Any("req", req))
		accesscmd.SetRequestState(next, &accesscmd.StateChange{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_DEADLINE_PASSED,
		})
		return w.updateRequestIfChanged(ctx, req, next)
	}

	if next.Status.Rule == nil {
		if w.requestCtl != nil && next.Status.Evaluation != nil &&
			next.Status.Evaluation.State == accessv1.Request_Status_Evaluation_RETRYING {
			return w.requestCtl.Reconcile(ctx, next)
		}
		return nil
	}

	review := accesscmd.GetReviewAction(next)
	if review == nil || next.Status.Review == nil || len(review.Steps) == 0 {
		return nil
	}

	if next.Status.Review.CurrentStep < 0 ||
		int(next.Status.Review.CurrentStep) >= len(review.Steps) {
		return nil
	}

	step := review.Steps[int(next.Status.Review.CurrentStep)]
	if step == nil {
		return nil
	}

	if next.Status.Review.CurrentStepStartedAt == nil && next.Status.Review.CurrentStep != 0 {
		next.Status.Review.CurrentStepStartedAt = pbutils.Now()
		return w.updateRequestIfChanged(ctx, req, next)
	}

	if accesscmd.IsStepTimedOut(next, now) {
		zap.L().Debug("Access Request review step timed out",
			zap.Any("req", req),
			zap.Int32("step", next.Status.Review.CurrentStep))

		if err := w.onStepTimeout(ctx, next, step, review, cc); err != nil {
			return err
		}

		return w.updateRequestIfChanged(ctx, req, next)
	}

	if accesscmd.ShouldEscalateStep(next, now) {
		if err := accesscmd.EscalateStep(ctx, w.octeliumC, next, cc); err != nil {
			return err
		}

		return w.updateRequestIfChanged(ctx, req, next)
	}

	if err := accesscmd.ReconcileReviewStep(ctx, &accesscmd.ReconcileReviewStepOpts{
		OcteliumC:     w.octeliumC,
		Request:       next,
		ClusterConfig: cc,
	}); err != nil {
		return err
	}

	return w.updateRequestIfChanged(ctx, req, next)
}

func (w *Watcher) onStepTimeout(ctx context.Context, req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	review *accessv1.Policy_Spec_Rule_Action_Review, cc *accessv1.ClusterConfig) error {
	accesscmd.EndCurrentStep(req, accessv1.Request_Status_Review_StepState_TIMED_OUT)

	switch step.OnTimeout {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP:
		if int(req.Status.Review.CurrentStep)+1 >= len(review.Steps) {
			accesscmd.SetRequestState(req, &accesscmd.StateChange{
				Status: accessv1.Request_Status_State_EXPIRED,
				Reason: accessv1.Request_Status_State_REVIEW_TIMED_OUT,
			})
			return nil
		}

		return accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
			OcteliumC:     w.octeliumC,
			Request:       req,
			Index:         req.Status.Review.CurrentStep + 1,
			ClusterConfig: cc,
		})

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT:
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status: accessv1.Request_Status_State_REJECTED,
			Reason: accessv1.Request_Status_State_REVIEW_TIMED_OUT,
		})

	default:
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_REVIEW_TIMED_OUT,
		})
	}

	return nil
}

func (w *Watcher) doCheckApprovedRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Status.AccessEndsAt != nil && req.Status.AccessEndsAt.IsValid() &&
		!req.Status.AccessEndsAt.AsTime().After(time.Now()) {
		zap.L().Debug("Expiring approved access Request because accessEndsAt passed", zap.Any("req", req))

		next := pbutils.Clone(req).(*accessv1.Request)
		accesscmd.SetRequestState(next, &accesscmd.StateChange{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_ACCESS_ENDED,
		})

		return w.updateRequestIfChanged(ctx, req, next)
	}

	if w.requestCtl == nil {
		return nil
	}

	grant := req.Status.Grant
	switch {
	case grant == nil,
		grant.State == accessv1.Request_Status_Grant_PROVISIONING,
		grant.State == accessv1.Request_Status_Grant_SCHEDULED:
		return w.requestCtl.Reconcile(ctx, req)
	default:
		return nil
	}
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

func (w *Watcher) Run(ctx context.Context) {
	go w.runRequests(ctx)
	go w.runPolicyTriggers(ctx)
	go w.runIntegrationBindings(ctx)
}

func (w *Watcher) runIntegrationBindings(ctx context.Context) {
	zap.L().Debug("Starting access IntegrationBinding watcher")

	doRun := func() error {
		itmList, err := w.listIntegrationBindings(ctx)
		if err != nil {
			return err
		}

		for _, itm := range itmList {
			if !isIntegrationBindingDue(itm) {
				continue
			}

			if err := w.bindingCtl.Reconcile(ctx, itm); err != nil {
				zap.L().Warn("Could not reconcile an IntegrationBinding",
					zap.String("binding", itm.Metadata.Name), zap.Error(err))
			}
		}

		return nil
	}

	tickerCh := time.NewTicker(integrationBindingWatchInterval)
	defer tickerCh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh.C:
			if err := doRun(); err != nil {
				zap.L().Error("Could not run access IntegrationBinding watcher doFn",
					zap.Error(err))
			}
		}
	}
}

func (w *Watcher) listIntegrationBindings(ctx context.Context) ([]*accessv1.IntegrationBinding, error) {
	var ret []*accessv1.IntegrationBinding
	var page uint32

	for {
		itmList, err := w.octeliumC.AccessC().ListIntegrationBinding(ctx, &rmetav1.ListOptions{
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

func isIntegrationBindingDue(itm *accessv1.IntegrationBinding) bool {
	if itm == nil || itm.Status == nil {
		return false
	}

	if itm.Status.State == accessv1.IntegrationBinding_Status_CLOSED &&
		itm.Status.AppliedRevision == itm.Status.DesiredRevision {
		return false
	}

	if itm.Status.State == accessv1.IntegrationBinding_Status_READY &&
		itm.Status.AppliedRevision == itm.Status.DesiredRevision {
		return false
	}

	if itm.Status.NextAttemptAt != nil && itm.Status.NextAttemptAt.IsValid() &&
		itm.Status.NextAttemptAt.AsTime().After(time.Now()) {
		return false
	}

	return true
}
