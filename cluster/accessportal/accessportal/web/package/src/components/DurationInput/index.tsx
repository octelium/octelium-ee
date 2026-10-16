import * as MetaP from "@/apis/metav1/metav1";
import { Button, NumberInput, Popover, Select } from "@mantine/core";
import { ChevronDown, Clock3 } from "lucide-react";
import * as React from "react";

import {
  DURATION_UNIT_OPTIONS,
  DurationUnit,
  durationToParts,
  durationToSeconds,
  formatDuration,
  partsToDuration,
} from "@/utils";

const PRESETS: {
  amount: number;
  unit: DurationUnit;
  label: string;
  accessibleLabel: string;
}[] = [
  { amount: 30, unit: "minutes", label: "30 min", accessibleLabel: "30 minutes" },
  { amount: 1, unit: "hours", label: "1 hour", accessibleLabel: "1 hour" },
  { amount: 4, unit: "hours", label: "4 hours", accessibleLabel: "4 hours" },
  { amount: 8, unit: "hours", label: "8 hours", accessibleLabel: "8 hours" },
  { amount: 1, unit: "days", label: "1 day", accessibleLabel: "1 day" },
];

const DurationInput = (props: {
  value?: MetaP.Duration;
  onChange: (v: MetaP.Duration) => void;
}) => {
  const { unit, amount } = durationToParts(props.value);
  const current = partsToDuration(unit, amount);
  const currentSeconds = durationToSeconds(current);
  const activePreset = PRESETS.find(
    (preset) =>
      durationToSeconds(partsToDuration(preset.unit, preset.amount)) === currentSeconds,
  );
  const [opened, setOpened] = React.useState(false);
  const [customAmount, setCustomAmount] = React.useState<number | "">(amount);
  const [customUnit, setCustomUnit] = React.useState<DurationUnit>(unit);
  const titleId = React.useId();
  const customIsValid =
    typeof customAmount === "number" &&
    Number.isInteger(customAmount) &&
    customAmount >= 1 &&
    customAmount <= 100_000;
  const customDuration = customIsValid
    ? partsToDuration(customUnit, customAmount)
    : undefined;

  const applyCustom = () => {
    if (!customDuration) return;
    props.onChange(customDuration);
    setOpened(false);
  };

  const closeOnEscape = (event: React.KeyboardEvent) => {
    if (!opened || event.key !== "Escape" || event.nativeEvent.isComposing) return;
    const target = event.target as HTMLElement;
    if (
      target.getAttribute("aria-haspopup") === "listbox" &&
      target.getAttribute("aria-expanded") === "true"
    ) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    setOpened(false);
  };

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="grid grid-cols-3 gap-1.5">
        {PRESETS.map((preset) => (
          <Button
            key={preset.label}
            type="button"
            size="compact-sm"
            variant={activePreset === preset ? "filled" : "default"}
            color="ink"
            aria-label={preset.accessibleLabel}
            aria-pressed={activePreset === preset}
            className="!h-9 !min-w-0 !px-2 [@media(pointer:coarse)]:!h-11"
            onClick={() => props.onChange(partsToDuration(preset.unit, preset.amount))}
          >
            {preset.label}
          </Button>
        ))}

        <Popover
          opened={opened}
          onChange={setOpened}
          width={320}
          position="bottom-end"
          trapFocus
          returnFocus
          closeOnEscape={false}
        >
          <Popover.Target>
            <Button
              type="button"
              size="compact-sm"
              variant={activePreset ? "default" : "filled"}
              color="ink"
              aria-label="Custom duration"
              aria-pressed={!activePreset}
              aria-haspopup="dialog"
              aria-expanded={opened}
              data-mantine-stop-propagation={opened || undefined}
              onKeyDownCapture={closeOnEscape}
              rightSection={
                <ChevronDown
                  size={12}
                  aria-hidden
                  className={opened ? "rotate-180" : undefined}
                />
              }
              className="!h-9 !min-w-0 !px-2 [@media(pointer:coarse)]:!h-11"
              onClick={() => {
                if (!opened) {
                  setCustomAmount(amount);
                  setCustomUnit(unit);
                }
                setOpened(!opened);
              }}
            >
              Custom
            </Button>
          </Popover.Target>

          <Popover.Dropdown
            p={0}
            role="dialog"
            aria-labelledby={titleId}
            data-mantine-stop-propagation
            onKeyDownCapture={closeOnEscape}
            style={{ maxWidth: "calc(100vw - 24px)", overflow: "hidden" }}
          >
            <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-4 py-3">
              <Clock3 size={14} aria-hidden className="text-slate-500" />
              <span id={titleId} className="text-body font-semibold text-slate-800">
                Custom duration
              </span>
            </div>

            <div className="flex flex-col gap-4 bg-white p-4">
              <div className="grid grid-cols-2 gap-3">
                <NumberInput
                  label="Amount"
                  aria-label="Duration amount"
                  data-autofocus
                  data-mantine-stop-propagation
                  min={1}
                  max={100_000}
                  allowDecimal={false}
                  allowNegative={false}
                  clampBehavior="strict"
                  value={customAmount}
                  onChange={(value) => setCustomAmount(value === "" ? "" : Number(value))}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" && !event.nativeEvent.isComposing) {
                      event.preventDefault();
                      event.stopPropagation();
                      applyCustom();
                    }
                  }}
                />
                <Select
                  label="Unit"
                  aria-label="Duration unit"
                  data={DURATION_UNIT_OPTIONS}
                  value={customUnit}
                  allowDeselect={false}
                  comboboxProps={{ withinPortal: false }}
                  attributes={{ input: { "data-mantine-stop-propagation": true } }}
                  onChange={(value) => value && setCustomUnit(value as DurationUnit)}
                />
              </div>

              <p className="text-xs font-normal text-slate-500" aria-live="polite">
                {customDuration
                  ? `Access for ${formatDuration(customDuration)}`
                  : "Enter a whole number from 1 to 100,000"}
              </p>

              <div className="flex items-center justify-end gap-2">
                <Button
                  type="button"
                  variant="default"
                  size="xs"
                  data-mantine-stop-propagation
                  onClick={() => setOpened(false)}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  color="ink"
                  size="xs"
                  disabled={!customIsValid}
                  data-mantine-stop-propagation
                  onClick={applyCustom}
                >
                  Apply duration
                </Button>
              </div>
            </div>
          </Popover.Dropdown>
        </Popover>
      </div>

      {!activePreset && (
        <p className="text-xs font-normal text-slate-600">
          Custom duration: {formatDuration(current)}
        </p>
      )}
    </div>
  );
};

export default DurationInput;
