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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	managedDevicesPath = "/v1.0/deviceManagement/managedDevices"
	graphPageTop       = 500
	graphHTTPTimeout   = 60 * time.Second
	tokenHTTPTimeout   = 30 * time.Second
	graphMaxRetries    = 4
	graphMaxRespByte   = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 65536
)

var selectFields = strings.Join([]string{
	"id", "deviceName", "serialNumber", "wiFiMacAddress", "ethernetMacAddress",
	"operatingSystem", "osVersion", "complianceState", "managementState",
	"lastSyncDateTime", "manufacturer", "model", "userPrincipalName",
	"azureADDeviceId", "deviceEnrollmentType", "jailBroken", "isEncrypted", "emailAddress",
	"managementAgent", "deviceRegistrationState", "partnerReportedThreatState",
	"userId", "managedDeviceOwnerType", "deviceCategoryDisplayName",
}, ",")

var deviceIDRe = regexp.MustCompile(`(?im)^\s*DeviceId\s*:\s*([0-9a-fA-F-]{36})`)

type Manager struct {
	graph                    *graphClient
	filter                   string
	isGracePeriodCompliant   bool
	isConfigManagerCompliant bool
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
		TokenURL:     ep.login + "/" + url.PathEscape(spec.TenantID) + "/oauth2/v2.0/token",
		Scopes:       []string{ep.graph + "/.default"},
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient,
		&http.Client{Timeout: tokenHTTPTimeout})
	hc := conf.Client(tokenCtx)
	hc.Timeout = graphHTTPTimeout

	rc := resty.NewWithClient(hc).
		SetHeader("Accept", "application/json").
		SetResponseBodyLimit(graphMaxRespByte).
		SetRetryCount(graphMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		graph:                    &graphClient{rc: rc, base: ep.graph},
		filter:                   spec.Filter,
		isGracePeriodCompliant:   spec.IsGracePeriodCompliant,
		isConfigManagerCompliant: spec.IsConfigManagerCompliant,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE
}

func (m *Manager) Close() error {
	m.graph.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			ID:      "entra-device-id-windows",
			OSTypes: []corev1.Device_Status_OSType{corev1.Device_Status_WINDOWS},
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
	return devicemgrcommon.ParseAgreedID(results, func(r *devicemgrcommon.ProbeResult) string {
		if mm := deviceIDRe.FindStringSubmatch(r.Text); len(mm) == 2 {
			return devicemgrcommon.NormalizeID(mm[1])
		}
		return ""
	})
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	devices, err := m.graph.listManagedDevices(ctx, m.filter)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(devices))
	for _, d := range devices {
		if d == nil || d.ID == "" {
			continue
		}
		entries = append(entries, m.toEntry(d))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type endpoints struct{ login, graph string }

func cloudEndpoints(c enterprisev1.DeviceManager_Spec_MicrosoftIntune_Cloud) endpoints {
	switch c {
	case enterprisev1.DeviceManager_Spec_MicrosoftIntune_US_GOV:
		return endpoints{"https://login.microsoftonline.us", "https://graph.microsoft.us"}
	case enterprisev1.DeviceManager_Spec_MicrosoftIntune_CHINA:
		return endpoints{"https://login.partner.microsoftonline.cn", "https://microsoftgraph.chinacloudapi.cn"}
	default:
		return endpoints{"https://login.microsoftonline.com", "https://graph.microsoft.com"}
	}
}

type graphClient struct {
	rc   *resty.Client
	base string
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
		if err := g.get(ctx, u, &page); err != nil {
			return nil, err
		}
		if page.Value == nil {
			return nil, errors.New("Invalid Intune response: missing value")
		}
		out = append(out, page.Value...)
		if page.NextLink != "" && !devicemgrcommon.IsSameOrigin(g.base, page.NextLink) {
			return nil, errors.Errorf("Intune nextLink is not in the Microsoft Graph origin: %s", page.NextLink)
		}
		u = page.NextLink
	}
	return out, nil
}

func (g *graphClient) get(ctx context.Context, u string, out any) error {
	resp, err := g.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "Intune request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("Intune status %d (check DeviceManagementManagedDevices.Read.All application consent): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("Intune status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	if err := devicemgrcommon.CheckJSONContentType(resp.Header().Get("Content-Type")); err != nil {
		return errors.Wrap(err, "Intune response")
	}
	return nil
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
	EmailAddress               string `json:"emailAddress"`
	AzureADDeviceID            string `json:"azureADDeviceId"`
	DeviceEnrollmentType       string `json:"deviceEnrollmentType"`
	JailBroken                 string `json:"jailBroken"`
	IsEncrypted                *bool  `json:"isEncrypted"`
	ManagementAgent            string `json:"managementAgent"`
	DeviceRegistrationState    string `json:"deviceRegistrationState"`
	PartnerReportedThreatState string `json:"partnerReportedThreatState"`
	UserID                     string `json:"userId"`
	ManagedDeviceOwnerType     string `json:"managedDeviceOwnerType"`
	DeviceCategoryDisplayName  string `json:"deviceCategoryDisplayName"`
}

func (m *Manager) toEntry(d *managedDevice) *devicemgrcommon.Entry {
	p := &corev1.Device_Status_Posture{
		DiskEncryption:  devicemgrcommon.SignalFromBool(d.IsEncrypted),
		Compliant:       m.complianceSignal(d.ComplianceState),
		ThreatFree:      threatSignal(d.PartnerReportedThreatState),
		Enrolled:        enrolledSignal(d.ManagementState),
		MobileIntegrity: mobileIntegritySignal(d.OperatingSystem, d.JailBroken),
		Attrs:           intuneAttrs(d),
	}
	if t, ok := parseTime(d.LastSyncDateTime); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	var aliases []string
	if id := devicemgrcommon.NormalizeID(d.AzureADDeviceID); id != "" {
		aliases = []string{id}
	}

	return &devicemgrcommon.Entry{
		ExternalID:  strings.ToLower(strings.TrimSpace(d.ID)),
		Aliases:     aliases,
		Serial:      d.SerialNumber,
		MACs:        intuneMACs(d),
		OwnerEmails: intuneOwnerEmails(d),
		Posture:     p,
	}
}

func intuneOwnerEmails(d *managedDevice) []string {
	var ret []string
	for _, email := range []string{d.EmailAddress, d.UserPrincipalName} {
		email = strings.TrimSpace(email)
		if strings.Contains(email, "@") {
			ret = append(ret, email)
		}
	}
	return ret
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
	put("operatingSystem", d.OperatingSystem)
	put("osVersion", d.OSVersion)
	put("azureADDeviceId", d.AzureADDeviceID)
	if d.IsEncrypted != nil {
		fields["isEncrypted"] = *d.IsEncrypted
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

func (m *Manager) complianceSignal(state string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "compliant":
		return corev1.Device_Status_Posture_PASS
	case "ingraceperiod":
		if m.isGracePeriodCompliant {
			return corev1.Device_Status_Posture_PASS
		}
		return corev1.Device_Status_Posture_FAIL
	case "configmanager":
		if m.isConfigManagerCompliant {
			return corev1.Device_Status_Posture_PASS
		}
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case "noncompliant", "conflict", "error":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func enrolledSignal(state string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "managed":
		return corev1.Device_Status_Posture_PASS
	case "", "unknown":
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	default:
		return corev1.Device_Status_Posture_FAIL
	}
}

func mobileIntegritySignal(operatingSystem, jailBroken string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(operatingSystem)) {
	case "ios", "ipados", "android":
	default:
		return corev1.Device_Status_Posture_NOT_APPLICABLE
	}

	switch strings.ToLower(strings.TrimSpace(jailBroken)) {
	case "true":
		return corev1.Device_Status_Posture_FAIL
	case "false":
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func threatSignal(s string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "secured":
		return corev1.Device_Status_Posture_PASS
	case "lowseverity", "mediumseverity", "highseverity", "compromised":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
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
