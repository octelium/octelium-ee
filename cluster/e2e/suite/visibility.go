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
	"net/http"
	"slices"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/authv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	allowedRequests = 5
	deniedRequests  = 3
)

func testAccessLogAggregations(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	svc := h.NewPublicService(t, "default")

	allowed := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})
	refused := h.CreateWorkloadUser(t, nil)

	allowedProbe := h.Probe(t, allowed, svc)
	refusedProbe := h.Probe(t, refused, svc)

	allowedProbe.MustBeAllowed(t)
	refusedProbe.MustBeDenied(t)

	from := pbutils.Timestamp(time.Now().Add(-time.Minute))

	for range allowedRequests {
		allowedProbe.MustBeAllowed(t)
	}
	for range deniedRequests {
		refusedProbe.MustBeDenied(t)
	}

	svcRef := umetav1.GetObjectReference(svc)

	waitAccessLogSummary(t, h, svcRef, from)

	t.Run("TheSummarySplitsAllowedAndDenied", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().GetAccessLogSummary(ctx,
			&visibilityv1.GetAccessLogSummaryRequest{ServiceRef: svcRef, From: from})
		require.Nil(t, err)

		assert.Equal(t, res.TotalNumber, res.TotalAllowed+res.TotalDenied)
		assert.True(t, res.TotalAllowed >= allowedRequests)
		assert.True(t, res.TotalDenied >= deniedRequests)
		assert.True(t, res.TotalUser >= 2)
		assert.Equal(t, uint64(1), res.TotalService)
	})

	t.Run("TheUsersAreRankedForTheService", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: svcRef,
				From:       from,
				Limit:      100,
			})
		require.Nil(t, err)

		assert.True(t, topUserCount(res.Items, allowed) >= allowedRequests)
		assert.True(t, topUserCount(res.Items, refused) >= deniedRequests)
	})

	t.Run("TheStatusNarrowsTheRanking", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		denied, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: svcRef,
				From:       from,
				Status:     corev1.AccessLog_Entry_Common_DENIED,
				Limit:      100,
			})
		require.Nil(t, err)
		assert.True(t, topUserCount(denied.Items, refused) >= deniedRequests)

		granted, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: svcRef,
				From:       from,
				Status:     corev1.AccessLog_Entry_Common_ALLOWED,
				Limit:      100,
			})
		require.Nil(t, err)
		assert.True(t, topUserCount(granted.Items, allowed) >= allowedRequests)
		assert.Zero(t, topUserCount(granted.Items, refused))
	})

	t.Run("TheLimitIsHonored", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: svcRef,
				From:       from,
				Limit:      1,
			})
		require.Nil(t, err)
		assert.Len(t, res.Items, 1)
	})

	t.Run("TheServiceIsRankedForTheUser", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().ListAccessLogTopService(ctx,
			&visibilityv1.ListAccessLogTopServiceRequest{
				UserRef: umetav1.GetObjectReference(allowed),
				From:    from,
				Limit:   100,
			})
		require.Nil(t, err)

		idx := slices.IndexFunc(res.Items,
			func(itm *visibilityv1.ListAccessLogTopServiceResponse_Item) bool {
				return itm.Service.Metadata.Uid == svc.Metadata.Uid
			})
		require.True(t, idx >= 0, "the Service is not ranked for the User")
		assert.True(t, res.Items[idx].Count >= allowedRequests)
	})

	t.Run("TheSessionsAreRanked", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().ListAccessLogTopSession(ctx,
			&visibilityv1.ListAccessLogTopSessionRequest{
				ServiceRef: svcRef,
				From:       from,
				Limit:      100,
			})
		require.Nil(t, err)
		require.NotEmpty(t, res.Items)

		for _, itm := range res.Items {
			assert.NotNil(t, itm.Session)
			assert.True(t, itm.Count > 0)
		}
	})

	t.Run("TheDenialsCarryTheirReason", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AccessLogC().ListAccessLogTopDenyReason(ctx,
			&visibilityv1.ListAccessLogTopDenyReasonRequest{
				UserRef: umetav1.GetObjectReference(refused),
				From:    from,
				Limit:   100,
			})
		require.Nil(t, err)
		require.NotEmpty(t, res.Items)

		var total uint64
		for _, itm := range res.Items {
			assert.NotEqual(t,
				corev1.AccessLog_Entry_Common_Reason_TYPE_UNKNOWN_REASON, itm.Reason)
			total += itm.Count
		}
		assert.True(t, total >= deniedRequests)
	})

	t.Run("AnUnrelatedServiceRanksNothing", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		other := h.NewPublicService(t, "default")

		res, err := h.AccessLogC().ListAccessLogTopUser(ctx,
			&visibilityv1.ListAccessLogTopUserRequest{
				ServiceRef: umetav1.GetObjectReference(other),
				From:       from,
				Limit:      100,
			})
		require.Nil(t, err)
		assert.Zero(t, topUserCount(res.Items, allowed))
		assert.Zero(t, topUserCount(res.Items, refused))
	})
}

func waitAccessLogSummary(t *testing.T, h *eeharness.H,
	svcRef *metav1.ObjectReference, from *timestamppb.Timestamp) {
	t.Helper()

	h.Eventually(t, "the access log summary to carry the generated traffic",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.AccessLogC().GetAccessLogSummary(ctx,
				&visibilityv1.GetAccessLogSummaryRequest{ServiceRef: svcRef, From: from})
			if err != nil {
				return err
			}
			if res.TotalAllowed < allowedRequests {
				return errors.Errorf("the summary counts %d allowed, want at least %d",
					res.TotalAllowed, allowedRequests)
			}
			if res.TotalDenied < deniedRequests {
				return errors.Errorf("the summary counts %d denied, want at least %d",
					res.TotalDenied, deniedRequests)
			}
			return nil
		})
}

func topUserCount(items []*visibilityv1.ListAccessLogTopUserResponse_Item,
	usr *corev1.User) int32 {
	idx := slices.IndexFunc(items,
		func(itm *visibilityv1.ListAccessLogTopUserResponse_Item) bool {
			return itm.User.GetMetadata().GetUid() == usr.Metadata.Uid
		})
	if idx < 0 {
		return 0
	}

	return items[idx].Count
}

func testAuthenticationLogFailure(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	from := pbutils.Timestamp(time.Now().Add(-time.Minute))

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})

	cred := h.CreateCredential(t, harness.CredentialOpts{
		User:        usr.Metadata.Name,
		Type:        corev1.Credential_Spec_AUTH_TOKEN,
		SessionType: corev1.Session_Status_CLIENTLESS,
	})

	tkn := h.CredentialToken(t, cred).GetAuthenticationToken()
	require.NotNil(t, tkn)

	{
		ctx, cancel := h.Ctx(t)
		defer cancel()

		cur, err := h.CoreC().GetCredential(ctx, &metav1.GetOptions{Uid: cred.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.IsDisabled = true
		_, err = h.CoreC().UpdateCredential(ctx, cur)
		require.Nil(t, err)
	}

	authC := h.AuthC(t)

	var entry *enterprisev1.AuthenticationLog

	h.Eventually(t, "the authentication log to carry the failed attempt",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			if _, err := authC.AuthenticateWithAuthenticationToken(ctx,
				&authv1.AuthenticateWithAuthenticationTokenRequest{
					AuthenticationToken: tkn.AuthenticationToken,
				}); err == nil {
				return errors.Errorf("the disabled Credential authenticated")
			}

			res, err := h.AuthenticationLogC().ListAuthenticationLog(ctx,
				&visibilityv1.ListAuthenticationLogRequest{
					CredentialRef: umetav1.GetObjectReference(cred),
					Status:        enterprisev1.AuthenticationLog_Entry_FAILURE,
					From:          from,
				})
			if err != nil {
				return err
			}
			if len(res.Items) < 1 {
				return errors.Errorf("the authentication log has no failed attempt for the Credential")
			}

			entry = res.Items[0]
			return nil
		})

	t.Run("TheEntryDescribesTheFailure", func(t *testing.T) {
		require.NotNil(t, entry.Entry)
		require.NotNil(t, entry.Entry.Failure)
		require.NotNil(t, entry.Entry.Authentication)

		assert.Equal(t, enterprisev1.AuthenticationLog_Entry_FAILURE, entry.Entry.Status)
		assert.Equal(t, enterprisev1.AuthenticationLog_Entry_Failure_REVOKED_CREDENTIAL,
			entry.Entry.Failure.Reason)
		assert.Equal(t, corev1.Session_Status_Authentication_Info_CREDENTIAL,
			entry.Entry.Authentication.Info.GetType())
		assert.Equal(t, cred.Metadata.Uid,
			entry.Entry.Authentication.Info.GetCredential().GetCredentialRef().GetUid())
		assert.Equal(t, usr.Metadata.Uid, entry.Entry.UserRef.GetUid())
		assert.Nil(t, entry.Entry.SessionRef)
		assert.NotEmpty(t, entry.Metadata.Id)
	})

	t.Run("TheSummaryCountsTheFailure", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().GetAuthenticationLogSummary(ctx,
			&visibilityv1.GetAuthenticationLogSummaryRequest{
				CredentialRef: umetav1.GetObjectReference(cred),
				From:          from,
			})
		require.Nil(t, err)

		assert.True(t, res.TotalFailure > 0)
		assert.Equal(t, uint64(0), res.TotalSuccess)
		assert.Equal(t, res.TotalNumber, res.TotalFailure)
		assert.True(t, res.TotalByFailureReason[enterprisev1.AuthenticationLog_Entry_Failure_REVOKED_CREDENTIAL.String()] > 0)
	})

	t.Run("TheSuccessfulLogsExcludeTheFailure", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().ListAuthenticationLog(ctx,
			&visibilityv1.ListAuthenticationLogRequest{
				CredentialRef: umetav1.GetObjectReference(cred),
				Status:        enterprisev1.AuthenticationLog_Entry_SUCCESS,
				From:          from,
			})
		require.Nil(t, err)
		assert.Empty(t, res.Items)
	})
}

func testAuthenticationLogTrail(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	from := pbutils.Timestamp(time.Now().Add(-time.Minute))

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})

	cred := h.CreateCredential(t, harness.CredentialOpts{
		User:        usr.Metadata.Name,
		Type:        corev1.Credential_Spec_ACCESS_TOKEN,
		SessionType: corev1.Session_Status_CLIENTLESS,
	})

	tkn := h.CredentialToken(t, cred)
	require.NotNil(t, tkn.GetAccessToken())

	h.WaitGetStatus(t, h.HTTPPublicToken("demo-nginx", tkn.GetAccessToken().AccessToken),
		"/", http.StatusOK)

	var entry *enterprisev1.AuthenticationLog

	h.Eventually(t, "the authentication log to carry the new Session",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			res, err := h.AuthenticationLogC().ListAuthenticationLog(ctx,
				&visibilityv1.ListAuthenticationLogRequest{
					UserRef: umetav1.GetObjectReference(usr),
					From:    from,
				})
			if err != nil {
				return err
			}
			if len(res.Items) < 1 {
				return errors.Errorf("the authentication log has no entry for the User")
			}

			entry = res.Items[0]
			return nil
		})

	t.Run("TheEntryIdentifiesTheSessionAndTheUser", func(t *testing.T) {
		require.NotNil(t, entry.Entry)
		require.NotNil(t, entry.Entry.UserRef)
		require.NotNil(t, entry.Entry.SessionRef)
		require.NotNil(t, entry.Entry.Authentication)

		assert.Equal(t, usr.Metadata.Uid, entry.Entry.UserRef.Uid)
		assert.NotEmpty(t, entry.Metadata.Id)
		assert.NotNil(t, entry.Entry.Authentication.SetAt)
	})

	t.Run("TheReferencedSessionExistsInTheCore", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		sess, err := h.CoreC().GetSession(ctx,
			&metav1.GetOptions{Uid: entry.Entry.SessionRef.Uid})
		require.Nil(t, err)
		assert.Equal(t, usr.Metadata.Uid, sess.Status.UserRef.Uid)
	})

	t.Run("TheCredentialScopesTheLog", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().ListAuthenticationLog(ctx,
			&visibilityv1.ListAuthenticationLogRequest{
				CredentialRef: umetav1.GetObjectReference(cred),
				From:          from,
			})
		require.Nil(t, err)
		require.NotEmpty(t, res.Items)

		for _, itm := range res.Items {
			assert.Equal(t, usr.Metadata.Uid, itm.Entry.UserRef.Uid)
		}
	})

	t.Run("TheUserIsRanked", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().ListAuthenticationLogTopUser(ctx,
			&visibilityv1.ListAuthenticationLogTopUserRequest{From: from, Limit: 100})
		require.Nil(t, err)

		idx := slices.IndexFunc(res.Items,
			func(itm *visibilityv1.ListAuthenticationLogTopUserResponse_Item) bool {
				return itm.User.GetMetadata().GetUid() == usr.Metadata.Uid
			})
		require.True(t, idx >= 0, "the authenticated User is not ranked")
		assert.True(t, res.Items[idx].Count > 0)
	})

	t.Run("TheCredentialIsRanked", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().ListAuthenticationLogTopCredential(ctx,
			&visibilityv1.ListAuthenticationLogTopCredentialRequest{From: from, Limit: 100})
		require.Nil(t, err)

		assert.True(t, slices.ContainsFunc(res.Items,
			func(itm *visibilityv1.ListAuthenticationLogTopCredentialResponse_Item) bool {
				return itm.Credential.GetMetadata().GetUid() == cred.Metadata.Uid
			}), "the Credential is not ranked")
	})

	t.Run("TheSummaryAgreesWithTheEntries", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		res, err := h.AuthenticationLogC().GetAuthenticationLogSummary(ctx,
			&visibilityv1.GetAuthenticationLogSummaryRequest{From: from})
		require.Nil(t, err)

		assert.True(t, res.TotalNumber > 0)
		assert.True(t, res.TotalUser > 0)
		assert.True(t, res.TotalSession > 0)
		assert.True(t, res.TotalSession <= res.TotalNumber)
		assert.True(t, res.TotalUser <= res.TotalNumber)
	})
}

func testLogStoreWindowPagination(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{InlinePolicies: harness.InlineAllowAny("allow")})
	svc := h.NewPublicService(t, "default")
	probe := h.Probe(t, usr, svc)
	from := pbutils.Timestamp(time.Now().Add(-time.Minute))
	for range 12 {
		probe.MustBeAllowed(t)
	}
	to := pbutils.Timestamp(time.Now().Add(time.Minute))
	h.Eventually(t, "the windowed log pages to match the complete scoped list", eeharness.IngestionBudget,
		func(ctx context.Context) error {
			all, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
				UserRef: umetav1.GetObjectReference(usr), ServiceRef: umetav1.GetObjectReference(svc),
				From: from, To: to, Common: &vmetav1.CommonListOptions{ItemsPerPage: 1000},
			})
			if err != nil {
				return err
			}
			if len(all.Items) < 12 || uint32(len(all.Items)) != all.GetListResponseMeta().GetTotalCount() {
				return errors.Errorf("the complete window has not converged")
			}
			want := make(map[string]int)
			for _, itm := range all.Items {
				want[itm.Metadata.Id]++
			}
			seen := make(map[string]int)
			count := 0
			for page := uint32(0); ; page++ {
				res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
					UserRef: umetav1.GetObjectReference(usr), ServiceRef: umetav1.GetObjectReference(svc),
					From: from, To: to, Common: &vmetav1.CommonListOptions{Page: page, ItemsPerPage: 3},
				})
				if err != nil {
					return err
				}
				meta := res.GetListResponseMeta()
				if meta.TotalCount != uint32(len(all.Items)) || meta.Page != page || meta.ItemsPerPage != 3 || len(res.Items) > 3 {
					return errors.Errorf("page %d has inconsistent pagination metadata", page)
				}
				for _, itm := range res.Items {
					if seen[itm.Metadata.Id] >= want[itm.Metadata.Id] || itm.Entry.Common.UserRef.Uid != usr.Metadata.Uid || itm.Entry.Common.ServiceRef.Uid != svc.Metadata.Uid {
						return errors.Errorf("page %d has a duplicate or unexpected log", page)
					}
					seen[itm.Metadata.Id]++
					count++
				}
				if !meta.HasMore {
					break
				}
				if len(res.Items) != 3 || page >= uint32(len(all.Items)) {
					return errors.Errorf("pagination did not advance")
				}
			}
			if count != len(all.Items) {
				return errors.Errorf("pagination returned %d of %d logs", count, len(all.Items))
			}
			return nil
		})
}

func testLogStoreWindowedSeries(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	svc := h.NewPublicService(t, "default")
	allowed := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{InlinePolicies: harness.InlineAllowAny("allow")})
	denied := h.CreateWorkloadUser(t, nil)
	allowedProbe := h.Probe(t, allowed, svc)
	deniedProbe := h.Probe(t, denied, svc)
	from := pbutils.Timestamp(time.Now().Add(-5 * time.Minute).Truncate(time.Minute))
	for range 3 {
		allowedProbe.MustBeAllowed(t)
		deniedProbe.MustBeDenied(t)
	}
	to := pbutils.Timestamp(time.Now().Add(5 * time.Minute).Truncate(time.Minute))
	previousFrom := pbutils.Timestamp(from.AsTime().Add(-24 * time.Hour))
	previousTo := pbutils.Timestamp(previousFrom.AsTime().Add(time.Hour))
	h.Eventually(t, "windowed totals, status series and ranking remainder to agree", eeharness.IngestionBudget,
		func(ctx context.Context) error {
			ref := umetav1.GetObjectReference(svc)
			summary, err := h.AccessLogC().GetAccessLogSummary(ctx, &visibilityv1.GetAccessLogSummaryRequest{
				ServiceRef: ref, From: from, To: to, CompareFrom: previousFrom, CompareTo: previousTo,
			})
			if err != nil {
				return err
			}
			if summary.TotalAllowed < 3 || summary.TotalDenied < 3 || summary.Previous == nil || summary.Previous.TotalNumber != 0 {
				return errors.Errorf("the summary or its comparison window has not converged")
			}
			points, err := h.AccessLogC().GetAccessLogDataPoint(ctx, &visibilityv1.GetAccessLogDataPointRequest{
				ServiceRef: ref, From: from, To: to, Interval: eeharness.Minutes(1), GroupBy: visibilityv1.GetAccessLogDataPointRequest_STATUS,
			})
			if err != nil {
				return err
			}
			var total int64
			for i, point := range points.Datapoints {
				total += point.Count
				if i > 0 && point.Timestamp.AsTime().Sub(points.Datapoints[i-1].Timestamp.AsTime()) != time.Minute {
					return errors.Errorf("the buckets are not continuous minutes")
				}
			}
			if uint64(total) != summary.TotalNumber || len(points.Series) != 2 || len(points.Datapoints) < 2 {
				return errors.Errorf("the series and summary totals differ")
			}
			if points.Datapoints[0].Count != 0 || points.Datapoints[len(points.Datapoints)-1].Count != 0 {
				return errors.Errorf("empty edge buckets were not zero filled")
			}
			for _, series := range points.Series {
				want := summary.TotalAllowed
				if series.Key == "DENIED" {
					want = summary.TotalDenied
				} else if series.Key != "ALLOWED" {
					return errors.Errorf("unexpected status series %q", series.Key)
				}
				var count int64
				for _, point := range series.Datapoints {
					count += point.Count
				}
				if uint64(series.Total) != want || count != series.Total || len(series.Datapoints) != len(points.Datapoints) {
					return errors.Errorf("the %s series is inconsistent", series.Key)
				}
			}
			top, err := h.AccessLogC().ListAccessLogTopUser(ctx, &visibilityv1.ListAccessLogTopUserRequest{ServiceRef: ref, From: from, To: to, Limit: 1})
			if err != nil {
				return err
			}
			if len(top.Items) != 1 || top.TotalCount != 2 || uint64(top.Items[0].Count)+top.TotalOther != summary.TotalNumber {
				return errors.Errorf("the limited ranking does not account for all requests")
			}
			return nil
		})
}

func testLogStoreAuditResourceTrail(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	from := pbutils.Timestamp(time.Now().Add(-time.Minute))
	usr := h.CreateWorkloadUser(t, nil)
	ref := umetav1.GetObjectReference(usr)
	wait := func(method string) {
		t.Helper()
		h.Eventually(t, "the audit log to retain "+method+" for the User", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				res, err := h.AuditLogC().ListAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
					ResourceRef: ref, From: from, Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
				})
				if err != nil {
					return err
				}
				for _, itm := range res.Items {
					if itm.Entry.ResourceRef.GetUid() != usr.Metadata.Uid {
						return errors.Errorf("a scoped audit entry belongs to another resource")
					}
					if itm.Entry.Method == method && itm.Metadata.Id != "" && itm.Entry.SessionRef.GetUid() != "" {
						return nil
					}
				}
				return errors.Errorf("the %s audit entry is missing", method)
			})
	}
	wait("CreateUser")
	usr.Metadata.DisplayName = h.Name()
	usr = h.UpdateUser(t, usr)
	wait("UpdateUser")
	_, err := h.CoreC().DeleteUser(t.Context(), &metav1.DeleteOptions{Uid: usr.Metadata.Uid})
	require.Nil(t, err)
	wait("DeleteUser")
	res, err := h.AuditLogC().ListAuditLog(t.Context(), &visibilityv1.ListAuditLogRequest{
		ResourceRef: ref, From: from, Common: &vmetav1.CommonListOptions{ItemsPerPage: 100},
	})
	require.Nil(t, err)
	var methods []string
	for _, itm := range res.Items {
		methods = append(methods, itm.Entry.Method)
	}
	assert.Contains(t, methods, "CreateUser")
	assert.Contains(t, methods, "UpdateUser")
	assert.Contains(t, methods, "DeleteUser")
}
