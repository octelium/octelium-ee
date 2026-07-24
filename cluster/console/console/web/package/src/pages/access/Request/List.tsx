import {
  ListRequestOptions,
  Request,
  Request_Status_State_Status,
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
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Request }) => {
  return <div></div>;
};

const DoSummary = (props: {
  totalNumber: number;
  totalPending: number;
  totalApproved: number;
}) => {
  const { totalNumber, totalPending, totalApproved } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={totalNumber} to={`/access/requests`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount count={totalPending}>Pending</SummaryItemCount>
        <SummaryItemCount count={totalApproved}>Approved</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Request"],
    queryFn: async () => {
      const { response } = await getClientAccess().listRequest(
        ListRequestOptions.create({}),
      );
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const items = qry.data.items;
  const totalNumber = items.length;
  const totalPending = items.filter(
    (x) => x.status?.state?.status === Request_Status_State_Status.PENDING,
  ).length;
  const totalApproved = items.filter(
    (x) => x.status?.state?.status === Request_Status_State_Status.APPROVED,
  ).length;

  return (
    <div>
      {totalNumber > 0 && (
        <div>
          <DoSummary
            totalNumber={totalNumber}
            totalPending={totalPending}
            totalApproved={totalApproved}
          />
        </div>
      )}
      {totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
