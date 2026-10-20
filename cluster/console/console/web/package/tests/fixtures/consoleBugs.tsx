import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@mantine/dates/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider, Select } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { Provider } from "react-redux";
import { User } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetStatusResponse } from "@/apis/userv1/userv1";
import {
  GetAccessLogSummaryResponse,
  ListAccessLogTopDenyReasonResponse,
  ListComponentLogTopComponentResponse,
  GetClusterHealthResponse,
  GetClusterHealthResponse_Status,
  GetClusterHealthResponse_Subsystem_Type,
  GetClusterSummaryRequest_Kind,
  GetClusterSummaryResponse,
  GetAccessLogDataPointResponse,
  GetComponentLogSummaryResponse,
  GetAuditLogSummaryResponse,
  ListAccessLogResponse,
  ListAuditLogResponse,
  ListAuthenticationLogResponse,
  ListComponentLogResponse,
  ListSSHSessionResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { AppErrorBoundary } from "@/components/ErrorBoundary";
import { StatusBadge, SubsystemCard } from "@/components/Dashboard/health";
import { StatTile, Trend } from "@/components/Dashboard/components";
import { useClusterSummary } from "@/components/Dashboard/queries";
import { AccessLogList as AccessLogs } from "@/components/AccessLogViewer";
import { AuthenticationLogList as AuthenticationLogs } from "@/components/AuthenticationLogViewer";
import AuditLogs from "@/components/AuditLogViewer";
import ComponentLogs from "@/components/ComponentLogViewer";
import { SSHSessionViewer } from "@/components/SSHRecordingPlayer";
import Sidebar from "@/components/SideBar";
import TopBar from "@/components/TopBar";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import Attention from "@/pages/Home/Attention";
import AccessAttention from "@/pages/access/Overview/Attention";
import EnterpriseHealth from "@/pages/enterprise/Overview/Health";
import VisibilityBreakdown from "@/pages/visibility/Main/Breakdown";
import VisibilityAccessLeaders from "@/pages/visibility/Main/AccessLeaders";
import VisibilityComponents from "@/pages/visibility/Main/Components";
import ComponentLogsPage from "@/pages/visibility/componentlogs";
import AccessLogsPage from "@/pages/visibility/accesslogs";
import StatsOverview from "@/components/LLMVisibility/StatsOverview";
import { CHART_INK, STATUS_COLORS } from "@/utils/charts/palette";
import StatBars from "@/components/LLMVisibility/StatBars";
import { metricAccessor } from "@/components/LLMVisibility/utils";
import { Metric } from "@/apis/visibilityv1/llm/vllmv1";
import LineChart from "@/components/Charts/LineChart";
import SeriesChart from "@/components/Charts/SeriesChart";
import MetricChart, { gaugeOp } from "@/components/Charts/MetricChart";
import { GaugeOperation_Function } from "@/apis/visibilityv1/metrics/vmetricsv1";
import * as echarts from "echarts";
import { setStatus } from "@/features/settings/slice";
import store from "@/store";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityCluster,
  getClientVisibilityComponentLog,
  getClientVisibilityMetrics,
} from "@/utils/client";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "shared";
const calls: { name: string; request: any }[] = [];
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
if (scenario === "overview") {
  queryClient.setQueryDefaults(["core.getUser"], { staleTime: Infinity });
  queryClient.setQueryData(["core.getUser", "alice"], {
    response: User.create({
      apiVersion: "core/v1",
      kind: "User",
      metadata: { name: "alice", uid: "user-uid" },
      spec: { email: "alice@example.test" },
    }),
  });
}
(window as any).calls = calls;
(Object.getPrototypeOf(getClientVisibilityAuditLog()) as any).getAuditLogSummary =
  async () => ({ response: GetAuditLogSummaryResponse.create() });
const points = [
  { ts: Timestamp.fromDate(new Date("2026-10-12T13:14:00Z")), value: 1 },
  { ts: Timestamp.fromDate(new Date("2026-10-12T14:14:00Z")), value: 2 },
];
(window as any).chartLabels = () =>
  Array.from(document.querySelectorAll(".echarts-for-react")).map((element) => {
    const axis = (
      echarts.getInstanceByDom(element as HTMLElement)?.getOption() as any
    )?.xAxis?.[0];
    return axis?.axisLabel.formatter(Date.parse("2026-10-12T13:14:00Z"));
  });
(Object.getPrototypeOf(getClientVisibilityMetrics()) as any).queryMetrics =
  async () => ({
    response: {
      series: [
        {
          labels: [],
          points: {
            oneofKind: "number",
            number: {
              points: points.map((point) => ({
                timestamp: point.ts,
                value: { oneofKind: "asDouble", asDouble: point.value },
              })),
            },
          },
        },
      ],
    },
  });

const cluster = Object.getPrototypeOf(getClientVisibilityCluster());
cluster.getClusterSummary = async (request: any) => {
  calls.push({ name: "summary", request });
  if (scenario === "access-error") throw new Error("Summary unavailable");
  return {
    response: GetClusterSummaryResponse.create({
      unavailables:
        scenario === "partial"
          ? [{ kind: GetClusterSummaryRequest_Kind.CORE_DEVICE, message: "unavailable" }]
          : [],
      access: {
        request: {
          totalDeadlinePassed:
            scenario === "deadline" || scenario === "decided-deadline" ? 5 : 0,
          totalPendingDeadlinePassed: scenario === "deadline" ? 3 : 0,
        },
        review: { totalPending: scenario === "decided-deadline" ? 2 : 0 },
      },
    }),
  };
};
cluster.getClusterHealth = async () => {
  if (scenario === "health-error") throw new Error("Health unavailable");
  if (scenario === "enterprise-health") {
    return {
      response: GetClusterHealthResponse.create({
        status: GetClusterHealthResponse_Status.CRITICAL,
        subsystems: [
          {
            type: GetClusterHealthResponse_Subsystem_Type.CERTIFICATES,
            status: GetClusterHealthResponse_Status.CRITICAL,
            message: "1 expired",
          },
          {
            type: GetClusterHealthResponse_Subsystem_Type.SECRET_STORES,
            status: GetClusterHealthResponse_Status.UNKNOWN,
            message: "Not evaluated",
          },
        ],
      }),
    };
  }
  const status =
    scenario === "critical"
      ? GetClusterHealthResponse_Status.CRITICAL
      : scenario === "unknown"
        ? GetClusterHealthResponse_Status.UNKNOWN
        : GetClusterHealthResponse_Status.OK;
  return {
    response: GetClusterHealthResponse.create({
      status,
      subsystems: [
        {
          type: GetClusterHealthResponse_Subsystem_Type.AUTHORIZATION,
          status,
          message: "Authorization signal",
        },
      ],
    }),
  };
};
(
  Object.getPrototypeOf(getClientVisibilityComponentLog()) as any
).getComponentLogSummary = async () => {
  if (scenario === "source-error") throw new Error("Logs unavailable");
  return { response: GetComponentLogSummaryResponse.create() };
};

const bucket = (minute: number) =>
  Timestamp.fromDate(new Date(Date.UTC(2026, 9, 19, 11, minute)));
(
  Object.getPrototypeOf(getClientVisibilityAccessLog()) as any
).getAccessLogDataPoint = async (request: any) => {
  calls.push({ name: "getAccessLogDataPoint", request });
  if (scenario !== "breakdown-other") {
    return { response: GetAccessLogDataPointResponse.create() };
  }
  return {
    response: GetAccessLogDataPointResponse.create({
      datapoints: [
        { timestamp: bucket(0), count: 10 },
        { timestamp: bucket(5), count: 20 },
      ],
      series: [
        {
          key: "ALLOWED",
          total: 9,
          datapoints: [
            { timestamp: bucket(0), count: 4 },
            { timestamp: bucket(5), count: 5 },
          ],
        },
        {
          key: "DENIED",
          total: 6,
          datapoints: [
            { timestamp: bucket(0), count: 2 },
            { timestamp: bucket(5), count: 4 },
          ],
        },
      ],
    }),
  };
};
(window as any).statusColors = {
  good: STATUS_COLORS.good,
  critical: STATUS_COLORS.critical,
  muted: CHART_INK.muted,
};
(
  Object.getPrototypeOf(getClientVisibilityAccessLog()) as any
).getAccessLogSummary = async (request: any) => {
  calls.push({ name: "getAccessLogSummary", request });
  return { response: GetAccessLogSummaryResponse.create({ totalNumber: 4 }) };
};
(
  Object.getPrototypeOf(getClientVisibilityAccessLog()) as any
).listAccessLogTopDenyReason = async () => ({
  response: ListAccessLogTopDenyReasonResponse.create({
    items: [
      {
        reason: 6,
        count: 4,
        policyRef: { name: "deny-all", uid: "p-1" },
      },
      { reason: 13, count: 1 },
    ],
    totalCount: 2,
  }),
});
(
  Object.getPrototypeOf(getClientVisibilityComponentLog()) as any
).listComponentLogTopComponent = async (request: any) => {
  calls.push({ name: "listComponentLogTopComponent", request });
  return {
    response: ListComponentLogTopComponentResponse.create({
      items: [
        {
          component: { namespace: "octelium", type: "vigil" },
          count: 3,
          countFatal: 1,
        },
      ],
      totalCount: 1,
    }),
  };
};

for (const [client, method, response] of [
  [getClientVisibilityAccessLog(), "listAccessLog", ListAccessLogResponse],
  [getClientVisibilityAccessLog(), "listSSHSession", ListSSHSessionResponse],
  [
    getClientVisibilityAuthenticationLog(),
    "listAuthenticationLog",
    ListAuthenticationLogResponse,
  ],
  [getClientVisibilityAuditLog(), "listAuditLog", ListAuditLogResponse],
  [
    getClientVisibilityComponentLog(),
    "listComponentLog",
    ListComponentLogResponse,
  ],
] as const) {
  (Object.getPrototypeOf(client) as any)[method] = async (request: any) => {
    calls.push({ name: method, request });
    return {
      response: response.create({
        items: [],
        listResponseMeta: { totalCount: 0 },
      }),
    };
  };
}

store.dispatch(
  setStatus({
    status: GetStatusResponse.create({
      user: {
        metadata: {
          displayName: params.get("displayName") ?? "",
          name: scenario === "avatar-empty" ? "" : "alice",
        },
        spec: params.has("email")
          ? { email: params.get("email") ?? "" }
          : undefined,
      },
    }),
  }),
);

if (scenario === "shadow") {
  const forbidden = document.createElement("div");
  Object.defineProperty(forbidden, "shadowRoot", {
    get() {
      throw new Error("Permission denied to access property shadowRoot");
    },
  });
  document.body.append(forbidden);
}

const Location = () => <output data-location>{useLocation().pathname}</output>;

const Search = () => <output data-search>{useLocation().search}</output>;

const Query = () => {
  const query = useClusterSummary("range", 60, 0);
  return <button onClick={() => query.refetch()}>Refetch window</button>;
};

const Fixture = () => (
  <Provider store={store}>
    <MantineProvider>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[params.get("path") ?? "/"]}>
          <AppErrorBoundary>
            {scenario.startsWith("sidebar") ? (
              <>
                <Sidebar />
                <Location />
              </>
            ) : scenario.startsWith("avatar") ? (
              <TopBar />
            ) : scenario === "query" ? (
              <Query />
            ) : scenario === "overview" ? (
              <Routes>
                <Route
                  path="/core/users/:name"
                  element={<ResourceItemMainPage unDeletable />}
                />
              </Routes>
            ) : scenario === "logs" ? (
              <>
                <AccessLogs periodMinutes={60} />
                <AuthenticationLogs periodMinutes={60} />
                <AuditLogs />
                <ComponentLogs />
                <SSHSessionViewer />
              </>
            ) : scenario === "charts" ? (
              <>
                <LineChart points={points} />
                <SeriesChart series={[{ name: "Requests", points }]} />
                <MetricChart
                  metric="test"
                  operation={gaugeOp(GaugeOperation_Function.AVG)}
                  from={points[0].ts}
                  to={points[1].ts}
                  autoRefresh={false}
                />
              </>
            ) : scenario === "shared" || scenario === "shadow" ? (
              <>
                <button data-control className="text-xs font-normal">
                  Refresh
                </button>
                <input data-control className="text-xs font-normal" />
                <StatusBadge
                  status={GetClusterHealthResponse_Status.CRITICAL}
                />
                <Select label="Options" data={["One", "Two"]} searchable />
                <Trend cur={81} prev={0} upIsGood rangeLabel="1h" />
                <Trend cur={0} prev={0} upIsGood rangeLabel="1h" />
                <SubsystemCard
                  subsystem={{
                    type: GetClusterHealthResponse_Subsystem_Type.COMPONENTS,
                    status: GetClusterHealthResponse_Status.UNKNOWN,
                    message:
                      "Could not be evaluated: rpc error: code = Unimplemented desc = unknown service",
                    total: 0,
                    unhealthy: 0,
                  }}
                />
                <StatTile
                  label="Approved · 24h"
                  value={81}
                  footer="81 new sessions in the last 24 hours, including all available client types"
                  color="#000"
                  bar={{ value: 81, total: 81, label: "sessions" }}
                />
              </>
            ) : scenario === "enterprise-health" ? (
              <EnterpriseHealth periodMinutes={60} />
            ) : scenario === "access-error" ? (
              <AccessAttention periodMinutes={60} />
            ) : scenario === "breakdown" || scenario === "breakdown-other" ? (
              <VisibilityBreakdown periodMinutes={60} />
            ) : scenario === "componentlogs" ? (
              <>
                <ComponentLogsPage />
                <Search />
              </>
            ) : scenario === "accesslogs" ? (
              <AccessLogsPage />
            ) : scenario === "access-leaders" ? (
              <VisibilityAccessLeaders periodMinutes={60} />
            ) : scenario === "components" ? (
              <VisibilityComponents periodMinutes={60} />
            ) : scenario === "llm-stats" ? (
              <StatsOverview
                rangeLabel="24h"
                stats={
                  {
                    requests: { total: 120, denied: 0, failed: 3 },
                    tokens: { total: 1000 },
                    latency: { p95Ms: 400 },
                  } as any
                }
                previous={
                  {
                    requests: { total: 100, denied: 0, failed: 6 },
                    tokens: { total: 1000 },
                    latency: { p95Ms: 200 },
                  } as any
                }
              />
            ) : scenario === "statbars" ? (
              <StatBars
                accessor={metricAccessor(Metric.LATENCY_P95)}
                items={[
                  {
                    key: "fast",
                    stats: { latency: { p95Ms: 200 } } as any,
                  },
                  {
                    key: "slow",
                    stats: { latency: { p95Ms: 800 } } as any,
                  },
                ]}
              />
            ) : (
              <Attention periodMinutes={60} />
            )}
          </AppErrorBoundary>
        </MemoryRouter>
      </QueryClientProvider>
    </MantineProvider>
  </Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Fixture />);
