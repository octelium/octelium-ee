import { Resource } from "@/utils/pb";
import * as React from "react";
import ResourcePicker from "./resourcePicker/ResourcePicker";
import TriggerField, { SelectedChip } from "./resourcePicker/TriggerField";
import { useResourcePicker } from "./resourcePicker/useResourcePicker";

const VISIBLE_CHIPS = 6;

const SelectResourceMultiple = (props: {
  api: string;
  kind: string;
  defaultValue?: string[];
  description?: string;
  required?: boolean;
  label?: string;
  labelDefault?: boolean;
  clearable?: boolean;
  onChange: (itemList?: Resource[]) => void;
}) => {
  const { api, kind } = props;
  const selected = React.useMemo(
    () => props.defaultValue ?? [],
    [props.defaultValue],
  );

  const emit = (
    names: string[],
    resolve: (name: string) => Resource | undefined,
  ) => {
    if (names.length === 0) {
      props.onChange(undefined);
      return;
    }
    props.onChange(
      names
        .map((name) => resolve(name))
        .filter((item): item is Resource => item !== undefined),
    );
  };

  const picker = useResourcePicker({
    api,
    kind,
    onCreated: (item, resolve) => {
      const name = item.metadata!.name;
      if (selected.includes(name)) return;
      emit([...selected, name], (value) =>
        value === name ? item : resolve(value),
      );
    },
  });

  const label = props.labelDefault ? `Select ${kind}` : props.label;
  const shown = selected.slice(0, VISIBLE_CHIPS);
  const overflow = selected.length - shown.length;

  const toggle = (item: Resource) => {
    const name = item.metadata!.name;
    const next = selected.includes(name)
      ? selected.filter((value) => value !== name)
      : [...selected, name];
    emit(next, picker.resolve);
  };

  return (
    <div className="w-full">
      <TriggerField
        label={label}
        description={props.description}
        required={props.required}
        placeholder={`Select one or more ${kind} resources…`}
        empty={selected.length === 0}
        onOpen={picker.open}
        onClear={
          props.clearable && selected.length > 0
            ? () => props.onChange(undefined)
            : undefined
        }
      >
        {shown.map((name) => (
          <SelectedChip
            key={name}
            name={name}
            item={picker.resolve(name)}
            api={api}
            kind={kind}
            onRemove={() =>
              emit(
                selected.filter((value) => value !== name),
                picker.resolve,
              )
            }
          />
        ))}
        {overflow > 0 && (
          <span className="shrink-0 rounded-lg border border-slate-200 bg-slate-50 px-1.5 py-1 text-micro font-semibold text-slate-600">
            +{overflow} more
          </span>
        )}
      </TriggerField>

      <ResourcePicker
        opened={picker.opened}
        onClose={picker.close}
        api={api}
        kind={kind}
        multiple
        selected={selected}
        items={picker.items}
        isLoading={picker.isLoading}
        isError={picker.isError}
        errorMessage={picker.errorMessage}
        onRetry={picker.refetch}
        search={picker.search}
        onSearchChange={picker.setSearch}
        onPick={toggle}
        onClear={
          selected.length > 0 ? () => props.onChange(undefined) : undefined
        }
        onCreate={picker.canCreate ? picker.openCreate : undefined}
      />
    </div>
  );
};

export default SelectResourceMultiple;
