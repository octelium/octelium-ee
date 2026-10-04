import { Workspace } from "@/apis/cordiumv1/cordiumv1";
import { Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import TimeAgo from "@/components/TimeAgo";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Hammer,
} from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { getFailureLabel } from "../utils";
import {
  QUEUE_ITEMS,
  useBuildingTemplates,
  useCordiumTotals,
  useFailedWorkspaces,
} from "./queries";

const SHOWN = 8;

const failedAt = (item: Workspace) =>
  item.status?.run?.stoppedAt ??
  item.status?.currentStateSetAt ??
  item.metadata?.createdAt;

const byFailure = (a: Workspace, b: Workspace) =>
  (failedAt(b)?.seconds ?? 0) - (failedAt(a)?.seconds ?? 0);

const Row = (props: { item: Workspace; returnTo: string }) => {
  const { item } = props;
  const md = item.metadata!;
  const status = item.status;

  return (
    <Link
      to={`/cordium/workspaces/${md.name}`}
      state={{ returnTo: props.returnTo }}
      preventScrollReset
      className="group flex min-w-0 items-center gap-3 rounded-lg border border-red-200 bg-red-50/40 px-3 py-2.5 outline-none transition-[border-color,box-shadow] duration-150 hover:border-red-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <span
        className="h-8 w-1 shrink-0 rounded-full bg-red-500"
        aria-hidden="true"
      />

      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-body font-semibold text-slate-800">
          {md.displayName || md.name}
        </span>
        <span className="truncate text-micro font-normal text-slate-500">
          {status?.userRef?.name ?? "Unknown user"}
          {status?.spaceRef?.name && <> · {status.spaceRef.name}</>}
          {status?.templateRef?.name && <> · {status.templateRef.name}</>}
          {status?.run?.failure?.message && (
            <> · {status.run.failure.message}</>
          )}
        </span>
      </span>

      <span className="hidden shrink-0 rounded-full border border-red-200 bg-white px-1.5 py-px text-micro font-semibold text-red-700 sm:inline">
        {getFailureLabel(status?.run?.failure)}
      </span>

      <span className="hidden shrink-0 items-center gap-1 text-micro font-normal text-slate-500 md:inline-flex">
        <TimeAgo rfc3339={failedAt(item)} />
      </span>

      <ChevronRight
        size={13}
        strokeWidth={2.4}
        aria-hidden="true"
        className="shrink-0 text-slate-400 transition-colors duration-150 group-hover:text-slate-800"
      />
    </Link>
  );
};

const Attention = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;

  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.critical);
  const failed = useFailedWorkspaces(QUERY_PRIORITY.critical);
  const building = useBuildingTemplates(QUERY_PRIORITY.low);

  const all = [...(failed.data?.items ?? [])].sort(byFailure);
  const items = all.slice(0, SHOWN);

  const totalFailed = n(totals.data?.cordium?.workspace?.totalFailed);
  const totalBuilding = n(
    totals.data?.cordium?.template?.totalWithRunningBuild,
  );
  const buildingTemplates = building.data?.items ?? [];

  return (
    <Panel
      icon={AlertTriangle}
      title="Needs attention"
      description={
        totalFailed > 0
          ? `${compact(totalFailed)} Workspace${totalFailed === 1 ? "" : "s"} whose last run failed · most recent failures first`
          : "No Workspace run is currently failing"
      }
      to="/cordium/workspaces?isFailed=true"
      toLabel="All failures"
      actions={
        totalBuilding > 0 ? (
          <Link
            to="/cordium/templates?isBuilding=true"
            className="inline-flex shrink-0 items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-micro font-semibold text-blue-700 outline-none hover:border-blue-300 focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            <Hammer size={10} strokeWidth={2.5} />
            {compact(totalBuilding)} building
          </Link>
        ) : undefined
      }
    >
      <div className="flex flex-col gap-3">
        {failed.isLoading ? (
          <div className="flex flex-col gap-1.5">
            {[0, 1, 2, 3].map((index) => (
              <div
                key={index}
                className="h-[54px] animate-pulse rounded-lg border border-slate-200 bg-slate-50"
              />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="flex min-h-24 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-emerald-200 bg-emerald-50/50 px-6 py-6 text-center">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-emerald-200 bg-white text-emerald-600">
              <CheckCircle2 size={17} strokeWidth={2.2} />
            </span>
            <span className="text-body font-semibold text-slate-800">
              Every Workspace is healthy
            </span>
            <span className="text-micro font-normal text-slate-500">
              None of the Workspaces failed during its last run.
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {items.map((item) => (
              <Row key={item.metadata?.uid} item={item} returnTo={returnTo} />
            ))}

            {totalFailed > items.length && (
              <Link
                to="/cordium/workspaces?isFailed=true"
                className="mt-1 flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-slate-200 px-3 py-2 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50/70 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                {compact(totalFailed - items.length)} more failed
                {all.length >= QUEUE_ITEMS &&
                  ` · ranked from the ${QUEUE_ITEMS} most recently created`}
              </Link>
            )}
          </div>
        )}

        {buildingTemplates.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5 border-t border-slate-100 pt-3">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Building now
            </span>
            {buildingTemplates.slice(0, SHOWN).map((tmpl) => (
              <Link
                key={tmpl.metadata?.uid}
                to={`/cordium/templates/${tmpl.metadata?.name}`}
                state={{ returnTo }}
                preventScrollReset
                className="inline-flex items-center gap-1 rounded-md border border-blue-200/80 bg-blue-50/70 px-2 py-1 text-micro font-semibold text-blue-700 outline-none hover:border-blue-300 hover:bg-white focus-visible:ring-2 focus-visible:ring-blue-500/40"
              >
                <Hammer size={10} strokeWidth={2.5} />
                {tmpl.metadata?.name}
              </Link>
            ))}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Attention;
