import {
  ListTemplateOptions,
  ListWorkspaceOptions,
} from "@/apis/visibilityv1/cordium/vcordiumv1";
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
import { getClientVisibilityCordium } from "@/utils/client";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

export const QUEUE_ITEMS = 25;

export const CORDIUM_SCOPE: ClusterScope = {
  name: "Cordium",
  kinds: [
    GetClusterSummaryRequest_Kind.CORDIUM_WORKSPACE,
    GetClusterSummaryRequest_Kind.CORDIUM_TEMPLATE,
    GetClusterSummaryRequest_Kind.CORDIUM_SPACE,
    GetClusterSummaryRequest_Kind.CORDIUM_MEMBERSHIP,
    GetClusterSummaryRequest_Kind.CORDIUM_GIT_PROVIDER,
    GetClusterSummaryRequest_Kind.CORDIUM_SECRET,
    GetClusterSummaryRequest_Kind.CORDIUM_USER_SECRET,
    GetClusterSummaryRequest_Kind.CORDIUM_REGION,
  ],
};

export const useCordiumTotals = (periodMinutes: number, priority: number) =>
  useClusterSummary("total", periodMinutes, priority, CORDIUM_SCOPE);

export const useCordiumCreated = (periodMinutes: number, priority: number) =>
  useClusterSummary("range", periodMinutes, priority, CORDIUM_SCOPE);

export const useFailedWorkspaces = (priority: number = QUERY_PRIORITY.high) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("cordium", "failedWorkspaces")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityCordium().listWorkspace(
          ListWorkspaceOptions.create({
            common: {
              itemsPerPage: QUEUE_ITEMS,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.DESC,
              },
            },
            isFailed: true,
          }),
          { abort: signal },
        )
      ).response,
  });

export const useBuildingTemplates = (priority: number = QUERY_PRIORITY.low) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("cordium", "buildingTemplates")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityCordium().listTemplate(
          ListTemplateOptions.create({
            common: {
              itemsPerPage: QUEUE_ITEMS,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.DESC,
              },
            },
            isBuilding: true,
          }),
          { abort: signal },
        )
      ).response,
  });
