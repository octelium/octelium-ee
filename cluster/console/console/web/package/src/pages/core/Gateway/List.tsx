import { Gateway } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { GetGatewaySummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { SummaryItemCount, SummaryItemCountWrap } from "@/components/Summary";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

const ItemDetails = (props: { item: Gateway; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Gateway }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel itemRef={item.status!.regionRef} />
      {item.status?.hostname && (
        <ResourceListLabel label="Hostname">
          {item.status.hostname}
        </ResourceListLabel>
      )}
      {!!item.status?.publicIPs.length && (
        <ResourceListLabel label="Public IPs">
          {item.status.publicIPs.length}
        </ResourceListLabel>
      )}
      {(item.status?.wireguard?.port ?? 0) > 0 && (
        <ResourceListLabel label="WireGuard">
          :{item.status!.wireguard!.port}
        </ResourceListLabel>
      )}
      {(item.status?.quicv0?.port ?? 0) > 0 && (
        <ResourceListLabel label="QUICv0">
          :{item.status!.quicv0!.port}
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Gateway }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetGatewaySummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/gateways`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = () => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Gateway"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getGatewaySummary(
        {},
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return <DoSummary resp={qry.data} />;
};
