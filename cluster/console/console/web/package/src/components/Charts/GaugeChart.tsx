import ReactEChartsCore from "echarts-for-react/lib/core";

import * as echarts from "echarts/core";

import { CanvasRenderer } from "echarts/renderers";

import { GaugeChart as GaugeChartC } from "echarts/charts";

echarts.use([CanvasRenderer, GaugeChartC]);

const GaugeChart = (props: {
  title?: string;
  name: string;
  num: number;
  total: number;
}) => {
  const total = props.total;

  const allowedPercentage = total === 0 ? 0 : (props.num / total) * 100;

  const displayValue = allowedPercentage.toFixed(1);

  const option = {
    title: {
      text: "Allowed Rate",

      left: "center",
      top: 10,
      textStyle: {
        fontSize: 16,
      },
    },

    tooltip: {},

    series: [
      {
        name: "Allowed Rate",
        type: "gauge",
        data: [
          {
            value: displayValue,
            name: "",
          },
        ],
        min: 0,
        max: 100,
        startAngle: 180,
        endAngle: 0,
        center: ["50%", "75%"],
        radius: "100%",

        axisLine: {
          lineStyle: {
            width: 5,
            color: [
              [0, "#666"],
              [props.total, "#666"],
            ],

            shadowBlur: 20,
            shadowOffsetX: 0,
            shadowOffsetY: 0,
            shadowColor: "rgba(0, 0, 0, 0.15)",
          },
        },

        pointer: { itemStyle: { color: "#222" } },

        axisTick: {},
        splitLine: {},

        detail: {
          valueAnimation: true,
          formatter: "{value}%",
          color: "auto",
          fontSize: 30,
          offsetCenter: [0, "70%"],
        },
      },
    ],
  };

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        style={{ height: "100%" }}
        notMerge={true}
        lazyUpdate={true}
      />
    </div>
  );
};

export default GaugeChart;
