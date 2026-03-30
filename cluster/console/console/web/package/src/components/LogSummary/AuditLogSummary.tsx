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
import { SummaryItemCount, SummaryItemCountWrap } from "../Summary";

const AuditLogSummary = (props: {
  userRef?: ObjectReference;
  deviceRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
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

  /*
  React.useEffect(() => {
    qry.refetch();
  }, []);
  */

  return (
    <div>
      {/**
       <div className="flex items-center mb-6">
        <div className="font-bold text-gray-700 text-shadow-2xs text-xl">
          Summary
        </div>
        <Button
          size="compact-sm"
          variant="outline"
          className="ml-2 shadow-md"
          loading={qry.isLoading}
          onClick={() => {
            qry.refetch();
          }}
        >
          <MdRefresh />
        </Button>
      </div>
       **/}
      <div className="ml-4 mt-4">
        {qry.data && (
          <div className="w-full flex items-center">
            <SummaryItemCountWrap>
              <SummaryItemCount count={qry.data.totalNumber}>
                Total
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalResource}>
                Resources
              </SummaryItemCount>
              {!(props.userRef || props.deviceRef || props.sessionRef) && (
                <SummaryItemCount count={qry.data.totalUser}>
                  Users
                </SummaryItemCount>
              )}

              {!props.deviceRef && (
                <SummaryItemCount count={qry.data.totalDevice}>
                  Devices
                </SummaryItemCount>
              )}
              {!props.sessionRef && (
                <SummaryItemCount count={qry.data.totalSession}>
                  Sessions
                </SummaryItemCount>
              )}
            </SummaryItemCountWrap>
          </div>
        )}
      </div>
    </div>
  );
};

export default AuditLogSummary;
