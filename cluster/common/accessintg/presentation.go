// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

type PresentationDecision struct {
	Reviewer      string `json:"reviewer,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Justification string `json:"justification,omitempty"`
}

type Presentation struct {
	Purpose string `json:"purpose,omitempty"`

	RequestName string `json:"requestName,omitempty"`
	RequestUID  string `json:"requestUID,omitempty"`
	State       string `json:"state,omitempty"`

	Requester string `json:"requester,omitempty"`
	Subject   string `json:"subject,omitempty"`

	Resource      string `json:"resource,omitempty"`
	Urgency       string `json:"urgency,omitempty"`
	Justification string `json:"justification,omitempty"`

	RequestedDuration string `json:"requestedDuration,omitempty"`
	EffectiveDuration string `json:"effectiveDuration,omitempty"`

	PolicyName string `json:"policyName,omitempty"`
	RuleName   string `json:"ruleName,omitempty"`

	StepName  string `json:"stepName,omitempty"`
	StepIndex int    `json:"stepIndex,omitempty"`
	StepCount int    `json:"stepCount,omitempty"`

	ApprovalSummary string `json:"approvalSummary,omitempty"`

	Decisions []*PresentationDecision `json:"decisions,omitempty"`

	PortalURL string `json:"portalURL,omitempty"`

	IsActionable bool `json:"isActionable,omitempty"`
	IsClosed     bool `json:"isClosed,omitempty"`
}

func (p *Presentation) Revision() string {
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type BuildPresentationOpts struct {
	OcteliumC     octeliumc.ClientInterface
	Binding       *accessv1.IntegrationBinding
	Request       *accessv1.Request
	ClusterDomain string
}

func BuildPresentation(ctx context.Context, opts *BuildPresentationOpts) (*Presentation, error) {
	req := opts.Request
	binding := opts.Binding

	ret := &Presentation{
		Purpose:       binding.Spec.Purpose.String(),
		RequestName:   req.Metadata.Name,
		RequestUID:    req.Metadata.Uid,
		State:         requestState(req),
		Urgency:       urgency(req),
		Justification: req.Spec.Justification,
	}

	requester, err := accesscmd.GetUser(ctx, opts.OcteliumC, req.Status.UserRef)
	if err != nil {
		return nil, err
	}
	ret.Requester = userLabel(requester, req.Status.UserRef)

	subjectRef := accesscmd.GetSubjectUserRef(req)
	if subjectRef != nil && (req.Status.UserRef == nil || subjectRef.Uid != req.Status.UserRef.Uid) {
		subject, err := accesscmd.GetUser(ctx, opts.OcteliumC, subjectRef)
		if err != nil {
			return nil, err
		}
		ret.Subject = userLabel(subject, subjectRef)
	}

	ret.Resource = resourceLabel(req)

	ret.RequestedDuration = durationLabel(req.Spec.Duration)
	ret.EffectiveDuration = durationLabel(req.Status.EffectiveDuration)

	if req.Status.PolicyRef != nil {
		ret.PolicyName = req.Status.PolicyRef.Name
	}
	if req.Status.Rule != nil {
		ret.RuleName = req.Status.Rule.Name
	}

	switch binding.Spec.Purpose {
	case accessv1.IntegrationBinding_Spec_REVIEW_SURFACE:
		ret.PortalURL = fmt.Sprintf("https://access.octelium.%s/reviewer/requests/%s",
			opts.ClusterDomain, req.Metadata.Name)
	default:
		ret.PortalURL = fmt.Sprintf("https://access.octelium.%s/user/requests/%s",
			opts.ClusterDomain, req.Metadata.Name)
	}

	isPending := req.Status.State != nil &&
		req.Status.State.Status == accessv1.Request_Status_State_PENDING

	ret.IsClosed = !isPending

	if binding.Spec.Purpose == accessv1.IntegrationBinding_Spec_REVIEW_SURFACE {
		steps := []*accessv1.Policy_Spec_Rule_Action_Review_Step{}
		if req.Status.Rule != nil && req.Status.Rule.Action != nil &&
			req.Status.Rule.Action.GetReview() != nil {
			steps = req.Status.Rule.Action.GetReview().Steps
		}

		ret.StepCount = len(steps)
		ret.StepIndex = int(binding.Spec.StepIndex) + 1
		ret.StepName = binding.Spec.StepName

		if int(binding.Spec.StepIndex) < len(steps) {
			step := steps[binding.Spec.StepIndex]

			decisions, err := stepDecisions(ctx, opts.OcteliumC, req, binding.Spec.StepIndex)
			if err != nil {
				return nil, err
			}
			ret.Decisions = decisions

			ret.ApprovalSummary, err = approvalSummary(ctx, opts.OcteliumC, req, step, decisions)
			if err != nil {
				return nil, err
			}
		}

		ret.IsActionable = isPending &&
			accesscmd.CurrentStepIndex(req) == binding.Spec.StepIndex &&
			binding.Spec.InteractionMode == accessv1.Policy_Spec_Rule_Surface_INTERACTIVE
	}

	return ret, nil
}

func stepDecisions(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request, stepIndex int32) ([]*PresentationDecision, error) {
	itemList, err := octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, err
	}

	ret := []*PresentationDecision{}

	for _, item := range itemList.Items {
		if item.Status.StepIndex != stepIndex ||
			item.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
			continue
		}

		usr, err := accesscmd.GetUser(ctx, octeliumC, item.Status.UserRef)
		if err != nil {
			return nil, err
		}

		ret = append(ret, &PresentationDecision{
			Reviewer:      userLabel(usr, item.Status.UserRef),
			Decision:      strings.TrimPrefix(item.Spec.Decision.String(), "DECISION_"),
			Justification: item.Spec.Justification,
		})
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Reviewer < ret[j].Reviewer
	})

	return ret, nil
}

func approvalSummary(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	decisions []*PresentationDecision) (string, error) {
	approvals := 0
	for _, decision := range decisions {
		if decision.Decision == "APPROVE" {
			approvals++
		}
	}

	switch step.ApprovalRequirement {
	case accessv1.Policy_Spec_Rule_Action_Review_Step_ANY:
		return fmt.Sprintf("%d of 1 approval", approvals), nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT:
		return fmt.Sprintf("%d of %d approvals", approvals, step.ApprovalCount), nil

	case accessv1.Policy_Spec_Rule_Action_Review_Step_ALL:
		usrs, err := accesscmd.GetStepReviewerUsers(ctx, octeliumC, step, req)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d of %d approvals", approvals, len(usrs)), nil

	default:
		return "", nil
	}
}

func requestState(req *accessv1.Request) string {
	if req.Status.State == nil {
		return accessv1.Request_Status_State_STATUS_UNKNOWN.String()
	}

	return req.Status.State.Status.String()
}

func urgency(req *accessv1.Request) string {
	if req.Spec.Urgency == accessv1.Request_Spec_URGENCY_UNSET {
		return ""
	}

	return req.Spec.Urgency.String()
}

func resourceLabel(req *accessv1.Request) string {
	if req.Spec.Resource == nil {
		return ""
	}

	switch req.Spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		return fmt.Sprintf("Service %s", req.Spec.Resource.GetServiceRef().GetName())

	case *accessv1.Request_Spec_Resource_Catalog_:
		return fmt.Sprintf("Catalog %s", req.Spec.Resource.GetCatalog().GetCatalogRef().GetName())

	default:
		return ""
	}
}

func durationLabel(d *metav1.Duration) string {
	if d == nil {
		return ""
	}

	dur := umetav1.ToDuration(d).ToGo()
	if dur <= 0 {
		return ""
	}

	return dur.String()
}

func userLabel(usr *corev1.User, ref *metav1.ObjectReference) string {
	if usr == nil {
		if ref != nil {
			return ref.Name
		}
		return ""
	}

	if usr.Spec != nil && usr.Spec.Email != "" {
		return fmt.Sprintf("%s (%s)", usr.Metadata.Name, usr.Spec.Email)
	}

	return usr.Metadata.Name
}
