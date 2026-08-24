import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuthenticationLogSummaryRequest,
  GetAuthenticationLogSummaryResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import {
  getClientVisibilityAuthenticationLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  Cpu,
  Fingerprint,
  KeyRound,
  LogIn,
  Network,
  RefreshCw,
  Shield,
  ShieldCheck,
  Timer,
  Users,
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
      if (isDev()) {
        return GetAuthenticationLogSummaryResponse.create({
          totalNumber: 100,

          totalUser: 14,
          totalSession: 24,
          totalIdentityProvider: 4,
          totalAuthenticator: 45,
        });
      }

      const req = GetAuthenticationLogSummaryRequest.create({
        ...props,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          req
        );
      return response;
    },
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
              <SummaryItemCount
                count={qry.data.totalNumber}
                icon={LogIn}
                to={authenticationLogsPath()}
              >
                Total
              </SummaryItemCount>

              {!props.identityProviderRef && (
                <SummaryItemCount count={qry.data.totalIdentityProvider} icon={Network}>
                  IdentityProviders
                </SummaryItemCount>
              )}
              <SummaryItemCount count={qry.data.totalAuthenticator} icon={Fingerprint}>
                Authenticators
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalCredential} icon={KeyRound}>
                Credentials
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorFIDO} icon={Fingerprint}>
                FIDO
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAuthenticatorTOTP} icon={Timer}>
                TOTP
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAuthenticatorTPM} icon={Cpu}>
                TPM
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAAL1} icon={Shield}>
                AAL1
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAAL2} icon={Shield}>
                AAL2
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAAL3} icon={ShieldCheck}>
                AAL3
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorPasskey} icon={KeyRound}>
                Passkey
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorMFA} icon={ShieldCheck}>
                MFA
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalNumber - qry.data.totalReauthentication}
                icon={LogIn}
              >
                Logins
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalReauthentication} icon={RefreshCw}>
                Re-Authentications
              </SummaryItemCount>

              {!(props.userRef || props.deviceRef || props.sessionRef) && (
                <SummaryItemCount count={qry.data.totalUser} icon={Users}>
                  Users
                </SummaryItemCount>
              )}
              {!props.sessionRef && (
                <SummaryItemCount count={qry.data.totalSession} icon={Activity}>
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

export default AuthenticationLogSummary;
