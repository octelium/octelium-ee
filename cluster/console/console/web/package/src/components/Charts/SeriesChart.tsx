import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  axisLabelStyle,
  CHART_INK,
  CHART_TOOLTIP,
  seriesColor,
  seriesColors,
  splitLineStyle,
  useChartColorScheme,
  withAlpha,
} from "@/utils/charts/palette";
import ReactEChartsCore from "echarts-for-react";
import { BarChart, LineChart as LineChartC } from "echarts/charts";
import {
  AriaComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useMemo } from "react";

echarts.use([
  AriaComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  BarChart,
  LineChartC,
  CanvasRenderer,
]);


export interface Series {
  name: string;
  points: { ts: Timestamp; value: number }[];
}

export interface Props {
  series: Series[];
  height?: number;
  stacked?: boolean;
  variant?: "line" | "bar";
  valueFormatter?: (value: number) => string;
  emptyLabel?: string;
}

const defaultFormatter = (value: number) =>
  value.toLocaleString(undefined, { maximumFractionDigits: 1 });

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

const SeriesChart = ({
  series,
  height = 260,
  stacked = false,
  variant = "line",
  valueFormatter = defaultFormatter,
  emptyLabel = "No activity in the selected range",
}: Props) => {
  const colorScheme = useChartColorScheme();
  const data = useMemo(
    () =>
      series.map((item) => ({
        name: item.name,
        points: item.points
          .map(
            (point) =>
              [Timestamp.toDate(point.ts).getTime(), point.value] as const,
          )
          .filter(([ts, value]) => Number.isFinite(ts) && Number.isFinite(value))
          .sort(([a], [b]) => a - b),
      })),
    [series],
  );

  const isEmpty = useMemo(
    () =>
      data.length === 0 ||
      data.every((item) =>
        item.points.every(([, value]) => value === 0),
      ),
    [data],
  );

  const option = useMemo(
    () => ({
      animationDuration: 600,
      animationEasing: "cubicOut",
      aria: { enabled: true, decal: { show: false } },
      color: seriesColors(),
      grid: {
        top: data.length > 1 ? 34 : 16,
        right: 14,
        bottom: 30,
        left: 8,
        containLabel: true,
      },
      legend:
        data.length > 1
          ? {
              type: "scroll",
              top: 0,
              icon: "roundRect",
              itemWidth: 9,
              itemHeight: 9,
              itemGap: 14,
              textStyle: {
                color: CHART_INK.muted,
                fontSize: 10,
                fontWeight: 700,
                fontFamily: "Ubuntu, sans-serif",
              },
            }
          : { show: false },
      tooltip: {
        trigger: "axis",
        confine: true,
        backgroundColor: CHART_TOOLTIP.backgroundColor,
        borderColor: CHART_TOOLTIP.borderColor,
        borderWidth: 1,
        padding: [10, 12],
        textStyle: {
          color: CHART_INK.onDark,
          fontSize: 12,
          fontFamily: "Ubuntu, sans-serif",
        },
        extraCssText:
          CHART_TOOLTIP.extraCssText,
        axisPointer: {
          type: variant === "bar" ? "shadow" : "line",
          lineStyle: { color: CHART_INK.marker, width: 1, type: "dashed" },
          shadowStyle: { color: withAlpha(seriesColor(0), 0.06) },
        },
        formatter: (
          params: Array<{
            value: [number, number];
            seriesName: string;
            marker: string;
          }>,
        ) => {
          if (!params.length) return "";
          const header = new Date(params[0].value[0]).toLocaleString([], {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          });
          const rows = params
            .slice()
            .sort((a, b) => b.value[1] - a.value[1])
            .map(
              (item) =>
                `<div style="display:flex;align-items:center;gap:10px;justify-content:space-between"><span style="min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:190px">${item.marker}${escapeHTML(item.seriesName)}</span><b style="font-variant-numeric:tabular-nums">${valueFormatter(item.value[1])}</b></div>`,
            )
            .join("");
          return `<div style="min-width:190px"><div style="margin-bottom:6px;color:${CHART_INK.onDarkMuted};font-size:11px;font-weight:700">${header}</div>${rows}</div>`;
        },
      },
      xAxis: {
        type: "time",
        boundaryGap: variant === "bar" ? ["3%", "3%"] : false,
        axisLine: { lineStyle: { color: CHART_INK.grid } },
        axisTick: { show: false },
        axisLabel: {
          color: CHART_INK.muted,
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
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: CHART_INK.muted,
          fontSize: 10,
          fontFamily: "Ubuntu, sans-serif",
          fontWeight: 600,
          margin: 10,
          formatter: valueFormatter,
        },
        splitLine: {
          ...splitLineStyle,
        },
      },
      series: data.map((item, index) => ({
        name: item.name,
        type: variant,
        data: item.points,
        stack: stacked ? "total" : undefined,
        emphasis: { focus: "series" },
        ...(variant === "line" && {
          smooth: 0.26,
          showSymbol: false,
          sampling: "lttb",
          lineStyle: { width: 2.2, cap: "round", join: "round" },
          ...(data.length === 1 && {
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: withAlpha(seriesColor(0), 0.2) },
                { offset: 1, color: withAlpha(seriesColor(0), 0) },
              ]),
            },
          }),
          ...(stacked && { areaStyle: { opacity: 0.28 } }),
        }),
        ...(variant === "bar" && {
          barMaxWidth: 20,
          itemStyle: {
            borderRadius: stacked ? [0, 0, 0, 0] : [4, 4, 1, 1],
            color: seriesColor(index),
          },
        }),
      })),
    }),
    [colorScheme, data, stacked, valueFormatter, variant],
  );

  if (isEmpty) {
    return (
      <div
        className="flex w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center"
        style={{ height }}
      >
        <p className="text-xs font-normal text-slate-500">{emptyLabel}</p>
      </div>
    );
  }

  return (
    <div
      className="w-full rounded-xl border border-slate-200/80 bg-slate-50/50 px-1 pt-1 sm:px-2"
      style={{ height }}
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

export default SeriesChart;
