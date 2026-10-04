import { Breakdown, Meter, Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Cpu, HardDrive, LucideIcon, MemoryStick } from "lucide-react";
import { formatMegabytes, formatMillicores } from "../utils";
import { useCordiumTotals } from "./queries";

const CapacityTile = (props: {
  label: string;
  icon: LucideIcon;
  value: string;
  average?: string;
  isLoading: boolean;
}) => {
  const Icon = props.icon;

  return (
    <div className="flex min-w-0 flex-col gap-1.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
      <span className="inline-flex items-center gap-1.5">
        <Icon size={12} strokeWidth={2.3} className="shrink-0 text-slate-500" />
        <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {props.label}
        </span>
      </span>
      {props.isLoading ? (
        <span className="block h-7 w-24 animate-pulse rounded bg-slate-100" />
      ) : (
        <span className="text-2xl font-bold leading-none tracking-[-0.03em] tabular-nums text-slate-900">
          {props.value}
        </span>
      )}
      <span className="truncate text-micro font-normal text-slate-500">
        {props.average ? `${props.average} per active Workspace` : "—"}
      </span>
    </div>
  );
};

const Capacity = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  useChartColorScheme();

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.normal);
  const data = totals.data?.cordium?.workspace;

  const total = n(data?.totalNumber);
  const active =
    n(data?.totalInitializing) +
    n(data?.totalPullingImage) +
    n(data?.totalBuildingImage) +
    n(data?.totalStartingRuntime) +
    n(data?.totalPreparing) +
    n(data?.totalRunning) +
    n(data?.totalStopping);
  const cpu = n(data?.totalCPUMillicores);
  const memory = n(data?.totalMemoryMegabytes);
  const storage = n(data?.totalStorageMegabytes);
  const average = (value: number) =>
    active > 0 ? Math.round(value / active) : undefined;

  const avgCPU = average(cpu);
  const avgMemory = average(memory);
  const avgStorage = average(storage);

  return (
    <Panel
      icon={Cpu}
      title="Compute allocation"
      description="Effective limits that are currently allocated to the active (i.e. non-stopped) Workspaces"
    >
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
          <CapacityTile
            label="CPU"
            icon={Cpu}
            value={formatMillicores(cpu)}
            average={
              avgCPU !== undefined ? formatMillicores(avgCPU) : undefined
            }
            isLoading={totals.isLoading}
          />
          <CapacityTile
            label="Memory"
            icon={MemoryStick}
            value={formatMegabytes(memory)}
            average={
              avgMemory !== undefined ? formatMegabytes(avgMemory) : undefined
            }
            isLoading={totals.isLoading}
          />
          <CapacityTile
            label="Storage"
            icon={HardDrive}
            value={formatMegabytes(storage)}
            average={
              avgStorage !== undefined ? formatMegabytes(avgStorage) : undefined
            }
            isLoading={totals.isLoading}
          />
        </div>

        <Breakdown
          title="Active Workspaces"
          note={`${compact(active)} of ${compact(total)}`}
        >
          <Meter
            value={active}
            total={total}
            label="hold an allocation"
            color={seriesColor(0)}
          />
        </Breakdown>
      </div>
    </Panel>
  );
};

export default Capacity;
