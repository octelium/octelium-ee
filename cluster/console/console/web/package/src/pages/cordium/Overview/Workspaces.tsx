import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  Meter,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  CirclePlay,
  Container,
  Globe,
  Hammer,
  Layers,
  LayoutTemplate,
  Share2,
  Sparkles,
  Terminal,
  UserRound,
} from "lucide-react";
import { useCordiumCreated, useCordiumTotals } from "./queries";

const Workspaces = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCordiumCreated(periodMinutes, QUERY_PRIORITY.high);

  const data = totals.data?.cordium?.workspace;
  const fresh = created.data?.cordium?.workspace;
  const total = n(data?.totalNumber);
  const starting =
    n(data?.totalInitializing) +
    n(data?.totalPullingImage) +
    n(data?.totalBuildingImage) +
    n(data?.totalStartingRuntime) +
    n(data?.totalPreparing);

  return (
    <Panel
      icon={Container}
      title="Workspaces"
      description={`Where the Workspaces are in their lifecycle and who runs them · deltas cover the last ${rangeLabel}`}
      to="/cordium/workspaces"
      toLabel="All Workspaces"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Users"
            value={n(data?.totalUser)}
            icon={UserRound}
            to="/core/users"
          />
          <MiniStat
            label="Sessions"
            value={n(data?.totalSession)}
            icon={Terminal}
            to="/core/sessions"
          />
          <MiniStat
            label="Spaces"
            value={n(data?.totalSpace)}
            icon={Layers}
            to="/cordium/spaces"
          />
          <MiniStat
            label="Templates"
            value={n(data?.totalTemplate)}
            icon={LayoutTemplate}
            to="/cordium/templates"
          />
          <MiniStat
            label="Regions"
            value={n(data?.totalRegion)}
            icon={Globe}
            to="/cordium/regions"
          />
          <MiniStat
            label="Builds"
            value={n(data?.totalBuild)}
            icon={Hammer}
            to="/cordium/workspaces?isBuild=true"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown title="Lifecycle" note={`${compact(total)} all time`}>
            <CompositionBar
              segments={[
                {
                  label: "Running",
                  value: n(data?.totalRunning),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Starting",
                  value: starting,
                  color: seriesColor(0),
                },
                {
                  label: "Stopping",
                  value: n(data?.totalStopping),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Failed",
                  value: n(data?.totalFailed),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "Stopped",
                  value: Math.max(
                    0,
                    n(data?.totalStopped) - n(data?.totalFailed),
                  ),
                  color: "var(--color-slate-400)",
                },
              ]}
              total={total}
            />
          </Breakdown>

          <Breakdown
            title="Startup pipeline"
            note={`${compact(starting)} starting now`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Initializing",
                  value: n(data?.totalInitializing),
                },
                {
                  label: "Pulling image",
                  value: n(data?.totalPullingImage),
                },
                {
                  label: "Building image",
                  value: n(data?.totalBuildingImage),
                },
                {
                  label: "Starting runtime",
                  value: n(data?.totalStartingRuntime),
                },
                {
                  label: "Preparing",
                  value: n(data?.totalPreparing),
                },
              ]}
              total={starting}
            />
          </Breakdown>

          <Breakdown title="Space ownership">
            <CompositionBar
              segments={[
                {
                  label: "Personal",
                  value: n(data?.totalUserSpace),
                  color: seriesColor(0),
                },
                {
                  label: "Organization",
                  value: n(data?.totalOrganizationSpace),
                  color: seriesColor(2),
                },
              ]}
              total={total}
            />
          </Breakdown>

          <Breakdown title="Behaviour">
            <Meter
              value={n(data?.totalEphemeral)}
              total={total}
              label="ephemeral"
              color={seriesColor(3)}
            />
            <Meter
              value={n(data?.totalAutoStop)}
              total={total}
              label="auto-stop"
              color={seriesColor(4)}
            />
            <Meter
              value={n(data?.totalShared)}
              total={total}
              label="shared"
              color={seriesColor(1)}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Created · ${rangeLabel}`}
            value={n(fresh?.totalNumber)}
            prev={n(fresh?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Container}
            to="/cordium/workspaces"
          />
          <DeltaStat
            label="Running"
            value={n(fresh?.totalRunning)}
            prev={n(fresh?.previous?.totalRunning)}
            rangeLabel={rangeLabel}
            icon={CirclePlay}
            to="/cordium/workspaces?state=RUNNING"
          />
          <DeltaStat
            label="Failed"
            value={n(fresh?.totalFailed)}
            prev={n(fresh?.previous?.totalFailed)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={AlertTriangle}
            to="/cordium/workspaces?isFailed=true"
          />
          <DeltaStat
            label="Ephemeral"
            value={n(fresh?.totalEphemeral)}
            prev={n(fresh?.previous?.totalEphemeral)}
            rangeLabel={rangeLabel}
            icon={Sparkles}
            to="/cordium/workspaces?isEphemeral=true"
          />
          <DeltaStat
            label="Shared"
            value={n(fresh?.totalShared)}
            prev={n(fresh?.previous?.totalShared)}
            rangeLabel={rangeLabel}
            icon={Share2}
            to="/cordium/workspaces?isShared=true"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Workspaces;
