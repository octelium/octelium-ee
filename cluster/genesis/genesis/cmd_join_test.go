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
	"testing"

	fakenad "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/fake"
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
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

	tst, err := otests.Initialize(nil)
	require.NoError(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	cc, err := tst.C.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)

	k8sC := fake.NewClientset(getTestJoinSecret(getTestJoinBootstrap(), getTestJoinRegion(t, "west", 3), cc.Status.Domain))
	g := &Genesis{
		k8sC: k8sC,
		nadC: fakenad.NewSimpleClientset(),
	}

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

		_, err = tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: "console.octelium"})
		assert.Error(t, err)
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
}

func TestSetupGuardsNonDefaultRegionComponents(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OCTELIUM_REGION_NAME", "")

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
}
