import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListSSHSessionRequest,
  ListSSHSessionResponse,
  SSHSession,
  SSHSession_State,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import {
  CheckCircle2,
  CircleHelp,
  Clock3,
  Radio,
  SquareTerminal,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardService from "../Card/CardService";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import {
  ResourceListItem,
  ResourceListLabel,
  ResourceListWrapper,
} from "../ResourceList";
import TimeAgo from "../TimeAgo";

dayjs.extend(relativeTime);
dayjs.extend(utc);

const StateBadge = ({ state }: { state: SSHSession_State }) => {
  const { label, className, icon: Icon } = match(state)
    .with(SSHSession_State.ONGOING, () => ({
      label: "Ongoing",
      className: "bg-emerald-50/80 text-emerald-700 border-emerald-200",
      icon: Radio,
    }))
    .with(SSHSession_State.COMPLETED, () => ({
      label: "Completed",
      className: "bg-blue-50/70 text-blue-700 border-blue-200",
      icon: CheckCircle2,
    }))
    .otherwise(() => ({
      label: "Unknown",
      className: "bg-slate-50 text-slate-500 border-slate-200",
      icon: CircleHelp,
    }));

  return (
    <span
      className={twMerge(
        "inline-flex h-6 items-center gap-1.5 rounded-full border px-2.5 text-[0.66rem] font-semibold",
        className,
      )}
    >
      {state === SSHSession_State.ONGOING && (
        <span className="relative flex h-1.5 w-1.5 shrink-0">
          <span className="absolute inline-flex h-full w-full motion-safe:animate-ping rounded-full bg-emerald-400 opacity-60" />
          <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
        </span>
      )}
      {state !== SSHSession_State.ONGOING && (
        <Icon size={11} strokeWidth={2.25} />
      )}
      {label}
    </span>
  );
};

export const SSHSessionC = (props: { item: SSHSession }) => {
  const { item } = props;

  const duration = item.startedAt
    ? dayjs(
        item.state === SSHSession_State.ONGOING || !item.endedAt
          ? new Date()
          : Timestamp.toDate(item.endedAt),
      ).from(Timestamp.toDate(item.startedAt), true)
    : null;
  const hasContext =
    item.sessionRef ||
    item.serviceRef ||
    item.userRef ||
    item.deviceRef ||
    item.namespaceRef;

  return (
    <div className="flex w-full min-w-0 flex-col">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
            <SquareTerminal size={16} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <span className="block text-[0.63rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              SSH recording
            </span>
            <span className="mt-0.5 block truncate text-[0.76rem] font-semibold text-slate-700">
              <CopyText value={item.id} truncate={32} />
            </span>
          </div>
        </div>
        <StateBadge state={item.state} />
      </div>

      <div className="mt-3 grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
        {item.startedAt && (
          <div className="flex min-w-0 items-center gap-2 rounded-lg border border-slate-200/80 bg-slate-50/60 px-2.5 py-1.5">
            <Clock3 size={12} className="shrink-0 text-slate-400" />
            <span className="min-w-0 text-[0.67rem] font-semibold text-slate-500">
              <span className="mr-1 font-bold text-slate-600">Started</span>
              <TimeAgo rfc3339={item.startedAt} />
            </span>
          </div>
        )}
        {item.endedAt && (
          <div className="flex min-w-0 items-center gap-2 rounded-lg border border-slate-200/80 bg-slate-50/60 px-2.5 py-1.5">
            <CheckCircle2 size={12} className="shrink-0 text-slate-400" />
            <span className="min-w-0 text-[0.67rem] font-semibold text-slate-500">
              <span className="mr-1 font-bold text-slate-600">Ended</span>
              <TimeAgo rfc3339={item.endedAt} />
            </span>
          </div>
        )}
        {duration && (
          <div className="flex min-w-0 items-center gap-2 rounded-lg border border-slate-200/80 bg-slate-50/60 px-2.5 py-1.5">
            <span className="text-[0.67rem] font-semibold text-slate-500">
              <span className="mr-1 font-bold text-slate-600">
                {item.state === SSHSession_State.ONGOING
                  ? "Elapsed"
                  : "Duration"}
              </span>
              {duration}
            </span>
          </div>
        )}
      </div>

      {hasContext && (
        <div className="mt-3 flex min-w-0 flex-wrap items-center gap-1.5 border-t border-slate-100 pt-2.5">
          {item.sessionRef && (
            <CardSession itemRef={item.sessionRef} />
          )}
          {item.serviceRef && (
            <CardService itemRef={item.serviceRef} />
          )}
          {!item.sessionRef && item.userRef && (
            <ResourceListLabel label="User" itemRef={item.userRef} />
          )}
          {!item.sessionRef && item.deviceRef && (
            <ResourceListLabel label="Device" itemRef={item.deviceRef} />
          )}
          {!item.serviceRef && item.namespaceRef && (
            <ResourceListLabel label="Namespace" itemRef={item.namespaceRef} />
          )}
        </div>
      )}
    </div>
  );
};

export const SSHSessionViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  deviceRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );
  const [page, setPage] = React.useState(props.page ?? 0);

  React.useEffect(() => {
    setPage(props.page ?? 0);
  }, [
    props.page,
    props.userRef?.uid,
    props.userRef?.name,
    props.sessionRef?.uid,
    props.sessionRef?.name,
    props.serviceRef?.uid,
    props.serviceRef?.name,
    props.namespaceRef?.uid,
    props.namespaceRef?.name,
    props.deviceRef?.uid,
    props.deviceRef?.name,
  ]);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listSSHSession",
      props.userRef?.uid,
      props.userRef?.name,
      props.sessionRef?.uid,
      props.sessionRef?.name,
      props.serviceRef?.uid,
      props.serviceRef?.name,
      props.namespaceRef?.uid,
      props.namespaceRef?.name,
      props.deviceRef?.uid,
      props.deviceRef?.name,
      page,
      props.itemsPerPage,
      from?.nanos,
      from?.seconds,
    ],
    queryFn: async () => {
      if (isDev()) {
        const startedAt = Timestamp.fromDate(
          dayjs().subtract(18, "minute").toDate(),
        );
        return ListSSHSessionResponse.create({
          items: [
            SSHSession.create({
              id: "12345",
              startedAt,
              endedAt: Timestamp.now(),
              sessionRef: props.sessionRef,
              userRef: props.userRef,
              deviceRef: props.deviceRef,
              serviceRef: props.serviceRef,
              namespaceRef: props.namespaceRef,
              state: SSHSession_State.COMPLETED,
            }),
          ],
          listResponseMeta: {
            totalCount: 1,
            page,
            itemsPerPage: 1,
            hasMore: false,
          },
        });
      }

      const { response } = await getClientVisibilityAccessLog().listSSHSession(
        ListSSHSessionRequest.create({
          userRef: props.userRef,
          sessionRef: props.sessionRef,
          serviceRef: props.serviceRef,
          namespaceRef: props.namespaceRef,
          deviceRef: props.deviceRef,
          common: {
            page,
            itemsPerPage: props.itemsPerPage ?? 100,
          },
          from,
        }),
      );
      return response;
    },
    refetchInterval: 60000,
  });

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500 shrink-0">
            Since
          </span>
          <SelectFromTimestamp
            initialValue="6 hour"
            onClear={() => {
              setPage(0);
              setFrom(undefined);
            }}
            onUpdate={(value) => {
              setPage(0);
              setFrom(value);
            }}
          />
        </div>
        <button
          type="button"
          onClick={() => qry.refetch()}
          disabled={qry.isLoading}
          className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1 text-[0.7rem] font-bold text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.05)] transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Radio
            size={11}
            strokeWidth={2.5}
            className={qry.isLoading ? "animate-pulse" : ""}
          />
          Refresh
        </button>
      </div>

      {qry.isError && (
        <div
          role="alert"
          className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[0.72rem] font-semibold text-red-700"
        >
          SSH sessions could not be loaded. Refresh to try again.
        </div>
      )}

      {qry.isLoading && !qry.data && (
        <div className="flex flex-col gap-2" aria-label="Loading SSH sessions">
          {[0, 1, 2].map((item) => (
            <div
              key={item}
              className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      )}

      {qry.data?.items.length === 0 && (
        <div className="flex items-center justify-center py-16">
          <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            No SSH sessions found
          </span>
        </div>
      )}

      {qry.data && qry.data.items.length > 0 && (
        <>
          <ResourceListWrapper>
            {qry.data.items.map((x) => (
              <ResourceListItem
                key={x.id || `${x.startedAt?.seconds ?? "session"}-${x.state}`}
                path={x.id ? `/visibility/ssh/${x.id}` : undefined}
              >
                <SSHSessionC item={x} />
              </ResourceListItem>
            ))}
          </ResourceListWrapper>
          <Paginator
            meta={qry.data.listResponseMeta}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
};

export default SSHSessionViewer;
