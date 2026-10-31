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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/client/octops/commands/install"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestGetJob(t *testing.T) {
	for _, spiffe := range []*install.SPIFFEOpts{nil, {CSIDriver: "test.csi", TrustDomain: "cluster.test"}} {
		job := GetJob(&Opts{
			Domain:  "example.com",
			Region:  "west",
			Package: "octeliumee",
			Cmd:     "join",
			Version: "v1.2.3",
			SPIFFE:  spiffe,
			Args:    []string{"--core-version", "2.0.0"},
		})
		assert.Equal(t, vutils.K8sNS, job.Namespace)
		assert.True(t, strings.HasPrefix(job.Name, "octelium-genesis-join-"))
		assert.Equal(t, "genesis", job.Labels["octelium.com/component"])
		assert.Equal(t, job.Labels, job.Spec.Template.Labels)
		pod := job.Spec.Template.Spec
		assert.Equal(t, "octelium-nocturne", pod.ServiceAccountName)
		require.Len(t, pod.Containers, 1)
		container := pod.Containers[0]
		assert.Equal(t, "octeliumee-genesis", container.Name)
		assert.True(t, strings.HasSuffix(container.Image, ":v1.2.3"), container.Image)
		assert.Equal(t, []string{"join", "--core-version", "2.0.0"}, container.Args)
		env := map[string]string{}
		for _, entry := range container.Env {
			env[entry.Name] = entry.Value
		}
		assert.Equal(t, "example.com", env["OCTELIUM_DOMAIN"])
		assert.Equal(t, "west", env["OCTELIUM_REGION_NAME"])
		if spiffe == nil {
			assert.Empty(t, pod.Volumes)
		} else {
			require.Len(t, pod.Volumes, 1)
			assert.Equal(t, "test.csi", pod.Volumes[0].CSI.Driver)
		}
	}

	{
		job := GetJob(&Opts{
			Domain:     "example.com",
			Region:     "west",
			Package:    "cordium",
			Cmd:        "init",
			Version:    "v1.2.3",
			SvcAccount: "octelium-genesis",
		})
		assert.True(t, strings.HasPrefix(job.Name, "octelium-genesis-init-"))
		assert.Equal(t, "octelium-genesis", job.Spec.Template.Spec.ServiceAccountName)
		assert.Equal(t, "cordium-genesis", job.Spec.Template.Spec.Containers[0].Name)
		assert.Equal(t, []string{"init"}, job.Spec.Template.Spec.Containers[0].Args)
	}
}

func TestJobStatusClassification(t *testing.T) {
	for _, tc := range []struct {
		name              string
		status            batchv1.JobStatus
		success, terminal bool
	}{
		{name: "Complete", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}}, success: true},
		{name: "Succeeded", status: batchv1.JobStatus{Succeeded: 1}, success: true},
		{name: "Active", status: batchv1.JobStatus{Active: 1}},
		{name: "Retrying", status: batchv1.JobStatus{Failed: 1}},
		{name: "FalseComplete", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionFalse}}}},
		{name: "FalseFailed", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionFalse}}}},
		{name: "TerminalFailure", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionTrue, Reason: "BackoffLimitExceeded", Message: "genesis failed"}}}, terminal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k8sC := fake.NewSimpleClientset(&batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade", Namespace: vutils.K8sNS}, Status: tc.status})
			err := Check(t.Context(), k8sC, "upgrade")
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tc.terminal, IsTerminalErr(err))
			if tc.terminal {
				assert.ErrorContains(t, err, "BackoffLimitExceeded")
				assert.ErrorContains(t, err, "genesis failed")
				assert.True(t, IsTerminalErr(fmt.Errorf("wrapped: %w", err)))
			}
		})
	}
}

func TestJobGetFailureIsRetriable(t *testing.T) {
	failure := errors.New("Kubernetes temporarily unavailable")
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("get", "jobs", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, failure })
	err := Check(t.Context(), k8sC, "upgrade")
	assert.ErrorIs(t, err, failure)
	assert.False(t, IsTerminalErr(err))
}

func TestWaitValidationAndCancellation(t *testing.T) {
	k8sC := fake.NewSimpleClientset()
	assert.Error(t, Wait(t.Context(), k8sC, nil, DefaultTimeout))
	assert.Error(t, Wait(t.Context(), k8sC, &batchv1.Job{}, DefaultTimeout))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, Wait(ctx, k8sC, &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade"}}, DefaultTimeout), context.Canceled)
	ctx, cancel = context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, Wait(ctx, k8sC, &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade"}}, DefaultTimeout), context.DeadlineExceeded)
	assert.ErrorIs(t, Wait(t.Context(), k8sC, &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade"}}, 20*time.Millisecond), context.DeadlineExceeded)
	assert.Empty(t, k8sC.Actions())
}

func TestWaitRetriesTransientKubernetesReadFailure(t *testing.T) {
	job := &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade", Namespace: vutils.K8sNS}, Status: batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}},
	}}
	k8sC := fake.NewSimpleClientset(job)
	calls := 0
	k8sC.PrependReactor("get", "jobs", func(ktesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls == 1 {
			return true, nil, errors.New("temporary API failure")
		}
		return false, nil, nil
	})
	require.NoError(t, Wait(t.Context(), k8sC, job, DefaultTimeout))
	assert.Equal(t, 2, calls)
}

func TestTerminalErrorUnwrap(t *testing.T) {
	failure := errors.New("genesis terminated")
	err := &terminalJobErr{err: failure}
	assert.ErrorIs(t, err, failure)
	assert.True(t, IsTerminalErr(fmt.Errorf("wrapped: %w", err)))
	var nilErr *terminalJobErr
	assert.NotEmpty(t, nilErr.Error())
	assert.Nil(t, nilErr.Unwrap())
	assert.NotEmpty(t, (&terminalJobErr{}).Error())
}

func TestRun(t *testing.T) {
	{
		assert.Error(t, Run(t.Context(), fake.NewSimpleClientset(), nil, DefaultTimeout))
	}

	{
		k8sC := fake.NewSimpleClientset()
		k8sC.PrependReactor("create", "jobs", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("forbidden") })
		err := Run(t.Context(), k8sC, GetJob(&Opts{Cmd: "join", Package: "octeliumee"}), DefaultTimeout)
		assert.ErrorContains(t, err, "could not create join job")
		assert.ErrorContains(t, err, "forbidden")
	}

	{
		k8sC := fake.NewSimpleClientset()
		k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
			job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
			job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}
			err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
			return true, job, err
		})
		err := Run(t.Context(), k8sC, GetJob(&Opts{Cmd: "join", Package: "octeliumee"}), DefaultTimeout)
		assert.ErrorContains(t, err, "join job")
		assert.ErrorContains(t, err, "BackoffLimitExceeded")
		assert.ErrorContains(t, err, "did not complete successfully")
	}

	{
		k8sC := fake.NewSimpleClientset()
		k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
			job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
			job.Status.Succeeded = 1
			err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
			return true, job, err
		})
		require.NoError(t, Run(t.Context(), k8sC, GetJob(&Opts{Cmd: "init", Package: "cordium"}), DefaultTimeout))
	}
}
