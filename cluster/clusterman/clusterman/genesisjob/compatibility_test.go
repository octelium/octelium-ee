package genesisjob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func completedJob() *batchv1.Job {
	return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "accepted-operation", Namespace: vutils.K8sNS, UID: "accepted-uid"}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}
}

func TestWaitFollowsSubmittedJob(t *testing.T) {
	for _, state := range []string{"complete", "retry-then-complete", "counter-only", "active", "terminating-pods", "failed", "contradictory", "replaced", "deleted", "terminating", "forbidden", "legacy-list-permissions"} {
		t.Run(state, func(t *testing.T) {
			expected := completedJob()
			current := expected.DeepCopy()
			switch state {
			case "retry-then-complete":
				current.Status.Failed = 2
			case "counter-only":
				current.Status.Conditions = nil
				current.Status.Succeeded = 1
			case "active":
				current.Status.Active = 1
			case "terminating-pods":
				current.Status.Terminating = new(int32(1))
			case "failed":
				current.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}
			case "contradictory":
				current.Status.Conditions = append(current.Status.Conditions, batchv1.JobCondition{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"})
			case "replaced":
				current.UID = "replacement"
			case "terminating":
				current.DeletionTimestamp = new(metav1.Now())
			}
			c := fake.NewSimpleClientset(current)
			if state == "deleted" {
				require.NoError(t, c.Tracker().Delete(batchv1.SchemeGroupVersion.WithResource("jobs"), vutils.K8sNS, current.Name))
			}
			if state == "forbidden" || state == "legacy-list-permissions" {
				c.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, k8serr.NewForbidden(action.GetResource().GroupResource(), expected.Name, io.EOF)
				})
			}
			if state == "forbidden" {
				c.PrependReactor("list", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, k8serr.NewForbidden(action.GetResource().GroupResource(), expected.Name, io.EOF)
				})
			}
			ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
			defer cancel()
			err := Wait(ctx, c, expected, DefaultTimeout)
			switch state {
			case "complete", "retry-then-complete", "legacy-list-permissions":
				require.NoError(t, err)
			case "counter-only", "active", "terminating-pods":
				require.ErrorIs(t, err, context.DeadlineExceeded)
			case "failed", "contradictory":
				require.ErrorContains(t, err, "BackoffLimitExceeded")
			default:
				require.Error(t, err)
			}
			for _, action := range c.Actions() {
				if action.GetResource().Resource == "jobs" {
					if action.GetVerb() == "get" {
						assert.Equal(t, expected.Name, action.(k8stesting.GetAction).GetName())
					} else {
						assert.Equal(t, "list", action.GetVerb())
						assert.Equal(t, "metadata.name="+expected.Name, action.(k8stesting.ListAction).GetListRestrictions().Fields.String())
					}
				} else {
					assert.Equal(t, "pods", action.GetResource().Resource)
					assert.Equal(t, "list", action.GetVerb())
				}
			}
		})
	}
}

func TestCheckLegacyCompletionSafety(t *testing.T) {
	for _, state := range []string{"legacy-complete", "active", "terminating", "deleted", "failed-and-complete", "partial-completions", "retried"} {
		t.Run(state, func(t *testing.T) {
			job := completedJob()
			job.Status.Conditions = nil
			job.Status.Succeeded = 1
			switch state {
			case "active":
				job.Status.Active = 1
			case "terminating":
				job.Status.Terminating = new(int32(1))
			case "deleted":
				job.DeletionTimestamp = new(metav1.Now())
			case "failed-and-complete":
				job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}, {Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
			case "partial-completions":
				job.Spec.Completions = new(int32(2))
			case "retried":
				job.Status.Failed = 3
			}
			err := Check(t.Context(), fake.NewSimpleClientset(job), job.Name)
			if state == "legacy-complete" || state == "retried" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, state == "deleted" || state == "failed-and-complete", IsTerminalErr(err))
		})
	}
}

func TestRunReconcilesUncertainCreation(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			c := fake.NewSimpleClientset()
			creates, readbacks := 0, 0
			c.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
				creates++
				job := action.(k8stesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
				job.UID = "accepted-job"
				job.Status = completedJob().Status
				if committed || creates > 1 {
					require.NoError(t, c.Tracker().Create(action.GetResource(), job, job.Namespace))
				}
				if creates == 1 {
					return true, nil, io.ErrUnexpectedEOF
				}
				return true, job, nil
			})
			c.PrependReactor("get", "jobs", func(k8stesting.Action) (bool, runtime.Object, error) {
				readbacks++
				if readbacks == 1 {
					return true, nil, k8serr.NewServiceUnavailable("readback unavailable")
				}
				return false, nil, nil
			})
			desired := GetJob(&Opts{Cmd: "upgrade", Package: "octelium", Domain: "example.com", Region: "default", Version: "2.0.0"})
			require.NoError(t, Run(t.Context(), c, desired, DefaultTimeout))
			if committed {
				assert.Equal(t, 1, creates)
			} else {
				assert.Equal(t, 2, creates)
			}
			jobs, err := c.BatchV1().Jobs(vutils.K8sNS).List(t.Context(), metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, jobs.Items, 1)
			assert.Equal(t, desired.Name, jobs.Items[0].Name)
			assert.Equal(t, desired.Spec, jobs.Items[0].Spec)
			for _, action := range c.Actions() {
				assert.NotContains(t, []string{"update", "delete", "patch"}, action.GetVerb())
			}
		})
	}
}

func TestRunRejectsForeignTargetBeforeCreate(t *testing.T) {
	annotations := map[string]string{
		"octelium.com/genesis-cluster-uid":   "target-cluster",
		"octelium.com/genesis-domain":        "example.com",
		"octelium.com/genesis-region-uid":    "target-region",
		"octelium.com/genesis-region-name":   "west",
		"octelium.com/genesis-namespace-uid": "target-namespace",
	}
	for _, pin := range []string{"OCTELIUM_GENESIS_CLUSTER_UID", "OCTELIUM_GENESIS_REGION_UID"} {
		t.Run(pin, func(t *testing.T) {
			c := fake.NewSimpleClientset(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "octelium-installation", Namespace: vutils.K8sNS, Annotations: annotations}})
			job := GetJob(&Opts{Cmd: "upgrade", Package: "cordium", Domain: "example.com", Region: "west", Version: "2.0.0"})
			job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: pin, Value: "source-identity"})
			err := Run(t.Context(), c, job, DefaultTimeout)
			require.ErrorContains(t, err, "does not match")
			for _, action := range c.Actions() {
				assert.Equal(t, "get", action.GetVerb())
			}
		})
	}
}

func TestRunHTTPUncertainAcknowledgement(t *testing.T) {
	var mu sync.Mutex
	var stored *batchv1.Job
	creates, reads, unexpected := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && (strings.HasSuffix(r.URL.Path, "/configmaps/octelium-installation") || strings.HasSuffix(r.URL.Path, "/secrets/octelium-init")):
			w.WriteHeader(http.StatusNotFound)
			assert.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Failure", Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound}))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/jobs"):
			assert.NoError(t, json.NewEncoder(w).Encode(&batchv1.JobList{TypeMeta: metav1.TypeMeta{Kind: "JobList", APIVersion: "batch/v1"}, Items: []batchv1.Job{}}))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pods"):
			assert.NoError(t, json.NewEncoder(w).Encode(&corev1.PodList{TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"}, Items: []corev1.Pod{}}))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/jobs"):
			creates++
			job := &batchv1.Job{}
			if err := json.NewDecoder(r.Body).Decode(job); !assert.NoError(t, err) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			job.UID = "server-assigned-uid"
			job.TypeMeta = metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"}
			job.Status = completedJob().Status
			stored = job
			w.WriteHeader(http.StatusServiceUnavailable)
			assert.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Failure", Reason: metav1.StatusReasonServiceUnavailable, Code: http.StatusServiceUnavailable}))
		case r.Method == http.MethodGet && stored != nil && strings.HasSuffix(r.URL.Path, "/jobs/"+stored.Name):
			reads++
			assert.NoError(t, json.NewEncoder(w).Encode(stored))
		default:
			unexpected++
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	c, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, QPS: 100, Burst: 100, ContentConfig: rest.ContentConfig{ContentType: "application/json"}})
	require.NoError(t, err)
	job := GetJob(&Opts{Cmd: "upgrade", Package: "octeliumee", Domain: "example.com", Region: "default", Version: "2.0.0"})
	require.NoError(t, Run(t.Context(), c, job, 10*time.Second))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, creates)
	assert.GreaterOrEqual(t, reads, 2)
	assert.Zero(t, unexpected)
	require.NotNil(t, stored)
	assert.Equal(t, job.Name, stored.Name)
	assert.Equal(t, job.Spec, stored.Spec)
}
