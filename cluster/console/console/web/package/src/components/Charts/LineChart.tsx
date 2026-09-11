import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  axisLabelStyle,
  CHART_FONT,
  CHART_INK,
  CHART_TOOLTIP,
  seriesColor,
  splitLineStyle,
  useChartColorScheme,
  withAlpha,
} from "@/utils/charts/palette";
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
  color?: string;
  height?: number;
  sparkline?: boolean;
  emptyLabel?: string;
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

export const summarizePoints = (points?: DataPoint[]) => {
  const values = (points ?? [])
    .map((item) => item.value)
    .filter((value) => Number.isFinite(value));
  if (values.length === 0) return { average: 0, peak: 0, latest: 0, count: 0 };
  return {
    average: values.reduce((sum, value) => sum + value, 0) / values.length,
    peak: Math.max(...values),
    latest: values.at(-1) ?? 0,
    count: values.length,
  };
};

const LineChart = ({
  title,
  points,
  variant = "line",
  color,
  height,
  sparkline,
  emptyLabel,
}: Props) => {
  const colorScheme = useChartColorScheme();
  const accent = color ?? seriesColor(0);

  const data = useMemo(
    () =>
      (points ?? [])
        .map(
          (item) => [Timestamp.toDate(item.ts).getTime(), item.value] as const,
        )
        .filter(
          ([timestamp, value]) =>
            Number.isFinite(timestamp) && Number.isFinite(value),
        )
        .sort(([timestampA], [timestampB]) => timestampA - timestampB),
    [points],
  );

  const average = useMemo(
    () =>
      data.length === 0
        ? 0
        : data.reduce((sum, [, value]) => sum + value, 0) / data.length,
    [data],
  );

  const option = useMemo(
    () => ({
      animationDuration: sparkline ? 0 : 450,
      animationEasing: "cubicOut",
      aria: { enabled: !sparkline, decal: { show: false } },
      grid: sparkline
        ? { top: 2, right: 2, bottom: 2, left: 2, containLabel: false }
        : { top: 16, right: 14, bottom: 30, left: 8, containLabel: true },
      tooltip: sparkline
        ? { show: false }
        : {
            ...CHART_TOOLTIP,
            trigger: "axis",
            confine: true,
            axisPointer: {
              type: variant === "bar" ? "shadow" : "line",
              snap: true,
              lineStyle: { color: accent, width: 1, type: "dashed" },
              shadowStyle: { color: withAlpha(accent, 0.06) },
              label: { show: false },
            },
            formatter: (
              params: Array<{ value: [number, number]; marker?: string }>,
            ) => {
              const point = params[0];
              if (!point) return "";

              const [timestamp, value] = point.value;
              const difference = value - average;
              const differenceText =
                Math.abs(difference) < 0.005
                  ? "At average"
                  : `${difference > 0 ? "+" : ""}${formatNumber(difference)} vs average`;

              return `<div style="min-width:154px"><div style="margin-bottom:7px;color:${CHART_INK.onDarkMuted};font-size:11px;font-weight:700">${formatTooltipDate(timestamp)}</div><div style="display:flex;align-items:baseline;justify-content:space-between;gap:16px"><span style="font-size:18px;font-weight:700">${formatNumber(value)}</span><span style="color:${CHART_INK.onDarkMuted};font-size:10px;font-weight:700">${differenceText}</span></div></div>`;
            },
          },
      xAxis: {
        type: "time",
        boundaryGap: variant === "bar" ? ["4%", "4%"] : false,
        show: !sparkline,
        axisLine: { lineStyle: { color: CHART_INK.axis } },
        axisTick: { show: false },
        axisLabel: {
          ...axisLabelStyle,
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
        show: !sparkline,
        max: ({ max }: { max: number }) =>
          max === 0 ? 1 : Math.ceil(max * 1.15),
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { ...axisLabelStyle, formatter: formatCompactNumber },
        splitLine: splitLineStyle,
      },
      series: [
        {
          name: title ?? "Activity",
          type: variant,
          data,
          emphasis: { focus: "series" },
          ...(variant === "line" && {
            smooth: 0.28,
            showSymbol: !sparkline && data.length <= 12,
            symbol: "circle",
            symbolSize: 8,
            sampling: "lttb",
            lineStyle: {
              color: accent,
              width: sparkline ? 1.75 : 2,
              cap: "round",
              join: "round",
            },
            itemStyle: {
              color: accent,
              borderColor: CHART_INK.surface,
              borderWidth: 2,
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: withAlpha(accent, 0.2) },
                { offset: 0.65, color: withAlpha(accent, 0.05) },
                { offset: 1, color: withAlpha(accent, 0) },
              ]),
            },
          }),
          ...(variant === "bar" && {
            large: data.length > 400,
            barMaxWidth: 22,
            itemStyle: { color: accent, borderRadius: [4, 4, 0, 0] },
            emphasis: { itemStyle: { color: withAlpha(accent, 0.85) } },
          }),
          ...(!sparkline && {
            markLine: {
              silent: true,
              symbol: "none",
              animation: false,
              lineStyle: { color: CHART_INK.axis, type: "dashed", width: 1 },
              label: {
                formatter: `Avg ${formatNumber(average)}`,
                position: "insideEndTop",
                color: CHART_INK.muted,
                backgroundColor: withAlpha(CHART_INK.surface, 0.92),
                borderRadius: 4,
                padding: [2, 5],
                fontSize: 10,
                fontWeight: 700,
                fontFamily: CHART_FONT,
              },
              data: [{ yAxis: average }],
            },
          }),
        },
      ],
    }),
    [accent, average, colorScheme, data, sparkline, title, variant],
  );

  if (data.length === 0) {
    if (sparkline) {
      return (
        <div
          className="flex w-full items-center"
          style={{ height: height ?? 32 }}
        >
          <span className="h-px w-full bg-slate-200" />
        </div>
      );
    }
    return (
      <div
        className="flex w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center"
        style={{ minHeight: height ?? 208 }}
      >
        <p className="text-xs font-normal text-slate-500">
          {emptyLabel ?? "No activity in this range"}
        </p>
      </div>
    );
  }

  return (
    <div
      className="w-full"
      style={{ height: height ?? (sparkline ? 32 : 224) }}
      role="img"
      aria-label={title ?? "Activity over time"}
    >
      <ReactEChartsCore
        key={colorScheme}
        echarts={echarts}
        option={option}
        style={{ height: "100%", width: "100%" }}
        notMerge
        lazyUpdate
      />
    </div>
  );
};

export default LineChart;
