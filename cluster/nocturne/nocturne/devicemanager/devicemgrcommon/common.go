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
)

const DefaultStaleAfter = time.Hour

type ManagerOpts struct {
	DeviceManager *enterprisev1.DeviceManager
	OcteliumC     octeliumc.ClientInterface
}

type ProviderType = enterprisev1.DeviceManager_Status_Type

type ProbeResult struct {
	Output []byte
	Err    error
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

type Probe struct {
	OSType           corev1.Device_Status_OSType
	RequireElevation bool
	RunCommand       *RunCommand
	ReadFile         *ReadFile
	ReadRegistry     *ReadRegistry
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

type Fleet struct {
	entries []*Entry

	byExternalID uniqueIndex
	bySerial     uniqueIndex
	byMAC        uniqueIndex
}

func NewFleet(entries []*Entry) *Fleet {
	f := &Fleet{
		entries:      make([]*Entry, 0, len(entries)),
		byExternalID: newUniqueIndex(),
		bySerial:     newUniqueIndex(),
		byMAC:        newUniqueIndex(),
	}

	for _, raw := range entries {
		entry := cloneEntry(raw)
		if entry == nil {
			continue
		}

		f.entries = append(f.entries, entry)

		if entry.ExternalID != "" {
			f.byExternalID.add(entry.ExternalID, entry)
		}

		if serial := NormalizeSerial(entry.Serial); serial != "" {
			f.bySerial.add(serial, entry)
		}

		for _, mac := range entry.MACs {
			if normalized := NormalizeMAC(mac); normalized != "" {
				f.byMAC.add(normalized, entry)
			}
		}
	}

	return f
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

	posture.LastSyncAt = pbutils.Timestamp(owner.CollectedAt)
	posture.ExpiresAt = pbutils.Timestamp(owner.ExpiresAt)

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

func ApprovalMode(dm *enterprisev1.DeviceManager) enterprisev1.DeviceManager_Spec_Linking_ApprovalMode {
	if dm == nil {
		return enterprisev1.DeviceManager_Spec_Linking_MANUAL
	}

	linking := dm.Spec.GetLinking()
	if linking == nil ||
		linking.GetApprovalMode() == enterprisev1.DeviceManager_Spec_Linking_APPROVAL_MODE_UNSET {
		return enterprisev1.DeviceManager_Spec_Linking_MANUAL
	}

	return linking.GetApprovalMode()
}

func UsesProbe(dm *enterprisev1.DeviceManager) bool {
	return LinkingStrategy(dm) != enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY
}

func UsesIdentity(dm *enterprisev1.DeviceManager) bool {
	return LinkingStrategy(dm) != enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY
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

func NormalizeSerial(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "",
		"0",
		"none",
		"default string",
		"system serial number",
		"to be filled by o.e.m.",
		"not specified",
		"not applicable",
		"unknown":
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
	if len(normalized) != 12 || normalized == "000000000000" {
		return ""
	}

	return normalized
}
