import { EXTENDED_PERIODS, PRIMARY_PERIODS } from "@/utils/visibility";
import { Menu, SegmentedControl } from "@mantine/core";
import { ChevronDown } from "lucide-react";
import { twMerge } from "tailwind-merge";

const PeriodSelector = (props: {
  value: number;
  onChange: (value: number) => void;
}) => {
  const extended = EXTENDED_PERIODS.find(
    (option) => option.minutes === props.value,
  );

  const data = [
    ...PRIMARY_PERIODS.map((option) => ({
      value: String(option.minutes),
      label: option.label,
    })),
    ...(extended
      ? [{ value: String(extended.minutes), label: extended.label }]
      : []),
  ];

  return (
    <div className="flex items-stretch gap-1.5">
      <SegmentedControl
        size="xs"
        aria-label="Time range"
        value={String(props.value)}
        onChange={(value) => props.onChange(Number(value))}
        data={data}
      />

      <Menu position="bottom-end" offset={4} withArrow={false}>
        <Menu.Target>
          <button
            type="button"
            aria-label="More time ranges"
            className="flex shrink-0 cursor-pointer items-center gap-1 rounded-[10px] border border-slate-200 bg-slate-100 px-2 text-body font-semibold text-slate-500 outline-none transition-colors duration-150 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            <ChevronDown size={12} strokeWidth={2.5} />
          </button>
        </Menu.Target>

        <Menu.Dropdown>
          <div className="flex min-w-[104px] flex-col py-1">
            {EXTENDED_PERIODS.map((option) => (
              <button
                type="button"
                key={option.minutes}
                onClick={() => props.onChange(option.minutes)}
                className={twMerge(
                  "flex h-8 cursor-pointer items-center rounded-md px-3 text-left text-body font-semibold transition-colors duration-150",
                  option.minutes === props.value
                    ? "bg-slate-900 text-white"
                    : "text-slate-600 hover:bg-slate-100 hover:text-slate-900",
                )}
              >
                Last {option.label}
              </button>
            ))}
          </div>
        </Menu.Dropdown>
      </Menu>
    </div>
  );
};

export default PeriodSelector;
