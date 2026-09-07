import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuditLogDataPointRequest,
  GetAuditLogSummaryRequest,
  ListAuditLogTopSessionRequest,
  ListAuditLogTopUserRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAuditLog,
  refetchIntervalChart,
} from "@/utils/client";
import { Button, Menu } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  Activity,
  ArrowUpRight,
  ChevronDown,
  ClipboardList,
  Laptop,
  Minus,
  TrendingDown,
  TrendingUp,
  Users,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import ChartPanel from "../Charts/ChartPanel";
import { LogWidgetHeader } from "../LogWidget";
import TopList from "../TopList";
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

const TrendBadge = ({ cur, prev }: { cur: number; prev: number }) => {
  const d = deltaPct(cur, prev);
  if (d === 0 || prev === 0)
    return (
      <span className="inline-flex items-center gap-0.5 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={3} /> —
      </span>
    );
  const up = d > 0;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-micro font-semibold",
        up ? "text-emerald-600" : "text-red-500",
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
  icon: Icon,
  to,
}: {
  label: string;
  value: number;
  prevValue: number;
  icon: React.FC<any>;
  to?: string;
}) => {
  const content = (
    <>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Icon size={13} className="text-slate-500" strokeWidth={2.5} />
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <TrendBadge cur={value} prev={prevValue} />
          {to && (
            <ArrowUpRight
              size={11}
              className="text-slate-300 transition-colors duration-200 group-hover:text-slate-500"
            />
          )}
        </div>
      </div>
      <span className="text-2xl font-bold tabular-nums text-slate-700">
        {value.toLocaleString()}
      </span>
      <span className="text-micro font-normal text-slate-500">
        Previous period: {prevValue.toLocaleString()}
      </span>
    </>
  );
  const className = twMerge(
    "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border border-slate-200 bg-slate-50 p-3",
    to &&
      "outline-none transition-[border-color,box-shadow] duration-200 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-blue-500/30",
  );

  return to ? (
    <Link to={to} className={className}>
      {content}
    </Link>
  ) : (
    <div className={className}>{content}</div>
  );
};

const MiniStat = ({ label, value }: { label: string; value: number }) => (
  <div className="flex flex-col gap-0.5 px-3 py-2.5 rounded-lg border border-slate-200 bg-white">
    <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {label}
    </span>
    <span className="text-sm font-bold text-slate-700 tabular-nums">
      {value.toLocaleString()}
    </span>
  </div>
);

interface AuditLogHealthWidgetProps {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  resourceRef?: ObjectReference;
  periodMinutes?: number;
  onPeriodChange?: (value: number) => void;
  hideRangeControl?: boolean;
}

const getAuditLogPath = (props: AuditLogHealthWidgetProps) => {
  const params = new URLSearchParams();
  const refs = [
    ["userRef", props.userRef],
    ["sessionRef", props.sessionRef],
    ["deviceRef", props.deviceRef],
    ["resourceRef", props.resourceRef],
  ] as const;

  refs.forEach(([key, ref]) => {
    if (ref?.name) params.set(`${key}.name`, ref.name);
    else if (ref?.uid) params.set(`${key}.uid`, ref.uid);
  });

  const query = params.toString();
  return `/visibility/auditlogs${query ? `?${query}` : ""}`;
};

const AuditLogHealthWidget = (props: AuditLogHealthWidgetProps) => {
  const [localPeriodMinutes, setLocalPeriodMinutes] = useState(60);
  const periodMinutes = props.periodMinutes ?? localPeriodMinutes;
  const setPeriodMinutes = props.onPeriodChange ?? setLocalPeriodMinutes;
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const rangeLabel = periodLabel(periodMinutes);

  const refKeys = {
    userRef: refKey(props.userRef),
    sessionRef: refKey(props.sessionRef),
    deviceRef: refKey(props.deviceRef),
    resourceRef: refKey(props.resourceRef),
  };

  const showTopUsers = !props.userRef && !props.sessionRef && !props.deviceRef;
  const showTopSessions = !props.sessionRef;

  const curSummary = useQuery({
    queryKey: visibilityKeys.auditSummary("current", periodMinutes, refKeys),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            resourceRef: props.resourceRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const prevSummary = useQuery({
    queryKey: visibilityKeys.auditSummary("previous", periodMinutes, refKeys),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            resourceRef: props.resourceRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const dataPoint = useQuery({
    queryKey: visibilityKeys.auditDataPoint(periodMinutes, refKeys),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogDataPoint(
          GetAuditLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: autoInterval,
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            resourceRef: props.resourceRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topUsers = useQuery({
    queryKey: ["auditLogTopUser", periodMinutes, refKeys],
    enabled: showTopUsers,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().listAuditLogTopUser(
          ListAuditLogTopUserRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            resourceRef: props.resourceRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topSessions = useQuery({
    queryKey: ["auditLogTopSession", periodMinutes, refKeys],
    enabled: showTopSessions,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().listAuditLogTopSession(
          ListAuditLogTopSessionRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            resourceRef: props.resourceRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const isSummaryLoading = curSummary.isLoading || prevSummary.isLoading;
  const cur = curSummary.data;
  const prev = prevSummary.data;

  const isAnyLoading =
    isSummaryLoading ||
    dataPoint.isLoading ||
    topUsers.isLoading ||
    topSessions.isLoading;
  const hasError = [
    curSummary,
    prevSummary,
    dataPoint,
    topUsers,
    topSessions,
  ].some((query) => query.isError);

  const updatedAt = Math.max(curSummary.dataUpdatedAt, dataPoint.dataUpdatedAt);

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    if (showTopUsers) topUsers.refetch();
    if (showTopSessions) topSessions.refetch();
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <LogWidgetHeader
        icon={ClipboardList}
        title="Audit activity"
        description={`Compared with the previous ${rangeLabel}`}
        isLoading={isAnyLoading}
        isError={hasError}
        updatedAt={updatedAt}
        onRefresh={refetchAll}
      >
        {!props.hideRangeControl && (
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
        )}
      </LogWidgetHeader>

      {hasError && (
        <div
          role="alert"
          className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800"
        >
          Some audit-log data could not be loaded. Showing the available
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
      ) : cur && prev ? (
        <>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
            <StatCard
              label="Total"
              value={n(cur.totalNumber)}
              prevValue={n(prev.totalNumber)}
              icon={Activity}
              to={getAuditLogPath(props)}
            />
            <StatCard
              label="Users"
              value={n(cur.totalUser)}
              prevValue={n(prev.totalUser)}
              icon={Users}
            />
            <StatCard
              label="Sessions"
              value={n(cur.totalSession)}
              prevValue={n(prev.totalSession)}
              icon={Laptop}
            />
          </div>

          {n(cur.totalNumber) > 0 && (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <MiniStat label="Resources" value={n(cur.totalResource)} />
              <MiniStat label="Devices" value={n(cur.totalDevice)} />
            </div>
          )}

          {n(cur.totalNumber) === 0 && (
            <div className="flex items-center justify-center py-8">
              <span className="text-body font-normal text-slate-500">
                No audit events in this period
              </span>
            </div>
          )}
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-body font-normal text-slate-500">
            No data available
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
        (showTopSessions &&
          topSessions.data &&
          topSessions.data?.items.length > 0)) && (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {showTopUsers && topUsers.data && topUsers.data?.items.length > 0 && (
            <TopList
              title="Top Users"
              to={getAuditLogPath(props)}
              items={topUsers.data.items.map((x) => ({
                resource: x.user!,
                count: x.count,
              }))}
            />
          )}
          {showTopSessions &&
            topSessions.data &&
            topSessions.data?.items.length > 0 && (
              <TopList
                title="Top Sessions"
                to={getAuditLogPath(props)}
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

export default AuditLogHealthWidget;
