package nocturne

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type nocturneTestClient struct {
	octeliumc.ClientInterface
	enterpriseC renterprisev1.ResourceServiceClient
}

func (c *nocturneTestClient) EnterpriseC() renterprisev1.ResourceServiceClient {
	return c.enterpriseC
}

type oidcTestResources struct {
	renterprisev1.ResourceServiceClient
	getErr, createErr error
}

func (c *oidcTestResources) GetSecret(ctx context.Context, opts *rmetav1.GetOptions, args ...grpc.CallOption) (*enterprisev1.Secret, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	return c.ResourceServiceClient.GetSecret(ctx, opts, args...)
}
func (c *oidcTestResources) CreateSecret(ctx context.Context, sec *enterprisev1.Secret, args ...grpc.CallOption) (*enterprisev1.Secret, error) {
	if c.createErr != nil {
		return nil, c.createErr
	}
	return c.ResourceServiceClient.CreateSecret(ctx, sec, args...)
}

func TestDeviceConditionEvaluation(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	engine, err := celengine.New(t.Context(), nil)
	require.NoError(t, err)
	evaluator := &dmConditionEvaluator{octeliumC: tst.C.OcteliumC, celEngine: engine}
	usr, err := tst.C.OcteliumC.CoreC().CreateUser(t.Context(), &corev1.User{
		Metadata: &metav1.Metadata{Name: "condition-user"},
		Spec:     &corev1.User_Spec{Type: corev1.User_Spec_HUMAN, Email: "user@example.com"},
		Status:   &corev1.User_Status{},
	})
	require.NoError(t, err)
	dev := &corev1.Device{Metadata: &metav1.Metadata{Name: "condition-device"}, Status: &corev1.Device_Status{UserRef: umetav1.GetObjectReference(usr)}}
	for _, tc := range []struct {
		expression string
		want       bool
	}{
		{"true", true},
		{"false", false},
		{"ctx.user.spec.email == 'user@example.com' && ctx.device.metadata.name == 'condition-device'", true},
		{"ctx.user.spec.email == 'other@example.com'", false},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			match, err := evaluator.MatchesDevice(t.Context(), &corev1.Condition{Type: &corev1.Condition_Match{Match: tc.expression}}, dev)
			require.NoError(t, err)
			assert.Equal(t, tc.want, match)
		})
	}
	match, err := evaluator.MatchesDevice(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.True(t, match)
	_, err = evaluator.MatchesDevice(t.Context(), &corev1.Condition{Type: &corev1.Condition_Match{Match: "ctx.user."}}, dev)
	assert.Error(t, err)
}

func TestDeviceConditionMissingAndCanceledUserLookup(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	engine, err := celengine.New(t.Context(), nil)
	require.NoError(t, err)
	evaluator := &dmConditionEvaluator{octeliumC: tst.C.OcteliumC, celEngine: engine}
	dev := &corev1.Device{Metadata: &metav1.Metadata{Name: "device"}, Status: &corev1.Device_Status{}}
	condition := &corev1.Condition{Type: &corev1.Condition_Match{Match: "!has(ctx.user)"}}
	for _, ref := range []*metav1.ObjectReference{nil, {Name: "absent"}, {Uid: vutils.UUIDv4()}} {
		dev.Status.UserRef = ref
		match, err := evaluator.MatchesDevice(t.Context(), condition, dev)
		require.NoError(t, err)
		assert.True(t, match)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = evaluator.MatchesDevice(ctx, condition, dev)
	assert.ErrorContains(t, err, "Could not resolve Device User")
}

func TestOIDCSecretsBootstrapAndIdempotence(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	var configCalls, jwksCalls atomic.Int32
	config := map[string]any{"issuer": "https://issuer.example.com", "jwks_uri": "https://issuer.example.com/keys"}
	jwks := map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			configCalls.Add(1)
			json.NewEncoder(w).Encode(config)
		case "/openid/v1/jwks":
			jwksCalls.Add(1)
			json.NewEncoder(w).Encode(jwks)
		default:
			t.Errorf("Unexpected Kubernetes discovery path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	k8sC, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	require.NoError(t, err)
	require.NoError(t, doSetOIDCSecret(t.Context(), tst.C.OcteliumC, k8sC))
	for _, tc := range []struct {
		name  string
		value map[string]any
	}{
		{ovutils.GetOIDCConfigSecretName(vutils.GetMyRegionName()), config},
		{ovutils.GetOIDC_JWKSSecretName(vutils.GetMyRegionName()), jwks},
	} {
		sec, err := tst.C.OcteliumC.EnterpriseC().GetSecret(t.Context(), &rmetav1.GetOptions{Name: tc.name})
		require.NoError(t, err)
		assert.True(t, sec.Metadata.IsSystem)
		assert.True(t, sec.Metadata.IsSystemHidden)
		var value map[string]any
		require.NoError(t, json.Unmarshal([]byte(sec.Data.GetValue()), &value))
		assert.Equal(t, tc.value, value)
	}
	require.NoError(t, doSetOIDCSecret(t.Context(), tst.C.OcteliumC, k8sC))
	assert.EqualValues(t, 1, configCalls.Load())
	assert.EqualValues(t, 1, jwksCalls.Load())
}

func TestOIDCSecretFetchFailureCreatesNothing(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	k8sC, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	require.NoError(t, err)
	assert.Error(t, doSetOIDCSecret(t.Context(), tst.C.OcteliumC, k8sC))
	for _, name := range []string{ovutils.GetOIDCConfigSecretName(vutils.GetMyRegionName()), ovutils.GetOIDC_JWKSSecretName(vutils.GetMyRegionName())} {
		_, err := tst.C.OcteliumC.EnterpriseC().GetSecret(t.Context(), &rmetav1.GetOptions{Name: name})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestOIDCSecretResourceErrors(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	failure := errors.New("resource service unavailable")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issuer": "https://issuer.example.com"})
	}))
	t.Cleanup(srv.Close)
	k8sC, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	require.NoError(t, err)
	for _, resources := range []*oidcTestResources{
		{ResourceServiceClient: tst.C.OcteliumC.EnterpriseC(), getErr: failure},
		{ResourceServiceClient: tst.C.OcteliumC.EnterpriseC(), createErr: failure},
	} {
		client := &nocturneTestClient{ClientInterface: tst.C.OcteliumC, enterpriseC: resources}
		assert.ErrorIs(t, doSetOIDCSecret(t.Context(), client, k8sC), failure)
	}
	assert.ErrorContains(t, doSetOIDCSecret(t.Context(), nil, fake.NewSimpleClientset()), "client set")
	require.NoError(t, doInit(t.Context(), nil, fake.NewSimpleClientset()))
}
