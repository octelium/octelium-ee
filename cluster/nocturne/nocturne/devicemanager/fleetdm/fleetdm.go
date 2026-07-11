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
	catalogRefHardwareUUID = "octelium/hardware-uuid/v1"

	hostsPath        = "/api/v1/fleet/hosts"
	fleetPageSize    = 100
	fleetHTTPTimeout = 60 * time.Second
	fleetMaxRetries  = 4
	fleetMaxRespByte = 64 << 20
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

	rc := resty.New().
		SetBaseURL(strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")).
		SetTimeout(fleetHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetAuthToken(uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(fleetMaxRespByte).
		SetRetryCount(fleetMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api:    &apiClient{rc: rc},
		teamID: spec.TeamID,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_FLEETDM
}

func (m *Manager) Close() error {
	m.api.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType: corev1.Device_Status_MAC,
		},
		{
			OSType: corev1.Device_Status_WINDOWS,
		},
		{
			OSType: corev1.Device_Status_LINUX,
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
	rc *resty.Client
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

		var resp hostsResponse
		if err := c.get(ctx, hostsPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Hosts...)
		if len(resp.Hosts) < fleetPageSize {
			break
		}
		page++
	}
	return out, nil
}

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "FleetDM request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("FleetDM status %d (check API-only user token): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("FleetDM status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	return nil
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
	Status                string       `json:"status"`
	SeenTime              string       `json:"seen_time"`
	DiskEncryptionEnabled *bool        `json:"disk_encryption_enabled"`
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

	diskEncryption := corev1.Device_Status_Posture_NOT_APPLICABLE
	if h.DiskEncryptionEnabled != nil {
		diskEncryption = passFail(*h.DiskEncryptionEnabled)
		if !*h.DiskEncryptionEnabled && score > 60 {
			score = 60
		}
	}

	signals := map[string]corev1.Device_Status_Posture_SignalState{
		"agentOnline": passFail(strings.EqualFold(h.Status, "online")),
	}
	if h.MDM != nil && strings.TrimSpace(h.MDM.EnrollmentStatus) != "" {
		signals["mdmEnrolled"] = passFail(
			strings.HasPrefix(strings.ToLower(h.MDM.EnrollmentStatus), "on"))
	}

	p := &corev1.Device_Status_Posture{
		RiskLevel:      riskBand(score),
		DiskEncryption: diskEncryption,
		Compliant:      passFail(compliant),
		ThreatFree:     corev1.Device_Status_Posture_NOT_APPLICABLE,
		Signals:        signals,
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
		ExternalID: strings.ToLower(strings.TrimSpace(h.UUID)),
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
	put("status", h.Status)
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
