// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package intune

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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	managedDevicesPath = "/v1.0/deviceManagement/managedDevices"
	graphPageTop       = 1000
	graphHTTPTimeout   = 60 * time.Second
	tokenHTTPTimeout   = 30 * time.Second
	graphMaxRetries    = 5
	graphMaxRespByte   = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 65536
)

var selectFields = strings.Join([]string{
	"id", "deviceName", "serialNumber", "wiFiMacAddress", "ethernetMacAddress",
	"operatingSystem", "osVersion", "complianceState", "managementState",
	"lastSyncDateTime", "manufacturer", "model", "userPrincipalName",
	"azureADDeviceId", "deviceEnrollmentType", "jailBroken", "isEncrypted",
	"managementAgent", "deviceRegistrationState", "partnerReportedThreatState",
	"userId", "managedDeviceOwnerType", "deviceCategoryDisplayName",
}, ",")

var deviceIDRe = regexp.MustCompile(`(?im)^\s*DeviceId\s*:\s*([0-9a-fA-F-]{36})`)

type Manager struct {
	graph  *graphClient
	filter string
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetMicrosoftIntune()
	if spec == nil {
		return nil, errors.Errorf("Not a MicrosoftIntune DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if spec.TenantID == "" || spec.ClientID == "" {
		return nil, errors.Errorf("Empty Intune tenantID or clientID")
	}
	if spec.GetClientSecret().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty Intune clientSecret")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetClientSecret().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	ep := cloudEndpoints(spec.Cloud)
	conf := &clientcredentials.Config{
		ClientID:     spec.ClientID,
		ClientSecret: uenterprisev1.ToSecret(sec).GetValueStr(),
		TokenURL:     ep.login + "/" + spec.TenantID + "/oauth2/v2.0/token",
		Scopes:       []string{ep.graph + "/.default"},
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient,
		&http.Client{Timeout: tokenHTTPTimeout})
	hc := conf.Client(tokenCtx)
	hc.Timeout = graphHTTPTimeout

	return &Manager{
		graph:  &graphClient{httpc: hc, base: ep.graph},
		filter: spec.Filter,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE
}

func (m *Manager) Close() error {
	m.graph.httpc.CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType: corev1.Device_Status_WINDOWS,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        `C:\Windows\System32\dsregcmd.exe`,
				Args:           []string{"/status"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	if osType != corev1.Device_Status_WINDOWS {
		return "", nil
	}
	for _, r := range results {
		if r == nil || r.Err != nil || len(r.Output) == 0 {
			continue
		}
		if mm := deviceIDRe.FindStringSubmatch(string(r.Output)); len(mm) == 2 {
			return strings.ToLower(mm[1]), nil
		}
	}
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	devices, err := m.graph.listManagedDevices(ctx, m.filter)
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

type endpoints struct{ login, graph string }

func cloudEndpoints(c enterprisev1.DeviceManager_Spec_MicrosoftIntune_Cloud) endpoints {
	switch c {
	case enterprisev1.DeviceManager_Spec_MicrosoftIntune_US_GOV:
		return endpoints{"https://login.microsoftonline.us", "https://graph.microsoft.us"}
	case enterprisev1.DeviceManager_Spec_MicrosoftIntune_CHINA:
		return endpoints{"https://login.chinacloudapi.cn", "https://microsoftgraph.chinacloudapi.cn"}
	default:
		return endpoints{"https://login.microsoftonline.com", "https://graph.microsoft.com"}
	}
}

type graphClient struct {
	httpc *http.Client
	base  string
}

type managedDevicesResponse struct {
	Value    []*managedDevice `json:"value"`
	NextLink string           `json:"@odata.nextLink"`
}

func (g *graphClient) listManagedDevices(ctx context.Context, filter string) ([]*managedDevice, error) {
	u := g.base + managedDevicesPath +
		"?$top=" + strconv.Itoa(graphPageTop) +
		"&$select=" + selectFields
	if strings.TrimSpace(filter) != "" {
		u += "&$filter=" + url.QueryEscape(filter)
	}

	var out []*managedDevice
	for u != "" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var page managedDevicesResponse
		if err := g.do(ctx, u, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Value...)
		u = page.NextLink
	}
	return out, nil
}

func (g *graphClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < graphMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "Intune build request")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := g.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "Intune request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, graphMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "Intune decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("Intune status %d (check DeviceManagementManagedDevices.Read.All consent): %s",
				resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("Intune status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("Intune status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type managedDevice struct {
	ID                         string `json:"id"`
	DeviceName                 string `json:"deviceName"`
	SerialNumber               string `json:"serialNumber"`
	WiFiMacAddress             string `json:"wiFiMacAddress"`
	EthernetMacAddress         string `json:"ethernetMacAddress"`
	OperatingSystem            string `json:"operatingSystem"`
	OSVersion                  string `json:"osVersion"`
	ComplianceState            string `json:"complianceState"`
	ManagementState            string `json:"managementState"`
	LastSyncDateTime           string `json:"lastSyncDateTime"`
	Manufacturer               string `json:"manufacturer"`
	Model                      string `json:"model"`
	UserPrincipalName          string `json:"userPrincipalName"`
	AzureADDeviceID            string `json:"azureADDeviceId"`
	DeviceEnrollmentType       string `json:"deviceEnrollmentType"`
	JailBroken                 string `json:"jailBroken"`
	IsEncrypted                bool   `json:"isEncrypted"`
	ManagementAgent            string `json:"managementAgent"`
	DeviceRegistrationState    string `json:"deviceRegistrationState"`
	PartnerReportedThreatState string `json:"partnerReportedThreatState"`
	UserID                     string `json:"userId"`
	ManagedDeviceOwnerType     string `json:"managedDeviceOwnerType"`
	DeviceCategoryDisplayName  string `json:"deviceCategoryDisplayName"`
}

func toEntry(d *managedDevice) *devicemgrcommon.Entry {
	score := complianceScore(d.ComplianceState)
	if !d.IsEncrypted {
		score -= 20
	}
	if strings.EqualFold(d.JailBroken, "true") {
		score = 0
	}
	if strings.EqualFold(d.PartnerReportedThreatState, "highRisk") && score > 20 {
		score = 20
	}
	if score < 0 {
		score = 0
	}

	p := &corev1.Device_Status_Posture{
		ExternalID:     strings.ToLower(d.AzureADDeviceID),
		RiskLevel:      riskBand(score),
		DiskEncryption: passFail(d.IsEncrypted),
		Compliant:      passFail(strings.EqualFold(d.ComplianceState, "compliant")),
		ThreatFree:     threatSignal(d.PartnerReportedThreatState),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			"notJailbroken": jailSignal(d.JailBroken),
		},
		Attrs: intuneAttrs(d),
	}
	if t, ok := parseTime(d.LastSyncDateTime); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(d.AzureADDeviceID),
		Serial:     d.SerialNumber,
		MACs:       intuneMACs(d),
		Posture:    p,
	}
}

func intuneMACs(d *managedDevice) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range []string{d.WiFiMacAddress, d.EthernetMacAddress} {
		nm := devicemgrcommon.NormalizeMAC(raw)
		if nm == "" {
			continue
		}
		if _, ok := seen[nm]; ok {
			continue
		}
		seen[nm] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func intuneAttrs(d *managedDevice) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("intuneDeviceId", d.ID)
	put("complianceState", d.ComplianceState)
	put("managementState", d.ManagementState)
	put("managementAgent", d.ManagementAgent)
	put("deviceEnrollmentType", d.DeviceEnrollmentType)
	put("deviceRegistrationState", d.DeviceRegistrationState)
	put("partnerReportedThreatState", d.PartnerReportedThreatState)
	put("ownerType", d.ManagedDeviceOwnerType)
	put("userPrincipalName", d.UserPrincipalName)
	fields["isEncrypted"] = d.IsEncrypted
	if len(fields) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return s
}

func complianceScore(state string) int32 {
	switch strings.ToLower(state) {
	case "compliant":
		return 100
	case "configmanager":
		return 80
	case "ingraceperiod":
		return 70
	case "conflict":
		return 40
	case "error":
		return 30
	case "noncompliant":
		return 20
	default:
		return 50
	}
}

func jailSignal(s string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(s) {
	case "true":
		return corev1.Device_Status_Posture_FAIL
	case "false":
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_NOT_APPLICABLE
	}
}

func threatSignal(s string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(s) {
	case "highrisk", "infected":
		return corev1.Device_Status_Posture_FAIL
	case "", "unknown":
		return corev1.Device_Status_Posture_NOT_APPLICABLE
	default:
		return corev1.Device_Status_Posture_PASS
	}
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
