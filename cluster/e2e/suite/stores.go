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
	"slices"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
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
		_, err := h.AccessLogC().GetAccessLogSummary(ctx,
			&visibilityv1.GetAccessLogSummaryRequest{From: from})
		assert.Nil(t, err)

		_, err = h.AccessLogC().GetAccessLogDataPoint(ctx,
			&visibilityv1.GetAccessLogDataPointRequest{From: from})
		assert.Nil(t, err)

		_, err = h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{From: from})
		assert.Nil(t, err)

		_, err = h.AccessLogC().ListAccessLogTopService(ctx,
			&visibilityv1.ListAccessLogTopServiceRequest{From: from})
		assert.Nil(t, err)
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

func testMetricStoreQueryPath(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	driveTraffic(t, h)

	t.Run("Capabilities", func(t *testing.T) {
		_, err := h.MetricsC().GetMetricsCapabilities(ctx,
			&vmetricsv1.GetMetricsCapabilitiesRequest{})
		assert.Nil(t, err)
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
		h.RestartEnterprise(t, "metricstore")

		h.Eventually(t, "the metricstore to answer after the restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
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
