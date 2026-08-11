// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/scenario"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	genesisBudget    = 20 * time.Minute
	deploymentBudget = 10 * time.Minute

	pollInterval    = 3 * time.Second
	pollReportEvery = 15 * time.Second
)

// stepWaitDeployments waits for the enterprise package to land.
// `octops install-package` only creates the genesis Job and returns, so the
// package is installed asynchronously.
func stepWaitDeployments(ctx context.Context, r *scenario.Runner) error {
	k8sC, err := r.K8sC()
	if err != nil {
		return err
	}

	if err := waitGenesis(ctx, k8sC); err != nil {
		return err
	}

	for _, name := range Deployments {
		zap.L().Debug("Waiting for the enterprise Deployment readiness",
			zap.String("deployment", name))

		err := pollUntil(ctx, fmt.Sprintf("the deployment %s", name), deploymentBudget,
			func(ctx context.Context) error {
				return deploymentReadiness(ctx, k8sC, name)
			})
		if err != nil {
			return err
		}
	}

	return nil
}

func waitGenesis(ctx context.Context, k8sC kubernetes.Interface) error {
	prefix := fmt.Sprintf("octelium-genesis-init-%s-", Package)

	return pollUntil(ctx, fmt.Sprintf("the %s genesis Job", Package), genesisBudget,
		func(ctx context.Context) error {
			job, err := latestJob(ctx, k8sC, prefix)
			if err != nil {
				return err
			}

			for _, c := range job.Status.Conditions {
				if c.Status != k8scorev1.ConditionTrue {
					continue
				}
				switch c.Type {
				case batchv1.JobComplete:
					return nil
				case batchv1.JobFailed:
					return fatal(errors.Errorf("The Job %s failed: %s %s",
						job.Name, c.Reason, c.Message))
				}
			}

			return errors.Errorf("The Job %s is still running (%d active, %d failed)",
				job.Name, job.Status.Active, job.Status.Failed)
		})
}

func latestJob(ctx context.Context,
	k8sC kubernetes.Interface, prefix string) (*batchv1.Job, error) {
	jobList, err := k8sC.BatchV1().Jobs(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{
		LabelSelector: "octelium.com/component=genesis",
	})
	if err != nil {
		return nil, err
	}

	var ret *batchv1.Job
	for i := range jobList.Items {
		job := &jobList.Items[i]
		if !strings.HasPrefix(job.Name, prefix) {
			continue
		}
		if ret == nil || job.CreationTimestamp.After(ret.CreationTimestamp.Time) {
			ret = job
		}
	}

	if ret == nil {
		return nil, errors.Errorf("No Job named %s* has been created yet", prefix)
	}

	return ret, nil
}

func deploymentReadiness(ctx context.Context,
	k8sC kubernetes.Interface, name string) error {
	dep, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	rs, err := k8sutils.GetNewReplicaSet(dep, k8sC.AppsV1())
	if err != nil {
		return err
	}
	if rs == nil {
		return errors.Errorf("The Deployment has no current ReplicaSet yet")
	}

	want := *dep.Spec.Replicas - k8sutils.MaxUnavailable(*dep)
	if rs.Status.ReadyReplicas >= want {
		return nil
	}

	return errors.Errorf("%d of %d replicas are ready",
		rs.Status.ReadyReplicas, want)
}

type fatalError struct {
	err error
}

func (f *fatalError) Error() string { return f.err.Error() }

func fatal(err error) error { return &fatalError{err: err} }

func pollUntil(ctx context.Context, what string,
	budget time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	started := time.Now()
	reported := time.Now()
	attempts := 0

	var lastErr error
	for {
		attempts++
		lastErr = fn(ctx)
		if lastErr == nil {
			zap.L().Debug("Condition met",
				zap.String("what", what),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", time.Since(started)))
			return nil
		}

		var f *fatalError
		if errors.As(lastErr, &f) {
			return errors.Errorf("Gave up waiting for %s after %s: %+v",
				what, time.Since(started).Truncate(time.Millisecond), f.err)
		}

		if time.Since(reported) >= pollReportEvery {
			reported = time.Now()
			zap.L().Info("Still waiting",
				zap.String("what", what),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", time.Since(started).Truncate(time.Second)),
				zap.Duration("budget", budget),
				zap.NamedError("lastError", lastErr))
		}

		select {
		case <-ctx.Done():
			return errors.Errorf(
				"Timed out after %s waiting for %s (%d attempts). Last error: %+v",
				time.Since(started).Truncate(time.Millisecond), what, attempts, lastErr)
		case <-time.After(pollInterval):
		}
	}
}
