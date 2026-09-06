import { ObjectReference } from "@/apis/metav1/metav1";
import { Stats } from "@/apis/visibilityv1/llm/vllmv1";
import { SERIES_COLORS } from "@/components/Charts/SeriesChart";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import {
  formatMetricValue,
  MetricAccessor,
  num,
  prettyKey,
  ratio,
} from "./utils";

export interface StatBarItem {
  key: string;
  ref?: ObjectReference;
  stats?: Stats;
}

const COLLAPSED_COUNT = 6;

const StatBars = (props: {
  items: StatBarItem[];
  accessor: MetricAccessor;
  other?: Stats;
  totalCount?: number;
  colored?: boolean;
  onSelect?: (item: StatBarItem) => void;
  renderLabel?: (item: StatBarItem) => React.ReactNode;
}) => {
  const [expanded, setExpanded] = React.useState(false);

  const rows = React.useMemo(() => {
    const base = props.items.map((item) => ({
      item,
      value: props.accessor.get(item.stats),
    }));
    if (props.other) {
      base.push({
        item: { key: "__other__", stats: props.other },
        value: props.accessor.get(props.other),
      });
    }
    return base;
  }, [props.accessor, props.items, props.other]);

  const total = React.useMemo(
    () => rows.reduce((sum, row) => sum + row.value, 0),
    [rows],
  );
  const peak = React.useMemo(
    () => rows.reduce((max, row) => Math.max(max, row.value), 0),
    [rows],
  );

  if (rows.length === 0) return null;

  const primary = rows.slice(0, COLLAPSED_COUNT);
  const rest = rows.slice(COLLAPSED_COUNT);

  const renderRow = (
    row: { item: StatBarItem; value: number },
    index: number,
  ) => {
    const isOther = row.item.key === "__other__";
    const width = peak <= 0 ? 0 : Math.max(1.5, (row.value / peak) * 100);
    const share = ratio(row.value, total);
    const color = props.colored
      ? SERIES_COLORS[index % SERIES_COLORS.length]
      : "#2563eb";

    const label = isOther
      ? "Other"
      : (props.renderLabel?.(row.item) ?? prettyKey(row.item.key));

    const inner = (
      <>
        <span
          aria-hidden="true"
          className="absolute inset-y-0 left-0 rounded-lg opacity-[0.12] transition-[width] duration-700 ease-out"
          style={{ width: `${width}%`, backgroundColor: color }}
        />
        <span className="relative flex min-w-0 flex-1 items-center gap-2.5">
          <span
            aria-hidden="true"
            className="h-2 w-2 shrink-0 rounded-full ring-2 ring-white"
            style={{ backgroundColor: isOther ? "#cbd5e1" : color }}
          />
          <span
            className={twMerge(
              "truncate text-[0.73rem] font-bold",
              isOther ? "text-slate-400" : "text-slate-700",
            )}
          >
            {label}
          </span>
          {row.item.ref?.name && !isOther && (
            <span className="truncate text-[0.62rem] font-semibold uppercase tracking-[0.05em] text-slate-400">
              {row.item.ref.name}
            </span>
          )}
        </span>
        <span className="relative flex shrink-0 items-baseline gap-2.5">
          <span className="text-[0.73rem] font-bold tabular-nums text-slate-800">
            {formatMetricValue(props.accessor.kind, row.value)}
          </span>
          <span className="w-11 text-right text-[0.63rem] font-bold tabular-nums text-slate-400">
            {share.toFixed(1)}%
          </span>
        </span>
      </>
    );

    const className =
      "group relative flex min-h-9 w-full items-center gap-3 overflow-hidden rounded-lg border border-transparent px-2.5 py-1.5 transition-[border-color,background-color] duration-400 hover:border-slate-200 hover:bg-slate-50/60";

    if (props.onSelect && !isOther) {
      return (
        <button
          key={row.item.key}
          type="button"
          onClick={() => props.onSelect?.(row.item)}
          className={twMerge(className, "cursor-pointer text-left")}
        >
          {inner}
        </button>
      );
    }

    return (
      <div key={row.item.key} className={className}>
        {inner}
      </div>
    );
  };

  return (
    <div className="flex w-full flex-col gap-0.5">
      {primary.map(renderRow)}
      <AnimatePresence initial={false}>
        {expanded && rest.length > 0 && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
            className="flex flex-col gap-0.5 overflow-hidden"
          >
            {rest.map((row, index) => renderRow(row, index + COLLAPSED_COUNT))}
          </motion.div>
        )}
      </AnimatePresence>
      {rest.length > 0 && (
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          aria-expanded={expanded}
          className="mt-1.5 flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-[0.68rem] font-bold text-slate-500 outline-none transition-colors duration-400 hover:bg-slate-50 hover:text-slate-800"
        >
          {expanded ? "Show less" : `Show ${rest.length} more`}
          <ChevronDown
            size={12}
            className={twMerge(
              "transition-transform duration-400",
              expanded && "rotate-180",
            )}
          />
        </button>
      )}
      {props.totalCount !== undefined &&
        props.totalCount > props.items.length && (
          <p className="mt-1 px-2.5 text-[0.63rem] font-semibold text-slate-400">
            {num(props.totalCount).toLocaleString()} distinct values in range
          </p>
        )}
    </div>
  );
};

export default StatBars;
