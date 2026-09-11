import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuditLogSummaryRequest,
  GetAuditLogSummaryResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import {
  getClientVisibilityAuditLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Boxes, ScrollText, Users } from "lucide-react";
import { SummaryItemCount, SummaryItemCountWrap } from "../Summary";

const AuditLogSummary = (props: {
  userRef?: ObjectReference;
  deviceRef?: ObjectReference;
  sessionRef?: ObjectReference;
  resourceRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "getAuditLogSummary", { ...props }],

    queryFn: async () => {
      if (isDev()) {
        return GetAuditLogSummaryResponse.create({
          totalNumber: 100,

          totalUser: 14,
          totalSession: 24,
          totalResource: 43,
          totalDevice: 5,
        });
      }

      const req = GetAuditLogSummaryRequest.create({
        ...props,
      });

      const { response } =
        await getClientVisibilityAuditLog().getAuditLogSummary(req);
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const auditLogsPath = () => {
    const params = new URLSearchParams();
    const refs: Array<[string, ObjectReference | undefined]> = [
      ["userRef", props.userRef],
      ["sessionRef", props.sessionRef],
      ["resourceRef", props.resourceRef],
      ["deviceRef", props.deviceRef],
    ];
    refs.forEach(([key, ref]) => {
      const value = ref?.name || ref?.uid;
      if (value) params.set(`${key}.${ref?.name ? "name" : "uid"}`, value);
    });
    const query = params.toString();
    return query ? `/visibility/auditlogs?${query}` : "/visibility/auditlogs";
  };

  return (
    <div className="min-h-[58px]">
      {qry.data ? (
        <SummaryItemCountWrap>
          <SummaryItemCount showZero count={qry.data.totalNumber} icon={ScrollText} to={auditLogsPath()}>
            Total changes
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalResource} icon={Boxes}>
            Resources affected
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalUser} icon={Users}>
            Actors
          </SummaryItemCount>
        </SummaryItemCountWrap>
      ) : qry.isError ? (
        <p className="text-xs font-semibold text-red-600">Summary unavailable</p>
      ) : (
        <div className="h-[58px] animate-pulse rounded-lg bg-slate-100" />
      )}
    </div>
  );
};

export default AuditLogSummary;
