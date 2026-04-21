import { twMerge } from "tailwind-merge";

const Label = (props: {
  children?: React.ReactNode;
  outlined?: boolean;
  isLink?: boolean;
}) => {
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-1",
        "px-2 h-[22px]",
        "rounded-full text-[0.7rem] font-bold leading-none whitespace-nowrap",
        "transition-colors duration-500 cursor-default",
        props.outlined
          ? "text-slate-700 border border-slate-300 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.06)]"
          : "bg-slate-800 text-slate-100 border border-slate-700",
        props.isLink
          ? "shadow-[0_1px_4px_rgba(15,23,42,0.15)] hover:bg-slate-700 hover:border-slate-600 cursor-pointer"
          : "shadow-[0_1px_2px_rgba(15,23,42,0.08)]",
      )}
    >
      {props.children}
    </span>
  );
};

export default Label;
