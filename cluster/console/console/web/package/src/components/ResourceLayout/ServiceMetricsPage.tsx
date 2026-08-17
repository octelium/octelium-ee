import {
  Namespace,
  Service,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import { Duration } from "@/apis/metav1/metav1";
import {
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import MetricChart, {
  counterOp,
  eqFilter,
  gaugeOp,
  histogramOp,
} from "@/components/Charts/MetricChart";
import PageWrap from "@/components/PageWrap";
import { refetchIntervalChart } from "@/utils/client";
import { ActionIcon, SegmentedControl, Switch, Tooltip } from "@mantine/core";
import { useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  ChartNoAxesCombined,
  Clock3,
  Gauge,
  Network,
  RefreshCw,
} from "lucide-react";
import * as React from "react";
import { useContextResource } from "./utils";

type MetricsResource = Service | Namespace;
type Range = "15m" | "1h" | "6h" | "24h";
type View = "overview" | "details";

const SERVICE_NAME = "octelium.vigil.svc.name";
const NAMESPACE_NAME = "octelium.vigil.svc.namespace.name";
const REGION_NAME = "octelium.vigil.svc.region.name";
const SERVICE_MODE = "octelium.vigil.svc.mode";

const VIGIL_COMPONENT = ComponentSelector.create({
  namespace: "octelium",
  type: "vigil",
});

const RANGE_SECONDS: Record<Range, number> = {
  "15m": 15 * 60,
  "1h": 60 * 60,
  "6h": 6 * 60 * 60,
  "24h": 24 * 60 * 60,
};

const rangeStep = (range: Range): Duration => {
  if (range === "15m") {
    return Duration.create({ type: { oneofKind: "seconds", seconds: 10 } });
  }
  if (range === "1h") {
    return Duration.create({ type: { oneofKind: "seconds", seconds: 30 } });
  }
  if (range === "6h") {
    return Duration.create({ type: { oneofKind: "minutes", minutes: 1 } });
  }
  return Duration.create({ type: { oneofKind: "minutes", minutes: 5 } });
};

const effectiveMode = (service: Service): Service_Spec_Mode =>
  service.spec?.mode === Service_Spec_Mode.MODE_UNSET || !service.spec
    ? Service_Spec_Mode.TCP
    : service.spec.mode;

const modeLabel = (mode: Service_Spec_Mode): string =>
  (Service_Spec_Mode[mode] ?? "UNKNOWN").replaceAll("_", " ");

const HTTP_MODES = new Set<Service_Spec_Mode>([
  Service_Spec_Mode.HTTP,
  Service_Spec_Mode.WEB,
  Service_Spec_Mode.GRPC,
  Service_Spec_Mode.KUBERNETES,
  Service_Spec_Mode.RDP_WEB,
  Service_Spec_Mode.MCP,
  Service_Spec_Mode.LLM,
]);

const STREAM_MODES = new Set<Service_Spec_Mode>([
  Service_Spec_Mode.TCP,
  Service_Spec_Mode.UDP,
  Service_Spec_Mode.SSH,
  Service_Spec_Mode.POSTGRES,
  Service_Spec_Mode.MYSQL,
  Service_Spec_Mode.SOCKS5,
]);

const SectionIntro = (props: {
  title: string;
  description: string;
  icon: React.ReactNode;
}) => (
  <div className="flex items-start gap-3 rounded-xl border border-slate-200/80 bg-white px-3.5 py-3">
    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
      {props.icon}
    </span>
    <div className="min-w-0">
      <h3 className="text-sm font-bold text-slate-800">{props.title}</h3>
      <p className="mt-0.5 text-[0.7rem] font-medium leading-5 text-slate-500">
        {props.description}
      </p>
    </div>
  </div>
);

const BaseCharts = (props: {
  scope: "service" | "namespace";
  shared: SharedChartProps;
}) => {
  const namespace = props.scope === "namespace";

  return (
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title={namespace ? "Request rate by Service" : "Request rate"}
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={namespace ? [SERVICE_NAME] : undefined}
        limitSeries={namespace ? 12 : 1}
        {...props.shared}
      />
      <MetricChart
        title={namespace ? "Active requests by Service" : "Active requests"}
        unit="requests"
        metric="req.active"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={namespace ? [SERVICE_NAME] : undefined}
        limitSeries={namespace ? 12 : 1}
        {...props.shared}
      />
      <MetricChart
        title="Request latency"
        unit="ms"
        metric="req.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        limitSeries={4}
        {...props.shared}
      />
      <MetricChart
        title="Allowed and denied requests"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["state"]}
        limitSeries={3}
        {...props.shared}
      />
      <MetricChart
        title="Denied requests by reason"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["reason"]}
        limitSeries={12}
        {...props.shared}
        filters={[...props.shared.filters, eqFilter("state", "DENIED")]}
      />
      <MetricChart
        title="Request rate by Region"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={[REGION_NAME]}
        limitSeries={10}
        {...props.shared}
      />
    </div>
  );
};

type SharedChartProps = {
  component: ComponentSelector;
  filters: ReturnType<typeof eqFilter>[];
  lookbackSeconds: number;
  step: Duration;
  autoRefresh: boolean;
  hideResolution: boolean;
  limitPointsPerSeries: number;
  height: number;
};

const HTTPDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="HTTP request characteristics"
      description="Method and response-class dimensions emitted by Vigil's HTTP gateway."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by HTTP method"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.method"]}
        limitSeries={10}
        {...props.shared}
      />
      <MetricChart
        title="Requests by response class"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.status"]}
        limitSeries={6}
        {...props.shared}
      />
    </div>
  </>
);

const MCPDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="Model Context Protocol traffic"
      description="MCP method and negotiated protocol-version dimensions reported by Vigil."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by MCP method"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.mcp.method"]}
        limitSeries={15}
        {...props.shared}
      />
      <MetricChart
        title="Requests by protocol version"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.mcp.protocol_version"]}
        limitSeries={10}
        {...props.shared}
      />
      <MetricChart
        title="Requests by response class"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.status"]}
        limitSeries={6}
        {...props.shared}
      />
    </div>
  </>
);

const LLMDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="LLM gateway traffic"
      description="Inference operation, provider protocol, and streaming dimensions reported by Vigil."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by operation"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.llm.operation"]}
        limitSeries={10}
        {...props.shared}
      />
      <MetricChart
        title="Requests by protocol"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.llm.protocol"]}
        limitSeries={10}
        {...props.shared}
      />
      <MetricChart
        title="Streaming and non-streaming requests"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.llm.stream"]}
        limitSeries={3}
        {...props.shared}
      />
      <MetricChart
        title="Requests by response class"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.status"]}
        limitSeries={6}
        {...props.shared}
      />
    </div>
  </>
);

const DNSDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="DNS request characteristics"
      description="Query types and response-code classes emitted by DNS-mode Vigil instances."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by query type"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["qtype"]}
        limitSeries={10}
        {...props.shared}
      />
      <MetricChart
        title="Requests by response class"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["rcode_class"]}
        limitSeries={8}
        {...props.shared}
      />
    </div>
  </>
);

const StreamDetails = (props: {
  shared: SharedChartProps;
  groupByService?: boolean;
}) => (
  <>
    <SectionIntro
      title="Connection throughput"
      description="Bytes transferred between downstream clients and stream-oriented Services."
      icon={<Activity size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Bytes received from clients"
        unit="bytes/s"
        metric="req.bytes_received"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={props.groupByService ? [SERVICE_NAME] : undefined}
        limitSeries={props.groupByService ? 12 : 1}
        {...props.shared}
      />
      <MetricChart
        title="Bytes sent to clients"
        unit="bytes/s"
        metric="req.bytes_sent"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={props.groupByService ? [SERVICE_NAME] : undefined}
        limitSeries={props.groupByService ? 12 : 1}
        {...props.shared}
      />
    </div>
  </>
);

const NamespaceDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="Namespace Service mix"
      description="Compare traffic modes, deployment Regions, and stream throughput across this Namespace."
      icon={<ChartNoAxesCombined size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Request rate by Service mode"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={[SERVICE_MODE]}
        limitSeries={12}
        {...props.shared}
      />
      <MetricChart
        title="Active requests by Service mode"
        unit="requests"
        metric="req.active"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={[SERVICE_MODE]}
        limitSeries={10}
        {...props.shared}
      />
    </div>
    <StreamDetails shared={props.shared} groupByService />
  </>
);

const ServiceDetails = (props: {
  service: Service;
  shared: SharedChartProps;
}) => {
  const mode = effectiveMode(props.service);
  if (mode === Service_Spec_Mode.MCP) {
    return <MCPDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.LLM) {
    return <LLMDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.DNS) {
    return <DNSDetails shared={props.shared} />;
  }
  if (STREAM_MODES.has(mode)) {
    return <StreamDetails shared={props.shared} />;
  }
  if (HTTP_MODES.has(mode)) {
    return <HTTPDetails shared={props.shared} />;
  }
  return null;
};

export const ResourceMetrics = (props: { resource: MetricsResource }) => {
  const { resource } = props;
  const [range, setRange] = React.useState<Range>("6h");
  const [view, setView] = React.useState<View>("overview");
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [visible, setVisible] = React.useState(
    () => document.visibilityState === "visible",
  );
  const queryClient = useQueryClient();
  const isService = resource.kind === "Service";
  const service = isService ? (resource as Service) : undefined;
  const namespaceName = isService
    ? service?.status?.namespaceRef?.name
    : resource.metadata?.name;
  const filters = [
    ...(isService && resource.metadata?.name
      ? [eqFilter(SERVICE_NAME, resource.metadata.name)]
      : []),
    ...(namespaceName ? [eqFilter(NAMESPACE_NAME, namespaceName)] : []),
  ];
  const mode = service ? effectiveMode(service) : undefined;
  const refresh = autoRefresh && visible;

  React.useEffect(() => {
    const onVisibilityChange = () =>
      setVisible(document.visibilityState === "visible");
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () =>
      document.removeEventListener("visibilitychange", onVisibilityChange);
  }, []);

  React.useEffect(() => {
    setView("overview");
  }, [resource.metadata?.uid]);

  if (resource.apiVersion !== "core/v1" || filters.length === 0) return null;

  const shared: SharedChartProps = {
    component: VIGIL_COMPONENT,
    filters,
    lookbackSeconds: RANGE_SECONDS[range],
    step: rangeStep(range),
    autoRefresh: refresh,
    hideResolution: true,
    limitPointsPerSeries: 500,
    height: 210,
  };

  return (
    <div className="space-y-4 pb-5">
      <header className="overflow-hidden rounded-xl border border-slate-200/80 bg-white">
        <div className="flex flex-col gap-4 px-4 py-3.5 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
              <ChartNoAxesCombined size={17} strokeWidth={2.2} />
            </span>
            <div className="min-w-0">
              <h2 className="text-sm font-bold text-slate-900">
                {isService ? "Service metrics" : "Namespace metrics"}
              </h2>
              <p className="mt-0.5 text-[0.7rem] font-medium leading-5 text-slate-500">
                Vigil traffic, authorization outcomes, latency, and protocol
                characteristics scoped to this {isService ? "Service" : "Namespace"}.
              </p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {mode !== undefined && (
                  <span className="rounded-md border border-blue-200 bg-blue-50 px-2 py-1 text-[0.62rem] font-bold text-blue-700">
                    {modeLabel(mode)}
                  </span>
                )}
                {namespaceName && isService && (
                  <span className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-[0.62rem] font-bold text-slate-500">
                    Namespace · {namespaceName}
                  </span>
                )}
                {service?.status?.regionRef?.name && (
                  <span className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-[0.62rem] font-bold text-slate-500">
                    Region · {service.status.regionRef.name}
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <Switch
              size="xs"
              label="Live"
              checked={autoRefresh}
              onChange={(event) => setAutoRefresh(event.currentTarget.checked)}
            />
            <Tooltip label="Refresh metrics" withArrow>
              <ActionIcon
                type="button"
                variant="light"
                color="gray"
                aria-label="Refresh metrics"
                onClick={() =>
                  queryClient.invalidateQueries({
                    queryKey: ["visibility", "queryMetrics"],
                  })
                }
              >
                <RefreshCw size={14} strokeWidth={2.2} />
              </ActionIcon>
            </Tooltip>
          </div>
        </div>

        <div className="flex flex-col gap-3 border-t border-slate-100 bg-slate-50/60 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <SegmentedControl
            value={view}
            onChange={(value) => setView(value as View)}
            data={[
              { value: "overview", label: "Overview" },
              {
                value: "details",
                label: isService ? "Protocol details" : "Service breakdown",
              },
            ]}
          />
          <div className="flex items-center gap-2">
            <Clock3 size={13} className="text-slate-400" />
            <SegmentedControl
              size="xs"
              value={range}
              onChange={(value) => setRange(value as Range)}
              data={["15m", "1h", "6h", "24h"]}
            />
          </div>
        </div>
      </header>

      {view === "overview" ? (
        <BaseCharts scope={isService ? "service" : "namespace"} shared={shared} />
      ) : service ? (
        <ServiceDetails service={service} shared={shared} />
      ) : (
        <NamespaceDetails shared={shared} />
      )}

      <p className="px-1 text-[0.62rem] font-semibold leading-5 text-slate-400">
        Live refresh runs every {Math.round(refetchIntervalChart / 1000)} seconds
        while this page is visible. Historical values follow the current
        metrics retention policy.
      </p>
    </div>
  );
};

export const ServiceMetrics = (props: { resource: Service }) => (
  <ResourceMetrics resource={props.resource} />
);

export const NamespaceMetrics = (props: { resource: Namespace }) => (
  <ResourceMetrics resource={props.resource} />
);

const ServiceMetricsPage = () => {
  const ctx = useContextResource();

  if (!ctx) return null;

  return (
    <PageWrap qry={ctx}>
      {ctx.data &&
        (ctx.data.kind === "Service" || ctx.data.kind === "Namespace") && (
          <ResourceMetrics resource={ctx.data as MetricsResource} />
        )}
    </PageWrap>
  );
};

export default ServiceMetricsPage;
