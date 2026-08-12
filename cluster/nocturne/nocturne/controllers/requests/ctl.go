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
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

const maxRequestStateHistory = 100
const defaultAccessRequestAllowPriority = -1

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
	return c.deletePolicyTriggerOf(ctx, itm)
}

func (c *Controller) reconcile(ctx context.Context, itm *accessv1.Request) error {
	next := pbutils.Clone(itm).(*accessv1.Request)

	if next.Status.State == nil ||
		next.Status.State.Status == accessv1.Request_Status_State_STATUS_UNKNOWN {
		if err := c.initializeRequest(ctx, next); err != nil {
			return err
		}
	}

	if next.Status.State.Status == accessv1.Request_Status_State_APPROVED {
		c.ensureAccessEndsAt(next)
		c.expireIfNeeded(next)
	}

	if !pbutils.IsEqual(itm, next) {
		_, err := c.octeliumC.AccessC().UpdateRequest(ctx, next)
		return err
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
		if rule.Action == nil || rule.Action.GetReview() == nil {
			return errors.Errorf("matched review rule %q has no review action", rule.Name)
		}

		if err := validateReviewAction(rule.Action.GetReview()); err != nil {
			return errors.Wrapf(err, "matched review rule %q is invalid", rule.Name)
		}

		c.setState(req, accessv1.Request_Status_State_PENDING)

		if req.Status.ApprovalStartAt == nil {
			req.Status.ApprovalStartAt = pbutils.Now()
		}

		if req.Status.Review == nil {
			req.Status.Review = &accessv1.Request_Status_Review{
				CurrentStep:          0,
				CurrentStepStartedAt: pbutils.Now(),
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
		if pol == nil || pol.Spec.IsDisabled {
			continue
		}

		rules := append([]*accessv1.Policy_Spec_Rule{}, pol.Spec.Rules...)
		sort.SliceStable(rules, func(i, j int) bool {
			if rules[i] == nil || rules[j] == nil {
				return false
			}
			if rules[i].Priority == rules[j].Priority {
				return false
			}
			return rules[i].Priority < rules[j].Priority
		})

		for _, rule := range rules {
			if rule == nil {
				continue
			}

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
		return c.matchSubject(ctx, req, cond.GetSubject())

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

func (c *Controller) matchSubject(
	ctx context.Context,
	req *accessv1.Request,
	subject *accessv1.Policy_Spec_Rule_Condition_Subject,
) (bool, error) {
	if subject == nil {
		return false, nil
	}

	switch subject.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Subject_UserRef:
		return refEqual(getRequestSubjectUserRef(req), subject.GetUserRef()), nil

	case *accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef:
		return c.matchSubjectGroup(ctx, getRequestSubjectUserRef(req), subject.GetGroupRef())

	default:
		return false, nil
	}
}

func (c *Controller) matchSubjectGroup(
	ctx context.Context,
	userRef *metav1.ObjectReference,
	groupRef *metav1.ObjectReference,
) (bool, error) {
	if userRef == nil || userRef.Uid == "" || groupRef == nil {
		return false, nil
	}

	grp, err := c.getGroup(ctx, groupRef)
	if err != nil {
		return false, err
	}
	if grp == nil {
		return false, nil
	}

	usr, err := c.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: userRef.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, groupName := range usr.Spec.Groups {
		if groupName == grp.Metadata.Name {
			return true, nil
		}
	}

	return false, nil
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

	ptDesired, err := c.buildPolicyTrigger(ctx, req, req.Status.Rule.Authorization)
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

		exists, err := c.requestExists(ctx, req)
		if err != nil {
			return err
		}
		if !exists {
			return c.deletePolicyTrigger(ctx, umetav1.GetObjectReference(pt))
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
	if err := c.deletePolicyTriggerOf(ctx, req); err != nil {
		return err
	}

	req.Status.PolicyTriggerRef = nil
	return nil
}

func (c *Controller) requestExists(ctx context.Context, req *accessv1.Request) (bool, error) {
	if req.Metadata.Uid == "" {
		return true, nil
	}

	if _, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: req.Metadata.Uid,
	}); err != nil {
		if grpcerr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (c *Controller) buildPolicyTrigger(
	ctx context.Context,
	req *accessv1.Request,
	authz *accessv1.Policy_Spec_Rule_Authorization,
) (*corev1.PolicyTrigger, error) {
	subjectRef := getRequestSubjectUserRef(req)
	if subjectRef == nil {
		return nil, errors.Errorf("request %q has no subject or requester userRef", req.Metadata.Name)
	}

	targetPreCondition, err := c.buildTargetPreCondition(ctx, req)
	if err != nil {
		return nil, err
	}

	preConditionItems := []*corev1.PolicyTrigger_Status_PreCondition{
		{
			Type: &corev1.PolicyTrigger_Status_PreCondition_UserRef{
				UserRef: pbutils.Clone(subjectRef).(*metav1.ObjectReference),
			},
		},
		targetPreCondition,
	}

	if req.Status.AccessEndsAt != nil {
		preConditionItems = append(preConditionItems, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_NotAfter{
				NotAfter: pbutils.Timestamp(req.Status.AccessEndsAt.AsTime()),
			},
		})
	}

	policies := []string{}
	inlinePolicies := []*corev1.InlinePolicy{
		buildDefaultAccessRequestInlinePolicy(),
	}

	if authz != nil {
		policies = clonePolicyNames(authz.Policies)
		inlinePolicies = append(inlinePolicies, cloneInlinePolicies(authz.InlinePolicies)...)
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
			Policies:       policies,
			InlinePolicies: inlinePolicies,
			IsDisabled:     false,
		},
	}, nil
}

func (c *Controller) buildTargetPreCondition(ctx context.Context, req *accessv1.Request) (*corev1.PolicyTrigger_Status_PreCondition, error) {
	if req.Spec.Resource == nil {
		return nil, errors.Errorf("request %q has no resource", req.Metadata.Name)
	}

	switch req.Spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		ref := req.Spec.Resource.GetServiceRef()
		if ref == nil {
			return nil, errors.Errorf("request %q has no serviceRef", req.Metadata.Name)
		}

		return servicePreCondition(ref), nil

	case *accessv1.Request_Spec_Resource_Catalog_:
		catalogRef := req.Spec.Resource.GetCatalog().GetCatalogRef()
		if catalogRef == nil {
			return nil, errors.Errorf("request %q has no catalogRef", req.Metadata.Name)
		}

		catalog, err := c.getCatalog(ctx, catalogRef)
		if err != nil {
			return nil, err
		}
		if catalog == nil {
			return nil, errors.Errorf("request %q target Catalog does not exist", req.Metadata.Name)
		}

		return c.buildCatalogTargetPreCondition(ctx, req, catalog)

	default:
		return nil, errors.Errorf("request %q has invalid resource type", req.Metadata.Name)
	}
}

func (c *Controller) buildCatalogTargetPreCondition(
	ctx context.Context,
	req *accessv1.Request,
	catalog *accessv1.Catalog,
) (*corev1.PolicyTrigger_Status_PreCondition, error) {
	if catalog.Spec.ResourceCollection == nil ||
		catalog.Spec.ResourceCollection.Service == nil {
		return nil, errors.Errorf("request %q target Catalog %q has no Service resource collection",
			req.Metadata.Name, catalog.Metadata.Name)
	}

	var preConditions []*corev1.PolicyTrigger_Status_PreCondition

	for _, svcName := range catalog.Spec.ResourceCollection.Service.Services {
		if svcName == "" {
			continue
		}

		svc, err := c.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Name: vutils.GetServiceFullNameFromName(svcName),
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		preConditions = append(preConditions, servicePreCondition(umetav1.GetObjectReference(svc)))
	}

	for _, nsName := range catalog.Spec.ResourceCollection.Service.Namespaces {
		if nsName == "" {
			continue
		}

		ns, err := c.octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
			Name: nsName,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		preConditions = append(preConditions, namespacePreCondition(umetav1.GetObjectReference(ns)))
	}

	if len(preConditions) == 0 {
		return nil, errors.Errorf("request %q target Catalog %q does not resolve to any Service or Namespace",
			req.Metadata.Name, catalog.Metadata.Name)
	}

	if len(preConditions) == 1 {
		return preConditions[0], nil
	}

	return &corev1.PolicyTrigger_Status_PreCondition{
		Type: &corev1.PolicyTrigger_Status_PreCondition_Any_{
			Any: &corev1.PolicyTrigger_Status_PreCondition_Any{
				Of: preConditions,
			},
		},
	}, nil
}

func (c *Controller) deletePolicyTriggerOf(ctx context.Context, req *accessv1.Request) error {
	_, err := c.octeliumC.CoreC().DeletePolicyTrigger(ctx, &rmetav1.DeleteOptions{
		Name: policyTriggerName(req),
	})
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
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

func (c *Controller) ensureAccessEndsAt(req *accessv1.Request) {
	if req.Status.AccessEndsAt != nil {
		return
	}

	if req.Status.Rule == nil {
		return
	}

	accessDuration := c.getAccessDuration(req, req.Status.Rule.Authorization)
	if accessDuration <= 0 {
		return
	}

	anchor := req.Status.ApprovalEndAt
	if anchor == nil {
		anchor = pbutils.Now()
		req.Status.ApprovalEndAt = anchor
	}

	req.Status.AccessEndsAt = pbutils.Timestamp(anchor.AsTime().Add(accessDuration))
}

func (c *Controller) expireIfNeeded(req *accessv1.Request) {
	if req.Status.AccessEndsAt == nil {
		return
	}

	now := pbutils.Now()
	if req.Status.AccessEndsAt.AsTime().After(now.AsTime()) {
		return
	}

	c.setState(req, accessv1.Request_Status_State_EXPIRED)
}

func (c *Controller) setState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
	now := pbutils.Now()

	if req.Status.State != nil && req.Status.State.Status == status {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		prevState := pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State)

		req.Status.LastStates = append(
			[]*accessv1.Request_Status_State{prevState},
			req.Status.LastStates...,
		)

		if len(req.Status.LastStates) > maxRequestStateHistory {
			req.Status.LastStates = req.Status.LastStates[:maxRequestStateHistory]
		}
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

func (c *Controller) getAccessDuration(
	req *accessv1.Request,
	authz *accessv1.Policy_Spec_Rule_Authorization,
) time.Duration {
	reqDuration := umetav1.ToDuration(req.Spec.Duration).ToGo()

	var maxDuration time.Duration
	if authz != nil {
		maxDuration = umetav1.ToDuration(authz.MaxAccessDuration).ToGo()
	}

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

func (c *Controller) getCatalog(ctx context.Context, ref *metav1.ObjectReference) (*accessv1.Catalog, error) {
	if ref == nil {
		return nil, nil
	}

	catalog, err := c.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return catalog, nil
}

func (c *Controller) getGroup(ctx context.Context, ref *metav1.ObjectReference) (*corev1.Group, error) {
	if ref == nil {
		return nil, nil
	}

	grp, err := c.octeliumC.CoreC().GetGroup(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return grp, nil
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

func buildDefaultAccessRequestInlinePolicy() *corev1.InlinePolicy {
	return &corev1.InlinePolicy{
		Name: "access-request-default",
		Spec: &corev1.Policy_Spec{
			Rules: []*corev1.Policy_Spec_Rule{
				{
					Effect: corev1.Policy_Spec_Rule_ALLOW,
					Condition: &corev1.Condition{
						Type: &corev1.Condition_MatchAny{
							MatchAny: true,
						},
					},
					Priority: defaultAccessRequestAllowPriority,
				},
			},
		},
	}
}

func servicePreCondition(ref *metav1.ObjectReference) *corev1.PolicyTrigger_Status_PreCondition {
	return &corev1.PolicyTrigger_Status_PreCondition{
		Type: &corev1.PolicyTrigger_Status_PreCondition_ServiceRef{
			ServiceRef: pbutils.Clone(ref).(*metav1.ObjectReference),
		},
	}
}

func namespacePreCondition(ref *metav1.ObjectReference) *corev1.PolicyTrigger_Status_PreCondition {
	return &corev1.PolicyTrigger_Status_PreCondition{
		Type: &corev1.PolicyTrigger_Status_PreCondition_NamespaceRef{
			NamespaceRef: pbutils.Clone(ref).(*metav1.ObjectReference),
		},
	}
}

func clonePolicyNames(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	ret := make([]string, 0, len(in))
	for _, itm := range in {
		if itm == "" {
			continue
		}
		ret = append(ret, itm)
	}

	return ret
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

func getRequestSubjectUserRef(req *accessv1.Request) *metav1.ObjectReference {
	if req.Spec.Subject != nil && req.Spec.Subject.GetUserRef() != nil {
		return req.Spec.Subject.GetUserRef()
	}

	return req.Status.UserRef
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
