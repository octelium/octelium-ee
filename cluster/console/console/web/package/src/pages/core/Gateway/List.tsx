import { ResourceListLabel } from "@/components/ResourceList";
import { Gateway } from "@/apis/corev1/corev1";

import { getDomain, toNumOrZero } from "@/utils";
import Label from "@/components/Label";
import { match } from "ts-pattern";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { SummaryItemCount, SummaryItemCountWrap } from "@/components/Summary";
import { GetGatewaySummaryResponse } from "@/apis/visibilityv1/core/vcorev1";

const ItemDetails = (props: { item: Gateway; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Gateway }) => {
  const { item } = props;

  return (
    <div className="w-full mt-1 flex flex-row">
      <ResourceListLabel itemRef={item.status!.regionRef}></ResourceListLabel>
    </div>
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
        {}
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return <DoSummary resp={qry.data} />;
};
