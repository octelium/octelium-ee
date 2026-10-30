// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster_config

import (
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterConfigurationReconcilesPublicServices(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateDNSProvider(t.Context(), &enterprisev1.DNSProvider{Metadata: &metav1.Metadata{Name: "default"}, Spec: &enterprisev1.DNSProvider_Spec{}})
	require.NoError(t, err)
	ctl := NewController(tst.C.OcteliumC)
	assert.NoError(t, ctl.OnUpdate(t.Context(), &enterprisev1.ClusterConfig{}, &enterprisev1.ClusterConfig{}))
	assert.NoError(t, ctl.setAllServicesCNAME(t.Context()))
}
