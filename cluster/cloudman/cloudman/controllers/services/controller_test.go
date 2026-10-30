// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package svccontroller

import (
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedServiceCertificateLifecycle(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	ctl := NewController(tst.C.OcteliumC)
	svc := &corev1.Service{
		Metadata: &metav1.Metadata{Name: "portal.default", Uid: vutils.UUIDv4()},
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{NamespaceRef: &metav1.ObjectReference{Name: "default"}},
	}
	assert.False(t, needsCert(svc))
	assert.NoError(t, doSetCert(t.Context(), nil, svc))
	assert.NoError(t, ctl.OnDelete(t.Context(), svc))
	svc.Status.ManagedService = &corev1.Service_Status_ManagedService{HasSubdomain: true}
	assert.True(t, needsCert(svc))
	require.NoError(t, doSetCert(t.Context(), tst.C.OcteliumC, svc))
	crt, err := tst.C.OcteliumC.EnterpriseC().GetCertificate(t.Context(), &rmetav1.GetOptions{Name: "svc-portal-default"})
	require.NoError(t, err)
	assert.Equal(t, enterprisev1.Certificate_Spec_MANUAL, crt.Spec.Mode)
	assert.Equal(t, svc.Metadata.Uid, crt.Status.ServiceRef.Uid)
	require.NoError(t, doSetCert(t.Context(), tst.C.OcteliumC, svc))
	assert.NoError(t, ctl.OnDelete(t.Context(), svc))
	assert.NoError(t, ctl.OnDelete(t.Context(), svc))
}
