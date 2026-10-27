// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package policyportal

import (
	"context"
	"fmt"
	"net"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/sessionc"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type policyPortalTestEnv struct {
	ctx       context.Context
	srv       *Server
	user      *corev1.User
	device    *corev1.Device
	session   *corev1.Session
	namespace *corev1.Namespace
	service   *corev1.Service
}

func newPolicyPortalTestEnv(t *testing.T) *policyPortalTestEnv {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		return nil
	}

	t.Cleanup(func() {
		cancel()
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	srv, err := NewServer(ctx, octeliumC)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	namespace, err := octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	region, err := octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	userName := fmt.Sprintf("policy-portal-%s", utilrand.GetRandomStringCanonical(8))
	user, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name: userName,
			Uid:  vutils.UUIDv4(),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	clusterConfig, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	session, err := sessionc.NewSession(ctx, &sessionc.CreateSessionOpts{
		Usr:           user,
		ClusterConfig: clusterConfig,
		SessType:      corev1.Session_Status_CLIENT,
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	session, err = octeliumC.CoreC().CreateSession(ctx, session)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	device, err := octeliumC.CoreC().CreateDevice(ctx, &corev1.Device{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindDevice,
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("device-%s", utilrand.GetRandomStringCanonical(8)),
			Uid:  vutils.UUIDv4(),
		},
		Spec: &corev1.Device_Spec{
			State: corev1.Device_Spec_ACTIVE,
		},
		Status: &corev1.Device_Status{
			UserRef: umetav1.GetObjectReference(user),
			OsType:  corev1.Device_Status_LINUX,
		},
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	serviceName := fmt.Sprintf(
		"policy-portal-%s.%s",
		utilrand.GetRandomStringCanonical(8),
		namespace.Metadata.Name,
	)
	service, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindService,
		Metadata: &metav1.Metadata{
			Name: serviceName,
			Uid:  vutils.UUIDv4(),
		},
		Spec: &corev1.Service_Spec{
			Mode: corev1.Service_Spec_HTTP,
			Port: 8080,
			Config: &corev1.Service_Spec_Config{
				Upstream: &corev1.Service_Spec_Config_Upstream{
					Type: &corev1.Service_Spec_Config_Upstream_Url{
						Url: "https://www.example.com",
					},
				},
			},
		},
		Status: &corev1.Service_Status{
			Port:            8080,
			NamespaceRef:    umetav1.GetObjectReference(namespace),
			RegionRef:       umetav1.GetObjectReference(region),
			PrimaryHostname: "policy-portal-test",
		},
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return nil
	}

	return &policyPortalTestEnv{
		ctx:       ctx,
		srv:       srv,
		user:      user,
		device:    device,
		session:   session,
		namespace: namespace,
		service:   service,
	}
}

func policyPortalInlinePolicy(
	name string,
	effect corev1.Policy_Spec_Rule_Effect,
) *corev1.InlinePolicy {
	return &corev1.InlinePolicy{
		Name: name,
		Spec: &corev1.Policy_Spec{
			Rules: []*corev1.Policy_Spec_Rule{
				{
					Name:   "match-any",
					Effect: effect,
					Condition: &corev1.Condition{
						Type: &corev1.Condition_MatchAny{
							MatchAny: true,
						},
					},
				},
			},
		},
	}
}

func policyPortalSessionNamespaceRequest(
	env *policyPortalTestEnv,
	inlinePolicies ...*corev1.InlinePolicy,
) *enterprisev1.IsAuthorizedRequest {
	return &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_SessionRef{
			SessionRef: umetav1.GetObjectReference(env.session),
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_NamespaceRef{
			NamespaceRef: umetav1.GetObjectReference(env.namespace),
		},
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			InlinePolicies: inlinePolicies,
		},
	}
}

func TestValidateAdditional(t *testing.T) {
	assert.Nil(t, validateAdditional(nil))
	assert.Nil(t, validateAdditional(&enterprisev1.IsAuthorizedRequest{}))
	assert.Nil(t, validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{},
	}))

	policies := make([]string, maxAdditionalPolicies)
	for idx := range policies {
		policies[idx] = fmt.Sprintf("policy-%d", idx)
	}

	assert.Nil(t, validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			Policies: policies,
		},
	}))

	err := validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			Policies: append(policies, "one-too-many"),
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	err = validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			Policies: []string{"INVALID POLICY NAME"},
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	inlinePolicies := make([]*corev1.InlinePolicy, maxAdditionalInlinePolicies)
	for idx := range inlinePolicies {
		inlinePolicies[idx] = policyPortalInlinePolicy(
			fmt.Sprintf("inline-%d", idx),
			corev1.Policy_Spec_Rule_ALLOW,
		)
	}

	assert.Nil(t, validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			InlinePolicies: inlinePolicies,
		},
	}))

	err = validateAdditional(&enterprisev1.IsAuthorizedRequest{
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			InlinePolicies: append(
				inlinePolicies,
				policyPortalInlinePolicy("one-too-many", corev1.Policy_Spec_Rule_ALLOW),
			),
		},
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestIsAuthorizedValidatesRequiredSelectors(t *testing.T) {
	srv := &Server{}

	resp, err := srv.IsAuthorized(context.Background(), &enterprisev1.IsAuthorizedRequest{})
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	resp, err = srv.IsAuthorized(context.Background(), &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
			UserRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(),
			},
		},
	})
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestIsAuthorizedRejectsInvalidAdditionalBeforeResourceLookup(t *testing.T) {
	srv := &Server{}

	resp, err := srv.IsAuthorized(context.Background(), &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
			UserRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(),
			},
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_NamespaceRef{
			NamespaceRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(),
			},
		},
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			Policies: make([]string, maxAdditionalPolicies+1),
		},
	})
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestIsAuthorizedSessionNamespaceNoPolicyMatch(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(
		env.ctx,
		policyPortalSessionNamespaceRequest(env),
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	assert.False(t, resp.IsAuthorized)
	assert.NotNil(t, resp.Reason)
	if resp.Reason != nil {
		assert.Equal(
			t,
			corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH,
			resp.Reason.Type,
		)
	}
}

func TestIsAuthorizedSessionNamespaceAdditionalAllow(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(
		env.ctx,
		policyPortalSessionNamespaceRequest(
			env,
			policyPortalInlinePolicy("allow", corev1.Policy_Spec_Rule_ALLOW),
		),
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	assert.True(t, resp.IsAuthorized)
	assert.NotNil(t, resp.Reason)
	if resp.Reason != nil {
		assert.Equal(
			t,
			corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH,
			resp.Reason.Type,
		)
	}
}

func TestIsAuthorizedSessionNamespaceAdditionalDeny(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(
		env.ctx,
		policyPortalSessionNamespaceRequest(
			env,
			policyPortalInlinePolicy("deny", corev1.Policy_Spec_Rule_DENY),
		),
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	assert.False(t, resp.IsAuthorized)
	assert.NotNil(t, resp.Reason)
	if resp.Reason != nil {
		assert.Equal(
			t,
			corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH,
			resp.Reason.Type,
		)
	}
}

func TestIsAuthorizedDeviceNamespaceAdditionalAllow(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(env.ctx, &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_DeviceRef{
			DeviceRef: umetav1.GetObjectReference(env.device),
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_NamespaceRef{
			NamespaceRef: umetav1.GetObjectReference(env.namespace),
		},
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			InlinePolicies: []*corev1.InlinePolicy{
				policyPortalInlinePolicy("allow-device", corev1.Policy_Spec_Rule_ALLOW),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	assert.True(t, resp.IsAuthorized)
	assert.NotNil(t, resp.Reason)
}

func TestIsAuthorizedSessionServiceAdditionalAllow(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(env.ctx, &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_SessionRef{
			SessionRef: umetav1.GetObjectReference(env.session),
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
			ServiceRef: umetav1.GetObjectReference(env.service),
		},
		Additional: &enterprisev1.IsAuthorizedRequest_Additional{
			InlinePolicies: []*corev1.InlinePolicy{
				policyPortalInlinePolicy("allow-service", corev1.Policy_Spec_Rule_ALLOW),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	assert.True(t, resp.IsAuthorized)
	assert.NotNil(t, resp.Reason)
}

func TestIsAuthorizedReturnsNotFoundForMissingSession(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	if env == nil {
		return
	}

	resp, err := env.srv.IsAuthorized(env.ctx, &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_SessionRef{
			SessionRef: &metav1.ObjectReference{
				Uid: vutils.UUIDv4(),
			},
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_NamespaceRef{
			NamespaceRef: umetav1.GetObjectReference(env.namespace),
		},
	})
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
}

func newPolicyPortalRPCClient(t *testing.T, env *policyPortalTestEnv) enterprisev1.PolicyPortalServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	enterprisev1.RegisterPolicyPortalServiceServer(srv, env.srv)
	go srv.Serve(listener)
	conn, err := grpc.NewClient("passthrough:///policyportal", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
		srv.Stop()
		listener.Close()
	})
	return enterprisev1.NewPolicyPortalServiceClient(conn)
}

func TestCloneAdditionalAuthorizationIsolation(t *testing.T) {
	assert.Nil(t, cloneAdditionalAuthorization(nil))
	original := &enterprisev1.IsAuthorizedRequest_Additional{
		Policies:       []string{"named-policy"},
		InlinePolicies: []*corev1.InlinePolicy{nil, policyPortalInlinePolicy("inline", corev1.Policy_Spec_Rule_ALLOW)},
	}
	copy := cloneAdditionalAuthorization(original)
	require.Len(t, copy.InlinePolicies, 2)
	assert.Nil(t, copy.InlinePolicies[0])
	copy.Policies[0] = "changed"
	copy.InlinePolicies[1].Name = "changed"
	copy.InlinePolicies[1].Spec.Rules[0].Effect = corev1.Policy_Spec_Rule_DENY
	assert.Equal(t, "named-policy", original.Policies[0])
	assert.Equal(t, "inline", original.InlinePolicies[1].Name)
	assert.Equal(t, corev1.Policy_Spec_Rule_ALLOW, original.InlinePolicies[1].Spec.Rules[0].Effect)
}

func TestPolicyPortalRPCDownstreamAndUpstreamMatrix(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	require.NotNil(t, env)
	group, err := env.srv.octeliumC.CoreC().CreateGroup(env.ctx, &corev1.Group{
		Metadata: &metav1.Metadata{Name: "policy-group-" + utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.Group_Spec{},
	})
	require.NoError(t, err)
	env.user.Spec.Groups = append(env.user.Spec.Groups, group.Metadata.Name)
	_, err = env.srv.octeliumC.CoreC().UpdateUser(env.ctx, env.user)
	require.NoError(t, err)
	env.session.Status.DeviceRef = umetav1.GetObjectReference(env.device)
	_, err = env.srv.octeliumC.CoreC().UpdateSession(env.ctx, env.session)
	require.NoError(t, err)
	env.service.Spec.IsPublic = true
	_, err = env.srv.octeliumC.CoreC().UpdateService(env.ctx, env.service)
	require.NoError(t, err)
	client := newPolicyPortalRPCClient(t, env)
	for _, downstream := range []string{"session", "device", "user"} {
		for _, upstream := range []string{"namespace", "service"} {
			if downstream == "user" && upstream == "namespace" {
				continue
			}
			for _, byName := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/Name=%t", downstream, upstream, byName), func(t *testing.T) {
					ref := func(md *metav1.Metadata) *metav1.ObjectReference {
						if byName {
							return &metav1.ObjectReference{Name: md.Name}
						}
						return &metav1.ObjectReference{Uid: md.Uid}
					}
					req := policyPortalSessionNamespaceRequest(env, policyPortalInlinePolicy("allow", corev1.Policy_Spec_Rule_ALLOW))
					switch downstream {
					case "session":
						req.Downstream = &enterprisev1.IsAuthorizedRequest_SessionRef{SessionRef: ref(env.session.Metadata)}
					case "device":
						req.Downstream = &enterprisev1.IsAuthorizedRequest_DeviceRef{DeviceRef: ref(env.device.Metadata)}
					case "user":
						req.Downstream = &enterprisev1.IsAuthorizedRequest_UserRef{UserRef: ref(env.user.Metadata)}
					}
					if upstream == "service" {
						req.Upstream = &enterprisev1.IsAuthorizedRequest_ServiceRef{ServiceRef: ref(env.service.Metadata)}
					} else {
						req.Upstream = &enterprisev1.IsAuthorizedRequest_NamespaceRef{NamespaceRef: ref(env.namespace.Metadata)}
					}
					res, err := client.IsAuthorized(env.ctx, req)
					require.NoError(t, err)
					assert.True(t, res.IsAuthorized)
					require.NotNil(t, res.Reason)
					assert.Equal(t, corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, res.Reason.Type)
				})
			}
		}
	}
}

func TestPolicyPortalRPCInvalidReferences(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	require.NotNil(t, env)
	client := newPolicyPortalRPCClient(t, env)
	for _, invalid := range []*metav1.ObjectReference{nil, {}, {Uid: "invalid"}, {Name: "invalid/name"}} {
		for _, selector := range []string{"session", "device", "user", "service", "namespace"} {
			req := policyPortalSessionNamespaceRequest(env)
			switch selector {
			case "session":
				req.Downstream = &enterprisev1.IsAuthorizedRequest_SessionRef{SessionRef: invalid}
			case "device":
				req.Downstream = &enterprisev1.IsAuthorizedRequest_DeviceRef{DeviceRef: invalid}
			case "user":
				req.Downstream = &enterprisev1.IsAuthorizedRequest_UserRef{UserRef: invalid}
			case "service":
				req.Upstream = &enterprisev1.IsAuthorizedRequest_ServiceRef{ServiceRef: invalid}
			case "namespace":
				req.Upstream = &enterprisev1.IsAuthorizedRequest_NamespaceRef{NamespaceRef: invalid}
			}
			_, err := client.IsAuthorized(env.ctx, req)
			assert.Equal(t, codes.InvalidArgument, status.Code(err), selector)
		}
	}
	_, err := env.srv.IsAuthorized(env.ctx, nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPolicyPortalRPCMissingResources(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	require.NotNil(t, env)
	client := newPolicyPortalRPCClient(t, env)
	for _, selector := range []string{"device", "user", "service", "namespace"} {
		req := policyPortalSessionNamespaceRequest(env)
		ref := &metav1.ObjectReference{Uid: vutils.UUIDv4()}
		switch selector {
		case "device":
			req.Downstream = &enterprisev1.IsAuthorizedRequest_DeviceRef{DeviceRef: ref}
		case "user":
			req.Downstream = &enterprisev1.IsAuthorizedRequest_UserRef{UserRef: ref}
		case "service":
			req.Upstream = &enterprisev1.IsAuthorizedRequest_ServiceRef{ServiceRef: ref}
		case "namespace":
			req.Upstream = &enterprisev1.IsAuthorizedRequest_NamespaceRef{NamespaceRef: ref}
		}
		res, err := client.IsAuthorized(env.ctx, req)
		assert.Nil(t, res)
		assert.Equal(t, codes.NotFound, status.Code(err), selector)
	}
}

func TestPolicyPortalAdditionalInputRemainsUnchanged(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	require.NotNil(t, env)
	req := policyPortalSessionNamespaceRequest(env, policyPortalInlinePolicy("allow", corev1.Policy_Spec_Rule_ALLOW))
	before := proto.Clone(req)
	for range 3 {
		res, err := env.srv.IsAuthorized(env.ctx, req)
		require.NoError(t, err)
		assert.True(t, res.IsAuthorized)
		assert.True(t, proto.Equal(before, req))
	}
}

func TestPolicyPortalCanceledRequest(t *testing.T) {
	env := newPolicyPortalTestEnv(t)
	require.NotNil(t, env)
	ctx, cancel := context.WithCancel(env.ctx)
	cancel()
	res, err := newPolicyPortalRPCClient(t, env).IsAuthorized(ctx, policyPortalSessionNamespaceRequest(env))
	assert.Nil(t, res)
	assert.Equal(t, codes.Canceled, status.Code(err))
}
