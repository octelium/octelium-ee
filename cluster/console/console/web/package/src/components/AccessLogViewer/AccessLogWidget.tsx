import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetAccessLogSummaryRequest } from "@/apis/visibilityv1/visibilityv1";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { Button } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  Activity,
  Minus,
  RefreshCw,
  ShieldCheck,
  ShieldX,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { twMerge } from "tailwind-merge";

interface PeriodOption {
  label: string;
  minutes: number;
}

const PERIOD_OPTIONS: PeriodOption[] = [
  { label: "30m", minutes: 30 },
  { label: "1h", minutes: 60 },
  { label: "3h", minutes: 180 },
  { label: "6h", minutes: 360 },
  { label: "12h", minutes: 720 },
  { label: "24h", minutes: 1440 },
];

const buildRequest = (fromMs: number, toMs: number) =>
  GetAccessLogSummaryRequest.create({
    from: Timestamp.fromDate(new Date(fromMs)),
    to: Timestamp.fromDate(new Date(toMs)),
  });

interface SummaryData {
  totalNumber: number;
  totalAllowed: number;
  totalDenied: number;
}

const useSummaryPair = (periodMinutes: number) => {
  const now = dayjs();
  const curFrom = now.subtract(periodMinutes, "minute").valueOf();
  const curTo = now.valueOf();
  const prevFrom = now.subtract(periodMinutes * 2, "minute").valueOf();
  const prevTo = curFrom;

  const current = useQuery({
    queryKey: ["accessLogSummary", "current", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          buildRequest(curFrom, curTo),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const previous = useQuery({
    queryKey: ["accessLogSummary", "previous", periodMinutes],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          buildRequest(prevFrom, prevTo),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  return { current, previous, curFrom, curTo };
};

const pct = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100);

const delta = (cur: number, prev: number): number =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

const TrendBadge = ({ cur, prev }: { cur: number; prev: number }) => {
  const d = delta(cur, prev);
  if (d === 0 || prev === 0) {
    return (
      <span className="inline-flex items-center gap-0.5 text-[0.65rem] font-bold text-slate-400">
        <Minus size={10} strokeWidth={3} />—
      </span>
    );
  }
  const positive = d > 0;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-[0.65rem] font-bold",
        positive ? "text-emerald-600" : "text-red-500",
      )}
    >
      {positive ? (
        <TrendingUp size={10} strokeWidth={2.5} />
      ) : (
        <TrendingDown size={10} strokeWidth={2.5} />
      )}
      {positive ? "+" : ""}
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
}: {
  label: string;
  value: number;
  prevValue: number;
  total: number;
  variant: "allowed" | "denied" | "total";
  icon: React.FC<any>;
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
    <div
      className={twMerge(
        "flex flex-col gap-3 p-4 rounded-xl border",
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
        <TrendBadge cur={value} prev={prevValue} />
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

      <div className="text-[0.65rem] font-semibold text-slate-400">
        prev period: {prevValue.toLocaleString()}
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
}) => (
  <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
    {PERIOD_OPTIONS.map((opt) => {
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
  </Button.Group>
);

const AccessLogHealthWidget = () => {
  const [periodMinutes, setPeriodMinutes] = useState(60);
  const { current, previous, curFrom, curTo } = useSummaryPair(periodMinutes);

  const isLoading = current.isLoading || previous.isLoading;
  const curData = current.data;
  const prevData = previous.data;

  const refetch = () => {
    current.refetch();
    previous.refetch();
  };

  const periodLabel =
    PERIOD_OPTIONS.find((o) => o.minutes === periodMinutes)?.label ?? "";

  return (
    <div className="w-full flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
          <span className="text-[0.68rem] font-semibold text-slate-400">
            vs previous {periodLabel}
          </span>
        </div>

        <button
          onClick={refetch}
          disabled={isLoading}
          className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] disabled:opacity-50"
        >
          <RefreshCw
            size={11}
            strokeWidth={2.5}
            className={isLoading ? "animate-spin" : ""}
          />
          Refresh
        </button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-3 gap-3">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="h-[140px] rounded-xl border border-slate-200 bg-slate-50 animate-pulse"
            />
          ))}
        </div>
      ) : curData && prevData ? (
        <>
          <div className="grid grid-cols-3 gap-3">
            <StatCard
              label="Total"
              value={Number(curData.totalNumber)}
              prevValue={Number(prevData.totalNumber)}
              total={Number(curData.totalNumber)}
              variant="total"
              icon={Activity}
            />
            <StatCard
              label="Allowed"
              value={Number(curData.totalAllowed)}
              prevValue={Number(prevData.totalAllowed)}
              total={Number(curData.totalNumber)}
              variant="allowed"
              icon={ShieldCheck}
            />
            <StatCard
              label="Denied"
              value={Number(curData.totalDenied)}
              prevValue={Number(prevData.totalDenied)}
              total={Number(curData.totalNumber)}
              variant="denied"
              icon={ShieldX}
            />
          </div>

          {Number(curData.totalNumber) > 0 && (
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-slate-50 border border-slate-200">
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

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
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
    </div>
  );
};

export default AccessLogHealthWidget;
