// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"testing"

	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"google.golang.org/grpc"
)

const (
	AgentService       = "octelium.api.main.enterprise.v1.AgentService"
	CordiumMainService = "octelium.api.main.cordium.v1.MainService"
)

func (h *H) AgentC(conn *grpc.ClientConn) enterprisev1.AgentServiceClient {
	return enterprisev1.NewAgentServiceClient(conn)
}

func (h *H) CordiumC(conn *grpc.ClientConn) cordiumv1.MainServiceClient {
	return cordiumv1.NewMainServiceClient(conn)
}

func (h *H) IsCordiumInstalled(t *testing.T) bool {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	rgn, err := h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: "default"})
	if err != nil {
		t.Fatalf("Could not get the default Region: %+v", err)
	}

	_, ok := rgn.GetStatus().GetVersionInfoMap()["cordium"]
	return ok
}
