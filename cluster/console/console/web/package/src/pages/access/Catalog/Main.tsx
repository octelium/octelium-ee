import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";

const LinkedResources = (props: {
  values: string[];
  kind: "Service" | "Namespace";
}) => {
  if (props.values.length === 0) {
    return (
      <span className="text-[0.72rem] font-semibold text-slate-400">
        No {props.kind.toLowerCase()}s configured
      </span>
    );
  }

  return (
    <div className="flex flex-wrap gap-1.5">
      {props.values.map((name) => (
        <ResourceListLabel
          key={`${props.kind}-${name}`}
          itemRef={ObjectReference.create({
            apiVersion: "core/v1",
            kind: props.kind,
            name,
          })}
        />
      ))}
    </div>
  );
};

export const ItemInfo = (props: { item: AccessC.Catalog }) => {
  const { item } = props;
  const service = item.spec?.resourceCollection?.service;
  return (
    <>
      <InfoItem title="Services">
        <span>{service?.services.length ?? 0}</span>
      </InfoItem>
      <InfoItem title="Namespaces">
        <span>{service?.namespaces.length ?? 0}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Catalog }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.Catalog;
}): ResourceMainInfo => {
  const { item } = props;
  const service = item.spec?.resourceCollection?.service;

  return {
    items: [
      {
        label: "Services",
        span: "full" as const,
        value: (
          <LinkedResources
            kind="Service"
            values={service?.services.filter(Boolean) ?? []}
          />
        ),
      },
      {
        label: "Namespaces",
        span: "full" as const,
        value: (
          <LinkedResources
            kind="Namespace"
            values={service?.namespaces.filter(Boolean) ?? []}
          />
        ),
      },
    ],
  };
};
