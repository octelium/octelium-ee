import * as CoreC from "@/apis/corev1/corev1";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { useUpdateResource } from "@/pages/utils/resource";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";
import { getType } from "./List";

export const ItemInfo = (props: { item: CoreC.IdentityProvider }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <>
      <InfoItem title="Type">
        <Label>{getType(item)}</Label>
      </InfoItem>
      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>
    </>
  );
};

export default (props: { item: CoreC.IdentityProvider }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};
