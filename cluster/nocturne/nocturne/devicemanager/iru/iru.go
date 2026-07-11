// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package iru

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	devicesPath    = "/api/v1/devices"
	iruPageSize    = 300
	iruHTTPTimeout = 60 * time.Second
	iruMaxRetries  = 4
	iruMaxRespByte = 64 << 20
)

type Manager struct {
	api *apiClient
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetIru()
	if spec == nil {
		return nil, errors.Errorf("Not an Iru DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if strings.TrimSpace(spec.BaseURL) == "" {
		return nil, errors.Errorf("Empty Iru baseURL")
	}
	if spec.GetApiToken().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty Iru apiToken")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetApiToken().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	rc := resty.New().
		SetBaseURL(strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")).
		SetTimeout(iruHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetAuthToken(uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(iruMaxRespByte).
		SetRetryCount(iruMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api: &apiClient{rc: rc},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_IRU
}

func (m *Manager) Close() error {
	m.api.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return nil
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	devices, err := m.api.listDevices(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(devices))
	for _, d := range devices {
		if d == nil || d.IsRemoved {
			continue
		}
		entries = append(entries, toEntry(d))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	rc *resty.Client
}

func (c *apiClient) listDevices(ctx context.Context) ([]*iruDevice, error) {
	var out []*iruDevice
	offset := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("limit", strconv.Itoa(iruPageSize))
		q.Set("offset", strconv.Itoa(offset))

		var page []*iruDevice
		if err := c.get(ctx, devicesPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < iruPageSize {
			break
		}
		offset += iruPageSize
	}
	return out, nil
}

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "Iru request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("Iru status %d (check API token and Device list permission): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("Iru status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	return nil
}

type iruDevice struct {
	DeviceID       string   `json:"device_id"`
	DeviceName     string   `json:"device_name"`
	SerialNumber   string   `json:"serial_number"`
	Platform       string   `json:"platform"`
	OSVersion      string   `json:"os_version"`
	Model          string   `json:"model"`
	MacAddress     string   `json:"mac_address"`
	LastCheckIn    string   `json:"last_check_in"`
	IsMissing      bool     `json:"is_missing"`
	IsRemoved      bool     `json:"is_removed"`
	MDMEnabled     bool     `json:"mdm_enabled"`
	AgentInstalled bool     `json:"agent_installed"`
	AgentVersion   string   `json:"agent_version"`
	AssetTag       string   `json:"asset_tag"`
	BlueprintID    string   `json:"blueprint_id"`
	User           *iruUser `json:"user"`
}

type iruUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func toEntry(d *iruDevice) *devicemgrcommon.Entry {
	healthy := d.MDMEnabled && !d.IsMissing

	score := int32(100)
	if d.IsMissing {
		score = 20
	} else if !d.MDMEnabled {
		score = 50
	}

	p := &corev1.Device_Status_Posture{
		RiskLevel:      riskBand(score),
		DiskEncryption: corev1.Device_Status_Posture_NOT_APPLICABLE,
		Compliant:      passFail(healthy),
		ThreatFree:     corev1.Device_Status_Posture_NOT_APPLICABLE,
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"mdmEnabled":     passFail(d.MDMEnabled),
			"agentInstalled": passFail(d.AgentInstalled),
			"notMissing":     passFail(!d.IsMissing),
		},
		Attrs: iruAttrs(d),
	}
	if t, ok := parseTime(d.LastCheckIn); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	var macs []string
	if d.MacAddress != "" {
		macs = []string{d.MacAddress}
	}

	var emails []string
	if d.User != nil && strings.Contains(d.User.Email, "@") {
		emails = []string{d.User.Email}
	}

	return &devicemgrcommon.Entry{
		Serial:      d.SerialNumber,
		MACs:        macs,
		OwnerEmails: emails,
		Posture:     p,
	}
}

func iruAttrs(d *iruDevice) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("deviceId", d.DeviceID)
	put("platform", d.Platform)
	put("osVersion", d.OSVersion)
	put("model", d.Model)
	put("agentVersion", d.AgentVersion)
	put("assetTag", d.AssetTag)
	put("blueprintId", d.BlueprintID)
	if d.User != nil {
		put("userEmail", d.User.Email)
		put("userName", d.User.Name)
	}
	fields["isMissing"] = d.IsMissing
	fields["mdmEnabled"] = d.MDMEnabled
	fields["agentInstalled"] = d.AgentInstalled
	if len(fields) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return s
}

func riskBand(score int32) corev1.Device_Status_Posture_RiskLevel {
	switch {
	case score >= 90:
		return corev1.Device_Status_Posture_LOW
	case score >= 70:
		return corev1.Device_Status_Posture_MEDIUM
	case score >= 40:
		return corev1.Device_Status_Posture_HIGH
	default:
		return corev1.Device_Status_Posture_CRITICAL
	}
}

func passFail(ok bool) corev1.Device_Status_Posture_SignalState {
	if ok {
		return corev1.Device_Status_Posture_PASS
	}
	return corev1.Device_Status_Posture_FAIL
}

func parseTime(s string) (time.Time, bool) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func retryCondition(r *resty.Response, err error) bool {
	if err != nil {
		return true
	}
	return r.StatusCode() == http.StatusTooManyRequests || r.StatusCode() >= 500
}

func retryAfter(c *resty.Client, r *resty.Response) (time.Duration, error) {
	if r != nil {
		if ra := r.Header().Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil && secs >= 0 && secs <= 300 {
				return time.Duration(secs) * time.Second, nil
			}
		}
	}
	return 0, nil
}

func snippet(b []byte) string {
	const max = 300
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
