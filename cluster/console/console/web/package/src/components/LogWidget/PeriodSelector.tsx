import {
  ALL_PERIODS,
  EXTENDED_PERIODS,
  PRIMARY_PERIODS,
} from "@/utils/visibility";
import { Menu } from "@mantine/core";
import { ChevronDown } from "lucide-react";
import { twMerge } from "tailwind-merge";

const CELL =
  "h-[26px] cursor-pointer px-2.5 text-micro font-semibold transition-colors duration-150";

const PeriodSelector = (props: {
  value: number;
  onChange: (value: number) => void;
}) => {
  const isExtended = EXTENDED_PERIODS.some((p) => p.minutes === props.value);
  const extendedLabel = isExtended
    ? ALL_PERIODS.find((p) => p.minutes === props.value)?.label
    : undefined;

  return (
    <div
      role="group"
      aria-label="Time range"
      className="flex overflow-hidden rounded-md border border-slate-200 shadow-card"
    >
      {PRIMARY_PERIODS.map((option) => {
        const active = option.minutes === props.value;
        return (
          <button
            type="button"
            key={option.minutes}
            aria-pressed={active}
            onClick={() => props.onChange(option.minutes)}
            className={twMerge(
              CELL,
              active
                ? "bg-slate-900 text-white hover:bg-slate-800"
                : "bg-white text-slate-600 hover:bg-slate-50 hover:text-slate-900",
            )}
          >
            {option.label}
          </button>
        );
      })}

      <Menu position="bottom-end" offset={4} withArrow={false}>
        <Menu.Target>
          <button
            type="button"
            aria-label="More time ranges"
            className={twMerge(
              CELL,
              "flex items-center gap-1 border-l border-slate-200 px-2",
              isExtended
                ? "bg-slate-900 text-white hover:bg-slate-800"
                : "bg-white text-slate-600 hover:bg-slate-50 hover:text-slate-900",
            )}
          >
            {extendedLabel ?? "More"}
            <ChevronDown size={10} strokeWidth={2.5} />
          </button>
        </Menu.Target>
        <Menu.Dropdown>
          <div className="flex min-w-[100px] flex-col py-1">
            {EXTENDED_PERIODS.map((option) => (
              <button
                type="button"
                key={option.minutes}
                onClick={() => props.onChange(option.minutes)}
                className={twMerge(
                  "flex h-8 cursor-pointer items-center px-3 text-left text-body font-normal transition-colors duration-150",
                  option.minutes === props.value
                    ? "bg-slate-900 text-white"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900",
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
        </Menu.Dropdown>
      </Menu>
    </div>
  );
};

export default PeriodSelector;
