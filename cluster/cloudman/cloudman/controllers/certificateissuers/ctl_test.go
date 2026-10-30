// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package certificateissuers

import (
	"context"
	"errors"
	"testing"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type recordingIssuerSetter struct {
	calls int
	force bool
	err   error
}

func (s *recordingIssuerSetter) SetCertificateIssuer(_ context.Context, _ *enterprisev1.CertificateIssuer, force bool) error {
	s.calls++
	s.force = force
	return s.err
}

func TestCertificateIssuerControllerReconciliation(t *testing.T) {
	setter := &recordingIssuerSetter{}
	ctl := NewController(nil, setter)
	iss := &enterprisev1.CertificateIssuer{Spec: &enterprisev1.CertificateIssuer_Spec{}}
	assert.NoError(t, ctl.OnAdd(t.Context(), iss))
	assert.Zero(t, setter.calls)
	iss.Spec.Type = &enterprisev1.CertificateIssuer_Spec_Acme{Acme: &enterprisev1.CertificateIssuer_Spec_ACME{Server: "https://acme.example.com/directory"}}
	assert.NoError(t, ctl.OnAdd(t.Context(), iss))
	assert.Equal(t, 1, setter.calls)
	assert.False(t, setter.force)
	old := proto.Clone(iss).(*enterprisev1.CertificateIssuer)
	assert.NoError(t, ctl.OnUpdate(t.Context(), iss, old))
	assert.False(t, setter.force)
	iss.Spec.GetAcme().Server = "https://other.example.com/directory"
	assert.NoError(t, ctl.OnUpdate(t.Context(), iss, old))
	assert.True(t, setter.force)
	setter.err = errors.New("ACME unavailable")
	assert.ErrorIs(t, ctl.OnAdd(t.Context(), iss), setter.err)
	assert.NoError(t, ctl.OnDelete(t.Context(), iss))
}
