import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuthenticationLogSummaryRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAuthenticationLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  LogIn,
  RefreshCw,
  Shield,
  ShieldCheck,
} from "lucide-react";
import { SummaryItemCount, SummaryItemCountWrap } from "../Summary";

const AuthenticationLogSummary = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  deviceRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "getAuthenticationLogSummary", { ...props }],

    queryFn: async () => {
      const req = GetAuthenticationLogSummaryRequest.create({
        ...props,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          req
        );
      return response;
    },
    refetchOnWindowFocus: false,
    refetchInterval: refetchIntervalChart,
  });

  const authenticationLogsPath = () => {
    const params = new URLSearchParams();
    const refs: Array<[string, ObjectReference | undefined]> = [
      ["userRef", props.userRef],
      ["sessionRef", props.sessionRef],
      ["deviceRef", props.deviceRef],
      ["identityProviderRef", props.identityProviderRef],
      ["credentialRef", props.credentialRef],
      ["authenticatorRef", props.authenticatorRef],
    ];
    refs.forEach(([key, ref]) => {
      const value = ref?.name || ref?.uid;
      if (value) params.set(`${key}.${ref?.name ? "name" : "uid"}`, value);
    });
    const query = params.toString();
    return query
      ? `/visibility/authenticationlogs?${query}`
      : "/visibility/authenticationlogs";
  };

  return (
    <div className="min-h-[58px]">
      {qry.data ? (
        <SummaryItemCountWrap>
          <SummaryItemCount showZero count={qry.data.totalNumber} icon={LogIn} to={authenticationLogsPath()}>
            Total events
          </SummaryItemCount>
          <SummaryItemCount
            showZero
            count={Math.max(
              0,
              qry.data.totalNumber - qry.data.totalReauthentication,
            )}
            icon={LogIn}
          >
            Logins
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalReauthentication} icon={RefreshCw}>
            Re-authentications
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalAAL2} icon={Shield}>
            AAL2
          </SummaryItemCount>
          <SummaryItemCount showZero count={qry.data.totalAAL3} icon={ShieldCheck}>
            AAL3
          </SummaryItemCount>
        </SummaryItemCountWrap>
      ) : qry.isError ? (
        <p className="text-xs font-semibold text-red-600">Summary unavailable</p>
      ) : (
        <div className="h-[58px] motion-safe:animate-pulse rounded-lg bg-slate-100" />
      )}
    </div>
  );
};

export default AuthenticationLogSummary;
