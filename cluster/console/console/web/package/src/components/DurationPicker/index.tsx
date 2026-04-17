import { Duration } from "@/apis/metav1/metav1";
import { toNumOrZero } from "@/utils";
import {
  Button,
  Group,
  Input,
  InputBase,
  NumberInput,
  Popover,
  Select,
} from "@mantine/core";
import { useState } from "react";
import { LuTimer, LuX } from "react-icons/lu";
import { match } from "ts-pattern";
import { timeRangePickList } from "../AccessLogViewer/utils";

const parseDuration = (
  duration?: Duration,
): { val: number; unit: string } | null => {
  if (!duration?.type) return null;
  return match(duration.type.oneofKind)
    .with("milliseconds", () => ({
      val: (duration.type as { milliseconds: number }).milliseconds,
      unit: "millisecond",
    }))
    .with("seconds", () => ({
      val: (duration.type as { seconds: number }).seconds,
      unit: "second",
    }))
    .with("minutes", () => ({
      val: (duration.type as { minutes: number }).minutes,
      unit: "minute",
    }))
    .with("hours", () => ({
      val: (duration.type as { hours: number }).hours,
      unit: "hour",
    }))
    .with("days", () => ({
      val: (duration.type as { days: number }).days,
      unit: "day",
    }))
    .with("weeks", () => ({
      val: (duration.type as { weeks: number }).weeks,
      unit: "week",
    }))
    .with("months", () => ({
      val: (duration.type as { months: number }).months,
      unit: "month",
    }))
    .otherwise(() => null);
};

const createDuration = (val: number, unit: string): Duration => {
  const typePayload = match(unit)
    .with("millisecond", () => ({
      oneofKind: "milliseconds" as const,
      milliseconds: val,
    }))
    .with("second", () => ({ oneofKind: "seconds" as const, seconds: val }))
    .with("minute", () => ({ oneofKind: "minutes" as const, minutes: val }))
    .with("hour", () => ({ oneofKind: "hours" as const, hours: val }))
    .with("day", () => ({ oneofKind: "days" as const, days: val }))
    .with("week", () => ({ oneofKind: "weeks" as const, weeks: val }))
    .with("month", () => ({ oneofKind: "months" as const, months: val }))
    .otherwise(() => ({ oneofKind: "seconds" as const, seconds: val }));
  return Duration.create({ type: typePayload as any });
};

const sharedInputStyles = {
  input: {
    fontSize: "0.78rem",
    fontWeight: 600,
    fontFamily: "Ubuntu, sans-serif",
    border: "1px solid #e2e8f0",
    borderRadius: "6px",
    color: "#1e293b",
    height: "32px",
    minHeight: "32px",
    boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
  },
  label: {
    fontSize: "0.68rem",
    fontWeight: 700,
    fontFamily: "Ubuntu, sans-serif",
    textTransform: "uppercase" as const,
    letterSpacing: "0.05em",
    color: "#64748b",
    marginBottom: "4px",
  },
  option: {
    fontSize: "0.78rem",
    fontWeight: 600,
    fontFamily: "Ubuntu, sans-serif",
  },
};

interface DurationPickerProps {
  value?: Duration;
  title?: string;
  description?: string;
  placeholder?: string;
  onChange: (arg?: Duration) => void;
}

const DurationPicker = ({
  value,
  title,
  description,
  placeholder = "Select duration",
  onChange,
}: DurationPickerProps) => {
  const [opened, setOpened] = useState(false);
  const [activeTab, setActiveTab] = useState<"presets" | "custom">("presets");

  const current = parseDuration(value);
  const [customVal, setCustomVal] = useState<number | "">(current?.val ?? 1);
  const [customUnit, setCustomUnit] = useState<string>(
    current?.unit ?? "minute",
  );

  const displayValue = current
    ? `${current.val} ${current.unit.charAt(0).toUpperCase() + current.unit.slice(1)}${current.val === 1 ? "" : "s"}`
    : "";

  const handlePresetChange = (v: string | null) => {
    if (!v) {
      onChange(undefined);
      return;
    }
    const [numStr, unit] = v.split(" ");
    onChange(createDuration(parseInt(numStr, 10), unit));
    setOpened(false);
  };

  const applyCustomChange = () => {
    if (customVal === "" || customVal <= 0) return;
    onChange(createDuration(customVal, customUnit));
    setOpened(false);
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(undefined);
  };

  const tabBtnStyles = (active: boolean, position: "left" | "right") => ({
    root: {
      height: "30px",
      fontSize: "0.72rem",
      fontWeight: 700,
      fontFamily: "Ubuntu, sans-serif",
      padding: "0 12px",
      flex: 1,
      backgroundColor: active ? "#0f172a" : "#f8fafc",
      color: active ? "#ffffff" : "#64748b",
      border: "none",
      borderRadius: 0,
      borderTopLeftRadius: position === "left" ? "8px" : 0,
      borderTopRightRadius: position === "right" ? "8px" : 0,
      transition: "background-color 150ms, color 150ms",
      "&:hover": {
        backgroundColor: active ? "#1e293b" : "#f1f5f9",
        color: active ? "#ffffff" : "#0f172a",
      },
    },
  });

  return (
    <Input.Wrapper
      label={
        title ? (
          <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-600 mb-1 block">
            {title}
          </span>
        ) : undefined
      }
      description={
        description ? (
          <span className="text-[0.7rem] font-semibold text-slate-400">
            {description}
          </span>
        ) : undefined
      }
      className="w-full"
    >
      <Popover
        opened={opened}
        onChange={setOpened}
        width="auto"
        position="bottom-start"
        withArrow
        shadow="xl"
        transitionProps={{ transition: "pop", duration: 180 }}
      >
        <Popover.Target>
          <InputBase
            component="button"
            type="button"
            pointer
            leftSection={
              <LuTimer
                size={13}
                className={displayValue ? "text-slate-500" : "text-slate-400"}
              />
            }
            rightSection={
              displayValue ? (
                <button
                  onClick={handleClear}
                  className="flex items-center justify-center text-slate-400 hover:text-slate-800 cursor-pointer transition-colors duration-150"
                >
                  <LuX size={12} />
                </button>
              ) : null
            }
            onClick={() => setOpened((o) => !o)}
            className="w-full"
            styles={{
              input: {
                height: "32px",
                fontSize: "0.78rem",
                fontWeight: 600,
                fontFamily: "Ubuntu, sans-serif",
                backgroundColor: "#ffffff",
                border: "1px solid #e2e8f0",
                borderRadius: "6px",
                color: "#1e293b",
                boxShadow: "0 1px 3px rgba(15,23,42,0.06)",
                cursor: "pointer",
                "&:hover": { borderColor: "#cbd5e1" },
                "&:focus": {
                  borderColor: "#94a3b8",
                  boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
                },
              },
            }}
          >
            {displayValue || (
              <span className="text-[0.78rem] font-semibold text-slate-400">
                {placeholder}
              </span>
            )}
          </InputBase>
        </Popover.Target>

        <Popover.Dropdown
          p={0}
          style={{
            border: "1px solid #e2e8f0",
            borderRadius: "10px",
            boxShadow: "0 8px 32px rgba(15,23,42,0.12)",
            overflow: "visible",
            minWidth: "300px",
          }}
        >
          <div className="border-b border-slate-100 bg-slate-50 rounded-t-[10px] overflow-hidden">
            <Button.Group style={{ display: "flex" }}>
              {(
                [
                  { value: "presets" as const, label: "Presets" },
                  { value: "custom" as const, label: "Custom" },
                ] as const
              ).map(({ value, label }, i) => (
                <Button
                  key={value}
                  onClick={() => setActiveTab(value)}
                  styles={tabBtnStyles(
                    activeTab === value,
                    i === 0 ? "left" : "right",
                  )}
                  leftSection={<LuTimer size={12} />}
                  style={{ flex: 1 }}
                >
                  {label}
                </Button>
              ))}
            </Button.Group>
          </div>

          <div className="p-3 bg-white rounded-b-[10px]">
            {activeTab === "presets" && (
              <Select
                data={timeRangePickList}
                placeholder="Quick select duration"
                searchable
                value={null}
                onChange={handlePresetChange}
                comboboxProps={{ withinPortal: true }}
                styles={sharedInputStyles}
              />
            )}

            {activeTab === "custom" && (
              <div className="flex flex-col gap-3">
                <Group align="flex-end" grow gap="sm">
                  <NumberInput
                    label="Value"
                    value={customVal}
                    min={1}
                    max={100000}
                    allowNegative={false}
                    allowDecimal={false}
                    onChange={(v) => setCustomVal(toNumOrZero(`${v}`))}
                    styles={sharedInputStyles}
                  />
                  <Select
                    label="Unit"
                    data={[
                      { value: "millisecond", label: "Milliseconds" },
                      { value: "second", label: "Seconds" },
                      { value: "minute", label: "Minutes" },
                      { value: "hour", label: "Hours" },
                      { value: "day", label: "Days" },
                      { value: "week", label: "Weeks" },
                      { value: "month", label: "Months" },
                    ]}
                    value={customUnit}
                    onChange={(v) => setCustomUnit(v || "minute")}
                    comboboxProps={{ withinPortal: true }}
                    styles={sharedInputStyles}
                  />
                </Group>

                <Button
                  fullWidth
                  onClick={applyCustomChange}
                  disabled={customVal === "" || customVal <= 0}
                  styles={{
                    root: {
                      height: "32px",
                      fontSize: "0.75rem",
                      fontWeight: 700,
                      fontFamily: "Ubuntu, sans-serif",
                      backgroundColor: "#0f172a",
                      borderRadius: "6px",
                      "&:hover": { backgroundColor: "#1e293b" },
                      "&:disabled": {
                        backgroundColor: "#f1f5f9",
                        color: "#94a3b8",
                      },
                    },
                  }}
                >
                  Apply
                </Button>
              </div>
            )}
          </div>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default DurationPicker;
