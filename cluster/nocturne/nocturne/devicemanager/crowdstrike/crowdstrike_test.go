// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package crowdstrike

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestParseExternalID(t *testing.T) {
	m := &Manager{}
	aid := "0123456789abcdef0123456789abcdef"

	raw, err := hex.DecodeString(aid)
	assert.Nil(t, err)

	for _, tc := range []struct {
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{nil, ""},
		{[]*devicemgrcommon.ProbeResult{nil, {}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Data: raw}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Data: raw[:8]}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: fmt.Sprintf(`aid="%s".`, aid)}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: "agentID: 01234567-89AB-CDEF-0123-456789ABCDEF\n"}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: " 0123456789ABCDEF0123456789ABCDEF\n"}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: "cid=ffffffffffffffffffffffffffffffff " + aid}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "ERROR: aid is not available"}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "invalid"}, {Text: `aid="` + aid + `"`}}, aid},
	} {
		ret, err := m.ParseExternalID(corev1.Device_Status_LINUX, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}
}

func TestIdentityProbes(t *testing.T) {
	m := &Manager{}
	seen := map[string]struct{}{}

	for _, probe := range m.IdentityProbes() {
		assert.NotEmpty(t, probe.ID)
		assert.NotEqual(t, corev1.Device_Status_OS_TYPE_UNKNOWN, probe.OSType)
		assert.True(t, probe.RequireElevation)

		_, ok := seen[probe.ID]
		assert.False(t, ok)
		seen[probe.ID] = struct{}{}
	}

	assert.Len(t, seen, 3)

	for _, probe := range m.IdentityProbes() {
		if probe.OSType == corev1.Device_Status_MAC {
			assert.Equal(t, []string{"stats", "agent_info"}, probe.RunCommand.Args)
		}
	}
}

func TestIdentityProbesZTA(t *testing.T) {
	m := &Manager{ztaEnabled: true}

	files := map[corev1.Device_Status_OSType]string{}
	for _, probe := range m.IdentityProbes() {
		if probe.ReadFile == nil {
			assert.True(t, probe.RequireElevation)
			continue
		}

		assert.False(t, probe.RequireElevation, "the file must be read without an elevated client")
		assert.Equal(t, uint32(ztaMaxBytes), probe.ReadFile.MaxBytes)
		files[probe.OSType] = probe.ReadFile.Path
	}

	assert.Equal(t, map[corev1.Device_Status_OSType]string{
		corev1.Device_Status_WINDOWS: `C:\ProgramData\CrowdStrike\ZeroTrustAssessment\data.zta`,
		corev1.Device_Status_MAC:     "/Library/Application Support/CrowdStrike/ZeroTrustAssessment/data.zta",
	}, files)
}

type tstZTAClaims struct {
	jwt.Claims
	CID          string         `json:"cid"`
	Typ          string         `json:"typ"`
	Platform     string         `json:"platform"`
	SerialNumber string         `json:"serial_number"`
	Assessment   map[string]any `json:"assessment"`
}

func tstZTAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.Nil(t, err)
	return key
}

func tstZTAKeySet(kid string, key *rsa.PrivateKey) *jose.JSONWebKeySet {
	return &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{Key: key.Public(), KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"},
		},
	}
}

func tstZTAToken(t *testing.T, alg jose.SignatureAlgorithm, key any, sub string, expiresAt *time.Time, now time.Time) string {
	t.Helper()

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "US-1/v1"))
	require.Nil(t, err)

	claims := &tstZTAClaims{
		Claims: jwt.Claims{
			Subject:  sub,
			IssuedAt: jwt.NewNumericDate(now.Add(-time.Minute)),
		},
		CID:          "abcdef123456789",
		Typ:          "crowdstrike-zta+jwt",
		Platform:     "Windows 10",
		SerialNumber: "serialNumber123",
		Assessment: map[string]any{
			"overall":       55,
			"os":            50,
			"sensor_config": 60,
			"version":       "",
		},
	}
	if expiresAt != nil {
		claims.Expiry = jwt.NewNumericDate(*expiresAt)
	}

	ret, err := jwt.Signed(signer).Claims(claims).Serialize()
	require.Nil(t, err)
	return ret
}

func TestParseSignedExternalID(t *testing.T) {
	const aid = "beefbeefbeefbeefbeefbeefbeef1111"

	now := time.Now()
	valid := now.Add(time.Hour)
	expired := now.Add(-time.Hour)

	key := tstZTAKey(t)
	otherKey := tstZTAKey(t)
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	m := &Manager{}

	getResults := func(text string) []*devicemgrcommon.ProbeResult {
		return []*devicemgrcommon.ProbeResult{{Text: text}}
	}

	token := tstZTAToken(t, jose.RS256, key, aid, &valid, now)

	assert.Empty(t, m.ParseSignedExternalID(getResults(token), now), "there must be no result without the keys")

	m.ztaKeys.Store(tstZTAKeySet("US-1/v1", key))

	assert.Equal(t, aid, m.ParseSignedExternalID(getResults(token), now))
	assert.Equal(t, aid, m.ParseSignedExternalID(getResults(" "+token+"\r\n"), now))
	assert.Equal(t, aid, m.ParseSignedExternalID([]*devicemgrcommon.ProbeResult{
		nil, {Text: "aid=" + aid}, {Text: token},
	}, now))

	assert.Equal(t, aid, m.ParseSignedExternalID(getResults(
		tstZTAToken(t, jose.RS256, key, strings.ToUpper(aid), &valid, now)), now))

	{
		m := &Manager{}
		m.ztaKeys.Store(tstZTAKeySet("v1", key))
		assert.Equal(t, aid, m.ParseSignedExternalID(getResults(token), now), "the key ID must not matter")
	}

	for name, text := range map[string]string{
		"Empty":      "",
		"AID":        aid,
		"NotAToken":  "a.b.c",
		"OtherKey":   tstZTAToken(t, jose.RS256, otherKey, aid, &valid, now),
		"Expired":    tstZTAToken(t, jose.RS256, key, aid, &expired, now),
		"NoExpiry":   tstZTAToken(t, jose.RS256, key, aid, nil, now),
		"NoSubject":  tstZTAToken(t, jose.RS256, key, "", &valid, now),
		"BadSubject": tstZTAToken(t, jose.RS256, key, "not-an-agent-id", &valid, now),
		"HS256":      tstZTAToken(t, jose.HS256, publicKeyBytes, aid, &valid, now),
		"RS512":      tstZTAToken(t, jose.RS512, key, aid, &valid, now),
		"PS256":      tstZTAToken(t, jose.PS256, key, aid, &valid, now),
		"NoneAlg": strings.Join([]string{
			base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)),
			strings.Split(token, ".")[1],
			"",
		}, "."),
		"Tampered": func() string {
			parts := strings.Split(token, ".")
			parts[1] = base64.RawURLEncoding.EncodeToString(
				[]byte(fmt.Sprintf(`{"sub":"%s","exp":%d}`, strings.Repeat("a", 32), valid.Unix())))
			return strings.Join(parts, ".")
		}(),
	} {
		assert.Empty(t, m.ParseSignedExternalID(getResults(text), now), name)
	}

	assert.Empty(t, m.ParseSignedExternalID(getResults(token), valid.Add(time.Hour)),
		"a token must stop being accepted once it expires")
}

func TestParseExternalIDIgnoresZTAToken(t *testing.T) {
	const aid = "beefbeefbeefbeefbeefbeefbeef1111"

	now := time.Now()
	valid := now.Add(time.Hour)
	token := tstZTAToken(t, jose.RS256, tstZTAKey(t), strings.Repeat("a", 32), &valid, now)

	m := &Manager{}
	ret, err := m.ParseExternalID(corev1.Device_Status_WINDOWS, []*devicemgrcommon.ProbeResult{
		{Text: token}, {Text: aid},
	})
	assert.Nil(t, err)
	assert.Equal(t, aid, ret)
}

func TestRefreshZTAKeys(t *testing.T) {
	key := tstZTAKey(t)

	body, err := json.Marshal(tstZTAKeySet("US-1/v1", key))
	require.Nil(t, err)

	var status atomic.Int32
	status.Store(http.StatusOK)

	var respBody atomic.Pointer[string]
	valid := string(body)
	respBody.Store(&valid)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
		fmt.Fprint(w, *respBody.Load())
	}))
	t.Cleanup(srv.Close)

	m := &Manager{ztaKeysURL: srv.URL}

	require.Nil(t, m.refreshZTAKeys(t.Context()))
	keys := m.ztaKeys.Load()
	require.NotNil(t, keys)
	assert.Len(t, keys.Keys, 1)

	status.Store(http.StatusInternalServerError)
	assert.NotNil(t, m.refreshZTAKeys(t.Context()))
	assert.True(t, keys == m.ztaKeys.Load(), "the last keys must be kept")

	status.Store(http.StatusOK)
	for _, invalid := range []string{"not json", `{"keys": []}`} {
		respBody.Store(&invalid)
		assert.NotNil(t, m.refreshZTAKeys(t.Context()), invalid)
		assert.True(t, keys == m.ztaKeys.Load(), "the last keys must be kept")
	}

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL, http.StatusFound)
	}))
	t.Cleanup(redirect.Close)

	respBody.Store(&valid)
	require.Nil(t, m.refreshZTAKeys(t.Context()))
	keys = m.ztaKeys.Load()

	m.ztaKeysURL = redirect.URL
	assert.NotNil(t, m.refreshZTAKeys(t.Context()), "a redirect to another origin must be rejected")
	assert.True(t, keys == m.ztaKeys.Load(), "the last keys must be kept")
}

func TestToEntry(t *testing.T) {
	now := time.Now()

	getDevice := func(status, rfm string, lastSeen time.Time) *models.DeviceapiDeviceSwagger {
		ret := &models.DeviceapiDeviceSwagger{}
		err := json.Unmarshal([]byte(fmt.Sprintf(`{
			"device_id": "0123456789ABCDEF0123456789ABCDEF",
			"serial_number": "C02XYZ",
			"mac_address": "a4-bb-cc-dd-ee-01",
			"connection_mac_address": "A4-BB-CC-DD-EE-01",
			"status": %q,
			"reduced_functionality_mode": %q,
			"last_seen": %q,
			"platform_name": "Windows",
			"hostname": "DESKTOP-1",
			"agent_version": "7.10.0"
		}`, status, rfm, lastSeen.UTC().Format(time.RFC3339))), ret)
		assert.Nil(t, err)
		return ret
	}

	{
		entry := toEntry(getDevice("normal", "no", now.Add(-time.Hour)), nil, now)
		assert.Equal(t, "0123456789abcdef0123456789abcdef", entry.ExternalID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"a4-bb-cc-dd-ee-01"}, entry.MACs)
		assert.Equal(t, "DESKTOP-1", entry.Hostname)
		assert.Equal(t, corev1.Device_Status_WINDOWS, entry.OSType)
		assert.Empty(t, entry.MachineID)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Nil(t, p.Score)
		assert.Nil(t, p.FailingChecks)
		assert.Nil(t, p.AssessedAt)
		assert.True(t, p.LastSeenAt.IsValid())
		assert.Equal(t, "7.10.0", p.Attrs.Fields["agentVersion"].GetStringValue())
	}

	{
		p := toEntry(getDevice("contained", "no", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
	}

	{
		p := toEntry(getDevice("lift_containment_pending", "yes", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("", "", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("normal", "no", now.Add(-72*time.Hour)), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("normal", "", now.Add(-72*time.Hour)), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		d := getDevice("normal", "no", now)
		d.LastSeen = ""
		p := toEntry(d, nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
		assert.Nil(t, p.LastSeenAt)
	}

	{
		zta := &models.DomainSignalProperties{}
		err := json.Unmarshal([]byte(`{
			"aid": "0123456789abcdef0123456789abcdef",
			"cid": "cid-1",
			"modified_time": "2026-09-28T10:00:00Z",
			"assessment": {"overall": 95, "os": 80, "sensor_config": 70},
			"assessment_items": {
				"os_signals": [
					{"signal_id": "a", "meets_criteria": "yes"},
					{"signal_id": "b", "meets_criteria": "no"},
					null
				],
				"sensor_signals": [
					{"signal_id": "c", "meets_criteria": "unknown"},
					{"signal_id": "d", "meets_criteria": "YES"}
				]
			}
		}`), zta)
		assert.Nil(t, err)

		p := toEntry(getDevice("normal", "no", now), zta, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel,
			"the Cluster must not invent a level out of a number")
		assert.EqualValues(t, 95, p.GetScore())
		assert.EqualValues(t, 2, p.GetFailingChecks())
		assert.Equal(t, time.Date(2026, time.September, 28, 10, 0, 0, 0, time.UTC), p.AssessedAt.AsTime())
		assert.Nil(t, p.Attrs.Fields["ztaScoreOverall"])
		assert.EqualValues(t, 80, p.Attrs.Fields["ztaScoreOs"].GetNumberValue())
		assert.EqualValues(t, 70, p.Attrs.Fields["ztaScoreSensor"].GetNumberValue())
		assert.Equal(t, "cid-1", p.Attrs.Fields["cid"].GetStringValue())
	}

	for _, tc := range []struct {
		name    string
		arg     string
		score   *uint32
		failing *uint32
	}{
		{
			name:  "Zero",
			arg:   `{"assessment": {"overall": 0}}`,
			score: proto.Uint32(0),
		},
		{
			name:  "OutOfRange",
			arg:   `{"assessment": {"overall": 250}}`,
			score: proto.Uint32(100),
		},
		{
			name: "NoAssessment",
			arg:  `{}`,
		},
		{
			name: "NoSignals",
			arg:  `{"assessment_items": {"os_signals": [], "sensor_signals": []}}`,
		},
		{
			name:    "AllSignalsMet",
			arg:     `{"assessment_items": {"os_signals": [{"meets_criteria": "yes"}]}}`,
			failing: proto.Uint32(0),
		},
		{
			name:    "MissingCriteriaIsNotMet",
			arg:     `{"assessment_items": {"sensor_signals": [{"signal_id": "a"}]}}`,
			failing: proto.Uint32(1),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			zta := &models.DomainSignalProperties{}
			assert.Nil(t, json.Unmarshal([]byte(tc.arg), zta))

			p := toEntry(getDevice("normal", "no", now), zta, now).Posture
			assert.Equal(t, tc.score, p.Score)
			assert.Equal(t, tc.failing, p.FailingChecks)
		})
	}

	{
		d := getDevice("normal", "no", now)
		d.PlatformName = "Windows"
		d.OsVersion = "Windows 11"
		d.ChassisTypeDesc = "Laptop"

		p := toEntry(d, nil, now).Posture
		assert.Equal(t, corev1.Device_Status_WINDOWS, p.Os.Type)
		assert.Equal(t, "Windows 11", p.Os.Version)
		assert.Nil(t, p.Os.Major, "`11` is a product name rather than a version")
		assert.Equal(t, corev1.Device_Status_Posture_WORKSTATION, p.DeviceClass)
	}

	{
		d := getDevice("normal", "no", now)
		d.ProductTypeDesc = "Domain Controller"
		assert.Equal(t, corev1.Device_Status_Posture_SERVER, toEntry(d, nil, now).Posture.DeviceClass)

		d.ChassisTypeDesc = "Virtual"
		assert.Equal(t, corev1.Device_Status_Posture_VIRTUAL_MACHINE, toEntry(d, nil, now).Posture.DeviceClass)
	}
}

func TestChunk(t *testing.T) {
	assert.Nil(t, chunk(nil, 2))
	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, chunk([]string{"a", "b", "c"}, 2))
	assert.Equal(t, [][]string{{"a"}}, chunk([]string{"a"}, 0))
}

func TestParseExternalIDDisagreement(t *testing.T) {
	m := &Manager{}

	_, err := m.ParseExternalID(corev1.Device_Status_LINUX, []*devicemgrcommon.ProbeResult{
		{Text: `aid="0123456789abcdef0123456789abcdef"`},
		{Text: `aid="ffffffffffffffffffffffffffffffff"`},
	})
	assert.NotNil(t, err)
}

func TestGetPayloadError(t *testing.T) {
	code := int32(404)
	message := "not found"

	assert.Nil(t, getPayloadError(nil))
	assert.Nil(t, getPayloadError([]*models.MsaAPIError{nil}))

	{
		err := getPayloadError([]*models.MsaAPIError{{Code: &code, Message: &message}})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Contains(t, err.Error(), message)
	}

	{
		err := getPayloadError([]*models.MsaAPIError{{Code: &code, Message: &message}, {}})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "2 errors")
	}

	{
		err := getPayloadError([]*models.MsaAPIError{{}})
		assert.NotNil(t, err)
	}
}

type tstTransport struct{}

func (tstTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req)
}

func TestCollect(t *testing.T) {
	aid1 := "0123456789abcdef0123456789abcdef"
	aid2 := "fedcba9876543210fedcba9876543210"

	mode := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/devices/queries/devices-scroll/v1":
			switch mode {
			case "scrollError":
				fmt.Fprint(w, `{"meta": {}, "resources": [], "errors": [{"code": 500, "message": "internal"}]}`)
			case "empty":
				fmt.Fprint(w, `{"meta": {"pagination": {"offset": "", "total": 0}}, "resources": [], "errors": []}`)
			default:
				fmt.Fprintf(w, `{"meta": {"pagination": {"offset": "", "total": 2}}, "resources": [%q, %q], "errors": []}`,
					strings.ToUpper(aid1), aid2)
			}
		case "/devices/entities/devices/v2":
			assert.Equal(t, http.MethodPost, r.Method)

			req := &models.MsaIdsRequest{}
			assert.Nil(t, json.NewDecoder(r.Body).Decode(req))
			assert.Equal(t, []string{strings.ToUpper(aid1), aid2}, req.Ids)

			errs := `[]`
			if mode == "detailsError" {
				errs = `[{"code": 404, "message": "not found"}]`
			}
			fmt.Fprintf(w, `{"meta": {}, "resources": [
				{"device_id": %q, "status": "normal", "serial_number": "SERIAL-1"},
				{"device_id": %q, "status": "contained"}
			], "errors": %s}`, strings.ToUpper(aid1), aid2, errs)
		case "/zero-trust-assessment/entities/assessments/v1":
			if mode == "ztaError" {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"meta": {}, "errors": [{"code": 403, "message": "access denied"}]}`)
				return
			}
			if mode == "ztaPartialError" {
				fmt.Fprintf(w, `{"meta": {}, "resources": [{"aid": %q, "assessment": {"overall": 95}}],
					"errors": [{"code": 500, "message": "partial failure"}]}`, aid1)
				return
			}
			fmt.Fprintf(w, `{"meta": {}, "resources": [{"aid": %q, "assessment": {"overall": 95}}],
				"errors": [{"code": 404, "message": "no assessment"}]}`, aid1)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := falcon.NewClient(&falcon.ApiConfig{
		Context:      context.Background(),
		AccessToken:  "token",
		HostOverride: strings.TrimPrefix(srv.URL, "http://"),
		TransportDecorator: func(http.RoundTripper) http.RoundTripper {
			return tstTransport{}
		},
	})
	assert.Nil(t, err)

	keysBody, err := json.Marshal(tstZTAKeySet("US-1/v1", tstZTAKey(t)))
	require.Nil(t, err)

	keysSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(keysBody)
	}))
	t.Cleanup(keysSrv.Close)

	m := &Manager{c: c, ztaEnabled: true, ztaKeysURL: keysSrv.URL}

	{
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.Empty(t, fleet.DegradedReason())
		assert.NotNil(t, m.ztaKeys.Load(), "the ZTA keys must be refreshed with the inventory")

		res := fleet.MatchExternalID(aid1)
		assert.Equal(t, devicemgrcommon.MatchStateUnique, res.State)
		assert.EqualValues(t, 95, res.Entry.Posture.GetScore())
		assert.Equal(t, corev1.Device_Status_Posture_PASS, res.Entry.Posture.NotContained)

		res = fleet.MatchExternalID(aid2)
		assert.Nil(t, res.Entry.Posture.Score)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, res.Entry.Posture.NotContained)
	}

	{
		mode = "ztaPartialError"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Contains(t, fleet.DegradedReason(), "partial failure")
		assert.EqualValues(t, 95, fleet.MatchExternalID(aid1).Entry.Posture.GetScore())
	}

	{
		mode = "ztaError"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.NotEmpty(t, fleet.DegradedReason())
		assert.Nil(t, fleet.MatchExternalID(aid1).Entry.Posture.Score)
	}

	{
		mode = "detailsError"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.Contains(t, fleet.DegradedReason(), "not found")
	}

	{
		mode = "scrollError"
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
	}

	{
		mode = "empty"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 0, fleet.Len())
	}

	{
		mode = ""
		m.ztaEnabled = false
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Empty(t, fleet.DegradedReason())
		assert.Nil(t, fleet.MatchExternalID(aid1).Entry.Posture.Score)
	}
}

func TestGetZTAPartial(t *testing.T) {
	var ids []string
	for i := range csZTABatch + 1 {
		ids = append(ids, fmt.Sprintf("%032x", i))
	}

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) > 1 {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"meta": {}, "errors": [{"code": 403, "message": "access denied"}]}`)
			return
		}

		batch := r.URL.Query()["ids"]
		assert.Len(t, batch, csZTABatch)

		var resources []string
		for _, id := range batch {
			resources = append(resources, fmt.Sprintf(`{"aid": %q, "assessment": {"overall": 50}}`, strings.ToUpper(id)))
		}
		fmt.Fprintf(w, `{"meta": {}, "resources": [%s], "errors": []}`, strings.Join(resources, ","))
	}))
	t.Cleanup(srv.Close)

	c, err := falcon.NewClient(&falcon.ApiConfig{
		Context:      context.Background(),
		AccessToken:  "token",
		HostOverride: strings.TrimPrefix(srv.URL, "http://"),
		TransportDecorator: func(http.RoundTripper) http.RoundTripper {
			return tstTransport{}
		},
	})
	require.Nil(t, err)

	m := &Manager{c: c, ztaEnabled: true}

	zta, err := m.getZTA(context.Background(), ids)
	assert.NotNil(t, err)
	assert.Len(t, zta, csZTABatch)
	assert.NotNil(t, zta[ids[0]])
	assert.Nil(t, zta[ids[csZTABatch]])
	assert.Equal(t, int32(2), requests.Load())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	zta, err = m.getZTA(ctx, ids)
	assert.NotNil(t, err)
	assert.Empty(t, zta)
}

func TestTokenContext(t *testing.T) {
	ctx := tokenContext(context.Background())
	hc, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
	require.True(t, ok)
	assert.Equal(t, tokenHTTPTimeout, hc.Timeout)
	assert.NotNil(t, hc.CheckRedirect)

	var foreignRequests atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignRequests.Add(1)
	}))
	t.Cleanup(foreign.Close)

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Redirect(w, r, foreign.URL+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)

	c, err := falcon.NewClient(&falcon.ApiConfig{
		Context:      ctx,
		AccessToken:  "token",
		HostOverride: strings.TrimPrefix(srv.URL, "http://"),
		TransportDecorator: func(next http.RoundTripper) http.RoundTripper {
			return tstTransport{}
		},
	})
	require.Nil(t, err)

	m := &Manager{c: c}
	_, err = m.Collect(context.Background())
	require.NotNil(t, err)
	assert.ErrorIs(t, err, devicemgrcommon.ErrRedirectNotAllowed)
	assert.Equal(t, int32(1), requests.Load())
	assert.Zero(t, foreignRequests.Load())
}

func TestConstructorCredentialsAndConfiguration(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateSecret(t.Context(), &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: "provider-credential"},
		Spec:     &enterprisev1.Secret_Spec{},
		Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "test-token"}},
	})
	require.NoError(t, err)
	configuration := func(t *testing.T) *enterprisev1.DeviceManager {
		t.Helper()
		ret := &enterprisev1.DeviceManager{}
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"crowdStrike\": {\"clientID\": \"client\", \"clientSecret\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		ret.Spec.GetCrowdStrike().Region = enterprisev1.DeviceManager_Spec_CrowdStrike_US_1
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_CROWDSTRIKE, manager.Type())
	assert.True(t, manager.ztaEnabled)
	assert.Equal(t, ztaKeysURL, manager.ztaKeysURL)
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"clientSecret", "clientID"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["crowdStrike"].(map[string]any), field)
			encoded, err = json.Marshal(document)
			require.NoError(t, err)
			bad := &enterprisev1.DeviceManager{}
			require.NoError(t, protojson.Unmarshal(encoded, bad))
			_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: bad})
			assert.Error(t, err)
		})
	}
	encoded, err := protojson.Marshal(configuration(t))
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(encoded, &document))
	document["spec"].(map[string]any)["crowdStrike"].(map[string]any)["clientSecret"] = map[string]any{"fromSecret": "missing-credential"}
	encoded, err = json.Marshal(document)
	require.NoError(t, err)
	missing := &enterprisev1.DeviceManager{}
	require.NoError(t, protojson.Unmarshal(encoded, missing))
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: missing})
	assert.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = New(ctx, tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: configuration(t)})
	assert.Error(t, err)

}

func TestExplicitCloudRegions(t *testing.T) {
	for _, tc := range []struct {
		region enterprisev1.DeviceManager_Spec_CrowdStrike_Region
		cloud  falcon.CloudType
	}{
		{enterprisev1.DeviceManager_Spec_CrowdStrike_US_1, falcon.CloudUs1},
		{enterprisev1.DeviceManager_Spec_CrowdStrike_US_2, falcon.CloudUs2},
		{enterprisev1.DeviceManager_Spec_CrowdStrike_EU_1, falcon.CloudEu1},
		{enterprisev1.DeviceManager_Spec_CrowdStrike_US_GOV_1, falcon.CloudUsGov1},
		{enterprisev1.DeviceManager_Spec_CrowdStrike_US_GOV_2, falcon.CloudUsGov2},
	} {
		assert.Equal(t, tc.cloud, cloudFromRegion(tc.region))
	}
}
