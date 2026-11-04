// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fakenad "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/fake"
	"github.com/octelium/octelium-ee/cluster/common/collectorutils"
	oc "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/components"
	"github.com/octelium/octelium/cluster/common/vutils"
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func getTestJoinBootstrap() *cbootstrapv1.Config {
	return &cbootstrapv1.Config{
		Spec: &cbootstrapv1.Config_Spec{
			PrimaryStorage: &cbootstrapv1.Config_Spec_PrimaryStorage{
				Type: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_{
					Postgresql: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql{
						Username: "octelium",
						PasswordType: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_Password{
							Password: "pg-password",
						},
						Host:     "postgres.example.com",
						Port:     5432,
						Database: "octelium",
						IsTLS:    true,
					},
				},
			},
			SecondaryStorage: &cbootstrapv1.Config_Spec_SecondaryStorage{
				Type: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_{
					Redis: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis{
						Username: "default",
						PasswordType: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_Password{
							Password: "redis-password",
						},
						Host:     "redis.example.com",
						Port:     6379,
						Database: 2,
					},
				},
			},
		},
	}
}

func getTestJoinRegion(t *testing.T, name string, index int32) *corev1.Region {
	ret := &corev1.Region{
		Metadata: &metav1.Metadata{
			Name:        name,
			DisplayName: "West Region",
		},
		Spec: &corev1.Region_Spec{},
		Status: &corev1.Region_Status{
			Index: index,
		},
	}

	require.NoError(t, regionutils.SetRegionExtInfo(ret, &enterprisev1.RegionExtInfo{
		Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
			Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
				Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
					Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
						FromSecret: "west-kubeconfig",
					},
				},
			},
		},
	}))

	return ret
}

func getTestJoinSecret(bs *cbootstrapv1.Config, rgn *corev1.Region, domain string) *k8scorev1.Secret {
	return &k8scorev1.Secret{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      joinInitSecretName,
			Namespace: vutils.K8sNS,
		},
		Data: map[string][]byte{
			"bootstrap": pbutils.MarshalMust(bs),
			"region":    pbutils.MarshalMust(rgn),
			"domain":    []byte(domain),
		},
	}
}

func TestCoreK8sCPinsCoreImages(t *testing.T) {
	ctx := context.Background()
	fakeC := fake.NewClientset()
	c := newCoreK8sC(fakeC, "9.9.9")

	opts := &gc.CommonOpts{
		K8sC:          c,
		ClusterConfig: &corev1.ClusterConfig{Status: &corev1.ClusterConfig_Status{}},
		Region: &corev1.Region{
			Metadata: &metav1.Metadata{Name: "west", Uid: "uid"},
			Status:   &corev1.Region_Status{},
		},
	}

	for range 2 {
		require.NoError(t, gc.CreateRscServer(ctx, opts))
		require.NoError(t, gc.CreateGatewayAgent(ctx, opts))
		require.NoError(t, gc.CreateNocturne(ctx, opts))
		require.NoError(t, gc.CreateOctovigil(ctx, opts))

		for name, component := range map[string]string{
			"octelium-rscserver": components.RscServer,
			"octelium-nocturne":  components.Nocturne,
			"octelium-octovigil": components.Octovigil,
		} {
			dep, err := fakeC.AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, components.GetImage(component, "9.9.9"), dep.Spec.Template.Spec.Containers[0].Image)
		}

		ds, err := fakeC.AppsV1().DaemonSets(vutils.K8sNS).Get(ctx, "octelium-gwagent", k8smetav1.GetOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, ds.Spec.Template.Spec.InitContainers)
		assert.Equal(t, components.GetImage(components.NodeInit, "9.9.9"), ds.Spec.Template.Spec.InitContainers[0].Image)
		assert.Equal(t, components.GetImage(components.GWAgent, "9.9.9"), ds.Spec.Template.Spec.Containers[0].Image)
	}

	{
		itm := &appsv1.Deployment{
			ObjectMeta: k8smetav1.ObjectMeta{Name: "other", Namespace: vutils.K8sNS},
			Spec: appsv1.DeploymentSpec{
				Template: k8scorev1.PodTemplateSpec{
					Spec: k8scorev1.PodSpec{
						Containers: []k8scorev1.Container{
							{Name: "a", Image: "nginx"},
							{Name: "b", Image: oc.GetImage(oc.Nocturne, "")},
						},
					},
				},
			},
		}

		ret, err := c.AppsV1().Deployments(vutils.K8sNS).Create(ctx, itm, k8smetav1.CreateOptions{})
		require.NoError(t, err)
		assert.Equal(t, "nginx", ret.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, oc.GetImage(oc.Nocturne, ""), ret.Spec.Template.Spec.Containers[1].Image)
		assert.Equal(t, components.GetImage(components.RscServer, ""), getCoreImage(components.GetImage(components.RscServer, ""), ""))
	}

	{
		spec := &k8scorev1.PodSpec{
			Containers: []k8scorev1.Container{{Image: components.GetImage(components.RscServer, "")}},
		}
		setCoreImages(spec, "")
		assert.Equal(t, components.GetImage(components.RscServer, ""), spec.Containers[0].Image)
	}
}

func TestParseJoinResources(t *testing.T) {
	bs := getTestJoinBootstrap()
	rgn := getTestJoinRegion(t, "west", 1)

	{
		res, err := parseJoinResources(getTestJoinSecret(bs, rgn, "example.com"))
		require.NoError(t, err)
		assert.Equal(t, "west", res.Region.Metadata.Name)
		assert.EqualValues(t, 1, res.Region.Status.Index)
		assert.Equal(t, "example.com", res.Domain)
		assert.Equal(t, "postgres.example.com", res.Bootstrap.Spec.PrimaryStorage.GetPostgresql().Host)
		assert.True(t, regionutils.IsJoinedRegion(res.Region))
	}

	for name, sec := range map[string]*k8scorev1.Secret{
		"NoBootstrap": func() *k8scorev1.Secret {
			ret := getTestJoinSecret(bs, rgn, "example.com")
			delete(ret.Data, "bootstrap")
			return ret
		}(),
		"InvalidBootstrap": func() *k8scorev1.Secret {
			ret := getTestJoinSecret(bs, rgn, "example.com")
			ret.Data["bootstrap"] = []byte("invalid")
			return ret
		}(),
		"NilBootstrapSpec": getTestJoinSecret(&cbootstrapv1.Config{}, rgn, "example.com"),
		"NoRegion": func() *k8scorev1.Secret {
			ret := getTestJoinSecret(bs, rgn, "example.com")
			delete(ret.Data, "region")
			return ret
		}(),
		"NoRegionName":  getTestJoinSecret(bs, &corev1.Region{Status: &corev1.Region_Status{Index: 1}}, "example.com"),
		"DefaultRegion": getTestJoinSecret(bs, getTestJoinRegion(t, "default", 1), "example.com"),
		"NoIndex":       getTestJoinSecret(bs, getTestJoinRegion(t, "west", 0), "example.com"),
		"NoDomain":      getTestJoinSecret(bs, rgn, ""),
	} {
		_, err := parseJoinResources(sec)
		assert.Error(t, err, name)
	}
}

func TestInitJoinStorage(t *testing.T) {
	ctx := context.Background()

	{
		g := &Genesis{k8sC: fake.NewClientset()}
		for range 2 {
			require.NoError(t, g.initJoinStorage(ctx, getTestJoinBootstrap()))
		}

		pg, err := g.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-postgres", k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"postgres-password": []byte("pg-password"),
			"username":          []byte("octelium"),
			"host":              []byte("postgres.example.com"),
			"database":          []byte("octelium"),
			"port":              []byte("5432"),
			"no_ssl":            []byte("false"),
		}, pg.Data)

		redis, err := g.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-redis", k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"password": []byte("redis-password"),
			"username": []byte("default"),
			"host":     []byte("redis.example.com"),
			"database": []byte("2"),
			"port":     []byte("6379"),
			"use_tls":  []byte("false"),
		}, redis.Data)
	}

	{
		g := &Genesis{k8sC: fake.NewClientset(&k8scorev1.Secret{
			ObjectMeta: k8smetav1.ObjectMeta{Name: "storage", Namespace: "default"},
			Data: map[string][]byte{
				"pg":       []byte("pg-from-secret"),
				"password": []byte("redis-from-secret"),
			},
		})}

		bs := getTestJoinBootstrap()
		bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_{
			PasswordFromSecret: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret{
				Name: "storage",
				Key:  "pg",
			},
		}
		bs.Spec.SecondaryStorage.GetRedis().PasswordType = &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_{
			PasswordFromSecret: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret{
				Name: "storage",
			},
		}

		require.NoError(t, g.initJoinStorage(ctx, bs))

		pg, err := g.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-postgres", k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "pg-from-secret", string(pg.Data["postgres-password"]))

		redis, err := g.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-redis", k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "redis-from-secret", string(redis.Data["password"]))
	}

	{
		g := &Genesis{k8sC: fake.NewClientset()}
		bs := getTestJoinBootstrap()
		bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_{
			PasswordFromSecret: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret{
				Name: "missing",
			},
		}
		assert.Error(t, g.initJoinStorage(ctx, bs))

		bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_{
			PasswordFromSecret: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret{},
		}
		assert.Error(t, g.initJoinStorage(ctx, bs))
	}

	for _, bs := range []*cbootstrapv1.Config{
		{Spec: &cbootstrapv1.Config_Spec{}},
		{Spec: &cbootstrapv1.Config_Spec{PrimaryStorage: getTestJoinBootstrap().Spec.PrimaryStorage}},
	} {
		g := &Genesis{k8sC: fake.NewClientset()}
		assert.Error(t, g.initJoinStorage(ctx, bs))
	}
}

func TestJoinInstallationAndNad(t *testing.T) {
	ctx := context.Background()

	{
		ret, err := getInstallationFromBootstrap(getTestJoinBootstrap())
		require.NoError(t, err)
		assert.Nil(t, ret)
	}

	bs := getTestJoinBootstrap()
	bs.Spec.Spiffe = &cbootstrapv1.Config_Spec_SPIFFE{
		Enable:      true,
		TrustDomain: "example.com",
	}
	bs.Spec.Ingress = &cbootstrapv1.Config_Spec_Ingress{
		FrontProxy: &cbootstrapv1.Config_Spec_Ingress_FrontProxy{Enable: true},
	}
	bs.Spec.Cni = &cbootstrapv1.Config_Spec_CNI{
		ConfDirType: &cbootstrapv1.Config_Spec_CNI_ConfDir{ConfDir: "/etc/cni"},
	}

	installation, err := getInstallationFromBootstrap(bs)
	require.NoError(t, err)
	assert.True(t, installation.GetSpiffe().GetEnable())
	assert.Equal(t, "example.com", installation.GetSpiffe().GetTrustDomain())
	assert.True(t, installation.GetIngress().GetFrontProxy().GetEnable())
	assert.Equal(t, "/etc/cni", installation.GetCni().GetConfDir())

	g := &Genesis{
		k8sC: fake.NewClientset(),
		nadC: fakenad.NewSimpleClientset(),
	}

	for range 2 {
		require.NoError(t, g.setInstallationConfigMap(ctx, &corev1.ClusterConfig{
			Status: &corev1.ClusterConfig_Status{
				Installation: installation,
			},
		}))
		require.NoError(t, g.setNadConfig(ctx))
	}

	cm, err := g.k8sC.CoreV1().ConfigMaps(vutils.K8sNS).Get(ctx, installationConfigMapName, k8smetav1.GetOptions{})
	require.NoError(t, err)
	got := &corev1.ClusterConfig_Status_Installation{}
	require.NoError(t, pbutils.UnmarshalJSON([]byte(cm.Data[installationConfigMapKey]), got))
	assert.True(t, pbutils.IsEqual(installation, got))

	_, err = g.nadC.K8sCniCncfIoV1().NetworkAttachmentDefinitions(vutils.K8sNS).Get(ctx, "octelium", k8smetav1.GetOptions{})
	require.NoError(t, err)

	require.NoError(t, g.setInstallationConfigMap(ctx, &corev1.ClusterConfig{
		Status: &corev1.ClusterConfig_Status{},
	}))
}

func assertDeploymentImage(t *testing.T, k8sC kubernetes.Interface, name string, image string) {
	t.Helper()
	dep, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(context.Background(), name, k8smetav1.GetOptions{})
	require.NoError(t, err, name)
	assert.Equal(t, image, dep.Spec.Template.Spec.Containers[0].Image, name)
}

func assertNoDeployment(t *testing.T, k8sC kubernetes.Interface, name string) {
	t.Helper()
	_, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(context.Background(), name, k8smetav1.GetOptions{})
	assert.True(t, k8serr.IsNotFound(err), "%s: %+v", name, err)
}

func TestRunJoin(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OCTELIUM_REGION_NAME", "")
	t.Setenv("OCTELIUM_DOMAIN", "example.com")

	tst, err := otests.Initialize(nil)
	require.NoError(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	cc, err := tst.C.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)

	k8sC := fake.NewClientset(
		&k8scorev1.Namespace{ObjectMeta: k8smetav1.ObjectMeta{Name: vutils.K8sNS, UID: "namespace-uid"}},
		getTestJoinSecret(getTestJoinBootstrap(), getTestJoinRegion(t, "west", 3), cc.Status.Domain),
	)
	g := &Genesis{
		k8sC: k8sC,
		nadC: fakenad.NewSimpleClientset(),
	}

	var created []string
	k8sC.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.GetResource().Resource {
		case "deployments", "daemonsets":
			if obj, err := meta.Accessor(action.(k8stesting.CreateAction).GetObject()); err == nil {
				created = append(created, fmt.Sprintf("%s/%s", action.GetResource().Resource, obj.GetName()))
			}
		}
		return false, nil, nil
	})

	{
		assert.Error(t, g.RunJoin(ctx, nil))
		assert.Error(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: " "}))
	}

	for range 2 {
		require.NoError(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "9.9.9"}))

		rgn, err := tst.C.OcteliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "west"})
		require.NoError(t, err)
		assert.EqualValues(t, 3, rgn.Status.Index)
		assert.True(t, rgn.Metadata.IsSystem)
		assert.Equal(t, "West Region", rgn.Metadata.DisplayName)
		assert.NotEmpty(t, rgn.Status.PublicHostname)
		assert.Equal(t, "9.9.9", rgn.Status.Version)
		assert.Equal(t, "9.9.9", regionutils.GetPackageVersion(rgn, regionutils.PackageCore))
		assert.Equal(t, ldflags.GetVersion(), regionutils.GetPackageVersion(rgn, regionutils.PackageEnterprise))
		assert.True(t, regionutils.IsJoinCompleted(rgn))

		extInfo, err := regionutils.GetRegionExtInfo(rgn)
		require.NoError(t, err)
		assert.Equal(t, "west-kubeconfig", extInfo.Kubernetes.GetKubeconfig().GetFromSecret())

		assertDeploymentImage(t, k8sC, "octelium-rscserver", components.GetImage(components.RscServer, "9.9.9"))
		assertDeploymentImage(t, k8sC, "octelium-nocturne", components.GetImage(components.Nocturne, "9.9.9"))
		assertDeploymentImage(t, k8sC, "octelium-octovigil", components.GetImage(components.Octovigil, "9.9.9"))

		ds, err := k8sC.AppsV1().DaemonSets(vutils.K8sNS).Get(ctx, "octelium-gwagent", k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, components.GetImage(components.GWAgent, "9.9.9"), ds.Spec.Template.Spec.Containers[0].Image)

		for _, component := range []string{oc.RscServer, oc.Nocturne, oc.SecretMan, oc.Collector} {
			assertDeploymentImage(t, k8sC, oc.OcteliumEnterpriseComponent(component), oc.GetImage(component, ""))
		}

		for _, name := range []string{
			"octelium-ingress",
			"octelium-ingress-dataplane",
			oc.OcteliumEnterpriseComponent(oc.ClusterMan),
			oc.OcteliumEnterpriseComponent(oc.CloudMan),
			oc.OcteliumEnterpriseComponent(oc.LogStore),
			oc.OcteliumEnterpriseComponent(oc.RscStore),
			oc.OcteliumEnterpriseComponent(oc.MetricStore),
			oc.OcteliumEnterpriseComponent(oc.PolicyPortal),
		} {
			assertNoDeployment(t, k8sC, name)
		}

		pvcList, err := k8sC.CoreV1().PersistentVolumeClaims(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, pvcList.Items)

		for _, name := range []string{"octelium-postgres", "octelium-redis"} {
			_, err := k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.NoError(t, err, name)
		}

		_, err = k8sC.CoreV1().ConfigMaps(vutils.K8sNS).Get(ctx, installationConfigMapName, k8smetav1.GetOptions{})
		require.NoError(t, err)

		for _, name := range []string{"console.octelium", collectorutils.ServiceName} {
			_, err = tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: name})
			assert.Error(t, err, name)
		}
	}

	{
		createdAt := func(name string) int {
			idx := slices.Index(created, name)
			require.GreaterOrEqual(t, idx, 0, "%s was never created: %v", name, created)
			return idx
		}

		rscServer := createdAt("deployments/octelium-rscserver")
		eeRscServer := createdAt("deployments/" + oc.OcteliumEnterpriseComponent(oc.RscServer))
		secretMan := createdAt("deployments/" + oc.OcteliumEnterpriseComponent(oc.SecretMan))

		assert.Less(t, rscServer, eeRscServer)
		assert.Less(t, eeRscServer, secretMan)

		for _, name := range []string{
			"daemonsets/octelium-gwagent",
			"deployments/octelium-nocturne",
			"deployments/octelium-octovigil",
			"deployments/" + oc.OcteliumEnterpriseComponent(oc.Nocturne),
			"deployments/" + oc.OcteliumEnterpriseComponent(oc.Collector),
		} {
			assert.Less(t, secretMan, createdAt(name), name)
		}
	}

	{
		k8sC := fake.NewClientset(getTestJoinSecret(getTestJoinBootstrap(), getTestJoinRegion(t, "west", 4), cc.Status.Domain))
		g := &Genesis{
			k8sC: k8sC,
			nadC: fakenad.NewSimpleClientset(),
		}
		assert.Error(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "9.9.9"}))
	}

	{
		k8sC := fake.NewClientset(getTestJoinSecret(getTestJoinBootstrap(), getTestJoinRegion(t, "east", 5), "other.example.com"))
		g := &Genesis{
			k8sC: k8sC,
			nadC: fakenad.NewSimpleClientset(),
		}
		assert.Error(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "9.9.9"}))

		_, err := tst.C.OcteliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "east"})
		assert.Error(t, err)
	}

	{
		g := &Genesis{
			k8sC: fake.NewClientset(),
			nadC: fakenad.NewSimpleClientset(),
		}
		assert.Error(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "9.9.9"}))
	}
	for _, name := range []string{"west", "deleted-region"} {
		requested := getTestJoinRegion(t, name, 3)
		requested.Metadata.Uid = "stale-region-uid"
		client := fake.NewClientset(getTestJoinSecret(getTestJoinBootstrap(), requested, cc.Status.Domain))
		worker := &Genesis{k8sC: client, nadC: fakenad.NewSimpleClientset()}
		require.ErrorContains(t, worker.RunJoin(ctx, &JoinOpts{CoreVersion: "9.9.9"}), "bootstrap Region")
		for _, action := range client.Actions() {
			if action.GetResource().Resource == "nodes" {
				assert.NotEqual(t, "update", action.GetVerb())
			}
		}
	}

}

func TestSetupGuardsNonDefaultRegionComponents(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OCTELIUM_REGION_NAME", "")
	t.Setenv("OCTELIUM_DOMAIN", "example.com")

	tst, err := otests.Initialize(nil)
	require.NoError(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	rgn, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, &corev1.Region{
		Metadata: &metav1.Metadata{Name: "west"},
		Spec:     &corev1.Region_Spec{},
		Status:   &corev1.Region_Status{Index: 1},
	})
	require.NoError(t, err)

	t.Setenv("OCTELIUM_REGION_NAME", rgn.Metadata.Name)

	k8sC := fake.NewClientset()
	g := &Genesis{
		k8sC:          k8sC,
		octeliumCInit: tst.C.OcteliumC,
	}

	require.NoError(t, g.RunInit(ctx, nil))

	assertNoDeployment(t, k8sC, oc.OcteliumEnterpriseComponent(oc.ClusterMan))
	assertNoDeployment(t, k8sC, oc.OcteliumEnterpriseComponent(oc.LogStore))
	assertDeploymentImage(t, k8sC, oc.OcteliumEnterpriseComponent(oc.SecretMan), oc.GetImage(oc.SecretMan, ""))

	pvcList, err := k8sC.CoreV1().PersistentVolumeClaims(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, pvcList.Items)

	require.NoError(t, g.RunUpgrade(ctx, nil))

	pvcList, err = k8sC.CoreV1().PersistentVolumeClaims(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, pvcList.Items)

	rgn, err = tst.C.OcteliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: rgn.Metadata.Name})
	require.NoError(t, err)
	assert.Equal(t, ldflags.GetVersion(), regionutils.GetPackageVersion(rgn, regionutils.PackageEnterprise))

	_, err = tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: collectorutils.ServiceName})
	assert.Error(t, err)
}

func TestJoinTargetRecordingPreservesInstallation(t *testing.T) {
	for _, failure := range []string{"none", "conflict", "before-commit", "after-commit"} {
		t.Run(failure, func(t *testing.T) {
			g, cc, region, c := newPackageTarget(t)
			updates := 0
			c.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				if updates > 1 || failure == "none" {
					return false, nil, nil
				}
				if failure == "conflict" {
					return true, nil, k8serr.NewConflict(action.GetResource().GroupResource(), "octelium-installation", io.EOF)
				}
				if failure == "after-commit" {
					require.NoError(t, c.Tracker().Update(action.GetResource(), action.(k8stesting.UpdateAction).GetObject(), action.GetNamespace()))
				}
				return true, nil, io.ErrUnexpectedEOF
			})
			require.NoError(t, g.recordRegionTarget(t.Context(), cc, region))
			cm, err := c.CoreV1().ConfigMaps(vutils.K8sNS).Get(t.Context(), "octelium-installation", k8smetav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"custom": "preserved"}, cm.Data)
			assert.Equal(t, packageTargetAnnotations(cc, region), cm.Annotations)
			if failure == "after-commit" {
				assert.Equal(t, 1, updates)
			}
			c.ClearActions()
			require.NoError(t, g.recordRegionTarget(t.Context(), cc, region))
			assertPackageTargetReadOnly(t, c)
		})
	}
	g, cc, region, c := newPackageTarget(t)
	cc.Metadata.Uid = ""
	require.Error(t, g.recordRegionTarget(t.Context(), cc, region))
	assertPackageTargetReadOnly(t, c)
}

func TestJoinStorageNeverOverwritesExistingCredentials(t *testing.T) {
	for _, name := range []string{"octelium-postgres", "octelium-redis"} {
		for _, failure := range []string{"matching", "different", "lost-acknowledgement"} {
			t.Run(name+"/"+failure, func(t *testing.T) {
				c := fake.NewSimpleClientset()
				g := &Genesis{k8sC: c}
				if failure != "lost-acknowledgement" {
					require.NoError(t, g.initJoinStorage(t.Context(), getTestJoinBootstrap()))
					secret, err := c.CoreV1().Secrets(vutils.K8sNS).Get(t.Context(), name, k8smetav1.GetOptions{})
					require.NoError(t, err)
					secret.Data["custom"] = []byte("preserved")
					if failure == "different" {
						secret.Data["host"] = []byte("existing-storage")
					}
					require.NoError(t, c.Tracker().Update(k8scorev1.SchemeGroupVersion.WithResource("secrets"), secret, vutils.K8sNS))
					c.ClearActions()
				} else {
					failed := false
					c.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
						secret := action.(k8stesting.CreateAction).GetObject().(*k8scorev1.Secret)
						if failed || secret.Name != name {
							return false, nil, nil
						}
						failed = true
						require.NoError(t, c.Tracker().Create(action.GetResource(), secret, action.GetNamespace()))
						return true, nil, io.ErrUnexpectedEOF
					})
				}
				err := g.initJoinStorage(t.Context(), getTestJoinBootstrap())
				if failure == "different" {
					require.ErrorContains(t, err, "Refusing to replace")
				} else {
					require.NoError(t, err)
				}
				secret, err := c.CoreV1().Secrets(vutils.K8sNS).Get(t.Context(), name, k8smetav1.GetOptions{})
				require.NoError(t, err)
				if failure == "different" {
					assert.Equal(t, "existing-storage", string(secret.Data["host"]))
				}
				if failure != "lost-acknowledgement" {
					assert.Equal(t, "preserved", string(secret.Data["custom"]))
				}
				for _, action := range c.Actions() {
					assert.NotContains(t, []string{"update", "delete", "patch"}, action.GetVerb())
				}
			})
		}
	}
}

func TestJoinPreservesRegisteredNodes(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, state := range []string{"registered", "new", "already-tainted", "registration-conflict", "lost-acknowledgement"} {
		t.Run(state, func(t *testing.T) {
			node := &k8scorev1.Node{ObjectMeta: k8smetav1.ObjectMeta{Name: "gateway", Labels: map[string]string{vutils.NodeLabelDataPlane: "", "custom": "preserved"}}, Spec: k8scorev1.NodeSpec{Taints: []k8scorev1.Taint{{Key: "custom", Effect: k8scorev1.TaintEffectNoSchedule}}}}
			if state == "registered" {
				node.Labels[vutils.NodeLabelGatewayRegistered] = "true"
			}
			if state == "already-tainted" {
				node.Spec.Taints = append(node.Spec.Taints, k8scorev1.Taint{Key: vutils.NodeTaintGatewayInit, Value: "true", Effect: k8scorev1.TaintEffectNoSchedule})
			}
			c := fake.NewSimpleClientset(node)
			g := &Genesis{k8sC: c}
			updates := 0
			c.PrependReactor("update", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updates++
				if updates > 1 {
					return false, nil, nil
				}
				if state == "registration-conflict" {
					registered := node.DeepCopy()
					registered.Labels[vutils.NodeLabelGatewayRegistered] = "true"
					require.NoError(t, c.Tracker().Update(action.GetResource(), registered, ""))
					return true, nil, k8serr.NewConflict(action.GetResource().GroupResource(), node.Name, io.EOF)
				}
				if state == "lost-acknowledgement" {
					require.NoError(t, c.Tracker().Update(action.GetResource(), action.(k8stesting.UpdateAction).GetObject(), ""))
					return true, nil, io.ErrUnexpectedEOF
				}
				return false, nil, nil
			})
			require.NoError(t, g.setDataPlaneNodeTaints(t.Context()))
			updated, err := c.CoreV1().Nodes().Get(t.Context(), node.Name, k8smetav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, "preserved", updated.Labels["custom"])
			assert.Equal(t, node.Spec.Taints[0], updated.Spec.Taints[0])
			if state == "registered" || state == "registration-conflict" {
				assert.Equal(t, "true", updated.Labels[vutils.NodeLabelGatewayRegistered])
				assert.Equal(t, node.Spec.Taints, updated.Spec.Taints)
			} else {
				assert.Len(t, updated.Spec.Taints, 2)
			}
			if state == "registered" || state == "already-tainted" {
				assert.Zero(t, updates)
			}
			if state == "registration-conflict" || state == "lost-acknowledgement" {
				assert.Equal(t, 1, updates)
			}
		})
	}
}

func TestJoinCancelsBeforeMutation(t *testing.T) {
	g := &Genesis{k8sC: fake.NewSimpleClientset()}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, g.RunInit(ctx, nil), context.Canceled)
	require.ErrorIs(t, g.RunUpgrade(ctx, nil), context.Canceled)
	require.ErrorIs(t, g.RunJoin(ctx, &JoinOpts{CoreVersion: "2.0.0"}), context.Canceled)
	assert.Empty(t, g.k8sC.(*fake.Clientset).Actions())
	ctx, cancel = context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	g.k8sC.(*fake.Clientset).PrependReactor("create", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, io.EOF })
	g.k8sC.(*fake.Clientset).PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, io.EOF })
	require.ErrorIs(t, g.initJoinStorage(ctx, getTestJoinBootstrap()), context.DeadlineExceeded)
}

func TestJoinRejectsPasswordReferencesBeforeMutation(t *testing.T) {
	for _, storage := range []string{"postgres", "redis"} {
		t.Run(storage, func(t *testing.T) {
			resetPasswordEnvironment(t)
			ldflags.TestMode = "true"
			t.Setenv("OCTELIUM_DOMAIN", "example.com")
			t.Setenv("OCTELIUM_REGION_NAME", "")
			bs := getTestJoinBootstrap()
			setPasswordReference(bs, storage, "credentials", "default", "missing")
			c := fake.NewSimpleClientset(
				getTestJoinSecret(bs, getTestJoinRegion(t, "west", 1), "example.com"),
				&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"password": []byte("wrong-fallback")}},
			)
			g := &Genesis{k8sC: c}
			require.ErrorContains(t, g.RunJoin(t.Context(), &JoinOpts{CoreVersion: "9.9.9"}), "missing or empty")
			assertPasswordResolutionReadOnly(t, c)
		})
	}
}

func newPasswordBootstrap(t *testing.T) *cbootstrapv1.Config {
	t.Helper()
	bs := &cbootstrapv1.Config{}
	require.NoError(t, pbutils.UnmarshalYAML([]byte("spec:\n  primaryStorage:\n    postgresql: {host: postgres, username: installer, password: inline-postgres}\n  secondaryStorage:\n    redis: {host: redis, password: inline-redis}\n"), bs))
	return bs
}

func setPasswordReference(bs *cbootstrapv1.Config, storage, name, namespace, key string) {
	if storage == "postgres" {
		bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_{PasswordFromSecret: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret{Name: name, Namespace: namespace, Key: key}}
	} else {
		bs.Spec.SecondaryStorage.GetRedis().PasswordType = &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_{PasswordFromSecret: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret{Name: name, Namespace: namespace, Key: key}}
	}
}

func initializePasswordStorage(ctx context.Context, g *Genesis, bs *cbootstrapv1.Config) error {
	return g.initJoinStorage(ctx, bs)
}

func resetPasswordEnvironment(t *testing.T) {
	t.Helper()
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, prefix := range []string{"OCTELIUM_POSTGRES_", "OCTELIUM_REDIS_"} {
		for _, suffix := range []string{"PASSWORD", "USERNAME", "HOST", "PORT", "DATABASE", "NOSSL", "USE_TLS"} {
			t.Setenv(prefix+suffix, "unchanged")
		}
	}
}

func assertPasswordResolutionReadOnly(t *testing.T, c *fake.Clientset) {
	t.Helper()
	for _, action := range c.Actions() {
		assert.Contains(t, []string{"get", "list"}, action.GetVerb())
	}
	assert.Equal(t, "unchanged", os.Getenv("OCTELIUM_POSTGRES_PASSWORD"))
	assert.Equal(t, "unchanged", os.Getenv("OCTELIUM_REDIS_PASSWORD"))
}

func TestStoragePasswordSelection(t *testing.T) {
	for _, storage := range []string{"postgres", "redis"} {
		for _, tc := range []struct {
			name, key string
			data      map[string][]byte
			want      string
		}{
			{name: "password-default", data: map[string][]byte{"password": []byte("password-value")}, want: "password-value"},
			{name: "data-default", data: map[string][]byte{"data": []byte("data-value")}, want: "data-value"},
			{name: "legacy-default-order", data: map[string][]byte{"password": []byte("password-value"), "data": []byte("data-value")}, want: "password-value"},
			{name: "legacy-empty-password-fallback", data: map[string][]byte{"password": {}, "data": []byte("data-value")}, want: "data-value"},
			{name: "explicit-custom", key: "custom", data: map[string][]byte{"custom": []byte("custom-value"), "password": []byte("fallback-value"), "data": []byte("other-value")}, want: "custom-value"},
			{name: "explicit-data", key: "data", data: map[string][]byte{"password": []byte("password-value"), "data": []byte("data-value")}, want: "data-value"},
			{name: "explicit-missing", key: "custom", data: map[string][]byte{"password": []byte("fallback-value"), "data": []byte("other-value")}},
			{name: "explicit-empty", key: "custom", data: map[string][]byte{"custom": {}, "password": []byte("fallback-value"), "data": []byte("other-value")}},
			{name: "explicit-nil", key: "custom", data: map[string][]byte{"custom": nil, "password": []byte("fallback-value")}},
			{name: "explicit-password-empty", key: "password", data: map[string][]byte{"password": {}, "data": []byte("fallback-value")}},
			{name: "missing-defaults", data: map[string][]byte{"custom": []byte("custom-value")}},
			{name: "empty-defaults", data: map[string][]byte{"password": {}, "data": nil}},
			{name: "no-data"},
			{name: "exact-whitespace", key: "custom", data: map[string][]byte{"custom": []byte(" \n\t")}, want: " \n\t"},
			{name: "case-sensitive-key", key: "PASSWORD", data: map[string][]byte{"PASSWORD": []byte("upper-value"), "password": []byte("lower-value")}, want: "upper-value"},
		} {
			t.Run(storage+"/"+tc.name, func(t *testing.T) {
				resetPasswordEnvironment(t)
				bs := newPasswordBootstrap(t)
				setPasswordReference(bs, storage, "credentials", "", tc.key)
				c := fake.NewSimpleClientset(&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: tc.data})
				g := &Genesis{k8sC: c}
				err := initializePasswordStorage(t.Context(), g, bs)
				if tc.want == "" {
					require.Error(t, err)
					for _, value := range tc.data {
						if len(value) > 0 {
							assert.NotContains(t, err.Error(), string(value))
						}
					}
					assertPasswordResolutionReadOnly(t, c)
					return
				}
				require.NoError(t, err)
				name, key := "octelium-redis", "password"
				if storage == "postgres" {
					name, key = "octelium-postgres", "postgres-password"
				}
				secret, err := c.CoreV1().Secrets("octelium").Get(t.Context(), name, k8smetav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.want, string(secret.Data[key]))
				source, err := c.CoreV1().Secrets("default").Get(t.Context(), "credentials", k8smetav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.data, source.Data)
			})
		}
	}
}

func TestStoragePasswordReferenceValidation(t *testing.T) {
	for _, tc := range []struct {
		name, namespace, key string
		valid                bool
	}{
		{name: "credentials", valid: true},
		{name: "credentials.v2", namespace: "storage", key: "KEY-name.value_1", valid: true},
		{name: strings.Repeat("a", 253), namespace: strings.Repeat("a", 63), key: strings.Repeat("a", 253), valid: true},
		{name: ""}, {name: "Credentials"}, {name: "a/b"}, {name: " credentials"}, {name: strings.Repeat("a", 254)},
		{name: "credentials", namespace: "a.b"}, {name: "credentials", namespace: "Storage"}, {name: "credentials", namespace: "a/b"}, {name: "credentials", namespace: strings.Repeat("a", 64)},
		{name: "credentials", key: "a/b"}, {name: "credentials", key: "bad key"}, {name: "credentials", key: ".."}, {name: "credentials", key: strings.Repeat("a", 254)},
	} {
		t.Run(fmt.Sprintf("%s/%s/%s", tc.name, tc.namespace, tc.key), func(t *testing.T) {
			err := validatePasswordSecretReference(tc.name, tc.namespace, tc.key)
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, storage := range []string{"postgres", "redis"} {
				resetPasswordEnvironment(t)
				bs := newPasswordBootstrap(t)
				setPasswordReference(bs, storage, tc.name, tc.namespace, tc.key)
				c := fake.NewSimpleClientset()
				require.Error(t, initializePasswordStorage(t.Context(), &Genesis{k8sC: c}, bs))
				assert.Empty(t, c.Actions())
				assertPasswordResolutionReadOnly(t, c)
			}
		})
	}
}

func TestStoragePasswordsUseOneSnapshotPerSecret(t *testing.T) {
	resetPasswordEnvironment(t)
	bs := newPasswordBootstrap(t)
	setPasswordReference(bs, "postgres", "credentials", "", "pg")
	setPasswordReference(bs, "redis", "credentials", "default", "redis")
	c := fake.NewSimpleClientset()
	calls := 0
	c.PrependReactor("get", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		assert.Equal(t, "default", action.GetNamespace())
		return true, &k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"pg": []byte(fmt.Sprintf("pg-%d", calls)), "redis": []byte(fmt.Sprintf("redis-%d", calls))}}, nil
	})
	g := &Genesis{k8sC: c}
	for i := 1; i <= 2; i++ {
		passwords, err := g.resolveStoragePasswords(t.Context(), bs)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("pg-%d", i), passwords.postgres)
		assert.Equal(t, fmt.Sprintf("redis-%d", i), passwords.redis)
		assert.Equal(t, i, calls)
	}
	assertPasswordResolutionReadOnly(t, c)
}

func TestStoragePasswordReferenceNamespaces(t *testing.T) {
	resetPasswordEnvironment(t)
	bs := newPasswordBootstrap(t)
	setPasswordReference(bs, "postgres", "credentials", "", "password")
	setPasswordReference(bs, "redis", "credentials", "storage", "password")
	c := fake.NewSimpleClientset(
		&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"password": []byte("pg-value")}},
		&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "storage"}, Data: map[string][]byte{"password": []byte("redis-value")}},
	)
	passwords, err := (&Genesis{k8sC: c}).resolveStoragePasswords(t.Context(), bs)
	require.NoError(t, err)
	assert.Equal(t, "pg-value", passwords.postgres)
	assert.Equal(t, "redis-value", passwords.redis)
	assert.Len(t, c.Actions(), 2)
	assertPasswordResolutionReadOnly(t, c)
}

func TestStoragePasswordReadFailures(t *testing.T) {
	for _, storage := range []string{"postgres", "redis"} {
		for _, failure := range []string{"transient", "forbidden", "not-found", "canceled", "deadline"} {
			t.Run(storage+"/"+failure, func(t *testing.T) {
				resetPasswordEnvironment(t)
				bs := newPasswordBootstrap(t)
				setPasswordReference(bs, storage, "credentials", "default", "password")
				c := fake.NewSimpleClientset(&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"password": []byte("secret-value")}})
				calls := 0
				c.PrependReactor("get", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
					calls++
					switch failure {
					case "transient":
						if calls == 1 {
							return true, nil, io.ErrUnexpectedEOF
						}
					case "forbidden":
						return true, nil, k8serr.NewForbidden(action.GetResource().GroupResource(), "credentials", io.EOF)
					case "not-found":
						return true, nil, k8serr.NewNotFound(action.GetResource().GroupResource(), "credentials")
					case "deadline":
						return true, nil, k8serr.NewServiceUnavailable("temporary outage")
					}
					return false, nil, nil
				})
				g := &Genesis{k8sC: c}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if failure == "canceled" {
					cancel()
				}
				if failure == "deadline" {
					ctx, cancel = context.WithTimeout(t.Context(), 30*time.Millisecond)
					defer cancel()
				}
				err := initializePasswordStorage(ctx, g, bs)
				switch failure {
				case "transient":
					require.NoError(t, err)
					assert.Equal(t, 2, calls)
					return
				case "forbidden":
					require.True(t, k8serr.IsForbidden(err))
					assert.Equal(t, 1, calls)
				case "not-found":
					require.True(t, k8serr.IsNotFound(err))
					assert.Equal(t, 1, calls)
				case "canceled":
					require.ErrorIs(t, err, context.Canceled)
					assert.Zero(t, calls)
				case "deadline":
					require.ErrorIs(t, err, context.DeadlineExceeded)
					assert.Equal(t, 1, calls)
				}
				assertPasswordResolutionReadOnly(t, c)
			})
		}
	}
}

func TestStoragePasswordsPreserveLegacyInlineInputs(t *testing.T) {
	for _, omitted := range []bool{false, true} {
		t.Run(fmt.Sprint(omitted), func(t *testing.T) {
			resetPasswordEnvironment(t)
			bs := newPasswordBootstrap(t)
			bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_Password{}
			bs.Spec.SecondaryStorage.GetRedis().PasswordType = &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_Password{}
			if omitted {
				bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = nil
				bs.Spec.SecondaryStorage.GetRedis().PasswordType = nil
			}
			c := fake.NewSimpleClientset()
			passwords, err := (&Genesis{k8sC: c}).resolveStoragePasswords(t.Context(), bs)
			require.NoError(t, err)
			assert.Equal(t, &storagePasswords{}, passwords)
			assert.Empty(t, c.Actions())
			assertPasswordResolutionReadOnly(t, c)
		})
	}
}

func TestStoragePasswordNilReferences(t *testing.T) {
	for _, storage := range []string{"postgres", "redis"} {
		for _, wrapper := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrapper=%t", storage, wrapper), func(t *testing.T) {
				resetPasswordEnvironment(t)
				bs := newPasswordBootstrap(t)
				if storage == "postgres" {
					if wrapper {
						bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = (*cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_)(nil)
					} else {
						bs.Spec.PrimaryStorage.GetPostgresql().PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_{}
					}
				} else {
					if wrapper {
						bs.Spec.SecondaryStorage.GetRedis().PasswordType = (*cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_)(nil)
					} else {
						bs.Spec.SecondaryStorage.GetRedis().PasswordType = &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_{}
					}
				}
				c := fake.NewSimpleClientset()
				require.Error(t, initializePasswordStorage(t.Context(), &Genesis{k8sC: c}, bs))
				assert.Empty(t, c.Actions())
				assertPasswordResolutionReadOnly(t, c)
			})
		}
	}
}

func TestStoragePasswordHTTPRecovery(t *testing.T) {
	for _, failure := range []string{"disconnect", "unavailable", "missing-key", "forbidden"} {
		t.Run(failure, func(t *testing.T) {
			var reads, unexpected atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/default/secrets/credentials" {
					unexpected.Add(1)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				n := reads.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if n == 1 && failure == "disconnect" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						conn.Close()
					}
					return
				}
				if (n == 1 && failure == "unavailable") || failure == "forbidden" {
					code, reason := http.StatusServiceUnavailable, k8smetav1.StatusReasonServiceUnavailable
					if failure == "forbidden" {
						code, reason = http.StatusForbidden, k8smetav1.StatusReasonForbidden
					}
					w.WriteHeader(code)
					assert.NoError(t, json.NewEncoder(w).Encode(&k8smetav1.Status{TypeMeta: k8smetav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: "Failure", Code: int32(code), Reason: reason}))
					return
				}
				data := map[string][]byte{"password": []byte("fallback-value"), "pg": []byte("pg-value"), "redis": []byte("redis-value")}
				if failure == "missing-key" {
					delete(data, "redis")
				}
				assert.NoError(t, json.NewEncoder(w).Encode(&k8scorev1.Secret{TypeMeta: k8smetav1.TypeMeta{Kind: "Secret", APIVersion: "v1"}, ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: data}))
			}))
			defer server.Close()
			c, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, QPS: -1})
			require.NoError(t, err)
			bs := newPasswordBootstrap(t)
			setPasswordReference(bs, "postgres", "credentials", "default", "pg")
			setPasswordReference(bs, "redis", "credentials", "", "redis")
			passwords, err := (&Genesis{k8sC: c}).resolveStoragePasswords(t.Context(), bs)
			if failure == "forbidden" {
				require.True(t, k8serr.IsForbidden(err))
				assert.Equal(t, int32(1), reads.Load())
			} else if failure == "missing-key" {
				require.ErrorContains(t, err, "missing or empty")
				assert.Equal(t, int32(1), reads.Load())
			} else {
				require.NoError(t, err)
				assert.Equal(t, &storagePasswords{postgres: "pg-value", redis: "redis-value"}, passwords)
				assert.Equal(t, int32(2), reads.Load())
			}
			assert.Zero(t, unexpected.Load())
		})
	}
}

func TestStoragePasswordHTTPStallCancellation(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(canceled) }))
	defer server.Close()
	c, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, QPS: -1})
	require.NoError(t, err)
	bs := newPasswordBootstrap(t)
	setPasswordReference(bs, "redis", "credentials", "default", "password")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := (&Genesis{k8sC: c}).resolveStoragePasswords(ctx, bs); result <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("Secret read did not start")
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("Secret read did not stop")
	}
	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP request was not canceled")
	}
}

func TestPasswordResolutionFailurePreservesExistingStorageSecrets(t *testing.T) {
	resetPasswordEnvironment(t)
	bs := newPasswordBootstrap(t)
	setPasswordReference(bs, "redis", "credentials", "default", "missing")
	pg := &k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-postgres", Namespace: "octelium", UID: "preserved-postgres", Annotations: map[string]string{"custom": "preserved"}}, Data: map[string][]byte{"postgres-password": []byte("stored-postgres"), "custom": []byte("preserved")}}
	redis := &k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-redis", Namespace: "octelium", UID: "preserved-redis"}, Data: map[string][]byte{"password": []byte("stored-redis")}}
	c := fake.NewSimpleClientset(pg, redis, &k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"password": []byte("fallback-value")}})
	require.Error(t, initializePasswordStorage(t.Context(), &Genesis{k8sC: c}, bs))
	assertPasswordResolutionReadOnly(t, c)
	for _, before := range []*k8scorev1.Secret{pg, redis} {
		stored, err := c.Tracker().Get(k8scorev1.SchemeGroupVersion.WithResource("secrets"), "octelium", before.Name)
		require.NoError(t, err)
		assert.Equal(t, before, stored)
	}
}
