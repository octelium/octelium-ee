import "@/index.css";
import "@mantine/core/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { Provider } from "react-redux";
import { MotionConfig } from "framer-motion";
import {
  AccessLog,
  ComponentLog,
  Service,
  Session,
} from "@/apis/corev1/corev1";
import { AuditLog, AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GaugeOperation_Function } from "@/apis/visibilityv1/metrics/vmetricsv1";
import { AccessLogC } from "@/components/AccessLogViewer";
import { AuditLogC } from "@/components/AuditLogViewer";
import { AuthenticationLogC } from "@/components/AuthenticationLogViewer";
import { ComponentLogC } from "@/components/ComponentLogViewer";
import MetricChart, { gaugeOp } from "@/components/Charts/MetricChart";
import CopyText from "@/components/CopyText";
import {
  Deferred,
  PanelSkeleton,
  StatTile,
} from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardQuery,
} from "@/components/Dashboard/utils";
import { ListLoading, PageLoading } from "@/components/Loading";
import { ServiceMetrics } from "@/components/ResourceLayout/ServiceMetricsPage";
import { useInView } from "@/hooks/use-in-view";
import { LabelComponent } from "@/pages/core/Session/List";
import Metrics from "@/pages/visibility/Metrics";
import store from "@/store";
import { getClientVisibilityMetrics } from "@/utils/client";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "sparklines";
const calls: string[] = [];
(window as any).calls = calls;
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
const points = [
  { ts: Timestamp.fromDate(new Date("2026-10-13T10:00:00Z")), value: 10 },
  { ts: Timestamp.fromDate(new Date("2026-10-13T11:00:00Z")), value: 20 },
  { ts: Timestamp.fromDate(new Date("2026-10-13T12:00:00Z")), value: 5 },
];
(Object.getPrototypeOf(getClientVisibilityMetrics()) as any).queryMetrics =
  async (request: any) => {
    calls.push(request.metric.selector.name);
    return {
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
    };
  };

const DashboardQuery = (props: { enabled?: boolean }) => {
  const [value, setValue] = React.useState("");
  const query = useDashboardQuery({
    queryKey: ["runtime", props.enabled ?? true],
    enabled: props.enabled,
    refetchMillis: 1000,
    fetch: async () => {
      calls.push(props.enabled === false ? "disabled" : "dashboard");
      return "Loaded";
    },
  });
  return (
    <>
      <output>{query.data}</output>
      <input
        aria-label={props.enabled === false ? "Disabled state" : "Panel state"}
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
    </>
  );
};

const Dashboard = () => {
  const [show, setShow] = React.useState(true);
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  return (
    <AutoRefreshContext.Provider value={autoRefresh}>
      <button onClick={() => setShow(false)}>Unmount dashboard</button>
      <button onClick={() => setAutoRefresh((value) => !value)}>
        Toggle auto refresh
      </button>
      {show && (
        <div data-deferred>
          <Deferred eager={params.get("eager") === "true"}>
            <DashboardQuery />
            <DashboardQuery enabled={false} />
          </Deferred>
        </div>
      )}
    </AutoRefreshContext.Provider>
  );
};

const OneShot = () => {
  const { ref, inView } = useInView<HTMLDivElement>();
  return (
    <div ref={ref} data-once>
      {inView ? "Visible" : "Waiting"}
    </div>
  );
};

const Sparklines = () => {
  const [updated, setUpdated] = React.useState(false);
  const datasets = [
    [...points].reverse(),
    points.map((point) => ({ ...point, value: 0 })),
    [points[0]],
    [
      points[0],
      { ...points[1], value: NaN },
      { ...points[2], value: Infinity },
    ],
    points.map((point, index) => ({
      ...point,
      value: index === 0 ? -Number.MAX_VALUE : Number.MAX_VALUE,
    })),
    [],
  ];
  return (
    <>
      <button onClick={() => setUpdated(true)}>Update sparklines</button>
      {datasets.map((data, index) => (
        <div data-tile={index} key={index}>
          <StatTile
            label={`Stat ${index}`}
            value={10}
            footer="Last hour"
            color={updated ? "#ef4444" : "#3b82f6"}
            points={updated ? [points[0], points[1]] : data}
          />
        </div>
      ))}
    </>
  );
};

const Copy = () => {
  const [show, setShow] = React.useState(true);
  return (
    <>
      <button onClick={() => setShow(false)}>Unmount copy</button>
      {show && <CopyText value="Example" />}
    </>
  );
};

const Logs = () => (
  <>
    <AccessLogC accessLog={AccessLog.create({ entry: { common: {} } })} />
    <AuthenticationLogC authLog={AuthenticationLog.create({ entry: {} })} />
    <AuditLogC auditLog={AuditLog.create({ entry: {} })} />
    <ComponentLogC
      log={ComponentLog.create({ entry: { message: "Example" } })}
    />
  </>
);

const Fixture = () => (
  <Provider store={store}>
    <MotionConfig reducedMotion="user">
      <MantineProvider>
        <QueryClientProvider client={queryClient}>
          <MemoryRouter>
            {scenario === "dashboard" ? (
              <Dashboard />
            ) : scenario === "once" ? (
              <OneShot />
            ) : scenario === "metric" ? (
              <MetricChart
                metric="test"
                title="Test metric"
                operation={gaugeOp(GaugeOperation_Function.AVG)}
                enabled={params.get("enabled") !== "false"}
              />
            ) : scenario === "metrics" ? (
              <Metrics />
            ) : scenario === "service" ? (
              <ServiceMetrics
                resource={Service.create({
                  apiVersion: "core/v1",
                  kind: "Service",
                  metadata: { name: "test" },
                })}
              />
            ) : scenario === "copy" ? (
              <Copy />
            ) : scenario === "logs" ? (
              <Logs />
            ) : scenario === "motion" ? (
              <>
                <ListLoading label="sessions" />
                <PageLoading />
                <PanelSkeleton height={80} />
                {[0, 1, 2].map((key) => (
                  <LabelComponent
                    key={key}
                    item={Session.create({ status: { isConnected: true } })}
                  />
                ))}
              </>
            ) : (
              <Sparklines />
            )}
          </MemoryRouter>
        </QueryClientProvider>
      </MantineProvider>
    </MotionConfig>
  </Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Fixture />);
