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
	inventoryPath    = "/api/v1/computers-inventory"
	tokenPath        = "/api/oauth/token"
	jamfPageSize     = 100
	jamfHTTPTimeout  = 60 * time.Second
	tokenHTTPTimeout = 30 * time.Second
	jamfMaxRetries   = 5
	jamfMaxRespByte  = 64 << 20

	probeTimeoutSeconds = 15
	probeMaxOutputBytes = 16384
)

var inventorySections = []string{
	"GENERAL", "HARDWARE", "OPERATING_SYSTEM",
	"USER_AND_LOCATION", "SECURITY", "DISK_ENCRYPTION",
}

var platformUUIDRe = regexp.MustCompile(`(?i)"IOPlatformUUID"\s*=\s*"([0-9a-f-]{36})"`)

type Manager struct {
	jamf   *jamfClient
	filter string
}

var _ devicemgrcommon.Manager = (*Manager)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {
	spec := opts.DeviceManager.Spec.GetJamf()
	if spec == nil {
		return nil, errors.Errorf("Not a Jamf DeviceManager: %s", opts.DeviceManager.Metadata.Name)
	}
	if strings.TrimSpace(spec.BaseURL) == "" || spec.ClientID == "" {
		return nil, errors.Errorf("Empty Jamf baseURL or clientID")
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

	base := strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")
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

	return &Manager{
		jamf:   &jamfClient{httpc: hc, base: base},
		filter: spec.Filter,
	}, nil
}

func (m *Manager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_JAMF_PRO
}

func (m *Manager) Close() error {
	m.jamf.httpc.CloseIdleConnections()
	return nil
}

func (m *Manager) IdentityProbes() []*devicemgrcommon.Probe {
	return []*devicemgrcommon.Probe{
		{
			OSType: corev1.Device_Status_MAC,

			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/usr/sbin/ioreg",
				Args:           []string{"-rd1", "-c", "IOPlatformExpertDevice"},
				TimeoutSeconds: probeTimeoutSeconds,
				MaxOutputBytes: probeMaxOutputBytes,
			},
		},
	}
}

func (m *Manager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	if osType != corev1.Device_Status_MAC {
		return "", nil
	}
	for _, r := range results {
		if r == nil || r.Err != nil || len(r.Output) == 0 {
			continue
		}
		if mm := platformUUIDRe.FindStringSubmatch(string(r.Output)); len(mm) == 2 {
			return strings.ToLower(mm[1]), nil
		}
	}
	return "", nil
}

func (m *Manager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	computers, err := m.jamf.listComputers(ctx, m.filter)
	if err != nil {
		return nil, err
	}
	entries := make([]*devicemgrcommon.Entry, 0, len(computers))
	for _, c := range computers {
		if c == nil {
			continue
		}
		entries = append(entries, toEntry(c))
	}
	return devicemgrcommon.NewFleet(entries), nil
}

type jamfClient struct {
	httpc *http.Client
	base  string
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
		if err := c.do(ctx, c.base+inventoryPath+"?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Results...)
		if len(resp.Results) == 0 || len(out) >= resp.TotalCount {
			break
		}
		page++
	}
	return out, nil
}

func (c *jamfClient) do(ctx context.Context, u string, out any) error {
	var lastErr error
	for attempt := 0; attempt < jamfMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "Jamf build request")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpc.Do(req)
		if err != nil {
			lastErr = errors.Wrap(err, "Jamf request")
			if !sleepBackoff(ctx, attempt, nil) {
				return lastErr
			}
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, jamfMaxRespByte))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return errors.Wrap(err, "Jamf decode")
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return errors.Errorf("Jamf status %d (check API Role has Read Computers): %s",
				resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = errors.Errorf("Jamf status %d: %s", resp.StatusCode, snippet(body))
			if !sleepBackoff(ctx, attempt, resp) {
				return lastErr
			}
		default:
			return errors.Errorf("Jamf status %d: %s", resp.StatusCode, snippet(body))
		}
	}
	return lastErr
}

type computer struct {
	ID              string           `json:"id"`
	UDID            string           `json:"udid"`
	General         *general         `json:"general"`
	Hardware        *hardware        `json:"hardware"`
	OperatingSystem *operatingSystem `json:"operatingSystem"`
	Security        *security        `json:"security"`
	DiskEncryption  *diskEncryption  `json:"diskEncryption"`
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
	Managed bool `json:"managed"`
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

type security struct {
	SipStatus             string `json:"sipStatus"`
	GatekeeperStatus      string `json:"gatekeeperStatus"`
	ActivationLockEnabled bool   `json:"activationLockEnabled"`
	FirewallEnabled       bool   `json:"firewallEnabled"`
	SecureBootLevel       string `json:"secureBootLevel"`
}

type diskEncryption struct {
	BootPartitionEncryptionDetails *bootPartitionEncryptionDetails `json:"bootPartitionEncryptionDetails"`
}

type bootPartitionEncryptionDetails struct {
	PartitionFileVault2State string `json:"partitionFileVault2State"`
}

func toEntry(c *computer) *devicemgrcommon.Entry {
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

	managed := gen.RemoteManagement != nil && gen.RemoteManagement.Managed
	encrypted := fileVaultEncrypted(c.DiskEncryption)
	sip := strings.EqualFold(sec.SipStatus, "ENABLED")
	gatekeeper := gatekeeperOK(sec.GatekeeperStatus)
	secureBootFull := strings.EqualFold(sec.SecureBootLevel, "FULL_SECURITY")

	score := int32(100)
	if !encrypted {
		score -= 30
	}
	if !sec.FirewallEnabled {
		score -= 15
	}
	if !sip {
		score -= 20
	}
	if !gatekeeper {
		score -= 15
	}
	if !managed {
		score -= 20
	}
	if hw.AppleSilicon && !secureBootFull {
		score -= 10
	}
	if score < 0 {
		score = 0
	}

	signals := map[string]corev1.Device_Status_Posture_SignalState{
		"firewall":   passFail(sec.FirewallEnabled),
		"sip":        passFail(sip),
		"gatekeeper": passFail(gatekeeper),
	}
	if hw.AppleSilicon {
		signals["secureBoot"] = passFail(secureBootFull)
	} else {
		signals["secureBoot"] = corev1.Device_Status_Posture_NOT_APPLICABLE
	}

	p := &corev1.Device_Status_Posture{
		ExternalID:     strings.ToLower(c.UDID),
		RiskLevel:      riskBand(score),
		DiskEncryption: passFail(encrypted),
		Compliant:      passFail(managed),
		Signals:        signals,
		Attrs:          jamfAttrs(c, gen, hw, os, sec),
	}
	if t, ok := parseTime(gen.LastContactTime); ok {
		p.LastSeenAt = timestamppb.New(t)
	}

	return &devicemgrcommon.Entry{
		ExternalID: strings.ToLower(c.UDID),
		Serial:     hw.SerialNumber,
		MACs:       jamfMACs(hw),
		Posture:    p,
	}
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

func fileVaultEncrypted(de *diskEncryption) bool {
	if de == nil || de.BootPartitionEncryptionDetails == nil {
		return false
	}
	switch strings.ToUpper(de.BootPartitionEncryptionDetails.PartitionFileVault2State) {
	case "ENCRYPTED", "VALID":
		return true
	}
	return false
}

func gatekeeperOK(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "APP_STORE", "APP_STORE_AND_IDENTIFIED_DEVELOPERS":
		return true
	}
	return false
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
