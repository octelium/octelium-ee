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

  //@ts-ignore
  let req = getPBResourceListFromAPI(api)![`List${kind}Options`]["create"]({
    common: CommonListOptions.create({
      itemsPerPage: 1000,
      orderBy: {
        type: CommonListOptions_OrderBy_Type.NAME,
      },
    }),
  });

  const { isLoading, isSuccess, data, error } = useQuery({
    queryKey: [`listSelectComponent`, api, kind],
    queryFn: async () => {
      // @ts-ignore
      const d = await getClientResourceList(api)?.[`list${kind}`](req);
      return d;
    },
  });

  if (!data || isLoading) {
    return <></>;
  }

  const itemList = data["response"] as ResourceList | undefined;

  if (!itemList) {
    return <></>;
  }

  const rscList = itemList?.items.map((x) => ({
    value: x.metadata!.name,
    label: printResourceNameWithDisplay(x),
  }));

  return (
    <div className="w-full">
      <Select
        label={props.labelDefault ? `Select ${props.kind}` : props.label}
        required={props.required}
        description={props.description}
        clearable={props.clearable}
        searchable
        data={rscList}
        defaultValue={props.defaultValue}
        disabled={rscList.length === 0}
        renderOption={({ option }) => {
          const item = itemList.items.find(
            (x) => x.metadata!.name === option.value,
          );
          if (!item) {
            return <></>;
          }

          return (
            <div>
              <div className="flex flex-col font-bold">
                <div className="text-black">{item.metadata!.name}</div>
                {item.metadata!.displayName && (
                  <div className="text-gray-600 text-xs">
                    {item.metadata!.displayName}
                  </div>
                )}
              </div>
            </div>
          );
        }}
        onChange={(v) => {
          if (!v) {
            props.onChange();
            return;
          }
          props.onChange(itemList?.items.find((x) => x.metadata?.name === v));
        }}
      />
    </div>
  );
};

export default SelectResource;
