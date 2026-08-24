import { Namespace, Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  AttributeFilter,
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricSelector,
  NumberPoint,
  QueryMetricsRequest,
  QueryMetricsRequest_LimitBehavior,
  QueryMetricsRequest_SeriesAggregation,
  TimeRange,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import MetricChart, {
  counterOp,
  eqFilter,
  gaugeOp,
  histogramOp,
} from "@/components/Charts/MetricChart";
import PageWrap from "@/components/PageWrap";
import { SummaryItemCount } from "@/components/Summary";
import {
  getClientVisibilityMetrics,
  refetchIntervalChart,
} from "@/utils/client";
import { ActionIcon, SegmentedControl, Switch, Tooltip } from "@mantine/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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

const rawCounterOperation = counterOp(CounterOperation_Function.RAW);
const lastGaugeOperation = gaugeOp(GaugeOperation_Function.LAST);

const numberPointValue = (point: NumberPoint): number | undefined => {
  switch (point.value.oneofKind) {
    case "asDouble":
      return point.value.asDouble;
    case "asInt":
      return Number(point.value.asInt);
    default:
      return undefined;
  }
};

const latestSeriesValue = (points: NumberPoint[]): number => {
  let latestTimestamp = Number.NEGATIVE_INFINITY;
  let latestValue = 0;

  for (const point of points) {
    const value = numberPointValue(point);
    if (value === undefined) continue;
    const timestamp = point.timestamp
      ? Number(point.timestamp.seconds) * 1000 + Number(point.timestamp.nanos ?? 0) / 1e6
      : 0;
    if (timestamp >= latestTimestamp) {
      latestTimestamp = timestamp;
      latestValue = value;
    }
  }

  return latestValue;
};

const metricCounterValue = (response: any): number =>
  (response?.series ?? []).reduce((total: number, series: any) => {
    if (series.points?.oneofKind !== "number") return total;
    return total + latestSeriesValue(series.points.number.points ?? []);
  }, 0);

type ServiceMetricCounterProps = {
  shared: SharedChartProps;
  title: string;
  label: string;
  metric: string;
  operation?: ReturnType<typeof counterOp> | ReturnType<typeof gaugeOp>;
  filters?: AttributeFilter[];
  formatter?: (value: number) => string;
};

const ServiceMetricCounter = (props: ServiceMetricCounterProps) => {
  const operation = props.operation ?? rawCounterOperation;
  const isRawCounter =
    operation.type.oneofKind === "counter" &&
    operation.type.counter.function === CounterOperation_Function.RAW;
  const operationFunction =
    operation.type.oneofKind === "counter"
      ? operation.type.counter.function
      : operation.type.oneofKind === "gauge"
        ? operation.type.gauge.function
        : undefined;
  const filters = [...props.shared.filters, ...(props.filters ?? [])];

  const qry = useQuery({
    queryKey: [
      "visibility",
      "serviceMetricCounter",
      props.metric,
      operation.type.oneofKind,
      operationFunction,
      filters.map((filter) => JSON.stringify(filter)),
      props.shared.lookbackSeconds,
    ],
    enabled: true,
    queryFn: async ({ signal }) => {
      const to = Timestamp.fromDate(new Date());
      const from = Timestamp.fromDate(
        new Date(Date.now() - props.shared.lookbackSeconds * 1000),
      );
      const { response } = await getClientVisibilityMetrics().queryMetrics(
        QueryMetricsRequest.create({
          metric: MetricSelector.create({
            selector: { oneofKind: "name", name: props.metric },
          }),
          timeRange: TimeRange.create({ from, to }),
          step: isRawCounter ? undefined : props.shared.step,
          component: props.shared.component,
          filters,
          operation,
          limitSeries: 64,
          limitPointsPerSeries: isRawCounter ? 32 : 2,
          seriesAggregation: isRawCounter
            ? QueryMetricsRequest_SeriesAggregation.NONE
            : QueryMetricsRequest_SeriesAggregation.SUM,
          limitBehavior: QueryMetricsRequest_LimitBehavior.TRUNCATE,
        }),
        { abort: signal },
      );
      return response;
    },
    refetchInterval: props.shared.autoRefresh ? refetchIntervalChart : false,
    refetchIntervalInBackground: false,
    retry: 1,
  });

  const value = Math.max(0, metricCounterValue(qry.data));

  return (
    <div className="min-w-0">
      {qry.isLoading ? (
        <div className="flex min-h-[76px] items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 shadow-[0_1px_2px_rgba(15,23,42,0.035)]">
          <span className="h-4 w-4 animate-spin rounded-full border-2 border-slate-200 border-t-slate-700" />
          <span className="text-[0.68rem] font-semibold text-slate-400">Loading…</span>
        </div>
      ) : qry.isError ? (
        <div className="flex min-h-[76px] flex-col justify-center rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 shadow-[0_1px_2px_rgba(15,23,42,0.035)]">
          <span className="truncate text-[0.72rem] font-bold text-slate-600">{props.title}</span>
          <span className="mt-0.5 truncate text-[0.65rem] font-semibold text-slate-400">Unavailable</span>
        </div>
      ) : (
        <SummaryItemCount
          count={value}
          showZero
          formatCount={props.formatter}
        >
          {props.title}
          <span className="ml-1 font-semibold text-slate-400">· {props.label}</span>
        </SummaryItemCount>
      )}
    </div>
  );
};

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
  mode?: Service_Spec_Mode;
}) => {
  const namespace = props.scope === "namespace";
  const stream = props.mode !== undefined && STREAM_MODES.has(props.mode);
  const http = props.mode !== undefined && HTTP_MODES.has(props.mode);
  const sessions = stream || http;

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
      {!namespace && (
        <>
          {sessions && (
            <MetricChart
              title="Active sessions"
              unit="sessions"
              metric="session.active"
              operation={gaugeOp(GaugeOperation_Function.LAST)}
              limitSeries={1}
              {...props.shared}
            />
          )}
          {http && (
            <MetricChart
              title="Time to first byte"
              unit="ms"
              metric="req.ttfb"
              operation={histogramOp(
                HistogramOperation_Function.QUANTILE,
                [0.5, 0.95, 0.99],
              )}
              limitSeries={4}
              {...props.shared}
            />
          )}
        </>
      )}
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

const formatCompactCounter = (value: number): string =>
  new Intl.NumberFormat(undefined, {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);

const formatBytesCounter = (value: number): string => {
  if (!Number.isFinite(value) || value === 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let current = value;
  let unit = 0;
  while (current >= 1024 && unit < units.length - 1) {
    current /= 1024;
    unit += 1;
  }
  return `${current.toFixed(current < 10 && unit > 0 ? 1 : 0)} ${units[unit]}`;
};

const MainCounters = (props: {
  shared: SharedChartProps;
  mode?: Service_Spec_Mode;
}) => {
  const isService = props.mode !== undefined;
  const httpOrStream =
    props.mode !== undefined &&
    (HTTP_MODES.has(props.mode) || STREAM_MODES.has(props.mode));
  const cards: React.ReactNode[] = [
    <ServiceMetricCounter
      key="total-requests"
      shared={props.shared}
      title="Total requests"
      label="requests"
      metric="req.total"
      formatter={formatCompactCounter}
    />,
    <ServiceMetricCounter
      key="allowed-requests"
      shared={props.shared}
      title="Allowed requests"
      label="requests"
      metric="req.total"
      filters={[eqFilter("state", "ALLOWED")]}
      formatter={formatCompactCounter}
    />,
    <ServiceMetricCounter
      key="denied-requests"
      shared={props.shared}
      title="Denied requests"
      label="requests"
      metric="req.total"
      filters={[eqFilter("state", "DENIED")]}
      formatter={formatCompactCounter}
    />,
    <ServiceMetricCounter
      key="active-requests"
      shared={props.shared}
      title="Active requests"
      label="requests"
      metric="req.active"
      operation={lastGaugeOperation}
      formatter={formatCompactCounter}
    />,
  ];

  if (httpOrStream) {
    cards.push(
      <ServiceMetricCounter
        key="bytes-received"
        shared={props.shared}
        title="Bytes received"
        label="downstream"
        metric="req.bytes_received"
        formatter={formatBytesCounter}
      />,
      <ServiceMetricCounter
        key="bytes-sent"
        shared={props.shared}
        title="Bytes sent"
        label="downstream"
        metric="req.bytes_sent"
        formatter={formatBytesCounter}
      />,
    );
  }

  if (isService && (HTTP_MODES.has(props.mode!) || STREAM_MODES.has(props.mode!))) {
    cards.push(
      <ServiceMetricCounter
        key="active-sessions"
        shared={props.shared}
        title="Active sessions"
        label="sessions"
        metric="session.active"
        operation={lastGaugeOperation}
        formatter={formatCompactCounter}
      />,
    );
  }

  if (props.mode === Service_Spec_Mode.LLM) {
    cards.push(
      <ServiceMetricCounter
        key="llm-total-tokens"
        shared={props.shared}
        title="Total LLM tokens"
        label="tokens"
        metric="llm.tokens.total"
        formatter={formatCompactCounter}
      />,
      <ServiceMetricCounter
        key="llm-input-tokens"
        shared={props.shared}
        title="Input tokens"
        label="tokens"
        metric="llm.tokens.input"
        formatter={formatCompactCounter}
      />,
      <ServiceMetricCounter
        key="llm-output-tokens"
        shared={props.shared}
        title="Output tokens"
        label="tokens"
        metric="llm.tokens.output"
        formatter={formatCompactCounter}
      />,
      <ServiceMetricCounter
        key="llm-stream-events"
        shared={props.shared}
        title="Stream events"
        label="events"
        metric="llm.stream.events"
        formatter={formatCompactCounter}
      />,
    );
  }

  if (props.mode === Service_Spec_Mode.MCP) {
    cards.push(
      <ServiceMetricCounter
        key="mcp-tool-calls"
        shared={props.shared}
        title="MCP tool calls"
        label="calls"
        metric="req.total"
        filters={[eqFilter("req.mcp.method", "tools/call")]}
        formatter={formatCompactCounter}
      />,
      <ServiceMetricCounter
        key="mcp-tool-errors"
        shared={props.shared}
        title="MCP tool errors"
        label="errors"
        metric="req.total"
        filters={[eqFilter("req.mcp.error", "TOOL")]}
        formatter={formatCompactCounter}
      />,
    );
  }

  if (props.mode === Service_Spec_Mode.SSH) {
    cards.push(
      <ServiceMetricCounter
        key="ssh-channels"
        shared={props.shared}
        title="SSH channels"
        label="channels"
        metric="ssh.channel.total"
        formatter={formatCompactCounter}
      />,
      <ServiceMetricCounter
        key="ssh-requests"
        shared={props.shared}
        title="SSH channel requests"
        label="requests"
        metric="ssh.request.total"
        formatter={formatCompactCounter}
      />,
    );
  }

  if (
    isService &&
    (props.mode === Service_Spec_Mode.POSTGRES ||
      props.mode === Service_Spec_Mode.MYSQL)
  ) {
    cards.push(
      <ServiceMetricCounter
        key="db-commands"
        shared={props.shared}
        title="Database commands"
        label="commands"
        metric="db.command.total"
        formatter={formatCompactCounter}
      />,
    );
  }

  if (props.mode === Service_Spec_Mode.DNS) {
    cards.push(
      <ServiceMetricCounter
        key="dns-malformed"
        shared={props.shared}
        title="Malformed DNS queries"
        label="queries"
        metric="dns.malformed.total"
        formatter={formatCompactCounter}
      />,
    );
  }

  return (
    <section>
      <SectionIntro
        title="Current counters"
        description="Latest cumulative totals and live gauges for this scope."
        icon={<Gauge size={16} strokeWidth={2.2} />}
      />
      <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {cards}
      </div>
    </section>
  );
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
      <MetricChart
        title="Requests by HTTP version"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.version"]}
        limitSeries={4}
        {...props.shared}
      />
      <MetricChart
        title="Streaming request types"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.http.stream"]}
        limitSeries={8}
        {...props.shared}
      />
      <MetricChart
        title="Streaming session duration"
        unit="ms"
        metric="session.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        groupBy={["req.http.stream"]}
        limitSeries={8}
        {...props.shared}
      />
    </div>
  </>
);

const GRPCDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="gRPC traffic"
      description="gRPC routes and status classes captured by the HTTP dataplane."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by gRPC method"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.grpc.method"]}
        limitSeries={20}
        {...props.shared}
      />
      <MetricChart
        title="Requests by gRPC service"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.grpc.service_full_name"]}
        limitSeries={20}
        {...props.shared}
      />
      <MetricChart
        title="gRPC status classes"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.grpc.status_class"]}
        limitSeries={6}
        {...props.shared}
      />
      <MetricChart
        title="gRPC status codes"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.grpc.status"]}
        limitSeries={20}
        {...props.shared}
      />
    </div>
  </>
);

const KubernetesDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="Kubernetes API traffic"
      description="Kubernetes verbs, resources, subresources, and streaming operations observed by Vigil."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      {[
        ["Requests by verb", "req.k8s.verb", 12],
        ["Requests by resource", "req.k8s.resource", 24],
        ["Requests by subresource", "req.k8s.subresource", 16],
        ["Requests by API group", "req.k8s.api_group", 16],
        ["Streaming operations", "req.http.stream", 8],
      ].map(([title, groupBy, limit]) => (
        <MetricChart
          key={groupBy as string}
          title={title as string}
          unit="requests/s"
          metric="req.total"
          operation={counterOp(CounterOperation_Function.RATE)}
          groupBy={[groupBy as string]}
          limitSeries={limit as number}
          {...props.shared}
        />
      ))}
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
      <MetricChart
        title="MCP names"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.mcp.name"]}
        limitSeries={20}
        {...props.shared}
      />
      <MetricChart
        title="MCP errors"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.mcp.error"]}
        limitSeries={5}
        {...props.shared}
      />
      <MetricChart
        title="Notifications"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.mcp.is_notification"]}
        limitSeries={3}
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
      <MetricChart
        title="Requests by model"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.llm.model"]}
        limitSeries={20}
        {...props.shared}
      />
      <MetricChart
        title="Finish reasons"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["req.llm.finish_reason"]}
        limitSeries={12}
        {...props.shared}
      />
      <MetricChart
        title="Input tokens"
        unit="tokens/s"
        metric="llm.tokens.input"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Output tokens"
        unit="tokens/s"
        metric="llm.tokens.output"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Total tokens"
        unit="tokens/s"
        metric="llm.tokens.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Cache-read tokens"
        unit="tokens/s"
        metric="llm.tokens.cache_read"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Cache-write tokens"
        unit="tokens/s"
        metric="llm.tokens.cache_write"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Reasoning tokens"
        unit="tokens/s"
        metric="llm.tokens.reasoning"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
      <MetricChart
        title="Time to first token"
        unit="ms"
        metric="llm.ttft"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        limitSeries={4}
        {...props.shared}
      />
      <MetricChart
        title="Stream events"
        unit="events/s"
        metric="llm.stream.events"
        operation={counterOp(CounterOperation_Function.RATE)}
        limitSeries={1}
        {...props.shared}
      />
    </div>
  </>
);

const DatabaseDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="Database traffic"
      description="Database commands, authorization outcomes, and command-level latency."
      icon={<Activity size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Commands by type"
        unit="commands/s"
        metric="db.command.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["db.command"]}
        limitSeries={20}
        {...props.shared}
      />
      <MetricChart
        title="Command outcomes"
        unit="commands/s"
        metric="db.command.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["state"]}
        limitSeries={3}
        {...props.shared}
      />
      <MetricChart
        title="Denied command reasons"
        unit="commands/s"
        metric="db.command.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["reason"]}
        limitSeries={12}
        {...props.shared}
        filters={[...props.shared.filters, eqFilter("state", "DENIED")]}
      />
      <MetricChart
        title="Authorization latency"
        unit="ms"
        metric="db.authz.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        limitSeries={4}
        {...props.shared}
      />
    </div>
  </>
);

const SOCKS5Details = (props: { shared: SharedChartProps }) => (
  <>
    <StreamDetails shared={props.shared} />
    <SectionIntro
      title="SOCKS5 destinations"
      description="Connection targets grouped by the address type negotiated with the downstream client."
      icon={<Network size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Requests by target type"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["socks5.target_type"]}
        limitSeries={4}
        {...props.shared}
      />
    </div>
  </>
);

const SSHDetails = (props: { shared: SharedChartProps }) => (
  <>
    <SectionIntro
      title="SSH activity"
      description="SSH channels and channel requests, including authorization outcomes and reasons."
      icon={<Activity size={16} strokeWidth={2.2} />}
    />
    <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
      <MetricChart
        title="Channels by type"
        unit="channels/s"
        metric="ssh.channel.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["ssh.channel.type"]}
        limitSeries={12}
        {...props.shared}
      />
      <MetricChart
        title="Channel outcomes"
        unit="channels/s"
        metric="ssh.channel.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["state"]}
        limitSeries={3}
        {...props.shared}
      />
      <MetricChart
        title="Requests by type"
        unit="requests/s"
        metric="ssh.request.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["ssh.request.type"]}
        limitSeries={16}
        {...props.shared}
      />
      <MetricChart
        title="Denied request reasons"
        unit="requests/s"
        metric="ssh.request.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["reason"]}
        limitSeries={12}
        {...props.shared}
        filters={[...props.shared.filters, eqFilter("state", "DENIED")]}
      />
      <MetricChart
        title="ESSH sessions"
        unit="sessions"
        metric="session.duration"
        operation={histogramOp(HistogramOperation_Function.COUNT)}
        groupBy={["req.ssh.is_essh"]}
        limitSeries={3}
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
      <MetricChart
        title="Malformed queries"
        unit="queries/s"
        metric="dns.malformed.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["reason"]}
        limitSeries={12}
        {...props.shared}
      />
      <MetricChart
        title="Response size"
        unit="bytes"
        metric="dns.response.bytes"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        groupBy={["dns.truncated"]}
        limitSeries={3}
        {...props.shared}
      />
    </div>
  </>
);

const StreamDetails = (props: {
  shared: SharedChartProps;
  groupByService?: boolean;
  showPackets?: boolean;
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
      <MetricChart
        title="Session duration"
        unit="ms"
        metric="session.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
        limitSeries={4}
        {...props.shared}
      />
      <MetricChart
        title="Rejected connections"
        unit="connections/s"
        metric="conn.rejected"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["stage"]}
        limitSeries={8}
        {...props.shared}
      />
      {props.showPackets && (
        <>
          <MetricChart
            title={props.groupByService ? "Packets received by Service" : "Packets received"}
            unit="packets/s"
            metric="req.packets_received"
            operation={counterOp(CounterOperation_Function.RATE)}
            groupBy={props.groupByService ? [SERVICE_NAME] : undefined}
            limitSeries={props.groupByService ? 12 : 1}
            {...props.shared}
          />
          <MetricChart
            title={props.groupByService ? "Packets sent by Service" : "Packets sent"}
            unit="packets/s"
            metric="req.packets_sent"
            operation={counterOp(CounterOperation_Function.RATE)}
            groupBy={props.groupByService ? [SERVICE_NAME] : undefined}
            limitSeries={props.groupByService ? 12 : 1}
            {...props.shared}
          />
        </>
      )}
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
    <StreamDetails shared={props.shared} groupByService showPackets />
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
  if (mode === Service_Spec_Mode.GRPC) {
    return <GRPCDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.KUBERNETES) {
    return <KubernetesDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.DNS) {
    return <DNSDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.SSH) {
    return <SSHDetails shared={props.shared} />;
  }
  if (
    mode === Service_Spec_Mode.POSTGRES ||
    mode === Service_Spec_Mode.MYSQL
  ) {
    return <DatabaseDetails shared={props.shared} />;
  }
  if (mode === Service_Spec_Mode.SOCKS5) {
    return <SOCKS5Details shared={props.shared} />;
  }
  if (STREAM_MODES.has(mode)) {
    return (
      <StreamDetails
        shared={props.shared}
        showPackets={mode === Service_Spec_Mode.UDP}
      />
    );
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

      <MainCounters shared={shared} mode={mode} />

      {view === "overview" ? (
        <BaseCharts
          scope={isService ? "service" : "namespace"}
          mode={mode}
          shared={shared}
        />
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
