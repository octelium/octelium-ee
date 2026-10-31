// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesisjob

import (
	"context"
	"fmt"
	"time"

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

const DefaultTimeout = 20 * time.Minute

type Opts struct {
	Domain     string
	Region     string
	Package    string
	Cmd        string
	Version    string
	SvcAccount string
	SPIFFE     *install.SPIFFEOpts
	Args       []string
}

func GetJob(o *Opts) *batchv1.Job {
	labels := map[string]string{
		"app":                         "octelium",
		"octelium.com/component":      "genesis",
		"octelium.com/component-type": "cluster",
	}

	svcAccount := o.SvcAccount
	if svcAccount == "" {
		svcAccount = "octelium-nocturne"
	}

	podSpec := install.GetGenesisPodSpec(&install.GenesisPodSpecOpts{
		Domain:     o.Domain,
		Cmd:        o.Cmd,
		Version:    o.Version,
		SvcAccount: svcAccount,
		Package:    o.Package,
		Region:     o.Region,
		SPIFFE:     o.SPIFFE,
	})

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, o.Args...)

	ret := &batchv1.Job{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      fmt.Sprintf("octelium-genesis-%s-%s", o.Cmd, utilrand.GetRandomStringLowercase(6)),
			Namespace: vutils.K8sNS,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: k8smetav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}

	return ret
}

func Run(ctx context.Context, k8sC kubernetes.Interface, job *batchv1.Job, timeout time.Duration) error {
	if job == nil {
		return errors.Errorf("nil job")
	}

	cmd := ""
	if len(job.Spec.Template.Spec.Containers) > 0 && len(job.Spec.Template.Spec.Containers[0].Args) > 0 {
		cmd = job.Spec.Template.Spec.Containers[0].Args[0]
	}

	if _, err := k8sC.BatchV1().Jobs(job.Namespace).Create(ctx, job, k8smetav1.CreateOptions{}); err != nil {
		return errors.Errorf("could not create %s job %s: %+v", cmd, job.Name, err)
	}

	zap.L().Info("Created genesis job",
		zap.String("job", job.Name),
		zap.String("cmd", cmd))

	if err := Wait(ctx, k8sC, job, timeout); err != nil {
		return errors.Errorf("%s job %s did not complete successfully: %+v", cmd, job.Name, err)
	}

	zap.L().Info("Genesis job completed successfully",
		zap.String("job", job.Name),
		zap.String("cmd", cmd))

	return nil
}

func Wait(ctx context.Context, k8sC kubernetes.Interface, job *batchv1.Job, timeout time.Duration) error {
	if job == nil {
		return errors.Errorf("nil job")
	}
	if job.Name == "" {
		return errors.Errorf("empty job name")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		if err := Check(ctx, k8sC, job.Name); err != nil {
			if IsTerminalErr(err) {
				return err
			}

			zap.L().Debug("Genesis job has not completed yet",
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

type terminalJobErr struct {
	err error
}

func (e *terminalJobErr) Error() string {
	if e == nil || e.err == nil {
		return "terminal genesis job error"
	}
	return e.err.Error()
}

func (e *terminalJobErr) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func IsTerminalErr(err error) bool {
	var terminalErr *terminalJobErr
	return errors.As(err, &terminalErr)
}

func terminalErr(format string, args ...any) error {
	return &terminalJobErr{
		err: errors.Errorf(format, args...),
	}
}

func Check(ctx context.Context, k8sC kubernetes.Interface, jobName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	job, err := k8sC.BatchV1().Jobs(vutils.K8sNS).Get(ctx, jobName, k8smetav1.GetOptions{})
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
				return terminalErr(
					"genesis job %s failed: reason=%s message=%s",
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
		"genesis job %s is not complete yet: active=%d succeeded=%d failedAttempts=%d",
		jobName,
		job.Status.Active,
		job.Status.Succeeded,
		job.Status.Failed,
	)
}
