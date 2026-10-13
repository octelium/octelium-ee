import "@/index.css";
import "@mantine/core/styles.layer.css";
import { Namespace, Service, User } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import FilterBar from "@/components/LLMVisibility/FilterBar";
import { LLMRange, rangeFromMinutes } from "@/components/LLMVisibility/utils";
import PeriodSelector from "@/components/LogWidget/PeriodSelector";
import { ResourceAccessLogs } from "@/components/ResourceLayout/ResourceAccessLogs";
import { ResourceAuthenticationLogs } from "@/components/ResourceLayout/ResourceAuthenticationLogs";
import { ResourceAuditLogs } from "@/components/ResourceLayout/ResourceAuditLogs";
import { ResourceMetrics } from "@/components/ResourceLayout/ServiceMetricsPage";
import Metrics from "@/pages/visibility/Metrics";
import store from "@/store";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityMetrics,
} from "@/utils/client";
import { MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { BrowserRouter, useLocation, useNavigate } from "react-router-dom";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "period";
const calls: { name: string; request: any; now: number }[] = [];
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
(window as any).consistency = { calls, queryClient };

for (const client of [
  getClientVisibilityMetrics(),
  getClientVisibilityAccessLog(),
  getClientVisibilityAuthenticationLog(),
  getClientVisibilityAuditLog(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      calls.push({ name: method.localName, request, now: Date.now() });
      return { response: method.O.create() };
    };
  }
}

const Period = () => {
  const [value, setValue] = React.useState(60);
  return (
    <>
      <PeriodSelector value={value} onChange={setValue} />
      <output data-period>{value}</output>
    </>
  );
};

const LLM = () => {
  const [minutes, setMinutes] = React.useState<number | undefined>(1440);
  const [range, setRange] = React.useState<LLMRange>(rangeFromMinutes(1440));
  const [status, setStatus] = React.useState(0);
  return (
    <>
      <FilterBar
        minutes={minutes}
        onMinutesChange={(value) => {
          setMinutes(value);
          setRange(rangeFromMinutes(value));
        }}
        range={range}
        onRangeChange={(value) => {
          setMinutes(undefined);
          setRange(value);
        }}
        status={status}
        onStatusChange={setStatus}
        drilldowns={[]}
        onRemoveDrilldown={() => {}}
        onClearDrilldowns={() => {}}
        onRefresh={() => {}}
      />
      <output data-llm>
        {JSON.stringify({
          minutes,
          from: range.from && Timestamp.toDate(range.from).toISOString(),
          to: range.to && Timestamp.toDate(range.to).toISOString(),
          status,
        })}
      </output>
    </>
  );
};

const service = Service.create({
  apiVersion: "core/v1",
  kind: "Service",
  metadata: { name: "web", uid: "service-uid" },
  status: { namespaceRef: { name: "production", uid: "namespace-uid" } },
});
const namespace = Namespace.create({
  apiVersion: "core/v1",
  kind: "Namespace",
  metadata: { name: "production", uid: "namespace-uid" },
});
const user = User.create({
  apiVersion: "core/v1",
  kind: "User",
  metadata: { name: "alice", uid: "user-uid" },
});

const Fixture = () => {
  const location = useLocation();
  const navigate = useNavigate();
  (window as any).consistency.navigate = navigate;
  return (
    <>
      <output data-route>{location.search}</output>
      {scenario === "metrics" ? (
        <Metrics />
      ) : scenario === "service" || scenario === "namespace" ? (
        <ResourceMetrics
          resource={scenario === "service" ? service : namespace}
        />
      ) : scenario === "access" ? (
        <ResourceAccessLogs resource={user} />
      ) : scenario === "authentication" ? (
        <ResourceAuthenticationLogs resource={user} />
      ) : scenario === "audit" ? (
        <ResourceAuditLogs resource={user} />
      ) : scenario === "llm" ? (
        <LLM />
      ) : (
        <Period />
      )}
    </>
  );
};

createRoot(document.getElementById("root")!).render(
  <Provider store={store}>
    <MantineProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Fixture />
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  </Provider>,
);
