// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jamf

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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
	inventoryPath    = "/api/v1/computers-inventory"
	tokenPath        = "/api/oauth/token"
	jamfPageSize     = 500
	jamfHTTPTimeout  = 60 * time.Second
	tokenHTTPTimeout = 30 * time.Second
	jamfMaxRetries   = 4
	jamfMaxRespByte  = 64 << 20
)

var inventorySections = []string{
	"GENERAL", "HARDWARE", "OPERATING_SYSTEM",
	"USER_AND_LOCATION", "SECURITY", "DISK_ENCRYPTION", "GROUP_MEMBERSHIPS",
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Manager struct {
	jamf            *jamfClient
	filter          string
	compliantGroups []string
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetJamf()
	if spec == nil {
		return nil, errors.Errorf("Not a Jamf DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	base, err := devicemgrcommon.ParseHTTPSURL(spec.BaseURL, "")
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Jamf baseURL")
	}
	if spec.ClientID == "" {
		return nil, errors.Errorf("Empty Jamf clientID")
	}
	if spec.GetClientSecret().GetFromSecret() == "" {
		return nil, errors.Errorf("Empty Jamf clientSecret")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.GetClientSecret().GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	conf := &clientcredentials.Config{
		ClientID:     spec.ClientID,
		ClientSecret: uenterprisev1.ToSecret(sec).GetValueStr(),
		TokenURL:     base + tokenPath,
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient,
		&http.Client{Timeout: tokenHTTPTimeout})
	hc := conf.Client(tokenCtx)
	hc.Timeout = jamfHTTPTimeout

	rc := resty.NewWithClient(hc).
		SetBaseURL(base).
		SetHeader("Accept", "application/json").
		SetResponseBodyLimit(jamfMaxRespByte).
		SetRetryCount(jamfMaxRetries).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(retryCondition).
		SetRetryAfter(retryAfter)

	return &Manager{
		jamf:            &jamfClient{rc: rc},
		filter:          spec.Filter,
		compliantGroups: spec.CompliantGroups,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_JAMF_PRO
}

func (m *Manager) Close() error {
	m.jamf.rc.GetClient().CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			ID:      "hardware-uuid-macos",
			OSTypes: []corev1.Device_Status_OSType{corev1.Device_Status_MAC},
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	if osType != corev1.Device_Status_MAC {
		return "", nil
	}
	return devicemgrcommon.ParseAgreedID(results, func(r *devicemgrcommon.ProbeResult) string {
		if id := strings.ToLower(strings.TrimSpace(r.Text)); uuidRe.MatchString(id) {
			return id
		}
		return ""
	})
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	computers, err := m.jamf.listComputers(ctx, m.filter)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	entries := make([]*devicemgrcommon.Entry, 0, len(computers))
	for _, c := range computers {
		if c == nil {
			continue
		}
		entries = append(entries, m.toEntry(c, now))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type jamfClient struct {
	rc *resty.Client
}

type inventoryResponse struct {
	TotalCount int         `json:"totalCount"`
	Results    []*computer `json:"results"`
}

func (c *jamfClient) listComputers(ctx context.Context, filter string) ([]*computer, error) {
	var out []*computer
	page := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("page-size", strconv.Itoa(jamfPageSize))
		q.Set("sort", "id:asc")
		for _, s := range inventorySections {
			q.Add("section", s)
		}
		if strings.TrimSpace(filter) != "" {
			q.Set("filter", filter)
		}

		var resp inventoryResponse
		if err := c.get(ctx, inventoryPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		if resp.Results == nil {
			return nil, errors.New("Invalid Jamf response: missing results")
		}
		out = append(out, resp.Results...)
		if len(resp.Results) == 0 || len(out) >= resp.TotalCount {
			break
		}
		page++
	}
	return out, nil
}

func (c *jamfClient) get(ctx context.Context, u string, out any) error {
	resp, err := c.rc.R().SetContext(ctx).SetResult(out).Get(u)
	if err != nil {
		return errors.Wrap(err, "Jamf request")
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden {
			return errors.Errorf("Jamf status %d (check the API Role has Read Computers): %s",
				resp.StatusCode(), snippet(resp.Body()))
		}
		return errors.Errorf("Jamf status %d: %s", resp.StatusCode(), snippet(resp.Body()))
	}
	if err := devicemgrcommon.CheckJSONContentType(resp.Header().Get("Content-Type")); err != nil {
		return errors.Wrap(err, "Jamf response")
	}
	return nil
}

type computer struct {
	ID               string             `json:"id"`
	UDID             string             `json:"udid"`
	General          *general           `json:"general"`
	Hardware         *hardware          `json:"hardware"`
	OperatingSystem  *operatingSystem   `json:"operatingSystem"`
	UserAndLocation  *userAndLocation   `json:"userAndLocation"`
	Security         *security          `json:"security"`
	DiskEncryption   *diskEncryption    `json:"diskEncryption"`
	GroupMemberships []*groupMembership `json:"groupMemberships"`
}

type groupMembership struct {
	GroupID    string `json:"groupId"`
	GroupName  string `json:"groupName"`
	SmartGroup bool   `json:"smartGroup"`
}

type general struct {
	Name              string            `json:"name"`
	JamfBinaryVersion string            `json:"jamfBinaryVersion"`
	Platform          string            `json:"platform"`
	ManagementId      string            `json:"managementId"`
	LastContactTime   string            `json:"lastContactTime"`
	Supervised        bool              `json:"supervised"`
	UserApprovedMdm   bool              `json:"userApprovedMdm"`
	RemoteManagement  *remoteManagement `json:"remoteManagement"`
	Site              *site             `json:"site"`
}

type remoteManagement struct {
	Managed *bool `json:"managed"`
}

type site struct {
	Name string `json:"name"`
}

type hardware struct {
	Make            string `json:"make"`
	Model           string `json:"model"`
	ModelIdentifier string `json:"modelIdentifier"`
	SerialNumber    string `json:"serialNumber"`
	MacAddress      string `json:"macAddress"`
	AltMacAddress   string `json:"altMacAddress"`
	AppleSilicon    bool   `json:"appleSilicon"`
}

type operatingSystem struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Build   string `json:"build"`
}

type userAndLocation struct {
	Username string `json:"username"`
	Realname string `json:"realname"`
	Email    string `json:"email"`
}

type security struct {
	SipStatus             string `json:"sipStatus"`
	GatekeeperStatus      string `json:"gatekeeperStatus"`
	ActivationLockEnabled bool   `json:"activationLockEnabled"`
	FirewallEnabled       *bool  `json:"firewallEnabled"`
	SecureBootLevel       string `json:"secureBootLevel"`
}

type diskEncryption struct {
	BootPartitionEncryptionDetails *bootPartitionEncryptionDetails `json:"bootPartitionEncryptionDetails"`
}

type bootPartitionEncryptionDetails struct {
	PartitionFileVault2State string `json:"partitionFileVault2State"`
}

func (m *Manager) toEntry(c *computer, now time.Time) *devicemgrcommon.Entry {
	gen := c.General
	if gen == nil {
		gen = &general{}
	}
	hw := c.Hardware
	if hw == nil {
		hw = &hardware{}
	}
	os := c.OperatingSystem
	if os == nil {
		os = &operatingSystem{}
	}
	sec := c.Security
	if sec == nil {
		sec = &security{}
	}

	var managed *bool
	if gen.RemoteManagement != nil {
		managed = gen.RemoteManagement.Managed
	}

	lastContact, haveContact := parseTime(gen.LastContactTime)

	p := &corev1.Device_Status_Posture{
		DiskEncryption: fileVaultSignal(c.DiskEncryption),
		Compliant:      m.complianceSignal(c.GroupMemberships),
		Firewall:       devicemgrcommon.SignalFromBool(sec.FirewallEnabled),
		SecureBoot:     secureBootSignal(sec.SecureBootLevel),
		Enrolled:       devicemgrcommon.SignalFromBool(managed),
		AgentHealthy:   devicemgrcommon.RecencySignal(lastContact, haveContact, now),
		Signals: map[string]corev1.Device_Status_Posture_SignalState{
			devicemgrcommon.SignalKey("jamf", "sip"):        sipSignal(sec.SipStatus),
			devicemgrcommon.SignalKey("jamf", "gatekeeper"): gatekeeperSignal(sec.GatekeeperStatus),
		},
		Attrs: jamfAttrs(c, gen, hw, os, sec),
	}
	if haveContact {
		p.LastSeenAt = timestamppb.New(lastContact)
	}

	return &devicemgrcommon.Entry{
		ExternalID:  strings.ToLower(strings.TrimSpace(c.UDID)),
		Serial:      hw.SerialNumber,
		MACs:        jamfMACs(hw),
		OwnerEmails: jamfOwnerEmails(c.UserAndLocation),
		Posture:     p,
	}
}

func (m *Manager) complianceSignal(groups []*groupMembership) corev1.Device_Status_Posture_SignalState {
	if len(m.compliantGroups) == 0 {
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}

	for _, group := range groups {
		if group == nil || !group.SmartGroup {
			continue
		}
		if slices.Contains(m.compliantGroups, group.GroupName) {
			return corev1.Device_Status_Posture_PASS
		}
	}

	return corev1.Device_Status_Posture_FAIL
}

func jamfOwnerEmails(u *userAndLocation) []string {
	if u == nil {
		return nil
	}
	if email := strings.TrimSpace(u.Email); strings.Contains(email, "@") {
		return []string{email}
	}
	if username := strings.TrimSpace(u.Username); strings.Contains(username, "@") {
		return []string{username}
	}
	return nil
}

func jamfMACs(hw *hardware) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range []string{hw.MacAddress, hw.AltMacAddress} {
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

func jamfAttrs(c *computer, gen *general, hw *hardware, os *operatingSystem, sec *security) *structpb.Struct {
	fields := map[string]interface{}{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			fields[k] = v
		}
	}
	put("jamfId", c.ID)
	put("managementId", gen.ManagementId)
	put("jamfBinaryVersion", gen.JamfBinaryVersion)
	put("platform", gen.Platform)
	put("modelIdentifier", hw.ModelIdentifier)
	put("osName", os.Name)
	put("osVersion", os.Version)
	put("sipStatus", sec.SipStatus)
	put("gatekeeperStatus", sec.GatekeeperStatus)
	put("secureBootLevel", sec.SecureBootLevel)
	if gen.Site != nil {
		put("site", gen.Site.Name)
	}
	fields["supervised"] = gen.Supervised
	fields["userApprovedMdm"] = gen.UserApprovedMdm
	fields["activationLockEnabled"] = sec.ActivationLockEnabled
	fields["appleSilicon"] = hw.AppleSilicon
	if len(fields) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return s
}

func fileVaultSignal(de *diskEncryption) corev1.Device_Status_Posture_SignalState {
	if de == nil || de.BootPartitionEncryptionDetails == nil {
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
	switch strings.ToUpper(strings.TrimSpace(de.BootPartitionEncryptionDetails.PartitionFileVault2State)) {
	case "ENCRYPTED":
		return corev1.Device_Status_Posture_PASS
	case "UNENCRYPTED", "DECRYPTED", "DECRYPTING", "DECRYPTING_PAUSED", "INELIGIBLE":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func secureBootSignal(level string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "FULL_SECURITY":
		return corev1.Device_Status_Posture_PASS
	case "MEDIUM_SECURITY", "NO_SECURITY":
		return corev1.Device_Status_Posture_FAIL
	case "NOT_SUPPORTED":
		return corev1.Device_Status_Posture_NOT_APPLICABLE
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func sipSignal(status string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ENABLED":
		return corev1.Device_Status_Posture_PASS
	case "DISABLED":
		return corev1.Device_Status_Posture_FAIL
	default:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	}
}

func gatekeeperSignal(status string) corev1.Device_Status_Posture_SignalState {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "APP_STORE", "APP_STORE_AND_IDENTIFIED_DEVELOPERS":
		return corev1.Device_Status_Posture_PASS
	case "DISABLED":
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
