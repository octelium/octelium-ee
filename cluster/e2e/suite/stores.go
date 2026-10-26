// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const trafficRequests = 6

const (
	processMetricPrefix = "process."
	metricLookback      = 15 * time.Minute
)

func metricStep() *metav1.Duration {
	return eeharness.Minutes(1)
}

func testLogStoreQueryPath(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})
	svc := h.NewPublicService(t, "default")

	probe := h.Probe(t, usr, svc)
	probe.MustBeAllowed(t)

	from := pbutils.Timestamp(time.Now().Add(-time.Minute))

	for range trafficRequests {
		probe.MustBeAllowed(t)
	}

	t.Run("ScopedToTheUserAndService", func(t *testing.T) {
		h.Eventually(t, "the access log to carry the generated requests",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.AccessLogC().ListAccessLog(ctx,
					&visibilityv1.ListAccessLogRequest{
						UserRef:    umetav1.GetObjectReference(usr),
						ServiceRef: umetav1.GetObjectReference(svc),
						From:       from,
					})
				if err != nil {
					return err
				}
				if len(res.Items) < trafficRequests {
					return errors.Errorf("got %d access log entries, want at least %d",
						len(res.Items), trafficRequests)
				}

				for _, itm := range res.Items {
					if itm.Entry.Common.UserRef.Uid != usr.Metadata.Uid {
						return errors.Errorf("an entry belongs to another User")
					}
				}

				return nil
			})
	})

	t.Run("Pagination", func(t *testing.T) {
		first, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: umetav1.GetObjectReference(usr),
			Common:  &vmetav1.CommonListOptions{Page: 0, ItemsPerPage: 2},
		})
		require.Nil(t, err)
		assert.True(t, len(first.Items) <= 2)

		second, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: umetav1.GetObjectReference(usr),
			Common:  &vmetav1.CommonListOptions{Page: 1, ItemsPerPage: 2},
		})
		require.Nil(t, err)

		for _, itm := range second.Items {
			assert.False(t, slices.ContainsFunc(first.Items,
				func(o *corev1.AccessLog) bool {
					return o.Metadata.Id == itm.Metadata.Id
				}))
		}
	})

	t.Run("Aggregations", func(t *testing.T) {
		summary, err := h.AccessLogC().GetAccessLogSummary(ctx,
			&visibilityv1.GetAccessLogSummaryRequest{
				UserRef:    umetav1.GetObjectReference(usr),
				ServiceRef: umetav1.GetObjectReference(svc),
				From:       from,
			})
		require.Nil(t, err)
		assert.GreaterOrEqual(t, summary.TotalNumber, uint64(trafficRequests))
		assert.GreaterOrEqual(t, summary.TotalAllowed, uint64(trafficRequests))

		points, err := h.AccessLogC().GetAccessLogDataPoint(ctx,
			&visibilityv1.GetAccessLogDataPointRequest{
				UserRef:    umetav1.GetObjectReference(usr),
				ServiceRef: umetav1.GetObjectReference(svc),
				From:       from,
			})
		require.Nil(t, err)
		assert.NotEmpty(t, points.Datapoints)

		topUsers, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: umetav1.GetObjectReference(svc),
				From:       from,
			})
		require.Nil(t, err)
		assert.True(t, slices.ContainsFunc(topUsers.Items,
			func(itm *visibilityv1.ListAccessLogTopUserResponse_Item) bool {
				return itm.User.Metadata.Uid == usr.Metadata.Uid && itm.Count >= trafficRequests
			}))

		topServices, err := h.AccessLogC().ListAccessLogTopService(ctx,
			&visibilityv1.ListAccessLogTopServiceRequest{
				UserRef: umetav1.GetObjectReference(usr),
				From:    from,
			})
		require.Nil(t, err)
		assert.True(t, slices.ContainsFunc(topServices.Items,
			func(itm *visibilityv1.ListAccessLogTopServiceResponse_Item) bool {
				return itm.Service.Metadata.Uid == svc.Metadata.Uid && itm.Count >= trafficRequests
			}))
	})

	t.Run("ComponentLog", func(t *testing.T) {
		res, err := h.ComponentLogC().ListComponentLog(ctx,
			&visibilityv1.ListComponentLogRequest{From: from})
		require.Nil(t, err)
		assert.True(t, len(res.Items) > 0)
	})

	t.Run("SurvivesARestart", func(t *testing.T) {
		before, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: umetav1.GetObjectReference(usr),
		})
		require.Nil(t, err)

		h.RestartEnterprise(t, "logstore")

		h.Eventually(t, "the logstore to answer after the restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				after, err := h.AccessLogC().ListAccessLog(ctx,
					&visibilityv1.ListAccessLogRequest{
						UserRef: umetav1.GetObjectReference(usr),
					})
				if err != nil {
					return err
				}
				if after.ListResponseMeta.TotalCount < before.ListResponseMeta.TotalCount {
					return errors.Errorf("the access log lost entries: %d, was %d",
						after.ListResponseMeta.TotalCount, before.ListResponseMeta.TotalCount)
				}
				return nil
			})
	})
}

func testAccessLogLifecycle(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, serviceReviewRule("access-log-lifecycle", c.alpha,
		userReviewStep(eeharness.UserReviewer(c.rita.User))))
	probe := h.Probe(t, c.alice.User, c.alpha)
	from := pbutils.Timestamp(time.Now().Add(-time.Second))
	probe.MustBeDenied(t)

	req := h.CreateRequest(t, c.alice,
		eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)
	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	approvedAt := time.Now()
	probe.MustBeAllowed(t)

	_, err := h.AccessC().RevokeRequest(t.Context(),
		&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
	require.Nil(t, err)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_REVOKED,
		eeharness.RequestBudget)
	revokedAt := time.Now()
	probe.MustBeDenied(t)

	h.Eventually(t, "the access log to carry the authorization lifecycle",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
				UserRef:    umetav1.GetObjectReference(c.alice.User),
				ServiceRef: umetav1.GetObjectReference(c.alpha),
				From:       from,
			})
			if err != nil {
				return err
			}

			var deniedBefore, allowed, deniedAfter bool
			for _, itm := range res.Items {
				if itm.Entry == nil || itm.Entry.Common == nil ||
					itm.Entry.Common.StartedAt == nil {
					continue
				}
				common := itm.Entry.Common
				if common.UserRef == nil || common.UserRef.Uid != c.alice.User.Metadata.Uid ||
					common.ServiceRef == nil || common.ServiceRef.Uid != c.alpha.Metadata.Uid {
					return errors.Errorf("an access entry has incorrect resource references")
				}

				started := common.StartedAt.AsTime()
				switch {
				case started.Before(approvedAt) &&
					common.Status == corev1.AccessLog_Entry_Common_DENIED:
					deniedBefore = true
				case !started.Before(approvedAt) && started.Before(revokedAt) &&
					common.Status == corev1.AccessLog_Entry_Common_ALLOWED:
					allowed = common.Reason != nil &&
						common.Reason.Type == corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH
				case !started.Before(revokedAt) &&
					common.Status == corev1.AccessLog_Entry_Common_DENIED:
					deniedAfter = true
				}
			}

			if !deniedBefore || !allowed || !deniedAfter {
				return errors.Errorf("the lifecycle entries are incomplete: before=%v allowed=%v after=%v",
					deniedBefore, allowed, deniedAfter)
			}
			return nil
		})
}

func testLogStoreIngestionRecovery(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})
	svc := h.NewPublicService(t, "default")
	probe := h.Probe(t, usr, svc)
	probe.MustBeAllowed(t)

	restore := h.StopEnterprise(t, "logstore")
	from := pbutils.Timestamp(time.Now().Add(-time.Second))
	for range 3 {
		probe.MustBeAllowed(t)
	}
	restore()

	h.Eventually(t, "the logstore to ingest entries produced while it was stopped",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
				UserRef:    umetav1.GetObjectReference(usr),
				ServiceRef: umetav1.GetObjectReference(svc),
				From:       from,
			})
			if err != nil {
				return err
			}
			if len(res.Items) < 3 {
				return errors.Errorf("got %d recovered entries, want at least 3", len(res.Items))
			}
			return nil
		})
}

func testMetricStoreQueryPath(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()
	var descriptor *vmetricsv1.MetricDescriptor

	newTrafficDriver(t, h).drive(t)

	t.Run("Capabilities", func(t *testing.T) {
		res, err := h.MetricsC().GetMetricsCapabilities(ctx,
			&vmetricsv1.GetMetricsCapabilitiesRequest{})
		require.Nil(t, err)
		assert.NotNil(t, res.QueryLimits)
		assert.NotNil(t, res.IngestionLimits)
		assert.NotNil(t, res.ServerTime)
		assert.NotEmpty(t, res.RetentionTiers)
		assert.NotEmpty(t, res.MetricKinds)
	})

	t.Run("Catalog", func(t *testing.T) {
		res, err := h.MetricsC().ListMetricCatalog(ctx,
			&vmetricsv1.ListMetricCatalogRequest{})
		require.Nil(t, err)
		assert.True(t, len(res.Items) > 0)
	})

	t.Run("Descriptors", func(t *testing.T) {
		h.Eventually(t, "the ingested Cluster metric descriptors to be listed",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.MetricsC().ListMetricDescriptors(ctx,
					&vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: processMetricPrefix})
				if err != nil {
					return err
				}
				if len(res.Items) < 1 {
					return errors.Errorf("no descriptor carries the %q prefix",
						processMetricPrefix)
				}
				descriptor = res.Items[0]
				if descriptor.Id == "" || descriptor.Name == "" ||
					descriptor.Kind == vmetricsv1.MetricDescriptor_KIND_UNSET {
					return errors.Errorf("the metric descriptor is incomplete")
				}
				return nil
			})
	})

	t.Run("Query", func(t *testing.T) {
		h.Eventually(t, "a catalog metric to answer with datapoints",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				return queryAnyCatalogMetric(ctx, h)
			})
	})

	t.Run("UnknownMetricIsRejected", func(t *testing.T) {
		_, err := h.MetricsC().QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
			Metric: &vmetricsv1.MetricSelector{
				Selector: &vmetricsv1.MetricSelector_Name{Name: h.Name()},
			},
			TimeRange: metricRange(),
			Step:      metricStep(),
			Operation: gaugeLast(),
		})
		assert.NotNil(t, err)
	})

	t.Run("SurvivesARestart", func(t *testing.T) {
		require.NotNil(t, descriptor)
		h.RestartEnterprise(t, "metricstore")

		h.Eventually(t, "the metricstore to answer after the restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.MetricsC().ListMetricDescriptors(ctx,
					&vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: descriptor.Name})
				if err != nil {
					return err
				}
				if !slices.ContainsFunc(res.Items, func(itm *vmetricsv1.MetricDescriptor) bool {
					return itm.Id == descriptor.Id
				}) {
					return errors.Errorf("the stable descriptor %s is missing", descriptor.Id)
				}
				return queryAnyCatalogMetric(ctx, h)
			})
	})
}

func queryAnyCatalogMetric(ctx context.Context, h *eeharness.H) error {
	catalog, err := h.MetricsC().ListMetricCatalog(ctx, &vmetricsv1.ListMetricCatalogRequest{})
	if err != nil {
		return err
	}

	var lastErr error

	for _, itm := range catalog.Items {
		res, err := h.MetricsC().QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
			Metric:            itm.Metric,
			TimeRange:         metricRange(),
			Step:              cmp.Or(itm.DefaultStep, metricStep()),
			Operation:         cmp.Or(itm.DefaultOperation, gaugeLast()),
			GroupBy:           itm.DefaultGroupBy,
			Filters:           itm.DefaultFilters,
			SeriesAggregation: itm.DefaultSeriesAggregation,
		})
		if err != nil {
			lastErr = errors.Errorf("the catalog metric %s could not be queried: %+v",
				itm.Id, err)
			continue
		}
		if len(res.Series) > 0 {
			return nil
		}

		lastErr = errors.Errorf("the catalog metric %s has no series yet", itm.Id)
	}

	if lastErr == nil {
		lastErr = errors.Errorf("the metric catalog is empty")
	}

	return lastErr
}

func metricRange() *vmetricsv1.TimeRange {
	return &vmetricsv1.TimeRange{
		From: pbutils.Timestamp(time.Now().Add(-metricLookback)),
		To:   pbutils.Timestamp(time.Now().Add(time.Minute)),
	}
}

func gaugeLast() *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{
		Type: &vmetricsv1.QueryOperation_Gauge{
			Gauge: &vmetricsv1.GaugeOperation{Function: vmetricsv1.GaugeOperation_LAST},
		},
	}
}

func testRscStoreReconciliation(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	usr := h.CreateWorkloadUser(t, nil)

	t.Run("Create", func(t *testing.T) {
		waitVisibilityUser(t, h, usr, true)
	})

	t.Run("Update", func(t *testing.T) {
		usr.Metadata.DisplayName = h.Name()
		usr = h.UpdateUser(t, usr)

		h.Eventually(t, "the visibility mirror to carry the update",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{})
				if err != nil {
					return err
				}

				idx := slices.IndexFunc(res.Items, func(itm *corev1.User) bool {
					return itm.Metadata.Uid == usr.Metadata.Uid
				})
				if idx < 0 {
					return errors.Errorf("the User is not mirrored")
				}
				if res.Items[idx].Metadata.DisplayName != usr.Metadata.DisplayName {
					return errors.Errorf("the mirrored displayName is %q, want %q",
						res.Items[idx].Metadata.DisplayName, usr.Metadata.DisplayName)
				}

				return nil
			})
	})

	t.Run("Delete", func(t *testing.T) {
		_, err := h.CoreC().DeleteUser(ctx, &metav1.DeleteOptions{Uid: usr.Metadata.Uid})
		require.Nil(t, err)

		waitVisibilityUser(t, h, usr, false)
	})

	t.Run("Counts", func(t *testing.T) {
		h.Eventually(t, "the mirrored User count to converge with the core count",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{})
				if err != nil {
					return err
				}

				usrList, err := h.CoreC().ListUser(ctx, &corev1.ListUserOptions{})
				if err != nil {
					return err
				}

				if res.ListResponseMeta.TotalCount != usrList.ListResponseMeta.TotalCount {
					return errors.Errorf("the mirror has %d Users, core has %d",
						res.ListResponseMeta.TotalCount, usrList.ListResponseMeta.TotalCount)
				}

				return nil
			})
	})

	t.Run("SurvivesARestart", func(t *testing.T) {
		survivor := h.CreateWorkloadUser(t, nil)
		waitVisibilityUser(t, h, survivor, true)

		h.RestartEnterprise(t, "rscstore")

		waitVisibilityUser(t, h, survivor, true)
	})

	t.Run("ReconcilesChangesMissedWhileStopped", func(t *testing.T) {
		restore := h.StopEnterprise(t, "rscstore")

		updated := h.CreateWorkloadUser(t, nil)
		updated.Metadata.DisplayName = h.Name()
		updated = h.UpdateUser(t, updated)

		deleted := h.CreateWorkloadUser(t, nil)
		_, err := h.CoreC().DeleteUser(t.Context(),
			&metav1.DeleteOptions{Uid: deleted.Metadata.Uid})
		require.Nil(t, err)

		restore()

		h.Eventually(t, "the visibility mirror to reconcile missed changes",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{})
				if err != nil {
					return err
				}

				idx := slices.IndexFunc(res.Items, func(itm *corev1.User) bool {
					return itm.Metadata.Uid == updated.Metadata.Uid
				})
				if idx < 0 || res.Items[idx].Metadata.DisplayName != updated.Metadata.DisplayName {
					return errors.Errorf("the updated User did not converge")
				}
				if slices.ContainsFunc(res.Items, func(itm *corev1.User) bool {
					return itm.Metadata.Uid == deleted.Metadata.Uid
				}) {
					return errors.Errorf("the deleted User is still mirrored")
				}
				return nil
			})
	})
}

func testRscStoreAccessResources(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("visibility-access",
		userReviewStep(eeharness.UserReviewer(c.rita.User))))
	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)
	review := h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	waitVisibilityRequestState(t, h, req,
		accessv1.Request_Status_State_APPROVED, true)

	h.Eventually(t, "the visibility mirror to carry the Review",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.VisibilityAccessC().ListReview(ctx, &vaccessv1.ListReviewOptions{
				RequestRef: umetav1.GetObjectReference(req),
			})
			if err != nil {
				return err
			}
			if !slices.ContainsFunc(res.Items, func(itm *accessv1.Review) bool {
				return itm.Metadata.Uid == review.Metadata.Uid
			}) {
				return errors.Errorf("the Review is not mirrored")
			}
			return nil
		})

	_, err := h.AccessC().RevokeRequest(t.Context(),
		&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
	require.Nil(t, err)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_REVOKED,
		eeharness.RequestBudget)
	waitVisibilityRequestState(t, h, req,
		accessv1.Request_Status_State_REVOKED, true)

	_, err = h.AccessC().DeleteRequest(t.Context(),
		&metav1.DeleteOptions{Uid: req.Metadata.Uid})
	require.Nil(t, err)
	waitVisibilityRequestState(t, h, req,
		accessv1.Request_Status_State_STATUS_UNKNOWN, false)
}

func testRscStoreCordiumResources(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	if !h.IsCordiumInstalled(t) {
		t.Skip("Cordium is not installed in this Cluster")
	}

	actor := h.NewActorWithAuthorization(t, &corev1.User_Spec_Authorization{
		InlinePolicies: eeharness.APIPolicy("cordium-api", eeharness.CordiumMainService),
	})
	cordiumC := h.CordiumC(actor.Conn)

	spc, err := cordiumC.CreateSpace(ctx, &cordiumv1.Space{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("%s.%s", h.Name(), actor.User.Metadata.Name),
		},
		Spec: &cordiumv1.Space_Spec{},
	})
	require.Nil(t, err)

	t.Cleanup(func() {
		cordiumC.DeleteSpace(context.Background(), &metav1.DeleteOptions{Uid: spc.Metadata.Uid})
	})

	waitVisibilitySpace(t, h, spc, true)

	t.Run("Summary", func(t *testing.T) {
		res, err := h.VisibilityCordiumC().GetSpaceSummary(ctx, &vcordiumv1.GetSpaceSummaryRequest{})
		require.Nil(t, err)
		assert.True(t, res.TotalNumber > 0)
		assert.True(t, res.TotalUserSpace > 0)
	})

	t.Run("EnabledRegions", func(t *testing.T) {
		res, err := h.VisibilityCordiumC().ListRegion(ctx, &vcordiumv1.ListRegionOptions{
			IsEnabled: true,
		})
		require.Nil(t, err)
		require.NotEmpty(t, res.Items)
		assert.Equal(t, "cordium/v1", res.Items[0].ApiVersion)

		summary, err := h.VisibilityCordiumC().GetRegionSummary(ctx, &vcordiumv1.GetRegionSummaryRequest{})
		require.Nil(t, err)
		assert.EqualValues(t, len(res.Items), summary.TotalEnabled)
	})

	t.Run("Delete", func(t *testing.T) {
		_, err := cordiumC.DeleteSpace(ctx, &metav1.DeleteOptions{Uid: spc.Metadata.Uid})
		require.Nil(t, err)

		waitVisibilitySpace(t, h, spc, false)
	})
}

func waitVisibilitySpace(t *testing.T, h *eeharness.H, spc *cordiumv1.Space, want bool) {
	t.Helper()

	what := "the visibility mirror to carry the Space"
	if !want {
		what = "the visibility mirror to drop the Space"
	}

	h.Eventually(t, what, eeharness.IngestionBudget, func(ctx context.Context) error {
		res, err := h.VisibilityCordiumC().ListSpace(ctx, &vcordiumv1.ListSpaceOptions{
			UserRef: spc.Status.UserRef,
		})
		if err != nil {
			return err
		}

		got := slices.ContainsFunc(res.Items, func(itm *cordiumv1.Space) bool {
			return itm.Metadata.Uid == spc.Metadata.Uid
		})
		if got != want {
			return errors.Errorf("the Space mirror presence is %v, want %v", got, want)
		}

		return nil
	})
}

func waitVisibilityRequestState(t *testing.T, h *eeharness.H, req *accessv1.Request,
	want accessv1.Request_Status_State_Status, present bool) {
	t.Helper()

	h.Eventually(t, "the visibility Request mirror to converge",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.VisibilityAccessC().ListRequest(ctx, &vaccessv1.ListRequestOptions{
				UserRef: req.Status.UserRef,
			})
			if err != nil {
				return err
			}

			idx := slices.IndexFunc(res.Items, func(itm *accessv1.Request) bool {
				return itm.Metadata.Uid == req.Metadata.Uid
			})
			if !present {
				if idx >= 0 {
					return errors.Errorf("the Request is still mirrored")
				}
				return nil
			}
			if idx < 0 {
				return errors.Errorf("the Request is not mirrored")
			}
			if res.Items[idx].Status.State.Status != want {
				return errors.Errorf("the mirrored Request is %s, want %s",
					res.Items[idx].Status.State.Status, want)
			}
			return nil
		})
}

func testRscStoreSecretRedaction(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	coreSecret := h.CreateCoreSecret(t, h.Name()+h.Name())
	enterpriseSecret := h.CreateEnterpriseSecret(t, h.Name()+h.Name())

	h.Eventually(t, "the visibility mirrors to carry redacted Secrets",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			coreList, err := h.VisibilityCoreC().ListSecret(ctx, &vcorev1.ListSecretOptions{})
			if err != nil {
				return err
			}
			coreIdx := slices.IndexFunc(coreList.Items, func(itm *corev1.Secret) bool {
				return itm.Metadata.Uid == coreSecret.Metadata.Uid
			})
			if coreIdx < 0 {
				return errors.Errorf("the core Secret is not mirrored")
			}
			if coreList.Items[coreIdx].Data != nil {
				return errors.Errorf("the core Secret mirror contains data")
			}

			enterpriseList, err := h.VisibilityEnterpriseC().ListSecret(ctx,
				&venterprisev1.ListSecretOptions{})
			if err != nil {
				return err
			}
			enterpriseIdx := slices.IndexFunc(enterpriseList.Items,
				func(itm *enterprisev1.Secret) bool {
					return itm.Metadata.Uid == enterpriseSecret.Metadata.Uid
				})
			if enterpriseIdx < 0 {
				return errors.Errorf("the enterprise Secret is not mirrored")
			}
			if enterpriseList.Items[enterpriseIdx].Data != nil {
				return errors.Errorf("the enterprise Secret mirror contains data")
			}

			return nil
		})
}

func waitVisibilityUser(t *testing.T, h *eeharness.H, usr *corev1.User, want bool) {
	t.Helper()

	what := "the visibility mirror to carry the User"
	if !want {
		what = "the visibility mirror to drop the User"
	}

	h.Eventually(t, what, eeharness.IngestionBudget, func(ctx context.Context) error {
		res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{})
		if err != nil {
			return err
		}

		got := slices.ContainsFunc(res.Items, func(itm *corev1.User) bool {
			return itm.Metadata.Uid == usr.Metadata.Uid
		})
		if got != want {
			return errors.Errorf("the User mirror presence is %v, want %v", got, want)
		}

		return nil
	})
}

func testVisibilityScoping(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	a := h.NewActor(t)

	_, err := userv1.NewMainServiceClient(a.Conn).GetStatus(ctx, &userv1.GetStatusRequest{})
	require.Nil(t, err)

	t.Run("UnauthorizedByDefault", func(t *testing.T) {
		_, err := visibilityv1.NewAccessLogServiceClient(a.Conn).ListAccessLog(ctx,
			&visibilityv1.ListAccessLogRequest{})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))
	})

}

func testRscStoreSearchLifecycle(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	oldQuery := h.Name()
	newQuery := h.Name()
	oldTag := h.Name()
	newTag := h.Name()
	var users []*corev1.User
	for range 5 {
		users = append(users, h.CreateUser(t, &corev1.User{
			Metadata: &metav1.Metadata{
				Description: oldQuery + " literal%_\\" + oldQuery,
				Tags:        []string{oldTag},
			},
			Spec: &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
		}))
	}
	h.CreateUser(t, &corev1.User{
		Metadata: &metav1.Metadata{Description: oldQuery, Tags: []string{h.Name()}},
		Spec:     &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
	})
	slices.SortFunc(users, func(a, b *corev1.User) int {
		return cmp.Compare(a.Metadata.Name, b.Metadata.Name)
	})

	h.Eventually(t, "the search and tag indexes to include all five Users",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{
				Common: &vmetav1.CommonListOptions{Query: oldQuery, Tag: oldTag},
			})
			if err != nil {
				return err
			}
			if len(res.Items) != len(users) || res.GetListResponseMeta().GetTotalCount() != uint32(len(users)) {
				return errors.Errorf("the search has not converged to five Users")
			}
			return nil
		})

	t.Run("PaginationAndLiteralSearch", func(t *testing.T) {
		var got []string
		var want []string
		for _, usr := range users {
			want = append(want, usr.Metadata.Uid)
		}
		for page := range uint32(3) {
			res, err := h.VisibilityCoreC().ListUser(t.Context(), &vcorev1.ListUserOptions{
				Common: &vmetav1.CommonListOptions{
					Query: "  " + strings.ToUpper(oldQuery) + "  ", Tag: oldTag,
					Page: page, ItemsPerPage: 2,
					OrderBy: &vmetav1.CommonListOptions_OrderBy{
						Type: vmetav1.CommonListOptions_OrderBy_NAME,
						Mode: vmetav1.CommonListOptions_OrderBy_ASC,
					},
				},
			})
			require.Nil(t, err)
			assert.EqualValues(t, 5, res.ListResponseMeta.TotalCount)
			assert.Equal(t, page, res.ListResponseMeta.Page)
			assert.EqualValues(t, 2, res.ListResponseMeta.ItemsPerPage)
			assert.Equal(t, page < 2, res.ListResponseMeta.HasMore)
			for _, usr := range res.Items {
				got = append(got, usr.Metadata.Uid)
			}
		}
		assert.Equal(t, want, got)
		res, err := h.VisibilityCoreC().ListUser(t.Context(), &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{Query: "%_\\" + oldQuery, Tag: oldTag},
		})
		require.Nil(t, err)
		assert.Len(t, res.Items, 5)
		res, err = h.VisibilityCoreC().ListUser(t.Context(), &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{Query: "' OR 1=1 --", Tag: oldTag},
		})
		require.Nil(t, err)
		assert.Empty(t, res.Items)
	})

	for i, usr := range users {
		usr.Metadata.Description = newQuery
		usr.Metadata.Tags = []string{newTag}
		users[i] = h.UpdateUser(t, usr)
	}
	waitUpdated := func(ctx context.Context, count uint32) error {
		old, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{Query: oldQuery, Tag: oldTag},
		})
		if err != nil {
			return err
		}
		if len(old.Items) != 0 || old.GetListResponseMeta().GetTotalCount() != 0 {
			return errors.Errorf("the old search and tag indexes still match updated Users")
		}
		res, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{Query: newQuery, Tag: newTag},
		})
		if err != nil {
			return err
		}
		if uint32(len(res.Items)) != count || res.GetListResponseMeta().GetTotalCount() != count {
			return errors.Errorf("the updated search has %d Users, want %d", len(res.Items), count)
		}
		for _, usr := range res.Items {
			if usr.Metadata.Description != newQuery || !slices.Equal(usr.Metadata.Tags, []string{newTag}) {
				return errors.Errorf("the indexed User does not carry the updated metadata")
			}
		}
		return nil
	}
	h.Eventually(t, "metadata updates to replace the searchable values and tags",
		eeharness.IngestionBudget, func(ctx context.Context) error { return waitUpdated(ctx, 5) })

	_, err := h.CoreC().DeleteUser(t.Context(), &metav1.DeleteOptions{Uid: users[0].Metadata.Uid})
	require.Nil(t, err)
	h.Eventually(t, "the search count to drop after deletion", eeharness.IngestionBudget,
		func(ctx context.Context) error { return waitUpdated(ctx, 4) })

	h.RestartEnterprise(t, "rscstore")
	h.Eventually(t, "the rebuilt search indexes to preserve updates and deletions",
		eeharness.IngestionBudget, func(ctx context.Context) error { return waitUpdated(ctx, 4) })
}

func testRscStoreSummaryWindows(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	first := h.CreateCatalog(t, nil, []string{"default"})
	second := h.CreateCatalog(t, nil, []string{"default"})
	from := pbutils.Timestamp(first.Metadata.CreatedAt.AsTime().Add(-time.Second))
	to := pbutils.Timestamp(second.Metadata.CreatedAt.AsTime().Add(time.Second))
	futureFrom := pbutils.Timestamp(to.AsTime().Add(24 * time.Hour))
	futureTo := pbutils.Timestamp(futureFrom.AsTime().Add(time.Hour))
	var count uint64

	h.Eventually(t, "the windowed Catalog list and summary to agree", eeharness.IngestionBudget,
		func(ctx context.Context) error {
			list, err := h.VisibilityAccessC().ListCatalog(ctx, &vaccessv1.ListCatalogOptions{
				Common: &vmetav1.CommonListOptions{From: from, To: to, ItemsPerPage: 1000},
			})
			if err != nil {
				return err
			}
			for _, uid := range []string{first.Metadata.Uid, second.Metadata.Uid} {
				if !slices.ContainsFunc(list.Items, func(itm *accessv1.Catalog) bool { return itm.Metadata.Uid == uid }) {
					return errors.Errorf("a Catalog is not in the windowed mirror")
				}
			}
			res, err := h.VisibilityAccessC().GetCatalogSummary(ctx, &vaccessv1.GetCatalogSummaryRequest{
				Common: &vmetav1.CommonSummaryOptions{
					From: from, To: to, CompareFrom: futureFrom, CompareTo: futureTo,
				},
			})
			if err != nil {
				return err
			}
			if res.TotalNumber != uint64(list.GetListResponseMeta().GetTotalCount()) ||
				res.Previous == nil || res.Previous.TotalNumber != 0 {
				return errors.Errorf("the Catalog summary or its empty comparison window has not converged")
			}
			count = res.TotalNumber
			return nil
		})

	t.Run("ComparisonUsesItsOwnWindow", func(t *testing.T) {
		res, err := h.VisibilityAccessC().GetCatalogSummary(t.Context(), &vaccessv1.GetCatalogSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From: futureFrom, To: futureTo, CompareFrom: from, CompareTo: to,
			},
		})
		require.Nil(t, err)
		assert.Zero(t, res.TotalNumber)
		require.NotNil(t, res.Previous)
		assert.Equal(t, count, res.Previous.TotalNumber)
		assert.Nil(t, res.Previous.Previous)
	})

	t.Run("ComparisonIsOptional", func(t *testing.T) {
		res, err := h.VisibilityAccessC().GetCatalogSummary(t.Context(), &vaccessv1.GetCatalogSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{From: from, To: to},
		})
		require.Nil(t, err)
		assert.Equal(t, count, res.TotalNumber)
		assert.Nil(t, res.Previous)
	})
}

func testRscStoreAccessActivity(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)
	c.alice.User.Metadata.DisplayName = h.Name()
	c.alice.User = h.UpdateUser(t, c.alice.User)
	policy := h.CreateAccessPolicy(t, c.reviewRule("visibility-activity",
		userReviewStep(eeharness.UserReviewer(c.rita.User))))
	approved := h.CreateRequest(t, c.alice, eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, approved, accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)
	h.Review(t, c.rita, approved, accessv1.Review_Spec_DECISION_APPROVE)
	approved = h.WaitRequestState(t, approved, accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)
	pending := h.CreateRequest(t, c.bob, eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	pending = h.WaitRequestState(t, pending, accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)
	from := pbutils.Timestamp(approved.Metadata.CreatedAt.AsTime().Add(-5 * time.Minute).Truncate(time.Minute))
	to := pbutils.Timestamp(pending.Metadata.CreatedAt.AsTime().Add(5 * time.Minute).Truncate(time.Minute))

	waitCounts := func(ctx context.Context, want map[string]int64) error {
		res, err := h.VisibilityAccessC().GetRequestDataPoint(ctx, &vaccessv1.GetRequestDataPointRequest{
			From: from, To: to, Interval: eeharness.Minutes(1),
			GroupBy:   vaccessv1.GetRequestDataPointRequest_STATE,
			PolicyRef: umetav1.GetObjectReference(policy), CatalogRef: umetav1.GetObjectReference(c.alphaCatalog),
		})
		if err != nil {
			return err
		}
		var total int64
		for i, point := range res.Datapoints {
			total += point.Count
			if i > 0 && point.Timestamp.AsTime().Sub(res.Datapoints[i-1].Timestamp.AsTime()) != time.Minute {
				return errors.Errorf("the activity datapoints are not a continuous minute series")
			}
		}
		var wantTotal int64
		for _, count := range want {
			wantTotal += count
		}
		if total != wantTotal || len(res.Series) != len(want) || len(res.Datapoints) < 2 {
			return errors.Errorf("the activity has total %d and %d state series", total, len(res.Series))
		}
		if res.Datapoints[0].Count != 0 || res.Datapoints[len(res.Datapoints)-1].Count != 0 {
			return errors.Errorf("empty activity buckets are not zero filled")
		}
		for _, series := range res.Series {
			if series.Total != want[series.Key] {
				return errors.Errorf("the %s series has total %d, want %d", series.Key, series.Total, want[series.Key])
			}
			var count int64
			for _, point := range series.Datapoints {
				count += point.Count
			}
			if count != series.Total || len(series.Datapoints) != len(res.Datapoints) {
				return errors.Errorf("the %s series does not match its total or bucket count", series.Key)
			}
		}
		return nil
	}
	h.Eventually(t, "the activity to count approved and pending Requests once",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			return waitCounts(ctx, map[string]int64{"APPROVED": 1, "PENDING": 1})
		})

	t.Run("RequesterRankingLimitsAndDisplayNames", func(t *testing.T) {
		h.Eventually(t, "the requester ranking to include enriched Users", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				res, err := h.VisibilityAccessC().ListRequestTop(ctx, &vaccessv1.ListRequestTopRequest{
					Dimension: vaccessv1.ListRequestTopRequest_REQUESTER,
					From:      from, To: to, CatalogRef: umetav1.GetObjectReference(c.alphaCatalog), Limit: 2,
				})
				if err != nil {
					return err
				}
				if len(res.Items) != 2 || res.TotalCount != 2 || res.TotalOther != 0 {
					return errors.Errorf("the requester ranking has not converged")
				}
				for _, itm := range res.Items {
					if itm.Count != 1 || itm.Ref.Kind != "User" || itm.Ref.ApiVersion != "core/v1" ||
						(itm.Ref.Uid != c.alice.User.Metadata.Uid && itm.Ref.Uid != c.bob.User.Metadata.Uid) {
						return errors.Errorf("the ranking contains an unexpected requester")
					}
					if itm.Ref.Uid == c.alice.User.Metadata.Uid && itm.DisplayName != c.alice.User.Metadata.DisplayName {
						return errors.Errorf("the requester displayName is not enriched")
					}
				}
				return nil
			})
		res, err := h.VisibilityAccessC().ListRequestTop(t.Context(), &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_REQUESTER,
			From:      from, To: to, CatalogRef: umetav1.GetObjectReference(c.alphaCatalog), Limit: 1,
		})
		require.Nil(t, err)
		assert.Len(t, res.Items, 1)
		assert.EqualValues(t, 2, res.TotalCount)
		assert.EqualValues(t, 1, res.TotalOther)
	})

	t.Run("ResourceAndPolicyRankings", func(t *testing.T) {
		for _, tt := range []struct {
			dimension vaccessv1.ListRequestTopRequest_Dimension
			ref       *metav1.ObjectReference
		}{
			{dimension: vaccessv1.ListRequestTopRequest_RESOURCE, ref: umetav1.GetObjectReference(c.alphaCatalog)},
			{dimension: vaccessv1.ListRequestTopRequest_POLICY, ref: umetav1.GetObjectReference(policy)},
		} {
			res, err := h.VisibilityAccessC().ListRequestTop(t.Context(), &vaccessv1.ListRequestTopRequest{
				Dimension: tt.dimension, From: from, To: to, CatalogRef: umetav1.GetObjectReference(c.alphaCatalog),
			})
			require.Nil(t, err)
			require.Len(t, res.Items, 1)
			assert.Equal(t, tt.ref.Uid, res.Items[0].Ref.Uid)
			assert.EqualValues(t, 2, res.Items[0].Count)
			assert.EqualValues(t, 1, res.TotalCount)
			assert.Zero(t, res.TotalOther)
		}
	})

	t.Run("ReviewerRanking", func(t *testing.T) {
		h.Eventually(t, "the approval Review to appear in reviewer activity", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				res, err := h.VisibilityAccessC().ListReviewTop(ctx, &vaccessv1.ListReviewTopRequest{
					Dimension: vaccessv1.ListReviewTopRequest_REVIEWER,
					From:      from, To: to, Decision: accessv1.Review_Spec_DECISION_APPROVE, Limit: 100,
				})
				if err != nil {
					return err
				}
				if !slices.ContainsFunc(res.Items, func(itm *vaccessv1.ListTopResponse_Item) bool {
					return itm.Ref.Uid == c.rita.User.Metadata.Uid && itm.Count == 1
				}) {
					return errors.Errorf("the approving reviewer is not ranked")
				}
				return nil
			})
	})

	_, err := h.AccessC().RevokeRequest(t.Context(), &accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(approved)})
	require.Nil(t, err)
	h.WaitRequestState(t, approved, accessv1.Request_Status_State_REVOKED, eeharness.RequestBudget)
	_, err = h.AccessUserC(c.bob.Conn).CancelRequest(t.Context(), &accessv1.CancelRequestRequest{RequestRef: umetav1.GetObjectReference(pending)})
	require.Nil(t, err)
	h.WaitRequestState(t, pending, accessv1.Request_Status_State_CANCELLED, eeharness.RequestBudget)
	h.Eventually(t, "state changes to move activity between series without increasing totals",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			return waitCounts(ctx, map[string]int64{"REVOKED": 1, "CANCELLED": 1})
		})
	_, err = h.AccessC().DeleteRequest(t.Context(), &metav1.DeleteOptions{Uid: pending.Metadata.Uid})
	require.Nil(t, err)
	h.Eventually(t, "deleting a Request to remove it from activity", eeharness.IngestionBudget,
		func(ctx context.Context) error { return waitCounts(ctx, map[string]int64{"REVOKED": 1}) })
}
