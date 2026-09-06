import { Loader } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Inbox } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

export const Panel = (props: {
  title: string;
  description?: string;
  icon?: React.ElementType<{ size?: number; strokeWidth?: number }>;
  action?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
}) => {
  const Icon = props.icon;
  return (
    <motion.section
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.24, ease: "easeOut" }}
      className={twMerge(
        "w-full overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_4px_rgba(15,23,42,0.04)]",
        props.className,
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
        <div className="flex min-w-0 items-center gap-2.5">
          {Icon && (
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
              <Icon size={14} strokeWidth={2.2} />
            </span>
          )}
          <div className="min-w-0">
            <h3 className="truncate text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800">
              {props.title}
            </h3>
            {props.description && (
              <p className="mt-0.5 truncate text-[0.68rem] font-semibold text-slate-400">
                {props.description}
              </p>
            )}
          </div>
        </div>
        {props.action}
      </div>
      <div className="px-4 py-4">{props.children}</div>
    </motion.section>
  );
};

export const LazySection = (props: {
  title: string;
  description?: string;
  icon?: React.ElementType<{ size?: number; strokeWidth?: number }>;
  defaultOpen?: boolean;
  badge?: React.ReactNode;
  children: (opened: boolean) => React.ReactNode;
}) => {
  const [opened, setOpened] = React.useState(props.defaultOpen ?? false);
  const [everOpened, setEverOpened] = React.useState(props.defaultOpen ?? false);
  const Icon = props.icon;

  const toggle = () => {
    setOpened((value) => {
      if (!value) setEverOpened(true);
      return !value;
    });
  };

  return (
    <motion.section
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.24, ease: "easeOut" }}
      className="w-full overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_4px_rgba(15,23,42,0.04)]"
    >
      <button
        type="button"
        onClick={toggle}
        aria-expanded={opened}
        className="flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left outline-none transition-colors duration-300 hover:bg-slate-50/70 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500/25"
      >
        {Icon && (
          <span
            className={twMerge(
              "flex h-7 w-7 shrink-0 items-center justify-center rounded-lg transition-colors duration-300",
              opened ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-500",
            )}
          >
            <Icon size={14} strokeWidth={2.2} />
          </span>
        )}
        <span className="min-w-0 flex-1">
          <span className="flex items-center gap-2">
            <span className="truncate text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-800">
              {props.title}
            </span>
            {props.badge}
          </span>
          {props.description && (
            <span className="mt-0.5 block truncate text-[0.68rem] font-semibold text-slate-400">
              {props.description}
            </span>
          )}
        </span>
        <ChevronDown
          size={15}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-slate-400 transition-transform duration-400",
            opened && "rotate-180",
          )}
        />
      </button>
      <AnimatePresence initial={false}>
        {opened && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.34, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden border-t border-slate-100"
          >
            <div className="px-4 py-4">{props.children(everOpened)}</div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.section>
  );
};

export const StatCard = (props: {
  label: string;
  value: string;
  hint?: string;
  tone?: "default" | "positive" | "warning" | "danger";
  icon?: React.ElementType<{ size?: number; strokeWidth?: number }>;
  onClick?: () => void;
  active?: boolean;
}) => {
  const Icon = props.icon;
  const tone = props.tone ?? "default";
  const content = (
    <>
      {Icon && (
        <span
          aria-hidden="true"
          className="pointer-events-none absolute -right-2 -top-2 text-slate-100"
        >
          <Icon size={54} strokeWidth={1.4} />
        </span>
      )}
      <span className="relative z-10 truncate text-[0.66rem] font-bold uppercase tracking-[0.07em] text-slate-400">
        {props.label}
      </span>
      <span
        className={twMerge(
          "relative z-10 mt-1 truncate text-[1.4rem] font-bold leading-7 tracking-[-0.03em] tabular-nums",
          tone === "positive" && "text-emerald-600",
          tone === "warning" && "text-amber-600",
          tone === "danger" && "text-red-600",
          tone === "default" && "text-slate-800",
        )}
      >
        {props.value}
      </span>
      {props.hint && (
        <span className="relative z-10 mt-0.5 truncate text-[0.66rem] font-semibold text-slate-400">
          {props.hint}
        </span>
      )}
    </>
  );

  const className = twMerge(
    "group relative flex min-h-[86px] min-w-0 flex-col justify-center overflow-hidden rounded-xl border bg-white px-3.5 py-3",
    "shadow-[0_1px_2px_rgba(15,23,42,0.035)] transition-[border-color,box-shadow,background-color] duration-400",
    props.active
      ? "border-slate-800 ring-1 ring-slate-900/10"
      : "border-slate-200",
    props.onClick &&
      "cursor-pointer text-left hover:border-slate-300 hover:shadow-[0_4px_12px_rgba(15,23,42,0.07)]",
  );

  if (props.onClick) {
    return (
      <button type="button" onClick={props.onClick} className={className}>
        {content}
      </button>
    );
  }
  return <div className={className}>{content}</div>;
};

export const StatGrid = (props: { children?: React.ReactNode }) => (
  <div className="grid w-full grid-cols-[repeat(auto-fill,minmax(158px,1fr))] gap-2.5">
    {props.children}
  </div>
);

export const QueryState = (props: {
  isLoading: boolean;
  isError: boolean;
  isEmpty?: boolean;
  emptyLabel?: string;
  minHeight?: number;
  children: React.ReactNode;
}) => {
  if (props.isLoading) {
    return (
      <div
        className="flex w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/50"
        style={{ minHeight: props.minHeight ?? 160 }}
      >
        <Loader size="sm" color="gray" />
      </div>
    );
  }

  if (props.isError) {
    return (
      <div
        className="flex w-full items-center justify-center rounded-xl border border-dashed border-red-200 bg-red-50/50 px-4 text-center"
        style={{ minHeight: props.minHeight ?? 160 }}
      >
        <p className="text-xs font-bold text-red-600">
          The visibility query failed
        </p>
      </div>
    );
  }

  if (props.isEmpty) {
    return (
      <div
        className="flex w-full flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-slate-200 bg-slate-50/50 px-4 text-center"
        style={{ minHeight: props.minHeight ?? 160 }}
      >
        <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-400">
          <Inbox size={16} strokeWidth={2} />
        </span>
        <p className="text-xs font-semibold text-slate-400">
          {props.emptyLabel ?? "No inference activity in this range"}
        </p>
      </div>
    );
  }

  return <>{props.children}</>;
};
