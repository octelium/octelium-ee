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
import LineChart from "../Charts/LineChart";
import { LogWidgetHeader } from "../LogWidget";
import TopList from "../TopList";

interface PeriodOption {
  label: string;
  minutes: number;
}

const PRIMARY_PERIODS: PeriodOption[] = [
  { label: "30m", minutes: 30 },
  { label: "1h", minutes: 60 },
  { label: "3h", minutes: 180 },
  { label: "6h", minutes: 360 },
  { label: "12h", minutes: 720 },
  { label: "24h", minutes: 1440 },
];

const EXTENDED_PERIODS: PeriodOption[] = [
  { label: "5m", minutes: 5 },
  { label: "10m", minutes: 10 },
  { label: "15m", minutes: 15 },
  { label: "2d", minutes: 2880 },
  { label: "3d", minutes: 4320 },
  { label: "7d", minutes: 10080 },
  { label: "14d", minutes: 20160 },
];

const ALL_PERIODS = [...PRIMARY_PERIODS, ...EXTENDED_PERIODS];

const createDuration = (val: number, unit: string): Duration => {
  const typePayload = match(unit)
    .with("millisecond", () => ({
      oneofKind: "milliseconds" as const,
      milliseconds: val,
    }))
    .with("second", () => ({ oneofKind: "seconds" as const, seconds: val }))
    .with("minute", () => ({ oneofKind: "minutes" as const, minutes: val }))
    .with("hour", () => ({ oneofKind: "hours" as const, hours: val }))
    .with("day", () => ({ oneofKind: "days" as const, days: val }))
    .with("week", () => ({ oneofKind: "weeks" as const, weeks: val }))
    .with("month", () => ({ oneofKind: "months" as const, months: val }))
    .otherwise(() => ({ oneofKind: "seconds" as const, seconds: val }));
  return Duration.create({ type: typePayload as any });
};

const getAutoInterval = (periodMinutes: number): Duration => {
  if (periodMinutes <= 15) return createDuration(30, "second");
  if (periodMinutes <= 60) return createDuration(1, "minute");
  if (periodMinutes <= 180) return createDuration(5, "minute");
  if (periodMinutes <= 360) return createDuration(10, "minute");
  if (periodMinutes <= 720) return createDuration(15, "minute");
  if (periodMinutes <= 1440) return createDuration(30, "minute");
  if (periodMinutes <= 4320) return createDuration(1, "hour");
  if (periodMinutes <= 10080) return createDuration(3, "hour");
  return createDuration(6, "hour");
};

const buildTimestamps = (periodMinutes: number) => {
  const now = dayjs();
  const curFrom = now.subtract(periodMinutes, "minute").valueOf();
  const curTo = now.valueOf();
  const prevFrom = now.subtract(periodMinutes * 2, "minute").valueOf();
  const prevTo = curFrom;
  return { curFrom, curTo, prevFrom, prevTo };
};

const toTs = (ms: number) => Timestamp.fromDate(new Date(ms));
const n = (v: unknown) => Number(v ?? 0);
const refKey = (ref?: ObjectReference) => ref?.uid ?? ref?.name ?? null;

const deltaPct = (cur: number, prev: number) =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

const TrendBadge = ({ cur, prev }: { cur: number; prev: number }) => {
  const d = deltaPct(cur, prev);
  if (d === 0 || prev === 0)
    return (
      <span className="inline-flex items-center gap-0.5 text-[0.65rem] font-bold text-slate-400">
        <Minus size={10} strokeWidth={3} /> —
      </span>
    );
  const up = d > 0;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-[0.65rem] font-bold",
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
          <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <TrendBadge cur={value} prev={prevValue} />
          {to && (
            <ArrowUpRight
              size={11}
              className="text-slate-300 transition-colors duration-500 group-hover:text-slate-500"
            />
          )}
        </div>
      </div>
      <span className="text-2xl font-bold tabular-nums text-slate-700">
        {value.toLocaleString()}
      </span>
      <span className="text-[0.64rem] font-semibold text-slate-400">
        Previous period: {prevValue.toLocaleString()}
      </span>
    </>
  );
  const className = twMerge(
    "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border border-slate-200 bg-slate-50 p-3",
    to &&
      "outline-none transition-[border-color,box-shadow] duration-500 hover:shadow-[0_5px_18px_rgba(15,23,42,0.07)] focus-visible:ring-2 focus-visible:ring-blue-500/30",
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
    <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {label}
    </span>
    <span className="text-[0.9rem] font-bold text-slate-700 tabular-nums">
      {value.toLocaleString()}
    </span>
  </div>
);

const PeriodSelector = ({
  value,
  onChange,
}: {
  value: number;
  onChange: (v: number) => void;
}) => {
  const isExtended = EXTENDED_PERIODS.some((p) => p.minutes === value);
  const extendedLabel = isExtended
    ? ALL_PERIODS.find((p) => p.minutes === value)?.label
    : undefined;

  return (
    <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
      {PRIMARY_PERIODS.map((opt) => {
        const active = opt.minutes === value;
        return (
          <Button
            type="button"
            key={opt.minutes}
            onClick={() => onChange(opt.minutes)}
            styles={{
              root: {
                height: "26px",
                fontSize: "0.7rem",
                fontWeight: 700,
                fontFamily: "Ubuntu, sans-serif",
                padding: "0 10px",
                backgroundColor: active ? "#0f172a" : "#ffffff",
                color: active ? "#ffffff" : "#64748b",
                border: "none",
                borderRadius: 0,
                transition: "background-color 150ms, color 150ms",
                "&:hover": {
                  backgroundColor: active ? "#1e293b" : "#f8fafc",
                  color: active ? "#ffffff" : "#0f172a",
                },
              },
            }}
          >
            {opt.label}
          </Button>
        );
      })}

      <Menu position="bottom-end" offset={4} withArrow={false}>
        <Menu.Target>
          <Button
            type="button"
            styles={{
              root: {
                height: "26px",
                fontSize: "0.7rem",
                fontWeight: 700,
                fontFamily: "Ubuntu, sans-serif",
                padding: "0 8px",
                backgroundColor: isExtended ? "#0f172a" : "#ffffff",
                color: isExtended ? "#ffffff" : "#64748b",
                border: "none",
                borderLeft: "1px solid #e2e8f0",
                borderRadius: 0,
                transition: "background-color 150ms, color 150ms",
                "&:hover": {
                  backgroundColor: isExtended ? "#1e293b" : "#f8fafc",
                  color: isExtended ? "#ffffff" : "#0f172a",
                },
              },
            }}
          >
            <span className="flex items-center gap-1">
              {extendedLabel ?? "More"}
              <ChevronDown size={10} strokeWidth={2.5} />
            </span>
          </Button>
        </Menu.Target>
        <Menu.Dropdown>
          <div className="flex flex-col py-1 min-w-[100px]">
            {EXTENDED_PERIODS.map((opt) => (
              <button
                type="button"
                key={opt.minutes}
                onClick={() => onChange(opt.minutes)}
                className={twMerge(
                  "flex items-center px-3 h-8 text-[0.75rem] font-bold cursor-pointer transition-colors duration-100 text-left",
                  opt.minutes === value
                    ? "bg-slate-900 text-white"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900",
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </Menu.Dropdown>
      </Menu>
    </Button.Group>
  );
};

interface AuditLogHealthWidgetProps {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  resourceRef?: ObjectReference;
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
  const [periodMinutes, setPeriodMinutes] = useState(60);
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const periodLabel =
    ALL_PERIODS.find((o) => o.minutes === periodMinutes)?.label ?? "";

  const refKeys = {
    userRef: refKey(props.userRef),
    sessionRef: refKey(props.sessionRef),
    deviceRef: refKey(props.deviceRef),
    resourceRef: refKey(props.resourceRef),
  };

  const showTopUsers = !props.userRef && !props.sessionRef && !props.deviceRef;
  const showTopSessions = !props.sessionRef;

  const curSummary = useQuery({
    queryKey: ["auditLogSummary", "current", periodMinutes, refKeys],
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
    queryKey: ["auditLogSummary", "previous", periodMinutes, refKeys],
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
    queryKey: ["auditLogDataPoint", periodMinutes, refKeys],
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
        description={`Compared with the previous ${periodLabel}`}
        isLoading={isAnyLoading}
        onRefresh={refetchAll}
      >
        <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
      </LogWidgetHeader>

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
              <span className="text-[0.75rem] font-semibold text-slate-400">
                No audit events in this period
              </span>
            </div>
          )}
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-[0.75rem] font-semibold text-slate-400">
            No data available
          </span>
        </div>
      )}

      {dataPoint.data?.datapoints && dataPoint.data.datapoints.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400 block mb-3">
            Activity — last {periodLabel}
          </span>
          <LineChart
            points={dataPoint.data.datapoints.map((x) => ({
              ts: x.timestamp!,
              value: x.count,
            }))}
          />
        </div>
      )}

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
