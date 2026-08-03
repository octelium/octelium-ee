import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  Button,
  Input,
  InputBase,
  Popover,
  SegmentedControl,
} from "@mantine/core";
import { DatePicker, TimeInput } from "@mantine/dates";
import dayjs from "dayjs";
import {
  CalendarDays,
  ChevronDown,
  Clock3,
  RotateCcw,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

type RelativeUnit = "minute" | "hour" | "day" | "week" | "month";

const PRESETS: Array<{
  value: number;
  unit: RelativeUnit;
  shortLabel: string;
}> = [
  { value: 1, unit: "minute", shortLabel: "1m" },
  { value: 5, unit: "minute", shortLabel: "5m" },
  { value: 15, unit: "minute", shortLabel: "15m" },
  { value: 30, unit: "minute", shortLabel: "30m" },
  { value: 1, unit: "hour", shortLabel: "1h" },
  { value: 2, unit: "hour", shortLabel: "2h" },
  { value: 6, unit: "hour", shortLabel: "6h" },
  { value: 12, unit: "hour", shortLabel: "12h" },
  { value: 1, unit: "day", shortLabel: "1d" },
  { value: 3, unit: "day", shortLabel: "3d" },
  { value: 1, unit: "week", shortLabel: "1w" },
  { value: 1, unit: "month", shortLabel: "1mo" },
];

interface TimestampPickerProps {
  value?: Timestamp;
  onChange: (arg?: Timestamp) => void;
  label?: string;
  description?: string;
  placeholder?: string;
  isFuture?: boolean;
  disableExcludePast?: boolean;
}

const formatTimestamp = (date: Date) =>
  dayjs(date).format("MMM D, YYYY · h:mm A");

const TimestampPicker = ({
  value,
  onChange,
  label,
  description,
  placeholder = "Select date and time",
  isFuture = false,
  disableExcludePast = false,
}: TimestampPickerProps) => {
  const [opened, setOpened] = useState(false);
  const [activeView, setActiveView] = useState<"relative" | "custom">(
    "relative",
  );
  const externalDate = useMemo(
    () => (value ? Timestamp.toDate(value) : null),
    [value],
  );
  const [selectedDate, setSelectedDate] = useState<Date | null>(externalDate);
  const [timeValue, setTimeValue] = useState(
    externalDate ? dayjs(externalDate).format("HH:mm") : dayjs().format("HH:mm"),
  );
  const [validationError, setValidationError] = useState<string>();
  const timeZone = useMemo(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "Local time",
    [],
  );

  useEffect(() => {
    setSelectedDate(externalDate);
    if (externalDate) setTimeValue(dayjs(externalDate).format("HH:mm"));
    setValidationError(undefined);
  }, [externalDate?.getTime()]);

  const customTimestamp = useMemo(() => {
    if (!selectedDate || !/^\d{2}:\d{2}$/.test(timeValue)) return null;
    const [hours, minutes] = timeValue.split(":").map(Number);
    if (hours > 23 || minutes > 59) return null;

    return dayjs(selectedDate)
      .hour(hours)
      .minute(minutes)
      .second(0)
      .millisecond(0)
      .toDate();
  }, [selectedDate, timeValue]);

  const chooseRelativeTime = (
    amount: number,
    unit: RelativeUnit,
  ) => {
    const target = isFuture
      ? dayjs().add(amount, unit)
      : dayjs().subtract(amount, unit);
    onChange(Timestamp.fromDate(target.toDate()));
    setOpened(false);
  };

  const applyCustomTime = () => {
    if (!customTimestamp) {
      setValidationError("Choose a valid date and time.");
      return;
    }

    if (
      isFuture &&
      !disableExcludePast &&
      dayjs(customTimestamp).isBefore(dayjs())
    ) {
      setValidationError("Choose a date and time in the future.");
      return;
    }

    setValidationError(undefined);
    onChange(Timestamp.fromDate(customTimestamp));
    setOpened(false);
  };

  return (
    <Input.Wrapper label={label} description={description} className="w-full">
      <Popover
        opened={opened}
        onChange={setOpened}
        width={520}
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
                strokeWidth={2.25}
                className={externalDate ? "text-slate-600" : "text-slate-400"}
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
            onClick={() => setOpened((current) => !current)}
            className="w-full text-left"
            styles={{
              input: {
                minWidth: "220px",
                cursor: "pointer",
                textAlign: "left",
                "&:hover": { borderColor: "#cbd5e1" },
              },
            }}
          >
            {externalDate ? (
              <span className="text-[0.78rem] font-bold text-slate-700">
                {formatTimestamp(externalDate)}
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
                <CalendarDays size={14} strokeWidth={2.25} />
              </span>
              <div className="min-w-0">
                <p className="text-[0.72rem] font-bold text-slate-700">
                  Choose timestamp
                </p>
                <p className="mt-0.5 truncate text-[0.62rem] font-semibold text-slate-400">
                  {timeZone} · {isFuture ? "Future time" : "Local time"}
                </p>
              </div>
            </div>
            {externalDate && (
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
                {
                  label: isFuture ? "From now" : "Before now",
                  value: "relative",
                },
                { label: "Date and time", value: "custom" },
              ]}
              onChange={(nextView) => {
                setActiveView(nextView as "relative" | "custom");
                setValidationError(undefined);
              }}
            />

            <div className="mt-3">
              {activeView === "relative" ? (
                <div>
                  <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
                    {PRESETS.map((preset) => (
                      <Button
                        key={`${preset.value}-${preset.unit}`}
                        type="button"
                        variant="default"
                        onClick={() =>
                          chooseRelativeTime(preset.value, preset.unit)
                        }
                        aria-label={`${isFuture ? "In" : "Before now by"} ${preset.value} ${preset.unit}${preset.value === 1 ? "" : "s"}`}
                        styles={{
                          root: {
                            height: "38px",
                            paddingInline: "8px",
                            fontSize: "0.72rem",
                            fontWeight: 700,
                          },
                        }}
                      >
                        <span className="flex items-center gap-1">
                          <span className="text-slate-400">
                            {isFuture ? "in" : "−"}
                          </span>
                          {preset.shortLabel}
                        </span>
                      </Button>
                    ))}
                  </div>
                  <p className="mt-3 text-center text-[0.62rem] font-semibold text-slate-400">
                    Relative timestamps are calculated when selected.
                  </p>
                </div>
              ) : (
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    applyCustomTime();
                  }}
                >
                  <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_170px]">
                    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 p-1">
                      <DatePicker
                        value={selectedDate}
                        onChange={(nextDate) => {
                          setSelectedDate(
                            nextDate ? dayjs(nextDate).toDate() : null,
                          );
                          setValidationError(undefined);
                        }}
                        minDate={
                          isFuture && !disableExcludePast
                            ? new Date()
                            : undefined
                        }
                        styles={{
                          calendarHeader: {
                            fontSize: "0.78rem",
                            fontWeight: 700,
                          },
                          weekday: {
                            color: "#94a3b8",
                            fontSize: "0.66rem",
                            fontWeight: 700,
                          },
                          day: {
                            borderRadius: "7px",
                            fontSize: "0.74rem",
                            fontWeight: 600,
                          },
                        }}
                      />
                    </div>

                    <div className="flex min-w-0 flex-col gap-3">
                      <TimeInput
                        label="Time"
                        value={timeValue}
                        leftSection={
                          <Clock3 size={12} className="text-slate-400" />
                        }
                        onChange={(event) => {
                          setTimeValue(event.target.value);
                          setValidationError(undefined);
                        }}
                      />

                      <div
                        className={`rounded-lg border px-3 py-2.5 ${
                          validationError
                            ? "border-red-200 bg-red-50"
                            : "border-slate-200 bg-slate-50/70"
                        }`}
                      >
                        <p
                          className={`text-[0.6rem] font-bold uppercase tracking-[0.06em] ${
                            validationError ? "text-red-500" : "text-slate-400"
                          }`}
                        >
                          {validationError ? "Check timestamp" : "Selection"}
                        </p>
                        <p
                          className={`mt-1 text-[0.7rem] font-bold leading-relaxed ${
                            validationError ? "text-red-700" : "text-slate-700"
                          }`}
                        >
                          {validationError ??
                            (customTimestamp
                              ? formatTimestamp(customTimestamp)
                              : "Choose a date and time")}
                        </p>
                      </div>

                      <Button
                        type="submit"
                        fullWidth
                        color="dark"
                        disabled={!customTimestamp}
                        className="mt-auto"
                      >
                        Apply timestamp
                      </Button>
                    </div>
                  </div>
                </form>
              )}
            </div>
          </div>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default TimestampPicker;
