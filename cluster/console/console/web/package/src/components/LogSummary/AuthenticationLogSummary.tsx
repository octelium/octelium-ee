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

              {!props.identityProviderRef && (
                <SummaryItemCount count={qry.data.totalIdentityProvider}>
                  IdentityProviders
                </SummaryItemCount>
              )}
              <SummaryItemCount count={qry.data.totalAuthenticator}>
                Authenticators
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalCredential}>
                Credentials
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorFIDO}>
                FIDO
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAuthenticatorTOTP}>
                TOTP
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAuthenticatorTPM}>
                TPM
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAAL1}>
                AAL1
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAAL1}>
                AAL2
              </SummaryItemCount>
              <SummaryItemCount count={qry.data.totalAAL1}>
                AAL3
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorPasskey}>
                Passkey
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalAuthenticatorMFA}>
                MFA
              </SummaryItemCount>

              <SummaryItemCount
                count={qry.data.totalNumber - qry.data.totalReauthentication}
              >
                Logins
              </SummaryItemCount>

              <SummaryItemCount count={qry.data.totalReauthentication}>
                Re-Authentications
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
            </SummaryItemCountWrap>
          </div>
        )}
      </div>
    </div>
  );
};

export default AuthenticationLogSummary;
