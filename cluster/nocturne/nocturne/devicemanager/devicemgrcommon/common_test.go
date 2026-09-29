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
		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("serial-dup", []string{"a4:bb:cc:dd:ee:01"}).State)
	}

	{
		res := fleet.MatchIdentity("", []string{"A4-BB-CC-DD-EE-02"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, MatchMethodMAC, res.Method)
		assert.Equal(t, "id-2", res.Entry.ExternalID)

		res = fleet.MatchIdentity("unknown", []string{"a4:bb:cc:dd:ee:03", "invalid"})
		assert.Equal(t, MatchStateUnique, res.State)
		assert.Equal(t, "id-3", res.Entry.ExternalID)

		assert.Equal(t, MatchStateAmbiguous, fleet.MatchIdentity("", []string{"a4:bb:cc:dd:ee:ff"}).State)
		assert.Equal(t, MatchStateAmbiguous,
			fleet.MatchIdentity("", []string{"a4:bb:cc:dd:ee:01", "a4:bb:cc:dd:ee:02"}).State)
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
