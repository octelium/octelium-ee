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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	probeAttemptTTL   = 10 * time.Minute
	manualApprovalTTL = 24 * time.Hour
	itemsPerPage      = 500
)

type Resolver interface {
	GetOwner(ownerUID string) (*devicemgrcommon.Owner, bool)
	ListOwners() []*devicemgrcommon.Owner
}

type ConditionEvaluator interface {
	MatchesDevice(ctx context.Context, condition *corev1.Condition, dev *corev1.Device) (bool, error)
}

type bindingKey struct {
	ownerUID   string
	externalID string
}

type Controller struct {
	octeliumC octeliumc.ClientInterface
	resolver  Resolver
	evaluator ConditionEvaluator

	deviceLocks sync.Map

	mu       sync.Mutex
	bindings map[bindingKey]string
	byDevice map[string]bindingKey
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
		bindings:  map[bindingKey]string{},
		byDevice:  map[string]bindingKey{},
	}
}

func (c *Controller) OnAdd(ctx context.Context, dev *corev1.Device) error {
	c.indexDevice(dev)
	return c.ReconcileDevice(ctx, dev)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Device) error {
	c.indexDevice(new)
	return c.ReconcileDevice(ctx, new)
}

func (c *Controller) OnDelete(ctx context.Context, dev *corev1.Device) error {
	if dev != nil {
		c.release(dev.Metadata.Uid)
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
	uid := dev.Metadata.Uid
	if uid == "" {
		return errors.New("Invalid Device")
	}

	unlock := c.LockDevice(uid)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: uid})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			c.release(uid)
			return nil
		}
		return errors.Wrap(err, "Could not get Device")
	}

	now := time.Now()
	changed := false
	claimed := false

	if expireStaleAttempt(dev, now) {
		changed = true
	}

	binding := dev.Status.Binding

	switch {
	case binding == nil,
		binding.State == corev1.Device_Status_Binding_STATE_UNKNOWN,
		binding.State == corev1.Device_Status_Binding_AMBIGUOUS:

		ch, cl, err := c.reconcileUnbound(ctx, dev, now)
		if err != nil {
			return err
		}
		changed = changed || ch
		claimed = cl

	case binding.State == corev1.Device_Status_Binding_WAITING_APPROVAL:
		if approvalExpired(binding, now) {
			c.release(uid)
			dev.Status.Binding = nil
			dev.Status.ProbeAttempt = nil
			changed = true
		}

	case binding.State == corev1.Device_Status_Binding_ACCEPTED,
		binding.State == corev1.Device_Status_Binding_REJECTED:

		if dev.Status.ProbeAttempt != nil {
			dev.Status.ProbeAttempt = nil
			changed = true
		}
	}

	if !changed {
		return nil
	}

	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		if claimed {
			c.release(uid)
		}
		return errors.Wrap(err, "Could not update Device binding state")
	}

	return nil
}

type ownerCandidate struct {
	owner        *devicemgrcommon.Owner
	entry        *devicemgrcommon.Entry
	fromProbe    bool
	fromIdentity bool
}

func (c *Controller) reconcileUnbound(
	ctx context.Context,
	dev *corev1.Device,
	now time.Time,
) (bool, bool, error) {
	candidates := map[string]*ownerCandidate{}
	var ambiguousReasons []string

	if err := c.collectProbeCandidates(ctx, dev, now, candidates, &ambiguousReasons); err != nil {
		return false, false, err
	}
	if err := c.collectIdentityCandidates(ctx, dev, now, candidates, &ambiguousReasons); err != nil {
		return false, false, err
	}

	selected, reasons := selectCandidate(candidates)
	ambiguousReasons = append(ambiguousReasons, reasons...)

	if selected == nil {
		if len(ambiguousReasons) > 0 {
			return setAmbiguous(dev, strings.Join(ambiguousReasons, "; ")), false, nil
		}
		return clearAmbiguous(dev), false, nil
	}

	deviceUID := dev.Metadata.Uid
	ownerUID := selected.owner.UID()
	externalID := selected.entry.ExternalID
	dmName := selected.owner.DM.Metadata.Name

	switch devicemgrcommon.ApprovalMode(selected.owner.DM) {
	case enterprisev1.DeviceManager_Spec_Linking_AUTOMATIC:
		if !c.claim(ownerUID, externalID, deviceUID) {
			return c.rejectTaken(dev, selected), false, nil
		}
		c.accept(dev, selected, corev1.Device_Status_Binding_AUTOMATIC)
		return true, true, nil

	case enterprisev1.DeviceManager_Spec_Linking_EMAIL:
		userEmail, err := c.deviceUserEmail(ctx, dev)
		if err != nil {
			return false, false, err
		}
		if !devicemgrcommon.OwnerEmailMatches(userEmail, selected.entry.OwnerEmails) {
			zap.L().Debug("Device binding candidate skipped: owner email does not match Device user",
				zap.String("device", dev.Metadata.Name),
				zap.String("deviceManager", dmName))
			return clearAmbiguous(dev), false, nil
		}
		if !c.claim(ownerUID, externalID, deviceUID) {
			return c.rejectTaken(dev, selected), false, nil
		}
		c.accept(dev, selected, corev1.Device_Status_Binding_EMAIL)
		return true, true, nil

	case enterprisev1.DeviceManager_Spec_Linking_MANUAL:
		if !c.claim(ownerUID, externalID, deviceUID) {
			return c.rejectTaken(dev, selected), false, nil
		}
		c.waitForApproval(dev, selected, now)
		return true, true, nil

	default:
		return false, false, errors.Errorf("Unsupported DeviceManager approval mode for %s", dmName)
	}
}

func (c *Controller) collectProbeCandidates(
	ctx context.Context,
	dev *corev1.Device,
	now time.Time,
	candidates map[string]*ownerCandidate,
	ambiguousReasons *[]string,
) error {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil || len(attempt.Results) == 0 || attemptExpired(attempt, now) {
		return nil
	}

	resultsByOwner := map[string][]*devicemgrcommon.ProbeResult{}
	for _, result := range attempt.Results {
		if result == nil {
			continue
		}

		idx, err := strconv.Atoi(result.ProbeID)
		if err != nil || idx < 0 || idx >= len(attempt.Probes) {
			continue
		}

		probe := attempt.Probes[idx]
		if probe == nil {
			continue
		}
		ownerUID := probe.OwnerRef.GetUid()
		if ownerUID == "" {
			continue
		}

		parsed := &devicemgrcommon.ProbeResult{}
		if output := result.GetOutput(); len(output) > 0 {
			parsed.Output = append([]byte(nil), output...)
		} else if result.GetError() != "" {
			parsed.Err = errors.New(result.GetError())
		} else {
			continue
		}

		resultsByOwner[ownerUID] = append(resultsByOwner[ownerUID], parsed)
	}

	ownerUIDs := make([]string, 0, len(resultsByOwner))
	for ownerUID := range resultsByOwner {
		ownerUIDs = append(ownerUIDs, ownerUID)
	}
	sort.Strings(ownerUIDs)

	for _, ownerUID := range ownerUIDs {
		owner, ok := c.resolver.GetOwner(ownerUID)
		if !ok || owner == nil || owner.Manager == nil || owner.Fleet == nil || !owner.Fresh(now) {
			continue
		}

		if !devicemgrcommon.UsesProbe(owner.DM) {
			continue
		}

		applicable, err := c.ownerApplicable(ctx, owner, dev)
		if err != nil {
			return err
		}
		if !applicable {
			continue
		}

		externalID, err := owner.Manager.ParseExternalID(dev.Status.OsType, resultsByOwner[ownerUID])
		if err != nil {
			zap.L().Warn("Could not parse Device probe output",
				zap.String("device", dev.Metadata.Name),
				zap.String("deviceManager", owner.DM.Metadata.Name),
				zap.Error(err))
			continue
		}
		if externalID == "" {
			continue
		}

		match := owner.Fleet.MatchExternalID(externalID)
		switch match.State {
		case devicemgrcommon.MatchStateAmbiguous:
			*ambiguousReasons = append(*ambiguousReasons,
				fmt.Sprintf("%s: external ID is not unique in provider inventory", owner.DM.Metadata.Name))
		case devicemgrcommon.MatchStateUnique:
			mergeCandidate(candidates, ambiguousReasons, owner, match.Entry, true, false)
		}
	}

	return nil
}

func (c *Controller) collectIdentityCandidates(
	ctx context.Context,
	dev *corev1.Device,
	now time.Time,
	candidates map[string]*ownerCandidate,
	ambiguousReasons *[]string,
) error {
	for _, owner := range c.resolver.ListOwners() {
		if owner == nil || owner.Fleet == nil || !owner.Fresh(now) {
			continue
		}

		if !devicemgrcommon.UsesIdentity(owner.DM) {
			continue
		}

		applicable, err := c.ownerApplicable(ctx, owner, dev)
		if err != nil {
			return err
		}
		if !applicable {
			continue
		}

		match := owner.Fleet.MatchIdentity(
			dev.Status.SerialNumber,
			dev.Status.MacAddresses,
		)
		switch match.State {
		case devicemgrcommon.MatchStateAmbiguous:
			*ambiguousReasons = append(*ambiguousReasons,
				fmt.Sprintf("%s: identity attributes are not unique in provider inventory", owner.DM.Metadata.Name))
		case devicemgrcommon.MatchStateUnique:
			mergeCandidate(candidates, ambiguousReasons, owner, match.Entry, false, true)
		}
	}

	return nil
}

func mergeCandidate(
	candidates map[string]*ownerCandidate,
	ambiguousReasons *[]string,
	owner *devicemgrcommon.Owner,
	entry *devicemgrcommon.Entry,
	fromProbe bool,
	fromIdentity bool,
) {
	ownerUID := owner.UID()

	existing, ok := candidates[ownerUID]
	if !ok {
		candidates[ownerUID] = &ownerCandidate{
			owner:        owner,
			entry:        entry,
			fromProbe:    fromProbe,
			fromIdentity: fromIdentity,
		}
		return
	}

	if existing.entry.ExternalID != entry.ExternalID {
		*ambiguousReasons = append(*ambiguousReasons,
			fmt.Sprintf("%s: probe and identity resolve to different inventory entries", owner.DM.Metadata.Name))
		delete(candidates, ownerUID)
		return
	}

	existing.fromProbe = existing.fromProbe || fromProbe
	existing.fromIdentity = existing.fromIdentity || fromIdentity
}

func selectCandidate(candidates map[string]*ownerCandidate) (*ownerCandidate, []string) {
	var eligible []*ownerCandidate

	for _, cand := range candidates {
		if cand == nil || cand.owner == nil || cand.entry == nil || cand.entry.ExternalID == "" {
			continue
		}

		linking := cand.owner.DM.Spec.Linking
		if linking.GetRequireAgreement() && !(cand.fromProbe && cand.fromIdentity) {
			continue
		}

		eligible = append(eligible, cand)
	}

	if len(eligible) == 0 {
		return nil, nil
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].owner.UID() < eligible[j].owner.UID()
	})

	var best []*ownerCandidate
	var bestPriority uint32

	for _, cand := range eligible {
		priority := cand.owner.DM.Spec.Linking.GetPriority()
		switch {
		case len(best) == 0 || priority > bestPriority:
			bestPriority = priority
			best = []*ownerCandidate{cand}
		case priority == bestPriority:
			best = append(best, cand)
		}
	}

	if len(best) == 1 {
		return best[0], nil
	}

	names := make([]string, 0, len(best))
	for _, cand := range best {
		names = append(names, cand.owner.DM.Metadata.Name)
	}
	return nil, []string{
		fmt.Sprintf("multiple DeviceManagers match with equal priority: %s", strings.Join(names, ", ")),
	}
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

func (c *Controller) accept(
	dev *corev1.Device,
	selected *ownerCandidate,
	method corev1.Device_Status_Binding_AcceptanceMethod,
) {
	dev.Status.Binding = &corev1.Device_Status_Binding{
		Uid:              newBindingUID(),
		OwnerRef:         selected.owner.OwnerRef(),
		ExternalID:       selected.entry.ExternalID,
		State:            corev1.Device_Status_Binding_ACCEPTED,
		AcceptanceMethod: method,
		AcceptedAt:       pbutils.Now(),
	}
	dev.Status.Posture = devicemgrcommon.MaterializePosture(selected.owner, selected.entry)
	dev.Status.ProbeAttempt = nil

	zap.L().Info("Accepted Device binding",
		zap.String("device", dev.Metadata.Name),
		zap.String("deviceManager", selected.owner.DM.Metadata.Name),
		zap.String("externalID", selected.entry.ExternalID))
}

func (c *Controller) waitForApproval(
	dev *corev1.Device,
	selected *ownerCandidate,
	now time.Time,
) {
	dev.Status.Binding = &corev1.Device_Status_Binding{
		Uid:        newBindingUID(),
		OwnerRef:   selected.owner.OwnerRef(),
		ExternalID: selected.entry.ExternalID,
		State:      corev1.Device_Status_Binding_WAITING_APPROVAL,
		ExpiresAt:  pbutils.Timestamp(now.Add(manualApprovalTTL)),
	}
	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	zap.L().Info("Device binding is waiting for administrator approval",
		zap.String("device", dev.Metadata.Name),
		zap.String("deviceManager", selected.owner.DM.Metadata.Name),
		zap.String("externalID", selected.entry.ExternalID))
}

func (c *Controller) rejectTaken(dev *corev1.Device, selected *ownerCandidate) bool {
	holder := c.holderOf(selected.owner.UID(), selected.entry.ExternalID)

	dev.Status.Binding = &corev1.Device_Status_Binding{
		Uid:        newBindingUID(),
		OwnerRef:   selected.owner.OwnerRef(),
		ExternalID: selected.entry.ExternalID,
		State:      corev1.Device_Status_Binding_REJECTED,
		Message: fmt.Sprintf(
			"External ID is already bound to another Device (%s)", holder),
	}
	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	zap.L().Warn("Rejected Device binding: external ID already bound",
		zap.String("device", dev.Metadata.Name),
		zap.String("deviceManager", selected.owner.DM.Metadata.Name),
		zap.String("externalID", selected.entry.ExternalID),
		zap.String("boundDevice", holder))

	return true
}

func setAmbiguous(dev *corev1.Device, message string) bool {
	binding := dev.Status.Binding
	if binding != nil &&
		binding.State == corev1.Device_Status_Binding_AMBIGUOUS &&
		binding.Message == message {
		return false
	}

	dev.Status.Binding = &corev1.Device_Status_Binding{
		Uid:     newBindingUID(),
		State:   corev1.Device_Status_Binding_AMBIGUOUS,
		Message: message,
	}
	dev.Status.Posture = nil

	return true
}

func clearAmbiguous(dev *corev1.Device) bool {
	binding := dev.Status.Binding
	if binding == nil || binding.State != corev1.Device_Status_Binding_AMBIGUOUS {
		return false
	}

	dev.Status.Binding = nil
	return true
}

func (c *Controller) ApproveBinding(ctx context.Context, deviceUID, bindingUID string) error {
	if deviceUID == "" || bindingUID == "" {
		return errors.New("Invalid Device or Binding uid")
	}

	unlock := c.LockDevice(deviceUID)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: deviceUID})
	if err != nil {
		return errors.Wrap(err, "Could not get Device")
	}

	binding := dev.Status.Binding
	if binding == nil ||
		binding.Uid != bindingUID ||
		binding.State != corev1.Device_Status_Binding_WAITING_APPROVAL {
		return errors.New("Device does not have the requested pending binding")
	}

	now := time.Now()

	if approvalExpired(binding, now) {
		c.release(deviceUID)
		dev.Status.Binding = nil
		dev.Status.ProbeAttempt = nil
		if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
			return err
		}
		return errors.New("Pending Device binding has expired")
	}

	owner, ok := c.resolver.GetOwner(binding.OwnerRef.GetUid())
	if !ok || owner == nil || !owner.Fresh(now) {
		return errors.New("DeviceManager inventory snapshot is unavailable or stale")
	}

	if devicemgrcommon.ApprovalMode(owner.DM) != enterprisev1.DeviceManager_Spec_Linking_MANUAL {
		return errors.New("DeviceManager no longer requires manual approval")
	}

	match := owner.Fleet.MatchExternalID(binding.ExternalID)
	if match.State != devicemgrcommon.MatchStateUnique {
		return errors.New("DeviceManager external ID no longer resolves uniquely")
	}

	if !c.claim(owner.UID(), match.Entry.ExternalID, deviceUID) {
		return errors.New("DeviceManager external ID is already bound to another Device")
	}

	binding.State = corev1.Device_Status_Binding_ACCEPTED
	binding.AcceptanceMethod = corev1.Device_Status_Binding_MANUAL
	binding.AcceptedAt = pbutils.Now()
	binding.ExpiresAt = nil
	binding.Message = ""

	dev.Status.Posture = devicemgrcommon.MaterializePosture(owner, match.Entry)
	dev.Status.ProbeAttempt = nil

	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not approve Device binding")
	}

	return nil
}

func (c *Controller) RejectBinding(ctx context.Context, deviceUID, bindingUID string) error {
	if deviceUID == "" || bindingUID == "" {
		return errors.New("Invalid Device or Binding uid")
	}

	unlock := c.LockDevice(deviceUID)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: deviceUID})
	if err != nil {
		return errors.Wrap(err, "Could not get Device")
	}

	binding := dev.Status.Binding
	if binding == nil ||
		binding.Uid != bindingUID ||
		binding.State != corev1.Device_Status_Binding_WAITING_APPROVAL {
		return errors.New("Device does not have the requested pending binding")
	}

	c.release(deviceUID)

	binding.State = corev1.Device_Status_Binding_REJECTED
	binding.AcceptanceMethod = corev1.Device_Status_Binding_ACCEPTANCE_METHOD_UNKNOWN
	binding.ExpiresAt = nil
	binding.Message = "Rejected by administrator"

	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not reject Device binding")
	}

	return nil
}

func (c *Controller) ResetDeviceBinding(ctx context.Context, deviceUID string) error {
	if deviceUID == "" {
		return errors.New("Invalid Device uid")
	}

	unlock := c.LockDevice(deviceUID)
	defer unlock()

	dev, err := c.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{Uid: deviceUID})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			c.release(deviceUID)
			return nil
		}
		return errors.Wrap(err, "Could not get Device")
	}

	if dev.Status.Binding == nil && dev.Status.Posture == nil && dev.Status.ProbeAttempt == nil {
		return nil
	}

	c.release(deviceUID)

	dev.Status.Binding = nil
	dev.Status.Posture = nil
	dev.Status.ProbeAttempt = nil

	if _, err := c.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
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

	devices, err := c.listDevices(ctx)
	if err != nil {
		return errors.Wrap(err, "Could not list Devices while resetting DeviceManager bindings")
	}

	var firstErr error
	for _, dev := range devices {
		if dev == nil {
			continue
		}

		binding := dev.Status.Binding
		if binding == nil || binding.OwnerRef.GetUid() != ownerUID {
			continue
		}

		if err := c.ResetDeviceBinding(ctx, dev.Metadata.Uid); err != nil {
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

func (c *Controller) listDevices(ctx context.Context) ([]*corev1.Device, error) {
	var ret []*corev1.Device
	var page uint32

	for {
		itmList, err := c.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: itemsPerPage,
			Page:         page,
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

func (c *Controller) indexDevice(dev *corev1.Device) {
	if dev == nil {
		return
	}
	deviceUID := dev.Metadata.Uid
	if deviceUID == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.releaseLocked(deviceUID)

	binding := dev.Status.Binding
	if binding == nil {
		return
	}

	switch binding.State {
	case corev1.Device_Status_Binding_ACCEPTED,
		corev1.Device_Status_Binding_WAITING_APPROVAL:
	default:
		return
	}

	ownerUID := binding.OwnerRef.GetUid()
	externalID := binding.ExternalID
	if ownerUID == "" || externalID == "" {
		return
	}

	key := bindingKey{ownerUID: ownerUID, externalID: externalID}

	if current, ok := c.bindings[key]; ok && current != deviceUID {
		zap.L().Warn("Duplicate Device binding observed for the same external ID",
			zap.String("externalID", externalID),
			zap.String("device", deviceUID),
			zap.String("boundDevice", current))
	}

	c.bindings[key] = deviceUID
	c.byDevice[deviceUID] = key
}

func (c *Controller) claim(ownerUID, externalID, deviceUID string) bool {
	if ownerUID == "" || externalID == "" || deviceUID == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := bindingKey{ownerUID: ownerUID, externalID: externalID}

	if current, ok := c.bindings[key]; ok {
		return current == deviceUID
	}

	c.releaseLocked(deviceUID)

	c.bindings[key] = deviceUID
	c.byDevice[deviceUID] = key

	return true
}

func (c *Controller) holderOf(ownerUID, externalID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.bindings[bindingKey{ownerUID: ownerUID, externalID: externalID}]
}

func (c *Controller) release(deviceUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.releaseLocked(deviceUID)
}

func (c *Controller) releaseLocked(deviceUID string) {
	key, ok := c.byDevice[deviceUID]
	if !ok {
		return
	}

	delete(c.byDevice, deviceUID)

	if c.bindings[key] == deviceUID {
		delete(c.bindings, key)
	}
}

func expireStaleAttempt(dev *corev1.Device, now time.Time) bool {
	attempt := dev.Status.ProbeAttempt
	if attempt == nil || !attemptExpired(attempt, now) {
		return false
	}

	dev.Status.ProbeAttempt = nil
	return true
}

func attemptExpired(attempt *corev1.Device_Status_ProbeAttempt, now time.Time) bool {
	startedAt := attempt.GetStartedAt()
	if startedAt == nil || !startedAt.IsValid() {
		return true
	}
	return now.Sub(startedAt.AsTime()) > probeAttemptTTL
}

func approvalExpired(binding *corev1.Device_Status_Binding, now time.Time) bool {
	expiresAt := binding.GetExpiresAt()
	if expiresAt == nil || !expiresAt.IsValid() {
		return true
	}
	return !now.Before(expiresAt.AsTime())
}

func newBindingUID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
