import { ListSecretOptions } from "@/apis/corev1/corev1";
import {
  API,
  getClient,
  printResourceNameWithDisplay,
  ResourceList,
} from "@/utils/pb";
import { Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

const SelectSecret = (props: {
  api: API;
  defaultValue?: string;
  onChange: (arg: string | null) => void;
  label?: string;
  description?: string;
}) => {
  const { isLoading, isSuccess, data } = useQuery({
    gcTime: 1,
    queryKey: ["selectSecret", props.api],
    queryFn: async () => {
      // @ts-ignore
      return await getClient(props.api)["listSecret"](
        ListSecretOptions.create({}),
      );
    },
  });

  if (!data) {
    return <></>;
  }

  const itemList = data["response"] as ResourceList | undefined;
  if (!itemList) {
    return <></>;
  }

  return (
    <div className="w-full">
      <Select
        label={props.label ? props.label : "Select Secret"}
        description={props.description}
        disabled={itemList.items.length === 0}
        required
        searchable
        data={itemList.items.map((x) => ({
          value: x.metadata!.name,
          label: printResourceNameWithDisplay(x),
        }))}
        defaultValue={props.defaultValue}
        onChange={props.onChange}
      />
    </div>
  );
};

export default SelectSecret;
