import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  Attribute,
  AttributeFilter,
  AttributeFilter_Operator,
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricSelector,
  NumberPoint,
  QueryMetricsRequest,
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
import { useMemo, useState } from "react";
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
  AttributeFilter.create({ key, operator: AttributeFilter_Operator.EQ, value });

const numberValue = (p: NumberPoint): number => {
  if (p.value.oneofKind === "asDouble") return p.value.asDouble;
  if (p.value.oneofKind === "asInt") return Number(p.value.asInt);
  return 0;
};

const tsToMillis = (t: Timestamp): number =>
  Number(t.seconds) * 1000 + Math.floor(t.nanos / 1e6);

const seriesLabel = (labels: Attribute[], fallback: string): string => {
  if (!labels || labels.length === 0) return fallback;
  return labels
    .map((l) =>
      l.key === "quantile"
        ? `p${Math.round(parseFloat(l.value) * 100)}`
        : l.value,
    )
    .join(" · ");
};

const formatBytes = (v: number): string => {
  if (!isFinite(v) || v <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = v;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(n < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
};

const formatValue = (v: number, unit?: string): string => {
  switch (unit) {
    case "bytes":
      return formatBytes(v);
    case "ms":
      return `${v.toFixed(v < 10 ? 1 : 0)} ms`;
    case "seconds":
      return `${v.toFixed(2)} s`;
    case "cores":
      return v.toFixed(2);
    case "requests/s":
    case "cycles/s":
      return `${v.toFixed(1)}/s`;
    case "requests":
      return Math.round(v).toLocaleString();
    default:
      if (Number.isInteger(v)) return v.toLocaleString();
      return v >= 1000 ? Math.round(v).toLocaleString() : v.toFixed(2);
  }
};

const extractSeries = (
  series: TimeSeries[] | undefined,
  fallback: string,
): { name: string; data: [number, number][] }[] => {
  if (!series) return [];
  return series
    .map((s) => {
      if (s.points.oneofKind !== "number") return null;
      return {
        name: seriesLabel(s.labels ?? [], fallback),
        data: s.points.number.points.map(
          (p) => [tsToMillis(p.timestamp!), numberValue(p)] as [number, number],
        ),
      };
    })
    .filter((x): x is { name: string; data: [number, number][] } => x !== null);
};

const MetricChart = (props: {
  title?: string;
  unit?: string;
  metric: string;
  operation: QueryOperation;
  component?: ComponentSelector;
  filters?: AttributeFilter[];
  groupBy?: string[];
  from?: Timestamp;
  to?: Timestamp;
  lookbackSeconds?: number;
  step?: Duration;
  limitSeries?: number;
  height?: number;
}) => {
  const {
    title,
    unit,
    metric,
    operation,
    component,
    filters,
    groupBy,
    limitSeries,
    height = 240,
  } = props;

  const [step, setStep] = useState(props.step);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "queryMetrics",
      {
        metric,
        operation,
        component,
        filters,
        groupBy,
        step,
        from: props.from,
        to: props.to,
        lookbackSeconds: props.lookbackSeconds,
        limitSeries,
      },
    ],

    queryFn: async () => {
      const now = new Date();
      const from =
        props.from ??
        Timestamp.fromDate(
          new Date(now.getTime() - (props.lookbackSeconds ?? 6 * 3600) * 1000),
        );
      const to = props.to ?? Timestamp.fromDate(now);

      const req = QueryMetricsRequest.create({
        metric: MetricSelector.create({ name: metric }),
        timeRange: TimeRange.create({ from, to }),
        step,
        component,
        filters,
        groupBy,
        operation,
        limitSeries,
      });

      const { response } = await getClientVisibilityMetrics().queryMetrics(req);

      return response;
    },

    refetchInterval: refetchIntervalChart,
  });

  const series = useMemo(
    () => extractSeries(qry.data?.series, title ?? metric),
    [qry.data, title, metric],
  );

  const option = useMemo(() => {
    const multi = series.length > 1;
    return {
      color: LINE_COLORS,
      grid: {
        left: 8,
        right: 16,
        top: 16,
        bottom: multi ? 28 : 8,
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
        valueFormatter: (v: number | string) => formatValue(Number(v), unit),
      },
      legend: multi
        ? {
            bottom: 0,
            icon: "circle",
            itemWidth: 8,
            itemHeight: 8,
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
          formatter: (v: number) => formatValue(v, unit),
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
  }, [series, unit]);

  return (
    <div className="w-full flex flex-col">
      <div className="flex items-end px-1 mb-1">
        {title && (
          <p className="flex-1 text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800">
            {title}
          </p>
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
          className="w-full flex items-center justify-center text-[0.72rem] font-semibold text-slate-300"
        >
          {qry.isLoading ? "Loading…" : "No data"}
        </div>
      )}
    </div>
  );
};

export default MetricChart;
