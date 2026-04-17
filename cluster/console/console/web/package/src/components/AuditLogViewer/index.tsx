import { AuditLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuditLogRequest,
  ListAuditLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityAuditLog } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, ClipboardList, RefreshCw } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import { ResourceListLabel } from "../ResourceList";
import TimeAgo from "../TimeAgo";

const DetailField = ({
  label,
  children,
  mono = true,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) => (
  <div className="flex flex-col gap-0.5">
    <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {label}
    </span>
    <span
      className={twMerge(
        "text-[0.75rem] font-semibold text-slate-700 break-all",
        mono && "font-mono",
      )}
    >
      {children}
    </span>
  </div>
);

const OperationBadge = ({ operation }: { operation: string }) => {
  const parts = operation.split(".");
  const method = parts.at(-1) ?? operation;
  const pkg = parts.slice(0, -1).join(".");
  return (
    <span className="flex items-center gap-1 min-w-0 overflow-hidden">
      {pkg && (
        <span className="text-[0.65rem] font-semibold text-slate-400 truncate hidden sm:block font-mono">
          {pkg}.
        </span>
      )}
      <span className="text-[0.72rem] font-bold text-slate-700 font-mono shrink-0">
        {method}
      </span>
    </span>
  );
};

const AuditLogDetails = ({ auditLog }: { auditLog: AuditLog }) => {
  const x = auditLog;
  const entry = x.entry;
  if (!entry) return null;

  return (
    <div className="px-4 py-3 border-t border-slate-100 bg-slate-50/40">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-3">
        <DetailField label="Log ID" mono>
          <CopyText value={x.metadata!.id} />
        </DetailField>

        {entry.operation && (
          <div className="col-span-full">
            <DetailField label="Operation">{entry.operation}</DetailField>
          </div>
        )}

        {entry.package && (
          <DetailField label="Package">{entry.package}</DetailField>
        )}

        {entry.service && (
          <DetailField label="Service">{entry.service}</DetailField>
        )}

        {entry.method && (
          <DetailField label="Method">{entry.method}</DetailField>
        )}

        {entry.resourceRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Resource ({entry.resourceRef.kind})
            </span>
            <ResourceListLabel itemRef={getResourceRef(entry.resourceRef)} />
          </div>
        )}

        {entry.sessionRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Session
            </span>
            <CardSession itemRef={entry.sessionRef} />
          </div>
        )}

        {entry.userRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              User
            </span>
            <ResourceListLabel itemRef={entry.userRef} />
          </div>
        )}

        {entry.deviceRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Device
            </span>
            <ResourceListLabel itemRef={entry.deviceRef} />
          </div>
        )}

        <div className="col-span-full flex justify-end pt-1">
          <Editor item={x} />
        </div>
      </div>
    </div>
  );
};

export const AuditLogC = ({ auditLog }: { auditLog: AuditLog }) => {
  const x = auditLog;
  const [expanded, setExpanded] = React.useState(false);
  const entry = x.entry;

  return (
    <div
      className={twMerge(
        "bg-white border border-slate-200 border-l-[3px] border-l-violet-500 rounded-lg overflow-hidden mb-1.5",
        "transition-[border-color,box-shadow] duration-150",
        "hover:border-slate-300 hover:shadow-[0_2px_8px_rgba(15,23,42,0.06)]",
      )}
    >
      <button
        className="w-full flex items-center gap-2 px-3.5 py-2 text-left cursor-pointer"
        onClick={() => setExpanded((v) => !v)}
      >
        <ClipboardList
          size={13}
          className="text-violet-500 shrink-0"
          strokeWidth={2.5}
        />

        <span className="text-[0.68rem] font-semibold text-slate-400 font-mono shrink-0">
          <TimeAgo rfc3339={x.metadata!.createdAt} />
        </span>

        <div className="flex items-center gap-1.5 flex-1 min-w-0 overflow-hidden">
          {entry?.sessionRef && (
            <span className="text-[0.72rem] font-semibold text-slate-600 bg-slate-100 border border-slate-200 rounded px-1.5 py-px truncate max-w-[140px] font-mono">
              {entry.sessionRef.name ?? entry.sessionRef.uid}
            </span>
          )}
          {entry?.sessionRef && entry?.resourceRef && (
            <span className="text-slate-300 text-[0.7rem] shrink-0">→</span>
          )}
          {entry?.resourceRef && (
            <span className="text-[0.72rem] font-semibold text-violet-700 bg-violet-50 border border-violet-200 rounded px-1.5 py-px truncate max-w-[140px] font-mono">
              {entry.resourceRef.name ?? entry.resourceRef.uid}
            </span>
          )}
          {entry?.resourceRef?.kind && (
            <span className="text-[0.65rem] font-semibold text-slate-400 shrink-0 hidden sm:block">
              · {entry.resourceRef.kind}
            </span>
          )}
        </div>

        {entry?.operation && (
          <div className="shrink-0 hidden md:flex items-center max-w-[260px] overflow-hidden">
            <OperationBadge operation={entry.operation} />
          </div>
        )}

        {(entry?.service || entry?.method) && (
          <span className="text-[0.62rem] font-bold px-1.5 py-px rounded bg-slate-800 text-slate-200 font-mono shrink-0 hidden lg:block">
            {entry.service ? `${entry.service}/${entry.method}` : entry.method}
          </span>
        )}

        <motion.span
          animate={{ rotate: expanded ? 180 : 0 }}
          transition={{ duration: 0.18 }}
          className="flex items-center shrink-0 text-slate-400"
        >
          <ChevronDown size={13} strokeWidth={2.5} />
        </motion.span>
      </button>

      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div
            key="details"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.18, ease: "easeInOut" }}
            className="overflow-hidden"
          >
            <AuditLogDetails auditLog={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const AuditLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  resourceRef?: ObjectReference;
  deviceRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listAuditLog",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.serviceRef?.uid,
      props.resourceRef?.uid,
      props.deviceRef?.uid,
      props.page,
      from ? Timestamp.toDate(from).toISOString() : undefined,
    ],
    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        return ListAuditLogResponse.create({
          items: [
            AuditLog.create({
              kind: "AuditLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                sessionRef: getResourceRef(sess!),
                userRef: sess?.status?.userRef,
                deviceRef: sess?.status?.deviceRef,
                resourceRef: getResourceRef(sess!),
                operation: "octelium.api.core.v1.ListUser",
                service: "MainService",
                method: "ListUser",
                package: "octelium.api.core.v1",
              },
            }),
          ],
        });
      }

      const { response } = await getClientVisibilityAuditLog().listAuditLog(
        ListAuditLogRequest.create({
          common: {
            page: props.page ?? 0,
            itemsPerPage: props.itemsPerPage ?? 100,
          },
          userRef: props.userRef,
          sessionRef: props.sessionRef,
          deviceRef: props.deviceRef,
          resourceRef: props.resourceRef,
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

      <div className="w-full">
        <div className="flex items-center justify-between mb-4">
          <span className="text-[0.68rem] font-semibold text-slate-400 tabular-nums">
            {qry.data?.items.length
              ? `${qry.data.items.length.toLocaleString()} entries`
              : ""}
          </span>
          <button
            onClick={() => qry.refetch()}
            disabled={qry.isLoading}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] disabled:opacity-50"
          >
            <RefreshCw
              size={11}
              strokeWidth={2.5}
              className={qry.isLoading ? "animate-spin" : ""}
            />
            Refresh
          </button>
        </div>

        {qry.data?.items.map((x) => (
          <AuditLogC key={x.metadata!.id} auditLog={x} />
        ))}

        {qry.isSuccess && qry.data?.items.length === 0 && (
          <div className="flex items-center justify-center py-16">
            <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
              No audit log entries found
            </span>
          </div>
        )}
      </div>

      {qry.data?.listResponseMeta && (
        <Paginator meta={qry.data.listResponseMeta} />
      )}
    </div>
  );
};

export default AuditLogViewer;
