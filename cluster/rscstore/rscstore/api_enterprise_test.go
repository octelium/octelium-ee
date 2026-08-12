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

func TestEnterpriseListFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvEnterprise{s: env.srv}
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
		Metadata:   newRscStoreMetadata("disabled-kafka-exporter", now.Add(time.Second)),
		Spec: &enterprisev1.CollectorExporter_Spec{
			IsDisabled: true,
			Type: &enterprisev1.CollectorExporter_Spec_Kafka_{
				Kafka: &enterprisev1.CollectorExporter_Spec_Kafka{},
			},
		},
		Status: &enterprisev1.CollectorExporter_Status{},
	})

	{
		resp, err := srv.ListCollectorExporter(env.ctx, &venterprisev1.ListCollectorExporterOptions{
			Type: venterprisev1.ListCollectorExporterOptions_OTLP,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "otlp-exporter", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCollectorExporter(env.ctx, &venterprisev1.ListCollectorExporterOptions{
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-kafka-exporter", resp.Items[0].Metadata.Name)
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
		},
	})

	{
		resp, err := srv.ListSecretStore(env.ctx, &venterprisev1.ListSecretStoreOptions{
			Type:                 enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT,
			State:                enterprisev1.SecretStore_Status_OK,
			SynchronizationState: enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "azure-store", resp.Items[0].Metadata.Name)
	}

	issuerRef := &metav1.ObjectReference{Name: "issuer-one", Uid: vutils.UUIDv4()}
	otherIssuerRef := &metav1.ObjectReference{Name: "issuer-two", Uid: vutils.UUIDv4()}
	serviceRef := &metav1.ObjectReference{Name: "svc-one", Uid: vutils.UUIDv4()}
	namespaceRef := &metav1.ObjectReference{Name: "ns-one", Uid: vutils.UUIDv4()}

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
			CertificateIssuerRef: otherIssuerRef,
			NamespaceRef:         namespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				State:     enterprisev1.Certificate_Status_Issuance_ISSUING,
				ExpiresAt: timestamppb.New(now.Add(10 * 24 * time.Hour)),
			},
		},
	})

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			Mode: enterprisev1.Certificate_Spec_MANAGED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expired-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			CertificateIssuerRef: issuerRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expired-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			ServiceRef: serviceRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expired-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			NamespaceRef: namespaceRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expiring-soon-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			IssuanceState: enterprisev1.Certificate_Status_Issuance_SUCCESS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expired-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			IsExpired: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expired-cert", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCertificate(env.ctx, &venterprisev1.ListCertificateOptions{
			IsExpiringSoon: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "expiring-soon-cert", resp.Items[0].Metadata.Name)
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
		resp, err := srv.ListCertificateIssuer(env.ctx, &venterprisev1.ListCertificateIssuerOptions{
			Type:  venterprisev1.ListCertificateIssuerOptions_ACME,
			State: enterprisev1.CertificateIssuer_Status_READY,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "acme-issuer", resp.Items[0].Metadata.Name)
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
	insertRscStoreObject(t, env, &enterprisev1.DNSProvider{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDNSProvider,
		Metadata:   newRscStoreMetadata("aws-dns", now.Add(time.Second)),
		Spec: &enterprisev1.DNSProvider_Spec{
			Type: &enterprisev1.DNSProvider_Spec_Aws{
				Aws: &enterprisev1.DNSProvider_Spec_AWS{Region: "us-east-1"},
			},
		},
		Status: &enterprisev1.DNSProvider_Status{},
	})

	{
		resp, err := srv.ListDNSProvider(env.ctx, &venterprisev1.ListDNSProviderOptions{
			Type: venterprisev1.ListDNSProviderOptions_CLOUDFLARE,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "cf-dns", resp.Items[0].Metadata.Name)
	}

	dirProvider1 := &enterprisev1.DirectoryProvider{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProvider,
		Metadata:   newRscStoreMetadata("scim-dp", now),
		Spec: &enterprisev1.DirectoryProvider_Spec{
			IsDisabled: true,
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
	insertRscStoreObject(t, env, dirProvider1)

	insertRscStoreObject(t, env, &enterprisev1.DirectoryProvider{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProvider,
		Metadata:   newRscStoreMetadata("keycloak-dp", now.Add(time.Second)),
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_{
				Keycloak: &enterprisev1.DirectoryProvider_Spec_Keycloak{Url: "https://kc.example.com"},
			},
		},
		Status: &enterprisev1.DirectoryProvider_Status{},
	})

	{
		resp, err := srv.ListDirectoryProvider(env.ctx, &venterprisev1.ListDirectoryProviderOptions{
			Type:       venterprisev1.ListDirectoryProviderOptions_SCIM,
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "scim-dp", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListDirectoryProvider(env.ctx, &venterprisev1.ListDirectoryProviderOptions{
			SynchronizationState: enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "scim-dp", resp.Items[0].Metadata.Name)
	}

	dpRef1 := &metav1.ObjectReference{Name: dirProvider1.Metadata.Name, Uid: dirProvider1.Metadata.Uid}
	dpUserRef := &metav1.ObjectReference{Name: "dp-user", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderUser{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderUser,
		Metadata:   newRscStoreMetadata("dp-user-1", now),
		Spec:       &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			DirectoryProviderRef: dpRef1,
			UserRef:              dpUserRef,
		},
	})
	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderUser{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderUser,
		Metadata:   newRscStoreMetadata("dp-user-2", now.Add(time.Second)),
		Spec:       &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			UserRef: &metav1.ObjectReference{Name: "other-user", Uid: vutils.UUIDv4()},
		},
	})

	{
		resp, err := srv.ListDirectoryProviderUser(env.ctx, &venterprisev1.ListDirectoryProviderUserOptions{
			DirectoryProviderRef: dpRef1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "dp-user-1", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListDirectoryProviderUser(env.ctx, &venterprisev1.ListDirectoryProviderUserOptions{
			UserRef: dpUserRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "dp-user-1", resp.Items[0].Metadata.Name)
	}

	dpGroupRef := &metav1.ObjectReference{Name: "dp-group", Uid: vutils.UUIDv4()}
	insertRscStoreObject(t, env, &enterprisev1.DirectoryProviderGroup{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDirectoryProviderGroup,
		Metadata:   newRscStoreMetadata("dp-group-1", now),
		Spec:       &enterprisev1.DirectoryProviderGroup_Spec{},
		Status: &enterprisev1.DirectoryProviderGroup_Status{
			DirectoryProviderRef: dpRef1,
			GroupRef:             dpGroupRef,
		},
	})

	{
		resp, err := srv.ListDirectoryProviderGroup(env.ctx, &venterprisev1.ListDirectoryProviderGroupOptions{
			DirectoryProviderRef: dpRef1,
			GroupRef:             dpGroupRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "dp-group-1", resp.Items[0].Metadata.Name)
	}

	insertRscStoreObject(t, env, &enterprisev1.DeviceManager{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDeviceManager,
		Metadata:   newRscStoreMetadata("crowdstrike-dm", now),
		Spec: &enterprisev1.DeviceManager_Spec{
			Polling: &enterprisev1.DeviceManager_Spec_Polling{IsDisabled: true},
			Linking: &enterprisev1.DeviceManager_Spec_Linking{
				Strategy:     enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
				ApprovalMode: enterprisev1.DeviceManager_Spec_Linking_AUTOMATIC,
			},
			Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
				CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{},
			},
		},
		Status: &enterprisev1.DeviceManager_Status{
			Type:  enterprisev1.DeviceManager_Status_CROWDSTRIKE,
			State: enterprisev1.DeviceManager_Status_OK,
		},
	})
	insertRscStoreObject(t, env, &enterprisev1.DeviceManager{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindDeviceManager,
		Metadata:   newRscStoreMetadata("sentinelone-dm", now.Add(time.Second)),
		Spec: &enterprisev1.DeviceManager_Spec{
			Type: &enterprisev1.DeviceManager_Spec_SentinelOne_{
				SentinelOne: &enterprisev1.DeviceManager_Spec_SentinelOne{},
			},
		},
		Status: &enterprisev1.DeviceManager_Status{
			Type:  enterprisev1.DeviceManager_Status_SENTINELONE,
			State: enterprisev1.DeviceManager_Status_DEGRADED,
		},
	})

	{
		resp, err := srv.ListDeviceManager(env.ctx, &venterprisev1.ListDeviceManagerOptions{
			Type:  enterprisev1.DeviceManager_Status_CROWDSTRIKE,
			State: enterprisev1.DeviceManager_Status_OK,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "crowdstrike-dm", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListDeviceManager(env.ctx, &venterprisev1.ListDeviceManagerOptions{
			Strategy:     enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
			ApprovalMode: enterprisev1.DeviceManager_Spec_Linking_AUTOMATIC,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "crowdstrike-dm", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListDeviceManager(env.ctx, &venterprisev1.ListDeviceManagerOptions{
			IsPollingDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "crowdstrike-dm", resp.Items[0].Metadata.Name)
	}

	insertRscStoreObject(t, env, &enterprisev1.Secret{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindSecret,
		Metadata:   newRscStoreMetadata("visible-secret", now),
		Spec:       &enterprisev1.Secret_Spec{},
		Status:     &enterprisev1.Secret_Status{},
	})

	{
		resp, err := srv.ListSecret(env.ctx, &venterprisev1.ListSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-secret", resp.Items[0].Metadata.Name)
	}
}
