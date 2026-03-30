// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package uenterprisev1

import (
	"fmt"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	KindSecret                 = "Secret"
	KindClusterConfig          = "ClusterConfig"
	KindCollectorExporter      = "CollectorExporter"
	KindCertificate            = "Certificate"
	KindCertificateIssuer      = "CertificateIssuer"
	KindDNSProvider            = "DNSProvider"
	KindDirectoryProvider      = "DirectoryProvider"
	KindDirectoryProviderUser  = "DirectoryProviderUser"
	KindDirectoryProviderGroup = "DirectoryProviderGroup"
	KindSecretStore            = "SecretStore"
	KindAuditLog               = "AuditLog"
	KindAuthenticationLog      = "AuthenticationLog"
)

type ResourceObjectRefG interface {
	*enterprisev1.Secret | *enterprisev1.Certificate | *enterprisev1.CertificateIssuer |
		*enterprisev1.DNSProvider |
		*enterprisev1.DirectoryProvider | *enterprisev1.DirectoryProviderUser | *enterprisev1.DirectoryProviderGroup |
		*enterprisev1.ClusterConfig | *enterprisev1.CollectorExporter | *enterprisev1.SecretStore
}

const API = "enterprise"
const Version = "v1"
const APIVersion = "enterprise/v1"

type Secret struct {
	*enterprisev1.Secret
}

type SecretList struct {
	*enterprisev1.SecretList
}

type CertificateIssuer struct {
	*enterprisev1.CertificateIssuer
}

type Certificate struct {
	*enterprisev1.Certificate
}

func NewObjectList(kind string) (umetav1.ObjectI, error) {

	switch kind {
	case KindSecret:
		return &enterprisev1.SecretList{}, nil
	case KindCertificate:
		return &enterprisev1.CertificateList{}, nil
	case KindCertificateIssuer:
		return &enterprisev1.CertificateIssuerList{}, nil
	case KindCollectorExporter:
		return &enterprisev1.CollectorExporterList{}, nil
	case KindDNSProvider:
		return &enterprisev1.DNSProviderList{}, nil
	case KindDirectoryProvider:
		return &enterprisev1.DirectoryProviderList{}, nil
	case KindDirectoryProviderUser:
		return &enterprisev1.DirectoryProviderUserList{}, nil
	case KindDirectoryProviderGroup:
		return &enterprisev1.DirectoryProviderGroupList{}, nil
	case KindSecretStore:
		return &enterprisev1.SecretStoreList{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func NewObjectListOptions(kind string) (proto.Message, error) {

	switch kind {
	case KindSecret:
		return &enterprisev1.ListSecretOptions{}, nil
	case KindCollectorExporter:
		return &enterprisev1.ListCollectorExporterOptions{}, nil
	case KindSecretStore:
		return &enterprisev1.ListSecretStoreOptions{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func NewObject(kind string) (umetav1.ResourceObjectI, error) {

	switch kind {
	case KindClusterConfig:
		return &enterprisev1.ClusterConfig{}, nil
	case KindSecret:
		return &enterprisev1.Secret{}, nil
	case KindCertificate:
		return &enterprisev1.Certificate{}, nil
	case KindCertificateIssuer:
		return &enterprisev1.CertificateIssuer{}, nil
	case KindCollectorExporter:
		return &enterprisev1.CollectorExporter{}, nil
	case KindDNSProvider:
		return &enterprisev1.DNSProvider{}, nil
	case KindDirectoryProvider:
		return &enterprisev1.DirectoryProvider{}, nil
	case KindDirectoryProviderUser:
		return &enterprisev1.DirectoryProviderUser{}, nil
	case KindDirectoryProviderGroup:
		return &enterprisev1.DirectoryProviderGroup{}, nil
	case KindSecretStore:
		return &enterprisev1.SecretStore{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func ToSecret(a *enterprisev1.Secret) *Secret {
	return &Secret{
		Secret: a,
	}
}

func ToCertificate(a *enterprisev1.Certificate) *Certificate {
	return &Certificate{
		Certificate: a,
	}
}

func ToCertificateIssuer(a *enterprisev1.CertificateIssuer) *CertificateIssuer {
	return &CertificateIssuer{
		CertificateIssuer: a,
	}
}

func ToSecretList(a *enterprisev1.SecretList) *SecretList {
	return &SecretList{
		SecretList: a,
	}
}

func (s *Secret) GetValueStr() string {
	if s.Data == nil {
		return ""
	}
	switch s.Data.Type.(type) {
	case *enterprisev1.Secret_Data_Value:
		return s.Data.GetValue()
	case *enterprisev1.Secret_Data_ValueBytes:
		return string(s.Data.GetValueBytes())
	default:
		return ""
	}
}

func (s *SecretList) GetByName(name string) (*enterprisev1.Secret, error) {
	for _, idp := range s.Items {
		if idp.Metadata.Name == name {
			return idp, nil
		}
	}
	return nil, errors.Errorf("No Secret exists with name: %s", name)
}

func (s *CertificateIssuer) GetACMEAccountSecretName() string {
	return fmt.Sprintf("sys:acme-acc-%s", s.Metadata.Name)
}

func (s *Certificate) GetSecretName() string {
	return fmt.Sprintf("crt-%s", s.Metadata.Name)
}

func (iss *CertificateIssuer) GetDirectoryURL() string {
	if iss.Spec.GetAcme() != nil && iss.Spec.GetAcme().Server != "" {
		return iss.Spec.GetAcme().Server
	}

	if ldflags.IsDev() {
		return "https://acme-staging-v02.api.letsencrypt.org/directory"
	} else {
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}
