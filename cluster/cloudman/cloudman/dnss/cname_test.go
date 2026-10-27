package dnss

import (
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceCNAMERegionAndManagedSubdomains(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateDNSProvider(t.Context(), &enterprisev1.DNSProvider{Metadata: &metav1.Metadata{Name: "default"}, Spec: &enterprisev1.DNSProvider_Spec{}})
	require.NoError(t, err)
	client := tst.C.OcteliumC
	cc, err := client.CoreV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	provider, err := client.EnterpriseC().GetDNSProvider(t.Context(), &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	region, err := client.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	svc := &corev1.Service{
		Metadata: &metav1.Metadata{Name: "portal.default"},
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{},
	}
	assert.ErrorContains(t, SetServiceCNAME(t.Context(), client, svc, cc, provider), "nil regionRef")
	svc.Status.RegionRef = &metav1.ObjectReference{Uid: vutils.UUIDv4()}
	assert.Error(t, SetServiceCNAME(t.Context(), client, svc, cc, provider))
	svc.Status.RegionRef = umetav1.GetObjectReference(region)
	assert.NoError(t, SetServiceCNAME(t.Context(), client, svc, cc, provider))
	svc.Status.PrimaryHostname = "portal.default"
	svc.Status.ManagedService = &corev1.Service_Status_ManagedService{HasSubdomain: true}
	assert.NoError(t, SetServiceCNAME(t.Context(), client, svc, cc, provider))
}
