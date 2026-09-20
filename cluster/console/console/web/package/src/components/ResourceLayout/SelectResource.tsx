import { Resource } from "@/utils/pb";
import ResourcePicker from "./resourcePicker/ResourcePicker";
import TriggerField, { SelectedChip } from "./resourcePicker/TriggerField";
import { useResourcePicker } from "./resourcePicker/useResourcePicker";

const SelectResource = (props: {
  api: string;
  kind: string;
  defaultValue?: string;
  description?: string;
  required?: boolean;
  label?: string;
  labelDefault?: boolean;
  clearable?: boolean;
  onChange: (item?: Resource) => void;
}) => {
  const { api, kind } = props;

  const picker = useResourcePicker({
    api,
    kind,
    onCreated: (item) => props.onChange(item),
  });

  const label = props.labelDefault ? `Select ${kind}` : props.label;
  const selected = props.defaultValue;
  const selectedItem = picker.resolve(selected);

  return (
    <div className="w-full">
      <TriggerField
        label={label}
        description={props.description}
        required={props.required}
        placeholder={`Select a ${kind}…`}
        empty={!selected}
        onOpen={picker.open}
        onClear={
          props.clearable && selected ? () => props.onChange() : undefined
        }
      >
        {selected && (
          <SelectedChip
            name={selected}
            item={selectedItem}
            api={api}
            kind={kind}
          />
        )}
      </TriggerField>

      <ResourcePicker
        opened={picker.opened}
        onClose={picker.close}
        api={api}
        kind={kind}
        selected={selected ? [selected] : []}
        items={picker.items}
        isLoading={picker.isLoading}
        isError={picker.isError}
        errorMessage={picker.errorMessage}
        onRetry={picker.refetch}
        search={picker.search}
        onSearchChange={picker.setSearch}
        onPick={(item) => {
          props.onChange(item);
          picker.close();
        }}
        onClear={
          props.clearable && selected
            ? () => {
                props.onChange();
                picker.close();
              }
            : undefined
        }
        onCreate={picker.canCreate ? picker.openCreate : undefined}
      />
    </div>
  );
};

export default SelectResource;
