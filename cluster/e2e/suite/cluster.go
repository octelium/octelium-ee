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
	"slices"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const componentLogLookback = 30 * time.Minute

var rscStoreSummaryKinds = []visibilityv1.GetClusterSummaryRequest_Kind{
	visibilityv1.GetClusterSummaryRequest_CORE_USER,
	visibilityv1.GetClusterSummaryRequest_CORE_SERVICE,
	visibilityv1.GetClusterSummaryRequest_ACCESS_POLICY,
	visibilityv1.GetClusterSummaryRequest_ENTERPRISE_SECRET,
}

var rscStoreSubsystems = []visibilityv1.GetClusterHealthResponse_Subsystem_Type{
	visibilityv1.GetClusterHealthResponse_Subsystem_CERTIFICATES,
	visibilityv1.GetClusterHealthResponse_Subsystem_DIRECTORY_PROVIDERS,
	visibilityv1.GetClusterHealthResponse_Subsystem_SECRET_STORES,
	visibilityv1.GetClusterHealthResponse_Subsystem_DEVICE_MANAGERS,
	visibilityv1.GetClusterHealthResponse_Subsystem_ENROLLMENT,
	visibilityv1.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS,
}

var logStoreSubsystems = []visibilityv1.GetClusterHealthResponse_Subsystem_Type{
	visibilityv1.GetClusterHealthResponse_Subsystem_COMPONENTS,
	visibilityv1.GetClusterHealthResponse_Subsystem_AUTHORIZATION,
}

func subsystemOf(res *visibilityv1.GetClusterHealthResponse,
	sType visibilityv1.GetClusterHealthResponse_Subsystem_Type,
) *visibilityv1.GetClusterHealthResponse_Subsystem {
	idx := slices.IndexFunc(res.Subsystems,
		func(itm *visibilityv1.GetClusterHealthResponse_Subsystem) bool {
			return itm.Type == sType
		})
	if idx < 0 {
		return nil
	}

	return res.Subsystems[idx]
}

func unavailableKinds(res *visibilityv1.GetClusterSummaryResponse,
) []visibilityv1.GetClusterSummaryRequest_Kind {
	var ret []visibilityv1.GetClusterSummaryRequest_Kind

	for _, itm := range res.Unavailables {
		ret = append(ret, itm.Kind)
	}

	return ret
}

func healthRank(status visibilityv1.GetClusterHealthResponse_Status) int {
	switch status {
	case visibilityv1.GetClusterHealthResponse_CRITICAL:
		return 3
	case visibilityv1.GetClusterHealthResponse_DEGRADED:
		return 2
	case visibilityv1.GetClusterHealthResponse_UNKNOWN:
		return 1
	default:
		return 0
	}
}

func testClusterSummaryFanOut(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	var before *visibilityv1.GetClusterSummaryResponse

	h.Eventually(t, "the Cluster summary to answer for every kind",
		eeharness.PropagationBudget, func(ctx context.Context) error {
			res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
				&visibilityv1.GetClusterSummaryRequest{})
			if err != nil {
				return err
			}
			if len(res.Unavailables) > 0 {
				return errors.Errorf("the kinds %v are unavailable", unavailableKinds(res))
			}

			before = res
			return nil
		})

	require.NotNil(t, before.Core)
	require.NotNil(t, before.Access)
	require.NotNil(t, before.Enterprise)

	usr := h.CreateWorkloadUser(t, nil)
	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomStringCanonical(24))

	t.Run("TheFanOutReachesEveryStore", func(t *testing.T) {
		assert.True(t, before.Core.User.TotalNumber > 0)
		assert.True(t, before.Core.Service.TotalNumber > 0)
		assert.True(t, before.Core.Region.TotalNumber > 0)
		assert.NotNil(t, before.Access.Policy)
		assert.NotNil(t, before.Enterprise.SecretStore)
		assert.NotNil(t, before.Enterprise.CertificateIssuer)
	})

	t.Run("TheCountsFollowTheCreatedResources", func(t *testing.T) {
		h.Eventually(t, "the Cluster summary to carry the created resources",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
					&visibilityv1.GetClusterSummaryRequest{})
				if err != nil {
					return err
				}
				if res.Core.User.TotalNumber <= before.Core.User.TotalNumber {
					return errors.Errorf("the User summary is %d, was %d",
						res.Core.User.TotalNumber, before.Core.User.TotalNumber)
				}
				if res.Core.User.TotalWorkload <= before.Core.User.TotalWorkload {
					return errors.Errorf("the workload User summary is %d, was %d",
						res.Core.User.TotalWorkload, before.Core.User.TotalWorkload)
				}
				if res.Enterprise.Secret.TotalNumber <= before.Enterprise.Secret.TotalNumber {
					return errors.Errorf("the enterprise Secret summary is %d, was %d",
						res.Enterprise.Secret.TotalNumber, before.Enterprise.Secret.TotalNumber)
				}
				return nil
			})
	})

	t.Run("TheKindsRestrictTheResponse", func(t *testing.T) {
		res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
			&visibilityv1.GetClusterSummaryRequest{
				Kinds: []visibilityv1.GetClusterSummaryRequest_Kind{
					visibilityv1.GetClusterSummaryRequest_CORE_USER,
				},
			})
		require.Nil(t, err)
		require.NotNil(t, res.Core.User)
		assert.True(t, res.Core.User.TotalNumber > 0)

		assert.Nil(t, res.Core.Session)
		assert.Nil(t, res.Core.Service)
		assert.Nil(t, res.Access.Policy)
		assert.Nil(t, res.Enterprise.Secret)
	})

	t.Run("TheHumanAndWorkloadCountsAddUp", func(t *testing.T) {
		res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
			&visibilityv1.GetClusterSummaryRequest{
				Kinds: []visibilityv1.GetClusterSummaryRequest_Kind{
					visibilityv1.GetClusterSummaryRequest_CORE_USER,
				},
			})
		require.Nil(t, err)
		assert.Equal(t, res.Core.User.TotalNumber,
			res.Core.User.TotalHuman+res.Core.User.TotalWorkload)
	})

	t.Run("AStoppedRscStoreDegradesInsteadOfFailing", func(t *testing.T) {
		restore := h.StopEnterprise(t, "rscstore")

		h.Eventually(t, "the Cluster summary to report the rscstore kinds as unavailable",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
					&visibilityv1.GetClusterSummaryRequest{Kinds: rscStoreSummaryKinds})
				if err != nil {
					return err
				}

				got := unavailableKinds(res)
				for _, kind := range rscStoreSummaryKinds {
					if !slices.Contains(got, kind) {
						return errors.Errorf("the kind %s is not reported as unavailable, got %v",
							kind, got)
					}
				}

				for _, itm := range res.Unavailables {
					if itm.Message == "" {
						return errors.Errorf("the unavailable kind %s carries no message", itm.Kind)
					}
				}

				return nil
			})

		restore()

		h.Eventually(t, "the Cluster summary to recover after the rscstore restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.VisibilityClusterC().GetClusterSummary(ctx,
					&visibilityv1.GetClusterSummaryRequest{Kinds: rscStoreSummaryKinds})
				if err != nil {
					return err
				}
				if len(res.Unavailables) > 0 {
					return errors.Errorf("the kinds %v are still unavailable",
						unavailableKinds(res))
				}
				if res.Core.User.TotalNumber < 1 {
					return errors.Errorf("the User summary is still empty")
				}
				return nil
			})
	})

	t.Run("TheCreatedResourcesSurviveTheOutage", func(t *testing.T) {
		_, err := h.EnterpriseC().GetSecret(ctx, &metav1.GetOptions{Uid: sec.Metadata.Uid})
		assert.Nil(t, err)

		_, err = h.CoreC().GetUser(ctx, &metav1.GetOptions{Uid: usr.Metadata.Uid})
		assert.Nil(t, err)
	})
}

func testClusterHealthSubsystems(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	res, err := h.VisibilityClusterC().GetClusterHealth(ctx,
		&visibilityv1.GetClusterHealthRequest{})
	require.Nil(t, err)

	t.Run("EverySubsystemIsEvaluated", func(t *testing.T) {
		require.NotNil(t, res.From)
		require.NotNil(t, res.To)
		assert.True(t, res.To.AsTime().After(res.From.AsTime()))

		for _, sType := range append(slices.Clone(rscStoreSubsystems), logStoreSubsystems...) {
			subsystem := subsystemOf(res, sType)
			if !assert.NotNil(t, subsystem, "the subsystem %s is missing", sType) {
				continue
			}
			assert.NotEqual(t, visibilityv1.GetClusterHealthResponse_STATUS_UNSET,
				subsystem.Status, "the subsystem %s has no status", sType)
			assert.NotEmpty(t, subsystem.Message,
				"the subsystem %s carries no message", sType)
		}
	})

	t.Run("TheDataPlaneCarriesTheRegions", func(t *testing.T) {
		subsystem := subsystemOf(res,
			visibilityv1.GetClusterHealthResponse_Subsystem_DATA_PLANE)
		require.NotNil(t, subsystem)
		require.NotEmpty(t, res.Regions)

		for _, region := range res.Regions {
			assert.NotNil(t, region.RegionRef)
			assert.NotEqual(t, visibilityv1.GetClusterHealthResponse_STATUS_UNSET,
				region.Status)
		}
	})

	t.Run("TheAggregateIsTheWorstSubsystem", func(t *testing.T) {
		assert.NotEqual(t, visibilityv1.GetClusterHealthResponse_STATUS_UNSET, res.Status)

		worst := visibilityv1.GetClusterHealthResponse_OK
		for _, subsystem := range res.Subsystems {
			if healthRank(subsystem.Status) > healthRank(worst) {
				worst = subsystem.Status
			}
		}
		assert.Equal(t, worst, res.Status)
	})

	t.Run("APendingRequestIsReported", func(t *testing.T) {
		c := newAccessCast(t, h)
		h.CreateAccessPolicy(t, serviceReviewRule("cluster-health", c.alpha,
			userReviewStep(eeharness.UserReviewer(c.rita.User))))

		req := h.CreateRequest(t, c.alice,
			eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

		h.Eventually(t, "the Cluster health to report the pending access request",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				cur, err := h.VisibilityClusterC().GetClusterHealth(ctx,
					&visibilityv1.GetClusterHealthRequest{})
				if err != nil {
					return err
				}

				subsystem := subsystemOf(cur,
					visibilityv1.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS)
				if subsystem == nil {
					return errors.Errorf("the access request subsystem is missing")
				}
				if subsystem.Unhealthy < 1 {
					return errors.Errorf("the access request subsystem reports %d pending",
						subsystem.Unhealthy)
				}
				if subsystem.Status == visibilityv1.GetClusterHealthResponse_OK {
					return errors.Errorf("the access request subsystem is still OK")
				}
				return nil
			})
	})

	t.Run("AStoppedLogStoreOnlyDegradesItsOwnSubsystems", func(t *testing.T) {
		restore := h.StopEnterprise(t, "logstore")

		h.Eventually(t, "the log-backed subsystems to be reported as unknown",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				cur, err := h.VisibilityClusterC().GetClusterHealth(ctx,
					&visibilityv1.GetClusterHealthRequest{})
				if err != nil {
					return err
				}

				for _, sType := range logStoreSubsystems {
					subsystem := subsystemOf(cur, sType)
					if subsystem == nil {
						return errors.Errorf("the subsystem %s is missing", sType)
					}
					if subsystem.Status != visibilityv1.GetClusterHealthResponse_UNKNOWN {
						return errors.Errorf("the subsystem %s is %s, want UNKNOWN",
							sType, subsystem.Status)
					}
				}

				for _, sType := range rscStoreSubsystems {
					subsystem := subsystemOf(cur, sType)
					if subsystem == nil {
						return errors.Errorf("the subsystem %s is missing", sType)
					}
					if subsystem.Status == visibilityv1.GetClusterHealthResponse_UNKNOWN {
						return errors.Errorf("the subsystem %s became UNKNOWN too", sType)
					}
				}

				return nil
			})

		restore()

		h.Eventually(t, "the log-backed subsystems to recover after the logstore restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				cur, err := h.VisibilityClusterC().GetClusterHealth(ctx,
					&visibilityv1.GetClusterHealthRequest{})
				if err != nil {
					return err
				}

				for _, sType := range logStoreSubsystems {
					subsystem := subsystemOf(cur, sType)
					if subsystem == nil {
						return errors.Errorf("the subsystem %s is missing", sType)
					}
					if subsystem.Status == visibilityv1.GetClusterHealthResponse_UNKNOWN {
						return errors.Errorf("the subsystem %s is still UNKNOWN", sType)
					}
				}

				return nil
			})
	})
}

func testComponentLogAggregations(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	from := pbutils.Timestamp(time.Now().Add(-componentLogLookback))

	t.Run("TheComponentsAreRanked", func(t *testing.T) {
		h.Eventually(t, "the component log to rank the emitting components",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.ComponentLogC().ListComponentLogTopComponent(ctx,
					&visibilityv1.ListComponentLogTopComponentRequest{From: from, Limit: 100})
				if err != nil {
					return err
				}
				if len(res.Items) < 1 {
					return errors.Errorf("no component is ranked")
				}
				if res.TotalCount < uint64(len(res.Items)) {
					return errors.Errorf("the total count %d is below the %d ranked components",
						res.TotalCount, len(res.Items))
				}

				for _, itm := range res.Items {
					if itm.Component == nil || itm.Component.Type == "" {
						return errors.Errorf("a ranked item does not identify its component")
					}
					if itm.Count < 1 {
						return errors.Errorf("the component %s is ranked with no entries",
							itm.Component.Type)
					}
					severe := itm.CountWarn + itm.CountError + itm.CountPanic + itm.CountFatal
					if severe > itm.Count {
						return errors.Errorf(
							"the component %s reports %d severe entries of %d",
							itm.Component.Type, severe, itm.Count)
					}
				}

				return nil
			})
	})

	t.Run("TheSummaryAgreesWithTheRanking", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		ranking, err := h.ComponentLogC().ListComponentLogTopComponent(ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{From: from, Limit: 100})
		require.Nil(t, err)

		summary, err := h.ComponentLogC().GetComponentLogSummary(ctx,
			&visibilityv1.GetComponentLogSummaryRequest{From: from})
		require.Nil(t, err)

		var ranked uint64
		for _, itm := range ranking.Items {
			ranked += itm.Count
		}

		assert.True(t, summary.TotalNumber >= ranked,
			"the summary counts %d entries, the ranking counts %d",
			summary.TotalNumber, ranked)
	})

	t.Run("TheRankingIsScopedToASingleComponent", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		ranking, err := h.ComponentLogC().ListComponentLogTopComponent(ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{From: from, Limit: 100})
		require.Nil(t, err)
		require.NotEmpty(t, ranking.Items)

		want := ranking.Items[0].Component

		res, err := h.ComponentLogC().ListComponentLogTopComponent(ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{
				From:      from,
				Component: &visibilityv1.ComponentSelector{Type: want.Type},
			})
		require.Nil(t, err)
		require.NotEmpty(t, res.Items)

		for _, itm := range res.Items {
			assert.Equal(t, want.Type, itm.Component.Type)
		}
	})
}
