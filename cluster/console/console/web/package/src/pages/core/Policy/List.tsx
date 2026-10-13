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
import StatusBadge from "@/components/StatusBadge";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Ban, ListChecks, ShieldCheck, ShieldX } from "lucide-react";

export const LabelComponent = (props: { item: Policy }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.spec?.isDisabled && (
        <ResourceListLabel>
          <StatusBadge label="Disabled" tone="danger" compact />
        </ResourceListLabel>
      )}
      {!!item.spec?.rules?.length && (
        <ResourceListLabel>{item.spec.rules.length} Rules</ResourceListLabel>
      )}
      {!!item.spec?.enforcementRules?.length && (
        <ResourceListLabel>
          {item.spec.enforcementRules.length} Enforcement Rules
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
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
          icon={Ban}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRule} icon={ListChecks}>
          Rules
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleAllow} icon={ShieldCheck}>
          Allowed Rules
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleDenied} icon={ShieldX}>
          Denied Rules
        </SummaryItemCount>
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
