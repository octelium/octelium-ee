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
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *Server) CreateSecret(ctx context.Context, req *enterprisev1.Secret) (*enterprisev1.Secret, error) {
	if err := s.validateSecret(ctx, req); err != nil {
		return nil, grpcutils.InvalidArgWithErr(err)
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
		Spec:     req.Spec,
		Status:   &enterprisev1.Secret_Status{},
		Data:     req.Data,
	}

	item, err := s.octeliumC.EnterpriseC().CreateSecret(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	item.Data = nil

	return item, nil
}

func (s *Server) ListSecret(ctx context.Context, req *enterprisev1.ListSecretOptions) (*enterprisev1.SecretList, error) {

	vSecrets, err := s.octeliumC.EnterpriseC().ListSecret(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, err
	}

	for _, secret := range vSecrets.Items {
		secret.Data = nil
	}

	return vSecrets, nil
}

func (s *Server) DeleteSecret(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	if err := apivalidation.CheckDeleteOptions(req, nil); err != nil {
		return nil, err
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(sec); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteSecret(ctx, &rmetav1.DeleteOptions{Uid: sec.Metadata.Uid})
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) GetSecret(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.Secret, error) {
	if err := apivalidation.CheckGetOptions(req, nil); err != nil {
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
		return nil, grpcutils.InvalidArgWithErr(err)
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(sec); err != nil {
		return nil, err
	}

	sec.Spec = req.Spec
	sec.Data = req.Data

	item, err := s.octeliumC.EnterpriseC().UpdateSecret(ctx, sec)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	item.Data = nil

	return item, nil
}

func (s *Server) validateSecret(ctx context.Context, itm *enterprisev1.Secret) error {

	if err := apivalidation.ValidateCommon(itm, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return err
	}

	if itm.Spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	if itm.Data == nil || itm.Data.Type == nil {
		return grpcutils.InvalidArg("Empty Secret data")
	}

	if itm.Spec.Data != nil {
		switch itm.Spec.Data.Type.(type) {
		case *enterprisev1.Secret_Spec_Data_Value:
			lenVal := len(itm.Spec.Data.GetValue())
			if lenVal == 0 || lenVal > 512*1024 {
				return grpcutils.InvalidArg("Invalid Secret size")
			}
		case *enterprisev1.Secret_Spec_Data_ValueBytes:
			lenVal := len(itm.Spec.Data.GetValueBytes())
			if lenVal == 0 || lenVal > 512*1024 {
				return grpcutils.InvalidArg("Invalid Secret size")
			}
		default:
			return grpcutils.InvalidArg("Invalid Secret data type")
		}

	}

	switch itm.Data.Type.(type) {
	case *enterprisev1.Secret_Data_Value:
		lenVal := len(itm.Data.GetValue())
		if lenVal == 0 || lenVal > 512*1024 {
			return grpcutils.InvalidArg("Invalid Secret size")
		}
	case *enterprisev1.Secret_Data_ValueBytes:
		lenVal := len(itm.Data.GetValueBytes())
		if lenVal == 0 || lenVal > 512*1024 {
			return grpcutils.InvalidArg("Invalid Secret size")
		}
	default:
		return grpcutils.InvalidArg("Invalid Secret data type")
	}

	return nil
}
