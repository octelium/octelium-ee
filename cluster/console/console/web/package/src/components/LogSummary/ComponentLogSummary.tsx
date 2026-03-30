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
    <div>
      <div className="ml-4 mt-4">
        {qry.data && (
          <div className="w-full flex items-center">
            <SummaryItemCountWrap>
              <SummaryItemCount
                count={qry.data.totalNumber}
                to={`/visibility/componentlogs`}
              >
                Total
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalDebug}
                to={`/visibility/componentlogs?level=DEBUG`}
              >
                Debug
              </SummaryItemCount>
              <SummaryItemCount
                count={qry.data.totalInfo}
                to={`/visibility/componentlogs?level=INFO`}
              >
                Info
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalWarn}
                to={`/visibility/componentlogs?level=WARN`}
              >
                Warn
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalError}
                to={`/visibility/componentlogs?level=ERROR`}
              >
                Error
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalPanic}
                to={`/visibility/componentlogs?level=PANIC`}
              >
                Panic
              </SummaryItemCount>
              <SummaryItemCount
                count={qry.data.totalFatal}
                to={`/visibility/componentlogs?level=FATAL`}
              >
                Fatal
              </SummaryItemCount>
            </SummaryItemCountWrap>
          </div>
        )}
      </div>
    </div>
  );
};

export default ComponentLogSummary;
