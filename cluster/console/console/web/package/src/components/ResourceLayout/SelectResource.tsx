import {
  CommonListOptions,
  CommonListOptions_OrderBy_Type,
} from "@/apis/metav1/metav1";
import {
  getClientResourceList,
  getPBResourceListFromAPI,
  printResourceNameWithDisplay,
  Resource,
  ResourceList,
} from "@/utils/pb";
import { Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";

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

  const req = React.useMemo(
    () =>
      // @ts-ignore
      getPBResourceListFromAPI(api)![`List${kind}Options`]["create"]({
        common: CommonListOptions.create({
          itemsPerPage: 1000,
          orderBy: { type: CommonListOptions_OrderBy_Type.NAME },
        }),
      }),
    [api, kind],
  );

  const { isLoading, data } = useQuery({
    queryKey: ["listSelectComponent", api, kind],
    queryFn: async () => {
      // @ts-ignore
      const d = await getClientResourceList(api)?.[`list${kind}`](req);
      return d;
    },
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

  const itemList = data?.["response"] as ResourceList | undefined;

  if (!itemList) return null;

  const rscList = itemList.items.map((x) => ({
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
      defaultValue={props.defaultValue}
      value={props.defaultValue}
      disabled={rscList.length === 0}
      placeholder={
        rscList.length === 0 ? `No ${kind}s found` : `Select ${kind}…`
      }
      nothingFoundMessage={`No ${kind}s match your search`}
      renderOption={({ option }) => {
        const item = itemList.items.find(
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
        props.onChange(itemList.items.find((x) => x.metadata?.name === v));
      }}
    />
  );
};

export default SelectResource;
