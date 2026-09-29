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
	defaultBaseURL  = "https://api.huntress.io"
	agentsPath      = "/v1/agents"
	huntPageLimit   = 500
	huntHTTPTimeout = 60 * time.Second
	huntMaxRetries  = 4
	huntMaxRespByte = 64 << 20
)

type Manager struct {
	api             *apiClient
	organizationIDs []int64
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

	base, err := devicemgrcommon.ParseHTTPSURL(spec.BaseURL, defaultBaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Huntress baseURL")
	}

	rc := resty.New().
		SetBaseURL(base).
		SetTimeout(huntHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetBasicAuth(spec.ApiKey, uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(huntMaxRespByte).
		SetRetryCount(huntMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api:             &apiClient{rc: rc},
		organizationIDs: spec.OrganizationIDs,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_HUNTRESS
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
	var agents []*huntressAgent

	if len(m.organizationIDs) == 0 {
		ret, err := m.api.listAgents(ctx, 0)
		if err != nil {
			return nil, err
		}
		agents = ret
	} else {
		for _, organizationID := range m.organizationIDs {
			ret, err := m.api.listAgents(ctx, organizationID)
			if err != nil {
				return nil, err
			}
			agents = append(agents, ret...)
		}
	}

	now := time.Now()
	entries := make([]*devicemgrcommon.Entry, 0, len(agents))
	for _, a := range agents {
		if a == nil || a.ID == 0 {
			continue
		}
		entries = append(entries, toEntry(a, now))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	rc *resty.Client
}

type agentsResponse struct {
	Pagination struct {
		NextPageToken string `json:"next_page_token"`
	} `json:"pagination"`
	Agents []*huntressAgent `json:"agents"`
}

func (c *apiClient) listAgents(ctx context.Context, organizationID int64) ([]*huntressAgent, error) {
	var out []*huntressAgent
	pageToken := ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("limit", strconv.Itoa(huntPageLimit))
		if organizationID > 0 {
			q.Set("organization_id", strconv.FormatInt(organizationID, 10))
		}
		if pageToken != "" {
			q.Set("page_token", pageToken)
		}

		var resp agentsResponse
		if err := c.get(ctx, agentsPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Agents...)
		if resp.Pagination.NextPageToken == "" ||
			resp.Pagination.NextPageToken == pageToken ||
			len(resp.Agents) == 0 {
			break
		}
		pageToken = resp.Pagination.NextPageToken
	}
	return out, nil
}

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "Huntress request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("Huntress status %d (check API key/secret): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("Huntress status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	return nil
}

type huntressAgent struct {
	ID                     int64    `json:"id"`
	Hostname               string   `json:"hostname"`
	MACAddresses           []string `json:"mac_addresses"`
	SerialNumber           string   `json:"serial_number"`
	OS                     string   `json:"os"`
	Platform               string   `json:"platform"`
	Version                string   `json:"version"`
	IPv4Address            string   `json:"ipv4_address"`
	ExternalIP             string   `json:"external_ip"`
	LastCallbackAt         string   `json:"last_callback_at"`
	DefenderStatus         string   `json:"defender_status"`
	FirewallStatus         string   `json:"firewall_status"`
	TamperProtectionActual *bool    `json:"tamper_protection_actual"`
	EDRVersion             string   `json:"edr_version"`
	OrganizationID         int64    `json:"organization_id"`
}

func toEntry(a *huntressAgent, now time.Time) *devicemgrcommon.Entry {
	lastCallback, haveCallback := parseTime(a.LastCallbackAt)

	signals := map[string]corev1.Device_Status_Posture_SignalState{
		devicemgrcommon.SignalKey("huntress", "defender"): defenderSignal(a.Platform, a.DefenderStatus),
	}
	if a.TamperProtectionActual != nil {
		signals[devicemgrcommon.SignalKey("huntress", "tamperProtection")] =
			devicemgrcommon.SignalFromBool(a.TamperProtectionActual)
	}

	p := &corev1.Device_Status_Posture{
		AgentHealthy: devicemgrcommon.RecencySignal(lastCallback, haveCallback, now),
		NotContained: notContainedSignal(a.FirewallStatus),
		Signals:      signals,
		Attrs:        huntressAttrs(a),
	}
	if haveCallback {
		p.LastSeenAt = timestamppb.New(lastCallback)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strconv.FormatInt(a.ID, 10),
		Serial:     a.SerialNumber,
		MACs:       a.MACAddresses,
		Posture:    p,
	}
}

func notContainedSignal(status string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "enabled", "disabled":
		return corev1.Device_Status_Posture_PASS
	case "pending isolation", "isolated", "pending release":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func defenderSignal(platform, status string) corev1.Device_Status_Posture_SignalState {
	if !strings.EqualFold(strings.TrimSpace(platform), "windows") {
		return corev1.Device_Status_Posture_NOT_APPLICABLE
	}

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case "active", "running", "enabled", "healthy", "protected", "managed":
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_FAIL
	}
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
	put("firewallStatus", a.FirewallStatus)
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
