// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

const tstLicenseKID = "tst-license-key"

type tstLicenseSigner struct {
	kid  string
	alg  string
	priv any
}

func tstNewEd25519Signer(t *testing.T) (*tstLicenseSigner, *licenseVerificationKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.Nil(t, err, "%+v", err)

	return &tstLicenseSigner{
			kid:  tstLicenseKID,
			alg:  jwt.SigningMethodEdDSA.Alg(),
			priv: priv,
		}, &licenseVerificationKey{
			alg: jwt.SigningMethodEdDSA.Alg(),
			key: pub,
		}
}

func tstNewES256Signer(t *testing.T) (*tstLicenseSigner, *licenseVerificationKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, err, "%+v", err)

	return &tstLicenseSigner{
			kid:  tstLicenseKID,
			alg:  jwt.SigningMethodES256.Alg(),
			priv: priv,
		}, &licenseVerificationKey{
			alg: jwt.SigningMethodES256.Alg(),
			key: &priv.PublicKey,
		}
}

func tstSetLicenseKeys(t *testing.T, keys map[string]*licenseVerificationKey) {
	orig := getLicenseVerificationKeys
	getLicenseVerificationKeys = func() (map[string]*licenseVerificationKey, error) {
		return keys, nil
	}
	t.Cleanup(func() {
		getLicenseVerificationKeys = orig
	})
}

func tstPadCoord(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	ret := make([]byte, 32)
	copy(ret[32-len(b):], b)
	return ret
}

func tstSigningMethod(alg string) jwt.SigningMethod {
	switch alg {
	case jwt.SigningMethodEdDSA.Alg():
		return jwt.SigningMethodEdDSA
	case jwt.SigningMethodES256.Alg():
		return jwt.SigningMethodES256
	case jwt.SigningMethodPS256.Alg():
		return jwt.SigningMethodPS256
	default:
		return nil
	}
}

func tstLicenseSpec() *enterprisev1.License {
	now := time.Now().UTC().Truncate(time.Second)
	return &enterprisev1.License{
		Version: 1,
		Uid:     utilrand.GetRandomStringCanonical(16),
		Organization: &enterprisev1.License_Organization{
			Uid:         utilrand.GetRandomStringCanonical(16),
			DisplayName: "Acme Inc",
		},
		IssuedAt:  pbutils.Timestamp(now.Add(-time.Hour)),
		NotBefore: pbutils.Timestamp(now.Add(-time.Hour)),
		NotAfter:  pbutils.Timestamp(now.Add(365 * 24 * time.Hour)),
		Type:      enterprisev1.License_SUBSCRIPTION,
	}
}

func tstLicenseClaims(lic *enterprisev1.License) map[string]any {
	ret := map[string]any{
		"iss": licenseIssuer,
		"aud": []string{licenseAudience},
		"jti": lic.Uid,
		"iat": lic.IssuedAt.AsTime().Unix(),
	}
	if lic.Organization != nil {
		ret["sub"] = lic.Organization.Uid
	}
	return ret
}

func tstSignLicense(
	t *testing.T,
	signer *tstLicenseSigner,
	claims map[string]any,
	lic *enterprisev1.License,
) string {
	full := map[string]any{}
	for k, v := range claims {
		full[k] = v
	}

	if lic != nil {
		licJSON, err := protojson.Marshal(lic)
		assert.Nil(t, err, "%+v", err)
		full["license"] = json.RawMessage(licJSON)
	}

	token := jwt.NewWithClaims(tstSigningMethod(signer.alg), jwt.MapClaims(full))
	token.Header = map[string]any{
		"alg": signer.alg,
		"typ": licenseJWTType,
		"kid": signer.kid,
	}

	signed, err := token.SignedString(signer.priv)
	assert.Nil(t, err, "%+v", err)
	return signed
}

func tstValidLicense(t *testing.T, signer *tstLicenseSigner) (string, *enterprisev1.License) {
	lic := tstLicenseSpec()
	return tstSignLicense(t, signer, tstLicenseClaims(lic), lic), lic
}

func TestVerifyLicense(t *testing.T) {
	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	now := time.Now()

	{
		jwtStr, lic := tstValidLicense(t, signer)
		got, err := verifyLicense(jwtStr, "", now)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, lic.Uid, got.Uid)
		assert.Equal(t, lic.Organization.Uid, got.Organization.Uid)
		assert.Equal(t, enterprisev1.License_SUBSCRIPTION, got.Type)
	}

	{
		_, err := verifyLicense("", "", now)
		assert.NotNil(t, err)
	}

	{
		_, err := verifyLicense("   ", "", now)
		assert.NotNil(t, err)
	}

	{
		_, err := verifyLicense(strings.Repeat("a", maxLicenseLen+1), "", now)
		assert.NotNil(t, err)
	}

	{
		_, err := verifyLicense("not.a.jwt", "", now)
		assert.NotNil(t, err)
	}

	{
		jwtStr, _ := tstValidLicense(t, signer)
		parts := strings.Split(jwtStr, ".")
		assert.Len(t, parts, 3)
		tampered := parts[0] + "." + parts[1] + "." + parts[2][:len(parts[2])-2] + "AA"
		_, err := verifyLicense(tampered, "", now)
		assert.NotNil(t, err)
	}
}

func TestVerifyLicenseWrongKey(t *testing.T) {
	signer, _ := tstNewEd25519Signer(t)

	_, otherVKey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: otherVKey,
	})

	jwtStr, _ := tstValidLicense(t, signer)
	_, err := verifyLicense(jwtStr, "", time.Now())
	assert.NotNil(t, err)
}

func TestVerifyLicenseUnknownKID(t *testing.T) {
	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		"a-different-kid": vkey,
	})

	jwtStr, _ := tstValidLicense(t, signer)
	_, err := verifyLicense(jwtStr, "", time.Now())
	assert.NotNil(t, err)
}

func TestVerifyLicenseAlgKeyMismatch(t *testing.T) {
	ecSigner, _ := tstNewES256Signer(t)
	_, edVKey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: edVKey,
	})

	jwtStr, _ := tstValidLicense(t, ecSigner)
	_, err := verifyLicense(jwtStr, "", time.Now())
	assert.NotNil(t, err)
}

func TestVerifyLicenseES256(t *testing.T) {
	signer, vkey := tstNewES256Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	jwtStr, lic := tstValidLicense(t, signer)
	got, err := verifyLicense(jwtStr, "", time.Now())
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, lic.Uid, got.Uid)
}

func TestVerifyLicenseHeader(t *testing.T) {
	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	now := time.Now()

	{
		lic := tstLicenseSpec()
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(tstFullClaims(t, lic)))
		token.Header = map[string]any{
			"alg": signer.alg,
			"typ": "JWT",
			"kid": signer.kid,
		}
		signed, err := token.SignedString(signer.priv)
		assert.Nil(t, err, "%+v", err)

		_, err = verifyLicense(signed, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(tstFullClaims(t, lic)))
		token.Header = map[string]any{
			"alg":   signer.alg,
			"typ":   licenseJWTType,
			"kid":   signer.kid,
			"extra": "field",
		}
		signed, err := token.SignedString(signer.priv)
		assert.Nil(t, err, "%+v", err)

		_, err = verifyLicense(signed, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(tstFullClaims(t, lic)))
		token.Header = map[string]any{
			"alg": signer.alg,
			"typ": licenseJWTType,
			"kid": "",
		}
		signed, err := token.SignedString(signer.priv)
		assert.Nil(t, err, "%+v", err)

		_, err = verifyLicense(signed, "", now)
		assert.NotNil(t, err)
	}
}

func tstFullClaims(t *testing.T, lic *enterprisev1.License) map[string]any {
	full := tstLicenseClaims(lic)
	licJSON, err := protojson.Marshal(lic)
	assert.Nil(t, err, "%+v", err)
	full["license"] = json.RawMessage(licJSON)
	return full
}

func TestVerifyLicenseClaims(t *testing.T) {
	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	now := time.Now()

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["iss"] = "https://evil.example.com"
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["aud"] = []string{"wrong-audience"}
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["aud"] = []string{licenseAudience, "second"}
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["sub"] = utilrand.GetRandomStringCanonical(16)
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["jti"] = utilrand.GetRandomStringCanonical(16)
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["iat"] = lic.IssuedAt.AsTime().Add(time.Hour).Unix()
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["nbf"] = now.Add(-time.Hour).Unix()
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		claims["exp"] = now.Add(time.Hour).Unix()
		jwtStr := tstSignLicense(t, signer, claims, lic)
		_, err := verifyLicense(jwtStr, "", now)
		assert.NotNil(t, err)
	}
}

func TestVerifyLicenseClaimMissing(t *testing.T) {
	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	now := time.Now()

	{
		claims := map[string]any{
			"iss": licenseIssuer,
			"aud": []string{licenseAudience},
			"jti": utilrand.GetRandomStringCanonical(16),
			"iat": now.Add(-time.Hour).Unix(),
			"sub": utilrand.GetRandomStringCanonical(16),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(claims))
		token.Header = map[string]any{
			"alg": signer.alg,
			"typ": licenseJWTType,
			"kid": signer.kid,
		}
		signed, err := token.SignedString(signer.priv)
		assert.Nil(t, err, "%+v", err)

		_, err = verifyLicense(signed, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		claims := tstLicenseClaims(lic)
		full := map[string]any{}
		for k, v := range claims {
			full[k] = v
		}
		full["license"] = json.RawMessage("null")
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(full))
		token.Header = map[string]any{
			"alg": signer.alg,
			"typ": licenseJWTType,
			"kid": signer.kid,
		}
		signed, err := token.SignedString(signer.priv)
		assert.Nil(t, err, "%+v", err)

		_, err = verifyLicense(signed, "", now)
		assert.NotNil(t, err)
	}
}

func TestValidateLicense(t *testing.T) {
	now := time.Now()

	{
		err := validateLicense(tstLicenseSpec(), "", now)
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := validateLicense(nil, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Version = 0
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Version = maxLicenseVersion + 1
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Uid = ""
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Organization = nil
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Organization.Uid = ""
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Type = enterprisev1.License_TYPE_UNKNOWN
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Type = enterprisev1.License_Type(1000)
		err := validateLicense(lic, "", now)
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Type = enterprisev1.License_PERPETUAL
		lic.NotAfter = nil
		err := validateLicense(lic, "", now)
		assert.Nil(t, err, "%+v", err)
	}
}

const (
	tstIssuerFixtureKID       = "c0ffee00-1111-4222-8333-444455556666"
	tstIssuerFixtureX         = "n9zD4WShFogoZGRYIiqdODl6FHzRvgArVqqMEAWCkus"
	tstIssuerFixtureY         = "o646v23GsPLgYh9LhDkewh-2GaW3b3plnAQbPmJDJIE"
	tstIssuerFixtureFinite    = "eyJhbGciOiJFUzI1NiIsImtpZCI6ImMwZmZlZTAwLTExMTEtNDIyMi04MzMzLTQ0NDQ1NTU1NjY2NiIsInR5cCI6Im9jdGVsaXVtLWxpY2Vuc2Urand0In0.eyJpc3MiOiJodHRwczovL2xpY2Vuc2Uub2N0ZWxpdW0uY29tIiwiYXVkIjpbIm9jdGVsaXVtLWVudGVycHJpc2UiXSwic3ViIjoiMWEyYjNjNGQtNWU2Zi00YTdiLThjOWQtMGUxZjJhM2I0YzVkIiwianRpIjoiN2QzZjFlMmEtNWI2Yy00ZDdlLThmOTAtYTFiMmMzZDRlNWY2IiwiaWF0IjoxNzkxMDAwMDAwLCJsaWNlbnNlIjp7InZlcnNpb24iOjEsInVpZCI6IjdkM2YxZTJhLTViNmMtNGQ3ZS04ZjkwLWExYjJjM2Q0ZTVmNiIsIm9yZ2FuaXphdGlvbiI6eyJ1aWQiOiIxYTJiM2M0ZC01ZTZmLTRhN2ItOGM5ZC0wZTFmMmEzYjRjNWQiLCJkaXNwbGF5TmFtZSI6IkFjbWUgSW5jLiDigJQgw5xuw69jb2RlIn0sImlzc3VlZEF0IjoiMjAyNi0xMC0wM1QwNDowMDowMFoiLCJub3RCZWZvcmUiOiIyMDI2LTEwLTAzVDA0OjAwOjAwWiIsIm5vdEFmdGVyIjoiMjAyNy0xMS0wMlQwNDowMDowMFoiLCJ0eXBlIjoiU1VCU0NSSVBUSU9OIiwiYWxsb3dlZENsdXN0ZXJEb21haW5zIjpbImNsdXN0ZXIuYWNtZS5leGFtcGxlIiwieG4tLWJjaGVyLWt2YS5leGFtcGxlIl0sInBsYW4iOnsidHlwZSI6IkVOVEVSUFJJU0UiLCJ0aWVyIjoiVDIiLCJzdXBwb3J0Ijp7InR5cGUiOiJQUkVNSVVNIn0sImxpbWl0cyI6eyJtYXhVc2VycyI6MTAwfSwiZmVhdHVyZXMiOlsic2llbS1leHBvcnQiXX19fQ.JcjSrH2ZRb8VAftqgtUkzJMo8dN6tmfnPwCaSA5jdQ-G4z66Odw8uUzVSupNMnNPYbU_Y4DiMH2-g0yay_0PPg"
	tstIssuerFixturePerpetual = "eyJhbGciOiJFUzI1NiIsImtpZCI6ImMwZmZlZTAwLTExMTEtNDIyMi04MzMzLTQ0NDQ1NTU1NjY2NiIsInR5cCI6Im9jdGVsaXVtLWxpY2Vuc2Urand0In0.eyJpc3MiOiJodHRwczovL2xpY2Vuc2Uub2N0ZWxpdW0uY29tIiwiYXVkIjpbIm9jdGVsaXVtLWVudGVycHJpc2UiXSwic3ViIjoiMWEyYjNjNGQtNWU2Zi00YTdiLThjOWQtMGUxZjJhM2I0YzVkIiwianRpIjoiOGU0ZjJhM2ItNmM3ZC00ZThmLTlhMDEtYjJjM2Q0ZTVmNmE3IiwiaWF0IjoxNzkxMDAwMDAwLCJsaWNlbnNlIjp7InZlcnNpb24iOjEsInVpZCI6IjhlNGYyYTNiLTZjN2QtNGU4Zi05YTAxLWIyYzNkNGU1ZjZhNyIsIm9yZ2FuaXphdGlvbiI6eyJ1aWQiOiIxYTJiM2M0ZC01ZTZmLTRhN2ItOGM5ZC0wZTFmMmEzYjRjNWQiLCJkaXNwbGF5TmFtZSI6IkFjbWUgSW5jLiJ9LCJpc3N1ZWRBdCI6IjIwMjYtMTAtMDNUMDQ6MDA6MDBaIiwibm90QmVmb3JlIjoiMjAyNi0xMC0wM1QwNDowMDowMFoiLCJ0eXBlIjoiUEVSUEVUVUFMIiwicGxhbiI6eyJ0eXBlIjoiUFJFTUlVTSIsInRpZXIiOiJUMSIsInN1cHBvcnQiOnsidHlwZSI6Ik5PTkUifSwibGltaXRzIjp7Im1heFVzZXJzIjo1MH19fX0.fJL8QAWT4HPikAcXb2G8BhjZvpPrEITR5ADoHvooLjeeHahqh3_ZsAFeXITcof5VeIfmawBGbGy73bt4i3wG8Q"
)

func TestVerifyLicenseIssuerFixture(t *testing.T) {
	vkey, err := parseLicenseVerificationKey(&licenseJWK{
		KeyType: "EC",
		KeyID:   tstIssuerFixtureKID,
		Use:     "sig",
		KeyOps:  []string{"verify"},
		Alg:     "ES256",
		Curve:   "P-256",
		X:       tstIssuerFixtureX,
		Y:       tstIssuerFixtureY,
	})
	assert.Nil(t, err, "%+v", err)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstIssuerFixtureKID: vkey,
	})

	now := time.Unix(1791000000, 0).Add(time.Hour)

	{
		lic, err := verifyLicense(tstIssuerFixtureFinite, "Cluster.Acme.Example", now)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "7d3f1e2a-5b6c-4d7e-8f90-a1b2c3d4e5f6", lic.Uid)
		assert.Equal(t, "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d", lic.Organization.Uid)
		assert.Equal(t, "Acme Inc. — Ünïcode", lic.Organization.DisplayName)
		assert.Equal(t, enterprisev1.License_SUBSCRIPTION, lic.Type)
		assert.Equal(t, []string{"cluster.acme.example", "xn--bcher-kva.example"}, lic.AllowedClusterDomains)
		assert.Equal(t, int64(1791000000), lic.IssuedAt.AsTime().Unix())
		assert.Equal(t, int64(1791000000+395*24*3600), lic.NotAfter.AsTime().Unix())
		assert.Equal(t, enterprisev1.License_Plan_ENTERPRISE, lic.Plan.Type)
		assert.Equal(t, "T2", lic.Plan.Tier)
		assert.Equal(t, enterprisev1.License_Plan_Support_PREMIUM, lic.Plan.Support.Type)
		assert.Equal(t, uint32(100), lic.Plan.Limits.MaxUsers)
		assert.Equal(t, []string{"siem-export"}, lic.Plan.Features)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE, getLicenseState(lic, now))
	}

	{
		_, err := verifyLicense(tstIssuerFixtureFinite, "other.example.com", now)
		assert.NotNil(t, err)
	}

	{
		lic, err := verifyLicense(tstIssuerFixturePerpetual, "any.example.com", now)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.License_PERPETUAL, lic.Type)
		assert.Nil(t, lic.NotAfter)
		assert.Equal(t, enterprisev1.License_Plan_PREMIUM, lic.Plan.Type)
		assert.Equal(t, enterprisev1.License_Plan_Support_NONE, lic.Plan.Support.Type)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE, getLicenseState(lic, now.Add(100*365*24*time.Hour)))
	}

	{
		parts := strings.Split(tstIssuerFixtureFinite, ".")
		assert.Len(t, parts, 3)
		_, err := verifyLicense(parts[0]+"."+parts[1]+"."+tstIssuerFixturePerpetual[strings.LastIndex(tstIssuerFixturePerpetual, ".")+1:], "cluster.acme.example", now)
		assert.NotNil(t, err)
	}
}

func TestValidateLicensePlan(t *testing.T) {
	{
		assert.Nil(t, validateLicensePlan(nil))
	}

	{
		err := validateLicensePlan(&enterprisev1.License_Plan{
			Type:     enterprisev1.License_Plan_ENTERPRISE,
			Tier:     "T1",
			Support:  &enterprisev1.License_Plan_Support{Type: enterprisev1.License_Plan_Support_STANDARD},
			Limits:   &enterprisev1.License_Plan_Limits{MaxUsers: 50},
			Features: []string{"siem-export"},
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := validateLicensePlan(&enterprisev1.License_Plan{
			Tier: strings.Repeat("a", maxLicensePlanTierLen+1),
		})
		assert.NotNil(t, err)
	}

	{
		err := validateLicensePlan(&enterprisev1.License_Plan{
			Tier: "T 1",
		})
		assert.NotNil(t, err)
	}

	{
		features := make([]string, maxLicensePlanFeatures+1)
		for i := range features {
			features[i] = "feature"
		}
		err := validateLicensePlan(&enterprisev1.License_Plan{
			Features: features,
		})
		assert.NotNil(t, err)
	}

	{
		err := validateLicensePlan(&enterprisev1.License_Plan{
			Features: []string{""},
		})
		assert.NotNil(t, err)
	}

	{
		lic := tstLicenseSpec()
		lic.Plan = &enterprisev1.License_Plan{
			Tier: "T1\n",
		}
		err := validateLicense(lic, "", time.Now())
		assert.NotNil(t, err)
	}
}

func TestValidateLicenseIdentifier(t *testing.T) {
	{
		assert.Nil(t, validateLicenseIdentifier("abc123", "test"))
	}

	{
		assert.NotNil(t, validateLicenseIdentifier("", "test"))
	}

	{
		assert.NotNil(t, validateLicenseIdentifier(strings.Repeat("a", maxLicenseUIDLen+1), "test"))
	}

	{
		assert.NotNil(t, validateLicenseIdentifier("has space", "test"))
	}

	{
		assert.NotNil(t, validateLicenseIdentifier("has\ttab", "test"))
	}

	{
		assert.NotNil(t, validateLicenseIdentifier("has\x00null", "test"))
	}
}

func TestValidateLicenseDisplayName(t *testing.T) {
	{
		assert.Nil(t, validateLicenseDisplayName(""))
	}

	{
		assert.Nil(t, validateLicenseDisplayName("Acme Inc, Ltd."))
	}

	{
		assert.NotNil(t, validateLicenseDisplayName(strings.Repeat("a", maxLicenseDisplayNameLen+1)))
	}

	{
		assert.NotNil(t, validateLicenseDisplayName("line\nbreak"))
	}

	{
		assert.NotNil(t, validateLicenseDisplayName("carriage\rreturn"))
	}

	{
		assert.NotNil(t, validateLicenseDisplayName("null\x00byte"))
	}
}

func TestValidateLicenseTimestamps(t *testing.T) {
	now := time.Now()

	{
		assert.Nil(t, validateLicenseTimestamps(tstLicenseSpec(), now))
	}

	{
		lic := tstLicenseSpec()
		lic.IssuedAt = nil
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.IssuedAt = pbutils.Timestamp(now.Add(time.Hour))
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.IssuedAt = pbutils.Timestamp(now.Add(-time.Hour).Add(500 * time.Millisecond))
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotBefore = nil
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotAfter = nil
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotBefore = pbutils.Timestamp(now.Add(24 * time.Hour))
		lic.NotAfter = pbutils.Timestamp(now.Add(-24 * time.Hour))
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.Type = enterprisev1.License_PERPETUAL
		lic.NotAfter = pbutils.Timestamp(now.Add(24 * time.Hour))
		assert.NotNil(t, validateLicenseTimestamps(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.Type = enterprisev1.License_PERPETUAL
		lic.NotAfter = nil
		assert.Nil(t, validateLicenseTimestamps(lic, now))
	}
}

func TestValidateLicenseClusterDomain(t *testing.T) {
	{
		lic := tstLicenseSpec()
		assert.Nil(t, validateLicenseClusterDomain(lic, "octelium.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"octelium.example.com"}
		assert.Nil(t, validateLicenseClusterDomain(lic, "octelium.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"octelium.example.com"}
		assert.Nil(t, validateLicenseClusterDomain(lic, "OCTELIUM.EXAMPLE.COM"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"octelium.example.com."}
		assert.Nil(t, validateLicenseClusterDomain(lic, "octelium.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"a.example.com", "b.example.com"}
		assert.Nil(t, validateLicenseClusterDomain(lic, "b.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"a.example.com"}
		assert.NotNil(t, validateLicenseClusterDomain(lic, "b.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"a.example.com"}
		assert.NotNil(t, validateLicenseClusterDomain(lic, ""))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"a.example.com", "a.example.com"}
		assert.NotNil(t, validateLicenseClusterDomain(lic, "a.example.com"))
	}

	{
		lic := tstLicenseSpec()
		lic.AllowedClusterDomains = []string{"not a domain"}
		assert.NotNil(t, validateLicenseClusterDomain(lic, "a.example.com"))
	}

	{
		lic := tstLicenseSpec()
		domains := make([]string, maxLicenseDomains+1)
		for i := range domains {
			domains[i] = utilrand.GetRandomStringCanonical(8) + ".example.com"
		}
		lic.AllowedClusterDomains = domains
		assert.NotNil(t, validateLicenseClusterDomain(lic, "a.example.com"))
	}
}

func TestNormalizeLicenseDomain(t *testing.T) {
	{
		got, err := normalizeLicenseDomain("Octelium.Example.COM")
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "octelium.example.com", got)
	}

	{
		got, err := normalizeLicenseDomain("  example.com.  ")
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "example.com", got)
	}

	{
		_, err := normalizeLicenseDomain("")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain("localhost")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain(strings.Repeat("a", maxLicenseDomainLen+1) + ".com")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain("-bad.example.com")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain("bad-.example.com")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain("under_score.example.com")
		assert.NotNil(t, err)
	}

	{
		_, err := normalizeLicenseDomain("a." + strings.Repeat("b", 64) + ".com")
		assert.NotNil(t, err)
	}
}

func TestGetLicenseState(t *testing.T) {
	now := time.Now()

	{
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_NONE,
			getLicenseState(nil, now))
	}

	{
		lic := tstLicenseSpec()
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE,
			getLicenseState(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotBefore = pbutils.Timestamp(now.Add(24 * time.Hour))
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_NOT_YET_VALID,
			getLicenseState(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotBefore = pbutils.Timestamp(now.Add(time.Minute))
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE,
			getLicenseState(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotAfter = pbutils.Timestamp(now.Add(-24 * time.Hour))
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_EXPIRED,
			getLicenseState(lic, now))
	}

	{
		lic := tstLicenseSpec()
		lic.NotAfter = pbutils.Timestamp(now.Add(-time.Minute))
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE,
			getLicenseState(lic, now))
	}
}

func TestParseLicenseVerificationKey(t *testing.T) {
	{
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		assert.Nil(t, err, "%+v", err)
		key := &licenseJWK{
			KeyType: "OKP",
			KeyID:   "k1",
			Use:     "sig",
			Alg:     jwt.SigningMethodEdDSA.Alg(),
			Curve:   "Ed25519",
			X:       base64.RawURLEncoding.EncodeToString(pub),
		}
		parsed, err := parseLicenseVerificationKey(key)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, jwt.SigningMethodEdDSA.Alg(), parsed.alg)
	}

	{
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.Nil(t, err, "%+v", err)
		key := &licenseJWK{
			KeyType: "EC",
			KeyID:   "k2",
			Alg:     jwt.SigningMethodES256.Alg(),
			Curve:   "P-256",
			X:       base64.RawURLEncoding.EncodeToString(tstPadCoord(priv.X.Bytes())),
			Y:       base64.RawURLEncoding.EncodeToString(tstPadCoord(priv.Y.Bytes())),
		}
		parsed, err := parseLicenseVerificationKey(key)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, jwt.SigningMethodES256.Alg(), parsed.alg)
	}

	{
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		assert.Nil(t, err, "%+v", err)
		key := &licenseJWK{
			KeyType: "OKP",
			KeyID:   "k1",
			Alg:     jwt.SigningMethodEdDSA.Alg(),
			Curve:   "Ed25519",
			X:       base64.RawURLEncoding.EncodeToString(pub),
			D:       base64.RawURLEncoding.EncodeToString([]byte("private")),
		}
		_, err = parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		key := &licenseJWK{
			KeyType: "OKP",
			KeyID:   "",
			Alg:     jwt.SigningMethodEdDSA.Alg(),
			Curve:   "Ed25519",
		}
		_, err := parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		assert.Nil(t, err, "%+v", err)
		key := &licenseJWK{
			KeyType: "OKP",
			KeyID:   "k1",
			Use:     "enc",
			Alg:     jwt.SigningMethodEdDSA.Alg(),
			Curve:   "Ed25519",
			X:       base64.RawURLEncoding.EncodeToString(pub),
		}
		_, err = parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		key := &licenseJWK{
			KeyType: "OKP",
			KeyID:   "k1",
			Alg:     jwt.SigningMethodEdDSA.Alg(),
			Curve:   "Ed25519",
			X:       base64.RawURLEncoding.EncodeToString([]byte("tooshort")),
		}
		_, err := parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		key := &licenseJWK{
			KeyType: "EC",
			KeyID:   "k2",
			Alg:     jwt.SigningMethodES256.Alg(),
			Curve:   "P-384",
		}
		_, err := parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		key := &licenseJWK{
			KeyType: "unsupported",
			KeyID:   "k3",
		}
		_, err := parseLicenseVerificationKey(key)
		assert.NotNil(t, err)
	}

	{
		_, err := parseLicenseVerificationKey(nil)
		assert.NotNil(t, err)
	}
}

func TestGetLicenseVerificationKey(t *testing.T) {
	_, vkey := tstNewEd25519Signer(t)
	keys := map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	}

	{
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{})
		token.Header["kid"] = tstLicenseKID
		got, err := getLicenseVerificationKey(token, keys)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, got)
	}

	{
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{})
		got, err := getLicenseVerificationKey(token, keys)
		assert.NotNil(t, err)
		assert.Nil(t, got)
	}

	{
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{})
		token.Header["kid"] = "unknown"
		_, err := getLicenseVerificationKey(token, keys)
		assert.NotNil(t, err)
	}

	{
		token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{})
		token.Header["kid"] = tstLicenseKID
		_, err := getLicenseVerificationKey(token, keys)
		assert.NotNil(t, err)
	}
}

func TestDecodeLicenseJWKValue(t *testing.T) {
	{
		got, err := decodeLicenseJWKValue(base64.RawURLEncoding.EncodeToString([]byte("hello")))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []byte("hello"), got)
	}

	{
		_, err := decodeLicenseJWKValue("")
		assert.NotNil(t, err)
	}

	{
		_, err := decodeLicenseJWKValue("not valid base64!!!")
		assert.NotNil(t, err)
	}
}

func TestIsSameObjectReference(t *testing.T) {
	{
		assert.True(t, isSameObjectReference(nil, nil))
	}

	{
		assert.False(t, isSameObjectReference(nil, &metav1.ObjectReference{Uid: "a"}))
	}

	{
		assert.True(t, isSameObjectReference(
			&metav1.ObjectReference{Uid: "a"},
			&metav1.ObjectReference{Uid: "a"}))
	}

	{
		assert.False(t, isSameObjectReference(
			&metav1.ObjectReference{Uid: "a"},
			&metav1.ObjectReference{Uid: "b"}))
	}

	{
		assert.True(t, isSameObjectReference(
			&metav1.ObjectReference{Name: "a"},
			&metav1.ObjectReference{Name: "a"}))
	}

	{
		assert.False(t, isSameObjectReference(
			&metav1.ObjectReference{Name: "a"},
			&metav1.ObjectReference{Name: "b"}))
	}
}

func TestGetLicense(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	{
		_, err := srv.GetLicense(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		resp, err := srv.GetLicense(ctx, &enterprisev1.GetLicenseRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_NONE, resp.State)
	}

	{
		jwtStr, lic := tstValidLicense(t, signer)
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr,
		})
		assert.Nil(t, err, "%+v", err)

		resp, err := srv.GetLicense(ctx, &enterprisev1.GetLicenseRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE, resp.State)
		assert.NotNil(t, resp.License)
		assert.Equal(t, lic.Uid, resp.License.Uid)
		assert.NotNil(t, resp.SetAt)
	}
}

func TestSetLicense(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	{
		_, err := srv.SetLicense(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: "   ",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: strings.Repeat("a", maxLicenseLen+1),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: "not.a.valid.jwt",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		lic := tstLicenseSpec()
		lic.NotBefore = pbutils.Timestamp(time.Now().Add(-2 * time.Hour))
		lic.NotAfter = pbutils.Timestamp(time.Now().Add(-time.Hour))
		jwtStr := tstSignLicense(t, signer, tstLicenseClaims(lic), lic)

		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		jwtStr, lic := tstValidLicense(t, signer)
		resp, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt:    jwtStr,
			DryRun: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE, resp.State)
		assert.Equal(t, lic.Uid, resp.License.Uid)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		if cc.Status != nil && cc.Status.LicenseInfo != nil {
			assert.NotEqual(t,
				enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE,
				cc.Status.LicenseInfo.State)
		}
	}

	{
		jwtStr, lic := tstValidLicense(t, signer)
		resp, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE, resp.State)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, cc.Status.LicenseInfo)
		assert.NotNil(t, cc.Status.LicenseInfo.LicenseSecretRef)
		assert.Equal(t,
			enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE,
			cc.Status.LicenseInfo.State)

		firstSecretRef := cc.Status.LicenseInfo.LicenseSecretRef

		sec, err := srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: firstSecretRef.Uid,
		})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, sec.Metadata.IsSystem)
		assert.True(t, sec.Metadata.IsUserHidden)
		assert.True(t, sec.Metadata.IsSystemHidden)
		assert.Equal(t, "true", sec.Metadata.SystemLabels[licenseSecretLabel])

		jwtStr2, _ := tstValidLicense(t, signer)
		_, err = srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr2,
		})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotEqual(t, firstSecretRef.Uid, cc.Status.LicenseInfo.LicenseSecretRef.Uid)

		_, err = srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: firstSecretRef.Uid,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

		_ = lic
	}
}

func TestDeleteLicense(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	signer, vkey := tstNewEd25519Signer(t)
	tstSetLicenseKeys(t, map[string]*licenseVerificationKey{
		tstLicenseKID: vkey,
	})

	{
		_, err := srv.DeleteLicense(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.DeleteLicense(ctx, &enterprisev1.DeleteLicenseRequest{})
		assert.Nil(t, err, "%+v", err)
	}

	{
		jwtStr, _ := tstValidLicense(t, signer)
		_, err := srv.SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: jwtStr,
		})
		assert.Nil(t, err, "%+v", err)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		secretRef := cc.Status.LicenseInfo.LicenseSecretRef
		assert.NotNil(t, secretRef)

		_, err = srv.DeleteLicense(ctx, &enterprisev1.DeleteLicenseRequest{})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, cc.Status.LicenseInfo)

		_, err = srv.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Uid: secretRef.Uid,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

		resp, err := srv.GetLicense(ctx, &enterprisev1.GetLicenseRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_NONE, resp.State)
	}
}

func TestLicenseSecretLifecycle(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	{
		ref, err := srv.createLicenseSecret(ctx, "some-jwt-value")
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, ref)
		assert.NotEmpty(t, ref.Uid)

		got, err := srv.getLicenseJWT(ctx, ref)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "some-jwt-value", got)

		err = srv.deleteLicenseSecret(ctx, ref)
		assert.Nil(t, err, "%+v", err)

		err = srv.deleteLicenseSecret(ctx, ref)
		assert.Nil(t, err, "%+v", err)
	}

	{
		_, err := srv.getLicenseJWT(ctx, nil)
		assert.NotNil(t, err)
	}

	{
		_, err := srv.getLicenseJWT(ctx, &metav1.ObjectReference{})
		assert.NotNil(t, err)
	}

	{
		err := srv.deleteLicenseSecret(ctx, nil)
		assert.Nil(t, err, "%+v", err)
	}

	{
		sec := &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: "not-a-license-secret",
				},
			},
		}
		sec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, sec)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.getLicenseJWT(ctx, umetav1.GetObjectReference(sec))
		assert.NotNil(t, err)
	}
}

func TestValidateLicenseSecret(t *testing.T) {
	tstSecret := func() *enterprisev1.Secret {
		return &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           licenseSecretNamePrefix + "-" + utilrand.GetRandomStringCanonical(12),
				IsSystem:       true,
				IsUserHidden:   true,
				IsSystemHidden: true,
				SystemLabels: map[string]string{
					licenseSecretLabel: "true",
				},
			},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: "jwt",
				},
			},
		}
	}

	{
		assert.Nil(t, validateLicenseSecret(tstSecret()))
	}

	{
		assert.NotNil(t, validateLicenseSecret(nil))
	}

	{
		sec := tstSecret()
		sec.Metadata = nil
		assert.NotNil(t, validateLicenseSecret(sec))
	}

	{
		sec := tstSecret()
		sec.Metadata.IsSystem = false
		assert.NotNil(t, validateLicenseSecret(sec))
	}

	{
		sec := tstSecret()
		sec.Metadata.SystemLabels = nil
		assert.NotNil(t, validateLicenseSecret(sec))
	}

	{
		sec := tstSecret()
		sec.Metadata.Name = "wrong-prefix-" + utilrand.GetRandomStringCanonical(8)
		assert.NotNil(t, validateLicenseSecret(sec))
	}

	{
		sec := tstSecret()
		sec.Data = nil
		assert.NotNil(t, validateLicenseSecret(sec))
	}

	{
		sec := tstSecret()
		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_ValueBytes{
				ValueBytes: []byte("jwt"),
			},
		}
		assert.NotNil(t, validateLicenseSecret(sec))
	}
}
