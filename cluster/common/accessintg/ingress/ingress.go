// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ingress

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/registry"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/pkg/errors"
)

const providerCacheTTL = 5 * time.Minute

type Server struct {
	octeliumC     octeliumc.ClientInterface
	clusterDomain string

	mu        sync.Mutex
	providers map[string]*cachedProvider
}

type cachedProvider struct {
	provider        accessintg.Provider
	resourceVersion string
	createdAt       time.Time
}

func NewServer(octeliumC octeliumc.ClientInterface, clusterDomain string) *Server {
	return &Server{
		octeliumC:     octeliumC,
		clusterDomain: clusterDomain,
		providers:     map[string]*cachedProvider{},
	}
}

func (s *Server) Handle(ctx context.Context, integrationID string,
	in *accessintg.InboundRequest) (*accessintg.InboundResponse, error) {
	integration, err := s.getIntegration(ctx, integrationID)
	if err != nil {
		return nil, err
	}

	if integration.Spec.IsDisabled {
		return nil, errors.Errorf("The Integration %s is disabled", integration.Metadata.Name)
	}

	provider, err := s.getProvider(ctx, integration)
	if err != nil {
		return nil, err
	}

	handler, ok := provider.(accessintg.InboundHandler)
	if !ok {
		return nil, errors.Errorf("The Integration %s does not accept inbound requests",
			integration.Metadata.Name)
	}

	in, err = stripProviderPath(provider, in)
	if err != nil {
		return nil, err
	}

	inbound, err := handler.DecodeInbound(ctx, in)
	if err != nil {
		return nil, err
	}

	if inbound.Response != nil || inbound.Action == nil {
		if inbound.Response != nil {
			return inbound.Response, nil
		}

		return &accessintg.InboundResponse{}, nil
	}

	res, err := s.executeAction(ctx, integration, provider, inbound.Action)
	if err != nil {
		return nil, err
	}

	return handler.RenderActionResponse(ctx, in, inbound.Action, res), nil
}

func (s *Server) executeAction(ctx context.Context, integration *accessv1.Integration,
	provider accessintg.Provider,
	action *accessintg.Action) (*accessintg.ActionResult, error) {
	switch action.Type {
	case accessintg.ActionTypeSetReviewDecision:
		return s.setReviewDecision(ctx, integration, provider, action)

	case accessintg.ActionTypeCreateRequest:
		return s.createRequest(ctx, integration, provider, action)

	default:
		return nil, errors.Errorf("Invalid inbound action")
	}
}

func (s *Server) getIntegration(ctx context.Context,
	integrationID string) (*accessv1.Integration, error) {
	if integrationID == "" {
		return nil, errors.Errorf("No Integration identifier was provided")
	}

	itemList, err := s.octeliumC.AccessC().ListIntegration(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.id", integrationID),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if len(itemList.Items) != 1 {
		return nil, errors.Errorf("No such Integration")
	}

	return itemList.Items[0], nil
}

func (s *Server) getProvider(ctx context.Context,
	integration *accessv1.Integration) (accessintg.Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cached, ok := s.providers[integration.Metadata.Uid]
	if ok {
		if cached.resourceVersion == integration.Metadata.ResourceVersion &&
			time.Since(cached.createdAt) < providerCacheTTL {
			return cached.provider, nil
		}

		cached.provider.Close()
		delete(s.providers, integration.Metadata.Uid)
	}

	provider, err := registry.New(ctx, s.octeliumC, integration)
	if err != nil {
		return nil, err
	}

	s.providers[integration.Metadata.Uid] = &cachedProvider{
		provider:        provider,
		resourceVersion: integration.Metadata.ResourceVersion,
		createdAt:       time.Now(),
	}

	return provider, nil
}

func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for uid, cached := range s.providers {
		cached.provider.Close()
		delete(s.providers, uid)
	}
}

func stripProviderPath(provider accessintg.Provider,
	in *accessintg.InboundRequest) (*accessintg.InboundRequest, error) {
	segment := providerPathSegment(provider.Type())
	if segment == "" {
		return nil, errors.Errorf("The Integration has an unknown provider")
	}

	if len(in.Path) < 1 || in.Path[0] != segment {
		return nil, errors.Errorf("Invalid path")
	}

	ret := *in
	ret.Path = in.Path[1:]

	return &ret, nil
}

func providerPathSegment(typ accessintg.Type) string {
	switch typ {
	case accessv1.Integration_Status_SLACK:
		return "slack"
	case accessv1.Integration_Status_JIRA:
		return "jira"
	case accessv1.Integration_Status_WEBHOOK:
		return "webhook"
	default:
		return ""
	}
}

func (s *Server) portalURL(path string) string {
	return fmt.Sprintf("https://access.octelium.%s/%s", s.clusterDomain, path)
}
