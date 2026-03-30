import { ListUserOptions, User } from "@/apis/corev1/corev1";
import { getClientCore } from "@/utils/client";
import { printUserWithEmail } from "@/utils/pb";
import { Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

const SelectUser = (props: {
  defaultValue?: string;
  description?: string;
  required?: boolean;
  label?: string;
  clearable?: boolean;

  onChange: (item?: User) => void;
}) => {
  const { isLoading, isSuccess, data } = useQuery({
    queryKey: ["selectUserComponent"],
    queryFn: async () => {
      return await getClientCore().listUser(
        ListUserOptions.create({
          common: {
            itemsPerPage: 1000,
          },
        }),
      );
    },
  });

  if (!data) {
    return <></>;
  }

  const itemList = data.response;

  const userList = itemList.items.map((x) => ({
    value: x.metadata!.name,
    label: printUserWithEmail(x),
  }));

  return (
    <div className="w-full">
      <Select
        label={props.label ? props.label : "Select User"}
        required={props.required}
        description={props.description}
        disabled={userList.length === 0}
        clearable={props.clearable}
        searchable
        data={userList}
        defaultValue={props.defaultValue}
        onChange={(v) => {
          if (!v) {
            props.onChange();
            return;
          }
          props.onChange(itemList.items.find((x) => x.metadata?.name === v));
        }}
      />
    </div>
  );
};

export default SelectUser;
