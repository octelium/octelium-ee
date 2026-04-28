import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration } from "@/apis/metav1/metav1";
import {
  GetComponentLogDataPointRequest,
  GetComponentLogSummaryRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityComponentLog,
  refetchIntervalChart,
} from "@/utils/client";
import { Button, Menu } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  Activity,
  ChevronDown,
  Minus,
  RefreshCw,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import LineChart from "../Charts/LineChart";

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

const deltaPct = (cur: number, prev: number) =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

const pct = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100);

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
  colorClass,
}: {
  label: string;
  value: number;
  prevValue: number;
  icon: React.FC<any>;
  colorClass: string;
}) => (
  <div className="flex flex-col gap-2.5 p-4 rounded-xl border bg-white border-slate-200">
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-1.5">
        <Icon size={13} className={colorClass} strokeWidth={2.5} />
        <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500">
          {label}
        </span>
      </div>
      <TrendBadge cur={value} prev={prevValue} />
    </div>
    <span className={twMerge("text-2xl font-bold tabular-nums", colorClass)}>
      {value.toLocaleString()}
    </span>
    <span className="text-[0.63rem] font-semibold text-slate-400">
      prev: {prevValue.toLocaleString()}
    </span>
  </div>
);

const LevelBar = ({
  cur,
  total,
}: {
  cur: {
    totalDebug?: number;
    totalInfo?: number;
    totalWarn?: number;
    totalError?: number;
    totalPanic?: number;
    totalFatal?: number;
  };
  total: number;
}) => {
  const segments = [
    { label: "Debug", value: n(cur?.totalDebug), color: "bg-slate-400" },
    { label: "Info", value: n(cur?.totalInfo), color: "bg-sky-500" },
    { label: "Warn", value: n(cur?.totalWarn), color: "bg-amber-400" },
    { label: "Error", value: n(cur?.totalError), color: "bg-red-500" },
    { label: "Panic", value: n(cur?.totalPanic), color: "bg-red-700" },
    { label: "Fatal", value: n(cur?.totalFatal), color: "bg-red-900" },
  ].filter((s) => s.value > 0);

  if (segments.length === 0) return null;

  return (
    <div className="flex flex-col gap-2 p-4 rounded-xl border border-slate-200 bg-white">
      <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-400">
        Level breakdown
      </span>
      <div className="flex items-center gap-2">
        <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden flex">
          {segments.map(({ label, value, color }) => (
            <div
              key={label}
              className={twMerge(
                "h-full transition-[width] duration-500",
                color,
              )}
              style={{ width: `${pct(value, total)}%` }}
            />
          ))}
        </div>
        <div className="flex items-center gap-3 shrink-0 flex-wrap">
          {segments.map(({ label, value, color }) => (
            <span
              key={label}
              className="flex items-center gap-1 text-[0.65rem] font-bold text-slate-500"
            >
              <span
                className={twMerge("w-2 h-2 rounded-full shrink-0", color)}
              />
              {label}: {value.toLocaleString()}
            </span>
          ))}
        </div>
      </div>
    </div>
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

const ComponentLogHealthWidget = () => {
  const [periodMinutes, setPeriodMinutes] = useState(60);
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const periodLabel =
    ALL_PERIODS.find((o) => o.minutes === periodMinutes)?.label ?? "";

  const curSummary = useQuery({
    queryKey: ["componentLogSummary", "current", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const prevSummary = useQuery({
    queryKey: ["componentLogSummary", "previous", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const dataPoint = useQuery({
    queryKey: ["componentLogDataPoint", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogDataPoint(
          GetComponentLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: autoInterval,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const dataPointErrors = useQuery({
    queryKey: ["componentLogDataPointErrors", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogDataPoint(
          GetComponentLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: autoInterval,
            level: ComponentLog_Entry_Level.ERROR,
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
    isSummaryLoading || dataPoint.isLoading || dataPointErrors.isLoading;

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    dataPointErrors.refetch();
  };

  const totalNum = n(cur?.totalNumber);

  return (
    <div className="w-full flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
          <span className="text-[0.68rem] font-semibold text-slate-400">
            vs previous {periodLabel}
          </span>
        </div>
        <button
          onClick={refetchAll}
          disabled={isAnyLoading}
          className="flex items-center justify-center w-7 h-7 rounded-md text-slate-400 border border-slate-200 bg-white hover:text-slate-700 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] disabled:opacity-50"
        >
          <RefreshCw
            size={12}
            strokeWidth={2.5}
            className={isAnyLoading ? "animate-spin" : ""}
          />
        </button>
      </div>

      {isSummaryLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-[120px] rounded-xl border border-slate-200 bg-slate-50 animate-pulse"
            />
          ))}
        </div>
      ) : cur && prev ? (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <StatCard
              label="Total"
              value={n(cur.totalNumber)}
              prevValue={n(prev.totalNumber)}
              icon={Activity}
              colorClass="text-slate-700"
            />
            <StatCard
              label="Warn"
              value={n(cur.totalWarn)}
              prevValue={n(prev.totalWarn)}
              icon={Activity}
              colorClass="text-amber-600"
            />
            <StatCard
              label="Error"
              value={n(cur.totalError)}
              prevValue={n(prev.totalError)}
              icon={Activity}
              colorClass="text-red-600"
            />
            <StatCard
              label="Fatal"
              value={n(cur.totalFatal) + n(cur.totalPanic)}
              prevValue={n(prev.totalFatal) + n(prev.totalPanic)}
              icon={Activity}
              colorClass="text-red-800"
            />
          </div>

          {totalNum > 0 && <LevelBar cur={cur} total={totalNum} />}

          {totalNum === 0 && (
            <div className="flex items-center justify-center py-8">
              <span className="text-[0.75rem] font-semibold text-slate-400">
                No component log events in this period
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

      {dataPointErrors.data?.datapoints &&
        dataPointErrors.data.datapoints.length > 0 && (
          <div className="rounded-xl border border-red-100 bg-red-50/40 p-4">
            <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-red-400 block mb-3">
              Errors — last {periodLabel}
            </span>
            <LineChart
              points={dataPointErrors.data.datapoints.map((x) => ({
                ts: x.timestamp!,
                value: x.count,
              }))}
            />
          </div>
        )}
    </div>
  );
};

export default ComponentLogHealthWidget;
