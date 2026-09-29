// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devicemgrcommon

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func TestFleet(t *testing.T) {
	fleet := NewFleet([]*Entry{
		{
			ExternalID: "id-1",
			Aliases:    []string{"ALIAS-1", "00000000-0000-0000-0000-000000000000"},
			Serial:     "SERIAL-1",
			MACs:       []string{"AA:BB:CC:DD:EE:01"},
		},
		{
			ExternalID: "id-2",
			Aliases:    []string{"00000000-0000-0000-0000-000000000000"},
			Serial:     "serial-dup",
			MACs:       []string{"aa-bb-cc-dd-ee-02", "aa:bb:cc:dd:ee:ff"},
		},
		{
			ExternalID: "id-3",
			Serial:     "SERIAL-DUP",
			MACs:       []string{"aabb.ccdd.ee03", "aa:bb:cc:dd:ee:ff"},
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
		assert.Equal(t, MatchMethodExternalID, res.Method)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		assert.Equal(t, MatchStateNone, fleet.MatchExternalID("alias-1").State)
		assert.Equal(t, MatchStateNone, fleet.MatchExternalID("").State)
	}

	{
		res := fleet.MatchProbeID(" alias-1 ")
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, MatchMethodProbeID, res.Method)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		res = fleet.MatchProbeID("ID-2")
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-2", res.Entry.ExternalID)

		assert.Equal(t, MatchStateNone, fleet.MatchProbeID("00000000-0000-0000-0000-000000000000").State)
		assert.Equal(t, MatchStateAmbiguous, fleet.MatchProbeID("shared-alias").State)
		assert.Equal(t, MatchStateNone, fleet.MatchProbeID("unknown").State)
	}

	{
		res := fleet.MatchIdentity("serial-1", nil)
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, MatchMethodSerial, res.Method)
		assert.Equal(t, "id-1", res.Entry.ExternalID)

		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("serial-dup", nil).State)
		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("serial-dup", []string{"aa:bb:cc:dd:ee:01"}).State)
	}

	{
		res := fleet.MatchIdentity("", []string{"AA-BB-CC-DD-EE-02"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, MatchMethodMAC, res.Method)
		assert.Equal(t, "id-2", res.Entry.ExternalID)

		res = fleet.MatchIdentity("unknown", []string{"aa:bb:cc:dd:ee:03", "invalid"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-3", res.Entry.ExternalID)

		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("", []string{"aa:bb:cc:dd:ee:ff"}).State)
		assert.Equal(t, MatchStateAmbiguous,
			fleet.MatchIdentity("", []string{"aa:bb:cc:dd:ee:01", "aa:bb:cc:dd:ee:02"}).State)
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
	}

	{
		entries := fleet.Entries()
		entries[0].Aliases[0] = "changed"
		assert.Equal(t, MatchStateUnique, fleet.MatchProbeID("alias-1").State)
	}
}

func TestNormalize(t *testing.T) {
	assert.Equal(t, "abc-def", NormalizeID(" ABC-DEF "))
	assert.Equal(t, "", NormalizeID("00000000-0000-0000-0000-000000000000"))
	assert.Equal(t, "", NormalizeID("0"))
	assert.Equal(t, "", NormalizeID(" "))

	assert.Equal(t, "c02abc", NormalizeSerial(" C02ABC "))
	for _, arg := range []string{"", "0", "None", "Default String", "To Be Filled By O.E.M.", "unknown"} {
		assert.Equal(t, "", NormalizeSerial(arg), arg)
	}

	assert.Equal(t, "aabbccddeeff", NormalizeMAC("AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, "aabbccddeeff", NormalizeMAC("aabb.ccdd.eeff"))
	assert.Equal(t, "", NormalizeMAC("00:00:00:00:00:00"))
	assert.Equal(t, "", NormalizeMAC("aa:bb:cc"))

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

	assert.Equal(t, "x.crowdstrike.name", SignalKey("crowdstrike", "name"))
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
	assert.Equal(t, enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE, LinkingStrategy(nil))
	assert.False(t, RequireAgreement(nil))
	assert.False(t, RequireOwnerMatch(nil))
	assert.Zero(t, VerificationInterval(nil))
	assert.Zero(t, MaxObservationAge(nil))
	assert.Equal(t, DefaultStaleAfter, StaleAfter(nil))

	dm := &enterprisev1.DeviceManager{
		Spec: &enterprisev1.DeviceManager_Spec{},
	}
	assert.Equal(t, enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE, LinkingStrategy(dm))
	assert.True(t, UsesProbe(dm))
	assert.True(t, UsesIdentity(dm))

	dm.Spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
		Strategy:             enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
		RequireAgreement:     true,
		RequireOwnerMatch:    true,
		VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
	}
	dm.Spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
		StaleAfter:        &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 30}},
		MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Days{Days: 2}},
	}

	assert.True(t, UsesProbe(dm))
	assert.False(t, UsesIdentity(dm))
	assert.True(t, RequireAgreement(dm))
	assert.True(t, RequireOwnerMatch(dm))
	assert.Equal(t, 2*time.Hour, VerificationInterval(dm))
	assert.Equal(t, 48*time.Hour, MaxObservationAge(dm))
	assert.Equal(t, 30*time.Minute, StaleAfter(dm))

	dm.Spec.Linking.Strategy = enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY
	assert.False(t, UsesProbe(dm))
	assert.True(t, UsesIdentity(dm))
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
		assert.Equal(t, collectedAt.Add(DefaultStaleAfter), posture.ExpiresAt.AsTime())
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
