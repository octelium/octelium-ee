// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const allowedPathPrefix = "/index"

func exactMatch(value string) *enterprisev1.Condition_Expression_StringMatch {
	return &enterprisev1.Condition_Expression_StringMatch{
		Type: &enterprisev1.Condition_Expression_StringMatch_Exact{Exact: value},
	}
}

func prefixMatch(value string) *enterprisev1.Condition_Expression_StringMatch {
	return &enterprisev1.Condition_Expression_StringMatch{
		Type: &enterprisev1.Condition_Expression_StringMatch_Prefix{Prefix: value},
	}
}

func httpPathCondition(match *enterprisev1.Condition_Expression_StringMatch) *enterprisev1.Condition {
	return &enterprisev1.Condition{
		Type: &enterprisev1.Condition_Expression_{
			Expression: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPPath_{
					RequestHTTPPath: &enterprisev1.Condition_Expression_RequestHTTPPath{
						Match: match,
					},
				},
			},
		},
	}
}

func httpMethodCondition(match *enterprisev1.Condition_Expression_StringMatch) *enterprisev1.Condition {
	return &enterprisev1.Condition{
		Type: &enterprisev1.Condition_Expression_{
			Expression: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPMethod_{
					RequestHTTPMethod: &enterprisev1.Condition_Expression_RequestHTTPMethod{
						Match: match,
					},
				},
			},
		},
	}
}

func allCondition(of ...*enterprisev1.Condition) *enterprisev1.Condition {
	return &enterprisev1.Condition{
		Type: &enterprisev1.Condition_All_{
			All: &enterprisev1.Condition_All{Of: of},
		},
	}
}

func inlineCondition(name string, condition *corev1.Condition) []*corev1.InlinePolicy {
	return []*corev1.InlinePolicy{
		{
			Name: name,
			Spec: &corev1.Policy_Spec{
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Name:      name,
						Effect:    corev1.Policy_Spec_Rule_ALLOW,
						Condition: condition,
					},
				},
			},
		},
	}
}

func waitMethodStatus(t *testing.T, h *eeharness.H,
	c *resty.Client, method, path string, want int) {
	t.Helper()

	h.Eventually(t, fmt.Sprintf("%s %s to return %d", method, path, want),
		eeharness.PropagationBudget, func(ctx context.Context) error {
			res, err := c.R().SetContext(ctx).Execute(method, path)
			if err != nil {
				return err
			}
			if res.StatusCode() != want {
				return errUnexpectedStatus(res.StatusCode(), want)
			}
			return nil
		})
}

func testEnterpriseConditionCompiler(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	structured := allCondition(
		httpMethodCondition(exactMatch("GET")),
		httpPathCondition(prefixMatch(allowedPathPrefix)))

	compiled, err := h.EnterpriseC().GetCoreCondition(ctx, structured)
	require.Nil(t, err)
	require.NotNil(t, compiled)

	t.Run("TheCompiledConditionIsACoreCondition", func(t *testing.T) {
		assert.NotNil(t, compiled.GetAll())
		assert.Len(t, compiled.GetAll().Of, 2)
	})

	t.Run("ALeafCompilesToACELMatch", func(t *testing.T) {
		res, err := h.EnterpriseC().GetCoreCondition(ctx,
			httpPathCondition(prefixMatch(allowedPathPrefix)))
		require.Nil(t, err)
		assert.Nil(t, res.GetAll())
		assert.Contains(t, res.GetMatch(), "ctx.request.http.path")
		assert.Contains(t, res.GetMatch(), allowedPathPrefix)
	})

	svc := h.NewPublicService(t, "default")
	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: inlineCondition("compiled", compiled),
	})
	c := h.ServiceClient(svc, h.AccessToken(t, usr))

	t.Run("TheDataPlaneHonorsTheCompiledCondition", func(t *testing.T) {
		waitMethodStatus(t, h, c, http.MethodGet, "/index.html", http.StatusOK)
	})

	t.Run("AnUnmatchedPathIsDenied", func(t *testing.T) {
		waitMethodStatus(t, h, c, http.MethodGet, "/", http.StatusForbidden)
	})

	t.Run("AnUnmatchedMethodIsDenied", func(t *testing.T) {
		waitMethodStatus(t, h, c, http.MethodPost, "/index.html", http.StatusForbidden)
	})

	t.Run("ThePolicyPortalNeverGrantsWithoutARequest", func(t *testing.T) {
		res, err := h.PolicyPortalC().IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
			Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
				UserRef: umetav1.GetObjectReference(usr),
			},
			Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(svc),
			},
		})
		assert.True(t, err != nil || !res.IsAuthorized,
			"the PolicyPortal granted a request-scoped Condition without a request")
	})

	t.Run("AMatchAnyConditionCompilesToMatchAny", func(t *testing.T) {
		res, err := h.EnterpriseC().GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_MatchAny{MatchAny: true},
		})
		require.Nil(t, err)
		assert.True(t, res.GetMatchAny())
	})

	t.Run("AGroupConditionIsValidatedAgainstTheCluster", func(t *testing.T) {
		grp := h.CreateGroup(t, &corev1.Group{
			Metadata: &metav1.Metadata{Name: h.Name()},
			Spec:     &corev1.Group_Spec{},
		})

		res, err := h.EnterpriseC().GetCoreCondition(ctx, groupCondition(
			umetav1.GetObjectReference(grp)))
		require.Nil(t, err)
		assert.NotNil(t, res)

		_, err = h.EnterpriseC().GetCoreCondition(ctx, groupCondition(
			&metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(10)}))
		assert.NotNil(t, err)
	})

	t.Run("AnEmptyExpressionIsRefused", func(t *testing.T) {
		_, err := h.EnterpriseC().GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Expression_{
				Expression: &enterprisev1.Condition_Expression{},
			},
		})
		assert.NotNil(t, err)

		_, err = h.EnterpriseC().GetCoreCondition(ctx, httpPathCondition(nil))
		assert.NotNil(t, err)
	})

	t.Run("ADenyAtTheSamePriorityWins", func(t *testing.T) {
		denied, err := h.EnterpriseC().GetCoreCondition(ctx,
			httpPathCondition(prefixMatch(allowedPathPrefix)))
		require.Nil(t, err)

		blocked := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
			InlinePolicies: append(inlineCondition("compiled", compiled),
				&corev1.InlinePolicy{
					Name: "blocked",
					Spec: &corev1.Policy_Spec{
						Rules: []*corev1.Policy_Spec_Rule{
							{
								Name:      "blocked",
								Effect:    corev1.Policy_Spec_Rule_DENY,
								Condition: denied,
							},
						},
					},
				}),
		})

		waitMethodStatus(t, h, h.ServiceClient(svc, h.AccessToken(t, blocked)),
			http.MethodGet, "/index.html", http.StatusForbidden)
	})
}

func groupCondition(ref *metav1.ObjectReference) *enterprisev1.Condition {
	return &enterprisev1.Condition{
		Type: &enterprisev1.Condition_Expression_{
			Expression: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Group_{
					Group: &enterprisev1.Condition_Expression_Group{GroupRef: ref},
				},
			},
		},
	}
}
