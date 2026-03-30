// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package middlewares

import (
	"context"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
)

type RequestContext struct {
	Session           *corev1.Session
	DirectoryProvider *enterprisev1.DirectoryProvider
}

type CtxKey string

const (
	CtxRequestContext CtxKey = "c-req-ctx"
)

func GetCtxRequestContext(ctx context.Context) *RequestContext {
	return ctx.Value(CtxRequestContext).(*RequestContext)
}

func SetCtxRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, CtxRequestContext, reqCtx)
}
