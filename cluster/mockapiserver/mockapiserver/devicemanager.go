// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func genDeviceManagers(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	secretRef := func(name string) *enterprisev1.DeviceManager_Spec_SecretRef {
		return &enterprisev1.DeviceManager_Spec_SecretRef{
			Type: &enterprisev1.DeviceManager_Spec_SecretRef_FromSecret{FromSecret: name},
		}
	}

	for _, name := range []string{
		"falcon-client-secret",
		"intune-client-secret",
		"jamf-client-secret",
		"sentinelone-token",
		"fleet-token",
		"kolide-token",
	} {
		if _, err := octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: name},
			Spec:     &enterprisev1.Secret_Spec{},
			Status:   &enterprisev1.Secret_Status{},
		}); err != nil {
			return err
		}
	}

	ago := func(d time.Duration) *timestamppb.Timestamp {
		return timestamppb.New(time.Now().Add(-d))
	}

	collection := func(managed uint32, lastError string) *enterprisev1.DeviceManager_Status_Collection {
		return &enterprisev1.DeviceManager_Status_Collection{
			LastAttemptAt:  ago(2 * time.Minute),
			LastSuccessAt:  ago(2 * time.Minute),
			ManagedDevices: managed,
			LastError:      lastError,
		}
	}

	managers := []*enterprisev1.DeviceManager{
		{
			Metadata: &metav1.Metadata{
				Name:        "crowdstrike",
				DisplayName: "CrowdStrike Falcon",
				Description: "Corporate laptops and servers",
			},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
					CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{
						Region:       enterprisev1.DeviceManager_Spec_CrowdStrike_EU_1,
						ClientID:     "0123456789abcdef0123456789abcdef",
						ClientSecret: secretRef("falcon-client-secret"),
						HostFilter:   "platform_name:'Windows'",
					},
				},
				Polling: &enterprisev1.DeviceManager_Spec_Polling{
					Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 10}},
				},
				Linking: &enterprisev1.DeviceManager_Spec_Linking{
					VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 12}},
				},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:       enterprisev1.DeviceManager_Status_CROWDSTRIKE,
				State:      enterprisev1.DeviceManager_Status_OK,
				Collection: collection(1240, ""),
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "intune"},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
					MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{
						TenantID:     "0b3f1e7a-24c4-4a42-8e0e-7c1c2f9e5d21",
						ClientID:     "6f1d2b9c-7a51-4c4e-9a0f-2d8f2f4f8b11",
						ClientSecret: secretRef("intune-client-secret"),
					},
				},
				Linking: &enterprisev1.DeviceManager_Spec_Linking{RequireOwnerMatch: true},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:       enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE,
				State:      enterprisev1.DeviceManager_Status_DEGRADED,
				Collection: collection(320, "compliance states could not be read"),
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "jamf"},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_Jamf_{
					Jamf: &enterprisev1.DeviceManager_Spec_Jamf{
						BaseURL:         "https://example.jamfcloud.com",
						ClientID:        "0f6a1c34-9d7e-4b1a-8a65-3a6d2f0c9e77",
						ClientSecret:    secretRef("jamf-client-secret"),
						CompliantGroups: []string{"Compliant Macs"},
					},
				},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:  enterprisev1.DeviceManager_Status_JAMF_PRO,
				State: enterprisev1.DeviceManager_Status_ERROR,
				Collection: &enterprisev1.DeviceManager_Status_Collection{
					LastAttemptAt:  ago(time.Minute),
					LastSuccessAt:  ago(3 * time.Hour),
					ManagedDevices: 90,
					LastError:      "401 Unauthorized: the API client secret was rejected",
				},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "sentinelone"},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_SentinelOne_{
					SentinelOne: &enterprisev1.DeviceManager_Spec_SentinelOne{
						ManagementURL: "https://example.sentinelone.net",
						ApiToken:      secretRef("sentinelone-token"),
						SiteIDs:       []string{"225494730938493804"},
					},
				},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:  enterprisev1.DeviceManager_Status_SENTINELONE,
				State: enterprisev1.DeviceManager_Status_LOADING,
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "fleet"},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
					FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
						BaseURL:  "https://fleet.example.com",
						ApiToken: secretRef("fleet-token"),
					},
				},
				Polling: &enterprisev1.DeviceManager_Spec_Polling{IsDisabled: true},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:  enterprisev1.DeviceManager_Status_FLEETDM,
				State: enterprisev1.DeviceManager_Status_DISABLED,
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "onepassword"},
			Spec: &enterprisev1.DeviceManager_Spec{
				Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
					OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{
						ApiToken: secretRef("kolide-token"),
					},
				},
			},
			Status: &enterprisev1.DeviceManager_Status{
				Type:       enterprisev1.DeviceManager_Status_ONEPASSWORD,
				State:      enterprisev1.DeviceManager_Status_OK,
				Collection: collection(64, ""),
			},
		},
	}

	for idx, dm := range managers {
		created, err := octeliumC.EnterpriseC().CreateDeviceManager(ctx, dm)
		if err != nil {
			return err
		}
		managers[idx] = created
	}

	devices, err := octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	linking := map[string]*enterprisev1.DeviceManager_Status_Linking{}
	for _, dm := range managers {
		linking[dm.Metadata.Uid] = &enterprisev1.DeviceManager_Status_Linking{LastSweepAt: ago(time.Minute)}
	}

	bound := managers[:3]
	for idx, dev := range devices.Items {
		dm := bound[idx%len(bound)]
		tally := linking[dm.Metadata.Uid]
		binding := &corev1.Device_Status_Binding{
			Uid:        utilrand.GetRandomStringLowercase(12),
			OwnerRef:   umetav1.GetObjectReference(dm),
			ExternalID: utilrand.GetRandomStringLowercase(24),
			State:      corev1.Device_Status_Binding_ACCEPTED,
			Validity:   corev1.Device_Status_Binding_VALID,
			AcceptedAt: ago(48 * time.Hour),
			Assurance:  corev1.Device_Status_Binding_PROVIDER_ID,
		}

		switch idx % 7 {
		case 3:
			binding.State = corev1.Device_Status_Binding_AMBIGUOUS
			binding.Validity = corev1.Device_Status_Binding_VALIDITY_UNKNOWN
			binding.Assurance = corev1.Device_Status_Binding_ASSURANCE_UNKNOWN
			tally.Ambiguous++
		case 4:
			binding.State = corev1.Device_Status_Binding_CONFLICT
			binding.Validity = corev1.Device_Status_Binding_VALIDITY_UNKNOWN
			binding.Assurance = corev1.Device_Status_Binding_ASSURANCE_UNKNOWN
			tally.Conflicts++
		case 5:
			binding.State = corev1.Device_Status_Binding_OWNER_MISMATCH
			binding.Validity = corev1.Device_Status_Binding_VALIDITY_UNKNOWN
			binding.Assurance = corev1.Device_Status_Binding_ASSURANCE_UNKNOWN
			tally.OwnerMismatches++
		case 6:
			binding.Validity = corev1.Device_Status_Binding_SUSPENDED
			tally.LinkedDevices++
			tally.Suspended++
		default:
			tally.LinkedDevices++
		}

		dev.Status.Binding = binding
		if _, err := octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
			return err
		}
	}

	for _, dm := range managers {
		dm.Status.Linking = linking[dm.Metadata.Uid]
		if _, err := octeliumC.EnterpriseC().UpdateDeviceManager(ctx, dm); err != nil {
			return err
		}
	}

	return nil
}
