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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rlockv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	itemsPerPage             = 500
	verificationGracePeriod  = 10 * time.Minute
	lastSeenRefreshThreshold = 10 * time.Minute
	claimLockTTLSeconds      = 30
	claimLockWaitSeconds     = 10
)

const (
	reasonAmbiguous           = "Ambiguous"
	reasonSourcesDisagree     = "SourcesDisagree"
	reasonConflict            = "Conflict"
	reasonNotApplicable       = "NotApplicable"
	reasonEntryNotFound       = "EntryNotFound"
	reasonEntryAmbiguous      = "EntryAmbiguous"
	reasonOwnerMismatch       = "OwnerMismatch"
	reasonVerificationFailed  = "VerificationFailed"
	reasonVerificationOverdue = "VerificationOverdue"
	reasonDisabled            = "Disabled"
)

type Resolver interface {
	GetOwner(ownerUID string) (*devicemgrcommon.Owner, bool)
	ListOwners() []*devicemgrcommon.Owner
}

type ConditionEvaluator interface {
	MatchesDevice(ctx context.Context, condition *corev1.Condition, dev *corev1.Device) (bool, error)
}

type Controller struct {
	octeliumC octeliumc.ClientInterface
	resolver  Resolver
	evaluator ConditionEvaluator

	deviceLocks sync.Map
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	resolver Resolver,
	evaluator ConditionEvaluator,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		resolver:  resolver,
		evaluator: evaluator,
	}
}

func (c *Controller) OnAdd(ctx context.Context, dev *corev1.Device) error {
	return c.ReconcileDevice(ctx, dev)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Device) error {
	if !isReconcileRequired(new, old) {
		return nil
	}
	return c.ReconcileDevice(ctx, new)
}

func isReconcileRequired(new, old *corev1.Device) bool {
	if old == nil || old.Status == nil || new.GetStatus() == nil {
		return true
	}

	newStatus := new.Status
	oldStatus := old.Status

	newAttempt := newStatus.ProbeAttempt
	if newAttempt.GetState() == corev1.Device_Status_ProbeAttempt_SUBMITTED &&
		(newAttempt.Uid != oldStatus.ProbeAttempt.GetUid() ||
			oldStatus.ProbeAttempt.GetState() != corev1.Device_Status_ProbeAttempt_SUBMITTED) {
		return true
	}

	if oldStatus.Binding != nil && newStatus.Binding == nil {
		return true
	}

	return newStatus.OsType != oldStatus.OsType ||
		newStatus.SerialNumber != oldStatus.SerialNumber ||
		!slices.Equal(newStatus.MacAddresses, oldStatus.MacAddresses) ||
		newStatus.UserRef.GetUid() != oldStatus.UserRef.GetUid()
}

func (c *Controller) OnDelete(ctx context.Context, dev *corev1.Device) error {
	if uid := dev.GetMetadata().GetUid(); uid != "" {
		c.deviceLocks.Delete(uid)
	}
	return nil
}

func (c *Controller) LockDevice(uid string) func() {
	if uid == "" {
		return func() {}
	}

	value, _ := c.deviceLocks.LoadOrStore(uid, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}

func (c *Controller) ReconcileDevice(ctx context.Context, dev *corev1.Device) error {
	if dev == nil {
		return nil
	}
	uid := dev.GetMetadata().GetUid()
	if uid == "" {
		return errors.New("Invalid Device")
	}

	unlock := c.LockDevice(uid)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: uid})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "Could not get Device")
	}

	now := time.Now()

	if dev.Status.Binding.GetState() == corev1.Device_Status_Binding_ACCEPTED {
		if !c.reconcileAccepted(ctx, dev, now) {
			return nil
		}
		return c.updateDevice(ctx, dev)
	}

	cand, changed, err := c.reconcileUnbound(ctx, dev, now)
	if err != nil {
		return err
	}

	if cand != nil {
		return c.accept(ctx, dev, cand, now)
	}

	if !changed {
		return nil
	}

	return c.updateDevice(ctx, dev)
}

type candidate struct {
	owner *devicemgrcommon.Owner
	entry *devicemgrcommon.Entry
}

type decisionKind int

const (
	decisionSkip decisionKind = iota
	decisionWait
	decisionAmbiguous
	decisionCandidate
)

type decision struct {
	kind    decisionKind
	entry   *devicemgrcommon.Entry
	reason  string
	message string
}

func (c *Controller) reconcileUnbound(
	ctx context.Context,
	dev *corev1.Device,
	now time.Time,
) (*candidate, bool, error) {
	owners, err := c.getOrderedOwners(ctx)
	if err != nil {
		return nil, false, err
	}

	results := getAttemptResults(dev)

	for _, owner := range owners {
		dec := c.evaluateOwner(ctx, dev, owner, results, now)

		switch dec.kind {
		case decisionSkip:
			continue
		case decisionWait:
			return nil, false, nil
		case decisionAmbiguous:
			changed := setUnboundBinding(dev, owner, "",
				corev1.Device_Status_Binding_AMBIGUOUS, dec.reason, dec.message)
			changed = markAttemptProcessed(dev) || changed
			return nil, changed, nil
		case decisionCandidate:
			return &candidate{
				owner: owner,
				entry: dec.entry,
			}, false, nil
		}
	}

	changed := false
	if dev.Status.Binding != nil || dev.Status.Posture != nil {
		dev.Status.Binding = nil
		dev.Status.Posture = nil
		changed = true
	}

	return nil, markAttemptProcessed(dev) || changed, nil
}

func (c *Controller) evaluateOwner(
	ctx context.Context,
	dev *corev1.Device,
	owner *devicemgrcommon.Owner,
	results map[string][]*devicemgrcommon.ProbeResult,
	now time.Time,
) *decision {
	if devicemgrcommon.IsDisabled(owner.DM) {
		return &decision{kind: decisionSkip}
	}

	if owner.Manager == nil || !owner.Fresh(now) {
		return &decision{kind: decisionWait}
	}

	applicable, err := c.ownerApplicable(ctx, owner, dev)
	if err != nil {
		zap.L().Warn("Could not evaluate DeviceManager Condition",
			zap.String("device", dev.Metadata.Name),
			zap.String("deviceManager", owner.Name()),
			zap.Error(err))
		return &decision{kind: decisionWait}
	}
	if !applicable {
		return &decision{kind: decisionSkip}
	}

	var probeMatch devicemgrcommon.MatchResult
	if devicemgrcommon.UsesProbe(owner.DM) {
		if ownerResults := results[owner.UID()]; len(ownerResults) > 0 {
			externalID, err := owner.Manager.ParseExternalID(dev.Status.OsType, ownerResults)
			if err != nil {
				return &decision{
					kind:   decisionAmbiguous,
					reason: reasonSourcesDisagree,
					message: fmt.Sprintf("The probe results of the DeviceManager %s are inconsistent: %s",
						owner.Name(), err.Error()),
				}
			}
			if externalID != "" {
				probeMatch = owner.Fleet.MatchProbeID(externalID)
			}
		}
	}

	var identityMatch devicemgrcommon.MatchResult
	if devicemgrcommon.UsesIdentity(owner.DM) {
		identityMatch = owner.Fleet.MatchIdentity(dev.Status.SerialNumber, dev.Status.MacAddresses)
	}

	if probeMatch.State == devicemgrcommon.MatchStateAmbiguous ||
		identityMatch.State == devicemgrcommon.MatchStateAmbiguous {
		return &decision{
			kind:    decisionAmbiguous,
			reason:  reasonAmbiguous,
			message: fmt.Sprintf("The Device matches more than one inventory entry of the DeviceManager %s", owner.Name()),
		}
	}

	if probeMatch.State == devicemgrcommon.MatchStateUnique &&
		identityMatch.State == devicemgrcommon.MatchStateUnique &&
		probeMatch.Entry.ExternalID != identityMatch.Entry.ExternalID {
		return &decision{
			kind:    decisionAmbiguous,
			reason:  reasonSourcesDisagree,
			message: fmt.Sprintf("The probe and identity sources resolve to different inventory entries of the DeviceManager %s", owner.Name()),
		}
	}

	entry := probeMatch.Entry
	if entry == nil {
		entry = identityMatch.Entry
	}

	if entry == nil || entry.ExternalID == "" ||
		(devicemgrcommon.RequireAgreement(owner.DM) &&
			(probeMatch.State != devicemgrcommon.MatchStateUnique ||
				identityMatch.State != devicemgrcommon.MatchStateUnique)) {
		if isAttemptPending(dev, owner.UID(), now) {
			return &decision{kind: decisionWait}
		}
		return &decision{kind: decisionSkip}
	}

	if devicemgrcommon.RequireOwnerMatch(owner.DM) {
		userEmail, err := c.deviceUserEmail(ctx, dev)
		if err != nil {
			zap.L().Warn("Could not get Device User email",
				zap.String("device", dev.Metadata.Name), zap.Error(err))
			return &decision{kind: decisionWait}
		}

		if !devicemgrcommon.OwnerEmailMatches(userEmail, entry.OwnerEmails) {
			return &decision{kind: decisionSkip}
		}
	}

	return &decision{
		kind:  decisionCandidate,
		entry: entry,
	}
}

func (c *Controller) accept(
	ctx context.Context,
	dev *corev1.Device,
	cand *candidate,
	now time.Time,
) error {
	ownerUID := cand.owner.UID()
	externalID := cand.entry.ExternalID

	release, err := c.lockClaim(ctx, ownerUID, externalID)
	if err != nil {
		return err
	}
	defer release()

	holder, err := c.getClaimHolder(ctx, ownerUID, externalID, dev.Metadata.Uid)
	if err != nil {
		return err
	}

	if holder != nil {
		changed := setUnboundBinding(dev, cand.owner, externalID,
			corev1.Device_Status_Binding_CONFLICT, reasonConflict,
			fmt.Sprintf("The inventory entry is already bound to the Device %s", holder.Metadata.Name))
		changed = markAttemptProcessed(dev) || changed
		if !changed {
			return nil
		}
		return c.updateDevice(ctx, dev)
	}

	binding := &corev1.Device_Status_Binding{
		Uid:            newBindingUID(),
		OwnerRef:       cand.owner.OwnerRef(),
		ExternalID:     externalID,
		State:          corev1.Device_Status_Binding_ACCEPTED,
		Validity:       corev1.Device_Status_Binding_VALID,
		AcceptedAt:     pbutils.Timestamp(now),
		LastVerifiedAt: pbutils.Timestamp(now),
	}
	if interval := devicemgrcommon.VerificationInterval(cand.owner.DM); interval > 0 {
		binding.NextVerificationAt = pbutils.Timestamp(now.Add(interval))
	}

	dev.Status.Binding = binding
	dev.Status.Posture = devicemgrcommon.MaterializePosture(cand.owner, cand.entry)
	markAttemptProcessed(dev)

	if err := c.updateDevice(ctx, dev); err != nil {
		return err
	}

	zap.L().Info("Accepted Device binding",
		zap.String("device", dev.Metadata.Name),
		zap.String("deviceManager", cand.owner.Name()),
		zap.String("externalID", externalID))

	return nil
}

type verificationResult int

const (
	verificationNone verificationResult = iota
	verificationSucceeded
	verificationFailed
)

func (c *Controller) reconcileAccepted(ctx context.Context, dev *corev1.Device, now time.Time) bool {
	binding := dev.Status.Binding

	owner, ok := c.resolver.GetOwner(binding.OwnerRef.GetUid())
	if !ok {
		return false
	}

	orig := pbutils.Clone(dev.Status).(*corev1.Device_Status)

	if devicemgrcommon.IsDisabled(owner.DM) {
		binding.Validity = corev1.Device_Status_Binding_SUSPENDED
		binding.Reason = reasonDisabled
		binding.Message = "The DeviceManager is disabled"
		dev.Status.Posture = nil
		markAttemptProcessed(dev)
		return !pbutils.IsEqual(orig, dev.Status)
	}

	if owner.Manager == nil || !owner.Fresh(now) {
		return false
	}

	entry, validity, reason, message, ok := c.checkBinding(ctx, dev, owner)
	if !ok {
		return false
	}

	interval := devicemgrcommon.VerificationInterval(owner.DM)

	if validity == corev1.Device_Status_Binding_VALID {
		switch verifyBinding(dev, owner) {
		case verificationSucceeded:
			binding.LastVerifiedAt = pbutils.Timestamp(now)
			if interval > 0 {
				binding.NextVerificationAt = pbutils.Timestamp(now.Add(interval))
			}
		case verificationFailed:
			validity = corev1.Device_Status_Binding_SUSPENDED
			reason = reasonVerificationFailed
			message = "The probe results do not resolve to the bound inventory entry"
		default:
			switch {
			case binding.Validity == corev1.Device_Status_Binding_SUSPENDED &&
				binding.Reason == reasonVerificationFailed:
				validity = binding.Validity
				reason = binding.Reason
				message = binding.Message
			case interval > 0 && isVerificationOverdue(binding, now):
				validity = corev1.Device_Status_Binding_SUSPENDED
				reason = reasonVerificationOverdue
				message = "The Binding has not been verified within its verification interval"
			}
		}
	}

	markAttemptProcessed(dev)

	switch {
	case interval <= 0:
		binding.NextVerificationAt = nil
	case !binding.NextVerificationAt.IsValid():
		binding.NextVerificationAt = pbutils.Timestamp(now.Add(interval))
	}

	binding.Validity = validity
	binding.Reason = reason
	binding.Message = message

	switch {
	case validity == corev1.Device_Status_Binding_LOST:
		dev.Status.Posture = nil
	case entry != nil:
		refreshPosture(dev, owner, entry, now)
	}

	return !pbutils.IsEqual(orig, dev.Status)
}

func (c *Controller) checkBinding(
	ctx context.Context,
	dev *corev1.Device,
	owner *devicemgrcommon.Owner,
) (*devicemgrcommon.Entry, corev1.Device_Status_Binding_Validity, string, string, bool) {
	applicable, err := c.ownerApplicable(ctx, owner, dev)
	if err != nil {
		zap.L().Warn("Could not evaluate DeviceManager Condition",
			zap.String("device", dev.Metadata.Name),
			zap.String("deviceManager", owner.Name()),
			zap.Error(err))
		return nil, corev1.Device_Status_Binding_VALIDITY_UNKNOWN, "", "", false
	}
	if !applicable {
		return nil, corev1.Device_Status_Binding_LOST, reasonNotApplicable,
			"The DeviceManager no longer applies to the Device", true
	}

	match := owner.Fleet.MatchExternalID(dev.Status.Binding.ExternalID)
	switch match.State {
	case devicemgrcommon.MatchStateNone:
		return nil, corev1.Device_Status_Binding_LOST, reasonEntryNotFound,
			"The inventory entry no longer exists", true
	case devicemgrcommon.MatchStateAmbiguous:
		return nil, corev1.Device_Status_Binding_SUSPENDED, reasonEntryAmbiguous,
			"The inventory entry is no longer unique", true
	}

	if devicemgrcommon.RequireOwnerMatch(owner.DM) {
		userEmail, err := c.deviceUserEmail(ctx, dev)
		if err != nil {
			zap.L().Warn("Could not get Device User email",
				zap.String("device", dev.Metadata.Name), zap.Error(err))
			return nil, corev1.Device_Status_Binding_VALIDITY_UNKNOWN, "", "", false
		}

		if !devicemgrcommon.OwnerEmailMatches(userEmail, match.Entry.OwnerEmails) {
			return nil, corev1.Device_Status_Binding_LOST, reasonOwnerMismatch,
				"The owner of the inventory entry no longer matches the User of the Device", true
		}
	}

	return match.Entry, corev1.Device_Status_Binding_VALID, "", "", true
}

func verifyBinding(dev *corev1.Device, owner *devicemgrcommon.Owner) verificationResult {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil || attempt.State != corev1.Device_Status_ProbeAttempt_SUBMITTED {
		return verificationNone
	}

	if !slices.ContainsFunc(attempt.Probes, func(p *corev1.ClusterConfig_Status_Device_Probe) bool {
		return p.GetOwnerRef().GetUid() == owner.UID()
	}) {
		return verificationNone
	}

	binding := dev.Status.Binding

	isVerificationAttempt := attempt.StartedAt.IsValid() && binding.AcceptedAt.IsValid() &&
		!attempt.StartedAt.AsTime().Before(binding.AcceptedAt.AsTime())

	var match devicemgrcommon.MatchResult
	if results := getAttemptResults(dev)[owner.UID()]; len(results) > 0 {
		if externalID, err := owner.Manager.ParseExternalID(dev.Status.OsType, results); err == nil && externalID != "" {
			match = owner.Fleet.MatchProbeID(externalID)
		}
	}

	switch {
	case match.State == devicemgrcommon.MatchStateUnique && match.Entry.ExternalID == binding.ExternalID:
		return verificationSucceeded
	case match.State != devicemgrcommon.MatchStateNone, isVerificationAttempt:
		return verificationFailed
	default:
		return verificationNone
	}
}

func isVerificationOverdue(binding *corev1.Device_Status_Binding, now time.Time) bool {
	nextAt := binding.GetNextVerificationAt()
	if !nextAt.IsValid() {
		return false
	}

	return !now.Before(nextAt.AsTime().Add(verificationGracePeriod))
}

func refreshPosture(
	dev *corev1.Device,
	owner *devicemgrcommon.Owner,
	entry *devicemgrcommon.Entry,
	now time.Time,
) {
	desired := devicemgrcommon.MaterializePosture(owner, entry)
	current := dev.Status.Posture

	if posturesEqualIgnoringTimestamps(current, desired) &&
		!isPostureRefreshDue(current, desired, now) &&
		!isLastSeenRefreshDue(current, desired) {
		return
	}

	dev.Status.Posture = desired
}

func isLastSeenRefreshDue(current, desired *corev1.Device_Status_Posture) bool {
	currentAt := current.GetLastSeenAt()
	desiredAt := desired.GetLastSeenAt()

	if currentAt.IsValid() != desiredAt.IsValid() {
		return true
	}
	if !desiredAt.IsValid() {
		return false
	}

	diff := desiredAt.AsTime().Sub(currentAt.AsTime())
	return diff >= lastSeenRefreshThreshold || diff <= -lastSeenRefreshThreshold
}

func posturesEqualIgnoringTimestamps(a, b *corev1.Device_Status_Posture) bool {
	return pbutils.IsEqual(normalizePosture(a), normalizePosture(b))
}

func normalizePosture(posture *corev1.Device_Status_Posture) *corev1.Device_Status_Posture {
	if posture == nil {
		return nil
	}

	out := pbutils.Clone(posture).(*corev1.Device_Status_Posture)
	out.LastSyncAt = nil
	out.LastSeenAt = nil
	out.ExpiresAt = nil

	return out
}

func isPostureRefreshDue(current, desired *corev1.Device_Status_Posture, now time.Time) bool {
	if current == nil || !current.LastSyncAt.IsValid() || !current.ExpiresAt.IsValid() {
		return true
	}

	if pbutils.IsEqual(current.ExpiresAt, desired.GetExpiresAt()) {
		return false
	}

	lastSyncAt := current.LastSyncAt.AsTime()
	expiresAt := current.ExpiresAt.AsTime()

	return !now.Before(lastSyncAt.Add(expiresAt.Sub(lastSyncAt) / 2))
}

func (c *Controller) getOrderedOwners(ctx context.Context) ([]*devicemgrcommon.Owner, error) {
	cc, err := c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get enterprise ClusterConfig")
	}

	owners := c.resolver.ListOwners()

	names := cc.GetSpec().GetDeviceManagers()
	if len(names) == 0 {
		sort.SliceStable(owners, func(i, j int) bool {
			return owners[i].Name() < owners[j].Name()
		})
		return owners, nil
	}

	byName := make(map[string]*devicemgrcommon.Owner, len(owners))
	for _, owner := range owners {
		byName[owner.Name()] = owner
	}

	ret := make([]*devicemgrcommon.Owner, 0, len(names))
	for _, name := range names {
		if owner, ok := byName[name]; ok {
			ret = append(ret, owner)
		}
	}

	return ret, nil
}

func (c *Controller) ownerApplicable(
	ctx context.Context,
	owner *devicemgrcommon.Owner,
	dev *corev1.Device,
) (bool, error) {
	if owner == nil || owner.DM == nil {
		return false, nil
	}

	condition := owner.DM.Spec.Condition
	if condition == nil {
		return true, nil
	}

	if c.evaluator == nil {
		return false, errors.New("DeviceManager Condition cannot be evaluated")
	}

	matched, err := c.evaluator.MatchesDevice(ctx, condition, dev)
	if err != nil {
		return false, errors.Wrap(err, "Could not evaluate DeviceManager Condition")
	}

	return matched, nil
}

func (c *Controller) lockClaim(ctx context.Context, ownerUID, externalID string) (func(), error) {
	key := []byte(fmt.Sprintf("nocturne.devicebinding.%s.%s", ownerUID, externalID))

	res, err := c.octeliumC.LockC().Lock(ctx, &rlockv1.LockRequest{
		Key: key,
		Ttl: &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: claimLockTTLSeconds,
			},
		},
		Wait: &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: claimLockWaitSeconds,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not acquire the Device binding lock")
	}
	if !res.Acquired {
		return nil, errors.New("Could not acquire the Device binding lock")
	}

	return func() {
		if _, err := c.octeliumC.LockC().Unlock(context.Background(), &rlockv1.UnlockRequest{
			Key:     key,
			LeaseID: res.LeaseID,
		}); err != nil {
			zap.L().Warn("Could not release the Device binding lock", zap.Error(err))
		}
	}, nil
}

func (c *Controller) getClaimHolder(
	ctx context.Context,
	ownerUID, externalID, deviceUID string,
) (*corev1.Device, error) {
	itemList, err := c.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.binding.ownerRef.uid", ownerUID),
			urscsrv.FilterFieldEQValStr("status.binding.externalID", externalID),
			urscsrv.FilterFieldEQValStr("status.binding.state",
				corev1.Device_Status_Binding_ACCEPTED.String()),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not list the Devices bound to the inventory entry")
	}

	for _, itm := range itemList.Items {
		if itm.Metadata.Uid != deviceUID {
			return itm, nil
		}
	}

	return nil, nil
}

func (c *Controller) ResetDeviceBinding(ctx context.Context, deviceUID, bindingUID string) error {
	if deviceUID == "" {
		return errors.New("Invalid Device uid")
	}

	unlock := c.LockDevice(deviceUID)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: deviceUID})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "Could not get Device")
	}

	binding := dev.Status.Binding
	if binding == nil || (bindingUID != "" && binding.Uid != bindingUID) {
		return nil
	}

	dev.Status.Binding = nil
	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	if err := c.updateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not reset Device binding")
	}

	zap.L().Info("Reset Device binding",
		zap.String("device", dev.Metadata.Name))

	return nil
}

func (c *Controller) ResetBindingsForOwner(ctx context.Context, ownerUID string) error {
	if ownerUID == "" {
		return errors.New("Invalid DeviceManager uid")
	}

	devices, err := c.listDevices(ctx, urscsrv.FilterFieldEQValStr("status.binding.ownerRef.uid", ownerUID))
	if err != nil {
		return errors.Wrap(err, "Could not list Devices while resetting DeviceManager bindings")
	}

	var firstErr error
	for _, dev := range devices {
		if err := c.ResetDeviceBinding(ctx, dev.Metadata.Uid, dev.Status.GetBinding().GetUid()); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			zap.L().Warn("Could not reset Device binding for deleted DeviceManager",
				zap.String("device", dev.Metadata.Name),
				zap.Error(err))
		}
	}

	return firstErr
}

func (c *Controller) listDevices(ctx context.Context, filters ...*rmetav1.ListOptions_Filter) ([]*corev1.Device, error) {
	var ret []*corev1.Device
	var page uint32

	for {
		itmList, err := c.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: itemsPerPage,
			Page:         page,
			Filters:      filters,
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, itmList.Items...)

		if itmList.ListResponseMeta == nil || !itmList.ListResponseMeta.HasMore {
			return ret, nil
		}

		page = page + 1
	}
}

func (c *Controller) deviceUserEmail(ctx context.Context, dev *corev1.Device) (string, error) {
	ref := dev.Status.UserRef
	if ref == nil || ref.GetUid() == "" {
		return "", nil
	}

	user, err := c.octeliumC.CoreC().GetUser(
		ctx,
		apivalidation.ObjectReferenceToRGetOptions(ref),
	)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "Could not resolve Device User")
	}

	return devicemgrcommon.NormalizeEmail(user.Spec.Email), nil
}

func (c *Controller) updateDevice(ctx context.Context, dev *corev1.Device) error {
	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not update Device")
	}

	return nil
}

func setUnboundBinding(
	dev *corev1.Device,
	owner *devicemgrcommon.Owner,
	externalID string,
	state corev1.Device_Status_Binding_State,
	reason, message string,
) bool {
	current := dev.Status.Binding

	isSame := current != nil &&
		current.State == state &&
		current.OwnerRef.GetUid() == owner.UID() &&
		current.ExternalID == externalID

	if isSame && current.Reason == reason && current.Message == message && dev.Status.Posture == nil {
		return false
	}

	uid := newBindingUID()
	if isSame {
		uid = current.Uid
	}

	dev.Status.Binding = &corev1.Device_Status_Binding{
		Uid:        uid,
		OwnerRef:   owner.OwnerRef(),
		ExternalID: externalID,
		State:      state,
		Reason:     reason,
		Message:    message,
	}
	dev.Status.Posture = nil

	return true
}

func getAttemptResults(dev *corev1.Device) map[string][]*devicemgrcommon.ProbeResult {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil {
		return nil
	}

	switch attempt.State {
	case corev1.Device_Status_ProbeAttempt_SUBMITTED,
		corev1.Device_Status_ProbeAttempt_PROCESSED:
	default:
		return nil
	}

	owners := make(map[string]string, len(attempt.Probes))
	for _, p := range attempt.Probes {
		if p != nil {
			owners[p.Id] = p.GetOwnerRef().GetUid()
		}
	}

	ret := make(map[string][]*devicemgrcommon.ProbeResult)
	for _, r := range attempt.Results {
		if r == nil || r.Status != corev1.Device_Status_ProbeAttempt_Result_OK || r.IsTruncated {
			continue
		}

		ownerUID := owners[r.ProbeID]
		if ownerUID == "" {
			continue
		}

		ret[ownerUID] = append(ret[ownerUID], &devicemgrcommon.ProbeResult{
			Text:  r.GetText(),
			Data:  r.GetData(),
			Items: r.GetList().GetItems(),
		})
	}

	return ret
}

func isAttemptPending(dev *corev1.Device, ownerUID string, now time.Time) bool {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil || attempt.State != corev1.Device_Status_ProbeAttempt_ISSUED {
		return false
	}

	if !attempt.ExpiresAt.IsValid() || !now.Before(attempt.ExpiresAt.AsTime()) {
		return false
	}

	return slices.ContainsFunc(attempt.Probes, func(p *corev1.ClusterConfig_Status_Device_Probe) bool {
		return p.GetOwnerRef().GetUid() == ownerUID
	})
}

func markAttemptProcessed(dev *corev1.Device) bool {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil || attempt.State != corev1.Device_Status_ProbeAttempt_SUBMITTED {
		return false
	}

	attempt.State = corev1.Device_Status_ProbeAttempt_PROCESSED
	return true
}

func newBindingUID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
