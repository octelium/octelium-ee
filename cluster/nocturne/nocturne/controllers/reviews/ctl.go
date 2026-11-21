// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package reviews

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

type RequestReconciler interface {
	Reconcile(ctx context.Context, itm *accessv1.Request) error
}

type Controller struct {
	octeliumC  octeliumc.ClientInterface
	requestCtl RequestReconciler
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	requestCtl RequestReconciler,
) (*Controller, error) {
	return &Controller{
		octeliumC:  octeliumC,
		requestCtl: requestCtl,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.Review) error {
	if itm.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
		return nil
	}

	return c.reconcile(ctx, itm)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.Review) error {
	return c.reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.Review) error {
	return nil
}

func (c *Controller) reconcile(ctx context.Context, rev *accessv1.Review) error {
	if rev.Status.RequestRef == nil || rev.Status.RequestRef.Uid == "" {
		return errors.Errorf("review %q has no requestRef", rev.Metadata.Name)
	}

	if rev.Status.UserRef == nil || rev.Status.UserRef.Uid == "" {
		return errors.Errorf("review %q has no userRef", rev.Metadata.Name)
	}

	req, err := c.getRequest(ctx, rev.Status.RequestRef)
	if err != nil {
		return err
	}
	if req == nil || !accesscmd.IsPending(req) || accesscmd.GetReviewAction(req) == nil {
		return nil
	}

	return c.requestCtl.Reconcile(ctx, req)
}

func (c *Controller) getRequest(ctx context.Context, ref *metav1.ObjectReference) (*accessv1.Request, error) {
	req, err := c.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid: ref.Uid,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return req, nil
}
