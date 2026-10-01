package harness

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/require"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FleetHost struct {
	ID                    int64  `json:"id"`
	UUID                  string `json:"uuid"`
	HardwareSerial        string `json:"hardware_serial"`
	Platform              string `json:"platform"`
	Status                string `json:"status"`
	SeenTime              string `json:"seen_time"`
	DiskEncryptionEnabled bool   `json:"disk_encryption_enabled"`
}

type Fleet struct {
	URL string

	mu     sync.Mutex
	bearer string
	hosts  []FleetHost
}

func (f *Fleet) SetBearer(bearer string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bearer = bearer
}

func (f *Fleet) SetHosts(hosts []FleetHost) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hosts = slices.Clone(hosts)
}

func (f *Fleet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Method != http.MethodGet || r.URL.Path != "/api/v1/fleet/hosts" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+f.bearer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 0 || page > 10000 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 || perPage > 1000 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	from := min(page*perPage, len(f.hosts))
	to := min(from+perPage, len(f.hosts))
	hosts := append([]FleetHost{}, f.hosts[from:to]...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Hosts []FleetHost `json:"hosts"`
	}{Hosts: hosts})
}

func (h *H) Fleet(t *testing.T, bearer string, hosts []FleetHost) *Fleet {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.Nil(t, err)

	crt := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: h.ExternalIP},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(h.ExternalIP); ip != nil {
		crt.IPAddresses = []net.IP{ip}
	} else {
		crt.DNSNames = []string{h.ExternalIP}
	}
	der, err := x509.CreateCertificate(rand.Reader, crt, crt, &key.PublicKey, key)
	require.Nil(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	port := h.Port()
	lis, err := listenAllInterfaces(port)
	require.Nil(t, err)
	ret := &Fleet{URL: fmt.Sprintf("https://%s", net.JoinHostPort(h.ExternalIP, strconv.Itoa(port)))}
	ret.SetBearer(bearer)
	ret.SetHosts(hosts)

	srv := &http.Server{
		Handler:           ret,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go srv.Serve(tls.NewListener(lis, &tls.Config{
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			{Certificate: [][]byte{der}, PrivateKey: key},
		},
	}))
	t.Cleanup(func() { srv.Close() })

	h.trustFleetCertificate(t, certPEM)

	return ret
}

func (h *H) trustFleetCertificate(t *testing.T, certPEM []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	name := h.Name()
	_, err := h.K8sC().CoreV1().ConfigMaps(vutils.K8sNS).Create(ctx, &k8scorev1.ConfigMap{
		ObjectMeta: k8smetav1.ObjectMeta{Name: name},
		Data:       map[string]string{"ca.pem": string(certPEM)},
	}, k8smetav1.CreateOptions{})
	require.Nil(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.K8sC().CoreV1().ConfigMaps(vutils.K8sNS).Delete(ctx, name,
			k8smetav1.DeleteOptions{}); err != nil {
			t.Errorf("Could not delete the Fleet CA ConfigMap: %+v", err)
		}
	})

	dep, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx,
		"octeliumee-nocturne", k8smetav1.GetOptions{})
	require.Nil(t, err)
	require.NotEmpty(t, dep.Spec.Template.Spec.Containers)
	containerName := dep.Spec.Template.Spec.Containers[0].Name
	env := slices.Clone(dep.Spec.Template.Spec.Containers[0].Env)
	mounts := slices.Clone(dep.Spec.Template.Spec.Containers[0].VolumeMounts)
	volumes := slices.Clone(dep.Spec.Template.Spec.Volumes)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		cur, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx,
			dep.Name, k8smetav1.GetOptions{})
		require.Nil(t, err)
		for i := range cur.Spec.Template.Spec.Containers {
			if cur.Spec.Template.Spec.Containers[i].Name == containerName {
				cur.Spec.Template.Spec.Containers[i].Env = env
				cur.Spec.Template.Spec.Containers[i].VolumeMounts = mounts
			}
		}
		cur.Spec.Template.Spec.Volumes = volumes
		_, err = h.K8sC().AppsV1().Deployments(vutils.K8sNS).Update(ctx,
			cur, k8smetav1.UpdateOptions{})
		require.Nil(t, err)
		require.Nil(t, h.WaitDeployment(ctx, dep.Name))
	})

	container := &dep.Spec.Template.Spec.Containers[0]
	container.Env = slices.DeleteFunc(container.Env, func(itm k8scorev1.EnvVar) bool {
		return itm.Name == "SSL_CERT_FILE"
	})
	container.Env = append(container.Env, k8scorev1.EnvVar{
		Name: "SSL_CERT_FILE", Value: "/etc/octelium-e2e-fleet/ca.pem",
	})
	container.VolumeMounts = append(container.VolumeMounts, k8scorev1.VolumeMount{
		Name: name, MountPath: "/etc/octelium-e2e-fleet", ReadOnly: true,
	})
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, k8scorev1.Volume{
		Name: name,
		VolumeSource: k8scorev1.VolumeSource{
			ConfigMap: &k8scorev1.ConfigMapVolumeSource{
				LocalObjectReference: k8scorev1.LocalObjectReference{Name: name},
			},
		},
	})
	_, err = h.K8sC().AppsV1().Deployments(vutils.K8sNS).Update(ctx,
		dep, k8smetav1.UpdateOptions{})
	require.Nil(t, err)
	h.MustWaitDeployment(t, dep.Name)
}
