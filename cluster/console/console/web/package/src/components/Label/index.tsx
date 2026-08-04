import { twMerge } from "tailwind-merge";

type LabelTone = "neutral" | "info" | "success" | "warning" | "danger";
type LabelSize = "sm" | "md";

const tones: Record<LabelTone, string> = {
  neutral: "border-slate-200 bg-slate-100/80 text-slate-700",
  info: "border-blue-200 bg-blue-50 text-blue-700",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
};

const sizes: Record<LabelSize, string> = {
  sm: "min-h-5 px-1.5 py-0.5 text-[0.65rem]",
  md: "min-h-[22px] px-2 py-1 text-[0.7rem]",
};

const Label = (props: {
  children?: React.ReactNode;
  outlined?: boolean;
  isLink?: boolean;
  tone?: LabelTone;
  size?: LabelSize;
  className?: string;
}) => (
  <span
    className={twMerge(
      "inline-flex max-w-full items-center gap-1 rounded-md border font-bold leading-none",
      "whitespace-nowrap shadow-[0_1px_2px_rgba(15,23,42,0.035)]",
      "cursor-default transition-[color,background-color,border-color,box-shadow,filter] duration-500",
      sizes[props.size ?? "md"],
      props.outlined
        ? "border-slate-200 bg-white text-slate-700"
        : tones[props.tone ?? "neutral"],
      props.isLink &&
        "cursor-pointer hover:border-slate-300 hover:brightness-[0.98] hover:shadow-[0_2px_5px_rgba(15,23,42,0.07)] focus-within:ring-2 focus-within:ring-blue-500/20",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export default Label;
