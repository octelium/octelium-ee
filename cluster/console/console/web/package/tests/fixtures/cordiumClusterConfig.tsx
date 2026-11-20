import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@mantine/notifications/styles.layer.css";
import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { Struct } from "@/apis/google/protobuf/struct";
import cordiumRouter from "@/pages/cordium/router";
import store from "@/store";
import { queryClient } from "@/utils";
import * as clients from "@/utils/client";
import { getResourcePB } from "@/utils/pb";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { JsonValue } from "@protobuf-ts/runtime";
import { QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Link, Outlet, useLocation } from "react-router";
import { RouterProvider } from "react-router/dom";

declare global {
  interface Window {
    cordiumConfig: {
      calls: { service: string; method: string; request: JsonValue }[];
      saved?: JsonValue;
      resolve?: () => void;
      get: () => Promise<unknown>;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario");
const calls: Window["cordiumConfig"]["calls"] = [];
window.cordiumConfig = {
  calls,
  get: async () => {
    const { response } = await getResourcePB("cordium", "ClusterConfig", {});
    return CordiumP.ClusterConfig.toJson(response as CordiumP.ClusterConfig);
  },
};

const matchAny = { type: { oneofKind: "matchAny" as const, matchAny: true } };
const limit = {
  cpu: { millicores: 2000 },
  memory: { megabytes: 4096 },
  storage: { megabytes: 20000 },
};

let config = CordiumP.ClusterConfig.create({
  apiVersion: "cordium/v1",
  kind: "ClusterConfig",
  metadata: { name: "cluster", uid: "cordium-cluster-config", resourceVersion: "1" },
  spec: {
    space: { ownership: { rules: [{ effect: CordiumP.ClusterConfig_Spec_Space_Ownership_Rule_Effect.ALLOW, condition: matchAny }] } },
    workspace: {
      storage: {
        storageClass: {
          rules: [
            { storageClass: "fast", condition: { type: { oneofKind: "match", match: "workspace.metadata.name == 'build'" } } },
            { storageClass: "standard", condition: matchAny },
          ],
        },
        volumeSnapshotClass: { rules: [{ volumeSnapshotClass: "snapshots", condition: matchAny }] },
      },
      limit: {
        maxPerUser: 10,
        maxActivePerUser: 3,
        maxSnapshotsPerUser: 5,
        buildLimit: limit,
        defaultUserSpaceLimit: { cpu: { millicores: 1000 } },
        defaultOrganizationSpaceLimit: limit,
        maxLimit: { cpu: { millicores: 8000 }, memory: { megabytes: 16384 } },
      },
      timeout: {
        defaultDuration: { type: { oneofKind: "minutes", minutes: 30 } },
        userSpaceDuration: { type: { oneofKind: "hours", hours: 1 } },
        organizationSpaceDuration: { type: { oneofKind: "hours", hours: 2 } },
        maxActiveDuration: { type: { oneofKind: "hours", hours: 24 } },
      },
      runtime: { capabilities: { add: ["NET_ADMIN"], drop: ["ALL"] } },
    },
    volume: {
      storage: { storageClass: { rules: [{ storageClass: "shared", condition: matchAny }] } },
      limit: { maxPerSpace: 8, maxMountsPerWorkspace: 4, maxSize: { megabytes: 100000 }, defaultSize: { megabytes: 10000 } },
    },
    agent: {
      version: "0.1.0",
      llm: { service: "llm.default", model: "default-model" },
      image: { type: { oneofKind: "registry", registry: { url: "ghcr.io/example/agent:latest" } } },
      limit,
      config: Struct.fromJson({ theme: "system", tools: { terminal: true } }),
    },
  },
  status: {},
});

if (scenario === "empty") config.spec = undefined;
let getCount = 0;

for (const client of [clients.getClientCordiumManagement(), clients.getClientCordium(), clients.getClientVisibilityCordium()]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: CordiumP.ClusterConfig) => {
      calls.push({ service: client.typeName, method: method.localName, request: method.I.toJson(request) });
      if (method.localName === "getClusterConfig") {
        getCount++;
        if (scenario === "loading") await new Promise<void>((resolve) => { window.cordiumConfig.resolve = resolve; });
        if (scenario === "load-error" && getCount === 1) throw new RpcError("Configuration access denied", "PERMISSION_DENIED");
        if (scenario === "unavailable") return { response: undefined };
        return { response: CordiumP.ClusterConfig.clone(config) };
      }
      if (method.localName === "updateClusterConfig") {
        if (scenario === "save-error") throw new RpcError("Update rejected", "INVALID_ARGUMENT");
        config = CordiumP.ClusterConfig.clone(request);
        config.metadata!.resourceVersion = String(Number(config.metadata!.resourceVersion) + 1);
        window.cordiumConfig.saved = CordiumP.ClusterConfig.toJson(config);
        return { response: CordiumP.ClusterConfig.clone(config) };
      }
      return { response: method.O.create() };
    };
  }
}

const Layout = () => {
  const location = useLocation();
  return (
    <main className="mx-auto min-w-0 max-w-6xl p-3 sm:p-6">
      <Link to="/elsewhere">Leave configuration</Link>
      <output data-testid="location" className="sr-only">{location.pathname}</output>
      <Outlet />
    </main>
  );
};

const router = createMemoryRouter([
  { path: "/", element: <Layout />, children: [cordiumRouter(), { path: "elsewhere", element: <span>Another page</span> }] },
], { initialEntries: ["/cordium/clusterconfig"] });

queryClient.setDefaultOptions({ queries: { retry: false } });

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider theme={themeMantine} cssVariablesResolver={cssVariablesResolver} forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}>
        <Notifications />
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
