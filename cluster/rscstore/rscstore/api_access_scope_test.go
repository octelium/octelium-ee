// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAccessScopeFiltersAndCounters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvAccess{s: env.srv}
	now := time.Now().UTC()

	ownerGroup := "owners"
	owner := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("owner", now),
		Spec:       &corev1.User_Spec{Groups: []string{ownerGroup}},
		Status:     &corev1.User_Status{},
	}
	insertRscStoreObject(t, env, owner)

	catalogRef := &metav1.ObjectReference{Name: "prod", Uid: vutils.UUIDv4()}
	billing := &metav1.ObjectReference{Name: "billing.production", Uid: vutils.UUIDv4()}
	grafana := &metav1.ObjectReference{Name: "grafana.monitoring", Uid: vutils.UUIDv4()}
	dynamicNs := &metav1.ObjectReference{Name: "dynamic", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("owned", now),
		Spec: &accessv1.Catalog_Spec{
			Owners: &accessv1.Catalog_Spec_Owners{Groups: []string{ownerGroup}},
		},
		Status: &accessv1.Catalog_Status{
			Issues: []*accessv1.ConfigurationIssue{{Type: accessv1.ConfigurationIssue_OVERRIDES_GUARDRAILS}},
		},
	})
	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("disabled", now),
		Spec:       &accessv1.Catalog_Spec{IsDisabled: true},
		Status: &accessv1.Catalog_Status{
			Issues: []*accessv1.ConfigurationIssue{{Type: accessv1.ConfigurationIssue_EMPTY_SCOPE}},
		},
	})
	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("healthy", now),
		Spec: &accessv1.Catalog_Spec{
			Owners: &accessv1.Catalog_Spec_Owners{Users: []string{"owner"}},
		},
		Status: &accessv1.Catalog_Status{},
	})
	insertRscStoreObject(t, env, &accessv1.Policy{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindPolicy,
		Metadata:   newRscStoreMetadata("broken-policy", now),
		Spec:       &accessv1.Policy_Spec{},
		Status: &accessv1.Policy_Status{
			Issues: []*accessv1.ConfigurationIssue{{Type: accessv1.ConfigurationIssue_MISSING_REFERENCE}},
		},
	})
	insertRscStoreObject(t, env, &accessv1.Policy{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindPolicy,
		Metadata:   newRscStoreMetadata("healthy-policy", now),
		Spec:       &accessv1.Policy_Spec{},
		Status:     &accessv1.Policy_Status{},
	})

	newRequest := func(name string, createdAt time.Time, fn func(req *accessv1.Request)) {
		req := &accessv1.Request{
			ApiVersion: uaccessv1.APIVersion,
			Kind:       uaccessv1.KindRequest,
			Metadata:   newRscStoreMetadata(name, createdAt),
			Spec: &accessv1.Request_Spec{
				Resource: &accessv1.Request_Spec_Resource{
					Type: &accessv1.Request_Spec_Resource_Catalog_{
						Catalog: &accessv1.Request_Spec_Resource_Catalog{
							CatalogRef:  catalogRef,
							AccessLevel: "read",
						},
					},
				},
			},
			Status: &accessv1.Request_Status{
				State: &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_PENDING},
				ResolvedScope: &accessv1.Request_Status_ResolvedScope{
					CatalogRef:  catalogRef,
					ServiceRefs: []*metav1.ObjectReference{billing},
				},
			},
		}
		fn(req)
		insertRscStoreObject(t, env, req)
	}

	newRequest("active", now.Add(-3*time.Hour), func(req *accessv1.Request) {
		req.Spec.Resource.GetCatalog().AccessLevel = "full"
		req.Status.State = &accessv1.Request_Status_State{
			Status: accessv1.Request_Status_State_APPROVED,
			Reason: accessv1.Request_Status_State_AUTO_APPROVED,
		}
		req.Status.ResolvedScope.OverridesGuardrails = true
		req.Status.ResolvedScope.ServiceRefs = []*metav1.ObjectReference{billing, grafana}
		req.Status.AccessEndsAt = timestamppb.New(now.Add(90 * time.Minute))
		req.Status.ApprovalEndAt = timestamppb.New(now.Add(-2 * time.Hour))
		req.Status.Grant = &accessv1.Request_Status_Grant{
			State:       accessv1.Request_Status_Grant_ACTIVE,
			ActivatedAt: timestamppb.New(now.Add(-2 * time.Hour)),
		}
	})
	newRequest("scheduled", now.Add(-time.Hour), func(req *accessv1.Request) {
		req.Status.State = &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_APPROVED}
		req.Status.AccessStartsAt = timestamppb.New(now.Add(time.Hour))
		req.Status.AccessEndsAt = timestamppb.New(now.Add(3 * time.Hour))
		req.Status.Grant = &accessv1.Request_Status_Grant{State: accessv1.Request_Status_Grant_SCHEDULED}
	})
	newRequest("unroutable", now.Add(-30*time.Hour), func(req *accessv1.Request) {
		req.Status.ApprovalStartAt = timestamppb.New(now.Add(-30 * time.Hour))
		req.Status.ResolvedScope.ServiceRefs = nil
		req.Status.ResolvedScope.NamespaceRefs = []*metav1.ObjectReference{dynamicNs}
		req.Status.Review = &accessv1.Request_Status_Review{
			Steps: []*accessv1.Request_Status_Review_StepState{
				{
					Name:      "lead",
					Outcome:   accessv1.Request_Status_Review_StepState_TIMED_OUT,
					StartedAt: timestamppb.New(now.Add(-30 * time.Hour)),
					EndedAt:   timestamppb.New(now.Add(-29 * time.Hour)),
				},
				{Name: "security", Index: 1, IsUnroutable: true, StartedAt: timestamppb.New(now.Add(-29 * time.Hour))},
			},
		}
	})
	newRequest("awaiting", now.Add(-2*time.Hour), func(req *accessv1.Request) {
		req.Status.ApprovalStartAt = timestamppb.New(now.Add(-2 * time.Hour))
		req.Status.Review = &accessv1.Request_Status_Review{
			Steps: []*accessv1.Request_Status_Review_StepState{
				{
					Name:        "lead",
					Outcome:     accessv1.Request_Status_Review_StepState_SATISFIED,
					IsEscalated: true,
					StartedAt:   timestamppb.New(now.Add(-2 * time.Hour)),
					EndedAt:     timestamppb.New(now.Add(-1 * time.Hour)),
				},
			},
			InformationRequests: []*accessv1.Request_Status_Review_InformationRequest{
				{Message: "why?", RequestedAt: timestamppb.New(now.Add(-time.Hour))},
			},
		}
	})
	newRequest("failed", now.Add(-4*time.Hour), func(req *accessv1.Request) {
		req.Status.State = &accessv1.Request_Status_State{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_PROVISIONING_FAILED,
		}
		req.Status.LastStates = []*accessv1.Request_Status_State{{Status: accessv1.Request_Status_State_APPROVED}}
		req.Status.Grant = &accessv1.Request_Status_Grant{
			State:   accessv1.Request_Status_Grant_FAILED,
			EndedAt: timestamppb.New(now.Add(-3 * time.Hour)),
		}
	})
	newRequest("ended", now.Add(-5*time.Hour), func(req *accessv1.Request) {
		req.Status.State = &accessv1.Request_Status_State{
			Status: accessv1.Request_Status_State_EXPIRED,
			Reason: accessv1.Request_Status_State_ENDED_EARLY,
		}
		req.Status.ResolvedScope.ServiceRefs = []*metav1.ObjectReference{grafana}
		req.Status.LastStates = []*accessv1.Request_Status_State{{Status: accessv1.Request_Status_State_APPROVED}}
		req.Status.Grant = &accessv1.Request_Status_Grant{
			State:   accessv1.Request_Status_Grant_ENDED,
			EndedAt: timestamppb.New(now.Add(-4 * time.Hour)),
		}
	})

	list := func(opts *vaccessv1.ListRequestOptions) []string {
		res, err := srv.ListRequest(env.ctx, opts)
		require.NoError(t, err)
		var ret []string
		for _, itm := range res.Items {
			ret = append(ret, itm.Metadata.Name)
		}
		return ret
	}

	assert.ElementsMatch(t, []string{"active", "scheduled", "awaiting", "failed"},
		list(&vaccessv1.ListRequestOptions{ServiceRef: &metav1.ObjectReference{Uid: billing.Uid}}))
	assert.ElementsMatch(t, []string{"active", "ended"},
		list(&vaccessv1.ListRequestOptions{ServiceRef: &metav1.ObjectReference{Name: grafana.Name}}))
	assert.ElementsMatch(t, []string{"active", "ended"},
		list(&vaccessv1.ListRequestOptions{NamespaceRef: &metav1.ObjectReference{Name: "monitoring"}}))
	assert.ElementsMatch(t, []string{"unroutable"},
		list(&vaccessv1.ListRequestOptions{NamespaceRef: &metav1.ObjectReference{Uid: dynamicNs.Uid}}))
	assert.ElementsMatch(t, []string{"failed"},
		list(&vaccessv1.ListRequestOptions{Reason: accessv1.Request_Status_State_PROVISIONING_FAILED}))
	assert.ElementsMatch(t, []string{"scheduled"},
		list(&vaccessv1.ListRequestOptions{GrantState: accessv1.Request_Status_Grant_SCHEDULED}))
	assert.ElementsMatch(t, []string{"active"},
		list(&vaccessv1.ListRequestOptions{AccessLevel: "full"}))
	assert.ElementsMatch(t, []string{"active", "scheduled", "failed", "ended"},
		list(&vaccessv1.ListRequestOptions{WasGranted: true}))
	assert.ElementsMatch(t, []string{"unroutable"},
		list(&vaccessv1.ListRequestOptions{IsUnroutable: true}))
	assert.ElementsMatch(t, []string{"awaiting"},
		list(&vaccessv1.ListRequestOptions{IsAwaitingRequester: true}))
	assert.ElementsMatch(t, []string{"active"},
		list(&vaccessv1.ListRequestOptions{OverridesGuardrails: true}))
	assert.ElementsMatch(t, []string{"unroutable"},
		list(&vaccessv1.ListRequestOptions{IsPendingStale: true}))
	assert.ElementsMatch(t, []string{"unroutable", "awaiting"},
		list(&vaccessv1.ListRequestOptions{
			IsPendingStale: true,
			StaleAfter:     &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 30}},
		}))
	assert.Empty(t, list(&vaccessv1.ListRequestOptions{IsExpiringSoon: true}))
	assert.ElementsMatch(t, []string{"active"},
		list(&vaccessv1.ListRequestOptions{
			IsExpiringSoon: true,
			ExpiringWithin: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
		}))
	assert.Equal(t, []string{"scheduled"},
		list(&vaccessv1.ListRequestOptions{
			Common: &vmetav1.CommonListOptions{},
			OrderBy: &vaccessv1.ListRequestOptions_OrderBy{
				Type: vaccessv1.ListRequestOptions_OrderBy_ACCESS_STARTS_AT,
				Mode: vmetav1.CommonListOptions_OrderBy_DESC,
			},
			State: accessv1.Request_Status_State_APPROVED,
		})[:1])

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{})
		require.NoError(t, err)
		assert.EqualValues(t, 6, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalService)
		assert.EqualValues(t, 1, resp.TotalProvisioningFailed)
		assert.EqualValues(t, 1, resp.TotalUnroutable)
		assert.EqualValues(t, 1, resp.TotalAwaitingRequester)
		assert.EqualValues(t, 1, resp.TotalOverridesGuardrails)
		assert.EqualValues(t, 1, resp.TotalScheduled)
		assert.EqualValues(t, 1, resp.TotalEndedEarly)
		assert.EqualValues(t, 1, resp.TotalPendingStale)
		assert.EqualValues(t, 0, resp.TotalExpiringSoon)

		stepNames := map[string]*vaccessv1.StepDecisionTime{}
		for _, itm := range resp.StepDecisionTimes {
			stepNames[itm.StepName] = itm
		}
		require.Contains(t, stepNames, "lead")
		assert.EqualValues(t, 1, stepNames["lead"].TotalTimedOut)
		assert.EqualValues(t, 1, stepNames["lead"].TotalEscalated)
		require.NotNil(t, stepNames["lead"].DecisionTime)
		assert.EqualValues(t, 1, stepNames["lead"].DecisionTime.Count)
		assert.InDelta(t, 3600, stepNames["lead"].DecisionTime.AverageSeconds, 1)
		require.Contains(t, stepNames, "security")
		assert.Nil(t, stepNames["security"].DecisionTime)
	}

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			ServiceRef:     &metav1.ObjectReference{Uid: grafana.Uid},
			StaleAfter:     &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 30}},
			ExpiringWithin: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
		})
		require.NoError(t, err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalExpiringSoon)
	}

	{
		res, err := srv.GetRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:     timestamppb.New(now.Add(-6 * time.Hour)),
			To:       timestamppb.New(now),
			Interval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
			Basis:    vaccessv1.GetRequestDataPointRequest_ENDED_AT,
			GroupBy:  vaccessv1.GetRequestDataPointRequest_REASON,
		})
		require.NoError(t, err)

		totals := map[string]int64{}
		for _, series := range res.Series {
			totals[series.Key] = series.Total
		}
		assert.Equal(t, map[string]int64{
			accessv1.Request_Status_State_PROVISIONING_FAILED.String(): 1,
			accessv1.Request_Status_State_ENDED_EARLY.String():         1,
		}, totals)

		res, err = srv.GetRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:    timestamppb.New(now.Add(-6 * time.Hour)),
			To:      timestamppb.New(now),
			Basis:   vaccessv1.GetRequestDataPointRequest_ACTIVATED_AT,
			GroupBy: vaccessv1.GetRequestDataPointRequest_ACCESS_LEVEL,
		})
		require.NoError(t, err)
		require.Len(t, res.Series, 1)
		assert.Equal(t, "full", res.Series[0].Key)
		assert.EqualValues(t, 1, res.Series[0].Total)
	}

	{
		res, err := srv.ListRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_SERVICE,
		})
		require.NoError(t, err)
		require.Len(t, res.Items, 2)
		assert.Equal(t, billing.Uid, res.Items[0].Ref.Uid)
		assert.Equal(t, billing.Name, res.Items[0].Ref.Name)
		assert.EqualValues(t, 4, res.Items[0].Count)
		assert.EqualValues(t, 2, res.Items[1].Count)
		assert.EqualValues(t, 2, res.TotalCount)

		res, err = srv.ListRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_RESOURCE,
		})
		require.NoError(t, err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, catalogRef.Uid, res.Items[0].Ref.Uid)
		assert.Equal(t, "Catalog", res.Items[0].Ref.Kind)
	}

	listCatalogs := func(opts *vaccessv1.ListCatalogOptions) []string {
		res, err := srv.ListCatalog(env.ctx, opts)
		require.NoError(t, err)
		var ret []string
		for _, itm := range res.Items {
			ret = append(ret, itm.Metadata.Name)
		}
		return ret
	}

	assert.ElementsMatch(t, []string{"disabled"}, listCatalogs(&vaccessv1.ListCatalogOptions{IsDisabled: true}))
	assert.ElementsMatch(t, []string{"owned", "disabled"}, listCatalogs(&vaccessv1.ListCatalogOptions{HasIssues: true}))
	assert.ElementsMatch(t, []string{"owned"}, listCatalogs(&vaccessv1.ListCatalogOptions{OverridesGuardrails: true}))
	assert.ElementsMatch(t, []string{"owned", "healthy"}, listCatalogs(&vaccessv1.ListCatalogOptions{
		OwnerUserRef: &metav1.ObjectReference{Name: "owner"},
	}))
	assert.ElementsMatch(t, []string{"owned", "healthy"}, listCatalogs(&vaccessv1.ListCatalogOptions{
		OwnerUserRef: &metav1.ObjectReference{Uid: owner.Metadata.Uid},
	}))

	policies, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{HasIssues: true})
	require.NoError(t, err)
	require.Len(t, policies.Items, 1)
	assert.Equal(t, "broken-policy", policies.Items[0].Metadata.Name)
}
