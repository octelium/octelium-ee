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
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type tstProbeManager struct {
	isPlatformProbe bool
}

func (m *tstProbeManager) Type() ProviderType {
	return enterprisev1.DeviceManager_Status_CROWDSTRIKE
}

func (m *tstProbeManager) IdentityProbes() []*Probe {
	ret := &Probe{
		ID:         "probe",
		OSType:     corev1.Device_Status_LINUX,
		RunCommand: &RunCommand{Command: "/opt/agent"},
	}

	if m.isPlatformProbe {
		ret.RunCommand = nil
		ret.PlatformIdentifier = &PlatformIdentifier{
			Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
		}
	}

	return []*Probe{ret}
}

func (m *tstProbeManager) ParseExternalID(osType corev1.Device_Status_OSType, results []*ProbeResult) (string, error) {
	return "", nil
}

func (m *tstProbeManager) Collect(ctx context.Context) (*Fleet, error) {
	return nil, nil
}

func (m *tstProbeManager) Close() error {
	return nil
}

func TestFleet(t *testing.T) {
	fleet := NewFleet([]*Entry{
		{
			ExternalID: "id-1",
			Aliases:    []string{"ALIAS-1", "00000000-0000-0000-0000-000000000000"},
			Serial:     "SERIAL-1",
			MACs:       []string{"A4:BB:CC:DD:EE:01"},
		},
		{
			ExternalID: "id-2",
			Aliases:    []string{"00000000-0000-0000-0000-000000000000"},
			Serial:     "serial-dup",
			MACs:       []string{"a4-bb-cc-dd-ee-02", "a4:bb:cc:dd:ee:ff"},
		},
		{
			ExternalID: "id-3",
			Serial:     "SERIAL-DUP",
			MACs:       []string{"a4bb.ccdd.ee03", "a4:bb:cc:dd:ee:ff"},
		},
		{
			ExternalID: "id-4",
			Aliases:    []string{"shared-alias"},
		},
		{
			ExternalID: "id-5",
			Aliases:    []string{"shared-alias"},
		},
		nil,
	})

	assert.Equal(t, 5, fleet.Len())
	assert.Len(t, fleet.Entries(), 5)

	{
		res := fleet.MatchExternalID("id-1")
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		assert.Equal(t, MatchStateNone, fleet.MatchExternalID("alias-1").State)
		assert.Equal(t, MatchStateNone, fleet.MatchExternalID("").State)
	}

	{
		res := fleet.MatchProbeID(" alias-1 ")
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		res = fleet.MatchProbeID("ID-2")
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-2", res.Entry.ExternalID)

		assert.Equal(t, MatchStateNone, fleet.MatchProbeID("00000000-0000-0000-0000-000000000000").State)
		assert.Equal(t, MatchStateNone, fleet.MatchProbeID("unknown").State)

		res = fleet.MatchProbeID("shared-alias")
		assert.Equal(t, MatchStateAmbiguous, res.State)
		assert.Equal(t, []string{"id-4", "id-5"}, getExternalIDs(res.Candidates))
	}

	{
		res := fleet.MatchIdentity("serial-1", nil)
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		res = fleet.MatchIdentity("serial-dup", nil)
		assert.Equal(t, MatchStateAmbiguous, res.State)
		assert.Equal(t, []string{"id-2", "id-3"}, getExternalIDs(res.Candidates))
		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("serial-dup", []string{"a4:bb:cc:dd:ee:01"}).State)
	}

	{
		res := fleet.MatchIdentity("", []string{"A4-BB-CC-DD-EE-02"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-2", res.Entry.ExternalID)

		res = fleet.MatchIdentity("unknown", []string{"a4:bb:cc:dd:ee:03", "invalid"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-3", res.Entry.ExternalID)

		res = fleet.MatchIdentity("", []string{"a4:bb:cc:dd:ee:ff"})
		assert.Equal(t, MatchStateAmbiguous, res.State)
		assert.Equal(t, []string{"id-2", "id-3"}, getExternalIDs(res.Candidates))

		res = fleet.MatchIdentity("", []string{"a4:bb:cc:dd:ee:01", "a4:bb:cc:dd:ee:02"})
		assert.Equal(t, MatchStateAmbiguous, res.State)
		assert.Equal(t, []string{"id-1", "id-2"}, getExternalIDs(res.Candidates))

		res = fleet.MatchIdentity("", []string{"a4:bb:cc:dd:ee:01", "a4:bb:cc:dd:ee:ff"})
		assert.Equal(t, MatchStateAmbiguous, res.State)
		assert.Equal(t, []string{"id-1", "id-2", "id-3"}, getExternalIDs(res.Candidates))
		assert.Equal(t, MatchStateNone, fleet.MatchIdentity("", []string{"00:00:00:00:00:00"}).State)
		assert.Equal(t, MatchStateNone, fleet.MatchIdentity("", nil).State)
	}

	{
		var nilFleet *Fleet
		assert.Equal(t, 0, nilFleet.Len())
		assert.Nil(t, nilFleet.Entries())
		assert.Equal(t, MatchStateNone, nilFleet.MatchExternalID("id-1").State)
		assert.Equal(t, MatchStateNone, nilFleet.MatchProbeID("id-1").State)
		assert.Equal(t, MatchStateNone, nilFleet.MatchIdentity("serial-1", nil).State)
		assert.Equal(t, MatchStateNone, nilFleet.MatchMachineID("id").State)
		assert.Nil(t, nilFleet.OwnedEntries("user@example.com"))
		assert.Equal(t, LinkStateNone, nilFleet.Link(&LinkRequest{SerialNumber: "serial-1"}).State)
	}

	{
		entries := fleet.Entries()
		entries[0].Aliases[0] = "changed"
		assert.Equal(t, MatchStateUnique, fleet.MatchProbeID("alias-1").State)
	}
}

func getExternalIDs(entries []*Entry) []string {
	var ret []string
	for _, entry := range entries {
		ret = append(ret, entry.ExternalID)
	}
	slices.Sort(ret)
	return ret
}

func TestFleetRecency(t *testing.T) {
	now := time.Now()

	getEntry := func(externalID, serial, mac string, lastSeenAt time.Time) *Entry {
		ret := &Entry{
			ExternalID: externalID,
			Serial:     serial,
			MACs:       []string{mac},
			Posture:    &corev1.Device_Status_Posture{},
		}
		if !lastSeenAt.IsZero() {
			ret.Posture.LastSeenAt = pbutils.Timestamp(lastSeenAt)
		}
		return ret
	}

	fleet := NewFleet([]*Entry{
		getEntry("reinstalled-old", "serial-1", "a4:bb:cc:dd:ee:01", now.Add(-72*time.Hour)),
		getEntry("reinstalled-new", "serial-1", "a4:bb:cc:dd:ee:01", now.Add(-time.Hour)),
		getEntry("reimaged-1", "serial-2", "a4:bb:cc:dd:ee:02", now.Add(-50*time.Hour)),
		getEntry("reimaged-2", "serial-2", "a4:bb:cc:dd:ee:02", now.Add(-30*time.Hour)),
		getEntry("reimaged-3", "serial-2", "a4:bb:cc:dd:ee:02", now.Add(-time.Minute)),
		getEntry("active-1", "serial-3", "a4:bb:cc:dd:ee:03", now.Add(-2*time.Hour)),
		getEntry("active-2", "serial-3", "a4:bb:cc:dd:ee:03", now.Add(-time.Hour)),
		getEntry("unknown-1", "serial-4", "a4:bb:cc:dd:ee:04", time.Time{}),
		getEntry("unknown-2", "serial-4", "a4:bb:cc:dd:ee:04", now),
		getEntry("same-1", "serial-5", "a4:bb:cc:dd:ee:05", now),
		getEntry("same-2", "serial-5", "a4:bb:cc:dd:ee:05", now),
	})

	for _, tc := range []struct {
		serial     string
		mac        string
		state      MatchState
		externalID string
	}{
		{"serial-1", "a4:bb:cc:dd:ee:01", MatchStateUnique, "reinstalled-new"},
		{"serial-2", "a4:bb:cc:dd:ee:02", MatchStateUnique, "reimaged-3"},
		{"serial-3", "a4:bb:cc:dd:ee:03", MatchStateAmbiguous, ""},
		{"serial-4", "a4:bb:cc:dd:ee:04", MatchStateAmbiguous, ""},
		{"serial-5", "a4:bb:cc:dd:ee:05", MatchStateAmbiguous, ""},
	} {
		for _, res := range []MatchResult{
			fleet.MatchIdentity(tc.serial, nil),
			fleet.MatchIdentity("", []string{tc.mac}),
		} {
			assert.Equal(t, tc.state, res.State, tc.serial)
			if tc.externalID != "" {
				assert.Equal(t, tc.externalID, res.Entry.ExternalID)
			}
		}
	}

	assert.Equal(t, MatchStateUnique, fleet.MatchExternalID("reinstalled-old").State)

	{
		entry := getEntry("single", "serial-1", "a4:bb:cc:dd:ee:01", time.Time{})
		entry.MACs = append(entry.MACs, "A4-BB-CC-DD-EE-01")
		res := NewFleet([]*Entry{entry}).MatchIdentity("", []string{"a4:bb:cc:dd:ee:01"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "single", res.Entry.ExternalID)
	}
}

func TestFleetLink(t *testing.T) {
	now := time.Now()
	machineID := "AAAAAAAA-1111-2222-3333-444444444444"

	getEntry := func(externalID string, osType corev1.Device_Status_OSType,
		lastSeenAt time.Time, ownerEmails ...string) *Entry {
		return &Entry{
			ExternalID:  externalID,
			OSType:      osType,
			OwnerEmails: ownerEmails,
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(lastSeenAt),
			},
		}
	}

	aliceMac := getEntry("alice-mac", corev1.Device_Status_MAC, now.Add(-time.Hour), "alice@example.com")
	aliceMac.Serial = "C02ALICE"
	aliceMac.MACs = []string{"a4:bb:cc:dd:ee:01"}
	aliceMac.MachineID = machineID
	aliceMac.Hostname = "Alice-MBP"

	aliceWin := getEntry("alice-win", corev1.Device_Status_WINDOWS, now.Add(-time.Hour),
		"Alice@Example.com", "alice.upn@example.com")
	aliceWin.Hostname = "ALICE-DESKTOP"
	aliceWin2 := getEntry("alice-win2", corev1.Device_Status_WINDOWS, now.Add(-2*time.Hour), "alice@example.com")
	aliceWin2.Hostname = "alice-laptop"

	aliceLinux := getEntry("alice-linux", corev1.Device_Status_LINUX, now.Add(-time.Hour), "alice@example.com")
	aliceLinux.Serial = "LNX-ALICE"
	aliceLinux.Aliases = []string{"agent-alice-linux"}

	bobMac := getEntry("bob-mac", corev1.Device_Status_MAC, now, "bob@example.com")
	bobMac.Serial = "C02BOB"
	bobMac.Aliases = []string{"agent-bob"}

	sharedA := getEntry("shared-a", corev1.Device_Status_MAC, now, "alice@example.com")
	sharedA.MACs = []string{"a4:bb:cc:dd:ee:09"}
	sharedB := getEntry("shared-b", corev1.Device_Status_MAC, now, "bob@example.com")
	sharedB.MACs = []string{"a4:bb:cc:dd:ee:09"}

	unowned := getEntry("unowned", corev1.Device_Status_LINUX, now)
	unowned.Serial = "LNX-UNOWNED"
	unowned.Aliases = []string{"agent-unowned"}

	fleet := NewFleet([]*Entry{
		aliceMac, aliceWin, aliceWin2, aliceLinux, bobMac, sharedA, sharedB, unowned,
		getEntry("alice-linux-stale", corev1.Device_Status_LINUX, now.Add(-30*24*time.Hour), "alice@example.com"),
		getEntry("alice-linux-future", corev1.Device_Status_LINUX, now.Add(time.Hour), "alice@example.com"),
		getEntry("alice-ios", corev1.Device_Status_IOS, now, "alice@example.com"),
	})

	assert.Equal(t, []string{"alice-ios", "alice-linux", "alice-linux-future", "alice-linux-stale",
		"alice-mac", "alice-win", "alice-win2", "shared-a"},
		getExternalIDs(fleet.OwnedEntries(" ALICE@example.com ")))
	assert.Equal(t, []string{"alice-win"}, getExternalIDs(fleet.OwnedEntries("alice.upn@example.com")))
	assert.Empty(t, fleet.OwnedEntries(""))

	{
		res := fleet.MatchMachineID(strings.ToUpper(HashMachineID(machineID)))
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "alice-mac", res.Entry.ExternalID)
		assert.Equal(t, MatchStateNone, fleet.MatchMachineID("").State)
		assert.Equal(t, MatchStateNone, fleet.MatchMachineID(HashMachineID("other")).State)
	}

	for _, tc := range []struct {
		name       string
		req        *LinkRequest
		state      LinkState
		externalID string
		evidence   []corev1.Device_Status_Binding_Evidence
	}{
		{
			name: "every identifier of an owned entry",
			req: &LinkRequest{
				DeviceID:     HashMachineID(strings.ToLower(machineID)),
				OSType:       corev1.Device_Status_MAC,
				SerialNumber: "c02alice",
				MACs:         []string{"A4-BB-CC-DD-EE-01"},
				Hostname:     "alice-mbp.local",
				UserEmail:    "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "alice-mac",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
				corev1.Device_Status_Binding_MACHINE_ID,
				corev1.Device_Status_Binding_SERIAL_NUMBER,
				corev1.Device_Status_Binding_MAC_ADDRESS,
				corev1.Device_Status_Binding_HOSTNAME,
			},
		},
		{
			name: "the machine ID of an owned entry",
			req: &LinkRequest{
				DeviceID:  HashMachineID(machineID),
				OSType:    corev1.Device_Status_MAC,
				UserEmail: "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "alice-mac",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
				corev1.Device_Status_Binding_MACHINE_ID,
			},
		},
		{
			name: "the serial number of another User's entry",
			req: &LinkRequest{
				OSType:       corev1.Device_Status_MAC,
				SerialNumber: "C02BOB",
				UserEmail:    "alice@example.com",
			},
			state:      LinkStateOwnerMismatch,
			externalID: "bob-mac",
		},
		{
			name: "the probe ID of another User's entry",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_MAC,
				ProbeID:   "AGENT-BOB",
				UserEmail: "alice@example.com",
			},
			state:      LinkStateOwnerMismatch,
			externalID: "bob-mac",
		},
		{
			name: "an owned entry for a User without an email",
			req: &LinkRequest{
				OSType:       corev1.Device_Status_MAC,
				SerialNumber: "C02BOB",
			},
			state:      LinkStateOwnerMismatch,
			externalID: "bob-mac",
		},
		{
			name: "an entry without an owner",
			req: &LinkRequest{
				OSType:       corev1.Device_Status_LINUX,
				SerialNumber: "lnx-unowned",
				ProbeID:      "agent-unowned",
				UserEmail:    "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "unowned",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_PROBE,
				corev1.Device_Status_Binding_SERIAL_NUMBER,
			},
		},
		{
			name: "identifiers of different entries",
			req: &LinkRequest{
				OSType:       corev1.Device_Status_LINUX,
				SerialNumber: "LNX-UNOWNED",
				ProbeID:      "agent-alice-linux",
				UserEmail:    "alice@example.com",
			},
			state: LinkStateSourcesDisagree,
		},
		{
			name: "a shared MAC address narrowed down to the User's entry",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_MAC,
				MACs:      []string{"a4:bb:cc:dd:ee:09"},
				UserEmail: "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "shared-a",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
				corev1.Device_Status_Binding_MAC_ADDRESS,
			},
		},
		{
			name: "a shared MAC address of other Users",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_MAC,
				MACs:      []string{"a4:bb:cc:dd:ee:09"},
				UserEmail: "carol@example.com",
			},
			state: LinkStateAmbiguous,
		},
		{
			name: "a shared MAC address without an email",
			req: &LinkRequest{
				OSType: corev1.Device_Status_MAC,
				MACs:   []string{"a4:bb:cc:dd:ee:09"},
			},
			state: LinkStateAmbiguous,
		},
		{
			name: "the only fresh entry of the User with the same OS",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_LINUX,
				Hostname:  "workstation",
				UserEmail: "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "alice-linux",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
			},
		},
		{
			name: "an entry of the User with another serial number",
			req: &LinkRequest{
				OSType:       corev1.Device_Status_LINUX,
				SerialNumber: "LNX-OTHER",
				UserEmail:    "alice@example.com",
			},
			state: LinkStateNone,
		},
		{
			name: "an unknown probe ID",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_LINUX,
				ProbeID:   "agent-unknown",
				UserEmail: "alice@example.com",
			},
			state: LinkStateNone,
		},
		{
			name: "entries of the User narrowed down by the hostname",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_WINDOWS,
				Hostname:  "alice-desktop.corp.example.com",
				UserEmail: "alice@example.com",
			},
			state:      LinkStateMatched,
			externalID: "alice-win",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
				corev1.Device_Status_Binding_HOSTNAME,
			},
		},
		{
			name: "entries of the User that the hostname does not narrow down",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_WINDOWS,
				Hostname:  "other",
				UserEmail: "alice@example.com",
			},
			state: LinkStateOwnerAmbiguous,
		},
		{
			name: "the only entry of another email of the User",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_WINDOWS,
				UserEmail: "alice.upn@example.com",
			},
			state:      LinkStateMatched,
			externalID: "alice-win",
			evidence: []corev1.Device_Status_Binding_Evidence{
				corev1.Device_Status_Binding_OWNER_EMAIL,
			},
		},
		{
			name: "no entry of the User with the same OS",
			req: &LinkRequest{
				OSType:    corev1.Device_Status_ANDROID,
				UserEmail: "alice@example.com",
			},
			state: LinkStateNone,
		},
		{
			name: "an unknown OS",
			req: &LinkRequest{
				UserEmail: "alice@example.com",
			},
			state: LinkStateNone,
		},
		{
			name: "no identifier and no email",
			req: &LinkRequest{
				OSType: corev1.Device_Status_LINUX,
			},
			state: LinkStateNone,
		},
	} {
		tc.req.Now = now
		tc.req.MaxObservationAge = DefaultMaxObservationAge

		res := fleet.Link(tc.req)
		assert.Equal(t, tc.state, res.State, tc.name)
		if tc.externalID == "" {
			assert.Nil(t, res.Entry, tc.name)
			continue
		}
		if assert.NotNil(t, res.Entry, tc.name) {
			assert.Equal(t, tc.externalID, res.Entry.ExternalID, tc.name)
		}
		assert.Equal(t, tc.evidence, res.Evidence, tc.name)
	}

	assert.Equal(t, LinkStateNone, fleet.Link(nil).State)
}

func TestEvidence(t *testing.T) {
	var nilEntry *Entry
	assert.Nil(t, nilEntry.Evidence(&LinkRequest{}))
	assert.False(t, nilEntry.IsContradicted(&LinkRequest{SerialNumber: "serial"}))

	entry := &Entry{
		ExternalID: "id-1",
		Aliases:    []string{"Alias-1"},
		Serial:     "SERIAL-1",
	}
	assert.Nil(t, entry.Evidence(nil))
	assert.Empty(t, entry.Evidence(&LinkRequest{}))
	assert.Equal(t, []corev1.Device_Status_Binding_Evidence{corev1.Device_Status_Binding_PROBE},
		entry.Evidence(&LinkRequest{ProbeID: " ALIAS-1 "}))

	assert.False(t, entry.IsContradicted(nil))
	assert.False(t, entry.IsContradicted(&LinkRequest{}))
	assert.False(t, entry.IsContradicted(&LinkRequest{SerialNumber: "serial-1"}))
	assert.False(t, entry.IsContradicted(&LinkRequest{SerialNumber: "To Be Filled By O.E.M."}))
	assert.True(t, entry.IsContradicted(&LinkRequest{SerialNumber: "serial-2"}))
	assert.False(t, (&Entry{Serial: "none"}).IsContradicted(&LinkRequest{SerialNumber: "serial-2"}))

	provider := &tstProbeManager{}
	platform := &tstProbeManager{isPlatformProbe: true}

	for _, tc := range []struct {
		evidence  []corev1.Device_Status_Binding_Evidence
		isDevice  bool
		isOwned   bool
		assurance corev1.Device_Status_Binding_Assurance
	}{
		{nil, false, false, corev1.Device_Status_Binding_ASSURANCE_UNKNOWN},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_HOSTNAME,
		}, false, false, corev1.Device_Status_Binding_ASSURANCE_UNKNOWN},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_PROBE,
		}, true, false, corev1.Device_Status_Binding_PROVIDER_ID},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, true, false, corev1.Device_Status_Binding_IDENTIFIER},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
		}, false, true, corev1.Device_Status_Binding_OWNER_ONLY},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_HOSTNAME,
		}, false, true, corev1.Device_Status_Binding_OWNER_ONLY},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_MAC_ADDRESS,
		}, true, true, corev1.Device_Status_Binding_IDENTIFIER},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_MACHINE_ID,
		}, true, true, corev1.Device_Status_Binding_IDENTIFIER},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_SERIAL_NUMBER,
			corev1.Device_Status_Binding_OWNER_EMAIL,
		}, true, true, corev1.Device_Status_Binding_IDENTIFIER},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_PROBE,
		}, true, true, corev1.Device_Status_Binding_PROVIDER_ID},
		{[]corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_PROBE,
			corev1.Device_Status_Binding_VENDOR_SIGNED,
		}, true, false, corev1.Device_Status_Binding_SIGNED},
	} {
		assert.Equal(t, tc.isDevice, HasDeviceEvidence(tc.evidence), "%v", tc.evidence)
		assert.Equal(t, tc.isOwned, IsOwnerMatched(tc.evidence), "%v", tc.evidence)
		assert.Equal(t, tc.assurance,
			GetAssurance(tc.evidence, provider, corev1.Device_Status_LINUX), "%v", tc.evidence)
	}

	{
		evidence := []corev1.Device_Status_Binding_Evidence{corev1.Device_Status_Binding_PROBE}

		assert.Equal(t, corev1.Device_Status_Binding_IDENTIFIER,
			GetAssurance(evidence, platform, corev1.Device_Status_LINUX),
			"a Probe of a hardware identifier is an identifier")
		assert.Equal(t, corev1.Device_Status_Binding_IDENTIFIER,
			GetAssurance(evidence, provider, corev1.Device_Status_WINDOWS),
			"a Probe of another OS does not apply")
		assert.Equal(t, corev1.Device_Status_Binding_IDENTIFIER,
			GetAssurance(evidence, nil, corev1.Device_Status_LINUX))
	}
}

func TestHasProbe(t *testing.T) {
	provider := &tstProbeManager{}
	platform := &tstProbeManager{isPlatformProbe: true}

	assert.False(t, HasProbe(nil, corev1.Device_Status_LINUX))
	assert.False(t, HasProviderProbe(nil, corev1.Device_Status_LINUX))

	assert.True(t, HasProbe(provider, corev1.Device_Status_LINUX))
	assert.True(t, HasProviderProbe(provider, corev1.Device_Status_LINUX))
	assert.False(t, HasProbe(provider, corev1.Device_Status_MAC))
	assert.False(t, HasProviderProbe(provider, corev1.Device_Status_MAC))

	assert.True(t, HasProbe(platform, corev1.Device_Status_LINUX))
	assert.False(t, HasProviderProbe(platform, corev1.Device_Status_LINUX))
}

func TestFleetDegraded(t *testing.T) {
	var nilFleet *Fleet
	nilFleet.SetDegraded("reason")
	assert.Empty(t, nilFleet.DegradedReason())

	fleet := NewFleet(nil)
	assert.Empty(t, fleet.DegradedReason())
	fleet.SetDegraded("reason")
	assert.Equal(t, "reason", fleet.DegradedReason())
}

func TestParseAgreedID(t *testing.T) {
	parse := func(r *ProbeResult) string {
		return NormalizeID(r.Text)
	}

	{
		id, err := ParseAgreedID(nil, parse)
		assert.Nil(t, err)
		assert.Empty(t, id)
	}

	{
		id, err := ParseAgreedID([]*ProbeResult{nil, {Text: ""}, {Text: " ID-1 "}, {Text: "id-1"}}, parse)
		assert.Nil(t, err)
		assert.Equal(t, "id-1", id)
	}

	{
		_, err := ParseAgreedID([]*ProbeResult{{Text: "id-1"}, {Text: "id-2"}}, parse)
		assert.NotNil(t, err)
	}
}

func TestCheckJSONContentType(t *testing.T) {
	for _, arg := range []string{
		"application/json",
		"application/json; charset=utf-8",
		"Application/JSON",
		"application/json;odata.metadata=minimal;odata.streaming=true;IEEE754Compatible=false;charset=utf-8",
		"application/vnd.api+json",
		"application/problem+json",
		"text/json",
	} {
		assert.Nil(t, CheckJSONContentType(arg), arg)
	}

	for _, arg := range []string{
		"",
		"text/html",
		"text/html; charset=utf-8",
		"application/xml",
		"text/plain",
		"application/jsonx",
		";;",
	} {
		assert.NotNil(t, CheckJSONContentType(arg), arg)
	}
}

func TestDeviceManagerSpecHelpers(t *testing.T) {
	assert.False(t, IsDisabled(nil))
	assert.Nil(t, GetSecretRef(nil))
	assert.Nil(t, GetSecretRef(&enterprisev1.DeviceManager{Spec: &enterprisev1.DeviceManager_Spec{}}))

	ref := &enterprisev1.DeviceManager_Spec_SecretRef{
		Type: &enterprisev1.DeviceManager_Spec_SecretRef_FromSecret{
			FromSecret: "secret",
		},
	}

	for _, spec := range []*enterprisev1.DeviceManager_Spec{
		{Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
			CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{ClientSecret: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_SentinelOne_{
			SentinelOne: &enterprisev1.DeviceManager_Spec_SentinelOne{ApiToken: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
			MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{ClientSecret: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_Jamf_{
			Jamf: &enterprisev1.DeviceManager_Spec_Jamf{ClientSecret: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
			OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{ApiToken: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
			FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{ApiToken: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_Huntress_{
			Huntress: &enterprisev1.DeviceManager_Spec_Huntress{ApiSecret: ref},
		}},
		{Type: &enterprisev1.DeviceManager_Spec_Iru_{
			Iru: &enterprisev1.DeviceManager_Spec_Iru{ApiToken: ref},
		}},
	} {
		dm := &enterprisev1.DeviceManager{Spec: spec}
		assert.Equal(t, "secret", GetSecretRef(dm).GetFromSecret())
		assert.False(t, IsDisabled(dm))

		spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{IsDisabled: true}
		assert.True(t, IsDisabled(dm))
	}
}

func TestNormalize(t *testing.T) {
	assert.Equal(t, "abc-def", NormalizeID(" ABC-DEF "))
	assert.Equal(t, "", NormalizeID("00000000-0000-0000-0000-000000000000"))
	assert.Equal(t, "", NormalizeID("0"))
	assert.Equal(t, "", NormalizeID(" "))

	assert.Equal(t, "c02abc", NormalizeSerial(" C02ABC "))
	assert.Equal(t, "0a1b2c", NormalizeSerial("0A1B2C"))
	for _, arg := range []string{"", "0", "0000000000", "None", "Default String", "To Be Filled By O.E.M.",
		"unknown", "N/A", "Chassis Serial Number", "System Serial Number", "0123456789", "Invalid",
		"Not Available", "OEM"} {
		assert.Equal(t, "", NormalizeSerial(arg), arg)
	}

	assert.Equal(t, "a4bbccddeeff", NormalizeMAC("A4:BB:CC:DD:EE:FF"))
	assert.Equal(t, "a4bbccddeeff", NormalizeMAC("a4bb.ccdd.eeff"))
	assert.Equal(t, "001b638445e6", NormalizeMAC("00-1B-63-84-45-E6"))
	for _, arg := range []string{
		"00:00:00:00:00:00",
		"a4:bb:cc",
		"ff:ff:ff:ff:ff:ff",
		"01:00:5e:00:00:fb",
		"33:33:00:00:00:01",
		"02:42:ac:11:00:02",
		"aa:bb:cc:dd:ee:ff",
		"de:ad:be:ef:00:01",
		"invalid",
	} {
		assert.Equal(t, "", NormalizeMAC(arg), arg)
	}

	assert.Equal(t, "alice-mbp", NormalizeHostname(" Alice-MBP.local "))
	assert.Equal(t, "desktop-1", NormalizeHostname("DESKTOP-1"))
	assert.Equal(t, "", NormalizeHostname("localhost.localdomain"))
	assert.Equal(t, "", NormalizeHostname(" "))

	{
		sum := sha256.Sum256([]byte("AAAAAAAA-1111-2222-3333-444444444444"))
		hash := fmt.Sprintf("%x", sum)
		assert.Equal(t, hash, HashMachineID("AAAAAAAA-1111-2222-3333-444444444444"))
		assert.Equal(t, hash, HashMachineID(" aaaaaaaa-1111-2222-3333-444444444444 "))
		assert.Equal(t, "", HashMachineID(""))
		assert.Equal(t, "", HashMachineID("00000000-0000-0000-0000-000000000000"))
	}

	for _, tc := range []struct {
		value  string
		osType corev1.Device_Status_OSType
	}{
		{"", corev1.Device_Status_OS_TYPE_UNKNOWN},
		{"Windows", corev1.Device_Status_WINDOWS},
		{"Windows 11 Pro", corev1.Device_Status_WINDOWS},
		{"Mac", corev1.Device_Status_MAC},
		{"darwin", corev1.Device_Status_MAC},
		{"macOS", corev1.Device_Status_MAC},
		{"macOS 14.2", corev1.Device_Status_MAC},
		{"Mac OS X", corev1.Device_Status_MAC},
		{"iOS", corev1.Device_Status_IOS},
		{"iPadOS", corev1.Device_Status_IOS},
		{"iPhone", corev1.Device_Status_IOS},
		{"iOS 17.1", corev1.Device_Status_IOS},
		{"Android", corev1.Device_Status_ANDROID},
		{"AndroidEnterprise", corev1.Device_Status_ANDROID},
		{"chrome", corev1.Device_Status_CHROMEOS},
		{"ChromeOS", corev1.Device_Status_CHROMEOS},
		{"linux", corev1.Device_Status_LINUX},
		{"ubuntu", corev1.Device_Status_LINUX},
		{"Ubuntu 22.04.3 LTS", corev1.Device_Status_LINUX},
		{"rhel", corev1.Device_Status_LINUX},
		{"Red Hat Enterprise Linux", corev1.Device_Status_LINUX},
		{"AppleTV", corev1.Device_Status_OS_TYPE_UNKNOWN},
		{"unknown", corev1.Device_Status_OS_TYPE_UNKNOWN},
	} {
		assert.Equal(t, tc.osType, ParseOSType(tc.value), tc.value)
	}

	assert.Equal(t, "user@example.com", NormalizeEmail(" User@Example.com "))
	assert.True(t, OwnerEmailMatches("USER@example.com", []string{"other@example.com", "user@EXAMPLE.com"}))
	assert.False(t, OwnerEmailMatches("", []string{""}))
	assert.False(t, OwnerEmailMatches("user@example.com", nil))
	assert.False(t, OwnerEmailMatches("user@example.com", []string{"other@example.com"}))
}

func TestSignals(t *testing.T) {
	yes := true
	no := false

	assert.Equal(t, corev1.Device_Status_Posture_PASS, SignalFromBool(&yes))
	assert.Equal(t, corev1.Device_Status_Posture_FAIL, SignalFromBool(&no))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, SignalFromBool(nil))

	now := time.Now()
	assert.Equal(t, corev1.Device_Status_Posture_PASS, RecencySignal(now.Add(-time.Hour), true, now))
	assert.Equal(t, corev1.Device_Status_Posture_FAIL, RecencySignal(now.Add(-AgentOfflineAfter), true, now))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, RecencySignal(time.Time{}, false, now))
	assert.Equal(t, corev1.Device_Status_Posture_PASS, RecencySignal(now.Add(MaxClockSkew), true, now))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN,
		RecencySignal(now.Add(MaxClockSkew+time.Second), true, now))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN,
		RecencySignal(now.Add(365*24*time.Hour), true, now))
}

func TestParseOS(t *testing.T) {
	type version struct {
		major, minor, patch, revision *uint32
	}

	num := proto.Uint32

	for _, tc := range []struct {
		osType corev1.Device_Status_OSType
		arg    string
		want   version
	}{
		{corev1.Device_Status_MAC, "14.5.1", version{num(14), num(5), num(1), nil}},
		{corev1.Device_Status_MAC, "macOS 14.5", version{num(14), num(5), nil, nil}},
		{corev1.Device_Status_MAC, "15.0", version{num(15), num(0), nil, nil}},
		{corev1.Device_Status_MAC, "14.5 (23F79)", version{num(14), num(5), nil, nil}},
		{corev1.Device_Status_WINDOWS, "10.0.22631.3880", version{num(10), num(0), num(22631), num(3880)}},
		{corev1.Device_Status_WINDOWS, "Windows 11 Pro 23H2 10.0.22631.3880",
			version{num(10), num(0), num(22631), num(3880)}},
		{corev1.Device_Status_WINDOWS, "Windows 11", version{}},
		{corev1.Device_Status_WINDOWS, "19044", version{}},
		{corev1.Device_Status_LINUX, "Ubuntu 22.04.4 LTS", version{num(22), num(4), num(4), nil}},
		{corev1.Device_Status_ANDROID, "14", version{num(14), nil, nil, nil}},
		{corev1.Device_Status_IOS, "17", version{num(17), nil, nil, nil}},
		{corev1.Device_Status_LINUX, "14", version{}},
		{corev1.Device_Status_OS_TYPE_UNKNOWN, "1.2.3.4.5", version{num(1), num(2), num(3), num(4)}},
		{corev1.Device_Status_MAC, "99999999999.1", version{}},
		{corev1.Device_Status_MAC, "", version{}},
	} {
		t.Run(tc.arg, func(t *testing.T) {
			ret := ParseOS(tc.osType, tc.arg)
			assert.Equal(t, tc.osType, ret.Type)
			assert.Equal(t, tc.arg, ret.Version)
			assert.Equal(t, tc.want, version{ret.Major, ret.Minor, ret.Patch, ret.Revision})
		})
	}

	assert.Nil(t, ParseOS(corev1.Device_Status_OS_TYPE_UNKNOWN, " "))
	assert.Equal(t, "14.5", ParseOS(corev1.Device_Status_MAC, " 14.5 ").Version)
}

func TestParseDeviceClass(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want corev1.Device_Status_Posture_DeviceClass
	}{
		{nil, corev1.Device_Status_Posture_DEVICE_CLASS_UNKNOWN},
		{[]string{"", " "}, corev1.Device_Status_Posture_DEVICE_CLASS_UNKNOWN},
		{[]string{"kubernetes node"}, corev1.Device_Status_Posture_DEVICE_CLASS_UNKNOWN},
		{[]string{"Laptop"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"desktop"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Notebook"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Workstation"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Computer"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Mac"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"MacBookPro18,3"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"iMac24,1"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Mac mini"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"Server"}, corev1.Device_Status_Posture_SERVER},
		{[]string{"Domain Controller"}, corev1.Device_Status_Posture_SERVER},
		{[]string{"Windows Server 2022"}, corev1.Device_Status_Posture_SERVER},
		{[]string{"Phone"}, corev1.Device_Status_Posture_PHONE},
		{[]string{"iPhone 14 Pro"}, corev1.Device_Status_Posture_PHONE},
		{[]string{"Tablet"}, corev1.Device_Status_Posture_TABLET},
		{[]string{"iPad Pro"}, corev1.Device_Status_Posture_TABLET},
		{[]string{"VMware Virtual Platform"}, corev1.Device_Status_Posture_VIRTUAL_MACHINE},
		{[]string{"Virtual Server"}, corev1.Device_Status_Posture_VIRTUAL_MACHINE},
		{[]string{"Virtual Machine", "Windows Server 2022"}, corev1.Device_Status_Posture_VIRTUAL_MACHINE},
		{[]string{"", "Laptop"}, corev1.Device_Status_Posture_WORKSTATION},
		{[]string{"unrecognized", "Server"}, corev1.Device_Status_Posture_SERVER},
	} {
		assert.Equal(t, tc.want, ParseDeviceClass(tc.args...), "%v", tc.args)
	}
}

func TestScoreAndCount(t *testing.T) {
	assert.EqualValues(t, 0, *Score(0))
	assert.EqualValues(t, 95, *Score(95))
	assert.EqualValues(t, 100, *Score(100))
	assert.EqualValues(t, 100, *Score(101))
	assert.EqualValues(t, 0, *Score(-1))

	count := func(value int) *int {
		return &value
	}

	assert.Nil(t, Count(nil))
	assert.EqualValues(t, 0, *Count(count(0)))
	assert.EqualValues(t, 3, *Count(count(3)))
	assert.Nil(t, Count(count(-1)))
}

func TestParseHTTPSURL(t *testing.T) {
	{
		ret, err := ParseHTTPSURL(" https://example.com/api/ ", "")
		assert.Nil(t, err)
		assert.Equal(t, "https://example.com/api", ret)
	}

	{
		ret, err := ParseHTTPSURL("", "https://default.example.com")
		assert.Nil(t, err)
		assert.Equal(t, "https://default.example.com", ret)
	}

	for _, arg := range []string{
		"",
		"http://example.com",
		"https://",
		"example.com",
		"https://user:pass@example.com",
		"https://example.com?a=b",
		"https://example.com#frag",
		"://",
	} {
		_, err := ParseHTTPSURL(arg, "")
		assert.NotNil(t, err, arg)
	}

	assert.True(t, IsSameOrigin("https://graph.microsoft.com", "https://graph.microsoft.com/v1.0/x?$skiptoken=1"))
	assert.False(t, IsSameOrigin("https://graph.microsoft.com", "https://evil.example.com/v1.0/x"))
	assert.False(t, IsSameOrigin("https://graph.microsoft.com", "http://graph.microsoft.com/v1.0/x"))
	assert.False(t, IsSameOrigin("https://graph.microsoft.com", "https://user@graph.microsoft.com/x"))
	assert.False(t, IsSameOrigin("https://graph.microsoft.com", "://"))
}

func TestLinkingOptions(t *testing.T) {
	assert.False(t, RequireOwnerMatch(nil))
	assert.Zero(t, VerificationInterval(nil))
	assert.Equal(t, DefaultMaxObservationAge, MaxObservationAge(nil))
	assert.Equal(t, DefaultStaleAfter, StaleAfter(nil))

	dm := &enterprisev1.DeviceManager{
		Spec: &enterprisev1.DeviceManager_Spec{},
	}
	assert.Equal(t, DefaultMaxObservationAge, MaxObservationAge(dm))

	dm.Spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
		MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 0}},
	}
	assert.Equal(t, DefaultMaxObservationAge, MaxObservationAge(dm))
	assert.False(t, RequireOwnerMatch(dm))

	dm.Spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
		RequireOwnerMatch:    true,
		VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
	}
	dm.Spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
		StaleAfter:        &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 30}},
		MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Days{Days: 2}},
	}

	assert.True(t, RequireOwnerMatch(dm))
	assert.Equal(t, 2*time.Hour, VerificationInterval(dm))
	assert.Equal(t, 48*time.Hour, MaxObservationAge(dm))
	assert.Equal(t, 30*time.Minute, StaleAfter(dm))
}

func TestMaterializePosture(t *testing.T) {
	collectedAt := time.Now().UTC().Truncate(time.Second)

	dm := &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Uid:  "dm-uid",
			Name: "dm",
		},
		Spec: &enterprisev1.DeviceManager_Spec{},
	}

	assert.Nil(t, MaterializePosture(nil, &Entry{}))
	assert.Nil(t, MaterializePosture(NewOwner(nil, dm, NewFleet(nil), collectedAt, 0), nil))

	{
		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, 0)
		posture := MaterializePosture(owner, &Entry{})
		assert.Equal(t, collectedAt, posture.LastSyncAt.AsTime())
		assert.Equal(t, collectedAt, posture.ExpiresAt.AsTime(),
			"a posture without an observation time must not be valid by default")
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-time.Minute)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, 0)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(DefaultStaleAfter), posture.ExpiresAt.AsTime())
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-8 * 24 * time.Hour)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(-24*time.Hour), posture.ExpiresAt.AsTime(),
			"an observation older than the default maximum age must not be renewed by a collection")
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				Compliant:  corev1.Device_Status_Posture_PASS,
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-3 * 24 * time.Hour)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, 30*time.Minute)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(30*time.Minute), posture.ExpiresAt.AsTime())
		assert.Equal(t, corev1.Device_Status_Posture_PASS, posture.Compliant)

		posture.Compliant = corev1.Device_Status_Posture_FAIL
		assert.Equal(t, corev1.Device_Status_Posture_PASS, entry.Posture.Compliant)
	}

	dm.Spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
		MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Days{Days: 1}},
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-3 * 24 * time.Hour)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(-2*24*time.Hour), posture.ExpiresAt.AsTime())
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-23*time.Hour - 30*time.Minute)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(30*time.Minute), posture.ExpiresAt.AsTime())
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(-time.Minute)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(time.Hour), posture.ExpiresAt.AsTime())
	}

	{
		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, &Entry{})
		assert.Equal(t, collectedAt, posture.ExpiresAt.AsTime())
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(365 * 24 * time.Hour)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt, posture.ExpiresAt.AsTime())
	}

	{
		entry := &Entry{
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(collectedAt.Add(time.Minute)),
			},
		}

		owner := NewOwner(nil, dm, NewFleet(nil), collectedAt, time.Hour)
		posture := MaterializePosture(owner, entry)
		assert.Equal(t, collectedAt.Add(time.Hour), posture.ExpiresAt.AsTime())
	}
}

func TestFleetFutureObservation(t *testing.T) {
	now := time.Now()

	getEntry := func(externalID string, lastSeenAt time.Time) *Entry {
		return &Entry{
			ExternalID: externalID,
			Serial:     "serial-1",
			Posture: &corev1.Device_Status_Posture{
				LastSeenAt: pbutils.Timestamp(lastSeenAt),
			},
		}
	}

	fleet := NewFleet([]*Entry{
		getEntry("old", now.Add(-72*time.Hour)),
		getEntry("future", now.Add(365*24*time.Hour)),
	})
	assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("serial-1", nil).State)

	fleet = NewFleet([]*Entry{
		getEntry("old", now.Add(-72*time.Hour)),
		getEntry("skewed", now.Add(time.Minute)),
	})
	res := fleet.MatchIdentity("serial-1", nil)
	assert.Equal(t, MatchStateUnique, res.State)
	assert.Equal(t, "skewed", res.Entry.ExternalID)
}

func TestCheckRedirect(t *testing.T) {
	getReq := func(arg string) *http.Request {
		u, err := url.Parse(arg)
		assert.Nil(t, err)
		return &http.Request{URL: u}
	}

	via := []*http.Request{getReq("https://api.example.com/v1/devices")}

	assert.Nil(t, CheckRedirect(getReq("https://api.example.com/v1/devices"), nil))
	assert.Nil(t, CheckRedirect(getReq("https://api.example.com/v2/devices?page=1"), via))

	for _, arg := range []string{
		"http://api.example.com/v1/devices",
		"https://evil.example.com/v1/devices",
		"https://sub.api.example.com/v1/devices",
		"https://api.example.com:8443/v1/devices",
		"https://user:password@evil.example.com/v1/devices",
	} {
		err := CheckRedirect(getReq(arg), via)
		assert.True(t, errors.Is(err, ErrRedirectNotAllowed), arg)
		assert.NotContains(t, err.Error(), "password")
	}

	var long []*http.Request
	for range maxRedirects {
		long = append(long, via[0])
	}
	err := CheckRedirect(getReq("https://api.example.com/v1/devices"), long)
	assert.NotNil(t, err)
	assert.False(t, errors.Is(err, ErrRedirectNotAllowed))
}

func TestRetryCondition(t *testing.T) {
	for _, tc := range []struct {
		code  int
		retry bool
	}{
		{200, false}, {301, false}, {400, false}, {401, false}, {403, false}, {404, false},
		{408, true}, {429, true}, {500, true}, {502, true}, {503, true}, {504, true},
	} {
		response := &resty.Response{RawResponse: &http.Response{StatusCode: tc.code}}
		assert.Equal(t, tc.retry, RetryCondition(response, nil), tc.code)
	}

	assert.True(t, RetryCondition(nil, errors.New("transport failure")))
	assert.False(t, RetryCondition(nil, &url.Error{
		Op:  "Get",
		URL: "https://evil.example.com",
		Err: errors.Wrap(ErrRedirectNotAllowed, "https://evil.example.com"),
	}))
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"", 0},
		{"0", 0},
		{" 2 ", 2 * time.Second},
		{"300", 300 * time.Second},
		{"301", 0},
		{"-1", 0},
		{"invalid", 0},
		{now.Add(30 * time.Second).Format(http.TimeFormat), 30 * time.Second},
		{now.Add(-30 * time.Second).Format(http.TimeFormat), 0},
		{now.Add(time.Hour).Format(http.TimeFormat), 0},
	} {
		assert.Equal(t, tc.want, ParseRetryAfter(tc.value, now), tc.value)
	}

	{
		delay, err := RetryAfter(nil, nil)
		assert.Nil(t, err)
		assert.Zero(t, delay)
	}

	{
		delay, err := RetryAfter(nil, &resty.Response{RawResponse: &http.Response{
			Header: http.Header{"Retry-After": []string{"7"}},
		}})
		assert.Nil(t, err)
		assert.Equal(t, 7*time.Second, delay)
	}
}

func TestRegistry(t *testing.T) {
	now := time.Now()

	newDM := func(uid, name string) *enterprisev1.DeviceManager {
		return &enterprisev1.DeviceManager{
			Metadata: &metav1.Metadata{
				Uid:  uid,
				Name: name,
			},
			Spec: &enterprisev1.DeviceManager_Spec{},
		}
	}

	registry := NewRegistry()

	pending := NewPendingOwner(newDM("uid-b", "dm-b"))
	assert.False(t, pending.Fresh(now))
	assert.Equal(t, "dm-b", pending.Name())
	assert.Equal(t, "uid-b", pending.OwnerRef().Uid)

	registry.SetOwner(pending)
	registry.SetOwner(NewOwner(nil, newDM("uid-a", "dm-a"), NewFleet(nil), now, time.Minute))
	registry.SetOwner(nil)
	registry.SetOwner(NewPendingOwner(nil))

	owners := registry.ListOwners()
	assert.Len(t, owners, 2)
	assert.Equal(t, "uid-a", owners[0].UID())
	assert.Equal(t, "uid-b", owners[1].UID())

	owner, ok := registry.GetOwner("uid-a")
	assert.True(t, ok)
	assert.True(t, owner.Fresh(now))
	assert.False(t, owner.Fresh(now.Add(time.Minute)))

	registry.DeleteOwner("uid-a")
	registry.DeleteOwner("")
	_, ok = registry.GetOwner("uid-a")
	assert.False(t, ok)
	assert.Len(t, registry.ListOwners(), 1)

	var nilOwner *Owner
	assert.Equal(t, "", nilOwner.UID())
	assert.Equal(t, "", nilOwner.Name())
	assert.Nil(t, nilOwner.OwnerRef())
	assert.False(t, nilOwner.Fresh(now))

	assert.True(t, RefUIDEqual(&metav1.ObjectReference{Uid: "a"}, &metav1.ObjectReference{Uid: "a"}))
	assert.False(t, RefUIDEqual(&metav1.ObjectReference{}, &metav1.ObjectReference{}))
	assert.False(t, RefUIDEqual(nil, &metav1.ObjectReference{Uid: "a"}))
}

type tstSignedManager struct {
	tstProbeManager
	signedID string
}

func (m *tstSignedManager) ParseExternalID(osType corev1.Device_Status_OSType, results []*ProbeResult) (string, error) {
	return results[0].Text, nil
}

func (m *tstSignedManager) ParseSignedExternalID(results []*ProbeResult, now time.Time) string {
	return m.signedID
}

func TestParseProbeID(t *testing.T) {
	results := []*ProbeResult{{Text: "unsigned"}}
	now := time.Now()

	{
		id, isSigned, err := ParseProbeID(&tstSignedManager{signedID: "signed"}, corev1.Device_Status_LINUX, results, now)
		assert.Nil(t, err)
		assert.Equal(t, "signed", id)
		assert.True(t, isSigned)
	}

	{
		id, isSigned, err := ParseProbeID(&tstSignedManager{}, corev1.Device_Status_LINUX, results, now)
		assert.Nil(t, err)
		assert.Equal(t, "unsigned", id)
		assert.False(t, isSigned)
	}

	{
		id, isSigned, err := ParseProbeID(&tstProbeManager{}, corev1.Device_Status_LINUX, results, now)
		assert.Nil(t, err)
		assert.Empty(t, id)
		assert.False(t, isSigned)
	}
}

func TestEvidenceSignedProbe(t *testing.T) {
	entry := &Entry{ExternalID: "id-1", Serial: "serial-1"}

	assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
		corev1.Device_Status_Binding_PROBE,
		corev1.Device_Status_Binding_VENDOR_SIGNED,
	}, entry.Evidence(&LinkRequest{ProbeID: "id-1", IsProbeSigned: true}))

	assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
		corev1.Device_Status_Binding_PROBE,
	}, entry.Evidence(&LinkRequest{ProbeID: "id-1"}))

	assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
		corev1.Device_Status_Binding_SERIAL_NUMBER,
	}, entry.Evidence(&LinkRequest{ProbeID: "id-2", IsProbeSigned: true, SerialNumber: "SERIAL-1"}),
		"a signed identifier of another entry must not count")
}

func TestParseTime(t *testing.T) {
	for _, val := range []string{"", "  ", "yesterday", "2026-09-28"} {
		_, ok := ParseTime(val)
		assert.False(t, ok, "%q", val)
	}

	for _, val := range []string{"2026-09-28T10:00:00Z", "2026-09-28T10:00:00.123456789+02:00"} {
		ret, ok := ParseTime(val)
		assert.True(t, ok, "%q", val)
		assert.False(t, ret.IsZero())
	}
}

func TestSnippet(t *testing.T) {
	assert.Equal(t, "", Snippet(nil))
	assert.Equal(t, "body", Snippet([]byte("  body\n")))
	assert.Equal(t, strings.Repeat("x", 300), Snippet([]byte(strings.Repeat("x", 300))))
	assert.Equal(t, strings.Repeat("x", 300)+"...", Snippet([]byte(strings.Repeat("x", 301))))
}
