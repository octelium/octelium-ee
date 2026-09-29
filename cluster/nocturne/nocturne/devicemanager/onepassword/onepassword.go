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
	defaultBaseURL = "https://api.kolide.com"
	apiVersion     = "2026-04-07"
	devicesPath    = "/devices"
	peoplePath     = "/people"
	opPageSize     = 100
	opHTTPTimeout  = 60 * time.Second
	opMaxRetries   = 4
	opMaxRespByte  = 64 << 20
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

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

	base, err := devicemgrcommon.ParseHTTPSURL(spec.BaseURL, defaultBaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid OnePassword baseURL")
	}

	rc := resty.New().
		SetBaseURL(base).
		SetTimeout(opHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetHeader("X-Kolide-Api-Version", apiVersion).
		SetAuthToken(uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(opMaxRespByte).
		SetRetryCount(opMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api: &apiClient{rc: rc},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_ONEPASSWORD
}

func (m *Manager) Close() error {
	m.api.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			ID: "hardware-uuid",
			OSTypes: []corev1.Device_Status_OSType{
				corev1.Device_Status_MAC,
				corev1.Device_Status_WINDOWS,
			},
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		},
		{
			ID:               "hardware-uuid-linux",
			OSTypes:          []corev1.Device_Status_OSType{corev1.Device_Status_LINUX},
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
	devices, err := m.api.listDevices(ctx)
	if err != nil {
		return nil, err
	}
	people, err := m.api.listPeople(ctx)
	if err != nil {
		return nil, err
	}

	emails := make(map[string]string, len(people))
	for _, person := range people {
		if person != nil && person.ID != "" {
			emails[person.ID] = person.Email
		}
	}

	now := time.Now()
	entries := make([]*devicemgrcommon.Entry, 0, len(devices))
	for _, d := range devices {
		if d == nil || d.ID == "" {
			continue
		}
		entries = append(entries, toEntry(d, emails, now))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	rc *resty.Client
}

type pagination struct {
	NextCursor string `json:"next_cursor"`
	Count      int    `json:"count"`
}

type devicesResponse struct {
	Data       []*kolideDevice `json:"data"`
	Pagination pagination      `json:"pagination"`
}

type peopleResponse struct {
	Data       []*kolidePerson `json:"data"`
	Pagination pagination      `json:"pagination"`
}

func (c *apiClient) listDevices(ctx context.Context) ([]*kolideDevice, error) {
	var out []*kolideDevice
	cursor := ""
	for {
		var page devicesResponse
		if err := c.getPage(ctx, devicesPath, cursor, &page); err != nil {
			return nil, err
		}
		if page.Data == nil {
			return nil, errors.New("Invalid OnePassword devices response: missing data")
		}
		out = append(out, page.Data...)
		if page.Pagination.NextCursor == "" || page.Pagination.NextCursor == cursor || len(page.Data) == 0 {
			break
		}
		cursor = page.Pagination.NextCursor
	}
	return out, nil
}

func (c *apiClient) listPeople(ctx context.Context) ([]*kolidePerson, error) {
	var out []*kolidePerson
	cursor := ""
	for {
		var page peopleResponse
		if err := c.getPage(ctx, peoplePath, cursor, &page); err != nil {
			return nil, err
		}
		if page.Data == nil {
			return nil, errors.New("Invalid OnePassword people response: missing data")
		}
		out = append(out, page.Data...)
		if page.Pagination.NextCursor == "" || page.Pagination.NextCursor == cursor || len(page.Data) == 0 {
			break
		}
		cursor = page.Pagination.NextCursor
	}
	return out, nil
}

func (c *apiClient) getPage(ctx context.Context, pth, cursor string, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	q := url.Values{}
	q.Set("per_page", strconv.Itoa(opPageSize))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	return c.get(ctx, pth+"?"+q.Encode(), out)
}

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "OnePassword request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("OnePassword status %d (check API token scope): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("OnePassword status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	if err := devicemgrcommon.CheckJSONContentType(resp.Header().Get("Content-Type")); err != nil {
		return errors.Wrap(err, "OnePassword response")
	}
	return nil
}

type kolideDevice struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	LastSeenAt          string      `json:"last_seen_at"`
	OperatingSystem     string      `json:"operating_system"`
	HardwareModel       string      `json:"hardware_model"`
	Serial              string      `json:"serial"`
	HardwareUUID        string      `json:"hardware_uuid"`
	Note                string      `json:"note"`
	AuthState           string      `json:"auth_state"`
	DeviceType          string      `json:"device_type"`
	FormFactor          string      `json:"form_factor"`
	RegisteredOwnerInfo *linkObject `json:"registered_owner_info"`
}

type linkObject struct {
	Identifier string `json:"identifier"`
}

type kolidePerson struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func toEntry(d *kolideDevice, emails map[string]string, now time.Time) *devicemgrcommon.Entry {
	lastSeen, haveSeen := parseTime(d.LastSeenAt)

	p := &corev1.Device_Status_Posture{
		Compliant:    authStateCompliant(d.AuthState),
		AgentHealthy: devicemgrcommon.RecencySignal(lastSeen, haveSeen, now),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			devicemgrcommon.SignalKey("onepassword", "noIssues"): authStateNoIssues(d.AuthState),
		},
		Attrs: kolideAttrs(d),
	}
	if haveSeen {
		p.LastSeenAt = timestamppb.New(lastSeen)
	}

	var aliases []string
	if id := devicemgrcommon.NormalizeID(d.HardwareUUID); id != "" {
		aliases = []string{id}
	}

	var ownerEmails []string
	if d.RegisteredOwnerInfo != nil {
		if email := strings.TrimSpace(emails[d.RegisteredOwnerInfo.Identifier]); strings.Contains(email, "@") {
			ownerEmails = []string{email}
		}
	}

	return &devicemgrcommon.Entry{
		ExternalID:  strings.ToLower(strings.TrimSpace(d.ID)),
		Aliases:     aliases,
		Serial:      d.Serial,
		OwnerEmails: ownerEmails,
		Posture:     p,
	}
}

func authStateCompliant(state string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "good", "notified", "will block":
		return corev1.Device_Status_Posture_PASS
	case "blocked":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func authStateNoIssues(state string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "good":
		return corev1.Device_Status_Posture_PASS
	case "notified", "will block", "blocked":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func kolideAttrs(d *kolideDevice) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("deviceId", d.ID)
	put("name", d.Name)
	put("deviceType", d.DeviceType)
	put("formFactor", d.FormFactor)
	put("operatingSystem", d.OperatingSystem)
	put("hardwareModel", d.HardwareModel)
	put("authState", d.AuthState)
	put("note", d.Note)
	if d.RegisteredOwnerInfo != nil {
		put("registeredOwnerId", d.RegisteredOwnerInfo.Identifier)
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
