import ReactEChartsCore from "echarts-for-react/lib/core";
import * as echarts from "echarts/core";

import { BarChart as BarChartC } from "echarts/charts";

import { CanvasRenderer } from "echarts/renderers";

import {
  LegendComponent,
  PolarComponent,
  TooltipComponent,
} from "echarts/components";

echarts.use([
  BarChartC,
  PolarComponent,
  LegendComponent,
  TooltipComponent,
  CanvasRenderer,
]);

const PolarChart = (props: { data: { name: string; value: number }[] }) => {
  const { data } = props;

  return (
    <div>
      <ReactEChartsCore
        echarts={echarts}
        option={{
          polar: {
            radius: [30, "80%"],
            center: ["50%", "85%"],
          },

          angleAxis: {
            type: "category",

            startAngle: 180,
            endAngle: 0,

            min: 0,
            max: 100,

            axisLabel: {
              show: true,
            },
            splitLine: { show: false },
            axisLine: { show: false },
          },

          radiusAxis: {
            type: "value",
            show: true,
            axisLabel: {
              show: false,
            },
            splitLine: {
              show: false,
            },
            data: data.map((x) => x.name),
          },

          tooltip: {},
          series: {
            data: data.map((x) => x.value),
            coordinateSystem: "polar",
            label: {
              show: false,
              position: "middle",
              formatter: "{b}: {c}",
            },
          },
        }}
      />
    </div>
  );
};

export default PolarChart;
