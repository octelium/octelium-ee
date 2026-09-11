// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package integrationbindings

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/registry"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	baseBackoff = 30 * time.Second
	maxBackoff  = 30 * time.Minute
)

type Controller struct {
	octeliumC     octeliumc.ClientInterface
	clusterDomain string

	mu        sync.Mutex
	inFlight  map[string]struct{}
	providers map[string]*cachedProvider
}

type cachedProvider struct {
	provider        accessintg.Provider
	resourceVersion string
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Controller{
		octeliumC:     octeliumC,
		clusterDomain: cc.Status.Domain,
		inFlight:      map[string]struct{}{},
		providers:     map[string]*cachedProvider{},
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *accessv1.IntegrationBinding) error {
	return c.Reconcile(ctx, itm)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *accessv1.IntegrationBinding) error {
	return c.Reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, itm *accessv1.IntegrationBinding) error {
	return nil
}

func (c *Controller) Reconcile(ctx context.Context, itm *accessv1.IntegrationBinding) error {
	if !c.acquire(itm.Metadata.Uid) {
		return nil
	}
	defer c.release(itm.Metadata.Uid)

	if itm.Status.NextAttemptAt != nil && itm.Status.NextAttemptAt.IsValid() &&
		itm.Status.NextAttemptAt.AsTime().After(time.Now()) {
		return nil
	}

	integration, err := accessintg.GetIntegration(ctx, c.octeliumC, itm.Spec.IntegrationRef)
	if err != nil {
		return err
	}
	if integration == nil || integration.Spec.IsDisabled {
		return nil
	}

	req, err := c.getRequest(ctx, itm)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}

	presentation, err := accessintg.BuildPresentation(ctx, &accessintg.BuildPresentationOpts{
		OcteliumC:     c.octeliumC,
		Binding:       itm,
		Request:       req,
		ClusterDomain: c.clusterDomain,
	})
	if err != nil {
		return err
	}

	desiredState := accessv1.IntegrationBinding_Status_READY
	if isClosed(itm, req) {
		desiredState = accessv1.IntegrationBinding_Status_CLOSED
	}

	revision := presentation.Revision()

	if itm.Status.State == desiredState && itm.Status.AppliedRevision == revision {
		return c.setDesiredRevision(ctx, itm, revision)
	}

	result, deliverErr := c.deliver(ctx, integration, itm, req, presentation, desiredState)

	next := pbutils.Clone(itm).(*accessv1.IntegrationBinding)
	next.Status.DesiredRevision = revision

	if deliverErr != nil {
		zap.L().Warn("Could not deliver an IntegrationBinding presentation",
			zap.String("binding", itm.Metadata.Name), zap.Error(deliverErr))

		next.Status.Attempts = next.Status.Attempts + 1
		next.Status.LastError = deliverErr.Error()
		next.Status.NextAttemptAt = pbutils.Timestamp(time.Now().Add(backoff(next.Status.Attempts)))

		if next.Status.ExternalID == "" {
			next.Status.State = accessv1.IntegrationBinding_Status_PENDING
		} else {
			next.Status.State = accessv1.IntegrationBinding_Status_DEGRADED
		}
	} else {
		next.Status.State = desiredState
		next.Status.AppliedRevision = revision
		next.Status.Attempts = 0
		next.Status.LastError = ""
		next.Status.NextAttemptAt = nil
		next.Status.LastSuccessAt = pbutils.Now()

		if result != nil {
			if result.ExternalID != "" {
				next.Status.ExternalID = result.ExternalID
			}
			if result.ExternalURL != "" {
				next.Status.ExternalURL = result.ExternalURL
			}
			if result.ExternalRecipientID != "" {
				next.Status.ExternalRecipientID = result.ExternalRecipientID
			}
		}
	}

	if pbutils.IsEqual(itm, next) {
		return nil
	}

	if _, err := c.octeliumC.AccessC().UpdateIntegrationBinding(ctx, next); err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func (c *Controller) deliver(ctx context.Context, integration *accessv1.Integration,
	itm *accessv1.IntegrationBinding, req *accessv1.Request,
	presentation *accessintg.Presentation,
	desiredState accessv1.IntegrationBinding_Status_State) (*accessintg.DeliveryResult, error) {
	provider, err := c.getProvider(ctx, integration)
	if err != nil {
		return nil, err
	}

	deliverer, ok := provider.(accessintg.PresentationDeliverer)
	if !ok {
		return nil, errors.Errorf("The Integration %s cannot deliver any presentation",
			integration.Metadata.Name)
	}

	target, err := accessintg.GetIntegrationTarget(ctx, c.octeliumC, itm.Spec.TargetRef)
	if err != nil {
		return nil, err
	}
	if itm.Spec.TargetRef != nil && target == nil {
		return nil, errors.Errorf("The IntegrationTarget of the IntegrationBinding %s no longer exists",
			itm.Metadata.Name)
	}

	recipientID, err := c.getRecipientID(ctx, integration, provider, itm)
	if err != nil {
		return nil, err
	}

	in := &accessintg.PresentationDelivery{
		Binding:      itm,
		Target:       target,
		RecipientID:  recipientID,
		Presentation: presentation,
	}

	isClosing := desiredState == accessv1.IntegrationBinding_Status_CLOSED

	switch {
	case itm.Status.ExternalID == "" && isClosing &&
		itm.Spec.Purpose == accessv1.IntegrationBinding_Spec_REVIEW_SURFACE:
		return &accessintg.DeliveryResult{}, nil

	case itm.Status.ExternalID == "":
		return deliverer.CreatePresentation(ctx, in)

	case isClosing:
		return deliverer.ClosePresentation(ctx, in)

	default:
		return deliverer.UpdatePresentation(ctx, in)
	}
}

func (c *Controller) getRecipientID(ctx context.Context, integration *accessv1.Integration,
	provider accessintg.Provider, itm *accessv1.IntegrationBinding) (string, error) {
	if itm.Spec.UserRef == nil {
		return "", nil
	}

	if itm.Status.ExternalRecipientID != "" {
		return itm.Status.ExternalRecipientID, nil
	}

	usr, err := accesscmd.GetUser(ctx, c.octeliumC, itm.Spec.UserRef)
	if err != nil {
		return "", err
	}
	if usr == nil {
		return "", errors.Errorf("The User of the IntegrationBinding %s no longer exists",
			itm.Metadata.Name)
	}

	opts := &accessintg.ResolveOpts{
		OcteliumC:   c.octeliumC,
		Integration: integration,
	}

	if resolver, ok := provider.(accessintg.IdentityResolver); ok {
		opts.Resolver = resolver
	}

	resolution, err := accessintg.ResolveExternalIDFromUser(ctx, opts, usr)
	if err != nil {
		return "", err
	}

	if !resolution.IsResolved {
		return "", errors.Errorf(
			"Could not resolve the User %s within the Integration %s: %s",
			usr.Metadata.Name, integration.Metadata.Name, resolution.Detail)
	}

	return resolution.ExternalID, nil
}

func (c *Controller) setDesiredRevision(ctx context.Context,
	itm *accessv1.IntegrationBinding, revision string) error {
	if itm.Status.DesiredRevision == revision {
		return nil
	}

	next := pbutils.Clone(itm).(*accessv1.IntegrationBinding)
	next.Status.DesiredRevision = revision

	if _, err := c.octeliumC.AccessC().UpdateIntegrationBinding(ctx, next); err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func (c *Controller) getRequest(ctx context.Context,
	itm *accessv1.IntegrationBinding) (*accessv1.Request, error) {
	if itm.Spec.RequestRef == nil {
		return nil, nil
	}

	req, err := c.octeliumC.AccessC().GetRequest(ctx,
		apivalidation.ObjectReferenceToRGetOptions(itm.Spec.RequestRef))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return req, nil
}

func (c *Controller) getProvider(ctx context.Context,
	integration *accessv1.Integration) (accessintg.Provider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, ok := c.providers[integration.Metadata.Uid]
	if ok {
		if cached.resourceVersion == integration.Metadata.ResourceVersion {
			return cached.provider, nil
		}

		cached.provider.Close()
		delete(c.providers, integration.Metadata.Uid)
	}

	provider, err := registry.New(ctx, c.octeliumC, integration)
	if err != nil {
		return nil, err
	}

	c.providers[integration.Metadata.Uid] = &cachedProvider{
		provider:        provider,
		resourceVersion: integration.Metadata.ResourceVersion,
	}

	return provider, nil
}

func (c *Controller) acquire(uid string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.inFlight[uid]; ok {
		return false
	}

	c.inFlight[uid] = struct{}{}
	return true
}

func (c *Controller) release(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.inFlight, uid)
}

func (c *Controller) DeleteBindingsOf(ctx context.Context, req *accessv1.Request) error {
	itemList, err := c.listBindingsOf(ctx, req)
	if err != nil {
		return err
	}

	for _, itm := range itemList {
		if _, err := c.octeliumC.AccessC().DeleteIntegrationBinding(ctx, &rmetav1.DeleteOptions{
			Uid: itm.Metadata.Uid,
		}); err != nil && !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete an IntegrationBinding", zap.Error(err))
		}
	}

	return nil
}

func (c *Controller) listBindingsOf(ctx context.Context,
	req *accessv1.Request) ([]*accessv1.IntegrationBinding, error) {
	itemList, err := c.octeliumC.AccessC().ListIntegrationBinding(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.requestRef.uid", req.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, err
	}

	return itemList.Items, nil
}

func isClosed(itm *accessv1.IntegrationBinding, req *accessv1.Request) bool {
	if req.Status.State == nil ||
		req.Status.State.Status != accessv1.Request_Status_State_PENDING {
		return true
	}

	if itm.Spec.Purpose != accessv1.IntegrationBinding_Spec_REVIEW_SURFACE {
		return false
	}

	return accesscmd.CurrentStepIndex(req) != itm.Spec.StepIndex
}

func backoff(attempts uint32) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	if attempts > 16 {
		attempts = 16
	}

	ret := time.Duration(math.Pow(2, float64(attempts-1))) * baseBackoff
	if ret > maxBackoff || ret <= 0 {
		return maxBackoff
	}

	return ret
}
