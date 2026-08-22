import { Duration } from "@/apis/metav1/metav1";
import {
  Button,
  Input,
  InputBase,
  NumberInput,
  Popover,
  SegmentedControl,
  Select,
} from "@mantine/core";
import { Check, ChevronDown, Clock3, RotateCcw } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { match } from "ts-pattern";

type DurationUnit =
  | "millisecond"
  | "second"
  | "minute"
  | "hour"
  | "day"
  | "week"
  | "month";

type ParsedDuration = {
  val: number;
  unit: DurationUnit;
};

const UNITS = [
  { value: "millisecond", label: "Milliseconds" },
  { value: "second", label: "Seconds" },
  { value: "minute", label: "Minutes" },
  { value: "hour", label: "Hours" },
  { value: "day", label: "Days" },
  { value: "week", label: "Weeks" },
  { value: "month", label: "Months" },
];

const PRESETS: Array<ParsedDuration & { label: string; shortLabel: string }> = [
  { val: 1, unit: "minute", label: "1 Minute", shortLabel: "1m" },
  { val: 5, unit: "minute", label: "5 Minutes", shortLabel: "5m" },
  { val: 15, unit: "minute", label: "15 Minutes", shortLabel: "15m" },
  { val: 30, unit: "minute", label: "30 Minutes", shortLabel: "30m" },
  { val: 1, unit: "hour", label: "1 Hour", shortLabel: "1h" },
  { val: 2, unit: "hour", label: "2 Hours", shortLabel: "2h" },
  { val: 6, unit: "hour", label: "6 Hours", shortLabel: "6h" },
  { val: 12, unit: "hour", label: "12 Hours", shortLabel: "12h" },
  { val: 1, unit: "day", label: "1 Day", shortLabel: "1d" },
  { val: 3, unit: "day", label: "3 Days", shortLabel: "3d" },
  { val: 1, unit: "week", label: "1 Week", shortLabel: "1w" },
  { val: 1, unit: "month", label: "1 Month", shortLabel: "1mo" },
];

const parseDuration = (duration?: Duration): ParsedDuration | null => {
  if (!duration?.type) return null;
  const type = duration.type;

  switch (type.oneofKind) {
    case "milliseconds":
      return { val: type.milliseconds, unit: "millisecond" };
    case "seconds":
      return { val: type.seconds, unit: "second" };
    case "minutes":
      return { val: type.minutes, unit: "minute" };
    case "hours":
      return { val: type.hours, unit: "hour" };
    case "days":
      return { val: type.days, unit: "day" };
    case "weeks":
      return { val: type.weeks, unit: "week" };
    case "months":
      return { val: type.months, unit: "month" };
    default:
      return null;
  }
};

const createDuration = (val: number, unit: DurationUnit): Duration => {
  const type = match(unit)
    .with("millisecond", () => ({
      oneofKind: "milliseconds" as const,
      milliseconds: val,
    }))
    .with("second", () => ({
      oneofKind: "seconds" as const,
      seconds: val,
    }))
    .with("minute", () => ({
      oneofKind: "minutes" as const,
      minutes: val,
    }))
    .with("hour", () => ({ oneofKind: "hours" as const, hours: val }))
    .with("day", () => ({ oneofKind: "days" as const, days: val }))
    .with("week", () => ({ oneofKind: "weeks" as const, weeks: val }))
    .with("month", () => ({ oneofKind: "months" as const, months: val }))
    .exhaustive();

  return Duration.create({ type });
};

const formatDuration = (duration: ParsedDuration) => {
  const unit = `${duration.unit.charAt(0).toUpperCase()}${duration.unit.slice(1)}`;
  return `${duration.val.toLocaleString()} ${unit}${duration.val === 1 ? "" : "s"}`;
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
  const [activeView, setActiveView] = useState<"presets" | "custom">(
    "presets",
  );
  const current = useMemo(() => parseDuration(value), [value]);
  const [customValue, setCustomValue] = useState<number | "">(
    current?.val ?? 1,
  );
  const [customUnit, setCustomUnit] = useState<DurationUnit>(
    current?.unit ?? "minute",
  );

  useEffect(() => {
    if (!current) return;
    setCustomValue(current.val);
    setCustomUnit(current.unit);
  }, [current?.unit, current?.val]);

  const activePreset = current
    ? PRESETS.find(
        (preset) =>
          preset.val === current.val && preset.unit === current.unit,
      )
    : undefined;
  const customIsValid =
    customValue !== "" &&
    Number.isFinite(customValue) &&
    customValue > 0 &&
    customValue <= 100_000;

  const selectDuration = (duration: ParsedDuration) => {
    onChange(createDuration(duration.val, duration.unit));
    setOpened(false);
  };

  const applyCustom = () => {
    if (!customIsValid) return;
    selectDuration({ val: customValue, unit: customUnit });
  };

  return (
    <Input.Wrapper
      label={title}
      className="w-full"
    >
      {description && (
        <p className="mb-1 text-[0.7rem] font-semibold leading-5 text-slate-400">
          {description}
        </p>
      )}
      <Popover
        opened={opened}
        onChange={setOpened}
        width={360}
        position="bottom-start"
        withArrow
        shadow="xl"
        trapFocus
        returnFocus
        transitionProps={{ transition: "pop", duration: 200 }}
      >
        <Popover.Target>
          <InputBase
            component="button"
            type="button"
            pointer
            aria-haspopup="dialog"
            aria-expanded={opened}
            leftSection={
              <Clock3
                size={14}
                className={current ? "text-slate-600" : "text-slate-400"}
                strokeWidth={2.25}
              />
            }
            rightSection={
              <ChevronDown
                size={13}
                strokeWidth={2.5}
                className={`text-slate-400 transition-transform duration-300 ${
                  opened ? "rotate-180" : ""
                }`}
              />
            }
            onClick={() => setOpened((currentOpened) => !currentOpened)}
            className="w-full text-left"
            styles={{
              input: {
                cursor: "pointer",
                textAlign: "left",
                "&:hover": { borderColor: "#cbd5e1" },
              },
            }}
          >
            {current ? (
              <span className="text-[0.78rem] font-bold text-slate-700">
                {formatDuration(current)}
              </span>
            ) : (
              <span className="text-[0.78rem] font-semibold text-slate-400">
                {placeholder}
              </span>
            )}
          </InputBase>
        </Popover.Target>

        <Popover.Dropdown
          p={0}
          style={{
            maxWidth: "calc(100vw - 24px)",
            overflow: "hidden",
          }}
        >
          <div className="flex items-center justify-between gap-3 border-b border-slate-100 bg-slate-50/70 px-4 py-3">
            <div className="flex min-w-0 items-center gap-2.5">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white shadow-sm">
                <Clock3 size={14} strokeWidth={2.25} />
              </span>
              <div className="min-w-0">
                <p className="text-[0.72rem] font-bold text-slate-700">
                  Choose duration
                </p>
                <p className="mt-0.5 truncate text-[0.62rem] font-semibold text-slate-400">
                  {current ? formatDuration(current) : "No duration selected"}
                </p>
              </div>
            </div>
            {current && (
              <Button
                type="button"
                size="compact-xs"
                variant="subtle"
                color="gray"
                leftSection={<RotateCcw size={11} strokeWidth={2.25} />}
                onClick={() => {
                  onChange(undefined);
                  setOpened(false);
                }}
              >
                Clear
              </Button>
            )}
          </div>

          <div className="bg-white p-3.5">
            <SegmentedControl
              fullWidth
              value={activeView}
              data={[
                { label: "Common presets", value: "presets" },
                { label: "Custom", value: "custom" },
              ]}
              onChange={(nextView) =>
                setActiveView(nextView as "presets" | "custom")
              }
            />

            <div className="mt-3">
              {activeView === "presets" ? (
                <div>
                  <div className="grid grid-cols-3 gap-2">
                    {PRESETS.map((preset) => {
                      const selected = activePreset === preset;
                      return (
                        <Button
                          key={`${preset.val}-${preset.unit}`}
                          type="button"
                          variant={selected ? "filled" : "default"}
                          color={selected ? "dark" : undefined}
                          leftSection={
                            selected ? (
                              <Check size={11} strokeWidth={2.75} />
                            ) : undefined
                          }
                          aria-label={preset.label}
                          aria-pressed={selected}
                          onClick={() => selectDuration(preset)}
                          styles={{
                            root: {
                              height: "36px",
                              paddingInline: "9px",
                              fontSize: "0.72rem",
                              fontWeight: 700,
                            },
                          }}
                        >
                          {preset.shortLabel}
                        </Button>
                      );
                    })}
                  </div>
                  <p className="mt-3 text-center text-[0.62rem] font-semibold text-slate-400">
                    Use Custom for seconds, milliseconds, or longer values.
                  </p>
                </div>
              ) : (
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    applyCustom();
                  }}
                >
                  <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-3">
                    <NumberInput
                      label="Value"
                      value={customValue}
                      min={1}
                      max={100_000}
                      allowNegative={false}
                      allowDecimal={false}
                      clampBehavior="strict"
                      onChange={(nextValue) =>
                        setCustomValue(
                          nextValue === "" ? "" : Number(nextValue),
                        )
                      }
                    />
                    <Select
                      label="Unit"
                      data={UNITS}
                      value={customUnit}
                      allowDeselect={false}
                      onChange={(nextUnit) =>
                        nextUnit && setCustomUnit(nextUnit as DurationUnit)
                      }
                    />
                  </div>

                  <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2.5">
                    <p className="text-[0.62rem] font-bold uppercase tracking-[0.06em] text-slate-400">
                      Result
                    </p>
                    <p className="mt-0.5 text-[0.75rem] font-bold text-slate-700">
                      {customIsValid
                        ? formatDuration({
                            val: customValue,
                            unit: customUnit,
                          })
                        : "Enter a value from 1 to 100,000"}
                    </p>
                  </div>

                  <Button
                    type="submit"
                    fullWidth
                    color="dark"
                    disabled={!customIsValid}
                    className="mt-3"
                  >
                    Apply duration
                  </Button>
                </form>
              )}
            </div>
          </div>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default DurationPicker;
