import {
  AccessLog_Entry_Common_Status,
  ComponentLog_Entry_Level,
} from "@/apis/corev1/corev1";
import {
  GetAccessLogDataPointRequest,
  GetAccessLogSummaryRequest,
  GetAuditLogDataPointRequest,
  GetAuditLogSummaryRequest,
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogSummaryRequest,
  GetComponentLogDataPointRequest,
  GetComponentLogSummaryRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAccess,
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityComponentLog,
  getClientVisibilityCore,
  getClientVisibilityEnterprise,
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
  homeKeys,
  POINT_REFETCH,
  rangeOptions,
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

export const useAccessSummary = (
  periodMinutes: number,
  window: "current" | "previous",
  priority: number,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const from = window === "current" ? curFrom : prevFrom;
  const to = window === "current" ? curTo : prevTo;

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.accessSummary(window, periodMinutes, "all", NO_REFS),
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({ from: toTs(from), to: toTs(to) }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAccessDataPoint = (
  periodMinutes: number,
  status: "all" | "denied",
  priority: number,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.accessDataPoint(periodMinutes, status, NO_REFS),
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
            status:
              status === "denied"
                ? AccessLog_Entry_Common_Status.DENIED
                : undefined,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuthSummary = (
  periodMinutes: number,
  window: "current" | "previous",
  priority: number,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const from = window === "current" ? curFrom : prevFrom;
  const to = window === "current" ? curTo : prevTo;

  return useDashboardQuery({
    queryKey: [...visibilityKeys.authSummary(window, periodMinutes, NO_REFS)],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(from),
            to: toTs(to),
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuthDataPoint = (periodMinutes: number, priority: number) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery({
    queryKey: [...visibilityKeys.authDataPoint(periodMinutes, NO_REFS)],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
          GetAuthenticationLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
          }),
          { abort: signal },
        )
      ).response,
  });
};

export const useAuditSummary = (
  periodMinutes: number,
  window: "current" | "previous",
  priority: number,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const from = window === "current" ? curFrom : prevFrom;
  const to = window === "current" ? curTo : prevTo;

  return useDashboardQuery({
    queryKey: [
      ...visibilityKeys.auditSummary(window, periodMinutes, AUDIT_REFS),
    ],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({ from: toTs(from), to: toTs(to) }),
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
  window: "current" | "previous",
  priority: number,
) => {
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const from = window === "current" ? curFrom : prevFrom;
  const to = window === "current" ? curTo : prevTo;

  return useDashboardQuery({
    queryKey: [...visibilityKeys.componentSummary(window, periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(from),
            to: toTs(to),
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

type SummaryClient = "core" | "access" | "enterprise";

const summaryClient = (api: SummaryClient) =>
  api === "core"
    ? (getClientVisibilityCore() as any)
    : api === "access"
      ? (getClientVisibilityAccess() as any)
      : (getClientVisibilityEnterprise() as any);

export const useResourceSummary = <T>(args: {
  api: SummaryClient;
  kind: string;
  method: string;
  priority?: number;
  enabled?: boolean;
}) =>
  useDashboardQuery<T>({
    queryKey: summaryKey(args.api, args.kind),
    priority: args.priority ?? QUERY_PRIORITY.normal,
    enabled: args.enabled,
    fetch: async (signal) =>
      (await summaryClient(args.api)[args.method]({}, { abort: signal }))
        .response,
  });

export const useResourceRangeSummary = <T>(args: {
  api: SummaryClient;
  kind: string;
  method: string;
  periodMinutes: number;
  priority?: number;
  enabled?: boolean;
}) => {
  const { curFrom, curTo } = buildTimestamps(args.periodMinutes);

  return useDashboardQuery<T>({
    queryKey: rangeSummaryKey(args.api, args.kind, args.periodMinutes),
    priority: args.priority ?? QUERY_PRIORITY.low,
    enabled: args.enabled,
    fetch: async (signal) =>
      (
        await summaryClient(args.api)[args.method](
          rangeOptions(curFrom, curTo),
          {
            abort: signal,
          },
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

export const useAccessTop = (
  resource: keyof typeof accessTopMethod,
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery<{ items: { count: number }[] }>({
    queryKey: [...homeKeys.accessTop(resource, periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await (getClientVisibilityAccessLog() as any)[
          accessTopMethod[resource]
        ]({ from: toTs(curFrom), to: toTs(curTo) }, { abort: signal })
      ).response,
  });
};

export const useAuthTop = (
  resource: keyof typeof authTopMethod,
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery<{ items: { count: number }[] }>({
    queryKey: [...homeKeys.authTop(resource, periodMinutes)],
    priority,
    fetch: async (signal) =>
      (
        await (getClientVisibilityAuthenticationLog() as any)[
          authTopMethod[resource]
        ]({ from: toTs(curFrom), to: toTs(curTo) }, { abort: signal })
      ).response,
  });
};
