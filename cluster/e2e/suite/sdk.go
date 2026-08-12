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

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waitIngested(t *testing.T, h *harness.H, what string,
	fn func(ctx context.Context) (int, error)) {
	t.Helper()

	h.Eventually(t, what, eeharness.IngestionBudget, func(ctx context.Context) error {
		n, err := fn(ctx)
		if err != nil {
			return err
		}
		if n < 1 {
			return errNotProvisioned(what)
		}
		return nil
	})
}

func testEnterpriseSDK(t *testing.T, h *harness.H) {
	ctx := t.Context()

	conn := h.Conn()
	coreC := h.CoreC()

	userC := userv1.NewMainServiceClient(conn)
	enterpriseC := enterprisev1.NewMainServiceClient(conn)
	accessLogC := visibilityv1.NewAccessLogServiceClient(conn)
	auditLogC := visibilityv1.NewAuditLogServiceClient(conn)
	authenticationLogC := visibilityv1.NewAuthenticationLogServiceClient(conn)
	visibilityCoreC := vcorev1.NewResourceServiceClient(conn)
	policyPortalC := enterprisev1.NewPolicyPortalServiceClient(conn)

	status, err := userC.GetStatus(ctx, &userv1.GetStatusRequest{})
	require.Nil(t, err)

	meUsr, err := coreC.GetUser(ctx, &metav1.GetOptions{Name: status.User.Metadata.Name})
	require.Nil(t, err)

	meSess, err := coreC.GetSession(ctx, &metav1.GetOptions{Uid: status.Session.Metadata.Uid})
	require.Nil(t, err)

	eeAPISvc, err := coreC.GetService(ctx, &metav1.GetOptions{Name: "enterprise.octelium-api"})
	require.Nil(t, err)

	t.Run("ClusterConfig", func(t *testing.T) {
		_, err := enterpriseC.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		assert.Nil(t, err)
	})

	t.Run("Secret", func(t *testing.T) {
		sec, err := enterpriseC.CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec:     &enterprisev1.Secret_Spec{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: utilrand.GetRandomStringCanonical(32),
				},
			},
		})
		require.Nil(t, err)

		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomStringCanonical(32),
			},
		}

		sec, err = enterpriseC.UpdateSecret(ctx, sec)
		require.Nil(t, err)

		_, err = enterpriseC.DeleteSecret(ctx, &metav1.DeleteOptions{Name: sec.Metadata.Name})
		assert.Nil(t, err)
	})

	t.Run("DefaultResources", func(t *testing.T) {
		_, err := enterpriseC.GetDNSProvider(ctx, &metav1.GetOptions{Name: "default"})
		assert.Nil(t, err)

		_, err = enterpriseC.GetCertificateIssuer(ctx, &metav1.GetOptions{Name: "default"})
		assert.Nil(t, err)

		_, err = enterpriseC.GetSecretStore(ctx, &metav1.GetOptions{Name: "default"})
		assert.Nil(t, err)
	})

	t.Run("AccessLog", func(t *testing.T) {
		waitIngested(t, h, "the access log to carry entries", func(ctx context.Context) (int, error) {
			res, err := accessLogC.ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
			if err != nil {
				return 0, err
			}
			return len(res.Items), nil
		})
	})

	t.Run("AccessLogScopedToUser", func(t *testing.T) {
		var res *visibilityv1.ListAccessLogResponse

		waitIngested(t, h, "the access log to carry the current User",
			func(ctx context.Context) (int, error) {
				cur, err := accessLogC.ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
					UserRef: umetav1.GetObjectReference(meUsr),
				})
				if err != nil {
					return 0, err
				}
				res = cur
				return len(cur.Items), nil
			})

		for _, itm := range res.Items {
			assert.Equal(t, status.User.Metadata.Uid, itm.Entry.Common.UserRef.Uid)
		}
	})

	t.Run("AuditLogScopedToUser", func(t *testing.T) {
		var res *visibilityv1.ListAuditLogResponse

		waitIngested(t, h, "the audit log to carry the current User",
			func(ctx context.Context) (int, error) {
				cur, err := auditLogC.ListAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
					UserRef: umetav1.GetObjectReference(meUsr),
				})
				if err != nil {
					return 0, err
				}
				res = cur
				return len(cur.Items), nil
			})

		for _, itm := range res.Items {
			assert.Equal(t, status.User.Metadata.Uid, itm.Entry.UserRef.Uid)
		}
	})

	t.Run("AuthenticationLogScopedToUser", func(t *testing.T) {
		a := eeharness.Wrap(h).NewActor(t)

		var res *visibilityv1.ListAuthenticationLogResponse

		waitIngested(t, h, "the authentication log to carry the authenticated User",
			func(ctx context.Context) (int, error) {
				cur, err := authenticationLogC.ListAuthenticationLog(ctx,
					&visibilityv1.ListAuthenticationLogRequest{
						UserRef: umetav1.GetObjectReference(a.User),
					})
				if err != nil {
					return 0, err
				}
				res = cur
				return len(cur.Items), nil
			})

		for _, itm := range res.Items {
			assert.Equal(t, a.User.Metadata.Uid, itm.Entry.UserRef.Uid)
		}
	})

	t.Run("PolicyPortalAuthorizedSession", func(t *testing.T) {
		res, err := policyPortalC.IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
			Downstream: &enterprisev1.IsAuthorizedRequest_SessionRef{
				SessionRef: umetav1.GetObjectReference(meSess),
			},
			Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(eeAPISvc),
			},
		})
		require.Nil(t, err)
		assert.True(t, res.IsAuthorized)
	})

	t.Run("PolicyPortalUnauthorizedUser", func(t *testing.T) {
		usr, err := coreC.CreateUser(ctx, &corev1.User{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec:     &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
		})
		require.Nil(t, err)

		res, err := policyPortalC.IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
			Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
				UserRef: umetav1.GetObjectReference(usr),
			},
			Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(eeAPISvc),
			},
		})
		require.Nil(t, err)
		assert.False(t, res.IsAuthorized)
	})

	t.Run("VisibilityMirrorsCore", func(t *testing.T) {
		h.Eventually(t, "the visibility mirror to converge with core",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := visibilityCoreC.ListUser(ctx, &vcorev1.ListUserOptions{})
				if err != nil {
					return err
				}

				usrList, err := coreC.ListUser(ctx, &corev1.ListUserOptions{})
				if err != nil {
					return err
				}

				for _, usr := range usrList.Items {
					if !slices.ContainsFunc(res.Items, func(itm *corev1.User) bool {
						return itm.Metadata.Uid == usr.Metadata.Uid
					}) {
						return errNotProvisioned("the mirrored User " + usr.Metadata.Name)
					}
				}

				if res.ListResponseMeta.TotalCount != usrList.ListResponseMeta.TotalCount {
					return errors.Errorf("the mirror has %d Users, core has %d",
						res.ListResponseMeta.TotalCount, usrList.ListResponseMeta.TotalCount)
				}

				return nil
			})
	})
}
