import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
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
  CheckCircle2,
  CircleDashed,
  GitBranch,
  Hammer,
  Layers,
  LayoutTemplate,
  UserRound,
} from "lucide-react";
import { useCordiumCreated, useCordiumTotals } from "./queries";

const Templates = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.normal);
  const created = useCordiumCreated(periodMinutes, QUERY_PRIORITY.normal);

  const data = totals.data?.cordium?.template;
  const fresh = created.data?.cordium?.template;
  const total = n(data?.totalNumber);
  const builds = n(data?.totalBuild);
  const canceled = n(data?.totalBuildCanceled);

  return (
    <Panel
      icon={LayoutTemplate}
      title="Templates & builds"
      description={`How the Templates are built into Workspace images · deltas cover the last ${rangeLabel}`}
      to="/cordium/templates"
      toLabel="All Templates"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Templates"
            value={total}
            icon={LayoutTemplate}
            to="/cordium/templates"
          />
          <MiniStat
            label="Ready"
            value={n(data?.totalWithReadyBuild)}
            icon={CheckCircle2}
            tone="positive"
            to="/cordium/templates?hasReadyBuild=true"
          />
          <MiniStat
            label="Building"
            value={n(data?.totalWithRunningBuild)}
            icon={Hammer}
            to="/cordium/templates?isBuilding=true"
          />
          <MiniStat
            label="Never built"
            value={n(data?.totalNeverBuilt)}
            icon={CircleDashed}
            tone={n(data?.totalNeverBuilt) > 0 ? "warning" : "default"}
          />
          <MiniStat
            label="Spaces"
            value={n(data?.totalSpace)}
            icon={Layers}
            to="/cordium/spaces"
          />
          <MiniStat
            label="Git providers"
            value={n(data?.totalGitProvider)}
            icon={GitBranch}
            to="/cordium/gitproviders"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Build outcomes"
            note={`${compact(builds)} builds recorded`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Ready",
                  value: n(data?.totalBuildReady),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Running",
                  value: n(data?.totalBuildRunning),
                  color: seriesColor(1),
                },
                {
                  label: "Failed",
                  value: Math.max(0, n(data?.totalBuildFailed) - canceled),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "Canceled",
                  value: canceled,
                  color: STATUS_COLORS.warning,
                },
              ]}
              total={builds}
            />
          </Breakdown>

          <Breakdown
            title="Template readiness"
            note={`${compact(total)} Templates`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Ready build",
                  value: n(data?.totalWithReadyBuild),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Never built",
                  value: n(data?.totalNeverBuilt),
                  color: STATUS_COLORS.warning,
                },
              ]}
              total={total}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Created · ${rangeLabel}`}
            value={n(fresh?.totalNumber)}
            prev={n(fresh?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={LayoutTemplate}
            to="/cordium/templates"
          />
          <DeltaStat
            label="Builds"
            value={n(fresh?.totalBuild)}
            prev={n(fresh?.previous?.totalBuild)}
            rangeLabel={rangeLabel}
            icon={Hammer}
          />
          <DeltaStat
            label="Failed builds"
            value={n(fresh?.totalBuildFailed)}
            prev={n(fresh?.previous?.totalBuildFailed)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={Hammer}
          />
          <DeltaStat
            label="Ready"
            value={n(fresh?.totalWithReadyBuild)}
            prev={n(fresh?.previous?.totalWithReadyBuild)}
            rangeLabel={rangeLabel}
            icon={CheckCircle2}
            to="/cordium/templates?hasReadyBuild=true"
          />
          <DeltaStat
            label="Owners"
            value={n(fresh?.totalUser)}
            prev={n(fresh?.previous?.totalUser)}
            rangeLabel={rangeLabel}
            icon={UserRound}
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Templates;
