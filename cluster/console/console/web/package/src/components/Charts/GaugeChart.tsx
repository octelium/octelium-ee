import ReactEChartsCore from "echarts-for-react";
import { GaugeChart as GaugeChartC } from "echarts/charts";
import { TooltipComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { CHART_INK, STATUS_COLORS } from "@/utils/charts/palette";

echarts.use([CanvasRenderer, GaugeChartC, TooltipComponent]);

const getGaugeColor = (pct: number): string => {
  if (pct >= 80) return STATUS_COLORS.good;
  if (pct >= 50) return STATUS_COLORS.warning;
  if (pct >= 25) return STATUS_COLORS.serious;
  return STATUS_COLORS.critical;
};

const GaugeChart = (props: {
  title?: string;
  name: string;
  num: number;
  total: number;
}) => {
  const { total, num, title, name } = props;
  const pct = total === 0 ? 0 : (num / total) * 100;
  const pctDisplay = parseFloat(pct.toFixed(1));
  const color = getGaugeColor(pctDisplay);

  const option = {
    tooltip: {
      formatter: () =>
        `<strong>${name}</strong><br/>${num.toLocaleString()} / ${total.toLocaleString()} &nbsp;<span style="color:#64748b">${pctDisplay}%</span>`,
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
        type: "gauge",
        data: [{ value: pctDisplay, name }],
        min: 0,
        max: 100,
        startAngle: 200,
        endAngle: -20,
        center: ["50%", "60%"],
        radius: "88%",

        axisLine: {
          roundCap: true,
          lineStyle: {
            width: 10,
            color: [
              [pct / 100, color],
              [1, "#f1f5f9"],
            ],
          },
        },

        progress: {
          show: true,
          roundCap: true,
          width: 10,
        },

        pointer: {
          show: true,
          length: "55%",
          width: 4,
          itemStyle: { color },
        },

        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },

        anchor: {
          show: true,
          size: 10,
          itemStyle: { color, borderWidth: 0 },
        },

        title: {
          show: true,
          offsetCenter: [0, "82%"],
          fontSize: 11,
          fontWeight: 700,
          fontFamily: "Ubuntu, sans-serif",
          color: CHART_INK.muted,
          formatter: name,
        },

        detail: {
          valueAnimation: true,
          formatter: "{value}%",
          color,
          fontSize: 26,
          fontWeight: 700,
          fontFamily: "Ubuntu, sans-serif",
          offsetCenter: [0, "30%"],
        },
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
        echarts={echarts}
        option={option}
        style={{ height: "180px", width: "100%" }}
        notMerge
        lazyUpdate
      />
      <div className="flex items-center justify-center gap-1 -mt-2">
        <span className="text-xs font-normal text-slate-500">
          {num.toLocaleString()}
        </span>
        <span className="text-micro text-slate-300">/</span>
        <span className="text-xs font-normal text-slate-500">
          {total.toLocaleString()}
        </span>
      </div>
    </div>
  );
};

export default GaugeChart;
