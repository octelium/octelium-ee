import ReactEChartsCore from "echarts-for-react/lib/core";
import * as echarts from "echarts/core";

import { BarChart, LineChart as LineChartC } from "echarts/charts";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { CanvasRenderer } from "echarts/renderers";

import {
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
} from "echarts/components";

echarts.use([
  TitleComponent,
  TooltipComponent,
  GridComponent,
  BarChart,
  CanvasRenderer,
]);

echarts.use([LineChartC, CanvasRenderer, LegendComponent]);

interface DataPoint {
  ts: Timestamp;
  value: number;
}

export interface Props {
  title?: string;
  points?: DataPoint[];
}

const LineChart = (props: Props) => {
  const { title, points } = props;

  if (!points || points.length === 0) {
    return <></>;
  }

  const data = points?.map((item) => {
    return [Timestamp.toDate(item.ts), item.value];
  });

  const option = {
    title: {
      text: title,
      left: "center",
    },
    tooltip: {
      trigger: "axis",
      formatter: (params: any) => {
        const date = params[0].value[0];
        const count = params[0].value[1];
        return `${date.toLocaleString()} (${count})`;
      },
    },
    xAxis: {
      type: "time",
      boundaryGap: false,
      hideOverlap: true,
      formatter: (value: number, index: number) => {
        const date = new Date(value);

        return echarts.format.formatTime("HH:mm", value);
      },
    },
    yAxis: {
      type: "value",
    },
    series: [
      {
        type: "bar",
        data: data,
        large: true,
      },
    ],
  };

  return (
    <div className="w-full">
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        // style={{ height: "400px", width: "100%" }}
      />
    </div>
  );
};

export default LineChart;
