// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const MultiRegionScenario = "k3s-flannel-ee-multi-region"

const CapMultiRegion scenario.Capability = "multi-region"

const (
	StateRegionName       = "regionName"
	StateRegionKubeconfig = "regionKubeconfig"
	StateRegionNodeIP     = "regionNodeIP"
	StateRegionNode       = "regionNode"
)

const RegionHostsMarker = "# octelium-e2e-region"

const (
	regionName = "region-1"
	regionNode = "octelium-e2e-region-1"

	regionNetwork        = "octelium-e2e-regions"
	regionNetworkSubnet  = "172.30.0.0/24"
	regionNetworkIPRange = "172.30.0.128/25"
	regionNetworkGateway = "172.30.0.1"
	regionNodeIP         = "172.30.0.10"

	regionAPIPort     = 16443
	regionClusterCIDR = "10.52.0.0/16"
	regionServiceCIDR = "10.53.0.0/16"
	regionClusterDNS  = "10.53.0.10"

	regionDataDirDefault  = "/mnt/octelium-e2e-regions"
	regionDataDirFallback = "/var/lib/octelium-e2e-regions"

	sharedStorageAddr = regionNetworkGateway

	sharedPostgres      = "octelium-e2e-postgres"
	sharedPostgresImage = "postgres:17"
	sharedPostgresPort  = 15432

	sharedRedis      = "octelium-e2e-redis"
	sharedRedisImage = "redis:7.4"
	sharedRedisPort  = 16379

	connectivityImage = "busybox:1.37"
)

var regionKubeletArgs = []string{
	"--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%",
	"--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%",
	"--kubelet-arg=image-gc-high-threshold=100",
}

func init() {
	register(MultiRegionScenario, opts{
		description: "The Enterprise scenario plus a second k3s cluster in a container, joined as a Region, " +
			"with Postgres and Redis shared outside both clusters",
		multiRegion: true,
		budget:      120 * time.Minute,
	})
}

func withMultiRegion(ret *scenario.Scenario) {
	ret.Caps = append(ret.Caps, CapMultiRegion)

	ret.Provisioner = &multiRegionProvisioner{Provisioner: ret.Provisioner}

	pg := &ret.Storage.Postgres
	pg.Chart = noopChartDir()
	pg.ChartVersion = ""
	pg.ImageRepository = ""
	pg.AllowInsecure = false
	pg.PVCName = ""
	pg.Host = sharedStorageAddr
	pg.Port = sharedPostgresPort

	redis := &ret.Storage.Redis
	redis.Chart = noopChartDir()
	redis.ChartVersion = ""
	redis.ImageRepository = ""
	redis.AllowInsecure = false
	redis.Host = sharedStorageAddr
	redis.Port = sharedRedisPort

	ret.Hooks.PostProvision = append(ret.Hooks.PostProvision,
		scenario.Step{Name: "multi-region/noop-storage-chart", Run: stepNoopStorageChart},
		scenario.Step{Name: "multi-region/network", Run: stepRegionNetwork},
		scenario.Step{Name: "multi-region/region-node", Run: stepRegionNode},
		scenario.Step{Name: "multi-region/region-ready", Run: stepRegionReady},
		scenario.Step{Name: "multi-region/region-labels", Run: stepRegionLabels},
	)

	ret.Hooks.PostPrepare = append(ret.Hooks.PostPrepare,
		scenario.Step{Name: "multi-region/shared-storage", Run: stepSharedStorage},
		scenario.Step{Name: "multi-region/region-multus", Run: stepRegionMultus},
		scenario.Step{Name: "multi-region/region-dns", Run: stepRegionDNS},
		scenario.Step{Name: "multi-region/connectivity", Run: stepRegionConnectivity},
	)
}

type multiRegionProvisioner struct {
	scenario.Provisioner
}

func (p *multiRegionProvisioner) Teardown(ctx context.Context, r *scenario.Runner) error {
	if err := r.Bash(ctx, regionTeardownScript()); err != nil {
		zap.L().Warn("Could not fully remove the Region cluster and the shared storage", zap.Error(err))
	}

	return p.Provisioner.Teardown(ctx, r)
}

func noopChartDir() string {
	return filepath.Join(os.TempDir(), "octelium-e2e", "noop-storage-chart")
}

func regionKubeconfigPath() string {
	return filepath.Join(os.TempDir(), "octelium-e2e", fmt.Sprintf("%s.kubeconfig", regionName))
}

func regionDataDir() string {
	if info, err := os.Stat(filepath.Dir(regionDataDirDefault)); err == nil && info.IsDir() {
		return regionDataDirDefault
	}

	return regionDataDirFallback
}

func regionAPIServer() string {
	return fmt.Sprintf("https://%s:%d", regionNetworkGateway, regionAPIPort)
}

const noopChartYAML = `apiVersion: v2
name: octelium-e2e-noop-storage
description: Takes the place of the in-cluster storage charts. Postgres and Redis run outside both clusters
type: application
version: 0.1.0
`

func stepNoopStorageChart(ctx context.Context, r *scenario.Runner) error {
	dir := noopChartDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(noopChartYAML), 0o644)
}

func stepRegionNetwork(ctx context.Context, r *scenario.Runner) error {
	return r.Bash(ctx, fmt.Sprintf(`
if sudo docker network inspect %[1]s >/dev/null 2>&1; then
  echo "the docker network %[1]s already exists"
  exit 0
fi

sudo docker network create \
  --driver bridge \
  --subnet %[2]s \
  --ip-range %[3]s \
  --gateway %[4]s \
  --opt com.docker.network.driver.mtu=1500 \
  %[1]s
`, regionNetwork, regionNetworkSubnet, regionNetworkIPRange, regionNetworkGateway))
}

func regionServerArgs() []string {
	ret := []string{
		"server",
		"--node-name", regionNode,
		"--disable", "traefik",
		"--disable", "metrics-server",
		"--tls-san", regionNetworkGateway,
		"--tls-san", regionNodeIP,
		"--cluster-cidr", regionClusterCIDR,
		"--service-cidr", regionServiceCIDR,
		"--cluster-dns", regionClusterDNS,
		"--write-kubeconfig", "/output/kubeconfig.yaml",
		"--write-kubeconfig-mode", "644",
		"--kube-proxy-arg=conntrack-max-per-core=0",
	}

	return append(ret, regionKubeletArgs...)
}

func regionNodeScript(dataDir string, cniBinDir string) string {
	var args []string
	for _, arg := range regionServerArgs() {
		args = append(args, shellQuote(arg))
	}

	return fmt.Sprintf(`
if sudo docker inspect %[1]s >/dev/null 2>&1; then
  echo "the Region node %[1]s already exists"
  exit 0
fi

VERSION=$(k3s --version | head -n1 | awk '{print $3}')
if [ -z "$VERSION" ]; then
  echo "could not determine the k3s server version" >&2
  exit 1
fi
IMAGE="rancher/k3s:${VERSION/+/-}"
sudo docker image inspect "$IMAGE" >/dev/null 2>&1 || sudo docker pull "$IMAGE"

sudo mkdir -p %[2]s/data %[2]s/output
sudo docker run -d --name %[1]s --hostname %[1]s \
  --privileged \
  --restart unless-stopped \
  --entrypoint /bin/sh \
  --network %[3]s --ip %[4]s \
  --publish %[5]s:%[6]d:6443 \
  --tmpfs /run --tmpfs /var/run \
  --ulimit nofile=1048576:1048576 \
  -v /lib/modules:/lib/modules:ro \
  -v %[2]s/data:/var/lib/rancher/k3s:rshared \
  -v %[2]s/output:/output \
  "$IMAGE" -c %[7]s k3s %[8]s

sudo docker ps --filter name=^%[1]s$ --format '{{.Names}} {{.Status}} {{.Networks}} {{.Ports}}'
`, regionNode, shellQuote(dataDir), regionNetwork, regionNodeIP,
		regionNetworkGateway, regionAPIPort,
		shellQuote(regionEntrypointScript(cniBinDir, "/bin")), strings.Join(args, " "))
}

func regionEntrypointScript(cniDir, binDir string) string {
	return fmt.Sprintf(`
set -e
BIN_DIR=%[1]s
CNI_BIN_DIR=%[2]s
test -x "$BIN_DIR/cni"
mkdir -p "$CNI_BIN_DIR"
ln -sf "$BIN_DIR/cni" "$CNI_BIN_DIR/cni"
for plugin in "$BIN_DIR"/*; do
  if [ -L "$plugin" ] && [ "$(readlink "$plugin")" = cni ]; then
    ln -sf "$BIN_DIR/cni" "$CNI_BIN_DIR/${plugin##*/}"
  fi
done
test -x "$CNI_BIN_DIR/host-local"
export PATH="$CNI_BIN_DIR:$PATH"
exec "$BIN_DIR/k3s" "$@"
`, shellQuote(binDir), shellQuote(cniDir))
}

func stepRegionNode(ctx context.Context, r *scenario.Runner) error {
	dataDir := regionDataDir()

	if err := r.Bash(ctx, regionNodeScript(dataDir,
		r.Scenario.Provisioner.CNIPaths().BinDir)); err != nil {
		return errors.Errorf("%+v\n%s", err, regionDiagnostics(ctx, r))
	}

	src := filepath.Join(dataDir, "output", "kubeconfig.yaml")
	dst := regionKubeconfigPath()

	err := pollUntil(ctx, "the Region kubeconfig", 5*time.Minute, func(ctx context.Context) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}

		ret, err := getRegionKubeconfig(data, regionAPIServer())
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fatal(err)
		}

		if err := os.WriteFile(dst, ret, 0o600); err != nil {
			return fatal(err)
		}

		return nil
	})
	if err != nil {
		return errors.Errorf("%+v\n%s", err, regionDiagnostics(ctx, r))
	}

	r.State.Set(StateRegionName, regionName)
	r.State.Set(StateRegionNode, regionNode)
	r.State.Set(StateRegionNodeIP, regionNodeIP)
	r.State.Set(StateRegionKubeconfig, dst)

	zap.L().Info("Started the Region cluster",
		zap.String("region", regionName),
		zap.String("node", regionNode),
		zap.String("nodeIP", regionNodeIP),
		zap.String("apiServer", regionAPIServer()),
		zap.String("kubeconfig", dst))

	return r.SaveState()
}

func getRegionKubeconfig(data []byte, server string) ([]byte, error) {
	cfg, err := clientcmd.Load(data)
	if err != nil {
		return nil, err
	}

	if len(cfg.Clusters) == 0 {
		return nil, errors.Errorf("The kubeconfig has no clusters")
	}

	for _, cluster := range cfg.Clusters {
		if cluster == nil {
			continue
		}
		if len(cluster.CertificateAuthorityData) == 0 {
			return nil, errors.Errorf("The kubeconfig cluster has no embedded certificate authority")
		}
		cluster.Server = server
	}

	return clientcmd.Write(*cfg)
}

func newRegionK8sC() (kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", regionKubeconfigPath())
	if err != nil {
		return nil, err
	}
	cfg.Timeout = 30 * time.Second

	return kubernetes.NewForConfig(cfg)
}

func stepRegionReady(ctx context.Context, r *scenario.Runner) error {
	k8sC, err := newRegionK8sC()
	if err != nil {
		return err
	}

	err = pollUntil(ctx, "the Region node to become Ready", 10*time.Minute,
		func(ctx context.Context) error {
			nodes, err := k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{})
			if err != nil {
				return err
			}
			if len(nodes.Items) == 0 {
				return errors.Errorf("The Region node has not registered yet")
			}

			for i := range nodes.Items {
				if !isNodeReady(&nodes.Items[i]) {
					return errors.Errorf("The Region node %s is not Ready yet", nodes.Items[i].Name)
				}
			}

			return nil
		})
	if err != nil {
		return errors.Errorf("%+v\n%s", err, regionDiagnostics(ctx, r))
	}

	return nil
}

func isNodeReady(node *k8scorev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == k8scorev1.NodeReady {
			return c.Status == k8scorev1.ConditionTrue
		}
	}

	return false
}

func regionKubectl() string {
	return fmt.Sprintf("export KUBECONFIG=%s\n", shellQuote(regionKubeconfigPath()))
}

func stepRegionLabels(ctx context.Context, r *scenario.Runner) error {
	var b strings.Builder
	b.WriteString(regionKubectl())

	for _, label := range r.Scenario.Topology.Labels {
		fmt.Fprintf(&b, "kubectl label nodes --all --overwrite %s=\n", label)
	}

	fmt.Fprintf(&b, "kubectl annotate nodes --all --overwrite octelium.com/public-ip-test=%s\n", regionNodeIP)
	b.WriteString("kubectl wait --for=condition=Ready nodes --all --timeout=600s\n")
	b.WriteString("kubectl get nodes -o wide --show-labels\n")

	return r.Bash(ctx, b.String())
}

func sharedStorageScript(username, database, pgPassword, redisPassword string) string {
	return fmt.Sprintf(`
if ! sudo docker inspect %[1]s >/dev/null 2>&1; then
  sudo docker run -d --name %[1]s --restart unless-stopped --network host \
    -e POSTGRES_USER=%[3]s \
    -e POSTGRES_PASSWORD=%[4]s \
    -e POSTGRES_DB=%[5]s \
    -e PGPORT=%[6]d \
    %[2]s -c listen_addresses=%[7]s -c max_connections=500
fi

if ! sudo docker inspect %[8]s >/dev/null 2>&1; then
  sudo docker run -d --name %[8]s --restart unless-stopped --network host \
    %[9]s redis-server --bind %[7]s --port %[10]d --requirepass %[11]s --save '' --appendonly no
fi

for i in $(seq 1 120); do
  if sudo docker exec %[1]s pg_isready -h %[7]s -p %[6]d -U %[3]s >/dev/null 2>&1 && \
    [ "$(sudo docker exec %[8]s redis-cli -h %[7]s -p %[10]d -a %[11]s --no-auth-warning ping 2>/dev/null)" = "PONG" ]; then
    echo "Postgres listens on %[7]s:%[6]d and Redis on %[7]s:%[10]d"
    exit 0
  fi
  sleep 2
done

echo "the shared Postgres and Redis did not become ready" >&2
sudo docker logs --tail 50 %[1]s >&2 || true
sudo docker logs --tail 50 %[8]s >&2 || true
exit 1
`, sharedPostgres, sharedPostgresImage, shellQuote(username), shellQuote(pgPassword), shellQuote(database),
		sharedPostgresPort, sharedStorageAddr,
		sharedRedis, sharedRedisImage, sharedRedisPort, shellQuote(redisPassword))
}

func stepSharedStorage(ctx context.Context, r *scenario.Runner) error {
	pg := r.Scenario.Storage.Postgres

	return r.Bash(ctx, sharedStorageScript(pg.Username, pg.Database,
		r.State.PostgresPassword, r.State.RedisPassword))
}

func stepRegionMultus(ctx context.Context, r *scenario.Runner) error {
	m := r.Scenario.Multus
	paths := r.Scenario.Provisioner.CNIPaths()

	args := []string{
		fmt.Sprintf("--set hostCNIBinDir=%s", paths.BinDir),
		fmt.Sprintf("--set hostCNINetDir=%s", paths.NetDir),
		`--set-json 'tolerations=[{"key":"octelium.com/gateway-init","operator":"Exists","effect":"NoSchedule"}]'`,
	}
	if m.ImageRepository != "" {
		args = append(args, fmt.Sprintf("--set image.repository=%s", m.ImageRepository))
	}
	if m.AllowInsecure {
		args = append(args, "--set global.security.allowInsecureImages=true")
	}
	if m.ChartVersion != "" {
		args = append(args, fmt.Sprintf("--version %s", m.ChartVersion))
	}

	return r.Bash(ctx, fmt.Sprintf(`%s
helm upgrade --install --namespace kube-system octelium-multus %s %s
kubectl rollout status daemonset -n kube-system -l app.kubernetes.io/instance=octelium-multus --timeout=10m
`, regionKubectl(), m.Chart, strings.Join(args, " ")))
}

func regionCoreDNSServer(domain string) string {
	return fmt.Sprintf(`%[1]s:53 {
    errors
    template IN A %[1]s {
        answer "{{ .Name }} 60 IN A %[2]s"
    }
    template IN AAAA %[1]s {
        rcode NOERROR
    }
}
`, domain, regionNetworkGateway)
}

func stepRegionDNS(ctx context.Context, r *scenario.Runner) error {
	cm := &k8scorev1.ConfigMap{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "coredns-custom",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"octelium-e2e.server": regionCoreDNSServer(r.Scenario.Domain),
		},
	}

	k8sC, err := newRegionK8sC()
	if err != nil {
		return err
	}

	cur, err := k8sC.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, k8smetav1.GetOptions{})
	switch {
	case err == nil:
		cur.Data = cm.Data
		_, err = k8sC.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cur, k8smetav1.UpdateOptions{})
	case k8serr.IsNotFound(err):
		_, err = k8sC.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, k8smetav1.CreateOptions{})
	}
	if err != nil {
		return err
	}

	return r.Bash(ctx, regionKubectl()+`
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout status deployment/coredns -n kube-system --timeout=5m
`)
}

func connectivityScript(name string, checks string) string {
	return fmt.Sprintf(`
kubectl delete pod %[1]s -n default --ignore-not-found --wait=true >/dev/null
kubectl run %[1]s -n default --image=%[2]s --restart=Never --rm -i --quiet \
  --pod-running-timeout=5m --command -- sh -c %[3]s
`, name, connectivityImage, shellQuote("set -e\n"+checks))
}

func regionConnectivityChecks(domain string) string {
	return fmt.Sprintf(`
echo "resolving collector.octelium.%[1]s"
nslookup collector.octelium.%[1]s | tee /tmp/out
grep -q "Address: %[2]s" /tmp/out
echo "reaching the shared Postgres at %[3]s:%[4]d"
nc -z -w 5 %[3]s %[4]d
echo "reaching the shared Redis at %[3]s:%[5]d"
nc -z -w 5 %[3]s %[5]d
`, domain, regionNetworkGateway, sharedStorageAddr, sharedPostgresPort, sharedRedisPort)
}

func defaultConnectivityChecks() string {
	return fmt.Sprintf(`
echo "reaching the shared Postgres at %[1]s:%[2]d"
nc -z -w 5 %[1]s %[2]d
echo "reaching the shared Redis at %[1]s:%[3]d"
nc -z -w 5 %[1]s %[3]d
echo "reaching the Region API server at %[4]s:%[5]d"
nc -z -w 5 %[4]s %[5]d
`, sharedStorageAddr, sharedPostgresPort, sharedRedisPort, regionNetworkGateway, regionAPIPort)
}

func stepRegionConnectivity(ctx context.Context, r *scenario.Runner) error {
	if err := r.Bash(ctx, regionKubectl()+connectivityScript("octelium-e2e-connectivity",
		regionConnectivityChecks(r.Scenario.Domain))); err != nil {
		return errors.Errorf("The Region cluster cannot reach the Cluster domain or the shared storage: %+v\n%s",
			err, regionDiagnostics(ctx, r))
	}

	if err := r.Bash(ctx, connectivityScript("octelium-e2e-connectivity",
		defaultConnectivityChecks())); err != nil {
		return errors.Errorf("The default Region cannot reach the shared storage or the Region API server: %+v\n%s",
			err, regionDiagnostics(ctx, r))
	}

	return nil
}

func regionDiagnostics(ctx context.Context, r *scenario.Runner) string {
	out, err := r.BashOutput(ctx, fmt.Sprintf(`
echo "--- containers ---"
sudo docker ps -a --filter name=^octelium-e2e- --format '{{.Names}} {{.Status}} {{.Networks}} {{.Ports}}' 2>&1
echo "--- %[1]s ---"
sudo docker logs --tail 50 %[1]s 2>&1 || true
echo "--- the %[2]s network ---"
sudo docker network inspect %[2]s --format '{{json .IPAM.Config}} {{json .Options}}' 2>&1 || true
echo "--- routes ---"
ip route 2>&1 || true
echo "--- nat rules for %[3]s ---"
sudo iptables -t nat -S 2>/dev/null | grep -E '%[3]s|%[4]d' | head -n 20 || true
`, regionNode, regionNetwork, regionNetworkGateway, regionAPIPort))
	if err != nil {
		return fmt.Sprintf("<could not collect the Region diagnostics: %+v>\n%s", err, out)
	}

	return out
}

func regionTeardownScript() string {
	return fmt.Sprintf(`
for c in %[1]s %[2]s %[3]s; do
  sudo docker rm -f "$c" >/dev/null 2>&1 || true
done
sudo docker network rm %[4]s >/dev/null 2>&1 || true
sudo rm -rf %[5]s %[6]s %[7]s
sudo sed -i '/%[8]s$/d' /etc/hosts || true
`, regionNode, sharedPostgres, sharedRedis, regionNetwork,
		shellQuote(regionDataDir()), shellQuote(regionKubeconfigPath()), shellQuote(noopChartDir()),
		RegionHostsMarker)
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
