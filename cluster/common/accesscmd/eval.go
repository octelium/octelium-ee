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
	"slices"
	"sort"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
)

const maxConditionDepth = 16

type Evaluator struct {
	octeliumC octeliumc.ClientInterface
	celEngine *celengine.CELEngine
}

func NewEvaluator(octeliumC octeliumc.ClientInterface, celEngine *celengine.CELEngine) *Evaluator {
	return &Evaluator{
		octeliumC: octeliumC,
		celEngine: celEngine,
	}
}

type EvalInput struct {
	Request   *accessv1.Request
	Requester *corev1.User
	Subject   *corev1.User
	Scope     *Scope
}

type RuleEvaluation struct {
	Policy *accessv1.Policy
	Rule   *accessv1.Policy_Spec_Rule
	Result accessv1.SimulateRequestResponse_RuleEvaluation_Result
	Err    error
}

type EvalResult struct {
	Policy *accessv1.Policy
	Rule   *accessv1.Policy_Spec_Rule
	Rules  []*RuleEvaluation
}

func (r *EvalResult) SkippedRules() []*accessv1.Request_Status_Evaluation_SkippedRule {
	var ret []*accessv1.Request_Status_Evaluation_SkippedRule
	for _, itm := range r.Rules {
		if itm.Result != accessv1.SimulateRequestResponse_RuleEvaluation_ERROR {
			continue
		}

		ret = append(ret, &accessv1.Request_Status_Evaluation_SkippedRule{
			PolicyRef: objectRef(itm.Policy),
			RuleName:  itm.Rule.Name,
			Error:     itm.Err.Error(),
		})
	}

	return ret
}

type PolicyRule struct {
	Policy *accessv1.Policy
	Rule   *accessv1.Policy_Spec_Rule
}

func GetSortedPolicyRules(policies []*accessv1.Policy) []*PolicyRule {
	var ret []*PolicyRule

	for _, pol := range policies {
		if pol == nil || pol.Spec == nil || pol.Spec.IsDisabled {
			continue
		}

		for _, rule := range pol.Spec.Rules {
			if rule == nil {
				continue
			}

			ret = append(ret, &PolicyRule{
				Policy: pol,
				Rule:   rule,
			})
		}
	}

	sort.SliceStable(ret, func(i, j int) bool {
		if ret[i].Rule.Priority != ret[j].Rule.Priority {
			return ret[i].Rule.Priority < ret[j].Rule.Priority
		}

		return ret[i].Policy.Metadata.Name < ret[j].Policy.Metadata.Name
	})

	return ret
}

func (e *Evaluator) Evaluate(ctx context.Context, in *EvalInput, collectAll bool) (*EvalResult, error) {
	polList, err := e.octeliumC.AccessC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &EvalResult{}
	ev := &evaluation{
		e:  e,
		in: in,
	}

	for _, itm := range GetSortedPolicyRules(polList.Items) {
		res := &RuleEvaluation{
			Policy: itm.Policy,
			Rule:   itm.Rule,
		}
		ret.Rules = append(ret.Rules, res)

		if ret.Rule != nil {
			if !collectAll {
				break
			}
			continue
		}

		matched, err := ev.matchCondition(ctx, itm.Rule.Condition, 0)
		switch {
		case err != nil:
			res.Result = accessv1.SimulateRequestResponse_RuleEvaluation_ERROR
			res.Err = err
		case matched:
			res.Result = accessv1.SimulateRequestResponse_RuleEvaluation_MATCHED
			ret.Policy = itm.Policy
			ret.Rule = itm.Rule
		default:
			res.Result = accessv1.SimulateRequestResponse_RuleEvaluation_NOT_MATCHED
		}
	}

	return ret, nil
}

type evaluation struct {
	e      *Evaluator
	in     *EvalInput
	celCtx map[string]any
}

func (ev *evaluation) matchCondition(ctx context.Context,
	cond *accessv1.Policy_Spec_Rule_Condition, depth int) (bool, error) {
	if cond == nil {
		return false, nil
	}

	if depth > maxConditionDepth {
		return false, errors.Errorf("the Condition is too deeply nested")
	}

	switch cond.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_MatchAny:
		return cond.GetMatchAny(), nil

	case *accessv1.Policy_Spec_Rule_Condition_Subject_:
		return ev.matchSubject(ctx, cond.GetSubject())

	case *accessv1.Policy_Spec_Rule_Condition_Resource_:
		return ev.matchResource(cond.GetResource()), nil

	case *accessv1.Policy_Spec_Rule_Condition_All_:
		arg := cond.GetAll()
		if len(arg.Of) == 0 {
			return false, nil
		}

		for _, sub := range arg.Of {
			matched, err := ev.matchCondition(ctx, sub, depth+1)
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
			matched, err := ev.matchCondition(ctx, sub, depth+1)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}

		return false, nil

	case *accessv1.Policy_Spec_Rule_Condition_UserRef:
		return RefEqual(objectRef(ev.in.Requester), cond.GetUserRef()), nil

	case *accessv1.Policy_Spec_Rule_Condition_Match:
		return ev.matchCEL(ctx, cond.GetMatch())

	case *accessv1.Policy_Spec_Rule_Condition_AccessLevel:
		return ev.in.Scope != nil && ev.in.Scope.Level != nil &&
			ev.in.Scope.Level.Name == cond.GetAccessLevel(), nil

	case *accessv1.Policy_Spec_Rule_Condition_IsOnBehalf:
		if !cond.GetIsOnBehalf() {
			return false, nil
		}

		return ev.in.Requester != nil && ev.in.Subject != nil &&
			ev.in.Requester.Metadata.Uid != ev.in.Subject.Metadata.Uid, nil

	default:
		return false, nil
	}
}

func (ev *evaluation) matchSubject(ctx context.Context,
	subject *accessv1.Policy_Spec_Rule_Condition_Subject) (bool, error) {
	if subject == nil || ev.in.Subject == nil {
		return false, nil
	}

	switch subject.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Subject_UserRef:
		return RefEqual(objectRef(ev.in.Subject), subject.GetUserRef()), nil

	case *accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef:
		grp, err := GetGroup(ctx, ev.e.octeliumC, subject.GetGroupRef())
		if err != nil {
			return false, err
		}
		if grp == nil {
			return false, nil
		}

		if grp.Metadata.Uid != subject.GetGroupRef().Uid && subject.GetGroupRef().Uid != "" {
			return false, nil
		}

		return ev.in.Subject.Spec != nil && slices.Contains(ev.in.Subject.Spec.Groups, grp.Metadata.Name), nil

	default:
		return false, nil
	}
}

func (ev *evaluation) matchResource(resource *accessv1.Policy_Spec_Rule_Condition_Resource) bool {
	scope := ev.in.Scope
	if resource == nil || scope == nil || scope.Status == nil {
		return false
	}

	switch resource.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef:
		ref := resource.GetServiceRef()
		for _, svc := range scope.Services {
			if RefEqual(objectRef(svc), ref) {
				return true
			}
		}

		return false

	case *accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef:
		return RefEqual(scope.Status.CatalogRef, resource.GetCatalogRef())

	case *accessv1.Policy_Spec_Rule_Condition_Resource_NamespaceRef:
		ref := resource.GetNamespaceRef()
		for _, ns := range scope.Namespaces {
			if RefEqual(objectRef(ns), ref) {
				return true
			}
		}

		for _, svc := range scope.Services {
			if svc.Status != nil && RefEqual(svc.Status.NamespaceRef, ref) {
				return true
			}
		}

		return false

	default:
		return false
	}
}

func (ev *evaluation) matchCEL(ctx context.Context, exp string) (bool, error) {
	if ev.e.celEngine == nil {
		return false, errors.Errorf("no CEL engine")
	}

	if ev.celCtx == nil {
		ev.celCtx = BuildCELContext(ev.in)
	}

	return ev.e.celEngine.EvalPolicy(ctx, exp, map[string]any{
		"ctx": ev.celCtx,
	})
}

func BuildCELContext(in *EvalInput) map[string]any {
	ret := map[string]any{}

	if in.Request != nil {
		ret["request"] = pbutils.MustConvertToMap(in.Request)
	}

	if in.Requester != nil {
		requester := pbutils.MustConvertToMap(in.Requester)
		ret["requester"] = requester
		ret["user"] = requester
	}

	if in.Subject != nil {
		ret["subject"] = pbutils.MustConvertToMap(in.Subject)
	}

	if in.Scope != nil {
		if in.Scope.Catalog != nil {
			ret["catalog"] = pbutils.MustConvertToMap(in.Scope.Catalog)
		}

		if in.Scope.Level != nil {
			ret["accessLevel"] = pbutils.MustConvertToMap(in.Scope.Level)
		}

		services := make([]any, 0, len(in.Scope.Services))
		for _, svc := range in.Scope.Services {
			services = append(services, pbutils.MustConvertToMap(svc))
		}
		ret["services"] = services

		namespaces := make([]any, 0, len(in.Scope.Namespaces))
		for _, ns := range in.Scope.Namespaces {
			namespaces = append(namespaces, pbutils.MustConvertToMap(ns))
		}
		ret["namespaces"] = namespaces
	}

	return ret
}
