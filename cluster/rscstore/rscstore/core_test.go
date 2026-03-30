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
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestCoreUser(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	for range utilrand.GetRandomRangeMath(100, 400) {
		rsc := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.User_Spec{
				Type: func() corev1.User_Spec_Type {
					if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
						return corev1.User_Spec_HUMAN
					}
					return corev1.User_Spec_WORKLOAD
				}(),
			},
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)
	}

	_, err = srv.getSummaryCoreUser(ctx, &vcorev1.GetUserSummaryRequest{})
	assert.Nil(t, err)

}

func TestCoreSession(t *testing.T) {
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

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  srv.octeliumC,
		IsEmbedded: true,
	})

	// uid := vutils.UUIDv4()
	for range 100 {
		usrT, err := tstuser.NewUserWithType(tst.C.OcteliumC, coreSrv, nil, nil,
			corev1.User_Spec_HUMAN,
			func() corev1.Session_Status_Type {
				if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
					return corev1.Session_Status_CLIENT
				}
				return corev1.Session_Status_CLIENTLESS
			}())
		assert.Nil(t, err)

		/*
			rsc := &corev1.Session{
				ApiVersion: ucorev1.APIVersion,
				Kind:       ucorev1.KindSession,
				Metadata: &metav1.Metadata{
					Name:            utilrand.GetRandomStringCanonical(8),
					Uid:             vutils.UUIDv4(),
					ResourceVersion: vutils.UUIDv7(),
					CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
				},
				Spec: &corev1.Session_Spec{
					State: func() corev1.Session_Spec_State {
						if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
							return corev1.Session_Spec_ACTIVE
						}
						return corev1.Session_Spec_REJECTED
					}(),
				},
				Status: &corev1.Session_Status{
					Type: func() corev1.Session_Status_Type {
						if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
							return corev1.Session_Status_CLIENT
						}
						return corev1.Session_Status_CLIENTLESS
					}(),

					IsConnected: getRandomBool(),
					UserRef: &metav1.ObjectReference{
						Uid: uid,
					},
				},
			}
		*/

		err = srv.insertResource(ctx, usrT.Session)
		assert.Nil(t, err)
	}

	{
		_, err := srv.getSummaryCoreSession(ctx, &vcorev1.GetSessionSummaryRequest{})
		assert.Nil(t, err)
	}

	sessList, err := srv.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{
		OrderBy: []*rmetav1.ListOptions_OrderBy{
			{
				Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
				Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
			},
		},
	})
	assert.Nil(t, err)
	sess := sessList.Items[0]

	sessI, err := srv.getSessionFromActorRef(ctx, umetav1.GetObjectReference(sess))
	assert.Nil(t, err)
	assert.True(t, pbutils.IsEqual(sess, sessI))

}

func TestCorePolicy(t *testing.T) {
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

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	// uid := vutils.UUIDv4()
	for range utilrand.GetRandomRangeMath(1000, 4000) {
		rsc := &corev1.Policy{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindPolicy,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.Policy_Spec{
				IsDisabled: getRandomBool(),
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Effect:    corev1.Policy_Spec_Rule_ALLOW,
						Condition: &corev1.Condition{},
					},
				},
			},
			Status: &corev1.Policy_Status{},
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)
	}

	{
		_, err := srv.getSummaryCorePolicy(ctx, &vcorev1.GetPolicySummaryRequest{})
		assert.Nil(t, err)
	}
}

func getRandomBool() bool {
	return utilrand.GetRandomRangeMath(1, 500)%2 == 0
}
