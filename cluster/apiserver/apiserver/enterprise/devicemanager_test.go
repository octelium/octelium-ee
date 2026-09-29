// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestDeviceManager(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	sec := tstCreateDeviceManagerSecret(ctx, t, srv)

	item, err := srv.CreateDeviceManager(ctx, &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: tstDeviceManagerCrowdStrikeSpec(sec.Metadata.Name),
		Status: &enterprisev1.DeviceManager_Status{
			State: enterprisev1.DeviceManager_Status_OK,
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, enterprisev1.DeviceManager_Status_CROWDSTRIKE, item.Status.Type)
	assert.Equal(t, enterprisev1.DeviceManager_Status_STATE_UNKNOWN, item.Status.State)

	{
		ret, err := srv.GetDeviceManager(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)

		ret, err = srv.GetDeviceManager(ctx, &metav1.GetOptions{Name: item.Metadata.Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
	}

	{
		_, err := srv.GetDeviceManager(ctx, &metav1.GetOptions{Name: utilrand.GetRandomStringCanonical(8)})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		itemList, err := srv.ListDeviceManager(ctx, nil)
		assert.Nil(t, err, "%+v", err)
		found := false
		for _, listItem := range itemList.Items {
			if listItem.Metadata.Uid == item.Metadata.Uid {
				found = true
			}
		}
		assert.True(t, found)
	}

	{
		_, err := srv.CreateDeviceManager(ctx, tstCloneDeviceManager(item))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateDeviceManager(ctx, &enterprisev1.DeviceManager{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: tstDeviceManagerCrowdStrikeSpec(utilrand.GetRandomStringCanonical(8)),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		item.Status.Collection = &enterprisev1.DeviceManager_Status_Collection{
			ManagedDevices: 10,
		}
		item, err = srv.octeliumC.EnterpriseC().UpdateDeviceManager(ctx, item)
		assert.Nil(t, err, "%+v", err)

		arg := tstCloneDeviceManager(item)
		arg.Spec = tstDeviceManagerSentinelOneSpec(sec.Metadata.Name)
		arg.Status = &enterprisev1.DeviceManager_Status{}

		updated, err := srv.UpdateDeviceManager(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.DeviceManager_Status_SENTINELONE, updated.Status.Type)
		assert.Equal(t, []string{"site-1", "site-2"}, updated.Spec.GetSentinelOne().SiteIDs)
		assert.Equal(t, uint32(10), updated.Status.Collection.ManagedDevices)
		item = updated
	}

	{
		arg := tstCloneDeviceManager(item)
		arg.Spec.GetSentinelOne().ManagementURL = "http://example.sentinelone.net"
		_, err := srv.UpdateDeviceManager(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		arg := tstCloneDeviceManager(item)
		arg.Metadata.Name = utilrand.GetRandomStringCanonical(8)
		_, err := srv.UpdateDeviceManager(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		assert.Nil(t, err, "%+v", err)

		arg := tstCloneClusterConfig(cc)
		arg.Spec.DeviceManagers = []string{item.Metadata.Name}
		_, err = srv.UpdateClusterConfig(ctx, arg)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.DeleteDeviceManager(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		arg.Spec.DeviceManagers = nil
		_, err = srv.UpdateClusterConfig(ctx, arg)
		assert.Nil(t, err, "%+v", err)
	}

	{
		_, err := srv.DeleteDeviceManager(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetDeviceManager(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

		_, err = srv.DeleteDeviceManager(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestValidateDeviceManager(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	secretName := tstCreateDeviceManagerSecret(ctx, t, srv).Metadata.Name

	validSpecs := []*enterprisev1.DeviceManager_Spec{
		tstDeviceManagerCrowdStrikeSpec(secretName),
		tstDeviceManagerSentinelOneSpec(secretName),
		{
			Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
				MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{
					TenantID:               "tenant-id",
					ClientID:               "client-id",
					ClientSecret:           tstDeviceManagerSecretRef(secretName),
					Cloud:                  enterprisev1.DeviceManager_Spec_MicrosoftIntune_US_GOV,
					Filter:                 "operatingSystem eq 'Windows'",
					IsGracePeriodCompliant: true,
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				RequireOwnerMatch: true,
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Jamf_{
				Jamf: &enterprisev1.DeviceManager_Spec_Jamf{
					BaseURL:         "https://example.jamfcloud.com",
					ClientID:        "client-id",
					ClientSecret:    tstDeviceManagerSecretRef(secretName),
					CompliantGroups: []string{"Compliant Macs", "Département"},
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
				OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL:  "https://fleet.example.com/",
					ApiToken: tstDeviceManagerSecretRef(secretName),
					TeamID:   3,
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy:             enterprisev1.DeviceManager_Spec_Linking_IDENTITY_AND_PROBE,
				RequireAgreement:     true,
				VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 12}},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Huntress_{
				Huntress: &enterprisev1.DeviceManager_Spec_Huntress{
					ApiKey:          "api-key",
					ApiSecret:       tstDeviceManagerSecretRef(secretName),
					OrganizationIDs: []int64{1, 2},
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy: enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Iru_{
				Iru: &enterprisev1.DeviceManager_Spec_Iru{
					BaseURL:  "https://example.api.kandji.io",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
			Condition: &corev1.Condition{
				Type: &corev1.Condition_All_{
					All: &corev1.Condition_All{
						Of: []*corev1.Condition{
							{
								Type: &corev1.Condition_Match{
									Match: `ctx.device.status.osType == "MAC"`,
								},
							},
							{
								Type: &corev1.Condition_Not{
									Not: `"contractors" in ctx.user.spec.groups`,
								},
							},
						},
					},
				},
			},
			Polling: &enterprisev1.DeviceManager_Spec_Polling{
				Interval:          &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 10}},
				Timeout:           &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 2}},
				StaleAfter:        &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 30}},
				MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Days{Days: 3}},
			},
		},
	}

	for _, spec := range validSpecs {
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.Nil(t, err, "%+v", err)
	}

	invalidSpecs := []*enterprisev1.DeviceManager_Spec{
		nil,
		{},
		{
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
					ClientSecret: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
					ClientID: "client-id",
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
					ClientID:     "client-id",
					ClientSecret: tstDeviceManagerSecretRef(secretName),
					Region:       enterprisev1.DeviceManager_Spec_CrowdStrike_Region(100),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
					ClientID:     "client-id",
					ClientSecret: tstDeviceManagerSecretRef(secretName),
					HostFilter:   "platform_name:'Linux'\n",
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL:  "http://fleet.example.com",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL:  "https://user:pass@fleet.example.com",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL:  "https://fleet.example.com/?next=https://evil.example.com",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
				OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{
					BaseURL:  "http://api.kolide.com",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Huntress_{
				Huntress: &enterprisev1.DeviceManager_Spec_Huntress{
					ApiKey:          "api-key",
					ApiSecret:       tstDeviceManagerSecretRef(secretName),
					OrganizationIDs: []int64{0},
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Jamf_{
				Jamf: &enterprisev1.DeviceManager_Spec_Jamf{
					BaseURL:         "https://example.jamfcloud.com",
					ClientID:        "client-id",
					ClientSecret:    tstDeviceManagerSecretRef(secretName),
					CompliantGroups: []string{" "},
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
				MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{
					ClientID:     "client-id",
					ClientSecret: tstDeviceManagerSecretRef(secretName),
				},
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Iru_{
				Iru: &enterprisev1.DeviceManager_Spec_Iru{
					BaseURL: "https://example.api.kandji.io",
				},
			},
		},
	}

	for _, spec := range invalidSpecs {
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDeviceManager(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDeviceManagerSentinelOneSpec(secretName)
		spec.GetSentinelOne().SiteIDs = []string{"site-1,site-2"}
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	for _, key := range []string{"siteIds", "accountIds", "cursor", "limit", "LIMIT"} {
		spec := tstDeviceManagerSentinelOneSpec(secretName)
		spec.GetSentinelOne().AgentFilters = map[string]string{
			key: "value",
		}
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	invalidPolling := []*enterprisev1.DeviceManager_Spec_Polling{
		{
			StaleAfter: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
		},
		{
			Interval:   &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
			StaleAfter: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
		},
		{
			Interval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
		},
		{
			Interval:   &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}},
			StaleAfter: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 20}},
		},
		{
			MaxObservationAge: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 0}},
		},
	}

	for _, polling := range invalidPolling {
		spec := tstDeviceManagerCrowdStrikeSpec(secretName)
		spec.Polling = polling
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDeviceManagerCrowdStrikeSpec(secretName)
		spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
			Interval:   &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 10}},
			StaleAfter: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 31}},
			IsDisabled: true,
		}
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.Nil(t, err, "%+v", err)
	}

	invalidLinking := []*enterprisev1.DeviceManager_Spec{
		{
			Type: tstDeviceManagerCrowdStrikeSpec(secretName).Type,
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy: enterprisev1.DeviceManager_Spec_Linking_Strategy(100),
			},
		},
		{
			Type: tstDeviceManagerCrowdStrikeSpec(secretName).Type,
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy:         enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
				RequireAgreement: true,
			},
		},
		{
			Type: tstDeviceManagerCrowdStrikeSpec(secretName).Type,
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy:             enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
				VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
			},
		},
		{
			Type: tstDeviceManagerCrowdStrikeSpec(secretName).Type,
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				RequireOwnerMatch: true,
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Huntress_{
				Huntress: &enterprisev1.DeviceManager_Spec_Huntress{
					ApiKey:    "api-key",
					ApiSecret: tstDeviceManagerSecretRef(secretName),
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy: enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Huntress_{
				Huntress: &enterprisev1.DeviceManager_Spec_Huntress{
					ApiKey:    "api-key",
					ApiSecret: tstDeviceManagerSecretRef(secretName),
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				RequireAgreement: true,
			},
		},
		{
			Type: &enterprisev1.DeviceManager_Spec_Iru_{
				Iru: &enterprisev1.DeviceManager_Spec_Iru{
					BaseURL:  "https://example.api.kandji.io",
					ApiToken: tstDeviceManagerSecretRef(secretName),
				},
			},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
			},
		},
	}

	for _, spec := range invalidLinking {
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	invalidConditions := []*corev1.Condition{
		{},
		{
			Type: &corev1.Condition_Match{
				Match: "",
			},
		},
		{
			Type: &corev1.Condition_Match{
				Match: "ctx.device.status.osType ==",
			},
		},
		{
			Type: &corev1.Condition_Not{
				Not: strings.Repeat("a", maxDeviceManagerCELBytes+1),
			},
		},
		{
			Type: &corev1.Condition_Any_{
				Any: &corev1.Condition_Any{},
			},
		},
		{
			Type: &corev1.Condition_None_{
				None: &corev1.Condition_None{
					Of: []*corev1.Condition{
						{},
					},
				},
			},
		},
		{
			Type: &corev1.Condition_Opa{
				Opa: &corev1.Condition_OPA{},
			},
		},
	}

	for _, condition := range invalidConditions {
		spec := tstDeviceManagerCrowdStrikeSpec(secretName)
		spec.Condition = condition
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDeviceManagerCrowdStrikeSpec(secretName)
		spec.Condition = &corev1.Condition{
			Type: &corev1.Condition_MatchAny{
				MatchAny: true,
			},
		}
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.Nil(t, err, "%+v", err)
	}

	{
		condition := &corev1.Condition{
			Type: &corev1.Condition_MatchAny{
				MatchAny: true,
			},
		}
		for i := 0; i <= maxConditionDepth+1; i++ {
			condition = &corev1.Condition{
				Type: &corev1.Condition_All_{
					All: &corev1.Condition_All{
						Of: []*corev1.Condition{condition},
					},
				},
			}
		}

		spec := tstDeviceManagerCrowdStrikeSpec(secretName)
		spec.Condition = condition
		err := srv.validateDeviceManager(ctx, &enterprisev1.DeviceManager{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestValidateDeviceManagerURL(t *testing.T) {
	for _, arg := range []string{
		"https://example.com",
		"https://example.com/",
		"https://example.com:8443/api",
	} {
		assert.Nil(t, validateDeviceManagerURL(arg, true, "baseURL"), arg)
	}

	for _, arg := range []string{
		"",
		" ",
		"example.com",
		"http://example.com",
		"ftp://example.com",
		"https://",
		"https://user@example.com",
		"https://example.com/?a=b",
		"https://example.com/#frag",
		"https://example.com/" + strings.Repeat("a", maxDeviceManagerURLBytes),
	} {
		err := validateDeviceManagerURL(arg, true, "baseURL")
		assert.NotNil(t, err, arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	assert.Nil(t, validateDeviceManagerURL("", false, "baseURL"))
}

func TestGetDeviceManagerStatusType(t *testing.T) {
	secretRef := tstDeviceManagerSecretRef("secret")

	for _, tc := range []struct {
		spec *enterprisev1.DeviceManager_Spec
		typ  enterprisev1.DeviceManager_Status_Type
	}{
		{nil, enterprisev1.DeviceManager_Status_TYPE_UNKNOWN},
		{&enterprisev1.DeviceManager_Spec{}, enterprisev1.DeviceManager_Status_TYPE_UNKNOWN},
		{tstDeviceManagerCrowdStrikeSpec("secret"), enterprisev1.DeviceManager_Status_CROWDSTRIKE},
		{tstDeviceManagerSentinelOneSpec("secret"), enterprisev1.DeviceManager_Status_SENTINELONE},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
			MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{ClientSecret: secretRef},
		}}, enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Jamf_{
			Jamf: &enterprisev1.DeviceManager_Spec_Jamf{},
		}}, enterprisev1.DeviceManager_Status_JAMF_PRO},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
			OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{},
		}}, enterprisev1.DeviceManager_Status_ONEPASSWORD},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
			FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{},
		}}, enterprisev1.DeviceManager_Status_FLEETDM},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Huntress_{
			Huntress: &enterprisev1.DeviceManager_Spec_Huntress{},
		}}, enterprisev1.DeviceManager_Status_HUNTRESS},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Iru_{
			Iru: &enterprisev1.DeviceManager_Spec_Iru{},
		}}, enterprisev1.DeviceManager_Status_IRU},
	} {
		assert.Equal(t, tc.typ, getDeviceManagerStatusType(tc.spec))
	}
}

func TestResetDeviceBinding(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	bindingUID := utilrand.GetRandomStringCanonical(32)

	dev, err := srv.octeliumC.CoreC().CreateDevice(ctx, &corev1.Device{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Device_Spec{
			State: corev1.Device_Spec_ACTIVE,
		},
		Status: &corev1.Device_Status{
			OsType: corev1.Device_Status_LINUX,
			Binding: &corev1.Device_Status_Binding{
				Uid:        bindingUID,
				ExternalID: "external-id",
				State:      corev1.Device_Status_Binding_ACCEPTED,
				Validity:   corev1.Device_Status_Binding_LOST,
				OwnerRef: &metav1.ObjectReference{
					Name: "dm",
					Uid:  utilrand.GetRandomStringCanonical(32),
				},
			},
			Posture: &corev1.Device_Status_Posture{
				Compliant: corev1.Device_Status_Posture_PASS,
				ExpiresAt: pbutils.Now(),
			},
			ProbeAttempt: &corev1.Device_Status_ProbeAttempt{
				Uid:   utilrand.GetRandomStringCanonical(32),
				State: corev1.Device_Status_ProbeAttempt_PROCESSED,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	{
		_, err := srv.ResetDeviceBinding(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			BindingUID: bindingUID,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef: umetav1.GetObjectReference(dev),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  umetav1.GetObjectReference(dev),
			BindingUID: "INVALID UID",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef: &metav1.ObjectReference{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			BindingUID: bindingUID,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  umetav1.GetObjectReference(dev),
			BindingUID: utilrand.GetRandomStringCanonical(32),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		dev, err := srv.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: dev.Metadata.Uid})
		assert.Nil(t, err)
		assert.Equal(t, bindingUID, dev.Status.Binding.Uid)
		assert.NotNil(t, dev.Status.Posture)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  umetav1.GetObjectReference(dev),
			BindingUID: bindingUID,
		})
		assert.Nil(t, err, "%+v", err)

		dev, err := srv.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: dev.Metadata.Uid})
		assert.Nil(t, err)
		assert.Nil(t, dev.Status.Binding)
		assert.Nil(t, dev.Status.Posture)
		assert.Nil(t, dev.Status.ProbeAttempt)
		assert.Equal(t, corev1.Device_Status_LINUX, dev.Status.OsType)
	}

	{
		_, err := srv.ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  umetav1.GetObjectReference(dev),
			BindingUID: bindingUID,
		})
		assert.Nil(t, err, "%+v", err)
	}
}

func TestClusterConfigDeviceManagers(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	secretName := tstCreateDeviceManagerSecret(ctx, t, srv).Metadata.Name

	var names []string
	for range 2 {
		item, err := srv.CreateDeviceManager(ctx, &enterprisev1.DeviceManager{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: tstDeviceManagerCrowdStrikeSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
		names = append(names, item.Metadata.Name)
	}

	assert.Nil(t, srv.validateClusterConfigDeviceManagers(ctx, nil))
	assert.Nil(t, srv.validateClusterConfigDeviceManagers(ctx, names))
	assert.Nil(t, srv.validateClusterConfigDeviceManagers(ctx, []string{names[1], names[0]}))

	for _, arg := range [][]string{
		{names[0], names[0]},
		{names[0], utilrand.GetRandomStringCanonical(8)},
		{"Invalid Name"},
		{""},
	} {
		err := srv.validateClusterConfigDeviceManagers(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		var arg []string
		for range maxClusterConfigDeviceManagers + 1 {
			arg = append(arg, names[0])
		}
		err := srv.validateClusterConfigDeviceManagers(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		assert.Nil(t, err, "%+v", err)

		arg := tstCloneClusterConfig(cc)
		arg.Spec.DeviceManagers = []string{names[1], names[0]}
		updated, err := srv.UpdateClusterConfig(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{names[1], names[0]}, updated.Spec.DeviceManagers)

		arg.Spec.DeviceManagers = []string{utilrand.GetRandomStringCanonical(8)}
		_, err = srv.UpdateClusterConfig(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func tstCreateDeviceManagerSecret(ctx context.Context, t *testing.T, srv *Server) *enterprisev1.Secret {
	sec, err := srv.CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return sec
}

func tstCloneDeviceManager(arg *enterprisev1.DeviceManager) *enterprisev1.DeviceManager {
	return &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec:   pbutils.Clone(arg.Spec).(*enterprisev1.DeviceManager_Spec),
		Status: arg.Status,
	}
}

func tstDeviceManagerSecretRef(secretName string) *enterprisev1.DeviceManager_Spec_SecretRef {
	return &enterprisev1.DeviceManager_Spec_SecretRef{
		Type: &enterprisev1.DeviceManager_Spec_SecretRef_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDeviceManagerCrowdStrikeSpec(secretName string) *enterprisev1.DeviceManager_Spec {
	return &enterprisev1.DeviceManager_Spec{
		Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
			CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
				Region:       enterprisev1.DeviceManager_Spec_CrowdStrike_EU_1,
				ClientID:     "client-id",
				ClientSecret: tstDeviceManagerSecretRef(secretName),
				HostFilter:   "platform_name:'Windows'",
			},
		},
	}
}

func tstDeviceManagerSentinelOneSpec(secretName string) *enterprisev1.DeviceManager_Spec {
	return &enterprisev1.DeviceManager_Spec{
		Type: &enterprisev1.DeviceManager_Spec_SentinelOne_{
			SentinelOne: &enterprisev1.DeviceManager_Spec_SentinelOne{
				ManagementURL: "https://example.sentinelone.net",
				ApiToken:      tstDeviceManagerSecretRef(secretName),
				SiteIDs:       []string{"site-1", "site-2"},
				AccountIDs:    []string{"account-1"},
				AgentFilters: map[string]string{
					"isDecommissioned": "false",
				},
			},
		},
	}
}
