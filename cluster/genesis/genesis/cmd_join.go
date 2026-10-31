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
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	iocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const joinInitSecretName = "octelium-init"
const installationConfigMapName = "octelium-installation"
const installationConfigMapKey = "installation.json"

type JoinOpts struct {
	CoreVersion string
}

type joinResources struct {
	Bootstrap *cbootstrapv1.Config
	Region    *corev1.Region
	Domain    string
}

func (g *Genesis) RunJoin(ctx context.Context, o *JoinOpts) error {
	zap.L().Info("Starting joining the Region to the Cluster")

	if o == nil {
		o = &JoinOpts{}
	}

	coreVersion := strings.TrimSpace(o.CoreVersion)
	if coreVersion == "" {
		return errors.Errorf("The core package version is not set")
	}

	if err := k8sutils.WaitForNodesReadiness(ctx, g.k8sC); err != nil {
		return err
	}

	if err := g.setDataPlaneNodeTaints(ctx); err != nil {
		return err
	}

	res, err := g.loadJoinResources(ctx)
	if err != nil {
		return err
	}

	regionName := res.Region.Metadata.Name
	os.Setenv("OCTELIUM_REGION_NAME", regionName)

	zap.L().Debug("Loaded join resources",
		zap.String("region", regionName),
		zap.Int32("index", res.Region.Status.Index),
		zap.String("coreVersion", coreVersion))

	if err := g.initJoinStorage(ctx, res.Bootstrap); err != nil {
		return err
	}

	installation, err := getInstallationFromBootstrap(res.Bootstrap)
	if err != nil {
		return err
	}

	coreK8sC := newCoreK8sC(g.k8sC, coreVersion)

	if err := gc.CreateRscServer(ctx, &gc.CommonOpts{
		K8sC: coreK8sC,
		ClusterConfig: &corev1.ClusterConfig{
			Status: &corev1.ClusterConfig_Status{
				Installation: installation,
			},
		},
	}); err != nil {
		return err
	}

	zap.L().Debug("Waiting for readiness of the core rscServer")

	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, "octelium-rscserver"); err != nil {
		return err
	}

	octeliumCInit, err := iocteliumc.NewClient(ctx)
	if err != nil {
		return err
	}
	g.octeliumCInit = octeliumCInit

	clusterCfg, err := octeliumCInit.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	if clusterCfg.Status.Domain != res.Domain {
		return errors.Errorf("The Cluster domain %s does not match the join domain %s",
			clusterCfg.Status.Domain, res.Domain)
	}

	rgn, err := g.createOrUpdateJoinRegion(ctx, res.Region)
	if err != nil {
		return err
	}

	if err := g.setInstallationConfigMap(ctx, clusterCfg); err != nil {
		return err
	}

	if err := g.setNadConfig(ctx); err != nil {
		return err
	}

	if err := g.installJoinCoreComponents(ctx, &gc.CommonOpts{
		OcteliumC:     octeliumCInit,
		K8sC:          coreK8sC,
		ClusterConfig: clusterCfg,
		Region:        rgn,
	}); err != nil {
		return err
	}

	rgn, err = g.setRegionCoreVersion(ctx, rgn, coreVersion)
	if err != nil {
		return err
	}

	if err := g.installComponents(ctx, &components.CommonOpts{
		CommonOpts: gc.CommonOpts{
			K8sC:          g.k8sC,
			ClusterConfig: clusterCfg,
			Region:        rgn,
		},
	}, false); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}
	g.octeliumC = octeliumC

	if err := g.installOcteliumResources(ctx, clusterCfg, rgn); err != nil {
		return err
	}

	if err := g.setRegionVersionMap(ctx, rgn); err != nil {
		return err
	}

	zap.L().Info("Successfully joined the Region to the Cluster", zap.String("region", regionName))

	return nil
}

func (g *Genesis) loadJoinResources(ctx context.Context) (*joinResources, error) {
	sec, err := g.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, joinInitSecretName, k8smetav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return parseJoinResources(sec)
}

func parseJoinResources(sec *k8scorev1.Secret) (*joinResources, error) {
	ret := &joinResources{}

	if sec.Data["bootstrap"] == nil {
		return nil, errors.Errorf("Could not find bootstrap field")
	}

	ret.Bootstrap = &cbootstrapv1.Config{}
	if err := pbutils.Unmarshal(sec.Data["bootstrap"], ret.Bootstrap); err != nil {
		return nil, err
	}

	if ret.Bootstrap.Spec == nil {
		return nil, errors.Errorf("Nil bootstrap spec")
	}

	if sec.Data["region"] == nil {
		return nil, errors.Errorf("Could not find region field")
	}

	ret.Region = &corev1.Region{}
	if err := pbutils.Unmarshal(sec.Data["region"], ret.Region); err != nil {
		return nil, err
	}

	if ret.Region.Metadata == nil || ret.Region.Metadata.Name == "" {
		return nil, errors.Errorf("The join Region has no name")
	}

	if ret.Region.Metadata.Name == regionutils.DefaultRegion {
		return nil, errors.Errorf("The default Region cannot be joined")
	}

	if ret.Region.Status == nil || ret.Region.Status.Index < 1 {
		return nil, errors.Errorf("The join Region has an invalid index")
	}

	ret.Domain = string(sec.Data["domain"])
	if ret.Domain == "" {
		return nil, errors.Errorf("Could not find domain field")
	}

	return ret, nil
}

func (g *Genesis) createOrUpdateJoinRegion(ctx context.Context, req *corev1.Region) (*corev1.Region, error) {
	extInfo, err := regionutils.GetRegionExtInfo(req)
	if err != nil {
		return nil, err
	}

	rgn, err := g.octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: req.Metadata.Name,
	})
	if err == nil {
		if rgn.Status.Index != req.Status.Index {
			return nil, errors.Errorf("The Region %s already exists with a different index %d",
				req.Metadata.Name, rgn.Status.Index)
		}

		zap.L().Debug("The Region already exists. Updating it", zap.String("region", req.Metadata.Name))

		rgn.Metadata.DisplayName = req.Metadata.DisplayName
		if err := regionutils.SetRegionExtInfo(rgn, extInfo); err != nil {
			return nil, err
		}
	} else if grpcerr.IsNotFound(err) {
		zap.L().Debug("Creating the Region", zap.String("region", req.Metadata.Name))

		rgn = &corev1.Region{
			Metadata: &metav1.Metadata{
				Name:        req.Metadata.Name,
				DisplayName: req.Metadata.DisplayName,
				IsSystem:    true,
			},
			Spec: &corev1.Region_Spec{},
			Status: &corev1.Region_Status{
				Index: req.Status.Index,
			},
		}

		if err := regionutils.SetRegionExtInfo(rgn, extInfo); err != nil {
			return nil, err
		}

		rgn, err = g.octeliumCInit.CoreC().CreateRegion(ctx, rgn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	vutils.SetRegionPublicHostName(rgn)

	return g.octeliumCInit.CoreC().UpdateRegion(ctx, rgn)
}

func (g *Genesis) setRegionCoreVersion(ctx context.Context, rgn *corev1.Region, version string) (*corev1.Region, error) {
	region, err := g.octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Uid: rgn.Metadata.Uid})
	if err != nil {
		return nil, err
	}

	region.Status.Version = version
	if region.Status.VersionInfoMap == nil {
		region.Status.VersionInfoMap = make(map[string]*corev1.Region_Status_VersionInfo)
	}

	region.Status.VersionInfoMap[regionutils.PackageCore] = &corev1.Region_Status_VersionInfo{
		Package: regionutils.PackageCore,
		SetAt:   pbutils.Now(),
		Version: version,
		Id:      os.Getenv("OCTELIUM_INSTALL_ID"),
	}

	return g.octeliumCInit.CoreC().UpdateRegion(ctx, region)
}

func (g *Genesis) installJoinCoreComponents(ctx context.Context, opts *gc.CommonOpts) error {
	zap.L().Debug("Installing core components",
		zap.String("region", opts.Region.Metadata.Name),
		zap.Bool("spiffe", gc.IsSPIFFEEnabled(opts)))

	{
		if err := gc.CreateGatewayAgent(ctx, opts); err != nil {
			return err
		}

		if err := k8sutils.WaitReadinessDaemonsetWithNS(ctx, g.k8sC, "octelium-gwagent", vutils.K8sNS); err != nil {
			return err
		}

		if err := waitGatewayAgentsRegistered(ctx, g.k8sC); err != nil {
			return err
		}
	}

	{
		if err := gc.CreateNocturne(ctx, opts); err != nil {
			return err
		}

		if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, "octelium-nocturne"); err != nil {
			return err
		}
	}

	{
		if err := gc.CreateOctovigil(ctx, opts); err != nil {
			return err
		}

		if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, "octelium-octovigil"); err != nil {
			return err
		}
	}

	if err := gc.InstallCommon(ctx, opts); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) initJoinStorage(ctx context.Context, bs *cbootstrapv1.Config) error {
	if bs.Spec.PrimaryStorage == nil || bs.Spec.PrimaryStorage.GetPostgresql() == nil {
		return errors.Errorf("No Postgres info")
	}
	if bs.Spec.SecondaryStorage == nil || bs.Spec.SecondaryStorage.GetRedis() == nil {
		return errors.Errorf("No Redis info")
	}

	zap.L().Debug("Initializing storage secrets")

	{
		info := bs.Spec.PrimaryStorage.GetPostgresql()

		var password string
		switch info.PasswordType.(type) {
		case *cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_Password:
			password = info.GetPassword()
		case *cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_:
			secSpec := info.GetPasswordFromSecret()
			val, err := g.getJoinStoragePassword(ctx, secSpec.Name, secSpec.Namespace, secSpec.Key)
			if err != nil {
				return err
			}
			password = val
		}

		if _, err := k8sutils.CreateOrUpdateSecret(ctx, g.k8sC, &k8scorev1.Secret{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "octelium-postgres",
				Namespace: vutils.K8sNS,
			},
			Data: map[string][]byte{
				"postgres-password": []byte(password),
				"username":          []byte(info.Username),
				"host":              []byte(info.Host),
				"database":          []byte(info.Database),
				"port":              []byte(fmt.Sprintf("%d", info.Port)),
				"no_ssl":            []byte(fmt.Sprintf("%t", !info.IsTLS)),
			},
		}); err != nil {
			return err
		}
	}

	{
		info := bs.Spec.SecondaryStorage.GetRedis()

		var password string
		switch info.PasswordType.(type) {
		case *cbootstrapv1.Config_Spec_SecondaryStorage_Redis_Password:
			password = info.GetPassword()
		case *cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_:
			secSpec := info.GetPasswordFromSecret()
			val, err := g.getJoinStoragePassword(ctx, secSpec.Name, secSpec.Namespace, secSpec.Key)
			if err != nil {
				return err
			}
			password = val
		}

		if _, err := k8sutils.CreateOrUpdateSecret(ctx, g.k8sC, &k8scorev1.Secret{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "octelium-redis",
				Namespace: vutils.K8sNS,
			},
			Data: map[string][]byte{
				"password": []byte(password),
				"username": []byte(info.Username),
				"host":     []byte(info.Host),
				"database": []byte(fmt.Sprintf("%d", info.Database)),
				"port":     []byte(fmt.Sprintf("%d", info.Port)),
				"use_tls":  []byte(fmt.Sprintf("%t", info.IsTLS)),
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (g *Genesis) getJoinStoragePassword(ctx context.Context, name, namespace, key string) (string, error) {
	if namespace == "" {
		namespace = "default"
	}
	if name == "" {
		return "", errors.Errorf("name of passwordFromSecret is empty")
	}

	sec, err := g.k8sC.CoreV1().Secrets(namespace).Get(ctx, name, k8smetav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var password string
	if key != "" {
		password = string(sec.Data[key])
	}
	if password == "" {
		password = string(sec.Data["password"])
	}
	if password == "" {
		password = string(sec.Data["data"])
	}

	return password, nil
}

func getInstallationFromBootstrap(bs *cbootstrapv1.Config) (*corev1.ClusterConfig_Status_Installation, error) {
	spec := bs.GetSpec()
	if spec.GetSpiffe() == nil && spec.GetCni() == nil && spec.GetIngress() == nil {
		return nil, nil
	}

	ret := &corev1.ClusterConfig_Status_Installation{}

	if spec.GetSpiffe() != nil {
		ret.Spiffe = &corev1.ClusterConfig_Status_Installation_SPIFFE{}
		if err := pbutils.MarshalInto(ret.Spiffe, spec.GetSpiffe()); err != nil {
			return nil, err
		}
	}

	if spec.GetCni() != nil {
		ret.Cni = &corev1.ClusterConfig_Status_Installation_CNI{}
		if err := pbutils.MarshalInto(ret.Cni, spec.GetCni()); err != nil {
			return nil, err
		}
	}

	if spec.GetIngress() != nil {
		ret.Ingress = &corev1.ClusterConfig_Status_Installation_Ingress{}
		if err := pbutils.MarshalInto(ret.Ingress, spec.GetIngress()); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (g *Genesis) setInstallationConfigMap(ctx context.Context, clusterCfg *corev1.ClusterConfig) error {
	installation := clusterCfg.Status.Installation
	if installation == nil {
		installation = &corev1.ClusterConfig_Status_Installation{}
	}

	val, err := pbutils.MarshalJSON(installation, true)
	if err != nil {
		return err
	}

	cm := &k8scorev1.ConfigMap{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      installationConfigMapName,
			Namespace: vutils.K8sNS,
			Labels: map[string]string{
				"app": "octelium",
			},
		},
		Data: map[string]string{
			installationConfigMapKey: string(val),
		},
	}

	zap.L().Debug("Setting the installation ConfigMap", zap.Any("cm", cm))

	if _, err := k8sutils.CreateOrUpdateConfigMap(ctx, g.k8sC, cm); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) setNadConfig(ctx context.Context) error {
	nad := &netv1.NetworkAttachmentDefinition{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "octelium",
			Namespace: vutils.K8sNS,
		},
	}

	zap.L().Debug("Setting k8s NAD configuration", zap.Any("nad", nad))

	if _, err := g.nadC.K8sCniCncfIoV1().NetworkAttachmentDefinitions(vutils.K8sNS).
		Create(ctx, nad, k8smetav1.CreateOptions{}); err != nil {
		if k8serr.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func (g *Genesis) setDataPlaneNodeTaints(ctx context.Context) error {
	if ldflags.IsTest() {
		return nil
	}

	dataPlaneNodes, err := g.k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
		LabelSelector: vutils.NodeLabelDataPlane,
	})
	if err != nil {
		return err
	}

	if len(dataPlaneNodes.Items) == 0 {
		return errors.Errorf(
			"No Kubernetes node is labeled as an Octelium data-plane node. At least one node must be labeled via: kubectl label nodes <NODE_NAME> %s=",
			vutils.NodeLabelDataPlane)
	}

	nodeList, err := g.k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s,!%s",
			vutils.NodeLabelDataPlane, vutils.NodeLabelControlPlane),
	})
	if err != nil {
		return err
	}

	for _, item := range nodeList.Items {
		nodeName := item.Name

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			node, err := g.k8sC.CoreV1().Nodes().Get(ctx, nodeName, k8smetav1.GetOptions{})
			if err != nil {
				return err
			}

			hasTaint := slices.ContainsFunc(node.Spec.Taints, func(taint k8scorev1.Taint) bool {
				return taint.Key == vutils.NodeTaintGatewayInit
			})

			if node.Labels == nil {
				node.Labels = make(map[string]string)
			}
			_, isRegistered := node.Labels[vutils.NodeLabelGatewayRegistered]

			if hasTaint && !isRegistered {
				return nil
			}

			if !hasTaint {
				node.Spec.Taints = append(node.Spec.Taints, k8scorev1.Taint{
					Key:    vutils.NodeTaintGatewayInit,
					Value:  "true",
					Effect: k8scorev1.TaintEffectNoSchedule,
				})
			}

			delete(node.Labels, vutils.NodeLabelGatewayRegistered)

			_, err = g.k8sC.CoreV1().Nodes().Update(ctx, node, k8smetav1.UpdateOptions{})
			return err
		}); err != nil {
			return errors.Errorf("Could not set the gateway-init taint on the node %s: %+v", nodeName, err)
		}

		zap.L().Debug("Set the gateway-init taint on the data-plane node",
			zap.String("node", nodeName))
	}

	zap.L().Info("Successfully set the gateway-init taint on the dedicated data-plane nodes",
		zap.Int("tainted", len(nodeList.Items)),
		zap.Int("dataPlaneNodes", len(dataPlaneNodes.Items)))

	return nil
}

func waitGatewayAgentsRegistered(ctx context.Context, k8sC kubernetes.Interface) error {
	if ldflags.IsTest() {
		return nil
	}

	zap.L().Debug("Checking that all Gateway Agents are registered")

	doCheck := func() (bool, error) {
		nodeList, err := k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
			LabelSelector: vutils.NodeLabelDataPlane,
		})
		if err != nil {
			return false, err
		}

		if len(nodeList.Items) == 0 {
			return false, errors.Errorf(
				"No Kubernetes node is labeled as an Octelium data-plane node. At least one node must be labeled via: kubectl label nodes <NODE_NAME> %s=",
				vutils.NodeLabelDataPlane)
		}

		for _, node := range nodeList.Items {
			if node.Labels[vutils.NodeLabelGatewayRegistered] != "true" {
				zap.L().Debug("Gateway Agent is not registered yet on the data-plane node",
					zap.String("node", node.Name))
				return false, nil
			}
		}

		zap.L().Debug("All Gateway Agents are now registered",
			zap.Int("nodes", len(nodeList.Items)))

		return true, nil
	}

	for i := range 900 {
		isDone, err := doCheck()
		if err != nil && !k8sutils.ShouldRetry(err) {
			return err
		}

		if isDone {
			return nil
		}

		zap.L().Debug("Gateway Agents are still not registered on all data-plane nodes. Trying again...",
			zap.Int("attempt", i+1))

		time.Sleep(2 * time.Second)
	}

	return errors.Errorf("Timed out waiting for the Gateway Agents to be registered on all data-plane nodes")
}
