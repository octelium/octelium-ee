import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import { ResourceMainInfo } from "@/pages/utils/types";

const Pills = (props: { values: string[] }) => {
  if (props.values.length === 0) {
    return <span className="text-[0.75rem] text-slate-400">None</span>;
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {props.values.map((v, idx) => (
        <span
          key={idx}
          className="px-2 py-0.5 rounded-md bg-slate-100 border border-slate-200 text-[0.72rem] font-mono font-semibold text-slate-700"
        >
          {v}
        </span>
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
        value: <Pills values={service?.services ?? []} />,
      },
      {
        label: "Namespaces",
        span: "full" as const,
        value: <Pills values={service?.namespaces ?? []} />,
      },
    ],
  };
};
