// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watcher

import (
	"context"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestNeedsIssuance(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	watcher := InitWatcher(tst.C.OcteliumC)

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec:   &enterprisev1.Certificate_Spec{},
			Status: &enterprisev1.Certificate_Status{},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crt, err = watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Nil(t, crt.Status.Issuance)
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					State: enterprisev1.Certificate_Status_Issuance_FAILED,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, crtI.Status.Issuance.State, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED)
		assert.True(t, len(crtI.Status.LastIssuances) == 1)
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					State: enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.True(t, pbutils.IsEqual(crt, crtI))
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					IssuanceStartedAt: pbutils.Now(),
					State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.True(t, pbutils.IsEqual(crt, crtI))
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					IssuanceStartedAt: pbutils.Timestamp(time.Now().Add(-2 * time.Hour)),
					State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.False(t, pbutils.IsEqual(crt, crtI))
		assert.True(t, len(crtI.Status.LastIssuances) == 1)
		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED, crtI.Status.Issuance.State)
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					IssuanceStartedAt: pbutils.Now(),
					ExpiresAt:         pbutils.Timestamp(time.Now().Add(3 * 30 * 24 * time.Hour)),
					State:             enterprisev1.Certificate_Status_Issuance_SUCCESS,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.True(t, pbutils.IsEqual(crt, crtI))
	}

	{
		crt, err := watcher.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: &enterprisev1.Certificate_Status{
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					IssuanceStartedAt: pbutils.Now(),
					ExpiresAt:         pbutils.Timestamp(time.Now().Add(20 * 24 * time.Hour)),
					State:             enterprisev1.Certificate_Status_Issuance_SUCCESS,
				},
			},
		})
		assert.Nil(t, err)

		err = watcher.doCheckAndIssueCrt(ctx, crt)
		assert.Nil(t, err)

		crtI, err := watcher.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.False(t, pbutils.IsEqual(crt, crtI))
		assert.True(t, len(crtI.Status.LastIssuances) == 1)
		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED, crtI.Status.Issuance.State)
	}

}
