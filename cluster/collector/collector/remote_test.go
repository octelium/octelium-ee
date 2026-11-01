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
	"crypto/x509/pkix"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/jwkctl"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

func createTestRemoteUser(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface) {
	t.Helper()

	_, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name:     remoteUserName,
			IsSystem: true,
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_WORKLOAD,
		},
		Status: &corev1.User_Status{},
	})
	require.NoError(t, err)
}

func newTestRemoteCert(t *testing.T, hosts []string, ips []net.IP) (*utils_cert.Cert, *utils_cert.Cert) {
	t.Helper()

	ca, err := utils_cert.GenerateCARoot()
	require.NoError(t, err)

	serialNumber, err := utils_cert.GenerateSerialNumber()
	require.NoError(t, err)

	leaf, err := utils_cert.GenerateCertificate(&x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: hosts[0],
		},
		DNSNames:    hosts,
		IPAddresses: ips,
		NotBefore:   time.Now().Add(-time.Minute),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, ca.Certificate, ca.PrivateKey, false)
	require.NoError(t, err)

	return ca, leaf
}

func setTestRemoteCertSecret(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface, name string, crt *utils_cert.Cert) {
	t.Helper()

	sec, err := octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{Name: name})
	if err == nil {
		ucorev1.ToSecret(sec).SetCertificate(string(crt.MustGetCertPEM()), string(crt.MustGetPrivateKeyPEM()))
		_, err = octeliumC.CoreC().UpdateSecret(ctx, sec)
		require.NoError(t, err)
		return
	}
	require.True(t, grpcerr.IsNotFound(err))

	sec = &corev1.Secret{
		Metadata: &metav1.Metadata{
			Name:     name,
			IsSystem: true,
		},
		Spec:   &corev1.Secret_Spec{},
		Status: &corev1.Secret_Status{},
	}
	ucorev1.ToSecret(sec).SetCertificate(string(crt.MustGetCertPEM()), string(crt.MustGetPrivateKeyPEM()))

	_, err = octeliumC.CoreC().CreateSecret(ctx, sec)
	require.NoError(t, err)
}

func getTestRemoteTLSCertificate(t *testing.T, crt *utils_cert.Cert) tls.Certificate {
	t.Helper()

	ret, err := tls.X509KeyPair(crt.MustGetCertPEM(), crt.MustGetPrivateKeyPEM())
	require.NoError(t, err)
	return ret
}

func TestRemoteControllerSessionLifecycle(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	c := newRemoteController(octeliumC)
	c.Run(ctx)

	_, err = c.GetAccessToken(ctx)
	require.Error(t, err)

	createTestRemoteUser(ctx, t, octeliumC)
	c.Refresh()

	jwkCtl, err := jwkctl.NewJWKController(ctx, octeliumC, nil)
	require.NoError(t, err)

	waitToken := func(prev string) string {
		var ret string
		require.Eventually(t, func() bool {
			tkn, err := c.GetAccessToken(ctx)
			if err != nil || tkn == prev {
				return false
			}
			ret = tkn
			return true
		}, 15*time.Second, 50*time.Millisecond)
		return ret
	}

	getSession := func(tkn string) (*corev1.Session, *jwkctl.AccessTokenClaims) {
		claims, err := jwkCtl.VerifyAccessToken(tkn)
		require.NoError(t, err)
		sess, err := octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{Uid: claims.SessionUID})
		require.NoError(t, err)
		return sess, claims
	}

	forceRefresh := func() {
		c.mu.Lock()
		c.refreshedAt = time.Time{}
		c.mu.Unlock()
		c.Refresh()
	}

	tkn := waitToken("")
	sess, claims := getSession(tkn)
	assert.Equal(t, remoteUserName, sess.Status.UserRef.Name)
	assert.Equal(t, corev1.Session_Status_CLIENTLESS, sess.Status.Type)
	assert.Equal(t, corev1.Session_Spec_ACTIVE, sess.Spec.State)
	assert.Equal(t, claims.TokenID, sess.Status.Authentication.TokenID)
	assert.Equal(t, corev1.Session_Status_Authentication_Info_INTERNAL, sess.Status.Authentication.Info.Type)
	assert.WithinDuration(t, time.Now().Add(remoteSessionDuration), sess.Spec.ExpiresAt.AsTime(), time.Minute)
	assert.True(t, pbutils.IsEqual(getRemoteSessionAuthorization(), sess.Spec.Authorization))
	assert.Equal(t, `ctx.service.metadata.name == "collector.octelium"`,
		sess.Spec.Authorization.InlinePolicies[0].Spec.Rules[0].Condition.GetNot())

	{
		c.Refresh()
		assert.Never(t, func() bool {
			cur, err := c.GetAccessToken(ctx)
			return err != nil || cur != tkn
		}, 500*time.Millisecond, 50*time.Millisecond)
	}

	{
		forceRefresh()
		tkn = waitToken(tkn)
		rotated, rotatedClaims := getSession(tkn)
		assert.Equal(t, sess.Metadata.Uid, rotated.Metadata.Uid)
		assert.NotEqual(t, claims.TokenID, rotatedClaims.TokenID)
		assert.Equal(t, rotatedClaims.TokenID, rotated.Status.Authentication.TokenID)
		assert.EqualValues(t, 2, rotated.Status.TotalAuthentications)
		sess = rotated
	}

	{
		_, err := octeliumC.CoreC().DeleteSession(ctx, &rmetav1.DeleteOptions{Uid: sess.Metadata.Uid})
		require.NoError(t, err)
		forceRefresh()
		tkn = waitToken(tkn)
		renewed, _ := getSession(tkn)
		assert.NotEqual(t, sess.Metadata.Uid, renewed.Metadata.Uid)
		sess = renewed
	}

	for _, mutate := range []func(*corev1.Session){
		func(s *corev1.Session) { s.Spec.ExpiresAt = pbutils.Timestamp(time.Now().Add(time.Hour)) },
		func(s *corev1.Session) { s.Spec.State = corev1.Session_Spec_REJECTED },
	} {
		mutate(sess)
		_, err := octeliumC.CoreC().UpdateSession(ctx, sess)
		require.NoError(t, err)
		forceRefresh()
		tkn = waitToken(tkn)
		renewed, _ := getSession(tkn)
		assert.NotEqual(t, sess.Metadata.Uid, renewed.Metadata.Uid)

		_, err = octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{Uid: sess.Metadata.Uid})
		assert.True(t, grpcerr.IsNotFound(err))
		sess = renewed
	}

	cancel()
	c.Close()

	_, err = octeliumC.CoreC().GetSession(t.Context(), &rmetav1.GetOptions{Uid: sess.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err))
	_, err = c.GetAccessToken(t.Context())
	assert.Error(t, err)
}

func TestRemoteControllerVerifyConnection(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC
	ctx := t.Context()

	_, leaf := newTestRemoteCert(t, []string{"collector.octelium.example.com", "localhost"}, nil)
	_, other := newTestRemoteCert(t, []string{"collector.octelium.example.com", "localhost"}, nil)

	lis, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{getTestRemoteTLSCertificate(t, leaf)},
	})
	require.NoError(t, err)
	t.Cleanup(func() { lis.Close() })
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.(*tls.Conn).Handshake()
			}()
		}
	}()

	c := newRemoteController(octeliumC)

	dial := func(serverName string) error {
		cfg := c.GetTLSConfig()
		cfg.ServerName = serverName
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", lis.Addr().String(), cfg)
		if err != nil {
			return err
		}
		return conn.Close()
	}

	assert.Error(t, dial("collector.octelium.example.com"))

	setTestRemoteCertSecret(ctx, t, octeliumC, vutils.ClusterCertSecretName, other)
	assert.Error(t, dial("collector.octelium.example.com"))

	setTestRemoteCertSecret(ctx, t, octeliumC, "crt-ns-octelium", leaf)
	require.NoError(t, dial("collector.octelium.example.com"))
	require.NoError(t, dial("localhost"))
	assert.Error(t, dial("other.example.com"))
	assert.Error(t, dial(""))

	assert.Error(t, c.verifyConnection(tls.ConnectionState{ServerName: "localhost"}))

	cfg := c.GetTLSConfig()
	assert.EqualValues(t, tls.VersionTLS12, cfg.MinVersion)
	assert.NotNil(t, cfg.VerifyConnection)
}

func TestRemoteRotateInterval(t *testing.T) {
	newSess := func(tokenHours uint32, expiresIn time.Duration) *corev1.Session {
		return &corev1.Session{
			Spec: &corev1.Session_Spec{
				ExpiresAt: pbutils.Timestamp(time.Now().Add(expiresIn)),
			},
			Status: &corev1.Session_Status{
				Authentication: &corev1.Session_Status_Authentication{
					AccessTokenDuration: &metav1.Duration{
						Type: &metav1.Duration_Hours{
							Hours: tokenHours,
						},
					},
				},
			},
		}
	}

	assert.Equal(t, 4*time.Hour, getAccessTokenDuration(newSess(4, time.Hour)))
	assert.Zero(t, getAccessTokenDuration(&corev1.Session{Status: &corev1.Session_Status{}}))
	assert.InDelta(t, float64(2*time.Hour), float64(getRemoteRotateInterval(newSess(4, 24*time.Hour))), float64(time.Second))
	assert.InDelta(t, float64(12*time.Hour), float64(getRemoteRotateInterval(newSess(24*30, 24*time.Hour))), float64(time.Second))
	assert.Equal(t, remoteMinRotateInterval, getRemoteRotateInterval(newSess(4, time.Second)))
	assert.Equal(t, remoteMinRotateInterval, getRemoteRotateInterval(newSess(4, -time.Hour)))
}

type remoteTestLogsSink struct {
	plogotlp.UnimplementedGRPCServer
	sink *remoteTestSink
}

func (s *remoteTestLogsSink) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	s.sink.add(ctx, req.Logs().LogRecordCount(), 0)
	return plogotlp.NewExportResponse(), nil
}

type remoteTestMetricsSink struct {
	pmetricotlp.UnimplementedGRPCServer
	sink *remoteTestSink
}

func (s *remoteTestMetricsSink) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	s.sink.add(ctx, 0, req.Metrics().DataPointCount())
	return pmetricotlp.NewExportResponse(), nil
}

type remoteTestSink struct {
	mu        sync.Mutex
	logRecs   int
	metricPts int
	tokens    []string
}

func (s *remoteTestSink) add(ctx context.Context, logRecs, metricPts int) {
	md, _ := metadata.FromIncomingContext(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.logRecs += logRecs
	s.metricPts += metricPts
	s.tokens = append(s.tokens, md.Get("x-octelium-auth")...)
}

func (s *remoteTestSink) get() (int, int, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logRecs, s.metricPts, append([]string(nil), s.tokens...)
}

func TestServerExportsInternalTelemetryToRemote(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC
	ctx, cancel := context.WithCancel(context.Background())
	createTestRemoteUser(ctx, t, octeliumC)

	_, leaf := newTestRemoteCert(t, []string{"localhost"}, nil)
	setTestRemoteCertSecret(ctx, t, octeliumC, "crt-ns-octelium", leaf)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	sink := &remoteTestSink{}
	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{getTestRemoteTLSCertificate(t, leaf)},
	})))
	plogotlp.RegisterGRPCServer(grpcSrv, &remoteTestLogsSink{sink: sink})
	pmetricotlp.RegisterGRPCServer(grpcSrv, &remoteTestMetricsSink{sink: sink})
	go grpcSrv.Serve(lis)
	t.Cleanup(func() { grpcSrv.Stop() })

	srv := &Server{
		octeliumC:    octeliumC,
		receiverPort: tests.GetPort(),
	}
	require.NoError(t, srv.setRemote(ctx))
	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("collector.octelium.%s:443", cc.Status.Domain), srv.remoteEndpoint)
	require.NotNil(t, srv.getExporterOpts())

	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)
	srv.remoteEndpoint = net.JoinHostPort("localhost", port)

	require.NoError(t, srv.doRun(ctx))
	assert.Equal(t, srv.remoteEndpoint, srv.p.getInternalLogstoreEndpoint())
	assert.Equal(t, srv.remoteEndpoint, srv.p.getInternalMetricstoreEndpoint())

	require.Eventually(t, func() bool {
		if err := exportTestLog(ctx, srv.p.port); err != nil {
			return false
		}
		if err := exportTestMetric(ctx, srv.p.port); err != nil {
			return false
		}
		logRecs, metricPts, _ := sink.get()
		return logRecs >= 1 && metricPts >= 1
	}, 20*time.Second, 200*time.Millisecond)

	jwkCtl, err := jwkctl.NewJWKController(ctx, octeliumC, nil)
	require.NoError(t, err)
	_, _, tokens := sink.get()
	require.NotEmpty(t, tokens)
	claims, err := jwkCtl.VerifyAccessToken(tokens[len(tokens)-1])
	require.NoError(t, err)
	sess, err := octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{Uid: claims.SessionUID})
	require.NoError(t, err)
	assert.Equal(t, remoteUserName, sess.Status.UserRef.Name)

	cancel()
	done := make(chan struct{})
	go func() {
		srv.waitRemote()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("could not stop the remote collector")
	}

	_, err = octeliumC.CoreC().GetSession(context.Background(), &rmetav1.GetOptions{Uid: sess.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestCollectorRemoteInternalExporterConfig(t *testing.T) {
	p := &provider{
		port:                        12345,
		internalLogstoreEndpoint:    "logs.internal:8080",
		internalMetricstoreEndpoint: "metrics.internal:8080",
	}
	cfg, err := p.getInitConfig(t.Context())
	require.NoError(t, err)
	for _, name := range []string{"logstore", "metricstore"} {
		assert.NotContains(t, collectorConfigValue(t, cfg, "exporters.octelium_otlp/"+name), "remote")
	}

	p.remoteEndpoint = "collector.octelium.example.com:443"
	cfg, err = p.getInitConfig(t.Context())
	require.NoError(t, err)
	for _, name := range []string{"logstore", "metricstore"} {
		assert.Equal(t, "collector.octelium.example.com:443",
			collectorConfigValue(t, cfg, "exporters.octelium_otlp/"+name+".endpoint"))
		assert.Equal(t, true, collectorConfigValue(t, cfg, "exporters.octelium_otlp/"+name+".remote"))
	}
	assert.Equal(t, 16, collectorConfigValue(t, cfg, "exporters.octelium_otlp/logstore.max_call_send_msg_size_mib"))
	assert.Equal(t, 8, collectorConfigValue(t, cfg, "exporters.octelium_otlp/metricstore.max_call_send_msg_size_mib"))

	s := &Server{}
	assert.Nil(t, s.getExporterOpts())
	s.waitRemote()
}
