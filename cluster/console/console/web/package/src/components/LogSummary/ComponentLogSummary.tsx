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
import {
  Bug,
  CircleAlert,
  Info,
  OctagonAlert,
  Skull,
  Terminal,
  TriangleAlert,
} from "lucide-react";
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
                icon={Terminal}
                to={`/visibility/componentlogs`}
              >
                Total
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalDebug}
                icon={Bug}
                to={`/visibility/componentlogs?level=DEBUG`}
              >
                Debug
              </SummaryItemCount>
              <SummaryItemCount
                count={qry.data.totalInfo}
                icon={Info}
                to={`/visibility/componentlogs?level=INFO`}
              >
                Info
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalWarn}
                icon={TriangleAlert}
                to={`/visibility/componentlogs?level=WARN`}
              >
                Warn
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalError}
                icon={CircleAlert}
                to={`/visibility/componentlogs?level=ERROR`}
              >
                Error
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalPanic}
                icon={OctagonAlert}
                to={`/visibility/componentlogs?level=PANIC`}
              >
                Panic
              </SummaryItemCount>
              <SummaryItemCount
                count={qry.data.totalFatal}
                icon={Skull}
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
