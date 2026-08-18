import { formatNumber } from "@/utils";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight, Inbox } from "lucide-react";
import { useRef } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

type SummaryIcon = React.ElementType<{
  className?: string;
  size?: number | string;
  strokeWidth?: number | string;
}>;

export const SummaryItemCount = (props: {
  children?: React.ReactNode;
  count?: number;
  to?: string;
  active?: boolean;
  icon?: SummaryIcon;
}) => {
  const { count, children, active, to, icon } = props;
  const Icon = icon;
  const prevCountRef = useRef<number | undefined>(undefined);

  if (!count || count < 1) return null;

  const prev = prevCountRef.current;
  const direction = prev === undefined || count >= prev ? 1 : -1;
  prevCountRef.current = count;

  const content = (
    <>
      <div className="flex min-w-0 items-start justify-between gap-2">
        <div className="h-8 min-w-0 overflow-hidden">
          <AnimatePresence mode="popLayout" initial={false}>
            <motion.span
              key={count}
              initial={{ y: `${direction * 110}%`, opacity: 0 }}
              animate={{ y: "0%", opacity: 1 }}
              exit={{ y: `${direction * -110}%`, opacity: 0 }}
              transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
              className={twMerge(
                "block truncate text-[1.55rem] font-bold leading-8 tracking-[-0.035em] tabular-nums",
                active ? "text-slate-900" : "text-slate-700",
              )}
            >
              {formatNumber(count)}
            </motion.span>
          </AnimatePresence>
        </div>

        {active ? (
          <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-slate-900 ring-4 ring-slate-900/10" />
        ) : to ? (
          <ArrowRight
            size={13}
            strokeWidth={2.5}
            className="mt-1 shrink-0 text-slate-300 transition-[color,transform] duration-500 group-hover:translate-x-0.5 group-hover:text-slate-600"
          />
        ) : null}
      </div>

      <span
        className={twMerge(
          "truncate text-[0.72rem] font-bold leading-4 text-slate-500 transition-colors duration-500",
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
        "shadow-[0_1px_2px_rgba(15,23,42,0.035)]",
        "transition-[background-color,border-color,box-shadow] duration-500 ease-out",
        active
          ? "border-slate-300 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.07)] ring-1 ring-slate-900/[0.04]"
          : to
            ? "border-slate-200 hover:border-slate-300 hover:shadow-[0_4px_12px_rgba(15,23,42,0.07)]"
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
          className="relative z-10 flex min-h-[76px] w-full flex-col justify-center gap-1 px-3.5 py-2.5 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-500"
        >
          {content}
        </Link>
      ) : (
        <div className="relative z-10 flex min-h-[76px] w-full flex-col justify-center gap-1 px-3.5 py-2.5">
          {content}
        </div>
      )}
    </motion.div>
  );
};

export const SummaryItemCountWrap = (props: { children?: React.ReactNode }) => (
  <div className="grid w-full grid-cols-[repeat(auto-fill,minmax(140px,1fr))] gap-2">
    {props.children}
  </div>
);

export const SummaryNoItems = (props: { children?: React.ReactNode }) => (
  <div className="flex min-h-[190px] w-full items-center justify-center rounded-xl border border-dashed border-slate-300 bg-slate-50/50 px-6 text-center">
    <div className="flex flex-col items-center gap-3">
      <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-400 shadow-sm">
        <Inbox size={18} strokeWidth={2} />
      </span>
      <div className="flex flex-col gap-1">
        <span className="text-sm font-bold text-slate-600">
          {props.children ?? "No items found"}
        </span>
        <span className="text-[0.72rem] font-semibold text-slate-400">
          Resources will appear here when they become available.
        </span>
      </div>
    </div>
  </div>
);
