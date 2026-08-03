import { formatNumber } from "@/utils";
import { getResourcePath, printResourceNameWithDisplay, Resource } from "@/utils/pb";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowUpRight, ChevronDown } from "lucide-react";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const COLLAPSED_COUNT = 5;

const TopList = (props: {
  title: string;
  items?: { resource: Resource; count: number }[];
  to?: string;
}) => {
  const [expanded, setExpanded] = React.useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const items = props.items ?? [];

  if (items.length === 0) return null;

  const topCount = Math.max(items[0]?.count ?? 0, 1);
  const primaryItems = items.slice(0, COLLAPSED_COUNT);
  const additionalItems = items.slice(COLLAPSED_COUNT);
  const returnTo = `${location.pathname}${location.search}`;

  const renderItem = (
    item: { resource: Resource; count: number },
    index: number,
  ) => {
    const resource = item.resource;
    const md = resource.metadata!;
    const percentage = Math.max(0, Math.min(100, (item.count / topCount) * 100));

    return (
      <motion.button
        layout
        key={md.uid || `${resource.kind}-${md.name}`}
        type="button"
        onClick={() =>
          navigate(getResourcePath(resource), {
            state: { returnTo },
            preventScrollReset: true,
          })
        }
        className={twMerge(
          "group relative flex min-h-12 w-full cursor-pointer items-center gap-3 overflow-hidden rounded-lg border border-slate-200/80 bg-white px-3 py-2 text-left",
          "shadow-[0_1px_2px_rgba(15,23,42,0.035)] outline-none",
          "transition-[border-color,box-shadow,background-color] duration-500 ease-out",
          "hover:border-slate-300 hover:bg-slate-50/40 hover:shadow-[0_3px_10px_rgba(15,23,42,0.065)]",
          "focus-visible:border-blue-400 focus-visible:ring-2 focus-visible:ring-blue-500/20",
        )}
      >
        <span
          aria-hidden="true"
          className="absolute inset-y-0 left-0 bg-blue-50/70 transition-[width,background-color] duration-700 ease-out group-hover:bg-blue-100/60"
          style={{ width: `${percentage}%` }}
        />

        <span className="relative flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-white text-[0.68rem] font-bold tabular-nums text-slate-500 shadow-sm">
          {index + 1}
        </span>

        <span className="relative flex min-w-0 flex-1 items-center gap-2.5">
          {md.picURL && (
            <img
              src={md.picURL}
              alt=""
              loading="lazy"
              className="h-9 w-9 shrink-0 rounded-lg border border-slate-200 bg-white object-cover shadow-sm"
            />
          )}
          <span className="flex min-w-0 flex-1 flex-col justify-center gap-0.5">
            <span className="truncate text-[0.76rem] font-bold text-slate-700 transition-colors duration-500 group-hover:text-slate-900">
              {printResourceNameWithDisplay(resource)}
            </span>
            <span className="truncate text-[0.62rem] font-semibold uppercase tracking-[0.05em] text-slate-400">
              {resource.kind}
            </span>
          </span>
        </span>

        <span className="relative flex shrink-0 items-center gap-2">
          <span className="text-[0.74rem] font-bold tabular-nums text-slate-700">
            {formatNumber(item.count)}
          </span>
          <ArrowUpRight
            size={13}
            aria-hidden="true"
            className="text-slate-300 transition-[color,transform] duration-500 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-blue-500"
          />
        </span>
      </motion.button>
    );
  };

  return (
    <motion.section
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      className="w-full rounded-xl border border-slate-200 bg-slate-50/50 p-3 shadow-[0_1px_4px_rgba(15,23,42,0.04)]"
    >
      <div className="mb-3 flex items-center justify-between gap-3 px-1">
        <div className="flex min-w-0 items-center gap-2">
          <h3 className="truncate text-[0.75rem] font-bold uppercase tracking-[0.06em] text-slate-700">
            {props.title}
          </h3>
          <span className="rounded bg-slate-200/70 px-1.5 py-0.5 text-[0.62rem] font-bold tabular-nums text-slate-500">
            {items.length}
          </span>
        </div>

        {props.to && (
          <Link
            to={props.to}
            preventScrollReset
            className="shrink-0 rounded-md px-1.5 py-1 text-[0.68rem] font-bold text-slate-500 outline-none transition-colors duration-500 hover:bg-white hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500/30"
          >
            View all
          </Link>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        {primaryItems.map((item, index) => renderItem(item, index))}

        <AnimatePresence initial={false}>
          {expanded && additionalItems.length > 0 && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
              className="flex flex-col gap-1.5 overflow-hidden"
            >
              {additionalItems.map((item, index) =>
                renderItem(item, index + COLLAPSED_COUNT),
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {additionalItems.length > 0 && (
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          aria-expanded={expanded}
          className="mt-2 flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-[0.7rem] font-bold text-slate-500 outline-none transition-[background-color,color] duration-500 hover:bg-white hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-blue-500/25"
        >
          {expanded
            ? "Show less"
            : `Show ${additionalItems.length} more`}
          <ChevronDown
            size={13}
            aria-hidden="true"
            className={twMerge(
              "transition-transform duration-500",
              expanded && "rotate-180",
            )}
          />
        </button>
      )}
    </motion.section>
  );
};

export default TopList;
