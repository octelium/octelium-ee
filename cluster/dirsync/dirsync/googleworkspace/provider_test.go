// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package googleworkspace

import (
	"context"
	"fmt"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
)

func TestUser(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	dp, err := fakeC.OcteliumC.EnterpriseC().CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{
				GoogleWorkspace: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace{
					Customer: utilrand.GetRandomStringCanonical(8),
				},
			},
		},
		Status: &enterprisev1.DirectoryProvider_Status{
			Id: utilrand.GetRandomStringCanonical(4),
		},
	})
	assert.Nil(t, err)

	p, err := NewProvider(ctx, fakeC.OcteliumC, &Opts{
		DirectorProviderRef: umetav1.GetObjectReference(dp),
	})
	assert.Nil(t, err)

	{
		gUsr := &admin.User{
			Id:           fmt.Sprintf("%d", utilrand.GetRandomRangeMath(10_000_000, 100_0000_000)),
			PrimaryEmail: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(8)),
		}

		err = p.setUser(ctx, gUsr, dp)
		assert.Nil(t, err)

		usr, err := p.getUser(ctx, gUsr)
		assert.Nil(t, err)

		dpUsr, err := p.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
			Name: usr.Metadata.Name,
		})
		assert.Nil(t, err)

		assert.Equal(t, usr.Metadata.Uid, dpUsr.Status.UserRef.Uid)

		assert.Equal(t, usr.Spec.Email, gUsr.PrimaryEmail)

		{
			err = p.setUser(ctx, gUsr, dp)
			assert.Nil(t, err)
		}

		{
			gUsr.PrimaryEmail = fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(8))

			err = p.setUser(ctx, gUsr, dp)
			assert.Nil(t, err)

			usr, err := p.getUser(ctx, gUsr)
			assert.Nil(t, err)
			assert.Equal(t, usr.Spec.Email, gUsr.PrimaryEmail)

			{
				nUsr, err := p.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
					Name: p.genUserName(gUsr),
				})
				assert.Nil(t, err)
				assert.Equal(t, usr.Metadata.Uid, nUsr.Metadata.Uid)

				dpUsr, err := p.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
					Name: p.genUserName(gUsr),
				})
				assert.Nil(t, err)

				assert.Equal(t, nUsr.Metadata.Uid, dpUsr.Status.UserRef.Uid)
			}
		}

		{
			gUsr.Suspended = true

			err = p.setUser(ctx, gUsr, dp)
			assert.Nil(t, err)

			usr, err := p.getUser(ctx, gUsr)
			assert.Nil(t, err)
			assert.True(t, usr.Spec.IsDisabled)
		}

		{
			gUsr.Suspended = false

			err = p.setUser(ctx, gUsr, dp)
			assert.Nil(t, err)

			usr, err := p.getUser(ctx, gUsr)
			assert.Nil(t, err)
			assert.False(t, usr.Spec.IsDisabled)
		}

		{
			err = p.deleteUser(ctx, gUsr)
			assert.Nil(t, err)

			err = p.deleteUser(ctx, gUsr)
			assert.Nil(t, err)
			_, err = p.getUser(ctx, gUsr)
			assert.NotNil(t, err)

			assert.True(t, grpcerr.IsNotFound(err))
		}
	}
}
