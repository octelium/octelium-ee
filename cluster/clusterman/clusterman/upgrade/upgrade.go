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
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/genesisjob"
	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/regionk8s"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/client/octops/commands/install"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
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
	clusterUID    string
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
	ret.clusterUID = cc.GetMetadata().GetUid()

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
	if c.clusterUID != "" {
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, k8scorev1.EnvVar{Name: "OCTELIUM_GENESIS_CLUSTER_UID", Value: c.clusterUID})
	}
	if t.region.GetMetadata().GetUid() != "" {
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, k8scorev1.EnvVar{Name: "OCTELIUM_GENESIS_REGION_UID", Value: t.region.Metadata.Uid})
	}

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

func IsDone(state enterprisev1.ClusterConfig_Status_UpgradeRequest_State) bool {
	switch state {
	case enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS,
		enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED:
		return true
	default:
		return false
	}
}

func GetRegionResults(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
	before *timestamppb.Timestamp) ([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, error) {
	if req == nil {
		return nil, errors.Errorf("nil upgrade request")
	}

	regions, err := getRequestRegions(ctx, octeliumC, req)
	if err != nil {
		return nil, err
	}

	for _, itm := range regions {
		if IsDone(itm.State) || (itm.RegionRef.GetName() == "" && itm.RegionRef.GetUid() == "") {
			continue
		}

		rgn, err := octeliumC.CoreC().GetRegion(ctx, apivalidation.ObjectReferenceToRGetOptions(itm.RegionRef))
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		if doneAt := getAppliedAt(rgn, req, before); doneAt != nil {
			itm.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
			itm.DoneAt = doneAt
			itm.Message = ""
		}
	}

	return regions, nil
}

func getRequestRegions(ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest) ([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, error) {
	if len(req.Regions) > 0 {
		ret := make([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, 0, len(req.Regions))
		for _, itm := range req.Regions {
			ret = append(ret, pbutils.Clone(itm).(*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region))
		}
		return ret, nil
	}

	rgnList, err := octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var targets []*corev1.Region
	for _, rgn := range rgnList.Items {
		if rgn.Metadata.Name != regionutils.DefaultRegion && !regionutils.IsJoinCompleted(rgn) {
			continue
		}

		if req.CreatedAt.IsValid() && rgn.Metadata.CreatedAt.IsValid() &&
			!rgn.Metadata.CreatedAt.AsTime().Before(req.CreatedAt.AsTime()) {
			continue
		}

		targets = append(targets, rgn)
	}

	slices.SortFunc(targets, func(a, b *corev1.Region) int {
		switch {
		case a.Metadata.Name == regionutils.DefaultRegion:
			return -1
		case b.Metadata.Name == regionutils.DefaultRegion:
			return 1
		default:
			return int(a.Status.Index) - int(b.Status.Index)
		}
	})

	ret := make([]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, 0, len(targets))
	for _, rgn := range targets {
		ret = append(ret, &enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
			RegionRef: umetav1.GetObjectReference(rgn),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		})
	}

	return ret, nil
}

func getAppliedAt(rgn *corev1.Region, req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
	before *timestamppb.Timestamp) *timestamppb.Timestamp {
	versions := make(map[string]string)

	if req.Request.GetCore() != nil {
		versions[regionutils.PackageCore] = req.Request.Core.Version
	}

	if req.Request.GetPackageEnterprise() != nil {
		versions[regionutils.PackageEnterprise] = req.Request.PackageEnterprise.Version
	}

	if req.Request.GetPackageCordium() != nil &&
		(rgn.Metadata.GetName() == regionutils.DefaultRegion || regionutils.HasPackage(rgn, regionutils.PackageCordium)) {
		versions[regionutils.PackageCordium] = req.Request.PackageCordium.Version
	}

	if len(versions) == 0 {
		return nil
	}

	var ret *timestamppb.Timestamp
	for pkg, version := range versions {
		if !regionutils.IsPackageSetWithin(rgn, pkg, req.CreatedAt, before) ||
			!isSameVersion(regionutils.GetPackageVersion(rgn, pkg), version) {
			return nil
		}

		setAt := rgn.Status.VersionInfoMap[pkg].SetAt
		if ret == nil || setAt.AsTime().After(ret.AsTime()) {
			ret = setAt
		}
	}

	return ret
}

func isSameVersion(installed string, requested string) bool {
	return requested == "latest" || strings.TrimPrefix(installed, "v") == strings.TrimPrefix(requested, "v")
}

func truncateMessage(msg string) string {
	if len(msg) > maxMessageLen {
		return msg[:maxMessageLen]
	}
	return msg
}
