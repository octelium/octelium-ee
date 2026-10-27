package certificates

import (
	"context"
	"errors"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type recordingCertificateSetter struct {
	calls int
	err   error
}

func (s *recordingCertificateSetter) SetCertificate(_ context.Context, _ *enterprisev1.Certificate) error {
	s.calls++
	return s.err
}

func TestCertificateControllerManagedIssuanceAndDeletion(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	setter := &recordingCertificateSetter{}
	ctl := NewController(tst.C.OcteliumC, setter)
	crt := &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{Name: "test-certificate"},
		Spec:     &enterprisev1.Certificate_Spec{Mode: enterprisev1.Certificate_Spec_MANUAL},
		Status:   &enterprisev1.Certificate_Status{},
	}
	assert.NoError(t, ctl.OnAdd(t.Context(), crt))
	assert.Zero(t, setter.calls)
	crt.Spec.Mode = enterprisev1.Certificate_Spec_MANAGED
	assert.NoError(t, ctl.OnAdd(t.Context(), crt))
	assert.Zero(t, setter.calls)
	crt.Status.CertificateIssuerRef = &metav1.ObjectReference{Uid: vutils.UUIDv4()}
	assert.NoError(t, ctl.OnAdd(t.Context(), crt))
	assert.Zero(t, setter.calls)
	iss, err := tst.C.OcteliumC.EnterpriseC().CreateCertificateIssuer(t.Context(), &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{Name: "controller-issuer"},
		Spec:     &enterprisev1.CertificateIssuer_Spec{Type: &enterprisev1.CertificateIssuer_Spec_Acme{Acme: &enterprisev1.CertificateIssuer_Spec_ACME{}}},
	})
	require.NoError(t, err)
	crt.Status.CertificateIssuerRef = umetav1.GetObjectReference(iss)
	require.NoError(t, ctl.OnAdd(t.Context(), crt))
	assert.Equal(t, 1, setter.calls)
	old := proto.Clone(crt).(*enterprisev1.Certificate)
	assert.NoError(t, ctl.OnUpdate(t.Context(), crt, old))
	assert.Equal(t, 1, setter.calls)
	crt.Status.Issuance = &enterprisev1.Certificate_Status_Issuance{State: enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED}
	assert.NoError(t, ctl.OnUpdate(t.Context(), crt, old))
	assert.Equal(t, 2, setter.calls)
	setter.err = errors.New("enqueue failed")
	assert.ErrorIs(t, ctl.OnAdd(t.Context(), crt), setter.err)
	name := uenterprisev1.ToCertificate(crt).GetSecretName()
	_, err = tst.C.OcteliumC.CoreC().CreateSecret(t.Context(), &corev1.Secret{Metadata: &metav1.Metadata{Name: name}, Spec: &corev1.Secret_Spec{}})
	require.NoError(t, err)
	assert.NoError(t, ctl.OnDelete(t.Context(), crt))
	assert.NoError(t, ctl.OnDelete(t.Context(), crt))
	_, err = tst.C.OcteliumC.CoreC().GetSecret(t.Context(), &rmetav1.GetOptions{Name: name})
	assert.True(t, grpcerr.IsNotFound(err))
}
