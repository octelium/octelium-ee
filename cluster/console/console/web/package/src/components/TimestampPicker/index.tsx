import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  Button,
  Group,
  Input,
  InputBase,
  Popover,
  Select,
} from "@mantine/core";
import { DatePicker, TimeInput } from "@mantine/dates";
import dayjs from "dayjs";
import { useRef, useState } from "react";
import { LuCalendar, LuClock, LuTimer } from "react-icons/lu";
import { timeRangePickList } from "../AccessLogViewer/utils";

interface TimestampPickerProps {
  value?: Timestamp;
  onChange: (arg?: Timestamp) => void;
  label?: string;
  description?: string;
  placeholder?: string;
  isFuture?: boolean;
  disableExcludePast?: boolean;
}

const TimestampPicker = ({
  value,
  onChange,
  label,
  description,
  placeholder = "Select date & time",
  isFuture = false,
  disableExcludePast,
}: TimestampPickerProps) => {
  const [opened, setOpened] = useState(false);
  const [activeTab, setActiveTab] = useState<"presets" | "custom">("presets");

  const initialDate = value ? Timestamp.toDate(value) : null;
  const [internalDate, setInternalDate] = useState<Date | null>(initialDate);
  const timeRef = useRef<HTMLInputElement>(null);

  const displayValue = value
    ? dayjs(Timestamp.toDate(value)).format("MMM D, YYYY h:mm A")
    : "";

  const handlePresetChange = (v: string | null) => {
    if (!v) return;
    const [numStr, unit] = v.split(" ");
    const num = parseInt(numStr, 10);
    const targetDate = isFuture
      ? dayjs().add(num, unit as dayjs.ManipulateType)
      : dayjs().subtract(num, unit as dayjs.ManipulateType);
    onChange(Timestamp.fromDate(targetDate.toDate()));
    setOpened(false);
  };

  const handleDateChange = (val: string | Date | null) => {
    if (!val) {
      setInternalDate(null);
      return;
    }
    setInternalDate(dayjs(val).toDate());
  };

  const applyCustomChange = () => {
    if (!internalDate) return;
    const timeVal = timeRef.current?.value || "00:00";
    const [hours, minutes] = timeVal.split(":").map(Number);
    const finalDate = dayjs(internalDate)
      .hour(hours)
      .minute(minutes)
      .second(0)
      .toDate();
    if (isFuture && dayjs(finalDate).isBefore(dayjs())) {
      alert("Please select a future date and time.");
      return;
    }
    onChange(Timestamp.fromDate(finalDate));
    setOpened(false);
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
        label ? (
          <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-600 mb-1 block">
            {label}
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
              <LuClock
                size={13}
                className={displayValue ? "text-slate-500" : "text-slate-400"}
              />
            }
            onClick={() => setOpened((o) => !o)}
            styles={{
              input: {
                minWidth: "220px",
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
          }}
        >
          <div className="min-w-[300px]">
            <div className="border-b border-slate-100 bg-slate-50 rounded-t-[10px] overflow-hidden">
              <Button.Group style={{ display: "flex" }}>
                {[
                  {
                    value: "presets" as const,
                    label: "Presets",
                    Icon: LuTimer,
                  },
                  {
                    value: "custom" as const,
                    label: "Custom",
                    Icon: LuCalendar,
                  },
                ].map(({ value, label: lbl, Icon }, i) => (
                  <Button
                    key={value}
                    onClick={() => setActiveTab(value)}
                    styles={tabBtnStyles(
                      activeTab === value,
                      i === 0 ? "left" : "right",
                    )}
                    leftSection={<Icon size={12} />}
                    style={{ flex: 1 }}
                  >
                    {lbl}
                  </Button>
                ))}
              </Button.Group>
            </div>

            <div className="p-3 bg-white rounded-b-[10px]">
              {activeTab === "presets" && (
                <Select
                  data={timeRangePickList}
                  placeholder="Quick select relative time"
                  searchable
                  onChange={handlePresetChange}
                  comboboxProps={{ withinPortal: true }}
                  styles={{
                    input: {
                      fontSize: "0.78rem",
                      fontWeight: 600,
                      fontFamily: "Ubuntu, sans-serif",
                      border: "1px solid #e2e8f0",
                      borderRadius: "6px",
                      color: "#1e293b",
                      boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
                    },
                    option: {
                      fontSize: "0.78rem",
                      fontWeight: 600,
                      fontFamily: "Ubuntu, sans-serif",
                    },
                  }}
                />
              )}

              {activeTab === "custom" && (
                <Group align="flex-start" gap="md" wrap="nowrap">
                  <div className="border border-slate-200 rounded-lg overflow-hidden bg-slate-50/50">
                    <DatePicker
                      value={internalDate}
                      onChange={handleDateChange}
                      minDate={isFuture ? new Date() : undefined}
                      styles={{
                        calendarHeader: {
                          fontSize: "0.78rem",
                          fontWeight: 700,
                          fontFamily: "Ubuntu, sans-serif",
                        },
                        weekday: {
                          fontSize: "0.68rem",
                          fontWeight: 700,
                          color: "#94a3b8",
                          fontFamily: "Ubuntu, sans-serif",
                        },
                        day: {
                          fontSize: "0.75rem",
                          fontWeight: 600,
                          fontFamily: "Ubuntu, sans-serif",
                          borderRadius: "6px",
                        },
                      }}
                    />
                  </div>

                  <div className="flex flex-col justify-between gap-4 py-1 min-w-[110px]">
                    <TimeInput
                      label={
                        <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-500">
                          Time
                        </span>
                      }
                      ref={timeRef}
                      defaultValue={dayjs(internalDate || undefined).format(
                        "HH:mm",
                      )}
                      leftSection={
                        <LuClock size={12} className="text-slate-400" />
                      }
                      styles={{
                        input: {
                          fontSize: "0.78rem",
                          fontWeight: 600,
                          fontFamily: "Ubuntu, sans-serif",
                          border: "1px solid #e2e8f0",
                          borderRadius: "6px",
                          color: "#1e293b",
                          height: "32px",
                          boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
                        },
                      }}
                    />

                    <Button
                      fullWidth
                      onClick={applyCustomChange}
                      disabled={!internalDate}
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
                </Group>
              )}
            </div>
          </div>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default TimestampPicker;
