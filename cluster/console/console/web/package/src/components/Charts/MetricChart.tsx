import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  Attribute,
  AttributeFilter,
  AttributeFilter_Operator,
  AttributeValue,
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricDescriptor_Kind,
  MetricSelector,
  NumberPoint,
  QueryMetricsRequest,
  QueryMetricsRequest_LimitBehavior,
  QueryMetricsRequest_SeriesAggregation,
  QueryOperation,
  TimeRange,
  TimeSeries,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import {
  getClientVisibilityMetrics,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import ReactEChartsCore from "echarts-for-react";
import { LineChart as LineChartC } from "echarts/charts";
import {
  AriaComponent,
  DataZoomComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useMemo, useState } from "react";
import DurationPicker from "../DurationPicker";

echarts.use([
  LineChartC,
  AriaComponent,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  CanvasRenderer,
]);

const LINE_COLORS = [
  "#2563eb",
  "#0891b2",
  "#4f46e5",
  "#0d9488",
  "#7c3aed",
  "#16a34a",
  "#d97706",
  "#db2777",
  "#475569",
  "#dc2626",
];

const DEFAULT_STEP = Duration.create({
  type: { oneofKind: "minutes", minutes: 1 },
});

export const counterOp = (fn: CounterOperation_Function): QueryOperation =>
  QueryOperation.create({
    type: { oneofKind: "counter", counter: { function: fn } },
  });

export const gaugeOp = (fn: GaugeOperation_Function): QueryOperation =>
  QueryOperation.create({
    type: { oneofKind: "gauge", gauge: { function: fn } },
  });

export const histogramOp = (
  fn: HistogramOperation_Function,
  quantiles: number[] = [],
): QueryOperation =>
  QueryOperation.create({
    type: { oneofKind: "histogram", histogram: { function: fn, quantiles } },
  });

export const eqFilter = (key: string, value: string): AttributeFilter =>
  AttributeFilter.create({
    key,
    operator: AttributeFilter_Operator.EQ,
    value: AttributeValue.create({
      value: { oneofKind: "stringValue", stringValue: value },
    }),
  });

export const eqBoolFilter = (key: string, value: boolean): AttributeFilter =>
  AttributeFilter.create({
    key,
    operator: AttributeFilter_Operator.EQ,
    value: AttributeValue.create({
      value: { oneofKind: "boolValue", boolValue: value },
    }),
  });

type ChartSeries = {
  name: string;
  data: [number, number | null][];
};

const NON_RETRYABLE_METRIC_CODES = new Set([
  "ABORTED",
  "CANCELLED",
  "CANCELED",
  "FAILED_PRECONDITION",
  "INVALID_ARGUMENT",
  "NOT_FOUND",
  "RESOURCE_EXHAUSTED",
]);

export const metricErrorCode = (error: unknown): string | undefined =>
  (error as { code?: string })?.code?.toUpperCase();

export const isMetricNotRecorded = (error: unknown): boolean =>
  metricErrorCode(error) === "NOT_FOUND";

export const retryMetricQuery = (failureCount: number, error: unknown) => {
  const code = metricErrorCode(error);
  return failureCount < 2 && (!code || !NON_RETRYABLE_METRIC_CODES.has(code));
};

export type MetricChartProps = {
  title?: string;
  unit?: string;
  metric: string;
  kind?: MetricDescriptor_Kind;
  operation: QueryOperation;
  component?: ComponentSelector;
  filters?: AttributeFilter[];
  groupBy?: string[];
  from?: Timestamp;
  to?: Timestamp;
  lookbackSeconds?: number;
  step?: Duration;
  limitSeries?: number;
  limitPointsPerSeries?: number;
  height?: number;
  enabled?: boolean;
  autoRefresh?: boolean;
  hideResolution?: boolean;
};

const numberValue = (p: NumberPoint): number | undefined => {
  if (!p.value) return undefined;

  switch (p.value.oneofKind) {
    case "asDouble":
      return p.value.asDouble;
    case "asInt":
      return Number(p.value.asInt);
    default:
      return undefined;
  }
};

const tsToMillis = (t?: Timestamp): number | undefined => {
  if (!t) return undefined;

  const seconds = Number(t.seconds);
  const nanos = Number(t.nanos ?? 0);

  if (!Number.isFinite(seconds) || !Number.isFinite(nanos)) {
    return undefined;
  }

  return seconds * 1000 + Math.floor(nanos / 1e6);
};

const timestampKey = (t?: Timestamp): string | undefined => {
  if (!t) return undefined;
  return `${String(t.seconds)}.${String(t.nanos ?? 0)}`;
};

const durationKey = (d?: Duration): string | undefined => {
  if (!d) return undefined;
  return JSON.stringify(d);
};

export const durationToMillis = (duration?: Duration): number | undefined => {
  switch (duration?.type?.oneofKind) {
    case "milliseconds":
      return duration.type.milliseconds;
    case "seconds":
      return duration.type.seconds * 1_000;
    case "minutes":
      return duration.type.minutes * 60_000;
    case "hours":
      return duration.type.hours * 3_600_000;
    case "days":
      return duration.type.days * 86_400_000;
    case "weeks":
      return duration.type.weeks * 604_800_000;
    default:
      return undefined;
  }
};

const normalizeMetricStep = (duration: Duration, rangeMillis: number) => {
  const current = durationToMillis(duration);
  const minimum = Math.max(1_000, Math.ceil(rangeMillis / 5_000));
  if (current !== undefined && current >= minimum) return duration;
  return Duration.create({
    type: {
      oneofKind: "seconds",
      seconds: Math.ceil(minimum / 1_000),
    },
  });
};

const operationKey = (op: QueryOperation): string => JSON.stringify(op);

const componentKey = (component?: ComponentSelector): string | undefined =>
  component ? JSON.stringify(component) : undefined;

const filtersKey = (filters?: AttributeFilter[]): string | undefined =>
  filters ? JSON.stringify(filters) : undefined;

const groupByKey = (groupBy?: string[]): string | undefined =>
  groupBy ? JSON.stringify(groupBy) : undefined;

const attributeValue = (value?: AttributeValue): string => {
  if (!value?.value) return "";

  switch (value.value.oneofKind) {
    case "stringValue":
      return value.value.stringValue;
    case "boolValue":
      return String(value.value.boolValue);
    case "intValue":
      return String(value.value.intValue);
    case "doubleValue":
      return String(value.value.doubleValue);
    default:
      return "";
  }
};

const attributeSortKey = (value?: AttributeValue): string => {
  if (!value?.value) return "";

  return `${value.value.oneofKind}:${attributeValue(value)}`;
};

const quantilePart = (quantile: number): string =>
  `p${Math.round(quantile * 100)}`;

const labelPart = (label: Attribute): string => {
  const shortKey = label.key.split(".").at(-1) || label.key;
  const rawValue = attributeValue(label.value);
  const value = label.key === "reason"
    ? rawValue.toLowerCase().replaceAll("_", " ").replace(/^./, (character) => character.toUpperCase())
    : rawValue;
  return `${shortKey}=${value}`;
};

const seriesLabel = (
  labels: Attribute[] | undefined,
  fallback: string,
  quantile?: number,
): string => {
  const parts = [...(labels ?? [])]
    .sort((a, b) => {
      if (a.key === b.key) {
        return attributeSortKey(a.value).localeCompare(
          attributeSortKey(b.value),
        );
      }
      return a.key.localeCompare(b.key);
    })
    .map(labelPart);

  if (quantile !== undefined) {
    parts.push(quantilePart(quantile));
  }

  return parts.length > 0 ? parts.join(" · ") : fallback;
};

const isRawCounterOperation = (operation: QueryOperation): boolean =>
  operation.type?.oneofKind === "counter" &&
  operation.type.counter.function === CounterOperation_Function.RAW;

const seriesAggregation = (
  operation: QueryOperation,
): QueryMetricsRequest_SeriesAggregation => {
  switch (operation.type?.oneofKind) {
    case "counter":
      return operation.type.counter.function === CounterOperation_Function.RAW
        ? QueryMetricsRequest_SeriesAggregation.NONE
        : QueryMetricsRequest_SeriesAggregation.SUM;

    case "gauge":
      switch (operation.type.gauge.function) {
        case GaugeOperation_Function.AVG:
          return QueryMetricsRequest_SeriesAggregation.AVG;
        case GaugeOperation_Function.MIN:
          return QueryMetricsRequest_SeriesAggregation.MIN;
        case GaugeOperation_Function.MAX:
          return QueryMetricsRequest_SeriesAggregation.MAX;
        case GaugeOperation_Function.LAST:
        case GaugeOperation_Function.SUM:
        default:
          return QueryMetricsRequest_SeriesAggregation.SUM;
      }

    case "histogram":
      return QueryMetricsRequest_SeriesAggregation.MERGE;

    default:
      return QueryMetricsRequest_SeriesAggregation.SERIES_AGGREGATION_UNSET;
  }
};

const formatBytes = (v: number): string => {
  if (!Number.isFinite(v)) return "—";
  if (v === 0) return "0 B";

  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let n = v;
  let i = 0;

  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }

  return `${n.toFixed(n < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
};

const escapeHTML = (value: string) =>
  value.replace(
    /[&<>'"]/g,
    (character) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#39;",
        '"': "&quot;",
      })[character] ?? character,
  );

const formatValue = (v: number, unit?: string): string => {
  if (!Number.isFinite(v)) return "—";

  switch (unit) {
    case "bytes":
      return formatBytes(v);

    case "bytes/s":
      return `${formatBytes(v)}/s`;

    case "ms":
      return `${v.toFixed(v < 10 ? 1 : 0)} ms`;

    case "us":
      return `${v.toFixed(v < 10 ? 1 : 0)} µs`;

    case "seconds":
    case "s":
      return `${v.toFixed(v < 10 ? 2 : 1)} s`;

    case "cores":
      return v.toFixed(2);

    case "requests/s":
    case "cycles/s":
    case "gc-cycles/s":
      return `${v.toFixed(v < 10 ? 2 : 1)}/s`;

    case "requests":
    case "goroutines":
    case "gc-cycles":
      return Math.round(v).toLocaleString();

    default:
      if (Number.isInteger(v)) return v.toLocaleString();
      return Math.abs(v) >= 1000
        ? Math.round(v).toLocaleString()
        : v.toFixed(2);
  }
};

const extractSeries = (
  series: TimeSeries[] | undefined,
  fallback: string,
  stepMillis?: number,
): ChartSeries[] => {
  if (!series) return [];

  return series
    .map((s) => {
      if (!s.points || s.points.oneofKind !== "number") {
        return null;
      }

      const points = s.points.number.points
        .map((p): [number, number] | null => {
          const ts = tsToMillis(p.timestamp);
          const val = numberValue(p);

          if (ts === undefined || val === undefined || !Number.isFinite(val)) {
            return null;
          }

          return [ts, val];
        })
        .filter((p): p is [number, number] => p !== null)
        .sort((a, b) => a[0] - b[0]);

      if (points.length === 0) {
        return null;
      }

      const data: [number, number | null][] = [];
      for (const point of points) {
        const previous = data.at(-1);
        if (
          stepMillis &&
          previous &&
          point[0] - previous[0] > stepMillis * 1.5
        ) {
          data.push([previous[0] + stepMillis, null]);
        }
        data.push(point);
      }

      return {
        name: seriesLabel(s.labels, fallback, s.quantile),
        data,
      };
    })
    .filter((x): x is ChartSeries => x !== null);
};

const MetricChart = (props: MetricChartProps) => {
  const {
    title,
    unit,
    metric,
    kind,
    operation,
    component,
    filters,
    groupBy,
    limitSeries,
    limitPointsPerSeries,
    height = 240,
    enabled = true,
    autoRefresh = true,
    hideResolution = false,
  } = props;

  const [step, setStep] = useState(props.step);

  useEffect(() => {
    setStep(props.step);
  }, [durationKey(props.step)]);

  const supportsStep = !isRawCounterOperation(operation);
  const explicitRange =
    tsToMillis(props.from) !== undefined && tsToMillis(props.to) !== undefined
      ? tsToMillis(props.to)! - tsToMillis(props.from)!
      : undefined;
  const rangeMillis = Math.max(
    1_000,
    explicitRange ?? (props.lookbackSeconds ?? 6 * 3_600) * 1_000,
  );
  const effectiveStep = useMemo(
    () =>
      supportsStep
        ? normalizeMetricStep(step ?? DEFAULT_STEP, rangeMillis)
        : undefined,
    [supportsStep, step, rangeMillis],
  );

  const queryKey = useMemo(
    () => [
      "visibility",
      "queryMetrics",
      metric,
      kind,
      operationKey(operation),
      componentKey(component),
      filtersKey(filters),
      groupByKey(groupBy),
      durationKey(effectiveStep),
      timestampKey(props.from),
      timestampKey(props.to),
      props.lookbackSeconds ?? 6 * 3600,
      limitSeries,
      limitPointsPerSeries,
    ],
    [
      metric,
      kind,
      operation,
      component,
      filters,
      groupBy,
      effectiveStep,
      props.from,
      props.to,
      props.lookbackSeconds,
      limitSeries,
      limitPointsPerSeries,
    ],
  );

  const qry = useQuery({
    queryKey,
    enabled,

    queryFn: async ({ signal }) => {
      const now = new Date();

      const from =
        props.from ??
        Timestamp.fromDate(
          new Date(now.getTime() - (props.lookbackSeconds ?? 6 * 3600) * 1000),
        );

      const to = props.to ?? Timestamp.fromDate(now);

      const req = QueryMetricsRequest.create({
        metric: MetricSelector.create({
          selector: { oneofKind: "name", name: metric },
          kind: kind ?? MetricDescriptor_Kind.KIND_UNSET,
        }),
        timeRange: TimeRange.create({ from, to }),
        step: effectiveStep,
        component,
        filters,
        groupBy,
        operation,
        limitSeries,
        limitPointsPerSeries,
        seriesAggregation: seriesAggregation(operation),
        limitBehavior: QueryMetricsRequest_LimitBehavior.TRUNCATE,
      });

      const { response } = await getClientVisibilityMetrics().queryMetrics(req, {
        abort: signal,
      });
      return response;
    },

    refetchInterval: autoRefresh ? refetchIntervalChart : false,
    refetchIntervalInBackground: false,
    retry: retryMetricQuery,
  });

  const effectiveUnit =
    unit ?? qry.data?.result?.unit ?? qry.data?.sourceDescriptor?.unit;

  const series = useMemo(
    () =>
      extractSeries(
        qry.data?.series,
        title ?? metric,
        durationToMillis(qry.data?.step ?? effectiveStep),
      ),
    [qry.data?.series, qry.data?.step, effectiveStep, title, metric],
  );

  const chartRange = useMemo(() => {
    const dataTimestamps = series
      .flatMap((item) => item.data)
      .map(([timestamp]) => timestamp);
    const dataFrom = dataTimestamps.length > 0
      ? Math.min(...dataTimestamps)
      : undefined;
    const dataTo = dataTimestamps.length > 0
      ? Math.max(...dataTimestamps)
      : undefined;
    const requestedTo =
      tsToMillis(props.to) ?? tsToMillis(qry.data?.snapshotTime);
    const requestedFrom =
      tsToMillis(props.from) ??
      (requestedTo !== undefined ? requestedTo - rangeMillis : undefined);

    return {
      from:
        requestedFrom === undefined
          ? dataFrom
          : dataFrom === undefined
            ? requestedFrom
            : Math.min(requestedFrom, dataFrom),
      to:
        requestedTo === undefined
          ? dataTo
          : dataTo === undefined
            ? requestedTo
            : Math.max(requestedTo, dataTo),
    };
  }, [series, props.from, props.to, qry.data?.snapshotTime, rangeMillis]);

  const statistics = useMemo(() => {
    const points = series
      .flatMap((item) => item.data)
      .filter((point): point is [number, number] => point[1] !== null);

    if (points.length === 0) {
      return { average: 0, latest: 0, peak: 0, pointCount: 0 };
    }

    const latestPoint = points.reduce((latest, point) =>
      point[0] > latest[0] ? point : latest,
    );
    const values = points.map(([, value]) => value);

    return {
      average: values.reduce((sum, value) => sum + value, 0) / values.length,
      latest: latestPoint[1],
      peak: Math.max(...values),
      pointCount: points.length,
    };
  }, [series]);

  const option = useMemo(() => {
    const multi = series.length > 1;

    return {
      color: LINE_COLORS,

      animation: statistics.pointCount <= 1_000,
      animationDuration: 650,
      animationDurationUpdate: 0,
      animationEasing: "cubicOut",

      aria: {
        enabled: true,
        decal: { show: false },
      },

      grid: {
        left: 10,
        right: 14,
        top: 20,
        bottom: multi ? 54 : 40,
        containLabel: true,
      },

      tooltip: {
        trigger: "axis",
        confine: true,
        backgroundColor: "#0f172a",
        borderColor: "#334155",
        borderWidth: 1,
        padding: [10, 12],
        textStyle: {
          color: "#f8fafc",
          fontSize: 12,
          fontFamily: "Ubuntu, sans-serif",
        },
        extraCssText:
          "border-radius:10px;box-shadow:0 10px 24px rgba(15,23,42,.18);max-width:320px;",
        axisPointer: {
          type: "line",
          snap: true,
          lineStyle: {
            color: "#60a5fa",
            width: 1,
            type: "dashed",
          },
          label: { show: false },
        },
        formatter: (params: unknown) => {
          const items = Array.isArray(params) ? params : [params];

          const first = items[0] as
            | { value?: [number, number]; axisValue?: number | string }
            | undefined;

          const ts =
            Array.isArray(first?.value) && typeof first.value[0] === "number"
              ? first.value[0]
              : typeof first?.axisValue === "number"
                ? first.axisValue
                : undefined;

          const header =
            ts !== undefined
              ? `<div style="margin-bottom:8px;color:#cbd5e1;font-size:11px;font-weight:700">${new Date(ts).toLocaleString([], {
                  month: "short",
                  day: "numeric",
                  hour: "2-digit",
                  minute: "2-digit",
                  second: "2-digit",
                })}</div>`
              : "";

          const body = items
            .map((item) => {
              const p = item as {
                marker?: string;
                seriesName?: string;
                value?: [number, number] | number;
              };

              const rawValue = Array.isArray(p.value) ? p.value[1] : p.value;
              const value =
                typeof rawValue === "number"
                  ? formatValue(rawValue, effectiveUnit)
                  : "—";

              return `<div style="display:flex;align-items:center;justify-content:space-between;gap:18px;margin-top:5px"><span style="min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#cbd5e1;font-size:11px;font-weight:600">${p.marker ?? ""}${escapeHTML(p.seriesName ?? "Metric")}</span><strong style="flex-shrink:0;color:#f8fafc;font-size:12px">${value}</strong></div>`;
            })
            .join("");

          return header + body;
        },
      },

      legend: multi
        ? {
            bottom: 0,
            icon: "circle",
            itemWidth: 8,
            itemHeight: 8,
            type: "scroll",
            pageIconColor: "#64748b",
            pageIconInactiveColor: "#cbd5e1",
            pageTextStyle: {
              color: "#94a3b8",
              fontFamily: "Ubuntu, sans-serif",
              fontSize: 10,
            },
            textStyle: {
              color: "#64748b",
              fontSize: 11,
              fontWeight: 600,
              fontFamily: "Ubuntu, sans-serif",
            },
          }
        : { show: false },

      xAxis: {
        type: "time",
        min: chartRange.from,
        max: chartRange.to,
        axisLine: { lineStyle: { color: "#e2e8f0" } },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          fontWeight: 600,
          lineHeight: 16,
          margin: 12,
          hideOverlap: true,
          formatter: (value: number) =>
            echarts.format.formatTime("MM/dd\nHH:mm", value),
        },
      },

      yAxis: {
        type: "value",
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: {
          lineStyle: { color: "#e2e8f0", type: "dashed", opacity: 0.7 },
        },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          fontWeight: 600,
          margin: 12,
          formatter: (v: number) => formatValue(v, effectiveUnit),
        },
      },

      dataZoom: [
        {
          type: "inside",
          zoomOnMouseWheel: "shift",
          moveOnMouseWheel: false,
          moveOnMouseMove: true,
        },
      ],

      series: series.map((s, i) => ({
        name: s.name,
        type: "line",
        step: supportsStep ? "start" : false,
        showSymbol: s.data.length <= 12,
        symbol: "circle",
        symbolSize: 6,
        sampling: "lttb",
        progressive: 400,
        progressiveThreshold: 800,
        lineStyle: {
          color: LINE_COLORS[i % LINE_COLORS.length],
          width: multi ? 2 : 2.5,
          cap: "round",
          join: "round",
        },
        emphasis: { focus: "series" },
        connectNulls: false,
        areaStyle:
          multi || series.length === 0
            ? undefined
            : {
                color: {
                  type: "linear",
                  x: 0,
                  y: 0,
                  x2: 0,
                  y2: 1,
                  colorStops: [
                    { offset: 0, color: "rgba(37,99,235,0.20)" },
                    { offset: 0.65, color: "rgba(37,99,235,0.05)" },
                    { offset: 1, color: "rgba(37,99,235,0)" },
                  ],
                },
              },
        data: s.data,
        itemStyle: {
          color: LINE_COLORS[i % LINE_COLORS.length],
          borderColor: "#ffffff",
          borderWidth: 2,
        },
      })),
    };
  }, [series, chartRange, effectiveUnit, statistics.pointCount, supportsStep]);

  const headerStatistics =
    series.length > 1
      ? [
          { label: "Series", value: series.length.toLocaleString() },
          { label: "Points", value: statistics.pointCount.toLocaleString() },
          { label: "Peak", value: formatValue(statistics.peak, effectiveUnit) },
        ]
      : [
          { label: "Latest", value: formatValue(statistics.latest, effectiveUnit) },
          {
            label: "Average",
            value: formatValue(statistics.average, effectiveUnit),
          },
          { label: "Peak", value: formatValue(statistics.peak, effectiveUnit) },
        ];

  const errorMessage =
    qry.error instanceof Error ? qry.error.message : "Unable to load this metric";

  const notRecorded = qry.isError && isMetricNotRecorded(qry.error);
  const failed = qry.isError && !notRecorded;

  return (
    <section
      className="my-3 flex w-full flex-col rounded-xl border border-slate-200/80 bg-white p-3.5 sm:p-4"
      aria-label={title ?? metric}
    >
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 pb-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-bold tracking-tight text-slate-800">
              {title ?? metric}
            </h3>
            {(qry.data?.truncation?.seriesTruncated ||
              qry.data?.truncation?.pointsTruncated) && (
              <span className="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[0.6rem] font-bold uppercase tracking-[0.06em] text-amber-700">
                Partial data
              </span>
            )}
            {qry.isFetching && !qry.isLoading && (
              <span className="flex items-center gap-1.5 text-[0.65rem] font-semibold text-slate-400">
                <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-500" />
                Updating
              </span>
            )}
            {failed && series.length > 0 && (
              <span className="flex items-center gap-1.5 text-[0.65rem] font-semibold text-amber-600">
                <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />
                Refresh failed · showing previous data
              </span>
            )}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-[0.65rem] font-semibold text-slate-400">
            <span>{metric}</span>
            {effectiveUnit && (
              <span className="rounded-md bg-slate-100 px-1.5 py-0.5 text-slate-500">
                {effectiveUnit}
              </span>
            )}
          </div>
        </div>

        {supportsStep && !hideResolution && (
          <div className="w-36 shrink-0">
            <span className="mb-1 block text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Resolution
            </span>
            <DurationPicker
              value={effectiveStep}
              placeholder="Resolution"
              onChange={setStep}
            />
          </div>
        )}
      </header>

      {series.length > 0 && (
        <>
          <dl className="flex flex-wrap items-center justify-end gap-4 px-1 py-3 sm:gap-6">
            {headerStatistics.map((statistic) => (
              <div key={statistic.label} className="text-right">
                <dt className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  {statistic.label}
                </dt>
                <dd className="mt-0.5 text-xs font-bold tabular-nums text-slate-700">
                  {statistic.value}
                </dd>
              </div>
            ))}
          </dl>

          <div className="w-full rounded-xl border border-slate-200/80 bg-slate-50/60 px-1 pt-1 sm:px-2">
            <ReactEChartsCore
              echarts={echarts}
              option={option}
              style={{ height, width: "100%" }}
              notMerge
              lazyUpdate
            />
          </div>
          <p className="mt-2 px-1 text-right text-[0.6rem] font-semibold text-slate-400">
            Drag to pan · Shift + scroll to zoom
          </p>
        </>
      )}

      {qry.isLoading && series.length === 0 && (
        <div
          style={{ height }}
          className="flex w-full flex-col items-center justify-center gap-3 text-center"
        >
          <span className="h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600" />
          <span className="text-xs font-semibold text-slate-400">
            Loading metric data…
          </span>
        </div>
      )}

      {failed && series.length === 0 && (
        <div
          style={{ height }}
          className="flex w-full items-center justify-center px-4 text-center"
        >
          <div className="max-w-md">
            <p className="text-sm font-bold text-slate-700">
              Metric unavailable
            </p>
            <p className="mt-1 line-clamp-2 text-xs font-semibold text-slate-400">
              {errorMessage}
            </p>
            <button
              type="button"
              onClick={() => qry.refetch()}
              className="mt-3 rounded-lg bg-slate-900 px-3 py-2 text-xs font-bold text-white transition-colors duration-500 hover:bg-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-offset-2"
            >
              Try again
            </button>
          </div>
        </div>
      )}

      {!qry.isLoading && !failed && series.length === 0 && (
        <div
          style={{ height }}
          className="flex w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center"
        >
          <div>
            <p className="text-sm font-bold text-slate-600">
              {notRecorded ? "Metric not recorded" : "No metric data"}
            </p>
            <p className="mt-1 text-xs font-semibold text-slate-400">
              {notRecorded
                ? `No component has reported ${metric} to the metricstore yet.`
                : "No numeric samples were returned for this time range."}
            </p>
          </div>
        </div>
      )}
    </section>
  );
};

export default MetricChart;
