// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func stepNames(steps []scenario.Step) []string {
	var ret []string
	for _, step := range steps {
		ret = append(ret, step.Name)
	}
	return ret
}

func TestScenarioMultiRegion(t *testing.T) {
	for _, val := range []string{"", "false", "true"} {
		t.Run("SPIFFE="+val, func(t *testing.T) {
			t.Setenv(EnvSPIFFE, val)
			t.Setenv("OCTELIUM_SPIFFE_TRUST_DOMAIN", "")
			t.Setenv(EnvCoreVersion, "")

			s, err := scenario.Get(MultiRegionScenario)
			require.NoError(t, err)

			assert.Equal(t, MultiRegionScenario, s.ID)
			assert.True(t, s.Caps.Has(CapMultiRegion))
			assert.False(t, s.Caps.Has(scenario.CapSPIFFE))
			assert.False(t, s.Caps.Has(scenario.CapUpgrade))
			assert.False(t, s.Caps.Has(CapVault))
			assert.False(t, s.Install.EnableSPIFFECSI)
			assert.False(t, s.Install.OTelCollector)
			assert.Equal(t, Components, s.Components)
			assert.Equal(t, 120*time.Minute, s.Budget)

			p, ok := s.Provisioner.(*multiRegionProvisioner)
			require.True(t, ok)
			assert.Equal(t, "k3s", p.Name())
			assert.Equal(t, "/etc/rancher/k3s/k3s.yaml", p.KubeconfigPath())
			assert.Equal(t, scenario.CNIFlannel, s.CNI)

			pg := s.Storage.Postgres
			assert.Equal(t, noopChartDir(), pg.Chart)
			assert.Empty(t, pg.ChartVersion)
			assert.Empty(t, pg.ImageRepository)
			assert.False(t, pg.AllowInsecure)
			assert.Empty(t, pg.PVCName)
			assert.Equal(t, sharedStorageAddr, pg.Host)
			assert.Equal(t, int32(sharedPostgresPort), pg.Port)
			assert.Equal(t, "octelium", pg.Username)
			assert.Equal(t, "octelium", pg.Database)

			redis := s.Storage.Redis
			assert.Equal(t, noopChartDir(), redis.Chart)
			assert.Empty(t, redis.ChartVersion)
			assert.Empty(t, redis.ImageRepository)
			assert.False(t, redis.AllowInsecure)
			assert.Equal(t, sharedStorageAddr, redis.Host)
			assert.Equal(t, int32(sharedRedisPort), redis.Port)

			assert.Equal(t, []string{
				"multi-region/noop-storage-chart",
				"multi-region/network",
				"multi-region/region-node",
				"multi-region/region-ready",
				"multi-region/region-labels",
			}, stepNames(s.Hooks.PostProvision))
			assert.Equal(t, []string{
				"multi-region/shared-storage",
				"multi-region/region-multus",
				"multi-region/region-dns",
				"multi-region/connectivity",
			}, stepNames(s.Hooks.PostPrepare))
			require.NotEmpty(t, s.Hooks.PostInstall)
			assert.Equal(t, "octeliumee/install-package", s.Hooks.PostInstall[0].Name)

			for _, step := range slices.Concat(s.Hooks.PostProvision, s.Hooks.PostPrepare) {
				assert.NotNil(t, step.Run, step.Name)
			}
		})
	}
}

func TestScenarioMultiRegionDoesNotChangeTheOthers(t *testing.T) {
	t.Setenv(EnvSPIFFE, "")
	t.Setenv(EnvCoreVersion, "")

	for _, id := range []string{DefaultScenario, CoreMainScenario, SPIREScenario, FullScenario} {
		t.Run(id, func(t *testing.T) {
			s, err := scenario.Get(id)
			require.NoError(t, err)

			assert.False(t, s.Caps.Has(CapMultiRegion))
			_, ok := s.Provisioner.(*multiRegionProvisioner)
			assert.False(t, ok)
			assert.NotEqual(t, sharedStorageAddr, s.Storage.Postgres.Host)
			assert.NotEqual(t, noopChartDir(), s.Storage.Postgres.Chart)
			assert.NotEqual(t, noopChartDir(), s.Storage.Redis.Chart)
		})
	}
}

func TestStepNoopStorageChart(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	require.NoError(t, stepNoopStorageChart(t.Context(), nil))

	out, err := os.ReadFile(filepath.Join(noopChartDir(), "Chart.yaml"))
	require.NoError(t, err)
	assert.Equal(t, noopChartYAML, string(out))

	entries, err := os.ReadDir(noopChartDir())
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	require.NoError(t, stepNoopStorageChart(t.Context(), nil))
}

func newTestKubeconfig(t *testing.T, ca []byte) []byte {
	t.Helper()

	cfg := clientcmdapi.NewConfig()
	cfg.Clusters["default"] = &clientcmdapi.Cluster{
		Server:                   "https://127.0.0.1:6443",
		CertificateAuthorityData: ca,
	}
	cfg.AuthInfos["default"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: []byte("crt"),
		ClientKeyData:         []byte("key"),
	}
	cfg.Contexts["default"] = &clientcmdapi.Context{
		Cluster:  "default",
		AuthInfo: "default",
	}
	cfg.CurrentContext = "default"

	ret, err := clientcmd.Write(*cfg)
	require.NoError(t, err)

	return ret
}

func TestGetRegionKubeconfig(t *testing.T) {
	assert.Equal(t, "https://172.30.0.1:16443", regionAPIServer())

	ret, err := getRegionKubeconfig(newTestKubeconfig(t, []byte("ca")), regionAPIServer())
	require.NoError(t, err)

	cfg, err := clientcmd.Load(ret)
	require.NoError(t, err)
	require.Contains(t, cfg.Clusters, "default")
	assert.Equal(t, regionAPIServer(), cfg.Clusters["default"].Server)
	assert.Equal(t, []byte("ca"), cfg.Clusters["default"].CertificateAuthorityData)
	assert.Equal(t, []byte("crt"), cfg.AuthInfos["default"].ClientCertificateData)
	assert.Equal(t, []byte("key"), cfg.AuthInfos["default"].ClientKeyData)
	assert.Empty(t, cfg.AuthInfos["default"].ClientCertificate)
	assert.Empty(t, cfg.AuthInfos["default"].ClientKey)
	assert.Equal(t, "default", cfg.CurrentContext)

	_, err = getRegionKubeconfig(newTestKubeconfig(t, nil), regionAPIServer())
	assert.Error(t, err)

	_, err = getRegionKubeconfig([]byte("apiVersion: v1\nkind: Config\n"), regionAPIServer())
	assert.Error(t, err)

	_, err = getRegionKubeconfig([]byte("{"), regionAPIServer())
	assert.Error(t, err)
}

func TestRegionNetworkDoesNotOverlap(t *testing.T) {
	subnet := netip.MustParsePrefix(regionNetworkSubnet)
	ipRange := netip.MustParsePrefix(regionNetworkIPRange)

	for _, addr := range []string{regionNetworkGateway, regionNodeIP} {
		ip := netip.MustParseAddr(addr)
		assert.True(t, subnet.Contains(ip), addr)
		assert.False(t, ipRange.Contains(ip), addr)
	}
	assert.True(t, subnet.Contains(ipRange.Addr()))
	assert.Equal(t, regionNetworkGateway, sharedStorageAddr)

	assert.True(t, netip.MustParsePrefix(regionServiceCIDR).Contains(netip.MustParseAddr(regionClusterDNS)))

	used := []string{
		regionNetworkSubnet,
		regionClusterCIDR,
		regionServiceCIDR,
		"10.42.0.0/16",
		"10.43.0.0/16",
		"172.17.0.0/16",
		"172.29.0.0/16",
	}

	for i, a := range used {
		for _, b := range used[i+1:] {
			assert.False(t, netip.MustParsePrefix(a).Overlaps(netip.MustParsePrefix(b)), "%s and %s", a, b)
		}
	}
}

func getArg(args []string, name string) []string {
	var ret []string
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			ret = append(ret, args[i+1])
		}
	}
	return ret
}

func TestRegionServerArgs(t *testing.T) {
	args := regionServerArgs()

	assert.Equal(t, "server", args[0])
	assert.Equal(t, []string{regionNode}, getArg(args, "--node-name"))
	assert.Equal(t, []string{"traefik", "metrics-server"}, getArg(args, "--disable"))
	assert.Equal(t, []string{regionNetworkGateway, regionNodeIP}, getArg(args, "--tls-san"))
	assert.Equal(t, []string{regionClusterCIDR}, getArg(args, "--cluster-cidr"))
	assert.Equal(t, []string{regionServiceCIDR}, getArg(args, "--service-cidr"))
	assert.Equal(t, []string{regionClusterDNS}, getArg(args, "--cluster-dns"))
	assert.Equal(t, []string{"/output/kubeconfig.yaml"}, getArg(args, "--write-kubeconfig"))
	assert.Contains(t, args, "--kube-proxy-arg=conntrack-max-per-core=0")

	for _, arg := range regionKubeletArgs {
		assert.Contains(t, args, arg)
	}
}

func checkBashSyntax(t *testing.T, script string) {
	t.Helper()

	out, err := exec.Command("bash", "-n", "-c", script).CombinedOutput()
	require.NoError(t, err, "%s\n%s", out, script)
}

func TestRegionNodeScript(t *testing.T) {
	script := regionNodeScript("/mnt/octelium-e2e-regions", "/var/lib/rancher/k3s/data/cni/")
	checkBashSyntax(t, script)

	for _, want := range []string{
		fmt.Sprintf("--network %s --ip %s", regionNetwork, regionNodeIP),
		fmt.Sprintf("--publish %s:%d:6443", regionNetworkGateway, regionAPIPort),
		"-v '/mnt/octelium-e2e-regions'/data:/var/lib/rancher/k3s:rshared",
		"-v '/mnt/octelium-e2e-regions'/output:/output",
		"--privileged",
		"'--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%'",
		fmt.Sprintf("'--cluster-cidr' '%s'", regionClusterCIDR),
	} {
		assert.Contains(t, script, want)
	}

	entrypoint := regionEntrypointScript("/var/lib/rancher/k3s/data/cni/", "/bin")
	checkBashSyntax(t, entrypoint)
	assert.Contains(t, entrypoint, "BIN_DIR='/bin'")
	assert.Contains(t, entrypoint, "CNI_BIN_DIR='/var/lib/rancher/k3s/data/cni/'")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(entrypoint), `exec "$BIN_DIR/k3s" "$@"`))
}

func TestSharedStorageScript(t *testing.T) {
	script := sharedStorageScript("octelium", "octelium", `p'g$pass`, `re"dis pass`)
	checkBashSyntax(t, script)

	for _, want := range []string{
		"--network host",
		fmt.Sprintf("-e PGPORT=%d", sharedPostgresPort),
		`-e POSTGRES_PASSWORD='p'\''g$pass'`,
		fmt.Sprintf("-c listen_addresses=%s", sharedStorageAddr),
		fmt.Sprintf("--bind %s --port %d --requirepass 're\"dis pass'", sharedStorageAddr, sharedRedisPort),
		sharedPostgresImage,
		sharedRedisImage,
	} {
		assert.Contains(t, script, want)
	}
}

func TestRegionCoreDNSServer(t *testing.T) {
	ret := regionCoreDNSServer("localhost")

	assert.True(t, strings.HasPrefix(ret, "localhost:53 {\n"))
	assert.Contains(t, ret, "template IN A localhost {")
	assert.Contains(t, ret, `answer "{{ .Name }} 60 IN A 172.30.0.1"`)
	assert.Contains(t, ret, "template IN AAAA localhost {")

	stripped := strings.ReplaceAll(ret, "{{ .Name }}", "")
	assert.Equal(t, 3, strings.Count(stripped, "{"))
	assert.Equal(t, 3, strings.Count(stripped, "}"))
}

func TestRegionConnectivityScripts(t *testing.T) {
	for _, checks := range []string{regionConnectivityChecks("localhost"), defaultConnectivityChecks()} {
		checkBashSyntax(t, checks)
		checkBashSyntax(t, connectivityScript("octelium-e2e-connectivity", checks))
	}

	assert.Contains(t, regionConnectivityChecks("localhost"), "nslookup collector.octelium.localhost")
	assert.Contains(t, regionConnectivityChecks("localhost"),
		fmt.Sprintf("nc -z -w 5 %s %d", sharedStorageAddr, sharedPostgresPort))
	assert.Contains(t, defaultConnectivityChecks(),
		fmt.Sprintf("nc -z -w 5 %s %d", regionNetworkGateway, regionAPIPort))
}

func TestRegionTeardownScript(t *testing.T) {
	script := regionTeardownScript()
	checkBashSyntax(t, script)

	for _, want := range []string{regionNode, sharedPostgres, sharedRedis, regionNetwork, RegionHostsMarker} {
		assert.Contains(t, script, want)
	}
}
