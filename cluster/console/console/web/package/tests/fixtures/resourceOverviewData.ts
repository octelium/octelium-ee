import * as AccessP from "@/apis/accessv1/accessv1";
import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Resource } from "@/utils/pb";

const at = (seconds: number) =>
  new Date(Date.now() + seconds * 1000).toISOString();

const HOUR = 3600;
const DAY = 24 * HOUR;

const ref = (apiVersion: string, kind: string, name: string) => ({
  apiVersion,
  kind,
  name,
  uid: `uid-${kind.toLowerCase()}-${name}`,
});

const md = (name: string, extra: Record<string, unknown> = {}) => ({
  name,
  uid: `9b1f0c1e-${name.length.toString().padStart(4, "0")}-4c3a-9d2e-5b7a${name.length
    .toString()
    .padStart(8, "0")}`,
  createdAt: at(-12 * DAY),
  updatedAt: at(-2 * HOUR),
  resourceVersion: "019a3b2c-7d4e-7f10-8a2b-3c4d5e6f7a8b",
  ...extra,
});

const geoip = {
  ip: "203.0.113.24",
  ipVersion: "V4",
  country: { code: "DE", name: "Germany" },
  region: { code: "BE", name: "Berlin" },
  city: { name: "Berlin" },
  continent: { code: "EU", name: "Europe" },
  network: {
    asn: 3320,
    isp: "Deutsche Telekom AG",
    organization: "Deutsche Telekom AG",
    domain: "telekom.de",
  },
  timezone: { id: "Europe/Berlin", offset: 1 },
  postalCode: "10115",
  coordinates: { latitude: 52.5244, longitude: 13.4105, accuracyRadius: 20 },
};

const browserAuthentication = (seconds: number, ip = "203.0.113.24") => ({
  setAt: at(seconds),
  tokenID: "3f2a9c1e-77b4-4d1a-9c55-0e1f2a3b4c5d",
  accessTokenDuration: { minutes: 30 },
  refreshTokenDuration: { hours: 12 },
  info: {
    type: "IDENTITY_PROVIDER",
    aal: "AAL2",
    identityProvider: {
      identityProviderRef: ref("core/v1", "IdentityProvider", "github"),
      type: "GITHUB",
      identifier: "alice-gh",
      email: "alice@example.com",
    },
    downstream: {
      ipAddress: ip,
      userAgent:
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.6723.92 Safari/537.36",
    },
    geoip,
  },
});

const sources: Record<string, () => Resource> = {
  "core/sessions/alice-browser-k3x9": () =>
    CoreP.Session.fromJson({
      apiVersion: "core/v1",
      kind: "Session",
      metadata: md("alice-browser-k3x9", { createdAt: at(-7 * HOUR) }),
      spec: {
        state: "ACTIVE",
        expiresAt: at(5 * HOUR),
        authorization: {
          policies: ["allow-engineering", "require-mfa"],
          inlinePolicies: [{ name: "deny-production-writes" }],
        },
      },
      status: {
        type: "CLIENTLESS",
        isBrowser: true,
        userRef: ref("core/v1", "User", "alice"),
        deviceRef: ref("core/v1", "Device", "alice-macbook"),
        authentication: browserAuthentication(-25 * 60),
        initialAuthentication: browserAuthentication(-7 * HOUR),
        lastAuthentications: [
          browserAuthentication(-3 * HOUR, "203.0.113.24"),
          browserAuthentication(-5 * HOUR, "198.51.100.7"),
        ],
        totalAuthentications: 4,
        authenticatorAction: "AUTHENTICATION_RECOMMENDED",
        scopes: [
          {
            service: {
              filter: { names: ["grafana", "wiki"], namespaces: ["internal"] },
            },
          },
          { api: { all: {} } },
        ],
      },
    }),
  "core/sessions/bob-client-q7m2": () =>
    CoreP.Session.fromJson({
      apiVersion: "core/v1",
      kind: "Session",
      metadata: md("bob-client-q7m2", { createdAt: at(-3 * DAY) }),
      spec: {
        state: "ACTIVE",
        expiresAt: at(40 * 60),
      },
      status: {
        type: "CLIENT",
        isConnected: true,
        userRef: ref("core/v1", "User", "bob"),
        deviceRef: ref("core/v1", "Device", "bob-thinkpad"),
        credentialRef: ref("core/v1", "Credential", "bob-auth-token-w8k2"),
        authentication: {
          setAt: at(-2 * HOUR),
          accessTokenDuration: { hours: 1 },
          refreshTokenDuration: { days: 3 },
          info: {
            type: "CREDENTIAL",
            aal: "AAL1",
            credential: {
              credentialRef: ref("core/v1", "Credential", "bob-auth-token-w8k2"),
              type: "AUTH_TOKEN",
              tokenID: "c0ffee00-1234-4567-89ab-cdef01234567",
            },
            downstream: {
              ipAddress: "198.51.100.42",
              userAgent: "octelium-cli/0.21.3 (linux; amd64)",
              clientVersion: "0.21.3",
            },
          },
        },
        totalAuthentications: 18,
        totalConnections: 9,
        lastConnections: [
          { startedAt: at(-30 * HOUR), endedAt: at(-26 * HOUR) },
          { startedAt: at(-50 * HOUR), endedAt: at(-49 * HOUR) },
        ],
        connection: {
          type: "WIREGUARD",
          l3Mode: "BOTH",
          startedAt: at(-2 * HOUR),
          lastSeenAt: at(-12),
          addresses: [{ v4: "100.64.0.12/32", v6: "fdee:b0b::c/128" }],
          eSSHEnable: true,
          eSSHPort: 22022,
          x25519PublicKey: "q2v7m9c3Zr4P1aYh0uQe6Hn8Kx5Wb2Ts7Lf3Ji9Dg1E=",
          upstreams: [
            {
              port: 15432,
              l4Type: "TCP",
              mode: "DEFAULT",
              serviceRef: ref("core/v1", "Service", "postgres-dev.default"),
              backend: { host: "localhost", port: 5432 },
            },
            {
              port: 22022,
              l4Type: "TCP",
              mode: "ESSH",
              serviceRef: ref("core/v1", "Service", "bob-laptop.default"),
            },
          ],
          publishedServices: [
            {
              serviceRef: ref("core/v1", "Service", "grafana.monitoring"),
              address: "127.0.0.1",
              port: 3000,
            },
          ],
          serviceOptions: {
            requestedServices: [
              { serviceRef: ref("core/v1", "Service", "postgres-dev.default") },
            ],
          },
        },
      },
    }),
  "core/services/ssh-k8s-2.default": () =>
    CoreP.Service.fromJson({
      apiVersion: "core/v1",
      kind: "Service",
      metadata: md("ssh-k8s-2.default", {
        description: "SSH access to the second Kubernetes control-plane node.",
        labels: { team: "platform", env: "production" },
      }),
      spec: {
        mode: "SSH",
        port: 22,
        config: {
          upstream: { url: "ssh://10.0.12.4:22", user: "ubuntu" },
          ssh: {
            eSSHMode: false,
            user: "ubuntu",
          },
        },
        dynamicConfig: {
          configs: [{ name: "readonly" }],
          rules: [{ configName: "readonly" }],
        },
        authorization: {
          policies: ["allow-platform"],
          inlinePolicies: [{ name: "ssh-business-hours" }],
        },
        attrs: { tier: "control-plane", criticality: "high" },
      },
      status: {
        namespaceRef: ref("core/v1", "Namespace", "default"),
        regionRef: ref("core/v1", "Region", "default"),
        port: 22,
        primaryHostname: "ssh-k8s-2",
        additionalHostnames: ["k8s-node-2"],
        addresses: [
          { dualStackIP: { ipv4: "100.64.1.17", ipv6: "fdee:5e7::11" } },
        ],
      },
    }),
  "core/services/grafana.monitoring": () =>
    CoreP.Service.fromJson({
      apiVersion: "core/v1",
      kind: "Service",
      metadata: md("grafana.monitoring", { displayName: "Grafana" }),
      spec: {
        mode: "WEB",
        isPublic: true,
        isTLS: true,
        config: {
          upstream: {
            loadbalance: {
              endpoints: [
                { url: "http://grafana-0.monitoring.svc:3000" },
                { url: "http://grafana-1.monitoring.svc:3000" },
              ],
            },
          },
          http: {},
        },
        deployment: { replicas: 2 },
      },
      status: {
        namespaceRef: ref("core/v1", "Namespace", "monitoring"),
        regionRef: ref("core/v1", "Region", "default"),
        port: 8080,
        primaryHostname: "grafana.monitoring",
      },
    }),
  "core/users/alice": () =>
    CoreP.User.fromJson({
      apiVersion: "core/v1",
      kind: "User",
      metadata: md("alice", {
        displayName: "Alice Johnson",
        picURL: "",
      }),
      spec: {
        type: "HUMAN",
        email: "alice@example.com",
        groups: ["engineering", "on-call"],
        info: {
          firstName: "Alice",
          lastName: "Johnson",
          country: "DE",
          locale: "en-US",
        },
        authentication: {
          identities: [{ identityProvider: "github", identifier: "alice-gh" }],
        },
        session: {
          clientDuration: { days: 7 },
          clientlessDuration: { hours: 12 },
          maxPerUser: 10,
        },
        authorization: { policies: ["allow-engineering"] },
        attrs: { team: "platform", level: 4 },
      },
      status: {},
    }),
  "core/groups/engineering": () =>
    CoreP.Group.fromJson({
      apiVersion: "core/v1",
      kind: "Group",
      metadata: md("engineering", { displayName: "Engineering" }),
      spec: {
        authorization: {
          policies: ["allow-engineering"],
          inlinePolicies: [{ name: "deny-finance" }],
        },
        attrs: { costCenter: "R&D" },
      },
      status: {},
    }),
  "core/namespaces/default": () =>
    CoreP.Namespace.fromJson({
      apiVersion: "core/v1",
      kind: "Namespace",
      metadata: md("default"),
      spec: { authorization: { policies: ["allow-platform"] } },
      status: {},
    }),
  "core/credentials/bob-auth-token-w8k2": () =>
    CoreP.Credential.fromJson({
      apiVersion: "core/v1",
      kind: "Credential",
      metadata: md("bob-auth-token-w8k2"),
      spec: {
        type: "AUTH_TOKEN",
        user: "bob",
        sessionType: "CLIENT",
        maxAuthentications: 100,
        expiresAt: at(23 * DAY),
        authorization: { policies: ["allow-ci"] },
      },
      status: {
        userRef: ref("core/v1", "User", "bob"),
        id: "crd-7f3a1b",
        tokenID: "c0ffee00-1234-4567-89ab-cdef01234567",
        lastRotationAt: at(-4 * DAY),
        totalRotations: 3,
        totalAuthentications: 18,
      },
    }),
  "core/devices/alice-macbook": () =>
    CoreP.Device.fromJson({
      apiVersion: "core/v1",
      kind: "Device",
      metadata: md("alice-macbook"),
      spec: { state: "ACTIVE" },
      status: {
        userRef: ref("core/v1", "User", "alice"),
        osType: "MAC",
        hostname: "alice-mbp.local",
        id: "C02XG2JHJGH5-7b6f",
        serialNumber: "C02XG2JHJGH5",
        macAddresses: ["a4:83:e7:12:9c:01", "a4:83:e7:12:9c:02"],
        posture: {
          riskLevel: "LOW",
          diskEncryption: "PASS",
          compliant: "PASS",
          threatFree: "PASS",
          firewall: "PASS",
          screenLock: "FAIL",
          secureBoot: "PASS",
          osUpToDate: "FAIL",
          lastSyncAt: at(-20 * 60),
          lastSeenAt: at(-5 * 60),
          expiresAt: at(6 * HOUR),
        },
        binding: {
          uid: "bnd-2c4e6a",
          ownerRef: ref("enterprise/v1", "DeviceManager", "kandji"),
          externalID: "kandji-8812",
          state: "ACCEPTED",
          validity: "VALID",
          acceptedAt: at(-9 * DAY),
          lastVerifiedAt: at(-20 * 60),
          nextVerificationAt: at(40 * 60),
        },
      },
    }),
  "core/authenticators/alice-yubikey": () =>
    CoreP.Authenticator.fromJson({
      apiVersion: "core/v1",
      kind: "Authenticator",
      metadata: md("alice-yubikey"),
      spec: { displayName: "YubiKey 5C NFC", state: "ACTIVE" },
      status: {
        userRef: ref("core/v1", "User", "alice"),
        type: "FIDO",
        isRegistered: true,
        successfulAuthentications: 41,
        failedAuthentications: 2,
        totalAuthenticationAttempts: 43,
        info: {
          fido: {
            type: "ROAMING",
            aaguid: "ee882879-721c-4913-9775-3dfcce97072a",
            isHardware: true,
            isAttestationVerified: true,
            signCount: 1287,
          },
        },
      },
    }),
  "core/identityproviders/github": () =>
    CoreP.IdentityProvider.fromJson({
      apiVersion: "core/v1",
      kind: "IdentityProvider",
      metadata: md("github"),
      spec: {
        displayName: "GitHub",
        github: {
          clientID: "Iv1.8a61f9b3a7aba766",
          clientSecret: { fromSecret: "github-client-secret" },
        },
        aalRules: [{ aal: "AAL2" }],
      },
      status: { type: "GITHUB" },
    }),
  "core/policies/allow-engineering": () =>
    CoreP.Policy.fromJson({
      apiVersion: "core/v1",
      kind: "Policy",
      metadata: md("allow-engineering", {
        description: "Engineers can reach the internal tooling Services.",
      }),
      spec: {
        rules: [
          {
            name: "allow-internal",
            effect: "ALLOW",
            priority: 1,
            condition: { match: "ctx.user.spec.groups.exists(g, g == 'engineering')" },
          },
          {
            name: "deny-production",
            effect: "DENY",
            priority: 0,
            condition: { match: "ctx.service.metadata.labels.env == 'production'" },
          },
        ],
        enforcementRules: [{ effect: "ENFORCE", condition: { match: "true" } }],
      },
      status: {},
    }),
  "core/regions/default": () =>
    CoreP.Region.fromJson({
      apiVersion: "core/v1",
      kind: "Region",
      metadata: md("default"),
      spec: {},
      status: {
        index: 0,
        version: "v0.21.3",
        publicHostname: "default.example.com",
        ingressAddresses: ["203.0.113.10", "203.0.113.11"],
      },
    }),
  "core/gateways/default-0": () =>
    CoreP.Gateway.fromJson({
      apiVersion: "core/v1",
      kind: "Gateway",
      metadata: md("default-0"),
      spec: {},
      status: {
        regionRef: ref("core/v1", "Region", "default"),
        id: "gw-0a1b2c3d",
        index: 0,
        cidr: { v4: "100.64.0.0/16", v6: "fdee:b0b::/64" },
        hostname: "gw-0.default.example.com",
        publicIPs: ["203.0.113.20"],
        wireguard: {
          port: 53820,
          publicKey: "m4r6J8Xn2pQw5sTk3vYb7cLz9eGh1uAo0iDf4RWyNkE=",
          keyRotatedAt: at(-6 * DAY),
        },
        quicv0: { port: 8443 },
      },
    }),
  "core/secrets/db-password": () =>
    CoreP.Secret.fromJson({
      apiVersion: "core/v1",
      kind: "Secret",
      metadata: md("db-password"),
      spec: {},
      status: {},
    }),
  "enterprise/certificates/cluster-wildcard": () =>
    EnterpriseP.Certificate.fromJson({
      apiVersion: "enterprise/v1",
      kind: "Certificate",
      metadata: md("cluster-wildcard"),
      spec: { mode: "MANAGED" },
      status: {
        certificateIssuerRef: ref("enterprise/v1", "CertificateIssuer", "letsencrypt"),
        secretRef: ref("enterprise/v1", "Secret", "crt-cluster-wildcard"),
        successfulIssuances: 6,
        failedIssuances: 1,
        issuance: {
          createdAt: at(-30 * DAY),
          state: "SUCCESS",
          issuanceStartedAt: at(-30 * DAY),
          issuanceCompletedAt: at(-30 * DAY + 90),
          expiresAt: at(60 * DAY),
        },
        lastIssuances: [
          {
            createdAt: at(-120 * DAY),
            state: "SUCCESS",
            issuanceCompletedAt: at(-120 * DAY + 80),
            expiresAt: at(-30 * DAY),
          },
          {
            createdAt: at(-121 * DAY),
            state: "FAILED",
            issuanceCompletedAt: at(-121 * DAY + 40),
          },
        ],
        info: {
          commonName: "*.example.com",
          subject: "CN=*.example.com",
          issuer: "CN=R11,O=Let's Encrypt,C=US",
          notBefore: at(-30 * DAY),
          notAfter: at(60 * DAY),
          dnsNames: ["*.example.com", "example.com", "*.local.example.com"],
        },
      },
    }),
  "enterprise/certificates/partner-api": () =>
    EnterpriseP.Certificate.fromJson({
      apiVersion: "enterprise/v1",
      kind: "Certificate",
      metadata: md("partner-api"),
      spec: { mode: "MANUAL" },
      status: {
        serviceRef: ref("core/v1", "Service", "partner-api.default"),
        info: {
          commonName: "partner-api.example.com",
          subject: "CN=partner-api.example.com,O=Example Corp",
          issuer: "CN=DigiCert Global G2 TLS RSA SHA256 2020 CA1",
          notBefore: at(-340 * DAY),
          notAfter: at(9 * DAY),
          dnsNames: ["partner-api.example.com"],
        },
      },
    }),
  "enterprise/certificateissuers/letsencrypt": () =>
    EnterpriseP.CertificateIssuer.fromJson({
      apiVersion: "enterprise/v1",
      kind: "CertificateIssuer",
      metadata: md("letsencrypt"),
      spec: {
        acme: {
          server: "https://acme-v02.api.letsencrypt.org/directory",
          email: "certs@example.com",
          solver: { dns: {} },
        },
      },
      status: { state: "READY" },
    }),
  "enterprise/secretstores/vault": () =>
    EnterpriseP.SecretStore.fromJson({
      apiVersion: "enterprise/v1",
      kind: "SecretStore",
      metadata: md("vault"),
      spec: {
        hashicorpVault: {
          address: "https://vault.example.com:8200",
          role: "octelium",
          key: "octelium-root",
        },
      },
      status: {
        type: "TYPE_HASHICORP_VAULT",
        state: "OK",
        synchronization: {
          state: "SUCCESS",
          createdAt: at(-3 * HOUR),
          completedAt: at(-3 * HOUR + 14),
        },
        lastSynchronizations: [
          {
            state: "FAILED",
            createdAt: at(-2 * DAY),
            completedAt: at(-2 * DAY + 4),
          },
        ],
      },
    }),
  "enterprise/directoryproviders/google": () =>
    EnterpriseP.DirectoryProvider.fromJson({
      apiVersion: "enterprise/v1",
      kind: "DirectoryProvider",
      metadata: md("google"),
      spec: {
        googleWorkspace: {
          customer: "C03az79cb",
          impersonateSubject: "admin@example.com",
          serviceAccount: { fromSecret: "gws-service-account" },
          polling: { interval: { minutes: 15 } },
        },
      },
      status: {
        id: "dp-6b1c2d",
        userRef: ref("core/v1", "User", "octelium-dirsync-google"),
        synchronization: {
          state: "SUCCESS",
          createdAt: at(-14 * 60),
          completedAt: at(-13 * 60),
        },
      },
    }),
  "enterprise/directoryproviders/okta": () =>
    EnterpriseP.DirectoryProvider.fromJson({
      apiVersion: "enterprise/v1",
      kind: "DirectoryProvider",
      metadata: md("okta"),
      spec: { scim: {} },
      status: { id: "c6a1f0e2-91d2-4a1e-8f1b-7d5e3c2b1a09" },
    }),
  "enterprise/dnsproviders/cloudflare": () =>
    EnterpriseP.DNSProvider.fromJson({
      apiVersion: "enterprise/v1",
      kind: "DNSProvider",
      metadata: md("cloudflare"),
      spec: {
        cloudflare: {
          email: "dns-admin@example.com",
          apiToken: { fromSecret: "cloudflare-api-token" },
        },
      },
      status: {},
    }),
  "enterprise/collectorexporters/datadog": () =>
    EnterpriseP.CollectorExporter.fromJson({
      apiVersion: "enterprise/v1",
      kind: "CollectorExporter",
      metadata: md("datadog"),
      spec: {
        datadog: {
          api: { site: "datadoghq.eu", key: { fromSecret: "datadog-api-key" } },
        },
      },
      status: {},
    }),
  "enterprise/secrets/crt-cluster-wildcard": () =>
    EnterpriseP.Secret.fromJson({
      apiVersion: "enterprise/v1",
      kind: "Secret",
      metadata: md("crt-cluster-wildcard"),
      spec: {},
      status: {},
    }),
  "access/requests/req-dba-live": () =>
    AccessP.Request.fromJson({
      apiVersion: "access/v1",
      kind: "Request",
      metadata: md("req-dba-live", { createdAt: at(-50 * 60) }),
      spec: {
        resource: { serviceRef: ref("core/v1", "Service", "postgres-prod.default") },
        justification: "Investigating the slow checkout queries (INC-2231).",
        duration: { hours: 2 },
        urgency: "HIGH",
      },
      status: {
        userRef: ref("core/v1", "User", "alice"),
        state: { status: "APPROVED", createdAt: at(-40 * 60) },
        approvalEndAt: at(-40 * 60),
        accessEndsAt: at(80 * 60),
        effectiveDuration: { hours: 2 },
      },
    }),
  "cordium/workspaces/aoo": () =>
    CordiumP.Workspace.fromJson({
      apiVersion: "cordium/v1",
      kind: "Workspace",
      metadata: md("aoo"),
      spec: {
        image: { registry: { url: "ghcr.io/example/dev:latest" } },
        repository: { url: "https://github.com/example/app" },
        limit: { cpu: { millicores: 4000 }, memory: { megabytes: 8192 } },
        applications: [{ name: "web", port: 3000 }],
      },
      status: {
        state: "RUNNING",
        userRef: ref("core/v1", "User", "alice"),
        spaceRef: ref("cordium/v1", "Space", "default.alice"),
        templateRef: ref("cordium/v1", "Template", "default.default.alice"),
        regionRef: ref("cordium/v1", "Region", "default"),
        sessionRef: ref("core/v1", "Session", "alice-browser-k3x9"),
        hostname: "aoo.default.alice",
        spaceType: "USER",
        successfulRuns: 12,
        currentStateSetAt: at(-2 * HOUR),
        lastActivityAt: at(-4 * 60),
        run: { id: "run-8c2d", initializedAt: at(-2 * HOUR) },
      },
    }),
  "cordium/templates/default.default.alice": () =>
    CordiumP.Template.fromJson({
      apiVersion: "cordium/v1",
      kind: "Template",
      metadata: md("default.default.alice"),
      spec: {
        image: { registry: { url: "ghcr.io/example/dev:latest" } },
        limit: { cpu: { millicores: 4000 }, memory: { megabytes: 8192 } },
      },
      status: {
        spaceRef: ref("cordium/v1", "Space", "default.alice"),
        userRef: ref("core/v1", "User", "alice"),
        buildInfo: {
          currentReadyBuildID: "bld-91f2",
          builds: [
            {
              id: "bld-91f2",
              state: "STATE_READY",
              tags: ["latest"],
              startedAt: at(-3 * DAY),
              doneAt: at(-3 * DAY + 420),
            },
          ],
        },
      },
    }),
  "cordium/spaces/default.alice": () =>
    CordiumP.Space.fromJson({
      apiVersion: "cordium/v1",
      kind: "Space",
      metadata: md("default.alice"),
      spec: {},
      status: { userRef: ref("core/v1", "User", "alice"), type: "USER" },
    }),
  "cordium/memberships/default.alice.bob": () =>
    CordiumP.Membership.fromJson({
      apiVersion: "cordium/v1",
      kind: "Membership",
      metadata: md("default.alice.bob"),
      spec: { role: "ADMIN" },
      status: {
        userRef: ref("core/v1", "User", "bob"),
        spaceRef: ref("cordium/v1", "Space", "default.alice"),
        userInfo: { displayName: "Bob Smith" },
      },
    }),
  "cordium/gitproviders/github.default.alice": () =>
    CordiumP.GitProvider.fromJson({
      apiVersion: "cordium/v1",
      kind: "GitProvider",
      metadata: md("github.default.alice"),
      spec: {
        github: { clientID: "Iv1.0f0a2b3c4d5e6f70", scopes: ["repo", "read:org"] },
      },
      status: {
        spaceRef: ref("cordium/v1", "Space", "default.alice"),
        userRef: ref("core/v1", "User", "alice"),
      },
    }),
  "cordium/usersecrets/alice-ssh": () =>
    CordiumP.UserSecret.fromJson({
      apiVersion: "cordium/v1",
      kind: "UserSecret",
      metadata: md("alice-ssh"),
      spec: {},
      status: {
        userRef: ref("core/v1", "User", "alice"),
        sshKey: {
          publicKey:
            "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGq2c0x8F1r7m3Zk9wPpX4yH6sD2vB8nT5uL1eQ3jR0a alice@workspace",
        },
      },
    }),
  "access/integrations/slack-access": () =>
    AccessP.Integration.fromJson({
      apiVersion: "access/v1",
      kind: "Integration",
      metadata: md("slack-access"),
      spec: {
        slack: {
          teamID: "T024BE7LD",
          channelID: "C01ACCESS",
          botToken: { fromSecret: "slack-bot-token" },
          signingSecret: { fromSecret: "slack-signing-secret" },
        },
      },
      status: {},
    }),
  "cordium/regions/default": () =>
    CordiumP.Region.fromJson({
      apiVersion: "cordium/v1",
      kind: "Region",
      metadata: md("default"),
      spec: {},
      status: { country: "DE", city: "Frankfurt" },
    }),
};

export const resources: Record<string, Resource> = {};
export const errors: string[] = [];

for (const [path, build] of Object.entries(sources)) {
  try {
    resources[path] = build();
  } catch (err) {
    errors.push(`${path}: ${err instanceof Error ? err.message : String(err)}`);
  }
}
