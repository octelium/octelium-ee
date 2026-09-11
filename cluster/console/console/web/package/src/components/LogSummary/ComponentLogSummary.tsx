import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  GetComponentLogSummaryRequest,
  GetComponentLogSummaryResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import {
  getClientVisibilityComponentLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { CircleAlert, OctagonAlert, Skull, Terminal, TriangleAlert } from "lucide-react";
import { SummaryItemCount, SummaryItemCountWrap } from "../Summary";

const ComponentLogSummary = (props: {
  level?: ComponentLog_Entry_Level;
  from?: Timestamp;
  to?: Timestamp;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "getComponentLogSummary", { ...props }],

    queryFn: async () => {
      if (isDev()) {
        return GetComponentLogSummaryResponse.create({
          totalNumber: 100,
          totalDebug: 70,
          totalInfo: 20,
          totalWarn: 10,
        });
      }

      const req = GetComponentLogSummaryRequest.create({ ...props });

      const { response } =
        await getClientVisibilityComponentLog().getComponentLogSummary(req);
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  return (
    <div className="min-h-[58px]">
      {qry.data ? (
        <SummaryItemCountWrap>
              <SummaryItemCount
                showZero
                count={qry.data.totalNumber}
                icon={Terminal}
                to={`/visibility/componentlogs`}
              >
                Total
              </SummaryItemCount>

              <SummaryItemCount
                showZero
                count={qry.data.totalWarn}
                icon={TriangleAlert}
                to={`/visibility/componentlogs?level=WARN`}
              >
                Warn
              </SummaryItemCount>

              <SummaryItemCount
                showZero
                count={qry.data.totalError}
                icon={CircleAlert}
                to={`/visibility/componentlogs?level=ERROR`}
              >
                Error
              </SummaryItemCount>

              <SummaryItemCount
                showZero
                count={qry.data.totalPanic}
                icon={OctagonAlert}
                to={`/visibility/componentlogs?level=PANIC`}
              >
                Panic
              </SummaryItemCount>
              <SummaryItemCount
                showZero
                count={qry.data.totalFatal}
                icon={Skull}
                to={`/visibility/componentlogs?level=FATAL`}
              >
                Fatal
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

export default ComponentLogSummary;
