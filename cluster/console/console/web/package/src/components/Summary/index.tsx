import { formatNumber } from "@/utils";
import { AnimatePresence, motion } from "framer-motion";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

import { useRef } from "react";

export const SummaryItemCount = (props: {
  children?: React.ReactNode;
  count?: number;
  to?: string;
  active?: boolean;
}) => {
  const { count, children, active, to } = props;
  const prevCountRef = useRef<number | undefined>(undefined);

  if (!count || count < 1) return null;

  const prev = prevCountRef.current;
  const direction = prev === undefined || count >= prev ? 1 : -1;
  prevCountRef.current = count;

  const labelContent = (
    <span
      className={twMerge(
        "text-[0.78rem] font-semibold tracking-[0.01em] transition-colors duration-150 whitespace-nowrap",
        active
          ? "text-slate-600"
          : to
            ? "text-slate-500 group-hover:text-slate-800 group-hover:underline underline-offset-2"
            : "text-slate-500",
      )}
    >
      {children}
      {to && !active && (
        <span className="ml-0.5 text-[0.62rem] opacity-50">↗</span>
      )}
    </span>
  );

  return (
    <motion.div
      initial={{ opacity: 0, x: -4 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className={twMerge(
        "group relative flex flex-col gap-1",
        "pl-4 pr-5 py-2.5 min-w-[120px]",
        "border-l-[3px] transition-[border-color] duration-150",
        active
          ? "border-l-slate-900"
          : "border-l-slate-500 hover:border-l-slate-700",
      )}
    >
      <div className="overflow-hidden h-[2.1rem]">
        <AnimatePresence mode="popLayout" initial={false}>
          <motion.span
            key={count}
            initial={{ y: `${direction * 110}%`, opacity: 0 }}
            animate={{ y: "0%", opacity: 1 }}
            exit={{ y: `${direction * -110}%`, opacity: 0 }}
            transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
            className={twMerge(
              "block text-[1.65rem] font-bold leading-none tracking-[-0.03em] tabular-nums",
              active ? "text-slate-900" : "text-slate-600",
            )}
          >
            {formatNumber(count)}
          </motion.span>
        </AnimatePresence>
      </div>

      <div className="h-[1.1rem] flex items-center">
        {to && !active ? (
          <Link to={to} className="contents">
            {labelContent}
          </Link>
        ) : (
          labelContent
        )}
      </div>
    </motion.div>
  );
};

export const SummaryItemCountWrap = (props: { children?: React.ReactNode }) => {
  return (
    <div className="flex flex-wrap items-stretch divide-x divide-slate-200">
      {props.children}
    </div>
  );
};

export const SummaryNoItems = (props: { children?: React.ReactNode }) => {
  return (
    <div className="w-full flex items-center justify-center h-[200px]">
      <span className="font-bold text-xl text-slate-400 tracking-wide select-none">
        No items found
      </span>
    </div>
  );
};
