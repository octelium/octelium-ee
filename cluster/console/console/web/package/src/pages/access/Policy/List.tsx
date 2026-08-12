import { Policy } from "@/apis/accessv1/accessv1";
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
import { getDomain } from "@/utils";
import { getClientVisibilityAccess } from "@/utils/client";
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
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Policy }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
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
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRule}>Rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleDeny} to="/access/policies?effect=DENY">Deny rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleReview} to="/access/policies?effect=REVIEW">Review rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleAutoApprove} to="/access/policies?effect=AUTO_APPROVE">Auto-approve rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleAuthorization}>Authorization rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalRuleMaxAccessDuration}>Duration-limited rules</SummaryItemCount>
        <SummaryItemCount count={resp.totalReviewStep}>Review steps</SummaryItemCount>
        <SummaryItemCount count={resp.totalReviewer}>Reviewers</SummaryItemCount>
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
