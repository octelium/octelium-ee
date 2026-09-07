import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAccessLogDataPointRequest,
  GetAccessLogSummaryRequest,
  ListAccessLogTopServiceRequest,
  ListAccessLogTopSessionRequest,
  ListAccessLogTopPolicyRequest,
  ListAccessLogTopUserRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAccessLog,
  refetchIntervalChart,
} from "@/utils/client";
import { Button, Menu, SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  Activity,
  ArrowUpRight,
  ChevronDown,
  Minus,
  ShieldCheck,
  ShieldX,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import ChartPanel from "../Charts/ChartPanel";
import { LogWidgetHeader } from "../LogWidget";
import TopList from "../TopList";
import {
  accessLogStatusValue,
  AccessLogStatusFilter,
} from "./utils";
import {
  ALL_PERIODS,
  buildTimestamps,
  deltaPct,
  EXTENDED_PERIODS,
  getAutoInterval,
  n,
  pct,
  periodLabel,
  PRIMARY_PERIODS,
  refKey,
  toTs,
  visibilityKeys,
} from "@/utils/visibility";
import PeriodSelector from "../LogWidget/PeriodSelector";

const TrendBadge = ({
  cur,
  prev,
  inverse,
}: {
  cur: number;
  prev: number;
  inverse?: boolean;
}) => {
  const d = deltaPct(cur, prev);
  if (d === 0 || prev === 0)
    return (
      <span className="inline-flex items-center gap-0.5 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={3} />—
      </span>
    );
  const up = d > 0;
  const favorable = inverse ? !up : up;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-micro font-semibold",
        favorable ? "text-emerald-600" : "text-red-500",
      )}
    >
      {up ? (
        <TrendingUp size={10} strokeWidth={2.5} />
      ) : (
        <TrendingDown size={10} strokeWidth={2.5} />
      )}
      {up ? "+" : ""}
      {d}%
    </span>
  );
};

const StatCard = ({
  label,
  value,
  prevValue,
  total,
  variant,
  icon: Icon,
  to,
}: {
  label: string;
  value: number;
  prevValue: number;
  total: number;
  variant: "allowed" | "denied" | "total";
  icon: React.FC<any>;
  to: string;
}) => {
  const rate = pct(value, total);
  const colors = {
    allowed: {
      bg: "bg-emerald-50",
      border: "border-emerald-100",
      icon: "text-emerald-600",
      bar: "bg-emerald-500",
      value: "text-emerald-700",
    },
    denied: {
      bg: "bg-red-50",
      border: "border-red-100",
      icon: "text-red-500",
      bar: "bg-red-500",
      value: "text-red-700",
    },
    total: {
      bg: "bg-slate-50",
      border: "border-slate-200",
      icon: "text-slate-500",
      bar: "bg-slate-500",
      value: "text-slate-700",
    },
  }[variant];

  return (
    <Link
      to={to}
      className={twMerge(
        "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border p-3 outline-none transition-[border-color,box-shadow] duration-200 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-blue-500/30",
        colors.bg,
        colors.border,
      )}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Icon size={14} className={colors.icon} strokeWidth={2.5} />
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <TrendBadge
            cur={value}
            prev={prevValue}
            inverse={variant === "denied"}
          />
          <ArrowUpRight
            size={11}
            className="text-slate-300 transition-colors duration-200 group-hover:text-slate-500"
          />
        </div>
      </div>
      <div className="flex items-baseline gap-2">
        <span
          className={twMerge("text-2xl font-bold tabular-nums", colors.value)}
        >
          {value.toLocaleString()}
        </span>
        {variant !== "total" && total > 0 && (
          <span className="text-xs font-normal text-slate-500">
            {rate}%
          </span>
        )}
      </div>
      {variant !== "total" && total > 0 && (
        <div className="h-1 w-full bg-white/60 rounded-full overflow-hidden">
          <div
            className={twMerge(
              "h-full rounded-full transition-[width] duration-200",
              colors.bar,
            )}
            style={{ width: `${rate}%` }}
          />
        </div>
      )}
      <div className="text-micro font-normal text-slate-500">
        Previous period: {prevValue.toLocaleString()}
      </div>
    </Link>
  );
};

const StatusSelector = ({
  value,
  onChange,
}: {
  value: AccessLogStatusFilter;
  onChange: (value: AccessLogStatusFilter) => void;
}) => (
  <SegmentedControl
    className="shrink-0"
    size="xs"
    value={value}
    onChange={(nextValue) =>
      onChange(nextValue as AccessLogStatusFilter)
    }
    data={[
      {
        value: "all",
        label: (
          <span className="flex items-center justify-center gap-1.5">
            <Activity size={12} strokeWidth={2.4} />
            All
          </span>
        ),
      },
      {
        value: "allowed",
        label: (
          <span className="flex items-center justify-center gap-1.5">
            <ShieldCheck size={12} strokeWidth={2.4} />
            Allowed
          </span>
        ),
      },
      {
        value: "denied",
        label: (
          <span className="flex items-center justify-center gap-1.5">
            <ShieldX size={12} strokeWidth={2.4} />
            Denied
          </span>
        ),
      },
    ]}
  />
);

interface AccessLogHealthWidgetProps {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  periodMinutes?: number;
  onPeriodChange?: (value: number) => void;
  hideRangeControl?: boolean;
  status?: AccessLogStatusFilter;
  onStatusChange?: (value: AccessLogStatusFilter) => void;
}

const getAccessLogPath = (
  props: AccessLogHealthWidgetProps,
  status?: "ALLOWED" | "DENIED",
) => {
  const params = new URLSearchParams();
  const refs = [
    ["userRef", props.userRef],
    ["sessionRef", props.sessionRef],
    ["serviceRef", props.serviceRef],
    ["namespaceRef", props.namespaceRef],
    ["regionRef", props.regionRef],
    ["deviceRef", props.deviceRef],
    ["policyRef", props.policyRef],
  ] as const;

  refs.forEach(([key, ref]) => {
    if (ref?.name) params.set(`${key}.name`, ref.name);
    else if (ref?.uid) params.set(`${key}.uid`, ref.uid);
  });
  if (status) params.set("status", status);

  const query = params.toString();
  return `/visibility/accesslogs${query ? `?${query}` : ""}`;
};

const AccessLogHealthWidget = (props: AccessLogHealthWidgetProps) => {
  const [localPeriodMinutes, setLocalPeriodMinutes] = useState(60);
  const [localStatus, setLocalStatus] = useState<AccessLogStatusFilter>("all");
  const periodMinutes = props.periodMinutes ?? localPeriodMinutes;
  const status = props.status ?? localStatus;
  const setPeriodMinutes = props.onPeriodChange ?? setLocalPeriodMinutes;
  const setStatus = props.onStatusChange ?? setLocalStatus;
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const rangeLabel = periodLabel(periodMinutes);
  const scopedLogsPath = getAccessLogPath(
    props,
    status === "allowed" ? "ALLOWED" : status === "denied" ? "DENIED" : undefined,
  );

  const refKeys = {
    userRef: refKey(props.userRef),
    sessionRef: refKey(props.sessionRef),
    serviceRef: refKey(props.serviceRef),
    namespaceRef: refKey(props.namespaceRef),
    regionRef: refKey(props.regionRef),
    deviceRef: refKey(props.deviceRef),
    policyRef: refKey(props.policyRef),
  };

  const showTopUsers = !props.userRef && !props.deviceRef && !props.sessionRef;
  const showTopServices = !props.serviceRef && !props.namespaceRef;
  const showTopSessions = !props.sessionRef;
  const showTopPolicies = !props.policyRef;

  const curSummary = useQuery({
    queryKey: visibilityKeys.accessSummary(
      "current",
      periodMinutes,
      status,
      refKeys,
    ),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const prevSummary = useQuery({
    queryKey: visibilityKeys.accessSummary(
      "previous",
      periodMinutes,
      status,
      refKeys,
    ),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const dataPoint = useQuery({
    queryKey: visibilityKeys.accessDataPoint(periodMinutes, status, refKeys),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogDataPoint(
          GetAccessLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: autoInterval,
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topUsers = useQuery({
    queryKey: ["accessLogTopUser", periodMinutes, status, refKeys],
    enabled: showTopUsers,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopUser(
          ListAccessLogTopUserRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            regionRef: props.regionRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topServices = useQuery({
    queryKey: ["accessLogTopService", periodMinutes, status, refKeys],
    enabled: showTopServices,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopService(
          ListAccessLogTopServiceRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topPolicies = useQuery({
    queryKey: ["accessLogTopPolicy", periodMinutes, status, refKeys],
    enabled: showTopPolicies,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopPolicy(
          ListAccessLogTopPolicyRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topSessions = useQuery({
    queryKey: ["accessLogTopSession", periodMinutes, status, refKeys],
    enabled: showTopSessions,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().listAccessLogTopSession(
          ListAccessLogTopSessionRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            regionRef: props.regionRef,
            deviceRef: props.deviceRef,
            serviceRef: props.serviceRef,
            namespaceRef: props.namespaceRef,
            policyRef: props.policyRef,
            status: accessLogStatusValue(status),
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const isSummaryLoading = curSummary.isLoading || prevSummary.isLoading;
  const curData = curSummary.data;
  const prevData = prevSummary.data;

  const isAnyLoading =
    isSummaryLoading ||
    dataPoint.isLoading ||
    topUsers.isLoading ||
    topServices.isLoading ||
    topPolicies.isLoading ||
    topSessions.isLoading;
  const hasError = [
    curSummary,
    prevSummary,
    dataPoint,
    topUsers,
    topServices,
    topPolicies,
    topSessions,
  ].some((query) => query.isError);

  const updatedAt = Math.max(curSummary.dataUpdatedAt, dataPoint.dataUpdatedAt);

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    if (showTopUsers) topUsers.refetch();
    if (showTopServices) topServices.refetch();
    if (showTopPolicies) topPolicies.refetch();
    if (showTopSessions) topSessions.refetch();
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <LogWidgetHeader
        icon={Activity}
        title="Access activity"
        description={`Compared with the previous ${rangeLabel}`}
        isLoading={isAnyLoading}
        isError={hasError}
        updatedAt={updatedAt}
        onRefresh={refetchAll}
      >
        <div className="flex flex-wrap items-center justify-end gap-1.5">
          <StatusSelector value={status} onChange={setStatus} />
          {!props.hideRangeControl && (
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
        )}
        </div>
      </LogWidgetHeader>

      {hasError && (
        <div
          role="alert"
          className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800"
        >
          Some access-log data could not be loaded. Showing the available
          results; try refreshing to retry.
        </div>
      )}

      {isSummaryLoading ? (
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : curData && prevData ? (
        <>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
            <StatCard
              label="Total"
              value={Number(curData.totalNumber)}
              prevValue={Number(prevData.totalNumber)}
              total={Number(curData.totalNumber)}
              variant="total"
              icon={Activity}
              to={getAccessLogPath(props)}
            />
            <StatCard
              label="Allowed"
              value={Number(curData.totalAllowed)}
              prevValue={Number(prevData.totalAllowed)}
              total={Number(curData.totalNumber)}
              variant="allowed"
              icon={ShieldCheck}
              to={getAccessLogPath(props, "ALLOWED")}
            />
            <StatCard
              label="Denied"
              value={Number(curData.totalDenied)}
              prevValue={Number(prevData.totalDenied)}
              total={Number(curData.totalNumber)}
              variant="denied"
              icon={ShieldX}
              to={getAccessLogPath(props, "DENIED")}
            />
          </div>

          {Number(curData.totalNumber) > 0 && (
            <div className="flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2">
              <div className="flex-1 h-2 rounded-full bg-slate-200 overflow-hidden flex">
                <div
                  className="h-full bg-emerald-500 transition-[width] duration-200"
                  style={{
                    width: `${pct(Number(curData.totalAllowed), Number(curData.totalNumber))}%`,
                  }}
                />
                <div
                  className="h-full bg-red-500 transition-[width] duration-200"
                  style={{
                    width: `${pct(Number(curData.totalDenied), Number(curData.totalNumber))}%`,
                  }}
                />
              </div>
              <span className="text-micro font-normal text-slate-500 shrink-0 tabular-nums">
                {pct(Number(curData.totalAllowed), Number(curData.totalNumber))}
                % allowed
              </span>
            </div>
          )}

          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {[
              { label: "Sessions", value: Number(curData.totalSession) },
              { label: "Users", value: Number(curData.totalUser) },
              { label: "Services", value: Number(curData.totalService) },
              { label: "Namespaces", value: Number(curData.totalNamespace) },
            ].map(({ label, value }) => (
              <div
                key={label}
                className="flex flex-col gap-0.5 px-3 py-2.5 rounded-lg border border-slate-200 bg-white"
              >
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                  {label}
                </span>
                <span className="text-sm font-bold text-slate-700 tabular-nums">
                  {value.toLocaleString()}
                </span>
              </div>
            ))}
          </div>
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-body font-normal text-slate-500">
            No data available for this period
          </span>
        </div>
      )}

      <ChartPanel
        caption={`Activity — last ${rangeLabel}`}
        points={(dataPoint.data?.datapoints ?? []).map((x) => ({
          ts: x.timestamp!,
          value: x.count,
        }))}
      />

      {((showTopUsers && topUsers.data && topUsers.data?.items.length > 0) ||
        (showTopServices &&
          topServices.data &&
          topServices.data?.items.length > 0) ||
        (showTopPolicies &&
          topPolicies.data &&
          topPolicies.data?.items.length > 0) ||
        (showTopSessions &&
          topSessions.data &&
          topSessions.data?.items.length > 0)) && (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {showTopUsers && topUsers.data && topUsers.data?.items.length > 0 && (
            <TopList
              title="Top Users"
              to={scopedLogsPath}
              items={topUsers.data.items.map((x) => ({
                resource: x.user!,
                count: x.count,
              }))}
            />
          )}
          {showTopServices &&
            topServices.data &&
            topServices.data?.items.length > 0 && (
              <TopList
                title="Top Services"
                to={scopedLogsPath}
                items={topServices.data.items.map((x) => ({
                  resource: x.service!,
                  count: x.count,
                }))}
              />
            )}
          {showTopPolicies &&
            topPolicies.data &&
            topPolicies.data?.items.length > 0 && (
              <TopList
                title="Top Policies"
                to={scopedLogsPath}
                items={topPolicies.data.items.map((x) => ({
                  resource: x.policy!,
                  count: x.count,
                }))}
              />
            )}
          {showTopSessions &&
            topSessions.data &&
            topSessions.data?.items.length > 0 && (
              <TopList
                title="Top Sessions"
                to={scopedLogsPath}
                items={topSessions.data.items.map((x) => ({
                  resource: x.session!,
                  count: x.count,
                }))}
              />
            )}
        </div>
      )}
    </div>
  );
};

export default AccessLogHealthWidget;
