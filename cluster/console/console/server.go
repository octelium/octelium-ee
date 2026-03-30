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
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/httputils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"go.uber.org/zap"
)

type Server struct {
	octeliumC octeliumc.ClientInterface

	svcUID string

	clusterDomain string
	// genCache      *cache.Cache

	rootURL string
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC: octeliumC,
		svcUID:    os.Getenv("OCTELIUM_SVC_UID"),
		// genCache:  cache.New(cache.NoExpiration, 1*time.Minute),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	svc, err := octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
		Uid: ret.svcUID,
	})
	if err != nil {
		return nil, err
	}

	ret.rootURL = fmt.Sprintf("https://%s", vutils.GetServicePublicFQDN(svc, ret.clusterDomain))

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch {

	case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/assets/"):
		s.handleAsset(w, r)
		return
	case r.Method == "GET":
		s.handleIndex(w, r)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) Run(ctx context.Context) error {

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
	zap.L().Info("Console is running...")

	<-ctx.Done()

	return nil
}
