import { formatNumber } from "@/utils";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight, Inbox } from "lucide-react";
import { createContext, useContext, useRef } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

type SummaryIcon = React.ElementType<{
  className?: string;
  size?: number | string;
  strokeWidth?: number | string;
}>;

const CompactSummaryContext = createContext(false);

export const CompactSummary = (props: { children: React.ReactNode }) => (
  <CompactSummaryContext.Provider value>
    {props.children}
  </CompactSummaryContext.Provider>
);

export const SummaryItemCount = (props: {
  children?: React.ReactNode;
  count?: number;
  to?: string;
  active?: boolean;
  icon?: SummaryIcon;
  showZero?: boolean;
  formatCount?: (count: number) => React.ReactNode;
}) => {
  const {
    count,
    children,
    active,
    to,
    icon,
    showZero = false,
    formatCount,
  } = props;
  const Icon = icon;
  const isCompact = useContext(CompactSummaryContext);
  const prevCountRef = useRef<number | undefined>(undefined);

  if (count === undefined || count < 0 || (!showZero && count < 1)) return null;

  const prev = prevCountRef.current;
  const direction = prev === undefined || count >= prev ? 1 : -1;
  prevCountRef.current = count;

  const content = (
    <>
      <div className="flex min-w-0 items-start justify-between gap-2">
        <div
          className={twMerge(
            "min-w-0 overflow-hidden",
            isCompact ? "h-7" : "h-8",
          )}
        >
          <AnimatePresence mode="popLayout" initial={false}>
            <motion.span
              key={count}
              initial={{ y: `${direction * 110}%`, opacity: 0 }}
              animate={{ y: "0%", opacity: 1 }}
              exit={{ y: `${direction * -110}%`, opacity: 0 }}
              transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
              className={twMerge(
                "block truncate font-bold tracking-[-0.035em] tabular-nums",
                isCompact ? "text-xl leading-7" : "text-2xl leading-8",
                active ? "text-slate-900" : "text-slate-700",
              )}
            >
              {formatCount ? formatCount(count) : formatNumber(count)}
            </motion.span>
          </AnimatePresence>
        </div>

        {active ? (
          <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-slate-900 ring-4 ring-slate-900/10" />
        ) : to ? (
          <ArrowRight
            size={13}
            strokeWidth={2.5}
            className="mt-1 shrink-0 text-slate-300 transition-[color,transform] duration-200 group-hover:translate-x-0.5 group-hover:text-slate-600"
          />
        ) : null}
      </div>

      <span
        className={twMerge(
          "truncate text-xs font-normal leading-4 text-slate-500 transition-colors duration-200",
          to && !active && "group-hover:text-slate-800",
          active && "text-slate-700",
        )}
      >
        {children}
      </span>
    </>
  );

  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className={twMerge(
        "group relative min-w-0 overflow-hidden rounded-lg border bg-white",
        "shadow-card",
        "transition-[background-color,border-color,box-shadow] duration-200 ease-out",
        active
          ? "border-slate-300 bg-slate-50 shadow-raised ring-1 ring-slate-900/[0.04]"
          : to
            ? "border-slate-200 hover:border-slate-300 hover:shadow-raised"
            : "border-slate-200",
      )}
    >
      {icon && (
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 z-0 flex items-center justify-center text-slate-200/75 [&>svg]:h-12 [&>svg]:w-12"
        >
          {Icon && <Icon size={48} strokeWidth={1.5} />}
        </div>
      )}
      {to && !active ? (
        <Link
          to={to}
          className={twMerge(
            "relative z-10 flex w-full flex-col justify-center gap-1 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-500",
            isCompact
              ? "min-h-[58px] px-3 py-2"
              : "min-h-[76px] px-3.5 py-2.5",
          )}
        >
          {content}
        </Link>
      ) : (
        <div
          className={twMerge(
            "relative z-10 flex w-full flex-col justify-center gap-1",
            isCompact
              ? "min-h-[58px] px-3 py-2"
              : "min-h-[76px] px-3.5 py-2.5",
          )}
        >
          {content}
        </div>
      )}
    </motion.div>
  );
};

export const SummaryItemCountWrap = (props: { children?: React.ReactNode }) => {
  const isCompact = useContext(CompactSummaryContext);
  return (
    <div
      className={twMerge(
        "grid w-full",
        isCompact
          ? "grid-cols-[repeat(auto-fill,minmax(120px,1fr))] gap-1.5"
          : "grid-cols-[repeat(auto-fill,minmax(140px,1fr))] gap-2",
      )}
    >
      {props.children}
    </div>
  );
};

export const SummaryNoItems = (props: { children?: React.ReactNode }) => (
  <div className="flex min-h-[190px] w-full items-center justify-center rounded-xl border border-dashed border-slate-300 bg-slate-50/50 px-6 text-center">
    <div className="flex flex-col items-center gap-3">
      <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500 shadow-sm">
        <Inbox size={18} strokeWidth={2} />
      </span>
      <div className="flex flex-col gap-1">
        <span className="text-sm font-bold text-slate-600">
          {props.children ?? "No items found"}
        </span>
        <span className="text-xs font-normal text-slate-500">
          Resources will appear here when they become available.
        </span>
      </div>
    </div>
  </div>
);
