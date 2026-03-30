// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type middleware struct {
	octeliumC octeliumc.ClientInterface
	next      http.Handler
}

func New(ctx context.Context, next http.Handler, octeliumC octeliumc.ClientInterface) (http.Handler, error) {

	return &middleware{
		next:      next,
		octeliumC: octeliumC,
	}, nil
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	reqCtx, err := m.getContext(req)
	if err != nil {
		zap.L().Debug("Could no get reqCtx", zap.Error(err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rCtx := middlewares.GetCtxRequestContext(req.Context())
	rCtx.Session = reqCtx.Session
	rCtx.DirectoryProvider = reqCtx.DirectoryProvider

	m.next.ServeHTTP(w, req)
}

func (m *middleware) getContext(r *http.Request) (*middlewares.RequestContext, error) {
	ctx := r.Context()

	sessUID := r.Header.Get("X-Octelium-Session-Uid")
	if sessUID == "" {
		return nil, errors.Errorf("Session UID is empty")
	}

	sess, err := m.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
		Uid: sessUID,
	})
	if err != nil {
		return nil, err
	}

	if sess.Status.Type != corev1.Session_Status_CLIENTLESS {
		return nil, errors.Errorf("Session type is not CLIENTLESS")
	}

	if sess.Status.UserRef == nil {
		return nil, errors.Errorf("Nil Session userRef")
	}

	usr, err := m.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
		Uid: sess.Status.UserRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	if usr.Status.Ext == nil {
		return nil, errors.Errorf("Nil User ext")
	}

	info := &enterprisev1.UserExtInfo{}
	if err := pbutils.StructToMessage(usr.Status.Ext[ovutils.ExtInfoKeyEnterprise], info); err != nil {
		return nil, err
	}

	directoryProvider, err := m.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: info.DirectoryProviderRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	if directoryProvider.Spec.IsDisabled {
		return nil, errors.Errorf("directoryProvider is deactivated")
	}

	if directoryProvider.Spec.GetScim() == nil {
		return nil, errors.Errorf("DirectoryProvider is not SCIM")
	}

	// e.g. /scim/<ID>/User/<UID>
	pathArgs := strings.Split(r.URL.Path, "/")
	if len(pathArgs) < 3 {
		return nil, errors.Errorf("Invalid URL: %s", r.URL.Path)
	}

	pathArgs = slices.DeleteFunc(pathArgs, func(itm string) bool {
		return itm == ""
	})

	if pathArgs[0] != "scim" {
		return nil, errors.Errorf("Invalid URL...: %s ... %s", r.URL.Path, pathArgs[0])
	}

	if pathArgs[1] != directoryProvider.Status.Id {
		return nil, errors.Errorf("DirectoryProvider generated ID does not match")
	}

	return &middlewares.RequestContext{
		Session:           sess,
		DirectoryProvider: directoryProvider,
	}, nil
}
