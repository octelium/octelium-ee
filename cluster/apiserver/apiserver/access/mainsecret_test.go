// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"testing"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func tstSecret(name, value string) *accessv1.Secret {
	return &accessv1.Secret{
		Metadata: &metav1.Metadata{
			Name: name,
		},
		Spec: &accessv1.Secret_Spec{},
		Data: &accessv1.Secret_Data{
			Type: &accessv1.Secret_Data_Value{
				Value: value,
			},
		},
	}
}

func TestSecret(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	name := utilrand.GetRandomStringCanonical(8)

	item, err := srv.CreateSecret(ctx, tstSecret(name, "xoxb-token"))
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, item.Data)

	{
		stored, err := octeliumC.AccessC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: item.Metadata.Uid,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "xoxb-token", uaccessv1.ToSecret(stored).GetValueStr())
	}

	{
		ret, err := srv.GetSecret(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
		assert.Nil(t, ret.Data)
	}

	{
		_, err := srv.CreateSecret(ctx, tstSecret(name, "another"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		itemList, err := srv.ListSecret(ctx, &accessv1.ListSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(itemList.Items))
		assert.Nil(t, itemList.Items[0].Data)
	}

	{
		ret, err := srv.UpdateSecret(ctx, tstSecret(name, "xoxb-next"))
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, ret.Data)

		stored, err := octeliumC.AccessC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: item.Metadata.Uid,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "xoxb-next", uaccessv1.ToSecret(stored).GetValueStr())
	}

	{
		_, err := srv.DeleteSecret(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetSecret(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestSecretEmptyDataIsRefused(t *testing.T) {
	ctx, srv, _ := newIntegrationTest(t)

	{
		_, err := srv.CreateSecret(ctx, &accessv1.Secret{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.Secret_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateSecret(ctx, tstSecret(utilrand.GetRandomStringCanonical(8), ""))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateSecret(ctx, tstSecret("", "value"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestSecretUsedByAnIntegrationCannotBeDeleted(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	botToken := tstCreateSecret(ctx, t, octeliumC)
	signingSecret := tstCreateSecret(ctx, t, octeliumC)

	integration, err := srv.CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: tstSlackIntegrationSpec(botToken, signingSecret),
	})
	assert.Nil(t, err, "%+v", err)

	{
		_, err := srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: botToken})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: signingSecret})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.DeleteIntegration(ctx, &metav1.DeleteOptions{
			Uid: integration.Metadata.Uid,
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: botToken})
		assert.Nil(t, err, "%+v", err)
	}
}

func TestIntegrationUnknownSecretIsRefused(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	_, err := srv.CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: tstSlackIntegrationSpec(utilrand.GetRandomStringCanonical(8),
			tstCreateSecret(ctx, t, octeliumC)),
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}
