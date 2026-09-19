import SeriesChart from "@/components/Charts/SeriesChart";
import TopList from "@/components/TopList";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity as ActivityIcon,
  Boxes,
  PanelTop,
  ShieldCheck,
  ShieldX,
  User,
} from "lucide-react";
import {
  EmptyHint,
  MiniStat,
  MiniStatGrid,
  Panel,
  subtractPoints,
  toPoints,
} from "./components";
import { useAccessDataPoint, useAccessSummary, useAccessTop } from "./queries";
import { compact } from "./utils";

const Activity = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAccessSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.critical,
  );
  const allPoints = useAccessDataPoint(
    periodMinutes,
    "all",
    QUERY_PRIORITY.critical,
  );
  const deniedDataPoints = useAccessDataPoint(
    periodMinutes,
    "denied",
    QUERY_PRIORITY.high,
  );

  const topUsers = useAccessTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const topServices = useAccessTop(
    "service",
    periodMinutes,
    QUERY_PRIORITY.normal,
  );
  const topPolicies = useAccessTop("policy", periodMinutes);
  const topSessions = useAccessTop("session", periodMinutes);

  const total = n(summary.data?.totalNumber);
  const allowed = n(summary.data?.totalAllowed);
  const denied = n(summary.data?.totalDenied);

  const totalPoints = toPoints(allPoints.data?.datapoints);
  const deniedPoints = toPoints(deniedDataPoints.data?.datapoints);
  const allowedPoints = subtractPoints(totalPoints, deniedPoints);

  const lists = [
    {
      title: "Top Users",
      to: "/visibility/accesslogs",
      items: (topUsers.data?.items ?? []).map((item: any) => ({
        resource: item.user,
        count: item.count,
      })),
    },
    {
      title: "Top Services",
      to: "/visibility/accesslogs",
      items: (topServices.data?.items ?? []).map((item: any) => ({
        resource: item.service,
        count: item.count,
      })),
    },
    {
      title: "Top Policies",
      to: "/visibility/accesslogs",
      items: (topPolicies.data?.items ?? []).map((item: any) => ({
        resource: item.policy,
        count: item.count,
      })),
    },
    {
      title: "Top Sessions",
      to: "/visibility/accesslogs",
      items: (topSessions.data?.items ?? []).map((item: any) => ({
        resource: item.session,
        count: item.count,
      })),
    },
  ].filter(
    (list) => list.items.length > 0 && list.items.every((x) => x.resource),
  );

  return (
    <Panel
      icon={ActivityIcon}
      title="Access activity"
      description={`Authorization decisions across every Service in the last ${rangeLabel}`}
      to="/visibility/accesslogs"
      toLabel="Access logs"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Requests"
            value={total}
            icon={ActivityIcon}
            to="/visibility/accesslogs"
          />
          <MiniStat
            label="Allowed"
            value={allowed}
            tone="positive"
            icon={ShieldCheck}
            to="/visibility/accesslogs?status=ALLOWED"
          />
          <MiniStat
            label="Denied"
            value={denied}
            tone={denied > 0 ? "critical" : "default"}
            icon={ShieldX}
            to="/visibility/accesslogs?status=DENIED"
          />
          <MiniStat
            label="Users"
            value={n(summary.data?.totalUser)}
            icon={User}
            to="/core/users"
          />
          <MiniStat
            label="Services"
            value={n(summary.data?.totalService)}
            icon={PanelTop}
            to="/core/services"
          />
          <MiniStat
            label="Namespaces"
            value={n(summary.data?.totalNamespace)}
            icon={Boxes}
            to="/core/namespaces"
          />
        </MiniStatGrid>

        {total > 0 && (
          <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-2.5">
            <div className="flex h-2 flex-1 overflow-hidden rounded-full bg-slate-200">
              <span
                className="h-full transition-[width] duration-300"
                style={{
                  width: `${pct(allowed, total)}%`,
                  backgroundColor: STATUS_COLORS.good,
                }}
              />
              <span
                className="h-full transition-[width] duration-300"
                style={{
                  width: `${pct(denied, total)}%`,
                  backgroundColor: STATUS_COLORS.critical,
                }}
              />
            </div>
            <span className="shrink-0 text-micro font-semibold tabular-nums text-slate-600">
              {pct(allowed, total)}% allowed
            </span>
            <span className="shrink-0 text-micro font-normal tabular-nums text-slate-500">
              {compact(total)} decisions
            </span>
          </div>
        )}

        <SeriesChart
          height={300}
          stacked
          colors={[STATUS_COLORS.good, STATUS_COLORS.critical]}
          series={[
            { name: "Allowed", points: allowedPoints },
            { name: "Denied", points: deniedPoints },
          ]}
          emptyLabel={`No authorization decisions in the last ${rangeLabel}`}
        />

        {lists.length > 0 ? (
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
            {lists.map((list) => (
              <TopList
                key={list.title}
                title={list.title}
                to={list.to}
                items={list.items}
              />
            ))}
          </div>
        ) : (
          !topUsers.isLoading && (
            <EmptyHint>
              No identities, Services or Policies were exercised in the last{" "}
              {rangeLabel}.
            </EmptyHint>
          )
        )}
      </div>
    </Panel>
  );
};

export default Activity;
