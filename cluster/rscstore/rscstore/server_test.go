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
	"fmt"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestTransformRsc(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	{
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
				Email: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(9)),
			},
			Status: &corev1.User_Status{},
		}

		rscJSON, err := pbutils.MarshalJSON(rsc, false)
		assert.Nil(t, err)
		_, err = srv.getRSCStr(rscJSON)
		assert.Nil(t, err)

	}

}
