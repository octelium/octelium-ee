import { Catalog, ListCatalogOptions } from "@/apis/accessv1/accessv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

const countServices = (item: Catalog): number =>
  item.spec?.resourceCollection?.service?.services.length ?? 0;

const countNamespaces = (item: Catalog): number =>
  item.spec?.resourceCollection?.service?.namespaces.length ?? 0;

export const LabelComponent = (props: { item: Catalog }) => {
  const { item } = props;
  const svc = countServices(item);
  const ns = countNamespaces(item);

  return (
    <ResourceListLabelWrap>
      {svc > 0 && <ResourceListLabel>{svc} Services</ResourceListLabel>}
      {ns > 0 && <ResourceListLabel>{ns} Namespaces</ResourceListLabel>}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Catalog }) => {
  return <div></div>;
};

const DoSummary = (props: { totalNumber: number; totalServices: number }) => {
  const { totalNumber, totalServices } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={totalNumber} to={`/access/catalogs`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount count={totalServices}>Services</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Catalog"],
    queryFn: async () => {
      const { response } = await getClientAccess().listCatalog(
        ListCatalogOptions.create({}),
      );
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const items = qry.data.items;
  const totalNumber = items.length;
  const totalServices = items.reduce((acc, x) => acc + countServices(x), 0);

  return (
    <div>
      {totalNumber > 0 && (
        <div>
          <DoSummary totalNumber={totalNumber} totalServices={totalServices} />
        </div>
      )}
      {totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
