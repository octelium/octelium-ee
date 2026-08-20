// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
)

const (
	CapVault    scenario.Capability = "vault"
	CapKeycloak scenario.Capability = "keycloak"
	CapOTLPSink scenario.Capability = "otlp-sink"
)

const (
	StateVaultAddr    = "vaultAddr"
	StateVaultRole    = "vaultRole"
	StateVaultKey     = "vaultKey"
	StateVaultPod     = "vaultPod"
	StateVaultToken   = "vaultToken"
	StateVaultNS      = "vaultNamespace"
	StateKeycloakURL  = "keycloakURL"
	StateKeycloakHost = "keycloakHostURL"
	StateKeycloakUser = "keycloakUser"
	StateKeycloakPass = "keycloakPassword"
	StateSinkGRPC     = "otlpSinkGRPC"
	StateSinkHTTP     = "otlpSinkHTTP"
	StateSinkAuthGRPC = "otlpSinkAuthGRPC"
	StateSinkAuthHTTP = "otlpSinkAuthHTTP"
	StateSinkToken    = "otlpSinkToken"
	StateSinkPod      = "otlpSinkPod"
	StateSinkNS       = "otlpSinkNamespace"
	StateSinkDir      = "otlpSinkDir"
)

const (
	fixtureNS = "e2e"

	vaultRelease   = "vault"
	vaultPod       = "vault-0"
	vaultRootToken = "root"
	vaultRole      = "octelium-e2e"
	vaultKey       = "octelium-e2e"

	keycloakUser     = "admin"
	keycloakPassword = "octelium-e2e"
	keycloakNodePort = 31080

	sinkDeployment = "otel-sink"
	sinkDir        = "/data"
	sinkToken      = "octelium-e2e-sink-token"
)

func init() {
	register(FullScenario, opts{
		description: "The Enterprise scenario plus SPIRE, Vault, Keycloak and an OTLP sink",
		spire:       true,
		budget:      120 * time.Minute,
		fixtures:    true,
	})
}

func withFixtures(ret *scenario.Scenario) {
	ret.Caps = append(ret.Caps, CapVault, CapKeycloak, CapOTLPSink)

	ret.Hooks.PostPrepare = append(ret.Hooks.PostPrepare,
		scenario.Step{Name: "fixtures/namespace", Run: stepFixtureNamespace},
		scenario.Step{Name: "fixtures/vault", Run: stepVault},
		scenario.Step{Name: "fixtures/keycloak", Run: stepKeycloak},
		scenario.Step{Name: "fixtures/otlp-sink", Run: stepOTLPSink},
	)
}

func stepFixtureNamespace(ctx context.Context, r *scenario.Runner) error {
	return r.Bash(ctx, fmt.Sprintf(
		`kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -`, fixtureNS))
}

func stepVault(ctx context.Context, r *scenario.Runner) error {
	if err := r.Bash(ctx, `
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update hashicorp
`); err != nil {
		return err
	}

	if err := r.Bash(ctx, fmt.Sprintf(`
helm upgrade --install %s hashicorp/vault --namespace %s \
  --set server.dev.enabled=true \
  --set server.dev.devRootToken=%s \
  --set injector.enabled=false \
  --set server.logLevel=debug \
  --wait --timeout 10m
kubectl wait --for=condition=Ready pod/%s --namespace %s --timeout=600s
`, vaultRelease, fixtureNS, vaultRootToken, vaultPod, fixtureNS)); err != nil {
		return err
	}

	if err := r.BashInput(ctx, `kubectl apply -f -`, fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: octelium-e2e-vault-auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
  - kind: ServiceAccount
    name: %s
    namespace: %s
`, vaultRelease, fixtureNS)); err != nil {
		return err
	}

	if err := r.Bash(ctx, fmt.Sprintf(`
kubectl exec -n %[1]s %[2]s -- sh -c '
set -e
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=%[3]s

vault secrets enable transit || true
vault write -f transit/keys/%[4]s

vault auth enable kubernetes || true
vault write auth/kubernetes/config \
  kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443"

vault policy write %[5]s - <<POLICY
path "transit/encrypt/%[4]s" {
  capabilities = ["update"]
}
path "transit/decrypt/%[4]s" {
  capabilities = ["update"]
}
path "transit/keys/%[4]s" {
  capabilities = ["read"]
}
POLICY

vault write auth/kubernetes/role/%[5]s \
  bound_service_account_names=octeliumee-secretman \
  bound_service_account_namespaces=octelium \
  policies=%[5]s \
  ttl=1h
'
`, fixtureNS, vaultPod, vaultRootToken, vaultKey, vaultRole)); err != nil {
		return err
	}

	r.State.Set(StateVaultNS, fixtureNS)
	r.State.Set(StateVaultPod, vaultPod)
	r.State.Set(StateVaultToken, vaultRootToken)
	r.State.Set(StateVaultRole, vaultRole)
	r.State.Set(StateVaultKey, vaultKey)
	r.State.Set(StateVaultAddr,
		fmt.Sprintf("http://%s.%s.svc:8200", vaultRelease, fixtureNS))

	return r.SaveState()
}

func stepKeycloak(ctx context.Context, r *scenario.Runner) error {
	if err := r.BashInput(ctx, `kubectl apply -f -`, fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak
  namespace: %[1]s
  labels:
    app: keycloak
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keycloak
  template:
    metadata:
      labels:
        app: keycloak
    spec:
      containers:
        - name: keycloak
          image: quay.io/keycloak/keycloak:26.0
          args: ["start-dev", "--http-port=8080"]
          env:
            - name: KC_BOOTSTRAP_ADMIN_USERNAME
              value: %[2]s
            - name: KC_BOOTSTRAP_ADMIN_PASSWORD
              value: %[3]s
            - name: KEYCLOAK_ADMIN
              value: %[2]s
            - name: KEYCLOAK_ADMIN_PASSWORD
              value: %[3]s
            - name: KC_HEALTH_ENABLED
              value: "true"
          ports:
            - containerPort: 8080
          readinessProbe:
            httpGet:
              path: /realms/master
              port: 8080
            initialDelaySeconds: 20
            periodSeconds: 5
            failureThreshold: 60
---
apiVersion: v1
kind: Service
metadata:
  name: keycloak
  namespace: %[1]s
spec:
  type: NodePort
  selector:
    app: keycloak
  ports:
    - port: 8080
      targetPort: 8080
      nodePort: %[4]d
`, fixtureNS, keycloakUser, keycloakPassword, keycloakNodePort)); err != nil {
		return err
	}

	if err := r.Bash(ctx, fmt.Sprintf(
		`kubectl wait --for=condition=available deployment/keycloak --namespace %s --timeout=600s`,
		fixtureNS)); err != nil {
		return err
	}

	r.State.Set(StateKeycloakUser, keycloakUser)
	r.State.Set(StateKeycloakPass, keycloakPassword)
	r.State.Set(StateKeycloakURL, fmt.Sprintf("http://keycloak.%s.svc:8080", fixtureNS))
	r.State.Set(StateKeycloakHost, fmt.Sprintf("http://127.0.0.1:%d", keycloakNodePort))

	return r.SaveState()
}

func stepOTLPSink(ctx context.Context, r *scenario.Runner) error {
	if err := r.BashInput(ctx, `kubectl apply -f -`, fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-sink
  namespace: %[1]s
data:
  config.yaml: |
    extensions:
      bearertokenauth/server:
        filename: %[2]s/auth-token
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
            include_metadata: true
          http:
            endpoint: 0.0.0.0:4318
            include_metadata: true
      otlp/auth:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4319
            include_metadata: true
            auth:
              authenticator: bearertokenauth/server
          http:
            endpoint: 0.0.0.0:4320
            include_metadata: true
            auth:
              authenticator: bearertokenauth/server
    processors:
      batch:
        timeout: 1s
    exporters:
      debug:
        verbosity: basic
      file/logs:
        path: %[2]s/logs.json
      file/metrics:
        path: %[2]s/metrics.json
    service:
      extensions: [bearertokenauth/server]
      telemetry:
        logs:
          level: debug
      pipelines:
        logs:
          receivers: [otlp, otlp/auth]
          processors: [batch]
          exporters: [file/logs, debug]
        metrics:
          receivers: [otlp, otlp/auth]
          processors: [batch]
          exporters: [file/metrics, debug]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %[3]s
  namespace: %[1]s
  labels:
    app: otel-sink
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otel-sink
  template:
    metadata:
      labels:
        app: otel-sink
    spec:
      securityContext:
        fsGroup: 10001
      initContainers:
        - name: auth-token
          image: busybox:1.37
          command: ["sh", "-c", "printf %%s %[4]s > %[2]s/auth-token"]
          volumeMounts:
            - name: data
              mountPath: %[2]s
      containers:
        - name: collector
          image: otel/opentelemetry-collector-contrib:0.153.0
          args: ["--config=/etc/otel/config.yaml"]
          ports:
            - containerPort: 4317
            - containerPort: 4318
            - containerPort: 4319
            - containerPort: 4320
          volumeMounts:
            - name: config
              mountPath: /etc/otel
            - name: data
              mountPath: %[2]s
      volumes:
        - name: config
          configMap:
            name: otel-sink
        - name: data
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: otel-sink
  namespace: %[1]s
spec:
  selector:
    app: otel-sink
  ports:
    - name: grpc
      port: 4317
      targetPort: 4317
    - name: http
      port: 4318
      targetPort: 4318
    - name: grpc-auth
      port: 4319
      targetPort: 4319
    - name: http-auth
      port: 4320
      targetPort: 4320
`, fixtureNS, sinkDir, sinkDeployment, sinkToken)); err != nil {
		return err
	}

	if err := r.Bash(ctx, fmt.Sprintf(
		`kubectl wait --for=condition=available deployment/%s --namespace %s --timeout=600s`,
		sinkDeployment, fixtureNS)); err != nil {
		return err
	}

	r.State.Set(StateSinkNS, fixtureNS)
	r.State.Set(StateSinkPod, fmt.Sprintf("deployment/%s", sinkDeployment))
	r.State.Set(StateSinkDir, sinkDir)
	r.State.Set(StateSinkGRPC, fmt.Sprintf("otel-sink.%s.svc:4317", fixtureNS))
	r.State.Set(StateSinkHTTP, fmt.Sprintf("http://otel-sink.%s.svc:4318", fixtureNS))
	r.State.Set(StateSinkAuthGRPC, fmt.Sprintf("otel-sink.%s.svc:4319", fixtureNS))
	r.State.Set(StateSinkAuthHTTP, fmt.Sprintf("http://otel-sink.%s.svc:4320", fixtureNS))
	r.State.Set(StateSinkToken, sinkToken)

	return r.SaveState()
}
