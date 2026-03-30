import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { useUpdateResource } from "@/pages/utils/resource";
import { getResourceRef } from "@/utils/pb";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

export const AccessLog = (props: { item: CoreC.User }) => {
  return <AccessLogViewer userRef={getResourceRef(props.item)} />;
};

export const ResourceItemInfo = (props: { item: CoreC.User }) => {
  let { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">
        <Label>
          {match(item.spec!.type)
            .with(CoreC.User_Spec_Type.HUMAN, () => "Human")
            .with(CoreC.User_Spec_Type.WORKLOAD, () => "Workload")
            .otherwise(() => "")}
        </Label>
      </InfoItem>

      {item.spec?.email && <InfoItem title="Email">{item.spec.email}</InfoItem>}

      {item.spec!.groups.length > 0 && (
        <InfoItem title="Groups">
          <div className="flex items-center">
            {item.spec!.groups.map((x) => (
              <Label key={x}>{x}</Label>
            ))}
          </div>
        </InfoItem>
      )}

      {item.spec!.authorization &&
        item.spec!.authorization.policies.length > 0 && (
          <InfoItem title="Policies">
            <div className="flex items-center">
              {item.spec!.authorization.policies.map((x) => (
                <Label key={x}>{x}</Label>
              ))}
            </div>
          </InfoItem>
        )}

      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined
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

export default (props: { item: CoreC.User }) => {
  let { item } = props;

  return (
    <div>
      <div className="w-full mb-8">
        <ResourceItemInfo item={item} />
      </div>
    </div>
  );
};
