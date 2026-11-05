// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/main/corev1"
	pkgmetav1 "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	gcomponents "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/cluster/genesis/genesis/genesisutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func TestRscServerDeployment(t *testing.T) {

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)

	opts := &components.CommonOpts{
		CommonOpts: gcomponents.CommonOpts{
			K8sC:          fakeC.K8sC,
			ClusterConfig: cc,
		},
	}
	err = components.CreateRscServer(ctx, opts)
	assert.Nil(t, err)

	g := &Genesis{}
	g.k8sC = fakeC.K8sC

	/*
		err = g.setSecretManEnvVars(ctx)
		assert.Nil(t, err)

		depList, err := g.k8sC.AppsV1().Deployments(vutils.K8sNS).List(ctx, v1.ListOptions{
			LabelSelector: "octelium.com/component=rscserver",
		})
		assert.Nil(t, err)
		assert.True(t, len(depList.Items) > 0)

		for _, dep := range depList.Items {
			assert.True(t, slices.ContainsFunc(dep.Spec.Template.Spec.Containers, func(c k8scorev1.Container) bool {
				return slices.ContainsFunc(c.Env, func(e k8scorev1.EnvVar) bool {
					return e.Name == "OCTELIUM_USE_SECRETMAN"
				})
			}))

			assert.True(t, slices.ContainsFunc(dep.Spec.Template.Spec.Containers, func(c k8scorev1.Container) bool {
				return slices.ContainsFunc(c.Env, func(e k8scorev1.EnvVar) bool {
					return e.Name == "OCTELIUM_SECRETMAN_ADDR"
				})
			}))
		}
	*/
}

func TestRscServerRolloutCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, (&Genesis{}).rolloutRestartRscServers(ctx), context.Canceled)
	for _, mode := range []string{"between-writes", "during-delay", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			tst, err := otests.Initialize(nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tst.Destroy()) })
			c := fake.NewSimpleClientset()
			for _, name := range []string{"first", "second"} {
				require.NoError(t, c.Tracker().Add(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "octelium", Labels: map[string]string{"octelium.com/component": "rscserver"}}}))
			}
			g := &Genesis{k8sC: c, octeliumCInit: tst.C.OcteliumC}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "deadline" {
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			writes := 0
			c.PrependReactor("update", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
				writes++
				if writes == 1 {
					if mode == "between-writes" {
						cancel()
					} else if mode == "during-delay" {
						time.AfterFunc(25*time.Millisecond, cancel)
					}
				}
				return false, nil, nil
			})
			started := time.Now()
			err = g.rolloutRestartRscServers(ctx)
			if mode == "deadline" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			} else {
				require.ErrorIs(t, err, context.Canceled)
			}
			assert.Less(t, time.Since(started), time.Second)
			if mode == "between-writes" {
				assert.Equal(t, 1, writes)
			} else {
				assert.Equal(t, 2, writes)
			}
		})
	}
}

func newCompletionProxy(t *testing.T, before, after func(string, []byte) error) {
	t.Helper()
	connection, err := grpc.NewClient(net.JoinHostPort("localhost", os.Getenv("OCTELIUM_TEST_RSCSERVER_PORT")), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		method, _ := grpc.MethodFromServerStream(stream)
		request, response := &emptypb.Empty{}, &emptypb.Empty{}
		if err := stream.RecvMsg(request); err != nil {
			return err
		}
		data, err := proto.Marshal(request)
		if err != nil {
			return err
		}
		if err := before(method, data); err != nil {
			return err
		}
		if err := connection.Invoke(stream.Context(), method, request, response); err != nil {
			return err
		}
		if err := after(method, data); err != nil {
			return err
		}
		return stream.SendMsg(response)
	}))
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		<-done
		require.NoError(t, connection.Close())
	})
	t.Setenv("OCTELIUM_TEST_RSCSERVER_PORT", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))
	t.Setenv("OCTELIUM_TEST_CORDIUM_RSCSERVER_PORT", "")
}

func TestPackageCompletionRecoveryE2E(t *testing.T) {
	for _, point := range []string{"init-version-denied", "upgrade-version-denied", "upgrade-version-conflict", "upgrade-version-response-lost", "init-version-response-lost", "upgrade-resource-denied"} {
		t.Run(point, func(t *testing.T) {
			t.Setenv("OCTELIUM_DOMAIN", "example.com")
			t.Setenv("OCTELIUM_REGION_NAME", "default")
			for _, name := range []string{"OCTELIUM_GENESIS_CLUSTER_UID", "OCTELIUM_GENESIS_REGION_UID", "OCTELIUM_GENESIS_NAMESPACE_UID"} {
				t.Setenv(name, "")
			}
			tst, err := otests.Initialize(nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tst.Destroy()) })
			for _, name := range []string{"octelium", "octelium-api"} {
				require.NoError(t, genesisutils.CreateOrUpdateNamespace(t.Context(), tst.C.OcteliumC, &corev1.Namespace{Metadata: &pkgmetav1.Metadata{Name: name, IsSystem: true}}))
			}
			region, err := tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Name: "default"})
			require.NoError(t, err)
			bootstrap := &k8scorev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "octelium-init", Namespace: vutils.K8sNS, UID: "bootstrap-uid"}, Data: map[string][]byte{
				"region": pbutils.MarshalMust(region), "domain": []byte("example.com"), "genesis-input-key": []byte("01234567890123456789012345678901"),
			}}
			cm := &k8scorev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "octelium-installation", Namespace: vutils.K8sNS}, Data: map[string]string{"custom": "preserved"}}
			k8sC := fake.NewSimpleClientset(&k8scorev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vutils.K8sNS, UID: "namespace-uid"}}, bootstrap, cm)
			g := &Genesis{k8sC: k8sC}
			require.NoError(t, g.RunInit(t.Context(), nil))
			secrets, err := k8sC.CoreV1().Secrets(vutils.K8sNS).List(t.Context(), metav1.ListOptions{})
			require.NoError(t, err)
			coreSecrets, err := tst.C.OcteliumC.CoreC().ListSecret(t.Context(), &rmetav1.ListOptions{})
			require.NoError(t, err)

			region, err = tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Name: "default"})
			require.NoError(t, err)
			region.Status.VersionInfoMap["octeliumee"] = &corev1.Region_Status_VersionInfo{Package: "octeliumee", Version: "previous", Id: "previous-operation"}
			region.Status.VersionInfoMap["peer"] = &corev1.Region_Status_VersionInfo{Package: "peer", Version: "peer-version", Id: "peer-operation"}
			if region.Metadata.Labels == nil {
				region.Metadata.Labels = make(map[string]string)
			}
			region.Metadata.Labels["custom"] = "preserved"
			region, err = tst.C.OcteliumC.CoreC().UpdateRegion(t.Context(), region)
			require.NoError(t, err)
			previous := proto.Clone(region.Status.VersionInfoMap["octeliumee"])
			coreVersion := region.Status.Version
			t.Setenv("OCTELIUM_INSTALL_ID", "new-operation")
			var failing atomic.Bool
			failing.Store(true)
			var hits, commits atomic.Int32
			isPublication := func(method string, data []byte) bool {
				if !strings.HasSuffix(method, "/UpdateRegion") {
					return false
				}
				candidate := &corev1.Region{}
				if proto.Unmarshal(data, candidate) != nil {
					return false
				}
				return candidate.GetStatus().GetVersionInfoMap()["octeliumee"].GetId() == "new-operation"
			}
			newCompletionProxy(t, func(method string, data []byte) error {
				if !failing.Load() {
					return nil
				}
				if point == "upgrade-resource-denied" && strings.HasSuffix(method, "/GetDNSProvider") {
					hits.Add(1)
					return status.Error(codes.PermissionDenied, "required package resource denied")
				}
				if !isPublication(method, data) {
					return nil
				}
				if strings.HasSuffix(point, "version-denied") {
					hits.Add(1)
					return status.Error(codes.PermissionDenied, "version publication denied")
				}
				if strings.HasSuffix(point, "version-conflict") && hits.Add(1) == 1 {
					current, err := tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Uid: region.Metadata.Uid})
					if err != nil {
						return err
					}
					current.Status.VersionInfoMap["peer"].Version = "concurrent-peer-version"
					current.Metadata.Labels["concurrent"] = "preserved"
					if _, err := tst.C.OcteliumC.CoreC().UpdateRegion(t.Context(), current); err != nil {
						return err
					}
					return status.Error(codes.OutOfRange, "concurrent Region change")
				}
				return nil
			}, func(method string, data []byte) error {
				if !isPublication(method, data) {
					return nil
				}
				commits.Add(1)
				if failing.Load() && strings.HasSuffix(point, "version-response-lost") && hits.Add(1) == 1 {
					return status.Error(codes.Unavailable, "committed publication response lost")
				}
				return nil
			})
			run := func() error {
				if strings.HasPrefix(point, "init-") {
					return g.RunInit(t.Context(), nil)
				}
				return g.RunUpgrade(t.Context(), nil)
			}
			err = run()
			current, readErr := tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Uid: region.Metadata.Uid})
			require.NoError(t, readErr)
			if strings.HasSuffix(point, "denied") {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "denied")
				assert.True(t, proto.Equal(previous, current.Status.VersionInfoMap["octeliumee"]))
				assert.Zero(t, commits.Load())
				failing.Store(false)
				require.NoError(t, run())
			} else {
				require.NoError(t, err)
			}
			assert.Positive(t, hits.Load())
			assert.EqualValues(t, 1, commits.Load())
			current, err = tst.C.OcteliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Uid: region.Metadata.Uid})
			require.NoError(t, err)
			assert.Equal(t, region.Metadata.Uid, current.Metadata.Uid)
			assert.Equal(t, "preserved", current.Metadata.Labels["custom"])
			assert.Equal(t, coreVersion, current.Status.Version)
			assert.Equal(t, ldflags.GetVersion(), current.Status.VersionInfoMap["octeliumee"].Version)
			assert.Equal(t, "new-operation", current.Status.VersionInfoMap["octeliumee"].Id)
			if strings.HasSuffix(point, "version-conflict") {
				assert.Equal(t, "preserved", current.Metadata.Labels["concurrent"])
				assert.Equal(t, "concurrent-peer-version", current.Status.VersionInfoMap["peer"].Version)
			} else {
				assert.Equal(t, "peer-version", current.Status.VersionInfoMap["peer"].Version)
			}
			for _, secret := range secrets.Items {
				stored, err := k8sC.CoreV1().Secrets(vutils.K8sNS).Get(t.Context(), secret.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secret.UID, stored.UID)
				assert.Equal(t, secret.Data, stored.Data)
			}
			for _, secret := range coreSecrets.Items {
				stored, err := tst.C.OcteliumC.CoreC().GetSecret(t.Context(), &rmetav1.GetOptions{Uid: secret.Metadata.Uid})
				require.NoError(t, err)
				assert.True(t, proto.Equal(secret.Data, stored.Data))
				assert.True(t, proto.Equal(secret.Spec, stored.Spec))
			}

			storedCM, err := k8sC.CoreV1().ConfigMaps(vutils.K8sNS).Get(t.Context(), cm.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, cm, storedCM)
		})
	}
}

func readyGatewayDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "octelium-gwagent", Namespace: vutils.K8sNS, UID: "gateway-daemonset", Generation: 2},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
			Template:       k8scorev1.PodTemplateSpec{Spec: k8scorev1.PodSpec{NodeSelector: map[string]string{vutils.NodeLabelDataPlane: ""}}},
		},
		Status: appsv1.DaemonSetStatus{ObservedGeneration: 2, CurrentNumberScheduled: 2, DesiredNumberScheduled: 2, UpdatedNumberScheduled: 2, NumberReady: 2, NumberAvailable: 2},
	}
}

func registeredGatewayNode(name string) *k8scorev1.Node {
	return &k8scorev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{vutils.NodeLabelDataPlane: "", vutils.NodeLabelGatewayRegistered: "true"}}}
}

func TestGatewayReadinessDoesNotTrustOldRegistration(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, state := range []string{"complete", "empty", "unobserved", "stale", "old-pods", "unavailable", "unregistered", "missing-label", "wrong-label-value"} {
		t.Run(state, func(t *testing.T) {
			ds := readyGatewayDaemonSet()
			node := registeredGatewayNode("node")
			switch state {
			case "empty":
				ds.Status = appsv1.DaemonSetStatus{ObservedGeneration: 2}
			case "unobserved":
				ds.Status = appsv1.DaemonSetStatus{}
			case "stale":
				ds.Status.ObservedGeneration--
			case "old-pods":
				ds.Status.UpdatedNumberScheduled = 1
			case "unavailable":
				ds.Status.NumberAvailable = 1
			case "unregistered":
				node.Labels[vutils.NodeLabelGatewayRegistered] = "false"
			case "missing-label":
				delete(node.Labels, vutils.NodeLabelGatewayRegistered)
			case "wrong-label-value":
				node.Labels[vutils.NodeLabelDataPlane] = "true"
			}
			c := fake.NewSimpleClientset(ds, node)
			ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
			defer cancel()
			err := waitGatewayAgentsRegistered(ctx, c)
			if state == "complete" {
				require.NoError(t, err)
			} else if state == "wrong-label-value" {
				require.ErrorContains(t, err, "No Kubernetes node is labeled")
			} else {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
			for _, action := range c.Actions() {
				assert.Contains(t, []string{"get", "list"}, action.GetVerb())
				if action.GetResource().Resource == "nodes" {
					assert.Contains(t, []string{"complete", "unregistered", "missing-label", "wrong-label-value"}, state)
					assert.Equal(t, vutils.NodeLabelDataPlane+"=", action.(k8stesting.ListAction).GetListRestrictions().Labels.String())
				}
			}
		})
	}
}

func TestGatewayReadinessErrorsAndCancellation(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, check := range []func(context.Context, kubernetes.Interface) error{waitGatewayAgentReady, waitGatewayAgentsRegistered} {
		c := fake.NewSimpleClientset()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, check(ctx, c), context.Canceled)
		assert.Empty(t, c.Actions())
		want := k8serr.NewForbidden(appsv1.Resource("daemonsets"), "octelium-gwagent", fmt.Errorf("denied"))
		c.PrependReactor("get", "daemonsets", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, want })
		require.ErrorIs(t, check(t.Context(), c), want)
		assert.Len(t, c.Actions(), 1)
	}
	c := fake.NewSimpleClientset(readyGatewayDaemonSet(), registeredGatewayNode("node"))
	want := k8serr.NewForbidden(k8scorev1.Resource("nodes"), "", fmt.Errorf("denied"))
	c.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, want })
	require.ErrorIs(t, waitGatewayAgentsRegistered(t.Context(), c), want)
	ldflags.TestMode = "true"
	c.ClearActions()
	require.NoError(t, waitGatewayAgentReady(t.Context(), c))
	require.NoError(t, waitGatewayAgentsRegistered(t.Context(), c))
	assert.Empty(t, c.Actions())
}

func TestGatewayReadyRequiresDataPlaneNodes(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	ds := readyGatewayDaemonSet()
	ds.Status = appsv1.DaemonSetStatus{ObservedGeneration: 2}
	node := registeredGatewayNode("node")
	node.Labels[vutils.NodeLabelDataPlane] = "true"
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.ErrorContains(t, waitGatewayAgentReady(ctx, fake.NewSimpleClientset(ds, node)), "No Kubernetes node is labeled")
	require.NoError(t, ctx.Err())
}

func TestGatewayReadinessHTTP(t *testing.T) {
	previous := ldflags.TestMode
	ldflags.TestMode = "false"
	t.Cleanup(func() { ldflags.TestMode = previous })
	for _, state := range []string{"empty", "stale", "old-pods", "unavailable", "missing", "daemonset-outage", "node-outage", "registration", "stalled"} {
		t.Run(state, func(t *testing.T) {
			var reads, nodeReads atomic.Int32
			finished := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/apis/apps/v1/namespaces/octelium/daemonsets/octelium-gwagent":
					n := reads.Add(1)
					if state == "stalled" {
						<-r.Context().Done()
						close(finished)
						return
					}
					if n == 1 && (state == "daemonset-outage" || state == "missing") {
						code, reason := http.StatusServiceUnavailable, metav1.StatusReasonServiceUnavailable
						if state == "missing" {
							code, reason = http.StatusNotFound, metav1.StatusReasonNotFound
						}
						w.WriteHeader(code)
						assert.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusFailure, Reason: reason, Code: int32(code)}))
						return
					}
					ds := readyGatewayDaemonSet()
					if n == 1 {
						switch state {
						case "empty":
							ds.Status = appsv1.DaemonSetStatus{ObservedGeneration: 2}
						case "stale":
							ds.Status.ObservedGeneration--
						case "old-pods":
							ds.Status.UpdatedNumberScheduled = 1
						case "unavailable":
							ds.Status.NumberAvailable = 1
						}
					}
					assert.NoError(t, json.NewEncoder(w).Encode(ds))
				case "/api/v1/nodes":
					n := nodeReads.Add(1)
					assert.Equal(t, vutils.NodeLabelDataPlane+"=", r.URL.Query().Get("labelSelector"))
					if state == "node-outage" && n == 1 {
						w.WriteHeader(http.StatusServiceUnavailable)
						assert.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusFailure, Reason: metav1.StatusReasonServiceUnavailable, Code: http.StatusServiceUnavailable}))
						return
					}
					node := registeredGatewayNode("node")
					if state == "registration" && n == 1 {
						node.Labels[vutils.NodeLabelGatewayRegistered] = "false"
					}
					assert.NoError(t, json.NewEncoder(w).Encode(&k8scorev1.NodeList{Items: []k8scorev1.Node{*node, *registeredGatewayNode("second-node")}}))
				default:
					t.Errorf("Unexpected Kubernetes request: %s", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
				}
			}))
			defer server.Close()
			c, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
			require.NoError(t, err)
			timeout := 7 * time.Second
			if state == "stalled" {
				timeout = 80 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(t.Context(), timeout)
			defer cancel()
			if state == "empty" {
				err = waitGatewayAgentReady(ctx, c)
			} else {
				err = waitGatewayAgentsRegistered(ctx, c)
			}
			if state == "stalled" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				select {
				case <-finished:
				case <-time.After(time.Second):
					t.Fatal("The gateway readiness request did not observe cancellation")
				}
			} else {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, reads.Load(), int32(2))
				if state != "empty" {
					assert.GreaterOrEqual(t, nodeReads.Load(), int32(1))
				}
			}
		})
	}
}

type versionTestClient struct {
	octeliumc.ClientInterface
	core *versionTestCore
}

func (c *versionTestClient) CoreC() rcorev1.ResourceServiceClient { return c.core }

type versionTestCore struct {
	rcorev1.ResourceServiceClient
	region  *corev1.Region
	gets    int
	updates int
	get     func(context.Context, *rmetav1.GetOptions) error
	update  func(context.Context, *corev1.Region) error
}

func (c *versionTestCore) GetRegion(ctx context.Context, opts *rmetav1.GetOptions, _ ...grpc.CallOption) (*corev1.Region, error) {
	c.gets++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.get != nil {
		if err := c.get(ctx, opts); err != nil {
			return nil, err
		}
	}
	return proto.Clone(c.region).(*corev1.Region), nil
}

func (c *versionTestCore) UpdateRegion(ctx context.Context, region *corev1.Region, _ ...grpc.CallOption) (*corev1.Region, error) {
	c.updates++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.update != nil {
		if err := c.update(ctx, region); err != nil {
			return nil, err
		}
	}
	c.region = proto.Clone(region).(*corev1.Region)
	c.region.Metadata.ResourceVersion = fmt.Sprintf("rv-%d", c.updates)
	return proto.Clone(c.region).(*corev1.Region), nil
}

func newVersionTestGenesis(t *testing.T) (*Genesis, *versionTestCore) {
	t.Helper()
	t.Setenv("OCTELIUM_INSTALL_ID", "existing-operation-id")
	c := &versionTestCore{region: &corev1.Region{
		Metadata: &pkgmetav1.Metadata{Name: "west", Uid: "region-uid", ResourceVersion: "rv-0", Labels: map[string]string{"custom": "preserved"}},
		Spec:     &corev1.Region_Spec{},
		Status: &corev1.Region_Status{
			Version: "old-core",
			VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{
				"octelium": {Package: "octelium", Version: "old-core", Id: "core-operation"},
				"other":    {Package: "other", Version: "other-version", Id: "other-operation"},
			},
		},
	}}
	client := &versionTestClient{core: c}
	return &Genesis{octeliumC: client, octeliumCInit: client}, c
}

func TestPackageVersionPublicationPreservesState(t *testing.T) {
	for _, mode := range []string{"existing", "empty-map", "empty-status", "name-only", "legacy-empty-install-id"} {
		t.Run(mode, func(t *testing.T) {
			g, c := newVersionTestGenesis(t)
			switch mode {
			case "empty-map":
				c.region.Status.VersionInfoMap = nil
			case "empty-status":
				c.region.Status = nil
			case "name-only":
				c.region.Metadata.Uid = ""
			case "legacy-empty-install-id":
				t.Setenv("OCTELIUM_INSTALL_ID", "")
			}
			before := proto.Clone(c.region).(*corev1.Region)
			c.get = func(_ context.Context, opts *rmetav1.GetOptions) error {
				if mode == "name-only" {
					assert.Equal(t, before.Metadata.Name, opts.Name)
				} else {
					assert.Equal(t, before.Metadata.Uid, opts.Uid)
				}
				return nil
			}
			require.NoError(t, g.setRegionVersionMap(t.Context(), before))
			assert.Equal(t, 1, c.updates)
			expectedMetadata := proto.Clone(before.Metadata).(*pkgmetav1.Metadata)
			expectedMetadata.ResourceVersion = "rv-1"
			assert.True(t, proto.Equal(expectedMetadata, c.region.Metadata))
			assert.Equal(t, before.Metadata.Labels, c.region.Metadata.Labels)
			assert.True(t, proto.Equal(before.Spec, c.region.Spec))
			assert.Equal(t, before.GetStatus().GetVersion(), c.region.Status.Version)
			for key, info := range before.GetStatus().GetVersionInfoMap() {
				assert.True(t, proto.Equal(info, c.region.Status.VersionInfoMap[key]))
			}
			info := c.region.Status.VersionInfoMap["octeliumee"]
			require.NotNil(t, info)
			assert.Equal(t, "octeliumee", info.Package)
			assert.Equal(t, ldflags.GetVersion(), info.Version)
			if mode == "legacy-empty-install-id" {
				assert.Empty(t, info.Id)
			} else {
				assert.Equal(t, "existing-operation-id", info.Id)
			}
			assert.NotNil(t, info.SetAt)
		})
	}
}

func TestPackageVersionPublicationRetries(t *testing.T) {
	for _, mode := range []string{"read-outage", "write-outage", "conflict", "response-lost"} {
		t.Run(mode, func(t *testing.T) {
			g, c := newVersionTestGenesis(t)
			input := proto.Clone(c.region).(*corev1.Region)
			var first *corev1.Region_Status_VersionInfo
			c.get = func(context.Context, *rmetav1.GetOptions) error {
				if mode == "read-outage" && c.gets == 1 {
					return status.Error(codes.Unavailable, "temporary read failure")
				}
				return nil
			}
			c.update = func(ctx context.Context, proposed *corev1.Region) error {
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				assert.LessOrEqual(t, time.Until(deadline), 30*time.Second)
				if first == nil {
					first = proto.Clone(proposed.Status.VersionInfoMap["octeliumee"]).(*corev1.Region_Status_VersionInfo)
				} else {
					assert.True(t, proto.Equal(first, proposed.Status.VersionInfoMap["octeliumee"]))
				}
				if c.updates != 1 {
					return nil
				}
				switch mode {
				case "write-outage":
					return status.Error(codes.Unavailable, "temporary write failure")
				case "conflict":
					c.region.Metadata.Labels["concurrent"] = "preserved"
					c.region.Status.VersionInfoMap["other"].Version = "new-peer-version"
					c.region.Metadata.ResourceVersion = "concurrent-rv"
					return status.Error(codes.OutOfRange, "resource changed")
				case "response-lost":
					c.region = proto.Clone(proposed).(*corev1.Region)
					return status.Error(codes.Unavailable, "committed response lost")
				}
				return nil
			}
			require.NoError(t, g.setRegionVersionMap(t.Context(), input))
			assert.Equal(t, 2, c.gets)
			if mode == "conflict" {
				assert.Equal(t, "preserved", c.region.Metadata.Labels["concurrent"])
				assert.Equal(t, "new-peer-version", c.region.Status.VersionInfoMap["other"].Version)
			}
			if mode == "response-lost" || mode == "read-outage" {
				assert.Equal(t, 1, c.updates)
			} else {
				assert.Equal(t, 2, c.updates)
			}
			assert.Equal(t, "old-core", input.Status.Version)
			assert.NotContains(t, input.Status.VersionInfoMap, "octeliumee")
		})
	}
}

func TestPackageVersionPublicationFailurePolicy(t *testing.T) {
	for _, tc := range []struct {
		code     codes.Code
		attempts int
	}{
		{codes.Unavailable, 5}, {codes.OutOfRange, 5}, {codes.Aborted, 5},
		{codes.ResourceExhausted, 5}, {codes.DeadlineExceeded, 5},
		{codes.PermissionDenied, 1}, {codes.InvalidArgument, 1}, {codes.NotFound, 1}, {codes.Canceled, 1},
	} {
		for _, point := range []string{"read", "write"} {
			t.Run(point+"/"+tc.code.String(), func(t *testing.T) {
				g, c := newVersionTestGenesis(t)
				before := proto.Clone(c.region)
				failure := status.Error(tc.code, "injected failure")
				if point == "read" {
					c.get = func(context.Context, *rmetav1.GetOptions) error { return failure }
				} else {
					c.update = func(context.Context, *corev1.Region) error { return failure }
				}
				require.ErrorIs(t, g.setRegionVersionMap(t.Context(), proto.Clone(c.region).(*corev1.Region)), failure)
				assert.Equal(t, tc.attempts, c.gets)
				if point == "write" {
					assert.Equal(t, tc.attempts, c.updates)
				} else {
					assert.Zero(t, c.updates)
				}
				assert.True(t, proto.Equal(before, c.region))
			})
		}
	}
}

func TestPackageVersionPublicationCancellation(t *testing.T) {
	for _, mode := range []string{"before", "during", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			g, c := newVersionTestGenesis(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "before" {
				cancel()
			} else if mode == "deadline" {
				ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
			}
			c.update = func(ctx context.Context, _ *corev1.Region) error {
				if mode == "during" {
					cancel()
				} else {
					<-ctx.Done()
				}
				return ctx.Err()
			}
			err := g.setRegionVersionMap(ctx, proto.Clone(c.region).(*corev1.Region))
			if mode == "deadline" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			} else {
				require.ErrorIs(t, err, context.Canceled)
			}
			assert.LessOrEqual(t, c.updates, 1)
			if mode == "before" {
				assert.Zero(t, c.gets)
			}
			assert.NotContains(t, c.region.Status.VersionInfoMap, "octeliumee")
		})
	}
}

func TestJoinCoreVersionPublicationRetries(t *testing.T) {
	for _, point := range []string{"success", "read-outage", "conflict", "response-lost", "denied", "exhausted", "cancelled"} {
		t.Run(point, func(t *testing.T) {
			g, c := newVersionTestGenesis(t)
			input := proto.Clone(c.region).(*corev1.Region)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			c.get = func(context.Context, *rmetav1.GetOptions) error {
				if point == "read-outage" && c.gets == 1 {
					return status.Error(codes.Unavailable, "temporary read failure")
				}
				return nil
			}
			var first *corev1.Region_Status_VersionInfo
			c.update = func(_ context.Context, proposed *corev1.Region) error {
				if first == nil {
					first = proto.Clone(proposed.Status.VersionInfoMap["octelium"]).(*corev1.Region_Status_VersionInfo)
				} else {
					assert.True(t, proto.Equal(first, proposed.Status.VersionInfoMap["octelium"]))
				}
				if point == "cancelled" {
					cancel()
					return ctx.Err()
				}
				if point == "denied" {
					return status.Error(codes.PermissionDenied, "version denied")
				}
				if point == "exhausted" || (point == "conflict" && c.updates == 1) {
					c.region.Status.VersionInfoMap["other"].Version = "concurrent-peer-version"
					return status.Error(codes.OutOfRange, "concurrent Region change")
				}
				if point == "response-lost" && c.updates == 1 {
					c.region = proto.Clone(proposed).(*corev1.Region)
					return status.Error(codes.Unavailable, "committed response lost")
				}
				return nil
			}
			result, err := g.setRegionCoreVersion(ctx, input, "v2.0.0")
			switch point {
			case "denied", "exhausted", "cancelled":
				require.Error(t, err)
				assert.Equal(t, "old-core", c.region.Status.Version)
				if point == "exhausted" {
					assert.Equal(t, 5, c.updates)
				} else {
					assert.Equal(t, 1, c.updates)
				}
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "v2.0.0", result.Status.Version)
				assert.Equal(t, "v2.0.0", result.Status.VersionInfoMap["octelium"].Version)
				assert.Equal(t, "existing-operation-id", result.Status.VersionInfoMap["octelium"].Id)
				assert.Equal(t, "preserved", result.Metadata.Labels["custom"])
			}
			if point == "conflict" {
				assert.Equal(t, "concurrent-peer-version", result.Status.VersionInfoMap["other"].Version)
			}
			if point == "response-lost" {
				assert.Equal(t, 1, c.updates)
				assert.Equal(t, 2, c.gets)
			}
			assert.Equal(t, "old-core", input.Status.Version)
		})
	}
}
