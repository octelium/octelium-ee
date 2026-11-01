// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/collectorutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/cluster/common/sessionc"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	remoteUserName           = "octelium"
	remoteSessionDuration    = 24 * time.Hour
	remoteRetryInterval      = 5 * time.Second
	remoteMinRefreshInterval = 30 * time.Second
	remoteMinRotateInterval  = time.Minute
	remoteRequestTimeout     = 10 * time.Second
	remoteShutdownTimeout    = 15 * time.Second
)

var remoteCertSecretNames = []string{
	"crt-ns-octelium",
	vutils.ClusterCertSecretName,
}

type remoteController struct {
	octeliumC octeliumc.ClientInterface

	mu          sync.RWMutex
	sess        *corev1.Session
	token       string
	refreshedAt time.Time

	refreshCh chan struct{}
	doneCh    chan struct{}
}

func newRemoteController(octeliumC octeliumc.ClientInterface) *remoteController {
	return &remoteController{
		octeliumC: octeliumC,
		refreshCh: make(chan struct{}, 1),
		doneCh:    make(chan struct{}),
	}
}

func (c *remoteController) Run(ctx context.Context) {
	go c.runLoop(ctx)
}

func (c *remoteController) runLoop(ctx context.Context) {
	defer close(c.doneCh)

	for {
		wait := remoteRetryInterval
		if next, err := c.rotate(ctx); err != nil {
			zap.L().Warn("Could not rotate the collector access token", zap.Error(err))
		} else {
			wait = next
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		case <-c.refreshCh:
			timer.Stop()
		}
	}
}

func (c *remoteController) rotate(ctx context.Context) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, remoteRequestTimeout)
	defer cancel()

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return 0, err
	}

	c.mu.RLock()
	current := c.sess
	c.mu.RUnlock()

	sess, err := c.getSession(ctx, cc, current)
	if err != nil {
		return 0, err
	}

	jwkCtl, err := jwkctl.NewJWKController(ctx, c.octeliumC, nil)
	if err != nil {
		return 0, err
	}

	tkn, err := jwkCtl.CreateAccessToken(sess)
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	c.sess = sess
	c.token = tkn
	c.refreshedAt = time.Now()
	c.mu.Unlock()

	if current != nil && current.Metadata.Uid != sess.Metadata.Uid {
		c.deleteSession(ctx, current)
	}

	zap.L().Debug("Rotated the collector access token", zap.String("session", sess.Metadata.Name))

	return getRemoteRotateInterval(sess), nil
}

func (c *remoteController) getSession(ctx context.Context,
	cc *corev1.ClusterConfig, current *corev1.Session) (*corev1.Session, error) {
	if current == nil {
		return c.createSession(ctx, cc)
	}

	sess, err := c.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{Uid: current.Metadata.Uid})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return nil, err
		}

		return c.createSession(ctx, cc)
	}

	if sess.Spec.State != corev1.Session_Spec_ACTIVE ||
		time.Until(sess.Spec.ExpiresAt.AsTime()) <= getAccessTokenDuration(sess) {
		return c.createSession(ctx, cc)
	}

	sessionc.SetCurrAuthentication(&sessionc.SetCurrAuthenticationOpts{
		Session:       sess,
		ClusterConfig: cc,
		AuthInfo: &corev1.Session_Status_Authentication_Info{
			Type: corev1.Session_Status_Authentication_Info_INTERNAL,
		},
	})

	return c.octeliumC.CoreC().UpdateSession(ctx, sess)
}

func (c *remoteController) createSession(ctx context.Context, cc *corev1.ClusterConfig) (*corev1.Session, error) {
	usr, err := c.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: remoteUserName})
	if err != nil {
		return nil, err
	}

	sess, err := sessionc.NewSession(ctx, &sessionc.CreateSessionOpts{
		Usr:           usr,
		OcteliumC:     c.octeliumC,
		ClusterConfig: cc,
		SessType:      corev1.Session_Status_CLIENTLESS,
		ExpiresAt:     time.Now().Add(remoteSessionDuration),
		Authorization: getRemoteSessionAuthorization(),
		AuthenticationInfo: &corev1.Session_Status_Authentication_Info{
			Type: corev1.Session_Status_Authentication_Info_INTERNAL,
		},
	})
	if err != nil {
		return nil, err
	}

	sess.Spec.State = corev1.Session_Spec_ACTIVE

	return c.octeliumC.CoreC().CreateSession(ctx, sess)
}

func (c *remoteController) deleteSession(ctx context.Context, sess *corev1.Session) {
	if _, err := c.octeliumC.CoreC().DeleteSession(ctx,
		&rmetav1.DeleteOptions{Uid: sess.Metadata.Uid}); err != nil && !grpcerr.IsNotFound(err) {
		zap.L().Warn("Could not delete the collector Session",
			zap.String("session", sess.Metadata.Name), zap.Error(err))
	}
}

func (c *remoteController) Close() {
	<-c.doneCh

	c.mu.Lock()
	sess := c.sess
	c.sess = nil
	c.token = ""
	c.mu.Unlock()

	if sess == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), remoteRequestTimeout)
	defer cancel()

	c.deleteSession(ctx, sess)
}

func (c *remoteController) GetAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.token == "" {
		return "", errors.Errorf("The collector access token is not set")
	}

	return c.token, nil
}

func (c *remoteController) Refresh() {
	c.mu.RLock()
	refreshedAt := c.refreshedAt
	c.mu.RUnlock()

	if time.Since(refreshedAt) < remoteMinRefreshInterval {
		return
	}

	select {
	case c.refreshCh <- struct{}{}:
	default:
	}
}

func (c *remoteController) GetTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
		VerifyConnection:   c.verifyConnection,
	}
}

func (c *remoteController) verifyConnection(cs tls.ConnectionState) error {
	if cs.ServerName == "" {
		return errors.Errorf("The TLS server name is not set")
	}

	if len(cs.PeerCertificates) == 0 {
		return errors.Errorf("The server did not present a certificate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), remoteRequestTimeout)
	defer cancel()

	opts := x509.VerifyOptions{
		DNSName:       cs.ServerName,
		Roots:         c.getRootCAs(ctx),
		Intermediates: x509.NewCertPool(),
	}

	for _, crt := range cs.PeerCertificates[1:] {
		opts.Intermediates.AddCert(crt)
	}

	_, err := cs.PeerCertificates[0].Verify(opts)
	return err
}

func (c *remoteController) getRootCAs(ctx context.Context) *x509.CertPool {
	ret, err := x509.SystemCertPool()
	if err != nil {
		ret = x509.NewCertPool()
	}

	for _, name := range remoteCertSecretNames {
		sec, err := c.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{Name: name})
		if err != nil {
			continue
		}

		chain, _, err := ucorev1.ToSecret(sec).GetCertificateChainAndKey()
		if err != nil {
			continue
		}

		ret.AppendCertsFromPEM(chain)
	}

	return ret
}

func getRemoteSessionAuthorization() *corev1.Session_Spec_Authorization {
	return &corev1.Session_Spec_Authorization{
		InlinePolicies: []*corev1.InlinePolicy{
			{
				Spec: &corev1.Policy_Spec{
					Rules: []*corev1.Policy_Spec_Rule{
						{
							Effect:   corev1.Policy_Spec_Rule_DENY,
							Priority: -4,
							Condition: &corev1.Condition{
								Type: &corev1.Condition_Not{
									Not: fmt.Sprintf(`ctx.service.metadata.name == "%s"`, collectorutils.ServiceName),
								},
							},
						},
					},
				},
			},
		},
	}
}

func getAccessTokenDuration(sess *corev1.Session) time.Duration {
	if sess.Status.Authentication == nil {
		return 0
	}

	return umetav1.ToDuration(sess.Status.Authentication.AccessTokenDuration).ToGo()
}

func getRemoteRotateInterval(sess *corev1.Session) time.Duration {
	ret := getAccessTokenDuration(sess) / 2
	if untilExpiry := time.Until(sess.Spec.ExpiresAt.AsTime()) / 2; untilExpiry < ret {
		ret = untilExpiry
	}

	if ret < remoteMinRotateInterval {
		return remoteMinRotateInterval
	}

	return ret
}
