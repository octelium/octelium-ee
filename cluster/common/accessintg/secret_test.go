// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func tstCreateAccessSecret(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface, value string) string {
	sec, err := octeliumC.AccessC().CreateSecret(ctx, &accessv1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Secret_Spec{},
		Data: &accessv1.Secret_Data{
			Type: &accessv1.Secret_Data_Value{
				Value: value,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sec.Metadata.Name
}

func TestGetSecretValue(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	name := tstCreateAccessSecret(ctx, t, octeliumC, "  xoxb-token  ")

	val, err := GetSecretValue(ctx, octeliumC, name)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "xoxb-token", val)
}

func TestGetSecretValueBytes(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	sec, err := octeliumC.AccessC().CreateSecret(ctx, &accessv1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Secret_Spec{},
		Data: &accessv1.Secret_Data{
			Type: &accessv1.Secret_Data_ValueBytes{
				ValueBytes: []byte("xoxb-bytes"),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	val, err := GetSecretValue(ctx, octeliumC, sec.Metadata.Name)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "xoxb-bytes", val)
}

func TestGetSecretValueEmptyName(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	_, err := GetSecretValue(ctx, octeliumC, "")
	assert.NotNil(t, err)
}

func TestGetSecretValueNotFound(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	_, err := GetSecretValue(ctx, octeliumC, utilrand.GetRandomStringCanonical(8))
	assert.NotNil(t, err)
}

func TestGetSecretValueDoesNotReadTheEnterpriseSecrets(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	sec, err := octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: "xoxb-token",
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	_, err = GetSecretValue(ctx, octeliumC, sec.Metadata.Name)
	assert.NotNil(t, err)
}
