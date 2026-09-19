import { ComponentLog, ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Value } from "@/apis/google/protobuf/struct";
import {
  ComponentSelector,
  ListComponentLogRequest,
  ListComponentLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { ListLoading } from "@/components/Loading";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityComponentLog } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { Select } from "@mantine/core";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, RefreshCw } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CopyText from "../CopyText";
import TimeAgo from "../TimeAgo";
import ComponentLogSummary from "../LogSummary/ComponentLogSummary";
import { CompactSummary } from "../Summary";

const DetailField = ({
  label,
  children,
  mono = false,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) => (
  <div className="min-w-0">
    <span className="text-[10px] font-semibold uppercase tracking-[0.07em] text-slate-500">
      {label}
    </span>
    <span
      className={twMerge(
        "mt-0.5 block min-w-0 break-words text-xs font-semibold leading-5 text-slate-700",
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
      bar: "bg-slate-400",
      badge: "bg-slate-50 text-slate-600 border-slate-200",
    }))
    .with(ComponentLog_Entry_Level.INFO, () => ({
      label: "Info",
      bar: "bg-sky-500",
      badge: "bg-sky-50 text-sky-700 border-sky-200",
    }))
    .with(ComponentLog_Entry_Level.WARN, () => ({
      label: "Warn",
      bar: "bg-amber-500",
      badge: "bg-amber-50 text-amber-700 border-amber-200",
    }))
    .with(ComponentLog_Entry_Level.ERROR, () => ({
      label: "Error",
      bar: "bg-red-500",
      badge: "bg-red-50 text-red-700 border-red-200",
    }))
    .with(ComponentLog_Entry_Level.PANIC, () => ({
      label: "Panic",
      bar: "bg-red-700",
      badge: "bg-red-100 text-red-800 border-red-300",
    }))
    .with(ComponentLog_Entry_Level.FATAL, () => ({
      label: "Fatal",
      bar: "bg-red-900",
      badge: "bg-red-200 text-red-900 border-red-400",
    }))
    .otherwise(() => ({
      label: ComponentLog_Entry_Level[level],
      bar: "bg-slate-300",
      badge: "bg-slate-50 text-slate-500 border-slate-200",
    }));

const StructFields = ({ fields }: { fields: Record<string, Value> }) => {
  const entries = Object.entries(fields);
  if (entries.length === 0) return null;
  return (
    <>
      {entries.map(([k, v]) => {
        const value = Value.toJson(v);
        return (
          <DetailField key={k} label={k} mono>
            {typeof value === "object"
              ? JSON.stringify(value)
              : String(value ?? "null")}
          </DetailField>
        );
      })}
    </>
  );
};

const ComponentLogDetails = ({ log }: { log: ComponentLog }) => {
  const x = log;
  const entry = x.entry;

  return (
    <div className="border-t border-slate-200 bg-slate-50/70">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-2.5 sm:px-4">
        <h4 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
          Event details
        </h4>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-3 px-3.5 py-3 sm:grid-cols-2 sm:px-4 lg:grid-cols-3">
        {x.metadata?.id && (
          <DetailField label="Log ID" mono>
            <CopyText value={x.metadata.id} />
          </DetailField>
        )}
        {entry?.time && (
          <DetailField label="Event time">
            <TimeAgo rfc3339={entry.time} />
          </DetailField>
        )}
        {x.metadata?.createdAt && (
          <DetailField label="Recorded time">
            <TimeAgo rfc3339={x.metadata.createdAt} />
          </DetailField>
        )}
        {entry?.message && (
          <DetailField label="Message" mono={false}>
            {entry.message}
          </DetailField>
        )}

        {entry?.component && (
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Component
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {entry.component.namespace && (
                <DetailField label="Namespace" mono>
                  {entry.component.namespace}
                </DetailField>
              )}
              {entry.component.type && (
                <DetailField label="Component type" mono>
                  {entry.component.type}
                </DetailField>
              )}
              {entry.component.uid && (
                <DetailField label="Component UID" mono>
                  <CopyText value={entry.component.uid} />
                </DetailField>
              )}
            </div>
          </div>
        )}

        {(entry?.function || entry?.file) && (
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Source location
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2">
              {entry.function && (
                <DetailField label="Function" mono>
                  {entry.function}
                </DetailField>
              )}
              {entry.file && (
                <DetailField label="File" mono>
                  {entry.file}
                  {entry.line ? `:${entry.line}` : ""}
                </DetailField>
              )}
            </div>
          </div>
        )}

        {entry?.fields?.fields &&
          Object.keys(entry.fields.fields).length > 0 && (
            <div className="col-span-full border-t border-slate-200 pt-3">
              <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
                Fields
              </span>
              <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <StructFields fields={entry.fields.fields} />
              </div>
            </div>
          )}
      </div>
    </div>
  );
};

export const ComponentLogC = ({ log }: { log: ComponentLog }) => {
  const x = log;
  const [expanded, setExpanded] = React.useState(false);
  const detailsID = React.useId();
  const entry = x.entry;

  if (!entry) return null;

  const level =
    entry.level != null && entry.level > 0
      ? entry.level
      : ComponentLog_Entry_Level.INFO;

  const meta = getLevelMeta(level);
  const componentName = entry.component
    ? [entry.component.namespace, entry.component.type]
        .filter(Boolean)
        .join("/")
    : undefined;

  return (
    <div
      className={twMerge(
        "mb-1.5 overflow-hidden rounded-lg border border-slate-200 bg-white",
        "transition-[border-color,background-color,box-shadow] duration-200 ease-out",
        "hover:border-slate-300 hover:bg-slate-50/50 hover:shadow-card",
        expanded && "border-slate-300 shadow-card",
      )}
    >
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={detailsID}
        className="group flex w-full cursor-pointer items-center gap-3 px-3 py-2.5 text-left outline-none transition-colors duration-200 hover:bg-slate-50/50 focus-visible:bg-blue-50/40"
        onClick={() => setExpanded((v) => !v)}
      >
        <span className={twMerge("h-9 w-1 shrink-0 rounded-full", meta.bar)} />

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 items-center gap-1.5">
            <span
              className="min-w-0 flex-1 truncate text-sm font-semibold text-slate-900"
              title={entry.message}
            >
              {entry.message || "No message provided"}
            </span>
            <span
              className={twMerge(
                "shrink-0 rounded border px-1.5 py-px text-[10px] font-semibold leading-4",
                meta.badge,
              )}
            >
              {meta.label}
            </span>

            {componentName && (
              <span className="max-w-56 truncate rounded border border-slate-200 bg-slate-50 px-1.5 py-px font-mono text-[10px] font-medium leading-4 text-slate-600">
                {componentName}
              </span>
            )}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-0.5 text-[11px] font-medium text-slate-500">
            {(entry.function || entry.file) && (
              <>
                {entry.function && (
                  <span className="max-w-56 truncate font-mono font-semibold text-slate-500">
                    {entry.function}
                  </span>
                )}
                {entry.file && (
                  <span className="max-w-64 truncate font-mono">
                    {entry.file}
                    {entry.line ? `:${entry.line}` : ""}
                  </span>
                )}
              </>
            )}
            {(entry.time || x.metadata?.createdAt) && (
              <TimeAgo rfc3339={entry.time ?? x.metadata!.createdAt} />
            )}
          </span>
        </span>

        {entry.fields?.fields &&
          Object.keys(entry.fields.fields).length > 0 && (
            <span className="hidden shrink-0 text-right sm:block">
              <span className="block text-[10px] font-semibold uppercase tracking-wide text-slate-400">
                Fields
              </span>
              <span className="block text-xs font-semibold text-slate-700">
                {Object.keys(entry.fields.fields).length}
              </span>
            </span>
          )}

        <motion.span
          animate={{ rotate: expanded ? 180 : 0 }}
          transition={{ duration: 0.35, ease: "easeOut" }}
          className="flex shrink-0 text-slate-400 transition-colors duration-200 group-hover:text-slate-600"
        >
          <ChevronDown size={15} strokeWidth={2.25} />
        </motion.span>
      </button>

      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div
            id={detailsID}
            key="details"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden"
          >
            <ComponentLogDetails log={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const LEVEL_OPTIONS = [
  { value: "all", label: "All severities" },
  { value: String(ComponentLog_Entry_Level.DEBUG), label: "Debug" },
  { value: String(ComponentLog_Entry_Level.INFO), label: "Info" },
  { value: String(ComponentLog_Entry_Level.WARN), label: "Warn" },
  { value: String(ComponentLog_Entry_Level.ERROR), label: "Error" },
  { value: String(ComponentLog_Entry_Level.PANIC), label: "Panic" },
  { value: String(ComponentLog_Entry_Level.FATAL), label: "Fatal" },
];

const ComponentLogViewer = (props: {
  itemsPerPage?: number;
  level?: ComponentLog_Entry_Level;
  onLevelChange?: (level?: ComponentLog_Entry_Level) => void;
  page?: number;
  onPageChange?: (page: number) => void;
  query?: string;
  component?: ComponentSelector;
}) => {
  const [page, setPage] = React.useState(props.page ?? 0);
  const [level, setLevel] = React.useState<
    ComponentLog_Entry_Level | undefined
  >(props.level);
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  React.useEffect(() => setLevel(props.level), [props.level]);
  React.useEffect(() => setPage(props.page ?? 0), [props.page]);

  const changePage = (nextPage: number) => {
    setPage(nextPage);
    props.onPageChange?.(nextPage);
  };

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listComponentLog",
      page,
      from ? Timestamp.toDate(from).toISOString() : undefined,
      level,
      props.query,
      props.component ? ComponentSelector.toJsonString(props.component) : null,
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
                level: level ?? ComponentLog_Entry_Level.INFO,
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
              itemsPerPage: props.itemsPerPage ?? 25,
              query: props.query,
            },
            from,
            level,
            component: props.component,
          }),
        );
      return response;
    },
    refetchInterval: 60000,
  });
  const totalCount = Number(
    qry.data?.listResponseMeta?.totalCount ?? qry.data?.items.length ?? 0,
  );

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex items-center gap-3">
          <span className="shrink-0 text-xs font-semibold uppercase tracking-[0.05em] text-slate-500">
            Since
          </span>
          <SelectFromTimestamp
            initialValue="6 hour"
            onUpdate={(value) => {
              setFrom(value);
              changePage(0);
            }}
          />
        </div>
        <Select
          size="xs"
          label="Severity"
          aria-label="Component log severity"
          allowDeselect={false}
          value={level === undefined ? "all" : String(level)}
          onChange={(value) => {
            const next =
              value && value !== "all"
                ? (Number(value) as ComponentLog_Entry_Level)
                : undefined;
            setLevel(next);
            setPage(0);
            if (props.onLevelChange) props.onLevelChange(next);
            else props.onPageChange?.(0);
          }}
          data={LEVEL_OPTIONS}
          className="w-40"
        />
      </div>

      {!props.query && (
        <CompactSummary>
          <ComponentLogSummary from={from} />
        </CompactSummary>
      )}

      <div className="w-full">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-normal text-slate-500 tabular-nums">
            {totalCount
              ? `${totalCount.toLocaleString()} entries`
              : "No entries"}
          </span>
          <button
            type="button"
            onClick={() => {
              qry.refetch();
            }}
            disabled={qry.isLoading}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-normal text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-card disabled:opacity-50"
          >
            <RefreshCw
              size={11}
              strokeWidth={2.5}
              className={qry.isLoading ? "animate-spin" : ""}
            />
            Refresh
          </button>
        </div>

        {qry.isError && (
          <div
            role="alert"
            className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700"
          >
            {qry.data
              ? "Refresh failed. Showing the last available entries."
              : "Component logs could not be loaded. Use Refresh to try again."}
          </div>
        )}

        {qry.isLoading && !qry.data ? (
          <ListLoading label="component logs" />
        ) : qry.data ? (
          <>
            {qry.data.items.map((x) => (
              <ComponentLogC key={x.metadata!.id} log={x} />
            ))}

            {qry.isSuccess && qry.data.items.length === 0 && (
              <div className="flex items-center justify-center py-16">
                <span className="text-body font-semibold uppercase tracking-[0.08em] text-slate-500">
                  No component log entries found
                </span>
              </div>
            )}
          </>
        ) : null}
      </div>

      {qry.data?.listResponseMeta && (
        <Paginator
          meta={qry.data.listResponseMeta}
          onPageChange={changePage}
          showFilters={false}
        />
      )}
    </div>
  );
};

export default ComponentLogViewer;
