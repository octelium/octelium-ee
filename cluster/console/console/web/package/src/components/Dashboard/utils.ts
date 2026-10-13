import { toTs } from "@/utils/visibility";
import { QUERY_PRIORITY, queued } from "@/utils/visibility/queue";
import { QueryFilters, useQuery } from "@tanstack/react-query";
import { retryQuery } from "@/utils/queryPolicy";
import { formatCount } from "@/utils/format";
import * as React from "react";

export { usePeriodRange as useDashboardRange } from "@/hooks/use-period-range";

export const SUMMARY_REFETCH = 60_000;
export const POINT_REFETCH = 30_000;
export const STALE_TIME = 20_000;

export const AutoRefreshContext = React.createContext(true);

export const DashboardVisibilityContext = React.createContext(true);

export const useAutoRefresh = () => React.useContext(AutoRefreshContext);

export const dashboardQueryFilters: QueryFilters = {
  predicate: (query) =>
    query.meta?.dashboard === true && query.getObserversCount() > 0,
};

export const summaryKey = (api: string, kind: string): unknown[] => [
  "visibility",
  api,
  "summary",
  kind,
];

export const rangeSummaryKey = (
  api: string,
  kind: string,
  periodMinutes: number,
): unknown[] => ["visibility", api, "summary", kind, "range", periodMinutes];

export const dashboardKeys = {
  accessTop: (resource: string, periodMinutes: number) =>
    ["accessLogTop", resource, periodMinutes, "all", null] as const,
  authTop: (resource: string, periodMinutes: number) =>
    ["authLogTop", resource, periodMinutes, null] as const,
  auditTop: (resource: string, periodMinutes: number) =>
    ["auditLogTop", resource, periodMinutes, null] as const,
  metricStat: (metric: string, periodMinutes: number, variant: string) =>
    ["visibility", "metricStat", metric, variant, periodMinutes] as const,
  queue: (api: string, name: string) =>
    ["visibility", api, "queue", name] as const,
};

export const useDashboardQuery = <T>(args: {
  queryKey: unknown[];
  fetch: (signal?: AbortSignal) => Promise<T>;
  priority?: number;
  enabled?: boolean;
  refetchMillis?: number;
  retry?:
    boolean | number | ((failureCount: number, error: unknown) => boolean);
}) => {
  const autoRefresh = useAutoRefresh();
  const visible = React.useContext(DashboardVisibilityContext);

  return useQuery({
    queryKey: args.queryKey,
    enabled: visible && (args.enabled ?? true),
    queryFn: ({ signal }) =>
      queued(args.priority ?? QUERY_PRIORITY.normal, args.fetch, signal),
    staleTime: STALE_TIME,
    retry:
      args.retry ??
      ((failureCount, error) =>
        failureCount < 1 && retryQuery(failureCount, error)),
    meta: { dashboard: true },
    refetchOnWindowFocus: false,
    refetchInterval:
      autoRefresh && visible ? (args.refetchMillis ?? SUMMARY_REFETCH) : false,
    refetchIntervalInBackground: false,
  });
};

export const rangeOptions = (from: number, to: number) => ({
  common: { from: toTs(from), to: toTs(to) },
});

export const compact = formatCount;

export const exact = (value: number) => formatCount(value, { compact: false });

export const ratio = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 1000) / 10;
