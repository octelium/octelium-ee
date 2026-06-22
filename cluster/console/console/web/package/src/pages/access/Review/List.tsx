import {
  ListReviewOptions,
  Review,
  Review_Spec_Decision,
} from "@/apis/accessv1/accessv1";
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
import { getDecisionMeta } from "./utils";

export const LabelComponent = (props: { item: Review }) => {
  const { item } = props;
  const meta = getDecisionMeta(item.spec?.decision);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Review }) => {
  return <div></div>;
};

const DoSummary = (props: {
  totalNumber: number;
  totalApproved: number;
  totalRejected: number;
}) => {
  const { totalNumber, totalApproved, totalRejected } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={totalNumber} to={`/access/reviews`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount count={totalApproved}>Approved</SummaryItemCount>
        <SummaryItemCount count={totalRejected}>Rejected</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Review"],
    queryFn: async () => {
      const { response } = await getClientAccess().listReview(
        ListReviewOptions.create({}),
      );
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const items = qry.data.items;
  const totalNumber = items.length;
  const totalApproved = items.filter(
    (x) => x.spec?.decision === Review_Spec_Decision.APPROVE,
  ).length;
  const totalRejected = items.filter(
    (x) => x.spec?.decision === Review_Spec_Decision.REJECT,
  ).length;

  return (
    <div>
      {totalNumber > 0 && (
        <div>
          <DoSummary
            totalNumber={totalNumber}
            totalApproved={totalApproved}
            totalRejected={totalRejected}
          />
        </div>
      )}
      {totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
