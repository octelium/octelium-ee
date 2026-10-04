package watchers

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

type CordiumV1Watcher struct {
	octeliumC octeliumc.ClientInterface
}

func NewCordiumV1(octeliumC octeliumc.ClientInterface) *CordiumV1Watcher {
	return &CordiumV1Watcher{
		octeliumC: octeliumC,
	}
}

func (c *CordiumV1Watcher) Workspace(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.Workspace) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.Workspace) error,
	onDelete func(ctx context.Context, item *cordiumv1.Workspace) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindWorkspace, onCreate, onUpdate, onDelete)
}

func (c *CordiumV1Watcher) Template(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.Template) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.Template) error,
	onDelete func(ctx context.Context, item *cordiumv1.Template) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindTemplate, onCreate, onUpdate, onDelete)
}

func (c *CordiumV1Watcher) Space(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.Space) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.Space) error,
	onDelete func(ctx context.Context, item *cordiumv1.Space) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindSpace, onCreate, onUpdate, onDelete)
}

func (c *CordiumV1Watcher) GitProvider(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.GitProvider) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.GitProvider) error,
	onDelete func(ctx context.Context, item *cordiumv1.GitProvider) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindGitProvider, onCreate, onUpdate, onDelete)
}

func (c *CordiumV1Watcher) Secret(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.Secret) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.Secret) error,
	onDelete func(ctx context.Context, item *cordiumv1.Secret) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindSecret, onCreate, onUpdate, onDelete)
}

func (c *CordiumV1Watcher) UserSecret(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *cordiumv1.UserSecret) error,
	onUpdate func(ctx context.Context, new, old *cordiumv1.UserSecret) error,
	onDelete func(ctx context.Context, item *cordiumv1.UserSecret) error,
) error {
	return runWatcherCordiumV1(ctx, c.octeliumC, opts, ucordiumv1.KindUserSecret, onCreate, onUpdate, onDelete)
}

func runWatcherCordiumV1[T ucordiumv1.ResourceObjectRefG](
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

	watcher, err := watchers.NewWatcher(ucordiumv1.API, ucordiumv1.Version, kind,
		doOnCreate, doOnUpdate, doOnDelete,
		octeliumC.CordiumC(), func() (umetav1.ResourceObjectI, error) {
			return ucordiumv1.NewObject(kind)
		},
	)
	if err != nil {
		return err
	}
	return watcher.Run(ctx)
}
