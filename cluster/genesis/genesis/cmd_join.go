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

	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	oc "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/k8sretry"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	iocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
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
	if err := g.waitForConcurrentGenesis(ctx); err != nil {
		return err
	}
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

	res, err := g.loadJoinResources(ctx)
	if err != nil {
		return err
	}

	if err := checkTargetDomain(os.Getenv("OCTELIUM_DOMAIN"), res.Domain); err != nil {
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

	if err := checkTargetDomain(res.Domain, clusterCfg.GetStatus().GetDomain()); err != nil {
		return err
	}
	existing, err := g.octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: res.Region.Metadata.Name})
	if grpcerr.IsNotFound(err) {
		if res.Region.GetMetadata().GetUid() != "" {
			return errors.New("The bootstrap Region no longer exists")
		}
		existing = res.Region
	} else if err != nil {
		return err
	}
	if uid := res.Region.GetMetadata().GetUid(); uid != "" && uid != existing.GetMetadata().GetUid() {
		return errors.New("The bootstrap Region identity does not match the live Region")
	}
	if err := g.validateRegionTarget(ctx, clusterCfg, existing); err != nil {
		return err
	}
	if err := g.setDataPlaneNodeTaints(ctx); err != nil {
		return err
	}

	rgn, err := g.createOrUpdateJoinRegion(ctx, res.Region)
	if err != nil {
		return err
	}

	if err := g.setInstallationConfigMap(ctx, clusterCfg); err != nil {
		return err
	}
	if err := g.recordRegionTarget(ctx, clusterCfg, rgn); err != nil {
		return err
	}

	if err := g.setNadConfig(ctx); err != nil {
		return err
	}

	if err := g.installJoinSecretManager(ctx, &components.CommonOpts{
		CommonOpts: gc.CommonOpts{
			K8sC:          g.k8sC,
			ClusterConfig: clusterCfg,
			Region:        rgn,
		},
	}); err != nil {
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
	sec, err := g.getK8sSecret(ctx, vutils.K8sNS, joinInitSecretName)
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
	info := &corev1.Region_Status_VersionInfo{
		Package: regionutils.PackageCore,
		SetAt:   pbutils.Now(),
		Version: version,
		Id:      os.Getenv("OCTELIUM_INSTALL_ID"),
	}
	var result *corev1.Region
	err := retryRegionVersion(ctx, func(ctx context.Context) error {
		region, err := g.octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Uid: rgn.Metadata.Uid})
		if err != nil {
			return err
		}
		updated := proto.Clone(region).(*corev1.Region)
		if updated.Status == nil {
			updated.Status = &corev1.Region_Status{}
		}
		updated.Status.Version = version
		if updated.Status.VersionInfoMap == nil {
			updated.Status.VersionInfoMap = make(map[string]*corev1.Region_Status_VersionInfo)
		}
		updated.Status.VersionInfoMap[regionutils.PackageCore] = info
		if proto.Equal(region, updated) {
			result = region
			return nil
		}
		result, err = g.octeliumCInit.CoreC().UpdateRegion(ctx, updated)
		return err
	})
	return result, errors.Wrap(err, "Could not publish joined Region core version")
}

func (g *Genesis) installJoinSecretManager(ctx context.Context, opts *components.CommonOpts) error {
	zap.L().Debug("Installing the enterprise secret manager before the core components",
		zap.String("region", opts.Region.Metadata.Name))

	if err := components.CreateRscServer(ctx, opts); err != nil {
		return err
	}

	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, oc.OcteliumEnterpriseComponent(oc.RscServer)); err != nil {
		return err
	}

	if err := components.CreateSecretMan(ctx, opts); err != nil {
		return err
	}

	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, oc.OcteliumEnterpriseComponent(oc.SecretMan)); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) installJoinCoreComponents(ctx context.Context, opts *gc.CommonOpts) error {
	zap.L().Debug("Installing core components",
		zap.String("region", opts.Region.Metadata.Name),
		zap.Bool("spiffe", gc.IsSPIFFEEnabled(opts)))

	{
		if err := gc.CreateGatewayAgent(ctx, opts); err != nil {
			return err
		}

		if err := waitGatewayAgentReady(ctx, g.k8sC); err != nil {
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
	passwords, err := g.resolveStoragePasswords(ctx, bs)
	if err != nil {
		return err
	}

	zap.L().Debug("Initializing storage secrets")

	{
		info := bs.Spec.PrimaryStorage.GetPostgresql()

		password := passwords.postgres

		if err := g.createStorageSecret(ctx, &k8scorev1.Secret{
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

		password := passwords.redis

		if err := g.createStorageSecret(ctx, &k8scorev1.Secret{
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

	dataPlaneNodes, err := k8sretry.Read(ctx, func(ctx context.Context) (*k8scorev1.NodeList, error) {
		return g.k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{LabelSelector: vutils.NodeLabelDataPlane})
	})
	if err != nil {
		return err
	}

	if len(dataPlaneNodes.Items) == 0 {
		return errors.Errorf(
			"No Kubernetes node is labeled as an Octelium data-plane node. At least one node must be labeled via: kubectl label nodes <NODE_NAME> %s=",
			vutils.NodeLabelDataPlane)
	}

	nodeList, err := k8sretry.Read(ctx, func(ctx context.Context) (*k8scorev1.NodeList, error) {
		return g.k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s,!%s", vutils.NodeLabelDataPlane, vutils.NodeLabelControlPlane),
		})
	})
	if err != nil {
		return err
	}

	hasGateway := true
	if _, err := k8sretry.Read(ctx, func(ctx context.Context) (*appsv1.DaemonSet, error) {
		return g.k8sC.AppsV1().DaemonSets(vutils.K8sNS).Get(ctx, "octelium-gwagent", k8smetav1.GetOptions{})
	}); k8serr.IsNotFound(err) {
		hasGateway = false
	} else if err != nil {
		return err
	}

	for _, item := range nodeList.Items {
		nodeName := item.Name

		if err := k8sretry.Do(ctx, func(ctx context.Context) error {
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
			isRegistered := node.Labels[vutils.NodeLabelGatewayRegistered] == "true"
			if isRegistered && hasGateway {
				return nil
			}

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

			if err := ctx.Err(); err != nil {
				return err
			}
			_, err = g.k8sC.CoreV1().Nodes().Update(ctx, node, k8smetav1.UpdateOptions{})
			return err
		}); err != nil {
			return errors.Wrapf(err, "Could not set the gateway-init taint on the node %s", nodeName)
		}

		zap.L().Debug("Set the gateway-init taint on the data-plane node",
			zap.String("node", nodeName))
	}

	zap.L().Info("Successfully set the gateway-init taint on the dedicated data-plane nodes",
		zap.Int("tainted", len(nodeList.Items)),
		zap.Int("dataPlaneNodes", len(dataPlaneNodes.Items)))

	return nil
}

type storagePasswords struct {
	postgres string
	redis    string
}

func validatePasswordSecretReference(name, namespace, key string) error {
	if name == "" {
		return errors.New("name of passwordFromSecret is empty")
	}
	if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
		return errors.Errorf("Invalid passwordFromSecret name %q: %s", name, strings.Join(errs, ", "))
	}
	if namespace != "" {
		if errs := validation.IsDNS1123Label(namespace); len(errs) > 0 {
			return errors.Errorf("Invalid passwordFromSecret namespace %q: %s", namespace, strings.Join(errs, ", "))
		}
	}
	if key != "" {
		if errs := validation.IsConfigMapKey(key); len(errs) > 0 {
			return errors.Errorf("Invalid passwordFromSecret key %q: %s", key, strings.Join(errs, ", "))
		}
	}
	return nil
}

func (g *Genesis) resolveStoragePasswords(ctx context.Context, bs *cbootstrapv1.Config) (*storagePasswords, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if bs.GetSpec() == nil {
		return nil, errors.New("Nil Bootstrap")
	}
	pg := bs.Spec.GetPrimaryStorage().GetPostgresql()
	if pg == nil {
		return nil, errors.New("No Postgres info")
	}
	redis := bs.Spec.GetSecondaryStorage().GetRedis()
	if redis == nil {
		return nil, errors.New("No Redis info")
	}

	secrets := make(map[string]*k8scorev1.Secret)
	getPassword := func(name, namespace, key string) (string, error) {
		if err := validatePasswordSecretReference(name, namespace, key); err != nil {
			return "", err
		}
		if namespace == "" {
			namespace = "default"
		}
		identity := namespace + "/" + name
		secret := secrets[identity]
		if secret == nil {
			var err error
			secret, err = g.getK8sSecret(ctx, namespace, name)
			if err != nil {
				return "", err
			}
			secrets[identity] = secret
		}
		if key != "" {
			if len(secret.Data[key]) == 0 {
				return "", errors.Errorf("The passwordFromSecret %s key %q is missing or empty", identity, key)
			}
			return string(secret.Data[key]), nil
		}
		for _, key := range []string{"password", "data"} {
			if len(secret.Data[key]) > 0 {
				return string(secret.Data[key]), nil
			}
		}
		return "", errors.Errorf("The passwordFromSecret %s has no non-empty password or data key", identity)
	}

	ret := &storagePasswords{}
	var err error
	switch value := pg.PasswordType.(type) {
	case *cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_Password:
		if value != nil {
			ret.postgres = value.Password
		}
	case *cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_PasswordFromSecret_:
		if value == nil {
			return nil, errors.New("name of passwordFromSecret is empty")
		}
		ref := value.PasswordFromSecret
		ret.postgres, err = getPassword(ref.GetName(), ref.GetNamespace(), ref.GetKey())
		if err != nil {
			return nil, errors.Wrap(err, "Could not resolve the Postgres password")
		}
	}
	switch value := redis.PasswordType.(type) {
	case *cbootstrapv1.Config_Spec_SecondaryStorage_Redis_Password:
		if value != nil {
			ret.redis = value.Password
		}
	case *cbootstrapv1.Config_Spec_SecondaryStorage_Redis_PasswordFromSecret_:
		if value == nil {
			return nil, errors.New("name of passwordFromSecret is empty")
		}
		ref := value.PasswordFromSecret
		ret.redis, err = getPassword(ref.GetName(), ref.GetNamespace(), ref.GetKey())
		if err != nil {
			return nil, errors.Wrap(err, "Could not resolve the Redis password")
		}
	}
	return ret, nil
}
