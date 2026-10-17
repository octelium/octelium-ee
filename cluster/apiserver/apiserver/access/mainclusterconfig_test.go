// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"testing"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestClusterConfig(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	cc, err := srv.GetClusterConfig(ctx, &accessv1.GetClusterConfigRequest{})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "default", cc.Metadata.Name)
	assert.Nil(t, cc.Spec.Authorization)

	setPriority := func(priority *int32) (*accessv1.ClusterConfig, error) {
		req := pbutils.Clone(cc).(*accessv1.ClusterConfig)
		req.Spec = &accessv1.ClusterConfig_Spec{
			Authorization: &accessv1.ClusterConfig_Spec_Authorization{
				Priority: priority,
			},
		}
		return srv.UpdateClusterConfig(ctx, req)
	}

	for _, priority := range []int32{-4, -3, -2, -1, 0} {
		ret, err := setPriority(proto.Int32(priority))
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, ret.Spec.Authorization.Priority)
		assert.Equal(t, priority, ret.Spec.Authorization.GetPriority())

		stored, err := octeliumC.AccessV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, stored.Spec.Authorization.Priority)
		assert.Equal(t, priority, stored.Spec.Authorization.GetPriority())
	}

	{
		ret, err := setPriority(nil)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, ret.Spec.Authorization.Priority)
	}

	for _, priority := range []int32{-5, 1, 4, -100} {
		_, err := setPriority(proto.Int32(priority))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		req := pbutils.Clone(cc).(*accessv1.ClusterConfig)
		req.Spec = nil
		_, err := srv.UpdateClusterConfig(ctx, req)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		req := pbutils.Clone(cc).(*accessv1.ClusterConfig)
		req.Spec = &accessv1.ClusterConfig_Spec{}
		ret, err := srv.UpdateClusterConfig(ctx, req)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, ret.Spec.Authorization)
	}
}
