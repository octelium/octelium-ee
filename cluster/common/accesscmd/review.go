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
	"fmt"
	"slices"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxReviewRevisions = 100

type SetReviewDecisionOpts struct {
	OcteliumC octeliumc.ClientInterface

	Reviewer *corev1.User
	Request  *accessv1.Request

	Decision      accessv1.Review_Spec_Decision
	Justification string
	Duration      *metav1.Duration

	ExpectedStepName     string
	ExpectedSpecRevision uint32

	Origin *accessv1.Origin
}

func SetReviewDecision(ctx context.Context, opts *SetReviewDecisionOpts) (*accessv1.Review, error) {
	if opts == nil || opts.OcteliumC == nil || opts.Reviewer == nil || opts.Request == nil {
		return nil, grpcutils.InvalidArg("Nil SetReviewDecision arguments")
	}

	switch opts.Decision {
	case accessv1.Review_Spec_DECISION_UNSET,
		accessv1.Review_Spec_DECISION_APPROVE,
		accessv1.Review_Spec_DECISION_REJECT:
	default:
		return nil, grpcutils.InvalidArg("Invalid Decision")
	}

	if TextLen(opts.Justification) > MaxJustificationLen {
		return nil, grpcutils.InvalidArg("Justification is too long")
	}

	req := opts.Request

	step, stepIndex, err := CurrentStep(req)
	if err != nil {
		return nil, err
	}

	if opts.ExpectedStepName != "" && opts.ExpectedStepName != StepName(step, stepIndex) {
		return nil, grpcutils.InvalidArg(
			"This Request is no longer waiting for the %s review Step", opts.ExpectedStepName)
	}

	if opts.ExpectedSpecRevision != 0 && opts.ExpectedSpecRevision != req.Status.SpecRevision {
		return nil, grpcutils.InvalidArg(
			"This Request was changed after you opened it. Review it again")
	}

	now := time.Now()

	if IsDeadlinePassed(req, now) {
		return nil, grpcutils.InvalidArg("The deadline of this Request has passed")
	}

	if IsStepTimedOut(req, now) {
		return nil, grpcutils.InvalidArg("The %s review Step has timed out", StepName(step, stepIndex))
	}

	if err := checkReasonRequirement(step, opts.Decision, opts.Justification); err != nil {
		return nil, err
	}

	if opts.Duration != nil {
		if opts.Decision != accessv1.Review_Spec_DECISION_APPROVE {
			return nil, grpcutils.InvalidArg("A duration can only be set for an approval")
		}

		if err := apivalidation.ValidateDuration(opts.Duration); err != nil {
			return nil, err
		}

		if effective := umetav1.ToDuration(req.Status.EffectiveDuration).ToGo(); effective > 0 &&
			umetav1.ToDuration(opts.Duration).ToGo() > effective {
			return nil, grpcutils.InvalidArg("The approved duration cannot exceed the requested duration")
		}
	}

	ok, err := CanReview(ctx, opts.OcteliumC, opts.Reviewer, req)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to review this Request")
	}

	item, err := GetStepReview(ctx, opts.OcteliumC, req, stepIndex, opts.Reviewer.Metadata.Uid)
	if err != nil {
		return nil, err
	}

	spec := &accessv1.Review_Spec{
		Decision:      opts.Decision,
		Justification: opts.Justification,
		Duration:      opts.Duration,
	}

	if opts.Decision == accessv1.Review_Spec_DECISION_UNSET {
		spec.Justification = ""
		spec.Duration = nil
	}

	if item == nil {
		name, err := generateReviewName(ctx, opts.OcteliumC, req.Metadata.Name)
		if err != nil {
			return nil, err
		}

		item = &accessv1.Review{
			Metadata: &metav1.Metadata{
				Name: name,
			},
			Spec: spec,
			Status: &accessv1.Review_Status{
				UserRef:      umetav1.GetObjectReference(opts.Reviewer),
				RequestRef:   umetav1.GetObjectReference(req),
				SetAt:        pbutils.Now(),
				StepIndex:    stepIndex,
				StepName:     StepName(step, stepIndex),
				Origin:       pbutils.Clone(opts.Origin).(*accessv1.Origin),
				SpecRevision: req.Status.SpecRevision,
			},
		}

		item, err = opts.OcteliumC.AccessC().CreateReview(ctx, item)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		return item, nil
	}

	if HasReviewBeenApplied(req, item) {
		return nil, grpcutils.InvalidArg("Applied Reviews can no longer be changed")
	}

	if pbutils.IsEqual(item.Spec, spec) && item.Status.SpecRevision == req.Status.SpecRevision {
		return item, nil
	}

	AppendReviewRevision(item)

	item.Spec = spec
	item.Status.SetAt = pbutils.Now()
	item.Status.StepName = StepName(step, stepIndex)
	item.Status.Origin = pbutils.Clone(opts.Origin).(*accessv1.Origin)
	item.Status.SpecRevision = req.Status.SpecRevision

	item, err = opts.OcteliumC.AccessC().UpdateReview(ctx, item)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}

func GetStepReview(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, stepIndex int32, reviewerUID string) (*accessv1.Review, error) {
	itemList, err := octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(reviewerUID),
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	for _, item := range itemList.Items {
		if item.Status.StepIndex == stepIndex {
			return item, nil
		}
	}

	return nil, nil
}

func CurrentStep(req *accessv1.Request) (*accessv1.Policy_Spec_Rule_Action_Review_Step, int32, error) {
	if req.Status.State == nil ||
		req.Status.State.Status != accessv1.Request_Status_State_PENDING {
		return nil, 0, grpcutils.InvalidArg("This Request is no longer pending")
	}

	if req.Status.Rule == nil ||
		req.Status.Rule.Action == nil ||
		req.Status.Rule.Action.GetReview() == nil {
		return nil, 0, grpcutils.InvalidArg("This Request has no review workflow")
	}

	steps := req.Status.Rule.Action.GetReview().Steps
	stepIndex := CurrentStepIndex(req)

	if stepIndex < 0 || int(stepIndex) >= len(steps) {
		return nil, 0, grpcutils.InvalidArg("This Request has an invalid current review Step")
	}

	step := steps[stepIndex]
	if step == nil {
		return nil, 0, grpcutils.InvalidArg("This Request has an invalid current review Step")
	}

	return step, stepIndex, nil
}

func StepName(step *accessv1.Policy_Spec_Rule_Action_Review_Step, stepIndex int32) string {
	if step != nil && step.Name != "" {
		return step.Name
	}

	return fmt.Sprintf("step-%d", stepIndex+1)
}

func CurrentStepIndex(req *accessv1.Request) int32 {
	if req == nil || req.Status == nil || req.Status.Review == nil {
		return 0
	}

	return req.Status.Review.CurrentStep
}

func CanReview(ctx context.Context, octeliumC octeliumc.ClientInterface,
	usr *corev1.User, req *accessv1.Request) (bool, error) {
	step, _, err := CurrentStep(req)
	if err != nil {
		return false, nil
	}

	if !IsUserEligible(usr) {
		return false, nil
	}

	if !IsSeparationOfDutiesSatisfied(step, usr, req) {
		return false, nil
	}

	stepState := CurrentStepState(req)
	if stepState != nil && stepState.TotalEligibleReviewer <= uint32(len(stepState.EligibleReviewers)) {
		for _, ref := range stepState.EligibleReviewers {
			if ref.Uid == usr.Metadata.Uid {
				return true, nil
			}
		}

		return false, nil
	}

	catalog, err := getRequestCatalog(ctx, octeliumC, req)
	if err != nil {
		return false, err
	}

	return UserMatchesAnyReviewer(ctx, octeliumC, usr, GetStepReviewers(step, stepState), catalog)
}

func IsParticipant(ctx context.Context, octeliumC octeliumc.ClientInterface,
	usr *corev1.User, req *accessv1.Request) (bool, error) {
	if usr == nil || req == nil || req.Status == nil {
		return false, nil
	}

	if req.Status.Review != nil {
		for _, itm := range req.Status.Review.LastSteps {
			if itm.UserRef != nil && itm.UserRef.Uid == usr.Metadata.Uid {
				return true, nil
			}
		}

		for _, stepState := range req.Status.Review.Steps {
			for _, ref := range stepState.EligibleReviewers {
				if ref.Uid == usr.Metadata.Uid {
					return true, nil
				}
			}
		}
	}

	revList, err := octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(usr.Metadata.Uid),
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return false, grpcutils.InternalWithErr(err)
	}
	if len(revList.Items) > 0 {
		return true, nil
	}

	return CanReview(ctx, octeliumC, usr, req)
}

func checkReasonRequirement(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	decision accessv1.Review_Spec_Decision, justification string) error {
	if justification != "" {
		return nil
	}

	switch step.ReasonRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALWAYS:
		if decision != accessv1.Review_Spec_DECISION_UNSET {
			return grpcutils.InvalidArg("A justification is required for this decision")
		}
	case accessv1.Policy_Spec_Rule_Action_Review_Step_REJECT_ONLY:
		if decision == accessv1.Review_Spec_DECISION_REJECT {
			return grpcutils.InvalidArg("A justification is required in order to reject this Request")
		}
	}

	return nil
}

func IsReasonRequired(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	decision accessv1.Review_Spec_Decision) bool {
	return checkReasonRequirement(step, decision, "") != nil
}

func IsUserEligible(usr *corev1.User) bool {
	if usr == nil || usr.Metadata == nil || usr.Metadata.Uid == "" {
		return false
	}

	if usr.Spec != nil && usr.Spec.IsDisabled {
		return false
	}

	if usr.Status != nil && usr.Status.IsLocked {
		return false
	}

	return true
}

func IsSeparationOfDutiesSatisfied(step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	usr *corev1.User, req *accessv1.Request) bool {
	sod := step.SeparationOfDuties

	if sod == nil || !sod.AllowRequesterReview {
		if req.Status.UserRef != nil && req.Status.UserRef.Uid == usr.Metadata.Uid {
			return false
		}
	}

	if sod == nil || !sod.AllowSubjectReview {
		if subjectRef := GetSubjectUserRef(req); subjectRef != nil &&
			subjectRef.Uid == usr.Metadata.Uid {
			return false
		}
	}

	return true
}

func GetSubjectUserRef(req *accessv1.Request) *metav1.ObjectReference {
	if req.Spec != nil && req.Spec.Subject != nil && req.Spec.Subject.GetUserRef() != nil {
		return req.Spec.Subject.GetUserRef()
	}

	return req.Status.UserRef
}

func UserMatchesAnyReviewer(ctx context.Context, octeliumC octeliumc.ClientInterface,
	usr *corev1.User, reviewers []*Reviewer, catalog *accessv1.Catalog) (bool, error) {
	for _, reviewer := range reviewers {
		if reviewer == nil {
			continue
		}

		ok, err := userMatchesReviewer(ctx, octeliumC, usr, reviewer, catalog)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

func userMatchesReviewer(ctx context.Context, octeliumC octeliumc.ClientInterface,
	usr *corev1.User, reviewer *Reviewer, catalog *accessv1.Catalog) (bool, error) {
	if usr == nil || usr.Metadata == nil || usr.Metadata.Uid == "" {
		return false, nil
	}

	switch reviewer.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
		ref := reviewer.GetUser().GetUserRef()
		return ref != nil && ref.Uid != "" && ref.Uid == usr.Metadata.Uid, nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
		grp, err := GetGroup(ctx, octeliumC, reviewer.GetGroup().GetGroupRef())
		if err != nil {
			return false, err
		}
		if grp == nil || usr.Spec == nil {
			return false, nil
		}

		return slices.Contains(usr.Spec.Groups, grp.Metadata.Name), nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_:
		return IsCatalogOwner(catalog, usr), nil

	default:
		return false, nil
	}
}

func GetStepReviewerUsers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	req *accessv1.Request) ([]*corev1.User, error) {
	return GetStepEligibleUsers(ctx, octeliumC, req, step, CurrentStepState(req), nil)
}

func HasReviewBeenApplied(req *accessv1.Request, review *accessv1.Review) bool {
	if req.Status.Review == nil {
		return false
	}

	for _, step := range req.Status.Review.LastSteps {
		if step.ReviewRef == nil ||
			step.ReviewRef.Uid == "" ||
			step.ReviewRef.Uid != review.Metadata.Uid {
			continue
		}

		if step.StepIndex < req.Status.Review.CurrentStep &&
			step.SpecRevision == req.Status.SpecRevision {
			return true
		}
	}

	return false
}

func AppendReviewRevision(review *accessv1.Review) {
	revisionSetAt := review.Status.SetAt
	if revisionSetAt == nil {
		revisionSetAt = pbutils.Now()
	}

	if review.Spec != nil {
		revision := &accessv1.Review_Status_Revision{
			Spec:   pbutils.Clone(review.Spec).(*accessv1.Review_Spec),
			SetAt:  revisionSetAt,
			Origin: pbutils.Clone(review.Status.Origin).(*accessv1.Origin),
		}

		review.Status.LastRevisions = append(
			[]*accessv1.Review_Status_Revision{revision},
			review.Status.LastRevisions...,
		)

		if len(review.Status.LastRevisions) > maxReviewRevisions {
			review.Status.LastRevisions = review.Status.LastRevisions[:maxReviewRevisions]
		}
	}

	if review.Status.SetAt != nil {
		review.Status.LastSetsAt = append(
			[]*timestamppb.Timestamp{review.Status.SetAt},
			review.Status.LastSetsAt...,
		)

		if len(review.Status.LastSetsAt) > maxReviewRevisions {
			review.Status.LastSetsAt = review.Status.LastSetsAt[:maxReviewRevisions]
		}
	}
}

func GetUser(ctx context.Context, octeliumC octeliumc.ClientInterface,
	ref *metav1.ObjectReference) (*corev1.User, error) {
	if ref == nil {
		return nil, nil
	}

	usr, err := octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if isUnresolvableRef(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return usr, nil
}

func GetGroup(ctx context.Context, octeliumC octeliumc.ClientInterface,
	ref *metav1.ObjectReference) (*corev1.Group, error) {
	if ref == nil {
		return nil, nil
	}

	grp, err := octeliumC.CoreC().GetGroup(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if isUnresolvableRef(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	return grp, nil
}

func isUnresolvableRef(err error) bool {
	return grpcerr.IsNotFound(err) || grpcerr.IsInvalidArg(err)
}

func generateReviewName(ctx context.Context, octeliumC octeliumc.ClientInterface,
	requestName string) (string, error) {
	const attemptsPerLength = 32

	for n := 3; n <= 8; n++ {
		for i := 0; i < attemptsPerLength; i++ {
			name := fmt.Sprintf("%s.%s", utilrand.GetRandomStringCanonical(n), requestName)

			_, err := octeliumC.AccessC().GetReview(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			if err == nil {
				continue
			}

			if grpcerr.IsNotFound(err) {
				return name, nil
			}

			return "", grpcutils.InternalWithErr(err)
		}
	}

	return "", grpcutils.Internal("Could not generate a unique Review name")
}
