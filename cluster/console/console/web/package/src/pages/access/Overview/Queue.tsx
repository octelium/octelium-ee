import { Request, Request_Spec_Urgency } from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import { Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import TimeAgo from "@/components/TimeAgo";
import { getUrgencyColor, getUrgencyLabel } from "@/pages/access/Request/utils";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  CalendarClock,
  CheckCircle2,
  ChevronRight,
  Inbox,
  Timer,
} from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { useAccessTotals, usePendingRequests, QUEUE_ITEMS } from "./queries";

const SHOWN = 8;

const isPast = (deadline?: Timestamp) =>
  !!deadline && Timestamp.toDate(deadline).getTime() < Date.now();

const resourceOf = (item: Request): ObjectReference | undefined => {
  const resource = item.spec?.resource?.type;
  if (resource?.oneofKind === "serviceRef") return resource.serviceRef;
  if (resource?.oneofKind === "catalog") return resource.catalog.catalogRef;
  return undefined;
};

const subjectOf = (item: Request): ObjectReference | undefined => {
  const subject = item.spec?.subject?.type;
  return subject?.oneofKind === "userRef" ? subject.userRef : undefined;
};

const waitingSince = (item: Request) =>
  item.status?.approvalStartAt ?? item.metadata?.createdAt;

const rank = (item: Request) =>
  (isPast(item.spec?.deadline) ? 0 : 1) * 100 -
  (item.spec?.urgency ?? Request_Spec_Urgency.URGENCY_UNSET);

const byPriority = (a: Request, b: Request) => {
  const byRank = rank(a) - rank(b);
  if (byRank !== 0) return byRank;
  return (waitingSince(a)?.seconds ?? 0) - (waitingSince(b)?.seconds ?? 0);
};

const Row = (props: { item: Request; returnTo: string }) => {
  const { item } = props;
  const md = item.metadata!;
  const overdue = isPast(item.spec?.deadline);
  const urgency = item.spec?.urgency ?? Request_Spec_Urgency.URGENCY_UNSET;
  const resource = resourceOf(item);
  const subject = subjectOf(item);

  return (
    <Link
      to={`/access/requests/${md.name}`}
      state={{ returnTo: props.returnTo }}
      preventScrollReset
      className={twMerge(
        "group flex min-w-0 items-center gap-3 rounded-lg border bg-white px-3 py-2.5 outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
        overdue
          ? "border-red-200 bg-red-50/40 hover:border-red-300"
          : "border-slate-200 hover:border-slate-300",
      )}
    >
      <span
        className="h-8 w-1 shrink-0 rounded-full"
        style={{ backgroundColor: getUrgencyColor(urgency) }}
        aria-hidden="true"
      />

      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-body font-semibold text-slate-800">
          {md.displayName || md.name}
        </span>
        <span className="truncate text-micro font-normal text-slate-500">
          {item.status?.userRef?.name ?? "Unknown requester"}
          {subject?.name && subject.name !== item.status?.userRef?.name && (
            <> → {subject.name}</>
          )}
          {resource?.name && <> · {resource.name}</>}
          {item.status?.policyRef?.name && (
            <> · via {item.status.policyRef.name}</>
          )}
        </span>
      </span>

      <span
        className="hidden shrink-0 rounded-full border px-1.5 py-px text-micro font-semibold sm:inline"
        style={{
          color: getUrgencyColor(urgency),
          borderColor: `color-mix(in oklab, ${getUrgencyColor(urgency)} 34%, transparent)`,
        }}
      >
        {getUrgencyLabel(urgency)}
      </span>

      <span className="hidden shrink-0 items-center gap-1 text-micro font-normal text-slate-500 md:inline-flex">
        <Timer size={11} strokeWidth={2.3} />
        <TimeAgo rfc3339={waitingSince(item)} />
      </span>

      {item.spec?.deadline && (
        <span
          className={twMerge(
            "hidden shrink-0 items-center gap-1 text-micro font-semibold lg:inline-flex",
            overdue ? "text-red-700" : "text-slate-500",
          )}
        >
          <CalendarClock size={11} strokeWidth={2.3} />
          <TimeAgo rfc3339={item.spec.deadline} />
        </span>
      )}

      <ChevronRight
        size={13}
        strokeWidth={2.4}
        aria-hidden="true"
        className="shrink-0 text-slate-400 transition-colors duration-150 group-hover:text-slate-800"
      />
    </Link>
  );
};

const Queue = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;

  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const pending = usePendingRequests(QUERY_PRIORITY.critical);

  const all = [...(pending.data?.items ?? [])].sort(byPriority);
  const items = all.slice(0, SHOWN);

  const totalPending = n(totals.data?.access?.request?.totalPending);
  const overdue = all.filter((item) => isPast(item.spec?.deadline)).length;

  return (
    <Panel
      icon={Inbox}
      title="Review queue"
      description={
        totalPending > 0
          ? `${compact(totalPending)} Request${totalPending === 1 ? "" : "s"} waiting for a decision · overdue and most urgent first`
          : "Every access Request has been decided"
      }
      to="/access/requests?state=PENDING"
      toLabel="Open queue"
      actions={
        overdue > 0 ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-red-200 bg-red-50 px-2 py-0.5 text-micro font-semibold text-red-700">
            <CalendarClock size={10} strokeWidth={2.5} />
            {overdue} past deadline
          </span>
        ) : undefined
      }
    >
      {pending.isLoading ? (
        <div className="flex flex-col gap-1.5">
          {[0, 1, 2, 3, 4].map((index) => (
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
            The review queue is empty
          </span>
          <span className="text-micro font-normal text-slate-500">
            No access Request is waiting for a reviewer right now.
          </span>
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {items.map((item) => (
            <Row key={item.metadata?.uid} item={item} returnTo={returnTo} />
          ))}

          {totalPending > items.length && (
            <Link
              to="/access/requests?state=PENDING"
              className="mt-1 flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-slate-200 px-3 py-2 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50/70 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              {compact(totalPending - items.length)} more pending
              {all.length >= QUEUE_ITEMS &&
                ` · ranked from the ${QUEUE_ITEMS} that have waited longest`}
            </Link>
          )}
        </div>
      )}
    </Panel>
  );
};

export default Queue;
