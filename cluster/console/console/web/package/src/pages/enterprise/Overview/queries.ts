import { ListCertificateOptions } from "@/apis/visibilityv1/enterprise/venterprisev1";
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
import { getClientVisibilityEnterprise } from "@/utils/client";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

const QUEUE_ITEMS = 25;

export const ENTERPRISE_SCOPE: ClusterScope = {
  name: "Enterprise",
  kinds: [
    GetClusterSummaryRequest_Kind.ENTERPRISE_CERTIFICATE,
    GetClusterSummaryRequest_Kind.ENTERPRISE_CERTIFICATE_ISSUER,
    GetClusterSummaryRequest_Kind.ENTERPRISE_DIRECTORY_PROVIDER,
    GetClusterSummaryRequest_Kind.ENTERPRISE_DIRECTORY_PROVIDER_USER,
    GetClusterSummaryRequest_Kind.ENTERPRISE_DIRECTORY_PROVIDER_GROUP,
    GetClusterSummaryRequest_Kind.ENTERPRISE_SECRET_STORE,
    GetClusterSummaryRequest_Kind.ENTERPRISE_SECRET,
    GetClusterSummaryRequest_Kind.ENTERPRISE_DEVICE_MANAGER,
    GetClusterSummaryRequest_Kind.ENTERPRISE_COLLECTOR_EXPORTER,
    GetClusterSummaryRequest_Kind.ENTERPRISE_DNSPROVIDER,
  ],
};

export const useEnterpriseTotals = (periodMinutes: number, priority: number) =>
  useClusterSummary("total", periodMinutes, priority, ENTERPRISE_SCOPE);

export const useEnterpriseCreated = (periodMinutes: number, priority: number) =>
  useClusterSummary("range", periodMinutes, priority, ENTERPRISE_SCOPE);

export const useExpiringCertificates = (
  priority: number = QUERY_PRIORITY.normal,
) =>
  useDashboardQuery({
    queryKey: [...dashboardKeys.queue("enterprise", "expiringCertificates")],
    priority,
    fetch: async (signal) =>
      (
        await getClientVisibilityEnterprise().listCertificate(
          ListCertificateOptions.create({
            common: {
              itemsPerPage: QUEUE_ITEMS,
              orderBy: {
                type: CommonListOptions_OrderBy_Type.CREATED_AT,
                mode: CommonListOptions_OrderBy_Mode.DESC,
              },
            },
            isExpiringSoon: true,
          }),
          { abort: signal },
        )
      ).response,
  });
