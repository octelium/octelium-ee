import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAccessLogTopServiceRequest,
  ListAccessLogTopSessionRequest,
  ListAccessLogTopUserRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAccessLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import TopList from "../TopList";

const AccessLogTopList = (props: {
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
  const qryUser = useQuery({
    queryKey: ["visibility", "listAccessLogTopUser", { ...props }],

    queryFn: async () => {
      const req = ListAccessLogTopUserRequest.create({
        serviceRef: props.serviceRef,
        namespaceRef: props.namespaceRef,
        regionRef: props.regionRef,
        policyRef: props.policyRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopUser(req);

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const qryService = useQuery({
    queryKey: ["visibility", "listAccessLogTopService", { ...props }],

    queryFn: async () => {
      const req = ListAccessLogTopServiceRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        regionRef: props.regionRef,
        deviceRef: props.deviceRef,
        policyRef: props.policyRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopService(req);

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const qryPolicy = useQuery({
    queryKey: ["visibility", "listAccessLogTopPolicy", { ...props }],

    queryFn: async () => {
      const req = ListAccessLogTopServiceRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        regionRef: props.regionRef,
        deviceRef: props.deviceRef,
        policyRef: props.policyRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopPolicy(req);

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const qrySession = useQuery({
    queryKey: ["visibility", "listAccessLogTopSession", { ...props }],

    queryFn: async () => {
      const req = ListAccessLogTopSessionRequest.create({
        userRef: props.userRef,
        regionRef: props.regionRef,
        deviceRef: props.deviceRef,
        policyRef: props.policyRef,
        from: props.from,
        to: props.to,
      });

      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopSession(req);

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  return (
    <div className="w-full grid grid-cols-2 gap-4">
      {qryUser.data && qryUser.isSuccess && qryUser.data.items.length > 0 && (
        <TopList
          title="Top Users"
          to={`/visibility/accesslogs`}
          items={qryUser.data.items.map((x) => ({
            resource: x.user!,
            count: x.count,
          }))}
        />
      )}
      {qryService.data &&
        qryService.isSuccess &&
        qryService.data.items.length > 0 && (
          <TopList
            title="Top Services"
            to={`/visibility/accesslogs`}
            items={qryService.data.items.map((x) => ({
              resource: x.service!,
              count: x.count,
            }))}
          />
        )}
      {qryPolicy.data &&
        qryPolicy.isSuccess &&
        qryPolicy.data.items.length > 0 && (
          <TopList
            title="Top Policies"
            to={`/visibility/accesslogs`}
            items={qryPolicy.data.items.map((x) => ({
              resource: x.policy!,
              count: x.count,
            }))}
          />
        )}

      {qrySession.data &&
        qrySession.isSuccess &&
        qrySession.data.items.length > 0 && (
          <TopList
            title="Top Sessions"
            to={`/visibility/accesslogs`}
            items={qrySession.data.items.map((x) => ({
              resource: x.session!,
              count: x.count,
            }))}
          />
        )}
    </div>
  );
};

export default AccessLogTopList;
