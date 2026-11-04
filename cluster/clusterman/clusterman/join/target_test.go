package join

import (
	"io"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestJoinRejectsForeignTargetBeforePreparation(t *testing.T) {
	for _, state := range []string{"bootstrap-domain", "cluster", "region", "partial", "malformed", "replaced-namespace", "terminating-namespace"} {
		t.Run(state, func(t *testing.T) {
			env := newTestJoinEnv(t, true)
			cc, err := env.tst.C.OcteliumC.CoreV1Utils().GetClusterConfig(t.Context())
			require.NoError(t, err)
			annotations := map[string]string{
				"octelium.com/genesis-cluster-uid":   cc.Metadata.Uid,
				"octelium.com/genesis-domain":        cc.Status.Domain,
				"octelium.com/genesis-region-uid":    "region-uid",
				"octelium.com/genesis-region-name":   "west",
				"octelium.com/genesis-namespace-uid": "namespace-uid",
			}
			switch state {
			case "cluster":
				annotations["octelium.com/genesis-cluster-uid"] = "foreign-cluster"
			case "region":
				annotations["octelium.com/genesis-region-name"] = "east"
			case "partial":
				delete(annotations, "octelium.com/genesis-namespace-uid")
			}
			if state == "bootstrap-domain" || state == "malformed" {
				data := pbutils.MarshalMust(&corev1.Region{Metadata: &metav1.Metadata{Name: "west"}})
				if state == "malformed" {
					data = []byte("invalid")
				}
				require.NoError(t, env.targetC.Tracker().Add(&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: initSecretName, Namespace: vutils.K8sNS}, Data: map[string][]byte{"domain": []byte("foreign.example"), "region": data}}))
			} else {
				require.NoError(t, env.targetC.Tracker().Add(&k8scorev1.ConfigMap{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-installation", Namespace: vutils.K8sNS, Annotations: annotations}}))
			}
			if state == "replaced-namespace" || state == "terminating-namespace" {
				namespace := &k8scorev1.Namespace{ObjectMeta: k8smetav1.ObjectMeta{Name: vutils.K8sNS, UID: "namespace-uid"}}
				if state == "replaced-namespace" {
					namespace.UID = "replacement"
				} else {
					namespace.DeletionTimestamp = new(k8smetav1.Now())
				}
				require.NoError(t, env.targetC.Tracker().Add(namespace))
			}
			_, err = JoinCluster(t.Context(), env.opts(getTestJoinRequest("west")))
			require.Error(t, err)
			for _, action := range env.targetC.Actions() {
				assert.Equal(t, "get", action.GetVerb())
			}
			assert.Empty(t, env.jobs)
		})
	}
}

func TestJoinTargetChecksRetryTransientReads(t *testing.T) {
	for _, resource := range []string{"namespaces", "secrets"} {
		for _, transient := range []bool{false, true} {
			t.Run(resource+map[bool]string{false: "-forbidden", true: "-transient"}[transient], func(t *testing.T) {
				c := fake.NewSimpleClientset(getTestNamespace("target-cluster"))
				calls := 0
				c.PrependReactor("get", resource, func(action k8stesting.Action) (bool, runtime.Object, error) {
					calls++
					if transient && calls > 1 {
						return false, nil, nil
					}
					if transient {
						return true, nil, io.ErrUnexpectedEOF
					}
					return true, nil, k8serr.NewForbidden(action.GetResource().GroupResource(), "target", io.EOF)
				})
				ctl := &controller{k8sC: getTestLocalK8sC(), req: getTestJoinRequest("west").Request}
				err := ctl.checkTarget(t.Context(), c)
				if transient {
					require.NoError(t, err)
					assert.Equal(t, 2, calls)
				} else {
					require.Error(t, err)
					assert.Equal(t, 1, calls)
				}
				for _, action := range c.Actions() {
					assert.Equal(t, "get", action.GetVerb())
				}
			})
		}
	}
}

func TestJoinExistingBoundRegion(t *testing.T) {
	env := newTestJoinEnv(t, true)
	cc, err := env.tst.C.OcteliumC.CoreV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	region := &corev1.Region{Metadata: &metav1.Metadata{Name: "west"}, Spec: &corev1.Region_Spec{}, Status: &corev1.Region_Status{Index: 1, VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{regionutils.PackageEnterprise: {Package: regionutils.PackageEnterprise, Version: "1.5.0"}}}}
	require.NoError(t, regionutils.SetRegionExtInfo(region, &enterprisev1.RegionExtInfo{Kubernetes: getTestKubernetes()}))
	region, err = env.tst.C.OcteliumC.CoreC().CreateRegion(t.Context(), region)
	require.NoError(t, err)
	annotations := map[string]string{
		"octelium.com/genesis-cluster-uid":   cc.Metadata.Uid,
		"octelium.com/genesis-domain":        cc.Status.Domain,
		"octelium.com/genesis-region-uid":    region.Metadata.Uid,
		"octelium.com/genesis-region-name":   region.Metadata.Name,
		"octelium.com/genesis-namespace-uid": "namespace-uid",
	}
	require.NoError(t, env.targetC.Tracker().Add(&k8scorev1.Namespace{ObjectMeta: k8smetav1.ObjectMeta{Name: vutils.K8sNS, UID: "namespace-uid"}}))
	cm := &k8scorev1.ConfigMap{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-installation", Namespace: vutils.K8sNS, Annotations: annotations}, Data: map[string]string{"custom": "preserved"}}
	require.NoError(t, env.targetC.Tracker().Add(cm))
	env.targetC.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		job := action.(k8stesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.UID = types.UID("job-" + job.Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		env.jobs = append(env.jobs, job)
		err := env.targetC.Tracker().Create(action.GetResource(), job, job.Namespace)
		return true, job, err
	})
	joined, err := JoinCluster(t.Context(), env.opts(getTestJoinRequest("west")))
	require.NoError(t, err)
	assert.Equal(t, region.Metadata.Uid, joined.Metadata.Uid)
	require.Len(t, env.jobs, 2)
	for _, job := range env.jobs {
		values := map[string]string{}
		for _, entry := range job.Spec.Template.Spec.Containers[0].Env {
			values[entry.Name] = entry.Value
		}
		assert.Equal(t, cc.Metadata.Uid, values["OCTELIUM_GENESIS_CLUSTER_UID"])
		assert.Equal(t, region.Metadata.Uid, values["OCTELIUM_GENESIS_REGION_UID"])
		assert.Equal(t, "namespace-uid", values["OCTELIUM_GENESIS_NAMESPACE_UID"])
	}
	stored, err := env.targetC.CoreV1().ConfigMaps(vutils.K8sNS).Get(t.Context(), cm.Name, k8smetav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, cm, stored)
}
