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
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingDNSSetter struct{ calls int }

func (s *recordingDNSSetter) SetDNSProvider(context.Context) error { s.calls++; return nil }
func TestDNSProviderControllerRefreshesChangedSpec(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateDNSProvider(t.Context(), &enterprisev1.DNSProvider{Metadata: &metav1.Metadata{Name: "default"}, Spec: &enterprisev1.DNSProvider_Spec{}})
	require.NoError(t, err)
	region, err := tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	services, err := tst.C.OcteliumC.CoreC().ListService(t.Context(), &rmetav1.ListOptions{})
	require.NoError(t, err)
	for _, service := range services.Items {
		if service.Status == nil {
			service.Status = &corev1.Service_Status{}
		}
		service.Status.RegionRef = umetav1.GetObjectReference(region)
		_, err = tst.C.OcteliumC.CoreC().UpdateService(t.Context(), service)
		require.NoError(t, err)
	}
	setter := &recordingDNSSetter{}
	ctl := NewController(tst.C.OcteliumC, setter)
	old := &enterprisev1.DNSProvider{Spec: &enterprisev1.DNSProvider_Spec{}}
	assert.NoError(t, ctl.OnAdd(t.Context(), old))
	assert.NoError(t, ctl.OnUpdate(t.Context(), old, old))
	assert.Zero(t, setter.calls)
	cur := &enterprisev1.DNSProvider{Spec: &enterprisev1.DNSProvider_Spec{Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{}}}
	assert.NoError(t, ctl.OnUpdate(t.Context(), cur, old))
	assert.Equal(t, 1, setter.calls)
	assert.NoError(t, ctl.OnDelete(t.Context(), cur))
}
