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
import { Button, Menu } from "@mantine/core";
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
import LineChart from "../Charts/LineChart";
import { LogWidgetHeader } from "../LogWidget";
import TopList from "../TopList";
import {
  accessLogStatusValue,
  AccessLogStatusFilter,
} from "./utils";

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

const pct = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100);

const deltaPct = (cur: number, prev: number) =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

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
      <span className="inline-flex items-center gap-0.5 text-[0.65rem] font-bold text-slate-400">
        <Minus size={10} strokeWidth={3} />—
      </span>
    );
  const up = d > 0;
  const favorable = inverse ? !up : up;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-[0.65rem] font-bold",
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
        "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border p-3 outline-none transition-[border-color,box-shadow] duration-500 hover:shadow-[0_5px_18px_rgba(15,23,42,0.07)] focus-visible:ring-2 focus-visible:ring-blue-500/30",
        colors.bg,
        colors.border,
      )}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Icon size={14} className={colors.icon} strokeWidth={2.5} />
          <span className="text-[0.7rem] font-bold uppercase tracking-[0.06em] text-slate-500">
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
            className="text-slate-300 transition-colors duration-500 group-hover:text-slate-500"
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
          <span className="text-[0.72rem] font-semibold text-slate-400">
            {rate}%
          </span>
        )}
      </div>
      {variant !== "total" && total > 0 && (
        <div className="h-1 w-full bg-white/60 rounded-full overflow-hidden">
          <div
            className={twMerge(
              "h-full rounded-full transition-[width] duration-500",
              colors.bar,
            )}
            style={{ width: `${rate}%` }}
          />
        </div>
      )}
      <div className="text-[0.64rem] font-semibold text-slate-400">
        Previous period: {prevValue.toLocaleString()}
      </div>
    </Link>
  );
};

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

const StatusSelector = ({
  value,
  onChange,
}: {
  value: AccessLogStatusFilter;
  onChange: (value: AccessLogStatusFilter) => void;
}) => (
  <div className="flex items-center rounded-md border border-slate-200 bg-white p-0.5 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
    {[
      { value: "all" as const, label: "All" },
      { value: "allowed" as const, label: "Allowed" },
      { value: "denied" as const, label: "Denied" },
    ].map((option) => {
      const active = option.value === value;
      return (
        <button
          type="button"
          key={option.value}
          onClick={() => onChange(option.value)}
          className={twMerge(
            "rounded px-2 py-1 text-[0.65rem] font-bold transition-colors duration-150",
            active
              ? option.value === "allowed"
                ? "bg-emerald-500 text-white"
                : option.value === "denied"
                  ? "bg-red-500 text-white"
                  : "bg-slate-900 text-white"
              : "text-slate-500 hover:bg-slate-50 hover:text-slate-800",
          )}
        >
          {option.label}
        </button>
      );
    })}
  </div>
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
  status?: AccessLogStatusFilter;
  onStatusChange?: (value: AccessLogStatusFilter) => void;
}

const refKey = (ref?: ObjectReference) => ref?.uid ?? ref?.name ?? null;

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
  const periodLabel =
    ALL_PERIODS.find((o) => o.minutes === periodMinutes)?.label ?? "";
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
    queryKey: ["accessLogSummary", "current", periodMinutes, status, refKeys],
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
    queryKey: ["accessLogSummary", "previous", periodMinutes, status, refKeys],
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
    queryKey: ["accessLogDataPoint", periodMinutes, status, refKeys],
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
        description={`Compared with the previous ${periodLabel}`}
        isLoading={isAnyLoading}
        onRefresh={refetchAll}
      >
        <div className="flex flex-wrap items-center justify-end gap-1.5">
          <StatusSelector value={status} onChange={setStatus} />
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
        </div>
      </LogWidgetHeader>

      {hasError && (
        <div
          role="alert"
          className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[0.7rem] font-semibold text-amber-800"
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
                  className="h-full bg-emerald-500 transition-[width] duration-500"
                  style={{
                    width: `${pct(Number(curData.totalAllowed), Number(curData.totalNumber))}%`,
                  }}
                />
                <div
                  className="h-full bg-red-500 transition-[width] duration-500"
                  style={{
                    width: `${pct(Number(curData.totalDenied), Number(curData.totalNumber))}%`,
                  }}
                />
              </div>
              <span className="text-[0.65rem] font-bold text-slate-500 shrink-0 tabular-nums">
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
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  {label}
                </span>
                <span className="text-[0.9rem] font-bold text-slate-700 tabular-nums">
                  {value.toLocaleString()}
                </span>
              </div>
            ))}
          </div>
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-[0.75rem] font-semibold text-slate-400">
            No data available for this period
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
