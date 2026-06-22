import { Loader2 } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { Tone, toneClasses } from "../utils";

export const Eyebrow = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "text-[0.65rem] font-bold uppercase tracking-[0.08em] text-slate-400",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const PageHeader = (props: {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: React.ReactNode;
}) => (
  <div className="w-full flex items-start justify-between gap-4 mb-6">
    <div className="flex flex-col gap-1">
      {props.eyebrow && <Eyebrow>{props.eyebrow}</Eyebrow>}
      <h1 className="text-[1.35rem] font-bold text-slate-900 leading-tight">
        {props.title}
      </h1>
      {props.description && (
        <p className="text-[0.82rem] font-medium text-slate-500 max-w-xl">
          {props.description}
        </p>
      )}
    </div>
    {props.actions && (
      <div className="flex items-center gap-2 shrink-0">{props.actions}</div>
    )}
  </div>
);

export const Card = (props: {
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
  interactive?: boolean;
}) => (
  <div
    onClick={props.onClick}
    className={twMerge(
      "rounded-xl border border-slate-200 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.04)]",
      props.interactive &&
        "cursor-pointer transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-[0_2px_10px_rgba(15,23,42,0.07)]",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const Badge = (props: {
  tone?: Tone;
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "inline-flex items-center px-2 py-0.5 rounded-md text-[0.68rem] font-bold uppercase tracking-[0.03em] border",
      toneClasses[props.tone ?? "slate"].badge,
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const SectionTitle = (props: { children: React.ReactNode }) => (
  <h2 className="text-[0.8rem] font-bold text-slate-700 mb-3">
    {props.children}
  </h2>
);

export const Field = (props: {
  label: string;
  description?: string;
  children: React.ReactNode;
}) => (
  <div className="flex flex-col gap-1.5">
    <div className="flex flex-col gap-0.5">
      <span className="text-[0.72rem] font-bold text-slate-700">
        {props.label}
      </span>
      {props.description && (
        <span className="text-[0.68rem] font-medium text-slate-400">
          {props.description}
        </span>
      )}
    </div>
    {props.children}
  </div>
);

export const KeyValue = (props: {
  label: string;
  children: React.ReactNode;
  full?: boolean;
}) => (
  <div className={twMerge("flex flex-col gap-1", props.full && "col-span-2")}>
    <Eyebrow>{props.label}</Eyebrow>
    <div className="text-[0.82rem] font-semibold text-slate-700">
      {props.children}
    </div>
  </div>
);

export const Loading = (props: { label?: string }) => (
  <div className="w-full flex items-center justify-center py-16">
    <div className="flex items-center gap-2 text-slate-400">
      <Loader2 size={16} strokeWidth={2.5} className="animate-spin" />
      <span className="text-[0.78rem] font-semibold">
        {props.label ?? "Loading..."}
      </span>
    </div>
  </div>
);

export const EmptyState = (props: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) => (
  <div className="w-full flex flex-col items-center justify-center text-center py-16 px-6">
    {props.icon && (
      <div className="flex items-center justify-center w-11 h-11 rounded-xl bg-slate-100 text-slate-400 mb-3">
        {props.icon}
      </div>
    )}
    <p className="text-[0.9rem] font-bold text-slate-700">{props.title}</p>
    {props.description && (
      <p className="text-[0.78rem] font-medium text-slate-400 mt-1 max-w-sm">
        {props.description}
      </p>
    )}
    {props.action && <div className="mt-4">{props.action}</div>}
  </div>
);
