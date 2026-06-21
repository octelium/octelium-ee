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
	"fmt"
	"sort"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	return &Controller{
		octeliumC: octeliumC,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.Request) error {
	return c.reconcile(ctx, itm)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.Request) error {
	return c.reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.Request) error {
	if itm.Status.PolicyTriggerRef == nil {
		return nil
	}

	return c.deletePolicyTrigger(ctx, itm.Status.PolicyTriggerRef)
}

func (c *Controller) reconcile(ctx context.Context, itm *accessv1.Request) error {
	next := pbutils.Clone(itm).(*accessv1.Request)

	if next.Status.State == nil ||
		next.Status.State.Status == accessv1.Request_Status_State_STATUS_UNKNOWN {
		if err := c.initializeRequest(ctx, next); err != nil {
			return err
		}
	}

	switch next.Status.State.Status {
	case accessv1.Request_Status_State_PENDING:
		if next.Status.Rule == nil {
			if err := c.initializeRequest(ctx, next); err != nil {
				return err
			}
		}

	case accessv1.Request_Status_State_APPROVED:
		if err := c.ensurePolicyTrigger(ctx, next); err != nil {
			return err
		}

	case accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		if err := c.ensureNoPolicyTrigger(ctx, next); err != nil {
			return err
		}
	}

	if pbutils.IsEqual(itm, next) {
		return nil
	}

	_, err := c.octeliumC.AccessC().UpdateRequest(ctx, next)
	return err
}

func (c *Controller) initializeRequest(ctx context.Context, req *accessv1.Request) error {
	pol, rule, err := c.matchPolicyRule(ctx, req)
	if err != nil {
		return err
	}

	if rule == nil {
		c.setState(req, accessv1.Request_Status_State_REJECTED)
		return nil
	}

	req.Status.PolicyRef = umetav1.GetObjectReference(pol)
	req.Status.Rule = pbutils.Clone(rule).(*accessv1.Policy_Spec_Rule)

	switch rule.Effect {
	case accessv1.Policy_Spec_Rule_DENY:
		c.setState(req, accessv1.Request_Status_State_REJECTED)

	case accessv1.Policy_Spec_Rule_REVIEW:
		if rule.Action == nil || rule.Action.GetReview() == nil || len(rule.Action.GetReview().Steps) == 0 {
			return errors.Errorf("matched review rule %q has no review steps", rule.Name)
		}

		c.setState(req, accessv1.Request_Status_State_PENDING)
		if req.Status.ApprovalStartAt == nil {
			req.Status.ApprovalStartAt = pbutils.Now()
		}
		if req.Status.Review == nil {
			req.Status.Review = &accessv1.Request_Status_Review{
				CurrentStep: 0,
			}
		}

	case accessv1.Policy_Spec_Rule_AUTO_APPROVE:
		c.setState(req, accessv1.Request_Status_State_APPROVED)
		if req.Status.ApprovalStartAt == nil {
			req.Status.ApprovalStartAt = pbutils.Now()
		}
		if req.Status.ApprovalEndAt == nil {
			req.Status.ApprovalEndAt = pbutils.Now()
		}
		if err := c.ensurePolicyTrigger(ctx, req); err != nil {
			return err
		}

	default:
		return errors.Errorf("matched rule %q has invalid effect", rule.Name)
	}

	return nil
}

func (c *Controller) matchPolicyRule(ctx context.Context, req *accessv1.Request) (*accessv1.Policy, *accessv1.Policy_Spec_Rule, error) {
	polList, err := c.octeliumC.AccessC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	policies := append([]*accessv1.Policy{}, polList.Items...)
	sort.SliceStable(policies, func(i, j int) bool {
		return policies[i].Metadata.Name < policies[j].Metadata.Name
	})

	for _, pol := range policies {
		if pol.Spec.IsDisabled {
			continue
		}

		rules := append([]*accessv1.Policy_Spec_Rule{}, pol.Spec.Rules...)
		sort.SliceStable(rules, func(i, j int) bool {
			if rules[i].Priority == rules[j].Priority {
				return false
			}
			return rules[i].Priority > rules[j].Priority
		})

		for _, rule := range rules {
			matched, err := c.matchCondition(ctx, req, rule.Condition)
			if err != nil {
				return nil, nil, err
			}
			if !matched {
				continue
			}

			return pol, rule, nil
		}
	}

	return nil, nil, nil
}

func (c *Controller) matchCondition(ctx context.Context, req *accessv1.Request, cond *accessv1.Policy_Spec_Rule_Condition) (bool, error) {
	if cond == nil {
		return false, nil
	}

	switch cond.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_MatchAny:
		return cond.GetMatchAny(), nil

	case *accessv1.Policy_Spec_Rule_Condition_Subject_:
		return c.matchSubject(req, cond.GetSubject()), nil

	case *accessv1.Policy_Spec_Rule_Condition_Resource_:
		return c.matchResource(req, cond.GetResource()), nil

	case *accessv1.Policy_Spec_Rule_Condition_All_:
		arg := cond.GetAll()
		if len(arg.Of) == 0 {
			return false, nil
		}
		for _, sub := range arg.Of {
			matched, err := c.matchCondition(ctx, req, sub)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil

	case *accessv1.Policy_Spec_Rule_Condition_Any_:
		arg := cond.GetAny()
		if len(arg.Of) == 0 {
			return false, nil
		}
		for _, sub := range arg.Of {
			matched, err := c.matchCondition(ctx, req, sub)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil

	case *accessv1.Policy_Spec_Rule_Condition_UserRef:
		return refEqual(req.Status.UserRef, cond.GetUserRef()), nil

	case *accessv1.Policy_Spec_Rule_Condition_Match:
		return false, errors.Errorf("access request Policy CEL match conditions are not supported by this controller yet")

	default:
		return false, nil
	}
}

func (c *Controller) matchSubject(req *accessv1.Request, subject *accessv1.Policy_Spec_Rule_Condition_Subject) bool {
	if subject == nil {
		return false
	}

	switch subject.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Subject_UserRef:
		return refEqual(req.Spec.Subject.GetUserRef(), subject.GetUserRef())

	case *accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef:
		return false

	default:
		return false
	}
}

func (c *Controller) matchResource(req *accessv1.Request, resource *accessv1.Policy_Spec_Rule_Condition_Resource) bool {
	if resource == nil {
		return false
	}

	switch resource.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef:
		return refEqual(req.Spec.Resource.GetServiceRef(), resource.GetServiceRef())

	case *accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef:
		return refEqual(req.Spec.Resource.GetCatalog().GetCatalogRef(), resource.GetCatalogRef())

	default:
		return false
	}
}

func (c *Controller) ensurePolicyTrigger(ctx context.Context, req *accessv1.Request) error {
	if req.Status.Rule == nil {
		return nil
	}

	authz := req.Status.Rule.Authorization

	ptDesired, err := c.buildPolicyTrigger(req, authz)
	if err != nil {
		return err
	}

	ptExisting, err := c.octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: ptDesired.Metadata.Name,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}

		pt, err := c.octeliumC.CoreC().CreatePolicyTrigger(ctx, ptDesired)
		if err != nil {
			return err
		}

		req.Status.PolicyTriggerRef = umetav1.GetObjectReference(pt)
		return nil
	}

	ptChanged := pbutils.Clone(ptExisting).(*corev1.PolicyTrigger)
	ptChanged.Status = ptDesired.Status

	if !pbutils.IsEqual(ptExisting, ptChanged) {
		pt, err := c.octeliumC.CoreC().UpdatePolicyTrigger(ctx, ptChanged)
		if err != nil {
			return err
		}

		req.Status.PolicyTriggerRef = umetav1.GetObjectReference(pt)
		return nil
	}

	req.Status.PolicyTriggerRef = umetav1.GetObjectReference(ptExisting)
	return nil
}

func (c *Controller) ensureNoPolicyTrigger(ctx context.Context, req *accessv1.Request) error {
	if req.Status.PolicyTriggerRef == nil {
		return nil
	}

	if err := c.deletePolicyTrigger(ctx, req.Status.PolicyTriggerRef); err != nil {
		return err
	}

	req.Status.PolicyTriggerRef = nil
	return nil
}

func (c *Controller) buildPolicyTrigger(req *accessv1.Request, authz *accessv1.Policy_Spec_Rule_Authorization) (*corev1.PolicyTrigger, error) {
	subjectRef := req.Spec.Subject.GetUserRef()
	if subjectRef == nil {
		subjectRef = req.Status.UserRef
	}
	if subjectRef == nil {
		return nil, errors.Errorf("request %q has no subject or requester userRef", req.Metadata.Name)
	}

	preConditionItems := []*corev1.PolicyTrigger_Status_PreCondition{
		{
			Type: &corev1.PolicyTrigger_Status_PreCondition_UserRef{
				UserRef: pbutils.Clone(subjectRef).(*metav1.ObjectReference),
			},
		},
	}

	accessDuration := c.getAccessDuration(req, authz)
	if accessDuration > 0 {
		preConditionItems = append(preConditionItems, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_NotAfter{
				NotAfter: pbutils.Timestamp(pbutils.Now().AsTime().Add(accessDuration)),
			},
		})
	}

	return &corev1.PolicyTrigger{
		Metadata: &metav1.Metadata{
			Name:           policyTriggerName(req),
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec: &corev1.PolicyTrigger_Spec{},
		Status: &corev1.PolicyTrigger_Status{
			OwnerRef: umetav1.GetObjectReference(req),
			PreCondition: &corev1.PolicyTrigger_Status_PreCondition{
				Type: &corev1.PolicyTrigger_Status_PreCondition_All_{
					All: &corev1.PolicyTrigger_Status_PreCondition_All{
						Of: preConditionItems,
					},
				},
			},
			Policies:       append([]string{}, authz.Policies...),
			InlinePolicies: cloneInlinePolicies(authz.InlinePolicies),
			IsDisabled:     false,
		},
	}, nil
}

func (c *Controller) deletePolicyTrigger(ctx context.Context, ref *metav1.ObjectReference) error {
	if ref == nil {
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

func (c *Controller) setState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
	now := pbutils.Now()

	if req.Status.State != nil && req.Status.State.Status == status {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		req.Status.LastStates = append(req.Status.LastStates,
			pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State))
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

func (c *Controller) getAccessDuration(req *accessv1.Request, authz *accessv1.Policy_Spec_Rule_Authorization) time.Duration {
	reqDuration := umetav1.ToDuration(req.Spec.Duration).ToGo()
	maxDuration := umetav1.ToDuration(authz.MaxAccessDuration).ToGo()

	switch {
	case reqDuration <= 0 && maxDuration <= 0:
		return 0
	case reqDuration <= 0:
		return maxDuration
	case maxDuration <= 0:
		return reqDuration
	case reqDuration > maxDuration:
		return maxDuration
	default:
		return reqDuration
	}
}

func cloneInlinePolicies(in []*corev1.InlinePolicy) []*corev1.InlinePolicy {
	if len(in) == 0 {
		return nil
	}

	ret := make([]*corev1.InlinePolicy, 0, len(in))
	for _, itm := range in {
		if itm == nil {
			continue
		}
		ret = append(ret, pbutils.Clone(itm).(*corev1.InlinePolicy))
	}
	return ret
}

func refEqual(a, b *metav1.ObjectReference) bool {
	return a != nil && b != nil && a.Uid != "" && b.Uid != "" && a.Uid == b.Uid
}

func policyTriggerName(req *accessv1.Request) string {
	if req.Metadata.Uid != "" {
		return fmt.Sprintf("access-request-%s", req.Metadata.Uid)
	}

	return fmt.Sprintf("access-request-%s", req.Metadata.Name)
}
