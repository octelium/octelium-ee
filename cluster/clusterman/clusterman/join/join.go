// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package join

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/genesisjob"
	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/regionk8s"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/client/octops/commands/install"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	joinJobTimeout    = 30 * time.Minute
	cordiumJobTimeout = genesisjob.DefaultTimeout

	initSecretName    = "octelium-init"
	kekSecretName     = "sys-init-kek"
	genesisSvcAccount = "octelium-genesis"

	defaultRegionBitsV4 = 4
	maxRegionsV6        = 1 << 16
)

type JoinClusterOpts struct {
	OcteliumC     octeliumc.ClientInterface
	K8sC          kubernetes.Interface
	JoinRequest   *enterprisev1.ClusterConfig_Status_JoinRequest
	NewRegionK8sC regionk8s.NewClientFn
}

func JoinCluster(ctx context.Context, o *JoinClusterOpts) (*corev1.Region, error) {
	ctl, err := newController(o)
	if err != nil {
		return nil, err
	}

	return ctl.run(ctx)
}

type controller struct {
	octeliumC     octeliumc.ClientInterface
	k8sC          kubernetes.Interface
	req           *enterprisev1.JoinClusterRequest
	newRegionK8sC regionk8s.NewClientFn
}

func newController(o *JoinClusterOpts) (*controller, error) {
	if o == nil {
		return nil, errors.Errorf("nil opts")
	}

	if o.JoinRequest == nil || o.JoinRequest.Request == nil {
		return nil, errors.Errorf("nil join request")
	}

	if o.JoinRequest.Request.Name == "" || o.JoinRequest.Request.Name == regionutils.DefaultRegion {
		return nil, errors.Errorf("Invalid join Region name: %s", o.JoinRequest.Request.Name)
	}

	ret := &controller{
		octeliumC:     o.OcteliumC,
		k8sC:          o.K8sC,
		req:           o.JoinRequest.Request,
		newRegionK8sC: o.NewRegionK8sC,
	}

	if ret.newRegionK8sC == nil {
		ret.newRegionK8sC = regionk8s.NewClientFactory(o.OcteliumC)
	}

	return ret, nil
}

type packageVersions struct {
	core       string
	enterprise string
	cordium    string
}

func (c *controller) run(ctx context.Context) (*corev1.Region, error) {
	regionName := c.req.Name

	zap.L().Info("Starting joining a new Region", zap.String("region", regionName))

	targetK8sC, err := c.newRegionK8sC(ctx, c.req.Kubernetes)
	if err != nil {
		return nil, errors.Errorf("Could not create the Kubernetes client of the Region: %+v", err)
	}

	if err := c.checkTarget(ctx, targetK8sC); err != nil {
		return nil, err
	}

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	defaultRgn, err := c.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: regionutils.DefaultRegion,
	})
	if err != nil {
		return nil, err
	}

	versions, err := getPackageVersions(defaultRgn)
	if err != nil {
		return nil, err
	}

	rgnList, err := c.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	idx, err := allocateRegionIndex(cc, rgnList.Items, regionName)
	if err != nil {
		return nil, err
	}

	bs, err := c.getBootstrap(ctx)
	if err != nil {
		return nil, err
	}

	kek, err := c.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, kekSecretName, k8smetav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("Could not get the KEK Secret: %+v", err)
	}

	rgn := &corev1.Region{
		Metadata: &metav1.Metadata{
			Name:        regionName,
			DisplayName: c.req.DisplayName,
		},
		Spec: &corev1.Region_Spec{},
		Status: &corev1.Region_Status{
			Index: idx,
		},
	}

	if err := regionutils.SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{
		Kubernetes: c.req.Kubernetes,
	}); err != nil {
		return nil, err
	}

	zap.L().Debug("Preparing the Kubernetes cluster of the Region",
		zap.String("region", regionName),
		zap.Int32("index", idx),
		zap.String("coreVersion", versions.core),
		zap.String("enterpriseVersion", versions.enterprise),
		zap.String("cordiumVersion", versions.cordium))

	if err := prepareTarget(ctx, targetK8sC, &targetResources{
		bootstrap: bs,
		region:    rgn,
		domain:    cc.Status.Domain,
		kek:       kek.Data,
	}); err != nil {
		return nil, errors.Errorf("Could not prepare the Kubernetes cluster of the Region: %+v", err)
	}

	spiffe := getSPIFFE(cc)

	if err := genesisjob.Run(ctx, targetK8sC, genesisjob.GetJob(&genesisjob.Opts{
		Domain:     cc.Status.Domain,
		Region:     regionName,
		Package:    regionutils.PackageEnterprise,
		Cmd:        "join",
		Version:    versions.enterprise,
		SvcAccount: genesisSvcAccount,
		SPIFFE:     spiffe,
		Args:       []string{"--core-version", versions.core},
	}), joinJobTimeout); err != nil {
		return nil, err
	}

	if versions.cordium != "" {
		if err := genesisjob.Run(ctx, targetK8sC, genesisjob.GetJob(&genesisjob.Opts{
			Domain:  cc.Status.Domain,
			Region:  regionName,
			Package: regionutils.PackageCordium,
			Cmd:     "init",
			Version: versions.cordium,
			SPIFFE:  spiffe,
		}), cordiumJobTimeout); err != nil {
			return nil, errors.Errorf("Could not install the Cordium package: %+v", err)
		}
	}

	ret, err := c.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: regionName,
	})
	if err != nil {
		return nil, err
	}

	if !regionutils.IsJoinCompleted(ret) {
		return nil, errors.Errorf("The Region %s has not completed joining the Cluster", regionName)
	}

	zap.L().Info("Successfully joined the new Region", zap.String("region", regionName))

	return ret, nil
}

func getPackageVersions(defaultRgn *corev1.Region) (*packageVersions, error) {
	ret := &packageVersions{
		core:       regionutils.GetPackageVersion(defaultRgn, regionutils.PackageCore),
		enterprise: regionutils.GetPackageVersion(defaultRgn, regionutils.PackageEnterprise),
		cordium:    regionutils.GetPackageVersion(defaultRgn, regionutils.PackageCordium),
	}

	if ret.core == "" {
		return nil, errors.Errorf("Could not find the core package version of the default Region")
	}

	if ret.enterprise == "" {
		ret.enterprise = ldflags.GetVersion()
	}

	return ret, nil
}

func getSPIFFE(cc *corev1.ClusterConfig) *install.SPIFFEOpts {
	spiffe := cc.GetStatus().GetInstallation().GetSpiffe()
	if !spiffe.GetEnable() {
		return nil
	}

	return &install.SPIFFEOpts{
		CSIDriver:   spiffe.GetCsiDriver().GetName(),
		TrustDomain: spiffe.GetTrustDomain(),
	}
}

func getMaxRegions(cc *corev1.ClusterConfig) int {
	if !ucorev1.ToClusterConfig(cc).HasV4() {
		return maxRegionsV6
	}

	regionBits := defaultRegionBitsV4
	if v4 := cc.GetStatus().GetNetworkConfig().GetV4(); v4.GetRegionBits() > 0 {
		regionBits = int(v4.GetRegionBits())
	}

	if regionBits > 16 {
		regionBits = 16
	}

	return 1 << uint(regionBits)
}

func allocateRegionIndex(cc *corev1.ClusterConfig, regions []*corev1.Region, name string) (int32, error) {
	used := map[int32]bool{
		0: true,
	}

	for _, rgn := range regions {
		if rgn.Metadata.Name == name {
			if rgn.Status.Index < 1 {
				return 0, errors.Errorf("The existing Region %s has an invalid index %d", name, rgn.Status.Index)
			}

			return rgn.Status.Index, nil
		}

		used[rgn.Status.Index] = true
	}

	maxRegions := getMaxRegions(cc)
	for i := 1; i < maxRegions; i++ {
		if !used[int32(i)] {
			return int32(i), nil
		}
	}

	return 0, errors.Errorf("No Region index is available. The Cluster supports up to %d Regions", maxRegions)
}

func (c *controller) getBootstrap(ctx context.Context) (*cbootstrapv1.Config, error) {
	bs, err := ovutils.GetBootstrapConfig(ctx, c.octeliumC)
	if err != nil {
		return nil, errors.Errorf("Could not get the bootstrap config: %+v", err)
	}

	pg := bs.GetSpec().GetPrimaryStorage().GetPostgresql()
	if pg == nil {
		return nil, errors.Errorf("The bootstrap config has no Postgres info")
	}

	redis := bs.GetSpec().GetSecondaryStorage().GetRedis()
	if redis == nil {
		return nil, errors.Errorf("The bootstrap config has no Redis info")
	}

	pgSecret, err := c.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-postgres", k8smetav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("Could not get the Postgres Secret: %+v", err)
	}

	pg.PasswordType = &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_Password{
		Password: string(pgSecret.Data["postgres-password"]),
	}

	redisSecret, err := c.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "octelium-redis", k8smetav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("Could not get the Redis Secret: %+v", err)
	}

	redis.PasswordType = &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_Password{
		Password: string(redisSecret.Data["password"]),
	}

	return bs, nil
}

func (c *controller) checkTarget(ctx context.Context, targetK8sC kubernetes.Interface) error {
	localNS, err := c.k8sC.CoreV1().Namespaces().Get(ctx, "kube-system", k8smetav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Could not get the kube-system namespace of the default Region: %+v", err)
	}

	targetNS, err := targetK8sC.CoreV1().Namespaces().Get(ctx, "kube-system", k8smetav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Could not connect to the Kubernetes cluster of the Region: %+v", err)
	}

	if localNS.UID != "" && localNS.UID == targetNS.UID {
		return errors.Errorf("The Kubernetes cluster is the one of the default Region")
	}

	sec, err := targetK8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, initSecretName, k8smetav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}
		return errors.Errorf("Could not check the Kubernetes cluster of the Region: %+v", err)
	}

	rgn := &corev1.Region{}
	if val := sec.Data["region"]; val != nil {
		if err := pbutils.Unmarshal(val, rgn); err != nil {
			return errors.Errorf("Could not parse the existing Region of the Kubernetes cluster: %+v", err)
		}
	}

	if rgn.GetMetadata().GetName() != c.req.Name {
		return errors.Errorf("The Kubernetes cluster is already used by another Region: %s",
			rgn.GetMetadata().GetName())
	}

	return nil
}

type targetResources struct {
	bootstrap *cbootstrapv1.Config
	region    *corev1.Region
	domain    string
	kek       map[string][]byte
}

func prepareTarget(ctx context.Context, k8sC kubernetes.Interface, r *targetResources) error {
	if _, err := k8sC.CoreV1().Namespaces().Create(ctx, &k8scorev1.Namespace{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: vutils.K8sNS,
			Labels: map[string]string{
				"app": "octelium",
			},
		},
	}, k8smetav1.CreateOptions{}); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}

	bootstrapPB, err := pbutils.Marshal(r.bootstrap)
	if err != nil {
		return err
	}

	regionPB, err := pbutils.Marshal(r.region)
	if err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateSecret(ctx, k8sC, &k8scorev1.Secret{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      initSecretName,
			Namespace: vutils.K8sNS,
		},
		Data: map[string][]byte{
			"bootstrap": bootstrapPB,
			"region":    regionPB,
			"domain":    []byte(r.domain),
		},
	}); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateSecret(ctx, k8sC, &k8scorev1.Secret{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      kekSecretName,
			Namespace: vutils.K8sNS,
		},
		Data: r.kek,
	}); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateServiceAccount(ctx, k8sC, &k8scorev1.ServiceAccount{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      genesisSvcAccount,
			Namespace: vutils.K8sNS,
		},
	}); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateClusterRole(ctx, k8sC, &rbacv1.ClusterRole{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: genesisSvcAccount,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*", "*.*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateClusterRoleBinding(ctx, k8sC, &rbacv1.ClusterRoleBinding{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: genesisSvcAccount,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     genesisSvcAccount,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      genesisSvcAccount,
				Namespace: vutils.K8sNS,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}
