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
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultStaleAfter        = time.Hour
	DefaultMaxObservationAge = 7 * 24 * time.Hour
	AgentOfflineAfter        = 24 * time.Hour
	MaxClockSkew             = 5 * time.Minute
	maxRetryAfter            = 5 * time.Minute
	maxRedirects             = 10
)

var ErrRedirectNotAllowed = errors.New("Redirects to another origin are not allowed")

var osVersionRe = regexp.MustCompile(`(?:^|[^0-9.])(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

var RedirectPolicy = resty.RedirectPolicyFunc(CheckRedirect)

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
	OSType             corev1.Device_Status_OSType
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
	MachineID  string
	Serial     string
	MACs       []string
	Hostname   string
	OSType     corev1.Device_Status_OSType

	OwnerEmails []string

	Posture *corev1.Device_Status_Posture
}

type MatchState int

const (
	MatchStateNone MatchState = iota
	MatchStateUnique
	MatchStateAmbiguous
)

type MatchResult struct {
	State      MatchState
	Entry      *Entry
	Candidates []*Entry
}

type uniqueIndex struct {
	values    map[string]*Entry
	ambiguous map[string][]*Entry
}

func newUniqueIndex() uniqueIndex {
	return uniqueIndex{
		values:    map[string]*Entry{},
		ambiguous: map[string][]*Entry{},
	}
}

func (i *uniqueIndex) add(key string, entry *Entry) {
	if key == "" || entry == nil {
		return
	}
	if candidates, ok := i.ambiguous[key]; ok {
		if !slices.Contains(candidates, entry) {
			i.ambiguous[key] = append(candidates, entry)
		}
		return
	}
	if previous, ok := i.values[key]; ok {
		if previous != entry {
			delete(i.values, key)
			i.ambiguous[key] = []*Entry{previous, entry}
		}
		return
	}
	i.values[key] = entry
}

func (i *uniqueIndex) get(key string) MatchResult {
	if key == "" {
		return MatchResult{State: MatchStateNone}
	}
	if candidates, ok := i.ambiguous[key]; ok {
		return MatchResult{
			State:      MatchStateAmbiguous,
			Candidates: candidates,
		}
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
			ret.ambiguous[key] = entries
		}
	}

	return ret
}

func getLatestEntry(entries []*Entry) *Entry {
	if len(entries) == 1 {
		return entries[0]
	}

	now := time.Now()

	var latest *Entry
	for _, entry := range entries {
		if !entry.Posture.GetLastSeenAt().IsValid() || IsFutureTime(entry.Posture.LastSeenAt.AsTime(), now) {
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
	byMachineID  uniqueIndex
	bySerial     uniqueIndex
	byMAC        uniqueIndex
	byOwner      map[string][]*Entry

	degradedReason string
}

func NewFleet(entries []*Entry) *Fleet {
	f := &Fleet{
		entries:      make([]*Entry, 0, len(entries)),
		byExternalID: newUniqueIndex(),
		byProbeID:    newUniqueIndex(),
		byMachineID:  newUniqueIndex(),
		byOwner:      map[string][]*Entry{},
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

		f.byMachineID.add(HashMachineID(entry.MachineID), entry)

		for _, email := range entry.OwnerEmails {
			addCandidate(f.byOwner, NormalizeEmail(email), entry)
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
	return f.byExternalID.get(externalID)
}

func (f *Fleet) MatchProbeID(id string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}
	return f.byProbeID.get(NormalizeID(id))
}

func (f *Fleet) MatchMachineID(deviceID string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}
	return f.byMachineID.get(strings.ToLower(strings.TrimSpace(deviceID)))
}

func (f *Fleet) MatchIdentity(serial string, macs []string) MatchResult {
	if f == nil {
		return MatchResult{State: MatchStateNone}
	}

	if normalized := NormalizeSerial(serial); normalized != "" {
		result := f.bySerial.get(normalized)
		switch result.State {
		case MatchStateUnique, MatchStateAmbiguous:
			return result
		}
	}

	var entries []*Entry
	isAmbiguous := false
	for _, raw := range macs {
		normalized := NormalizeMAC(raw)
		if normalized == "" {
			continue
		}

		result := f.byMAC.get(normalized)
		switch result.State {
		case MatchStateAmbiguous:
			isAmbiguous = true
			for _, entry := range result.Candidates {
				if !slices.Contains(entries, entry) {
					entries = append(entries, entry)
				}
			}
		case MatchStateUnique:
			if !slices.Contains(entries, result.Entry) {
				entries = append(entries, result.Entry)
			}
		}
	}

	switch {
	case len(entries) == 0:
		return MatchResult{State: MatchStateNone}
	case len(entries) == 1 && !isAmbiguous:
		return MatchResult{
			State: MatchStateUnique,
			Entry: entries[0],
		}
	default:
		return MatchResult{
			State:      MatchStateAmbiguous,
			Candidates: entries,
		}
	}
}

func (f *Fleet) OwnedEntries(email string) []*Entry {
	if f == nil {
		return nil
	}
	return f.byOwner[NormalizeEmail(email)]
}

type LinkState int

const (
	LinkStateNone LinkState = iota
	LinkStateMatched
	LinkStateAmbiguous
	LinkStateSourcesDisagree
	LinkStateOwnerAmbiguous
	LinkStateOwnerMismatch
)

type LinkRequest struct {
	DeviceID          string
	OSType            corev1.Device_Status_OSType
	SerialNumber      string
	MACs              []string
	Hostname          string
	ProbeID           string
	UserEmail         string
	Now               time.Time
	MaxObservationAge time.Duration
}

type LinkResult struct {
	State      LinkState
	Entry      *Entry
	Confidence corev1.Device_Status_Binding_Confidence
	Evidence   []corev1.Device_Status_Binding_Evidence
}

func (f *Fleet) Link(req *LinkRequest) *LinkResult {
	if f == nil || req == nil {
		return &LinkResult{State: LinkStateNone}
	}

	userEmail := NormalizeEmail(req.UserEmail)

	var entry *Entry
	for _, result := range []MatchResult{
		f.MatchProbeID(req.ProbeID),
		f.MatchMachineID(req.DeviceID),
		f.MatchIdentity(req.SerialNumber, req.MACs),
	} {
		result = narrowToOwner(result, userEmail)

		switch result.State {
		case MatchStateAmbiguous:
			return &LinkResult{State: LinkStateAmbiguous}
		case MatchStateUnique:
			if entry != nil && entry != result.Entry {
				return &LinkResult{State: LinkStateSourcesDisagree}
			}
			entry = result.Entry
		}
	}

	switch {
	case entry != nil:
	case NormalizeID(req.ProbeID) != "" || userEmail == "":
		return &LinkResult{State: LinkStateNone}
	default:
		var state LinkState
		entry, state = f.linkByOwner(req, userEmail)
		if entry == nil {
			return &LinkResult{State: state}
		}
	}

	if len(entry.OwnerEmails) > 0 && !OwnerEmailMatches(userEmail, entry.OwnerEmails) {
		return &LinkResult{
			State: LinkStateOwnerMismatch,
			Entry: entry,
		}
	}

	evidence := entry.Evidence(req)

	return &LinkResult{
		State:      LinkStateMatched,
		Entry:      entry,
		Confidence: GetConfidence(evidence),
		Evidence:   evidence,
	}
}

func (f *Fleet) linkByOwner(req *LinkRequest, userEmail string) (*Entry, LinkState) {
	if req.OSType == corev1.Device_Status_OS_TYPE_UNKNOWN {
		return nil, LinkStateNone
	}

	var candidates []*Entry
	for _, entry := range f.byOwner[userEmail] {
		if entry.OSType == req.OSType && entry.isObservedWithin(req.Now, req.MaxObservationAge) &&
			!entry.IsContradicted(req) {
			candidates = append(candidates, entry)
		}
	}

	if hostname := NormalizeHostname(req.Hostname); len(candidates) > 1 && hostname != "" {
		var matched []*Entry
		for _, entry := range candidates {
			if NormalizeHostname(entry.Hostname) == hostname {
				matched = append(matched, entry)
			}
		}
		if len(matched) == 1 {
			candidates = matched
		}
	}

	switch len(candidates) {
	case 0:
		return nil, LinkStateNone
	case 1:
		return candidates[0], LinkStateMatched
	default:
		return nil, LinkStateOwnerAmbiguous
	}
}

func narrowToOwner(result MatchResult, userEmail string) MatchResult {
	if result.State != MatchStateAmbiguous || userEmail == "" {
		return result
	}

	var owned *Entry
	for _, entry := range result.Candidates {
		if !OwnerEmailMatches(userEmail, entry.OwnerEmails) {
			continue
		}
		if owned != nil {
			return result
		}
		owned = entry
	}

	if owned == nil {
		return result
	}

	return MatchResult{
		State: MatchStateUnique,
		Entry: owned,
	}
}

func (e *Entry) Evidence(req *LinkRequest) []corev1.Device_Status_Binding_Evidence {
	if e == nil || req == nil {
		return nil
	}

	var ret []corev1.Device_Status_Binding_Evidence

	if OwnerEmailMatches(req.UserEmail, e.OwnerEmails) {
		ret = append(ret, corev1.Device_Status_Binding_OWNER_EMAIL)
	}

	if id := NormalizeID(req.ProbeID); id != "" && slices.ContainsFunc(append([]string{e.ExternalID}, e.Aliases...),
		func(itm string) bool {
			return NormalizeID(itm) == id
		}) {
		ret = append(ret, corev1.Device_Status_Binding_PROBE)
	}

	if machineID := HashMachineID(e.MachineID); machineID != "" &&
		machineID == strings.ToLower(strings.TrimSpace(req.DeviceID)) {
		ret = append(ret, corev1.Device_Status_Binding_MACHINE_ID)
	}

	if serial := NormalizeSerial(req.SerialNumber); serial != "" && serial == NormalizeSerial(e.Serial) {
		ret = append(ret, corev1.Device_Status_Binding_SERIAL_NUMBER)
	}

	if slices.ContainsFunc(req.MACs, func(mac string) bool {
		normalized := NormalizeMAC(mac)
		return normalized != "" && slices.ContainsFunc(e.MACs, func(itm string) bool {
			return NormalizeMAC(itm) == normalized
		})
	}) {
		ret = append(ret, corev1.Device_Status_Binding_MAC_ADDRESS)
	}

	if hostname := NormalizeHostname(req.Hostname); hostname != "" && hostname == NormalizeHostname(e.Hostname) {
		ret = append(ret, corev1.Device_Status_Binding_HOSTNAME)
	}

	return ret
}

func (e *Entry) IsContradicted(req *LinkRequest) bool {
	if e == nil || req == nil {
		return false
	}

	serial := NormalizeSerial(req.SerialNumber)
	entrySerial := NormalizeSerial(e.Serial)

	return serial != "" && entrySerial != "" && serial != entrySerial
}

func (e *Entry) isObservedWithin(now time.Time, maxAge time.Duration) bool {
	lastSeenAt := e.Posture.GetLastSeenAt()
	if !lastSeenAt.IsValid() || IsFutureTime(lastSeenAt.AsTime(), now) {
		return false
	}

	return now.Sub(lastSeenAt.AsTime()) < maxAge
}

func HasDeviceEvidence(evidence []corev1.Device_Status_Binding_Evidence) bool {
	return slices.ContainsFunc(evidence, func(itm corev1.Device_Status_Binding_Evidence) bool {
		switch itm {
		case corev1.Device_Status_Binding_PROBE,
			corev1.Device_Status_Binding_MACHINE_ID,
			corev1.Device_Status_Binding_SERIAL_NUMBER,
			corev1.Device_Status_Binding_MAC_ADDRESS:
			return true
		default:
			return false
		}
	})
}

func GetConfidence(evidence []corev1.Device_Status_Binding_Evidence) corev1.Device_Status_Binding_Confidence {
	isOwned := slices.Contains(evidence, corev1.Device_Status_Binding_OWNER_EMAIL)

	switch {
	case isOwned && HasDeviceEvidence(evidence):
		return corev1.Device_Status_Binding_HIGH
	case isOwned:
		return corev1.Device_Status_Binding_MEDIUM
	default:
		return corev1.Device_Status_Binding_LOW
	}
}

func cloneEntry(entry *Entry) *Entry {
	if entry == nil {
		return nil
	}

	out := &Entry{
		ExternalID:  entry.ExternalID,
		Aliases:     append([]string(nil), entry.Aliases...),
		MachineID:   entry.MachineID,
		Serial:      entry.Serial,
		MACs:        append([]string(nil), entry.MACs...),
		Hostname:    entry.Hostname,
		OSType:      entry.OSType,
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
	maxAge := MaxObservationAge(owner.DM)
	observedAt := owner.CollectedAt.Add(-maxAge)
	if posture.LastSeenAt.IsValid() && !IsFutureTime(posture.LastSeenAt.AsTime(), owner.CollectedAt) {
		observedAt = posture.LastSeenAt.AsTime()
	}
	if observationExpiresAt := observedAt.Add(maxAge); observationExpiresAt.Before(expiresAt) {
		expiresAt = observationExpiresAt
	}

	posture.LastSyncAt = pbutils.Timestamp(owner.CollectedAt)
	posture.ExpiresAt = pbutils.Timestamp(expiresAt)

	return posture
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
	if dm != nil {
		if polling := dm.Spec.GetPolling(); polling != nil && polling.GetMaxObservationAge() != nil {
			if value := umetav1.ToDuration(polling.GetMaxObservationAge()).ToGo(); value > 0 {
				return value
			}
		}
	}

	return DefaultMaxObservationAge
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

func NormalizeHostname(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if i := strings.IndexByte(value, '.'); i >= 0 {
		value = value[:i]
	}

	switch value {
	case "localhost":
		return ""
	default:
		return value
	}
}

func HashMachineID(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if NormalizeID(value) == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func ParseOSType(value string) corev1.Device_Status_OSType {
	value = strings.ToLower(strings.TrimSpace(value))

	switch {
	case value == "":
		return corev1.Device_Status_OS_TYPE_UNKNOWN
	case strings.HasPrefix(value, "windows"):
		return corev1.Device_Status_WINDOWS
	case value == "mac", value == "darwin", value == "macmdm",
		strings.HasPrefix(value, "macos"), strings.HasPrefix(value, "mac os"), strings.HasPrefix(value, "os x"):
		return corev1.Device_Status_MAC
	case value == "ios", value == "ipados", value == "iphone", value == "ipad",
		strings.HasPrefix(value, "ios "), strings.HasPrefix(value, "ipados "):
		return corev1.Device_Status_IOS
	case strings.HasPrefix(value, "android"):
		return corev1.Device_Status_ANDROID
	case value == "chrome", strings.HasPrefix(value, "chromeos"), strings.HasPrefix(value, "chrome os"):
		return corev1.Device_Status_CHROMEOS
	case strings.Contains(value, "linux"), slices.Contains([]string{
		"ubuntu", "debian", "rhel", "centos", "fedora", "amzn", "arch",
		"opensuse", "sles", "rocky", "almalinux", "ol", "kali", "manjaro", "gentoo",
	}, strings.Fields(value)[0]):
		return corev1.Device_Status_LINUX
	default:
		return corev1.Device_Status_OS_TYPE_UNKNOWN
	}
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

func IsFutureTime(t, now time.Time) bool {
	return t.After(now.Add(MaxClockSkew))
}

func RecencySignal(lastSeenAt time.Time, ok bool, now time.Time) corev1.Device_Status_Posture_SignalState {
	switch {
	case !ok, IsFutureTime(lastSeenAt, now):
		return corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN
	case now.Sub(lastSeenAt) < AgentOfflineAfter:
		return corev1.Device_Status_Posture_PASS
	default:
		return corev1.Device_Status_Posture_FAIL
	}
}

func ParseOS(osType corev1.Device_Status_OSType, version string) *corev1.Device_Status_Posture_OS {
	version = strings.TrimSpace(version)
	if osType == corev1.Device_Status_OS_TYPE_UNKNOWN && version == "" {
		return nil
	}

	ret := &corev1.Device_Status_Posture_OS{
		Type:    osType,
		Version: version,
	}

	parts := getOSVersionParts(osType, version)
	for i, target := range []**uint32{&ret.Major, &ret.Minor, &ret.Patch, &ret.Revision} {
		if i >= len(parts) || parts[i] == "" {
			break
		}

		value, err := strconv.ParseUint(parts[i], 10, 32)
		if err != nil {
			break
		}

		*target = proto.Uint32(uint32(value))
	}

	return ret
}

func getOSVersionParts(osType corev1.Device_Status_OSType, version string) []string {
	if match := osVersionRe.FindStringSubmatch(version); match != nil {
		return match[1:]
	}

	switch osType {
	case corev1.Device_Status_ANDROID, corev1.Device_Status_IOS:
		if _, err := strconv.ParseUint(version, 10, 32); err == nil {
			return []string{version}
		}
	}

	return nil
}

func ParseDeviceClass(values ...string) corev1.Device_Status_Posture_DeviceClass {
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))

		switch {
		case value == "":
		case strings.Contains(value, "virtual"):
			return corev1.Device_Status_Posture_VIRTUAL_MACHINE
		case strings.Contains(value, "server"), value == "domain controller":
			return corev1.Device_Status_Posture_SERVER
		case strings.Contains(value, "iphone"), value == "phone":
			return corev1.Device_Status_Posture_PHONE
		case strings.Contains(value, "ipad"), value == "tablet":
			return corev1.Device_Status_Posture_TABLET
		case strings.Contains(value, "macbook"), strings.Contains(value, "imac"),
			strings.HasPrefix(value, "mac mini"), strings.HasPrefix(value, "mac studio"),
			strings.HasPrefix(value, "mac pro"),
			slices.Contains([]string{
				"mac", "computer", "desktop", "laptop", "notebook", "workstation",
			}, value):
			return corev1.Device_Status_Posture_WORKSTATION
		}
	}

	return corev1.Device_Status_Posture_DEVICE_CLASS_UNKNOWN
}

func Score(value int32) *uint32 {
	return proto.Uint32(uint32(min(max(value, 0), 100)))
}

func Count(value *int) *uint32 {
	if value == nil || *value < 0 {
		return nil
	}

	return proto.Uint32(uint32(*value))
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

func CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.Errorf("Too many redirects: %d", len(via))
	}

	if len(via) > 0 && (req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host) {
		return errors.Wrapf(ErrRedirectNotAllowed, "%s", req.URL.Redacted())
	}

	return nil
}

func RetryCondition(r *resty.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, ErrRedirectNotAllowed)
	}

	switch code := r.StatusCode(); {
	case code == http.StatusRequestTimeout, code == http.StatusTooManyRequests:
		return true
	default:
		return code >= 500
	}
}

func RetryAfter(c *resty.Client, r *resty.Response) (time.Duration, error) {
	if r == nil {
		return 0, nil
	}

	return ParseRetryAfter(r.Header().Get("Retry-After"), time.Now()), nil
}

func ParseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 || time.Duration(secs)*time.Second > maxRetryAfter {
			return 0
		}
		return time.Duration(secs) * time.Second
	}

	if t, err := http.ParseTime(value); err == nil {
		if d := t.Sub(now); d > 0 && d <= maxRetryAfter {
			return d
		}
	}

	return 0
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
