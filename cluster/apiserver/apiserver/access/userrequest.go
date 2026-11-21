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
	"fmt"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

const expiringSoonWithin = time.Hour

func (s *ServerUser) CreateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	return s.doCreateRequest(ctx, req, false)
}

func (s *ServerUser) CreateRequestForSubject(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	return s.doCreateRequest(ctx, req, true)
}

func (s *ServerUser) doCreateRequest(ctx context.Context, req *accessv1.Request, forSubject bool) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{},
	}); err != nil {
		return nil, err
	}

	celEngine, err := s.cel.get(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	item, err := accesscmd.CreateRequest(ctx, &accesscmd.CreateRequestOpts{
		OcteliumC:  s.octeliumC,
		CELEngine:  celEngine,
		Requester:  i.User,
		Spec:       req.Spec,
		ForSubject: forSubject,
		Origin:     apiOrigin(i),
	})
	if err != nil {
		return nil, err
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerUser) PreviewRequest(ctx context.Context, req *accessv1.Request) (*accessv1.RequestPreview, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil {
		return nil, grpcutils.InvalidArg("Nil Request")
	}

	celEngine, err := s.cel.get(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	prepared, err := accesscmd.PrepareRequest(ctx, &accesscmd.PrepareRequestOpts{
		OcteliumC:  s.octeliumC,
		CELEngine:  celEngine,
		Requester:  i.User,
		Spec:       req.Spec,
		ForSubject: isForSubject(i.User, req.Spec),
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	res, err := buildPreview(ctx, s.octeliumC, celEngine, prepared, false)
	if err != nil {
		return nil, err
	}

	return res.preview, nil
}

func isForSubject(usr *corev1.User, spec *accessv1.Request_Spec) bool {
	ref := spec.GetSubject().GetUserRef()
	if ref == nil {
		return false
	}

	if ref.Uid != "" {
		return ref.Uid != usr.Metadata.Uid
	}

	return ref.Name != "" && ref.Name != usr.Metadata.Name
}

func (s *ServerUser) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkUserCanReadRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerUser) ListRequest(ctx context.Context, req *accessv1.ListUserRequestOptions) (*accessv1.RequestList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	var filters []*rmetav1.ListOptions_Filter

	if req.State != accessv1.Request_Status_State_STATUS_UNKNOWN {
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.state.status", req.State.String()))
	}

	if req.Urgency != accessv1.Request_Spec_URGENCY_UNSET {
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("spec.urgency", req.Urgency.String()))
	}

	filters, err = appendRefFilter(filters, "spec.resource.catalog.catalogRef", req.CatalogRef,
		&apivalidation.CheckGetOptionsOpts{})
	if err != nil {
		return nil, err
	}

	serviceRef := req.ServiceRef
	if serviceRef != nil {
		if err := apivalidation.CheckObjectRef(serviceRef, &apivalidation.CheckGetOptionsOpts{
			ParentsMax: 1,
		}); err != nil && (serviceRef.Uid != "" || serviceRef.Name != "") {
			return nil, err
		}

		if serviceRef.Uid != "" && serviceRef.Name == "" {
			svc, err := s.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Uid: serviceRef.Uid})
			if err != nil && !grpcerr.IsNotFound(err) {
				return nil, serr.InternalWithErr(err)
			}
			if svc != nil {
				serviceRef = umetav1.GetObjectReference(svc)
			}
		}
	}

	userUID := i.User.Metadata.Uid

	var created, forMe []*accessv1.Request

	switch req.Relation {
	case accessv1.ListUserRequestOptions_CREATED_BY_ME:
		created, err = listAllRequests(ctx, s.octeliumC,
			append([]*rmetav1.ListOptions_Filter{urscsrv.FilterStatusUserUID(userUID)}, filters...)...)
	case accessv1.ListUserRequestOptions_FOR_ME:
		forMe, err = listAllRequests(ctx, s.octeliumC,
			append([]*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", userUID)}, filters...)...)
	default:
		created, err = listAllRequests(ctx, s.octeliumC,
			append([]*rmetav1.ListOptions_Filter{urscsrv.FilterStatusUserUID(userUID)}, filters...)...)
		if err == nil {
			forMe, err = listAllRequests(ctx, s.octeliumC,
				append([]*rmetav1.ListOptions_Filter{
					urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", userUID)}, filters...)...)
		}
	}
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	now := time.Now()
	var items []*accessv1.Request

	for _, itm := range mergeRequests(created, forMe) {
		if req.IsActive && !accesscmd.IsAccessActive(itm, now) {
			continue
		}

		if !requestIncludesService(itm, serviceRef) {
			continue
		}

		if !matchesRequestQuery(itm, req.Query) {
			continue
		}

		items = append(items, accesscmd.RedactRequest(itm))
	}

	return paginateRequests(items, req.Common), nil
}

func (s *ServerUser) UpdateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if req.Spec == nil {
		return nil, grpcutils.InvalidArg("Nil Spec")
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkUserOwnsRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if err := checkUserRequestPending(item); err != nil {
		return nil, err
	}

	if req.Spec.Resource != nil && !pbutils.IsEqual(item.Spec.Resource, req.Spec.Resource) {
		return nil, grpcutils.InvalidArg("Cannot change Request resource")
	}

	if req.Spec.Subject != nil && !pbutils.IsEqual(item.Spec.Subject, req.Spec.Subject) {
		return nil, grpcutils.InvalidArg("Cannot change Request subject")
	}

	if req.Spec.ExtendsRef != nil && !pbutils.IsEqual(item.Spec.ExtendsRef, req.Spec.ExtendsRef) {
		return nil, grpcutils.InvalidArg("Cannot change the extended Request")
	}

	catalog, err := accesscmd.GetCatalog(ctx, s.octeliumC, item.Spec.GetResource().GetCatalog().GetCatalogRef())
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	cc, err := accesscmd.GetClusterConfig(ctx, s.octeliumC)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	next := pbutils.Clone(item).(*accessv1.Request)
	next.Spec.Urgency = req.Spec.Urgency
	next.Spec.Justification = req.Spec.Justification
	next.Spec.Ticket = req.Spec.Ticket
	next.Spec.Deadline = req.Spec.Deadline
	next.Spec.StartAt = req.Spec.StartAt
	next.Spec.Duration = req.Spec.Duration

	if next.Spec.Deadline == nil {
		next.Spec.Deadline = item.Spec.Deadline
	}

	if err := accesscmd.ProblemsToError(accesscmd.ValidateRequestFields(next.Spec, catalog, cc)); err != nil {
		return nil, err
	}

	if pbutils.IsEqual(item.Spec, next.Spec) {
		return accesscmd.RedactRequest(item), nil
	}

	if isMaterialSpecChange(item, next) {
		next.Status.SpecRevision++
	}

	if next.Status.Rule != nil {
		next.Status.EffectiveDuration = accesscmd.DurationFromGo(accesscmd.GetEffectiveDuration(next.Spec,
			next.Status.GetResolvedScope().GetAccessLevel(), next.Status.Rule))
	}

	if accesscmd.HasStaleAppliedSteps(next) {
		accesscmd.EndCurrentStep(next, accessv1.Request_Status_Review_StepState_INTERRUPTED)
		if err := accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
			OcteliumC:     s.octeliumC,
			Request:       next,
			Index:         0,
			ClusterConfig: cc,
		}); err != nil {
			return nil, serr.InternalWithErr(err)
		}
	}

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, next)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return accesscmd.RedactRequest(item), nil
}

func isMaterialSpecChange(prev, next *accessv1.Request) bool {
	if prev.Spec.Justification != next.Spec.Justification ||
		prev.Spec.Ticket != next.Spec.Ticket ||
		!pbutils.IsEqual(prev.Spec.StartAt, next.Spec.StartAt) {
		return true
	}

	level := prev.Status.GetResolvedScope().GetAccessLevel()

	return accesscmd.GetEffectiveDuration(next.Spec, level, prev.Status.Rule) >
		accesscmd.GetEffectiveDuration(prev.Spec, level, prev.Status.Rule)
}

func (s *ServerUser) CancelRequest(ctx context.Context, req *accessv1.CancelRequestRequest) (*metav1.OperationResult, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

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

	if err := checkUserCanReadRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if item.Status.State != nil &&
		item.Status.State.Status == accessv1.Request_Status_State_CANCELLED {
		return &metav1.OperationResult{}, nil
	}

	if err := checkUserRequestPending(item); err != nil {
		return nil, err
	}

	accesscmd.SetRequestState(item, &accesscmd.StateChange{
		Status:   accessv1.Request_Status_State_CANCELLED,
		Reason:   accessv1.Request_Status_State_CANCELLED_BY_USER,
		Message:  req.Reason,
		ActorRef: umetav1.GetObjectReference(i.User),
	})

	if _, err := s.octeliumC.AccessC().UpdateRequest(ctx, item); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerUser) EndAccess(ctx context.Context, req *accessv1.EndAccessRequest) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

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

	if err := checkUserCanReadRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if item.Status.State != nil &&
		item.Status.State.Reason == accessv1.Request_Status_State_ENDED_EARLY {
		if err := accesscmd.DeletePolicyTriggerOf(ctx, s.octeliumC, item); err != nil {
			return nil, serr.InternalWithErr(err)
		}

		return accesscmd.RedactRequest(item), nil
	}

	if !accesscmd.IsApproved(item) {
		return nil, grpcutils.InvalidArg("Only the access of approved Requests can be ended")
	}

	accesscmd.SetRequestState(item, &accesscmd.StateChange{
		Status:   accessv1.Request_Status_State_EXPIRED,
		Reason:   accessv1.Request_Status_State_ENDED_EARLY,
		Message:  req.Reason,
		ActorRef: umetav1.GetObjectReference(i.User),
	})

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if err := accesscmd.DeletePolicyTriggerOf(ctx, s.octeliumC, item); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerUser) RespondToRequest(ctx context.Context, req *accessv1.RespondToRequestRequest) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if req.Response == "" {
		return nil, grpcutils.InvalidArg("Response must be set")
	}

	if accesscmd.TextLen(req.Response) > accesscmd.MaxJustificationLen {
		return nil, grpcutils.InvalidArg("Response is too long")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkUserCanReadRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if !accesscmd.IsAwaitingRequester(item) {
		return nil, grpcutils.InvalidArg("This Request is not waiting for any information")
	}

	infoReqs := item.Status.Review.InformationRequests
	last := infoReqs[len(infoReqs)-1]
	last.Response = req.Response
	last.RespondedAt = pbutils.Now()
	last.ResponderRef = umetav1.GetObjectReference(i.User)

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerUser) GetInbox(ctx context.Context, req *accessv1.GetInboxRequest) (*accessv1.Inbox, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	userUID := i.User.Metadata.Uid
	ret := &accessv1.Inbox{}

	pending, err := listAllRequests(ctx, s.octeliumC,
		urscsrv.FilterFieldEQValStr("status.state.status", accessv1.Request_Status_State_PENDING.String()))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	for _, itm := range pending {
		isMine := isRequestCreator(userUID, itm) || isRequestSubject(userUID, itm)
		if isMine {
			ret.TotalMyPending++
			if accesscmd.IsAwaitingRequester(itm) && isRequestCreator(userUID, itm) {
				ret.TotalAwaitingMyResponse++
			}
		}

		ok, err := accesscmd.CanReview(ctx, s.octeliumC, i.User, itm)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		decided, err := hasDecidedCurrentStep(ctx, s.octeliumC, userUID, itm)
		if err != nil {
			return nil, err
		}
		if !decided {
			ret.TotalNeedsMyDecision++
		}
	}

	approved, err := listAllRequests(ctx, s.octeliumC,
		urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", userUID),
		urscsrv.FilterFieldEQValStr("status.state.status", accessv1.Request_Status_State_APPROVED.String()))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	now := time.Now()
	for _, itm := range approved {
		if !accesscmd.IsAccessActive(itm, now) {
			continue
		}

		ret.TotalActiveAccess++
		if itm.Status.AccessEndsAt != nil && itm.Status.AccessEndsAt.AsTime().Before(now.Add(expiringSoonWithin)) {
			ret.TotalExpiringSoon++
		}
	}

	isReviewer, err := s.isReviewerAnywhere(ctx, i.User)
	if err != nil {
		return nil, err
	}
	ret.IsReviewer = isReviewer

	catalogList, err := s.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	for _, catalog := range catalogList.Items {
		if !catalog.GetSpec().GetIsDisabled() && accesscmd.CanRequestOnBehalf(catalog, i.User) {
			ret.CanRequestOnBehalf = true
			break
		}
	}

	return ret, nil
}

func (s *ServerUser) isReviewerAnywhere(ctx context.Context, usr *corev1.User) (bool, error) {
	polList, err := s.octeliumC.AccessC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return false, serr.InternalWithErr(err)
	}

	catalogList, err := s.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return false, serr.InternalWithErr(err)
	}

	isOwner := false
	for _, catalog := range catalogList.Items {
		if accesscmd.IsCatalogOwner(catalog, usr) {
			isOwner = true
			break
		}
	}

	cc, err := accesscmd.GetClusterConfig(ctx, s.octeliumC)
	if err != nil {
		return false, serr.InternalWithErr(err)
	}

	ok, err := accesscmd.UserMatchesAnyReviewer(ctx, s.octeliumC, usr,
		cc.GetSpec().GetReview().GetFallbackReviewers(), nil)
	if err != nil || ok {
		return ok, err
	}

	for _, pol := range polList.Items {
		if pol.GetSpec().GetIsDisabled() {
			continue
		}

		for _, rule := range pol.GetSpec().GetRules() {
			for _, step := range rule.GetAction().GetReview().GetSteps() {
				reviewers := append(append([]*accesscmd.Reviewer{}, step.Reviewers...),
					step.GetEscalation().GetReviewers()...)

				for _, reviewer := range reviewers {
					if reviewer.GetCatalogOwners() != nil && isOwner {
						return true, nil
					}
				}

				ok, err := accesscmd.UserMatchesAnyReviewer(ctx, s.octeliumC, usr, reviewers, nil)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func appendRefFilter(filters []*rmetav1.ListOptions_Filter,
	field string, ref *metav1.ObjectReference,
	opts *apivalidation.CheckGetOptionsOpts) ([]*rmetav1.ListOptions_Filter, error) {
	if ref == nil || (ref.Uid == "" && ref.Name == "") {
		return filters, nil
	}

	if err := apivalidation.CheckObjectRef(ref, opts); err != nil {
		return nil, err
	}

	if ref.Uid != "" {
		return append(filters, urscsrv.FilterFieldEQValStr(fmt.Sprintf("%s.uid", field), ref.Uid)), nil
	}

	return append(filters, urscsrv.FilterFieldEQValStr(fmt.Sprintf("%s.name", field), ref.Name)), nil
}

func withDefaultOrder(listOpts *rmetav1.ListOptions) *rmetav1.ListOptions {
	if len(listOpts.OrderBy) > 0 {
		return listOpts
	}

	listOpts.OrderBy = []*rmetav1.ListOptions_OrderBy{
		{
			Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
			Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
		},
	}

	return listOpts
}

func isRequestCreator(userUID string, req *accessv1.Request) bool {
	return req.Status.UserRef != nil && req.Status.UserRef.Uid == userUID
}

func isRequestSubject(userUID string, req *accessv1.Request) bool {
	ref := accesscmd.GetSubjectUserRef(req)
	return ref != nil && ref.Uid == userUID
}

func checkUserOwnsRequest(userUID string, req *accessv1.Request) error {
	if !isRequestCreator(userUID, req) {
		return grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	return nil
}

func checkUserCanReadRequest(userUID string, req *accessv1.Request) error {
	if !isRequestCreator(userUID, req) && !isRequestSubject(userUID, req) {
		return grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	return nil
}

func checkUserRequestPending(req *accessv1.Request) error {
	if req.Status.State == nil ||
		req.Status.State.Status == accessv1.Request_Status_State_STATUS_UNKNOWN ||
		req.Status.State.Status == accessv1.Request_Status_State_PENDING {
		return nil
	}

	return grpcutils.InvalidArg("Only pending Requests can be updated or cancelled")
}
