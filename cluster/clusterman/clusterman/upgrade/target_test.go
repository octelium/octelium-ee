package upgrade

import (
	"maps"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpgradeBindsAllPackagesToSourceClusterAndRegion(t *testing.T) {
	for _, mode := range []string{"legacy", "bound", "foreign-cluster", "foreign-region", "incomplete"} {
		t.Run(mode, func(t *testing.T) {
			var order []string
			c := newCompletingK8sC(&order, "west", false)
			region := getTestJoinedRegion(t, "west", 1, regionutils.PackageEnterprise, regionutils.PackageCordium)
			annotations := map[string]string{
				"octelium.com/genesis-cluster-uid":   "source-cluster",
				"octelium.com/genesis-domain":        "example.com",
				"octelium.com/genesis-region-uid":    region.Metadata.Uid,
				"octelium.com/genesis-region-name":   region.Metadata.Name,
				"octelium.com/genesis-namespace-uid": "namespace-uid",
			}
			switch mode {
			case "foreign-cluster":
				annotations["octelium.com/genesis-cluster-uid"] = "foreign"
			case "foreign-region":
				annotations["octelium.com/genesis-region-uid"] = "foreign"
			case "incomplete":
				delete(annotations, "octelium.com/genesis-namespace-uid")
			}
			if mode != "legacy" {
				require.NoError(t, c.Tracker().Add(&k8scorev1.ConfigMap{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-installation", Namespace: vutils.K8sNS, Annotations: maps.Clone(annotations)}}))
			}
			ctl := &controller{clusterUID: "source-cluster", domain: "example.com", req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "2.0.0"},
				PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "2.0.0"},
				PackageCordium:    &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{Version: "2.0.0"},
			}}}
			err := ctl.runTarget(t.Context(), &regionTarget{region: region, k8sC: c})
			if mode != "legacy" && mode != "bound" {
				require.Error(t, err)
				assert.Empty(t, order)
				for _, action := range c.Actions() {
					assert.Equal(t, "get", action.GetVerb())
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{"west/octelium-genesis", "west/octeliumee-genesis", "west/cordium-genesis"}, order)
			jobs, err := c.BatchV1().Jobs(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, jobs.Items, 3)
			for _, job := range jobs.Items {
				env := map[string]string{}
				for _, entry := range job.Spec.Template.Spec.Containers[0].Env {
					env[entry.Name] = entry.Value
				}
				assert.Equal(t, "source-cluster", env["OCTELIUM_GENESIS_CLUSTER_UID"])
				assert.Equal(t, region.Metadata.Uid, env["OCTELIUM_GENESIS_REGION_UID"])
				assert.Equal(t, region.Metadata.Name, env["OCTELIUM_REGION_NAME"])
				if mode == "bound" {
					assert.Equal(t, "namespace-uid", env["OCTELIUM_GENESIS_NAMESPACE_UID"])
				} else {
					assert.Empty(t, env["OCTELIUM_GENESIS_NAMESPACE_UID"])
				}
			}
		})
	}
}
