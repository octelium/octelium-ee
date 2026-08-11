// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"google.golang.org/grpc"
)

const (
	SyncBudget        = 3 * time.Minute
	PropagationBudget = 90 * time.Second
	IngestionBudget   = 3 * time.Minute
)

type H struct {
	*harness.H
}

func Wrap(h *harness.H) *H {
	return &H{H: h}
}

func (h *H) EnterpriseC() enterprisev1.MainServiceClient {
	return enterprisev1.NewMainServiceClient(h.Conn())
}

func (h *H) ClusterC() enterprisev1.ClusterServiceClient {
	return enterprisev1.NewClusterServiceClient(h.Conn())
}

func (h *H) PolicyPortalC() enterprisev1.PolicyPortalServiceClient {
	return enterprisev1.NewPolicyPortalServiceClient(h.Conn())
}

func (h *H) AccessC() accessv1.MainServiceClient {
	return accessv1.NewMainServiceClient(h.Conn())
}

func (h *H) AccessLogC() visibilityv1.AccessLogServiceClient {
	return visibilityv1.NewAccessLogServiceClient(h.Conn())
}

func (h *H) AuditLogC() visibilityv1.AuditLogServiceClient {
	return visibilityv1.NewAuditLogServiceClient(h.Conn())
}

func (h *H) AuthenticationLogC() visibilityv1.AuthenticationLogServiceClient {
	return visibilityv1.NewAuthenticationLogServiceClient(h.Conn())
}

func (h *H) ComponentLogC() visibilityv1.ComponentLogServiceClient {
	return visibilityv1.NewComponentLogServiceClient(h.Conn())
}

func (h *H) MetricsC() visibilityv1.MetricsServiceClient {
	return visibilityv1.NewMetricsServiceClient(h.Conn())
}

func (h *H) VisibilityCoreC() vcorev1.ResourceServiceClient {
	return vcorev1.NewResourceServiceClient(h.Conn())
}

func (h *H) AccessUserC(conn *grpc.ClientConn) accessv1.UserServiceClient {
	return accessv1.NewUserServiceClient(conn)
}

func (h *H) AccessReviewerC(conn *grpc.ClientConn) accessv1.ReviewerServiceClient {
	return accessv1.NewReviewerServiceClient(conn)
}

func (h *H) AccessMainC(conn *grpc.ClientConn) accessv1.MainServiceClient {
	return accessv1.NewMainServiceClient(conn)
}

func (h *H) StateValue(t *testing.T, key string) string {
	t.Helper()

	ret := h.State.Get(key)
	if ret == "" {
		t.Fatalf("The scenario state has no %q entry. Is the scenario providing the fixture?", key)
	}

	return ret
}

func (h *H) Ctx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	if t.Context().Err() == nil {
		return context.WithCancel(t.Context())
	}

	return context.WithTimeout(context.Background(), 60*time.Second)
}
