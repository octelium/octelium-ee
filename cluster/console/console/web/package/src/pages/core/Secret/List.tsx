import { ResourceListLabel } from "@/components/ResourceList";
import { Secret } from "@/apis/corev1/corev1";

import { getDomain, toNumOrZero } from "@/utils";
import Label from "@/components/Label";
import { match } from "ts-pattern";
import { useQuery } from "@tanstack/react-query";
import { getClientVisibilityCore } from "@/utils/client";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { GetSecretSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";

const ItemDetails = (props: { item: Secret; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Secret }) => {
  const { item } = props;

  return <div className="w-full mt-1 flex flex-row"></div>;
};

export const ExtraComponent = (props: { item: Secret }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetSecretSummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/secrets`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Secret"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getSecretSummary({});

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }
  const d = qry.data;
  return (
    <div>
      {d.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
