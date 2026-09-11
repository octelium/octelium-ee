import ReactEChartsCore from "echarts-for-react";
import { BarChart as BarChartC } from "echarts/charts";
import {
  LegendComponent,
  PolarComponent,
  TooltipComponent,
} from "echarts/components";
import {
  CHART_FONT,
  CHART_INK,
  CHART_TOOLTIP,
  sliceColors,
  useChartColorScheme,
} from "@/utils/charts/palette";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([
  BarChartC,
  PolarComponent,
  LegendComponent,
  TooltipComponent,
  CanvasRenderer,
]);

const PolarChart = (props: {
  data: { name: string; value: number }[];
  title?: string;
}) => {
  const { data, title } = props;

  const colorScheme = useChartColorScheme();
  const slices = sliceColors();

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
        color: CHART_INK.muted,
        fontSize: 11,
        fontWeight: 700,
        fontFamily: CHART_FONT,
      },
      splitLine: {
        show: true,
        lineStyle: { color: CHART_INK.track, type: "solid" },
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
            <span style="font-size:11px;color:${CHART_INK.onDarkMuted}">${pct}%</span>
          </div>
        `;
      },
      backgroundColor: CHART_TOOLTIP.backgroundColor,
      borderColor: CHART_TOOLTIP.borderColor,
      borderWidth: 1,
      textStyle: CHART_TOOLTIP.textStyle,
      extraCssText: "border-radius:6px; padding:8px 12px;",
    },

    series: [
      {
        type: "bar",
        data: data.map((x, i) => ({
          value: x.value,
          itemStyle: {
            color: slices[i % slices.length],
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: slices[i % slices.length],
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
          color: CHART_INK.onAccent,
          fontSize: 10,
          fontWeight: 700,
          fontFamily: CHART_FONT,
        },
        roundCap: true,
      },
    ],
  };

  return (
    <div className="w-full flex flex-col">
      {title && (
        <p className="text-body font-semibold uppercase tracking-[0.05em] text-slate-800 mb-1 px-1">
          {title}
        </p>
      )}
      <ReactEChartsCore
        key={colorScheme}
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
            className="flex items-center gap-1.5 text-xs font-normal text-slate-500"
          >
            <span
              className="w-2 h-2 rounded-full shrink-0"
              style={{ background: slices[i % slices.length] }}
            />
            {item.name} — {item.value.toLocaleString()}
          </span>
        ))}
      </div>
    </div>
  );
};

export default PolarChart;
