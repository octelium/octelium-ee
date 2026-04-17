import ReactEChartsCore from "echarts-for-react/lib/core";
import { PieChart as PieChartC } from "echarts/charts";
import { LegendComponent, TooltipComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([PieChartC, CanvasRenderer, LegendComponent, TooltipComponent]);

const SLICE_COLORS = ["#1d4ed8", "#60a5fa", "#bfdbfe", "#dbeafe"];

const PieChart = (props: { data: { name: string; value: number }[] }) => {
  const { data } = props;
  const total = data.reduce((acc, item) => acc + item.value, 0);

  if (total === 0) return null;

  return (
    <div>
      <ReactEChartsCore
        echarts={echarts}
        style={{ height: 160 }}
        option={{
          tooltip: {
            trigger: "item",
            formatter: (params: any) => {
              const pct = ((params.value / total) * 100).toFixed(1);
              return `<strong>${params.name}</strong><br/>${params.value.toLocaleString()} &nbsp;<span style="color:#94a3b8">${pct}%</span>`;
            },
            backgroundColor: "#1e293b",
            borderColor: "#334155",
            borderWidth: 1,
            textStyle: { color: "#f8fafc", fontSize: 12 },
            extraCssText: "border-radius:6px; padding:8px 12px;",
          },
          legend: { show: false },
          series: [
            {
              type: "pie",
              radius: ["52%", "78%"],
              center: ["50%", "50%"],
              avoidLabelOverlap: true,
              itemStyle: {
                borderColor: "#fff",
                borderWidth: 2,
              },
              label: { show: false },
              labelLine: { show: false },
              emphasis: {
                itemStyle: {
                  borderWidth: 3,
                },
                scale: true,
                scaleSize: 4,
              },
              data: data.map((item, i) => ({
                ...item,
                itemStyle: { color: SLICE_COLORS[i % SLICE_COLORS.length] },
              })),
            },
          ],
        }}
      />

      <div className="flex flex-wrap gap-x-4 gap-y-1.5 mt-2">
        {data.map((item, i) => (
          <span
            key={item.name}
            className="flex items-center gap-1.5 text-[0.72rem] font-semibold text-slate-500"
          >
            <span
              className="w-2 h-2 rounded-full flex-shrink-0"
              style={{ background: SLICE_COLORS[i % SLICE_COLORS.length] }}
            />
            {item.name} — {item.value.toLocaleString()}
          </span>
        ))}
      </div>
    </div>
  );
};

export default PieChart;
