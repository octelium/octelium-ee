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
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
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
		secretName := tstSecretName()
		secretValue := []byte("topsecret")
		sec, err := srv.CreateSecret(ctx, tstSecret(secretName, tstSecretValueBytes(secretValue)))
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, sec.Data)
		assert.NotEmpty(t, sec.Metadata.Uid)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, secretValue, uenterprisev1.ToSecret(secret).GetValueBytes())

		_, err = srv.CreateSecret(ctx, tstSecret(secretName, tstSecretValueBytes(secretValue)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)

		secret.Data = tstSecretValue("updated")
		secU, err := srv.UpdateSecret(ctx, secret)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, secU.Data)

		secret, err = srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "updated", uenterprisev1.ToSecret(secret).GetValueStr())

		secI, err := srv.GetSecret(ctx, &metav1.GetOptions{Uid: secret.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, secretName, secI.Metadata.Name)
		assert.Nil(t, secI.Data)

		secList, err := srv.ListSecret(ctx, &enterprisev1.ListSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, secList)
		for _, sec := range secList.Items {
			assert.Nil(t, sec.Data)
		}

		secList, err = srv.ListSecret(ctx, nil)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, secList)

		_, err = srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetSecret(ctx, &metav1.GetOptions{Name: secretName})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestSecretDataTypes(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		secretName := tstSecretName()
		_, err := srv.CreateSecret(ctx, tstSecret(secretName, tstSecretValue("topsecret")))
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "topsecret", uenterprisev1.ToSecret(secret).GetValueStr())
	}

	{
		secretName := tstSecretName()
		_, err := srv.CreateSecret(ctx, tstSecret(secretName, tstSecretValueBytes([]byte("topsecret"))))
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []byte("topsecret"), uenterprisev1.ToSecret(secret).GetValueBytes())
	}

	{
		secretName := tstSecretName()
		_, err := srv.CreateSecret(ctx, tstSecret(secretName, tstSecretDataMap(map[string][]byte{
			"username": []byte("octelium"),
			"password": []byte("topsecret"),
		})))
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []byte("octelium"), secret.Data.GetDataMap().GetMap()["username"])
		assert.Equal(t, []byte("topsecret"), secret.Data.GetDataMap().GetMap()["password"])
	}

	{
		secretName := tstSecretName()
		attrs, err := structpb.NewStruct(map[string]any{
			"username": "octelium",
			"enabled":  true,
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.CreateSecret(ctx, tstSecret(secretName, tstSecretAttrs(attrs)))
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "octelium", secret.Data.GetAttrs().GetFields()["username"].GetStringValue())
		assert.True(t, secret.Data.GetAttrs().GetFields()["enabled"].GetBoolValue())
	}

	{
		secretName := tstSecretName()
		sec := tstSecret(secretName, tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_Value{
				Value: "spec-secret",
			},
		}

		_, err := srv.CreateSecret(ctx, sec)
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "spec-secret", secret.Spec.GetData().GetValue())
	}

	{
		secretName := tstSecretName()
		sec := tstSecret(secretName, tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_ValueBytes{
				ValueBytes: []byte("spec-secret"),
			},
		}

		_, err := srv.CreateSecret(ctx, sec)
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []byte("spec-secret"), secret.Spec.GetData().GetValueBytes())
	}

	{
		secretName := tstSecretName()
		attrs, err := structpb.NewStruct(map[string]any{
			"scope": "sync",
		})
		assert.Nil(t, err, "%+v", err)

		sec := tstSecret(secretName, tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_Attrs{
				Attrs: attrs,
			},
		}

		_, err = srv.CreateSecret(ctx, sec)
		assert.Nil(t, err, "%+v", err)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "sync", secret.Spec.GetData().GetAttrs().GetFields()["scope"].GetStringValue())
	}
}

func TestValidateSecret(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretValue("topsecret")))
		assert.Nil(t, err, "%+v", err)
	}

	{
		attrs, err := structpb.NewStruct(map[string]any{"key": "value"})
		assert.Nil(t, err, "%+v", err)
		err = srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretAttrs(attrs)))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, &enterprisev1.Secret{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: tstSecretName()},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: tstSecretName()},
			Spec:     &enterprisev1.Secret_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: tstSecretName()},
			Spec:     &enterprisev1.Secret_Spec{},
			Data:     &enterprisev1.Secret_Data{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretValue("")))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretValue(strings.Repeat("a", maxSecretDataBytes+1))))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretValueBytes(nil)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretValueBytes([]byte(strings.Repeat("a", maxSecretDataBytes+1)))))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_DataMap_{},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretDataMap(map[string][]byte{})))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretDataMap(map[string][]byte{
			"": []byte("value"),
		})))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretDataMap(map[string][]byte{
			"bad\nkey": []byte("value"),
		})))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretDataMap(map[string][]byte{
			"key": nil,
		})))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretDataMap(map[string][]byte{
			"key": []byte(strings.Repeat("a", maxSecretDataBytes+1)),
		})))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecret(ctx, tstSecret(tstSecretName(), &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Attrs{},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		attrs, err := structpb.NewStruct(map[string]any{})
		assert.Nil(t, err, "%+v", err)
		err = srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretAttrs(attrs)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		attrs, err := structpb.NewStruct(map[string]any{
			"key": strings.Repeat("a", maxSecretDataBytes+1),
		})
		assert.Nil(t, err, "%+v", err)
		err = srv.validateSecret(ctx, tstSecret(tstSecretName(), tstSecretAttrs(attrs)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		sec := tstSecret(tstSecretName(), tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{}
		err := srv.validateSecret(ctx, sec)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		sec := tstSecret(tstSecretName(), tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_Value{},
		}
		err := srv.validateSecret(ctx, sec)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		sec := tstSecret(tstSecretName(), tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_ValueBytes{
				ValueBytes: []byte(strings.Repeat("a", maxSecretDataBytes+1)),
			},
		}
		err := srv.validateSecret(ctx, sec)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		sec := tstSecret(tstSecretName(), tstSecretValue("topsecret"))
		sec.Spec.Data = &enterprisev1.Secret_Spec_Data{
			Type: &enterprisev1.Secret_Spec_Data_Attrs{},
		}
		err := srv.validateSecret(ctx, sec)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestSecretSystem(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		secretName := tstSecretName()
		sec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           secretName,
				IsSystem:       true,
				IsUserHidden:   true,
				IsSystemHidden: true,
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data:   tstSecretValue("topsecret"),
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetSecret(ctx, &metav1.GetOptions{Name: secretName})
		assert.NotNil(t, err)

		arg := tstSecret(secretName, tstSecretValue("updated"))
		arg.Metadata.Uid = sec.Metadata.Uid
		_, err = srv.UpdateSecret(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)

		_, err = srv.DeleteSecret(ctx, &metav1.DeleteOptions{Uid: sec.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
	}
}

func TestSecretStatusIsServerOwned(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		secretName := tstSecretName()
		attrs, err := structpb.NewStruct(map[string]any{"owner": "controller"})
		assert.Nil(t, err, "%+v", err)

		sec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: secretName},
			Spec:     &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{
				Ext: map[string]*structpb.Struct{
					"test": attrs,
				},
			},
			Data: tstSecretValue("topsecret"),
		})
		assert.Nil(t, err, "%+v", err)

		arg := tstSecret(secretName, tstSecretValue("updated"))
		arg.Metadata.Uid = sec.Metadata.Uid
		arg.Status = &enterprisev1.Secret_Status{}
		updated, err := srv.UpdateSecret(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, updated.Data)

		secret, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Uid: sec.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "controller", secret.Status.Ext["test"].GetFields()["owner"].GetStringValue())
		assert.Equal(t, "updated", uenterprisev1.ToSecret(secret).GetValueStr())
	}
}

func TestSecretErrors(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		_, err := srv.GetSecret(ctx, &metav1.GetOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetSecret(ctx, &metav1.GetOptions{Name: tstSecretName()})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.UpdateSecret(ctx, tstSecret(tstSecretName(), tstSecretValue("topsecret")))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.DeleteSecret(ctx, &metav1.DeleteOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.DeleteSecret(ctx, &metav1.DeleteOptions{Name: tstSecretName()})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func tstSecretName() string {
	return fmt.Sprintf("secret-%s", utilrand.GetRandomStringLowercase(8))
}

func tstSecret(name string, data *enterprisev1.Secret_Data) *enterprisev1.Secret {
	return &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: name,
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: data,
	}
}

func tstSecretValue(value string) *enterprisev1.Secret_Data {
	return &enterprisev1.Secret_Data{
		Type: &enterprisev1.Secret_Data_Value{
			Value: value,
		},
	}
}

func tstSecretValueBytes(value []byte) *enterprisev1.Secret_Data {
	return &enterprisev1.Secret_Data{
		Type: &enterprisev1.Secret_Data_ValueBytes{
			ValueBytes: value,
		},
	}
}

func tstSecretDataMap(value map[string][]byte) *enterprisev1.Secret_Data {
	return &enterprisev1.Secret_Data{
		Type: &enterprisev1.Secret_Data_DataMap_{
			DataMap: &enterprisev1.Secret_Data_DataMap{
				Map: value,
			},
		},
	}
}

func tstSecretAttrs(value *structpb.Struct) *enterprisev1.Secret_Data {
	return &enterprisev1.Secret_Data{
		Type: &enterprisev1.Secret_Data_Attrs{
			Attrs: value,
		},
	}
}
