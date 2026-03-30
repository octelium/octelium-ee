import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  Button,
  Group,
  Input,
  InputBase,
  Popover,
  Select,
  Tabs,
  Text,
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
  placeholder = "Select Date & Time",
  isFuture = false,
  disableExcludePast,
}: TimestampPickerProps) => {
  const [opened, setOpened] = useState(false);
  const [activeTab, setActiveTab] = useState<string | null>("presets");

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

  return (
    <Input.Wrapper label={label} description={description}>
      <Popover
        opened={opened}
        onChange={setOpened}
        width="auto"
        position="bottom-start"
        withArrow
        shadow="xl"
        transitionProps={{ transition: "pop", duration: 200 }}
      >
        <Popover.Target>
          <InputBase
            component="button"
            type="button"
            pointer
            leftSection={<LuClock size={16} />}
            onClick={() => setOpened((o) => !o)}
            className="min-w-[260px]"
          >
            {displayValue || (
              <Text c="dimmed" size="sm">
                {placeholder}
              </Text>
            )}
          </InputBase>
        </Popover.Target>

        <Popover.Dropdown p="md">
          <Tabs value={activeTab} onChange={setActiveTab} variant="outline">
            <Tabs.List grow mb="md">
              <Tabs.Tab value="presets" leftSection={<LuTimer size={14} />}>
                Presets
              </Tabs.Tab>
              <Tabs.Tab value="custom" leftSection={<LuCalendar size={14} />}>
                Custom
              </Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="presets">
              <div className="min-w-[280px]">
                <Select
                  data={timeRangePickList}
                  placeholder="Quick select relative time"
                  searchable
                  onChange={handlePresetChange}
                  comboboxProps={{ withinPortal: false }}
                />
              </div>
            </Tabs.Panel>

            <Tabs.Panel value="custom">
              <Group align="flex-start" gap="lg" wrap="nowrap">
                <div className="border rounded-md p-1 bg-gray-50/50">
                  <DatePicker
                    value={internalDate}
                    onChange={handleDateChange}
                    minDate={isFuture ? new Date() : undefined}
                  />
                </div>
                <div className="flex flex-col h-full justify-between gap-4 py-2">
                  <TimeInput
                    label="Set Time"
                    ref={timeRef}
                    defaultValue={dayjs(internalDate || undefined).format(
                      "HH:mm",
                    )}
                  />
                  <div className="mt-auto">
                    <Button
                      fullWidth
                      onClick={applyCustomChange}
                      disabled={!internalDate}
                    >
                      Apply
                    </Button>
                  </div>
                </div>
              </Group>
            </Tabs.Panel>
          </Tabs>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default TimestampPicker;
