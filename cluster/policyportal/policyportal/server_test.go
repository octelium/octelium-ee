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
