import ReactEChartsCore from "echarts-for-react";
import { PieChart as PieChartC } from "echarts/charts";
import { AriaComponent, TooltipComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useMemo } from "react";

echarts.use([PieChartC, CanvasRenderer, AriaComponent, TooltipComponent]);

const SLICE_COLORS = [
  "#2563eb",
  "#0891b2",
  "#4f46e5",
  "#0d9488",
  "#7c3aed",
  "#16a34a",
  "#d97706",
  "#db2777",
  "#475569",
  "#dc2626",
];

type PieChartItem = {
  name: string;
  value: number;
};

type PieChartProps = {
  data: PieChartItem[];
  title?: string;
};

const formatNumber = (value: number) => value.toLocaleString();

const escapeHTML = (value: string) =>
  value.replace(
    /[&<>'"]/g,
    (character) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#39;",
        '"': "&quot;",
      })[character] ?? character,
  );

const PieChart = ({ data, title }: PieChartProps) => {
  const chartData = useMemo(
    () =>
      data
        .filter(
          (item) =>
            item.name.trim().length > 0 &&
            Number.isFinite(item.value) &&
            item.value > 0,
        )
        .map((item, index) => ({
          ...item,
          itemStyle: {
            color: SLICE_COLORS[index % SLICE_COLORS.length],
          },
        })),
    [data],
  );

  const total = useMemo(
    () => chartData.reduce((sum, item) => sum + item.value, 0),
    [chartData],
  );

  const option = useMemo(
    () => ({
      animationDuration: 650,
      animationEasing: "cubicOut",
      aria: {
        enabled: true,
        decal: { show: false },
      },
      tooltip: {
        trigger: "item",
        confine: true,
        backgroundColor: "#0f172a",
        borderColor: "#334155",
        borderWidth: 1,
        padding: [10, 12],
        textStyle: {
          color: "#f8fafc",
          fontFamily: "Ubuntu, sans-serif",
          fontSize: 12,
        },
        extraCssText:
          "border-radius:10px;box-shadow:0 10px 24px rgba(15,23,42,.18);",
        formatter: (params: { name: string; value: number }) => {
          const percentage = ((params.value / total) * 100).toFixed(1);
          return `<div style="min-width:112px"><div style="margin-bottom:5px;color:#cbd5e1;font-size:11px;font-weight:700">${escapeHTML(params.name)}</div><div style="display:flex;align-items:baseline;justify-content:space-between;gap:16px"><span style="font-size:16px;font-weight:700">${formatNumber(params.value)}</span><span style="color:#93c5fd;font-size:11px;font-weight:700">${percentage}%</span></div></div>`;
        },
      },
      series: [
        {
          name: title ?? "Distribution",
          type: "pie",
          radius: ["58%", "82%"],
          center: ["50%", "50%"],
          minAngle: 2,
          padAngle: 2,
          itemStyle: {
            borderColor: "#f8fafc",
            borderRadius: 4,
            borderWidth: 2,
          },
          label: {
            show: true,
            position: "center",
            formatter: `{value|${formatNumber(total)}}\n{label|TOTAL}`,
            rich: {
              value: {
                color: "#0f172a",
                fontFamily: "Ubuntu, sans-serif",
                fontSize: 21,
                fontWeight: 700,
                lineHeight: 28,
              },
              label: {
                color: "#94a3b8",
                fontFamily: "Ubuntu, sans-serif",
                fontSize: 9,
                fontWeight: 700,
                letterSpacing: 1.4,
                lineHeight: 14,
              },
            },
          },
          labelLine: { show: false },
          emphasis: {
            scale: true,
            scaleSize: 5,
            itemStyle: {
              shadowBlur: 12,
              shadowColor: "rgba(15, 23, 42, 0.14)",
            },
          },
          data: chartData,
        },
      ],
    }),
    [chartData, title, total],
  );

  if (total === 0) {
    return (
      <div className="flex min-h-40 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center">
        <div>
          {title && (
            <p className="mb-1 text-xs font-bold text-slate-700">{title}</p>
          )}
          <p className="text-xs font-semibold text-slate-400">
            No data available
          </p>
        </div>
      </div>
    );
  }

  return (
    <section
      className="w-full rounded-xl border border-slate-200/80 bg-slate-50/70 p-3.5 sm:p-4"
      aria-label={title ?? "Distribution chart"}
    >
      {title && (
        <p className="mb-2 text-xs font-bold tracking-tight text-slate-800">
          {title}
        </p>
      )}

      <div className="flex flex-wrap items-center justify-center gap-x-5 gap-y-3">
        <div className="h-44 min-w-40 flex-[0_1_190px]">
          <ReactEChartsCore
            echarts={echarts}
            option={option}
            style={{ height: "100%", width: "100%" }}
            notMerge
            lazyUpdate
          />
        </div>

        <dl className="grid min-w-44 flex-1 grid-cols-1 gap-1.5 sm:min-w-48">
          {chartData.map((item, index) => {
            const percentage = (item.value / total) * 100;

            return (
              <div
                key={item.name}
                className="group flex min-w-0 items-center gap-2.5 rounded-lg px-2.5 py-2 transition-colors duration-500 hover:bg-white"
              >
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full ring-2 ring-white"
                  style={{
                    backgroundColor:
                      SLICE_COLORS[index % SLICE_COLORS.length],
                  }}
                />
                <dt className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-600">
                  {item.name}
                </dt>
                <dd className="flex shrink-0 items-baseline gap-2">
                  <span className="text-xs font-bold tabular-nums text-slate-800">
                    {formatNumber(item.value)}
                  </span>
                  <span className="w-10 text-right text-[0.65rem] font-bold tabular-nums text-slate-400">
                    {percentage.toFixed(1)}%
                  </span>
                </dd>
              </div>
            );
          })}
        </dl>
      </div>
    </section>
  );
};

export default PieChart;
