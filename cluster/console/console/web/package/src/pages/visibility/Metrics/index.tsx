import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  AttributeFilter,
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricDescriptor_Kind,
  MetricSelector,
  QueryMetricsRequest,
  QueryMetricsRequest_LimitBehavior,
  QueryMetricsRequest_SeriesAggregation,
  QueryOperation,
  TimeRange,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import MetricChart, {
  counterOp,
  eqFilter,
  gaugeOp,
  histogramOp,
} from "@/components/Charts/MetricChart";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import {
  getClientVisibilityMetrics,
  refetchIntervalChart,
} from "@/utils/client";
import {
  ActionIcon,
  SegmentedControl,
  Select,
  Switch,
  Tooltip,
} from "@mantine/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Clock3,
  ChevronDown,
  Cpu,
  Gauge,
  RefreshCw,
  ServerCog,
  ShieldCheck,
  Waypoints,
} from "lucide-react";
import * as React from "react";

type View = "overview" | "traffic" | "authorization" | "components";
type Range = "15m" | "1h" | "6h" | "24h";
type Resolution = "auto" | "10s" | "30s" | "1m" | "5m";

const ranges: Record<Range, number> = {
  "15m": 15 * 60,
  "1h": 60 * 60,
  "6h": 6 * 60 * 60,
  "24h": 24 * 60 * 60,
};

const duration = (value: Resolution, range: Range): Duration => {
  const resolved = value === "auto"
    ? range === "15m" ? "10s" : range === "1h" ? "30s" : range === "6h" ? "1m" : "5m"
    : value;
  if (resolved === "10s") return Duration.create({ type: { oneofKind: "seconds", seconds: 10 } });
  if (resolved === "30s") return Duration.create({ type: { oneofKind: "seconds", seconds: 30 } });
  if (resolved === "1m") return Duration.create({ type: { oneofKind: "minutes", minutes: 1 } });
  return Duration.create({ type: { oneofKind: "minutes", minutes: 5 } });
};

const aggregation = (operation: QueryOperation) => {
  if (operation.type.oneofKind === "gauge") {
    return operation.type.gauge.function === GaugeOperation_Function.AVG
      ? QueryMetricsRequest_SeriesAggregation.AVG
      : QueryMetricsRequest_SeriesAggregation.SUM;
  }
  return operation.type.oneofKind === "histogram"
    ? QueryMetricsRequest_SeriesAggregation.MERGE
    : QueryMetricsRequest_SeriesAggregation.SUM;
};

const numericValue = (point?: { value: { oneofKind: "asDouble"; asDouble: number } | { oneofKind: "asInt"; asInt: number } | { oneofKind: undefined } }) =>
  point?.value.oneofKind === "asDouble"
    ? point.value.asDouble
    : point?.value.oneofKind === "asInt"
      ? Number(point.value.asInt)
      : undefined;

const formatValue = (value: number | undefined, unit: string) => {
  if (value === undefined || !Number.isFinite(value)) return "—";
  if (unit === "bytes") {
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let current = value;
    let index = 0;
    while (current >= 1024 && index < units.length - 1) { current /= 1024; index++; }
    return `${current.toFixed(current < 10 && index > 0 ? 1 : 0)} ${units[index]}`;
  }
  if (unit === "us") return value >= 1000 ? `${(value / 1000).toFixed(1)} ms` : `${value.toFixed(value < 10 ? 1 : 0)} µs`;
  if (unit === "cores") return `${value.toFixed(2)} cores`;
  if (unit.endsWith("/s")) return `${value.toFixed(value < 10 ? 2 : 1)}/s`;
  return Math.round(value).toLocaleString();
};

const MetricStat = ({
  title,
  metric,
  operation,
  unit,
  icon,
  lookbackSeconds,
  step,
  component,
  filters,
  autoRefresh,
}: {
  title: string;
  metric: string;
  operation: QueryOperation;
  unit: string;
  icon: React.ReactNode;
  lookbackSeconds: number;
  step: Duration;
  component?: ComponentSelector;
  filters?: AttributeFilter[];
  autoRefresh: boolean;
}) => {
  const query = useQuery({
    queryKey: ["visibility", "metricStat", metric, JSON.stringify(operation), JSON.stringify(component), JSON.stringify(filters), lookbackSeconds, JSON.stringify(step)],
    queryFn: async () => {
      const now = new Date();
      const request = QueryMetricsRequest.create({
        metric: MetricSelector.create({ selector: { oneofKind: "name", name: metric } }),
        timeRange: TimeRange.create({ from: Timestamp.fromDate(new Date(now.getTime() - lookbackSeconds * 1000)), to: Timestamp.fromDate(now) }),
        step,
        component,
        filters,
        operation,
        seriesAggregation: aggregation(operation),
        limitSeries: 1,
        limitPointsPerSeries: operation.type.oneofKind === "counter" && operation.type.counter.function === CounterOperation_Function.INCREASE ? 500 : 2,
        limitBehavior: QueryMetricsRequest_LimitBehavior.TRUNCATE,
      });
      return (await getClientVisibilityMetrics().queryMetrics(request)).response;
    },
    refetchInterval: autoRefresh ? refetchIntervalChart : false,
    refetchIntervalInBackground: false,
  });
  const value = React.useMemo(() => {
    const points = query.data?.series[0]?.points;
    if (points?.oneofKind !== "number") return undefined;
    const values = points.number.points.map(numericValue).filter((item): item is number => item !== undefined);
    if (operation.type.oneofKind === "counter" && operation.type.counter.function === CounterOperation_Function.INCREASE) {
      return values.reduce((sum, item) => sum + item, 0);
    }
    return values.at(-1);
  }, [query.data, operation]);

  return (
    <div className="rounded-xl border border-slate-200/80 bg-white p-3.5 shadow-[0_1px_2px_rgba(15,23,42,0.03)]">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-[0.64rem] font-bold uppercase tracking-[0.07em] text-slate-400">{title}</div>
          <div className="mt-1.5 text-xl font-bold tabular-nums tracking-tight text-slate-800">{query.isLoading ? <span className="inline-block h-5 w-20 animate-pulse rounded bg-slate-100" /> : formatValue(value, unit)}</div>
        </div>
        <span className="rounded-lg bg-slate-100 p-2 text-slate-500">{icon}</span>
      </div>
      {query.isError && <div className="mt-2 text-[0.65rem] font-semibold text-red-500">Metric unavailable</div>}
    </div>
  );
};

const Metrics = () => {
  const [view, setView] = React.useState<View>("overview");
  const [range, setRange] = React.useState<Range>("6h");
  const [resolution, setResolution] = React.useState<Resolution>("auto");
  const [componentType, setComponentType] = React.useState<string | null>(null);
  const [serviceName, setServiceName] = React.useState<string>();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [runtimeExpanded, setRuntimeExpanded] = React.useState(false);
  const [visible, setVisible] = React.useState(() => document.visibilityState === "visible");
  const queryClient = useQueryClient();

  React.useEffect(() => {
    const update = () => setVisible(document.visibilityState === "visible");
    document.addEventListener("visibilitychange", update);
    return () => document.removeEventListener("visibilitychange", update);
  }, []);

  const lookbackSeconds = ranges[range];
  const step = React.useMemo(() => duration(resolution, range), [resolution, range]);
  const refresh = autoRefresh && visible;
  const selectedComponent = componentType ? ComponentSelector.create({ type: componentType }) : undefined;
  const vigil = ComponentSelector.create({ type: "vigil" });
  const octovigil = ComponentSelector.create({ type: "octovigil" });
  const rscserver = ComponentSelector.create({ type: "rscserver" });
  const serviceFilters = serviceName ? [eqFilter("octelium.vigil.svc.name", serviceName)] : undefined;
  const shared = { lookbackSeconds, step, autoRefresh: refresh, hideResolution: true, limitPointsPerSeries: 500, height: 250 };

  return (
    <div className="space-y-4 pb-8">
      <header className="rounded-xl border border-slate-200/80 bg-white p-3.5 sm:p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-lg font-bold tracking-tight text-slate-900">Cluster metrics</h1>
            <p className="mt-1 text-xs font-medium text-slate-500">Traffic, authorization decisions, and component runtime health</p>
          </div>
          <div className="flex items-center gap-3">
            <Switch size="sm" label="Auto-refresh" checked={autoRefresh} onChange={(event) => setAutoRefresh(event.currentTarget.checked)} />
            <Tooltip label="Refresh visible metrics"><ActionIcon type="button" variant="light" color="gray" onClick={() => { queryClient.invalidateQueries({ queryKey: ["visibility", "queryMetrics"] }); queryClient.invalidateQueries({ queryKey: ["visibility", "metricStat"] }); }} aria-label="Refresh metrics"><RefreshCw size={15} /></ActionIcon></Tooltip>
          </div>
        </div>
        <div className="mt-4 grid gap-3 border-t border-slate-100 pt-4 sm:grid-cols-2 xl:grid-cols-[auto_auto_minmax(180px,1fr)_minmax(220px,1.2fr)]">
          <div><div className="mb-1 text-[0.6rem] font-bold uppercase tracking-wide text-slate-400">Time range</div><SegmentedControl fullWidth value={range} onChange={(value) => setRange(value as Range)} data={["15m", "1h", "6h", "24h"]} /></div>
          <div><div className="mb-1 text-[0.6rem] font-bold uppercase tracking-wide text-slate-400">Resolution</div><Select value={resolution} onChange={(value) => setResolution((value ?? "auto") as Resolution)} data={[{ value: "auto", label: "Auto" }, { value: "10s", label: "10 seconds" }, { value: "30s", label: "30 seconds" }, { value: "1m", label: "1 minute" }, { value: "5m", label: "5 minutes" }]} /></div>
          <Select label="Component type" placeholder="All components" clearable searchable value={componentType} onChange={setComponentType} data={["vigil", "octovigil", "rscserver", "nocturne", "apiserver", "gateway", "portal"]} />
          <SelectResource api="core" kind="Service" label="Vigil service" clearable defaultValue={serviceName} onChange={(item) => setServiceName(item?.metadata?.name)} />
        </div>
      </header>

      <div className="overflow-x-auto pb-1"><SegmentedControl value={view} onChange={(value) => setView(value as View)} data={[{ value: "overview", label: "Overview" }, { value: "traffic", label: "Service traffic" }, { value: "authorization", label: "Authorization" }, { value: "components", label: "Components" }]} /></div>

      {view === "overview" && <>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <MetricStat title="Request rate" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} unit="requests/s" icon={<Activity size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={selectedComponent} autoRefresh={refresh} />
          <MetricStat title="Active requests" metric="req.active" operation={gaugeOp(GaugeOperation_Function.LAST)} unit="requests" icon={<Gauge size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={selectedComponent} autoRefresh={refresh} />
          <MetricStat title="Authorization p95" metric="authorization.req.duration" operation={histogramOp(HistogramOperation_Function.QUANTILE, [0.95])} unit="us" icon={<ShieldCheck size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={octovigil} autoRefresh={refresh} />
          <MetricStat title="CPU usage" metric="process.cpu.seconds" operation={counterOp(CounterOperation_Function.RATE)} unit="cores" icon={<Cpu size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={selectedComponent} autoRefresh={refresh} />
        </div>
        <div className="grid gap-4 xl:grid-cols-2">
          <MetricChart title="Request rate by component" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} groupBy={["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Active requests by component" unit="requests" metric="req.active" operation={gaugeOp(GaugeOperation_Function.LAST)} groupBy={["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="CPU usage by component" unit="cores" metric="process.cpu.seconds" operation={counterOp(CounterOperation_Function.RATE)} groupBy={["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Heap allocation by component" unit="bytes" metric="process.mem.heap_alloc" operation={gaugeOp(GaugeOperation_Function.LAST)} groupBy={["octelium.component.type"]} limitSeries={12} {...shared} />
        </div>
      </>}

      {view === "traffic" && <>
        <div className="flex items-center gap-2 rounded-xl border border-blue-100 bg-blue-50/60 px-3.5 py-3 text-xs font-semibold text-blue-800"><Waypoints size={15} />Vigil traffic{serviceName ? ` for ${serviceName}` : " across all services"}</div>
        <div className="grid gap-4 xl:grid-cols-2">
          <MetricChart title="Request rate" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} component={vigil} filters={serviceFilters} {...shared} />
          <MetricChart title="Active requests" unit="requests" metric="req.active" operation={gaugeOp(GaugeOperation_Function.LAST)} component={vigil} filters={serviceFilters} {...shared} />
          <MetricChart title="Request latency" unit="ms" metric="req.duration" operation={histogramOp(HistogramOperation_Function.QUANTILE, [0.5, 0.95, 0.99])} component={vigil} filters={serviceFilters} limitSeries={6} {...shared} />
          <MetricChart title="Request rate by service" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} component={vigil} filters={serviceFilters} groupBy={["octelium.vigil.svc.name"]} limitSeries={10} {...shared} />
          <MetricChart title="Request rate by region" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} component={vigil} filters={serviceFilters} groupBy={["octelium.vigil.svc.region.name"]} limitSeries={10} {...shared} />
          <MetricChart title="Request rate by service mode" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} component={vigil} filters={serviceFilters} groupBy={["octelium.vigil.svc.mode"]} limitSeries={10} {...shared} />
        </div>
      </>}

      {view === "authorization" && <>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <MetricStat title="Authorization rate" metric="authorization.req.total" operation={counterOp(CounterOperation_Function.RATE)} unit="requests/s" icon={<ShieldCheck size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={octovigil} autoRefresh={refresh} />
          <MetricStat title="Active authorizations" metric="authorization.req.active" operation={gaugeOp(GaugeOperation_Function.LAST)} unit="requests" icon={<Activity size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={octovigil} autoRefresh={refresh} />
          <MetricStat title="p95 latency" metric="authorization.req.duration" operation={histogramOp(HistogramOperation_Function.QUANTILE, [0.95])} unit="us" icon={<Clock3 size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={octovigil} autoRefresh={refresh} />
          <MetricStat title="Decisions in range" metric="authorization.req.total" operation={counterOp(CounterOperation_Function.INCREASE)} unit="requests" icon={<ServerCog size={16} />} lookbackSeconds={lookbackSeconds} step={step} component={octovigil} autoRefresh={refresh} />
        </div>
        <div className="grid gap-4 xl:grid-cols-2">
          <MetricChart title="Authorization rate" unit="requests/s" metric="authorization.req.total" operation={counterOp(CounterOperation_Function.RATE)} component={octovigil} {...shared} />
          <MetricChart title="Active authorization requests" unit="requests" metric="authorization.req.active" operation={gaugeOp(GaugeOperation_Function.LAST)} component={octovigil} {...shared} />
          <MetricChart title="Authorization latency" unit="us" metric="authorization.req.duration" operation={histogramOp(HistogramOperation_Function.QUANTILE, [0.5, 0.75, 0.95, 0.99])} component={octovigil} limitSeries={6} {...shared} />
          <MetricChart title="Authorization rate by replica" unit="requests/s" metric="authorization.req.total" kind={MetricDescriptor_Kind.COUNTER} operation={counterOp(CounterOperation_Function.RATE)} component={octovigil} groupBy={["octelium.component.name"]} limitSeries={10} {...shared} />
        </div>
      </>}

      {view === "components" && <>
        <div className="grid gap-4 xl:grid-cols-2">
          <MetricChart title="CPU usage" unit="cores" metric="process.cpu.seconds" operation={counterOp(CounterOperation_Function.RATE)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Heap allocated" unit="bytes" metric="process.mem.heap_alloc" operation={gaugeOp(GaugeOperation_Function.LAST)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Total runtime memory" unit="bytes" metric="process.mem.total" operation={gaugeOp(GaugeOperation_Function.LAST)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Goroutines" unit="goroutines" metric="process.goroutines" operation={gaugeOp(GaugeOperation_Function.LAST)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
          <MetricChart title="Resource API request rate" unit="requests/s" metric="req.total" operation={counterOp(CounterOperation_Function.RATE)} component={rscserver} groupBy={["octelium.component.name"]} limitSeries={10} {...shared} />
          <MetricChart title="Resource API latency" unit="us" metric="req.duration" operation={histogramOp(HistogramOperation_Function.QUANTILE, [0.5, 0.95, 0.99])} component={rscserver} limitSeries={6} {...shared} />
        </div>
        <section className="rounded-xl border border-slate-200/80 bg-white p-4">
          <button type="button" className="flex w-full items-center justify-between text-left text-sm font-bold text-slate-800" aria-expanded={runtimeExpanded} onClick={() => setRuntimeExpanded((value) => !value)}>Runtime details<ChevronDown size={16} className={`transition-transform duration-500 ${runtimeExpanded ? "rotate-180" : ""}`} /></button>
          {runtimeExpanded && <div className="mt-3 grid gap-4 border-t border-slate-100 pt-3 xl:grid-cols-2">
            <MetricChart title="Stack memory" unit="bytes" metric="process.mem.stacks" operation={gaugeOp(GaugeOperation_Function.LAST)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
            <MetricChart title="GC cycles" unit="gc-cycles/s" metric="process.gc.cycles" operation={counterOp(CounterOperation_Function.RATE)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
            <MetricChart title="GC heap goal" unit="bytes" metric="process.gc.heap_goal" operation={gaugeOp(GaugeOperation_Function.LAST)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
            <MetricChart title="GC CPU usage" unit="cores" metric="process.cpu.gc_seconds" operation={counterOp(CounterOperation_Function.RATE)} component={selectedComponent} groupBy={componentType ? ["octelium.component.name"] : ["octelium.component.type"]} limitSeries={12} {...shared} />
          </div>}
        </section>
      </>}
    </div>
  );
};

export default Metrics;
