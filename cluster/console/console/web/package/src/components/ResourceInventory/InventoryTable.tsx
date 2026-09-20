import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";

export type Segment = { label: string; value: number; color?: string };

export const CompositionBar = (props: {
  segments: Segment[];
  total: number;
}) => {
  useChartColorScheme();
  const shown = props.segments.filter((segment) => segment.value > 0);
  if (shown.length === 0 || props.total === 0) {
    return <span className="text-micro font-normal text-slate-500">—</span>;
  }

  const covered = shown.reduce((sum, segment) => sum + segment.value, 0);
  const rest = Math.max(0, props.total - covered);

  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex h-1.5 w-full gap-0.5 overflow-hidden rounded-full">
        {shown.map((segment, index) => (
          <span
            key={segment.label}
            className="h-full rounded-full"
            style={{
              width: `${(segment.value / props.total) * 100}%`,
              backgroundColor: segment.color ?? seriesColor(index),
            }}
          />
        ))}
        {rest > 0 && (
          <span
            className="h-full rounded-full bg-slate-200"
            style={{ width: `${(rest / props.total) * 100}%` }}
          />
        )}
      </div>

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        {shown.map((segment, index) => (
          <span
            key={segment.label}
            className="inline-flex items-center gap-1.5 text-micro font-normal text-slate-600"
          >
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: segment.color ?? seriesColor(index) }}
            />
            {segment.label}
            <span className="font-semibold tabular-nums text-slate-800">
              {segment.value.toLocaleString()}
            </span>
          </span>
        ))}
        {rest > 0 && (
          <span className="inline-flex items-center gap-1.5 text-micro font-normal text-slate-500">
            <span className="h-2 w-2 shrink-0 rounded-full bg-slate-300" />
            Other
            <span className="font-semibold tabular-nums text-slate-700">
              {rest.toLocaleString()}
            </span>
          </span>
        )}
      </div>
    </div>
  );
};
