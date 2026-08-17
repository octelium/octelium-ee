import { Slider } from "@mantine/core";
import { twMerge } from "tailwind-merge";

const PRIORITY_STEPS = [-4, -3, -2, -1, 0, 1, 2, 3, 4] as const;
type Priority = (typeof PRIORITY_STEPS)[number];

const priorityMeta = (priority: Priority) => {
  if (priority <= -3) {
    return {
      hint: "Highest",
      color: "border-slate-700 bg-slate-800 text-white",
    };
  }
  if (priority <= -1) {
    return {
      hint: "High",
      color: "border-slate-300 bg-slate-100 text-slate-700",
    };
  }
  if (priority === 0) {
    return {
      hint: "Default",
      color: "border-blue-200 bg-blue-50 text-blue-700",
    };
  }
  if (priority <= 2) {
    return {
      hint: "Low",
      color: "border-amber-200 bg-amber-50 text-amber-700",
    };
  }
  return {
    hint: "Lowest",
    color: "border-orange-200 bg-orange-50 text-orange-700",
  };
};

const PriorityPicker = (props: {
  value: number;
  onChange: (value: number) => void;
  label?: string;
  description?: string;
}) => {
  const value = Math.min(4, Math.max(-4, props.value)) as Priority;
  const valueLabel = value > 0 ? `+${value}` : `${value}`;
  const meta = priorityMeta(value);

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 shadow-[0_1px_3px_rgba(15,23,42,0.035)]">
      <div className="flex items-start justify-between gap-3 border-b border-slate-100 bg-white px-3 py-2.5">
        <div className="min-w-0">
          {props.label && (
            <div className="text-[0.72rem] font-bold text-slate-700">
              {props.label}
            </div>
          )}
          {props.description && (
            <div className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
              {props.description}
            </div>
          )}
        </div>
        <span
          className={twMerge(
            "inline-flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-[0.65rem] font-bold",
            meta.color,
          )}
        >
          {meta.hint}
          <span className="opacity-70">{valueLabel}</span>
        </span>
      </div>

      <div className="px-4 pb-5 pt-4">
        <Slider
          aria-label={props.label ?? "Rule priority"}
          min={-4}
          max={4}
          step={1}
          value={value}
          onChange={props.onChange}
          color="dark"
          size="sm"
          label={(next) => {
            const priority = next as Priority;
            const formatted = priority > 0 ? `+${priority}` : `${priority}`;
            return `${priorityMeta(priority).hint} · ${formatted}`;
          }}
          marks={[
            { value: -4, label: "−4 · Earlier" },
            { value: 0, label: "0 · Default" },
            { value: 4, label: "+4 · Later" },
          ]}
          styles={{
            markLabel: {
              fontSize: "0.61rem",
              fontWeight: 700,
              color: "#94a3b8",
              whiteSpace: "nowrap",
            },
          }}
        />
      </div>
    </div>
  );
};

export default PriorityPicker;
