import ReactEChartsCore from "echarts-for-react/lib/core";
import * as echarts from "echarts/core";

import {
  // LineChart,
  // BarChart,
  PieChart as PieChartC,
} from "echarts/charts";

import { LegendComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([PieChartC, CanvasRenderer, LegendComponent]);

const PieChart = (props: { data: { name: string; value: number }[] }) => {
  const { data } = props;

  if (data.reduce((acc, item) => acc + item.value, 0) === 0) {
    return <></>;
  }

  return (
    <div>
      <ReactEChartsCore
        echarts={echarts}
        option={{
          tooltip: {
            trigger: "item",
            show: true,
          },
          legend: {
            type: "scroll",
            orient: "horizontal",
            bottom: 10,
            data: data.map((x) => x.name),
            textStyle: {
              fontWeight: "bold",
              fontSize: 14,
              color: "#333",
            },
          },
          grid: {
            bottom: 60,
            containLabel: true,
          },

          series: [
            {
              type: "pie",
              radius: ["40%", "70%"],
              avoidLabelOverlap: true,
              itemStyle: {
                borderRadius: 4,
                borderColor: "#fff",
                borderWidth: 2,

                shadowBlur: 20,
                shadowOffsetX: 0,
                shadowOffsetY: 0,
                shadowColor: "rgba(0, 0, 0, 0.15)",
              },
              label: {
                show: true,
                position: "top",
                fontWeight: "bold",
                formatter: "{b} ({c})",

                minShowLabelAngle: 1,
              },
              emphasis: {
                label: {
                  show: true,
                  fontWeight: "bold",
                },
              },
              labelLine: {
                show: true,
                showAbove: true,
              },
              data: data,
            },
          ],
        }}
      />
    </div>
  );
};

export default PieChart;
