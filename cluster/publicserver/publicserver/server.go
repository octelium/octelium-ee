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
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
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
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string
	rootURL       string

	oidcInfo atomic.Value
}

type oidcInfo struct {
	regionRef *metav1.ObjectReference
	oidc      []byte
	jwks      []byte
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

type jwksJSON struct {
	Keys []json.RawMessage `json:"keys"`
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {
	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret := &Server{
		octeliumC:     octeliumC,
		clusterDomain: cc.Status.Domain,
		rootURL:       fmt.Sprintf("https://public.octelium.%s", cc.Status.Domain),
	}

	ret.oidcInfo.Store(map[string]*oidcInfo{})

	return ret, nil
}

func Run(ctx context.Context) error {
	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
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

func (s *Server) Run(ctx context.Context) error {
	if err := s.doInit(ctx); err != nil {
		return err
	}

	handler, err := s.getHTTPHandler(ctx)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           handler,
		Addr:              vutils.ManagedServiceAddr,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	lis, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	go s.startOIDCRefreshLoop(ctx)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			zap.L().Warn("Could not gracefully shutdown publicserver", zap.Error(err))
		}
	}()

	go func() {
		if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
			zap.L().Error("publicserver HTTP server exited", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) doInit(ctx context.Context) error {
	if err := s.setupK8sOIDC(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) getHTTPHandler(ctx context.Context) (http.Handler, error) {
	chain := httputils.New()

	handler, err := chain.Then(s)
	if err != nil {
		return nil, err
	}

	return handler, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/.well-known"):
		s.handleWellKnown(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func (s *Server) handleWellKnown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 4 ||
		parts[0] != ".well-known" ||
		parts[1] != "regions" {
		http.NotFound(w, r)
		return
	}

	regionUID := parts[2]
	doc := parts[3]

	if !govalidator.IsUUIDv4(regionUID) {
		http.NotFound(w, r)
		return
	}

	info, ok := s.getOIDCInfo(regionUID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")

	switch doc {
	case "openid-configuration":
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		if _, err := w.Write(info.oidc); err != nil {
			zap.L().Debug("Could not write OIDC discovery response", zap.Error(err))
		}

	case "jwks":
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		if _, err := w.Write(info.jwks); err != nil {
			zap.L().Debug("Could not write JWKS response", zap.Error(err))
		}

	default:
		http.NotFound(w, r)
	}
}

func (s *Server) getOIDCInfo(regionUID string) (*oidcInfo, bool) {
	m, ok := s.oidcInfo.Load().(map[string]*oidcInfo)
	if !ok || m == nil {
		return nil, false
	}

	info, ok := m[regionUID]
	return info, ok
}

func (s *Server) startOIDCRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.setupK8sOIDC(ctx); err != nil {
				zap.L().Warn("Could not refresh public OIDC metadata", zap.Error(err))
			}
		}
	}
}

func (s *Server) setupK8sOIDC(ctx context.Context) error {
	regionList, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{
		Paginate:     true,
		ItemsPerPage: 1000,
	})
	if err != nil {
		return err
	}

	newMap := make(map[string]*oidcInfo)

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

		secJWKS, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: ovutils.GetOIDC_JWKSSecretName(region.Metadata.Name),
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return err
		}

		oidcCfg := uenterprisev1.ToSecret(secOIDC).GetValueStr()
		oidcJWKS := uenterprisev1.ToSecret(secJWKS).GetValueStr()

		providerInfo := &oidcProviderJSON{}
		if err := json.Unmarshal([]byte(oidcCfg), providerInfo); err != nil {
			return errors.Errorf("could not unmarshal OIDC provider config for region %s: %+v",
				region.Metadata.Name, err)
		}

		if err := validateOIDCProviderJSON(providerInfo); err != nil {
			return errors.Errorf("invalid OIDC provider config for region %s: %+v",
				region.Metadata.Name, err)
		}

		if err := validateJWKS([]byte(oidcJWKS)); err != nil {
			return errors.Errorf("invalid JWKS for region %s: %+v",
				region.Metadata.Name, err)
		}

		providerInfo.JWKSURL = fmt.Sprintf("%s/.well-known/regions/%s/jwks",
			s.rootURL, region.Metadata.Uid)

		oidcProviderJSON, err := json.Marshal(providerInfo)
		if err != nil {
			return err
		}

		info := &oidcInfo{
			regionRef: umetav1.GetObjectReference(region),
			oidc:      oidcProviderJSON,
			jwks:      []byte(oidcJWKS),
		}

		newMap[region.Metadata.Uid] = info
	}

	s.oidcInfo.Store(newMap)

	return nil
}

func validateOIDCProviderJSON(providerInfo *oidcProviderJSON) error {
	if providerInfo == nil {
		return errors.Errorf("nil OIDC provider config")
	}

	if providerInfo.Issuer == "" {
		return errors.Errorf("missing issuer")
	}

	issuerURL, err := url.Parse(providerInfo.Issuer)
	if err != nil {
		return errors.Errorf("invalid issuer URL: %+v", err)
	}

	if issuerURL.Scheme != "https" {
		return errors.Errorf("issuer must use https")
	}

	if issuerURL.Host == "" {
		return errors.Errorf("issuer host is empty")
	}

	return nil
}

func validateJWKS(jwksBytes []byte) error {
	if len(jwksBytes) == 0 {
		return errors.Errorf("empty JWKS")
	}

	var jwks jwksJSON
	if err := json.Unmarshal(jwksBytes, &jwks); err != nil {
		return err
	}

	if len(jwks.Keys) == 0 {
		return errors.Errorf("JWKS contains no keys")
	}

	return nil
}
