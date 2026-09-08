import { seriesColor, STATUS_COLORS } from "@/utils/charts/palette";
import { SegmentedControl } from "@mantine/core";
import { ChevronRight, LucideIcon } from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

export type Segment = { label: string; value: number; color?: string };
export type AttentionItem = { label: string; value: number; color?: string };

export type InventoryRow = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  total: number;
  segments: Segment[];
  attention: AttentionItem[];
  isLoading: boolean;
};

export const CompositionBar = (props: { segments: Segment[]; total: number }) => {
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

type SortMode = "name" | "total";

export const InventoryTable = (props: {
  title: string;
  unitLabel?: string;
  rows: InventoryRow[];
}) => {
  const [sort, setSort] = React.useState<SortMode>("total");
  const unitLabel = props.unitLabel ?? "resources";

  const sorted = [...props.rows].sort((a, b) =>
    sort === "total" ? b.total - a.total : a.label.localeCompare(b.label),
  );

  const grandTotal = props.rows.reduce((sum, row) => sum + row.total, 0);

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
        <div className="min-w-0">
          <h2 className="text-body font-semibold text-slate-800">
            {props.title}
          </h2>
          <p className="mt-0.5 text-micro font-normal text-slate-500">
            {grandTotal.toLocaleString()} {unitLabel} across{" "}
            {props.rows.length} kinds
          </p>
        </div>

        <SegmentedControl
          size="xs"
          value={sort}
          onChange={(value) => setSort(value as SortMode)}
          data={[
            { value: "total", label: "By count" },
            { value: "name", label: "By name" },
          ]}
        />
      </header>

      <div className="w-full overflow-x-auto">
        <table className="w-full min-w-[720px] border-collapse">
          <thead>
            <tr className="border-b border-slate-100 bg-slate-50/60">
              <th className="px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Kind
              </th>
              <th className="w-20 px-4 py-2 text-right text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Total
              </th>
              <th className="w-1/2 px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Composition
              </th>
              <th className="px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Needs attention
              </th>
              <th className="w-8 px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {sorted.map((row) => {
              const Icon = row.icon;
              return (
                <tr
                  key={row.kind}
                  className="group border-b border-slate-100 last:border-b-0 hover:bg-slate-50/70"
                >
                  <td className="px-4 py-3 align-middle">
                    <Link
                      to={row.to}
                      className="inline-flex items-center gap-2.5 outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
                    >
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                        <Icon size={13} strokeWidth={2.2} />
                      </span>
                      <span className="text-body font-semibold text-slate-800 group-hover:text-slate-900">
                        {row.label}
                      </span>
                    </Link>
                  </td>

                  <td className="px-4 py-3 text-right align-middle">
                    {row.isLoading ? (
                      <span className="ml-auto block h-4 w-10 animate-pulse rounded bg-slate-100" />
                    ) : (
                      <span
                        className={twMerge(
                          "text-sm font-semibold tabular-nums",
                          row.total === 0 ? "text-slate-500" : "text-slate-900",
                        )}
                      >
                        {row.total.toLocaleString()}
                      </span>
                    )}
                  </td>

                  <td className="w-1/2 px-4 py-3 align-middle">
                    <CompositionBar segments={row.segments} total={row.total} />
                  </td>

                  <td className="px-4 py-3 align-middle">
                    {row.attention.length === 0 ? (
                      <span className="text-micro font-normal text-slate-500">
                        None
                      </span>
                    ) : (
                      <span className="flex flex-wrap gap-1.5">
                        {row.attention.map((item) => (
                          <span
                            key={item.label}
                            className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-micro font-semibold"
                            style={{
                              borderColor: `${item.color ?? STATUS_COLORS.warning}55`,
                              color: item.color ?? STATUS_COLORS.critical,
                            }}
                          >
                            {item.label}
                            <span className="tabular-nums">
                              {item.value.toLocaleString()}
                            </span>
                          </span>
                        ))}
                      </span>
                    )}
                  </td>

                  <td className="px-4 py-3 align-middle">
                    <ChevronRight
                      size={13}
                      strokeWidth={2.4}
                      className="text-slate-500 transition-colors duration-150 group-hover:text-slate-800"
                    />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
};
