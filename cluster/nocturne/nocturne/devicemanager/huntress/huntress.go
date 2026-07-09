// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package huntress

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
	defaultBaseURL  = "https://api.huntress.io"
	agentsPath      = "/v1/agents"
	huntPageLimit   = 100
	huntHTTPTimeout = 60 * time.Second
	huntMaxRetries  = 5
	huntMaxRespByte = 64 << 20

	offlineAfter = 24 * time.Hour
)

type Manager struct {
	api *apiClient
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetHuntress()
	if spec == nil {
		return nil, errors.Errorf("Not a Huntress DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if spec.ApiKey == "" {
		return nil, errors.Errorf("Empty Huntress apiKey")
	}
	if spec.GetApiSecret().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty Huntress apiSecret")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetApiSecret().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}

	basic := base64.StdEncoding.EncodeToString(
		[]byte(spec.ApiKey + ":" + uenterprisev1.ToSecret(sec).GetValueStr()))

	return &Manager{
		api: &apiClient{
			httpc: &http.Client{Timeout: huntHTTPTimeout},
			base:  base,
			basic: basic,
		},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_HUNTRESS
}

func (m *Manager) Close() error {
	m.api.httpc.CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return nil
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	agents, err := m.api.listAgents(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(agents))
	for _, a := range agents {
		if a == nil {
			continue
		}
		entries = append(entries, toEntry(a))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	httpc *http.Client
	base  string
	basic string
}

type agentsResponse struct {
	Pagination struct {
		NextPage    *int `json:"next_page"`
		CurrentPage int  `json:"current_page"`
		TotalCount  int  `json:"total_count"`
	} `json:"pagination"`
	Agents []*huntressAgent `json:"agents"`
}

func (c *apiClient) listAgents(ctx context.Context) ([]*huntressAgent, error) {
	var out []*huntressAgent
	page := 1
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(huntPageLimit))

		var resp agentsResponse
		if err := c.do(ctx, c.base+agentsPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Agents...)
		if resp.Pagination.NextPage == nil || len(resp.Agents) == 0 {
			break
		}
		page = *resp.Pagination.NextPage
	}
	return out, nil
}

func (c *apiClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < huntMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "Huntress build request")
		}
		req.Header.Set("Authorization", "Basic "+c.basic)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "Huntress request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, huntMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "Huntress decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("Huntress status %d (check API key/secret): %s", resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("Huntress status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("Huntress status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type huntressAgent struct {
	ID             int64    `json:"id"`
	Hostname       string   `json:"hostname"`
	MACAddresses   []string `json:"mac_addresses"`
	SerialNumber   string   `json:"serial_number"`
	OS             string   `json:"os"`
	Platform       string   `json:"platform"`
	Version        string   `json:"version"`
	IPv4Address    string   `json:"ipv4_address"`
	ExternalIP     string   `json:"external_ip"`
	LastSeen       string   `json:"last_seen"`
	DefenderStatus string   `json:"defender_status"`
	EDRVersion     string   `json:"edr_version"`
	OrganizationID int64    `json:"organization_id"`
}

func toEntry(a *huntressAgent) *devicemgrcommon.Entry {
	lastSeen, haveSeen := parseTime(a.LastSeen)
	reporting := haveSeen && time.Since(lastSeen) < offlineAfter
	defenderOK := isDefenderOK(a.DefenderStatus)

	score := int32(100)
	if !defenderOK {
		score -= 40
	}
	if !reporting {
		score -= 20
	}
	if score < 20 {
		score = 20
	}

	p := &corev1.Device_Status_Posture{
		ExternalID: strconv.FormatInt(a.ID, 10),
		RiskLevel:  riskBand(score),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"defenderEnabled": passFail(defenderOK),
			"agentReporting":  passFail(reporting),
		},
		Attrs: huntressAttrs(a),
	}
	if haveSeen {
		p.LastSeenAt = timestamppb.New(lastSeen)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strconv.FormatInt(a.ID, 10),
		Serial:     a.SerialNumber,
		MACs:       a.MACAddresses,
		Posture:    p,
	}
}

func isDefenderOK(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "running", "enabled", "healthy":
		return true
	}
	return false
}

func huntressAttrs(a *huntressAgent) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("huntressAgentId", strconv.FormatInt(a.ID, 10))
	put("os", a.OS)
	put("platform", a.Platform)
	put("agentVersion", a.Version)
	put("edrVersion", a.EDRVersion)
	put("defenderStatus", a.DefenderStatus)
	put("externalIP", a.ExternalIP)
	if a.OrganizationID != 0 {
		fields["organizationId"] = float64(a.OrganizationID)
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
