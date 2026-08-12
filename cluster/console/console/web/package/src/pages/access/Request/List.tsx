import {
  Request,
} from "@/apis/accessv1/accessv1";
import { GetRequestSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
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
import { getStatusMeta, getUrgencyLabel } from "./utils";

export const LabelComponent = (props: { item: Request }) => {
  const { item } = props;
  const meta = getStatusMeta(item.status?.state?.status);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      {item.spec!.urgency !== undefined && item.spec!.urgency !== 0 && (
        <ResourceListLabel>
          {getUrgencyLabel(item.spec!.urgency)}
        </ResourceListLabel>
      )}
      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
      {item.status?.policyRef && (
        <ResourceListLabel itemRef={item.status!.policyRef}></ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Request }) => {
  return <div></div>;
};

const DoSummary = ({ resp }: { resp: GetRequestSummaryResponse }) => {

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/requests">
          Total
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalPending} to="/access/requests?state=PENDING">Pending</SummaryItemCount>
        <SummaryItemCount count={resp.totalActive} to="/access/requests?isActive=true">Active</SummaryItemCount>
        <SummaryItemCount count={resp.totalApproved} to="/access/requests?state=APPROVED">Approved</SummaryItemCount>
        <SummaryItemCount count={resp.totalRejected} to="/access/requests?state=REJECTED">Rejected</SummaryItemCount>
        <SummaryItemCount count={resp.totalRevoked} to="/access/requests?state=REVOKED">Revoked</SummaryItemCount>
        <SummaryItemCount count={resp.totalExpired} to="/access/requests?state=EXPIRED">Expired</SummaryItemCount>
        <SummaryItemCount count={resp.totalCancelled} to="/access/requests?state=CANCELLED">Cancelled</SummaryItemCount>
        <SummaryItemCount count={resp.totalDeadlinePassed}>Past deadline</SummaryItemCount>
        <SummaryItemCount count={resp.totalWithDeadline}>With deadline</SummaryItemCount>
        <SummaryItemCount count={resp.totalUrgencyHigh + resp.totalUrgencyVeryHigh + resp.totalUrgencyHighest}>High urgency</SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Requesters</SummaryItemCount>
        <SummaryItemCount count={resp.totalSubjectUser}>Subject users</SummaryItemCount>
        <SummaryItemCount count={resp.totalService}>Services</SummaryItemCount>
        <SummaryItemCount count={resp.totalCatalog}>Catalogs</SummaryItemCount>
        <SummaryItemCount count={resp.totalPolicy}>Policies</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Request"],
    queryFn: async () => {
      const { response } = await getClientVisibilityAccess().getRequestSummary({});
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
