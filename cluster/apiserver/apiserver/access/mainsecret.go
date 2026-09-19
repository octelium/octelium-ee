// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

const maxSecretDataBytes = 512 * 1024

func (s *ServerMain) CreateSecret(ctx context.Context,
	req *accessv1.Secret) (*accessv1.Secret, error) {
	if err := s.validateSecret(req); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetSecret(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The Secret %s already exists", req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	item := &accessv1.Secret{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     &accessv1.Secret_Spec{},
		Status:   &accessv1.Secret_Status{},
		Data:     req.Data,
	}

	item, err = s.octeliumC.AccessC().CreateSecret(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	item.Data = nil

	return item, nil
}

func (s *ServerMain) GetSecret(ctx context.Context,
	req *metav1.GetOptions) (*accessv1.Secret, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetSecret(ctx,
		apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	item.Data = nil

	return item, nil
}

func (s *ServerMain) ListSecret(ctx context.Context,
	req *accessv1.ListSecretOptions) (*accessv1.SecretList, error) {
	itemList, err := s.octeliumC.AccessC().ListSecret(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	for _, item := range itemList.Items {
		item.Data = nil
	}

	return itemList, nil
}

func (s *ServerMain) UpdateSecret(ctx context.Context,
	req *accessv1.Secret) (*accessv1.Secret, error) {
	if err := s.validateSecret(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetSecret(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Data = req.Data

	item, err = s.octeliumC.AccessC().UpdateSecret(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	item.Data = nil

	return item, nil
}

func (s *ServerMain) DeleteSecret(ctx context.Context,
	req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetSecret(ctx,
		apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	if err := s.checkSecretUnused(ctx, item); err != nil {
		return nil, err
	}

	if _, err := s.octeliumC.AccessC().DeleteSecret(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	}); err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) validateSecret(req *accessv1.Secret) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil Secret")
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return err
	}

	if req.Data == nil || req.Data.Type == nil {
		return grpcutils.InvalidArg("Empty Secret data")
	}

	var sz int

	switch req.Data.Type.(type) {
	case *accessv1.Secret_Data_Value:
		sz = len(req.Data.GetValue())
	case *accessv1.Secret_Data_ValueBytes:
		sz = len(req.Data.GetValueBytes())
	default:
		return grpcutils.InvalidArg("Invalid Secret data type")
	}

	if sz == 0 || sz > maxSecretDataBytes {
		return grpcutils.InvalidArg("Invalid Secret size")
	}

	return nil
}

func (s *ServerMain) checkSecretUnused(ctx context.Context, item *accessv1.Secret) error {
	itemList, err := s.octeliumC.AccessC().ListIntegration(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	for _, integration := range itemList.Items {
		for _, name := range integrationSecretNames(integration) {
			if name == item.Metadata.Name {
				return grpcutils.InvalidArg(
					"The Secret %s is still used by the Integration %s",
					item.Metadata.Name, integration.Metadata.Name)
			}
		}
	}

	return nil
}
