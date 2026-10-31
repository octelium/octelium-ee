// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package regionk8s

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestCA(t *testing.T) string {
	crt, err := utils_cert.GenerateSelfSignedCert("k8s.example.com", nil, time.Hour)
	require.NoError(t, err)
	ret, err := crt.GetCertPEM()
	require.NoError(t, err)
	return ret
}

func getTestKubeconfig(ca string, user string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: west
clusters:
- name: west
  cluster:
    server: https://203.0.113.10:6443
    certificate-authority-data: %s
- name: east
  cluster:
    server: https://203.0.113.20:6443
    certificate-authority-data: %s
contexts:
- name: west
  context:
    cluster: west
    user: admin
- name: east
  context:
    cluster: east
    user: admin
users:
- name: admin
  user:
%s
`, base64.StdEncoding.EncodeToString([]byte(ca)), base64.StdEncoding.EncodeToString([]byte(ca)), user)
}

func getTestKubeconfigK8s(secret, kubeContext string) *enterprisev1.JoinClusterRequest_Kubernetes {
	return &enterprisev1.JoinClusterRequest_Kubernetes{
		Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
			Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
					FromSecret: secret,
				},
				Context: kubeContext,
			},
		},
	}
}

func getTestBearerK8s(secret, server string, cas []string) *enterprisev1.JoinClusterRequest_Kubernetes {
	return &enterprisev1.JoinClusterRequest_Kubernetes{
		Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_{
			BearerToken: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_FromSecret{
					FromSecret: secret,
				},
				Server:     server,
				TrustedCAs: cas,
			},
		},
	}
}

func TestGetRESTConfig(t *testing.T) {
	ctx := t.Context()

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })

	ca := getTestCA(t)

	createSecret := func(name, value string) {
		_, err := tst.C.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: name},
			Spec:     &enterprisev1.Secret_Spec{},
			Status:   &enterprisev1.Secret_Status{},
			Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: value}},
		})
		require.NoError(t, err)
	}

	createSecret("valid", getTestKubeconfig(ca, "    token: west-token"))
	createSecret("exec", getTestKubeconfig(ca, "    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: /bin/sh"))
	createSecret("auth-provider", getTestKubeconfig(ca, "    auth-provider:\n      name: gcp"))
	createSecret("token-file", getTestKubeconfig(ca, "    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token"))
	createSecret("client-cert-file", getTestKubeconfig(ca, "    client-certificate: /tmp/crt.pem\n    client-key: /tmp/key.pem"))
	createSecret("ca-file", `apiVersion: v1
kind: Config
current-context: west
clusters:
- name: west
  cluster:
    server: https://203.0.113.10:6443
    certificate-authority: /tmp/ca.pem
contexts:
- name: west
  context:
    cluster: west
    user: admin
users:
- name: admin
  user:
    token: west-token
`)
	createSecret("invalid", "{{{")
	createSecret("token", " bearer-token \n")
	createSecret("empty", " ")

	{
		cfg, err := GetRESTConfig(ctx, tst.C.OcteliumC, getTestKubeconfigK8s("valid", ""))
		require.NoError(t, err)
		assert.Equal(t, "https://203.0.113.10:6443", cfg.Host)
		assert.Equal(t, "west-token", cfg.BearerToken)
		assert.Equal(t, requestTimeout, cfg.Timeout)
		assert.Equal(t, ca, string(cfg.TLSClientConfig.CAData))

		cfg, err = GetRESTConfig(ctx, tst.C.OcteliumC, getTestKubeconfigK8s("valid", "east"))
		require.NoError(t, err)
		assert.Equal(t, "https://203.0.113.20:6443", cfg.Host)

		_, err = NewClient(ctx, tst.C.OcteliumC, getTestKubeconfigK8s("valid", ""))
		require.NoError(t, err)

		_, err = NewClientFactory(tst.C.OcteliumC)(ctx, getTestKubeconfigK8s("valid", "east"))
		require.NoError(t, err)
	}

	for _, name := range []string{"exec", "auth-provider", "token-file", "client-cert-file", "ca-file", "invalid", "empty", "missing"} {
		_, err := GetRESTConfig(ctx, tst.C.OcteliumC, getTestKubeconfigK8s(name, ""))
		assert.Error(t, err, name)
	}

	{
		_, err := GetRESTConfig(ctx, tst.C.OcteliumC, getTestKubeconfigK8s("valid", "missing"))
		assert.Error(t, err)
	}

	{
		cfg, err := GetRESTConfig(ctx, tst.C.OcteliumC, getTestBearerK8s("token", "https://203.0.113.10:6443", []string{ca, ca}))
		require.NoError(t, err)
		assert.Equal(t, "https://203.0.113.10:6443", cfg.Host)
		assert.Equal(t, "bearer-token", cfg.BearerToken)
		assert.Equal(t, ca+"\n"+ca, string(cfg.TLSClientConfig.CAData))
		assert.Equal(t, requestTimeout, cfg.Timeout)

		cfg, err = GetRESTConfig(ctx, tst.C.OcteliumC, getTestBearerK8s("token", "https://203.0.113.10:6443", nil))
		require.NoError(t, err)
		assert.Empty(t, cfg.TLSClientConfig.CAData)
	}

	for _, k := range []*enterprisev1.JoinClusterRequest_Kubernetes{
		nil,
		{},
		getTestBearerK8s("token", "http://203.0.113.10:6443", nil),
		getTestBearerK8s("token", "https://", nil),
		getTestBearerK8s("empty", "https://203.0.113.10:6443", nil),
		getTestBearerK8s("missing", "https://203.0.113.10:6443", nil),
		getTestBearerK8s("", "https://203.0.113.10:6443", nil),
	} {
		_, err := GetRESTConfig(ctx, tst.C.OcteliumC, k)
		assert.Error(t, err)
	}
}
