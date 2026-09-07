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
  ArrowUpRight,
  ChevronDown,
  CircleX,
  Minus,
  ScrollText,
  Siren,
  TriangleAlert,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import ChartPanel from "../Charts/ChartPanel";
import { STATUS_COLORS } from "@/utils/charts/palette";
import { LogWidgetHeader } from "../LogWidget";
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

const getComponentLogPath = (level?: string) =>
  `/visibility/componentlogs${level ? `?level=${level}` : ""}`;

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
        <Minus size={10} strokeWidth={3} /> —
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
  icon: Icon,
  colorClass,
  inverse,
  to,
}: {
  label: string;
  value: number;
  prevValue: number;
  icon: React.FC<any>;
  colorClass: string;
  inverse?: boolean;
  to?: string;
}) => {
  const content = (
    <>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Icon size={13} className={colorClass} strokeWidth={2.5} />
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <TrendBadge cur={value} prev={prevValue} inverse={inverse} />
          {to && (
            <ArrowUpRight
              size={11}
              className="text-slate-300 transition-colors duration-200 group-hover:text-slate-500"
            />
          )}
        </div>
      </div>
      <span className={twMerge("text-2xl font-bold tabular-nums", colorClass)}>
        {value.toLocaleString()}
      </span>
      <span className="text-micro font-normal text-slate-500">
        Previous period: {prevValue.toLocaleString()}
      </span>
    </>
  );
  const className = twMerge(
    "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border border-slate-200 bg-white p-3",
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
    {
      label: "Debug",
      level: "DEBUG",
      value: n(cur?.totalDebug),
      color: "bg-slate-400",
    },
    {
      label: "Info",
      level: "INFO",
      value: n(cur?.totalInfo),
      color: "bg-sky-500",
    },
    {
      label: "Warn",
      level: "WARN",
      value: n(cur?.totalWarn),
      color: "bg-amber-400",
    },
    {
      label: "Error",
      level: "ERROR",
      value: n(cur?.totalError),
      color: "bg-red-500",
    },
    {
      label: "Panic",
      level: "PANIC",
      value: n(cur?.totalPanic),
      color: "bg-red-700",
    },
    {
      label: "Fatal",
      level: "FATAL",
      value: n(cur?.totalFatal),
      color: "bg-red-900",
    },
  ].filter((s) => s.value > 0);

  if (segments.length === 0) return null;

  return (
    <div className="flex flex-col gap-2 rounded-xl border border-slate-200 bg-white p-3">
      <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
        Level breakdown
      </span>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden flex">
          {segments.map(({ label, value, color }) => (
            <div
              key={label}
              className={twMerge(
                "h-full transition-[width] duration-200",
                color,
              )}
              style={{ width: `${pct(value, total)}%` }}
            />
          ))}
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-1.5">
          {segments.map(({ label, level, value, color }) => (
            <Link
              key={label}
              to={getComponentLogPath(level)}
              className="flex items-center gap-1 rounded-md px-1.5 py-1 text-micro font-normal text-slate-500 outline-none transition-colors duration-200 hover:bg-slate-50 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-blue-500/30"
            >
              <span
                className={twMerge("w-2 h-2 rounded-full shrink-0", color)}
              />
              {label}: {value.toLocaleString()}
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
};

interface ComponentLogHealthWidgetProps {
  periodMinutes?: number;
  onPeriodChange?: (value: number) => void;
  hideRangeControl?: boolean;
}

const ComponentLogHealthWidget = (props: ComponentLogHealthWidgetProps = {}) => {
  const [localPeriodMinutes, setLocalPeriodMinutes] = useState(60);
  const periodMinutes = props.periodMinutes ?? localPeriodMinutes;
  const setPeriodMinutes = props.onPeriodChange ?? setLocalPeriodMinutes;
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const rangeLabel = periodLabel(periodMinutes);

  const curSummary = useQuery({
    queryKey: visibilityKeys.componentSummary("current", periodMinutes),
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
    queryKey: visibilityKeys.componentSummary("previous", periodMinutes),
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
    queryKey: visibilityKeys.componentDataPoint(periodMinutes),
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
    queryKey: visibilityKeys.componentErrorDataPoint(periodMinutes),
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

  const hasError = [curSummary, prevSummary, dataPoint, dataPointErrors].some(
    (query) => query.isError,
  );

  const updatedAt = Math.max(curSummary.dataUpdatedAt, dataPoint.dataUpdatedAt);

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    dataPointErrors.refetch();
  };

  const totalNum = n(cur?.totalNumber);

  return (
    <div className="flex w-full flex-col gap-4">
      <LogWidgetHeader
        icon={ScrollText}
        title="Component activity"
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
          Some component-log data could not be loaded. Showing the available
          results; try refreshing to retry.
        </div>
      )}

      {isSummaryLoading ? (
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : cur && prev ? (
        <>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              label="Total"
              value={n(cur.totalNumber)}
              prevValue={n(prev.totalNumber)}
              icon={Activity}
              colorClass="text-slate-700"
              to={getComponentLogPath()}
            />
            <StatCard
              label="Warn"
              value={n(cur.totalWarn)}
              prevValue={n(prev.totalWarn)}
              icon={TriangleAlert}
              colorClass="text-amber-600"
              inverse
              to={getComponentLogPath("WARN")}
            />
            <StatCard
              label="Error"
              value={n(cur.totalError)}
              prevValue={n(prev.totalError)}
              icon={CircleX}
              colorClass="text-red-600"
              inverse
              to={getComponentLogPath("ERROR")}
            />
            <StatCard
              label="Critical"
              value={n(cur.totalFatal) + n(cur.totalPanic)}
              prevValue={n(prev.totalFatal) + n(prev.totalPanic)}
              icon={Siren}
              colorClass="text-red-800"
              inverse
            />
          </div>

          {totalNum > 0 && <LevelBar cur={cur} total={totalNum} />}

          {totalNum === 0 && (
            <div className="flex items-center justify-center py-8">
              <span className="text-body font-normal text-slate-500">
                No component log events in this period
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

      <ChartPanel
        tone="critical"
        color={STATUS_COLORS.critical}
        caption={`Errors — last ${rangeLabel}`}
        emptyLabel="No errors in this range"
        points={(dataPointErrors.data?.datapoints ?? []).map((x) => ({
          ts: x.timestamp!,
          value: x.count,
        }))}
      />
    </div>
  );
};

export default ComponentLogHealthWidget;
