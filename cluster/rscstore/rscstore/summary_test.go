// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSummaryCommonOptions(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	now := time.Now().UTC()

	insertUsersAt := func(n int, createdAt time.Time) {
		for range n {
			err := srv.insertResource(ctx, &corev1.User{
				ApiVersion: ucorev1.APIVersion,
				Kind:       ucorev1.KindUser,
				Metadata: &metav1.Metadata{
					Name:            utilrand.GetRandomStringCanonical(8),
					Uid:             vutils.UUIDv4(),
					ResourceVersion: vutils.UUIDv7(),
					CreatedAt:       pbutils.Timestamp(createdAt),
				},
				Spec: &corev1.User_Spec{
					Type: corev1.User_Spec_HUMAN,
				},
			})
			assert.Nil(t, err)
		}
	}

	nOld := 7
	nRecent := 11

	insertUsersAt(nOld, now.Add(-100*time.Minute))
	insertUsersAt(nRecent, now.Add(-10*time.Minute))

	getTotal := func(common *vmetav1.CommonSummaryOptions) uint32 {
		resp, err := srv.getSummaryCoreUser(ctx, &vcorev1.GetUserSummaryRequest{
			Common: common,
		})
		assert.Nil(t, err, "%+v", err)
		return resp.TotalNumber
	}

	assert.Equal(t, uint32(nOld+nRecent), getTotal(nil))
	assert.Equal(t, uint32(nOld+nRecent), getTotal(&vmetav1.CommonSummaryOptions{}))

	assert.Equal(t, uint32(nRecent), getTotal(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now.Add(-50 * time.Minute)),
	}))

	assert.Equal(t, uint32(nOld), getTotal(&vmetav1.CommonSummaryOptions{
		To: pbutils.Timestamp(now.Add(-50 * time.Minute)),
	}))

	assert.Equal(t, uint32(nRecent), getTotal(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now.Add(-50 * time.Minute)),
		To:   pbutils.Timestamp(now.Add(-5 * time.Minute)),
	}))

	assert.Equal(t, uint32(nOld+nRecent), getTotal(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now.Add(-200 * time.Minute)),
		To:   pbutils.Timestamp(now),
	}))

	assert.Equal(t, uint32(0), getTotal(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now.Add(-5 * time.Minute)),
		To:   pbutils.Timestamp(now),
	}))

	{
		_, err := srv.getSummaryCoreSession(ctx, &vcorev1.GetSessionSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From: pbutils.Timestamp(now.Add(-50 * time.Minute)),
				To:   pbutils.Timestamp(now),
			},
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		_, err := srv.getSummaryCoreUser(ctx, &vcorev1.GetUserSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From: pbutils.Timestamp(now),
				To:   pbutils.Timestamp(now.Add(-time.Hour)),
			},
		})
		assert.NotNil(t, err)
	}
}

func TestValidateCommonSummaryOptions(t *testing.T) {
	now := time.Now().UTC()

	assert.Nil(t, validateCommonSummaryOptions(nil))
	assert.Nil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{}))

	assert.Nil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now.Add(-time.Hour)),
		To:   pbutils.Timestamp(now),
	}))

	assert.Nil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now),
		To:   pbutils.Timestamp(now),
	}))

	assert.NotNil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{
		From: pbutils.Timestamp(now),
		To:   pbutils.Timestamp(now.Add(-time.Hour)),
	}))

	assert.NotNil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{
		From: &timestamppb.Timestamp{Seconds: -100000000000},
	}))

	assert.NotNil(t, validateCommonSummaryOptions(&vmetav1.CommonSummaryOptions{
		To: &timestamppb.Timestamp{Seconds: -100000000000},
	}))
}
