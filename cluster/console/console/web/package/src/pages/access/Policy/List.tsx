import { Policy } from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { GetPolicySummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import StatusBadge from "@/components/StatusBadge";
import { getClientVisibilityAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  Ban,
  BellRing,
  ListOrdered,
  ShieldCheck,
  ShieldX,
  Users,
} from "lucide-react";
import { EFFECT_META, EffectBadge, EffectKey, getEffectKey } from "../common";

const integrationNames = (item: Policy): string[] => {
  const names = new Set<string>();
  (item.spec?.rules ?? []).forEach((rule) => {
    [
      ...rule.notifications,
      ...(rule.action?.type.oneofKind === "review"
        ? rule.action.type.review.steps.flatMap((step) => step.surfaces)
        : []),
    ].forEach((surface) => {
      const name = surface.destination?.integrationRef?.name;
      if (name) names.add(name);
    });
  });
  return [...names];
};

const effectCounts = (item: Policy) => {
  const counts = new Map<EffectKey, number>();
  (item.spec?.rules ?? []).forEach((rule) => {
    const key = getEffectKey(rule.effect);
    if (key) counts.set(key, (counts.get(key) ?? 0) + 1);
  });
  return (["AUTO_APPROVE", "REVIEW", "DENY"] as EffectKey[])
    .filter((key) => counts.has(key))
    .map((key) => ({ key, count: counts.get(key)! }));
};

export const LabelComponent = (props: { item: Policy }) => {
  const { item } = props;
  const integrations = integrationNames(item);
  const rules = item.spec?.rules.length ?? 0;

  return (
    <ResourceListLabelWrap>
      {item.spec?.isDisabled && (
        <StatusBadge label="Disabled" tone="danger" compact />
      )}
      <ResourceListLabel>
        {rules === 0 ? "No rule" : `${rules} ${rules === 1 ? "rule" : "rules"}`}
      </ResourceListLabel>
      {effectCounts(item).map(({ key, count }) => (
        <ResourceListLabel key={key}>
          <EffectBadge effect={EFFECT_META[key].value} />
          {count > 1 && <span className="ml-1 text-micro text-slate-500">×{count}</span>}
        </ResourceListLabel>
      ))}
      {integrations.map((name) => (
        <ResourceListLabel
          key={name}
          itemRef={ObjectReference.create({
            apiVersion: "access/v1",
            kind: "Integration",
            name,
          })}
        />
      ))}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetPolicySummaryResponse }) => {

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/policies">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDisabled}
          to={`/access/policies?isDisabled=true`}
          icon={Ban}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleAutoApprove} to="/access/policies?effect=AUTO_APPROVE" icon={ShieldCheck}>
          Auto-approve rules
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleReview} to="/access/policies?effect=REVIEW" icon={Users}>
          Review rules
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleDeny} to="/access/policies?effect=DENY" icon={ShieldX}>
          Deny rules
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalReviewStep} icon={ListOrdered}>
          Review steps
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalReviewer} icon={Users}>
          Reviewers
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalReviewSurface + resp.totalRuleNotification} icon={BellRing}>
          Deliveries
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Policy"],
    queryFn: async () => {
      const { response } = await getClientVisibilityAccess().getPolicySummary({});
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
