// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devcontroller

import (
	"context"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	probeResultTTL    = 10 * time.Minute
	defaultStaleAfter = time.Hour
)

type Resolver interface {
	GetOwner(ownerUID string) (*devicemgrcommon.Owner, bool)
}

type Controller struct {
	octeliumC octeliumc.ClientInterface
	resolver  Resolver
}

func NewController(octeliumC octeliumc.ClientInterface, resolver Resolver) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		resolver:  resolver,
	}
}

func (c *Controller) OnAdd(ctx context.Context, dev *corev1.Device) error {
	return c.reconcile(ctx, dev)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Device) error {
	return c.reconcile(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, dev *corev1.Device) error {
	return nil
}

func (c *Controller) reconcile(ctx context.Context, dev *corev1.Device) error {
	probe := dev.Status.GetProbe()
	if probe == nil || len(probe.GetResults()) == 0 {
		return nil
	}

	changed, err := c.processProbe(ctx, dev)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return err
	}
	return nil
}

func (c *Controller) processProbe(ctx context.Context, dev *corev1.Device) (bool, error) {
	probe := dev.Status.Probe

	userEmails, err := c.deviceUserEmails(ctx, dev)
	if err != nil {
		zap.L().Warn("Could not resolve Device user emails", zap.Error(err))
	}

	stale := probe.GetSetAt() != nil && time.Since(probe.GetSetAt().AsTime()) > probeResultTTL

	transient := false
	for _, res := range probe.GetResults() {
		ownerRef := res.GetOwnerRef()
		out := res.GetOutput()
		if ownerRef == nil || len(out) == 0 {
			continue
		}

		owner, ok := c.resolver.GetOwner(ownerRef.GetUid())
		if !ok || owner.Fleet == nil || owner.Manager == nil {
			transient = true
			continue
		}

		externalID, err := owner.Manager.ParseExternalID(dev.Status.GetOsType(),
			[]*devicemgrcommon.ProbeResult{{Output: out}})
		if err != nil {
			zap.L().Warn("Could not parse Device probe output",
				zap.String("device", dev.Metadata.Name), zap.Error(err))
			continue
		}
		if externalID == "" {
			continue
		}

		entry := owner.Fleet.EntryByExternalID(externalID)
		if entry == nil {
			transient = true
			continue
		}

		if !ownershipVerified(userEmails, entry.OwnerEmails) {
			zap.L().Warn("Refusing to link Device: probe device does not belong to the Device user",
				zap.String("device", dev.Metadata.Name),
				zap.String("deviceManager", owner.DM.GetMetadata().GetName()))
			continue
		}

		dev.Status.Posture = buildPosture(ownerRef, externalID, entry, staleAfter(owner.DM))
		dev.Status.Probe = nil
		zap.L().Debug("Linked Device to DeviceManager via probe",
			zap.String("device", dev.Metadata.Name),
			zap.String("deviceManager", owner.DM.GetMetadata().GetName()))
		return true, nil
	}

	if transient && !stale {
		return false, nil
	}

	dev.Status.Probe = nil
	return true, nil
}

func (c *Controller) deviceUserEmails(ctx context.Context, dev *corev1.Device) ([]string, error) {
	ref := dev.Status.GetUserRef()
	if ref == nil {
		return nil, nil
	}
	usr, err := c.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		return nil, err
	}
	var emails []string
	if e := usr.GetSpec().GetEmail(); e != "" {
		emails = append(emails, e)
	}
	return emails, nil
}

func ownershipVerified(userEmails, ownerEmails []string) bool {
	if len(userEmails) == 0 || len(ownerEmails) == 0 {
		return false
	}
	set := map[string]struct{}{}
	for _, e := range userEmails {
		if n := normalizeEmail(e); n != "" {
			set[n] = struct{}{}
		}
	}
	for _, e := range ownerEmails {
		if n := normalizeEmail(e); n != "" {
			if _, ok := set[n]; ok {
				return true
			}
		}
	}
	return false
}

func buildPosture(ownerRef *metav1.ObjectReference, externalID string, entry *devicemgrcommon.Entry, staleAfter time.Duration) *corev1.Device_Status_Posture {
	var p *corev1.Device_Status_Posture
	if entry.Posture != nil {
		p = proto.Clone(entry.Posture).(*corev1.Device_Status_Posture)
	} else {
		p = &corev1.Device_Status_Posture{}
	}
	p.OwnerRef = ownerRef
	p.ExternalID = externalID
	p.LastSyncAt = timestamppb.Now()
	if staleAfter > 0 {
		p.ExpiresAt = timestamppb.New(time.Now().Add(staleAfter))
	}
	return p
}

func staleAfter(dm *enterprisev1.DeviceManager) time.Duration {
	if p := dm.GetSpec().GetPolling(); p != nil && p.StaleAfter != nil {
		if v := umetav1.ToDuration(p.StaleAfter).ToGo(); v > 0 {
			return v
		}
	}
	return defaultStaleAfter
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
