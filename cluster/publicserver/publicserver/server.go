// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package publicserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/httputils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string
	genCache      *cache.Cache

	rootURL string

	oidcInfoMap map[string]*oidcInfo
}

type oidcInfo struct {
	regionRef *metav1.ObjectReference
	oidc      []byte
	jwks      []byte
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC:   octeliumC,
		oidcInfoMap: make(map[string]*oidcInfo),

		genCache: cache.New(cache.NoExpiration, 1*time.Minute),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	ret.rootURL = fmt.Sprintf("https://public.octelium.%s", cc.Status.Domain)

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch {

	case strings.HasPrefix(r.URL.Path, "/.well-known"):
		s.handleWellKnown(w, r)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) Run(ctx context.Context) error {

	if err := s.doInit(ctx); err != nil {
		return err
	}

	srvErr := make(chan error)

	handler, err := s.getHTTPHandler(ctx)
	if err != nil {
		return err
	}

	go func() {
		srv := &http.Server{
			Handler:           handler,
			Addr:              vutils.ManagedServiceAddr,
			WriteTimeout:      15 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			srvErr <- err
		}
	}()

	return nil
}

func (s *Server) getHTTPHandler(ctx context.Context) (http.Handler, error) {
	chain := httputils.New()

	handler, err := chain.Then(s)
	if err != nil {
		return nil, err
	}

	handler = http.AllowQuerySemicolons(handler)

	return handler, nil
}

func Run(ctx context.Context) error {
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := s.Run(ctx); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortManagedService)
	zap.S().Infof("Public EE server is running...")

	<-ctx.Done()

	return nil
}

func (s *Server) doInit(ctx context.Context) error {

	if err := s.setupK8sOIDC(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) handleWellKnown(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	prefix := strings.TrimPrefix(r.URL.Path, "/.well-known")
	if prefix == "" || prefix[0] != '/' {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch {
	case strings.HasPrefix(prefix, "/regions"):

		args := strings.Split(prefix, "/")
		if len(args) < 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		regionUID := args[2]
		req := args[3]

		if !govalidator.IsUUIDv4(regionUID) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		info, ok := s.oidcInfoMap[regionUID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch req {
		case "openid-configuration":
			w.Write(info.oidc)
		case "jwks":
			w.Write(info.jwks)
		}
	}
}

func (s *Server) setupK8sOIDC(ctx context.Context) error {

	regionList, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, region := range regionList.Items {
		secOIDC, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: ovutils.GetOIDCConfigSecretName(region.Metadata.Name),
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return err
		}

		oidcCfg := uenterprisev1.ToSecret(secOIDC).GetValueStr()

		secJWKS, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: ovutils.GetOIDC_JWKSSecretName(region.Metadata.Name),
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return err
		}

		oidcJWKS := uenterprisev1.ToSecret(secJWKS).GetValueStr()

		providerInfo := &oidcProviderJSON{}

		if err := json.Unmarshal([]byte(oidcCfg), providerInfo); err != nil {
			return err
		}

		providerInfo.JWKSURL = fmt.Sprintf("%s/.well-known/regions/%s/jwks", s.rootURL, region.Metadata.Uid)

		oidcProviderJSON, err := json.Marshal(providerInfo)
		if err != nil {
			return err
		}

		oidcInfo := &oidcInfo{
			regionRef: umetav1.GetObjectReference(region),
			oidc:      []byte(oidcProviderJSON),
			jwks:      []byte(oidcJWKS),
		}

		s.oidcInfoMap[region.Metadata.Uid] = oidcInfo
		s.oidcInfoMap[region.Metadata.Name] = oidcInfo
	}

	return nil
}

type oidcProviderJSON struct {
	Issuer                 string   `json:"issuer,omitempty"`
	AuthURL                string   `json:"authorization_endpoint,omitempty"`
	TokenURL               string   `json:"token_endpoint,omitempty"`
	DeviceAuthURL          string   `json:"device_authorization_endpoint,omitempty"`
	JWKSURL                string   `json:"jwks_uri,omitempty"`
	UserInfoURL            string   `json:"userinfo_endpoint,omitempty"`
	Algorithms             []string `json:"id_token_signing_alg_values_supported,omitempty"`
	SubjectTypesSupported  []string `json:"subject_types_supported,omitempty"`
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`
}
