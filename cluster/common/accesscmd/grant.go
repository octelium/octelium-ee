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
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

const PolicyTriggerNamePrefix = "access-request-"

const DefaultGrantInlinePolicyName = "access-request-default"

const nonSystemServiceMatch = `!(has(ctx.service.metadata.isSystem) && ctx.service.metadata.isSystem) && ` +
	`!(has(ctx.service.metadata.isSystemHidden) && ctx.service.metadata.isSystemHidden) && ` +
	`!(has(ctx.service.metadata.isUserHidden) && ctx.service.metadata.isUserHidden)`

func PolicyTriggerName(req *accessv1.Request) string {
	if req.Metadata.Uid != "" {
		return fmt.Sprintf("%s%s", PolicyTriggerNamePrefix, req.Metadata.Uid)
	}

	return fmt.Sprintf("%s%s", PolicyTriggerNamePrefix, req.Metadata.Name)
}

func BuildPolicyTrigger(req *accessv1.Request) (*corev1.PolicyTrigger, error) {
	scope := req.Status.ResolvedScope
	if scope == nil || scope.AccessLevel == nil {
		return nil, errors.Errorf("request %q has no resolved scope", req.Metadata.Name)
	}

	subjectRef := GetSubjectUserRef(req)
	if subjectRef == nil || subjectRef.Uid == "" {
		return nil, errors.Errorf("request %q has no subject", req.Metadata.Name)
	}

	var targets []*corev1.PolicyTrigger_Status_PreCondition

	for _, ref := range scope.ServiceRefs {
		targets = append(targets, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_ServiceRef{
				ServiceRef: cloneRef(ref),
			},
		})
	}

	for _, ref := range scope.NamespaceRefs {
		targets = append(targets, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_All_{
				All: &corev1.PolicyTrigger_Status_PreCondition_All{
					Of: []*corev1.PolicyTrigger_Status_PreCondition{
						{
							Type: &corev1.PolicyTrigger_Status_PreCondition_NamespaceRef{
								NamespaceRef: cloneRef(ref),
							},
						},
						{
							Type: &corev1.PolicyTrigger_Status_PreCondition_Condition{
								Condition: &corev1.Condition{
									Type: &corev1.Condition_Match{
										Match: nonSystemServiceMatch,
									},
								},
							},
						},
					},
				},
			},
		})
	}

	if len(targets) == 0 {
		return nil, errors.Errorf("request %q has an empty scope", req.Metadata.Name)
	}

	preConditions := []*corev1.PolicyTrigger_Status_PreCondition{
		{
			Type: &corev1.PolicyTrigger_Status_PreCondition_UserRef{
				UserRef: cloneRef(subjectRef),
			},
		},
	}

	if len(targets) == 1 {
		preConditions = append(preConditions, targets[0])
	} else {
		preConditions = append(preConditions, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_Any_{
				Any: &corev1.PolicyTrigger_Status_PreCondition_Any{
					Of: targets,
				},
			},
		})
	}

	if req.Status.AccessStartsAt != nil {
		preConditions = append(preConditions, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_NotBefore{
				NotBefore: pbutils.Timestamp(req.Status.AccessStartsAt.AsTime()),
			},
		})
	}

	if req.Status.AccessEndsAt != nil {
		preConditions = append(preConditions, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_NotAfter{
				NotAfter: pbutils.Timestamp(req.Status.AccessEndsAt.AsTime()),
			},
		})
	}

	if cond := buildVerificationPreCondition(scope.AccessLevel.Verification); cond != nil {
		preConditions = append(preConditions, cond)
	}

	policies, inlinePolicies, err := buildGrantPolicies(scope)
	if err != nil {
		return nil, err
	}

	return &corev1.PolicyTrigger{
		Metadata: &metav1.Metadata{
			Name:           PolicyTriggerName(req),
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec: &corev1.PolicyTrigger_Spec{},
		Status: &corev1.PolicyTrigger_Status{
			OwnerRef: objectRef(req),
			PreCondition: &corev1.PolicyTrigger_Status_PreCondition{
				Type: &corev1.PolicyTrigger_Status_PreCondition_All_{
					All: &corev1.PolicyTrigger_Status_PreCondition_All{
						Of: preConditions,
					},
				},
			},
			Policies:       policies,
			InlinePolicies: inlinePolicies,
		},
	}, nil
}

func buildGrantPolicies(scope *accessv1.Request_Status_ResolvedScope) ([]string, []*corev1.InlinePolicy, error) {
	level := scope.AccessLevel

	switch level.Type.(type) {
	case *accessv1.Catalog_Spec_AccessLevel_Restricted_:
		restricted := level.GetRestricted()

		var policies []string
		for _, name := range restricted.Policies {
			if name != "" {
				policies = append(policies, name)
			}
		}

		var inlinePolicies []*corev1.InlinePolicy
		for _, pol := range restricted.InlinePolicies {
			if pol != nil {
				inlinePolicies = append(inlinePolicies, pbutils.Clone(pol).(*corev1.InlinePolicy))
			}
		}

		return policies, inlinePolicies, nil

	case *accessv1.Catalog_Spec_AccessLevel_Full_:
		return nil, []*corev1.InlinePolicy{
			{
				Name: DefaultGrantInlinePolicyName,
				Spec: &corev1.Policy_Spec{
					Rules: []*corev1.Policy_Spec_Rule{
						{
							Effect: corev1.Policy_Spec_Rule_ALLOW,
							Condition: &corev1.Condition{
								Type: &corev1.Condition_MatchAny{
									MatchAny: true,
								},
							},
							Priority: scope.Priority,
						},
					},
				},
			},
		}, nil

	default:
		return nil, nil, errors.Errorf("access level %q has an unsupported type", level.Name)
	}
}

func buildVerificationPreCondition(
	v *accessv1.Catalog_Spec_AccessLevel_Verification) *corev1.PolicyTrigger_Status_PreCondition {
	if v == nil || v.MinAAL == corev1.Session_Status_Authentication_Info_AAL_UNSET {
		return nil
	}

	var levels []string
	for _, aal := range []corev1.Session_Status_Authentication_Info_AAL{
		corev1.Session_Status_Authentication_Info_AAL1,
		corev1.Session_Status_Authentication_Info_AAL2,
		corev1.Session_Status_Authentication_Info_AAL3,
	} {
		if aal >= v.MinAAL {
			levels = append(levels, fmt.Sprintf("%q", aal.String()))
		}
	}

	return &corev1.PolicyTrigger_Status_PreCondition{
		Type: &corev1.PolicyTrigger_Status_PreCondition_Condition{
			Condition: &corev1.Condition{
				Type: &corev1.Condition_Match{
					Match: fmt.Sprintf("ctx.session.status.authentication.info.aal in [%s]",
						strings.Join(levels, ", ")),
				},
			},
		},
	}
}

func DeletePolicyTriggerOf(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) error {
	_, err := octeliumC.CoreC().DeletePolicyTrigger(ctx, &rmetav1.DeleteOptions{
		Name: PolicyTriggerName(req),
	})
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}
