import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetDeviceManagerSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import SelectDeviceManagers from "@/pages/enterprise/ClusterConfig/SelectDeviceManagers";
import DeviceManagerEdit from "@/pages/enterprise/DeviceManager/Edit";
import {
  LabelComponent,
  Summary,
} from "@/pages/enterprise/DeviceManager/List";
import { ManagerTile, PROVIDERS } from "@/pages/enterprise/DeviceManager/providers";
import managerInfo from "@/pages/enterprise/DeviceManager/router";
import store from "@/store";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";

declare global {
  interface Window {
    manager: {
      calls: { method: string; request: any }[];
      value?: any;
      order?: string[];
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: { method: string; request: any }[] = [];
window.manager = { calls };

const secret = (fromSecret: string) => ({
  type: { oneofKind: "fromSecret" as const, fromSecret },
});

const manager = (
  name: string,
  type: EnterpriseP.DeviceManager_Spec["type"],
  spec: Partial<EnterpriseP.DeviceManager_Spec> = {},
  status: Partial<EnterpriseP.DeviceManager_Status> = {},
) =>
  EnterpriseP.DeviceManager.create({
    apiVersion: "enterprise/v1",
    kind: "DeviceManager",
    metadata: { name, uid: `uid-${name}`, resourceVersion: "1" },
    spec: { ...spec, type },
    status,
  });

const managers: Record<string, EnterpriseP.DeviceManager> = {
  crowdstrike: manager(
    "crowdstrike",
    {
      oneofKind: "crowdStrike",
      crowdStrike: EnterpriseP.DeviceManager_Spec_CrowdStrike.create({
        clientID: "0123456789abcdef",
        clientSecret: secret("falcon-client-secret"),
        hostFilter: "platform_name:'Windows'",
      }),
    },
    {},
    {
      state: EnterpriseP.DeviceManager_Status_State.OK,
      collection: {
        lastAttemptAt: Timestamp.now(),
        lastSuccessAt: Timestamp.now(),
        managedDevices: 1240,
        lastError: "",
      },
      linking: {
        lastSweepAt: Timestamp.now(),
        linkedDevices: 1180,
        ambiguous: 2,
        conflicts: 1,
        suspended: 0,
        failedUpdates: 0,
        ownerMismatches: 0,
      },
    },
  ),
  intune: manager(
    "intune",
    {
      oneofKind: "microsoftIntune",
      microsoftIntune: EnterpriseP.DeviceManager_Spec_MicrosoftIntune.create({
        tenantID: "0b3f1e7a-24c4-4a42-8e0e-7c1c2f9e5d21",
        clientID: "6f1d2b9c-7a51-4c4e-9a0f-2d8f2f4f8b11",
        clientSecret: secret("intune-client-secret"),
      }),
    },
    {
      linking: {
        requireOwnerMatch: true,
        verificationInterval: {
          type: { oneofKind: "hours", hours: 12 },
        },
      },
    },
  ),
  sentinelone: manager("sentinelone", {
    oneofKind: "sentinelOne",
    sentinelOne: EnterpriseP.DeviceManager_Spec_SentinelOne.create({
      managementURL: "https://example.sentinelone.net",
      apiToken: secret("sentinelone-token"),
      agentFilters: { isDecommissioned: "false" },
    }),
  }),
  fleet: manager("fleet", {
    oneofKind: "fleetDM",
    fleetDM: EnterpriseP.DeviceManager_Spec_FleetDM.create({
      baseURL: "https://fleet.example.com",
      apiToken: secret("fleet-token"),
    }),
  }),
  jamf: manager(
    "jamf",
    {
      oneofKind: "jamf",
      jamf: EnterpriseP.DeviceManager_Spec_Jamf.create({
        baseURL: "https://example.jamfcloud.com",
        clientID: "0f6a1c34-9d7e-4b1a-8a65-3a6d2f0c9e77",
        clientSecret: secret("jamf-client-secret"),
      }),
    },
    { polling: { isDisabled: false, interval: { type: { oneofKind: "hours", hours: 2 } } } },
  ),
  huntress: manager("huntress", {
    oneofKind: "huntress",
    huntress: EnterpriseP.DeviceManager_Spec_Huntress.create({
      apiKey: "hk_0123456789",
      apiSecret: secret("huntress-secret"),
      organizationIDs: [1042],
    }),
  }),
};

const secrets = [
  "falcon-client-secret",
  "intune-client-secret",
  "sentinelone-token",
  "fleet-token",
  "huntress-secret",
  "jamf-client-secret",
].map((name) =>
  EnterpriseP.Secret.create({
    apiVersion: "enterprise/v1",
    kind: "Secret",
    metadata: { name, uid: `uid-${name}` },
    spec: {},
    status: {},
  }),
);

const summary = GetDeviceManagerSummaryResponse.create({
  totalNumber: 3,
  totalCrowdStrike: 1,
  totalMicrosoftIntune: 1,
  totalIru: 1,
  totalOK: 1,
  totalDegraded: 1,
  totalError: 1,
  totalPollingDisabled: 1,
  totalManagedDevices: 1500,
  totalLinkedDevices: 1300,
  totalAmbiguous: 2,
  totalConflicts: 3,
  totalSuspended: 4,
  totalOwnerMismatches: 5,
  totalFailedUpdates: 6,
});

const handlers: Record<string, (request: any) => unknown> = {
  getDeviceManager: (request) => managers[request?.name],
  getDeviceManagerSummary: () => summary,
  listSecret: () => EnterpriseP.SecretList.create({ items: secrets }),
  getSecret: (request) =>
    secrets.find((item) => item.metadata?.name === request?.name),
  listDeviceManager: () =>
    EnterpriseP.DeviceManagerList.create({ items: Object.values(managers) }),
  createDeviceManager: (request) => request,
  updateDeviceManager: (request) => request,
};

for (const client of [
  clients.getClientEnterprise(),
  clients.getClientVisibilityEnterprise(),
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

const EditOnly = () => {
  const name = params.get("name");
  const item = React.useMemo(
    () =>
      name && managers[name]
        ? managers[name]
        : managerInfo.Item.createResource!(),
    [name],
  );
  return (
    <div
      className="mx-auto p-6"
      style={{ maxWidth: Number(params.get("width") ?? 896) }}
    >
      <DeviceManagerEdit
        item={item}
        onUpdate={(next: EnterpriseP.DeviceManager) => {
          window.manager.value = EnterpriseP.DeviceManager.toJson(next);
        }}
      />
    </div>
  );
};

const ListOnly = () => (
  <div className="mx-auto flex max-w-5xl flex-col gap-4 p-6">
    <Summary />
    {Object.values(managers).map((item) => (
      <div key={item.metadata?.name} data-testid={item.metadata?.name}>
        <LabelComponent item={item} />
      </div>
    ))}
    <div data-testid="tiles" className="flex flex-wrap gap-2">
      {PROVIDERS.map((provider) => (
        <ManagerTile key={provider.kind} kind={provider.kind} />
      ))}
    </div>
  </div>
);

const OrderOnly = () => {
  const [order, setOrder] = React.useState(["intune", "crowdstrike", "fleet"]);
  window.manager.order = order;
  return (
    <div className="mx-auto max-w-2xl p-6">
      <SelectDeviceManagers value={order} onChange={setOrder} />
    </div>
  );
};

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Outlet />,
      children: [
        { path: "edit-only", element: <EditOnly /> },
        { path: "list-only", element: <ListOnly /> },
        { path: "order-only", element: <OrderOnly /> },
        {
          path: "enterprise/devicemanagers",
          element: <Outlet />,
          children: [
            {
              path: "create",
              element: (
                <div className="mx-auto max-w-5xl p-6">
                  <ResourceCreatePage
                    specComponent={managerInfo.Item.Edit!}
                    createResource={managerInfo.Item.createResource}
                  />
                </div>
              ),
            },
            {
              path: ":name",
              element: <ResourceItemDrawer />,
              children: [
                {
                  path: "",
                  element: (
                    <ResourceItemMainPage
                      infoComponent={managerInfo.infoItemsGetter}
                    />
                  ),
                },
                {
                  path: "edit",
                  element: (
                    <ResourceEditPage specComponent={managerInfo.Item.Edit!} />
                  ),
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  { initialEntries: [params.get("path") ?? "/edit-only"] },
);

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <QueryClientProvider client={queryClient}>
          <React.Suspense fallback={null}>
            <RouterProvider router={router} />
          </React.Suspense>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
