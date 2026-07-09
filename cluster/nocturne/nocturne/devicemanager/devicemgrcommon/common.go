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
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ManagerOpts struct {
	DeviceManager *enterprisev1.DeviceManager
	OcteliumC     octeliumc.ClientInterface
}

type Owner struct {
	Manager Manager
	DM      *enterprisev1.DeviceManager
	Fleet   *Fleet
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
	ExternalID  string
	Serial      string
	MACs        []string
	OwnerEmails []string
	Posture     *corev1.Device_Status_Posture
}

type Fleet struct {
	Entries      []*Entry
	byExternalID map[string]*Entry
	bySerial     map[string]*Entry
	byMAC        map[string]*Entry
}

func NewFleet(entries []*Entry) *Fleet {
	f := &Fleet{
		Entries:      entries,
		byExternalID: map[string]*Entry{},
		bySerial:     map[string]*Entry{},
		byMAC:        map[string]*Entry{},
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		if e.ExternalID != "" {
			f.byExternalID[strings.ToLower(e.ExternalID)] = e
		}
		if s := NormalizeSerial(e.Serial); s != "" {
			if _, ok := f.bySerial[s]; !ok {
				f.bySerial[s] = e
			}
		}
		for _, m := range e.MACs {
			if nm := NormalizeMAC(m); nm != "" {
				if _, ok := f.byMAC[nm]; !ok {
					f.byMAC[nm] = e
				}
			}
		}
	}
	return f
}

type MatchMethod int

const (
	MatchNone MatchMethod = iota
	MatchExternalID
	MatchSerial
	MatchMAC
)

func (f *Fleet) EntryByExternalID(externalID string) *Entry {
	if externalID == "" {
		return nil
	}
	return f.byExternalID[strings.ToLower(externalID)]
}

func (f *Fleet) Match(externalID, serial string, macs []string, allowIdentity bool) (*Entry, MatchMethod) {
	if externalID != "" {
		if e, ok := f.byExternalID[strings.ToLower(externalID)]; ok {
			return e, MatchExternalID
		}
	}
	if !allowIdentity {
		return nil, MatchNone
	}
	if s := NormalizeSerial(serial); s != "" {
		if e, ok := f.bySerial[s]; ok {
			return e, MatchSerial
		}
	}
	for _, m := range macs {
		if nm := NormalizeMAC(m); nm != "" {
			if e, ok := f.byMAC[nm]; ok {
				return e, MatchMAC
			}
		}
	}
	return nil, MatchNone
}

func UpsertPosture(dev *corev1.Device, dm *enterprisev1.DeviceManager, fleet *Fleet, now *timestamppb.Timestamp) bool {
	if dev.Status == nil {
		dev.Status = &corev1.Device_Status{}
	}

	ownerRef := umetav1.GetObjectReference(dm)
	ownedByUs := dev.Status.Posture != nil && RefUIDEqual(dev.Status.Posture.OwnerRef, ownerRef)

	externalID := ""
	if ownedByUs {
		externalID = dev.Status.Posture.ExternalID
	}

	entry, method := fleet.Match(externalID, dev.Status.SerialNumber, dev.Status.MacAddresses, allowIdentity(dm))

	if entry == nil || (method == MatchExternalID && !verifyIdentity(dev, entry)) {
		if ownedByUs {
			dev.Status.Posture = nil
			return true
		}
		return false
	}

	if dev.Status.Posture != nil && !ownedByUs {
		return false
	}

	posture := proto.Clone(entry.Posture).(*corev1.Device_Status_Posture)
	posture.OwnerRef = ownerRef
	posture.ExternalID = entry.ExternalID
	posture.LastSyncAt = now

	if dev.Status.Posture != nil && proto.Equal(dev.Status.Posture, posture) {
		return false
	}
	dev.Status.Posture = posture
	return true
}

func allowIdentity(dm *enterprisev1.DeviceManager) bool {
	l := dm.Spec.GetLinking()
	return l == nil || l.Strategy != enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY
}

func verifyIdentity(dev *corev1.Device, e *Entry) bool {
	if s := NormalizeSerial(dev.Status.SerialNumber); s != "" && s == NormalizeSerial(e.Serial) {
		return true
	}
	devMACs := map[string]struct{}{}
	for _, m := range dev.Status.MacAddresses {
		if nm := NormalizeMAC(m); nm != "" {
			devMACs[nm] = struct{}{}
		}
	}
	for _, m := range e.MACs {
		if nm := NormalizeMAC(m); nm != "" {
			if _, ok := devMACs[nm]; ok {
				return true
			}
		}
	}
	return false
}

func RefUIDEqual(a, b *metav1.ObjectReference) bool {
	return a != nil && b != nil && a.Uid == b.Uid
}

func NormalizeSerial(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "0", "none", "default string", "system serial number",
		"to be filled by o.e.m.", "not specified", "not applicable", "unknown":
		return ""
	}
	return s
}

func NormalizeMAC(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			b.WriteRune(r)
		}
	}
	m := b.String()
	if len(m) != 12 || m == "000000000000" {
		return ""
	}
	return m
}
