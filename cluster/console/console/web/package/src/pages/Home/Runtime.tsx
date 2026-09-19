import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricSelector,
  QueryMetricsRequest,
  QueryMetricsRequest_LimitBehavior,
  QueryMetricsRequest_SeriesAggregation,
  QueryOperation,
  TimeRange,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import SeriesChart from "@/components/Charts/SeriesChart";
import MetricChart, {
  counterOp,
  gaugeOp,
  histogramOp,
  POINTS_PER_SERIES_LIMIT,
  retryMetricQuery,
  TOTAL_POINTS_LIMIT,
} from "@/components/Charts/MetricChart";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { getClientVisibilityMetrics } from "@/utils/client";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity,
  Cpu,
  Gauge,
  ScrollText,
  ServerCog,
  ShieldCheck,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { MiniStat, Panel, toPoints } from "@/components/Dashboard/components";
import {
  useComponentDataPoint,
  useComponentSummary,
  useTopComponents,
} from "@/components/Dashboard/queries";
import {
  dashboardKeys,
  useAutoRefresh,
  useDashboardQuery,
} from "@/components/Dashboard/utils";

const OCTOVIGIL = ComponentSelector.create({
  type: "octovigil",
  namespace: "octelium",
});

const metricStep = (periodMinutes: number): Duration =>
  Duration.create({
    type: {
      oneofKind: "minutes",
      minutes: periodMinutes <= 60 ? 1 : periodMinutes <= 360 ? 5 : 15,
    },
  });

const formatValue = (value: number | undefined, unit: string): string => {
  if (value === undefined || !Number.isFinite(value)) return "—";
  if (unit === "us")
    return value >= 1000
      ? `${(value / 1000).toFixed(1)} ms`
      : `${value.toFixed(value < 10 ? 1 : 0)} µs`;
  if (unit === "cores") return `${value.toFixed(2)}`;
  if (unit.endsWith("/s")) return `${value.toFixed(value < 10 ? 2 : 1)}`;
  return Math.round(value).toLocaleString();
};

const numericValue = (point?: {
  value:
    | { oneofKind: "asDouble"; asDouble: number }
    | { oneofKind: "asInt"; asInt: number }
    | { oneofKind: undefined };
}) =>
  point?.value.oneofKind === "asDouble"
    ? point.value.asDouble
    : point?.value.oneofKind === "asInt"
      ? Number(point.value.asInt)
      : undefined;

const MetricStat = (props: {
  label: string;
  suffix: string;
  metric: string;
  operation: QueryOperation;
  unit: string;
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  component?: ComponentSelector;
  periodMinutes: number;
}) => {
  const Icon = props.icon;
  const step = metricStep(props.periodMinutes);

  const query = useDashboardQuery({
    queryKey: [
      ...dashboardKeys.metricStat(
        props.metric,
        props.periodMinutes,
        props.unit,
      ),
    ],
    priority: QUERY_PRIORITY.low,
    retry: retryMetricQuery,
    fetch: async (signal) => {
      const now = new Date();
      const request = QueryMetricsRequest.create({
        metric: MetricSelector.create({
          selector: { oneofKind: "name", name: props.metric },
        }),
        timeRange: TimeRange.create({
          from: Timestamp.fromDate(
            new Date(now.getTime() - props.periodMinutes * 60_000),
          ),
          to: Timestamp.fromDate(now),
        }),
        step,
        component: props.component,
        operation: props.operation,
        seriesAggregation:
          props.operation.type.oneofKind === "histogram"
            ? QueryMetricsRequest_SeriesAggregation.MERGE
            : props.operation.type.oneofKind === "gauge"
              ? QueryMetricsRequest_SeriesAggregation.SUM
              : QueryMetricsRequest_SeriesAggregation.SUM,
        limitSeries: 1,
        limitPointsPerSeries: POINTS_PER_SERIES_LIMIT,
        limitTotalPoints: TOTAL_POINTS_LIMIT,
        limitBehavior: QueryMetricsRequest_LimitBehavior.TRUNCATE,
      });
      return (
        await getClientVisibilityMetrics().queryMetrics(request, {
          abort: signal,
        })
      ).response;
    },
  });

  const value = React.useMemo(() => {
    const points = query.data?.series[0]?.points;
    if (points?.oneofKind !== "number") return undefined;
    return points.number.points
      .map(numericValue)
      .filter(Number.isFinite)
      .at(-1);
  }, [query.data]);

  return (
    <div className="flex min-w-0 flex-col gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3.5">
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {props.label}
        </span>
        <span className="shrink-0 text-slate-400">
          <Icon size={14} strokeWidth={2.2} />
        </span>
      </div>

      <div className="flex items-baseline gap-1.5">
        {query.isLoading ? (
          <span className="block h-6 w-20 animate-pulse rounded bg-slate-100" />
        ) : (
          <>
            <span className="text-2xl font-bold leading-none tabular-nums tracking-[-0.03em] text-slate-900">
              {formatValue(value, props.unit)}
            </span>
            <span className="text-micro font-normal text-slate-500">
              {props.suffix}
            </span>
          </>
        )}
      </div>

      {query.isError && (
        <span className="truncate text-micro font-semibold text-amber-700">
          Metric unavailable
        </span>
      )}
    </div>
  );
};

const Runtime = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();
  const autoRefresh = useAutoRefresh();
  const step = metricStep(periodMinutes);

  const componentSummary = useComponentSummary(
    periodMinutes,
    QUERY_PRIORITY.high,
  );
  const componentPoints = useComponentDataPoint(
    periodMinutes,
    "all",
    QUERY_PRIORITY.low,
  );
  const topComponents = useTopComponents(periodMinutes, QUERY_PRIORITY.low);

  const data = componentSummary.data;
  const levels = [
    { label: "Info", value: n(data?.totalInfo) },
    { label: "Debug", value: n(data?.totalDebug) },
    {
      label: "Warn",
      value: n(data?.totalWarn),
      color: STATUS_COLORS.warning,
    },
    {
      label: "Error",
      value: n(data?.totalError),
      color: STATUS_COLORS.critical,
    },
    {
      label: "Panic",
      value: n(data?.totalPanic),
      color: STATUS_COLORS.critical,
    },
    {
      label: "Fatal",
      value: n(data?.totalFatal),
      color: STATUS_COLORS.critical,
    },
  ];

  return (
    <Panel
      icon={ServerCog}
      title="Runtime & platform"
      description={`Live throughput, authorization latency and component logs over the last ${rangeLabel}`}
      to="/visibility/metrics"
      toLabel="Metrics"
    >
      <div className="flex flex-col gap-5">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <MetricStat
            label="Request rate"
            suffix="req/s"
            metric="req.total"
            operation={counterOp(CounterOperation_Function.RATE)}
            unit="requests/s"
            icon={Activity}
            periodMinutes={periodMinutes}
          />
          <MetricStat
            label="Active requests"
            suffix="in flight"
            metric="req.active"
            operation={gaugeOp(GaugeOperation_Function.LAST)}
            unit="requests"
            icon={Gauge}
            periodMinutes={periodMinutes}
          />
          <MetricStat
            label="Authorization p95"
            suffix="per decision"
            metric="authorization.req.duration"
            operation={histogramOp(
              HistogramOperation_Function.QUANTILE,
              [0.95],
            )}
            unit="us"
            icon={ShieldCheck}
            component={OCTOVIGIL}
            periodMinutes={periodMinutes}
          />
          <MetricStat
            label="CPU usage"
            suffix="cores"
            metric="process.cpu.seconds"
            operation={counterOp(CounterOperation_Function.RATE)}
            unit="cores"
            icon={Cpu}
            periodMinutes={periodMinutes}
          />
        </div>

        <MetricChart
          title="Request rate by component"
          unit="requests/s"
          metric="req.total"
          operation={counterOp(CounterOperation_Function.RATE)}
          groupBy={["octelium.component.namespace", "octelium.component.type"]}
          limitSeries={12}
          limitPointsPerSeries={POINTS_PER_SERIES_LIMIT}
          lookbackSeconds={periodMinutes * 60}
          step={step}
          height={280}
          autoRefresh={autoRefresh}
          hideResolution
        />

        <div className="flex flex-col gap-4 lg:flex-row">
          <div className="flex min-w-0 flex-1 flex-col gap-3">
            <div className="flex items-center justify-between gap-2">
              <span className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                <ScrollText size={12} strokeWidth={2.3} />
                Component logs
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {n(data?.totalNumber).toLocaleString()} entries
              </span>
            </div>

            <CompositionBar segments={levels} total={n(data?.totalNumber)} />

            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              <MiniStat
                label="Warnings"
                value={n(data?.totalWarn)}
                tone={n(data?.totalWarn) > 0 ? "warning" : "default"}
                to="/visibility/componentlogs?level=WARN"
              />
              <MiniStat
                label="Errors"
                value={n(data?.totalError)}
                tone={n(data?.totalError) > 0 ? "critical" : "default"}
                to="/visibility/componentlogs?level=ERROR"
              />
              <MiniStat
                label="Panics & fatals"
                value={n(data?.totalPanic) + n(data?.totalFatal)}
                tone={
                  n(data?.totalPanic) + n(data?.totalFatal) > 0
                    ? "critical"
                    : "default"
                }
                to="/visibility/componentlogs?level=PANIC"
              />
            </div>
          </div>

          <div className="flex min-w-0 flex-1 flex-col gap-3">
            <SeriesChart
              height={220}
              series={[
                {
                  name: "Component logs",
                  points: toPoints(componentPoints.data?.datapoints),
                },
              ]}
              emptyLabel={`No component logs in the last ${rangeLabel}`}
            />

            {(topComponents.data?.items ?? []).length > 0 && (
              <div className="flex flex-col gap-1.5">
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                  Busiest components
                </span>

                {(topComponents.data?.items ?? []).slice(0, 5).map((item) => (
                  <Link
                    key={`${item.component?.namespace}/${item.component?.type}`}
                    to={`/visibility/componentlogs?component.type=${item.component?.type ?? ""}`}
                    className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-1.5 outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400"
                  >
                    <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                      {item.component?.type}
                      <span className="ml-1.5 text-micro font-normal text-slate-500">
                        {item.component?.namespace}
                      </span>
                    </span>
                    {n(item.countError) +
                      n(item.countPanic) +
                      n(item.countFatal) >
                      0 && (
                      <span className="shrink-0 text-micro font-semibold tabular-nums text-red-700">
                        {n(item.countError) +
                          n(item.countPanic) +
                          n(item.countFatal)}
                      </span>
                    )}
                    <span className="shrink-0 text-micro font-semibold tabular-nums text-slate-700">
                      {n(item.count).toLocaleString()}
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </Panel>
  );
};

export default Runtime;
