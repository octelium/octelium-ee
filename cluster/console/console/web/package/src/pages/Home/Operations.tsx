import SeriesChart from "@/components/Charts/SeriesChart";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Boxes, LaptopMinimal, Library, Terminal, User } from "lucide-react";
import { MiniStat, Panel, toPoints } from "./components";
import { useAuditDataPoint, useAuditSummary } from "./queries";
import { compact } from "./utils";

const Operations = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAuditSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.normal,
  );
  const previous = useAuditSummary(
    periodMinutes,
    "previous",
    QUERY_PRIORITY.low,
  );
  const dataPoints = useAuditDataPoint(periodMinutes, QUERY_PRIORITY.low);

  const data = summary.data;
  const changes = n(data?.totalNumber);
  const changesPrev = n(previous.data?.totalNumber);

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
            label="Resources"
            value={n(data?.totalResource)}
            icon={Boxes}
          />
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
        </div>

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
