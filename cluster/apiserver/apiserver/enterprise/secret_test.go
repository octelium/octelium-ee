// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"fmt"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestSecret(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		secretName := fmt.Sprintf("secret-%s", utilrand.GetRandomStringLowercase(4))
		secretValue := []byte("topsecret")
		sec, err := srv.CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name: secretName,
			},
			Spec: &enterprisev1.Secret_Spec{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_ValueBytes{
					ValueBytes: []byte(secretValue),
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, sec.Data)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, secretValue, uenterprisev1.ToSecret(secret).GetValueBytes())

		secret.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		}
		secU, err := srv.UpdateSecret(ctx, secret)
		assert.Nil(t, err)
		assert.Nil(t, secU.Data)

		secI, err := srv.GetSecret(ctx, &metav1.GetOptions{Uid: secret.Metadata.Uid})
		assert.Nil(t, err)
		assert.Equal(t, secretName, secI.Metadata.Name)
		assert.Nil(t, secI.Data)

		secList, err := srv.ListSecret(ctx, &enterprisev1.ListSecretOptions{})
		assert.Nil(t, err)

		for _, sec := range secList.Items {
			assert.Nil(t, sec.Data)
		}

		_, err = srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: secretName})
		assert.Nil(t, err)
	}

}
