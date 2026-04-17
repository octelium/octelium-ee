import { formatNumber } from "@/utils";
import {
  getRefNameQueryArgStr,
  getResourcePath,
  getResourceRef,
  printResourceNameWithDisplay,
  Resource,
} from "@/utils/pb";
import ReactEChartsCore from "echarts-for-react/lib/core";
import { BarChart } from "echarts/charts";
import { GridComponent, TooltipComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { ResourceHoverCard } from "../ResourceList";

echarts.use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

const BAR_HEIGHT = 28;
const BAR_GAP = 10;

const TopList = (props: {
  title: string;
  items?: { resource: Resource; count: number }[];
  to?: string;
}) => {
  const { items, to } = props;

  if (!items || items.length < 1) return null;

  const topCount = items[0].count;
  const chartHeight = items.length * (BAR_HEIGHT + BAR_GAP) + 16;

  const labelColumnWidth = 180;

  const option = {
    grid: {
      top: 4,
      bottom: 4,
      left: labelColumnWidth,
      right: 72,
      containLabel: false,
    },
    tooltip: {
      trigger: "axis",
      axisPointer: {
        type: "shadow",
        shadowStyle: { color: "rgba(96,165,250,0.07)" },
      },
      backgroundColor: "#1e293b",
      borderColor: "#334155",
      borderWidth: 1,
      padding: [8, 12],
      textStyle: {
        color: "#f8fafc",
        fontSize: 12,
        fontFamily: "Ubuntu, sans-serif",
      },
      formatter: (params: any) => {
        const pct = Math.round((params[0].value / topCount) * 100);
        return `
          <span style="font-weight:700;font-size:15px;">${params[0].value.toLocaleString()}</span>
          <span style="color:#60a5fa;font-size:11px;margin-left:6px;">${pct}% of top</span>
        `;
      },
    },
    xAxis: {
      type: "value",
      show: false,
      max: topCount,
    },
    yAxis: {
      type: "category",
      inverse: true,
      data: items.map((_, i) => String(i)),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },
    series: [
      {
        type: "bar",
        data: items.map((item) => item.count),
        barMaxWidth: BAR_HEIGHT,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
            { offset: 0, color: "#0f172a" },
            { offset: 1, color: "#334155" },
          ]),
          borderRadius: [0, 3, 3, 0],
        },
        emphasis: {
          itemStyle: {
            color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
              { offset: 0, color: "#1e293b" },
              { offset: 1, color: "#475569" },
            ]),
          },
        },
        label: {
          show: true,
          position: "right",
          formatter: (params: any) => formatNumber(params.value),
          color: "#475569",
          fontSize: 11,
          fontWeight: "bold",
          fontFamily: "Ubuntu, sans-serif",
        },
        animationDuration: 600,
        animationEasing: "cubicOut" as const,
        animationDelay: (idx: number) => idx * 40,
      },
    ],
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className="w-full"
    >
      <div className="flex items-center justify-between mb-3">
        <span className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800">
          {props.title}
        </span>
        {to && (
          <Link
            to={to}
            className="flex items-center gap-1 text-[0.72rem] font-semibold text-slate-500 hover:text-slate-900 transition-colors duration-150"
          >
            View all
            <svg width="11" height="11" viewBox="0 0 12 12" fill="none">
              <path
                d="M2.5 9.5L9.5 2.5M9.5 2.5H4.5M9.5 2.5V7.5"
                stroke="currentColor"
                strokeWidth="1.6"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </Link>
        )}
      </div>

      <div className="relative w-full" style={{ height: chartHeight }}>
        <ReactEChartsCore
          echarts={echarts}
          option={option}
          style={{ height: chartHeight, width: "100%" }}
        />

        <div
          className="absolute top-[4px] left-0 flex flex-col pointer-events-none"
          style={{ width: labelColumnWidth }}
        >
          {items.map((x, i) => {
            const nameNode = (
              <ResourceHoverCard itemRef={getResourceRef(x.resource)}>
                <Link
                  to={
                    to
                      ? `${to}?${getRefNameQueryArgStr(x.resource)}`
                      : getResourcePath(x.resource)
                  }
                  className="truncate text-[0.78rem] font-semibold text-slate-600 hover:text-slate-900 transition-colors duration-150 pointer-events-auto"
                >
                  {printResourceNameWithDisplay(x.resource)}
                </Link>
              </ResourceHoverCard>
            );

            return (
              <div
                key={x.resource.metadata?.uid}
                className="flex items-center pr-3"
                style={{ height: BAR_HEIGHT + BAR_GAP }}
              >
                <span className="text-[0.68rem] font-bold text-slate-400 w-5 shrink-0 tabular-nums">
                  {i + 1}
                </span>
                <div className="truncate min-w-0 flex-1">{nameNode}</div>
              </div>
            );
          })}
        </div>
      </div>
    </motion.div>
  );
};

export default TopList;
