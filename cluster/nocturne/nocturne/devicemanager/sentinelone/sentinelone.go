// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package sentinelone

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
	agentsPath    = "/web/api/v2.1/agents"
	s1PageLimit   = 1000
	s1HTTPTimeout = 60 * time.Second
	s1MaxRetries  = 5
	s1MaxRespByte = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 8192
)

var uuidRe = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

type Manager struct {
	api        *apiClient
	siteIDs    string
	accountIDs string
	agentQuery string
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetSentinelOne()
	if spec == nil {
		return nil, errors.Errorf("Not a SentinelOne DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if strings.TrimSpace(spec.ManagementURL) == "" {
		return nil, errors.Errorf("Empty SentinelOne managementURL")
	}
	if spec.GetApiToken().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty SentinelOne apiToken")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetApiToken().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	return &Manager{
		api: &apiClient{
			httpc: &http.Client{Timeout: s1HTTPTimeout},
			base:  strings.TrimRight(strings.TrimSpace(spec.ManagementURL), "/"),
			token: uenterprisev1.ToSecret(sec).GetValueStr(),
		},
		siteIDs:    spec.SiteIDs,
		accountIDs: spec.AccountIDs,
		agentQuery: spec.AgentQuery,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_SENTINELONE
}

func (m *Manager) Close() error {
	m.api.httpc.CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType: corev1.Device_Status_LINUX,

			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/opt/sentinelone/bin/sentinelctl",
				Args:           []string{"agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType: corev1.Device_Status_MAC,

			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/Library/Sentinel/sentinel-agent.bundle/Contents/MacOS/sentinelctl",
				Args:           []string{"agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType: corev1.Device_Status_WINDOWS,

			RequireElevation: true,
			ReadRegistry: &devicemgrcommon.ReadRegistry{
				Key:  `HKLM\SOFTWARE\SentinelOne\Monitor`,
				Name: "AgentUuid",
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
		if u := uuidRe.FindString(s); u != "" {
			return strings.ToLower(u), nil
		}
		if t := strings.ToLower(strings.TrimSpace(s)); t != "" && !strings.ContainsAny(t, " \t") {
			return t, nil
		}
	}
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	q := url.Values{}
	if m.siteIDs != "" {
		q.Set("siteIds", m.siteIDs)
	}
	if m.accountIDs != "" {
		q.Set("accountIds", m.accountIDs)
	}
	if m.agentQuery != "" {
		if extra, err := url.ParseQuery(m.agentQuery); err == nil {
			for k, vs := range extra {
				for _, v := range vs {
					q.Add(k, v)
				}
			}
		}
	}

	agents, err := m.api.listAgents(ctx, q)
	if err != nil {
		return nil, err
	}

	entries := make([]*devicemgrcommon.Entry, 0, len(agents))
	for _, a := range agents {
		if a == nil || a.UUID == "" {
			continue
		}
		entries = append(entries, toEntry(a))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	httpc *http.Client
	base  string
	token string
}

type agentsResponse struct {
	Data       []*s1Agent `json:"data"`
	Pagination struct {
		NextCursor string `json:"nextCursor"`
		TotalItems int    `json:"totalItems"`
	} `json:"pagination"`
}

func (c *apiClient) listAgents(ctx context.Context, base url.Values) ([]*s1Agent, error) {
	var out []*s1Agent
	cursor := ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		for k, vs := range base {
			q[k] = append([]string(nil), vs...)
		}
		q.Set("limit", strconv.Itoa(s1PageLimit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}

		var page agentsResponse
		if err := c.do(ctx, http.MethodGet, agentsPath, q, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data...)
		if page.Pagination.NextCursor == "" {
			break
		}
		cursor = page.Pagination.NextCursor
	}
	return out, nil
}

func (c *apiClient) do(ctx context.Context, method, path string, q url.Values, out any) error {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var lastErr error
	for attempt := 0; attempt < s1MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, nil)
		if err != nil {
			return errors.Wrap(err, "SentinelOne build request")
		}
		req.Header.Set("Authorization", "ApiToken "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "SentinelOne request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, s1MaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "SentinelOne decode")
			}
			return nil
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("SentinelOne status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("SentinelOne status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type s1Agent struct {
	ID                    string       `json:"id"`
	UUID                  string       `json:"uuid"`
	ComputerName          string       `json:"computerName"`
	SerialNumber          string       `json:"serialNumber"`
	ModelName             string       `json:"modelName"`
	MachineType           string       `json:"machineType"`
	OSName                string       `json:"osName"`
	OSType                string       `json:"osType"`
	AgentVersion          string       `json:"agentVersion"`
	IsActive              bool         `json:"isActive"`
	IsUpToDate            bool         `json:"isUpToDate"`
	Infected              bool         `json:"infected"`
	ActiveThreats         int          `json:"activeThreats"`
	NetworkStatus         string       `json:"networkStatus"`
	LastActiveDate        string       `json:"lastActiveDate"`
	Domain                string       `json:"domain"`
	SiteName              string       `json:"siteName"`
	GroupName             string       `json:"groupName"`
	EncryptedApplications bool         `json:"encryptedApplications"`
	FirewallEnabled       bool         `json:"firewallEnabled"`
	ScanStatus            string       `json:"scanStatus"`
	MitigationMode        string       `json:"mitigationMode"`
	NetworkInterfaces     []s1NetIface `json:"networkInterfaces"`
}

type s1NetIface struct {
	Name     string `json:"name"`
	Physical string `json:"physical"`
}

func toEntry(a *s1Agent) *devicemgrcommon.Entry {
	threat := a.Infected || a.ActiveThreats > 0

	score := int32(100)
	if !a.IsActive {
		score -= 30
	}
	if !a.IsUpToDate {
		score -= 15
	}
	if !a.FirewallEnabled {
		score -= 15
	}
	if !a.EncryptedApplications {
		score -= 25
	}
	if threat && score > 20 {
		score = 20
	}
	if score < 0 {
		score = 0
	}

	p := &corev1.Device_Status_Posture{
		ExternalID:     strings.ToLower(a.UUID),
		RiskLevel:      riskBand(score),
		DiskEncryption: passFail(a.EncryptedApplications),
		Compliant:      passFail(a.IsActive),
		ThreatFree:     passFail(!threat),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"firewall":      passFail(a.FirewallEnabled),
			"agentUpToDate": passFail(a.IsUpToDate),
		},
		Attrs: s1Attrs(a),
	}
	if t, ok := parseTime(a.LastActiveDate); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(a.UUID),
		Serial:     a.SerialNumber,
		MACs:       s1MACs(a),
		Posture:    p,
	}
}

func s1MACs(a *s1Agent) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, ni := range a.NetworkInterfaces {
		nm := devicemgrcommon.NormalizeMAC(ni.Physical)
		if nm == "" {
			continue
		}
		if _, ok := seen[nm]; ok {
			continue
		}
		seen[nm] = struct{}{}
		out = append(out, ni.Physical)
	}
	return out
}

func s1Attrs(a *s1Agent) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("agentId", a.ID)
	put("agentVersion", a.AgentVersion)
	put("networkStatus", a.NetworkStatus)
	put("scanStatus", a.ScanStatus)
	put("mitigationMode", a.MitigationMode)
	put("machineType", a.MachineType)
	put("domain", a.Domain)
	put("siteName", a.SiteName)
	put("groupName", a.GroupName)
	put("osName", a.OSName)
	fields["infected"] = a.Infected
	fields["activeThreats"] = float64(a.ActiveThreats)
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
	const max = 200
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
