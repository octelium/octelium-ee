import SeriesChart from "@/components/Charts/SeriesChart";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Boxes,
  FilePen,
  FilePlus2,
  FileX2,
  LaptopMinimal,
  Library,
  Terminal,
  User,
} from "lucide-react";
import { MiniStat, Panel, toPoints } from "./components";
import { useAuditDataPoint, useAuditSummary } from "./queries";
import { compact } from "./utils";

const Operations = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAuditSummary(periodMinutes, QUERY_PRIORITY.normal);
  const dataPoints = useAuditDataPoint(periodMinutes, QUERY_PRIORITY.low);

  const data = summary.data;
  const changes = n(data?.totalNumber);
  const changesPrev = n(data?.previous?.totalNumber);
  const kinds = Object.entries(data?.totalByResourceKind ?? {})
    .map(([label, value]) => ({ label, value: n(value) }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 8);

  return (
    <Panel
      icon={Library}
      title="Configuration changes"
      description={`${compact(changes)} audited changes in the last ${rangeLabel} · ${compact(changesPrev)} in the previous one`}
      to="/visibility/auditlogs"
      toLabel="Audit logs"
    >
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          <MiniStat
            label="Created"
            value={n(data?.totalCreate)}
            tone="positive"
            icon={FilePlus2}
          />
          <MiniStat
            label="Updated"
            value={n(data?.totalUpdate)}
            icon={FilePen}
          />
          <MiniStat
            label="Deleted"
            value={n(data?.totalDelete)}
            tone={n(data?.totalDelete) > 0 ? "warning" : "default"}
            icon={FileX2}
          />
          <MiniStat
            label="Resources"
            value={n(data?.totalResource)}
            icon={Boxes}
          />
        </div>

        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          <MiniStat label="Users" value={n(data?.totalUser)} icon={User} />
          <MiniStat
            label="Sessions"
            value={n(data?.totalSession)}
            icon={Terminal}
          />
          <MiniStat
            label="Devices"
            value={n(data?.totalDevice)}
            icon={LaptopMinimal}
          />
          <MiniStat
            label="Other calls"
            value={n(data?.totalOther)}
            icon={Library}
          />
        </div>

        {kinds.length > 0 && (
          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Changes by resource kind
            </span>
            <CompositionBar segments={kinds} total={changes} />
          </div>
        )}

        <SeriesChart
          height={240}
          colors={[seriesColor(5)]}
          series={[
            {
              name: "Changes",
              points: toPoints(dataPoints.data?.datapoints),
            },
          ]}
          variant="bar"
          emptyLabel={`No configuration changes in the last ${rangeLabel}`}
        />
      </div>
    </Panel>
  );
};

export default Operations;
