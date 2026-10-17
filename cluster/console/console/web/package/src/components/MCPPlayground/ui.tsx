import { CopyButton } from "@/pages/Agent/ui";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowLeft, type LucideIcon } from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";

const ease = [0.22, 1, 0.36, 1] as const;

export const Collapse = (props: {
  opened: boolean;
  children: React.ReactNode;
}) => (
  <AnimatePresence initial={false}>
    {props.opened && (
      <motion.div
        initial={{ height: 0, opacity: 0 }}
        animate={{ height: "auto", opacity: 1 }}
        exit={{ height: 0, opacity: 0 }}
        transition={{ duration: 0.25, ease }}
        className="overflow-hidden"
      >
        {props.children}
      </motion.div>
    )}
  </AnimatePresence>
);

export const CodeBlock = (props: {
  children: string;
  className?: string;
  wrap?: boolean;
}) => (
  <div className="group/code relative min-w-0">
    <pre
      className={twMerge(
        "max-h-[420px] overflow-auto rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs leading-5",
        props.className,
      )}
    >
      <code
        className={twMerge(
          "font-mono text-xs leading-5 text-slate-700",
          props.wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre",
        )}
      >
        {props.children}
      </code>
    </pre>
    <span className="absolute right-1.5 top-1.5 rounded-md bg-white opacity-0 shadow-sm transition-opacity focus-within:opacity-100 group-hover/code:opacity-100 [@media(hover:none)]:opacity-100">
      <CopyButton value={() => props.children} />
    </span>
  </div>
);

export const CardHeader = (props: {
  icon: LucideIcon;
  title: React.ReactNode;
  meta?: React.ReactNode;
  children?: React.ReactNode;
}) => (
  <div className="flex min-h-10 items-center gap-2 border-b border-slate-100 py-1.5 pl-3 pr-1.5">
    <props.icon size={13} className="shrink-0 text-slate-500" />
    <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
      {props.title}
    </span>
    {props.meta && (
      <span className="shrink-0 font-mono text-micro text-slate-400">
        {props.meta}
      </span>
    )}
    {props.children}
  </div>
);

export const EmptyState = (props: {
  icon: LucideIcon;
  title: string;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "flex flex-col items-center justify-center px-6 py-10 text-center",
      props.className,
    )}
  >
    <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
      <props.icon size={18} strokeWidth={2} />
    </span>
    <p className="mt-3 text-body font-semibold text-slate-800">{props.title}</p>
    {props.description && (
      <p className="mt-1 max-w-sm text-xs leading-5 text-slate-500">
        {props.description}
      </p>
    )}
    {props.action && <div className="mt-4">{props.action}</div>}
  </div>
);

export const BackButton = (props: { label: string; onClick: () => void }) => (
  <button
    type="button"
    onClick={props.onClick}
    className="-ml-1 flex w-fit cursor-pointer items-center gap-1 rounded-md px-1 py-0.5 text-xs font-semibold text-slate-500 transition-colors hover:text-slate-900 @3xl:hidden"
  >
    <ArrowLeft size={13} strokeWidth={2.4} />
    {props.label}
  </button>
);

export const Description = (props: { text: string }) => {
  const [expanded, setExpanded] = React.useState(false);
  const isLong = props.text.length > 280 || props.text.split("\n").length > 5;

  return (
    <div className="mt-1.5">
      <p
        className={twMerge(
          "whitespace-pre-wrap break-words text-xs leading-5 text-slate-600",
          isLong && !expanded && "line-clamp-4",
        )}
      >
        {props.text}
      </p>
      {isLong && (
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          className="mt-1 cursor-pointer text-micro font-semibold text-slate-500 hover:text-slate-900"
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      )}
    </div>
  );
};
