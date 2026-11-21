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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

func (s *ServerMain) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListRequest(ctx context.Context, req *accessv1.ListRequestOptions) (*accessv1.RequestList, error) {
	itemList, err := s.octeliumC.AccessC().ListRequest(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeleteRequest(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) RevokeRequest(ctx context.Context, req *accessv1.RevokeRequestRequest) (*metav1.OperationResult, error) {
	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if accesscmd.TextLen(req.Reason) > accesscmd.MaxJustificationLen {
		return nil, grpcutils.InvalidArg("Reason is too long")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if item.Status.State != nil &&
		item.Status.State.Status == accessv1.Request_Status_State_REVOKED {
		return &metav1.OperationResult{}, nil
	}

	if err := checkRequestRevocable(item); err != nil {
		return nil, err
	}

	change := &accesscmd.StateChange{
		Status:  accessv1.Request_Status_State_REVOKED,
		Reason:  accessv1.Request_Status_State_ACCESS_REVOKED,
		Message: req.Reason,
	}

	if i, err := userctx.GetUserCtx(ctx); err == nil && i.User != nil {
		change.ActorRef = umetav1.GetObjectReference(i.User)
	}

	accesscmd.SetRequestState(item, change)

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if err := accesscmd.DeletePolicyTriggerOf(ctx, s.octeliumC, item); err != nil {
		zap.L().Warn("Could not delete the PolicyTrigger of a revoked Request",
			zap.String("request", item.Metadata.Name), zap.Error(err))
	}

	return &metav1.OperationResult{}, nil
}

func checkRequestRevocable(req *accessv1.Request) error {
	if req.Status.State == nil {
		return nil
	}

	switch req.Status.State.Status {
	case accessv1.Request_Status_State_STATUS_UNKNOWN,
		accessv1.Request_Status_State_PENDING,
		accessv1.Request_Status_State_APPROVED:
		return nil
	}

	return grpcutils.InvalidArg("Only pending or approved Requests can be revoked")
}

func (s *ServerMain) SimulateRequest(ctx context.Context,
	req *accessv1.SimulateRequestRequest) (*accessv1.SimulateRequestResponse, error) {
	if req == nil || req.Spec == nil {
		return nil, grpcutils.InvalidArg("Spec must be set")
	}

	subjectRef := req.Spec.GetSubject().GetUserRef()
	if subjectRef == nil {
		return nil, grpcutils.InvalidArg("The subject must be set")
	}

	if err := apivalidation.CheckObjectRef(subjectRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	subject, err := accesscmd.GetUser(ctx, s.octeliumC, subjectRef)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, grpcutils.NotFound("The subject User does not exist")
	}

	requester := subject
	if req.RequesterRef != nil {
		if err := apivalidation.CheckObjectRef(req.RequesterRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		requester, err = accesscmd.GetUser(ctx, s.octeliumC, req.RequesterRef)
		if err != nil {
			return nil, err
		}
		if requester == nil {
			return nil, grpcutils.NotFound("The requester User does not exist")
		}
	}

	celEngine, err := s.cel.get(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	prepared, err := accesscmd.PrepareRequest(ctx, &accesscmd.PrepareRequestOpts{
		OcteliumC:    s.octeliumC,
		CELEngine:    celEngine,
		Requester:    requester,
		Spec:         req.Spec,
		ForSubject:   requester.Metadata.Uid != subject.Metadata.Uid,
		SkipDefaults: true,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	res, err := buildPreview(ctx, s.octeliumC, celEngine, prepared, true)
	if err != nil {
		return nil, err
	}

	ret := &accessv1.SimulateRequestResponse{
		Preview: res.preview,
	}

	if res.eval == nil {
		return ret, nil
	}

	for _, itm := range res.eval.Rules {
		evaluation := &accessv1.SimulateRequestResponse_RuleEvaluation{
			PolicyRef: umetav1.GetObjectReference(itm.Policy),
			RuleName:  itm.Rule.Name,
			Priority:  itm.Rule.Priority,
			Effect:    itm.Rule.Effect,
			Result:    itm.Result,
		}

		if itm.Err != nil {
			evaluation.Error = itm.Err.Error()
		}

		ret.Rules = append(ret.Rules, evaluation)
	}

	if res.eval.Rule == nil {
		return ret, nil
	}

	ret.Rule = res.eval.Rule
	ret.PolicyRef = umetav1.GetObjectReference(res.eval.Policy)

	simReq := res.request
	simReq.Metadata = &metav1.Metadata{
		Name: "simulation",
	}

	for idx, step := range res.eval.Rule.GetAction().GetReview().GetSteps() {
		stepState := &accessv1.Request_Status_Review_StepState{
			Name:  accesscmd.StepName(step, int32(idx)),
			Index: int32(idx),
		}

		usrs, err := accesscmd.GetStepEligibleUsers(ctx, s.octeliumC, simReq, step, stepState, nil)
		if err != nil {
			return nil, serr.InternalWithErr(err)
		}

		stepState.TotalEligibleReviewer = uint32(len(usrs))
		stepState.IsUnroutable = len(usrs) == 0
		stepState.RequiredApprovals = accesscmd.GetRequiredApprovals(step, stepState.TotalEligibleReviewer)

		for i, usr := range usrs {
			if i >= accesscmd.MaxStepEligibleReviewers {
				break
			}
			stepState.EligibleReviewers = append(stepState.EligibleReviewers, umetav1.GetObjectReference(usr))
		}

		ret.Reviewers = append(ret.Reviewers, stepState)
	}

	if res.eval.Rule.Effect != accessv1.Policy_Spec_Rule_DENY {
		now := time.Now()
		startsAt := now
		if startAt := req.Spec.GetStartAt(); startAt != nil && startAt.IsValid() && startAt.AsTime().After(now) {
			startsAt = startAt.AsTime()
		}

		simReq.Status.AccessStartsAt = pbutils.Timestamp(startsAt)
		simReq.Status.AccessEndsAt = pbutils.Timestamp(startsAt.Add(
			umetav1.ToDuration(res.preview.EffectiveDuration).ToGo()))

		pt, err := accesscmd.BuildPolicyTrigger(simReq)
		if err == nil {
			ret.PolicyTrigger = pt
		}
	}

	return ret, nil
}

func (s *ServerMain) validateRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Status == nil {
		return grpcutils.InvalidArg("Nil Status")
	}

	if req.Spec.Resource == nil {
		return grpcutils.InvalidArg("Resource must be set")
	}

	switch req.Spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		if err := apivalidation.CheckObjectRef(req.Spec.Resource.GetServiceRef(), &apivalidation.CheckGetOptionsOpts{
			ParentsMax: 1,
		}); err != nil {
			return err
		}
	case *accessv1.Request_Spec_Resource_Catalog_:
		if req.Spec.Resource.GetCatalog() == nil {
			return grpcutils.InvalidArg("Catalog resource must be set")
		}
		if err := apivalidation.CheckObjectRef(req.Spec.Resource.GetCatalog().CatalogRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}
	default:
		return grpcutils.InvalidArg("Resource type must be set")
	}

	if req.Spec.Subject != nil {
		switch req.Spec.Subject.Type.(type) {
		case *accessv1.Request_Spec_Subject_UserRef:
			if err := apivalidation.CheckObjectRef(req.Spec.Subject.GetUserRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}
		default:
			return grpcutils.InvalidArg("Subject type must be set")
		}
	}

	switch req.Spec.Urgency {
	case accessv1.Request_Spec_URGENCY_UNSET,
		accessv1.Request_Spec_VERY_LOW,
		accessv1.Request_Spec_LOW,
		accessv1.Request_Spec_NORMAL,
		accessv1.Request_Spec_HIGH,
		accessv1.Request_Spec_VERY_HIGH,
		accessv1.Request_Spec_HIGHEST:
	default:
		return grpcutils.InvalidArg("Invalid Urgency")
	}

	return nil
}
