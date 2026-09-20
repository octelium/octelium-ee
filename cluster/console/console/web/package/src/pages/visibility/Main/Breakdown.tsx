import SeriesChart from "@/components/Charts/SeriesChart";
import { EmptyHint, Panel, toPoints } from "@/components/Dashboard/components";
import { compact, exact } from "@/components/Dashboard/utils";
import { seriesColors, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { SegmentedControl, Select } from "@mantine/core";
import { ChartSpline } from "lucide-react";
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import {
  BREAKDOWN_SERIES,
  DIMENSIONS,
  StreamKey,
  STREAMS,
  useBreakdown,
} from "./queries";

const isStream = (value: string | null): value is StreamKey =>
  STREAMS.some((item) => item.key === value);

const useBreakdownParams = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const streamParam = searchParams.get("stream");
  const stream: StreamKey = isStream(streamParam) ? streamParam : "access";

  const dimensions = DIMENSIONS[stream];
  const byParam = searchParams.get("by");
  const dimension =
    dimensions.find((item) => item.value === byParam) ?? dimensions[0];

  const patch = (next: { stream?: StreamKey; by?: string }) => {
    const params = new URLSearchParams(searchParams);

    if (next.stream !== undefined) {
      if (next.stream === "access") params.delete("stream");
      else params.set("stream", next.stream);
      params.delete("by");
    }

    if (next.by !== undefined) {
      const fallback = DIMENSIONS[next.stream ?? stream][0].value;
      if (next.by === fallback) params.delete("by");
      else params.set("by", next.by);
    }

    setSearchParams(params, { replace: true, preventScrollReset: true });
  };

  return { stream, dimension, dimensions, patch };
};

const Legend = (props: {
  entries: { name: string; total: number; color: string }[];
  total: number;
}) => {
  if (props.entries.length === 0) return null;

  return (
    <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2 xl:grid-cols-4">
      {props.entries.map((entry) => {
        const share =
          props.total === 0 ? 0 : Math.round((entry.total / props.total) * 100);

        return (
          <div
            key={entry.name}
            className="flex min-w-0 items-center gap-2.5 rounded-lg border border-slate-200 bg-white px-3 py-2"
          >
            <span
              aria-hidden="true"
              className="h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: entry.color }}
            />
            <span className="min-w-0 flex-1 truncate text-micro font-semibold text-slate-700">
              {entry.name}
            </span>
            <span
              className="shrink-0 text-micro font-bold tabular-nums text-slate-900"
              title={exact(entry.total)}
            >
              {compact(entry.total)}
            </span>
            <span className="w-8 shrink-0 text-right text-micro font-normal tabular-nums text-slate-500">
              {share}%
            </span>
          </div>
        );
      })}
    </div>
  );
};

const Breakdown = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const { stream, dimension, dimensions, patch } = useBreakdownParams();
  const query = useBreakdown(stream, dimension, periodMinutes);

  const palette = seriesColors();

  const series = React.useMemo(
    () =>
      (query.data?.series ?? []).map((item, index) => ({
        name: item.displayName || item.key || "Unknown",
        total: n(item.total),
        color: palette[index % palette.length],
        points: toPoints(item.datapoints),
      })),
    [query.data, palette],
  );

  const overall = (query.data?.datapoints ?? []).reduce(
    (sum, point) => sum + n(point.count),
    0,
  );

  const meta = STREAMS.find((item) => item.key === stream)!;

  return (
    <Panel
      icon={ChartSpline}
      title="Breakdown explorer"
      description={`Split the last ${rangeLabel} of a log stream by any dimension · top ${BREAKDOWN_SERIES} series`}
      to={meta.to}
      toLabel={`${meta.label} logs`}
      actions={
        <Select
          size="xs"
          aria-label="Break down by"
          allowDeselect={false}
          checkIconPosition="right"
          value={dimension.value}
          onChange={(value) => value && patch({ by: value })}
          data={dimensions.map((item) => ({
            value: item.value,
            label: `by ${item.label}`,
          }))}
          className="w-40"
        />
      }
    >
      <div className="flex flex-col gap-4">
        <SegmentedControl
          size="xs"
          fullWidth
          value={stream}
          onChange={(value) => patch({ stream: value as StreamKey })}
          data={STREAMS.map((item) => {
            const Icon = item.icon;
            return {
              value: item.key,
              label: (
                <span className="flex items-center justify-center gap-1.5 px-1 py-0.5">
                  <Icon size={12} strokeWidth={2.3} />
                  <span>{item.label}</span>
                </span>
              ),
            };
          })}
        />

        {series.length === 0 && !query.isLoading ? (
          <EmptyHint>
            The {meta.label.toLowerCase()} stream produced no entries grouped by{" "}
            {dimension.label.toLowerCase()} in the last {rangeLabel}.
          </EmptyHint>
        ) : (
          <>
            <SeriesChart
              height={320}
              stacked
              colors={series.map((item) => item.color)}
              series={series.map((item) => ({
                name: item.name,
                points: item.points,
              }))}
              emptyLabel={`No entries in the last ${rangeLabel}`}
            />

            <Legend
              entries={series.map((item) => ({
                name: item.name,
                total: item.total,
                color: item.color,
              }))}
              total={overall}
            />
          </>
        )}
      </div>
    </Panel>
  );
};

export default Breakdown;
