import {
  Review,
} from "@/apis/accessv1/accessv1";
import { GetReviewSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
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
import { getDecisionMeta } from "./utils";

export const LabelComponent = (props: { item: Review }) => {
  const { item } = props;
  const meta = getDecisionMeta(item.spec?.decision);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      <ResourceListLabel itemRef={item.status!.requestRef}></ResourceListLabel>
      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Review }) => {
  return <div></div>;
};

const DoSummary = ({ resp }: { resp: GetReviewSummaryResponse }) => {

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/reviews">
          Total
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalPending} to="/access/reviews?isDecided=false">Pending</SummaryItemCount>
        <SummaryItemCount count={resp.totalApproved} to="/access/reviews?decision=APPROVE">Approved</SummaryItemCount>
        <SummaryItemCount count={resp.totalRejected} to="/access/reviews?decision=REJECT">Rejected</SummaryItemCount>
        <SummaryItemCount count={resp.totalRevised}>Revised</SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Reviewers</SummaryItemCount>
        <SummaryItemCount count={resp.totalRequest}>Requests</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Review"],
    queryFn: async () => {
      const { response } = await getClientVisibilityAccess().getReviewSummary({});
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
