import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { useSecretUsage } from "./List";

const Usage = (props: { item: AccessC.Secret }) => {
  const usage = useSecretUsage(props.item.metadata?.name);

  if (usage.isError) {
    return (
      <span className="text-xs font-normal text-slate-500">
        The Integrations could not be loaded
      </span>
    );
  }
  if (usage.isLoading) {
    return <span className="text-xs font-normal text-slate-500">Loading…</span>;
  }
  if (usage.refs.length === 0) {
    return (
      <span className="text-xs font-normal text-slate-500">
        No Integration refers to this Secret
      </span>
    );
  }

  return (
    <div className="flex flex-wrap gap-1.5">
      {usage.refs.map((itemRef) => (
        <ResourceListLabel key={itemRef.uid} itemRef={itemRef} />
      ))}
    </div>
  );
};

export const ItemInfo = (props: { item: AccessC.Secret }) => {
  const usage = useSecretUsage(props.item.metadata?.name);
  return (
    <>
      <InfoItem title="Integrations">
        <span>{usage.refs.length}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Secret }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.Secret;
}): ResourceMainInfo => {
  const { item } = props;

  return {
    items: [
      {
        label: "Data",
        value: <Label>Write-only</Label>,
        hint: "The data of an access Secret is only ever read by the Cluster itself and it is never returned by the API.",
      },
      {
        label: "Used by",
        span: "full" as const,
        value: <Usage item={item} />,
      },
    ],
  };
};
