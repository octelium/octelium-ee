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
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
)

type EnterpriseV1Watcher struct {
	octeliumC octeliumc.ClientInterface
}

func NewEnterpriseV1(octeliumC octeliumc.ClientInterface) *EnterpriseV1Watcher {
	return &EnterpriseV1Watcher{
		octeliumC: octeliumC,
	}
}

func (c *EnterpriseV1Watcher) ClusterConfig(
	ctx context.Context,
	opts *watchers.Opts,
	onUpdate func(ctx context.Context, new, old *enterprisev1.ClusterConfig) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindClusterConfig, nil, onUpdate, nil)
}

func (c *EnterpriseV1Watcher) Secret(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.Secret) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.Secret) error,
	onDelete func(ctx context.Context, item *enterprisev1.Secret) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindSecret, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) Certificate(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.Certificate) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.Certificate) error,
	onDelete func(ctx context.Context, item *enterprisev1.Certificate) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindCertificate, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) CertificateIssuer(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.CertificateIssuer) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.CertificateIssuer) error,
	onDelete func(ctx context.Context, item *enterprisev1.CertificateIssuer) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindCertificateIssuer, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) DNSProvider(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.DNSProvider) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.DNSProvider) error,
	onDelete func(ctx context.Context, item *enterprisev1.DNSProvider) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindDNSProvider, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) DirectoryProvider(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.DirectoryProvider) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.DirectoryProvider) error,
	onDelete func(ctx context.Context, item *enterprisev1.DirectoryProvider) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindDirectoryProvider, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) CollectorExporter(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.CollectorExporter) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.CollectorExporter) error,
	onDelete func(ctx context.Context, item *enterprisev1.CollectorExporter) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindCollectorExporter, onCreate, onUpdate, onDelete)
}

func (c *EnterpriseV1Watcher) SecretStore(
	ctx context.Context,
	opts *watchers.Opts,
	onCreate func(ctx context.Context, item *enterprisev1.SecretStore) error,
	onUpdate func(ctx context.Context, new, old *enterprisev1.SecretStore) error,
	onDelete func(ctx context.Context, item *enterprisev1.SecretStore) error,
) error {
	return runWatcherEnterpriseV1(ctx, c.octeliumC, opts, uenterprisev1.KindSecretStore, onCreate, onUpdate, onDelete)
}

func runWatcherEnterpriseV1[T uenterprisev1.ResourceObjectRefG](
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

	watcher, err := watchers.NewWatcher(uenterprisev1.API, uenterprisev1.Version, kind,
		doOnCreate, doOnUpdate, doOnDelete,
		octeliumC.EnterpriseC(), func() (umetav1.ResourceObjectI, error) {
			return uenterprisev1.NewObject(kind)
		},
	)
	if err != nil {
		return err
	}
	return watcher.Run(ctx)
}
