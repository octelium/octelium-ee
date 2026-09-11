import ReactEChartsCore from "echarts-for-react";
import { GaugeChart as GaugeChartC } from "echarts/charts";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import {
  CHART_INK,
  seriesColor,
  useChartColorScheme,
} from "@/utils/charts/palette";

echarts.use([CanvasRenderer, GaugeChartC]);

const CounterChart = (props: {
  value: number;
  label: string;
  title?: string;
  suffix?: string;
  color?: string;
  valueFormatter?: (value: number) => string;
}) => {
  const {
    value,
    label,
    title,
    suffix = "",
    color,
    valueFormatter,
  } = props;

  const colorScheme = useChartColorScheme();
  const accent = color ?? seriesColor(0);

  const option = {
    series: [
      {
        type: "gauge",
        startAngle: 90,
        endAngle: 90,
        min: 0,
        max: value === 0 ? 1 : value,
        pointer: { show: false },
        progress: { show: false },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        anchor: { show: false },
        title: {
          show: true,
          offsetCenter: [0, "40%"],
          fontSize: 11,
          fontWeight: 700,
          fontFamily: "Ubuntu, sans-serif",
          color: CHART_INK.muted,
        },
        detail: {
          valueAnimation: true,
          formatter: (currentValue: number) =>
            `${valueFormatter ? valueFormatter(currentValue) : currentValue}${suffix}`,
          color: accent,
          fontSize: 32,
          fontWeight: 700,
          fontFamily: "Ubuntu, sans-serif",
          offsetCenter: [0, "-10%"],
        },
        data: [{ value, name: label }],
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
        style={{ height: "120px", width: "100%" }}
        notMerge
        lazyUpdate
      />
    </div>
  );
};

export default CounterChart;
