// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package onepassword

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
	defaultBaseURL = "https://k2.kolide.com"
	devicesPath    = "/api/v0/devices"
	opPageSize     = 100
	opHTTPTimeout  = 60 * time.Second
	opMaxRetries   = 5
	opMaxRespByte  = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 16384
)

var (
	platformUUIDRe = regexp.MustCompile(`(?i)"IOPlatformUUID"\s*=\s*"([0-9a-f-]{36})"`)
	uuidRe         = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

type Manager struct {
	api *apiClient
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetOnePassword()
	if spec == nil {
		return nil, errors.Errorf("Not a OnePassword DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if spec.GetApiToken().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty OnePassword apiToken")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetApiToken().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}

	return &Manager{
		api: &apiClient{
			httpc: &http.Client{Timeout: opHTTPTimeout},
			base:  base,
			token: uenterprisev1.ToSecret(sec).GetValueStr(),
		},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_ONEPASSWORD
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
		{
			OSType: corev1.Device_Status_WINDOWS,

			RunCommand: &devicemgrcommon.RunCommand{
				Command:        `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
				Args:           []string{"-NoProfile", "-Command", "(Get-CimInstance -ClassName Win32_ComputerSystemProduct).UUID"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType: corev1.Device_Status_LINUX,

			RequireElevation: true,
			ReadFile: &devicemgrcommon.ReadFile{
				Path:     "/sys/class/dmi/id/product_uuid",
				MaxBytes: 128,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	for _, r := range results {
		if r == nil || r.Err != nil || len(r.Output) == 0 {
			continue
		}
		s := string(r.Output)
		if osType == corev1.Device_Status_MAC {
			if mm := platformUUIDRe.FindStringSubmatch(s); len(mm) == 2 {
				return strings.ToLower(mm[1]), nil
			}
			continue
		}
		if u := uuidRe.FindString(s); u != "" {
			return strings.ToLower(u), nil
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

type devicesResponse struct {
	Data       []*kolideDevice `json:"data"`
	Pagination struct {
		Next  string `json:"next"`
		Count int    `json:"count"`
	} `json:"pagination"`
}

func (c *apiClient) listDevices(ctx context.Context) ([]*kolideDevice, error) {
	var out []*kolideDevice
	cursor := ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(opPageSize))
		if cursor != "" {
			q.Set("cursor", cursor)
		}

		var page devicesResponse
		if err := c.do(ctx, c.base+devicesPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data...)
		if page.Pagination.Next == "" || len(page.Data) == 0 {
			break
		}
		cursor = page.Pagination.Next
	}
	return out, nil
}

func (c *apiClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < opMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "OnePassword build request")
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "OnePassword request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, opMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "OnePassword decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("OnePassword status %d (check API token scope): %s", resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("OnePassword status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("OnePassword status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type kolideDevice struct {
	ID                   int64        `json:"id"`
	Name                 string       `json:"name"`
	HardwareUUID         string       `json:"hardware_uuid"`
	Serial               string       `json:"serial"`
	Platform             string       `json:"platform"`
	OperatingSystem      string       `json:"operating_system"`
	OSVersion            string       `json:"os_version"`
	LastSeenAt           string       `json:"last_seen_at"`
	FailureCount         int          `json:"failure_count"`
	ResolvedFailureCount int          `json:"resolved_failure_count"`
	AuthState            string       `json:"auth_state"`
	Note                 string       `json:"note"`
	PrimaryUserName      string       `json:"primary_user_name"`
	AssignedOwner        *kolideOwner `json:"assigned_owner"`
}

type kolideOwner struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func toEntry(d *kolideDevice) *devicemgrcommon.Entry {
	compliant := d.FailureCount == 0

	score := int32(100)
	if !compliant {
		score = 100 - int32(d.FailureCount)*20
		if score < 20 {
			score = 20
		}
	}

	p := &corev1.Device_Status_Posture{
		ExternalID: strings.ToLower(d.HardwareUUID),
		RiskLevel:  riskBand(score),
		Compliant:  passFail(compliant),
		Attrs:      kolideAttrs(d),
	}
	if t, ok := parseTime(d.LastSeenAt); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(d.HardwareUUID),
		Serial:     d.Serial,
		Posture:    p,
	}
}

func kolideOwnerEmail(d *kolideDevice) string {
	if d.AssignedOwner != nil && d.AssignedOwner.Email != "" {
		return d.AssignedOwner.Email
	}
	return d.PrimaryUserName
}

func kolideAttrs(d *kolideDevice) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("deviceId", strconv.FormatInt(d.ID, 10))
	put("platform", d.Platform)
	put("operatingSystem", d.OperatingSystem)
	put("osVersion", d.OSVersion)
	put("authState", d.AuthState)
	put("ownerEmail", kolideOwnerEmail(d))
	put("note", d.Note)
	fields["failureCount"] = float64(d.FailureCount)
	fields["resolvedFailureCount"] = float64(d.ResolvedFailureCount)
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
