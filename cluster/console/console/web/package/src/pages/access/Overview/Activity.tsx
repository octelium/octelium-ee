import {
  GetRequestDataPointRequest_GroupBy,
  ListRequestTopRequest_Dimension,
} from "@/apis/visibilityv1/access/vaccessv1";
import SeriesChart from "@/components/Charts/SeriesChart";
import {
  Breakdown,
  MiniStat,
  MiniStatGrid,
  Panel,
  toPoints,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import {
  CHART_INK,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity as ActivityIcon,
  CircleCheck,
  CircleX,
  Gauge,
  Inbox,
  Layers,
  Plug,
  Timer,
  UserRound,
  UsersRound,
} from "lucide-react";
import { formatSeconds } from "../common";
import {
  useAccessCreated,
  useRequestDataPoint,
  useRequestTop,
} from "./queries";
import { TopRefList } from "./shared";

const STATE_SERIES: { key: string; name: string; color: () => string }[] = [
  { key: "APPROVED", name: "Approved", color: () => STATUS_COLORS.good },
  { key: "PENDING", name: "Pending", color: () => STATUS_COLORS.warning },
  { key: "REJECTED", name: "Rejected", color: () => STATUS_COLORS.critical },
  { key: "REVOKED", name: "Revoked", color: () => STATUS_COLORS.serious },
  { key: "EXPIRED", name: "Expired", color: () => CHART_INK.muted },
  { key: "CANCELLED", name: "Cancelled", color: () => CHART_INK.axis },
];

const Activity = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.critical);
  const byState = useRequestDataPoint(
    periodMinutes,
    GetRequestDataPointRequest_GroupBy.STATE,
    QUERY_PRIORITY.high,
  );
  const topRequesters = useRequestTop(
    ListRequestTopRequest_Dimension.REQUESTER,
    periodMinutes,
  );
  const topResources = useRequestTop(
    ListRequestTopRequest_Dimension.RESOURCE,
    periodMinutes,
  );

  const fresh = created.data?.access?.request;
  const total = n(fresh?.totalNumber);
  const decision = fresh?.decisionTime;

  const series = STATE_SERIES.map((entry) => ({
    ...entry,
    points: toPoints(
      byState.data?.series.find((item) => item.key === entry.key)?.datapoints,
    ),
  })).filter((entry) => entry.points.some((point) => point.value > 0));

  return (
    <Panel
      icon={ActivityIcon}
      title="Request activity"
      description={`Requests created in the last ${rangeLabel}, by their current outcome`}
      to="/access/requests"
      toLabel="All requests"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Created"
            value={total}
            icon={Inbox}
            to="/access/requests"
          />
          <MiniStat
            label="Granted"
            value={n(fresh?.totalGranted)}
            tone="positive"
            icon={CircleCheck}
            to="/access/requests?state=APPROVED"
          />
          <MiniStat
            label="Rejected"
            value={n(fresh?.totalRejected)}
            tone={n(fresh?.totalRejected) > 0 ? "critical" : "default"}
            icon={CircleX}
            to="/access/requests?state=REJECTED"
          />
          <MiniStat
            label="Requesters"
            value={n(fresh?.totalUser)}
            icon={UserRound}
          />
          <MiniStat
            label="On behalf"
            value={n(fresh?.totalOnBehalf)}
            icon={UsersRound}
            to="/access/requests?isOnBehalf=true"
          />
          <MiniStat
            label="From integrations"
            value={n(fresh?.totalFromIntegration)}
            icon={Plug}
            to="/access/requests?originType=INTEGRATION"
          />
        </MiniStatGrid>

        <SeriesChart
          height={260}
          stacked
          variant="bar"
          colors={series.map((entry) => entry.color())}
          series={series.map((entry) => ({
            name: entry.name,
            points: entry.points,
          }))}
          emptyLabel={`No access request was created in the last ${rangeLabel}`}
        />

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="How requests were decided"
            note={`${compact(total)} requests`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Auto-approved",
                  value: n(fresh?.totalAutoApproved),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Reviewed",
                  value: n(fresh?.totalReviewed),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Denied by a rule",
                  value: n(fresh?.totalDenied),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "No matching rule",
                  value: n(fresh?.totalUnmatched),
                  color: STATUS_COLORS.serious,
                },
              ]}
              total={total}
            />
          </Breakdown>

          <Breakdown
            title="Time to decision"
            note={
              decision
                ? `${compact(n(decision.count))} reviewed`
                : "no reviewed decision"
            }
          >
            <div className="grid grid-cols-3 gap-2">
              {[
                { label: "Median", value: decision?.p50Seconds, icon: Timer },
                { label: "p90", value: decision?.p90Seconds, icon: Gauge },
                { label: "Slowest", value: decision?.maxSeconds, icon: Timer },
              ].map((stat) => (
                <div
                  key={stat.label}
                  className="flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2"
                >
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    {stat.label}
                  </span>
                  <span className="text-lg font-bold leading-6 tabular-nums text-slate-800">
                    {stat.value !== undefined ? formatSeconds(stat.value) : "—"}
                  </span>
                </div>
              ))}
            </div>
          </Breakdown>
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <TopRefList
            title="Top requesters"
            icon={UserRound}
            avatars
            items={topRequesters.data?.items}
            totalCount={n(topRequesters.data?.totalCount)}
            totalOther={n(topRequesters.data?.totalOther)}
            unit="requests"
            isLoading={topRequesters.isLoading}
            emptyLabel={`No request in the last ${rangeLabel}`}
          />
          <TopRefList
            title="Most requested"
            icon={Layers}
            items={topResources.data?.items}
            totalCount={n(topResources.data?.totalCount)}
            totalOther={n(topResources.data?.totalOther)}
            unit="requests"
            isLoading={topResources.isLoading}
            emptyLabel={`No request in the last ${rangeLabel}`}
          />
        </div>
      </div>
    </Panel>
  );
};

export default Activity;
