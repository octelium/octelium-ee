// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"context"

	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *server) GetSecret(ctx context.Context, req *csecretmanv1.GetSecretRequest) (*csecretmanv1.GetSecretResponse, error) {

	res, err := s.doGetDataSecret(ctx, req)
	if err != nil {
		return nil, err
	}

	return &csecretmanv1.GetSecretResponse{
		Data: res.Data,
	}, nil
}

func (s *server) SetSecret(ctx context.Context, req *csecretmanv1.SetSecretRequest) (*csecretmanv1.SetSecretResponse, error) {

	if err := s.doSetDataSecret(ctx, req); err != nil {
		return nil, grpcutils.Internal("Could not update Secret: %+v", err)
	}

	return &csecretmanv1.SetSecretResponse{}, nil
}

func (s *server) DeleteSecret(ctx context.Context, req *csecretmanv1.DeleteSecretRequest) (*csecretmanv1.DeleteSecretResponse, error) {

	err := s.doDeleteDataSecret(ctx, req)
	if err != nil {
		return nil, err
	}

	return &csecretmanv1.DeleteSecretResponse{}, nil
}

func (s *server) ListSecret(ctx context.Context, req *csecretmanv1.ListSecretRequest) (*csecretmanv1.ListSecretResponse, error) {

	return s.doListDataSecret(ctx, req)

}
