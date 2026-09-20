import { Panel } from "@/components/Dashboard/components";
import { useAuditSummary, useAuditTop } from "@/components/Dashboard/queries";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { compact } from "@/components/Dashboard/utils";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Library, Terminal, User } from "lucide-react";
import Leaderboard, { fromResources } from "./Leaderboard";

const ChangeLeaders = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAuditSummary(periodMinutes, QUERY_PRIORITY.high);
  const users = useAuditTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const sessions = useAuditTop("session", periodMinutes);

  const kinds = Object.entries(summary.data?.totalByResourceKind ?? {})
    .map(([label, value]) => ({ label, value: n(value) }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 10);

  return (
    <Panel
      icon={Library}
      title="Configuration leaders"
      description={`Who changed the Cluster in the last ${rangeLabel}, and what they touched`}
      to="/visibility/auditlogs"
      toLabel="Audit logs"
    >
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          <Leaderboard
            title="Operators"
            icon={User}
            items={fromResources(
              (users.data?.items ?? []).map((item: any) => ({
                resource: item.user,
                count: item.count,
              })),
            )}
            totalCount={n(users.data?.totalCount)}
            totalOther={n(users.data?.totalOther)}
            to="/core/users"
            isLoading={users.isLoading}
            unit="calls"
            emptyLabel={`Nobody changed the Cluster in the last ${rangeLabel}.`}
          />

          <Leaderboard
            title="Sessions"
            icon={Terminal}
            items={fromResources(
              (sessions.data?.items ?? []).map((item: any) => ({
                resource: item.session,
                count: item.count,
              })),
            )}
            totalCount={n(sessions.data?.totalCount)}
            totalOther={n(sessions.data?.totalOther)}
            to="/core/sessions"
            isLoading={sessions.isLoading}
            unit="calls"
            emptyLabel={`No Session issued an API call in the last ${rangeLabel}.`}
          />
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                What changed
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(summary.data?.totalNumber))} calls
              </span>
            </div>
            <CompositionBar
              segments={[
                {
                  label: "Create",
                  value: n(summary.data?.totalCreate),
                  color: STATUS_COLORS.good,
                },
                { label: "Update", value: n(summary.data?.totalUpdate) },
                {
                  label: "Delete",
                  value: n(summary.data?.totalDelete),
                  color: STATUS_COLORS.critical,
                },
                { label: "Other", value: n(summary.data?.totalOther) },
              ]}
              total={n(summary.data?.totalNumber)}
            />
          </div>

          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                Resource kinds touched
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(summary.data?.totalResource))} resources
              </span>
            </div>
            <CompositionBar
              segments={kinds}
              total={n(summary.data?.totalNumber)}
            />
          </div>
        </div>
      </div>
    </Panel>
  );
};

export default ChangeLeaders;
