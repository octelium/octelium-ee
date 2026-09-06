import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as CoreP from "@/apis/corev1/corev1";
import TimestampPicker from "@/components/TimestampPicker";
import { ActionIcon, Button, SegmentedControl, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { CalendarRange, RotateCcw, SlidersHorizontal, X } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import {
  Drilldown,
  drilldownID,
  drilldownLabel,
  LLMRange,
  RANGE_PRESETS,
} from "./utils";

const STATUS_OPTIONS = [
  { value: "0", label: "All" },
  { value: String(CoreP.AccessLog_Entry_Common_Status.ALLOWED), label: "Allowed" },
  { value: String(CoreP.AccessLog_Entry_Common_Status.DENIED), label: "Denied" },
];

const FilterBar = (props: {
  minutes?: number;
  onMinutesChange: (minutes: number) => void;
  range: LLMRange;
  onRangeChange: (range: LLMRange) => void;
  status: CoreP.AccessLog_Entry_Common_Status;
  onStatusChange: (status: CoreP.AccessLog_Entry_Common_Status) => void;
  drilldowns: Drilldown[];
  onRemoveDrilldown: (drilldown: Drilldown) => void;
  onClearDrilldowns: () => void;
  onRefresh: () => void;
}) => {
  const [custom, setCustom] = React.useState(props.minutes === undefined);

  return (
    <div className="flex w-full flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-[0_1px_4px_rgba(15,23,42,0.04)]">
      <div className="flex flex-wrap items-center gap-2">
        <SegmentedControl
          size="xs"
          value={custom ? "custom" : String(props.minutes ?? 1440)}
          onChange={(value) => {
            if (value === "custom") {
              setCustom(true);
              return;
            }
            setCustom(false);
            props.onMinutesChange(Number(value));
          }}
          data={[
            ...RANGE_PRESETS.map((preset) => ({
              value: String(preset.minutes),
              label: preset.label,
            })),
            {
              value: "custom",
              label: (
                <span className="flex items-center gap-1 px-0.5">
                  <CalendarRange size={12} strokeWidth={2.5} />
                  Custom
                </span>
              ),
            },
          ]}
        />

        <SegmentedControl
          size="xs"
          value={String(props.status)}
          onChange={(value) => props.onStatusChange(Number(value))}
          data={STATUS_OPTIONS}
        />

        <div className="flex-1" />

        <Tooltip label="Refresh" withArrow>
          <ActionIcon variant="default" size="md" onClick={props.onRefresh}>
            <RotateCcw size={14} strokeWidth={2.4} />
          </ActionIcon>
        </Tooltip>
      </div>

      <AnimatePresence initial={false}>
        {custom && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden"
          >
            <div className="grid gap-2 pt-1 sm:grid-cols-2">
              <TimestampPicker
                label="From"
                disableExcludePast
                description="Start of range (inclusive)"
                value={props.range.from}
                onChange={(value: Timestamp | undefined) =>
                  props.onRangeChange({ ...props.range, from: value })
                }
              />
              <TimestampPicker
                label="To"
                disableExcludePast
                description="End of range (exclusive)"
                value={props.range.to}
                onChange={(value: Timestamp | undefined) =>
                  props.onRangeChange({ ...props.range, to: value })
                }
              />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {props.drilldowns.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 border-t border-slate-100 pt-2.5">
          <span className="flex items-center gap-1.5 text-[0.63rem] font-bold uppercase tracking-[0.06em] text-slate-400">
            <SlidersHorizontal size={11} strokeWidth={2.6} />
            Filters
          </span>
          {props.drilldowns.map((drilldown) => (
            <span
              key={drilldownID(drilldown)}
              className={twMerge(
                "inline-flex items-center gap-1 rounded-md border border-slate-200 bg-slate-50 py-0.5 pl-2 pr-0.5",
                "text-[0.66rem] font-bold text-slate-600",
              )}
            >
              {drilldownLabel(drilldown)}
              <button
                type="button"
                aria-label={`Remove ${drilldownLabel(drilldown)}`}
                onClick={() => props.onRemoveDrilldown(drilldown)}
                className="flex h-4 w-4 cursor-pointer items-center justify-center rounded text-slate-400 transition-colors duration-300 hover:bg-slate-200 hover:text-slate-700"
              >
                <X size={10} strokeWidth={3} />
              </button>
            </span>
          ))}
          <Button
            size="compact-xs"
            variant="subtle"
            color="gray"
            onClick={props.onClearDrilldowns}
          >
            Clear all
          </Button>
        </div>
      )}
    </div>
  );
};

export default FilterBar;
