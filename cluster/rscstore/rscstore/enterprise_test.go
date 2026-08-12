// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEnterpriseSummaries(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()

	insertRscStoreObject(t, env, &enterprisev1.CollectorExporter{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindCollectorExporter,
		Metadata:   newRscStoreMetadata("otlp-exporter", now),
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_Otlp{
				Otlp: &enterprisev1.CollectorExporter_Spec_OTLP{Endpoint: "localhost:4317"},
			},
		},
		Status: &enterprisev1.CollectorExporter_Status{},
	})
	insertRscStoreObject(t, env, &enterprisev1.CollectorExporter{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindCollectorExporter,
		Metadata:   newRscStoreMetadata("datadog-exporter", now.Add(time.Second)),
		Spec: &enterprisev1.CollectorExporter_Spec{
			IsDisabled: true,
			Type: &enterprisev1.CollectorExporter_Spec_Datadog_{
				Datadog: &enterprisev1.CollectorExporter_Spec_Datadog{},
			},
		},
		Status: &enterprisev1.CollectorExporter_Status{},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseCollectorExporter(env.ctx, &venterprisev1.GetCollectorExporterSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDisabled)
		assert.EqualValues(t, 1, resp.TotalOTLP)
		assert.EqualValues(t, 1, resp.TotalDatadog)
		assert.EqualValues(t, 0, resp.TotalKafka)
	}

	insertRscStoreObject(t, env, &enterprisev1.Secret{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindSecret,
		Metadata:   newRscStoreMetadata("visible-secret", now),
		Spec:       &enterprisev1.Secret_Spec{},
		Status:     &enterprisev1.Secret_Status{},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseSecret(env.ctx, &venterprisev1.GetSecretSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
	}

	insertRscStoreObject(t, env, &enterprisev1.SecretStore{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindSecretStore,
		Metadata:   newRscStoreMetadata("azure-store", now),
		Spec:       &enterprisev1.SecretStore_Spec{},
		Status: &enterprisev1.SecretStore_Status{
			Type:  enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT,
			State: enterprisev1.SecretStore_Status_OK,
			Synchronization: &enterprisev1.SecretStore_Status_Synchronization{
				State: enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
			},
		},
	})
	insertRscStoreObject(t, env, &enterprisev1.SecretStore{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindSecretStore,
		Metadata:   newRscStoreMetadata("k8s-store", now.Add(time.Second)),
		Spec:       &enterprisev1.SecretStore_Spec{},
		Status: &enterprisev1.SecretStore_Status{
			Type:  enterprisev1.SecretStore_Status_KUBERNETES,
			State: enterprisev1.SecretStore_Status_LOADING,
			Synchronization: &enterprisev1.SecretStore_Status_Synchronization{
				State: enterprisev1.SecretStore_Status_Synchronization_FAILED,
			},
		},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseSecretStore(env.ctx, &venterprisev1.GetSecretStoreSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalAzureKeyVault)
		assert.EqualValues(t, 1, resp.TotalKubernetes)
		assert.EqualValues(t, 1, resp.TotalOK)
		assert.EqualValues(t, 1, resp.TotalLoading)
		assert.EqualValues(t, 1, resp.TotalSynchronizationSuccess)
		assert.EqualValues(t, 1, resp.TotalSynchronizationFailed)
	}

	issuerRef := &metav1.ObjectReference{Name: "issuer-one", Uid: vutils.UUIDv4()}
	serviceRef := &metav1.ObjectReference{Name: "svc-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &enterprisev1.Certificate{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindCertificate,
		Metadata:   newRscStoreMetadata("expired-cert", now),
		Spec:       &enterprisev1.Certificate_Spec{Mode: enterprisev1.Certificate_Spec_MANAGED},
		Status: &enterprisev1.Certificate_Status{
			CertificateIssuerRef: issuerRef,
			ServiceRef:           serviceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
				ExpiresAt: timestamppb.New(now.Add(-24 * time.Hour)),
			},
		},
	})
	insertRscStoreObject(t, env, &enterprisev1.Certificate{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindCertificate,
		Metadata:   newRscStoreMetadata("expiring-soon-cert", now.Add(time.Second)),
		Spec:       &enterprisev1.Certificate_Spec{Mode: enterprisev1.Certificate_Spec_MANUAL},
		Status: &enterprisev1.Certificate_Status{
			CertificateIssuerRef: issuerRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				State:     enterprisev1.Certificate_Status_Issuance_ISSUING,
				ExpiresAt: timestamppb.New(now.Add(10 * 24 * time.Hour)),
			},
		},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseCertificate(env.ctx, &venterprisev1.GetCertificateSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalManaged)
		assert.EqualValues(t, 1, resp.TotalManual)
		assert.EqualValues(t, 1, resp.TotalIssuanceSuccess)
		assert.EqualValues(t, 1, resp.TotalIssuing)
		assert.EqualValues(t, 1, resp.TotalExpired)
		assert.EqualValues(t, 1, resp.TotalExpiringSoon)
		assert.EqualValues(t, 1, resp.TotalService)
		assert.EqualValues(t, 1, resp.TotalCertificateIssuer)
	}

	insertRscStoreObject(t, env, &enterprisev1.CertificateIssuer{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindCertificateIssuer,
		Metadata:   newRscStoreMetadata("acme-issuer", now),
		Spec: &enterprisev1.CertificateIssuer_Spec{
			Type: &enterprisev1.CertificateIssuer_Spec_Acme{
				Acme: &enterprisev1.CertificateIssuer_Spec_ACME{Email: "a@example.com"},
			},
		},
		Status: &enterprisev1.CertificateIssuer_Status{State: enterprisev1.CertificateIssuer_Status_READY},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseCertificateIssuer(env.ctx, &venterprisev1.GetCertificateIssuerSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalACME)
		assert.EqualValues(t, 1, resp.TotalReady)
	}

	insertRscStoreObject(t, env, &enterprisev1.DNSProvider{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDNSProvider,
		Metadata:   newRscStoreMetadata("cf-dns", now),
		Spec: &enterprisev1.DNSProvider_Spec{
			Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{
				Cloudflare: &enterprisev1.DNSProvider_Spec_Cloudflare{Email: "a@example.com"},
			},
		},
		Status: &enterprisev1.DNSProvider_Status{},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseDNSProvider(env.ctx, &venterprisev1.GetDNSProviderSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalCloudflare)
	}

	dirProvider := &enterprisev1.DirectoryProvider{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProvider,
		Metadata:   newRscStoreMetadata("scim-dp", now),
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
		Status: &enterprisev1.DirectoryProvider_Status{
			Synchronization: &enterprisev1.DirectoryProvider_Status_Synchronization{
				State: enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS,
			},
		},
	}
	insertRscStoreObject(t, env, dirProvider)

	dpUserRef1 := &metav1.ObjectReference{Name: "u1", Uid: vutils.UUIDv4()}
	dpUserRef2 := &metav1.ObjectReference{Name: "u2", Uid: vutils.UUIDv4()}
	dpRef := &metav1.ObjectReference{Name: dirProvider.Metadata.Name, Uid: dirProvider.Metadata.Uid}

	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderUser{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderUser,
		Metadata:   newRscStoreMetadata("dp-user-1", now),
		Spec:       &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			DirectoryProviderRef: dpRef,
			UserRef:              dpUserRef1,
		},
	})
	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderUser{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderUser,
		Metadata:   newRscStoreMetadata("dp-user-2", now.Add(time.Second)),
		Spec:       &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			DirectoryProviderRef: dpRef,
			UserRef:              dpUserRef2,
		},
	})

	dpGroupRef := &metav1.ObjectReference{Name: "g1", Uid: vutils.UUIDv4()}
	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderGroup{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderGroup,
		Metadata:   newRscStoreMetadata("dp-group-1", now),
		Spec:       &enterprisev1.DirectoryProviderGroup_Spec{},
		Status: &enterprisev1.DirectoryProviderGroup_Status{
			DirectoryProviderRef: dpRef,
			GroupRef:             dpGroupRef,
		},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseDirectoryProvider(env.ctx, &venterprisev1.GetDirectoryProviderSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalSCIM)
		assert.EqualValues(t, 1, resp.TotalSynchronizationSuccess)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 1, resp.TotalGroup)
	}

	{
		resp, err := env.srv.getSummaryEnterpriseDirectoryProviderUser(env.ctx, &venterprisev1.GetDirectoryProviderUserSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDirectoryProvider)
		assert.EqualValues(t, 2, resp.TotalUser)
	}

	{
		resp, err := env.srv.getSummaryEnterpriseDirectoryProviderGroup(env.ctx, &venterprisev1.GetDirectoryProviderGroupSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDirectoryProvider)
		assert.EqualValues(t, 1, resp.TotalGroup)
	}

	insertRscStoreObject(t, env, &enterprisev1.DeviceManager{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDeviceManager,
		Metadata:   newRscStoreMetadata("crowdstrike-dm", now),
		Spec: &enterprisev1.DeviceManager_Spec{
			Polling: &enterprisev1.DeviceManager_Spec_Polling{IsDisabled: true},
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{},
			},
		},
		Status: &enterprisev1.DeviceManager_Status{
			Type:  enterprisev1.DeviceManager_Status_CROWDSTRIKE,
			State: enterprisev1.DeviceManager_Status_OK,
			Collection: &enterprisev1.DeviceManager_Status_Collection{
				ManagedDevices: 7,
			},
			Linking: &enterprisev1.DeviceManager_Status_Linking{
				LinkedDevices:   5,
				WaitingApproval: 2,
				Ambiguous:       1,
				FailedUpdates:   3,
			},
		},
	})

	{
		resp, err := env.srv.getSummaryEnterpriseDeviceManager(env.ctx, &venterprisev1.GetDeviceManagerSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalCrowdStrike)
		assert.EqualValues(t, 1, resp.TotalOK)
		assert.EqualValues(t, 1, resp.TotalPollingDisabled)
		assert.EqualValues(t, 7, resp.TotalManagedDevices)
		assert.EqualValues(t, 5, resp.TotalLinkedDevices)
		assert.EqualValues(t, 2, resp.TotalWaitingApproval)
		assert.EqualValues(t, 1, resp.TotalAmbiguous)
		assert.EqualValues(t, 3, resp.TotalFailedUpdates)
	}
}
