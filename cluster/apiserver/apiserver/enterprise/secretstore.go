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
	"net/url"
	"strings"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

const maxSecretStoreStringBytes = 512

func (s *Server) GetSecretStore(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.SecretStore, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) UpdateSecretStore(ctx context.Context, req *enterprisev1.SecretStore) (*enterprisev1.SecretStore, error) {
	if err := validateSecretStore(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	statusType, err := getSecretStoreStatusType(req.Spec)
	if err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec
	if item.Status == nil {
		item.Status = &enterprisev1.SecretStore_Status{}
	}
	item.Status.Type = statusType

	item, err = s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) ListSecretStore(ctx context.Context, req *enterprisev1.ListSecretStoreOptions) (*enterprisev1.SecretStoreList, error) {
	if req == nil {
		req = &enterprisev1.ListSecretStoreOptions{}
	}

	itemList, err := s.octeliumC.EnterpriseC().ListSecretStore(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) SynchronizeSecretStore(ctx context.Context,
	req *enterprisev1.SynchronizeSecretStoreRequest) (*enterprisev1.SynchronizeSecretStoreResponse, error) {

	if err := apivalidation.CheckObjectRef(req.GetSecretStoreRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.SecretStoreRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if item.Status.Synchronization != nil &&
		item.Status.Synchronization.State == enterprisev1.SecretStore_Status_Synchronization_SYNCING {
		return nil, grpcutils.InvalidArg("SecretStore is already SYNCING")
	}

	item.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED,
	}

	if _, err := s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &enterprisev1.SynchronizeSecretStoreResponse{}, nil
}

func validateSecretStore(req *enterprisev1.SecretStore) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil SecretStore")
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return err
	}

	return validateSecretStoreSpec(req.Spec)
}

func validateSecretStoreSpec(spec *enterprisev1.SecretStore_Spec) error {
	if spec == nil {
		return grpcutils.InvalidArg("SecretStore spec must be set")
	}

	if spec.Type == nil {
		return grpcutils.InvalidArg("SecretStore type must be set")
	}

	switch typ := spec.Type.(type) {
	case *enterprisev1.SecretStore_Spec_AzureKeyVault_:
		if typ.AzureKeyVault == nil {
			return grpcutils.InvalidArg("Nil Azure Key Vault SecretStore spec")
		}
		if err := validateSecretStoreString(typ.AzureKeyVault.ClientID, true, "Azure Key Vault clientID"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.AzureKeyVault.TenantID, true, "Azure Key Vault tenantID"); err != nil {
			return err
		}
		if err := validateSecretStoreURL(typ.AzureKeyVault.VaultURL, "Azure Key Vault vaultURL"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.AzureKeyVault.Key, true, "Azure Key Vault key"); err != nil {
			return err
		}

	case *enterprisev1.SecretStore_Spec_HashicorpVault_:
		if typ.HashicorpVault == nil {
			return grpcutils.InvalidArg("Nil HashiCorp Vault SecretStore spec")
		}
		if err := validateSecretStoreURL(typ.HashicorpVault.Address, "HashiCorp Vault address"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.HashicorpVault.Role, true, "HashiCorp Vault role"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.HashicorpVault.Key, true, "HashiCorp Vault key"); err != nil {
			return err
		}

	case *enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_:
		if typ.GoogleCloudKeyManagementService == nil {
			return grpcutils.InvalidArg("Nil Google Cloud KMS SecretStore spec")
		}
		if err := validateSecretStoreString(typ.GoogleCloudKeyManagementService.Project, true, "Google Cloud KMS project"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.GoogleCloudKeyManagementService.Location, true, "Google Cloud KMS location"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.GoogleCloudKeyManagementService.KeyRing, true, "Google Cloud KMS keyRing"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.GoogleCloudKeyManagementService.Key, true, "Google Cloud KMS key"); err != nil {
			return err
		}

	case *enterprisev1.SecretStore_Spec_AwsKeyManagementService:
		if typ.AwsKeyManagementService == nil {
			return grpcutils.InvalidArg("Nil AWS KMS SecretStore spec")
		}
		if err := validateSecretStoreString(typ.AwsKeyManagementService.KeyID, true, "AWS KMS keyID"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.AwsKeyManagementService.Region, true, "AWS KMS region"); err != nil {
			return err
		}
		if err := validateSecretStoreString(typ.AwsKeyManagementService.RoleARN, false, "AWS KMS roleARN"); err != nil {
			return err
		}
		if typ.AwsKeyManagementService.RoleARN != "" && !isValidSecretStoreAWSRoleARN(typ.AwsKeyManagementService.RoleARN) {
			return grpcutils.InvalidArg("Invalid AWS KMS roleARN")
		}

	case *enterprisev1.SecretStore_Spec_Kubernetes_:
		if typ.Kubernetes == nil {
			return grpcutils.InvalidArg("Nil Kubernetes SecretStore spec")
		}

	default:
		return grpcutils.InvalidArg("Unsupported SecretStore type")
	}

	return nil
}

func getSecretStoreStatusType(spec *enterprisev1.SecretStore_Spec) (enterprisev1.SecretStore_Status_Type, error) {
	if err := validateSecretStoreSpec(spec); err != nil {
		return enterprisev1.SecretStore_Status_TYPE_UNKNOWN, err
	}

	switch spec.Type.(type) {
	case *enterprisev1.SecretStore_Spec_AzureKeyVault_:
		return enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT, nil
	case *enterprisev1.SecretStore_Spec_HashicorpVault_:
		return enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT, nil
	case *enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_:
		return enterprisev1.SecretStore_Status_TYPE_GCP_KMS, nil
	case *enterprisev1.SecretStore_Spec_AwsKeyManagementService:
		return enterprisev1.SecretStore_Status_TYPE_AWS_KMS, nil
	case *enterprisev1.SecretStore_Spec_Kubernetes_:
		return enterprisev1.SecretStore_Status_KUBERNETES, nil
	default:
		return enterprisev1.SecretStore_Status_TYPE_UNKNOWN, grpcutils.InvalidArg("Unsupported SecretStore type")
	}
}

func validateSecretStoreString(v string, required bool, field string) error {
	if strings.TrimSpace(v) == "" {
		if required {
			return grpcutils.InvalidArg("%s is required", field)
		}
		return nil
	}

	if len(v) > maxSecretStoreStringBytes {
		return grpcutils.InvalidArg("%s is too long", field)
	}

	return nil
}

func validateSecretStoreURL(v string, field string) error {
	if err := validateSecretStoreString(v, true, field); err != nil {
		return err
	}

	parsed, err := url.Parse(v)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return grpcutils.InvalidArg("Invalid %s", field)
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return grpcutils.InvalidArg("Invalid %s scheme", field)
	}

	return nil
}

func isValidSecretStoreAWSRoleARN(v string) bool {
	return strings.HasPrefix(v, "arn:") && strings.Contains(v, ":iam::") && strings.Contains(v, ":role/")
}
