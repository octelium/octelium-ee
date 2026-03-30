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

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
)

/*
func (s *Server) CreateSecretStore(ctx context.Context, req *enterprisev1.SecretStore) (*enterprisev1.SecretStore, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err == nil {
		return nil, serr.InvalidArg("The SecretStore %s already exists", req.Metadata.Name)
	}

	if !grpcerr.IsNotFound(err) {
		return nil, serr.K8sInternal(err)
	}

	item := &enterprisev1.SecretStore{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.SecretStore_Status{},
	}

	item, err = s.octeliumC.EnterpriseC().CreateSecretStore(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}
*/

func (s *Server) GetSecretStore(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.SecretStore, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

/*
func (s *Server) ListSecretStore(ctx context.Context, req *enterprisev1.ListSecretStoreOptions) (*enterprisev1.SecretStoreList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListSecretStore(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteSecretStore(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteSecretStore(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}
*/

func (s *Server) UpdateSecretStore(ctx context.Context, req *enterprisev1.SecretStore) (*enterprisev1.SecretStore, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	switch item.Spec.Type.(type) {
	case *enterprisev1.SecretStore_Spec_AwsKeyManagementService:
		if item.Status.Type != enterprisev1.SecretStore_Status_TYPE_AWS_KMS {
			item.Status.State = enterprisev1.SecretStore_Status_LOADING
		}

		item.Status.Type = enterprisev1.SecretStore_Status_TYPE_AWS_KMS
	case *enterprisev1.SecretStore_Spec_AzureKeyVault_:
		if item.Status.Type != enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT {
			item.Status.State = enterprisev1.SecretStore_Status_LOADING
		}

		item.Status.Type = enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT
	case *enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_:
		if item.Status.Type != enterprisev1.SecretStore_Status_TYPE_GCP_KMS {
			item.Status.State = enterprisev1.SecretStore_Status_LOADING
		}

		item.Status.Type = enterprisev1.SecretStore_Status_TYPE_GCP_KMS
	case *enterprisev1.SecretStore_Spec_HashicorpVault_:
		if item.Status.Type != enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT {
			item.Status.State = enterprisev1.SecretStore_Status_LOADING
		}

		item.Status.Type = enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT
	case *enterprisev1.SecretStore_Spec_Kubernetes_:
		item.Status.Type = enterprisev1.SecretStore_Status_KUBERNETES
	}

	item, err = s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) ListSecretStore(ctx context.Context, req *enterprisev1.ListSecretStoreOptions) (*enterprisev1.SecretStoreList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListSecretStore(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}
