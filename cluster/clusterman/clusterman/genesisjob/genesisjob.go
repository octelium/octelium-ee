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
	"github.com/octelium/octelium/cluster/common/k8sretry"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
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
		return errors.New("nil job")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := ""
	if len(job.Spec.Template.Spec.Containers) > 0 && len(job.Spec.Template.Spec.Containers[0].Args) > 0 {
		cmd = job.Spec.Template.Spec.Containers[0].Args[0]
	}
	bound, err := install.BindGenesisJobTarget(ctx, k8sC, job)
	if err != nil {
		return errors.Wrap(err, "Could not validate the genesis target")
	}
	accepted, err := install.SubmitGenesisJobWithResult(ctx, k8sC, bound, nil)
	if err != nil {
		return errors.Wrapf(err, "could not create %s job %s", cmd, job.Name)
	}
	zap.L().Info("Created genesis job", zap.String("job", accepted.Name), zap.String("cmd", cmd))
	if err := Wait(ctx, k8sC, accepted, timeout); err != nil {
		return errors.Wrapf(err, "%s job %s did not complete successfully", cmd, accepted.Name)
	}
	zap.L().Info("Genesis job completed successfully", zap.String("job", accepted.Name), zap.String("cmd", cmd))
	return nil
}

func Wait(ctx context.Context, k8sC kubernetes.Interface, job *batchv1.Job, timeout time.Duration) error {
	if job == nil {
		return errors.New("nil job")
	}
	if job.Name == "" {
		return errors.New("empty job name")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	job = job.DeepCopy()
	if job.Namespace == "" {
		job.Namespace = vutils.K8sNS
	}
	if job.UID == "" {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		name, namespace := job.Name, job.Namespace
		err := k8sretry.Wait(ctx, 3*time.Second, timeout, func(ctx context.Context) (bool, error) {
			current, err := k8sC.BatchV1().Jobs(namespace).Get(ctx, name, k8smetav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if current.UID == "" {
				return false, errors.New("The genesis Job has no server-assigned identity")
			}
			job = current
			return true, nil
		})
		if err != nil {
			return err
		}
	}
	return install.WaitGenesisJob(ctx, k8sC, job, 3*time.Second, timeout)
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
		if cond.Type == batchv1.JobFailed && cond.Status == k8scorev1.ConditionTrue {
			return terminalErr("genesis job %s failed: reason=%s message=%s", jobName, cond.Reason, cond.Message)
		}
	}
	if job.DeletionTimestamp != nil {
		return terminalErr("genesis job %s is being deleted", jobName)
	}
	if job.Status.Active == 0 && (job.Status.Terminating == nil || *job.Status.Terminating == 0) {
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == k8scorev1.ConditionTrue {
				return nil
			}
		}
		completions := int32(1)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}
		if job.Status.Succeeded > 0 && job.Status.Succeeded >= completions {
			return nil
		}
	}

	return errors.Errorf(
		"genesis job %s is not complete yet: active=%d succeeded=%d failedAttempts=%d",
		jobName,
		job.Status.Active,
		job.Status.Succeeded,
		job.Status.Failed,
	)
}
