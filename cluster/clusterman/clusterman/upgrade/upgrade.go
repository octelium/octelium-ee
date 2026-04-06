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
	"fmt"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/client/octops/commands/install"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type UpgradeClusterOpts struct {
	OcteliumC      octeliumc.ClientInterface
	K8sC           kubernetes.Interface
	UpgradeRequest *enterprisev1.ClusterConfig_Status_UpgradeRequest
}

func UpgradeCluster(ctx context.Context, o *UpgradeClusterOpts) error {

	ctl, err := newController(ctx, o)
	if err != nil {
		return err
	}

	return ctl.run(ctx)
}

type controller struct {
	octeliumC octeliumc.ClientInterface
	k8sC      kubernetes.Interface
	req       *enterprisev1.ClusterConfig_Status_UpgradeRequest
	domain    string
}

func newController(ctx context.Context, o *UpgradeClusterOpts) (*controller, error) {
	if o == nil {
		return nil, errors.Errorf("nil opts")
	}

	ret := &controller{
		octeliumC: o.OcteliumC,
		k8sC:      o.K8sC,
		req:       o.UpgradeRequest,
	}

	cc, err := o.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.domain = cc.Status.Domain

	return ret, nil
}

func (c *controller) run(ctx context.Context) error {
	if c.req.Request == nil {
		zap.L().Debug("Nil upgrade request. Nothing to be done...")
		return nil
	}

	if c.req.Request.Core != nil {

		job := getGenesisJob(c.domain, vutils.GetMyRegionName(), "octelium", c.req.Request.Core.Version)
		if _, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Create(ctx,
			job,
			k8smetav1.CreateOptions{}); err != nil {
			return err
		}

		time.Sleep(3 * time.Second)
		if err := c.waitUpgrade(ctx, job); err != nil {
			zap.L().Warn("Could not waitUpgrade", zap.Any("job", job), zap.Error(err))
		}
	}

	if c.req.Request.PackageEnterprise != nil {
		job := getGenesisJob(c.domain, vutils.GetMyRegionName(), "octeliumee", c.req.Request.PackageEnterprise.Version)
		if _, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Create(ctx,
			job,
			k8smetav1.CreateOptions{}); err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
		if err := c.waitUpgrade(ctx, job); err != nil {
			zap.L().Warn("Could not waitUpgrade ee", zap.Any("job", job), zap.Error(err))
		}
	}

	if c.req.Request.PackageCordium != nil {
		job := getGenesisJob(c.domain, vutils.GetMyRegionName(), "cordium", c.req.Request.PackageCordium.Version)
		if _, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Create(ctx,
			job,
			k8smetav1.CreateOptions{}); err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
		if err := c.waitUpgrade(ctx, job); err != nil {
			zap.L().Warn("Could not waitUpgrade for cordium", zap.Any("job", job), zap.Error(err))
		}
	}

	return nil
}

func (c *controller) waitUpgrade(ctx context.Context, job *batchv1.Job) error {

	doFn := func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		job, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Get(ctx, job.Name, k8smetav1.GetOptions{})
		if err != nil {
			return err
		}
		if job.Status.Succeeded > 0 {
			return nil
		}

		zap.L().Debug("Job has not succeeded yet. Trying again...", zap.Any("job", job))

		return errors.Errorf("Not ready yet...")
	}

	for range 300 {
		if err := doFn(); err == nil {
			return nil
		}

		time.Sleep(3 * time.Second)
	}

	return errors.Errorf("Could not check job: %s", job.Name)
}

func (c *controller) getRegionVersion(ctx context.Context) (string, error) {
	rgn, err := c.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	if err != nil {
		return "", err
	}

	return rgn.Status.Version, nil
}

func getGenesisJob(domain string, regionName string, pkg string, version string) *batchv1.Job {
	labels := map[string]string{
		"app":                         "octelium",
		"octelium.com/component":      "genesis",
		"octelium.com/component-type": "cluster",
	}

	return &batchv1.Job{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      fmt.Sprintf("octelium-genesis-upgrade-%s", utilrand.GetRandomStringLowercase(6)),
			Namespace: vutils.K8sNS,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: k8smetav1.ObjectMeta{
					Labels: labels,
				},
				Spec: install.GetGenesisPodSpec(domain, "upgrade", version, "octelium-nocturne", pkg, regionName),
			},
		},
	}
}
