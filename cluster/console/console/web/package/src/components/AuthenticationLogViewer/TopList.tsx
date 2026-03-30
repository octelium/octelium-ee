import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuthenticationLogTopIdentityProviderRequest,
  ListAuthenticationLogTopUserRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAuthenticationLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import TopList from "../TopList";

const AuthenticationLogTopList = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  deviceRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
}) => {
  const qryUser = useQuery({
    queryKey: ["visibility", "listAuthenticationLogTopUser", { ...props }],

    queryFn: async () => {
      const req = ListAuthenticationLogTopUserRequest.create({
        identityProviderRef: props.identityProviderRef,
        credentialRef: props.credentialRef,
        authenticatorRef: props.authenticatorRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopUser(
          req
        );

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const qryIdentityProvider = useQuery({
    queryKey: [
      "visibility",
      "listAuthenticationLogTopIdentityProvider",
      { ...props },
    ],

    queryFn: async () => {
      const req = ListAuthenticationLogTopIdentityProviderRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        deviceRef: props.deviceRef,
        identityProviderRef: props.identityProviderRef,
        credentialRef: props.credentialRef,
        authenticatorRef: props.authenticatorRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopIdentityProvider(
          req
        );

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const qryCredential = useQuery({
    queryKey: [
      "visibility",
      "listAuthenticationLogTopCredential",
      { ...props },
    ],

    queryFn: async () => {
      const req = ListAuthenticationLogTopIdentityProviderRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        deviceRef: props.deviceRef,
        identityProviderRef: props.identityProviderRef,
        credentialRef: props.credentialRef,
        authenticatorRef: props.authenticatorRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopCredential(
          req
        );

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  return (
    <div className="w-full grid grid-cols-2 gap-4">
      {qryUser.data && qryUser.isSuccess && qryUser.data.items.length > 0 && (
        <TopList
          title="Top Users"
          items={qryUser.data.items.map((x) => ({
            resource: x.user!,
            count: x.count,
          }))}
        />
      )}
      {qryIdentityProvider.data &&
        qryIdentityProvider.isSuccess &&
        qryIdentityProvider.data.items.length > 0 && (
          <TopList
            title="Top IdentityProviders"
            items={qryIdentityProvider.data.items.map((x) => ({
              resource: x.identityProvider!,
              count: x.count,
            }))}
          />
        )}
      {qryCredential.data &&
        qryCredential.isSuccess &&
        qryCredential.data.items.length > 0 && (
          <TopList
            title="Top Policies"
            items={qryCredential.data.items.map((x) => ({
              resource: x.credential!,
              count: x.count,
            }))}
          />
        )}
    </div>
  );
};

export default AuthenticationLogTopList;
