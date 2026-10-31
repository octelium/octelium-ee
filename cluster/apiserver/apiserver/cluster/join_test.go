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

	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func getTestJoinKubeconfig(secretName string) *enterprisev1.JoinClusterRequest_Kubernetes {
	return &enterprisev1.JoinClusterRequest_Kubernetes{
		Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
			Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
					FromSecret: secretName,
				},
			},
		},
	}
}

func getTestJoinCA(t *testing.T) string {
	crt, err := utils_cert.GenerateSelfSignedCert("k8s.example.com", nil, time.Hour)
	assert.Nil(t, err, "%+v", err)
	ret, err := crt.GetCertPEM()
	assert.Nil(t, err, "%+v", err)
	return ret
}

func TestJoinCluster(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	sec, err := tst.C.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: "west-kubeconfig"},
		Spec:     &enterprisev1.Secret_Spec{},
		Status:   &enterprisev1.Secret_Status{},
		Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "kubeconfig"}},
	})
	assert.Nil(t, err, "%+v", err)

	{
		_, err := srv.JoinCluster(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       "default",
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name: "west",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       "west",
			Kubernetes: getTestJoinKubeconfig("missing-kubeconfig"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		rgn, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, &corev1.Region{
			Metadata: &metav1.Metadata{Name: "manual"},
			Spec:     &corev1.Region_Spec{},
			Status:   &corev1.Region_Status{Index: 1},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       rgn.Metadata.Name,
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		rgn := &corev1.Region{
			Metadata: &metav1.Metadata{Name: "joined"},
			Spec:     &corev1.Region_Spec{},
			Status: &corev1.Region_Status{
				Index: 2,
				VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{
					regionutils.PackageEnterprise: {
						Package: regionutils.PackageEnterprise,
						Version: "1.0.0",
					},
				},
			},
		}
		assert.Nil(t, regionutils.SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		}))
		_, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, rgn)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       rgn.Metadata.Name,
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: pbutils.Now(),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "1.0.0",
				},
			},
		}
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       "west",
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)
	}

	{
		resp, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:        "west",
			DisplayName: " West Region ",
			Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
					Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
						Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
							FromSecret: " " + sec.Metadata.Name + " ",
						},
						Context: " admin@west ",
					},
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, resp)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, cc.Status.JoinRequest)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED, cc.Status.JoinRequest.State)
		assert.NotNil(t, cc.Status.JoinRequest.CreatedAt)
		assert.Equal(t, "west", cc.Status.JoinRequest.Request.Name)
		assert.Equal(t, "West Region", cc.Status.JoinRequest.Request.DisplayName)
		assert.Equal(t, sec.Metadata.Name, cc.Status.JoinRequest.Request.Kubernetes.GetKubeconfig().GetFromSecret())
		assert.Equal(t, "admin@west", cc.Status.JoinRequest.Request.Kubernetes.GetKubeconfig().Context)
		assert.Empty(t, cc.Status.LastJoinRequests)
	}

	{
		_, err := srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       "east",
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "1.2.3",
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.JoinRequest.CreatedAt = timestamppb.New(time.Now().Add(-staleJoinRequestAfter - time.Minute))
		cc.Status.JoinRequest.State = enterprisev1.ClusterConfig_Status_JoinRequest_JOINING
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       "east",
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "east", cc.Status.JoinRequest.Request.Name)
		assert.Len(t, cc.Status.LastJoinRequests, 1)
		assert.Equal(t, "west", cc.Status.LastJoinRequests[0].Request.Name)
	}

	{
		rgn := &corev1.Region{
			Metadata: &metav1.Metadata{Name: "retried"},
			Spec:     &corev1.Region_Spec{},
			Status:   &corev1.Region_Status{Index: 3},
		}
		assert.Nil(t, regionutils.SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		}))
		_, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, rgn)
		assert.Nil(t, err, "%+v", err)

		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.JoinRequest.State = enterprisev1.ClusterConfig_Status_JoinRequest_FAILED
		cc.Status.LastJoinRequests = make([]*enterprisev1.ClusterConfig_Status_JoinRequest, maxLastJoinRequests)
		for i := range cc.Status.LastJoinRequests {
			cc.Status.LastJoinRequests[i] = &enterprisev1.ClusterConfig_Status_JoinRequest{
				CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
				State:     enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS,
				Request: &enterprisev1.JoinClusterRequest{
					Name: "old",
				},
			}
		}
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:       rgn.Metadata.Name,
			Kubernetes: getTestJoinKubeconfig(sec.Metadata.Name),
		})
		assert.Nil(t, err, "%+v", err)

		cc, err = srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, rgn.Metadata.Name, cc.Status.JoinRequest.Request.Name)
		assert.Len(t, cc.Status.LastJoinRequests, maxLastJoinRequests)
		assert.Equal(t, "east", cc.Status.LastJoinRequests[0].Request.Name)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, cc.Status.LastJoinRequests[0].State)
	}
}

func TestValidateAndNormalizeJoinRequest(t *testing.T) {
	ca := getTestJoinCA(t)

	{
		_, err := validateAndNormalizeJoinRequest(nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	for _, name := range []string{"", "default", "West", "west_1", "-west", "west--1", strings.Repeat("a", 41)} {
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name:       name,
			Kubernetes: getTestJoinKubeconfig("kubeconfig"),
		})
		assert.NotNil(t, err, name)
		assert.True(t, grpcerr.IsInvalidArg(err), "%s: %+v", name, err)
	}

	{
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name:        "west",
			DisplayName: strings.Repeat("a", maxJoinDisplayNameLen+1),
			Kubernetes:  getTestJoinKubeconfig("kubeconfig"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name:       "west",
			Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name: "west",
			Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
					Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name:       "west",
			Kubernetes: getTestJoinKubeconfig("Invalid_Secret"),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name: "west",
			Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
					Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
						Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
							FromSecret: "kubeconfig",
						},
						Context: strings.Repeat("a", maxJoinContextLen+1),
					},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	getBearerReq := func(server string, cas []string) *enterprisev1.JoinClusterRequest {
		return &enterprisev1.JoinClusterRequest{
			Name: "west",
			Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_{
					BearerToken: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken{
						Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_FromSecret{
							FromSecret: "token",
						},
						Server:     server,
						TrustedCAs: cas,
					},
				},
			},
		}
	}

	for _, server := range []string{
		"",
		"http://203.0.113.10:6443",
		"https://",
		"https://user:pass@203.0.113.10:6443",
		"203.0.113.10:6443",
		"https://" + strings.Repeat("a", maxJoinServerLen),
	} {
		_, err := validateAndNormalizeJoinRequest(getBearerReq(server, nil))
		assert.NotNil(t, err, server)
		assert.True(t, grpcerr.IsInvalidArg(err), "%s: %+v", server, err)
	}

	for _, cas := range [][]string{
		{"not a pem"},
		{"-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----"},
		{ca + ca},
		func() []string {
			var ret []string
			for range maxJoinTrustedCAs + 1 {
				ret = append(ret, ca)
			}
			return ret
		}(),
	} {
		_, err := validateAndNormalizeJoinRequest(getBearerReq("https://203.0.113.10:6443", cas))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		ret, err := validateAndNormalizeJoinRequest(getBearerReq(" https://203.0.113.10:6443 ", []string{"\n" + ca + "\n"}))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "west", ret.Name)
		assert.Equal(t, "token", ret.Kubernetes.GetBearerToken().GetFromSecret())
		assert.Equal(t, "https://203.0.113.10:6443", ret.Kubernetes.GetBearerToken().Server)
		assert.Equal(t, []string{strings.TrimSpace(ca)}, ret.Kubernetes.GetBearerToken().TrustedCAs)
		assert.Equal(t, "token", getJoinSecretName(ret.Kubernetes))
	}

	{
		ret, err := validateAndNormalizeJoinRequest(&enterprisev1.JoinClusterRequest{
			Name:       "west-1",
			Kubernetes: getTestJoinKubeconfig("kubeconfig"),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "west-1", ret.Name)
		assert.Equal(t, "kubeconfig", getJoinSecretName(ret.Kubernetes))
		assert.Empty(t, ret.Kubernetes.GetKubeconfig().Context)
	}

	{
		assert.Empty(t, getJoinSecretName(&enterprisev1.JoinClusterRequest_Kubernetes{}))
	}
}

func TestJoinRequestStateHelpers(t *testing.T) {
	{
		assert.False(t, isActiveJoinRequest(nil))
	}

	for _, state := range []enterprisev1.ClusterConfig_Status_JoinRequest_State{
		enterprisev1.ClusterConfig_Status_JoinRequest_STATE_UNSET,
		enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS,
		enterprisev1.ClusterConfig_Status_JoinRequest_FAILED,
	} {
		assert.False(t, isActiveJoinRequest(&enterprisev1.ClusterConfig_Status_JoinRequest{
			State: state,
		}))
	}

	for _, state := range []enterprisev1.ClusterConfig_Status_JoinRequest_State{
		enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED,
		enterprisev1.ClusterConfig_Status_JoinRequest_JOINING,
	} {
		assert.True(t, isActiveJoinRequest(&enterprisev1.ClusterConfig_Status_JoinRequest{
			State: state,
		}))
	}

	{
		assert.True(t, isStaleJoinRequest(nil, time.Minute))
	}

	{
		assert.True(t, isStaleJoinRequest(&enterprisev1.ClusterConfig_Status_JoinRequest{}, time.Minute))
	}

	{
		assert.True(t, isStaleJoinRequest(&enterprisev1.ClusterConfig_Status_JoinRequest{
			CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
		}, time.Minute))
	}

	{
		assert.False(t, isStaleJoinRequest(&enterprisev1.ClusterConfig_Status_JoinRequest{
			CreatedAt: timestamppb.New(time.Now()),
		}, time.Minute))
	}
}

func TestUpgradeStaleAfterScalesWithRegions(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	assert.Equal(t, staleUpgradeRequestAfter, srv.getUpgradeStaleAfter(ctx))

	for i, name := range []string{"west", "east"} {
		_, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, &corev1.Region{
			Metadata: &metav1.Metadata{Name: name},
			Spec:     &corev1.Region_Spec{},
			Status:   &corev1.Region_Status{Index: int32(i + 1)},
		})
		assert.Nil(t, err, "%+v", err)
	}

	assert.Equal(t, 3*staleUpgradeRequestAfter, srv.getUpgradeStaleAfter(ctx))

	{
		cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err, "%+v", err)
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: timestamppb.New(time.Now().Add(-2 * staleUpgradeRequestAfter)),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "1.0.0",
				},
			},
		}
		_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err, "%+v", err)

		_, err = srv.UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
			Request: &enterprisev1.UpgradeClusterRequest_Request{
				Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
					Version: "1.2.3",
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}
