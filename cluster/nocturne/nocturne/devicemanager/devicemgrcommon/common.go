// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devicemgrcommon

import (
	"context"
	"encoding/hex"
	"mime"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
)

const (
	DefaultStaleAfter = time.Hour
	AgentOfflineAfter = 24 * time.Hour
)

type ManagerOpts struct {
	DeviceManager *enterprisev1.DeviceManager
	OcteliumC     octeliumc.ClientInterface
}

type ProviderType = enterprisev1.DeviceManager_Status_Type

type ProbeResult struct {
	Text  string
	Data  []byte
	Items []string
}

type RunCommand struct {
	Command        string
	Args           []string
	TimeoutSeconds uint32
	MaxOutputBytes uint32
}

type ReadFile struct {
	Path     string
	MaxBytes uint32
}

type ReadRegistry struct {
	Key  string
	Name string
}

type PlatformIdentifier struct {
	Kind corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_Kind
}

type Probe struct {
	ID                 string
	OSTypes            []corev1.Device_Status_OSType
	RequireElevation   bool
	RunCommand         *RunCommand
	ReadFile           *ReadFile
	ReadRegistry       *ReadRegistry
	PlatformIdentifier *PlatformIdentifier
}

type Manager interface {
	Type() ProviderType
	IdentityProbes() []*Probe
	ParseExternalID(osType corev1.Device_Status_OSType, results []*ProbeResult) (string, error)
	Collect(ctx context.Context) (*Fleet, error)
	Close() error
}

type Entry struct {
	ExternalID string
	Aliases    []string
	Serial     string
	MACs       []string

	OwnerEmails []string

	Posture *corev1.Device_Status_Posture
}

type MatchState int

const (
	MatchStateNone MatchState = iota
	MatchStateUnique
	MatchStateAmbiguous
)

type MatchMethod int

const (
	MatchMethodNone MatchMethod = iota
	MatchMethodExternalID
	MatchMethodProbeID
	MatchMethodSerial
	MatchMethodMAC
)

type MatchResult struct {
	State  MatchState
	Method MatchMethod
	Entry  *Entry
}

type uniqueIndex struct {
	values    map[string]*Entry
	ambiguous map[string]struct{}
}

func newUniqueIndex() uniqueIndex {
	return uniqueIndex{
		values:    map[string]*Entry{},
		ambiguous: map[string]struct{}{},
	}
}

func (i *uniqueIndex) add(key string, entry *Entry) {
	if key == "" || entry == nil {
		return
	}
	if _, ok := i.ambiguous[key]; ok {
		return
	}
	if previous, ok := i.values[key]; ok {
		if previous != entry {
			delete(i.values, key)
			i.ambiguous[key] = struct{}{}
		}
		return
	}
	i.values[key] = entry
}

func (i *uniqueIndex) get(key string) MatchResult {
	if key == "" {
		return MatchResult{State: MatchStateNone}
	}
	if _, ok := i.ambiguous[key]; ok {
		return MatchResult{State: MatchStateAmbiguous}
	}
	entry, ok := i.values[key]
	if !ok {
		return MatchResult{State: MatchStateNone}
	}
	return MatchResult{
		State: MatchStateUnique,
		Entry: entry,
	}
}

func newIdentityIndex(candidates map[string][]*Entry) uniqueIndex {
	ret := newUniqueIndex()

	for key, entries := range candidates {
		if entry := getLatestEntry(entries); entry != nil {
			ret.values[key] = entry
		} else {
			ret.ambiguous[key] = struct{}{}
		}
	}

	return ret
}

func getLatestEntry(entries []*Entry) *Entry {
	if len(entries) == 1 {
		return entries[0]
	}

	var latest *Entry
	for _, entry := range entries {
		if !entry.Posture.GetLastSeenAt().IsValid() {
			return nil
		}
		if latest == nil || entry.Posture.LastSeenAt.AsTime().After(latest.Posture.LastSeenAt.AsTime()) {
			latest = entry
		}
	}

	if latest == nil {
		return nil
	}

	latestSeenAt := latest.Posture.LastSeenAt.AsTime()
	for _, entry := range entries {
		if entry != latest && latestSeenAt.Sub(entry.Posture.LastSeenAt.AsTime()) < AgentOfflineAfter {
			return nil
		}
	}

	return latest
}

func addCandidate(candidates map[string][]*Entry, key string, entry *Entry) {
	if key == "" || slices.Contains(candidates[key], entry) {
		return
	}

	candidates[key] = append(candidates[key], entry)
}

type Fleet struct {
	entries []*Entry

	byExternalID uniqueIndex
	byProbeID    uniqueIndex
	bySerial     uniqueIndex
	byMAC        uniqueIndex

	degradedReason string
}

func NewFleet(entries []*Entry) *Fleet {
	f := &Fleet{
		entries:      make([]*Entry, 0, len(entries)),
		byExternalID: newUniqueIndex(),
		byProbeID:    newUniqueIndex(),
	}

	serials := map[string][]*Entry{}
	macs := map[string][]*Entry{}

	for _, raw := range entries {
		entry := cloneEntry(raw)
		if entry == nil {
			continue
		}

		f.entries = append(f.entries, entry)

		if entry.ExternalID != "" {
			f.byExternalID.add(entry.ExternalID, entry)
		}

		for _, id := range append([]string{entry.ExternalID}, entry.Aliases...) {
			if normalized := NormalizeID(id); normalized != "" {
				f.byProbeID.add(normalized, entry)
			}
		}

		addCandidate(serials, NormalizeSerial(entry.Serial), entry)

		for _, mac := range entry.MACs {
			addCandidate(macs, NormalizeMAC(mac), entry)
		}
	}

	f.bySerial = newIdentityIndex(serials)
	f.byMAC = newIdentityIndex(macs)

	return f
}

func (f *Fleet) SetDegraded(reason string) {
	if f == nil {
		return
	}
	f.degradedReason = reason
}

func (f *Fleet) DegradedReason() string {
	if f == nil {
		return ""
	}
	return f.degradedReason
}

func (f *Fleet) Len() int {
	if f == nil {
		return 0
	}
	return len(f.entries)
}

func (f *Fleet) Entries() []*Entry {
	if f == nil {
		return nil
	}
	out := make([]*Entry, 0, len(f.entries))
	for _, entry := range f.entries {
		out = append(out, cloneEntry(entry))
	}
	return out
}

func (f *Fleet) MatchExternalID(externalID string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}
	result := f.byExternalID.get(externalID)
	result.Method = MatchMethodExternalID
	return result
}

func (f *Fleet) MatchProbeID(id string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}
	result := f.byProbeID.get(NormalizeID(id))
	result.Method = MatchMethodProbeID
	return result
}

func (f *Fleet) MatchIdentity(serial string, macs []string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}

	if normalized := NormalizeSerial(serial); normalized != "" {
		result := f.bySerial.get(normalized)
		result.Method = MatchMethodSerial
		switch result.State {
		case MatchStateUnique, MatchStateAmbiguous:
			return result
		}
	}

	entries := map[*Entry]struct{}{}
	for _, raw := range macs {
		normalized := NormalizeMAC(raw)
		if normalized == "" {
			continue
		}

		result := f.byMAC.get(normalized)
		if result.State == MatchStateAmbiguous {
			return MatchResult{
				State:  MatchStateAmbiguous,
				Method: MatchMethodMAC,
			}
		}
		if result.State == MatchStateUnique {
			entries[result.Entry] = struct{}{}
		}
	}

	switch len(entries) {
	case 0:
		return MatchResult{
			State:  MatchStateNone,
			Method: MatchMethodMAC,
		}
	case 1:
		for entry := range entries {
			return MatchResult{
				State:  MatchStateUnique,
				Method: MatchMethodMAC,
				Entry:  entry,
			}
		}
	}

	return MatchResult{
		State:  MatchStateAmbiguous,
		Method: MatchMethodMAC,
	}
}

func cloneEntry(entry *Entry) *Entry {
	if entry == nil {
		return nil
	}

	out := &Entry{
		ExternalID:  entry.ExternalID,
		Aliases:     append([]string(nil), entry.Aliases...),
		Serial:      entry.Serial,
		MACs:        append([]string(nil), entry.MACs...),
		OwnerEmails: append([]string(nil), entry.OwnerEmails...),
	}

	if entry.Posture != nil {
		out.Posture = pbutils.Clone(entry.Posture).(*corev1.Device_Status_Posture)
	}

	return out
}

type Owner struct {
	Manager Manager
	DM      *enterprisev1.DeviceManager
	Fleet   *Fleet

	CollectedAt time.Time
	ExpiresAt   time.Time
}

func NewOwner(
	manager Manager,
	dm *enterprisev1.DeviceManager,
	fleet *Fleet,
	collectedAt time.Time,
	staleAfter time.Duration,
) *Owner {
	if staleAfter <= 0 {
		staleAfter = DefaultStaleAfter
	}

	var clonedDM *enterprisev1.DeviceManager
	if dm != nil {
		clonedDM = pbutils.Clone(dm).(*enterprisev1.DeviceManager)
	}

	return &Owner{
		Manager:     manager,
		DM:          clonedDM,
		Fleet:       fleet,
		CollectedAt: collectedAt,
		ExpiresAt:   collectedAt.Add(staleAfter),
	}
}

func NewPendingOwner(dm *enterprisev1.DeviceManager) *Owner {
	return NewOwner(nil, dm, nil, time.Time{}, 0)
}

func (o *Owner) UID() string {
	if o == nil || o.DM == nil {
		return ""
	}
	return o.DM.GetMetadata().GetUid()
}

func (o *Owner) Fresh(now time.Time) bool {
	if o == nil || o.Fleet == nil || o.CollectedAt.IsZero() || o.ExpiresAt.IsZero() {
		return false
	}
	return now.Before(o.ExpiresAt)
}

func (o *Owner) Name() string {
	if o == nil || o.DM == nil {
		return ""
	}
	return o.DM.GetMetadata().GetName()
}

func (o *Owner) OwnerRef() *metav1.ObjectReference {
	if o == nil || o.DM == nil {
		return nil
	}
	return umetav1.GetObjectReference(o.DM)
}

type Registry struct {
	mu sync.RWMutex
	m  map[string]*Owner
}

func NewRegistry() *Registry {
	return &Registry{
		m: map[string]*Owner{},
	}
}

func (r *Registry) SetOwner(owner *Owner) {
	if owner == nil || owner.UID() == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.m[owner.UID()] = owner
}

func (r *Registry) DeleteOwner(uid string) {
	if uid == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.m, uid)
}

func (r *Registry) GetOwner(uid string) (*Owner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owner, ok := r.m[uid]
	return owner, ok
}

func (r *Registry) ListOwners() []*Owner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*Owner, 0, len(r.m))
	for _, owner := range r.m {
		out = append(out, owner)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].UID() < out[j].UID()
	})

	return out
}

func MaterializePosture(owner *Owner, entry *Entry) *corev1.Device_Status_Posture {
	if owner == nil || entry == nil {
		return nil
	}

	var posture *corev1.Device_Status_Posture
	if entry.Posture != nil {
		posture = pbutils.Clone(entry.Posture).(*corev1.Device_Status_Posture)
	} else {
		posture = &corev1.Device_Status_Posture{}
	}

	expiresAt := owner.ExpiresAt
	if maxAge := MaxObservationAge(owner.DM); maxAge > 0 {
		observedAt := owner.CollectedAt.Add(-maxAge)
		if posture.LastSeenAt.IsValid() {
			observedAt = posture.LastSeenAt.AsTime()
		}
		if observationExpiresAt := observedAt.Add(maxAge); observationExpiresAt.Before(expiresAt) {
			expiresAt = observationExpiresAt
		}
	}

	posture.LastSyncAt = pbutils.Timestamp(owner.CollectedAt)
	posture.ExpiresAt = pbutils.Timestamp(expiresAt)

	return posture
}

func LinkingStrategy(dm *enterprisev1.DeviceManager) enterprisev1.DeviceManager_Spec_Linking_Strategy {
	if dm == nil {
		return enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE
	}

	linking := dm.Spec.GetLinking()
	if linking == nil || linking.GetStrategy() == enterprisev1.DeviceManager_Spec_Linking_STRATEGY_UNSET {
		return enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE
	}

	return linking.GetStrategy()
}

func RequireAgreement(dm *enterprisev1.DeviceManager) bool {
	return dm != nil && dm.Spec.GetLinking().GetRequireAgreement()
}

func RequireOwnerMatch(dm *enterprisev1.DeviceManager) bool {
	return dm != nil && dm.Spec.GetLinking().GetRequireOwnerMatch()
}

func VerificationInterval(dm *enterprisev1.DeviceManager) time.Duration {
	if dm == nil || dm.Spec.GetLinking().GetVerificationInterval() == nil {
		return 0
	}

	return umetav1.ToDuration(dm.Spec.GetLinking().GetVerificationInterval()).ToGo()
}

func MaxObservationAge(dm *enterprisev1.DeviceManager) time.Duration {
	if dm == nil || dm.Spec.GetPolling().GetMaxObservationAge() == nil {
		return 0
	}

	return umetav1.ToDuration(dm.Spec.GetPolling().GetMaxObservationAge()).ToGo()
}

func UsesProbe(dm *enterprisev1.DeviceManager) bool {
	return LinkingStrategy(dm) != enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY
}

func UsesIdentity(dm *enterprisev1.DeviceManager) bool {
	return LinkingStrategy(dm) != enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY
}

func IsDisabled(dm *enterprisev1.DeviceManager) bool {
	return dm != nil && dm.Spec.GetPolling().GetIsDisabled()
}

func GetSecretRef(dm *enterprisev1.DeviceManager) *enterprisev1.DeviceManager_Spec_SecretRef {
	spec := dm.GetSpec()

	switch {
	case spec.GetCrowdStrike() != nil:
		return spec.GetCrowdStrike().ClientSecret
	case spec.GetSentinelOne() != nil:
		return spec.GetSentinelOne().ApiToken
	case spec.GetMicrosoftIntune() != nil:
		return spec.GetMicrosoftIntune().ClientSecret
	case spec.GetJamf() != nil:
		return spec.GetJamf().ClientSecret
	case spec.GetOnePassword() != nil:
		return spec.GetOnePassword().ApiToken
	case spec.GetFleetDM() != nil:
		return spec.GetFleetDM().ApiToken
	case spec.GetHuntress() != nil:
		return spec.GetHuntress().ApiSecret
	case spec.GetIru() != nil:
		return spec.GetIru().ApiToken
	default:
		return nil
	}
}

func StaleAfter(dm *enterprisev1.DeviceManager) time.Duration {
	if dm != nil {
		if polling := dm.Spec.GetPolling(); polling != nil && polling.GetStaleAfter() != nil {
			if value := umetav1.ToDuration(polling.GetStaleAfter()).ToGo(); value > 0 {
				return value
			}
		}
	}

	return DefaultStaleAfter
}

func RefUIDEqual(a, b *metav1.ObjectReference) bool {
	return a != nil && b != nil && a.GetUid() != "" && a.GetUid() == b.GetUid()
}

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func OwnerEmailMatches(userEmail string, ownerEmails []string) bool {
	userEmail = NormalizeEmail(userEmail)
	if userEmail == "" || len(ownerEmails) == 0 {
		return false
	}

	for _, email := range ownerEmails {
		if NormalizeEmail(email) == userEmail {
			return true
		}
	}

	return false
}

func NormalizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.Trim(value, "0-") == "" {
		return ""
	}

	return value
}

func ParseAgreedID(results []*ProbeResult, parse func(result *ProbeResult) string) (string, error) {
	ret := ""

	for _, result := range results {
		if result == nil {
			continue
		}

		id := parse(result)
		if id == "" {
			continue
		}

		if ret != "" && id != ret {
			return "", errors.Errorf("The probe results resolve to different IDs: %s, %s", ret, id)
		}

		ret = id
	}

	return ret, nil
}

func CheckJSONContentType(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.Errorf("Invalid response Content-Type: %q", contentType)
	}

	switch {
	case mediaType == "application/json", mediaType == "text/json", strings.HasSuffix(mediaType, "+json"):
		return nil
	default:
		return errors.Errorf("Unexpected response Content-Type: %q", contentType)
	}
}

func SignalFromBool(value *bool) corev1.Device_Status_Posture_SignalState {
	switch {
	case value == nil:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case *value:
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_FAIL
	}
}

func RecencySignal(lastSeenAt time.Time, ok bool, now time.Time) corev1.Device_Status_Posture_SignalState {
	switch {
	case !ok:
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case now.Sub(lastSeenAt) < AgentOfflineAfter:
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_FAIL
	}
}

func SignalKey(provider, name string) string {
	return "x." + provider + "." + name
}

func ParseHTTPSURL(value, defaultValue string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		value = defaultValue
	}

	u, err := url.Parse(value)
	if err != nil {
		return "", errors.Errorf("Invalid URL: %s", value)
	}

	if u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.Errorf("URL must be a valid HTTPS URL: %s", value)
	}

	return value, nil
}

func IsSameOrigin(base, target string) bool {
	baseURL, err := url.Parse(base)
	if err != nil {
		return false
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return false
	}

	return targetURL.Scheme == baseURL.Scheme && targetURL.Host == baseURL.Host && targetURL.User == nil
}

func NormalizeSerial(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "",
		"none",
		"null",
		"n/a",
		"na",
		"default",
		"default string",
		"system serial number",
		"chassis serial number",
		"serial number",
		"to be filled by o.e.m.",
		"to be filled by oem",
		"oem",
		"o.e.m.",
		"not specified",
		"not applicable",
		"not available",
		"invalid",
		"unknown",
		"0123456789",
		"123456789",
		"1234567890",
		"12345678":
		return ""
	}

	if strings.Trim(value, "0") == "" {
		return ""
	}

	return value
}

func NormalizeMAC(value string) string {
	value = strings.ToLower(value)

	var builder strings.Builder
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			builder.WriteRune(r)
		}
	}

	normalized := builder.String()
	if len(normalized) != 12 {
		return ""
	}

	firstOctet, err := hex.DecodeString(normalized[:2])
	if err != nil || firstOctet[0]&0x03 != 0 || strings.Trim(normalized, "0") == "" {
		return ""
	}

	return normalized
}
