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

const MaxJustificationLen = 1500

const maxReviewRevisions = 100

type SetReviewDecisionOpts struct {
	OcteliumC octeliumc.ClientInterface

	Reviewer *corev1.User
	Request  *accessv1.Request

	Decision      accessv1.Review_Spec_Decision
	Justification string

	ExpectedStepName string

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

	if len(opts.Justification) > MaxJustificationLen {
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
	}

	if opts.Decision == accessv1.Review_Spec_DECISION_UNSET {
		spec.Justification = ""
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
				UserRef:    umetav1.GetObjectReference(opts.Reviewer),
				RequestRef: umetav1.GetObjectReference(req),
				SetAt:      pbutils.Now(),
				StepIndex:  stepIndex,
				StepName:   StepName(step, stepIndex),
				Origin:     pbutils.Clone(opts.Origin).(*accessv1.Origin),
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

	if pbutils.IsEqual(item.Spec, spec) {
		return item, nil
	}

	AppendReviewRevision(item)

	item.Spec = spec
	item.Status.SetAt = pbutils.Now()
	item.Status.StepName = StepName(step, stepIndex)
	item.Status.Origin = pbutils.Clone(opts.Origin).(*accessv1.Origin)

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

	return UserMatchesAnyReviewer(ctx, octeliumC, umetav1.GetObjectReference(usr), step.Reviewers)
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
	userRef *metav1.ObjectReference,
	reviewers []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) (bool, error) {
	for _, reviewer := range reviewers {
		if reviewer == nil {
			continue
		}

		ok, err := userMatchesReviewer(ctx, octeliumC, userRef, reviewer)
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
	userRef *metav1.ObjectReference,
	reviewer *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) (bool, error) {
	if userRef == nil || userRef.Uid == "" {
		return false, nil
	}

	switch reviewer.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
		ref := reviewer.GetUser().GetUserRef()
		return ref != nil && ref.Uid != "" && ref.Uid == userRef.Uid, nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
		grp, err := GetGroup(ctx, octeliumC, reviewer.GetGroup().GetGroupRef())
		if err != nil {
			return false, err
		}
		if grp == nil {
			return false, nil
		}

		usr, err := GetUser(ctx, octeliumC, userRef)
		if err != nil {
			return false, err
		}
		if usr == nil {
			return false, nil
		}

		for _, groupName := range usr.Spec.Groups {
			if groupName == grp.Metadata.Name {
				return true, nil
			}
		}

		return false, nil

	default:
		return false, nil
	}
}

func GetStepReviewerUsers(ctx context.Context, octeliumC octeliumc.ClientInterface,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	req *accessv1.Request) ([]*corev1.User, error) {
	seen := map[string]struct{}{}
	ret := []*corev1.User{}

	addUser := func(usr *corev1.User) {
		if usr == nil || !IsUserEligible(usr) {
			return
		}

		if !IsSeparationOfDutiesSatisfied(step, usr, req) {
			return
		}

		if _, ok := seen[usr.Metadata.Uid]; ok {
			return
		}
		seen[usr.Metadata.Uid] = struct{}{}

		ret = append(ret, usr)
	}

	for _, reviewer := range step.Reviewers {
		if reviewer == nil {
			continue
		}

		switch reviewer.Type.(type) {
		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
			usr, err := GetUser(ctx, octeliumC, reviewer.GetUser().GetUserRef())
			if err != nil {
				return nil, err
			}
			addUser(usr)

		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
			grp, err := GetGroup(ctx, octeliumC, reviewer.GetGroup().GetGroupRef())
			if err != nil {
				return nil, err
			}
			if grp == nil {
				continue
			}

			usrList, err := octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
				Filters: []*rmetav1.ListOptions_Filter{
					urscsrv.FilterFieldIncludesValStr("spec.groups", grp.Metadata.Name),
				},
			})
			if err != nil {
				return nil, err
			}

			for _, usr := range usrList.Items {
				addUser(usr)
			}
		}
	}

	return ret, nil
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

		if step.StepIndex != req.Status.Review.CurrentStep {
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
