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
	"net/http"
	"net/url"
	"regexp"
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
	agentsPath    = "/web/api/v2.1/agents"
	s1PageLimit   = 1000
	s1HTTPTimeout = 60 * time.Second
	s1MaxRetries  = 4
	s1MaxRespByte = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 8192
)

var (
	uuidRe    = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	agentIDRe = regexp.MustCompile(`^[0-9a-f-]{16,64}$`)
)

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

	rc := resty.New().
		SetBaseURL(strings.TrimRight(strings.TrimSpace(spec.ManagementURL), "/")).
		SetTimeout(s1HTTPTimeout).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "ApiToken "+uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(s1MaxRespByte).
		SetRetryCount(s1MaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api:        &apiClient{rc: rc},
		siteIDs:    spec.SiteIDs,
		accountIDs: spec.AccountIDs,
		agentQuery: spec.AgentQuery,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_SENTINELONE
}

func (m *Manager) Close() error {
	m.api.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType:           corev1.Device_Status_LINUX,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/opt/sentinelone/bin/sentinelctl",
				Args:           []string{"agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType:           corev1.Device_Status_MAC,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/usr/local/bin/sentinelctl",
				Args:           []string{"agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType:           corev1.Device_Status_MAC,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/Library/Sentinel/sentinel-agent.bundle/Contents/MacOS/sentinelctl",
				Args:           []string{"agent_id"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
		{
			OSType:           corev1.Device_Status_WINDOWS,
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
		if t := strings.ToLower(strings.TrimSpace(s)); agentIDRe.MatchString(t) {
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
		extra, err := url.ParseQuery(m.agentQuery)
		if err != nil {
			return nil, errors.Wrap(err, "Invalid SentinelOne agentQuery")
		}
		for k, vs := range extra {
			for _, v := range vs {
				q.Add(k, v)
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
	rc *resty.Client
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
		if err := c.get(ctx, agentsPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data...)
		if page.Pagination.NextCursor == "" || len(page.Data) == 0 {
			break
		}
		cursor = page.Pagination.NextCursor
	}
	return out, nil
}

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "SentinelOne request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("SentinelOne status %d (check API token and its scope): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("SentinelOne status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	return nil
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
		RiskLevel:      riskBand(score),
		DiskEncryption: passFail(a.EncryptedApplications),
		Compliant:      corev1.Device_Status_Posture_NOT_APPLICABLE,
		ThreatFree:     passFail(!threat),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"agentRunning":  passFail(a.IsActive),
			"agentUpToDate": passFail(a.IsUpToDate),
			"firewall":      passFail(a.FirewallEnabled),
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
