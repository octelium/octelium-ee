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
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
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
)

func (s *ServerReviewer) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
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

	ok, err := accesscmd.IsParticipant(ctx, s.octeliumC, i.User, item)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerReviewer) GetReviewCase(ctx context.Context, req *accessv1.GetReviewCaseRequest) (*accessv1.ReviewCase, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	ok, err := accesscmd.IsParticipant(ctx, s.octeliumC, i.User, item)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	ret := &accessv1.ReviewCase{
		Request: accesscmd.RedactRequest(item),
		Flags: &accessv1.ReviewCase_Flags{
			IsOnBehalf: item.Status.UserRef != nil && accesscmd.GetSubjectUserRef(item) != nil &&
				item.Status.UserRef.Uid != accesscmd.GetSubjectUserRef(item).Uid,
			OverridesGuardrails: item.Status.GetResolvedScope().GetOverridesGuardrails(),
			IsExtension:         item.Spec.ExtendsRef != nil,
			IsFullAccess:        accesscmd.IsFullAccessLevel(item.Status.GetResolvedScope().GetAccessLevel()),
			IsAwaitingRequester: accesscmd.IsAwaitingRequester(item),
		},
	}

	revList, err := s.octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", item.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}
	ret.Reviews = revList.Items

	canReview, err := accesscmd.CanReview(ctx, s.octeliumC, i.User, item)
	if err != nil {
		return nil, err
	}

	stepIndex := accesscmd.CurrentStepIndex(item)
	for _, rev := range revList.Items {
		if rev.Status.UserRef != nil && rev.Status.UserRef.Uid == i.User.Metadata.Uid &&
			rev.Status.StepIndex == stepIndex {
			ret.MyReview = rev
			if accesscmd.IsPending(item) &&
				rev.Spec.Decision != accessv1.Review_Spec_DECISION_UNSET &&
				rev.Status.SpecRevision != item.Status.SpecRevision {
				ret.Flags.IsSpecChanged = true
			}
		}
	}

	if canReview {
		ret.AllowedActions = append(ret.AllowedActions,
			accessv1.ReviewCase_APPROVE,
			accessv1.ReviewCase_REJECT,
			accessv1.ReviewCase_REQUEST_INFORMATION)

		if ret.MyReview != nil && ret.MyReview.Spec.Decision != accessv1.Review_Spec_DECISION_UNSET {
			ret.AllowedActions = append(ret.AllowedActions, accessv1.ReviewCase_WITHDRAW)
		}
	}

	subject, err := accesscmd.GetUser(ctx, s.octeliumC, accesscmd.GetSubjectUserRef(item))
	if err != nil {
		return nil, err
	}
	if subject != nil && subject.Spec != nil {
		ret.SubjectGroups = subject.Spec.Groups
	}

	if catalogRef := item.Spec.GetResource().GetCatalog().GetCatalogRef(); catalogRef != nil &&
		catalogRef.Uid != "" && accesscmd.GetSubjectUserRef(item) != nil {
		history, err := listAllRequests(ctx, s.octeliumC,
			urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", accesscmd.GetSubjectUserRef(item).Uid),
			urscsrv.FilterFieldEQValStr("spec.resource.catalog.catalogRef.uid", catalogRef.Uid))
		if err != nil {
			return nil, serr.InternalWithErr(err)
		}

		now := time.Now()
		hasPrevious := false

		for _, itm := range history {
			if itm.Metadata.Uid == item.Metadata.Uid {
				continue
			}

			if itm.Metadata.CreatedAt.AsTime().Before(item.Metadata.CreatedAt.AsTime()) {
				hasPrevious = true
			}

			if accesscmd.IsAccessActive(itm, now) {
				ret.ActiveRequests = append(ret.ActiveRequests, accesscmd.RedactRequest(itm))
			}

			if len(ret.RecentRequests) < maxReviewCaseRecentRequests {
				ret.RecentRequests = append(ret.RecentRequests, accesscmd.RedactRequest(itm))
			}
		}

		ret.Flags.IsFirstRequest = !hasPrevious
	}

	return ret, nil
}

const maxReviewCaseRecentRequests = 10

func (s *ServerReviewer) ListRequest(ctx context.Context,
	req *accessv1.ListReviewerRequestOptions) (*accessv1.RequestList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	userUID := i.User.Metadata.Uid

	var filters []*rmetav1.ListOptions_Filter
	if req.Filter != accessv1.ListReviewerRequestOptions_DECIDED_BY_ME {
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.state.status", accessv1.Request_Status_State_PENDING.String()))
	}

	if req.Urgency != accessv1.Request_Spec_URGENCY_UNSET {
		filters = append(filters, urscsrv.FilterFieldEQValStr("spec.urgency", req.Urgency.String()))
	}

	candidates, err := listAllRequests(ctx, s.octeliumC, filters...)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	var reviewed map[string]struct{}
	if req.Filter == accessv1.ListReviewerRequestOptions_DECIDED_BY_ME {
		reviewed, err = s.getReviewedRequestUIDs(ctx, userUID)
		if err != nil {
			return nil, err
		}
	}

	var items []*accessv1.Request

	for _, item := range candidates {
		if !matchesRequestQuery(item, req.Query) {
			continue
		}

		switch req.Filter {
		case accessv1.ListReviewerRequestOptions_DECIDED_BY_ME:
			if _, ok := reviewed[item.Metadata.Uid]; !ok {
				continue
			}

		default:
			ok, err := s.canReviewRequest(ctx, i.User, item)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			decided, err := hasDecidedCurrentStep(ctx, s.octeliumC, userUID, item)
			if err != nil {
				return nil, err
			}

			if req.Filter == accessv1.ListReviewerRequestOptions_WAITING_ON_OTHERS {
				if !decided {
					continue
				}
			} else if decided {
				continue
			}
		}

		items = append(items, accesscmd.RedactRequest(item))
	}

	return paginateRequests(items, req.Common), nil
}

func (s *ServerReviewer) RequestInformation(ctx context.Context,
	req *accessv1.RequestInformationRequest) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if req.Message == "" {
		return nil, grpcutils.InvalidArg("Message must be set")
	}

	if accesscmd.TextLen(req.Message) > accesscmd.MaxJustificationLen {
		return nil, grpcutils.InvalidArg("Message is too long")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	ok, err := s.canReviewRequest(ctx, i.User, item)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to review this Request")
	}

	if accesscmd.IsAwaitingRequester(item) {
		return nil, grpcutils.InvalidArg("This Request is already waiting for its requester")
	}

	item.Status.Review.InformationRequests = append(item.Status.Review.InformationRequests,
		&accessv1.Request_Status_Review_InformationRequest{
			UserRef:     umetav1.GetObjectReference(i.User),
			Message:     req.Message,
			RequestedAt: pbutils.Now(),
		})

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return accesscmd.RedactRequest(item), nil
}

func (s *ServerReviewer) getReviewedRequestUIDs(ctx context.Context, userUID string) (map[string]struct{}, error) {
	revList, err := s.octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(userUID),
		},
	})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	ret := map[string]struct{}{}
	for _, rev := range revList.Items {
		if rev.Status.RequestRef != nil && rev.Spec.Decision != accessv1.Review_Spec_DECISION_UNSET {
			ret[rev.Status.RequestRef.Uid] = struct{}{}
		}
	}

	return ret, nil
}

func hasDecidedCurrentStep(ctx context.Context, octeliumC octeliumc.ClientInterface,
	userUID string, req *accessv1.Request) (bool, error) {
	rev, err := accesscmd.GetStepReview(ctx, octeliumC, req, accesscmd.CurrentStepIndex(req), userUID)
	if err != nil {
		return false, err
	}

	return rev != nil &&
		rev.Spec.Decision != accessv1.Review_Spec_DECISION_UNSET &&
		rev.Status.SpecRevision == req.Status.SpecRevision, nil
}

func (s *ServerReviewer) canReviewRequest(ctx context.Context,
	usr *corev1.User, req *accessv1.Request) (bool, error) {
	return accesscmd.CanReview(ctx, s.octeliumC, usr, req)
}
