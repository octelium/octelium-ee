// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func newTestWorkKey() workKey {
	return workKey{
		typ: workTypeCertificate,
		uid: utilrand.GetRandomStringCanonical(8),
	}
}

func TestWorkKey(t *testing.T) {
	assert.Equal(t, "Certificate/abc", workKey{typ: workTypeCertificate, uid: "abc"}.String())
	assert.Equal(t, "CertificateIssuer/abc",
		workKey{typ: workTypeCertificateIssuer, uid: "abc"}.String())
	assert.Equal(t, "Unknown/abc", workKey{uid: "abc"}.String())
}

func TestEnqueue(t *testing.T) {
	c := NewController(nil)
	t.Cleanup(c.shutdown)

	key := newTestWorkKey()

	c.enqueue(workKey{}, false)
	c.enqueue(workKey{typ: workTypeCertificate}, false)
	_, ok := c.takePending()
	assert.False(t, ok)

	c.enqueue(key, false)
	c.enqueue(key, false)

	got, ok := c.takePending()
	assert.True(t, ok)
	assert.Equal(t, key, got)

	_, ok = c.takePending()
	assert.False(t, ok)

	c.enqueue(key, false)
	_, ok = c.takePending()
	assert.False(t, ok)

	c.finish(key)

	got, ok = c.takePending()
	assert.True(t, ok)
	assert.Equal(t, key, got)

	c.finish(key)
	_, ok = c.takePending()
	assert.False(t, ok)
}

func TestEnqueueRespectsBackoff(t *testing.T) {
	c := NewController(nil)
	t.Cleanup(c.shutdown)

	key := newTestWorkKey()
	err := errors.Errorf("Could not obtain the Certificate")

	c.enqueue(key, false)
	_, ok := c.takePending()
	assert.True(t, ok)

	delay := c.scheduleRetry(key, err)
	assert.True(t, delay >= time.Second && delay <= maxRetryDelay, "%s", delay)

	c.finish(key)
	_, ok = c.takePending()
	assert.False(t, ok)

	c.enqueue(key, false)
	_, ok = c.takePending()
	assert.False(t, ok)

	c.mu.Lock()
	failures := c.failures[key]
	c.mu.Unlock()
	assert.Equal(t, uint32(1), failures)

	c.enqueue(key, true)

	got, ok := c.takePending()
	assert.True(t, ok)
	assert.Equal(t, key, got)

	c.mu.Lock()
	failures = c.failures[key]
	_, hasTimer := c.timers[key]
	c.mu.Unlock()
	assert.Equal(t, uint32(1), failures)
	assert.False(t, hasTimer)

	c.resetFailures(key)
	c.finish(key)

	c.mu.Lock()
	_, hasFailures := c.failures[key]
	_, hasNotBefore := c.notBefore[key]
	c.mu.Unlock()
	assert.False(t, hasFailures)
	assert.False(t, hasNotBefore)
}

func TestRetryTimer(t *testing.T) {
	c := NewController(nil)
	t.Cleanup(c.shutdown)

	key := newTestWorkKey()

	c.mu.Lock()
	c.notBefore[key] = time.Now().Add(-time.Second)
	c.mu.Unlock()

	c.onRetryTimer(key)

	got, ok := c.takePending()
	assert.True(t, ok)
	assert.Equal(t, key, got)

	c.mu.Lock()
	c.notBefore[key] = time.Now().Add(-time.Second)
	c.mu.Unlock()

	c.onRetryTimer(key)

	c.mu.Lock()
	_, isDirty := c.dirty[key]
	c.mu.Unlock()
	assert.True(t, isDirty)

	c.finish(key)

	got, ok = c.takePending()
	assert.True(t, ok)
	assert.Equal(t, key, got)
}

func TestShutdownStopsEnqueue(t *testing.T) {
	c := NewController(nil)

	key := newTestWorkKey()
	c.scheduleRetry(key, errors.Errorf("Could not obtain the Certificate"))

	c.mu.Lock()
	_, hasTimer := c.timers[key]
	c.mu.Unlock()
	assert.True(t, hasTimer)

	c.shutdown()

	c.mu.Lock()
	assert.Equal(t, 0, len(c.timers))
	c.mu.Unlock()

	c.enqueue(key, true)
	_, ok := c.takePending()
	assert.False(t, ok)
}

func TestSetCertificate(t *testing.T) {
	ctx := context.Background()

	c := NewController(nil)
	t.Cleanup(c.shutdown)

	assert.NotNil(t, c.SetCertificate(ctx, nil))
	assert.NotNil(t, c.SetCertificate(ctx, &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{},
	}))

	newCrt := func(mode enterprisev1.Certificate_Spec_Mode,
		issuerRef *metav1.ObjectReference) *enterprisev1.Certificate {
		return &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Uid:  utilrand.GetRandomStringCanonical(8),
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: mode,
			},
			Status: &enterprisev1.Certificate_Status{
				CertificateIssuerRef: issuerRef,
			},
		}
	}

	{
		crt := newCrt(enterprisev1.Certificate_Spec_MANUAL, &metav1.ObjectReference{
			Uid: utilrand.GetRandomStringCanonical(8),
		})
		assert.Nil(t, c.SetCertificate(ctx, crt))
		_, ok := c.takePending()
		assert.False(t, ok)
	}

	{
		crt := newCrt(enterprisev1.Certificate_Spec_MANAGED, nil)
		assert.Nil(t, c.SetCertificate(ctx, crt))
		_, ok := c.takePending()
		assert.False(t, ok)
	}

	{
		crt := newCrt(enterprisev1.Certificate_Spec_MANAGED, &metav1.ObjectReference{
			Uid: utilrand.GetRandomStringCanonical(8),
		})
		assert.Nil(t, c.SetCertificate(ctx, crt))

		got, ok := c.takePending()
		assert.True(t, ok)
		assert.Equal(t, crt.Metadata.Uid, got.uid)
		c.finish(got)
	}
}

func TestSetCertificateIssuer(t *testing.T) {
	ctx := context.Background()

	c := NewController(nil)
	t.Cleanup(c.shutdown)

	assert.NotNil(t, c.SetCertificateIssuer(ctx, nil, false))

	{
		iss := &enterprisev1.CertificateIssuer{
			Metadata: &metav1.Metadata{
				Uid: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.CertificateIssuer_Spec{},
		}

		assert.Nil(t, c.SetCertificateIssuer(ctx, iss, true))
		_, ok := c.takePending()
		assert.False(t, ok)
	}

	iss := &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{
			Uid: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.CertificateIssuer_Spec{
			Type: &enterprisev1.CertificateIssuer_Spec_Acme{
				Acme: &enterprisev1.CertificateIssuer_Spec_ACME{},
			},
		},
	}

	assert.Nil(t, c.SetCertificateIssuer(ctx, iss, true))
	assert.True(t, c.hasIssuerForce(iss.Metadata.Uid))

	got, ok := c.takePending()
	assert.True(t, ok)
	assert.Equal(t, iss.Metadata.Uid, got.uid)
	assert.Equal(t, workTypeCertificateIssuer, got.typ)
	c.finish(got)

	c.cleanupIssuerForce(map[string]struct{}{
		iss.Metadata.Uid: {},
	})
	assert.True(t, c.hasIssuerForce(iss.Metadata.Uid))

	c.cleanupIssuerForce(map[string]struct{}{})
	assert.False(t, c.hasIssuerForce(iss.Metadata.Uid))
}

func TestNeedsReconcile(t *testing.T) {
	now := time.Now()

	newCrt := func(issuance *enterprisev1.Certificate_Status_Issuance) *enterprisev1.Certificate {
		return &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Uid: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				CertificateIssuerRef: &metav1.ObjectReference{
					Uid: utilrand.GetRandomStringCanonical(8),
				},
				Issuance: issuance,
			},
		}
	}

	assert.False(t, needsReconcile(nil, now))
	assert.True(t, needsReconcile(newCrt(nil), now))

	{
		crt := newCrt(nil)
		crt.Spec.Mode = enterprisev1.Certificate_Spec_MANUAL
		assert.False(t, needsReconcile(crt, now))
	}

	{
		crt := newCrt(nil)
		crt.Status.CertificateIssuerRef = nil
		assert.False(t, needsReconcile(crt, now))
	}

	assert.True(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State: enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
	}), now))

	assert.True(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State: enterprisev1.Certificate_Status_Issuance_FAILED,
	}), now))

	assert.False(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
		IssuanceStartedAt: pbutils.Now(),
	}), now))

	assert.True(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
		IssuanceStartedAt: pbutils.Timestamp(now.Add(-2 * staleIssuanceAfter)),
	}), now))

	assert.False(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
		ExpiresAt: pbutils.Timestamp(now.Add(2 * renewBefore)),
	}), now))

	assert.True(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
		ExpiresAt: pbutils.Timestamp(now.Add(renewBefore / 2)),
	}), now))

	assert.True(t, needsReconcile(newCrt(&enterprisev1.Certificate_Status_Issuance{
		State: enterprisev1.Certificate_Status_Issuance_SUCCESS,
	}), now))
}

func TestIsSameAttempt(t *testing.T) {
	createdAt := pbutils.Now()
	startedAt := pbutils.Now()

	crt := &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Uid: utilrand.GetRandomStringCanonical(8),
		},
		Status: &enterprisev1.Certificate_Status{
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
				CreatedAt:         createdAt,
				IssuanceStartedAt: startedAt,
			},
		},
	}

	attempt := getIssuanceAttempt(crt)

	assert.True(t, isSameAttempt(crt, attempt))
	assert.False(t, isSameAttempt(crt, nil))
	assert.False(t, isSameAttempt(nil, attempt))

	crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_SUCCESS
	assert.False(t, isSameAttempt(crt, attempt))

	crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_ISSUING
	crt.Status.Issuance.IssuanceStartedAt = pbutils.Timestamp(
		startedAt.AsTime().Add(time.Second))
	assert.False(t, isSameAttempt(crt, attempt))
}
