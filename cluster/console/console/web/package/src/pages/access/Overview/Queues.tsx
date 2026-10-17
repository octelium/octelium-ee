import {
  Policy_Spec_Rule_Effect,
  Request,
  Request_Spec_Urgency,
} from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { EmptyHint, Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { twMerge } from "@/lib/utils";
import { getUrgencyColor, getUrgencyLabel } from "@/pages/access/Request/utils";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  CalendarClock,
  ChevronRight,
  Hourglass,
  KeyRound,
  Workflow,
} from "lucide-react";
import * as React from "react";
import { Link, useLocation } from "react-router-dom";
import { Avatar, formatSeconds } from "../common";
import { useAccessTotals, useLiveAccess, useOldestPending } from "./queries";
import { resourceIcon, resourceOf, subjectOf } from "./shared";

const seconds = (ts?: Timestamp) =>
  ts ? Timestamp.toDate(ts).getTime() / 1000 : undefined;

export const useNow = (intervalMs = 30_000) => {
  const [now, setNow] = React.useState(() => Date.now());
  React.useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(timer);
  }, [intervalMs]);
  return now / 1000;
};

const stepOf = (item: Request) => {
  const steps =
    item.status?.rule?.action?.type.oneofKind === "review"
      ? item.status.rule.action.type.review.steps
      : [];
  if (steps.length === 0) return undefined;
  const index = Math.min(item.status?.review?.currentStep ?? 0, steps.length - 1);
  return {
    name: steps[index]?.name || `step-${index + 1}`,
    index,
    count: steps.length,
  };
};

const RowLink = (props: {
  item: Request;
  children: React.ReactNode;
  className?: string;
}) => {
  const location = useLocation();
  return (
    <Link
      to={`/access/requests/${props.item.metadata?.name}`}
      state={{ returnTo: `${location.pathname}${location.search}` }}
      preventScrollReset
      className={twMerge(
        "group flex min-w-0 items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5 outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
        props.className,
      )}
    >
      {props.children}
      <ChevronRight
        size={13}
        strokeWidth={2.4}
        aria-hidden="true"
        className="shrink-0 text-slate-400 transition-colors duration-150 group-hover:text-slate-800"
      />
    </Link>
  );
};

export const PendingRow = (props: { item: Request; now: number }) => {
  const { item, now } = props;
  const resource = resourceOf(item);
  const subject = subjectOf(item);
  const requester = item.status?.userRef;
  const ResourceIcon = resourceIcon(resource);
  const urgency = item.spec?.urgency ?? Request_Spec_Urgency.URGENCY_UNSET;
  const since = seconds(item.status?.approvalStartAt ?? item.metadata?.createdAt);
  const waited = since ? now - since : 0;
  const deadline = seconds(item.spec?.deadline);
  const overdue = deadline !== undefined && deadline < now;
  const step = stepOf(item);

  return (
    <RowLink
      item={item}
      className={overdue ? "border-red-200 bg-red-50/40 hover:border-red-300" : undefined}
    >
      <span
        aria-hidden="true"
        className="h-9 w-1 shrink-0 rounded-full"
        style={{ backgroundColor: getUrgencyColor(urgency) }}
      />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex min-w-0 items-center gap-1.5">
          <ResourceIcon size={12} strokeWidth={2.3} className="shrink-0 text-slate-500" />
          <span className="truncate text-body font-semibold text-slate-800">
            {resource?.name ?? item.metadata?.name}
          </span>
        </span>
        <span className="truncate text-micro font-normal text-slate-500">
          {requester?.name ?? "Unknown"}
          {subject?.name && subject.name !== requester?.name && <> for {subject.name}</>}
          {step && (
            <>
              {" · "}
              {step.count > 1 ? `step ${step.index + 1}/${step.count} ` : ""}
              {step.name}
            </>
          )}
        </span>
      </span>
      {urgency >= Request_Spec_Urgency.HIGH && (
        <span
          className="hidden shrink-0 rounded-full border px-1.5 py-px text-micro font-semibold sm:inline"
          style={{
            color: getUrgencyColor(urgency),
            borderColor: `color-mix(in oklab, ${getUrgencyColor(urgency)} 34%, transparent)`,
          }}
        >
          {getUrgencyLabel(urgency)}
        </span>
      )}
      {overdue ? (
        <span className="hidden shrink-0 items-center gap-1 text-micro font-semibold text-red-700 md:inline-flex">
          <CalendarClock size={11} strokeWidth={2.3} />
          overdue
        </span>
      ) : null}
      <span
        className={twMerge(
          "shrink-0 text-micro font-semibold tabular-nums",
          waited > 86400
            ? "text-red-700"
            : waited > 3600
              ? "text-amber-700"
              : "text-slate-500",
        )}
        title="Waiting for"
      >
        {formatSeconds(waited)}
      </span>
    </RowLink>
  );
};

export const LiveRow = (props: { item: Request; now: number }) => {
  const { item, now } = props;
  const resource = resourceOf(item);
  const subject = subjectOf(item) ?? item.status?.userRef;
  const ResourceIcon = resourceIcon(resource);
  const start = seconds(item.status?.approvalEndAt ?? item.status?.state?.createdAt);
  const end = seconds(item.status?.accessEndsAt);
  const remaining = end !== undefined ? end - now : undefined;
  const total = start !== undefined && end !== undefined ? end - start : undefined;
  const share =
    total && remaining !== undefined ? Math.min(100, Math.max(0, (remaining / total) * 100)) : 100;
  const isAuto = item.status?.rule?.effect === Policy_Spec_Rule_Effect.AUTO_APPROVE;

  return (
    <RowLink item={item}>
      <Avatar name={subject?.name ?? "?"} size="md" />
      <span className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="truncate text-body font-semibold text-slate-800">
            {subject?.name ?? "Unknown"}
          </span>
          <ChevronRight size={11} strokeWidth={2.4} className="shrink-0 text-slate-400" />
          <ResourceIcon size={12} strokeWidth={2.3} className="shrink-0 text-slate-500" />
          <span className="truncate text-xs font-medium text-slate-700">
            {resource?.name}
          </span>
        </span>
        <span className="flex items-center gap-2">
          <span
            role="progressbar"
            aria-label="Time left"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(share)}
            className="h-1 min-w-0 flex-1 overflow-hidden rounded-full bg-slate-100"
          >
            <span
              className={twMerge(
                "block h-full rounded-full",
                remaining !== undefined && remaining < 3600 ? "bg-amber-500" : "bg-emerald-500",
              )}
              style={{ width: `${share}%` }}
            />
          </span>
          <span className="shrink-0 text-micro font-normal text-slate-500">
            {isAuto ? "auto-approved" : "reviewed"}
          </span>
        </span>
      </span>
      <span
        className={twMerge(
          "shrink-0 text-micro font-semibold tabular-nums",
          remaining !== undefined && remaining < 3600 ? "text-amber-700" : "text-slate-600",
        )}
      >
        {remaining === undefined
          ? "no end"
          : remaining <= 0
            ? "ending"
            : `${formatSeconds(remaining)} left`}
      </span>
    </RowLink>
  );
};

const Skeleton = () => (
  <div className="flex flex-col gap-1.5">
    {[0, 1, 2, 3].map((index) => (
      <div
        key={index}
        className="h-[56px] motion-safe:animate-pulse rounded-lg border border-slate-200 bg-slate-50"
      />
    ))}
  </div>
);

export const Pending = (props: { periodMinutes: number }) => {
  const now = useNow();
  const totals = useAccessTotals(props.periodMinutes, QUERY_PRIORITY.critical);
  const pending = useOldestPending();
  const total = n(totals.data?.access?.request?.totalPending);
  const items = pending.data?.items ?? [];

  return (
    <Panel
      icon={Hourglass}
      title="Waiting for a decision"
      description={
        total > 0
          ? `${compact(total)} pending · longest wait first`
          : "Nothing is waiting for a reviewer"
      }
      to="/access/requests?state=PENDING"
      toLabel="All pending"
    >
      {pending.isLoading ? (
        <Skeleton />
      ) : items.length === 0 ? (
        <EmptyHint>No request is waiting for a decision.</EmptyHint>
      ) : (
        <div className="flex flex-col gap-1.5">
          {items.map((item) => (
            <PendingRow key={item.metadata?.uid} item={item} now={now} />
          ))}
          {total > items.length && (
            <span className="flex items-center gap-1 px-1 text-micro font-normal text-slate-500">
              <Workflow size={11} strokeWidth={2.4} />
              {compact(total - items.length)} more waiting
            </span>
          )}
        </div>
      )}
    </Panel>
  );
};

export const LiveAccess = (props: { periodMinutes: number }) => {
  const now = useNow();
  const totals = useAccessTotals(props.periodMinutes, QUERY_PRIORITY.critical);
  const live = useLiveAccess();
  const total = n(totals.data?.access?.request?.totalActive);
  const items = live.data?.items ?? [];

  return (
    <Panel
      icon={KeyRound}
      title="Live access"
      description={
        total > 0
          ? `${compact(total)} grant${total === 1 ? "" : "s"} in effect · ending soonest first`
          : "Nobody holds just-in-time access right now"
      }
      to="/access/requests?isActive=true"
      toLabel="All live access"
    >
      {live.isLoading ? (
        <Skeleton />
      ) : items.length === 0 ? (
        <EmptyHint>No approved request currently grants access.</EmptyHint>
      ) : (
        <div className="flex flex-col gap-1.5">
          {items.map((item) => (
            <LiveRow key={item.metadata?.uid} item={item} now={now} />
          ))}
          {total > items.length && (
            <span className="px-1 text-micro font-normal text-slate-500">
              {compact(total - items.length)} more · revoke any of them from
              the request page
            </span>
          )}
        </div>
      )}
    </Panel>
  );
};
