import { AuditLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuditLogRequest,
  ListAuditLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { ListLoading } from "@/components/Loading";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityAuditLog } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowRight,
  ChevronDown,
  RefreshCw,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import { ResourceListLabel } from "../ResourceList";
import TimeAgo from "../TimeAgo";
import { CompactSummary } from "../Summary";
import AuditLogSummary from "../LogSummary/AuditLogSummary";

const DetailField = ({
  label,
  children,
  mono = true,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) => (
  <div className="min-w-0">
    <span className="block truncate text-[10px] font-semibold uppercase tracking-[0.06em] text-slate-500">
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

const OperationBadge = ({ operation }: { operation: string }) => {
  const parts = operation.split(".");
  const method = parts.at(-1) ?? operation;
  const pkg = parts.slice(0, -1).join(".");
  return (
    <span className="flex min-w-0 items-center gap-1 overflow-hidden rounded-md border border-slate-200 bg-slate-50 px-1.5 py-px">
      {pkg && (
        <span className="hidden truncate font-mono text-[10px] font-medium text-slate-500 sm:block">
          {pkg}.
        </span>
      )}
      <span className="shrink-0 font-mono text-[10px] font-semibold leading-4 text-slate-700">
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
    <div className="border-t border-slate-200 bg-slate-50/70">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-2.5 sm:px-4">
        <h4 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
          Event details
        </h4>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-3 px-3.5 py-3 sm:grid-cols-2 sm:px-4 lg:grid-cols-3">
        {x.metadata?.id && (
          <DetailField label="Log ID">
            <CopyText value={x.metadata.id} />
          </DetailField>
        )}
        {entry.sessionRef && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Session
            </span>
            <CardSession itemRef={entry.sessionRef} />
          </div>
        )}

        {(entry.userRef || entry.deviceRef) && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Actor context
            </span>
            <div className="flex flex-wrap items-center gap-1.5">
              {entry.userRef && (
                <ResourceListLabel label="User" itemRef={entry.userRef} />
              )}
              {entry.deviceRef && (
                <ResourceListLabel label="Device" itemRef={entry.deviceRef} />
              )}
            </div>
          </div>
        )}

        {entry.resourceRef && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Target resource
            </span>
            <div className="flex flex-wrap items-center gap-1.5">
              <ResourceListLabel
                label={entry.resourceRef.kind || "Resource"}
                itemRef={getResourceRef(entry.resourceRef)}
              />
            </div>
          </div>
        )}

        {(entry.operation || entry.package || entry.service || entry.method) && (
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Operation details
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {entry.operation && (
                <div className="sm:col-span-2 lg:col-span-3">
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
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export const AuditLogC = ({ auditLog }: { auditLog: AuditLog }) => {
  const x = auditLog;
  const [expanded, setExpanded] = React.useState(false);
  const detailsID = React.useId();
  const entry = x.entry;

  if (!entry) return null;

  const actorRef = entry.userRef ?? entry.sessionRef;
  const actorName = actorRef?.name ?? actorRef?.uid;
  const actorLabel = entry.userRef ? "User" : "Session";
  const resourceName = entry.resourceRef?.name ?? entry.resourceRef?.uid;

  return (
    <div
      className={twMerge(
        "mb-1.5 overflow-hidden rounded-lg border border-slate-200 bg-white",
        "transition-[border-color,background-color,box-shadow] duration-200 ease-out",
        "hover:border-slate-300 hover:bg-slate-50/50 hover:shadow-card",
        expanded &&
          "border-slate-300 shadow-raised",
      )}
    >
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={detailsID}
        className="group flex w-full cursor-pointer items-center gap-3 px-3 py-2.5 text-left outline-none transition-colors duration-200 hover:bg-slate-50/70 focus-visible:bg-slate-50"
        onClick={() => setExpanded((v) => !v)}
      >
        <span
          aria-hidden="true"
          className="h-9 w-1 shrink-0 rounded-full bg-indigo-500"
        />

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span className="flex min-w-0 items-center gap-1.5 text-body font-semibold text-slate-800">
              <span className="max-w-44 truncate" title={actorName}>
                {actorName ?? "Unknown actor"}
              </span>
              <ArrowRight size={12} className="shrink-0 text-slate-300" />
              <span className="max-w-48 truncate" title={resourceName}>
                {resourceName ?? "Unknown resource"}
              </span>
            </span>

            {entry.resourceRef?.kind && (
              <span className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-px text-[10px] font-semibold leading-4 text-slate-600">
                {entry.resourceRef.kind}
              </span>
            )}
            {entry.operation && <OperationBadge operation={entry.operation} />}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-[10px] font-medium text-slate-500">
            <span>{actorLabel}</span>
            {(entry.service || entry.method) && (
              <span className="max-w-64 truncate font-mono">
                {entry.service
                  ? `${entry.service}${entry.method ? `/${entry.method}` : ""}`
                  : entry.method}
              </span>
            )}
            {x.metadata?.createdAt && (
              <TimeAgo rfc3339={x.metadata.createdAt} />
            )}
          </span>
        </span>

        <motion.span
          animate={{ rotate: expanded ? 180 : 0 }}
          transition={{ duration: 0.35, ease: "easeOut" }}
          className="flex shrink-0 text-slate-500 transition-colors duration-200 group-hover:text-slate-700"
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
            <AuditLogDetails auditLog={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const getListAuditLogResponseTest = async () => {
  const { response } = await getClientCore().listSession({});
  const session = response.items.at(0);
  const sessionRef = session
    ? getResourceRef(session)
    : ObjectReference.create({ uid: "dev-session", name: "dev-session" });
  const userRef =
    session?.status?.userRef ??
    ObjectReference.create({ uid: "dev-user", name: "alice" });
  const deviceRef =
    session?.status?.deviceRef ??
    ObjectReference.create({ uid: "dev-device", name: "alice-laptop" });
  const createdAt = (minutesAgo: number) =>
    Timestamp.fromDate(dayjs().subtract(minutesAgo, "minute").toDate());
  const target = (kind: string, name: string) =>
    ObjectReference.create({
      apiVersion: "core/v1",
      kind,
      uid: `dev-${kind.toLowerCase()}-${name}`,
      name,
    });
  const makeLog = (
    id: string,
    minutesAgo: number,
    resourceRef: ObjectReference,
    operation: string,
    service: string,
    method: string,
  ) =>
    AuditLog.create({
      kind: "AuditLog",
      metadata: { id, createdAt: createdAt(minutesAgo), actorRef: userRef },
      entry: {
        sessionRef,
        userRef,
        deviceRef,
        resourceRef,
        operation,
        package: "octelium.api.core.v1",
        service,
        method,
      },
    });
  const items = [
    makeLog(
      "dev-audit-create-user",
      2,
      target("User", "new-operator"),
      "octelium.api.core.v1.UserService.CreateUser",
      "UserService",
      "CreateUser",
    ),
    makeLog(
      "dev-audit-update-policy",
      6,
      target("Policy", "production-access"),
      "octelium.api.core.v1.PolicyService.UpdatePolicy",
      "PolicyService",
      "UpdatePolicy",
    ),
    makeLog(
      "dev-audit-update-service",
      11,
      target("Service", "admin-console"),
      "octelium.api.core.v1.ServiceService.UpdateService",
      "ServiceService",
      "UpdateService",
    ),
    makeLog(
      "dev-audit-read-secret",
      18,
      target("Secret", "database-credentials"),
      "octelium.api.core.v1.SecretService.GetSecret",
      "SecretService",
      "GetSecret",
    ),
    makeLog(
      "dev-audit-delete-device",
      25,
      target("Device", "retired-laptop"),
      "octelium.api.core.v1.DeviceService.DeleteDevice",
      "DeviceService",
      "DeleteDevice",
    ),
    makeLog(
      "dev-audit-update-config",
      34,
      target("Config", "cluster"),
      "octelium.api.core.v1.ConfigService.UpdateConfig",
      "ConfigService",
      "UpdateConfig",
    ),
  ];

  return ListAuditLogResponse.create({
    items,
    listResponseMeta: {
      totalCount: items.length,
      page: 0,
      itemsPerPage: items.length,
      hasMore: false,
    },
  });
};

const AuditLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  resourceRef?: ObjectReference;
  deviceRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
  onPageChange?: (page: number) => void;
  query?: string;
  periodMinutes?: number;
  onPeriodChange?: (value: number) => void;
}) => {
  const [page, setPage] = React.useState(props.page ?? 0);
  const [localFrom, setLocalFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );
  const controlledFrom = React.useMemo(
    () =>
      props.periodMinutes === undefined
        ? undefined
        : Timestamp.fromDate(
            dayjs().subtract(props.periodMinutes, "minute").toDate(),
          ),
    [props.periodMinutes],
  );
  const from = controlledFrom ?? localFrom;

  React.useEffect(() => {
    setPage(props.page ?? 0);
  }, [
    props.page,
    props.userRef?.uid,
    props.userRef?.name,
    props.sessionRef?.uid,
    props.sessionRef?.name,
    props.resourceRef?.uid,
    props.resourceRef?.name,
    props.deviceRef?.uid,
    props.deviceRef?.name,
    from.seconds,
    from.nanos,
  ]);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listAuditLog",
      {
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        resourceRef: props.resourceRef,
        deviceRef: props.deviceRef,
        page,
        from: Timestamp.toDate(from).toISOString(),
        query: props.query,
      },
    ],
    queryFn: async () => {
      if (isDev()) {
        const response = await getListAuditLogResponseTest();
        const matchesRef = (candidate?: ObjectReference) =>
          !props.resourceRef ||
          Boolean(
            (candidate?.uid &&
              props.resourceRef.uid &&
              candidate.uid === props.resourceRef.uid) ||
              (candidate?.name &&
                props.resourceRef.name &&
                candidate.name === props.resourceRef.name),
          );
        const items = response.items.filter((item) =>
          matchesRef(item.entry?.resourceRef),
        );
        return ListAuditLogResponse.create({
          ...response,
          items,
          listResponseMeta: {
            ...response.listResponseMeta,
            totalCount: items.length,
            itemsPerPage: items.length,
            hasMore: false,
          },
        });
      }

      const { response } = await getClientVisibilityAuditLog().listAuditLog(
        ListAuditLogRequest.create({
          common: {
            page,
            itemsPerPage: props.itemsPerPage ?? 25,
            query: props.query,
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
  const changePage = (nextPage: number) => {
    setPage(nextPage);
    props.onPageChange?.(nextPage);
  };
  const totalCount = Number(
    qry.data?.listResponseMeta?.totalCount ?? qry.data?.items.length ?? 0,
  );

  return (
    <div className="w-full flex flex-col gap-6">
      {props.periodMinutes === undefined && (
        <div className="flex items-center gap-3">
          <span className="text-xs font-semibold uppercase tracking-[0.05em] text-slate-500 shrink-0">
            Since
          </span>
          <SelectFromTimestamp
            initialValue="6 hour"
            onUpdate={(value) => {
              setLocalFrom(value);
              changePage(0);
            }}
          />
        </div>
      )}

      {!props.query && (
        <CompactSummary>
          <AuditLogSummary
            userRef={props.userRef}
            sessionRef={props.sessionRef}
            resourceRef={props.resourceRef}
            deviceRef={props.deviceRef}
            from={from}
          />
        </CompactSummary>
      )}

      <div className="w-full">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-normal text-slate-500 tabular-nums">
            {totalCount ? `${totalCount.toLocaleString()} entries` : "No entries"}
          </span>
          <button
            type="button"
            onClick={() => qry.refetch()}
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
              ? "Refresh failed. Showing the last loaded audit logs."
              : "Audit logs could not be loaded. Refresh to try again."}
          </div>
        )}

        {qry.isLoading && !qry.data ? (
          <ListLoading label="audit logs" />
        ) : qry.data ? (
          <>
            {qry.data?.items.map((x) => (
              <AuditLogC key={x.metadata!.id} auditLog={x} />
            ))}

            {qry.isSuccess && qry.data?.items.length === 0 && (
              <div className="flex items-center justify-center py-16">
                <span className="text-body font-semibold uppercase tracking-[0.08em] text-slate-500">
                  No audit log entries found
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

export default AuditLogViewer;
