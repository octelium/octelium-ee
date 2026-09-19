// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"crypto"
	stderrors "errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/registration"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	defaultWorkerCount = 4
	workQueueSize      = 32
	listItemsPerPage   = 100

	attemptTimeout       = 45 * time.Minute
	statusUpdateTimeout  = 30 * time.Second
	statusUpdateRetries  = 8
	statusUpdateInterval = 250 * time.Millisecond

	resyncInterval     = 3 * time.Minute
	staleIssuanceAfter = 1 * time.Hour
	renewBefore        = 21 * 24 * time.Hour

	minRetryDelay = 30 * time.Second
	maxRetryDelay = 6 * time.Hour

	acmeHTTPTimeout       = 2 * time.Minute
	acmeOrderTimeout      = 10 * time.Minute
	dnsPropagationTimeout = 10 * time.Minute
	dnsQueryTimeout       = 20 * time.Second
)

var recursiveNameservers = []string{"8.8.8.8", "1.1.1.1", "9.9.9.9"}

var dns01Once sync.Once

type CertificateSetter interface {
	SetCertificate(ctx context.Context, crt *enterprisev1.Certificate) error
}

type CertificateIssuerSetter interface {
	SetCertificateIssuer(ctx context.Context, iss *enterprisev1.CertificateIssuer, force bool) error
}

type DNSProviderSetter interface {
	SetDNSProvider(ctx context.Context) error
}

type Interface interface {
	CertificateSetter
	CertificateIssuerSetter
	DNSProviderSetter
}

type Account struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (a *Account) GetEmail() string {
	return a.Email
}

func (a *Account) GetPrivateKey() crypto.PrivateKey {
	return a.key
}

func (a *Account) GetRegistration() *registration.Resource {
	return a.Registration
}

type workType uint8

const (
	workTypeCertificate workType = iota + 1
	workTypeCertificateIssuer
)

type workKey struct {
	typ workType
	uid string
}

func (k workKey) String() string {
	switch k.typ {
	case workTypeCertificate:
		return fmt.Sprintf("Certificate/%s", k.uid)
	case workTypeCertificateIssuer:
		return fmt.Sprintf("CertificateIssuer/%s", k.uid)
	default:
		return fmt.Sprintf("Unknown/%s", k.uid)
	}
}

type Controller struct {
	octeliumC octeliumc.ClientInterface

	workerCount int
	workCh      chan workKey
	wakeCh      chan struct{}

	runOnce sync.Once
	wg      sync.WaitGroup

	mu          sync.Mutex
	stopped     bool
	pending     map[workKey]struct{}
	active      map[workKey]struct{}
	dirty       map[workKey]struct{}
	failures    map[workKey]uint32
	notBefore   map[workKey]time.Time
	timers      map[workKey]*time.Timer
	issuerForce map[string]struct{}

	locks *keyedLocker
}

func NewController(octeliumC octeliumc.ClientInterface) *Controller {
	return &Controller{
		octeliumC:   octeliumC,
		workerCount: defaultWorkerCount,
		workCh:      make(chan workKey, workQueueSize),
		wakeCh:      make(chan struct{}, 1),
		pending:     make(map[workKey]struct{}),
		active:      make(map[workKey]struct{}),
		dirty:       make(map[workKey]struct{}),
		failures:    make(map[workKey]uint32),
		notBefore:   make(map[workKey]time.Time),
		timers:      make(map[workKey]*time.Timer),
		issuerForce: make(map[string]struct{}),
		locks:       newKeyedLocker(),
	}
}

func (c *Controller) Run(ctx context.Context) {
	c.runOnce.Do(func() {
		setDNS01Defaults()
		go c.doRun(ctx)
	})
}

func (c *Controller) doRun(ctx context.Context) {
	zap.L().Info("Starting the ACME Controller",
		zap.Int("workers", c.workerCount))

	for range c.workerCount {
		c.wg.Add(1)
		go c.runWorker(ctx)
	}

	c.wg.Add(1)
	go c.runResync(ctx)

	defer func() {
		c.shutdown()
		c.wg.Wait()
		zap.L().Info("The ACME Controller is done")
	}()

	for {
		if key, ok := c.takePending(); ok {
			select {
			case <-ctx.Done():
				return
			case c.workCh <- key:
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-c.wakeCh:
		}
	}
}

func (c *Controller) runWorker(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case key := <-c.workCh:
			c.runOne(ctx, key)
		}
	}
}

func (c *Controller) runOne(ctx context.Context, key workKey) {
	defer c.finish(key)

	attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	var err error
	switch key.typ {
	case workTypeCertificate:
		err = c.reconcileCertificate(attemptCtx, key.uid)
	case workTypeCertificateIssuer:
		err = c.reconcileCertificateIssuer(attemptCtx, key.uid)
	default:
		err = &permanentError{
			err: errors.Errorf("Invalid ACME work type: %d", key.typ),
		}
	}

	if err == nil {
		c.resetFailures(key)
		return
	}

	if ctx.Err() != nil {
		return
	}

	delay := c.scheduleRetry(key, err)
	zap.L().Warn("Could not reconcile the ACME resource. Trying again later...",
		zap.String("key", key.String()),
		zap.Duration("retryAfter", delay),
		zap.Error(err))
}

func (c *Controller) runResync(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(resyncInterval)
	defer ticker.Stop()

	for {
		if err := c.resync(ctx); err != nil && ctx.Err() == nil {
			zap.L().Warn("Could not resync the ACME resources", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Controller) resync(ctx context.Context) error {
	now := time.Now()

	if err := c.listCertificates(ctx, nil, func(crt *enterprisev1.Certificate) error {
		if !needsReconcile(crt, now) {
			return nil
		}

		c.enqueue(workKey{
			typ: workTypeCertificate,
			uid: crt.Metadata.Uid,
		}, false)

		return nil
	}); err != nil {
		return err
	}

	seenIssuers := make(map[string]struct{})

	if err := c.listCertificateIssuers(ctx, func(iss *enterprisev1.CertificateIssuer) error {
		if iss.Spec.GetAcme() == nil {
			return nil
		}

		seenIssuers[iss.Metadata.Uid] = struct{}{}

		c.enqueue(workKey{
			typ: workTypeCertificateIssuer,
			uid: iss.Metadata.Uid,
		}, false)

		return nil
	}); err != nil {
		return err
	}

	c.cleanupIssuerForce(seenIssuers)

	return nil
}

func (c *Controller) SetCertificate(ctx context.Context, crt *enterprisev1.Certificate) error {
	if crt == nil || crt.Metadata == nil || crt.Metadata.Uid == "" {
		return errors.Errorf("Invalid Certificate")
	}

	if !isACMECertificate(crt) {
		return nil
	}

	key := workKey{
		typ: workTypeCertificate,
		uid: crt.Metadata.Uid,
	}

	c.enqueue(key, isIssuanceRequested(crt))

	return nil
}

func (c *Controller) SetCertificateIssuer(ctx context.Context,
	iss *enterprisev1.CertificateIssuer, force bool) error {
	if iss == nil || iss.Metadata == nil || iss.Metadata.Uid == "" {
		return errors.Errorf("Invalid CertificateIssuer")
	}

	if iss.Spec.GetAcme() == nil {
		return nil
	}

	if force {
		c.mu.Lock()
		c.issuerForce[iss.Metadata.Uid] = struct{}{}
		c.mu.Unlock()
	}

	c.enqueue(workKey{
		typ: workTypeCertificateIssuer,
		uid: iss.Metadata.Uid,
	}, force)

	return nil
}

func (c *Controller) SetDNSProvider(ctx context.Context) error {
	return c.listCertificates(ctx, nil, func(crt *enterprisev1.Certificate) error {
		if !isACMECertificate(crt) {
			return nil
		}

		key := workKey{
			typ: workTypeCertificate,
			uid: crt.Metadata.Uid,
		}

		c.mu.Lock()
		delete(c.failures, key)
		c.mu.Unlock()

		c.enqueue(key, true)

		return nil
	})
}

func (c *Controller) enqueue(key workKey, force bool) {
	if key.uid == "" || key.typ == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}

	if force {
		delete(c.notBefore, key)
		c.stopTimerLocked(key)
	}

	if _, ok := c.active[key]; ok {
		c.dirty[key] = struct{}{}
		return
	}

	if at, ok := c.notBefore[key]; ok && time.Now().Before(at) {
		c.startTimerLocked(key, at)
		return
	}

	delete(c.notBefore, key)

	if _, ok := c.pending[key]; ok {
		return
	}

	c.pending[key] = struct{}{}
	c.signalLocked()
}

func (c *Controller) takePending() (workKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.pending {
		delete(c.pending, key)
		c.active[key] = struct{}{}
		return key, true
	}

	return workKey{}, false
}

func (c *Controller) finish(key workKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.active, key)

	if _, ok := c.dirty[key]; !ok {
		return
	}
	delete(c.dirty, key)

	if c.stopped {
		return
	}

	if at, ok := c.notBefore[key]; ok && time.Now().Before(at) {
		c.startTimerLocked(key, at)
		return
	}

	c.pending[key] = struct{}{}
	c.signalLocked()
}

func (c *Controller) scheduleRetry(key workKey, err error) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures[key] = c.failures[key] + 1
	delay := getRetryDelay(c.failures[key], err)
	at := time.Now().Add(delay)

	c.notBefore[key] = at
	c.stopTimerLocked(key)

	if !c.stopped {
		c.startTimerLocked(key, at)
	}

	return delay
}

func (c *Controller) resetFailures(key workKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.failures, key)
	delete(c.notBefore, key)
	c.stopTimerLocked(key)
}

func (c *Controller) startTimerLocked(key workKey, at time.Time) {
	if _, ok := c.timers[key]; ok {
		return
	}

	c.timers[key] = time.AfterFunc(time.Until(at), func() {
		c.onRetryTimer(key)
	})
}

func (c *Controller) stopTimerLocked(key workKey) {
	if timer, ok := c.timers[key]; ok {
		timer.Stop()
		delete(c.timers, key)
	}
}

func (c *Controller) onRetryTimer(key workKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.timers, key)

	if c.stopped {
		return
	}

	if at, ok := c.notBefore[key]; ok && time.Now().Before(at) {
		c.startTimerLocked(key, at)
		return
	}

	delete(c.notBefore, key)

	if _, ok := c.active[key]; ok {
		c.dirty[key] = struct{}{}
		return
	}

	if _, ok := c.pending[key]; ok {
		return
	}

	c.pending[key] = struct{}{}
	c.signalLocked()
}

func (c *Controller) signalLocked() {
	select {
	case c.wakeCh <- struct{}{}:
	default:
	}
}

func (c *Controller) shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stopped = true

	for key := range c.timers {
		c.stopTimerLocked(key)
	}
}

func (c *Controller) cleanupIssuerForce(seen map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for uid := range c.issuerForce {
		if _, ok := seen[uid]; !ok {
			delete(c.issuerForce, uid)
		}
	}
}

func (c *Controller) hasIssuerForce(uid string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.issuerForce[uid]
	return ok
}

func (c *Controller) clearIssuerForce(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.issuerForce, uid)
}

func (c *Controller) listCertificates(ctx context.Context,
	filters []*rmetav1.ListOptions_Filter,
	fn func(*enterprisev1.Certificate) error) error {

	for page := uint32(0); ; page++ {
		lst, err := c.octeliumC.EnterpriseC().ListCertificate(ctx, &rmetav1.ListOptions{
			Filters:      filters,
			Paginate:     true,
			Page:         page,
			ItemsPerPage: listItemsPerPage,
		})
		if err != nil {
			return err
		}

		for _, crt := range lst.Items {
			if crt.Metadata == nil {
				continue
			}
			if err := fn(crt); err != nil {
				return err
			}
		}

		if !lst.GetListResponseMeta().GetHasMore() {
			return nil
		}
	}
}

func (c *Controller) listCertificateIssuers(ctx context.Context,
	fn func(*enterprisev1.CertificateIssuer) error) error {

	for page := uint32(0); ; page++ {
		lst, err := c.octeliumC.EnterpriseC().ListCertificateIssuer(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			Page:         page,
			ItemsPerPage: listItemsPerPage,
		})
		if err != nil {
			return err
		}

		for _, iss := range lst.Items {
			if iss.Metadata == nil || iss.Spec == nil {
				continue
			}
			if err := fn(iss); err != nil {
				return err
			}
		}

		if !lst.GetListResponseMeta().GetHasMore() {
			return nil
		}
	}
}

func (c *Controller) detachedContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), statusUpdateTimeout)
}

func setDNS01Defaults() {
	dns01Once.Do(func() {
		_ = dns01.AddDNSTimeout(dnsQueryTimeout)(nil)
		_ = dns01.AddRecursiveNameservers(recursiveNameservers)(nil)
	})
}

func isCanceled(err error) bool {
	return stderrors.Is(err, context.Canceled) ||
		stderrors.Is(err, context.DeadlineExceeded)
}
