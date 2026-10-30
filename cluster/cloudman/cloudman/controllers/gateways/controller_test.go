// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gwcontroller

import (
	"context"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGatewayDNSControllerLifecycle(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateDNSProvider(t.Context(), &enterprisev1.DNSProvider{Metadata: &metav1.Metadata{Name: "default"}, Spec: &enterprisev1.DNSProvider_Spec{}})
	require.NoError(t, err)
	ctl := NewController(tst.C.OcteliumC)
	resource := &corev1.Gateway{Metadata: &metav1.Metadata{Name: "default"}, Status: &corev1.Gateway_Status{}}
	assert.NoError(t, ctl.OnAdd(t.Context(), resource))
	assert.NoError(t, ctl.OnUpdate(t.Context(), resource, resource))
	assert.NoError(t, ctl.OnDelete(t.Context(), resource))
	updated := proto.Clone(resource).(*corev1.Gateway)
	updated.Status.PublicIPs = []string{"192.0.2.1", "2001:db8::1"}
	updated.Status.Hostname = "gateway.example.com"

	require.NoError(t, ctl.OnUpdate(t.Context(), updated, resource))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.Error(t, doSetDNS(ctx, tst.C.OcteliumC, updated))
}
