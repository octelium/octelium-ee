// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

//go:embed jwks.json
var licenseJWKSBytes []byte

const (
	licenseIssuer           = "https://license.octelium.com"
	licenseAudience         = "octelium-enterprise"
	licenseJWTType          = "octelium-license+jwt"
	licenseSecretNamePrefix = "sys-octelium-license"
	licenseSecretLabel      = "octelium-license"

	maxLicenseVersion        = 1
	maxLicenseLen            = 16 * 1024
	maxLicenseUIDLen         = 256
	maxLicenseDisplayNameLen = 512
	maxLicenseDomains        = 64
	maxLicenseDomainLen      = 253
	licenseClockSkew         = 5 * time.Minute
)

var (
	licenseJWTAlgorithms = []string{
		jwt.SigningMethodEdDSA.Alg(),
		jwt.SigningMethodES256.Alg(),
		jwt.SigningMethodPS256.Alg(),
	}

	getLicenseVerificationKeys = sync.OnceValues(parseLicenseVerificationKeys)
)

type licenseClaims struct {
	jwt.RegisteredClaims
	License json.RawMessage `json:"license"`
}

type licenseJWKSet struct {
	Keys []*licenseJWK `json:"keys"`
}

type licenseJWK struct {
	KeyType string   `json:"kty"`
	KeyID   string   `json:"kid"`
	Use     string   `json:"use"`
	KeyOps  []string `json:"key_ops"`
	Alg     string   `json:"alg"`
	Curve   string   `json:"crv"`
	X       string   `json:"x"`
	Y       string   `json:"y"`
	N       string   `json:"n"`
	E       string   `json:"e"`
	D       string   `json:"d"`
	P       string   `json:"p"`
	Q       string   `json:"q"`
	DP      string   `json:"dp"`
	DQ      string   `json:"dq"`
	QI      string   `json:"qi"`
}

type licenseVerificationKey struct {
	alg string
	key any
}

func (s *Server) GetLicense(ctx context.Context, req *enterprisev1.GetLicenseRequest) (
	*enterprisev1.GetLicenseResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cc.Status == nil || cc.Status.LicenseInfo == nil ||
		cc.Status.LicenseInfo.State == enterprisev1.ClusterConfig_Status_LicenseInfo_NONE {
		return &enterprisev1.GetLicenseResponse{
			State: enterprisev1.ClusterConfig_Status_LicenseInfo_NONE,
		}, nil
	}

	info := cc.Status.LicenseInfo
	if info.LicenseSecretRef == nil {
		return &enterprisev1.GetLicenseResponse{
			State: enterprisev1.ClusterConfig_Status_LicenseInfo_INVALID,
			SetAt: info.SetAt,
		}, nil
	}

	jwtStr, err := s.getLicenseJWT(ctx, info.LicenseSecretRef)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not find the License Secret",
				zap.String("secretUID", info.LicenseSecretRef.Uid))
			return &enterprisev1.GetLicenseResponse{
				State: enterprisev1.ClusterConfig_Status_LicenseInfo_INVALID,
				SetAt: info.SetAt,
			}, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	clusterDomain, err := s.getLicenseClusterDomain(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lic, err := verifyLicense(jwtStr, clusterDomain, now)
	if err != nil {
		zap.L().Warn("Could not verify the currently set License", zap.Error(err))
		return &enterprisev1.GetLicenseResponse{
			State: enterprisev1.ClusterConfig_Status_LicenseInfo_INVALID,
			SetAt: info.SetAt,
		}, nil
	}

	return &enterprisev1.GetLicenseResponse{
		License: lic,
		State:   getLicenseState(lic, now),
		SetAt:   info.SetAt,
	}, nil
}

func (s *Server) SetLicense(ctx context.Context, req *enterprisev1.SetLicenseRequest) (
	*enterprisev1.SetLicenseResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	jwtStr := strings.TrimSpace(req.Jwt)
	if jwtStr == "" {
		return nil, grpcutils.InvalidArg("License is required")
	}
	if len(jwtStr) > maxLicenseLen {
		return nil, grpcutils.InvalidArg("License is too large")
	}

	clusterDomain, err := s.getLicenseClusterDomain(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lic, err := verifyLicense(jwtStr, clusterDomain, now)
	if err != nil {
		zap.L().Debug("Could not verify the provided License", zap.Error(err))
		return nil, grpcutils.InvalidArg("Could not verify the provided License")
	}

	state := getLicenseState(lic, now)
	if state == enterprisev1.ClusterConfig_Status_LicenseInfo_EXPIRED {
		return nil, grpcutils.InvalidArg("The provided License expired at %s",
			lic.NotAfter.AsTime().Format(time.RFC3339))
	}

	if req.DryRun {
		return &enterprisev1.SetLicenseResponse{
			License: lic,
			State:   state,
		}, nil
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	var oldSecretRef *metav1.ObjectReference
	if cc.Status.LicenseInfo != nil {
		oldSecretRef = cc.Status.LicenseInfo.LicenseSecretRef
	}

	secretRef, err := s.createLicenseSecret(ctx, jwtStr)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	checkedAt := pbutils.Now()
	cc.Status.LicenseInfo = &enterprisev1.ClusterConfig_Status_LicenseInfo{
		LicenseSecretRef: secretRef,
		State:            state,
		SetAt:            checkedAt,
		StateCheckedAt:   checkedAt,
	}

	if _, err := s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if cleanupErr := s.deleteLicenseSecret(cleanupCtx, secretRef); cleanupErr != nil {
			zap.L().Warn("Could not delete the unused License Secret",
				zap.String("secretUID", secretRef.Uid),
				zap.Error(cleanupErr))
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	if oldSecretRef != nil && oldSecretRef.Uid != "" && oldSecretRef.Uid != secretRef.Uid {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := s.deleteLicenseSecret(cleanupCtx, oldSecretRef); err != nil {
			zap.L().Warn("Could not delete the previous License Secret",
				zap.String("secretUID", oldSecretRef.Uid),
				zap.Error(err))
		}
	}

	zap.L().Debug("License successfully set",
		zap.String("uid", lic.Uid),
		zap.String("state", state.String()),
		zap.String("secretUID", secretRef.Uid))

	return &enterprisev1.SetLicenseResponse{
		License: lic,
		State:   state,
	}, nil
}

func (s *Server) DeleteLicense(ctx context.Context, req *enterprisev1.DeleteLicenseRequest) (
	*enterprisev1.DeleteLicenseResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cc.Status == nil || cc.Status.LicenseInfo == nil {
		return &enterprisev1.DeleteLicenseResponse{}, nil
	}

	secretRef := cc.Status.LicenseInfo.LicenseSecretRef
	if secretRef == nil {
		cc.Status.LicenseInfo = nil
		if _, err := s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}
		return &enterprisev1.DeleteLicenseResponse{}, nil
	}

	cc.Status.LicenseInfo.State = enterprisev1.ClusterConfig_Status_LicenseInfo_NONE
	cc.Status.LicenseInfo.StateCheckedAt = pbutils.Now()

	cc, err = s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.deleteLicenseSecret(ctx, secretRef); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if cc.Status != nil && cc.Status.LicenseInfo != nil &&
		cc.Status.LicenseInfo.State == enterprisev1.ClusterConfig_Status_LicenseInfo_NONE &&
		isSameObjectReference(cc.Status.LicenseInfo.LicenseSecretRef, secretRef) {
		cc.Status.LicenseInfo = nil
		if _, err := s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	zap.L().Debug("License successfully deleted",
		zap.String("secretUID", secretRef.Uid))

	return &enterprisev1.DeleteLicenseResponse{}, nil
}

func verifyLicense(
	jwtStr string,
	clusterDomain string,
	now time.Time,
) (*enterprisev1.License, error) {
	jwtStr = strings.TrimSpace(jwtStr)
	if jwtStr == "" {
		return nil, errors.Errorf("Empty License")
	}
	if len(jwtStr) > maxLicenseLen {
		return nil, errors.Errorf("License is too large")
	}

	keys, err := getLicenseVerificationKeys()
	if err != nil {
		return nil, errors.Wrap(err, "Could not parse the embedded License key set")
	}

	claims := &licenseClaims{}
	token, err := jwt.ParseWithClaims(jwtStr, claims,
		func(token *jwt.Token) (any, error) {
			return getLicenseVerificationKey(token, keys)
		},
		jwt.WithValidMethods(licenseJWTAlgorithms),
		jwt.WithStrictDecoding(),
		jwt.WithoutClaimsValidation(),
	)
	if err != nil {
		return nil, errors.Errorf("Could not verify the License signature")
	}
	if token == nil || !token.Valid {
		return nil, errors.Errorf("Invalid License signature")
	}

	if err := validateLicenseJWTHeader(token); err != nil {
		return nil, err
	}

	licenseJSON := bytes.TrimSpace(claims.License)
	if len(licenseJSON) == 0 || bytes.Equal(licenseJSON, []byte("null")) {
		return nil, errors.Errorf("The License claim is not set")
	}

	var version struct {
		Version uint32 `json:"version"`
	}
	if err := json.Unmarshal(licenseJSON, &version); err != nil {
		return nil, errors.Errorf("Could not parse the License version")
	}
	if version.Version == 0 || version.Version > maxLicenseVersion {
		return nil, errors.Errorf("Unsupported License version: %d", version.Version)
	}

	lic := &enterprisev1.License{}
	if err := pbutils.UnmarshalJSON(licenseJSON, lic); err != nil {
		return nil, errors.Errorf("Could not parse the License claim")
	}

	if err := validateLicense(lic, clusterDomain, now); err != nil {
		return nil, err
	}
	if err := validateLicenseJWTClaims(claims, lic); err != nil {
		return nil, err
	}

	return lic, nil
}

func validateLicenseJWTHeader(token *jwt.Token) error {
	if token == nil {
		return errors.Errorf("Nil License token")
	}

	if len(token.Header) != 3 {
		return errors.Errorf("Invalid License header fields")
	}

	typ, ok := token.Header["typ"].(string)
	if !ok || typ != licenseJWTType {
		return errors.Errorf("Invalid License typ header")
	}

	kid, ok := token.Header["kid"].(string)
	if !ok || strings.TrimSpace(kid) == "" {
		return errors.Errorf("Invalid License kid header")
	}

	alg, ok := token.Header["alg"].(string)
	if !ok || alg == "" || token.Method == nil || token.Method.Alg() != alg {
		return errors.Errorf("Invalid License alg header")
	}

	return nil
}

func validateLicenseJWTClaims(
	claims *licenseClaims,
	lic *enterprisev1.License,
) error {
	if claims == nil {
		return errors.Errorf("Nil License claims")
	}

	if claims.Issuer != licenseIssuer {
		return errors.Errorf("Invalid License issuer")
	}

	if len(claims.Audience) != 1 || claims.Audience[0] != licenseAudience {
		return errors.Errorf("Invalid License audience")
	}

	if lic.Organization == nil || claims.Subject != lic.Organization.Uid {
		return errors.Errorf("Invalid License subject")
	}

	if claims.ID == "" || claims.ID != lic.Uid {
		return errors.Errorf("Invalid License ID")
	}

	if claims.IssuedAt == nil || lic.IssuedAt == nil ||
		!claims.IssuedAt.Time.Equal(lic.IssuedAt.AsTime()) {
		return errors.Errorf("Invalid License issued-at claim")
	}

	if claims.NotBefore != nil {
		return errors.Errorf("The License JWT must not set nbf")
	}
	if claims.ExpiresAt != nil {
		return errors.Errorf("The License JWT must not set exp")
	}

	return nil
}

func validateLicense(
	lic *enterprisev1.License,
	clusterDomain string,
	now time.Time,
) error {
	if lic == nil {
		return errors.Errorf("Nil License")
	}

	if lic.Version == 0 || lic.Version > maxLicenseVersion {
		return errors.Errorf("Unsupported License version: %d", lic.Version)
	}

	if err := validateLicenseIdentifier(lic.Uid, "License uid"); err != nil {
		return err
	}

	if lic.Organization == nil {
		return errors.Errorf("The License organization is not set")
	}
	if err := validateLicenseIdentifier(lic.Organization.Uid, "License organization uid"); err != nil {
		return err
	}
	if err := validateLicenseDisplayName(lic.Organization.DisplayName); err != nil {
		return err
	}

	switch lic.Type {
	case enterprisev1.License_TRIAL,
		enterprisev1.License_SUBSCRIPTION,
		enterprisev1.License_PERPETUAL,
		enterprisev1.License_INTERNAL:
	case enterprisev1.License_TYPE_UNKNOWN:
		return errors.Errorf("The License type is not set")
	default:
		return errors.Errorf("Invalid License type")
	}

	if err := validateLicenseTimestamps(lic, now); err != nil {
		return err
	}
	if err := validateLicenseClusterDomain(lic, clusterDomain); err != nil {
		return err
	}

	return nil
}

func validateLicenseIdentifier(val string, field string) error {
	if val == "" {
		return errors.Errorf("The %s is not set", field)
	}
	if len(val) > maxLicenseUIDLen || !utf8.ValidString(val) {
		return errors.Errorf("The %s is invalid", field)
	}

	for _, r := range val {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return errors.Errorf("The %s is invalid", field)
		}
	}

	return nil
}

func validateLicenseDisplayName(val string) error {
	if len(val) > maxLicenseDisplayNameLen || !utf8.ValidString(val) {
		return errors.Errorf("The License organization display name is invalid")
	}

	for _, r := range val {
		if r == '\x00' || r == '\r' || r == '\n' {
			return errors.Errorf("The License organization display name is invalid")
		}
	}

	return nil
}

func validateLicenseTimestamps(lic *enterprisev1.License, now time.Time) error {
	if lic.IssuedAt == nil || !lic.IssuedAt.IsValid() || lic.IssuedAt.Nanos != 0 {
		return errors.Errorf("The License issuedAt is invalid")
	}
	if lic.IssuedAt.AsTime().After(now.Add(licenseClockSkew)) {
		return errors.Errorf("The License issuedAt is in the future")
	}

	if lic.NotBefore == nil || !lic.NotBefore.IsValid() || lic.NotBefore.Nanos != 0 {
		return errors.Errorf("The License notBefore is invalid")
	}

	if lic.Type == enterprisev1.License_PERPETUAL {
		if lic.NotAfter != nil {
			return errors.Errorf("A perpetual License must not set notAfter")
		}
		return nil
	}

	if lic.NotAfter == nil || !lic.NotAfter.IsValid() || lic.NotAfter.Nanos != 0 {
		return errors.Errorf("The License notAfter is invalid")
	}
	if !lic.NotAfter.AsTime().After(lic.NotBefore.AsTime()) {
		return errors.Errorf("The License notAfter is not after notBefore")
	}

	return nil
}

func validateLicenseClusterDomain(
	lic *enterprisev1.License,
	clusterDomain string,
) error {
	if len(lic.AllowedClusterDomains) == 0 {
		return nil
	}
	if len(lic.AllowedClusterDomains) > maxLicenseDomains {
		return errors.Errorf("The License has too many allowed Cluster domains")
	}

	clusterDomain, err := normalizeLicenseDomain(clusterDomain)
	if err != nil {
		return errors.Errorf("The Cluster domain is invalid")
	}

	seen := make(map[string]struct{}, len(lic.AllowedClusterDomains))
	isAllowed := false
	for _, domain := range lic.AllowedClusterDomains {
		domain, err = normalizeLicenseDomain(domain)
		if err != nil {
			return errors.Errorf("The License contains an invalid Cluster domain")
		}
		if _, ok := seen[domain]; ok {
			return errors.Errorf("The License contains duplicate Cluster domains")
		}
		seen[domain] = struct{}{}

		if domain == clusterDomain {
			isAllowed = true
		}
	}

	if !isAllowed {
		return errors.Errorf("The License is not valid for the Cluster domain: %s", clusterDomain)
	}

	return nil
}

func normalizeLicenseDomain(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" || len(domain) > maxLicenseDomainLen {
		return "", errors.Errorf("Invalid domain")
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return "", errors.Errorf("Invalid domain")
	}

	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 ||
			label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.Errorf("Invalid domain")
		}

		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c >= 'a' && c <= 'z') ||
				(c >= '0' && c <= '9') || c == '-' {
				continue
			}
			return "", errors.Errorf("Invalid domain")
		}
	}

	return domain, nil
}

func getLicenseState(
	lic *enterprisev1.License,
	now time.Time,
) enterprisev1.ClusterConfig_Status_LicenseInfo_State {
	if lic == nil {
		return enterprisev1.ClusterConfig_Status_LicenseInfo_NONE
	}

	if lic.NotBefore != nil && lic.NotBefore.IsValid() &&
		now.Add(licenseClockSkew).Before(lic.NotBefore.AsTime()) {
		return enterprisev1.ClusterConfig_Status_LicenseInfo_NOT_YET_VALID
	}

	if lic.NotAfter != nil && lic.NotAfter.IsValid() &&
		now.Add(-licenseClockSkew).After(lic.NotAfter.AsTime()) {
		return enterprisev1.ClusterConfig_Status_LicenseInfo_EXPIRED
	}

	return enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE
}

func parseLicenseVerificationKeys() (map[string]*licenseVerificationKey, error) {
	set := &licenseJWKSet{}
	dec := json.NewDecoder(bytes.NewReader(licenseJWKSBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(set); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal License JWKS")
	}
	if err := ensureLicenseJSONEOF(dec); err != nil {
		return nil, err
	}
	if len(set.Keys) == 0 {
		return nil, errors.Errorf("The License JWKS contains no keys")
	}

	ret := make(map[string]*licenseVerificationKey, len(set.Keys))
	for _, key := range set.Keys {
		parsed, err := parseLicenseVerificationKey(key)
		if err != nil {
			return nil, err
		}
		if _, ok := ret[key.KeyID]; ok {
			return nil, errors.Errorf("Duplicate License JWK kid: %s", key.KeyID)
		}
		ret[key.KeyID] = parsed
	}

	return ret, nil
}

func ensureLicenseJSONEOF(dec *json.Decoder) error {
	if dec == nil {
		return errors.Errorf("Nil JSON decoder")
	}

	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.Errorf("The License JWKS contains trailing JSON data")
		}
		return errors.Wrap(err, "Could not finish parsing License JWKS")
	}

	return nil
}

func parseLicenseVerificationKey(key *licenseJWK) (*licenseVerificationKey, error) {
	if key == nil {
		return nil, errors.Errorf("Nil License JWK")
	}
	if key.KeyID == "" || len(key.KeyID) > 128 {
		return nil, errors.Errorf("Invalid License JWK kid")
	}
	if key.Use != "" && key.Use != "sig" {
		return nil, errors.Errorf("Invalid use for License JWK: %s", key.KeyID)
	}
	if len(key.KeyOps) > 0 {
		if len(key.KeyOps) != 1 || key.KeyOps[0] != "verify" {
			return nil, errors.Errorf("Invalid key_ops for License JWK: %s", key.KeyID)
		}
	}
	if key.D != "" || key.P != "" || key.Q != "" || key.DP != "" ||
		key.DQ != "" || key.QI != "" {
		return nil, errors.Errorf("The embedded License JWKS contains private key material")
	}

	switch key.KeyType {
	case "OKP":
		if key.Alg != jwt.SigningMethodEdDSA.Alg() || key.Curve != "Ed25519" {
			return nil, errors.Errorf("Unsupported License OKP JWK: %s", key.KeyID)
		}

		x, err := decodeLicenseJWKValue(key.X)
		if err != nil || len(x) != ed25519.PublicKeySize {
			return nil, errors.Errorf("Invalid License Ed25519 JWK: %s", key.KeyID)
		}

		return &licenseVerificationKey{
			alg: key.Alg,
			key: ed25519.PublicKey(x),
		}, nil

	case "EC":
		if key.Alg != jwt.SigningMethodES256.Alg() || key.Curve != "P-256" {
			return nil, errors.Errorf("Unsupported License EC JWK: %s", key.KeyID)
		}

		xBytes, err := decodeLicenseJWKValue(key.X)
		if err != nil || len(xBytes) != 32 {
			return nil, errors.Errorf("Invalid License EC x coordinate: %s", key.KeyID)
		}
		yBytes, err := decodeLicenseJWKValue(key.Y)
		if err != nil || len(yBytes) != 32 {
			return nil, errors.Errorf("Invalid License EC y coordinate: %s", key.KeyID)
		}

		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		if !elliptic.P256().IsOnCurve(x, y) {
			return nil, errors.Errorf("The License EC JWK is not on P-256: %s", key.KeyID)
		}

		return &licenseVerificationKey{
			alg: key.Alg,
			key: &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			},
		}, nil

	case "RSA":
		if key.Alg != jwt.SigningMethodPS256.Alg() {
			return nil, errors.Errorf("Unsupported License RSA JWK: %s", key.KeyID)
		}

		nBytes, err := decodeLicenseJWKValue(key.N)
		if err != nil {
			return nil, errors.Errorf("Invalid License RSA modulus: %s", key.KeyID)
		}
		eBytes, err := decodeLicenseJWKValue(key.E)
		if err != nil {
			return nil, errors.Errorf("Invalid License RSA exponent: %s", key.KeyID)
		}

		n := new(big.Int).SetBytes(nBytes)
		eBig := new(big.Int).SetBytes(eBytes)
		if n.Sign() <= 0 || n.BitLen() < 2048 || !eBig.IsInt64() {
			return nil, errors.Errorf("Invalid License RSA JWK: %s", key.KeyID)
		}
		e := eBig.Int64()
		if e < 3 || e > int64(^uint(0)>>1) || e%2 == 0 {
			return nil, errors.Errorf("Invalid License RSA exponent: %s", key.KeyID)
		}

		return &licenseVerificationKey{
			alg: key.Alg,
			key: &rsa.PublicKey{
				N: n,
				E: int(e),
			},
		}, nil

	default:
		return nil, errors.Errorf("Unsupported License JWK key type: %s", key.KeyType)
	}
}

func getLicenseVerificationKey(
	token *jwt.Token,
	keys map[string]*licenseVerificationKey,
) (any, error) {
	if token == nil || token.Method == nil {
		return nil, errors.Errorf("Invalid License token")
	}

	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, errors.Errorf("The License kid header is not set")
	}

	key, ok := keys[kid]
	if !ok {
		return nil, errors.Errorf("Unknown License signing key")
	}
	if token.Method.Alg() != key.alg {
		return nil, errors.Errorf("The License signing algorithm does not match its key")
	}

	return key.key, nil
}

func decodeLicenseJWKValue(val string) ([]byte, error) {
	if val == "" {
		return nil, errors.Errorf("Empty JWK value")
	}

	ret, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil || len(ret) == 0 {
		return nil, errors.Errorf("Invalid JWK value")
	}

	return ret, nil
}

func (s *Server) getLicenseClusterDomain(ctx context.Context) (string, error) {
	cc, err := s.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return "", err
	}
	if cc.Status == nil {
		return "", nil
	}

	return cc.Status.Domain, nil
}

func (s *Server) getLicenseJWT(
	ctx context.Context,
	ref *metav1.ObjectReference,
) (string, error) {
	if ref == nil || ref.Uid == "" {
		return "", errors.Errorf("Invalid License Secret reference")
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		return "", err
	}
	if err := validateLicenseSecret(sec); err != nil {
		return "", err
	}

	jwtStr := strings.TrimSpace(sec.Data.GetValue())
	if jwtStr == "" {
		return "", errors.Errorf("The License Secret is empty")
	}
	if len(jwtStr) > maxLicenseLen {
		return "", errors.Errorf("The License Secret is too large")
	}

	return jwtStr, nil
}

func (s *Server) createLicenseSecret(
	ctx context.Context,
	jwtStr string,
) (*metav1.ObjectReference, error) {
	sec := &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name:           licenseSecretNamePrefix + "-" + utilrand.GetRandomStringCanonical(12),
			IsSystem:       true,
			IsUserHidden:   true,
			IsSystemHidden: true,
			SystemLabels: map[string]string{
				licenseSecretLabel: "true",
			},
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: jwtStr,
			},
		},
	}

	sec, err := s.octeliumC.EnterpriseC().CreateSecret(ctx, sec)
	if err != nil {
		return nil, err
	}

	return umetav1.GetObjectReference(sec), nil
}

func (s *Server) deleteLicenseSecret(
	ctx context.Context,
	ref *metav1.ObjectReference,
) error {
	if ref == nil || ref.Uid == "" {
		return nil
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := validateLicenseSecret(sec); err != nil {
		return err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteSecret(ctx, apivalidation.ObjectToRDeleteOptions(sec))
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func validateLicenseSecret(sec *enterprisev1.Secret) error {
	if sec == nil || sec.Metadata == nil {
		return errors.Errorf("Invalid License Secret")
	}
	if !sec.Metadata.IsSystem || !sec.Metadata.IsUserHidden ||
		!sec.Metadata.IsSystemHidden ||
		sec.Metadata.SystemLabels[licenseSecretLabel] != "true" ||
		!strings.HasPrefix(sec.Metadata.Name, licenseSecretNamePrefix+"-") {
		return errors.Errorf("The Secret is not owned by the License subsystem")
	}
	if sec.Data == nil || sec.Data.Type == nil {
		return errors.Errorf("The License Secret has no data")
	}
	if _, ok := sec.Data.Type.(*enterprisev1.Secret_Data_Value); !ok {
		return errors.Errorf("The License Secret has invalid data")
	}

	return nil
}

func isSameObjectReference(a, b *metav1.ObjectReference) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Uid != "" || b.Uid != "" {
		return a.Uid != "" && a.Uid == b.Uid
	}
	return a.Name != "" && a.Name == b.Name
}
