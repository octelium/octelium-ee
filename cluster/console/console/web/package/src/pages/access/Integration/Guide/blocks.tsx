import { twMerge } from "@/lib/utils";
import { CopyButton, Tooltip } from "@mantine/core";
import { motion } from "framer-motion";
import { Check, ChevronRight, Copy, LucideIcon } from "lucide-react";
import * as React from "react";

export const P = (props: { children: React.ReactNode; className?: string }) => (
  <p
    className={twMerge(
      "max-w-3xl text-body font-normal leading-6 text-slate-600",
      props.className,
    )}
  >
    {props.children}
  </p>
);

export const Strong = (props: { children: React.ReactNode }) => (
  <strong className="font-semibold text-slate-800">{props.children}</strong>
);

export const Code = (props: { children: React.ReactNode }) => (
  <code className="rounded border border-slate-200 bg-slate-50 px-1 py-px font-mono text-[0.78rem] text-slate-800">
    {props.children}
  </code>
);

export const Path = (props: { items: string[] }) => (
  <span className="inline-flex flex-wrap items-center gap-0.5 rounded-md border border-slate-200 bg-white px-1.5 py-px align-middle text-xs font-semibold text-slate-700 shadow-card">
    {props.items.map((item, index) => (
      <React.Fragment key={`${item}-${index}`}>
        {index > 0 && (
          <ChevronRight
            size={11}
            strokeWidth={2.5}
            aria-hidden="true"
            className="text-slate-400"
          />
        )}
        <span>{item}</span>
      </React.Fragment>
    ))}
  </span>
);

export const Steps = (props: { children: React.ReactNode }) => (
  <ol className="flex max-w-3xl list-none flex-col gap-2 [counter-reset:guide-step]">
    {props.children}
  </ol>
);

export const Step = (props: { children: React.ReactNode }) => (
  <li className="relative pl-7 text-body leading-6 text-slate-600 [counter-increment:guide-step] before:absolute before:left-0 before:top-0.5 before:flex before:h-5 before:w-5 before:items-center before:justify-center before:rounded-full before:border before:border-slate-200 before:bg-slate-50 before:text-micro before:font-semibold before:text-slate-600 before:content-[counter(guide-step)]">
    {props.children}
  </li>
);

export const Bullets = (props: { children: React.ReactNode }) => (
  <ul className="flex max-w-3xl flex-col gap-1.5 pl-1">{props.children}</ul>
);

export const Bullet = (props: { children: React.ReactNode }) => (
  <li className="relative pl-4 text-body leading-6 text-slate-600 before:absolute before:left-0 before:top-[0.6rem] before:h-1.5 before:w-1.5 before:rounded-full before:bg-slate-300">
    {props.children}
  </li>
);

export const CodeBlock = (props: {
  code: string;
  language?: string;
  title?: string;
  className?: string;
}) => (
  <div
    className={twMerge(
      "max-w-3xl overflow-hidden rounded-lg border border-code-border bg-terminal",
      props.className,
    )}
  >
    <div className="flex items-center justify-between gap-2 border-b border-terminal-line px-3 py-1.5">
      <span className="text-micro font-semibold uppercase tracking-[0.06em] text-terminal-muted">
        {props.title ?? props.language ?? "Example"}
      </span>
      <CopyButton value={props.code}>
        {({ copied, copy }) => (
          <button
            type="button"
            onClick={copy}
            aria-label="Copy snippet"
            className="inline-flex cursor-pointer items-center gap-1 rounded px-1.5 py-0.5 text-micro font-semibold text-terminal-muted outline-none transition-colors duration-150 hover:text-terminal-text focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            {copied ? (
              <Check size={12} strokeWidth={2.5} />
            ) : (
              <Copy size={12} strokeWidth={2.2} />
            )}
            {copied ? "Copied" : "Copy"}
          </button>
        )}
      </CopyButton>
    </div>
    <pre className="overflow-x-auto px-3 py-2.5 font-mono text-xs leading-5 text-terminal-text">
      <code>{props.code}</code>
    </pre>
  </div>
);

export const ValueRow = (props: {
  label: string;
  value?: string;
  placeholder?: string;
  hint?: React.ReactNode;
}) => (
  <div className="flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2 sm:flex-row sm:items-center sm:gap-3">
    <span className="shrink-0 text-xs font-semibold text-slate-600 sm:w-44">
      {props.label}
    </span>
    <span className="flex min-w-0 flex-1 items-center gap-2">
      {props.value ? (
        <code className="min-w-0 flex-1 break-all font-mono text-xs text-slate-800">
          {props.value}
        </code>
      ) : (
        <span className="min-w-0 flex-1 text-xs italic text-slate-400">
          {props.placeholder ?? "Not set yet"}
        </span>
      )}
      {props.value && (
        <CopyButton value={props.value}>
          {({ copied, copy }) => (
            <Tooltip label={copied ? "Copied" : "Copy"} withArrow>
              <button
                type="button"
                onClick={copy}
                aria-label={`Copy ${props.label}`}
                className="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-md text-slate-500 outline-none transition-colors duration-150 hover:bg-slate-100 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                {copied ? (
                  <Check size={13} strokeWidth={2.5} />
                ) : (
                  <Copy size={13} strokeWidth={2.2} />
                )}
              </button>
            </Tooltip>
          )}
        </CopyButton>
      )}
    </span>
    {props.hint && (
      <span className="text-micro font-normal leading-4 text-slate-500 sm:hidden">
        {props.hint}
      </span>
    )}
  </div>
);

export const Table = (props: {
  head: string[];
  rows: React.ReactNode[][];
  className?: string;
}) => (
  <div
    className={twMerge(
      "max-w-3xl overflow-x-auto rounded-lg border border-slate-200",
      props.className,
    )}
  >
    <table className="w-full min-w-[480px] border-collapse text-left text-xs">
      <thead className="bg-slate-50">
        <tr>
          {props.head.map((cell) => (
            <th
              key={cell}
              className="border-b border-slate-200 px-3 py-2 font-semibold text-slate-600"
            >
              {cell}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {props.rows.map((row, index) => (
          <tr key={index} className="border-b border-slate-100 last:border-0">
            {row.map((cell, cellIndex) => (
              <td
                key={cellIndex}
                className="px-3 py-2 align-top leading-5 text-slate-600"
              >
                {cell}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

export type FlowNode = {
  icon: React.ReactNode;
  title: string;
  caption: string;
};

export const FlowDiagram = (props: { nodes: FlowNode[] }) => (
  <div className="relative">
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {props.nodes.map((node, index) => (
        <div key={node.title} className="relative flex items-stretch">
          <motion.div
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: index * 0.08, ease: "easeOut" }}
            className="flex w-full flex-col gap-2 rounded-xl border border-slate-200 bg-white px-3.5 py-3 shadow-card"
          >
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50">
                {node.icon}
              </span>
              <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-400">
                {index + 1}
              </span>
            </div>
            <span className="text-body font-semibold text-slate-800">
              {node.title}
            </span>
            <span className="text-xs font-normal leading-5 text-slate-500">
              {node.caption}
            </span>
          </motion.div>
          {index < props.nodes.length - 1 && (
            <span
              aria-hidden="true"
              className="pointer-events-none absolute -right-3 top-1/2 z-10 hidden h-0.5 w-3 -translate-y-1/2 overflow-hidden bg-slate-200 lg:block"
            >
              <motion.span
                className="absolute inset-y-0 left-0 w-1.5 rounded-full bg-slate-500"
                animate={{ x: [0, 10] }}
                transition={{
                  duration: 1.2,
                  repeat: Infinity,
                  delay: index * 0.4,
                  ease: "easeInOut",
                }}
              />
            </span>
          )}
        </div>
      ))}
    </div>
  </div>
);

export const Figure = (props: {
  caption: string;
  children: React.ReactNode;
  className?: string;
}) => (
  <figure className={twMerge("flex max-w-3xl flex-col gap-2", props.className)}>
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-100/70 p-3 sm:p-4">
      {props.children}
    </div>
    <figcaption className="text-micro font-normal text-slate-500">
      {props.caption}
    </figcaption>
  </figure>
);

export const Aside = (props: {
  icon: LucideIcon;
  title: string;
  children: React.ReactNode;
  tone?: "info" | "warning";
}) => {
  const Icon = props.icon;
  return (
    <div
      className={twMerge(
        "flex max-w-3xl items-start gap-2.5 rounded-lg border px-3 py-2.5",
        props.tone === "warning"
          ? "border-amber-200 bg-amber-50/70"
          : "border-blue-200 bg-blue-50/60",
      )}
    >
      <Icon
        size={14}
        strokeWidth={2.3}
        aria-hidden="true"
        className={twMerge(
          "mt-0.5 shrink-0",
          props.tone === "warning" ? "text-amber-700" : "text-blue-700",
        )}
      />
      <div className="min-w-0 text-xs leading-5 text-slate-700">
        <div
          className={twMerge(
            "font-semibold",
            props.tone === "warning" ? "text-amber-800" : "text-blue-800",
          )}
        >
          {props.title}
        </div>
        <div className="mt-0.5">{props.children}</div>
      </div>
    </div>
  );
};

export const Faq = (props: {
  items: { question: string; answer: React.ReactNode }[];
}) => (
  <div className="flex max-w-3xl flex-col divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white">
    {props.items.map((item) => (
      <details key={item.question} className="group">
        <summary className="flex cursor-pointer list-none items-center gap-2 px-3.5 py-2.5 text-body font-semibold text-slate-700 outline-none transition-colors duration-150 hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400 [&::-webkit-details-marker]:hidden">
          <ChevronRight
            size={13}
            strokeWidth={2.5}
            aria-hidden="true"
            className="shrink-0 text-slate-400 transition-transform duration-150 group-open:rotate-90"
          />
          {item.question}
        </summary>
        <div className="flex flex-col gap-2 px-3.5 pb-3 pl-9 text-body leading-6 text-slate-600">
          {item.answer}
        </div>
      </details>
    ))}
  </div>
);
