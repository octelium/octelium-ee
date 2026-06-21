// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watchers

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

type AccessV1Watcher struct {
	octeliumC octeliumc.ClientInterface
}

func NewAccessV1(octeliumC octeliumc.ClientInterface) *AccessV1Watcher {
	return &AccessV1Watcher{
		octeliumC: octeliumC,
	}
}

func (c *AccessV1Watcher) Catalog(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *accessv1.Catalog) error,
	onUpdate func(ctx context.Context, new, old *accessv1.Catalog) error,
	onDelete func(ctx context.Context, item *accessv1.Catalog) error,
) error {
	return runWatcherAccessV1(ctx, c.octeliumC, opts, uaccessv1.KindCatalog, onCreate, onUpdate, onDelete)
}

func (c *AccessV1Watcher) Policy(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *accessv1.Policy) error,
	onUpdate func(ctx context.Context, new, old *accessv1.Policy) error,
	onDelete func(ctx context.Context, item *accessv1.Policy) error,
) error {
	return runWatcherAccessV1(ctx, c.octeliumC, opts, uaccessv1.KindPolicy, onCreate, onUpdate, onDelete)
}

func (c *AccessV1Watcher) Request(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *accessv1.Request) error,
	onUpdate func(ctx context.Context, new, old *accessv1.Request) error,
	onDelete func(ctx context.Context, item *accessv1.Request) error,
) error {
	return runWatcherAccessV1(ctx, c.octeliumC, opts, uaccessv1.KindRequest, onCreate, onUpdate, onDelete)
}

func (c *AccessV1Watcher) Review(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *accessv1.Review) error,
	onUpdate func(ctx context.Context, new, old *accessv1.Review) error,
	onDelete func(ctx context.Context, item *accessv1.Review) error,
) error {
	return runWatcherAccessV1(ctx, c.octeliumC, opts, uaccessv1.KindReview, onCreate, onUpdate, onDelete)
}

func runWatcherAccessV1[T uaccessv1.ResourceObjectRefG](
	ctx context.Context, octeliumC octeliumc.ClientInterface,
	opts *watchers.Opts,
	kind string,
	onCreate func(ctx context.Context, item T) error,
	onUpdate func(ctx context.Context, new, old T) error,
	onDelete func(ctx context.Context, item T) error,
) error {

	var doOnCreate func(ctx context.Context, itm umetav1.ResourceObjectI) error
	var doOnUpdate func(ctx context.Context, new, old umetav1.ResourceObjectI) error
	var doOnDelete func(ctx context.Context, itm umetav1.ResourceObjectI) error

	if onCreate != nil {
		doOnCreate = func(ctx context.Context, itm umetav1.ResourceObjectI) error {
			return onCreate(ctx, itm.(T))
		}
	}

	if onUpdate != nil {
		doOnUpdate = func(ctx context.Context, new, old umetav1.ResourceObjectI) error {
			return onUpdate(ctx, new.(T), old.(T))
		}
	}

	if onDelete != nil {
		doOnDelete = func(ctx context.Context, itm umetav1.ResourceObjectI) error {
			return onDelete(ctx, itm.(T))
		}
	}

	watcher, err := watchers.NewWatcher(uaccessv1.API, uaccessv1.Version, kind,
		doOnCreate, doOnUpdate, doOnDelete,
		octeliumC.AccessC(), func() (umetav1.ResourceObjectI, error) {
			return uaccessv1.NewObject(kind)
		},
	)
	if err != nil {
		return err
	}
	return watcher.Run(ctx)
}
