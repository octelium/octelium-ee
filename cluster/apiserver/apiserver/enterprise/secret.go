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
	"github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *Server) CreateSecret(ctx context.Context, req *enterprisev1.Secret) (*enterprisev1.Secret, error) {

	if err := s.validateSecret(ctx, req); err != nil {
		return nil, err
	}

	{
		_, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.ObjectToRGetOptions(req))
		if err == nil {
			return nil, grpcutils.AlreadyExists("The Secret %s already exists", req.Metadata.Name)
		}
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	item := &enterprisev1.Secret{
		Metadata: common.MetadataFrom(req.Metadata),
		Spec:     &enterprisev1.Secret_Spec{},
		Status:   &enterprisev1.Secret_Status{},
		Data:     req.Data,
	}

	item, err := s.octeliumC.EnterpriseC().CreateSecret(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) ListSecret(ctx context.Context, req *enterprisev1.ListSecretOptions) (*enterprisev1.SecretList, error) {

	vSecrets, err := s.octeliumC.EnterpriseC().ListSecret(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	for _, secret := range vSecrets.Items {
		secret.Data = nil
	}

	return vSecrets, nil
}

func (s *Server) DeleteSecret(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if sec.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot delete a system object")
	}

	_, err = s.octeliumC.EnterpriseC().DeleteSecret(ctx, &rmetav1.DeleteOptions{Uid: sec.Metadata.Uid})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) GetSecret(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.Secret, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystemHidden(ret); err != nil {
		return nil, err
	}

	ret.Data = nil

	return ret, nil
}

func (s *Server) UpdateSecret(ctx context.Context, req *enterprisev1.Secret) (*enterprisev1.Secret, error) {

	if err := s.validateSecret(ctx, req); err != nil {
		return nil, err
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	sec.Spec = req.Spec
	sec.Data = req.Data

	item, err := s.octeliumC.EnterpriseC().UpdateSecret(ctx, sec)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) validateSecret(ctx context.Context, req *enterprisev1.Secret) error {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
		RequireData: true,
	}); err != nil {
		return err
	}

	maxSize := 1500

	switch req.Data.Type.(type) {
	case *enterprisev1.Secret_Data_Value:
		lenVal := len(req.Data.GetValue())
		if lenVal == 0 {
			return grpcutils.InvalidArg("Empty value")
		}
		if lenVal > maxSize {
			return grpcutils.InvalidArg("Secret data is too large")
		}

	case *enterprisev1.Secret_Data_ValueBytes:
		lenVal := len(req.Data.GetValueBytes())
		if lenVal == 0 {
			return grpcutils.InvalidArg("Empty value")
		}
		if lenVal > maxSize {
			return grpcutils.InvalidArg("Secret data is too large")
		}
	default:
		return grpcutils.InvalidArg("Invalid Secret data type")
	}

	return nil

}
