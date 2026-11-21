// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package requests

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const maxEvaluationAttempts = 5
const maxProvisioningAttempts = 5

const PolicyTriggerNamePrefix = accesscmd.PolicyTriggerNamePrefix

type Controller struct {
	octeliumC     octeliumc.ClientInterface
	celEngine     *celengine.CELEngine
	evaluator     *accesscmd.Evaluator
	clusterDomain string
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	celEngine *celengine.CELEngine,
) (*Controller, error) {
	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Controller{
		octeliumC:     octeliumC,
		celEngine:     celEngine,
		evaluator:     accesscmd.NewEvaluator(octeliumC, celEngine),
		clusterDomain: cc.Status.Domain,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.Request) error {
	return c.reconcile(ctx, itm)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.Request) error {
	return c.reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.Request) error {
	if err := c.deleteIntegrationBindings(ctx, itm); err != nil {
		return err
	}

	return accesscmd.DeletePolicyTriggerOf(ctx, c.octeliumC, itm)
}

func (c *Controller) Reconcile(ctx context.Context, itm *accessv1.Request) error {
	return c.reconcile(ctx, itm)
}

func (c *Controller) reconcile(ctx context.Context, itm *accessv1.Request) error {
	if err := c.doReconcile(ctx, itm); err != nil {
		return err
	}

	return c.ensureIntegrationBindings(ctx, itm)
}

func (c *Controller) doReconcile(ctx context.Context, itm *accessv1.Request) error {
	next := pbutils.Clone(itm).(*accessv1.Request)

	cc, err := accesscmd.GetClusterConfig(ctx, c.octeliumC)
	if err != nil {
		return err
	}

	switch {
	case (needsEvaluation(next) || accesscmd.IsPending(next)) && accesscmd.IsDeadlinePassed(next, time.Now()):
		accesscmd.SetRequestState(next, &accesscmd.StateChange{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_DEADLINE_PASSED,
		})
	case needsEvaluation(next):
		if err := c.evaluate(ctx, next, cc); err != nil {
			return err
		}
	case accesscmd.IsPending(next):
		if err := c.reconcilePending(ctx, next, cc); err != nil {
			return err
		}
	}

	if accesscmd.IsApproved(next) {
		c.ensureAccessWindow(next)
		c.expireIfNeeded(next)
	}

	if !pbutils.IsEqual(itm, next) {
		_, err := c.octeliumC.AccessC().UpdateRequest(ctx, next)
		return err
	}

	switch {
	case accesscmd.IsApproved(next):
		if err := c.ensurePolicyTrigger(ctx, next); err != nil {
			return err
		}

	case next.Status.State != nil && accesscmd.IsFinalState(next.Status.State.Status):
		if err := c.ensureNoPolicyTrigger(ctx, next); err != nil {
			return err
		}
	}

	if pbutils.IsEqual(itm, next) {
		return nil
	}

	_, err = c.octeliumC.AccessC().UpdateRequest(ctx, next)
	return err
}

func needsEvaluation(req *accessv1.Request) bool {
	if req.Status.State == nil ||
		req.Status.State.Status == accessv1.Request_Status_State_STATUS_UNKNOWN {
		return true
	}

	if req.Status.State.Status != accessv1.Request_Status_State_PENDING || req.Status.Rule != nil {
		return false
	}

	return req.Status.Evaluation == nil ||
		req.Status.Evaluation.State != accessv1.Request_Status_Evaluation_FAILED
}

type requestContext struct {
	requester *corev1.User
	subject   *corev1.User
	scope     *accesscmd.Scope
}

func (c *Controller) loadRequestContext(ctx context.Context, req *accessv1.Request,
	cc *accessv1.ClusterConfig) (*requestContext, *accesscmd.Problem, error) {
	ret := &requestContext{}

	requester, err := accesscmd.GetUser(ctx, c.octeliumC, req.Status.UserRef)
	if err != nil {
		return nil, nil, err
	}
	ret.requester = requester

	subject, err := accesscmd.GetUser(ctx, c.octeliumC, accesscmd.GetSubjectUserRef(req))
	if err != nil {
		return nil, nil, err
	}
	if subject == nil || !accesscmd.IsUserEligible(subject) {
		return nil, &accesscmd.Problem{
			Message: "The subject User can no longer be granted any access",
		}, nil
	}
	ret.subject = subject

	resCatalog := req.Spec.GetResource().GetCatalog()
	if resCatalog == nil {
		return nil, &accesscmd.Problem{
			Message: "The Request must be made for a Catalog",
		}, nil
	}

	catalog, err := accesscmd.GetCatalog(ctx, c.octeliumC, resCatalog.CatalogRef)
	if err != nil {
		return nil, nil, err
	}

	problem, err := accesscmd.CheckRequestable(ctx, &accesscmd.CheckRequestableOpts{
		CELEngine: c.celEngine,
		Catalog:   catalog,
		Requester: requester,
		Subject:   subject,
	})
	if err != nil {
		return nil, nil, err
	}
	if problem != nil {
		return nil, problem, nil
	}

	scope, problems, err := accesscmd.ResolveScope(ctx, c.octeliumC, &accesscmd.ResolveScopeOpts{
		Catalog:         catalog,
		AccessLevel:     resCatalog.AccessLevel,
		ServiceRefs:     resCatalog.ServiceRefs,
		DefaultPriority: accesscmd.GetDefaultAccessPriority(cc),
	})
	if err != nil {
		return nil, nil, err
	}
	if len(problems) > 0 {
		return nil, problems[0], nil
	}
	ret.scope = scope

	return ret, nil, nil
}

func (c *Controller) evaluate(ctx context.Context, req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	if req.Status.Evaluation == nil {
		req.Status.Evaluation = &accessv1.Request_Status_Evaluation{}
	}
	req.Status.Evaluation.LastAttemptAt = pbutils.Now()

	rctx, problem, err := c.loadRequestContext(ctx, req, cc)
	if err == nil && problem != nil {
		req.Status.Evaluation.State = accessv1.Request_Status_Evaluation_EVALUATED
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_REJECTED,
			Reason:  accessv1.Request_Status_State_NOT_REQUESTABLE,
			Message: problem.Message,
		})
		return nil
	}

	var res *accesscmd.EvalResult
	if err == nil {
		res, err = c.evaluator.Evaluate(ctx, &accesscmd.EvalInput{
			Request:   req,
			Requester: rctx.requester,
			Subject:   rctx.subject,
			Scope:     rctx.scope,
		}, false)
	}

	if err != nil {
		c.setEvaluationFailure(req, err)
		return nil
	}

	if blocking := res.BlockingRule(); blocking != nil {
		req.Status.Evaluation.SkippedRules = res.SkippedRules()
		c.setEvaluationFailure(req, errors.Errorf("the Rule %s of the access Policy %s could not be evaluated: %s",
			blocking.Rule.Name, blocking.Policy.Metadata.Name, blocking.Err))
		return nil
	}

	req.Status.Evaluation.State = accessv1.Request_Status_Evaluation_EVALUATED
	req.Status.Evaluation.LastError = ""
	req.Status.Evaluation.Attempts = 0
	req.Status.Evaluation.SkippedRules = res.SkippedRules()

	for _, skipped := range req.Status.Evaluation.SkippedRules {
		zap.L().Warn("Skipping an access Policy Rule that could not be evaluated",
			zap.String("request", req.Metadata.Name),
			zap.String("rule", skipped.RuleName),
			zap.String("error", skipped.Error))
	}

	req.Status.ResolvedScope = rctx.scope.Status
	req.Status.SubjectInfo = accesscmd.GetUserInfo(rctx.subject)
	if rctx.requester != nil {
		req.Status.RequesterInfo = accesscmd.GetUserInfo(rctx.requester)
	}

	if res.Rule == nil {
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_REJECTED,
			Reason:  accessv1.Request_Status_State_NO_MATCHING_RULE,
			Message: cc.GetSpec().GetRequest().GetNoMatchMessage(),
		})
		return nil
	}

	rule := pbutils.Clone(res.Rule).(*accessv1.Policy_Spec_Rule)
	if len(rule.Notifications) == 0 {
		for _, surface := range cc.GetSpec().GetNotification().GetDefaultSurfaces() {
			rule.Notifications = append(rule.Notifications,
				pbutils.Clone(surface).(*accessv1.Policy_Spec_Rule_Surface))
		}
	}

	req.Status.PolicyRef = objectRef(res.Policy)
	req.Status.Rule = rule

	workflow, err := accesscmd.BuildWorkflow(ctx, c.octeliumC, rule)
	if err != nil {
		return err
	}
	req.Status.Workflow = workflow

	switch rule.Effect {
	case accessv1.Policy_Spec_Rule_DENY:
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_REJECTED,
			Reason:  accessv1.Request_Status_State_DENIED_BY_RULE,
			Message: rule.Message,
		})

	case accessv1.Policy_Spec_Rule_REVIEW:
		c.setEffectiveDuration(req, rctx.scope.Level, rule)

		if err := validateReviewAction(rule.GetAction().GetReview()); err != nil {
			accesscmd.SetRequestState(req, &accesscmd.StateChange{
				Status:  accessv1.Request_Status_State_REJECTED,
				Reason:  accessv1.Request_Status_State_EVALUATION_FAILED,
				Message: "The matched access Policy Rule has an invalid review workflow",
			})
			return nil
		}

		if req.Status.ApprovalStartAt == nil {
			req.Status.ApprovalStartAt = pbutils.Now()
		}

		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_PENDING,
			Reason:  accessv1.Request_Status_State_AWAITING_REVIEW,
			Message: rule.Message,
		})

		if err := accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
			OcteliumC:     c.octeliumC,
			Request:       req,
			Index:         0,
			ClusterConfig: cc,
		}); err != nil {
			return err
		}

	case accessv1.Policy_Spec_Rule_AUTO_APPROVE:
		c.setEffectiveDuration(req, rctx.scope.Level, rule)

		if req.Status.ApprovalStartAt == nil {
			req.Status.ApprovalStartAt = pbutils.Now()
		}

		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_APPROVED,
			Reason:  accessv1.Request_Status_State_AUTO_APPROVED,
			Message: rule.Message,
		})

	default:
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_REJECTED,
			Reason:  accessv1.Request_Status_State_EVALUATION_FAILED,
			Message: "The matched access Policy Rule has an invalid effect",
		})
	}

	return nil
}

func (c *Controller) setEvaluationFailure(req *accessv1.Request, err error) {
	zap.L().Warn("Could not evaluate an access Request",
		zap.String("request", req.Metadata.Name), zap.Error(err))

	req.Status.Evaluation.Attempts++
	req.Status.Evaluation.LastError = err.Error()

	if req.Status.Evaluation.Attempts < maxEvaluationAttempts {
		req.Status.Evaluation.State = accessv1.Request_Status_Evaluation_RETRYING
		return
	}

	req.Status.Evaluation.State = accessv1.Request_Status_Evaluation_FAILED
	accesscmd.SetRequestState(req, &accesscmd.StateChange{
		Status:  accessv1.Request_Status_State_REJECTED,
		Reason:  accessv1.Request_Status_State_EVALUATION_FAILED,
		Message: "The Request could not be evaluated against the access Policies",
	})
}

func (c *Controller) reconcilePending(ctx context.Context, req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	rctx, problem, err := c.loadRequestContext(ctx, req, cc)
	if err != nil {
		return err
	}

	if problem != nil {
		accesscmd.SetRequestState(req, &accesscmd.StateChange{
			Status:  accessv1.Request_Status_State_REJECTED,
			Reason:  accessv1.Request_Status_State_NOT_REQUESTABLE,
			Message: problem.Message,
		})
		return nil
	}

	if req.Status.ResolvedScope != nil &&
		!accesscmd.IsScopeMateriallyEqual(req.Status.ResolvedScope, rctx.scope) {
		return c.reevaluate(ctx, req, cc)
	}

	if !accesscmd.IsScopeEqual(req.Status.ResolvedScope, rctx.scope.Status) {
		req.Status.ResolvedScope = rctx.scope.Status
	}

	c.setEffectiveDuration(req, rctx.scope.Level, req.Status.Rule)

	if accesscmd.GetReviewAction(req) == nil {
		return nil
	}

	if accesscmd.IsReviewStepStale(req) {
		accesscmd.EndCurrentStep(req, accessv1.Request_Status_Review_StepState_INTERRUPTED)
		if err := accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
			OcteliumC:     c.octeliumC,
			Request:       req,
			Index:         0,
			ClusterConfig: cc,
		}); err != nil {
			return err
		}
	}

	if accesscmd.CurrentStepState(req) == nil {
		var idx int32
		if req.Status.Review != nil {
			idx = req.Status.Review.CurrentStep
		}

		if err := accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
			OcteliumC:     c.octeliumC,
			Request:       req,
			Index:         idx,
			ClusterConfig: cc,
		}); err != nil {
			return err
		}
	}

	return accesscmd.ReconcileReviewStep(ctx, &accesscmd.ReconcileReviewStepOpts{
		OcteliumC:     c.octeliumC,
		Request:       req,
		ClusterConfig: cc,
	})
}

func (c *Controller) reevaluate(ctx context.Context, req *accessv1.Request, cc *accessv1.ClusterConfig) error {
	accesscmd.RestartEvaluation(req)

	return c.evaluate(ctx, req, cc)
}

func (c *Controller) ensureAccessWindow(req *accessv1.Request) {
	if req.Status.ApprovalEndAt == nil {
		req.Status.ApprovalEndAt = pbutils.Now()
	}

	if req.Status.AccessStartsAt == nil {
		startsAt := req.Status.ApprovalEndAt.AsTime()
		if startAt := req.Spec.GetStartAt(); startAt != nil && startAt.IsValid() && startAt.AsTime().After(startsAt) {
			startsAt = startAt.AsTime()
		}
		req.Status.AccessStartsAt = pbutils.Timestamp(startsAt)
	}

	if req.Status.AccessEndsAt == nil {
		accessDuration := umetav1.ToDuration(req.Status.EffectiveDuration).ToGo()
		if accessDuration <= 0 {
			accessDuration = accesscmd.GetEffectiveDuration(req.Spec,
				req.Status.GetResolvedScope().GetAccessLevel(), req.Status.Rule)
		}

		req.Status.AccessEndsAt = pbutils.Timestamp(req.Status.AccessStartsAt.AsTime().Add(accessDuration))
	}

	if req.Status.Grant == nil {
		req.Status.Grant = &accessv1.Request_Status_Grant{
			State: accessv1.Request_Status_Grant_PROVISIONING,
		}
	}
}

func (c *Controller) expireIfNeeded(req *accessv1.Request) {
	if req.Status.AccessEndsAt == nil {
		return
	}

	if req.Status.AccessEndsAt.AsTime().After(time.Now()) {
		return
	}

	accesscmd.SetRequestState(req, &accesscmd.StateChange{
		Status: accessv1.Request_Status_State_EXPIRED,
		Reason: accessv1.Request_Status_State_ACCESS_ENDED,
	})
}

func (c *Controller) setEffectiveDuration(req *accessv1.Request,
	level *accessv1.Catalog_Spec_AccessLevel, rule *accessv1.Policy_Spec_Rule) {
	req.Status.EffectiveDuration = accesscmd.DurationFromGo(
		accesscmd.GetEffectiveDuration(req.Spec, level, rule))
}

func (c *Controller) ensurePolicyTrigger(ctx context.Context, req *accessv1.Request) error {
	if req.Status.ResolvedScope == nil {
		return c.ensureLegacyPolicyTrigger(ctx, req)
	}

	ptDesired, err := accesscmd.BuildPolicyTrigger(req)
	if err != nil {
		c.setProvisioningFailure(req, err)
		return nil
	}

	pt, err := c.applyPolicyTrigger(ctx, req, ptDesired)
	if err != nil {
		c.setProvisioningFailure(req, err)
		return nil
	}
	if pt == nil {
		return nil
	}

	req.Status.PolicyTriggerRef = objectRef(pt)
	c.setGrantProvisioned(req)

	if req.Status.Grant.State == accessv1.Request_Status_Grant_ACTIVE {
		return c.supersedeExtendedRequest(ctx, req)
	}

	return nil
}

func (c *Controller) ensureLegacyPolicyTrigger(ctx context.Context, req *accessv1.Request) error {
	pt, err := c.octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: accesscmd.PolicyTriggerName(req),
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}

		c.setProvisioningFailure(req, errors.Errorf("the Request has no resolved scope"))
		return nil
	}

	req.Status.PolicyTriggerRef = objectRef(pt)
	c.setGrantProvisioned(req)

	return nil
}

func (c *Controller) applyPolicyTrigger(ctx context.Context,
	req *accessv1.Request, ptDesired *corev1.PolicyTrigger) (*corev1.PolicyTrigger, error) {
	ptExisting, err := c.octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: ptDesired.Metadata.Name,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return nil, err
		}

		approved, err := c.isRequestApproved(ctx, req)
		if err != nil || !approved {
			return nil, err
		}

		pt, err := c.octeliumC.CoreC().CreatePolicyTrigger(ctx, ptDesired)
		if err != nil {
			return nil, err
		}

		approved, err = c.isRequestApproved(ctx, req)
		if err != nil || !approved {
			if errDel := c.deletePolicyTrigger(ctx, objectRef(pt)); errDel != nil {
				return nil, errDel
			}
			return nil, err
		}

		return pt, nil
	}

	ptChanged := pbutils.Clone(ptExisting).(*corev1.PolicyTrigger)
	ptChanged.Status = ptDesired.Status

	if pbutils.IsEqual(ptExisting, ptChanged) {
		return ptExisting, nil
	}

	return c.octeliumC.CoreC().UpdatePolicyTrigger(ctx, ptChanged)
}

func (c *Controller) setGrantProvisioned(req *accessv1.Request) {
	if req.Status.Grant == nil {
		req.Status.Grant = &accessv1.Request_Status_Grant{}
	}

	grant := req.Status.Grant
	grant.Attempts = 0
	grant.LastError = ""

	if req.Status.AccessStartsAt != nil && req.Status.AccessStartsAt.AsTime().After(time.Now()) {
		grant.State = accessv1.Request_Status_Grant_SCHEDULED
		return
	}

	grant.State = accessv1.Request_Status_Grant_ACTIVE
	if grant.ActivatedAt == nil {
		grant.ActivatedAt = pbutils.Now()
	}
}

func (c *Controller) setProvisioningFailure(req *accessv1.Request, err error) {
	zap.L().Warn("Could not provision the access of an approved Request",
		zap.String("request", req.Metadata.Name), zap.Error(err))

	if req.Status.Grant == nil {
		req.Status.Grant = &accessv1.Request_Status_Grant{}
	}

	req.Status.Grant.Attempts++
	req.Status.Grant.LastError = err.Error()
	req.Status.Grant.State = accessv1.Request_Status_Grant_PROVISIONING

	if req.Status.Grant.Attempts < maxProvisioningAttempts {
		return
	}

	accesscmd.SetRequestState(req, &accesscmd.StateChange{
		Status:  accessv1.Request_Status_State_EXPIRED,
		Reason:  accessv1.Request_Status_State_PROVISIONING_FAILED,
		Message: "The approved access could not be activated",
	})
	req.Status.Grant.State = accessv1.Request_Status_Grant_FAILED
}

func (c *Controller) supersedeExtendedRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Spec.ExtendsRef == nil || req.Spec.ExtendsRef.Uid == "" {
		return nil
	}

	extended, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: req.Spec.ExtendsRef.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !accesscmd.IsApproved(extended) {
		return nil
	}

	if extended.Status.AccessEndsAt != nil && req.Status.AccessEndsAt != nil &&
		!req.Status.AccessEndsAt.AsTime().After(extended.Status.AccessEndsAt.AsTime()) {
		return nil
	}

	if extended.Status.ResolvedScope != nil &&
		!accesscmd.ScopeCovers(req.Status.ResolvedScope, extended.Status.ResolvedScope) {
		return nil
	}

	extended.Status.SupersededByRef = objectRef(req)
	accesscmd.SetRequestState(extended, &accesscmd.StateChange{
		Status:   accessv1.Request_Status_State_EXPIRED,
		Reason:   accessv1.Request_Status_State_SUPERSEDED,
		ActorRef: req.Status.UserRef,
	})

	_, err = c.octeliumC.AccessC().UpdateRequest(ctx, extended)
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func (c *Controller) ensureNoPolicyTrigger(ctx context.Context, req *accessv1.Request) error {
	if err := accesscmd.DeletePolicyTriggerOf(ctx, c.octeliumC, req); err != nil {
		return err
	}

	req.Status.PolicyTriggerRef = nil
	return nil
}

func (c *Controller) isRequestApproved(ctx context.Context, req *accessv1.Request) (bool, error) {
	if req.Metadata.Uid == "" {
		return true, nil
	}

	current, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return accesscmd.IsApproved(current), nil
}

func (c *Controller) deletePolicyTrigger(ctx context.Context, ref *metav1.ObjectReference) error {
	if ref == nil || ref.Uid == "" {
		return nil
	}

	_, err := c.octeliumC.CoreC().DeletePolicyTrigger(ctx, &rmetav1.DeleteOptions{
		Uid: ref.Uid,
	})
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func validateReviewAction(review *accessv1.Policy_Spec_Rule_Action_Review) error {
	if review == nil {
		return errors.Errorf("review action is nil")
	}

	if len(review.Steps) == 0 {
		return errors.Errorf("review action has no steps")
	}

	for i, step := range review.Steps {
		if step == nil {
			return errors.Errorf("review step %d is nil", i)
		}

		if len(step.Reviewers) == 0 {
			return errors.Errorf("review step %d has no reviewers", i)
		}

		switch step.ApprovalRequirement {
		case accessv1.Policy_Spec_Rule_Action_Review_Step_ANY:
			if step.ApprovalCount != 0 {
				return errors.Errorf("review step %d with ANY must not set approvalCount", i)
			}

		case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
			if step.ApprovalCount != 0 {
				return errors.Errorf("review step %d with ALL must not set approvalCount", i)
			}

		case accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT:
			if step.ApprovalCount == 0 {
				return errors.Errorf("review step %d with COUNT must set approvalCount", i)
			}

		case accessv1.Policy_Spec_Rule_Action_Review_Step_APPROVAL_REQUIREMENT_UNSET:
			return errors.Errorf("review step %d has unset approvalRequirement", i)

		default:
			return errors.Errorf("review step %d has invalid approvalRequirement", i)
		}

		switch step.OnTimeout {
		case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_UNSET,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT:
		default:
			return errors.Errorf("review step %d has invalid onTimeout", i)
		}
	}

	return nil
}

func objectRef(itm umetav1.ResourceObjectI) *metav1.ObjectReference {
	return cloneRef(umetav1.GetObjectReference(itm))
}

func cloneRef(ref *metav1.ObjectReference) *metav1.ObjectReference {
	if ref == nil {
		return nil
	}

	ret := pbutils.Clone(ref).(*metav1.ObjectReference)
	ret.ResourceVersion = ""

	return ret
}

func policyTriggerName(req *accessv1.Request) string {
	return accesscmd.PolicyTriggerName(req)
}
