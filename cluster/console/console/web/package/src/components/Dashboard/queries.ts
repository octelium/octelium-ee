import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import {
  GetAccessLogDataPointRequest,
  GetAccessLogDataPointRequest_GroupBy,
  GetAccessLogSummaryRequest,
  GetAuditLogDataPointRequest,
  GetAuditLogSummaryRequest,
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogDataPointRequest_GroupBy,
  GetAuthenticationLogSummaryRequest,
  GetClusterHealthRequest,
  GetClusterSummaryRequest,
  GetClusterSummaryRequest_Kind,
  GetComponentLogDataPointRequest,
  GetComponentLogSummaryRequest,
  ListAccessLogTopDenyReasonRequest,
  ListComponentLogTopComponentRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityCluster,
  getClientVisibilityComponentLog,
} from "@/utils/client";
import {
  buildTimestamps,
  getAutoInterval,
  NO_REFS,
  toTs,
  visibilityKeys,
} from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  dashboardKeys,
  POINT_REFETCH,
  rangeSummaryKey,
  summaryKey,
  useDashboardQuery,
} from "./utils";

const AUDIT_REFS = {
  userRef: null,
  sessionRef: null,
  deviceRef: null,
  resourceRef: null,
};

const TOP_LIMIT = 10;

export type ClusterScope = {
  name: string;
  kinds: GetClusterSummaryRequest_Kind[];
};

// useClusterSummary returns every requested resource summary of the Cluster in
// a single round trip. The "total" scope covers the whole Cluster while the
// "range" scope only counts the resources created within the selected window
// and carries the preceding window in its `previous` field.
export const useClusterSummary = (
  scope: "total" | "range",
  periodMinutes: number,
  priority: number,
  clusterScope?: ClusterScope,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const name = clusterScope?.name ?? "Cluster";

  return useDashboardQuery({
    queryKey:
      scope === "total"
        ? summaryKey("cluster", name)
        : rangeSummaryKey("cluster", name, periodMinutes),
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityCluster().getClusterSummary(
          GetClusterSummaryRequest.create({
            kinds: clusterScope?.kinds ?? [],
            common:
              scope === "total"
                ? undefined
                : {
                    from: toTs(curFrom),
                    to: toTs(curTo),
                    compareFrom: toTs(prevFrom),
                    compareTo: toTs(prevTo),
                  },
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useClusterHealth = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: ["visibility", "cluster", "health", periodMinutes],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityCluster().getClusterHealth(
          GetClusterHealthRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAccessSummary = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.accessSummary("current", periodMinutes, "all", NO_REFS),
      "compare",
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            compareFrom: toTs(prevFrom),
            compareTo: toTs(prevTo),
          }),
          { abort: signal },
        )
      ).response,
  });
};

// useAccessDataPoint returns the ungrouped access activity together with a
// per-status series, so the allowed/denied breakdown costs a single request.
export const useAccessDataPoint = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.accessDataPoint(periodMinutes, "byStatus", NO_REFS),
    ],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccessLog().getAccessLogDataPoint(
          GetAccessLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
            groupBy: GetAccessLogDataPointRequest_GroupBy.STATUS,
            limitSeries: 2,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAccessDenyReasons = (
  periodMinutes: number,
  priority: number,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [...dashboardKeys.accessTop("denyReason", periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccessLog().listAccessLogTopDenyReason(
          ListAccessLogTopDenyReasonRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            limit: TOP_LIMIT,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuthSummary = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.authSummary("current", periodMinutes, NO_REFS),
      "compare",
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            compareFrom: toTs(prevFrom),
            compareTo: toTs(prevTo),
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuthDataPoint = (
  periodMinutes: number,
  priority: number,
  groupBy?: {
    name: string;
    dimension: GetAuthenticationLogDataPointRequest_GroupBy;
    limitSeries: number;
  },
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.authDataPoint(periodMinutes, NO_REFS),
      groupBy?.name ?? null,
    ],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
          GetAuthenticationLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
            groupBy: groupBy?.dimension,
            limitSeries: groupBy?.limitSeries,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuditSummary = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.auditSummary("current", periodMinutes, AUDIT_REFS),
      "compare",
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            compareFrom: toTs(prevFrom),
            compareTo: toTs(prevTo),
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuditDataPoint = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [...visibilityKeys.auditDataPoint(periodMinutes, AUDIT_REFS)],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuditLog().getAuditLogDataPoint(
          GetAuditLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useComponentSummary = (
  periodMinutes: number,
  priority: number,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.componentSummary("current", periodMinutes),
      "compare",
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            compareFrom: toTs(prevFrom),
            compareTo: toTs(prevTo),
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useComponentDataPoint = (
  periodMinutes: number,
  level: "all" | "error",
  priority: number,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...(level === "error"
        ? visibilityKeys.componentErrorDataPoint(periodMinutes)
        : visibilityKeys.componentDataPoint(periodMinutes)),
    ],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) =>
      (
        await getClientVisibilityComponentLog().getComponentLogDataPoint(
          GetComponentLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
            level:
              level === "error" ? ComponentLog_Entry_Level.ERROR : undefined,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useTopComponents = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: ["componentLogTopComponent", periodMinutes],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityComponentLog().listComponentLogTopComponent(
          ListComponentLogTopComponentRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            limit: TOP_LIMIT,
          }),
          { abort: signal },
        )
      ).response,
  });
};

const accessTopMethod = {
  user: "listAccessLogTopUser",
  service: "listAccessLogTopService",
  policy: "listAccessLogTopPolicy",
  session: "listAccessLogTopSession",
} as const;

const authTopMethod = {
  user: "listAuthenticationLogTopUser",
  identityProvider: "listAuthenticationLogTopIdentityProvider",
  credential: "listAuthenticationLogTopCredential",
} as const;

type TopResponse = {
  items: { count: number }[];
  totalCount: number;
  totalOther: number;
};

export const useAccessTop = (
  resource: keyof typeof accessTopMethod,
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery<TopResponse>({
    queryKey: [...dashboardKeys.accessTop(resource, periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await (getClientVisibilityAccessLog() as any)[
          accessTopMethod[resource]
        ](
          { from: toTs(curFrom), to: toTs(curTo), limit: TOP_LIMIT },
          { abort: signal },
        )
      ).response,
  });
};

export const useAuthTop = (
  resource: keyof typeof authTopMethod,
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery<TopResponse>({
    queryKey: [...dashboardKeys.authTop(resource, periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await (getClientVisibilityAuthenticationLog() as any)[
          authTopMethod[resource]
        ](
          { from: toTs(curFrom), to: toTs(curTo), limit: TOP_LIMIT },
          { abort: signal },
        )
      ).response,
  });
};
