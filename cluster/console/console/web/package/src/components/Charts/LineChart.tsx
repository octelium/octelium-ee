import { Timestamp } from "@/apis/google/protobuf/timestamp";
import ReactEChartsCore from "echarts-for-react";
import { BarChart, LineChart as LineChartC } from "echarts/charts";
import {
  AriaComponent,
  GridComponent,
  MarkLineComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useMemo } from "react";

echarts.use([
  AriaComponent,
  TooltipComponent,
  GridComponent,
  BarChart,
  LineChartC,
  CanvasRenderer,
  MarkLineComponent,
]);

interface DataPoint {
  ts: Timestamp;
  value: number;
}

export interface Props {
  title?: string;
  points?: DataPoint[];
  variant?: "line" | "bar";
}

const formatNumber = (value: number, maximumFractionDigits = 1) =>
  value.toLocaleString(undefined, { maximumFractionDigits });

const formatCompactNumber = (value: number) =>
  new Intl.NumberFormat(undefined, {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);

const formatTooltipDate = (value: number) =>
  new Date(value).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

const LineChart = ({ title, points, variant = "line" }: Props) => {
  const data = useMemo(
    () =>
      (points ?? [])
        .map((item) => [Timestamp.toDate(item.ts).getTime(), item.value] as const)
        .filter(
          ([timestamp, value]) =>
            Number.isFinite(timestamp) && Number.isFinite(value),
        )
        .sort(([timestampA], [timestampB]) => timestampA - timestampB),
    [points],
  );

  const statistics = useMemo(() => {
    if (data.length === 0) {
      return { average: 0, peak: 0, latest: 0 };
    }

    const values = data.map(([, value]) => value);

    return {
      average: values.reduce((sum, value) => sum + value, 0) / values.length,
      peak: Math.max(...values),
      latest: values.at(-1) ?? 0,
    };
  }, [data]);

  const option = useMemo(
    () => ({
      animationDuration: 650,
      animationEasing: "cubicOut",
      aria: {
        enabled: true,
        decal: { show: false },
      },
      grid: {
        top: 22,
        right: 14,
        bottom: 34,
        left: 8,
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
          "border-radius:10px;box-shadow:0 10px 24px rgba(15,23,42,.18);",
        axisPointer: {
          type: variant === "bar" ? "shadow" : "line",
          snap: true,
          lineStyle: {
            color: "#60a5fa",
            width: 1,
            type: "dashed",
          },
          shadowStyle: { color: "rgba(37,99,235,0.06)" },
          label: { show: false },
        },
        formatter: (
          params: Array<{ value: [number, number]; marker?: string }>,
        ) => {
          const point = params[0];
          if (!point) return "";

          const [timestamp, value] = point.value;
          const difference = value - statistics.average;
          const differenceText =
            Math.abs(difference) < 0.005
              ? "At average"
              : `${difference > 0 ? "+" : ""}${formatNumber(difference)} vs average`;

          return `<div style="min-width:154px"><div style="margin-bottom:7px;color:#cbd5e1;font-size:11px;font-weight:700">${formatTooltipDate(timestamp)}</div><div style="display:flex;align-items:baseline;justify-content:space-between;gap:16px"><span style="font-size:18px;font-weight:700">${formatNumber(value)}</span><span style="color:${difference >= 0 ? "#93c5fd" : "#cbd5e1"};font-size:10px;font-weight:700">${differenceText}</span></div></div>`;
        },
      },
      xAxis: {
        type: "time",
        boundaryGap: variant === "bar" ? ["4%", "4%"] : false,
        axisLine: { lineStyle: { color: "#e2e8f0" } },
        axisTick: { show: false },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          fontWeight: 600,
          margin: 12,
          hideOverlap: true,
          formatter: (value: number) =>
            echarts.format.formatTime("MM/dd\nHH:mm", value),
        },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        min: 0,
        minInterval: 1,
        max: ({ max }: { max: number }) =>
          max === 0 ? 1 : Math.ceil(max * 1.15),
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: "#94a3b8",
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          fontWeight: 600,
          margin: 12,
          formatter: formatCompactNumber,
        },
        splitLine: {
          lineStyle: { color: "#e2e8f0", type: "dashed", opacity: 0.7 },
        },
      },
      series: [
        {
          name: title ?? "Activity",
          type: variant,
          data,
          emphasis: { focus: "series" },
          ...(variant === "line" && {
            smooth: 0.28,
            showSymbol: data.length <= 12,
            symbol: "circle",
            symbolSize: 6,
            sampling: "lttb",
            lineStyle: {
              color: "#2563eb",
              width: 2.5,
              cap: "round",
              join: "round",
            },
            itemStyle: {
              color: "#2563eb",
              borderColor: "#ffffff",
              borderWidth: 2,
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: "rgba(37,99,235,0.20)" },
                { offset: 0.65, color: "rgba(37,99,235,0.05)" },
                { offset: 1, color: "rgba(37,99,235,0)" },
              ]),
            },
          }),
          ...(variant === "bar" && {
            large: data.length > 400,
            barMaxWidth: 22,
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: "#60a5fa" },
                { offset: 1, color: "#2563eb" },
              ]),
              borderRadius: [5, 5, 1, 1],
            },
            emphasis: {
              itemStyle: { color: "#1d4ed8" },
            },
          }),
          markLine: {
            silent: true,
            symbol: "none",
            animation: false,
            lineStyle: {
              color: "#94a3b8",
              type: "dashed",
              width: 1,
              opacity: 0.75,
            },
            label: {
              formatter: `Avg ${formatNumber(statistics.average)}`,
              position: "insideEndTop",
              color: "#64748b",
              backgroundColor: "rgba(248,250,252,0.9)",
              borderRadius: 4,
              padding: [2, 5],
              fontSize: 10,
              fontWeight: 700,
              fontFamily: "Ubuntu, sans-serif",
            },
            data: [{ yAxis: statistics.average }],
          },
        },
      ],
    }),
    [data, statistics, title, variant],
  );

  if (data.length === 0) {
    return (
      <div className="flex min-h-52 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center">
        <div>
          {title && (
            <p className="mb-1 text-xs font-bold text-slate-700">{title}</p>
          )}
          <p className="text-xs font-semibold text-slate-400">
            No activity data available
          </p>
        </div>
      </div>
    );
  }

  return (
    <section className="w-full" aria-label={title ?? "Activity over time"}>
      <div className="flex flex-wrap items-end justify-between gap-3 px-1 pb-2">
        <div>
          <p className="text-[0.62rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            {title ?? "Activity over time"}
          </p>
          <p className="mt-0.5 text-xs font-semibold text-slate-500">
            {data.length.toLocaleString()} data point
            {data.length === 1 ? "" : "s"}
          </p>
        </div>

        <dl className="flex items-center gap-4 sm:gap-5">
          {[
            { label: "Latest", value: statistics.latest },
            { label: "Average", value: statistics.average },
            { label: "Peak", value: statistics.peak },
          ].map((statistic) => (
            <div key={statistic.label} className="text-right">
              <dt className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                {statistic.label}
              </dt>
              <dd className="mt-0.5 text-xs font-bold tabular-nums text-slate-700">
                {formatNumber(statistic.value)}
              </dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="h-56 w-full rounded-xl border border-slate-200/80 bg-slate-50/60 px-1 pt-1 sm:h-60 sm:px-2">
        <ReactEChartsCore
          echarts={echarts}
          option={option}
          style={{ height: "100%", width: "100%" }}
          notMerge
          lazyUpdate
        />
      </div>
    </section>
  );
};

export default LineChart;
