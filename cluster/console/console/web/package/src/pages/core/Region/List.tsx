import { Region } from "@/apis/corev1/corev1";

import { GetRegionSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { SummaryItemCount, SummaryItemCountWrap } from "@/components/Summary";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

export const LabelComponent = (props: { item: Region }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.status?.version && (
        <ResourceListLabel label="Version">
          {item.status.version}
        </ResourceListLabel>
      )}
      {item.status?.publicHostname && (
        <ResourceListLabel label="Hostname">
          {item.status.publicHostname}
        </ResourceListLabel>
      )}
      {!!item.status?.ingressAddresses.length && (
        <ResourceListLabel label="Ingress addresses">
          {item.status.ingressAddresses.length}
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Region }) => {
  const { item } = props;
  const domain = getDomain();
  return <></>;
};

const DoSummary = (props: { resp: GetRegionSummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/regions`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = () => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Region"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getRegionSummary({});

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return <DoSummary resp={qry.data} />;
};
