import * as MetaP from "@/apis/metav1/metav1";
import { Group, NumberInput, Select } from "@mantine/core";
import {
  DURATION_UNIT_OPTIONS,
  DurationUnit,
  durationToParts,
  partsToDuration,
} from "../../utils";

const DurationInput = (props: {
  value?: MetaP.Duration;
  onChange: (v: MetaP.Duration) => void;
}) => {
  const { unit, amount } = durationToParts(props.value);

  return (
    <Group grow>
      <NumberInput
        min={1}
        value={amount}
        onChange={(v) =>
          props.onChange(partsToDuration(unit, (v as number) || 1))
        }
      />
      <Select
        data={DURATION_UNIT_OPTIONS}
        value={unit}
        allowDeselect={false}
        onChange={(v) =>
          v && props.onChange(partsToDuration(v as DurationUnit, amount))
        }
      />
    </Group>
  );
};

export default DurationInput;
