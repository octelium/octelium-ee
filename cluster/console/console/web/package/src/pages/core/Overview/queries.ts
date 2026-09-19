import {
  GetAuthenticationLogDataPointRequest_GroupBy,
  GetClusterSummaryRequest_Kind,
} from "@/apis/visibilityv1/visibilityv1";
import {
  ClusterScope,
  useClusterSummary,
} from "@/components/Dashboard/queries";

export const CORE_SCOPE: ClusterScope = {
  name: "Core",
  kinds: [
    GetClusterSummaryRequest_Kind.CORE_USER,
    GetClusterSummaryRequest_Kind.CORE_SESSION,
    GetClusterSummaryRequest_Kind.CORE_DEVICE,
    GetClusterSummaryRequest_Kind.CORE_SERVICE,
    GetClusterSummaryRequest_Kind.CORE_NAMESPACE,
    GetClusterSummaryRequest_Kind.CORE_POLICY,
    GetClusterSummaryRequest_Kind.CORE_GROUP,
    GetClusterSummaryRequest_Kind.CORE_CREDENTIAL,
    GetClusterSummaryRequest_Kind.CORE_IDENTITY_PROVIDER,
    GetClusterSummaryRequest_Kind.CORE_AUTHENTICATOR,
    GetClusterSummaryRequest_Kind.CORE_SECRET,
    GetClusterSummaryRequest_Kind.CORE_GATEWAY,
    GetClusterSummaryRequest_Kind.CORE_REGION,
  ],
};

export const useCoreTotals = (periodMinutes: number, priority: number) =>
  useClusterSummary("total", periodMinutes, priority, CORE_SCOPE);

export const useCoreCreated = (periodMinutes: number, priority: number) =>
  useClusterSummary("range", periodMinutes, priority, CORE_SCOPE);

export const AUTH_BY_ASSURANCE = {
  name: "assurance",
  dimension: GetAuthenticationLogDataPointRequest_GroupBy.ASSURANCE_LEVEL,
  limitSeries: 3,
};
