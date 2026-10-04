import { StatTile } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { useCordiumCreated, useCordiumTotals } from "./queries";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCordiumCreated(periodMinutes, QUERY_PRIORITY.critical);

  const workspaces = totals.data?.cordium?.workspace;
  const templates = totals.data?.cordium?.template;
  const fresh = created.data?.cordium?.workspace;

  const total = n(workspaces?.totalNumber);
  const running = n(workspaces?.totalRunning);
  const starting =
    n(workspaces?.totalInitializing) +
    n(workspaces?.totalPullingImage) +
    n(workspaces?.totalBuildingImage) +
    n(workspaces?.totalStartingRuntime) +
    n(workspaces?.totalPreparing);

  return (
    <section
      aria-label="Cordium signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Running now"
        value={running}
        badge={total > 0 ? `${pct(running, total)}%` : undefined}
        footer={`${compact(starting)} starting · ${compact(n(workspaces?.totalStopping))} stopping`}
        bar={{ value: running, total, label: "of all Workspaces" }}
        color={STATUS_COLORS.good}
        to="/cordium/workspaces?state=RUNNING"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Starting up"
        value={starting}
        footer={`${compact(n(workspaces?.totalPullingImage))} pulling · ${compact(n(workspaces?.totalBuildingImage))} building images`}
        bar={{
          value: starting,
          total: starting + running,
          label: "of the active Workspaces",
        }}
        color={seriesColor(0)}
        to="/cordium/workspaces"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Failed"
        value={n(workspaces?.totalFailed)}
        footer={`${compact(n(workspaces?.totalStopped))} Workspaces stopped`}
        bar={{
          value: n(workspaces?.totalFailed),
          total,
          label: "of all Workspaces",
        }}
        color={STATUS_COLORS.critical}
        to="/cordium/workspaces?isFailed=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label={`Created · ${rangeLabel}`}
        value={n(fresh?.totalNumber)}
        trend={{
          cur: n(fresh?.totalNumber),
          prev: n(fresh?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        footer={`${compact(n(fresh?.totalUser))} users · ${compact(n(fresh?.totalTemplate))} Templates`}
        bar={{
          value: n(fresh?.totalRunning),
          total: n(fresh?.totalNumber),
          label: "running now",
        }}
        color={seriesColor(0)}
        to="/cordium/workspaces"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label="Templates ready"
        value={n(templates?.totalWithReadyBuild)}
        footer={`${compact(n(templates?.totalWithRunningBuild))} building · ${compact(n(templates?.totalNeverBuilt))} never built`}
        bar={{
          value: n(templates?.totalWithReadyBuild),
          total: n(templates?.totalNumber),
          label: "of all Templates",
        }}
        color={seriesColor(2)}
        to="/cordium/templates?hasReadyBuild=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Shared"
        value={n(workspaces?.totalShared)}
        footer={`${compact(n(workspaces?.totalSharedAll))} with everyone · ${compact(n(workspaces?.totalSharedMembers))} with members`}
        bar={{
          value: n(workspaces?.totalShared),
          total,
          label: "of all Workspaces",
        }}
        color={seriesColor(3)}
        to="/cordium/workspaces?isShared=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />
    </section>
  );
};

export default Signals;
