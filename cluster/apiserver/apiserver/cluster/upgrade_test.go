// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUpgradeCluster(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	{
		_, err := srv.UpgradeCluster(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "bad version",
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: strings.Repeat("a", 101),
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		resp, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: " 1.2.3 ",
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, resp)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, cc.Status.UpgradeRequest)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, cc.Status.UpgradeRequest.State)
		assert.Equal(t, "1.2.3", cc.Status.UpgradeRequest.Request.Core.Version)
		assert.NotNil(t, cc.Status.UpgradeRequest.CreatedAt)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
					Version: "1.2.4",
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: timestamppb.New(time.Now().Add(-staleUpgradeRequestAfter - time.Minute)),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
					Version: "2.0.0",
				},
			},
		}
		cc.Status.LastUpgradeRequests = nil
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{
					Version: "3.0.0",
				},
			},
		})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "3.0.0", cc.Status.UpgradeRequest.Request.PackageCordium.Version)
		assert.Len(t, cc.Status.LastUpgradeRequests, 1)
		assert.Equal(t, "2.0.0", cc.Status.LastUpgradeRequests[0].Request.PackageEnterprise.Version)
	}

	{
		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: timestamppb.New(time.Now().Add(-staleUpgradeRequestAfter - time.Minute)),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "4.0.0",
				},
			},
		}
		cc.Status.LastUpgradeRequests = make([]*enterprisev1.ClusterConfig_Status_UpgradeRequest, maxLastUpgradeRequests)
		for i := range cc.Status.LastUpgradeRequests {
			cc.Status.LastUpgradeRequests[i] = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
				CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
				State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS,
				Request: &enterprisev1.UpgradeClusterRequest_Request{
					Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
						Version: "1.0.0",
					},
				},
			}
		}
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
					Version: "5.0.0",
				},
			},
		})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, cc.Status.LastUpgradeRequests, maxLastUpgradeRequests)
		assert.Equal(t, "4.0.0", cc.Status.LastUpgradeRequests[0].Request.Core.Version)
	}
}

func TestValidateAndNormalizeUpgradeRequest(t *testing.T) {
	{
		_, err := validateAndNormalizeUpgradeRequest(nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeUpgradeRequest(&enterprisev1.UpgradeClusterRequest_Request{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		ret, err := validateAndNormalizeUpgradeRequest(&enterprisev1.UpgradeClusterRequest_Request{
			Core: &enterprisev1.UpgradeClusterRequest_Request_Core{},
		})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, ret.Core)
		assert.Equal(t, "latest", ret.Core.Version)
	}

	{
		ret, err := validateAndNormalizeUpgradeRequest(&enterprisev1.UpgradeClusterRequest_Request{
			Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
				Version: " 1.2.3 ",
			},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: "stable",
			},
			PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{
				Version: "main",
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "1.2.3", ret.Core.Version)
		assert.Equal(t, "stable", ret.PackageEnterprise.Version)
		assert.Equal(t, "main", ret.PackageCordium.Version)
	}

	{
		_, err := validateAndNormalizeUpgradeRequest(&enterprisev1.UpgradeClusterRequest_Request{
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: "bad version",
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeUpgradeRequest(&enterprisev1.UpgradeClusterRequest_Request{
			PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{
				Version: strings.Repeat("a", 101),
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestUpgradeRequestStateHelpers(t *testing.T) {
	{
		assert.False(t, isActiveUpgradeRequest(nil))
	}

	{
		assert.False(t, isActiveUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{
			State: enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS,
		}))
	}

	{
		assert.True(t, isActiveUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{
			State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		}))
	}

	{
		assert.True(t, isActiveUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{
			State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		}))
	}

	{
		assert.True(t, isStaleUpgradeRequest(nil, time.Minute))
	}

	{
		assert.True(t, isStaleUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{}, time.Minute))
	}

	{
		assert.True(t, isStaleUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
		}, time.Minute))
	}

	{
		assert.False(t, isStaleUpgradeRequest(&enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: timestamppb.New(time.Now()),
		}, time.Minute))
	}
}
