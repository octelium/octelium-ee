import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { HeartPulse } from "lucide-react";
import { Link } from "react-router-dom";
import { Panel } from "@/components/Dashboard/components";
import { StatusBadge, SubsystemGrid } from "@/components/Dashboard/health";
import { useClusterHealth } from "@/components/Dashboard/queries";
import { compact } from "@/components/Dashboard/utils";

const Health = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const health = useClusterHealth(periodMinutes, QUERY_PRIORITY.high);
  const data = health.data;

  const subsystems = data?.subsystems ?? [];
  const components = (data?.components ?? []).slice(0, 6);
  const regions = data?.regions ?? [];

  return (
    <Panel
      icon={HeartPulse}
      title="Cluster health"
      description={`Subsystem health derived from the Cluster's own signals over the last ${rangeLabel}`}
      actions={data ? <StatusBadge status={data.status} /> : undefined}
    >
      {health.isLoading && !data ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2, 3, 4, 5].map((index) => (
            <div
              key={index}
              className="h-[86px] animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          <SubsystemGrid subsystems={subsystems} />

          {(components.length > 0 || regions.length > 0) && (
            <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
              {components.length > 0 && (
                <div className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    Noisiest components
                  </span>

                  {components.map((item) => (
                    <Link
                      key={`${item.component?.namespace}/${item.component?.type}`}
                      to={`/visibility/componentlogs?level=ERROR&component.type=${item.component?.type ?? ""}`}
                      className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2 outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400"
                    >
                      <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                        {item.component?.type}
                        <span className="ml-1.5 text-micro font-normal text-slate-500">
                          {item.component?.namespace}
                        </span>
                      </span>
                      {n(item.totalWarn) > 0 && (
                        <span className="shrink-0 text-micro font-semibold tabular-nums text-amber-700">
                          {compact(n(item.totalWarn))} warn
                        </span>
                      )}
                      <span className="shrink-0 text-micro font-semibold tabular-nums text-red-700">
                        {compact(
                          n(item.totalError) +
                            n(item.totalPanic) +
                            n(item.totalFatal),
                        )}{" "}
                        err
                      </span>
                    </Link>
                  ))}
                </div>
              )}

              {regions.length > 0 && (
                <div className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    Regions
                  </span>

                  {regions.map((region) => (
                    <div
                      key={region.regionRef?.uid}
                      className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2"
                    >
                      <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                        {region.regionRef?.name}
                        {region.version && (
                          <span className="ml-1.5 text-micro font-normal text-slate-500">
                            {region.version}
                          </span>
                        )}
                      </span>
                      <span className="shrink-0 text-micro font-normal tabular-nums text-slate-500">
                        {n(region.totalGateway)} gateways
                      </span>
                      <StatusBadge status={region.status} />
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </Panel>
  );
};

export default Health;
