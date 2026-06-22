import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";

export const ItemInfo = (props: { item: AccessC.Policy }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <>
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
            checked={!item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = !v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>

      <InfoItem title="Rules">
        <span>{item.spec!.rules.length}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Policy }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: { item: AccessC.Policy }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "Active",
        value: (
          <EditItemWrap
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  item.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },
      {
        label: "Rules",
        value: (
          <span className="text-[0.75rem] font-semibold text-slate-700">
            {item.spec!.rules.length}
          </span>
        ),
      },
    ],
  };
};
