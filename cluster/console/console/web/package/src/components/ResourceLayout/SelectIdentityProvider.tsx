import { ListIdentityProviderOptions } from "@/apis/corev1/corev1";
import { MainServiceClient } from "@/apis/corev1/corev1.client";
import { getClientCore } from "@/utils/client";
import {
  API,
  getClient,
  printResourceNameWithDisplay,
  ResourceList,
} from "@/utils/pb";
import { Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";

const SelectIdentityProvider = (props: {
  defaultValue?: string;
  description?: string;
  required?: boolean;
  label?: string;
  clearable?: boolean;

  onChange: (arg: string | null) => void;
}) => {
  const { isLoading, isSuccess, data } = useQuery({
    queryKey: ["selectIdentityProviderComponent"],
    queryFn: async () => {
      return await getClientCore().listIdentityProvider(
        ListIdentityProviderOptions.create({
          common: {
            itemsPerPage: 1000,
          },
        })
      );
    },
  });

  if (!data) {
    return <></>;
  }

  const itemList = data.response;

  const userList = itemList.items.map((x) => ({
    value: x.metadata!.name,
    label: printResourceNameWithDisplay(x),
  }));

  return (
    <div className="w-full">
      <Select
        label={props.label ? props.label : "Select IdentityProvider"}
        required={props.required}
        description={props.description}
        clearable={props.clearable}
        searchable
        data={userList}
        defaultValue={props.defaultValue}
        onChange={props.onChange}
      />
    </div>
  );
};

export default SelectIdentityProvider;
