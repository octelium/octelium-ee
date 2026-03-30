import { Policy } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { GetPolicySummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

const ItemDetails = (props: { item: Policy; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Policy }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.spec!.isDisabled && (
        <ResourceListLabel>
          <span className="text-red-400">Disabled</span>
        </ResourceListLabel>
      )}
      {item.spec!.rules && item.spec!.rules.length > 0 && (
        <ResourceListLabel>{item.spec!.rules.length} Rules</ResourceListLabel>
      )}
      {item.spec!.rules && item.spec!.enforcementRules.length > 0 && (
        <ResourceListLabel>
          {item.spec!.enforcementRules.length} Enforcement Rules
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Policy }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetPolicySummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/policies`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDisabled}
          to={`/core/policies?isDisabled=true`}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRule}>Rules</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Policy"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getPolicySummary({});

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }
  const d = qry.data;
  return (
    <div>
      {d.totalNumber > 0 && (
        <div>
          <DoSummary resp={qry.data} />
        </div>
      )}
      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
