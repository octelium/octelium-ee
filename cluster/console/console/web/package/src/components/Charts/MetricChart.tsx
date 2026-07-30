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
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  CanvasRenderer,
]);

const LINE_COLORS = [
  "#1d4ed8",
  "#16a34a",
  "#d97706",
  "#9333ea",
  "#0891b2",
  "#db2777",
  "#65a30d",
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

type ChartSeries = {
  name: string;
  data: [number, number][];
};

type MetricChartProps = {
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
  return `${shortKey}=${attributeValue(label.value)}`;
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
  if (v <= 0) return "0 B";

  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let n = v;
  let i = 0;

  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }

  return `${n.toFixed(n < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
};

const formatValue = (v: number, unit?: string): string => {
  if (!Number.isFinite(v)) return "—";

  switch (unit) {
    case "bytes":
      return formatBytes(v);

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
): ChartSeries[] => {
  if (!series) return [];

  return series
    .map((s) => {
      if (!s.points || s.points.oneofKind !== "number") {
        return null;
      }

      const data = s.points.number.points
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

      if (data.length === 0) {
        return null;
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
  } = props;

  const [step, setStep] = useState(props.step);

  useEffect(() => {
    setStep(props.step);
  }, [durationKey(props.step)]);

  const effectiveStep = isRawCounterOperation(operation)
    ? undefined
    : (step ?? DEFAULT_STEP);

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

    queryFn: async () => {
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
        seriesPageSize: limitSeries,
        limitPointsPerSeries,
        seriesAggregation: seriesAggregation(operation),
        limitBehavior: QueryMetricsRequest_LimitBehavior.TRUNCATE,
      });

      const { response } = await getClientVisibilityMetrics().queryMetrics(req);
      return response;
    },

    refetchInterval: refetchIntervalChart,
  });

  const effectiveUnit =
    unit ?? qry.data?.result?.unit ?? qry.data?.sourceDescriptor?.unit;

  const series = useMemo(
    () => extractSeries(qry.data?.series, title ?? metric),
    [qry.data?.series, title, metric],
  );

  const option = useMemo(() => {
    const multi = series.length > 1;

    return {
      color: LINE_COLORS,

      grid: {
        left: 8,
        right: 16,
        top: 16,
        bottom: multi ? 32 : 8,
        containLabel: true,
      },

      tooltip: {
        trigger: "axis",
        backgroundColor: "#1e293b",
        borderColor: "#334155",
        borderWidth: 1,
        textStyle: {
          color: "#f8fafc",
          fontSize: 12,
          fontFamily: "Ubuntu, sans-serif",
        },
        extraCssText: "border-radius:6px; padding:8px 12px;",
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
              ? `<div style="margin-bottom:4px;color:#cbd5e1">${new Date(ts).toLocaleString()}</div>`
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

              return `<div>${p.marker ?? ""}<span style="color:#f8fafc">${p.seriesName ?? ""}</span>: <b>${value}</b></div>`;
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
        axisLine: { lineStyle: { color: "#e2e8f0" } },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          hideOverlap: true,
        },
      },

      yAxis: {
        type: "value",
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: "#f1f5f9" } },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          formatter: (v: number) => formatValue(v, effectiveUnit),
        },
      },

      dataZoom: [{ type: "inside" }],

      series: series.map((s, i) => ({
        name: s.name,
        type: "line",
        showSymbol: false,
        smooth: 0.2,
        lineStyle: { width: 2 },
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
                    { offset: 0, color: "rgba(29,78,216,0.18)" },
                    { offset: 1, color: "rgba(29,78,216,0)" },
                  ],
                },
              },
        data: s.data,
        itemStyle: { color: LINE_COLORS[i % LINE_COLORS.length] },
      })),
    };
  }, [series, effectiveUnit]);

  const emptyMessage = useMemo(() => {
    if (qry.isLoading) return "Loading…";
    if (qry.isError) return "Failed to load metric";
    return "No numeric data";
  }, [qry.isLoading, qry.isError]);

  return (
    <div className="flex w-full flex-col">
      <div className="mb-1 flex items-end px-1">
        {title && (
          <p className="flex-1 text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800">
            {title}
          </p>
        )}

        {(qry.data?.truncation?.seriesTruncated ||
          qry.data?.truncation?.pointsTruncated ||
          Boolean(qry.data?.nextSeriesPageToken)) && (
          <span className="mr-2 rounded-full bg-amber-50 px-2 py-0.5 text-[0.65rem] font-bold uppercase tracking-[0.04em] text-amber-700">
            Truncated
          </span>
        )}

        <div className="ml-auto">
          <DurationPicker
            value={step}
            title="Resolution duration interval"
            onChange={(v) => setStep(v)}
          />
        </div>
      </div>

      {qry.isSuccess && series.length > 0 ? (
        <ReactEChartsCore
          echarts={echarts}
          option={option}
          style={{ height, width: "100%" }}
          notMerge
          lazyUpdate
        />
      ) : (
        <div
          style={{ height }}
          className="flex w-full items-center justify-center text-[0.72rem] font-semibold text-slate-300"
        >
          {emptyMessage}
        </div>
      )}
    </div>
  );
};

export default MetricChart;
