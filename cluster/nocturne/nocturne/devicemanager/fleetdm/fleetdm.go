// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package fleetdm

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
	hostsPath        = "/api/v1/fleet/hosts"
	fleetPageSize    = 100
	fleetHTTPTimeout = 60 * time.Second
	fleetMaxRetries  = 5
	fleetMaxRespByte = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 16384
)

var (
	platformUUIDRe = regexp.MustCompile(`(?i)"IOPlatformUUID"\s*=\s*"([0-9a-f-]{36})"`)
	uuidRe         = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

type Manager struct {
	api    *apiClient
	teamID uint32
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetFleetDM()
	if spec == nil {
		return nil, errors.Errorf("Not a FleetDM DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if strings.TrimSpace(spec.BaseURL) == "" {
		return nil, errors.Errorf("Empty FleetDM baseURL")
	}
	if spec.GetApiToken().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty FleetDM apiToken")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetApiToken().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	return &Manager{
		api: &apiClient{
			httpc: &http.Client{Timeout: fleetHTTPTimeout},
			base:  strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/"),
			token: uenterprisev1.ToSecret(sec).GetValueStr(),
		},
		teamID: spec.TeamID,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_FLEETDM
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
			OSType:           corev1.Device_Status_LINUX,
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
	hosts, err := m.api.listHosts(ctx, m.teamID)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(hosts))
	for _, h := range hosts {
		if h == nil {
			continue
		}
		entries = append(entries, toEntry(h))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	httpc *http.Client
	base  string
	token string
}

type hostsResponse struct {
	Hosts []*fleetHost `json:"hosts"`
}

func (c *apiClient) listHosts(ctx context.Context, teamID uint32) ([]*fleetHost, error) {
	var out []*fleetHost
	page := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(fleetPageSize))
		if teamID > 0 {
			q.Set("team_id", strconv.FormatUint(uint64(teamID), 10))
		}

		var page0 hostsResponse
		if err := c.do(ctx, c.base+hostsPath+"?"+q.Encode(), &page0); err != nil {
			return nil, err
		}
		out = append(out, page0.Hosts...)
		if len(page0.Hosts) < fleetPageSize {
			break
		}
		page++
	}
	return out, nil
}

func (c *apiClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < fleetMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "FleetDM build request")
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "FleetDM request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, fleetMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "FleetDM decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("FleetDM status %d (check API token): %s", resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("FleetDM status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("FleetDM status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type fleetHost struct {
	ID                    int64        `json:"id"`
	Hostname              string       `json:"hostname"`
	UUID                  string       `json:"uuid"`
	HardwareSerial        string       `json:"hardware_serial"`
	PrimaryMAC            string       `json:"primary_mac"`
	Platform              string       `json:"platform"`
	OSVersion             string       `json:"os_version"`
	PublicIP              string       `json:"public_ip"`
	SeenTime              string       `json:"seen_time"`
	DiskEncryptionEnabled bool         `json:"disk_encryption_enabled"`
	Issues                *fleetIssues `json:"issues"`
	MDM                   *fleetMDM    `json:"mdm"`
}

type fleetIssues struct {
	FailingPoliciesCount int `json:"failing_policies_count"`
	TotalIssuesCount     int `json:"total_issues_count"`
}

type fleetMDM struct {
	EnrollmentStatus string `json:"enrollment_status"`
	Name             string `json:"name"`
}

func toEntry(h *fleetHost) *devicemgrcommon.Entry {
	failing := 0
	if h.Issues != nil {
		failing = h.Issues.FailingPoliciesCount
	}
	compliant := failing == 0

	score := int32(100)
	if !compliant {
		score = 100 - int32(failing)*15
		if score < 20 {
			score = 20
		}
	}

	p := &corev1.Device_Status_Posture{
		ExternalID:     strings.ToLower(h.UUID),
		RiskLevel:      riskBand(score),
		DiskEncryption: passFail(h.DiskEncryptionEnabled),
		Compliant:      passFail(compliant),
		Attrs:          fleetAttrs(h),
	}
	if t, ok := parseTime(h.SeenTime); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	var macs []string
	if h.PrimaryMAC != "" {
		macs = []string{h.PrimaryMAC}
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(h.UUID),
		Serial:     h.HardwareSerial,
		MACs:       macs,
		Posture:    p,
	}
}

func fleetAttrs(h *fleetHost) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("fleetHostId", strconv.FormatInt(h.ID, 10))
	put("hostname", h.Hostname)
	put("platform", h.Platform)
	put("osVersion", h.OSVersion)
	put("publicIP", h.PublicIP)
	if h.MDM != nil {
		put("mdmEnrollmentStatus", h.MDM.EnrollmentStatus)
		put("mdmName", h.MDM.Name)
	}
	if h.Issues != nil {
		fields["failingPoliciesCount"] = float64(h.Issues.FailingPoliciesCount)
	}
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
