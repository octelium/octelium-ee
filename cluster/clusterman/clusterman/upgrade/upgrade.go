// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package upgrade

import (
	"context"
	"slices"
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/genesisjob"
	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/regionk8s"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/client/octops/commands/install"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
)

const RegionTimeout = 20 * time.Minute

const maxMessageLen = 1024

type UpgradeClusterOpts struct {
	OcteliumC      octeliumc.ClientInterface
	K8sC           kubernetes.Interface
	UpgradeRequest *enterprisev1.ClusterConfig_Status_UpgradeRequest
	NewRegionK8sC  regionk8s.NewClientFn
	OnRegions      func(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region)
}

func UpgradeCluster(ctx context.Context, o *UpgradeClusterOpts) error {

	ctl, err := newController(ctx, o)
	if err != nil {
		return err
	}

	return ctl.run(ctx)
}

type controller struct {
	octeliumC     octeliumc.ClientInterface
	k8sC          kubernetes.Interface
	req           *enterprisev1.ClusterConfig_Status_UpgradeRequest
	domain        string
	spiffe        *install.SPIFFEOpts
	newRegionK8sC regionk8s.NewClientFn
	onRegions     func(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region)
}

func newController(ctx context.Context, o *UpgradeClusterOpts) (*controller, error) {
	if o == nil {
		return nil, errors.Errorf("nil opts")
	}

	ret := &controller{
		octeliumC:     o.OcteliumC,
		k8sC:          o.K8sC,
		req:           o.UpgradeRequest,
		newRegionK8sC: o.NewRegionK8sC,
		onRegions:     o.OnRegions,
	}

	if ret.newRegionK8sC == nil {
		ret.newRegionK8sC = regionk8s.NewClientFactory(o.OcteliumC)
	}

	cc, err := o.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.domain = cc.Status.Domain

	spiffe := cc.GetStatus().GetInstallation().GetSpiffe()
	if spiffe.GetEnable() {
		ret.spiffe = &install.SPIFFEOpts{
			CSIDriver:   spiffe.GetCsiDriver().GetName(),
			TrustDomain: spiffe.GetTrustDomain(),
		}
	}

	return ret, nil
}

type regionTarget struct {
	region *corev1.Region
	k8sC   kubernetes.Interface
}

func (c *controller) run(ctx context.Context) error {
	if c.req == nil || c.req.Request == nil {
		zap.L().Debug("Nil upgrade request. Nothing to be done...")
		return nil
	}

	if err := validateRequest(c.req.Request); err != nil {
		return err
	}

	if c.req.Request.Core == nil && c.req.Request.PackageEnterprise == nil && c.req.Request.PackageCordium == nil {
		zap.L().Debug("No package to be upgraded. Nothing to be done...")
		return nil
	}

	targets, err := c.getTargets(ctx)
	if err != nil {
		return err
	}

	return c.runTargets(ctx, targets)
}

func validateRequest(req *enterprisev1.UpgradeClusterRequest_Request) error {
	if req.Core != nil && req.Core.Version == "" {
		return errors.Errorf("core upgrade failed: empty upgrade version for package %s", regionutils.PackageCore)
	}

	if req.PackageEnterprise != nil && req.PackageEnterprise.Version == "" {
		return errors.Errorf("enterprise package upgrade failed: empty upgrade version for package %s",
			regionutils.PackageEnterprise)
	}

	if req.PackageCordium != nil && req.PackageCordium.Version == "" {
		return errors.Errorf("cordium package upgrade failed: empty upgrade version for package %s",
			regionutils.PackageCordium)
	}

	return nil
}

func (c *controller) getTargets(ctx context.Context) ([]*regionTarget, error) {
	rgnList, err := c.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var ret []*regionTarget
	var others []*corev1.Region

	for _, rgn := range rgnList.Items {
		switch {
		case rgn.Metadata.Name == regionutils.DefaultRegion:
			ret = append(ret, &regionTarget{
				region: rgn,
				k8sC:   c.k8sC,
			})
		case regionutils.IsJoinCompleted(rgn):
			others = append(others, rgn)
		default:
			zap.L().Debug("Skipping a Region that is not managed by the Cluster",
				zap.String("region", rgn.Metadata.Name))
		}
	}

	if len(ret) == 0 {
		return nil, errors.Errorf("Could not find the default Region")
	}

	slices.SortFunc(others, func(a, b *corev1.Region) int {
		return int(a.Status.Index) - int(b.Status.Index)
	})

	for _, rgn := range others {
		extInfo, err := regionutils.GetRegionExtInfo(rgn)
		if err != nil {
			return nil, err
		}

		k8sC, err := c.newRegionK8sC(ctx, extInfo.Kubernetes)
		if err != nil {
			return nil, errors.Errorf("Could not create the Kubernetes client of the Region %s: %+v",
				rgn.Metadata.Name, err)
		}

		ret = append(ret, &regionTarget{
			region: rgn,
			k8sC:   k8sC,
		})
	}

	return ret, nil
}

func (c *controller) runTargets(ctx context.Context, targets []*regionTarget) error {
	regions := make([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, 0, len(targets))
	for _, t := range targets {
		regions = append(regions, &enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
			RegionRef: umetav1.GetObjectReference(t.region),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		})
	}

	c.reportRegions(ctx, regions)

	for i, t := range targets {
		regions[i].State = enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING
		c.reportRegions(ctx, regions)

		if err := c.runTarget(ctx, t); err != nil {
			regions[i].State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
			regions[i].DoneAt = pbutils.Now()
			regions[i].Message = truncateMessage(err.Error())
			c.reportRegions(ctx, regions)

			return errors.Errorf("Could not upgrade the Region %s: %+v", t.region.Metadata.Name, err)
		}

		regions[i].State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
		regions[i].DoneAt = pbutils.Now()
		c.reportRegions(ctx, regions)
	}

	return nil
}

func (c *controller) reportRegions(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
	if c.onRegions == nil {
		return
	}

	ret := make([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, 0, len(regions))
	for _, rgn := range regions {
		ret = append(ret, pbutils.Clone(rgn).(*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region))
	}

	c.onRegions(ctx, ret)
}

func (c *controller) runTarget(ctx context.Context, t *regionTarget) error {
	ctx, cancel := context.WithTimeout(ctx, RegionTimeout)
	defer cancel()

	req := c.req.Request
	isDefault := t.region.Metadata.Name == regionutils.DefaultRegion

	zap.L().Info("Upgrading Region", zap.String("region", t.region.Metadata.Name))

	if req.Core != nil {
		if err := c.runUpgradeJob(ctx, t, regionutils.PackageCore, req.Core.Version); err != nil {
			return errors.Errorf("core upgrade failed: %+v", err)
		}
	}

	if req.PackageEnterprise != nil {
		if err := c.runUpgradeJob(ctx, t, regionutils.PackageEnterprise, req.PackageEnterprise.Version); err != nil {
			return errors.Errorf("enterprise package upgrade failed: %+v", err)
		}
	}

	if req.PackageCordium != nil && (isDefault || regionutils.HasPackage(t.region, regionutils.PackageCordium)) {
		if err := c.runUpgradeJob(ctx, t, regionutils.PackageCordium, req.PackageCordium.Version); err != nil {
			return errors.Errorf("cordium package upgrade failed: %+v", err)
		}
	}

	return nil
}

func (c *controller) runUpgradeJob(ctx context.Context, t *regionTarget, pkg string, version string) error {
	if version == "" {
		return errors.Errorf("empty upgrade version for package %s", pkg)
	}

	job := getGenesisJob(c.domain, t.region.Metadata.Name, pkg, version, c.spiffe)

	zap.L().Info("Running upgrade job",
		zap.String("job", job.Name),
		zap.String("region", t.region.Metadata.Name),
		zap.String("pkg", pkg),
		zap.String("version", version),
	)

	if err := genesisjob.Run(ctx, t.k8sC, job, genesisjob.DefaultTimeout); err != nil {
		return errors.Errorf("package %s version %s: %+v", pkg, version, err)
	}

	return nil
}

func getGenesisJob(domain string, regionName string, pkg string, version string,
	spiffe *install.SPIFFEOpts) *batchv1.Job {
	return genesisjob.GetJob(&genesisjob.Opts{
		Domain:  domain,
		Region:  regionName,
		Package: pkg,
		Cmd:     "upgrade",
		Version: version,
		SPIFFE:  spiffe,
	})
}

func truncateMessage(msg string) string {
	if len(msg) > maxMessageLen {
		return msg[:maxMessageLen]
	}
	return msg
}
