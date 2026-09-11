// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package integrations

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/registry"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxSynchronizations = 10

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	return &Controller{
		octeliumC: octeliumC,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.Integration) error {
	return c.reconcile(ctx, itm)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.Integration) error {
	return c.reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.Integration) error {
	if err := c.deleteIdentities(ctx, itm); err != nil {
		return err
	}

	return c.deleteTargets(ctx, itm)
}

func (c *Controller) reconcile(ctx context.Context, itm *accessv1.Integration) error {
	if itm.Spec.IsDisabled {
		return nil
	}

	if !c.needsSynchronization(itm) {
		return nil
	}

	next := pbutils.Clone(itm).(*accessv1.Integration)

	next.Status.Synchronization = &accessv1.Integration_Status_Synchronization{
		CreatedAt: syncCreatedAt(itm),
		State:     accessv1.Integration_Status_Synchronization_SYNCING,
	}

	next, err := c.octeliumC.AccessC().UpdateIntegration(ctx, next)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	tenant, syncErr := c.validate(ctx, next)

	now := pbutils.Now()

	if syncErr != nil {
		zap.L().Warn("Could not validate the Integration",
			zap.String("integration", next.Metadata.Name), zap.Error(syncErr))

		next.Status.State = accessv1.Integration_Status_ERROR
		if next.Status.LastSuccessAt != nil {
			next.Status.State = accessv1.Integration_Status_DEGRADED
		}
		next.Status.LastFailureAt = now
		next.Status.LastError = syncErr.Error()
		next.Status.Synchronization.State = accessv1.Integration_Status_Synchronization_FAILED
	} else {
		next.Status.State = accessv1.Integration_Status_READY
		next.Status.LastSuccessAt = now
		next.Status.LastError = ""
		next.Status.Synchronization.State = accessv1.Integration_Status_Synchronization_SUCCESS

		if tenant != nil {
			next.Status.ExternalTenantID = tenant.ID
			next.Status.ExternalTenantName = tenant.Name
		}
	}

	next.Status.Synchronization.CompletedAt = now

	appendSynchronization(next)

	if _, err := c.octeliumC.AccessC().UpdateIntegration(ctx, next); err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func (c *Controller) validate(ctx context.Context,
	itm *accessv1.Integration) (*accessintg.TenantInfo, error) {
	provider, err := registry.New(ctx, c.octeliumC, itm)
	if err != nil {
		return nil, err
	}
	defer provider.Close()

	return provider.Validate(ctx)
}

func (c *Controller) needsSynchronization(itm *accessv1.Integration) bool {
	if itm.Status.Synchronization == nil {
		return true
	}

	switch itm.Status.Synchronization.State {
	case accessv1.Integration_Status_Synchronization_SYNC_REQUESTED:
		return true
	default:
		return false
	}
}

func (c *Controller) deleteIdentities(ctx context.Context, itm *accessv1.Integration) error {
	itemList, err := c.octeliumC.AccessC().ListIntegrationIdentity(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", itm.Metadata.Uid),
		},
	})
	if err != nil {
		return err
	}

	for _, identity := range itemList.Items {
		if _, err := c.octeliumC.AccessC().DeleteIntegrationIdentity(ctx, &rmetav1.DeleteOptions{
			Uid: identity.Metadata.Uid,
		}); err != nil && !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete an IntegrationIdentity", zap.Error(err))
		}
	}

	return nil
}

func (c *Controller) deleteTargets(ctx context.Context, itm *accessv1.Integration) error {
	itemList, err := c.octeliumC.AccessC().ListIntegrationTarget(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.integrationRef.uid", itm.Metadata.Uid),
		},
	})
	if err != nil {
		return err
	}

	for _, target := range itemList.Items {
		if _, err := c.octeliumC.AccessC().DeleteIntegrationTarget(ctx, &rmetav1.DeleteOptions{
			Uid: target.Metadata.Uid,
		}); err != nil && !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete an IntegrationTarget", zap.Error(err))
		}
	}

	return nil
}

func syncCreatedAt(itm *accessv1.Integration) *timestamppb.Timestamp {
	if itm.Status.Synchronization != nil && itm.Status.Synchronization.CreatedAt != nil {
		return itm.Status.Synchronization.CreatedAt
	}

	return pbutils.Now()
}

func appendSynchronization(itm *accessv1.Integration) {
	itm.Status.LastSynchronizations = append(
		[]*accessv1.Integration_Status_Synchronization{
			pbutils.Clone(itm.Status.Synchronization).(*accessv1.Integration_Status_Synchronization),
		},
		itm.Status.LastSynchronizations...,
	)

	if len(itm.Status.LastSynchronizations) > maxSynchronizations {
		itm.Status.LastSynchronizations = itm.Status.LastSynchronizations[:maxSynchronizations]
	}
}
