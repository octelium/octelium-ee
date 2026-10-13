import "@/index.css";
import "@mantine/core/styles.layer.css";
import { User, UserList } from "@/apis/corev1/corev1";
import {
  ClusterConfig,
  GetClusterInfoResponse,
  GetLicenseResponse,
} from "@/apis/enterprisev1/enterprisev1";
import { DashboardHeader } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  DashboardVisibilityContext,
  useDashboardQuery,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import { useResourcePicker } from "@/components/ResourceLayout/resourcePicker/useResourcePicker";
import ClusterCard from "@/pages/Home/ClusterCard";
import ClusterMain from "@/pages/clusterman/Main";
import { queryClient } from "@/utils";
import * as clients from "@/utils/client";
import { onQueryError, queryRetryDelay, retryQuery } from "@/utils/queryPolicy";
import { rpcErrorCode } from "@/utils/rpcError";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import {
  QueryClientProvider,
  focusManager,
  useQuery,
} from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "utilities";
const calls: { name: string; aborted?: boolean }[] = [];
const record = (name: string) => calls.push({ name });
const resolves = new Map<string, () => void>();
(window as any).network = {
  calls,
  clients,
  queryClient,
  retryQuery,
  queryRetryDelay,
  rpcErrorCode,
  onQueryError,
  setFocused: (focused: boolean) => focusManager.setFocused(focused),
  resolvePicker: (query: string) => resolves.get(query)?.(),
};

Object.getPrototypeOf(clients.getClientCluster()).getClusterInfo = async () => {
  record("clusterInfo");
  return { response: GetClusterInfoResponse.create() };
};
Object.getPrototypeOf(clients.getClientCluster()).getLicense = async () => {
  record("license");
  return { response: GetLicenseResponse.create() };
};
Object.getPrototypeOf(clients.getClientEnterprise()).getClusterConfig =
  async () => {
    record("clusterConfig");
    return { response: ClusterConfig.create({ status: {} }) };
  };
Object.getPrototypeOf(clients.getClientVisibilityCore()).listUser = async (
  request: any,
  options: any,
) => {
  const query = request.common?.query ?? "";
  const call = { name: `picker:${query}`, aborted: false };
  calls.push(call);
  const response = () => ({
    response: UserList.create({
      items: [
        User.create({
          apiVersion: "core/v1",
          kind: "User",
          metadata: { name: query || "alice" },
        }),
      ],
    }),
  });
  if (!query) return response();
  return new Promise((resolve, reject) => {
    const abort = () => {
      call.aborted = true;
      resolves.delete(query);
      reject(new RpcError("Cancelled", "CANCELLED"));
    };
    options?.abort?.addEventListener("abort", abort, { once: true });
    resolves.set(query, () => {
      options?.abort?.removeEventListener("abort", abort);
      resolve(response());
    });
  });
};

const Query = () => {
  const fetch = async () => {
    record("query");
    throw new RpcError("Fixture failure", params.get("code") ?? "UNAVAILABLE");
  };
  const query = useQuery({ queryKey: ["network-query"], queryFn: fetch });
  return <output>{query.status}</output>;
};

const DashboardQuery = (props: { name?: string; enabled?: boolean }) => {
  const name = props.name ?? "dashboard";
  const query = useDashboardQuery({
    queryKey: [name],
    enabled: props.enabled,
    fetch: async () => {
      record(name);
      if (scenario === "dashboard-retry")
        throw new RpcError(
          "Fixture failure",
          params.get("code") ?? "UNAVAILABLE",
        );
      return name;
    },
  });
  return (
    <output>
      {query.status}:{query.data}
    </output>
  );
};

const Unrelated = () => {
  useQuery({
    queryKey: ["user", "status"],
    queryFn: async () => {
      record("status");
      return "status";
    },
  });
  useQuery({
    queryKey: ["other"],
    queryFn: async () => {
      record("other");
      return "other";
    },
  });
  return null;
};

const Header = (props: {
  periodMinutes?: number;
  onPeriodChange?: (period: number) => void;
}) => (
  <DashboardHeader
    title="Network dashboard"
    description="Fixture"
    periodMinutes={props.periodMinutes ?? 60}
    onPeriodChange={props.onPeriodChange ?? (() => {})}
    autoRefresh={false}
    onAutoRefreshChange={() => {}}
    shortcuts={[]}
    shortcutsLabel="Shortcuts"
  />
);

const Dashboard = () => {
  const [visible, setVisible] = React.useState(false);
  return (
    <>
      <Header />
      <DashboardQuery />
      <DashboardVisibilityContext.Provider value={visible}>
        <DashboardQuery name="hidden-dashboard" />
      </DashboardVisibilityContext.Provider>
      <ClusterCard />
      <Unrelated />
      <button onClick={() => setVisible(true)}>Show panel</button>
    </>
  );
};

const Focus = () => {
  useQuery({
    queryKey: ["focus-static"],
    queryFn: async () => {
      record("static");
      return "static";
    },
  });
  return (
    <>
      <DashboardQuery />
      <DashboardVisibilityContext.Provider value={false}>
        <DashboardQuery />
      </DashboardVisibilityContext.Provider>
    </>
  );
};

const Range = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const location = useLocation();
  const navigate = useNavigate();
  const query = useDashboardQuery({
    queryKey: ["range", periodMinutes],
    fetch: async () => {
      record(`range:${periodMinutes}`);
      return periodMinutes;
    },
  });
  (window as any).network.setPeriod = setPeriodMinutes;
  (window as any).network.navigate = navigate;
  return (
    <>
      <Header periodMinutes={periodMinutes} onPeriodChange={setPeriodMinutes} />
      <output data-route>
        {location.pathname}
        {location.search}
      </output>
      <output data-period>{query.data}</output>
    </>
  );
};

const RangeFixture = () => {
  const [show, setShow] = React.useState(true);
  return (
    <>
      <button onClick={() => setShow(false)}>Unmount range</button>
      {show && <Range />}
    </>
  );
};

const Picker = () => {
  const picker = useResourcePicker({
    api: "core",
    kind: "User",
    onCreated: () => {},
  });
  return (
    <>
      <input
        aria-label="Search picker"
        value={picker.search}
        onChange={(event) => picker.setSearch(event.target.value)}
      />
      <output>
        {picker.items.map((item) => item.metadata!.name).join(",")}
      </output>
    </>
  );
};

if (scenario === "dashboard") {
  queryClient.setQueryDefaults(["old-dashboard"], {
    meta: { dashboard: true },
  });
  queryClient.setQueryData(["old-dashboard"], "cached");
  queryClient.setQueryData(["hidden-dashboard"], "cached");
}

const content =
  scenario === "query" ? (
    <Query />
  ) : scenario === "dashboard-retry" ? (
    <DashboardQuery />
  ) : scenario === "dashboard" ? (
    <Dashboard />
  ) : scenario === "cluster" ? (
    <>
      <ClusterMain />
      <Unrelated />
    </>
  ) : scenario === "focus" ? (
    <Focus />
  ) : scenario === "range" ? (
    <RangeFixture />
  ) : scenario === "picker" ? (
    <Picker />
  ) : (
    <div>Network fixture</div>
  );

createRoot(document.getElementById("root")!).render(
  <MantineProvider>
    <QueryClientProvider client={queryClient}>
      <AutoRefreshContext.Provider value={scenario === "focus"}>
        <MemoryRouter initialEntries={["/dashboard?keep=yes"]}>
          {content}
        </MemoryRouter>
      </AutoRefreshContext.Provider>
    </QueryClientProvider>
  </MantineProvider>,
);
