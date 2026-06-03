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

/*
func (c *controller) run(ctx context.Context) error {
	if c.req == nil || c.req.Request == nil {
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
		job := getGenesisJob(c.domain, vutils.GetMyRegionName(),
			"octeliumee", c.req.Request.PackageEnterprise.Version)
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
		job := getGenesisJob(c.domain, vutils.GetMyRegionName(),
			"cordium", c.req.Request.PackageCordium.Version)
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
*/

func (c *controller) run(ctx context.Context) error {
	if c.req == nil || c.req.Request == nil {
		zap.L().Debug("Nil upgrade request. Nothing to be done...")
		return nil
	}

	if c.req.Request.Core != nil {
		if err := c.runUpgradeJob(ctx, "octelium", c.req.Request.Core.Version); err != nil {
			return errors.Errorf("core upgrade failed: %+v", err)
		}
	}

	if c.req.Request.PackageEnterprise != nil {
		if err := c.runUpgradeJob(ctx, "octeliumee", c.req.Request.PackageEnterprise.Version); err != nil {
			return errors.Errorf("enterprise package upgrade failed: %+v", err)
		}
	}

	if c.req.Request.PackageCordium != nil {
		if err := c.runUpgradeJob(ctx, "cordium", c.req.Request.PackageCordium.Version); err != nil {
			return errors.Errorf("cordium package upgrade failed: %+v", err)
		}
	}

	return nil
}

func (c *controller) runUpgradeJob(ctx context.Context, pkg string, version string) error {
	if version == "" {
		return errors.Errorf("empty upgrade version for package %s", pkg)
	}

	job := getGenesisJob(c.domain, vutils.GetMyRegionName(), pkg, version)

	if _, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Create(ctx, job, k8smetav1.CreateOptions{}); err != nil {
		return errors.Errorf("could not create upgrade job %s for package %s version %s: %+v",
			job.Name, pkg, version, err)
	}

	zap.L().Info("Created upgrade job",
		zap.String("job", job.Name),
		zap.String("pkg", pkg),
		zap.String("version", version),
	)

	if err := c.waitUpgrade(ctx, job); err != nil {
		return errors.Errorf("upgrade job %s for package %s version %s did not complete successfully: %+v",
			job.Name, pkg, version, err)
	}

	zap.L().Info("Upgrade job completed successfully",
		zap.String("job", job.Name),
		zap.String("pkg", pkg),
		zap.String("version", version),
	)

	return nil
}

func (c *controller) waitUpgrade(ctx context.Context, job *batchv1.Job) error {
	if job == nil {
		return errors.Errorf("nil job")
	}
	if job.Name == "" {
		return errors.Errorf("empty job name")
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	// initial wait to avoid k8s api race condition
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		if err := c.checkUpgradeJob(ctx, job.Name); err != nil {
			if isTerminalUpgradeJobErr(err) {
				return err
			}

			zap.L().Debug("Upgrade job has not completed yet",
				zap.String("job", job.Name),
				zap.Error(err),
			)
		} else {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type terminalUpgradeJobErr struct {
	err error
}

func (e *terminalUpgradeJobErr) Error() string {
	if e == nil || e.err == nil {
		return "terminal upgrade job error"
	}
	return e.err.Error()
}

func (e *terminalUpgradeJobErr) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func isTerminalUpgradeJobErr(err error) bool {
	var terminalErr *terminalUpgradeJobErr
	return errors.As(err, &terminalErr)
}

func terminalUpgradeErr(format string, args ...any) error {
	return &terminalUpgradeJobErr{
		err: errors.Errorf(format, args...),
	}
}

func (c *controller) checkUpgradeJob(ctx context.Context, jobName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	job, err := c.k8sC.BatchV1().Jobs(vutils.K8sNS).Get(ctx, jobName, k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, cond := range job.Status.Conditions {
		switch cond.Type {
		case batchv1.JobComplete:
			if cond.Status == k8scorev1.ConditionTrue {
				return nil
			}

		case batchv1.JobFailed:
			if cond.Status == k8scorev1.ConditionTrue {
				return terminalUpgradeErr(
					"upgrade job %s failed: reason=%s message=%s",
					jobName,
					cond.Reason,
					cond.Message,
				)
			}
		}
	}

	if job.Status.Succeeded > 0 {
		return nil
	}

	return errors.Errorf(
		"upgrade job %s is not complete yet: active=%d succeeded=%d failedAttempts=%d",
		jobName,
		job.Status.Active,
		job.Status.Succeeded,
		job.Status.Failed,
	)
}

func getGenesisJob(domain string, regionName string, pkg string, version string) *batchv1.Job {
	labels := map[string]string{
		"app":                         "octelium",
		"octelium.com/component":      "genesis",
		"octelium.com/component-type": "cluster",
	}

	ret := &batchv1.Job{
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

	return ret
}
