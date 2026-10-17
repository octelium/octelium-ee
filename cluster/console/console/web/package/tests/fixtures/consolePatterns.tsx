import "@/index.css";
import "@mantine/core/styles.layer.css";
import { Policy as AccessPolicy } from "@/apis/accessv1/accessv1";
import {
  AccessLog,
  ComponentLog,
  Device,
  Policy,
  Session,
  User,
  UserList,
} from "@/apis/corev1/corev1";
import { AuditLog, AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { Stats } from "@/apis/visibilityv1/llm/vllmv1";
import { AccessLogList } from "@/components/AccessLogViewer";
import LogEditor from "@/components/AccessLogViewer/Editor";
import { NoLogFound } from "@/components/AccessLogViewer/utils";
import AuditLogList from "@/components/AuditLogViewer";
import { AuthenticationLogList } from "@/components/AuthenticationLogViewer";
import ComponentLogList from "@/components/ComponentLogViewer";
import { StatTile } from "@/components/Dashboard/components";
import { StatusBadge as HealthStatusBadge } from "@/components/Dashboard/health";
import { compact, exact } from "@/components/Dashboard/utils";
import { QueryState } from "@/components/LLMVisibility/Primitives";
import StatsOverview from "@/components/LLMVisibility/StatsOverview";
import {
  formatMetricValue,
  formatTokens,
} from "@/components/LLMVisibility/utils";
import LogPageShell from "@/components/LogViewer/LogPageShell";
import PageHeader from "@/components/PageHeader";
import { PageHeaderProvider } from "@/components/PageHeader/context";
import RefreshButton from "@/components/RefreshButton";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { StatusPill } from "@/components/ResourceLayout/ResourceStatus";
import SearchList from "@/components/SearchList";
import { SSHSessionViewer } from "@/components/SSHRecordingPlayer";
import { SummaryItemCount } from "@/components/Summary";
import { LabelComponent as AccessPolicyLabel } from "@/pages/access/Policy/List";
import { LabelComponent as DeviceLabel } from "@/pages/core/Device/List";
import { LabelComponent as PolicyLabel } from "@/pages/core/Policy/List";
import { LabelComponent as SessionLabel } from "@/pages/core/Session/List";
import { LabelComponent as UserLabel } from "@/pages/core/User/List";
import SSHReplay from "@/pages/visibility/ssh/Terminal";
import { ResourceComponentInfo } from "@/pages/utils/types";
import store from "@/store";
import { formatNumber } from "@/utils";
import * as clients from "@/utils/client";
import { formatCount } from "@/utils/format";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "counts";
const kind = params.get("kind") ?? "access";
const calls: { name: string; request?: any }[] = [];
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
const state: {
  calls: typeof calls;
  hold?: boolean;
  failure?: boolean;
  resolve?: () => void;
} = { calls, failure: params.get("error") === "true" };
(window as any).patterns = {
  state,
  queryClient,
  formatCount,
  compact,
  exact,
  formatTokens,
  formatMetricValue,
  formatNumber,
};

for (const client of [
  clients.getClientVisibilityCore(),
  clients.getClientVisibilityAccessLog(),
  clients.getClientVisibilityAuthenticationLog(),
  clients.getClientVisibilityAuditLog(),
  clients.getClientVisibilityComponentLog(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      calls.push({ name: method.localName, request });
      if (state.failure) throw new RpcError("Fixture failure", "INTERNAL");
      const response =
        method.localName === "listUser"
          ? UserList.create({
              apiVersion: "core/v1",
              kind: "UserList",
              items: [
                User.create({
                  apiVersion: "core/v1",
                  kind: "User",
                  metadata: { name: "alice", uid: "user-uid" },
                }),
              ],
            })
          : method.O.create();
      if (state.hold && method.localName === "listUser") {
        return new Promise((resolve) => {
          state.resolve = () => resolve({ response });
        });
      }
      return { response };
    };
  }
}

const info: ResourceComponentInfo = {
  API: "core",
  Kind: "User",
  List: {},
  Item: {},
  unCreatable: true,
};

const Resource = () => {
  useQuery({
    queryKey: ["unrelated"],
    queryFn: async () => {
      calls.push({ name: "unrelated" });
      return "Other";
    },
  });
  return (
    <PageHeaderProvider>
      <PageHeader />
      <ResourceListPage info={info} />
    </PageHeaderProvider>
  );
};

const Labels = () => {
  const state = Number(params.get("state") ?? "1");
  const disabled = params.get("disabled") !== "false";
  return (
    <>
      <div data-kind="session">
        <SessionLabel
          item={Session.create({
            spec: { state },
            status: { isConnected: true },
          })}
        />
      </div>
      <div data-kind="device">
        <DeviceLabel item={Device.create({ spec: { state }, status: {} })} />
      </div>
      <div data-kind="user">
        <UserLabel item={User.create({ spec: { isDisabled: disabled } })} />
      </div>
      <div data-kind="policy">
        <PolicyLabel item={Policy.create({ spec: { isDisabled: disabled } })} />
      </div>
      <div data-kind="access-policy">
        <AccessPolicyLabel
          item={AccessPolicy.create({ spec: { isDisabled: disabled } })}
        />
      </div>
      <div data-kind="detail">
        <StatusPill label="Active" tone="success" />
      </div>
      <div data-kind="health">
        <HealthStatusBadge status={99} />
      </div>
    </>
  );
};

const Counts = () => (
  <>
    <div data-kind="home">
      <StatTile
        label="Requests"
        value={48000}
        footer="Fixture"
        color="#64748b"
      />
    </div>
    <div data-kind="summary">
      <SummaryItemCount count={48000}>Requests</SummaryItemCount>
    </div>
    <div data-kind="llm">
      <StatsOverview
        stats={Stats.create({
          requests: { total: 48000 },
          tokens: { total: 48000 },
        })}
      />
    </div>
  </>
);

const Refresh = () => (
  <>
    <RefreshButton
      onClick={() => calls.push({ name: "refresh" })}
      loading={params.get("busy") === "true"}
      disabled={params.get("disabled") === "true"}
    />
    <input aria-label="Typing target" />
  </>
);

const Logs = () =>
  kind === "authentication" ? (
    <AuthenticationLogList />
  ) : kind === "audit" ? (
    <AuditLogList />
  ) : kind === "component" ? (
    <ComponentLogList />
  ) : kind === "ssh" ? (
    <SSHSessionViewer />
  ) : (
    <AccessLogList />
  );

const Fixture = () => {
  const location = useLocation();
  const navigate = useNavigate();
  (window as any).patterns.navigate = navigate;
  return (
    <>
      <output data-route>
        {location.pathname}
        {location.search}
      </output>
      {scenario === "labels" ? (
        <Labels />
      ) : scenario === "replay" ? (
        <Routes>
          <Route path="/visibility/ssh/:name" element={<SSHReplay />} />
        </Routes>
      ) : scenario === "json" ? (
        <LogEditor
          item={
            {
              access: AccessLog.create({ kind: "AccessLog" }),
              authentication: AuthenticationLog.create({
                kind: "AuthenticationLog",
              }),
              audit: AuditLog.create({ kind: "AuditLog" }),
              component: ComponentLog.create({ kind: "ComponentLog" }),
            }[kind]!
          }
        />
      ) : scenario === "resource" ? (
        <Resource />
      ) : scenario === "refresh" ? (
        <Refresh />
      ) : scenario === "logs" ? (
        <Logs />
      ) : scenario === "search" ? (
        <SearchList />
      ) : scenario === "log-search" ? (
        <LogPageShell title="Access logs">
          <p>Events</p>
        </LogPageShell>
      ) : scenario === "empty" ? (
        <>
          <NoLogFound />
          <QueryState isEmpty isError={false} isLoading={false}>
            <p>Loaded</p>
          </QueryState>
        </>
      ) : (
        <Counts />
      )}
      <input aria-label="Unrelated input" />
    </>
  );
};

createRoot(document.getElementById("root")!).render(
  <Provider store={store}>
    <MantineProvider
      theme={themeMantine}
      cssVariablesResolver={cssVariablesResolver}
      forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
    >
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[params.get("route") ?? "/core/users"]}>
          <Fixture />
        </MemoryRouter>
      </QueryClientProvider>
    </MantineProvider>
  </Provider>,
);
