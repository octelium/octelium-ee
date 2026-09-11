import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAccessLogSummaryRequest,
  GetAccessLogSummaryResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import {
  getClientVisibilityAccessLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Activity, ShieldCheck, ShieldX } from "lucide-react";
import { SummaryItemCount, SummaryItemCountWrap } from "../Summary";

const AccessLogSummary = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "getAccessLogSummary", { ...props }],

    queryFn: async () => {
      if (isDev()) {
        return GetAccessLogSummaryResponse.create({
          totalNumber: 100,
          totalAllowed: 56,
          totalDenied: 44,
          totalUser: 14,
          totalSession: 24,
          totalService: 12,
          totalNamespace: 2,
        });
      }

      const req = GetAccessLogSummaryRequest.create({ ...props });

      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(req);
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const accessLogsPath = (status?: "ALLOWED" | "DENIED") => {
    const params = new URLSearchParams();
    if (status) params.set("status", status);

    const refs: Array<[string, ObjectReference | undefined]> = [
      ["userRef", props.userRef],
      ["sessionRef", props.sessionRef],
      ["serviceRef", props.serviceRef],
      ["namespaceRef", props.namespaceRef],
      ["regionRef", props.regionRef],
      ["deviceRef", props.deviceRef],
      ["policyRef", props.policyRef],
    ];
    refs.forEach(([key, ref]) => {
      const value = ref?.name || ref?.uid;
      if (value) params.set(`${key}.${ref?.name ? "name" : "uid"}`, value);
    });

    const query = params.toString();
    return query ? `/visibility/accesslogs?${query}` : "/visibility/accesslogs";
  };

  return (
    <div className="min-h-[58px]">
      {qry.data ? (
        <SummaryItemCountWrap>
          <SummaryItemCount showZero count={qry.data.totalNumber} icon={Activity} to={accessLogsPath()}>
            Total requests
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalAllowed} icon={ShieldCheck} to={accessLogsPath("ALLOWED")}>
            Allowed
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalDenied} icon={ShieldX} to={accessLogsPath("DENIED")}>
            Denied
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

export default AccessLogSummary;
