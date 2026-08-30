import * as AccessP from "@/apis/accessv1/accessv1";
import { twMerge } from "tailwind-merge";

import { URGENCY_OPTIONS, toneClasses, urgencyMeta } from "@/utils";

const UrgencyPicker = (props: {
  value: AccessP.Request_Spec_Urgency;
  onChange: (value: AccessP.Request_Spec_Urgency) => void;
}) => (
  <div className="grid grid-cols-3 gap-1.5">
    {URGENCY_OPTIONS.map((option) => {
      const meta = urgencyMeta(option.value);
      const selected = props.value === option.value;
      return (
        <button
          key={option.value}
          type="button"
          aria-pressed={selected}
          onClick={() => props.onChange(option.value)}
          className={twMerge(
            "flex h-8 items-center justify-center gap-1.5 whitespace-nowrap rounded-md border px-1.5 text-[0.68rem] font-bold transition-[background-color,border-color,color] duration-150",
            selected
              ? "border-slate-900 bg-slate-900 text-white shadow-[0_1px_3px_rgba(15,23,42,0.2)]"
              : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:bg-slate-50",
          )}
        >
          <span
            className={twMerge(
              "h-1.5 w-1.5 rounded-full",
              selected ? "bg-white" : toneClasses[meta.tone].dot,
            )}
          />
          {option.label}
        </button>
      );
    })}
  </div>
);

export default UrgencyPicker;
