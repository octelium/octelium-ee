import ReactEChartsCore from "echarts-for-react/lib/core";
import * as echarts from "echarts/core";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { BarChart, LineChart as LineChartC } from "echarts/charts";
import {
  GridComponent,
  LegendComponent,
  MarkAreaComponent,
  MarkLineComponent,
  TitleComponent,
  TooltipComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([
  TitleComponent,
  TooltipComponent,
  GridComponent,
  BarChart,
  LineChartC,
  CanvasRenderer,
  LegendComponent,
  MarkLineComponent,
  MarkAreaComponent,
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

const LineChart = (props: Props) => {
  const { title, points, variant = "line" } = props;

  if (!points || points.length === 0) return null;

  const data = points.map((item) => [Timestamp.toDate(item.ts), item.value]);
  const values = points.map((p) => p.value);
  const avg = Math.round(values.reduce((a, b) => a + b, 0) / values.length);
  const peak = Math.max(...values);

  const yMax = Math.ceil(peak * 1.25);

  const option = {
    grid: {
      top: 28,
      right: 20,
      bottom: 42,
      left: 52,
      containLabel: false,
    },

    tooltip: {
      trigger: "axis",
      axisPointer: {
        type: variant === "bar" ? "shadow" : "line",
        lineStyle: { color: "#93c5fd", width: 1, type: "dashed" },
        shadowStyle: { color: "rgba(96,165,250,0.06)" },
      },
      backgroundColor: "#1e293b",
      borderColor: "#334155",
      borderWidth: 1,
      padding: [8, 12],
      textStyle: {
        color: "#f8fafc",
        fontSize: 12,
        fontFamily: "Ubuntu, sans-serif",
      },
      formatter: (params: any) => {
        const date: Date = params[0].value[0];
        const count: number = params[0].value[1];
        const diff = count - avg;
        const diffLabel =
          diff === 0
            ? `<span style="color:#94a3b8">avg</span>`
            : diff > 0
              ? `<span style="color:#60a5fa">+${diff} above avg</span>`
              : `<span style="color:#94a3b8">${diff} below avg</span>`;
        return `
          <div style="font-weight:700;margin-bottom:4px;">${date.toLocaleString(
            [],
            {
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            },
          )}</div>
          <div style="display:flex;align-items:baseline;gap:6px;">
            <span style="font-size:18px;font-weight:700;">${count.toLocaleString()}</span>
            <span style="font-size:11px;">${diffLabel}</span>
          </div>
        `;
      },
    },

    xAxis: {
      type: "time",
      boundaryGap: variant === "bar" ? ["5%", "5%"] : false,
      axisLine: { lineStyle: { color: "#e2e8f0" } },
      axisTick: { show: false },
      axisLabel: {
        color: "#94a3b8",
        fontSize: 11,
        fontFamily: "Ubuntu, sans-serif",
        fontWeight: 600,
        formatter: (value: number) =>
          echarts.format.formatTime("MM/dd HH:mm", value),
        hideOverlap: true,
      },
      splitLine: { show: false },
    },

    yAxis: {
      type: "value",
      max: yMax,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: "#94a3b8",
        fontSize: 11,
        fontFamily: "Ubuntu, sans-serif",
        fontWeight: 600,
        formatter: (v: number) =>
          v >= 1000 ? `${(v / 1000).toFixed(1)}k` : String(v),
      },
      splitLine: { lineStyle: { color: "#f1f5f9", type: "solid" } },
    },

    series: [
      {
        type: variant,
        data,
        large: true,

        ...(variant === "line" && {
          smooth: 0.4,
          symbol: "none",
          lineStyle: { color: "#1d4ed8", width: 2 },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "rgba(29,78,216,0.18)" },
              { offset: 1, color: "rgba(29,78,216,0.01)" },
            ]),
          },
        }),

        ...(variant === "bar" && {
          barMaxWidth: 24,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "#3b82f6" },
              { offset: 1, color: "#1d4ed8" },
            ]),
            borderRadius: [3, 3, 0, 0],
          },
          emphasis: {
            itemStyle: { color: "#60a5fa" },
          },
        }),

        markLine: {
          silent: true,
          symbol: "none",
          lineStyle: { color: "#93c5fd", type: "dashed", width: 1 },
          label: {
            formatter: `avg ${avg.toLocaleString()}`,
            position: "insideEndTop",
            color: "#60a5fa",
            fontSize: 11,
            fontWeight: 600,
            fontFamily: "Ubuntu, sans-serif",
          },
          data: [{ type: "average" }],
        },
      },
    ],
  };

  return (
    <div className="w-full">
      {title && (
        <p className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800 mb-2 px-1">
          {title}
        </p>
      )}
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        style={{ height: "220px", width: "100%" }}
      />
    </div>
  );
};

export default LineChart;
