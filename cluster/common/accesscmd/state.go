// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accesscmd

import (
	"time"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

const MaxRequestStates = 100

type StateChange struct {
	Status   accessv1.Request_Status_State_Status
	Reason   accessv1.Request_Status_State_Reason
	Message  string
	ActorRef *metav1.ObjectReference
}

func SetRequestState(req *accessv1.Request, change *StateChange) {
	now := pbutils.Now()

	if req.Status.State != nil &&
		req.Status.State.Status == change.Status &&
		req.Status.State.Reason == change.Reason {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		prevState := pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State)

		req.Status.LastStates = append(
			[]*accessv1.Request_Status_State{prevState},
			req.Status.LastStates...,
		)

		if len(req.Status.LastStates) > MaxRequestStates {
			req.Status.LastStates = req.Status.LastStates[:MaxRequestStates]
		}
	}

	req.Status.State = &accessv1.Request_Status_State{
		CreatedAt: now,
		Status:    change.Status,
		Reason:    change.Reason,
		Message:   change.Message,
		ActorRef:  cloneRef(change.ActorRef),
	}

	switch change.Status {
	case accessv1.Request_Status_State_APPROVED,
		accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		if req.Status.ApprovalEndAt == nil {
			req.Status.ApprovalEndAt = now
		}
	}

	if IsFinalState(change.Status) {
		EndCurrentStep(req, accessv1.Request_Status_Review_StepState_INTERRUPTED)
		endGrant(req, now.AsTime())
	}
}

func IsFinalState(status accessv1.Request_Status_State_Status) bool {
	switch status {
	case accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		return true
	default:
		return false
	}
}

func IsPending(req *accessv1.Request) bool {
	return req != nil && req.Status != nil && req.Status.State != nil &&
		req.Status.State.Status == accessv1.Request_Status_State_PENDING
}

func IsApproved(req *accessv1.Request) bool {
	return req != nil && req.Status != nil && req.Status.State != nil &&
		req.Status.State.Status == accessv1.Request_Status_State_APPROVED
}

func IsAccessActive(req *accessv1.Request, now time.Time) bool {
	if !IsApproved(req) {
		return false
	}

	if req.Status.AccessStartsAt != nil && req.Status.AccessStartsAt.AsTime().After(now) {
		return false
	}

	if req.Status.AccessEndsAt != nil && !req.Status.AccessEndsAt.AsTime().After(now) {
		return false
	}

	return true
}

func IsAwaitingRequester(req *accessv1.Request) bool {
	if !IsPending(req) || req.Status.Review == nil {
		return false
	}

	items := req.Status.Review.InformationRequests
	if len(items) == 0 {
		return false
	}

	return items[len(items)-1].RespondedAt == nil
}

func endGrant(req *accessv1.Request, now time.Time) {
	if req.Status.Grant == nil {
		return
	}

	switch req.Status.Grant.State {
	case accessv1.Request_Status_Grant_ACTIVE,
		accessv1.Request_Status_Grant_SCHEDULED,
		accessv1.Request_Status_Grant_PROVISIONING:
		req.Status.Grant.State = accessv1.Request_Status_Grant_ENDED
		req.Status.Grant.EndedAt = pbutils.Timestamp(now)
	}
}

func cloneRef(ref *metav1.ObjectReference) *metav1.ObjectReference {
	if ref == nil {
		return nil
	}

	ret := pbutils.Clone(ref).(*metav1.ObjectReference)
	ret.ResourceVersion = ""

	return ret
}
