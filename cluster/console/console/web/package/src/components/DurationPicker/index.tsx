import { Duration } from "@/apis/metav1/metav1";
import {
  Button,
  Group,
  Input,
  InputBase,
  NumberInput,
  Popover,
  Select,
  Tabs,
  Text,
} from "@mantine/core";
import { useState } from "react";
import { LuTimer } from "react-icons/lu";
import { match } from "ts-pattern";

import { toNumOrZero } from "@/utils";
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
    .otherwise(() => ({ oneofKind: "seconds" as const, seconds: val })); // Fallback

  return Duration.create({ type: typePayload as any });
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
  const [activeTab, setActiveTab] = useState<string | null>("presets");

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
    const num = parseInt(numStr, 10);

    onChange(createDuration(num, unit));
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

  return (
    <Input.Wrapper label={title} description={description} className="w-full">
      <Popover
        opened={opened}
        onChange={setOpened}
        width={320}
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
            leftSection={<LuTimer size={16} />}
            onClick={() => setOpened((o) => !o)}
            className="w-full"
            rightSection={
              displayValue ? (
                <Text
                  size="xs"
                  c="dimmed"
                  onClick={handleClear}
                  className="hover:text-red-500 mr-2"
                >
                  ✕
                </Text>
              ) : null
            }
          >
            {displayValue || (
              <Text c="dimmed" size="sm">
                {placeholder}
              </Text>
            )}
          </InputBase>
        </Popover.Target>

        <Popover.Dropdown p="md">
          <Tabs
            value={activeTab}
            onChange={setActiveTab}
            variant="outline"
            defaultValue="presets"
          >
            <Tabs.List grow mb="md">
              <Tabs.Tab value="presets">Presets</Tabs.Tab>
              <Tabs.Tab value="custom">Custom</Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="presets">
              <Select
                data={timeRangePickList}
                placeholder="Quick select duration..."
                searchable
                value={null}
                onChange={handlePresetChange}
                comboboxProps={{ withinPortal: false }}
              />
            </Tabs.Panel>

            <Tabs.Panel value="custom">
              <div className="flex flex-col gap-4 pt-2">
                <Group align="flex-end" grow gap="sm">
                  <NumberInput
                    label="Value"
                    value={customVal}
                    min={1}
                    max={100000}
                    allowNegative={false}
                    allowDecimal={false}
                    onChange={(v) => {
                      setCustomVal(toNumOrZero(`${v}`));
                    }}
                  />
                  <Select
                    label="Time Unit"
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
                    comboboxProps={{ withinPortal: false }}
                  />
                </Group>

                <Button
                  fullWidth
                  onClick={applyCustomChange}
                  disabled={customVal === "" || customVal <= 0}
                >
                  Apply Duration
                </Button>
              </div>
            </Tabs.Panel>
          </Tabs>
        </Popover.Dropdown>
      </Popover>
    </Input.Wrapper>
  );
};

export default DurationPicker;
