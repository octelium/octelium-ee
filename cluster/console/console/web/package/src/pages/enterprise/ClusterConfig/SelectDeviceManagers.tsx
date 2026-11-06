import SelectResource from "@/components/ResourceLayout/SelectResource";
import { ActionIcon, Tooltip } from "@mantine/core";
import { ArrowDown, ArrowUp, Trash2 } from "lucide-react";
import * as React from "react";

const SelectDeviceManagers = (props: {
  value: string[];
  onChange: (value: string[]) => void;
}) => {
  const { value } = props;
  const [picker, setPicker] = React.useState(0);
  const move = (index: number, offset: number) => {
    const next = [...value];
    const [name] = next.splice(index, 1);
    next.splice(index + offset, 0, name);
    props.onChange(next);
  };

  return (
    <div className="flex flex-col gap-3">
      <SelectResource
        key={picker}
        api="enterprise"
        kind="DeviceManager"
        label="Add a device manager"
        description="Only the listed managers bind Devices. When the list is empty, every manager is considered in ascending name order."
        onChange={(item) => {
          const name = item?.metadata?.name;
          if (!name) return;
          if (!value.includes(name)) props.onChange([...value, name]);
          setPicker((current) => current + 1);
        }}
      />
      {value.length > 0 && (
        <ol
          aria-label="Selection order"
          className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white"
        >
          {value.map((name, index) => (
            <li key={name} className="flex items-center gap-2.5 px-3 py-1.5">
              <span className="w-5 shrink-0 text-center text-xs font-semibold tabular-nums text-slate-500">
                {index + 1}
              </span>
              <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-800">
                {name}
              </span>
              <Tooltip label="Move up" withArrow>
                <ActionIcon
                  type="button"
                  variant="subtle"
                  color="gray"
                  size="sm"
                  aria-label={`Move ${name} up`}
                  disabled={index === 0}
                  onClick={() => move(index, -1)}
                >
                  <ArrowUp size={13} strokeWidth={2.2} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label="Move down" withArrow>
                <ActionIcon
                  type="button"
                  variant="subtle"
                  color="gray"
                  size="sm"
                  aria-label={`Move ${name} down`}
                  disabled={index === value.length - 1}
                  onClick={() => move(index, 1)}
                >
                  <ArrowDown size={13} strokeWidth={2.2} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label="Remove" withArrow>
                <ActionIcon
                  type="button"
                  variant="subtle"
                  color="gray"
                  size="sm"
                  aria-label={`Remove ${name}`}
                  onClick={() =>
                    props.onChange(value.filter((item) => item !== name))
                  }
                >
                  <Trash2 size={13} strokeWidth={2.2} />
                </ActionIcon>
              </Tooltip>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
};

export default SelectDeviceManagers;
