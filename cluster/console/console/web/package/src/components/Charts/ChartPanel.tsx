import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import LineChart, { summarizePoints } from "./LineChart";

const formatNumber = (value: number) =>
  value.toLocaleString(undefined, { maximumFractionDigits: 1 });

export interface ChartPanelProps {
  caption: string;
  points?: { ts: Timestamp; value: number }[];
  variant?: "line" | "bar";
  color?: string;
  height?: number;
  tone?: "default" | "critical";
  emptyLabel?: string;
  action?: React.ReactNode;
}

const ChartPanel = (props: ChartPanelProps) => {
  const stats = summarizePoints(props.points);
  const critical = props.tone === "critical";

  return (
    <section
      className={twMerge(
        "rounded-xl border p-4",
        critical ? "border-red-200 bg-red-50/40" : "border-slate-200 bg-white",
      )}
    >
      <div className="mb-3 flex flex-wrap items-end justify-between gap-x-4 gap-y-2">
        <span
          className={twMerge(
            "text-micro font-semibold uppercase tracking-[0.07em]",
            critical ? "text-red-700" : "text-slate-500",
          )}
        >
          {props.caption}
        </span>

        <div className="flex items-center gap-4">
          {stats.count > 0 && (
            <dl className="flex items-center gap-4">
              {[
                { label: "Latest", value: stats.latest },
                { label: "Avg", value: stats.average },
                { label: "Peak", value: stats.peak },
              ].map((stat) => (
                <div key={stat.label} className="flex items-baseline gap-1.5">
                  <dt className="text-micro font-normal uppercase tracking-[0.06em] text-slate-500">
                    {stat.label}
                  </dt>
                  <dd className="text-xs font-semibold tabular-nums text-slate-700">
                    {formatNumber(stat.value)}
                  </dd>
                </div>
              ))}
            </dl>
          )}
          {props.action}
        </div>
      </div>

      <LineChart
        points={props.points}
        variant={props.variant}
        color={props.color}
        height={props.height}
        title={props.caption}
        emptyLabel={props.emptyLabel}
      />
    </section>
  );
};

export default ChartPanel;
