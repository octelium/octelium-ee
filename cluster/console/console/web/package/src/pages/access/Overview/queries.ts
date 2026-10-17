import { Request_Status_State_Status } from "@/apis/accessv1/accessv1";
import {
  GetRequestDataPointRequest,
  GetRequestDataPointRequest_GroupBy,
  ListIntegrationBindingOptions,
  ListIntegrationOptions,
  ListRequestOptions,
  ListRequestOptions_OrderBy_Type,
  ListRequestTopRequest,
  ListRequestTopRequest_Dimension,
  ListReviewTopRequest,
  ListReviewTopRequest_Dimension,
} from "@/apis/visibilityv1/access/vaccessv1";
import {
  CommonListOptions_OrderBy_Mode,
  CommonListOptions_OrderBy_Type,
} from "@/apis/visibilityv1/meta/vmetav1";
import { GetClusterSummaryRequest_Kind } from "@/apis/visibilityv1/visibilityv1";
import {
  ClusterScope,
  useClusterSummary,
} from "@/components/Dashboard/queries";
import {
  dashboardKeys,
  POINT_REFETCH,
  useDashboardQuery,
} from "@/components/Dashboard/utils";
import { getClientVisibilityAccess } from "@/utils/client";
import { buildTimestamps, getAutoInterval, toTs } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

export const LIST_ITEMS = 6;

export const ACCESS_SCOPE: ClusterScope = {
  name: "Access",
  kinds: [
    GetClusterSummaryRequest_Kind.ACCESS_REQUEST,
    GetClusterSummaryRequest_Kind.ACCESS_REVIEW,
    GetClusterSummaryRequest_Kind.ACCESS_POLICY,
    GetClusterSummaryRequest_Kind.ACCESS_CATALOG,
    GetClusterSummaryRequest_Kind.ACCESS_INTEGRATION,
    GetClusterSummaryRequest_Kind.ACCESS_INTEGRATION_BINDING,
    GetClusterSummaryRequest_Kind.ACCESS_SECRET,
  ],
};

export const useAccessTotals = (periodMinutes: number, priority: number) =>
  useClusterSummary("total", periodMinutes, priority, ACCESS_SCOPE);

export const useAccessCreated = (periodMinutes: number, priority: number) =>
  useClusterSummary("range", periodMinutes, priority, ACCESS_SCOPE);

export const useRequestDataPoint = (
  periodMinutes: number,
  groupBy: GetRequestDataPointRequest_GroupBy,
  priority: number = QUERY_PRIORITY.high,
) =>
  useDashboardQuery({
    queryKey: [
      ...dashboardKeys.queue("access", "requestDataPoint"),
      periodMinutes,
      groupBy,
    ],
    priority,
    refetchMillis: POINT_REFETCH,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAccess().getRequestDataPoint(
          GetRequestDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: getAutoInterval(periodMinutes),
            groupBy,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useRequestTop = (
  dimension: ListRequestTopRequest_Dimension,
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) =>
  useDashboardQuery({
    queryKey: [
      ...dashboardKeys.queue("access", "requestTop"),
      dimension,
      periodMinutes,
    ],
    priority,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAccess().listRequestTop(
          ListRequestTopRequest.create({
            dimension,
            from: toTs(curFrom),
            to: toTs(curTo),
            limit: 5,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useReviewerTop = (
  periodMinutes: number,
  priority: number = QUERY_PRIORITY.low,
) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "reviewTop"), periodMinutes],
    priority,
    fetch: async (signal) => {
      const { curFrom, curTo } = buildTimestamps(periodMinutes);
      return (
        await getClientVisibilityAccess().listReviewTop(
          ListReviewTopRequest.create({
            dimension: ListReviewTopRequest_Dimension.REVIEWER,
            from: toTs(curFrom),
            to: toTs(curTo),
            limit: 5,
          }),
          { abort: signal },
        )
      ).response;
    },
  });

export const useOldestPending = (priority: number = QUERY_PRIORITY.critical) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "oldestPending")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listRequest(
          ListRequestOptions.create({
            common: { itemsPerPage: LIST_ITEMS },
            state: Request_Status_State_Status.PENDING,
            orderBy: {
              type: ListRequestOptions_OrderBy_Type.APPROVAL_START_AT,
              mode: CommonListOptions_OrderBy_Mode.ASC,
            },
          }),
          { abort: signal },
        )
      ).response,
  });

export const useLiveAccess = (priority: number = QUERY_PRIORITY.high) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "liveAccess")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listRequest(
          ListRequestOptions.create({
            common: { itemsPerPage: LIST_ITEMS },
            isActive: true,
            orderBy: {
              type: ListRequestOptions_OrderBy_Type.ACCESS_ENDS_AT,
              mode: CommonListOptions_OrderBy_Mode.ASC,
            },
          }),
          { abort: signal },
        )
      ).response,
  });

export const useIntegrationList = (priority: number = QUERY_PRIORITY.normal) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "integrations")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listIntegration(
          ListIntegrationOptions.create({
            common: {
              itemsPerPage: 100,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.NAME,
                mode: CommonListOptions_OrderBy_Mode.ASC,
              },
            },
          }),
          { abort: signal },
        )
      ).response,
  });

export const useFailingDeliveries = (priority: number = QUERY_PRIORITY.low) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "failingDeliveries")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listIntegrationBinding(
          ListIntegrationBindingOptions.create({
            common: {
              itemsPerPage: 5,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.DESC,
              },
            },
            isFailing: true,
          }),
          { abort: signal },
        )
      ).response,
  });
