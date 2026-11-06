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
	hostsPath        = "/api/v1/fleet/hosts"
	fleetPageSize    = 100
	fleetHTTPTimeout = 60 * time.Second
	fleetMaxRetries  = 4
	fleetMaxRespByte = 64 << 20
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var fleetNeverAt = time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)

type Manager struct {
	api    *apiClient
	teamID *uint32
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetFleetDM()
	if spec == nil {
		return nil, errors.Errorf("Not a FleetDM DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	base, err := devicemgrcommon.ParseHTTPSURL(spec.BaseURL, "")
	if err != nil {
		return nil, errors.Wrap(err, "Invalid FleetDM baseURL")
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
		SetBaseURL(base).
		SetTimeout(fleetHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetAuthToken(uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(fleetMaxRespByte).
		SetRetryCount(fleetMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(devicemgrcommon.RetryCondition).
		SetRetryAfter(devicemgrcommon.RetryAfter).
		SetRedirectPolicy(devicemgrcommon.RedirectPolicy)

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
			ID:     "hardware-uuid-macos",
			OSType: corev1.Device_Status_MAC,
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		},
		{
			ID:     "hardware-uuid-windows",
			OSType: corev1.Device_Status_WINDOWS,
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		},
		{
			ID:               "hardware-uuid-linux",
			OSType:           corev1.Device_Status_LINUX,
			RequireElevation: true,
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	return devicemgrcommon.ParseAgreedID(results, func(r *devicemgrcommon.ProbeResult) string {
		if id := strings.ToLower(strings.TrimSpace(r.Text)); uuidRe.MatchString(id) {
			return id
		}
		return ""
	})
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	hosts, err := m.api.listHosts(ctx, m.teamID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	entries := make([]*devicemgrcommon.Entry, 0, len(hosts))
	for _, h := range hosts {
		if h == nil {
			continue
		}
		entries = append(entries, toEntry(h, now))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	rc *resty.Client
}

type hostsResponse struct {
	Hosts []*fleetHost `json:"hosts"`
}

func (c *apiClient) listHosts(ctx context.Context, teamID *uint32) ([]*fleetHost, error) {
	var out []*fleetHost
	page := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(fleetPageSize))
		q.Set("order_key", "id")
		q.Set("order_direction", "asc")
		if teamID != nil {
			q.Set("team_id", strconv.FormatUint(uint64(*teamID), 10))
		}

		var resp hostsResponse
		if err := c.get(ctx, hostsPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		if resp.Hosts == nil {
			return nil, errors.New("Invalid FleetDM response: missing hosts")
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
				resp.StatusCode(), devicemgrcommon.Snippet(resp.Body()))
		}
		return errors.Errorf("FleetDM status %d: %s", resp.StatusCode(), devicemgrcommon.Snippet(resp.Body()))
	}
	if err := devicemgrcommon.CheckJSONContentType(resp.Header().Get("Content-Type")); err != nil {
		return errors.Wrap(err, "FleetDM response")
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
	HardwareModel         string       `json:"hardware_model"`
	PolicyUpdatedAt       string       `json:"policy_updated_at"`
	PublicIP              string       `json:"public_ip"`
	Status                string       `json:"status"`
	SeenTime              string       `json:"seen_time"`
	DiskEncryptionEnabled *bool        `json:"disk_encryption_enabled"`
	Issues                *fleetIssues `json:"issues"`
	MDM                   *fleetMDM    `json:"mdm"`
}

type fleetIssues struct {
	FailingPoliciesCount         *int `json:"failing_policies_count"`
	CriticalVulnerabilitiesCount *int `json:"critical_vulnerabilities_count"`
	TotalIssuesCount             int  `json:"total_issues_count"`
}

type fleetMDM struct {
	EnrollmentStatus string `json:"enrollment_status"`
	Name             string `json:"name"`
}

func toEntry(h *fleetHost, now time.Time) *devicemgrcommon.Entry {
	osType := devicemgrcommon.ParseOSType(h.Platform)

	p := &corev1.Device_Status_Posture{
		DiskEncryption: devicemgrcommon.SignalFromBool(h.DiskEncryptionEnabled),
		Compliant:      fleetCompliant(h),
		AgentHealthy:   fleetAgentHealthy(h, now),
		Os:             devicemgrcommon.ParseOS(osType, h.OSVersion),
		DeviceClass:    devicemgrcommon.ParseDeviceClass(h.HardwareModel, h.OSVersion),
		Attrs:          fleetAttrs(h),
	}
	if h.Issues != nil {
		p.FailingChecks = devicemgrcommon.Count(h.Issues.FailingPoliciesCount)
	}
	if h.MDM != nil {
		p.Enrolled = fleetEnrolled(h.MDM.EnrollmentStatus)
		p.Ownership = fleetOwnership(h.MDM.EnrollmentStatus)
	}
	if t, ok := devicemgrcommon.ParseTime(h.SeenTime); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	var macs []string
	if h.PrimaryMAC != "" {
		macs = []string{h.PrimaryMAC}
	}

	var machineID string
	if osType == corev1.Device_Status_MAC {
		machineID = h.UUID
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(strings.TrimSpace(h.UUID)),
		MachineID:  machineID,
		Serial:     h.HardwareSerial,
		MACs:       macs,
		Hostname:   h.Hostname,
		OSType:     osType,
		Posture:    p,
	}
}

func fleetCompliant(h *fleetHost) corev1.Device_Status_Posture_SignalState {
	switch {
	case h.Issues == nil || h.Issues.FailingPoliciesCount == nil:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case *h.Issues.FailingPoliciesCount > 0:
		return corev1.Device_Status_Posture_FAIL
	case fleetHasPolicyResults(h):
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func fleetHasPolicyResults(h *fleetHost) bool {
	updatedAt, ok := devicemgrcommon.ParseTime(h.PolicyUpdatedAt)
	return ok && updatedAt.After(fleetNeverAt)
}

func fleetAgentHealthy(h *fleetHost, now time.Time) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(h.Status)) {
	case "online":
		return corev1.Device_Status_Posture_PASS
	case "offline":
		if seenAt, ok := devicemgrcommon.ParseTime(h.SeenTime); ok {
			return devicemgrcommon.RecencySignal(seenAt, true, now)
		}
		return corev1.Device_Status_Posture_FAIL
	case "missing":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func fleetEnrolled(status string) corev1.Device_Status_Posture_SignalState {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(status, "on"):
		return corev1.Device_Status_Posture_PASS
	case status == "off", status == "pending":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func fleetOwnership(status string) corev1.Device_Status_Posture_Ownership {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "on (automatic)", "on (manual)":
		return corev1.Device_Status_Posture_COMPANY
	case "on (personal)", "on (manual - personal)":
		return corev1.Device_Status_Posture_PERSONAL
	default:
		return corev1.Device_Status_Posture_OWNERSHIP_UNKNOWN
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
	put("publicIP", h.PublicIP)
	put("status", h.Status)
	if h.MDM != nil {
		put("mdmEnrollmentStatus", h.MDM.EnrollmentStatus)
		put("mdmName", h.MDM.Name)
	}
	if h.Issues != nil && h.Issues.CriticalVulnerabilitiesCount != nil {
		fields["criticalVulnerabilities"] = float64(*h.Issues.CriticalVulnerabilitiesCount)
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
