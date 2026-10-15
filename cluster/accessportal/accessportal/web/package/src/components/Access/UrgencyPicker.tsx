import * as AccessP from "@/apis/accessv1/accessv1";
import { Select } from "@mantine/core";
import { twMerge } from "tailwind-merge";

import { URGENCY_OPTIONS, toneClasses, urgencyMeta } from "@/utils";

const UrgencyPicker = (props: {
  value: AccessP.Request_Spec_Urgency;
  onChange: (value: AccessP.Request_Spec_Urgency) => void;
}) => {
  const meta = urgencyMeta(props.value);

  return (
    <Select
      aria-label="Urgency"
      value={String(props.value)}
      onChange={(value) => {
        if (value !== null) props.onChange(Number(value));
      }}
      allowDeselect={false}
      data={URGENCY_OPTIONS.map((option) => ({
        value: String(option.value),
        label: option.label,
      }))}
      leftSection={
        <span
          aria-hidden
          className={twMerge("h-2 w-2 rounded-full", toneClasses[meta.tone].dot)}
        />
      }
      renderOption={({ option }) => (
        <span className="flex items-center gap-2.5">
          <span
            aria-hidden
            className={twMerge(
              "h-2 w-2 rounded-full",
              toneClasses[urgencyMeta(Number(option.value)).tone].dot,
            )}
          />
          {option.label}
        </span>
      )}
    />
  );
};

export default UrgencyPicker;
