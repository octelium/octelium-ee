// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package requests

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

func (c *Controller) ensureIntegrationBindings(ctx context.Context, itm *accessv1.Request) error {
	req, err := c.getRequest(ctx, itm)
	if err != nil {
		return err
	}
	if req == nil || !hasIntegrationSurfaces(req) {
		return nil
	}

	existing, err := c.listIntegrationBindings(ctx, req)
	if err != nil {
		return err
	}

	desired, err := accessintg.DesiredBindings(ctx, c.octeliumC, req)
	if err != nil {
		return err
	}

	for _, itm := range desired {
		if _, ok := existing[itm.Name]; ok {
			continue
		}

		binding := &accessv1.IntegrationBinding{
			Metadata: &metav1.Metadata{
				Name:     itm.Name,
				IsSystem: true,
			},
			Spec:   &accessv1.IntegrationBinding_Spec{},
			Status: itm.Status,
		}

		revision, err := c.bindingRevision(ctx, binding, req)
		if err != nil {
			return err
		}
		binding.Status.DesiredRevision = revision

		if _, err := c.octeliumC.AccessC().CreateIntegrationBinding(ctx, binding); err != nil {
			if grpcerr.AlreadyExists(err) {
				continue
			}
			return err
		}
	}

	for _, binding := range existing {
		revision, err := c.bindingRevision(ctx, binding, req)
		if err != nil {
			return err
		}

		if binding.Status.DesiredRevision == revision {
			continue
		}

		next := pbutils.Clone(binding).(*accessv1.IntegrationBinding)
		next.Status.DesiredRevision = revision

		if _, err := c.octeliumC.AccessC().UpdateIntegrationBinding(ctx, next); err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return err
		}
	}

	return nil
}

func (c *Controller) deleteIntegrationBindings(ctx context.Context, req *accessv1.Request) error {
	existing, err := c.listIntegrationBindings(ctx, req)
	if err != nil {
		return err
	}

	for _, binding := range existing {
		if _, err := c.octeliumC.AccessC().DeleteIntegrationBinding(ctx, &rmetav1.DeleteOptions{
			Uid: binding.Metadata.Uid,
		}); err != nil && !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete an IntegrationBinding", zap.Error(err))
		}
	}

	return nil
}

func (c *Controller) listIntegrationBindings(ctx context.Context,
	req *accessv1.Request) (map[string]*accessv1.IntegrationBinding, error) {
	itemList, err := c.octeliumC.AccessC().ListIntegrationBinding(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, err
	}

	ret := map[string]*accessv1.IntegrationBinding{}
	for _, itm := range itemList.Items {
		ret[itm.Metadata.Name] = itm
	}

	return ret, nil
}

func (c *Controller) bindingRevision(ctx context.Context,
	binding *accessv1.IntegrationBinding, req *accessv1.Request) (string, error) {
	presentation, err := accessintg.BuildPresentation(ctx, &accessintg.BuildPresentationOpts{
		OcteliumC:     c.octeliumC,
		Binding:       binding,
		Request:       req,
		ClusterDomain: c.clusterDomain,
	})
	if err != nil {
		return "", err
	}

	return presentation.Revision(), nil
}

func (c *Controller) getRequest(ctx context.Context,
	req *accessv1.Request) (*accessv1.Request, error) {
	item, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return item, nil
}

func hasIntegrationSurfaces(req *accessv1.Request) bool {
	if req.Status.Rule == nil {
		return false
	}

	if len(req.Status.Rule.Notifications) > 0 {
		return true
	}

	if req.Status.Rule.Action == nil || req.Status.Rule.Action.GetReview() == nil {
		return false
	}

	for _, step := range req.Status.Rule.Action.GetReview().Steps {
		if step != nil && len(step.Surfaces) > 0 {
			return true
		}
	}

	return false
}
