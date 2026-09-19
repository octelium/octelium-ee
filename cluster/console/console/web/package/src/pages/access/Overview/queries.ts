import { Request_Status_State_Status } from "@/apis/accessv1/accessv1";
import { ListRequestOptions } from "@/apis/visibilityv1/access/vaccessv1";
import {
  CommonListOptions_OrderBy_Mode,
  CommonListOptions_OrderBy_Type,
} from "@/apis/visibilityv1/meta/vmetav1";
import { GetClusterSummaryRequest_Kind } from "@/apis/visibilityv1/visibilityv1";
import {
  ClusterScope,
  useClusterSummary,
} from "@/components/Dashboard/queries";
import { dashboardKeys, useDashboardQuery } from "@/components/Dashboard/utils";
import { getClientVisibilityAccess } from "@/utils/client";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

export const QUEUE_ITEMS = 25;

export const ACCESS_SCOPE: ClusterScope = {
  name: "Access",
  kinds: [
    GetClusterSummaryRequest_Kind.ACCESS_REQUEST,
    GetClusterSummaryRequest_Kind.ACCESS_REVIEW,
    GetClusterSummaryRequest_Kind.ACCESS_POLICY,
    GetClusterSummaryRequest_Kind.ACCESS_CATALOG,
  ],
};

export const useAccessTotals = (periodMinutes: number, priority: number) =>
  useClusterSummary("total", periodMinutes, priority, ACCESS_SCOPE);

export const useAccessCreated = (periodMinutes: number, priority: number) =>
  useClusterSummary("range", periodMinutes, priority, ACCESS_SCOPE);

export const usePendingRequests = (priority: number = QUERY_PRIORITY.high) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "pendingRequests")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listRequest(
          ListRequestOptions.create({
            common: {
              itemsPerPage: QUEUE_ITEMS,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.ASC,
              },
            },
            state: Request_Status_State_Status.PENDING,
          }),
          { abort: signal },
        )
      ).response,
  });

export const useActiveGrants = (priority: number = QUERY_PRIORITY.low) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("access", "activeGrants")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityAccess().listRequest(
          ListRequestOptions.create({
            common: {
              itemsPerPage: QUEUE_ITEMS,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.DESC,
              },
            },
            isActive: true,
          }),
          { abort: signal },
        )
      ).response,
  });
