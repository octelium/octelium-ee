import {
  AccessLog,
  AccessLog_Entry_Common_Reason_Type,
  AccessLog_Entry_Common_Status,
  AccessLog_Entry_Info_DNS_Type,
  AccessLog_Entry_Info_MySQL_Type,
  AccessLog_Entry_Info_Postgres_Type,
  AccessLog_Entry_Info_SSH_Type,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAccessLogRequest,
  ListAccessLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityAccessLog } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowRight,
  ChevronDown,
  RefreshCw,
  ShieldCheck,
  ShieldX,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import CardService from "../Card/CardService";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import AccessLogSummary from "../LogSummary/AccessLogSummary";
import { ResourceListLabel } from "../ResourceList";
import TimeAgo from "../TimeAgo";
import Editor from "./Editor";
import {
  accessLogStatusValue,
  AccessLogStatusFilter,
  SelectFromTimestamp,
} from "./utils";

export function convertBytes(
  bytes: number,
  options: { useBinaryUnits?: boolean; decimals?: number } = {},
): string {
  const { useBinaryUnits = false, decimals = 2 } = options;
  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(base));
  return `${(bytes / Math.pow(base, i)).toFixed(decimals)} ${units[i]}`;
}

export const getPolicyReason = (arg?: AccessLog_Entry_Common_Reason_Type) =>
  match(arg)
    .with(AccessLog_Entry_Common_Reason_Type.POLICY_MATCH, () => "Policy match")
    .with(
      AccessLog_Entry_Common_Reason_Type.NO_POLICY_MATCH,
      () => "No policy match",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.USER_DEACTIVATED,
      () => "User deactivated",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_NOT_ACTIVE,
      () => "Session not active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_EXPIRED,
      () => "Session expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.ACCESS_TOKEN_EXPIRED,
      () => "Access token expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_AUTHENTICATION_REQUIRED,
      () => "Authenticator required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_REGISTRATION_REQUIRED,
      () => "Authenticator registration required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SCOPE_UNAUTHORIZED,
      () => "Unauthorized scope",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_NOT_ACTIVE,
      () => "Device not active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_CLIENT_TYPE_INVALID,
      () => "Invalid session type",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_LOCKED,
      () => "Device locked",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_LOCKED,
      () => "Session locked",
    )
    .with(AccessLog_Entry_Common_Reason_Type.USER_LOCKED, () => "User locked")
    .otherwise((t) => (t ? AccessLog_Entry_Common_Reason_Type[t] : ""));

const getProtoLabel = (mode?: Service_Spec_Mode): string =>
  match(mode)
    .with(Service_Spec_Mode.HTTP, () => "HTTP")
    .with(Service_Spec_Mode.TCP, () => "TCP")
    .with(Service_Spec_Mode.SSH, () => "SSH")
    .with(Service_Spec_Mode.WEB, () => "WEB")
    .with(Service_Spec_Mode.KUBERNETES, () => "K8S")
    .with(Service_Spec_Mode.POSTGRES, () => "PG")
    .with(Service_Spec_Mode.MYSQL, () => "MySQL")
    .with(Service_Spec_Mode.UDP, () => "UDP")
    .with(Service_Spec_Mode.GRPC, () => "gRPC")
    .with(Service_Spec_Mode.DNS, () => "DNS")
    .otherwise(() => "");

const DetailField = ({
  label,
  children,
  mono = true,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) => (
  <div className="flex min-h-14 min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5 shadow-[0_1px_2px_rgba(15,23,42,0.025)]">
    <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
      {label}
    </span>
    <span
      className={twMerge(
        "min-w-0 break-words text-[0.74rem] font-semibold leading-5 text-slate-700",
        mono && "font-mono",
      )}
    >
      {children}
    </span>
  </div>
);

const HttpMethodBadge = ({ method }: { method: string }) => {
  const colors: Record<string, string> = {
    GET: "bg-blue-50 text-blue-700 border-blue-200",
    POST: "bg-green-50 text-green-700 border-green-200",
    PUT: "bg-amber-50 text-amber-700 border-amber-200",
    PATCH: "bg-amber-50 text-amber-700 border-amber-200",
    DELETE: "bg-red-50 text-red-700 border-red-200",
  };
  return (
    <span
      className={twMerge(
        "text-[0.62rem] font-bold px-1.5 py-px rounded border font-mono",
        colors[method.toUpperCase()] ??
          "bg-slate-50 text-slate-600 border-slate-200",
      )}
    >
      {method.toUpperCase()}
    </span>
  );
};

const HttpStatusBadge = ({ code }: { code: number }) => {
  const color =
    code >= 500
      ? "text-red-600"
      : code >= 400
        ? "text-amber-600"
        : code >= 300
          ? "text-blue-600"
          : "text-emerald-600";
  return (
    <span className={twMerge("font-mono font-bold text-[0.75rem]", color)}>
      {code}
    </span>
  );
};

const AccessLogDetails = ({ accessLog }: { accessLog: AccessLog }) => {
  const x = accessLog;
  const common = x.entry?.common;
  const info = x.entry?.info;

  return (
    <div className="border-t border-slate-200 bg-slate-50/70 px-4 py-4 sm:px-5">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h4 className="text-[0.75rem] font-bold text-slate-700">
            Log details
          </h4>
        </div>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {common?.connectionID && (
          <DetailField label="Connection ID">{common.connectionID}</DetailField>
        )}

        {common?.sessionRef && (
          <div className="col-span-full flex min-h-14 min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
              Session
            </span>
            <CardSession itemRef={common.sessionRef} />
          </div>
        )}

        {common?.serviceRef && (
          <div className="col-span-full flex min-h-14 min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
              Service
            </span>
            <CardService itemRef={common.serviceRef} />
          </div>
        )}

        {common?.reason?.details?.type.oneofKind === "policyMatch" && (
          <div className="flex min-h-14 min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
              Policy
            </span>
            {common.reason.details.type.policyMatch.type.oneofKind ===
              "policy" && (
              <ResourceListLabel
                label="Policy"
                itemRef={
                  common.reason.details.type.policyMatch.type.policy.policyRef
                }
              />
            )}
            {common.reason.details.type.policyMatch.type.oneofKind ===
              "inlinePolicy" && (
              <ResourceListLabel
                label="Inline policy"
                itemRef={
                  common.reason.details.type.policyMatch.type.inlinePolicy
                    .resourceRef
                }
              />
            )}
          </div>
        )}

        {info?.type.oneofKind && (
          <div className="col-span-full flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-100/60 p-3">
            <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-600">
              Protocol details
            </span>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {info.type.oneofKind === "http" && (
          <>
            {info.type.http.request?.path && (
              <DetailField label="Path">
                {info.type.http.request.path}
              </DetailField>
            )}
            {info.type.http.request?.method && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Method
                </span>
                <HttpMethodBadge method={info.type.http.request.method} />
              </div>
            )}
            {info.type.http.response?.code &&
              info.type.http.response.code > 0 && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                    Status
                  </span>
                  <HttpStatusBadge code={info.type.http.response.code} />
                </div>
              )}
            {info.type.http.request?.userAgent && (
              <DetailField label="User agent" mono={false}>
                {info.type.http.request.userAgent}
              </DetailField>
            )}
            {info.type.http.request?.bodyBytes != null &&
              info.type.http.request.bodyBytes > 0 && (
                <DetailField label="Req body">
                  {convertBytes(info.type.http.request.bodyBytes)}
                </DetailField>
              )}

            {info.type.http.response?.bodyBytes != null &&
              info.type.http.response.bodyBytes > 0 && (
                <DetailField label="Resp body">
                  {convertBytes(info.type.http.response.bodyBytes)}
                </DetailField>
              )}
          </>
        )}

        {info.type.oneofKind === "kubernetes" && (
          <>
            {info.type.kubernetes.verb && (
              <DetailField label="Verb">
                {info.type.kubernetes.verb}
              </DetailField>
            )}
            {info.type.kubernetes.resource && (
              <DetailField label="Resource">
                {info.type.kubernetes.resource}
              </DetailField>
            )}
            {info.type.kubernetes.subresource && (
              <DetailField label="Sub-resource">
                {info.type.kubernetes.subresource}
              </DetailField>
            )}
            {info.type.kubernetes.namespace && (
              <DetailField label="Namespace">
                {info.type.kubernetes.namespace}
              </DetailField>
            )}
            {info.type.kubernetes.name && (
              <DetailField label="Name">
                {info.type.kubernetes.name}
              </DetailField>
            )}
            {info.type.kubernetes.apiGroup && (
              <DetailField label="API group">
                {info.type.kubernetes.apiGroup}
              </DetailField>
            )}
            {info.type.kubernetes.apiVersion && (
              <DetailField label="API version">
                {info.type.kubernetes.apiVersion}
              </DetailField>
            )}
            {info.type.kubernetes.http?.request?.method && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Method
                </span>
                <HttpMethodBadge
                  method={info.type.kubernetes.http.request.method}
                />
              </div>
            )}
          </>
        )}

        {info.type.oneofKind === "grpc" && (
          <>
            {info.type.grpc.method && (
              <DetailField label="Method">{info.type.grpc.method}</DetailField>
            )}
            {info.type.grpc.service && (
              <DetailField label="Service">
                {info.type.grpc.service}
              </DetailField>
            )}
            {info.type.grpc.package && (
              <DetailField label="Package">
                {info.type.grpc.package}
              </DetailField>
            )}
            {info.type.grpc.serviceFullName && (
              <DetailField label="Full name">
                {info.type.grpc.serviceFullName}
              </DetailField>
            )}
            {info.type.grpc.status !== 0 && (
              <DetailField label="gRPC status">
                {info.type.grpc.status}
              </DetailField>
            )}
          </>
        )}

        {info.type.oneofKind === "postgres" && (
          <>
            {info.type.postgres.type && (
              <DetailField label="Type">
                {AccessLog_Entry_Info_Postgres_Type[info.type.postgres.type]}
              </DetailField>
            )}
            {info.type.postgres.details.oneofKind === "query" &&
              info.type.postgres.details.query?.query && (
                <div className="col-span-full">
                  <DetailField label="Query">
                    {info.type.postgres.details.query.query}
                  </DetailField>
                </div>
              )}
          </>
        )}

        {info.type.oneofKind === "mysql" && (
          <>
            {info.type.mysql.type && (
              <DetailField label="Type">
                {AccessLog_Entry_Info_MySQL_Type[info.type.mysql.type]}
              </DetailField>
            )}
            {info.type.mysql.details.oneofKind === "query" &&
              info.type.mysql.details.query?.query && (
                <div className="col-span-full">
                  <DetailField label="Query">
                    {info.type.mysql.details.query.query}
                  </DetailField>
                </div>
              )}
          </>
        )}

        {info.type.oneofKind === "dns" && (
          <>
            {info.type.dns.type && (
              <DetailField label="Type">
                {AccessLog_Entry_Info_DNS_Type[info.type.dns.type]}
              </DetailField>
            )}
            {info.type.dns.name && (
              <DetailField label="Name">{info.type.dns.name}</DetailField>
            )}
            {info.type.dns.answer && (
              <DetailField label="Answer">{info.type.dns.answer}</DetailField>
            )}
          </>
        )}

        {info.type.oneofKind === "ssh" && info.type.ssh.type && (
          <DetailField label="SSH type">{info.type.ssh.type}</DetailField>
        )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export const AccessLogC = ({ accessLog }: { accessLog: AccessLog }) => {
  const x = accessLog;
  const [expanded, setExpanded] = React.useState(false);
  const detailsID = React.useId();

  if (!x.entry?.common) return null;

  const common = x.entry.common;
  const info = x.entry.info;
  const isAllowed = common.status === AccessLog_Entry_Common_Status.ALLOWED;
  const protoLabel = getProtoLabel(common.mode);
  const reason = getPolicyReason(common.reason?.type);
  const hasReason =
    common.reason?.type != null &&
    (common.reason.type as number) !==
      (AccessLog_Entry_Common_Reason_Type.TYPE_UNKNOWN_REASON as number);
  const sourceRef = common.userRef ?? common.sessionRef;
  const sourceName = sourceRef?.name ?? sourceRef?.uid;
  const sourceLabel = common.userRef ? "User" : "Session";
  const serviceName = common.serviceRef?.name ?? common.serviceRef?.uid;
  let operation: string | undefined;
  let target: string | undefined;
  let response: React.ReactNode;

  if (info?.type.oneofKind === "http") {
    operation = info.type.http.request?.method;
    target = info.type.http.request?.path;
    if (info.type.http.response?.code) {
      response = <HttpStatusBadge code={info.type.http.response.code} />;
    }
  } else if (info?.type.oneofKind === "kubernetes") {
    operation = info.type.kubernetes.verb;
    target = [
      info.type.kubernetes.namespace,
      info.type.kubernetes.resource,
      info.type.kubernetes.name,
    ]
      .filter(Boolean)
      .join("/");
  } else if (info?.type.oneofKind === "grpc") {
    operation = info.type.grpc.method;
    target = info.type.grpc.serviceFullName || info.type.grpc.service;
  } else if (info?.type.oneofKind === "dns") {
    operation = info.type.dns.type
      ? AccessLog_Entry_Info_DNS_Type[info.type.dns.type]
      : undefined;
    target = info.type.dns.name;
  } else if (info?.type.oneofKind === "postgres") {
    operation = info.type.postgres.type
      ? AccessLog_Entry_Info_Postgres_Type[info.type.postgres.type]
      : undefined;
  } else if (info?.type.oneofKind === "mysql") {
    operation = info.type.mysql.type
      ? AccessLog_Entry_Info_MySQL_Type[info.type.mysql.type]
      : undefined;
  } else if (info?.type.oneofKind === "ssh") {
    operation = info.type.ssh.type
      ? AccessLog_Entry_Info_SSH_Type[info.type.ssh.type]
      : undefined;
  }

  return (
    <div
      className={twMerge(
        "mb-2 overflow-hidden rounded-xl border bg-white",
        "shadow-[0_1px_3px_rgba(15,23,42,0.04)] transition-[border-color,box-shadow] duration-500 ease-out",
        "hover:border-slate-300 hover:shadow-[0_4px_14px_rgba(15,23,42,0.065)]",
        isAllowed
          ? "border-slate-200"
          : "border-red-200/80 shadow-[0_1px_4px_rgba(220,38,38,0.045)]",
        expanded && "border-slate-300 shadow-[0_4px_16px_rgba(15,23,42,0.07)]",
      )}
    >
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={detailsID}
        className="group flex w-full cursor-pointer items-start gap-3 px-3.5 py-3 text-left outline-none transition-colors duration-500 hover:bg-slate-50/50 focus-visible:bg-blue-50/40 sm:px-4"
        onClick={() => setExpanded((v) => !v)}
      >
        <span
          className={twMerge(
            "mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border",
            isAllowed
              ? "border-emerald-200 bg-emerald-50 text-emerald-600"
              : "border-red-200 bg-red-50 text-red-600",
          )}
        >
          {isAllowed ? (
            <ShieldCheck size={16} strokeWidth={2.4} />
          ) : (
            <ShieldX size={16} strokeWidth={2.4} />
          )}
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-2">
          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5">
            <span
              className={twMerge(
                "rounded-md border px-2 py-1 text-[0.65rem] font-bold",
                isAllowed
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                  : "border-red-200 bg-red-50 text-red-700",
              )}
            >
              {isAllowed ? "Allowed" : "Denied"}
            </span>

            <span className="flex min-w-0 items-center gap-1.5 text-[0.75rem] font-semibold text-slate-700">
              {sourceName ? (
                <>
                  <span className="shrink-0 text-[0.62rem] font-bold uppercase tracking-[0.05em] text-slate-400">
                    {sourceLabel}
                  </span>
                  <span className="max-w-40 truncate font-mono text-[0.7rem]">
                    {sourceName}
                  </span>
                </>
              ) : (
                <span className="text-slate-400">Unknown source</span>
              )}
              <ArrowRight size={12} className="shrink-0 text-slate-300" />
              <span className="max-w-44 truncate font-mono text-[0.7rem] text-blue-700">
                {serviceName ?? "Unknown service"}
              </span>
            </span>

            {common.namespaceRef?.name && (
              <span className="hidden truncate text-[0.66rem] font-semibold text-slate-400 sm:inline">
                in {common.namespaceRef.name}
              </span>
            )}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1.5">
            {protoLabel && (
              <span className="rounded-md border border-slate-200 bg-slate-100 px-1.5 py-0.5 font-mono text-[0.61rem] font-bold text-slate-600">
                {protoLabel}
              </span>
            )}

            {operation && (
              <span className="font-mono text-[0.67rem] font-bold text-slate-600">
                {operation}
              </span>
            )}
            {target && (
              <span className="max-w-64 truncate font-mono text-[0.67rem] font-medium text-slate-500">
                {target}
              </span>
            )}
            {response}

            {hasReason && (
              <span className="truncate text-[0.67rem] font-semibold text-slate-500">
                {reason}
              </span>
            )}
          </span>
        </span>

        <motion.span
          animate={{ rotate: expanded ? 180 : 0 }}
          transition={{ duration: 0.35, ease: "easeOut" }}
          className="mt-2 flex shrink-0 text-slate-400 transition-colors duration-500 group-hover:text-slate-600"
        >
          <ChevronDown size={15} strokeWidth={2.25} />
        </motion.span>
      </button>

      <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 border-t border-slate-100 bg-slate-50/40 px-4 py-1.5 pl-[60px] text-[0.6rem] font-semibold text-slate-400">
        <TimeAgo rfc3339={x.metadata!.createdAt} />
        <span aria-hidden="true" className="text-slate-300">
          ·
        </span>
        <span className="flex min-w-0 items-center gap-1 font-mono">
          <span className="shrink-0 uppercase tracking-[0.05em]">Log ID</span>
          <span className="min-w-0 truncate text-slate-500">
            <CopyText value={x.metadata!.id} />
          </span>
        </span>
      </div>

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
            <AccessLogDetails accessLog={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export const getListAccessLogResponseTest = async () => {
  const r = await getClientCore().listSession({});
  const sess = r.response.items.at(0);
  const rSvcs = await getClientCore().listService({});
  const svc = rSvcs.response.items.at(0);
  const sessionRef = sess
    ? getResourceRef(sess)
    : ObjectReference.create({ uid: "dev-session", name: "dev-session" });
  const userRef =
    sess?.status?.userRef ??
    ObjectReference.create({ uid: "dev-user", name: "alice" });
  const deviceRef =
    sess?.status?.deviceRef ??
    ObjectReference.create({ uid: "dev-device", name: "alice-laptop" });
  const serviceRef = svc
    ? getResourceRef(svc)
    : ObjectReference.create({ uid: "dev-service", name: "admin-console" });
  const namespaceRef =
    svc?.status?.namespaceRef ??
    ObjectReference.create({ uid: "dev-namespace", name: "default" });
  const createdAt = (minutesAgo: number) =>
    Timestamp.fromDate(dayjs().subtract(minutesAgo, "minute").toDate());
  const common = {
    sessionRef,
    userRef,
    deviceRef,
    serviceRef,
    namespaceRef,
  };
  const items = [
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(1), id: "dev-access-http-get" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-http-01",
          mode: Service_Spec_Mode.HTTP,
          status: AccessLog_Entry_Common_Status.ALLOWED,
          reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
        },
        info: {
          type: {
            oneofKind: "http",
            http: {
              httpVersion: 2,
              request: {
                method: "GET",
                path: "/api/v1/users?limit=25",
                userAgent: "Mozilla/5.0 Octelium Console Dev",
                bodyBytes: 0,
              } as any,
              response: { code: 200, bodyBytes: 18432 } as any,
            },
          },
        },
      },
    }),
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(3), id: "dev-access-http-denied" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-http-02",
          mode: Service_Spec_Mode.HTTP,
          status: AccessLog_Entry_Common_Status.DENIED,
          reason: {
            type: AccessLog_Entry_Common_Reason_Type.NO_POLICY_MATCH,
          },
        },
        info: {
          type: {
            oneofKind: "http",
            http: {
              httpVersion: 1,
              request: {
                method: "POST",
                path: "/api/v1/policies",
                userAgent: "curl/8.6.0",
                bodyBytes: 926,
              } as any,
              response: { code: 403, bodyBytes: 148 } as any,
            },
          },
        },
      },
    }),
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(7), id: "dev-access-kubernetes" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-k8s-01",
          mode: Service_Spec_Mode.KUBERNETES,
          status: AccessLog_Entry_Common_Status.ALLOWED,
          reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
        },
        info: {
          type: {
            oneofKind: "kubernetes",
            kubernetes: {
              verb: "get",
              resource: "pods",
              subresource: "log",
              namespace: "production",
              name: "gateway-7d9f6c8c5b-z2m4q",
              apiPrefix: "/api",
              apiGroup: "",
              apiVersion: "v1",
              http: {
                httpVersion: 2,
                request: { method: "GET" } as any,
              },
            },
          },
        },
      },
    }),
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(12), id: "dev-access-grpc" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-grpc-01",
          mode: Service_Spec_Mode.GRPC,
          status: AccessLog_Entry_Common_Status.ALLOWED,
          reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
        },
        info: {
          type: {
            oneofKind: "grpc",
            grpc: {
              package: "octelium.api.core.v1",
              service: "UserService",
              method: "ListUser",
              serviceFullName: "octelium.api.core.v1.UserService",
              status: 0,
            } as any,
          },
        },
      },
    }),
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(18), id: "dev-access-dns" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-dns-01",
          mode: Service_Spec_Mode.DNS,
          status: AccessLog_Entry_Common_Status.ALLOWED,
          reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
        },
        info: {
          type: {
            oneofKind: "dns",
            dns: {
              type: 1,
              name: "cluster.internal",
              answer: "10.20.0.15",
            } as any,
          },
        },
      },
    }),
    AccessLog.create({
      kind: "AccessLog",
      metadata: { createdAt: createdAt(26), id: "dev-access-postgres" },
      entry: {
        common: {
          ...common,
          connectionID: "conn-postgres-01",
          mode: Service_Spec_Mode.POSTGRES,
          status: AccessLog_Entry_Common_Status.ALLOWED,
          reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
        },
        info: {
          type: {
            oneofKind: "postgres",
            postgres: {
              type: 1,
              details: {
                oneofKind: "query",
                query: {
                  query:
                    "SELECT id, email, created_at FROM users WHERE id = $1 LIMIT 1",
                },
              },
            },
          },
        },
      },
    }),
  ];

  return ListAccessLogResponse.create({
    items,
    listResponseMeta: {
      totalCount: items.length,
      page: 0,
      itemsPerPage: items.length,
      hasMore: false,
    },
  });
};

const DoAccessLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  itemsPerPage?: number;
  from?: Timestamp;
  status?: AccessLogStatusFilter;
}) => {
  const [page, setPage] = React.useState(0);

  React.useEffect(() => {
    setPage(0);
  }, [
    props.userRef?.uid,
    props.userRef?.name,
    props.sessionRef?.uid,
    props.sessionRef?.name,
    props.serviceRef?.uid,
    props.serviceRef?.name,
    props.namespaceRef?.uid,
    props.namespaceRef?.name,
    props.regionRef?.uid,
    props.regionRef?.name,
    props.deviceRef?.uid,
    props.deviceRef?.name,
    props.policyRef?.uid,
    props.policyRef?.name,
    props.from?.seconds,
    props.from?.nanos,
    props.status,
  ]);

  const qry = useQuery({
    queryKey: ["visibility", "listAccessLog", { ...props, page }],
    queryFn: async () => {
      if (isDev()) {
        const response = await getListAccessLogResponseTest();
        const items =
          props.status === "allowed"
            ? response.items.filter(
                (item) =>
                  item.entry?.common?.status ===
                  AccessLog_Entry_Common_Status.ALLOWED,
              )
            : props.status === "denied"
              ? response.items.filter(
                  (item) =>
                    item.entry?.common?.status ===
                    AccessLog_Entry_Common_Status.DENIED,
                )
              : response.items;
        return ListAccessLogResponse.create({
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
      const req = ListAccessLogRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        serviceRef: props.serviceRef,
        namespaceRef: props.namespaceRef,
        regionRef: props.regionRef,
        policyRef: props.policyRef,
        deviceRef: props.deviceRef,
        common: { page, itemsPerPage: props.itemsPerPage ?? 100 },
        from: props.from,
        status: accessLogStatusValue(props.status ?? "all"),
      });
      const { response } =
        await getClientVisibilityAccessLog().listAccessLog(req);
      return response;
    },
    refetchInterval: 60000,
  });
  const totalCount = Number(
    qry.data?.listResponseMeta?.totalCount ?? qry.data?.items.length ?? 0,
  );

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        <span className="text-[0.68rem] font-semibold text-slate-400 tabular-nums">
          {totalCount ? `${totalCount.toLocaleString()} entries` : "No entries"}
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

      {qry.isError && (
        <div
          role="alert"
          className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[0.72rem] font-semibold text-red-700"
        >
          Access logs could not be loaded. Refresh to try again.
        </div>
      )}

      {qry.isLoading && !qry.data && (
        <div className="flex flex-col gap-2" aria-label="Loading access logs">
          {[0, 1, 2].map((item) => (
            <div
              key={item}
              className="h-20 animate-pulse rounded-lg border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      )}

      <div className="w-full">
        {qry.data?.items.map((x) => (
          <AccessLogC key={x.metadata!.id} accessLog={x} />
        ))}
        {qry.data?.items.length === 0 && (
          <div className="flex items-center justify-center py-16">
            <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
              No log entries found
            </span>
          </div>
        )}
      </div>

      {qry.data?.listResponseMeta && (
        <div className="mt-4">
          <Paginator
            meta={qry.data.listResponseMeta}
            onPageChange={setPage}
          />
        </div>
      )}
    </div>
  );
};

export const AccessLogList = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  itemsPerPage?: number;
  periodMinutes?: number;
  status?: AccessLogStatusFilter;
}) => {
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
  const from =
    controlledFrom ?? localFrom;

  return (
    <div className="flex w-full flex-col gap-4">
      {props.periodMinutes === undefined && (
        <div className="flex items-center gap-3">
          <span className="shrink-0 text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
            Since
          </span>
          <SelectFromTimestamp onUpdate={setLocalFrom} />
        </div>
      )}
      <DoAccessLogViewer {...props} from={from} />
    </div>
  );
};

const AccessLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500 shrink-0">
          Since
        </span>
        <SelectFromTimestamp onUpdate={setFrom} />
      </div>

      <AccessLogSummary
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        serviceRef={props.serviceRef}
        namespaceRef={props.namespaceRef}
        regionRef={props.regionRef}
        policyRef={props.policyRef}
        deviceRef={props.deviceRef}
        from={from}
      />

      <DoAccessLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        serviceRef={props.serviceRef}
        namespaceRef={props.namespaceRef}
        regionRef={props.regionRef}
        policyRef={props.policyRef}
        deviceRef={props.deviceRef}
        from={from}
      />
    </div>
  );
};

export default AccessLogViewer;
