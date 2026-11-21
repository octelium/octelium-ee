// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessconfig

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
	celEngine *celengine.CELEngine
}

func NewController(octeliumC octeliumc.ClientInterface, celEngine *celengine.CELEngine) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		celEngine: celEngine,
	}
}

func (c *Controller) OnCatalogAdd(ctx context.Context, itm *accessv1.Catalog) error {
	return c.reconcileCatalog(ctx, itm)
}

func (c *Controller) OnCatalogUpdate(ctx context.Context, new, old *accessv1.Catalog) error {
	return c.reconcileCatalog(ctx, new)
}

func (c *Controller) OnCatalogDelete(ctx context.Context, itm *accessv1.Catalog) error {
	return c.reconcileAllPolicies(ctx)
}

func (c *Controller) OnPolicyAdd(ctx context.Context, itm *accessv1.Policy) error {
	return c.reconcileAllPolicies(ctx)
}

func (c *Controller) OnPolicyUpdate(ctx context.Context, new, old *accessv1.Policy) error {
	if pbutils.IsEqual(new.Spec, old.Spec) {
		return nil
	}

	return c.reconcileAllPolicies(ctx)
}

func (c *Controller) OnPolicyDelete(ctx context.Context, itm *accessv1.Policy) error {
	return c.reconcileAllPolicies(ctx)
}

func (c *Controller) ResyncAll(ctx context.Context) error {
	catalogList, err := c.octeliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, itm := range catalogList.Items {
		if err := c.reconcileCatalog(ctx, itm); err != nil {
			zap.L().Warn("Could not reconcile the Catalog status",
				zap.String("catalog", itm.Metadata.Name), zap.Error(err))
		}
	}

	return c.reconcileAllPolicies(ctx)
}

func (c *Controller) reconcileCatalog(ctx context.Context, itm *accessv1.Catalog) error {
	status, err := c.GetCatalogStatus(ctx, itm)
	if err != nil {
		return err
	}

	if isStatusEqual(itm.Status, status) {
		return nil
	}

	next := pbutils.Clone(itm).(*accessv1.Catalog)
	next.Status = status

	_, err = c.octeliumC.AccessC().UpdateCatalog(ctx, next)
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func (c *Controller) reconcileAllPolicies(ctx context.Context) error {
	polList, err := c.octeliumC.AccessC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, itm := range polList.Items {
		status, err := c.GetPolicyStatus(ctx, itm, polList.Items)
		if err != nil {
			zap.L().Warn("Could not compute the access Policy status",
				zap.String("policy", itm.Metadata.Name), zap.Error(err))
			continue
		}

		if isPolicyStatusEqual(itm.Status, status) {
			continue
		}

		next := pbutils.Clone(itm).(*accessv1.Policy)
		next.Status = status

		if _, err := c.octeliumC.AccessC().UpdatePolicy(ctx, next); err != nil && !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not update the access Policy status",
				zap.String("policy", itm.Metadata.Name), zap.Error(err))
		}
	}

	return nil
}

func isStatusEqual(a, b *accessv1.Catalog_Status) bool {
	if a == nil {
		a = &accessv1.Catalog_Status{}
	}

	ac := pbutils.Clone(a).(*accessv1.Catalog_Status)
	bc := pbutils.Clone(b).(*accessv1.Catalog_Status)
	ac.CheckedAt = nil
	bc.CheckedAt = nil

	return pbutils.IsEqual(ac, bc)
}

func isPolicyStatusEqual(a, b *accessv1.Policy_Status) bool {
	if a == nil {
		a = &accessv1.Policy_Status{}
	}

	ac := pbutils.Clone(a).(*accessv1.Policy_Status)
	bc := pbutils.Clone(b).(*accessv1.Policy_Status)
	ac.CheckedAt = nil
	bc.CheckedAt = nil

	return pbutils.IsEqual(ac, bc)
}

func newIssue(typ accessv1.ConfigurationIssue_Type,
	severity accessv1.ConfigurationIssue_Severity, format string, args ...any) *accessv1.ConfigurationIssue {
	return &accessv1.ConfigurationIssue{
		Type:     typ,
		Severity: severity,
		Message:  fmt.Sprintf(format, args...),
	}
}

func (c *Controller) GetCatalogStatus(ctx context.Context, catalog *accessv1.Catalog) (*accessv1.Catalog_Status, error) {
	ret := &accessv1.Catalog_Status{
		CheckedAt: pbutils.Now(),
	}

	addIssue := func(issue *accessv1.ConfigurationIssue) {
		ret.Issues = append(ret.Issues, issue)
	}

	if catalog.Spec == nil {
		return ret, nil
	}

	if coll := catalog.Spec.GetResourceCollection().GetService(); coll != nil {
		for _, name := range coll.Services {
			exists, err := c.exists(ctx, func() error {
				_, err := c.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
					Name: vutils.GetServiceFullNameFromName(name),
				})
				return err
			})
			if err != nil {
				return nil, err
			}
			if !exists {
				addIssue(newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
					accessv1.ConfigurationIssue_WARNING, "The Service %s does not exist", name))
			}
		}

		for _, name := range coll.Namespaces {
			exists, err := c.exists(ctx, func() error {
				_, err := c.octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
					Name: name,
				})
				return err
			})
			if err != nil {
				return nil, err
			}
			if !exists {
				addIssue(newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
					accessv1.ConfigurationIssue_WARNING, "The Namespace %s does not exist", name))
			}
		}
	}

	if err := c.checkUsersGroups(ctx, catalog.Spec.GetAudience().GetUsers(),
		catalog.Spec.GetAudience().GetGroups(), "audience", addIssue); err != nil {
		return nil, err
	}

	if err := c.checkUsersGroups(ctx, catalog.Spec.GetAudience().GetOnBehalf().GetUsers(),
		catalog.Spec.GetAudience().GetOnBehalf().GetGroups(), "on-behalf audience", addIssue); err != nil {
		return nil, err
	}

	if err := c.checkUsersGroups(ctx, catalog.Spec.GetOwners().GetUsers(),
		catalog.Spec.GetOwners().GetGroups(), "owners", addIssue); err != nil {
		return nil, err
	}

	if match := catalog.Spec.GetAudience().GetMatch(); match != "" && c.celEngine != nil {
		if err := c.celEngine.AddPolicy(ctx, match); err != nil {
			addIssue(newIssue(accessv1.ConfigurationIssue_INVALID_EXPRESSION,
				accessv1.ConfigurationIssue_ERROR, "The audience expression is invalid: %s", err.Error()))
		}
	}

	if len(catalog.Spec.AccessLevels) == 0 {
		addIssue(newIssue(accessv1.ConfigurationIssue_NO_ACCESS_LEVEL,
			accessv1.ConfigurationIssue_ERROR, "The Catalog has no access level and cannot be requested"))
		return ret, nil
	}

	cc, err := accesscmd.GetClusterConfig(ctx, c.octeliumC)
	if err != nil {
		return nil, err
	}

	for _, level := range catalog.Spec.AccessLevels {
		for _, name := range level.GetRestricted().GetPolicies() {
			exists, err := c.exists(ctx, func() error {
				_, err := c.octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
					Name: name,
				})
				return err
			})
			if err != nil {
				return nil, err
			}
			if !exists {
				issue := newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
					accessv1.ConfigurationIssue_ERROR, "The core Policy %s does not exist", name)
				issue.AccessLevel = level.Name
				addIssue(issue)
			}
		}
	}

	scope, problems, err := accesscmd.ResolveScope(ctx, c.octeliumC, &accesscmd.ResolveScopeOpts{
		Catalog:         catalog,
		DefaultPriority: accesscmd.GetDefaultAccessPriority(cc),
	})
	if err != nil {
		return nil, err
	}

	if len(problems) > 0 {
		addIssue(newIssue(accessv1.ConfigurationIssue_EMPTY_SCOPE,
			accessv1.ConfigurationIssue_ERROR, "%s", problems[0].Message))
	}

	if scope != nil {
		ret.TotalService = uint32(len(scope.Services))

		if scope.Status.TotalExcludedService > 0 {
			addIssue(newIssue(accessv1.ConfigurationIssue_SYSTEM_RESOURCE_EXCLUDED,
				accessv1.ConfigurationIssue_WARNING,
				"%d system or hidden Services are excluded from the Catalog", scope.Status.TotalExcludedService))
		}
	}

	for _, level := range catalog.Spec.AccessLevels {
		levelScope, _, err := accesscmd.ResolveScope(ctx, c.octeliumC, &accesscmd.ResolveScopeOpts{
			Catalog:         catalog,
			AccessLevel:     level.Name,
			DefaultPriority: accesscmd.GetDefaultAccessPriority(cc),
		})
		if err != nil {
			return nil, err
		}

		if levelScope != nil && levelScope.Status.OverridesGuardrails {
			issue := newIssue(accessv1.ConfigurationIssue_OVERRIDES_GUARDRAILS,
				accessv1.ConfigurationIssue_WARNING,
				"The access level %s takes precedence over the core Policy rules of the default priority, including their DENY rules",
				level.Name)
			issue.AccessLevel = level.Name
			addIssue(issue)
		}
	}

	return ret, nil
}

func (c *Controller) checkUsersGroups(ctx context.Context, users, groups []string, what string,
	addIssue func(*accessv1.ConfigurationIssue)) error {
	for _, name := range users {
		exists, err := c.exists(ctx, func() error {
			_, err := c.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			return err
		})
		if err != nil {
			return err
		}
		if !exists {
			addIssue(newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
				accessv1.ConfigurationIssue_WARNING, "The User %s of the %s does not exist", name, what))
		}
	}

	for _, name := range groups {
		exists, err := c.exists(ctx, func() error {
			_, err := c.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			return err
		})
		if err != nil {
			return err
		}
		if !exists {
			addIssue(newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
				accessv1.ConfigurationIssue_WARNING, "The Group %s of the %s does not exist", name, what))
		}
	}

	return nil
}

func (c *Controller) exists(ctx context.Context, fn func() error) (bool, error) {
	if err := fn(); err != nil {
		if grpcerr.IsNotFound(err) || grpcerr.IsInvalidArg(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (c *Controller) GetPolicyStatus(ctx context.Context,
	pol *accessv1.Policy, all []*accessv1.Policy) (*accessv1.Policy_Status, error) {
	ret := &accessv1.Policy_Status{
		CheckedAt: pbutils.Now(),
	}

	if pol.Spec == nil {
		return ret, nil
	}

	sorted := accesscmd.GetSortedPolicyRules(all)

	for _, rule := range pol.Spec.Rules {
		if rule == nil {
			continue
		}

		addIssue := func(issue *accessv1.ConfigurationIssue) {
			issue.RuleName = rule.Name
			ret.Issues = append(ret.Issues, issue)
		}

		if err := c.checkCondition(ctx, rule.Condition, 0, addIssue); err != nil {
			return nil, err
		}

		if !pol.Spec.IsDisabled {
			if shadow := getShadowingRule(sorted, pol, rule); shadow != nil {
				addIssue(newIssue(accessv1.ConfigurationIssue_SHADOWED_RULE,
					accessv1.ConfigurationIssue_WARNING,
					"The Rule is never reached because the Rule %s of the Policy %s matches every Request before it",
					shadow.Rule.Name, shadow.Policy.Metadata.Name))
			}
		}

		for idx, step := range rule.GetAction().GetReview().GetSteps() {
			if step == nil {
				continue
			}

			stepIssue := func(issue *accessv1.ConfigurationIssue) {
				issue.StepName = accesscmd.StepName(step, int32(idx))
				addIssue(issue)
			}

			if err := c.checkStep(ctx, step, stepIssue); err != nil {
				return nil, err
			}
		}
	}

	return ret, nil
}

func getShadowingRule(sorted []*accesscmd.PolicyRule,
	pol *accessv1.Policy, rule *accessv1.Policy_Spec_Rule) *accesscmd.PolicyRule {
	for _, itm := range sorted {
		if itm.Policy.Metadata.Uid == pol.Metadata.Uid && itm.Rule.Name == rule.Name {
			return nil
		}

		if itm.Rule.Condition != nil && itm.Rule.Condition.GetMatchAny() {
			return itm
		}
	}

	return nil
}

func (c *Controller) checkCondition(ctx context.Context, cond *accessv1.Policy_Spec_Rule_Condition,
	depth int, addIssue func(*accessv1.ConfigurationIssue)) error {
	if cond == nil || depth > 16 {
		return nil
	}

	missing := func(kind string, ref *metav1.ObjectReference, fn func() error) error {
		exists, err := c.exists(ctx, fn)
		if err != nil {
			return err
		}
		if !exists {
			issue := newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
				accessv1.ConfigurationIssue_ERROR, "The %s %s does not exist", kind, ref.GetName())
			issue.Ref = ref
			addIssue(issue)
		}
		return nil
	}

	switch cond.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Subject_:
		subject := cond.GetSubject()
		if ref := subject.GetUserRef(); ref != nil {
			return missing("User", ref, func() error {
				_, err := c.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
				return err
			})
		}
		if ref := subject.GetGroupRef(); ref != nil {
			return missing("Group", ref, func() error {
				_, err := c.octeliumC.CoreC().GetGroup(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
				return err
			})
		}

	case *accessv1.Policy_Spec_Rule_Condition_Resource_:
		resource := cond.GetResource()
		if ref := resource.GetServiceRef(); ref != nil {
			return missing("Service", ref, func() error {
				_, err := c.octeliumC.CoreC().GetService(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
				return err
			})
		}
		if ref := resource.GetCatalogRef(); ref != nil {
			return missing("Catalog", ref, func() error {
				_, err := c.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
				return err
			})
		}
		if ref := resource.GetNamespaceRef(); ref != nil {
			return missing("Namespace", ref, func() error {
				_, err := c.octeliumC.CoreC().GetNamespace(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
				return err
			})
		}

	case *accessv1.Policy_Spec_Rule_Condition_UserRef:
		ref := cond.GetUserRef()
		return missing("User", ref, func() error {
			_, err := c.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
			return err
		})

	case *accessv1.Policy_Spec_Rule_Condition_Match:
		if c.celEngine != nil {
			if err := c.celEngine.AddPolicy(ctx, cond.GetMatch()); err != nil {
				addIssue(newIssue(accessv1.ConfigurationIssue_INVALID_EXPRESSION,
					accessv1.ConfigurationIssue_ERROR, "The CEL expression is invalid: %s", err.Error()))
			}
		}

	case *accessv1.Policy_Spec_Rule_Condition_All_:
		for _, sub := range cond.GetAll().GetOf() {
			if err := c.checkCondition(ctx, sub, depth+1, addIssue); err != nil {
				return err
			}
		}

	case *accessv1.Policy_Spec_Rule_Condition_Any_:
		for _, sub := range cond.GetAny().GetOf() {
			if err := c.checkCondition(ctx, sub, depth+1, addIssue); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) checkStep(ctx context.Context, step *accessv1.Policy_Spec_Rule_Action_Review_Step,
	addIssue func(*accessv1.ConfigurationIssue)) error {
	hasCatalogOwners := false
	eligible := map[string]struct{}{}

	for _, reviewer := range append(append([]*accesscmd.Reviewer{}, step.Reviewers...),
		step.GetEscalation().GetReviewers()...) {
		switch reviewer.Type.(type) {
		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
			usr, err := accesscmd.GetUser(ctx, c.octeliumC, reviewer.GetUser().GetUserRef())
			if err != nil {
				return err
			}
			if usr == nil {
				issue := newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
					accessv1.ConfigurationIssue_ERROR, "The reviewer User %s does not exist",
					reviewer.GetUser().GetUserRef().GetName())
				issue.Ref = reviewer.GetUser().GetUserRef()
				addIssue(issue)
				continue
			}
			if accesscmd.IsUserEligible(usr) {
				eligible[usr.Metadata.Uid] = struct{}{}
			}

		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
			grp, err := accesscmd.GetGroup(ctx, c.octeliumC, reviewer.GetGroup().GetGroupRef())
			if err != nil {
				return err
			}
			if grp == nil {
				issue := newIssue(accessv1.ConfigurationIssue_MISSING_REFERENCE,
					accessv1.ConfigurationIssue_ERROR, "The reviewer Group %s does not exist",
					reviewer.GetGroup().GetGroupRef().GetName())
				issue.Ref = reviewer.GetGroup().GetGroupRef()
				addIssue(issue)
				continue
			}

			usrList, err := c.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
				Filters: []*rmetav1.ListOptions_Filter{
					urscsrv.FilterFieldIncludesValStr("spec.groups", grp.Metadata.Name),
				},
			})
			if err != nil {
				return err
			}

			for _, usr := range usrList.Items {
				if accesscmd.IsUserEligible(usr) {
					eligible[usr.Metadata.Uid] = struct{}{}
				}
			}

		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_:
			hasCatalogOwners = true
		}
	}

	if hasCatalogOwners {
		return nil
	}

	if len(eligible) == 0 {
		addIssue(newIssue(accessv1.ConfigurationIssue_NO_ELIGIBLE_REVIEWER,
			accessv1.ConfigurationIssue_ERROR, "The review Step has no User that can review it"))
		return nil
	}

	if step.ApprovalRequirement == accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT &&
		step.ApprovalCount > uint32(len(eligible)) {
		addIssue(newIssue(accessv1.ConfigurationIssue_IMPOSSIBLE_APPROVAL_COUNT,
			accessv1.ConfigurationIssue_ERROR,
			"The review Step requires %d approvals but only %d Users can review it",
			step.ApprovalCount, len(eligible)))
	}

	return nil
}
