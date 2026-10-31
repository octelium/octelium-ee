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
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

const (
	maxLastJoinRequests   = 200
	staleJoinRequestAfter = 60 * time.Minute

	maxJoinDisplayNameLen = 120
	maxJoinContextLen     = 253
	maxJoinServerLen      = 2048
	maxJoinTrustedCAs     = 16
)

func (s *Server) JoinCluster(ctx context.Context, req *enterprisev1.JoinClusterRequest) (
	*enterprisev1.JoinClusterResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	joinReq, err := validateAndNormalizeJoinRequest(req)
	if err != nil {
		return nil, err
	}

	if err := s.checkJoinRegion(ctx, joinReq.Name); err != nil {
		return nil, err
	}

	if err := s.checkJoinSecret(ctx, getJoinSecretName(joinReq.Kubernetes)); err != nil {
		return nil, err
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cc.Status == nil {
		cc.Status = &enterprisev1.ClusterConfig_Status{}
	}

	if isActiveUpgradeRequest(cc.Status.UpgradeRequest) &&
		!isStaleUpgradeRequest(cc.Status.UpgradeRequest, s.getUpgradeStaleAfter(ctx)) {
		return nil, grpcutils.InvalidArg("Cluster upgrade is currently requested or in progress")
	}

	if cc.Status.JoinRequest != nil {
		if isActiveJoinRequest(cc.Status.JoinRequest) &&
			!isStaleJoinRequest(cc.Status.JoinRequest, staleJoinRequestAfter) {
			return nil, grpcutils.InvalidArg("A Region join is already requested or in progress")
		}

		cc.Status.LastJoinRequests = append(
			[]*enterprisev1.ClusterConfig_Status_JoinRequest{cc.Status.JoinRequest},
			cc.Status.LastJoinRequests...,
		)

		if len(cc.Status.LastJoinRequests) > maxLastJoinRequests {
			cc.Status.LastJoinRequests = cc.Status.LastJoinRequests[:maxLastJoinRequests]
		}
	}

	cc.Status.JoinRequest = &enterprisev1.ClusterConfig_Status_JoinRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED,
		Request:   joinReq,
	}

	if _, err := s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	zap.L().Debug("JoinCluster new request", zap.String("region", joinReq.Name))

	return &enterprisev1.JoinClusterResponse{}, nil
}

func validateAndNormalizeJoinRequest(req *enterprisev1.JoinClusterRequest) (*enterprisev1.JoinClusterRequest, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	if req.Name == "" {
		return nil, grpcutils.InvalidArg("The Region name is required")
	}

	if err := apivalidation.ValidateName(req.Name, 0, 0); err != nil {
		return nil, grpcutils.InvalidArg("Invalid Region name: %s", req.Name)
	}

	if req.Name == regionutils.DefaultRegion {
		return nil, grpcutils.InvalidArg("The default Region cannot be joined")
	}

	ret := &enterprisev1.JoinClusterRequest{
		Name:        req.Name,
		DisplayName: strings.TrimSpace(req.DisplayName),
	}

	if len(ret.DisplayName) > maxJoinDisplayNameLen {
		return nil, grpcutils.InvalidArg("The displayName is too long")
	}

	k8s, err := validateAndNormalizeJoinKubernetes(req.Kubernetes)
	if err != nil {
		return nil, err
	}
	ret.Kubernetes = k8s

	return ret, nil
}

func validateAndNormalizeJoinKubernetes(
	req *enterprisev1.JoinClusterRequest_Kubernetes,
) (*enterprisev1.JoinClusterRequest_Kubernetes, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("The Kubernetes connection is required")
	}

	switch req.Type.(type) {
	case *enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_:
		spec := req.GetKubeconfig()

		secretName, err := validateJoinSecretName(spec.GetFromSecret())
		if err != nil {
			return nil, err
		}

		kubeContext := strings.TrimSpace(spec.Context)
		if len(kubeContext) > maxJoinContextLen {
			return nil, grpcutils.InvalidArg("The kubeconfig context is too long")
		}

		return &enterprisev1.JoinClusterRequest_Kubernetes{
			Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
				Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
					Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
						FromSecret: secretName,
					},
					Context: kubeContext,
				},
			},
		}, nil

	case *enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_:
		spec := req.GetBearerToken()

		secretName, err := validateJoinSecretName(spec.GetFromSecret())
		if err != nil {
			return nil, err
		}

		server, err := validateJoinServer(spec.Server)
		if err != nil {
			return nil, err
		}

		if len(spec.TrustedCAs) > maxJoinTrustedCAs {
			return nil, grpcutils.InvalidArg("Too many trustedCAs")
		}

		var trustedCAs []string
		for _, ca := range spec.TrustedCAs {
			ca = strings.TrimSpace(ca)
			if err := validateJoinTrustedCA(ca); err != nil {
				return nil, err
			}
			trustedCAs = append(trustedCAs, ca)
		}

		return &enterprisev1.JoinClusterRequest_Kubernetes{
			Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_{
				BearerToken: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken{
					Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_FromSecret{
						FromSecret: secretName,
					},
					Server:     server,
					TrustedCAs: trustedCAs,
				},
			},
		}, nil

	default:
		return nil, grpcutils.InvalidArg("You must set either a kubeconfig or a bearerToken")
	}
}

func validateJoinSecretName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", grpcutils.InvalidArg("You must set the fromSecret")
	}

	if err := apivalidation.ValidateName(name, 0, 0); err != nil {
		return "", grpcutils.InvalidArg("Invalid Secret name: %s", name)
	}

	return name, nil
}

func validateJoinServer(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", grpcutils.InvalidArg("The Kubernetes API server URL is required")
	}

	if len(server) > maxJoinServerLen {
		return "", grpcutils.InvalidArg("The Kubernetes API server URL is too long")
	}

	u, err := url.Parse(server)
	if err != nil {
		return "", grpcutils.InvalidArg("Invalid Kubernetes API server URL: %s", server)
	}

	if u.Scheme != "https" {
		return "", grpcutils.InvalidArg("The Kubernetes API server URL must be an https URL")
	}

	if u.Host == "" {
		return "", grpcutils.InvalidArg("The Kubernetes API server URL has no host")
	}

	if u.User != nil {
		return "", grpcutils.InvalidArg("The Kubernetes API server URL must not contain credentials")
	}

	return server, nil
}

func validateJoinTrustedCA(ca string) error {
	block, rest := pem.Decode([]byte(ca))
	if block == nil || block.Type != "CERTIFICATE" {
		return grpcutils.InvalidArg("Every trustedCA must be a PEM-encoded certificate")
	}

	if len(strings.TrimSpace(string(rest))) > 0 {
		return grpcutils.InvalidArg("Every trustedCA must contain a single certificate")
	}

	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return grpcutils.InvalidArg("Could not parse the trustedCA certificate")
	}

	return nil
}

func getJoinSecretName(req *enterprisev1.JoinClusterRequest_Kubernetes) string {
	switch req.Type.(type) {
	case *enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_:
		return req.GetKubeconfig().GetFromSecret()
	case *enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_:
		return req.GetBearerToken().GetFromSecret()
	default:
		return ""
	}
}

func (s *Server) checkJoinRegion(ctx context.Context, name string) error {
	rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return grpcutils.InternalWithErr(err)
	}

	if !regionutils.IsJoinedRegion(rgn) || regionutils.IsJoinCompleted(rgn) {
		return grpcutils.AlreadyExists("The Region %s already exists", name)
	}

	return nil
}

func (s *Server) checkJoinSecret(ctx context.Context, name string) error {
	if _, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: name,
	}); err != nil {
		if grpcerr.IsNotFound(err) {
			return grpcutils.InvalidArg("The Secret %s is not found", name)
		}
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *Server) getUpgradeStaleAfter(ctx context.Context) time.Duration {
	rgnList, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
	if err != nil {
		zap.L().Debug("Could not list Regions to compute the upgrade staleness", zap.Error(err))
		return staleUpgradeRequestAfter
	}

	if len(rgnList.Items) < 2 {
		return staleUpgradeRequestAfter
	}

	return time.Duration(len(rgnList.Items)) * staleUpgradeRequestAfter
}

func isActiveJoinRequest(req *enterprisev1.ClusterConfig_Status_JoinRequest) bool {
	if req == nil {
		return false
	}

	switch req.State {
	case enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED,
		enterprisev1.ClusterConfig_Status_JoinRequest_JOINING:
		return true
	default:
		return false
	}
}

func isStaleJoinRequest(
	req *enterprisev1.ClusterConfig_Status_JoinRequest,
	maxAge time.Duration,
) bool {
	if req == nil || req.CreatedAt == nil || !req.CreatedAt.IsValid() {
		return true
	}

	return time.Since(req.CreatedAt.AsTime()) > maxAge
}
