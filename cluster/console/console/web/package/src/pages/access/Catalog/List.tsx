import { Catalog } from "@/apis/accessv1/accessv1";
import { GetCatalogSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientVisibilityAccess } from "@/utils/client";
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

const DoSummary = ({ resp }: { resp: GetCatalogSummaryResponse }) => {

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/catalogs">
          Total
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalService}>Services</SummaryItemCount>
        <SummaryItemCount count={resp.totalNamespace}>Namespaces</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Catalog"],
    queryFn: async () => {
      const { response } = await getClientVisibilityAccess().getCatalogSummary({});
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return (
    <div>
      {qry.data.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {qry.data.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
