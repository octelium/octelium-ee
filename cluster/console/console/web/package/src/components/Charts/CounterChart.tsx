import ReactEChartsCore from "echarts-for-react/lib/core";
import { GaugeChart as GaugeChartC } from "echarts/charts";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([CanvasRenderer, GaugeChartC]);

const CounterChart = (props: {
  value: number;
  label: string;
  title?: string;
  suffix?: string;
  color?: string;
}) => {
  const { value, label, title, suffix = "", color = "#1d4ed8" } = props;

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
          color: "#94a3b8",
        },
        detail: {
          valueAnimation: true,
          formatter: `{value}${suffix}`,
          color,
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
        <p className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800 mb-1 px-1">
          {title}
        </p>
      )}
      <ReactEChartsCore
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
