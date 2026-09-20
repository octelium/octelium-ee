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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
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
