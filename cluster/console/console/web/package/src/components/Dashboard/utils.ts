import {
  DEFAULT_PERIOD_MINUTES,
  isKnownPeriod,
  toTs,
} from "@/utils/visibility";
import { QUERY_PRIORITY, queued } from "@/utils/visibility/queue";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { useSearchParams } from "react-router-dom";

export const SUMMARY_REFETCH = 60_000;
export const POINT_REFETCH = 30_000;
export const STALE_TIME = 20_000;

export const AutoRefreshContext = React.createContext(true);

export const useAutoRefresh = () => React.useContext(AutoRefreshContext);

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

  return useQuery({
    queryKey: args.queryKey,
    enabled: args.enabled ?? true,
    queryFn: ({ signal }) =>
      queued(args.priority ?? QUERY_PRIORITY.normal, args.fetch, signal),
    staleTime: STALE_TIME,
    retry: args.retry ?? 1,
    refetchInterval: autoRefresh
      ? (args.refetchMillis ?? SUMMARY_REFETCH)
      : false,
    refetchIntervalInBackground: false,
  });
};

export const useDashboardRange = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const rangeParam = Number(searchParams.get("range"));
  const periodMinutes = isKnownPeriod(rangeParam)
    ? rangeParam
    : DEFAULT_PERIOD_MINUTES;

  const setPeriodMinutes = (value: number) => {
    const next = new URLSearchParams(searchParams);
    if (value === DEFAULT_PERIOD_MINUTES) next.delete("range");
    else next.set("range", String(value));
    setSearchParams(next, { replace: true, preventScrollReset: true });
  };

  return { periodMinutes, setPeriodMinutes };
};

export const rangeOptions = (from: number, to: number) => ({
  common: { from: toTs(from), to: toTs(to) },
});

export const compact = (value: number) =>
  new Intl.NumberFormat(undefined, {
    notation: value >= 10_000 ? "compact" : "standard",
    maximumFractionDigits: 1,
  }).format(value);

export const exact = (value: number) => value.toLocaleString();

export const ratio = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 1000) / 10;
