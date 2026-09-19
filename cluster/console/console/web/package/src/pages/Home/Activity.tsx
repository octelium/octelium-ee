import {
  AccessLog_Entry_Common_Reason_Type,
  AccessLog_Entry_Common_Status,
} from "@/apis/corev1/corev1";
import SeriesChart from "@/components/Charts/SeriesChart";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TopList from "@/components/TopList";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity as ActivityIcon,
  ArrowDownToLine,
  ArrowUpFromLine,
  Boxes,
  PanelTop,
  ShieldCheck,
  ShieldX,
  Timer,
  User,
} from "lucide-react";
import {
  EmptyHint,
  MiniStat,
  MiniStatGrid,
  Panel,
  seriesPoints,
} from "./components";
import {
  useAccessDataPoint,
  useAccessDenyReasons,
  useAccessSummary,
  useAccessTop,
} from "./queries";
import { compact, exact } from "./utils";

const formatMillis = (value?: number) => {
  if (value === undefined || !Number.isFinite(value)) return "—";
  if (value >= 1000) return `${(value / 1000).toFixed(2)} s`;
  return `${value.toFixed(value < 10 ? 1 : 0)} ms`;
};

const formatBytes = (value: number) => {
  const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  let current = value;
  let index = 0;
  while (current >= 1024 && index < units.length - 1) {
    current = current / 1024;
    index++;
  }
  return `${current.toFixed(current < 10 && index > 0 ? 1 : 0)} ${units[index]}`;
};

const denyReasonLabel = (reason: number) =>
  (AccessLog_Entry_Common_Reason_Type[reason] ?? "Unknown")
    .toLowerCase()
    .replace(/_/g, " ")
    .replace(/^./, (char) => char.toUpperCase());

const Activity = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAccessSummary(periodMinutes, QUERY_PRIORITY.critical);
  const dataPoints = useAccessDataPoint(periodMinutes, QUERY_PRIORITY.critical);
  const denyReasons = useAccessDenyReasons(periodMinutes, QUERY_PRIORITY.low);

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

  const allowedPoints = seriesPoints(
    dataPoints.data?.series,
    AccessLog_Entry_Common_Status[AccessLog_Entry_Common_Status.ALLOWED],
  );
  const deniedPoints = seriesPoints(
    dataPoints.data?.series,
    AccessLog_Entry_Common_Status[AccessLog_Entry_Common_Status.DENIED],
  );

  const latency = summary.data?.latency;
  const modes = Object.entries(summary.data?.totalByMode ?? {})
    .map(([label, value]) => ({ label, value: n(value) }))
    .sort((a, b) => b.value - a.value);
  const reasons = denyReasons.data?.items ?? [];

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

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <div className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                <Timer size={12} strokeWidth={2.3} />
                Latency &amp; volume
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(latency?.count))} measured
              </span>
            </div>

            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              {[
                { label: "p50", value: latency?.p50Milliseconds },
                { label: "p95", value: latency?.p95Milliseconds },
                { label: "p99", value: latency?.p99Milliseconds },
              ].map((stat) => (
                <div
                  key={stat.label}
                  className="flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
                >
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    {stat.label}
                  </span>
                  <span className="text-lg font-bold leading-6 tabular-nums text-slate-800">
                    {formatMillis(stat.value)}
                  </span>
                </div>
              ))}
            </div>

            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span className="inline-flex items-center gap-1">
                <ArrowDownToLine size={11} strokeWidth={2.3} />
                Received
                <span className="font-semibold tabular-nums text-slate-700">
                  {formatBytes(n(summary.data?.totalBytesReceived))}
                </span>
              </span>
              <span className="inline-flex items-center gap-1">
                <ArrowUpFromLine size={11} strokeWidth={2.3} />
                Sent
                <span className="font-semibold tabular-nums text-slate-700">
                  {formatBytes(n(summary.data?.totalBytesSent))}
                </span>
              </span>
              <span>
                Public
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(summary.data?.totalPublic))}
                </span>
              </span>
              <span>
                Anonymous
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(summary.data?.totalAnonymous))}
                </span>
              </span>
            </div>
          </div>

          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Traffic by Service mode
            </span>
            <CompositionBar segments={modes} total={total} />
          </div>
        </div>

        {reasons.length > 0 && (
          <div className="flex flex-col gap-2.5 rounded-lg border border-red-200 bg-red-50/40 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-red-700">
                <ShieldX size={12} strokeWidth={2.3} />
                Why requests were denied
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(denied)} denials
              </span>
            </div>

            <div className="flex flex-col gap-1.5">
              {reasons.map((item) => (
                <div
                  key={`${item.reason}-${item.policyRef?.uid ?? ""}`}
                  className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2"
                >
                  <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                    {denyReasonLabel(item.reason)}
                    {item.policyRef?.name && (
                      <span className="ml-1.5 text-micro font-normal text-slate-500">
                        via {item.policyRef.name}
                      </span>
                    )}
                  </span>
                  <span className="h-1.5 w-24 shrink-0 overflow-hidden rounded-full bg-slate-100">
                    <span
                      className="block h-full rounded-full"
                      style={{
                        width: `${pct(n(item.count), denied)}%`,
                        backgroundColor: STATUS_COLORS.critical,
                      }}
                    />
                  </span>
                  <span
                    className="shrink-0 text-xs font-semibold tabular-nums text-slate-800"
                    title={exact(n(item.count))}
                  >
                    {compact(n(item.count))}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

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
