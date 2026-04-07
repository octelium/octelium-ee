// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package e2e

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/userv1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/client/common/client"
	"github.com/octelium/octelium/client/common/cliutils"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type server struct {
	domain         string
	homedir        string
	t              *CustomT
	k8sC           kubernetes.Interface
	externalIP     string
	createdAt      time.Time
	installedAt    time.Time
	kubeConfigPath string
}

func initServer(ctx context.Context) (*server, error) {

	ret := &server{
		domain:         "localhost",
		t:              &CustomT{},
		createdAt:      time.Now(),
		kubeConfigPath: "/etc/rancher/k3s/k3s.yaml",
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	zap.L().Info("Current user", zap.Any("info", u))

	ret.homedir = fmt.Sprintf("/home/%s", u.Username)

	return ret, nil
}

func (s *server) run(ctx context.Context) error {
	t := s.t
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := s.installCluster(ctx); err != nil {
		return err
	}
	s.installedAt = time.Now()

	assert.Nil(t, s.installClusterCert(ctx))

	{
		cmd := s.getCmd(ctx,
			`ip addr show $(ip route show default | ip route show default | awk '/default/ {print $5}') | grep "inet " | awk '{print $2}' | cut -d'/' -f1`)
		out, err := cmd.CombinedOutput()
		assert.Nil(t, err)
		s.externalIP = strings.TrimSpace(string(out))
		zap.L().Debug("The VM IP addr", zap.String("addr", s.externalIP))
	}

	{
		os.Setenv("OCTELIUM_DOMAIN", s.domain)

		os.Unsetenv("OCTELIUM_INSECURE_TLS")
		os.Setenv("OCTELIUM_INSECURE_TLS", "false")
		os.Setenv("OCTELIUM_PRODUCTION", "true")
		os.Setenv("HOME", s.homedir)
		os.Setenv("KUBECONFIG", s.kubeConfigPath)
	}

	{
		s.runCmd(ctx, "id")
		s.runCmd(ctx, "mkdir -p ~/.ssh")
		s.runCmd(ctx, "chmod 700 ~/.ssh")
		s.runCmd(ctx, "cat /etc/rancher/k3s/k3s.yaml")
	}
	{
		zap.L().Info("Env vars", zap.Strings("env", os.Environ()))
	}

	{
		k8sC, err := s.getK8sC()
		if err != nil {
			return err
		}
		s.k8sC = k8sC

		assert.Nil(t, s.runK8sInitChecks(ctx))
	}

	{

		s.startKubectlLog(ctx, "-l octelium.com/component=nocturne")
		s.startKubectlLog(ctx, "-l octelium.com/component=octovigil")
		s.startKubectlLog(ctx, "-l octelium.com/component=secretman")

		assert.Nil(t, s.runCmd(ctx, "kubectl get pods -A"))
		assert.Nil(t, s.runCmd(ctx, "kubectl get deployment -A"))
		assert.Nil(t, s.runCmd(ctx, "kubectl get svc -A"))
		assert.Nil(t, s.runCmd(ctx, "kubectl get daemonset -A"))

		assert.Nil(t, s.waitDeploymentSvc(ctx, "demo-nginx"))
		assert.Nil(t, s.waitDeploymentSvc(ctx, "portal"))
		assert.Nil(t, s.waitDeploymentSvc(ctx, "default"))
	}

	{
		assert.Nil(t, s.runCmd(ctx, "octelium version"))
		assert.Nil(t, s.runCmd(ctx, "octelium version -o json"))
		assert.Nil(t, s.runCmd(ctx, "octeliumctl version"))
		assert.Nil(t, s.runCmd(ctx, "octelium status"))

		assert.Nil(t, s.runCmd(ctx, "octeliumctl get rgn default"))
		assert.Nil(t, s.runCmd(ctx, "octeliumctl get gw -o yaml"))
	}
	{
		res, err := s.httpC().R().Get("https://localhost")
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
	}
	{

		res, err := s.httpCPublic("demo-nginx").R().Get("/")
		assert.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode())
	}
	{

		res, err := s.httpCPublic("portal").R().Get("/")
		assert.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode())
	}

	if err := s.runSDK(ctx); err != nil {
		return err
	}

	if err := s.runOcteliumConnectCommands(ctx); err != nil {
		return err
	}

	if err := s.checkComponents(ctx); err != nil {
		return err
	}

	zap.L().Debug("Test done", zap.Duration("duration", time.Since(s.createdAt)))

	return nil
}

func (s *server) httpCPublic(svc string) *resty.Client {
	return s.httpC().SetBaseURL(fmt.Sprintf("https://%s.localhost", svc))
}

func (s *server) httpCPublicAccessToken(svc, accessToken string) *resty.Client {
	return s.httpC().SetBaseURL(fmt.Sprintf("https://%s.localhost", svc)).SetAuthScheme("Bearer").
		SetAuthToken(accessToken)
}

func (s *server) httpCPublicAccessTokenCheck(svc, accessToken string) {
	t := s.t

	res, err := s.httpCPublicAccessToken(svc, accessToken).R().Get("/")
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode())
}

func (s *server) httpC() *resty.Client {
	return resty.New().SetRetryCount(20).SetRetryWaitTime(500 * time.Millisecond).SetRetryMaxWaitTime(2 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if r.StatusCode() >= 500 && r.StatusCode() < 600 {
				return true
			}
			return false
		}).
		AddRetryHook(func(r *resty.Response, err error) {
			zap.L().Debug("Retrying....", zap.Error(err))
		}).SetTimeout(40 * time.Second).SetLogger(zap.S())
}

func (s *server) runOcteliumConnectCommands(ctx context.Context) error {
	t := s.t

	ctx, cancel := context.WithTimeout(ctx, 500*time.Second)
	defer cancel()

	{
		connCmd, err := s.startOcteliumConnectRootless(ctx, []string{
			"-p demo-nginx:15001",
		})
		assert.Nil(t, err)

		{
			res, err := s.httpC().R().Get("http://localhost:15001")
			assert.Nil(t, err)
			assert.Equal(t, http.StatusOK, res.StatusCode())
		}

		assert.Nil(t, s.runCmd(ctx, "octelium disconnect"))

		connCmd.Wait()

		zap.L().Debug("octelium connect exited")
	}

	return nil
}

func Run(ctx context.Context) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)

	s, err := initServer(ctx)
	if err != nil {
		return err
	}

	if err := s.run(ctx); err != nil {
		return err
	}

	if s.t.errs > 0 {
		panic(fmt.Sprintf("e2e err: %d", s.t.errs))
	}

	return nil
}

type CustomT struct {
	errs int
}

func (t *CustomT) Errorf(format string, args ...interface{}) {
	t.errs++
	zap.S().Errorf(format, args...)
}

func (t *CustomT) FailNow() {
	panic("")
}

func (s *server) getK8sC() (kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", s.kubeConfigPath)
	if err != nil {
		return nil, err
	}

	k8sC, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return k8sC, nil
}

func (s *server) runK8sInitChecks(ctx context.Context) error {
	t := s.t

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	assert.Nil(t, s.waitDeploymentComponent(ctx, "nocturne"))
	assert.Nil(t, s.waitDeploymentComponent(ctx, "octovigil"))
	assert.Nil(t, s.waitDeploymentComponent(ctx, "ingress"))
	assert.Nil(t, s.waitDeploymentComponent(ctx, "rscserver"))
	assert.Nil(t, s.waitDeploymentComponent(ctx, "ingress-dataplane"))

	assert.Nil(t, k8sutils.WaitReadinessDaemonsetWithNS(ctx, s.k8sC, "octelium-gwagent", vutils.K8sNS))

	return nil
}

func (s *server) waitDeploymentComponent(ctx context.Context, name string) error {
	return k8sutils.WaitReadinessDeployment(ctx, s.k8sC, fmt.Sprintf("octelium-%s", name))
}

func (s *server) waitDeploymentSvc(ctx context.Context, name string) error {
	return k8sutils.WaitReadinessDeployment(ctx, s.k8sC, k8sutils.GetSvcHostname(&corev1.Service{
		Metadata: &metav1.Metadata{
			Name: vutils.GetServiceFullNameFromName(name),
		},
	}))
}

func (s *server) waitDeploymentSvcUpstream(ctx context.Context, name string) error {
	return k8sutils.WaitReadinessDeployment(ctx, s.k8sC, k8sutils.GetSvcK8sUpstreamHostname(&corev1.Service{
		Metadata: &metav1.Metadata{
			Name: vutils.GetServiceFullNameFromName(name),
		},
	}, ""))
}

func (s *server) execServiceUpstream(ctx context.Context, svc string, cmd string) error {
	return s.runCmd(ctx,
		fmt.Sprintf(
			`kubectl exec -n octelium -it $(kubectl get pod -n octelium -l octelium.com/svc=%s,octelium.com/component=svc-k8s-upstream -o jsonpath='{.items[0].metadata.name}') -- %s`,
			vutils.GetServiceFullNameFromName(svc), cmd))
}

func (s *server) describeUpstreamPod(ctx context.Context, svc string) error {
	return s.runCmd(ctx,
		fmt.Sprintf(
			`kubectl describe pod -n octelium $(kubectl get pod -n octelium -l octelium.com/svc=%s,octelium.com/component=svc-k8s-upstream -o jsonpath='{.items[0].metadata.name}')`,
			vutils.GetServiceFullNameFromName(svc)))
}

func (s *server) logServiceUpstream(ctx context.Context, svc string) error {
	cmdStr := fmt.Sprintf(
		`kubectl logs -f -n octelium -l octelium.com/component=svc-k8s-upstream,octelium.com/svc=%s`,
		vutils.GetServiceFullNameFromName(svc))

	cmd := s.getCmd(ctx, cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Start()
}

func (s *server) logVigil(ctx context.Context, svc string) error {
	cmdStr := fmt.Sprintf(
		`kubectl logs -f -n octelium -l octelium.com/component=svc,octelium.com/svc=%s`,
		vutils.GetServiceFullNameFromName(svc))

	cmd := s.getCmd(ctx, cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Start()
}

func calculateSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	written, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), written, nil
}

func connectWithRetry(driverName, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	maxRetries := 30

	for attempt := 1; ; attempt++ {
		db, err = sql.Open(driverName, dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				return db, nil
			}
		}

		if maxRetries > 0 && attempt >= maxRetries {
			break
		}

		zap.L().Debug("Retrying connection to db", zap.String("dsn", dsn), zap.Error(err))
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to database: %w", err)
}

func getFileSize(pth string) int64 {

	fileInfo, err := os.Stat(pth)
	if err != nil {
		return 0
	}
	return fileInfo.Size()

}

func (s *server) listComponentPods(ctx context.Context, name string) (*k8scorev1.PodList, error) {
	return s.k8sC.CoreV1().Pods(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{
		LabelSelector: fmt.Sprintf("octelium.com/component=%s", name),
	})
}

func (s *server) getComponentPod(ctx context.Context, name string) (*k8scorev1.Pod, error) {
	podList, err := s.listComponentPods(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(podList.Items) < 1 {
		return nil, errors.Errorf("No pods")
	}

	return &podList.Items[0], nil
}

func (s *server) checkComponentRestarts(ctx context.Context, name string) error {

	pod, err := s.getComponentPod(ctx, name)
	if err != nil {
		return err
	}

	totalRestarts := 0
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += int(cs.RestartCount)
	}

	assert.Zero(s.t, totalRestarts)

	return nil
}

func (s *server) checkComponents(ctx context.Context) error {

	t := s.t

	components := []string{
		"ingress",
		"ingress-dataplane",
		"nocturne",
		"rscserver",
		"octovigil",
		"gwagent",

		"clusterman",
		"secretman",
		"logstore",
		"rscstore",
		"metricstore",
		"cloudman",
		"collector",
		"policyportal",
	}

	zap.L().Debug("Starting checking components",
		zap.Duration("installedSince", time.Since(s.installedAt)))

	for _, comp := range components {
		assert.Nil(t, s.checkComponentRestarts(ctx, comp))
	}

	return nil
}

func (s *server) installClusterCert(ctx context.Context) error {

	t := s.t

	domain := s.domain
	sans := []string{
		domain,
		fmt.Sprintf("*.%s", domain),

		fmt.Sprintf("*.octelium.%s", domain),
		fmt.Sprintf("*.octelium-api.%s", domain),

		fmt.Sprintf("*.local.%s", domain),
		fmt.Sprintf("*.default.%s", domain),
		fmt.Sprintf("*.default.local.%s", domain),

		fmt.Sprintf("*.octelium.local.%s", domain),
		fmt.Sprintf("*.octelium-api.local.%s", domain),
	}

	zap.L().Debug("Setting initial Cluster Certificate",
		zap.String("domain", domain),
		zap.Strings("sans", sans))

	initCrt, err := utils_cert.GenerateSelfSignedCert(domain, sans, 4*12*30*24*time.Hour)
	if err != nil {
		return err
	}

	crtPEM, err := initCrt.GetCertPEM()
	assert.Nil(t, err)

	privPEM, err := initCrt.GetPrivateKeyPEM()
	assert.Nil(t, err)

	keyPath := "/tmp/octelium-private-key.pem"
	certPath := "/tmp/octelium-cert.pem"

	assert.Nil(t, os.WriteFile(keyPath, []byte(privPEM), 0644))
	assert.Nil(t, os.WriteFile(certPath, []byte(crtPEM), 0644))

	if err := s.runCmd(ctx,
		fmt.Sprintf("sudo cp %s /usr/local/share/ca-certificates/octelium-cluster.crt", certPath)); err != nil {
		return err
	}

	if err := s.runCmd(ctx, "sudo update-ca-certificates"); err != nil {
		return err
	}

	cmdStr := fmt.Sprintf(`octops cert %s --key %s --cert %s --kubeconfig %s`,
		s.domain, keyPath, certPath, s.kubeConfigPath)
	if err := s.runCmd(ctx, cmdStr); err != nil {
		return err
	}

	return nil
}

func (s *server) runSDK(ctx context.Context) error {

	t := s.t
	if err := cliutils.OpenDB(""); err != nil {
		return err
	}
	defer cliutils.CloseDB()

	conn, err := client.GetGRPCClientConn(ctx, s.domain)
	assert.Nil(t, err)
	defer conn.Close()

	coreC := corev1.NewMainServiceClient(conn)

	userC := userv1.NewMainServiceClient(conn)

	enterpriseC := enterprisev1.NewMainServiceClient(conn)
	accessLogC := visibilityv1.NewAccessLogServiceClient(conn)
	auditLogC := visibilityv1.NewAuditLogServiceClient(conn)
	authenticationLogC := visibilityv1.NewAuthenticationLogServiceClient(conn)
	visibilityCoreC := vcorev1.NewResourceServiceClient(conn)
	policyPortalC := enterprisev1.NewPolicyPortalServiceClient(conn)

	status, err := userC.GetStatus(ctx, &userv1.GetStatusRequest{})
	assert.Nil(t, err)

	meUsr, err := coreC.GetUser(ctx, &metav1.GetOptions{
		Name: status.User.Metadata.Name,
	})
	assert.Nil(t, err)

	meSess, err := coreC.GetSession(ctx, &metav1.GetOptions{
		Uid: status.Session.Metadata.Uid,
	})
	assert.Nil(t, err)

	eeAPISvc, err := coreC.GetService(ctx, &metav1.GetOptions{
		Name: "enterprise.octelium-api",
	})
	assert.Nil(t, err)

	{
		_, err = coreC.CreateUser(ctx, &corev1.User{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &corev1.User_Spec{
				Type: corev1.User_Spec_WORKLOAD,
			},
		})
		assert.Nil(t, err)
	}
	{
		_, err := enterpriseC.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		assert.Nil(t, err)
	}

	{
		sec, err := enterpriseC.CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Secret_Spec{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: utilrand.GetRandomStringCanonical(32),
				},
			},
		})
		assert.Nil(t, err)

		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomStringCanonical(32),
			},
		}

		sec, err = enterpriseC.UpdateSecret(ctx, sec)
		assert.Nil(t, err)

		_, err = enterpriseC.DeleteSecret(ctx, &metav1.DeleteOptions{
			Name: sec.Metadata.Name,
		})
		assert.Nil(t, err)
	}

	{
		_, err = enterpriseC.GetDNSProvider(ctx, &metav1.GetOptions{
			Name: "default",
		})
		assert.Nil(t, err)

		_, err = enterpriseC.GetCertificateIssuer(ctx, &metav1.GetOptions{
			Name: "default",
		})
		assert.Nil(t, err)

		_, err = enterpriseC.GetSecretStore(ctx, &metav1.GetOptions{
			Name: "default",
		})
		assert.Nil(t, err)
	}

	{
		res, err := accessLogC.ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{})
		assert.Nil(t, err)

		assert.True(t, len(res.Items) > 0)
	}

	{
		res, err := accessLogC.ListAccessLog(ctx, &visibilityv1.ListAccessLogRequest{
			UserRef: umetav1.GetObjectReference(meUsr),
		})
		assert.Nil(t, err)

		assert.True(t, len(res.Items) > 0)

		for _, itm := range res.Items {
			assert.Equal(t, status.User.Metadata.Uid, itm.Entry.Common.UserRef.Uid)
		}
	}

	{
		res, err := auditLogC.ListAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
			UserRef: umetav1.GetObjectReference(meUsr),
		})
		assert.Nil(t, err)

		assert.True(t, len(res.Items) > 0)

		for _, itm := range res.Items {
			assert.Equal(t, status.User.Metadata.Uid, itm.Entry.UserRef.Uid)
		}
	}

	{
		res, err := authenticationLogC.ListAuthenticationLog(ctx, &visibilityv1.ListAuthenticationLogRequest{
			UserRef: umetav1.GetObjectReference(meUsr),
		})
		assert.Nil(t, err)

		assert.True(t, len(res.Items) > 0)

		for _, itm := range res.Items {
			assert.Equal(t, status.User.Metadata.Uid, itm.Entry.UserRef.Uid)
		}
	}

	{
		res, err := policyPortalC.IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
			Downstream: &enterprisev1.IsAuthorizedRequest_SessionRef{
				SessionRef: umetav1.GetObjectReference(meSess),
			},
			Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(eeAPISvc),
			},
		})
		assert.Nil(t, err)

		assert.True(t, res.IsAuthorized)
	}

	{
		usr, err := coreC.CreateUser(ctx, &corev1.User{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &corev1.User_Spec{
				Type: corev1.User_Spec_WORKLOAD,
			},
		})
		assert.Nil(t, err)

		res, err := policyPortalC.IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
			Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
				UserRef: umetav1.GetObjectReference(usr),
			},
			Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(eeAPISvc),
			},
		})
		assert.Nil(t, err)

		assert.False(t, res.IsAuthorized)
	}

	{
		res, err := visibilityCoreC.ListUser(ctx, &vcorev1.ListUserOptions{})
		assert.Nil(t, err)

		usrList, err := coreC.ListUser(ctx, &corev1.ListUserOptions{})
		assert.Nil(t, err)

		assert.Equal(t, res.ListResponseMeta.TotalCount, usrList.ListResponseMeta.TotalCount)

		for _, usr := range usrList.Items {
			assert.True(t, slices.ContainsFunc(res.Items, func(itm *corev1.User) bool {
				return itm.Metadata.Uid == usr.Metadata.Uid
			}))
		}
	}

	return nil
}
