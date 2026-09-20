import { compact, exact } from "@/components/Dashboard/utils";
import {
  getResourcePath,
  printResourceNameWithDisplay,
  Resource,
} from "@/utils/pb";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { ArrowUpRight, LucideIcon } from "lucide-react";
import * as React from "react";
import { Link, useLocation } from "react-router-dom";
import { twMerge } from "tailwind-merge";

export type LeaderboardItem = {
  id: string;
  name: string;
  sub?: string;
  picURL?: string;
  count: number;
  to?: string;
  accent?: string;
  trailing?: React.ReactNode;
};

export const fromResource = (
  resource: Resource | undefined,
  count: number,
): LeaderboardItem | undefined => {
  const md = resource?.metadata;
  if (!md) return undefined;

  return {
    id: md.uid || md.name,
    name: printResourceNameWithDisplay(resource!),
    sub: resource!.kind,
    picURL: md.picURL || undefined,
    count,
    to: getResourcePath(resource!),
  };
};

export const fromResources = (
  items: { resource?: Resource; count: number | bigint }[] | undefined,
): LeaderboardItem[] =>
  (items ?? [])
    .map((item) => fromResource(item.resource, Number(item.count)))
    .filter((item): item is LeaderboardItem => !!item);

const Row = (props: {
  item: LeaderboardItem;
  index: number;
  peak: number;
  returnTo: string;
  unit: string;
}) => {
  const { item } = props;
  const share = props.peak === 0 ? 0 : (item.count / props.peak) * 100;
  const color = item.accent ?? seriesColor(0);

  const content = (
    <>
      <span
        className={twMerge(
          "flex h-6 w-6 shrink-0 items-center justify-center rounded-md border text-micro font-semibold tabular-nums",
          props.index < 3
            ? "border-slate-300 bg-slate-100 text-slate-700"
            : "border-slate-200 bg-white text-slate-500",
        )}
      >
        {props.index + 1}
      </span>

      {item.picURL && (
        <img
          src={item.picURL}
          alt=""
          loading="lazy"
          className="h-8 w-8 shrink-0 rounded-lg border border-slate-200 bg-white object-cover"
        />
      )}

      <span className="flex min-w-0 flex-[2] flex-col justify-center gap-0.5">
        <span className="truncate text-body font-semibold text-slate-800">
          {item.name}
        </span>
        {item.sub && (
          <span className="truncate text-micro font-normal text-slate-500">
            {item.sub}
          </span>
        )}
      </span>

      <span className="hidden min-w-0 flex-1 items-center md:flex">
        <span className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100">
          <span
            className="block h-full rounded-full transition-[width] duration-300"
            style={{
              width: `${Math.max(share, item.count > 0 ? 3 : 0)}%`,
              backgroundColor: color,
            }}
          />
        </span>
      </span>

      {item.trailing}

      <span className="flex shrink-0 flex-col items-end">
        <span
          className="text-sm font-bold tabular-nums text-slate-900"
          title={exact(item.count)}
        >
          {compact(item.count)}
        </span>
        <span className="text-micro font-normal text-slate-500">
          {props.unit}
        </span>
      </span>

      {item.to && (
        <ArrowUpRight
          size={13}
          strokeWidth={2.5}
          aria-hidden="true"
          className="shrink-0 text-slate-300 transition-colors duration-150 group-hover:text-slate-700"
        />
      )}
    </>
  );

  const className =
    "group flex min-h-12 min-w-0 items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2";

  if (!item.to) return <div className={className}>{content}</div>;

  return (
    <Link
      to={item.to}
      state={{ returnTo: props.returnTo }}
      preventScrollReset
      className={twMerge(
        className,
        "outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {content}
    </Link>
  );
};

const Leaderboard = (props: {
  title: string;
  icon: LucideIcon;
  items: LeaderboardItem[];
  totalCount?: number;
  totalOther?: number;
  to?: string;
  isLoading?: boolean;
  emptyLabel: string;
  unit?: string;
}) => {
  const Icon = props.icon;
  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;
  useChartColorScheme();

  const unit = props.unit ?? "events";
  const shown = props.items.reduce((sum, item) => sum + item.count, 0);
  const other = props.totalOther ?? 0;
  const overall = shown + other;
  const coverage = overall === 0 ? 0 : Math.round((shown / overall) * 100);
  const peak = props.items.reduce((max, item) => Math.max(max, item.count), 0);

  return (
    <section className="flex min-w-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50">
      <header className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-b border-slate-200 bg-white px-3.5 py-2.5">
        <span className="flex min-w-0 items-center gap-2">
          <Icon
            size={13}
            strokeWidth={2.3}
            className="shrink-0 text-slate-500"
          />
          <span className="truncate text-body font-semibold text-slate-800">
            {props.title}
          </span>
        </span>

        {props.totalCount !== undefined && props.totalCount > 0 && (
          <span className="shrink-0 rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold tabular-nums text-slate-600">
            top {props.items.length} of {compact(props.totalCount)}
          </span>
        )}
      </header>

      {props.isLoading ? (
        <div className="flex flex-col gap-1.5 p-3">
          {[0, 1, 2, 3, 4].map((index) => (
            <div
              key={index}
              className="h-12 animate-pulse rounded-lg border border-slate-200 bg-white"
            />
          ))}
        </div>
      ) : props.items.length === 0 ? (
        <div className="flex min-h-24 items-center justify-center px-4 py-6 text-center">
          <p className="text-xs font-normal text-slate-500">
            {props.emptyLabel}
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-1.5 p-3">
          {props.items.map((item, index) => (
            <Row
              key={item.id}
              item={item}
              index={index}
              peak={peak}
              returnTo={returnTo}
              unit={unit}
            />
          ))}
        </div>
      )}

      {props.items.length > 0 && overall > 0 && (
        <footer className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-t border-slate-200 bg-white px-3.5 py-2.5">
          <span className="flex min-w-0 items-center gap-2">
            <span className="h-1.5 w-20 shrink-0 overflow-hidden rounded-full bg-slate-100">
              <span
                className="block h-full rounded-full bg-slate-700"
                style={{ width: `${coverage}%` }}
              />
            </span>
            <span className="truncate text-micro font-normal text-slate-500">
              {coverage}% of {compact(overall)} {unit}
              {other > 0 && ` · ${compact(other)} in the long tail`}
            </span>
          </span>

          {props.to && (
            <Link
              to={props.to}
              className="shrink-0 rounded px-1.5 py-0.5 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:bg-slate-50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              Explore
            </Link>
          )}
        </footer>
      )}
    </section>
  );
};

export default Leaderboard;
