// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dirsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/googleworkspace"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares/auth"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/httputils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"go.uber.org/zap"
)

type server struct {
	octeliumC octeliumc.ClientInterface
	coreSrv   *admin.Server
	dpMap     struct {
		sync.RWMutex
		dpMap map[string]*enterprisev1.DirectoryProvider
	}
}

func newServer(ctx context.Context,
	octeliumC octeliumc.ClientInterface) (*server, error) {

	ret := &server{
		octeliumC: octeliumC,
		coreSrv: admin.NewServer(&admin.Opts{
			OcteliumC:  octeliumC,
			IsEmbedded: true,
		}),
		dpMap: struct {
			sync.RWMutex
			dpMap map[string]*enterprisev1.DirectoryProvider
		}{
			dpMap: make(map[string]*enterprisev1.DirectoryProvider),
		},
	}

	return ret, nil
}

func (s *server) SetDirectoryProvider(dp *enterprisev1.DirectoryProvider) {
	s.dpMap.Lock()
	s.dpMap.dpMap[dp.Metadata.Uid] = dp
	s.dpMap.Unlock()
}

func (s *server) DeleteDirectoryProvider(dp *enterprisev1.DirectoryProvider) {
	s.dpMap.Lock()
	delete(s.dpMap.dpMap, dp.Metadata.Uid)
	s.dpMap.Unlock()
}

func (s *server) run(ctx context.Context) error {

	handler, err := s.getHTTPHandler(ctx)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:      handler,
		Addr:         vutils.ManagedServiceAddr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {

			return middlewares.SetCtxRequestContext(ctx, &middlewares.RequestContext{})
		},
	}

	go func() {
		srv.ListenAndServe()
	}()

	return nil
}

const maxItemsPerPage = 300

func (s *server) handleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	res := &serviceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		Patch: serviceProviderConfigPatch{
			Supported: true,
		},
		Bulk: serviceProviderConfigBulk{
			Supported: false,
		},
		Filter: serviceProviderConfigFilter{
			Supported:  true,
			MaxResults: maxItemsPerPage,
		},
		ChangePassword: serviceProviderConfigChangePassword{
			Supported: false,
		},
		Sort: serviceProviderConfigSort{
			Supported: false,
		},
		Etag: serviceProviderConfigEtag{
			Supported: false,
		},
		AuthenticationSchemes: []serviceProviderConfigAuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "Bearer Token Authentication",
				Primary:     true,
				SpecURI:     "https://www.rfc-editor.org/rfc/rfc6750",
				Description: "Authentication using a static OAuth 2.0 Bearer Token",
			},
		},
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		s.setErrorInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(resBytes)
}

func (s *server) handleSchemas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	resp := &schemasResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		ItemsPerPage: 50,
		StartIndex:   1,
		TotalResults: 2,
		Resources: []schemasResponseResource{
			{
				Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
				ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
				Name:        "User",
				Description: "User Account",
				Attributes: []schemasResponseResourceAttribute{
					{
						Name:        "userName",
						Type:        "string",
						MultiValued: false,
						Description: "Unique identifier for the User",
						Required:    true,
						CaseExact:   false,
						Mutability:  "readWrite",
						Returned:    "default",
						Uniqueness:  "server",
					},
				},
				Meta: schemasResponseResourceAttributeMeta{
					ResourceType: "Schema",
					Location:     "/v2/Schemas/urn:ietf:params:scim:schemas:core:2.0:User",
				},
			},
			{
				Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
				ID:          "urn:ietf:params:scim:schemas:core:2.0:Group",
				Name:        "Group",
				Description: "Group",
				Attributes: []schemasResponseResourceAttribute{
					{
						Name:        "displayName",
						Type:        "string",
						MultiValued: false,
						Description: "A human-readable name for the Group.",
						Required:    false,
						CaseExact:   false,
						Returned:    "default",
						Uniqueness:  "none",
					},
				},
				Meta: schemasResponseResourceAttributeMeta{
					ResourceType: "Schema",
					Location:     "/v2/Schemas/urn:ietf:params:scim:schemas:core:2.0:Group",
				},
			},
		},
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(respBytes)
}

func (s *server) handleResourceTypes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	res := &responseResourceTypes{
		ItemsPerPage: 2,
		TotalResults: 2,
		StartIndex:   1,
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		Resources: []responseResourceTypesResource{
			{
				resourceCommon: resourceCommon{
					Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
					ID:      "User",
				},
				Name:        "User",
				Endpoint:    "/Users",
				Description: "User Account",
				Schema:      "urn:ietf:params:scim:schemas:core:2.0:User",
			},
			{
				resourceCommon: resourceCommon{
					Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
					ID:      "Group",
				},
				Name:        "Group",
				Endpoint:    "/Groups",
				Description: "Group",
				Schema:      "urn:ietf:params:scim:schemas:core:2.0:Group",
			},
		},
	}

	respBytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(respBytes)
}

func (s *server) setError(w http.ResponseWriter, status int, detail string) {
	resp := &responseError{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Status:  fmt.Sprintf("%d", status),
		Detail:  detail,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	w.Write(respBytes)
}

func (s *server) setErrorAlreadyExists(w http.ResponseWriter, err error) {
	zap.L().Debug("already exists request", zap.Error(err))
	s.setError(w, http.StatusConflict, "Resource already exists")
}

func (s *server) setErrorInternal(w http.ResponseWriter, err error) {
	zap.L().Debug("returning internal error", zap.Error(err))
	s.setError(w, http.StatusInternalServerError, "Internal error")
}

func (s *server) setErrorBadRequestWithErr(w http.ResponseWriter, err error) {
	zap.L().Debug("returning bad request", zap.Error(err))
	s.setError(w, http.StatusBadRequest, "Internal error")
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

	if err := s.run(ctx); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortManagedService)
	zap.L().Info("SCIM server is now running...")

	<-ctx.Done()

	return nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqCtx := middlewares.GetCtxRequestContext(r.Context())
	dp := reqCtx.DirectoryProvider
	if reqCtx.DirectoryProvider == nil || reqCtx.Session == nil {
		zap.L().Warn("Could not find reqCtx info. This should not happen in production")
		s.setError(w, 404, "")
		return
	}

	zap.L().Debug("New SCIM req", zap.String("path", r.RequestURI), zap.String("method", r.Method))

	switch {
	case strings.HasPrefix(r.URL.Path, s.getUserPrefix(dp)):
		s.serveUser(w, r, dp)
		return
	case strings.HasPrefix(r.URL.Path, s.getGroupPrefix(dp)):
		s.serveGroup(w, r, dp)
		return
	case r.URL.Path == fmt.Sprintf("%s/ServiceProviderConfig", s.getPrefix(dp)) && r.Method == http.MethodGet:
		s.handleServiceProviderConfig(w, r)
		return
	case r.URL.Path == fmt.Sprintf("%s/ResourceTypes", s.getPrefix(dp)) && r.Method == http.MethodGet:
		s.handleResourceTypes(w, r)
		return
	default:
		zap.L().Debug("Invalid path", zap.String("path", r.URL.Path))
		s.setError(w, 404, "")
		return
	}
}

func (s *server) serveUser(w http.ResponseWriter, r *http.Request, dp *enterprisev1.DirectoryProvider) {

	suffix := strings.TrimPrefix(r.URL.Path, s.getUserPrefix(dp))

	if suffix == "" {
		switch {
		case r.Method == "GET":
			s.handleListUser(w, r)
			return
		case r.Method == "POST":
			s.handleCreateUser(w, r)
			return
		default:
			s.setError(w, 404, "")
			return
		}
	}

	switch {
	case r.Method == "GET":
		s.handleGetUser(w, r)
		return
	case r.Method == "PUT":
		s.handleUpdateUser(w, r)
		return
	case r.Method == "PATCH":
		s.handlePatchUser(w, r)
		return
	case r.Method == "DELETE":
		s.handleDeleteUser(w, r)
		return
	}

	s.setError(w, 404, "")
}

func (s *server) serveGroup(w http.ResponseWriter, r *http.Request, dp *enterprisev1.DirectoryProvider) {

	suffix := strings.TrimPrefix(r.URL.Path, s.getGroupPrefix(dp))

	if suffix == "" {
		switch {
		case r.Method == "GET":
			s.handleListGroup(w, r)
			return
		case r.Method == "POST":
			s.handleCreateGroup(w, r)
			return
		default:
			s.setError(w, 404, "")
			return
		}
	}

	switch {
	case r.Method == "GET":
		s.handleGetGroup(w, r)
		return
	case r.Method == "PUT":
		s.handleUpdateGroup(w, r)
		return
	case r.Method == "PATCH":
		s.handlePatchGroup(w, r)
		return
	case r.Method == "DELETE":
		s.handleDeleteGroup(w, r)
		return
	}

	s.setError(w, 404, "")
}

func (s *server) getHTTPHandler(ctx context.Context) (http.Handler, error) {
	chain := httputils.New()

	chain = chain.Append(func(next http.Handler) (http.Handler, error) {
		return auth.New(ctx, next, s.octeliumC)
	})

	handler, err := chain.Then(s)
	if err != nil {
		return nil, err
	}

	handler, err = chain.Then(handler)
	if err != nil {
		return nil, err
	}

	handler = http.AllowQuerySemicolons(handler)

	return handler, nil
}

func (s *server) getPrefix(dp *enterprisev1.DirectoryProvider) string {
	return fmt.Sprintf("%s/%s", prefixScim, dp.Status.Id)
}

func (s *server) getUserPrefix(dp *enterprisev1.DirectoryProvider) string {
	return fmt.Sprintf("%s%s", s.getPrefix(dp), prefixUser)
}

func (s *server) getGroupPrefix(dp *enterprisev1.DirectoryProvider) string {
	return fmt.Sprintf("%s%s", s.getPrefix(dp), prefixGroup)
}

const prefixScim = "/scim"
const prefixUser = "/Users"
const prefixGroup = "/Groups"

func (s *server) Synchronize(ctx context.Context, dp *enterprisev1.DirectoryProvider) error {

	switch dp.Spec.Type.(type) {
	case *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_:
		p, err := googleworkspace.NewProvider(ctx, s.octeliumC, &googleworkspace.Opts{
			DirectorProviderRef: umetav1.GetObjectReference(dp),
		})
		if err != nil {
			return err
		}

		return p.Synchronize(ctx)
	}

	return nil
}

func getItemsPerPage(r *http.Request) uint32 {
	raw := r.URL.Query().Get("count")
	if raw == "" {
		return maxItemsPerPage
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return maxItemsPerPage
	}
	if val < 0 || val > maxItemsPerPage {
		return maxItemsPerPage
	}

	return uint32(val)
}

func getPage(r *http.Request) uint32 {
	raw := r.URL.Query().Get("startIndex")

	if raw == "" {
		return 0
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	if val < 1 || val > 100000 {
		val = 0
	}

	return uint32(val - 1)
}
