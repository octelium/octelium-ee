import { Timestamp } from "@/apis/google/protobuf/timestamp";
import LineChart from "@/components/Charts/LineChart";
import { useInView } from "@/hooks/use-in-view";
import { deltaPct } from "@/utils/visibility";
import {
  ArrowUpRight,
  LucideIcon,
  Minus,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { compact, exact } from "./utils";

export type Point = { ts: Timestamp; value: number };

export const toPoints = (
  datapoints?: { timestamp?: Timestamp; count: number | bigint }[],
): Point[] =>
  (datapoints ?? [])
    .filter((item) => !!item.timestamp)
    .map((item) => ({ ts: item.timestamp!, value: Number(item.count) }));

export const seriesPoints = (
  series:
    | {
        key: string;
        datapoints: { timestamp?: Timestamp; count: number | bigint }[];
      }[]
    | undefined,
  key: string,
): Point[] => toPoints(series?.find((item) => item.key === key)?.datapoints);

export const Panel = (props: {
  icon: LucideIcon;
  title: string;
  description?: string;
  to?: string;
  toLabel?: string;
  actions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) => {
  const Icon = props.icon;

  return (
    <section
      className={twMerge(
        "overflow-hidden rounded-xl border border-slate-200 bg-white",
        props.className,
      )}
    >
      <header className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5 border-b border-slate-100 px-4 py-3.5 sm:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <Icon size={15} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <h2 className="truncate text-body font-semibold text-slate-800">
              {props.title}
            </h2>
            {props.description && (
              <p className="truncate text-micro font-normal text-slate-500">
                {props.description}
              </p>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {props.actions}
          {props.to && (
            <Link
              to={props.to}
              className="group inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-micro font-semibold text-slate-600 outline-none transition-colors duration-150 hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              {props.toLabel ?? "Open"}
              <ArrowUpRight
                size={12}
                strokeWidth={2.5}
                className="transition-transform duration-150 group-hover:-translate-y-0.5 group-hover:translate-x-0.5"
              />
            </Link>
          )}
        </div>
      </header>

      <div className="px-4 py-4 sm:px-5 sm:py-5">{props.children}</div>
    </section>
  );
};

export const PanelSkeleton = (props: { height: number }) => (
  <div
    aria-hidden="true"
    className="w-full animate-pulse rounded-xl border border-slate-200 bg-slate-50/70"
    style={{ height: props.height }}
  />
);

export const Deferred = (props: {
  height?: number;
  children: React.ReactNode;
}) => {
  const { ref, inView } = useInView<HTMLDivElement>();
  const height = props.height ?? 280;

  return (
    <div ref={ref}>
      {inView ? props.children : <PanelSkeleton height={height} />}
    </div>
  );
};

export const Trend = (props: {
  cur: number;
  prev: number;
  upIsGood: boolean;
  rangeLabel: string;
}) => {
  const change = deltaPct(props.cur, props.prev);

  if (props.prev === 0 && props.cur === 0) {
    return (
      <span className="inline-flex items-center gap-1 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={2.5} />
        No activity
      </span>
    );
  }

  if (change === 0) {
    return (
      <span className="inline-flex items-center gap-1 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={2.5} />
        Flat
      </span>
    );
  }

  const up = change > 0;
  const favorable = up === props.upIsGood;
  const Icon = up ? TrendingUp : TrendingDown;

  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-micro font-semibold tabular-nums",
        favorable ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-700",
      )}
      title={`${exact(props.cur)} now vs ${exact(props.prev)} in the previous ${props.rangeLabel}`}
    >
      <Icon size={10} strokeWidth={2.5} />
      {up ? "+" : ""}
      {change}%
    </span>
  );
};

const ProportionBar = (props: {
  value: number;
  total: number;
  label: string;
  color: string;
}) => {
  const share = props.total === 0 ? 0 : (props.value / props.total) * 100;

  return (
    <div className="flex h-10 flex-col justify-center gap-1.5">
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100">
        <span
          className="block h-full rounded-full transition-[width] duration-300"
          style={{
            width: `${Math.max(share, props.value > 0 ? 3 : 0)}%`,
            backgroundColor: props.color,
          }}
        />
      </div>
      <span className="text-micro font-normal tabular-nums text-slate-500">
        {Math.round(share)}% {props.label}
      </span>
    </div>
  );
};

export const StatTile = (props: {
  label: string;
  value: number;
  badge?: string;
  footer: React.ReactNode;
  trend?: { cur: number; prev: number; upIsGood: boolean; rangeLabel: string };
  points?: Point[];
  bar?: { value: number; total: number; label: string };
  color: string;
  to: string;
  isLoading?: boolean;
  isError?: boolean;
  hasData?: boolean;
}) => {
  const unavailable = props.isError && !props.hasData;

  return (
    <Link
      to={props.to}
      className="group relative flex flex-col gap-2.5 rounded-xl border border-slate-200 bg-white px-4 py-4 outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
          {props.label}
        </span>
        {!unavailable && props.trend && (
          <Trend
            cur={props.trend.cur}
            prev={props.trend.prev}
            upIsGood={props.trend.upIsGood}
            rangeLabel={props.trend.rangeLabel}
          />
        )}
      </div>

      <div className="flex items-baseline gap-2">
        {props.isLoading ? (
          <span className="block h-8 w-24 animate-pulse rounded bg-slate-100" />
        ) : unavailable ? (
          <span className="text-3xl font-bold leading-none text-slate-400">
            —
          </span>
        ) : (
          <span
            className="text-3xl font-bold leading-none tracking-[-0.03em] tabular-nums text-slate-900"
            title={exact(props.value)}
          >
            {compact(props.value)}
          </span>
        )}
        {!unavailable && props.badge && (
          <span className="shrink-0 rounded-full border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-micro font-semibold tabular-nums text-slate-600">
            {props.badge}
          </span>
        )}
      </div>

      {!unavailable &&
        (props.bar ? (
          <ProportionBar
            value={props.bar.value}
            total={props.bar.total}
            label={props.bar.label}
            color={props.color}
          />
        ) : (
          <div className="-mx-1">
            <LineChart
              sparkline
              points={props.points}
              color={props.color}
              height={40}
            />
          </div>
        ))}

      <span className="truncate text-micro font-normal text-slate-500">
        {unavailable ? "Currently unavailable" : props.footer}
      </span>

      <ArrowUpRight
        size={13}
        strokeWidth={2.5}
        aria-hidden="true"
        className="absolute right-3.5 bottom-3.5 text-slate-300 opacity-0 transition-opacity duration-150 group-hover:opacity-100"
      />
    </Link>
  );
};

export const MiniStat = (props: {
  label: string;
  value: number;
  to?: string;
  icon?: LucideIcon;
  tone?: "default" | "positive" | "critical" | "warning";
}) => {
  const Icon = props.icon;
  const tone = props.tone ?? "default";

  const content = (
    <>
      <div className="flex items-center gap-1.5">
        {Icon && (
          <Icon
            size={12}
            strokeWidth={2.3}
            className="shrink-0 text-slate-500"
          />
        )}
        <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {props.label}
        </span>
      </div>
      <span
        className={twMerge(
          "text-lg font-bold leading-6 tabular-nums",
          tone === "positive" && "text-emerald-700",
          tone === "critical" && "text-red-700",
          tone === "warning" && "text-amber-700",
          tone === "default" && "text-slate-800",
        )}
        title={exact(props.value)}
      >
        {compact(props.value)}
      </span>
    </>
  );

  const className =
    "flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5";

  if (!props.to) return <div className={className}>{content}</div>;

  return (
    <Link
      to={props.to}
      className={twMerge(
        className,
        "outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-white focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {content}
    </Link>
  );
};

export const MiniStatGrid = (props: { children: React.ReactNode }) => (
  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
    {props.children}
  </div>
);

export const EmptyHint = (props: { children: React.ReactNode }) => (
  <div className="flex min-h-24 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center">
    <p className="text-xs font-normal text-slate-500">{props.children}</p>
  </div>
);
