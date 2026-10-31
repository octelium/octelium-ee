import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Struct } from "@/apis/google/protobuf/struct";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import Notifications from "@/components/Notifications";
import ClustermanMain from "@/pages/clusterman/Main";
import store from "@/store";
import { queryClient } from "@/utils";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, RouterProvider } from "react-router-dom";

declare global {
  interface Window {
    clusterman: {
      calls: { method: string; request: any }[];
      joins: any[];
    };
  }
}

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "idle";
const calls: { method: string; request: any }[] = [];
const joins: any[] = [];
window.clusterman = { calls, joins };

const ago = (minutes: number) =>
  Timestamp.fromDate(new Date(Date.now() - minutes * 60_000));

const versionInfo = (pkg: string, version: string) =>
  CoreP.Region_Status_VersionInfo.create({
    package: pkg,
    version,
    setAt: ago(600),
  });

const kubernetesExt = (secret: string) => ({
  enterpriseV1: Struct.fromJson({
    kubernetes: { kubeconfig: { fromSecret: secret } },
  }),
});

const region = (
  name: string,
  index: number,
  options: {
    displayName?: string;
    versions?: Record<string, string>;
    ext?: Record<string, Struct>;
  } = {},
) =>
  CoreP.Region.create({
    apiVersion: "core/v1",
    kind: "Region",
    metadata: {
      name,
      uid: `uid-${name}`,
      displayName: options.displayName ?? "",
      createdAt: ago(60 * 24 * (10 - index)),
    },
    spec: {},
    status: {
      index,
      versionInfoMap: Object.fromEntries(
        Object.entries(options.versions ?? {}).map(([pkg, version]) => [
          pkg,
          versionInfo(pkg, version),
        ]),
      ),
      ext: options.ext ?? {},
    },
  });

const regions = [
  region("east", 2, {
    versions: { octelium: "0.47.0" },
    ext: kubernetesExt("east-kubeconfig"),
  }),
  region("default", 0, {
    versions: { octelium: "0.47.0", octeliumee: "0.28.0", cordium: "0.14.0" },
  }),
  region("west", 1, {
    displayName: "US West",
    versions: { octelium: "0.47.0", octeliumee: "0.28.0" },
    ext: kubernetesExt("west-kubeconfig"),
  }),
];

const secrets = ["west-kubeconfig", "east-kubeconfig", "south-token"].map(
  (name) =>
    EnterpriseP.Secret.create({
      apiVersion: "enterprise/v1",
      kind: "Secret",
      metadata: { name, uid: `uid-${name}` },
      spec: {},
      status: {},
    }),
);

const kubeconfig = (secret: string) =>
  EnterpriseP.JoinClusterRequest_Kubernetes.create({
    type: {
      oneofKind: "kubeconfig",
      kubeconfig: { type: { oneofKind: "fromSecret", fromSecret: secret } },
    },
  });

const upgradeRegion = (
  name: string,
  state: EnterpriseP.ClusterConfig_Status_UpgradeRequest_State,
  message = "",
) =>
  EnterpriseP.ClusterConfig_Status_UpgradeRequest_Region.create({
    regionRef: { name, uid: `uid-${name}` },
    state,
    message,
    doneAt:
      state === EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.SUCCESS ||
      state === EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.FAILED
        ? ago(1)
        : undefined,
  });

const status = EnterpriseP.ClusterConfig_Status.create({
  licenseInfo: {
    state: EnterpriseP.ClusterConfig_Status_LicenseInfo_State.NONE,
  },
  totalSuccessfulUpgrades: 3,
  totalFailedUpgrades: 1,
  lastUpgradeRequests: [
    {
      createdAt: ago(300),
      doneAt: ago(280),
      state: EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.FAILED,
      request: { core: { version: "0.47.0" } },
      regions: [
        upgradeRegion(
          "default",
          EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.SUCCESS,
        ),
        upgradeRegion(
          "west",
          EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.FAILED,
          "core upgrade failed: genesis job failed: reason=BackoffLimitExceeded",
        ),
      ],
    },
  ],
  lastJoinRequests: [
    {
      createdAt: ago(90),
      doneAt: ago(80),
      state: EnterpriseP.ClusterConfig_Status_JoinRequest_State.FAILED,
      request: { name: "east", kubernetes: kubeconfig("east-kubeconfig") },
      regionRef: { name: "east", uid: "uid-east" },
      message:
        "Could not connect to the Kubernetes cluster of the Region: dial tcp 203.0.113.20:6443: connect: connection refused",
    },
    {
      createdAt: ago(2000),
      doneAt: ago(1980),
      state: EnterpriseP.ClusterConfig_Status_JoinRequest_State.SUCCESS,
      request: { name: "west", kubernetes: kubeconfig("west-kubeconfig") },
      regionRef: { name: "west", uid: "uid-west" },
    },
  ],
});

if (scenario === "joining") {
  status.joinRequest = EnterpriseP.ClusterConfig_Status_JoinRequest.create({
    createdAt: ago(3),
    state: EnterpriseP.ClusterConfig_Status_JoinRequest_State.JOINING,
    request: { name: "south", kubernetes: kubeconfig("south-token") },
  });
}

if (scenario === "upgrading") {
  status.upgradeRequest =
    EnterpriseP.ClusterConfig_Status_UpgradeRequest.create({
      createdAt: ago(4),
      state: EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.UPGRADING,
      request: { packageEnterprise: { version: "0.29.0" } },
      regions: [
        upgradeRegion(
          "default",
          EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.SUCCESS,
        ),
        upgradeRegion(
          "west",
          EnterpriseP.ClusterConfig_Status_UpgradeRequest_State.UPGRADING,
        ),
      ],
    });
}

const packageInfo = (current: string, latest: string) => ({
  currentVersion: current,
  latestVersion: latest,
  canUpgrade: current !== latest,
  setAt: ago(600),
});

const handlers: Record<string, (request: any) => unknown> = {
  getClusterConfig: () =>
    EnterpriseP.ClusterConfig.create({
      apiVersion: "enterprise/v1",
      kind: "ClusterConfig",
      metadata: { name: "default", uid: "uid-cc" },
      spec: {},
      status,
    }),
  getClusterInfo: () =>
    EnterpriseP.GetClusterInfoResponse.create({
      core: packageInfo("0.47.0", "0.47.0"),
      packageEnterprise: packageInfo("0.28.0", "0.29.0"),
      packageCordium: packageInfo("0.14.0", "0.14.0"),
    }),
  getLicense: () => EnterpriseP.GetLicenseResponse.create({}),
  listRegion: () => CoreP.RegionList.create({ items: regions }),
  listSecret: () => EnterpriseP.SecretList.create({ items: secrets }),
  getSecret: (request) =>
    secrets.find((item) => item.metadata?.name === request?.name),
  joinCluster: (request) => {
    joins.push(EnterpriseP.JoinClusterRequest.toJson(request));
    if (request?.name === "conflict") {
      throw new RpcError(
        "A Region join is already requested",
        "INVALID_ARGUMENT",
      );
    }
    return EnterpriseP.JoinClusterResponse.create({});
  },
};

for (const client of [
  clients.getClientEnterprise(),
  clients.getClientVisibilityEnterprise(),
  clients.getClientCluster(),
  clients.getClientCore(),
  clients.getClientVisibilityCore(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      calls.push({ method: method.localName, request });
      const handler = handlers[method.localName];
      if (handler) {
        const response = handler(request);
        if (!response) throw new RpcError("Not found", "NOT_FOUND");
        return { response };
      }
      return { response: method.O.create() };
    };
  }
}

const router = createMemoryRouter(
  [
    {
      path: "*",
      element: (
        <div
          className="mx-auto p-4"
          style={{ maxWidth: Number(params.get("width") ?? 1280) }}
        >
          <ClustermanMain />
        </div>
      ),
    },
  ],
  { initialEntries: ["/clusterman"] },
);

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <QueryClientProvider client={queryClient}>
          <Notifications />
          <React.Suspense fallback={null}>
            <RouterProvider router={router} />
          </React.Suspense>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
