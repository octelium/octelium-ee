import * as React from "react";
import { twMerge } from "tailwind-merge";

const ContainerGen = (props: {
  children?: React.ReactNode;
  title?: React.ReactNode;
  description?: string;
  className?: string;
}) => {
  return (
    <div
      className={twMerge(
        "w-full bg-transparent border border-slate-200 rounded-xl overflow-hidden shadow-[0_1px_4px_rgba(15,23,42,0.06)]",
        props.className,
      )}
    >
      {props.title && (
        <div className="flex flex-col gap-0.5 px-5 py-3.5 border-b border-slate-100 bg-slate-50/60">
          <span className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-700">
            {props.title}
          </span>
          {props.description && (
            <span className="text-[0.7rem] font-semibold text-slate-400">
              {props.description}
            </span>
          )}
        </div>
      )}
      <div className="w-full p-5">{props.children}</div>
    </div>
  );
};

export default ContainerGen;
