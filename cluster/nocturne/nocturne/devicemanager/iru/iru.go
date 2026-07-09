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
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	iruMaxRetries  = 5
	iruMaxRespByte = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 16384
)

var platformUUIDRe = regexp.MustCompile(`(?i)"IOPlatformUUID"\s*=\s*"([0-9a-f-]{36})"`)

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

	return &Manager{
		api: &apiClient{
			httpc: &http.Client{Timeout: iruHTTPTimeout},
			base:  strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/"),
			token: uenterprisev1.ToSecret(sec).GetValueStr(),
		},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_IRU
}

func (m *Manager) Close() error {
	m.api.httpc.CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType: corev1.Device_Status_MAC,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/usr/sbin/ioreg",
				Args:           []string{"-rd1", "-c", "IOPlatformExpertDevice"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	if osType != corev1.Device_Status_MAC {
		return "", nil
	}
	for _, r := range results {
		if r == nil || r.Err != nil || len(r.Output) == 0 {
			continue
		}
		if mm := platformUUIDRe.FindStringSubmatch(string(r.Output)); len(mm) == 2 {
			return strings.ToLower(mm[1]), nil
		}
	}
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	devices, err := m.api.listDevices(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(devices))
	for _, d := range devices {
		if d == nil {
			continue
		}
		entries = append(entries, toEntry(d))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	httpc *http.Client
	base  string
	token string
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
		if err := c.do(ctx, c.base+devicesPath+"?"+q.Encode(), &page); err != nil {
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

func (c *apiClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < iruMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "Iru build request")
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "Iru request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, iruMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "Iru decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("Iru status %d (check API token): %s", resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("Iru status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("Iru status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type iruDevice struct {
	DeviceID     string   `json:"device_id"`
	DeviceName   string   `json:"device_name"`
	SerialNumber string   `json:"serial_number"`
	Platform     string   `json:"platform"`
	OSVersion    string   `json:"os_version"`
	Model        string   `json:"model"`
	MacAddress   string   `json:"mac_address"`
	LastCheckIn  string   `json:"last_check_in"`
	IsMissing    bool     `json:"is_missing"`
	BlueprintID  string   `json:"blueprint_id"`
	User         *iruUser `json:"user"`
}

type iruUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func toEntry(d *iruDevice) *devicemgrcommon.Entry {
	missing := d.IsMissing

	score := int32(100)
	if missing {
		score = 20
	}

	p := &corev1.Device_Status_Posture{
		ExternalID: strings.ToLower(d.DeviceID),
		RiskLevel:  riskBand(score),
		Compliant:  passFail(!missing),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"managed": corev1.Device_Status_Posture_PASS,
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
		ExternalID:  strings.ToLower(d.DeviceID),
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
	put("blueprintId", d.BlueprintID)
	if d.User != nil {
		put("userEmail", d.User.Email)
		put("userName", d.User.Name)
	}
	fields["isMissing"] = d.IsMissing
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

func sleepBackoff(ctx context.Context, attempt int, resp *http.Response) bool {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil && secs >= 0 {
				d = time.Duration(secs) * time.Second
			}
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func snippet(b []byte) string {
	const max = 300
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
