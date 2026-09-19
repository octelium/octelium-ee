import {
  ClusterConfig_Status,
  ClusterConfig_Status_UpgradeRequest,
  ClusterConfig_Status_UpgradeRequest_State,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import TimeAgo from "@/components/TimeAgo";
import { motion } from "framer-motion";
import {
  ChevronDown,
  History,
  Loader2,
  PackageCheck,
  TriangleAlert,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { UpgradeStateBadge, VersionChips } from "./components";
import { isUpgradeActive } from "./queries";

const PREVIEW = 4;

const STATE_DOT: Record<number, string> = {
  [ClusterConfig_Status_UpgradeRequest_State.SUCCESS]: "bg-emerald-500",
  [ClusterConfig_Status_UpgradeRequest_State.FAILED]: "bg-red-500",
  [ClusterConfig_Status_UpgradeRequest_State.UPGRADING]: "bg-blue-500",
  [ClusterConfig_Status_UpgradeRequest_State.UPGRADE_REQUESTED]: "bg-amber-500",
  [ClusterConfig_Status_UpgradeRequest_State.STATE_UNSET]: "bg-slate-400",
};

const took = (from?: Timestamp, to?: Timestamp) => {
  if (!from || !to) return undefined;
  const seconds = to.seconds - from.seconds;
  if (seconds <= 0) return undefined;
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${(seconds / 3600).toFixed(1)}h`;
};

const ActiveBanner = (props: {
  request: ClusterConfig_Status_UpgradeRequest;
}) => (
  <div className="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50/60 px-3.5 py-3">
    <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-white text-blue-700">
      <Loader2 size={14} className="animate-spin" />
    </span>
    <div className="min-w-0 flex-1">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-body font-semibold text-slate-800">
          An upgrade is rolling out
        </span>
        <UpgradeStateBadge state={props.request.state} />
      </div>
      <div className="mt-1 text-micro font-normal text-slate-600">
        Started <TimeAgo rfc3339={props.request.createdAt} /> · the affected
        workloads restart as the rollout progresses.
      </div>
      <div className="mt-2">
        <VersionChips request={props.request.request} />
      </div>
    </div>
  </div>
);

const FailedBanner = (props: {
  request: ClusterConfig_Status_UpgradeRequest;
}) => (
  <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50/60 px-3.5 py-3">
    <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-red-200 bg-white text-red-600">
      <TriangleAlert size={14} strokeWidth={2.3} />
    </span>
    <div className="min-w-0 flex-1">
      <div className="text-body font-semibold text-slate-800">
        The last upgrade did not complete
      </div>
      <div className="mt-1 text-micro font-normal text-slate-600">
        It failed <TimeAgo rfc3339={props.request.doneAt} />. Review the
        component logs of the affected packages before retrying.
      </div>
      <div className="mt-2">
        <VersionChips request={props.request.request} />
      </div>
    </div>
  </div>
);

const Entry = (props: {
  item: ClusterConfig_Status_UpgradeRequest;
  isLast: boolean;
}) => {
  const { item } = props;
  const elapsed = took(item.createdAt, item.doneAt);

  return (
    <li className="relative flex gap-3 pb-3 last:pb-0">
      {!props.isLast && (
        <span
          aria-hidden="true"
          className="absolute top-4 bottom-0 left-[5px] w-px bg-slate-200"
        />
      )}

      <span
        aria-hidden="true"
        className={twMerge(
          "relative mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full ring-2 ring-white",
          STATE_DOT[item.state] ??
            STATE_DOT[ClusterConfig_Status_UpgradeRequest_State.STATE_UNSET],
        )}
      />

      <div className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <UpgradeStateBadge state={item.state} />
          <VersionChips request={item.request} />
        </div>
        <span className="shrink-0 text-micro font-normal text-slate-500">
          <TimeAgo rfc3339={item.doneAt ?? item.createdAt} />
          {elapsed && <span className="ml-1.5">· took {elapsed}</span>}
        </span>
      </div>
    </li>
  );
};

const Rollout = (props: { status: ClusterConfig_Status }) => {
  const { status } = props;
  const [expanded, setExpanded] = React.useState(false);

  const current = status.upgradeRequest;
  const history = status.lastUpgradeRequests;
  const active = isUpgradeActive(current?.state) ? current : undefined;
  const failed =
    !active &&
    current?.state === ClusterConfig_Status_UpgradeRequest_State.FAILED
      ? current
      : undefined;

  const last = history.find(
    (item) => item.state === ClusterConfig_Status_UpgradeRequest_State.SUCCESS,
  );
  const visible = expanded ? history : history.slice(0, PREVIEW);

  return (
    <Panel
      icon={History}
      title="Upgrade rollout"
      description="The current rollout and everything the Cluster has applied before"
      actions={active ? <UpgradeStateBadge state={active.state} /> : undefined}
    >
      <div className="flex flex-col gap-4">
        {active && <ActiveBanner request={active} />}
        {failed && <FailedBanner request={failed} />}

        <MiniStatGrid>
          <MiniStat
            label="Successful"
            value={Number(status.totalSuccessfulUpgrades)}
            tone="positive"
            icon={PackageCheck}
          />
          <MiniStat
            label="Failed"
            value={Number(status.totalFailedUpgrades)}
            tone={
              Number(status.totalFailedUpgrades) > 0 ? "critical" : "default"
            }
            icon={TriangleAlert}
          />
          <MiniStat label="Recorded" value={history.length} icon={History} />
          <div className="flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
            <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Last successful
            </span>
            <span className="truncate text-body font-semibold text-slate-800">
              {last ? <TimeAgo rfc3339={last.doneAt ?? last.createdAt} /> : "—"}
            </span>
          </div>
        </MiniStatGrid>

        {history.length === 0 ? (
          <div className="flex min-h-20 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 text-center">
            <p className="text-xs font-normal text-slate-500">
              No upgrade has been applied to this Cluster yet.
            </p>
          </div>
        ) : (
          <div className="rounded-xl border border-slate-200 bg-white px-3.5 py-3">
            <ul className="flex flex-col">
              {visible.map((item, index) => (
                <Entry
                  key={`${item.createdAt?.seconds ?? index}-${index}`}
                  item={item}
                  isLast={index === visible.length - 1}
                />
              ))}
            </ul>

            {history.length > PREVIEW && (
              <button
                type="button"
                onClick={() => setExpanded((value) => !value)}
                aria-expanded={expanded}
                className="mt-1 flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg border-t border-slate-100 px-3 py-2 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:bg-slate-50 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                {expanded ? "Show less" : `Show all ${history.length}`}
                <motion.span
                  animate={{ rotate: expanded ? 180 : 0 }}
                  transition={{ duration: 0.18, ease: "easeInOut" }}
                  className="flex items-center"
                >
                  <ChevronDown size={12} strokeWidth={2.5} />
                </motion.span>
              </button>
            )}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Rollout;
