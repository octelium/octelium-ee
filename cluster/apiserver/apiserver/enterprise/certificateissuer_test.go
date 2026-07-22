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
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestCertificateIssuer(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		item := tstCreateCertificateIssuer(ctx, t, srv, tstCertificateIssuerACMESpec(
			"https://acme-staging-v02.api.letsencrypt.org/directory",
			"admin@example.com",
		))

		ret, err := srv.GetCertificateIssuer(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, ret.Status.State)

		ret, err = srv.GetCertificateIssuer(ctx, &metav1.GetOptions{Name: item.Metadata.Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)

		itemList, err := srv.ListCertificateIssuer(ctx, &enterprisev1.ListCertificateIssuerOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, len(itemList.Items) >= 1)

		arg := tstCloneCertificateIssuer(item)
		arg.Spec = tstCertificateIssuerACMESpec(
			"https://acme-v02.api.letsencrypt.org/directory",
			"ops@example.com",
		)
		arg.Status = &enterprisev1.CertificateIssuer_Status{
			State: enterprisev1.CertificateIssuer_Status_NOT_READY,
		}

		updated, err := srv.UpdateCertificateIssuer(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, updated.Metadata.Uid)
		assert.Equal(t, "https://acme-v02.api.letsencrypt.org/directory", updated.Spec.GetAcme().GetServer())
		assert.Equal(t, "ops@example.com", updated.Spec.GetAcme().GetEmail())
		assert.Equal(t, enterprisev1.CertificateIssuer_Status_READY, updated.Status.State)
		assert.NotNil(t, updated.Status.GetAcme())
	}

	{
		_, err := srv.GetCertificateIssuer(ctx, &metav1.GetOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetCertificateIssuer(ctx, &metav1.GetOptions{Name: utilrand.GetRandomStringCanonical(8)})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.UpdateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpdateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: tstCertificateIssuerACMESpec(
				"https://acme-v02.api.letsencrypt.org/directory",
				"admin@example.com",
			),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		item := tstCreateCertificateIssuer(ctx, t, srv, tstCertificateIssuerACMESpec(
			"https://acme-v02.api.letsencrypt.org/directory",
			"admin@example.com",
		))
		arg := tstCloneCertificateIssuer(item)
		arg.Spec = &enterprisev1.CertificateIssuer_Spec{}
		_, err := srv.UpdateCertificateIssuer(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestValidateCertificateIssuer(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec(
				"https://acme-staging-v02.api.letsencrypt.org/directory",
				"admin@example.com",
			),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: &enterprisev1.CertificateIssuer_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: &enterprisev1.CertificateIssuer_Spec{
				Type: &enterprisev1.CertificateIssuer_Spec_Acme{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("", "admin@example.com"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("not-a-url", "admin@example.com"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("https://"+strings.Repeat("a", maxCertificateIssuerServerURLBytes)+".example.com/directory", "admin@example.com"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("https://acme-v02.api.letsencrypt.org/directory", ""),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("https://acme-v02.api.letsencrypt.org/directory", "invalid-email"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerACMESpec("https://acme-v02.api.letsencrypt.org/directory", strings.Repeat("a", maxCertificateIssuerEmailBytes)+"@example.com"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		acme := tstCertificateIssuerACME("https://acme-v02.api.letsencrypt.org/directory", "admin@example.com")
		acme.Solver = nil
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerSpecFromACME(acme),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		acme := tstCertificateIssuerACME("https://acme-v02.api.letsencrypt.org/directory", "admin@example.com")
		acme.Solver = &enterprisev1.CertificateIssuer_Spec_ACME_Solver{}
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerSpecFromACME(acme),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		acme := tstCertificateIssuerACME("https://acme-v02.api.letsencrypt.org/directory", "admin@example.com")
		acme.Solver = &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
			Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{},
		}
		err := srv.validateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
			Spec: tstCertificateIssuerSpecFromACME(acme),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func tstCreateCertificateIssuer(ctx context.Context, t *testing.T, srv *Server, spec *enterprisev1.CertificateIssuer_Spec) *enterprisev1.CertificateIssuer {
	item, err := srv.octeliumC.EnterpriseC().CreateCertificateIssuer(ctx, &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: spec,
		Status: &enterprisev1.CertificateIssuer_Status{
			State: enterprisev1.CertificateIssuer_Status_READY,
			Type: &enterprisev1.CertificateIssuer_Status_Acme{
				Acme: &enterprisev1.CertificateIssuer_Status_ACME{
					SecretRef: &metav1.ObjectReference{
						Name: utilrand.GetRandomStringCanonical(8),
					},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return item
}

func tstCertificateIssuerACMESpec(server string, email string) *enterprisev1.CertificateIssuer_Spec {
	return tstCertificateIssuerSpecFromACME(tstCertificateIssuerACME(server, email))
}

func tstCertificateIssuerSpecFromACME(acme *enterprisev1.CertificateIssuer_Spec_ACME) *enterprisev1.CertificateIssuer_Spec {
	return &enterprisev1.CertificateIssuer_Spec{
		Type: &enterprisev1.CertificateIssuer_Spec_Acme{
			Acme: acme,
		},
	}
}

func tstCertificateIssuerACME(server string, email string) *enterprisev1.CertificateIssuer_Spec_ACME {
	return &enterprisev1.CertificateIssuer_Spec_ACME{
		Server: server,
		Email:  email,
		Solver: &enterprisev1.CertificateIssuer_Spec_ACME_Solver{
			Type: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns{
				Dns: &enterprisev1.CertificateIssuer_Spec_ACME_Solver_DNS{},
			},
		},
	}
}

func tstCloneCertificateIssuer(arg *enterprisev1.CertificateIssuer) *enterprisev1.CertificateIssuer {
	return &enterprisev1.CertificateIssuer{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec: arg.Spec,
	}
}
