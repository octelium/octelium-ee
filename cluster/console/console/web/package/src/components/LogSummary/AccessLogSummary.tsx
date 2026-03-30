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

  /*
  React.useEffect(() => {
    qry.refetch();
  }, []);
  */

  return (
    <div>
      <div className="ml-4 mt-4">
        {qry.data && (
          <div className="w-full flex items-center">
            <SummaryItemCountWrap>
              <SummaryItemCount count={qry.data.totalNumber}>
                Total
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAllowed}>
                Allowed
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalDenied}>
                Denied
              </SummaryItemCount>
              {!(props.userRef || props.deviceRef || props.sessionRef) && (
                <SummaryItemCount count={qry.data.totalUser}>
                  Users
                </SummaryItemCount>
              )}
              {!props.sessionRef && (
                <SummaryItemCount count={qry.data.totalSession}>
                  Sessions
                </SummaryItemCount>
              )}
              {!(props.deviceRef || props.sessionRef) && (
                <SummaryItemCount count={qry.data.totalDevice}>
                  Devices
                </SummaryItemCount>
              )}

              {!(props.policyRef || props.policyRef) && (
                <SummaryItemCount count={qry.data.totalMatchPolicy}>
                  Policies
                </SummaryItemCount>
              )}
              {!props.serviceRef && (
                <SummaryItemCount count={qry.data.totalService}>
                  Services
                </SummaryItemCount>
              )}
              {!(props.namespaceRef || props.serviceRef) && (
                <SummaryItemCount count={qry.data.totalNamespace}>
                  Namespaces
                </SummaryItemCount>
              )}
            </SummaryItemCountWrap>
          </div>
        )}
      </div>
    </div>
  );
};

export default AccessLogSummary;
