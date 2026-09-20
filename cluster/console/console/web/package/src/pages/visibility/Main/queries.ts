import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import {
  DataPointSeries,
  GetAccessLogDataPointRequest,
  GetAccessLogDataPointRequest_GroupBy,
  GetAuditLogDataPointRequest,
  GetAuditLogDataPointRequest_GroupBy,
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogDataPointRequest_GroupBy,
  GetComponentLogDataPointRequest,
  GetComponentLogDataPointRequest_GroupBy,
  ListComponentLogTopComponentRequest,
} from "@/apis/visibilityv1/visibilityv1";
import { POINT_REFETCH, useDashboardQuery } from "@/components/Dashboard/utils";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityComponentLog,
} from "@/utils/client";
import { buildTimestamps, getAutoInterval, toTs } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity,
  Library,
  LucideIcon,
  ScrollText,
  ShieldUser,
} from "lucide-react";

export const LEADERBOARD_LIMIT = 10;
export const COMPONENT_LIMIT = 12;
export const BREAKDOWN_SERIES = 8;

export type StreamKey = "access" | "auth" | "audit" | "component";

export type StreamMeta = {
  key: StreamKey;
  label: string;
  icon: LucideIcon;
  to: string;
};

export const STREAMS: StreamMeta[] = [
  {
    key: "access",
    label: "Access",
    icon: Activity,
    to: "/visibility/accesslogs",
  },
  {
    key: "auth",
    label: "Authentication",
    icon: ShieldUser,
    to: "/visibility/authenticationlogs",
  },
  { key: "audit", label: "Audit", icon: Library, to: "/visibility/auditlogs" },
  {
    key: "component",
    label: "Components",
    icon: ScrollText,
    to: "/visibility/componentlogs",
  },
];

export type Dimension = { value: string; label: string; groupBy: number };

export const DIMENSIONS: Record<StreamKey, Dimension[]> = {
  access: [
    {
      value: "service",
      label: "Service",
      groupBy: GetAccessLogDataPointRequest_GroupBy.SERVICE,
    },
    {
      value: "user",
      label: "User",
      groupBy: GetAccessLogDataPointRequest_GroupBy.USER,
    },
    {
      value: "namespace",
      label: "Namespace",
      groupBy: GetAccessLogDataPointRequest_GroupBy.NAMESPACE,
    },
    {
      value: "policy",
      label: "Policy",
      groupBy: GetAccessLogDataPointRequest_GroupBy.POLICY,
    },
    {
      value: "mode",
      label: "Service mode",
      groupBy: GetAccessLogDataPointRequest_GroupBy.MODE,
    },
    {
      value: "status",
      label: "Decision",
      groupBy: GetAccessLogDataPointRequest_GroupBy.STATUS,
    },
    {
      value: "reason",
      label: "Deny reason",
      groupBy: GetAccessLogDataPointRequest_GroupBy.REASON,
    },
    {
      value: "region",
      label: "Region",
      groupBy: GetAccessLogDataPointRequest_GroupBy.REGION,
    },
    {
      value: "device",
      label: "Device",
      groupBy: GetAccessLogDataPointRequest_GroupBy.DEVICE,
    },
    {
      value: "session",
      label: "Session",
      groupBy: GetAccessLogDataPointRequest_GroupBy.SESSION,
    },
  ],
  auth: [
    {
      value: "type",
      label: "Method",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.TYPE,
    },
    {
      value: "aal",
      label: "Assurance level",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.ASSURANCE_LEVEL,
    },
    {
      value: "user",
      label: "User",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.USER,
    },
    {
      value: "identityProvider",
      label: "Identity provider",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.IDENTITY_PROVIDER,
    },
    {
      value: "credential",
      label: "Credential",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.CREDENTIAL,
    },
    {
      value: "authenticator",
      label: "Authenticator",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.AUTHENTICATOR,
    },
    {
      value: "device",
      label: "Device",
      groupBy: GetAuthenticationLogDataPointRequest_GroupBy.DEVICE,
    },
  ],
  audit: [
    {
      value: "action",
      label: "Action",
      groupBy: GetAuditLogDataPointRequest_GroupBy.ACTION,
    },
    {
      value: "resourceKind",
      label: "Resource kind",
      groupBy: GetAuditLogDataPointRequest_GroupBy.RESOURCE_KIND,
    },
    {
      value: "user",
      label: "User",
      groupBy: GetAuditLogDataPointRequest_GroupBy.USER,
    },
    {
      value: "session",
      label: "Session",
      groupBy: GetAuditLogDataPointRequest_GroupBy.SESSION,
    },
  ],
  component: [
    {
      value: "level",
      label: "Level",
      groupBy: GetComponentLogDataPointRequest_GroupBy.LEVEL,
    },
    {
      value: "type",
      label: "Component",
      groupBy: GetComponentLogDataPointRequest_GroupBy.COMPONENT_TYPE,
    },
    {
      value: "namespace",
      label: "Namespace",
      groupBy: GetComponentLogDataPointRequest_GroupBy.COMPONENT_NAMESPACE,
    },
  ],
};

type BreakdownResponse = {
  datapoints: { timestamp?: unknown; count: number }[];
  series: DataPointSeries[];
};

export const useBreakdown = (
  stream: StreamKey,
  dimension: Dimension,
  periodMinutes: number,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);

  return useDashboardQuery<BreakdownResponse>({
    queryKey: [
      "visibility",
      "breakdown",
      stream,
      dimension.value,
      periodMinutes,
    ],
    priority: QUERY_PRIORITY.normal,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) => {
      const common = {
        from: toTs(curFrom),
        to: toTs(curTo),
        interval,
        groupBy: dimension.groupBy,
        limitSeries: BREAKDOWN_SERIES,
      };

      switch (stream) {
        case "auth":
          return (
            await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
              GetAuthenticationLogDataPointRequest.create(common as any),
              { abort: signal },
            )
          ).response;
        case "audit":
          return (
            await getClientVisibilityAuditLog().getAuditLogDataPoint(
              GetAuditLogDataPointRequest.create(common as any),
              { abort: signal },
            )
          ).response;
        case "component":
          return (
            await getClientVisibilityComponentLog().getComponentLogDataPoint(
              GetComponentLogDataPointRequest.create(common as any),
              { abort: signal },
            )
          ).response;
        default:
          return (
            await getClientVisibilityAccessLog().getAccessLogDataPoint(
              GetAccessLogDataPointRequest.create(common as any),
              { abort: signal },
            )
          ).response;
      }
    },
  });
};

export const useComponentLeaderboard = (
  periodMinutes: number,
  level: ComponentLog_Entry_Level | undefined,
) => {
  const { curFrom, curTo } = buildTimestamps(periodMinutes);

  return useDashboardQuery({
    queryKey: [
      "componentLogTopComponent",
      periodMinutes,
      level ?? null,
      COMPONENT_LIMIT,
    ],
    priority: QUERY_PRIORITY.normal,
    fetch: async (signal) =>
      (
        await getClientVisibilityComponentLog().listComponentLogTopComponent(
          ListComponentLogTopComponentRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            level,
            limit: COMPONENT_LIMIT,
          }),
          { abort: signal },
        )
      ).response,
  });
};
