package nscontroller

import (
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceCertificateLifecycle(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	ctl := NewController(tst.C.OcteliumC)
	ns := &corev1.Namespace{Metadata: &metav1.Metadata{Name: "controller-namespace"}}
	require.NoError(t, ctl.OnAdd(t.Context(), ns))
	crt, err := tst.C.OcteliumC.EnterpriseC().GetCertificate(t.Context(), &rmetav1.GetOptions{Name: "ns-controller-namespace"})
	require.NoError(t, err)
	assert.Equal(t, enterprisev1.Certificate_Spec_MANUAL, crt.Spec.Mode)
	assert.Equal(t, ns.Metadata.Name, crt.Status.NamespaceRef.Name)
	require.NoError(t, ctl.OnAdd(t.Context(), ns))
	got, err := tst.C.OcteliumC.EnterpriseC().GetCertificate(t.Context(), &rmetav1.GetOptions{Uid: crt.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, crt.Metadata.ResourceVersion, got.Metadata.ResourceVersion)
	assert.NoError(t, ctl.OnUpdate(t.Context(), ns, ns))
	require.NoError(t, ctl.OnDelete(t.Context(), ns))
	assert.NoError(t, ctl.OnDelete(t.Context(), ns))
	_, err = tst.C.OcteliumC.EnterpriseC().GetCertificate(t.Context(), &rmetav1.GetOptions{Uid: crt.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestNamespaceManagedCertificateUsesDefaultIssuer(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	cc, err := tst.C.OcteliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	cc.Spec.Certificate = &enterprisev1.ClusterConfig_Spec_Certificate{DefaultMode: enterprisev1.Certificate_Spec_MANAGED}
	_, err = tst.C.OcteliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	iss, err := tst.C.OcteliumC.EnterpriseC().GetCertificateIssuer(t.Context(), &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	assert.NoError(t, doSetCert(t.Context(), tst.C.OcteliumC, &corev1.Namespace{Metadata: &metav1.Metadata{Name: "managed"}}))
	crt, err := tst.C.OcteliumC.EnterpriseC().GetCertificate(t.Context(), &rmetav1.GetOptions{Name: "ns-managed"})
	require.NoError(t, err)
	assert.Equal(t, iss.Metadata.Uid, crt.Status.CertificateIssuerRef.Uid)
	assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED, crt.Status.Issuance.State)
}
