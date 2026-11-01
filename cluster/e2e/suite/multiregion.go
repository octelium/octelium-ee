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
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	multiRegionJoinBudget    = 35 * time.Minute
	multiRegionUpgradeBudget = 50 * time.Minute
	multiRegionGatewayBudget = 5 * time.Minute
	multiRegionAccessBudget  = 3 * time.Minute

	multiRegionSecretName  = "e2e-multi-region-kubeconfig"
	multiRegionDisplayName = "E2E Region"
)

var regionDeployments = []string{
	"octelium-rscserver",
	"octelium-nocturne",
	"octelium-octovigil",
	"octeliumee-rscserver",
	"octeliumee-nocturne",
	"octeliumee-secretman",
	"octeliumee-collector",
}

var defaultRegionDeployments = []string{
	"octelium-ingress",
	"octelium-ingress-dataplane",
	"octeliumee-clusterman",
	"octeliumee-cloudman",
	"octeliumee-logstore",
	"octeliumee-rscstore",
	"octeliumee-metricstore",
	"octeliumee-policyportal",
}

var defaultRegionK8sServices = []string{
	"octelium-ingress",
	"octelium-ingress-dataplane",
}

func dumpRegionOnFailure(t *testing.T, h *eeharness.H, rgn *eeharness.Region) {
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		for _, args := range []string{
			"get nodes -o wide",
			"get pods -A -o wide",
			"get jobs -n octelium -o wide",
			"get events -n octelium --sort-by=.lastTimestamp",
			"logs -n octelium -l octelium.com/component=genesis --all-containers --tail=200",
			"logs -n octelium -l octelium.com/component=collector --all-containers --tail=100",
		} {
			h.Run(ctx, rgn.Kubectl(args))
		}
	})
}

func getRegionSecret(t *testing.T, h *eeharness.H, rgn *eeharness.Region) *enterprisev1.Secret {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	data := &enterprisev1.Secret_Data{
		Type: &enterprisev1.Secret_Data_Value{Value: rgn.KubeconfigData(t)},
	}

	sec, err := h.EnterpriseC().GetSecret(ctx, &metav1.GetOptions{Name: multiRegionSecretName})
	if err == nil {
		sec.Data = data
		sec, err = h.EnterpriseC().UpdateSecret(ctx, sec)
		require.Nil(t, err)
		return sec
	}
	require.True(t, grpcerr.IsNotFound(err), "%+v", err)

	sec, err = h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: multiRegionSecretName},
		Spec:     &enterprisev1.Secret_Spec{},
		Data:     data,
	})
	require.Nil(t, err)

	return sec
}

func waitJoinDone(t *testing.T, h *eeharness.H, beforeCount int,
	budget time.Duration) *enterprisev1.ClusterConfig_Status_JoinRequest {
	t.Helper()

	var ret *enterprisev1.ClusterConfig_Status_JoinRequest
	h.Eventually(t, "the join to finish", budget,
		func(ctx context.Context) error {
			cc, err := h.EnterpriseC().GetClusterConfig(ctx,
				&enterprisev1.GetClusterConfigRequest{})
			if err != nil {
				return err
			}
			if cc.Status.JoinRequest != nil {
				return errors.Errorf("the join is %s", cc.Status.JoinRequest.State)
			}
			if len(cc.Status.LastJoinRequests) <= beforeCount {
				return errors.Errorf("the join is not recorded yet")
			}
			ret = cc.Status.LastJoinRequests[0]
			return nil
		})

	require.NotNil(t, ret)
	return ret
}

func getJoinedRegion(t *testing.T, h *eeharness.H, name string) *corev1.Region {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: name})
	require.Nil(t, err, "the Region %s has not joined the Cluster: %+v", name, err)
	require.NotNil(t, ret.Status.VersionInfoMap)
	require.Contains(t, ret.Status.VersionInfoMap, eescenario.Package,
		"the Region %s has not completed joining the Cluster", name)

	return ret
}

func getRegionGateways(ctx context.Context, h *eeharness.H, rgn *corev1.Region) ([]*corev1.Gateway, error) {
	gws, err := h.ListGateways(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*corev1.Gateway
	for _, gw := range gws {
		if gw.Status.GetRegionRef().GetUid() == rgn.Metadata.Uid {
			ret = append(ret, gw)
		}
	}

	return ret, nil
}

func isRegionCollectorSession(sess *corev1.Session) bool {
	if sess.Status.Type != corev1.Session_Status_CLIENTLESS {
		return false
	}

	if sess.Status.GetAuthentication().GetInfo().GetType() != corev1.Session_Status_Authentication_Info_INTERNAL {
		return false
	}

	for _, p := range sess.Spec.GetAuthorization().GetInlinePolicies() {
		for _, rule := range p.GetSpec().GetRules() {
			if rule.Effect == corev1.Policy_Spec_Rule_DENY &&
				strings.Contains(rule.GetCondition().GetNot(), collectorRemoteServiceName) {
				return true
			}
		}
	}

	return false
}

func newRegionJobs(t *testing.T, rgn *eeharness.Region, existing map[types.UID]bool) []batchv1.Job {
	t.Helper()

	jobs, err := rgn.K8sC().BatchV1().Jobs(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{
		LabelSelector: "octelium.com/component=genesis",
	})
	require.Nil(t, err)

	var ret []batchv1.Job
	for _, job := range jobs.Items {
		if !existing[job.UID] {
			ret = append(ret, job)
		}
	}

	return ret
}

func isJobComplete(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == k8scorev1.ConditionTrue {
			return true
		}
	}

	return job.Status.Succeeded > 0
}

func assertNoDefaultRegionComponents(t *testing.T, rgn *eeharness.Region) {
	t.Helper()

	ctx := t.Context()

	for _, name := range defaultRegionDeployments {
		_, err := rgn.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
		assert.True(t, k8serr.IsNotFound(err), "the Deployment %s exists in the Region: %+v", name, err)
	}

	for _, name := range defaultRegionK8sServices {
		_, err := rgn.K8sC().CoreV1().Services(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
		assert.True(t, k8serr.IsNotFound(err), "the Kubernetes Service %s exists in the Region: %+v", name, err)
	}
}

func testMultiRegionJoin(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)

	ctx := t.Context()

	sec := getRegionSecret(t, h, rgn)

	before := h.EnterpriseClusterConfig(t)
	beforeCount := len(before.Status.LastJoinRequests)

	_, err := h.ClusterC().JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
		Name:        rgn.Name,
		DisplayName: multiRegionDisplayName,
		Kubernetes:  getJoinKubeconfig(sec.Metadata.Name),
	})
	require.Nil(t, err)

	last := waitJoinDone(t, h, beforeCount, multiRegionJoinBudget)
	require.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS, last.State,
		"the join failed: %s", last.Message)
	require.NotNil(t, last.RegionRef)
	assert.Equal(t, rgn.Name, last.RegionRef.Name)
	assert.Equal(t, rgn.Name, last.Request.Name)
	assert.NotNil(t, last.DoneAt)

	region := getJoinedRegion(t, h, rgn.Name)

	t.Run("TheRegionIsRecorded", func(t *testing.T) {
		defaultRgn, err := h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: "default"})
		require.Nil(t, err)

		assert.Equal(t, last.RegionRef.Uid, region.Metadata.Uid)
		assert.Equal(t, multiRegionDisplayName, region.Metadata.DisplayName)
		assert.GreaterOrEqual(t, region.Status.Index, int32(1))
		assert.NotEqual(t, defaultRgn.Status.Index, region.Status.Index)
		assert.NotEmpty(t, region.Status.PublicHostname)

		for _, pkg := range []string{"octelium", eescenario.Package} {
			require.Contains(t, region.Status.VersionInfoMap, pkg)
			require.Contains(t, defaultRgn.Status.VersionInfoMap, pkg)
			assert.Equal(t, defaultRgn.Status.VersionInfoMap[pkg].Version,
				region.Status.VersionInfoMap[pkg].Version, "package %s", pkg)
		}
	})

	t.Run("TheRegionHasItsOwnGateway", func(t *testing.T) {
		h.Eventually(t, "the Gateway of the Region to register", multiRegionGatewayBudget,
			func(ctx context.Context) error {
				gws, err := getRegionGateways(ctx, h, region)
				if err != nil {
					return err
				}
				if len(gws) == 0 {
					return errNotProvisioned("the Gateway of the Region")
				}
				for _, gw := range gws {
					if !slices.Contains(gw.Status.PublicIPs, rgn.NodeIP) {
						return errors.Errorf("the Gateway %s publishes %v, want %s",
							gw.Metadata.Name, gw.Status.PublicIPs, rgn.NodeIP)
					}
				}
				return nil
			})
	})

	t.Run("AnotherNameForTheSameClusterIsRejected", func(t *testing.T) {
		otherName := fmt.Sprintf("%s-other", rgn.Name)
		beforeCount := len(h.EnterpriseClusterConfig(t).Status.LastJoinRequests)

		_, err := h.ClusterC().JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       otherName,
			Kubernetes: getJoinKubeconfig(sec.Metadata.Name),
		})
		require.Nil(t, err)

		last := waitJoinDone(t, h, beforeCount, regionJoinBudget)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, last.State)
		assert.Equal(t, otherName, last.Request.Name)
		assert.Contains(t, last.Message, rgn.Name)
		assert.Nil(t, last.RegionRef)

		_, err = h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: otherName})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	})
}

func testMultiRegionComponents(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)
	getJoinedRegion(t, h, rgn.Name)

	ctx := t.Context()

	t.Run("TheRegionComponentsAreUp", func(t *testing.T) {
		for _, name := range regionDeployments {
			rgn.MustWaitDeployment(t, name)
		}
		rgn.MustWaitDaemonSet(t, "octelium-gwagent")
	})

	t.Run("TheDefaultRegionComponentsAreNotInstalled", func(t *testing.T) {
		assertNoDefaultRegionComponents(t, rgn)

		deps, err := rgn.K8sC().AppsV1().Deployments(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{
			LabelSelector: "octelium.com/component=svc",
		})
		require.Nil(t, err)
		for _, dep := range deps.Items {
			assert.Fail(t, "a system Service runs in the Region", "%s", dep.Name)
		}
	})

	t.Run("TheRegionRunsTheImagesOfTheDefaultRegion", func(t *testing.T) {
		for _, name := range regionDeployments {
			want, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.Nil(t, err)
			got, err := rgn.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.Nil(t, err)

			require.Len(t, got.Spec.Template.Spec.Containers, len(want.Spec.Template.Spec.Containers), name)
			for i := range want.Spec.Template.Spec.Containers {
				assert.Equal(t, want.Spec.Template.Spec.Containers[i].Image,
					got.Spec.Template.Spec.Containers[i].Image, name)
			}
		}
	})

	t.Run("TheJoinJobCompleted", func(t *testing.T) {
		jobs := newRegionJobs(t, rgn, nil)

		var completed int
		for i := range jobs {
			if strings.HasPrefix(jobs[i].Name, "octelium-genesis-join-") && isJobComplete(&jobs[i]) {
				completed++
			}
		}
		assert.GreaterOrEqual(t, completed, 1)
	})
}

func testMultiRegionTelemetry(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)
	getJoinedRegion(t, h, rgn.Name)

	ctx := t.Context()

	usr, err := h.CoreC().GetUser(ctx, &metav1.GetOptions{Name: "octelium"})
	require.Nil(t, err)

	sessions := map[string]bool{}

	t.Run("TheRegionCollectorsHoldTheirOwnSessions", func(t *testing.T) {
		h.Eventually(t, "the collectors of the Region to create their Sessions", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				sessList, err := h.CoreC().ListSession(ctx, &corev1.ListSessionOptions{
					UserRef: umetav1.GetObjectReference(usr),
				})
				if err != nil {
					return err
				}
				for _, sess := range sessList.Items {
					if isRegionCollectorSession(sess) && sess.Spec.State == corev1.Session_Spec_ACTIVE {
						sessions[sess.Metadata.Uid] = true
					}
				}
				if len(sessions) == 0 {
					return errNotProvisioned("the Session of a collector of the Region")
				}
				return nil
			})
	})

	t.Run("TheRegionExportsThroughTheCollectorService", func(t *testing.T) {
		svc, err := h.CoreC().GetService(ctx, &metav1.GetOptions{Name: collectorRemoteServiceName})
		require.Nil(t, err)

		h.Eventually(t, "the exports of the Region to be logged by the collector Service",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
					ServiceRef: umetav1.GetObjectReference(svc),
					UserRef:    umetav1.GetObjectReference(usr),
					Status:     corev1.AccessLog_Entry_Common_ALLOWED,
				})
				if err != nil {
					return err
				}
				for _, itm := range res.Items {
					if sessions[itm.GetEntry().GetCommon().GetSessionRef().GetUid()] {
						return nil
					}
				}
				return errNotProvisioned("an AccessLog of an export of the Region")
			})
	})
}

func testMultiRegionService(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)
	region := getJoinedRegion(t, h, rgn.Name)

	ctx := t.Context()

	svc := h.CreateService(t, &corev1.Service{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec: &corev1.Service_Spec{
			Mode:   corev1.Service_Spec_HTTP,
			Region: rgn.Name,
			Config: &corev1.Service_Spec_Config{
				Upstream: &corev1.Service_Spec_Config_Upstream{
					Type: &corev1.Service_Spec_Config_Upstream_Container_{
						Container: &corev1.Service_Spec_Config_Upstream_Container{
							Image: "nginx",
							Port:  80,
						},
					},
				},
			},
			Authorization: &corev1.Service_Spec_Authorization{
				InlinePolicies: harness.InlineAllowAny("allow"),
			},
		},
	})
	require.Equal(t, region.Metadata.Uid, svc.Status.GetRegionRef().GetUid())

	t.Run("ItRunsInTheRegionOnly", func(t *testing.T) {
		rgn.MustWaitDeployment(t, h.ServiceDeployment(svc.Metadata.Name))
		rgn.MustWaitDeployment(t, k8sutils.GetSvcK8sUpstreamHostname(svc, ""))

		_, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx,
			h.ServiceDeployment(svc.Metadata.Name), k8smetav1.GetOptions{})
		assert.True(t, k8serr.IsNotFound(err), "%+v", err)
	})

	t.Run("ItIsReachedThroughTheRegionGateway", func(t *testing.T) {
		var gws []*corev1.Gateway
		h.Eventually(t, "the Gateway of the Region", multiRegionGatewayBudget,
			func(ctx context.Context) error {
				var err error
				gws, err = getRegionGateways(ctx, h, region)
				if err != nil {
					return err
				}
				if len(gws) == 0 {
					return errNotProvisioned("the Gateway of the Region")
				}
				return nil
			})
		rgn.SetGatewayHosts(t, gws)

		port := h.Port()
		conn := h.Connect(t, harness.ConnectOpts{
			Publish: map[string]int{svc.Metadata.Name: port},
		})

		h.Eventually(t, "the Service of the Region to answer", multiRegionAccessBudget,
			func(ctx context.Context) error {
				got, err := h.StatusOf(ctx, h.HTTP(), conn.URL(svc.Metadata.Name))
				if err != nil {
					return err
				}
				if got != http.StatusOK {
					return errUnexpectedStatus(got, http.StatusOK)
				}
				return nil
			})
	})

	t.Run("ItsAccessLogsReachTheDefaultRegion", func(t *testing.T) {
		h.Eventually(t, "the AccessLogs of the Region to reach the LogStore", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				res, err := h.AccessLogC().ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
					ServiceRef: umetav1.GetObjectReference(svc),
					RegionRef:  umetav1.GetObjectReference(region),
				})
				if err != nil {
					return err
				}
				if len(res.Items) == 0 {
					return errNotProvisioned("an AccessLog of the Service of the Region")
				}
				return nil
			})
	})
}

func testMultiRegionUpgrade(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)
	getJoinedRegion(t, h, rgn.Name)

	ctx := t.Context()

	info, err := h.ClusterC().GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
	require.Nil(t, err)
	require.NotNil(t, info.Core)
	require.NotNil(t, info.PackageEnterprise)

	before := h.EnterpriseClusterConfig(t)
	require.Nil(t, before.Status.UpgradeRequest)

	existing := map[types.UID]bool{}
	for _, job := range newRegionJobs(t, rgn, nil) {
		existing[job.UID] = true
	}

	startedAt := time.Now()

	_, err = h.ClusterC().UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
				Version: info.Core.CurrentVersion,
			},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: info.PackageEnterprise.CurrentVersion,
			},
		},
	})
	require.Nil(t, err)

	var failed *enterprisev1.ClusterConfig_Status_UpgradeRequest
	h.Eventually(t, "the upgrade of every Region to finish", multiRegionUpgradeBudget,
		func(ctx context.Context) error {
			cc, err := h.EnterpriseC().GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
			if err != nil {
				return err
			}
			if cc.Status.TotalFailedUpgrades > before.Status.TotalFailedUpgrades {
				if len(cc.Status.LastUpgradeRequests) > 0 {
					failed = cc.Status.LastUpgradeRequests[0]
				}
				return nil
			}
			if cc.Status.UpgradeRequest == nil &&
				cc.Status.TotalSuccessfulUpgrades > before.Status.TotalSuccessfulUpgrades {
				return nil
			}
			if cc.Status.UpgradeRequest == nil {
				return errors.Errorf("the upgrade is not started yet")
			}

			var regions []string
			for _, itm := range cc.Status.UpgradeRequest.Regions {
				regions = append(regions, fmt.Sprintf("%s=%s", itm.RegionRef.GetName(), itm.State))
			}
			return errors.Errorf("the upgrade is %s [%s]",
				cc.Status.UpgradeRequest.State, strings.Join(regions, " "))
		})
	require.Nil(t, failed, "the upgrade failed: %+v", failed)

	after := h.EnterpriseClusterConfig(t)
	require.NotEmpty(t, after.Status.LastUpgradeRequests)
	history := after.Status.LastUpgradeRequests[0]

	t.Run("EveryRegionIsUpgradedInOrder", func(t *testing.T) {
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.State)
		require.Len(t, history.Regions, 2)

		assert.Equal(t, "default", history.Regions[0].RegionRef.GetName())
		assert.Equal(t, rgn.Name, history.Regions[1].RegionRef.GetName())

		for _, itm := range history.Regions {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, itm.State,
				"%s: %s", itm.RegionRef.GetName(), itm.Message)
			require.NotNil(t, itm.DoneAt)
		}

		assert.False(t, history.Regions[1].DoneAt.AsTime().Before(history.Regions[0].DoneAt.AsTime()))
	})

	t.Run("TheRegionRanTheUpgradeJobs", func(t *testing.T) {
		var containers []string
		for _, job := range newRegionJobs(t, rgn, existing) {
			if !isJobComplete(&job) {
				continue
			}
			for _, c := range job.Spec.Template.Spec.Containers {
				containers = append(containers, c.Name)
			}
		}

		assert.Contains(t, containers, "octelium-genesis")
		assert.Contains(t, containers, "octeliumee-genesis")
	})

	t.Run("TheRegionStaysWithoutTheDefaultRegionComponents", func(t *testing.T) {
		for _, name := range regionDeployments {
			rgn.MustWaitDeployment(t, name)
		}
		rgn.MustWaitDaemonSet(t, "octelium-gwagent")

		assertNoDefaultRegionComponents(t, rgn)
	})

	t.Run("TheRegionVersionsAreRecorded", func(t *testing.T) {
		region := getJoinedRegion(t, h, rgn.Name)

		for _, pkg := range []string{"octelium", eescenario.Package} {
			require.Contains(t, region.Status.VersionInfoMap, pkg)
			itm := region.Status.VersionInfoMap[pkg]
			require.NotNil(t, itm.SetAt, pkg)
			assert.False(t, itm.SetAt.AsTime().Before(startedAt.Add(-time.Minute)), pkg)
		}
	})
}
