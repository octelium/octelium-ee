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
        "w-full bg-transparent border border-slate-200 rounded-xl overflow-hidden shadow-card",
        props.className,
      )}
    >
      {props.title && (
        <div className="flex flex-col gap-0.5 border-b border-slate-100 bg-slate-50/60 px-5 py-3.5">
          <span className="text-sm font-semibold text-slate-900">
            {props.title}
          </span>
          {props.description && (
            <span className="text-xs font-normal text-slate-500">
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
