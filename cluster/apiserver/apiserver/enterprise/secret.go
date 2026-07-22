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

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	maxSecretDataBytes = 512 * 1024
	maxSecretKeyBytes  = 256
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
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
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
	if req == nil {
		req = &enterprisev1.ListSecretOptions{}
	}

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
	if err := apivalidation.CheckDeleteOptions(req, nil); err != nil {
		return nil, err
	}

	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
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

	apisrvcommon.MetadataUpdate(sec.Metadata, req.Metadata)
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
	if itm == nil {
		return grpcutils.InvalidArg("Nil Secret")
	}

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

	if itm.Spec.Data != nil {
		if err := validateSecretSpecData(itm.Spec.Data); err != nil {
			return err
		}
	}

	if err := validateSecretData(itm.Data); err != nil {
		return err
	}

	return nil
}

func validateSecretSpecData(data *enterprisev1.Secret_Spec_Data) error {
	if data.Type == nil {
		return grpcutils.InvalidArg("Empty Secret spec data")
	}

	switch data.Type.(type) {
	case *enterprisev1.Secret_Spec_Data_Value:
		return validateSecretPayloadSize(len(data.GetValue()))
	case *enterprisev1.Secret_Spec_Data_ValueBytes:
		return validateSecretPayloadSize(len(data.GetValueBytes()))
	case *enterprisev1.Secret_Spec_Data_Attrs:
		return validateSecretAttrs(data.GetAttrs())
	default:
		return grpcutils.InvalidArg("Invalid Secret spec data type")
	}
}

func validateSecretData(data *enterprisev1.Secret_Data) error {
	if data == nil || data.Type == nil {
		return grpcutils.InvalidArg("Empty Secret data")
	}

	switch data.Type.(type) {
	case *enterprisev1.Secret_Data_Value:
		return validateSecretPayloadSize(len(data.GetValue()))
	case *enterprisev1.Secret_Data_ValueBytes:
		return validateSecretPayloadSize(len(data.GetValueBytes()))
	case *enterprisev1.Secret_Data_DataMap_:
		return validateSecretDataMap(data.GetDataMap())
	case *enterprisev1.Secret_Data_Attrs:
		return validateSecretAttrs(data.GetAttrs())
	default:
		return grpcutils.InvalidArg("Invalid Secret data type")
	}
}

func validateSecretPayloadSize(sz int) error {
	if sz == 0 || sz > maxSecretDataBytes {
		return grpcutils.InvalidArg("Invalid Secret size")
	}

	return nil
}

func validateSecretDataMap(dataMap *enterprisev1.Secret_Data_DataMap) error {
	if dataMap == nil || len(dataMap.GetMap()) == 0 {
		return grpcutils.InvalidArg("Empty Secret data map")
	}

	total := 0
	for k, v := range dataMap.GetMap() {
		if err := validateSecretDataMapKey(k); err != nil {
			return err
		}

		if len(v) == 0 {
			return grpcutils.InvalidArg("Empty Secret data map value")
		}

		total += len(k) + len(v)
		if total > maxSecretDataBytes {
			return grpcutils.InvalidArg("Invalid Secret size")
		}
	}

	return nil
}

func validateSecretDataMapKey(k string) error {
	if k == "" {
		return grpcutils.InvalidArg("Secret data map key is required")
	}

	if len(k) > maxSecretKeyBytes {
		return grpcutils.InvalidArg("Secret data map key is too long")
	}

	if !isSecretASCII(k) || strings.ContainsAny(k, "\x00\r\n") {
		return grpcutils.InvalidArg("Invalid Secret data map key")
	}

	return nil
}

func validateSecretAttrs(attrs *structpb.Struct) error {
	if attrs == nil || len(attrs.GetFields()) == 0 {
		return grpcutils.InvalidArg("Empty Secret attrs")
	}

	if proto.Size(attrs) > maxSecretDataBytes {
		return grpcutils.InvalidArg("Invalid Secret size")
	}

	return nil
}

func isSecretASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}

	return true
}
