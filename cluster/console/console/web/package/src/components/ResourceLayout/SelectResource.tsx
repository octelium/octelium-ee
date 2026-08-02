import {
  printResourceNameWithDisplay,
  Resource,
} from "@/utils/pb";
import { Alert, Button, Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { listResourcesForSelect } from "./listResourcesForSelect";

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

  const { isLoading, isError, error, data, refetch } = useQuery({
    queryKey: ["listSelectComponent", api, kind],
    queryFn: () => listResourcesForSelect(api, kind),
  });

  const label = props.labelDefault ? `Select ${kind}` : props.label;

  if (isLoading) {
    return (
      <Select
        label={label}
        required={props.required}
        description={props.description}
        data={[]}
        disabled
        placeholder="Loading…"
      />
    );
  }

  if (isError) {
    return (
      <Alert color="red" title={`Could not load ${kind}s`}>
        <div className="flex flex-col gap-2">
          <span className="text-xs">{error.message}</span>
          <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  if (!data) {
    return (
      <Alert color="red" title={`Could not load ${kind}s`}>
        <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
          Retry
        </Button>
      </Alert>
    );
  }

  const rscList = data.map((x) => ({
    value: x.metadata!.name,
    label: printResourceNameWithDisplay(x),
  }));

  return (
    <Select
      label={label}
      required={props.required}
      description={props.description}
      clearable={props.clearable}
      searchable
      data={rscList}
      value={props.defaultValue ?? null}
      disabled={rscList.length === 0}
      placeholder={
        rscList.length === 0 ? `No ${kind}s found` : `Select ${kind}…`
      }
      nothingFoundMessage={`No ${kind}s match your search`}
      renderOption={({ option }) => {
        const item = data.find(
          (x) => x.metadata!.name === option.value,
        );
        if (!item) return null;

        return (
          <div className="flex flex-col gap-0.5 py-0.5">
            <span className="text-[0.78rem] font-bold text-slate-800">
              {item.metadata!.name}
            </span>
            {item.metadata!.displayName && (
              <span className="text-[0.7rem] font-semibold text-slate-500">
                {item.metadata!.displayName}
              </span>
            )}
            {item.metadata!.description && (
              <span className="text-[0.68rem] font-semibold text-slate-400 truncate">
                {item.metadata!.description}
              </span>
            )}
          </div>
        );
      }}
      onChange={(v) => {
        if (!v) {
          props.onChange();
          return;
        }
        props.onChange(data.find((x) => x.metadata?.name === v));
      }}
    />
  );
};

export default SelectResource;
