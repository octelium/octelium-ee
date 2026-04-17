import ReactEChartsCore from "echarts-for-react/lib/core";
import { BarChart as BarChartC } from "echarts/charts";
import {
  LegendComponent,
  PolarComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([
  BarChartC,
  PolarComponent,
  LegendComponent,
  TooltipComponent,
  CanvasRenderer,
]);

const SLICE_COLORS = ["#1d4ed8", "#60a5fa", "#bfdbfe", "#dbeafe"];

const PolarChart = (props: {
  data: { name: string; value: number }[];
  title?: string;
}) => {
  const { data, title } = props;

  if (!data || data.length === 0) return null;

  const total = data.reduce((a, b) => a + b.value, 0);
  const max = Math.max(...data.map((d) => d.value));

  const option = {
    polar: {
      radius: ["15%", "80%"],
      center: ["50%", "55%"],
    },

    angleAxis: {
      type: "value",
      startAngle: 90,
      clockwise: false,
      max,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },

    radiusAxis: {
      type: "category",
      data: data.map((x) => x.name),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: "#64748b",
        fontSize: 11,
        fontWeight: 700,
        fontFamily: "Ubuntu, sans-serif",
      },
      splitLine: {
        show: true,
        lineStyle: { color: "#f1f5f9", type: "solid" },
      },
    },

    tooltip: {
      trigger: "item",
      formatter: (params: any) => {
        const pct =
          total === 0 ? "0.0" : ((params.value / total) * 100).toFixed(1);
        return `
          <div style="font-weight:700;margin-bottom:4px;font-family:Ubuntu,sans-serif">${params.name}</div>
          <div style="display:flex;align-items:baseline;gap:6px;">
            <span style="font-size:16px;font-weight:700;">${params.value.toLocaleString()}</span>
            <span style="font-size:11px;color:#94a3b8">${pct}%</span>
          </div>
        `;
      },
      backgroundColor: "#1e293b",
      borderColor: "#334155",
      borderWidth: 1,
      textStyle: {
        color: "#f8fafc",
        fontSize: 12,
        fontFamily: "Ubuntu, sans-serif",
      },
      extraCssText: "border-radius:6px; padding:8px 12px;",
    },

    series: [
      {
        type: "bar",
        data: data.map((x, i) => ({
          value: x.value,
          itemStyle: {
            color: SLICE_COLORS[i % SLICE_COLORS.length],
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: SLICE_COLORS[i % SLICE_COLORS.length],
              opacity: 0.85,
            },
          },
        })),
        coordinateSystem: "polar",
        barMaxWidth: 16,
        label: {
          show: true,
          position: "middle",
          formatter: (params: any) => params.value.toLocaleString(),
          color: "#ffffff",
          fontSize: 10,
          fontWeight: 700,
          fontFamily: "Ubuntu, sans-serif",
        },
        roundCap: true,
      },
    ],
  };

  return (
    <div className="w-full flex flex-col">
      {title && (
        <p className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800 mb-1 px-1">
          {title}
        </p>
      )}
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        style={{ height: "220px", width: "100%" }}
        notMerge
        lazyUpdate
      />
      <div className="flex flex-wrap gap-x-4 gap-y-1.5 mt-1 px-1">
        {data.map((item, i) => (
          <span
            key={item.name}
            className="flex items-center gap-1.5 text-[0.72rem] font-semibold text-slate-500"
          >
            <span
              className="w-2 h-2 rounded-full shrink-0"
              style={{ background: SLICE_COLORS[i % SLICE_COLORS.length] }}
            />
            {item.name} — {item.value.toLocaleString()}
          </span>
        ))}
      </div>
    </div>
  );
};

export default PolarChart;
