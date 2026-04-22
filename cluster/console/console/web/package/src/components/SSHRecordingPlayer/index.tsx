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
import { SquareTerminal } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardService from "../Card/CardService";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import TimeAgo from "../TimeAgo";

dayjs.extend(relativeTime);
dayjs.extend(utc);

const StateBadge = ({ state }: { state: SSHSession_State }) => {
  const { label, className } = match(state)
    .with(SSHSession_State.ONGOING, () => ({
      label: "Ongoing",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
    }))
    .with(SSHSession_State.COMPLETED, () => ({
      label: "Completed",
      className: "bg-slate-50 text-slate-600 border-slate-200",
    }))
    .otherwise(() => ({
      label: "Unknown",
      className: "bg-slate-50 text-slate-400 border-slate-200",
    }));

  return (
    <span
      className={twMerge(
        "inline-flex items-center h-[22px] px-2 rounded text-[0.68rem] font-bold border",
        className,
      )}
    >
      {state === SSHSession_State.ONGOING && (
        <span className="relative flex w-1.5 h-1.5 mr-1.5 shrink-0">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
          <span className="relative inline-flex rounded-full w-1.5 h-1.5 bg-emerald-500" />
        </span>
      )}
      {label}
    </span>
  );
};

export const SSHSessionC = (props: { item: SSHSession }) => {
  const { item } = props;

  const duration =
    item.startedAt && item.endedAt
      ? dayjs(Timestamp.toDate(item.endedAt)).from(
          Timestamp.toDate(item.startedAt),
          true,
        )
      : null;

  return (
    <div className="flex flex-col gap-2 w-full min-w-0">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <SquareTerminal
            size={13}
            className="text-slate-400 shrink-0"
            strokeWidth={2.5}
          />
          <span className="text-[0.72rem] font-mono font-semibold text-slate-600 truncate">
            <CopyText value={item.id} truncate={24} />
          </span>
        </div>
        <StateBadge state={item.state} />
      </div>

      <div className="flex items-center gap-4 flex-wrap">
        {item.startedAt && (
          <span className="inline-flex items-center gap-1 text-[0.68rem] font-semibold text-slate-400">
            <span className="font-bold text-slate-500">Started</span>
            <TimeAgo rfc3339={item.startedAt} />
          </span>
        )}
        {item.endedAt && (
          <span className="inline-flex items-center gap-1 text-[0.68rem] font-semibold text-slate-400">
            <span className="font-bold text-slate-500">Ended</span>
            <TimeAgo rfc3339={item.endedAt} />
          </span>
        )}
        {duration && (
          <span className="inline-flex items-center gap-1 text-[0.68rem] font-semibold text-slate-400">
            <span className="font-bold text-slate-500">Duration</span>
            {duration}
          </span>
        )}
      </div>

      {(item.sessionRef || item.serviceRef) && (
        <div className="flex flex-col gap-1.5 pt-1 border-t border-slate-100">
          {item.sessionRef && (
            <div className="flex items-center gap-2">
              <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400 w-14 shrink-0">
                Session
              </span>
              <CardSession itemRef={item.sessionRef} />
            </div>
          )}
          {item.serviceRef && (
            <div className="flex items-center gap-2">
              <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400 w-14 shrink-0">
                Service
              </span>
              <CardService itemRef={item.serviceRef} />
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const SSHSessionViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listSSHSession",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.serviceRef?.uid,
      props.namespaceRef?.uid,
      props.regionRef?.uid,
      props.deviceRef?.uid,
      props.page,
      props.itemsPerPage,
      from?.nanos,
      from?.seconds,
    ],
    queryFn: async () => {
      if (isDev()) {
        return ListSSHSessionResponse.create({
          items: [
            SSHSession.create({
              id: "12345",
              startedAt: Timestamp.now(),
              endedAt: Timestamp.now(),
              sessionRef: props.sessionRef,
              serviceRef: props.serviceRef,
              state: SSHSession_State.COMPLETED,
            }),
          ],
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
            page: props.page ?? 0,
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
      <div className="flex items-center gap-3">
        <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500 shrink-0">
          Since
        </span>
        <SelectFromTimestamp onUpdate={setFrom} />
      </div>

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
              <ResourceListItem key={x.id} path={`/visibility/ssh/${x.id}`}>
                <SSHSessionC item={x} />
              </ResourceListItem>
            ))}
          </ResourceListWrapper>
          <Paginator meta={qry.data.listResponseMeta} />
        </>
      )}
    </div>
  );
};

export default SSHSessionViewer;
