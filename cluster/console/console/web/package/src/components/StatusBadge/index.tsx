import type { ResourceStatusTone } from "@/pages/utils/types";
import { twMerge } from "@/lib/utils";
import type { ElementType } from "react";

const TONE_CLASSES: Record<ResourceStatusTone, string> = {
  neutral: "border-slate-200 bg-slate-50 text-slate-600",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
  info: "border-blue-200 bg-blue-50 text-blue-700",
};

const DOT_CLASSES: Record<ResourceStatusTone, string> = {
  neutral: "bg-slate-400",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  info: "bg-blue-500",
};

const StatusBadge = (props: {
  label: string;
  tone?: ResourceStatusTone;
  compact?: boolean;
  className?: string;
  icon?: ElementType<{
    size?: number;
    strokeWidth?: number;
    "aria-hidden"?: boolean;
  }>;
}) => {
  const tone = props.tone ?? "neutral";
  const Icon = props.icon;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-1.5 rounded-full border font-semibold leading-none",
        props.compact ? "px-1.5 py-1 text-micro" : "px-2.5 py-1 text-xs",
        TONE_CLASSES[tone],
        props.className,
      )}
    >
      {Icon ? (
        <Icon size={9} strokeWidth={2.5} aria-hidden={true} />
      ) : (
        <span
          aria-hidden="true"
          className={twMerge("h-1.5 w-1.5 rounded-full", DOT_CLASSES[tone])}
        />
      )}
      {props.label}
    </span>
  );
};

export default StatusBadge;
