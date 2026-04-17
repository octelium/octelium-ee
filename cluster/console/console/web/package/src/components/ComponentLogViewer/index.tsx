import { ComponentLog, ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  ListComponentLogRequest,
  ListComponentLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityComponentLog } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, RefreshCw, ScrollText } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CopyText from "../CopyText";
import TimeAgo from "../TimeAgo";

const DetailField = ({
  label,
  children,
  mono = false,
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

type Level = ComponentLog_Entry_Level;

const getLevelMeta = (level: Level) =>
  match(level)
    .with(ComponentLog_Entry_Level.DEBUG, () => ({
      label: "Debug",
      border: "border-l-slate-400",
      icon: "text-slate-400",
      badge: "bg-slate-50 text-slate-600 border-slate-200",
    }))
    .with(ComponentLog_Entry_Level.INFO, () => ({
      label: "Info",
      border: "border-l-sky-400",
      icon: "text-sky-500",
      badge: "bg-sky-50 text-sky-700 border-sky-200",
    }))
    .with(ComponentLog_Entry_Level.WARN, () => ({
      label: "Warn",
      border: "border-l-amber-400",
      icon: "text-amber-500",
      badge: "bg-amber-50 text-amber-700 border-amber-200",
    }))
    .with(ComponentLog_Entry_Level.ERROR, () => ({
      label: "Error",
      border: "border-l-red-500",
      icon: "text-red-500",
      badge: "bg-red-50 text-red-700 border-red-200",
    }))
    .with(ComponentLog_Entry_Level.PANIC, () => ({
      label: "Panic",
      border: "border-l-red-700",
      icon: "text-red-700",
      badge: "bg-red-100 text-red-800 border-red-300",
    }))
    .with(ComponentLog_Entry_Level.FATAL, () => ({
      label: "Fatal",
      border: "border-l-red-900",
      icon: "text-red-900",
      badge: "bg-red-200 text-red-900 border-red-400",
    }))
    .otherwise(() => ({
      label: ComponentLog_Entry_Level[level],
      border: "border-l-slate-300",
      icon: "text-slate-400",
      badge: "bg-slate-50 text-slate-500 border-slate-200",
    }));

const StructFields = ({ fields }: { fields: Record<string, unknown> }) => {
  const entries = Object.entries(fields);
  if (entries.length === 0) return null;
  return (
    <>
      {entries.map(([k, v]) => (
        <DetailField key={k} label={k} mono>
          {typeof v === "object" ? JSON.stringify(v) : String(v)}
        </DetailField>
      ))}
    </>
  );
};

const ComponentLogDetails = ({ log }: { log: ComponentLog }) => {
  const x = log;
  const entry = x.entry;

  return (
    <div className="px-4 py-3 border-t border-slate-100 bg-slate-50/40">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-3">
        <DetailField label="Log ID" mono>
          <CopyText value={x.metadata!.id} />
        </DetailField>

        {entry?.message && (
          <div className="col-span-full">
            <DetailField label="Message" mono={false}>
              {entry.message}
            </DetailField>
          </div>
        )}

        {entry?.component?.namespace && (
          <DetailField label="Namespace" mono>
            {entry.component.namespace}
          </DetailField>
        )}

        {entry?.component?.type && (
          <DetailField label="Component type" mono>
            {entry.component.type}
          </DetailField>
        )}

        {entry?.component?.uid && (
          <DetailField label="Component UID" mono>
            <CopyText value={entry.component.uid} />
          </DetailField>
        )}

        {entry?.function && (
          <DetailField label="Function" mono>
            {entry.function}
          </DetailField>
        )}

        {entry?.file && (
          <DetailField label="File" mono>
            {entry.file}
            {entry.line ? `:${entry.line}` : ""}
          </DetailField>
        )}

        {entry?.fields?.fields &&
          Object.keys(entry.fields.fields).length > 0 && (
            <div className="col-span-full border-t border-slate-100 pt-3">
              <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400 block mb-2">
                Fields
              </span>
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-3">
                <StructFields
                  fields={
                    entry.fields.fields as unknown as Record<string, unknown>
                  }
                />
              </div>
            </div>
          )}

        <div className="col-span-full flex justify-end pt-1">
          <Editor item={x} />
        </div>
      </div>
    </div>
  );
};

export const ComponentLogC = ({ log }: { log: ComponentLog }) => {
  const x = log;
  const [expanded, setExpanded] = React.useState(false);
  const entry = x.entry;

  const level =
    entry?.level != null && entry.level > 0
      ? entry.level
      : ComponentLog_Entry_Level.INFO;

  const meta = getLevelMeta(level);

  return (
    <div
      className={twMerge(
        "bg-white border border-slate-200 border-l-[3px] rounded-lg overflow-hidden mb-1.5",
        "transition-[border-color,box-shadow] duration-150",
        "hover:border-slate-300 hover:shadow-[0_2px_8px_rgba(15,23,42,0.06)]",
        meta.border,
      )}
    >
      <button
        className="w-full flex items-center gap-2 px-3.5 py-2 text-left cursor-pointer"
        onClick={() => setExpanded((v) => !v)}
      >
        <ScrollText
          size={13}
          className={twMerge("shrink-0", meta.icon)}
          strokeWidth={2.5}
        />

        <span className="text-[0.68rem] font-semibold text-slate-400 font-mono shrink-0">
          <TimeAgo rfc3339={x.metadata!.createdAt} />
        </span>

        <span
          className={twMerge(
            "text-[0.65rem] font-bold px-1.5 py-px rounded border shrink-0",
            meta.badge,
          )}
        >
          {meta.label}
        </span>

        {entry?.component && (
          <span className="text-[0.72rem] font-semibold text-slate-600 bg-slate-100 border border-slate-200 rounded px-1.5 py-px font-mono shrink-0">
            {entry.component.namespace}/{entry.component.type}
          </span>
        )}

        <span className="text-[0.75rem] font-semibold text-slate-700 flex-1 min-w-0 truncate">
          {entry?.message ?? ""}
        </span>

        {entry?.function && (
          <span className="text-[0.65rem] font-semibold text-slate-400 font-mono shrink-0 hidden lg:block truncate max-w-[180px]">
            {entry.function}
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
            <ComponentLogDetails log={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const ComponentLogViewer = (props: { itemsPerPage?: number }) => {
  const [page, setPage] = React.useState(0);
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listComponentLog",
      page,
      from ? Timestamp.toDate(from).toISOString() : undefined,
    ],
    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        return ListComponentLogResponse.create({
          items: [
            ComponentLog.create({
              kind: "ComponentLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                component: {
                  namespace: "octelium",
                  type: "nocturne",
                  uid: "abc-123",
                },
                level: ComponentLog_Entry_Level.INFO,
                message: "Component is starting...",
                function: "main.Run",
                file: "cmd/nocturne/main.go",
                line: 42,
              },
            }),
          ],
        });
      }

      const { response } =
        await getClientVisibilityComponentLog().listComponentLog(
          ListComponentLogRequest.create({
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
            onClick={() => {
              setPage(0);
              qry.refetch();
            }}
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
          <ComponentLogC key={x.metadata!.id} log={x} />
        ))}

        {qry.isSuccess && qry.data?.items.length === 0 && (
          <div className="flex items-center justify-center py-16">
            <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
              No component log entries found
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

export default ComponentLogViewer;
