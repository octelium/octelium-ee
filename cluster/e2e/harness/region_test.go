// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"os/exec"
	"strings"
	"testing"

	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGateway(name, hostname string, ips ...string) *corev1.Gateway {
	return &corev1.Gateway{
		Metadata: &metav1.Metadata{Name: name},
		Status: &corev1.Gateway_Status{
			Hostname:  hostname,
			PublicIPs: ips,
		},
	}
}

func TestRegionGatewayHostsScript(t *testing.T) {
	script, err := regionGatewayHostsScript([]*corev1.Gateway{
		newTestGateway("gw-1", "octelium-gw-1.localhost", "172.30.0.10", "10.0.0.1"),
		newTestGateway("gw-2", "octelium-gw-2.localhost", "172.30.0.11"),
	})
	require.NoError(t, err)

	out, err := exec.Command("bash", "-n", "-c", script).CombinedOutput()
	require.NoError(t, err, "%s", out)

	lines := strings.Split(strings.TrimSpace(script), "\n")
	require.Len(t, lines, 5)
	assert.Equal(t, "set -e", lines[0])
	assert.Equal(t, "sudo sed -i '/"+eescenario.RegionHostsMarker+"$/d' /etc/hosts", lines[1])
	assert.Equal(t, "echo '172.30.0.10 octelium-gw-1.localhost "+eescenario.RegionHostsMarker+
		"' | sudo tee -a /etc/hosts >/dev/null", lines[2])
	assert.Equal(t, "echo '172.30.0.11 octelium-gw-2.localhost "+eescenario.RegionHostsMarker+
		"' | sudo tee -a /etc/hosts >/dev/null", lines[3])
	assert.Equal(t, "grep -F '"+eescenario.RegionHostsMarker+"' /etc/hosts", lines[4])
}

func TestRegionGatewayHostsScriptErrors(t *testing.T) {
	for _, gws := range [][]*corev1.Gateway{
		nil,
		{{Metadata: &metav1.Metadata{Name: "gw"}}},
		{newTestGateway("gw", "", "172.30.0.10")},
		{newTestGateway("gw", "octelium-gw.localhost")},
		{newTestGateway("gw-1", "octelium-gw-1.localhost", "172.30.0.10"), newTestGateway("gw-2", "")},
	} {
		_, err := regionGatewayHostsScript(gws)
		assert.Error(t, err)
	}
}
