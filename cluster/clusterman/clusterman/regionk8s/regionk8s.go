// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package regionk8s

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const requestTimeout = 30 * time.Second

type NewClientFn func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error)

func NewClientFactory(octeliumC octeliumc.ClientInterface) NewClientFn {
	return func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return NewClient(ctx, octeliumC, k)
	}
}

func NewClient(ctx context.Context,
	octeliumC octeliumc.ClientInterface, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
	cfg, err := GetRESTConfig(ctx, octeliumC, k)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func GetRESTConfig(ctx context.Context,
	octeliumC octeliumc.ClientInterface, k *enterprisev1.JoinClusterRequest_Kubernetes) (*rest.Config, error) {
	if k == nil {
		return nil, errors.Errorf("The Kubernetes connection is not set")
	}

	switch k.Type.(type) {
	case *enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_:
		spec := k.GetKubeconfig()
		val, err := getSecretValue(ctx, octeliumC, spec.GetFromSecret())
		if err != nil {
			return nil, err
		}

		return getKubeconfigRESTConfig(val, spec.Context)

	case *enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_:
		spec := k.GetBearerToken()
		val, err := getSecretValue(ctx, octeliumC, spec.GetFromSecret())
		if err != nil {
			return nil, err
		}

		return getBearerTokenRESTConfig(spec, strings.TrimSpace(string(val)))

	default:
		return nil, errors.Errorf("Unsupported Kubernetes connection type")
	}
}

func getSecretValue(ctx context.Context, octeliumC octeliumc.ClientInterface, name string) ([]byte, error) {
	if name == "" {
		return nil, errors.Errorf("Empty Secret name")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	ret := uenterprisev1.ToSecret(sec).GetValueBytes()
	if len(ret) == 0 {
		return nil, errors.Errorf("The Secret %s is empty", name)
	}

	return ret, nil
}

func getKubeconfigRESTConfig(data []byte, kubeContext string) (*rest.Config, error) {
	cfg, err := clientcmd.Load(data)
	if err != nil {
		return nil, errors.Errorf("Could not parse the kubeconfig: %+v", err)
	}

	if err := validateKubeconfig(cfg); err != nil {
		return nil, err
	}

	ret, err := clientcmd.NewNonInteractiveClientConfig(*cfg, kubeContext,
		&clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, errors.Errorf("Could not load the kubeconfig: %+v", err)
	}

	ret.Timeout = requestTimeout

	return ret, nil
}

func validateKubeconfig(cfg *clientcmdapi.Config) error {
	for name, cluster := range cfg.Clusters {
		if cluster == nil {
			continue
		}
		if cluster.CertificateAuthority != "" {
			return errors.Errorf("The kubeconfig cluster %s must embed its certificate authority data instead of a file path", name)
		}
	}

	for name, authInfo := range cfg.AuthInfos {
		if authInfo == nil {
			continue
		}
		if authInfo.Exec != nil {
			return errors.Errorf("The kubeconfig user %s must not use an exec credential plugin", name)
		}
		if authInfo.AuthProvider != nil {
			return errors.Errorf("The kubeconfig user %s must not use an auth provider plugin", name)
		}
		if authInfo.ClientCertificate != "" || authInfo.ClientKey != "" || authInfo.TokenFile != "" {
			return errors.Errorf("The kubeconfig user %s must embed its credentials instead of file paths", name)
		}
	}

	return nil
}

func getBearerTokenRESTConfig(spec *enterprisev1.JoinClusterRequest_Kubernetes_BearerToken, token string) (*rest.Config, error) {
	if token == "" {
		return nil, errors.Errorf("Empty bearer token")
	}

	u, err := url.Parse(spec.Server)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.Errorf("Invalid Kubernetes API server URL: %s", spec.Server)
	}

	ret := &rest.Config{
		Host:        spec.Server,
		BearerToken: token,
		Timeout:     requestTimeout,
	}

	if len(spec.TrustedCAs) > 0 {
		ret.TLSClientConfig.CAData = []byte(strings.Join(spec.TrustedCAs, "\n"))
	}

	return ret, nil
}
