package dnsctl

import (
	"testing"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
	k8scorev1 "k8s.io/api/core/v1"
)

func TestKubernetesServiceIngressAddresses(t *testing.T) {
	svc := &k8scorev1.Service{
		Spec: k8scorev1.ServiceSpec{ExternalIPs: []string{"192.0.2.1", "2001:db8::1"}},
		Status: k8scorev1.ServiceStatus{
			LoadBalancer: k8scorev1.LoadBalancerStatus{Ingress: []k8scorev1.LoadBalancerIngress{{IP: "192.0.2.2"}, {Hostname: "lb.example.com"}}},
		},
	}
	assert.Equal(t, svc.Spec.ExternalIPs, doGetIPs(svc))
	svc.Spec.ExternalIPs = nil
	assert.Equal(t, []string{"192.0.2.2"}, doGetIPs(svc))
	svc.Status.LoadBalancer.Ingress = nil
	assert.Empty(t, doGetIPs(svc))
}

func TestDNSProviderUnsetConfigurations(t *testing.T) {
	for _, spec := range []*enterprisev1.DNSProvider_Spec{
		{}, {Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{}}, {Type: &enterprisev1.DNSProvider_Spec_Digitalocean{}}, {Type: &enterprisev1.DNSProvider_Spec_Aws{}}, {Type: &enterprisev1.DNSProvider_Spec_Linode_{}}, {Type: &enterprisev1.DNSProvider_Spec_Azure_{}}, {Type: &enterprisev1.DNSProvider_Spec_Google_{}}, {Type: &enterprisev1.DNSProvider_Spec_Ovh{}},
	} {
		provider := &enterprisev1.DNSProvider{Spec: spec}
		got, err := getProvider(t.Context(), nil, provider, "example.com")
		assert.NoError(t, err)
		assert.Nil(t, got)
		cc := &corev1.ClusterConfig{Status: &corev1.ClusterConfig_Status{Domain: "example.com"}}
		assert.NoError(t, SetCNAME(t.Context(), nil, provider, cc, "app.example.com", "region.example.com"))
		assert.NoError(t, SetAddresses(t.Context(), nil, provider, cc, []string{"192.0.2.1"}, "app.example.com"))
	}
}
