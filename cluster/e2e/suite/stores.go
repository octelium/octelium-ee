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
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
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

	driveTraffic(t, h)

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
