import { GetClusterHealthResponse_Subsystem_Type } from "@/apis/visibilityv1/visibilityv1";
import { EmptyHint, MiniStat, Panel } from "@/components/Dashboard/components";
import { StatusBadge, SubsystemGrid } from "@/components/Dashboard/health";
import { useClusterHealth } from "@/components/Dashboard/queries";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { DoorClosed, Globe, KeyRound } from "lucide-react";
import { useCoreTotals } from "./queries";

const SUBSYSTEMS = [
  GetClusterHealthResponse_Subsystem_Type.DATA_PLANE,
  GetClusterHealthResponse_Subsystem_Type.ENROLLMENT,
  GetClusterHealthResponse_Subsystem_Type.AUTHORIZATION,
];

const DataPlane = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const health = useClusterHealth(periodMinutes, QUERY_PRIORITY.low);

  const regions = health.data?.regions ?? [];

  return (
    <Panel
      icon={Globe}
      title="Data plane"
      description={`Regions, Gateways and the Cluster signals they produced over the last ${rangeLabel}`}
      to="/core/regions"
      toLabel="Regions"
    >
      <div className="flex flex-col gap-5">
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
          <MiniStat
            label="Regions"
            value={n(totals.data?.core?.region?.totalNumber)}
            icon={Globe}
            to="/core/regions"
          />
          <MiniStat
            label="Gateways"
            value={n(totals.data?.core?.gateway?.totalNumber)}
            icon={DoorClosed}
            to="/core/gateways"
          />
          <MiniStat
            label="Secrets"
            value={n(totals.data?.core?.secret?.totalNumber)}
            icon={KeyRound}
            to="/core/secrets"
          />
        </div>

        <SubsystemGrid
          subsystems={health.data?.subsystems ?? []}
          types={SUBSYSTEMS}
        />

        {regions.length > 0 ? (
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
        ) : (
          !health.isLoading && (
            <EmptyHint>
              The Cluster health could not be evaluated right now.
            </EmptyHint>
          )
        )}
      </div>
    </Panel>
  );
};

export default DataPlane;
