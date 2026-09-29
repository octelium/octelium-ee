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
	devicesPath    = "/api/v1/devices"
	iruPageSize    = 300
	iruHTTPTimeout = 60 * time.Second
	iruMaxRetries  = 4
	iruMaxRespByte = 64 << 20
)

type Manager struct {
	api *apiClient
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetIru()
	if spec == nil {
		return nil, errors.Errorf("Not an Iru DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	base, err := devicemgrcommon.ParseHTTPSURL(spec.BaseURL, "")
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Iru baseURL")
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

	rc := resty.New().
		SetBaseURL(base).
		SetTimeout(iruHTTPTimeout).
		SetHeader("Accept", "application/json").
		SetAuthToken(uenterprisev1.ToSecret(sec).GetValueStr()).
		SetResponseBodyLimit(iruMaxRespByte).
		SetRetryCount(iruMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		api: &apiClient{rc: rc},
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_IRU
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
	devices, err := m.api.listDevices(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	entries := make([]*devicemgrcommon.Entry, 0, len(devices))
	for _, d := range devices {
		if d == nil || d.IsRemoved || d.DeviceID == "" {
			continue
		}
		entries = append(entries, toEntry(d, now))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type apiClient struct {
	rc *resty.Client
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
		if err := c.get(ctx, devicesPath+"?"+q.Encode(), &page); err != nil {
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

func (c *apiClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "Iru request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("Iru status %d (check API token and Device list permission): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("Iru status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	return nil
}

type iruDevice struct {
	DeviceID       string          `json:"device_id"`
	DeviceName     string          `json:"device_name"`
	SerialNumber   string          `json:"serial_number"`
	Platform       string          `json:"platform"`
	OSVersion      string          `json:"os_version"`
	Model          string          `json:"model"`
	MacAddress     string          `json:"mac_address"`
	LastCheckIn    string          `json:"last_check_in"`
	IsMissing      *bool           `json:"is_missing"`
	IsRemoved      bool            `json:"is_removed"`
	MDMEnabled     *bool           `json:"mdm_enabled"`
	AgentInstalled *bool           `json:"agent_installed"`
	AgentVersion   string          `json:"agent_version"`
	AssetTag       string          `json:"asset_tag"`
	BlueprintID    string          `json:"blueprint_id"`
	User           json.RawMessage `json:"user"`
}

type iruUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (d *iruDevice) getUser() *iruUser {
	if len(d.User) == 0 || d.User[0] != '{' {
		return nil
	}

	ret := &iruUser{}
	if err := json.Unmarshal(d.User, ret); err != nil {
		return nil
	}

	return ret
}

func toEntry(d *iruDevice, now time.Time) *devicemgrcommon.Entry {
	lastCheckIn, haveCheckIn := parseTime(d.LastCheckIn)

	p := &corev1.Device_Status_Posture{
		Enrolled:     devicemgrcommon.SignalFromBool(d.MDMEnabled),
		AgentHealthy: agentHealthySignal(d, lastCheckIn, haveCheckIn, now),
		Attrs:        iruAttrs(d),
	}
	if haveCheckIn {
		p.LastSeenAt = timestamppb.New(lastCheckIn)
	}

	var macs []string
	if d.MacAddress != "" {
		macs = []string{d.MacAddress}
	}

	var emails []string
	if usr := d.getUser(); usr != nil && strings.Contains(usr.Email, "@") {
		emails = []string{usr.Email}
	}

	return &devicemgrcommon.Entry{
		ExternalID:  strings.ToLower(strings.TrimSpace(d.DeviceID)),
		Serial:      d.SerialNumber,
		MACs:        macs,
		OwnerEmails: emails,
		Posture:     p,
	}
}

func agentHealthySignal(d *iruDevice, lastCheckIn time.Time, haveCheckIn bool, now time.Time) corev1.Device_Status_Posture_SignalState {
	switch {
	case d.AgentInstalled == nil || d.IsMissing == nil:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case !*d.AgentInstalled || *d.IsMissing:
		return corev1.Device_Status_Posture_FAIL
	default:
		return devicemgrcommon.RecencySignal(lastCheckIn, haveCheckIn, now)
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
	put("agentVersion", d.AgentVersion)
	put("assetTag", d.AssetTag)
	put("blueprintId", d.BlueprintID)
	if usr := d.getUser(); usr != nil {
		put("userEmail", usr.Email)
		put("userName", usr.Name)
	}
	if d.IsMissing != nil {
		fields["isMissing"] = *d.IsMissing
	}
	if d.MDMEnabled != nil {
		fields["mdmEnabled"] = *d.MDMEnabled
	}
	if d.AgentInstalled != nil {
		fields["agentInstalled"] = *d.AgentInstalled
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
