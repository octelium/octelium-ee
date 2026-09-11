import {
  AccessLog,
  AccessLog_Entry_Common_Reason_Type,
  AccessLog_Entry_Common_Status,
  AccessLog_Entry_Info_DNS_Type,
  AccessLog_Entry_Info_MySQL_Type,
  AccessLog_Entry_Info_MCP_Type,
  AccessLog_Entry_Info_Postgres_Type,
  AccessLog_Entry_Info_SOCKS5_AddressType,
  AccessLog_Entry_Info_SOCKS5_Type,
  AccessLog_Entry_Info_SSH_Type,
  AccessLog_Entry_Info_TCP_Type,
  AccessLog_Entry_Info_UDP_Type,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAccessLogRequest,
  ListAccessLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { ListLoading } from "@/components/Loading";
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
import { CompactSummary } from "../Summary";
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
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 Bytes";
  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.min(
    units.length - 1,
    Math.floor(Math.log(bytes) / Math.log(base)),
  );
  return `${(bytes / Math.pow(base, i)).toFixed(Math.max(0, decimals))} ${units[i]}`;
}

const durationBetween = (startedAt?: Timestamp, endedAt?: Timestamp) => {
  if (!startedAt || !endedAt) return undefined;
  const milliseconds =
    (Number(endedAt.seconds) - Number(startedAt.seconds)) * 1000 +
    (endedAt.nanos - startedAt.nanos) / 1_000_000;
  if (!Number.isFinite(milliseconds) || milliseconds < 0) return undefined;
  if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
  if (milliseconds < 60_000) {
    return `${(milliseconds / 1000).toFixed(milliseconds < 10_000 ? 2 : 1)} s`;
  }
  return `${(milliseconds / 60_000).toFixed(1)} min`;
};

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
    .with(Service_Spec_Mode.SOCKS5, () => "SOCKS5")
    .with(Service_Spec_Mode.RDP_WEB, () => "RDP Web")
    .with(Service_Spec_Mode.MCP, () => "MCP")
    .with(Service_Spec_Mode.LLM, () => "LLM")
    .with(Service_Spec_Mode.RDP, () => "RDP")
    .otherwise((value) =>
      value ? Service_Spec_Mode[value].replaceAll("_", " ") : "",
    );

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
        "rounded-md border px-1.5 py-px font-mono text-[10px] font-semibold leading-4",
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
    <span className={twMerge("font-mono font-semibold text-body", color)}>
      {code}
    </span>
  );
};

const ContextChip = ({ label }: { label: string }) => (
  <span className="rounded border border-slate-200 bg-slate-50 px-1.5 py-px text-[10px] font-semibold leading-4 text-slate-600">
    {label}
  </span>
);

const AccessLogDetails = ({ accessLog }: { accessLog: AccessLog }) => {
  const x = accessLog;
  const common = x.entry?.common;
  const info = x.entry?.info;

  return (
    <div className="border-t border-slate-200 bg-slate-50/70">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-2.5 sm:px-4">
        <h4 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
          Event details
        </h4>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-3 px-3.5 py-3 sm:grid-cols-2 sm:px-4 lg:grid-cols-3 xl:grid-cols-4">
        {x.metadata?.id && (
          <DetailField label="Log ID">
            <CopyText value={x.metadata.id} />
          </DetailField>
        )}
        {common?.connectionID && (
          <DetailField label="Connection ID">{common.connectionID}</DetailField>
        )}
        {common?.sessionID && (
          <DetailField label="Session ID">{common.sessionID}</DetailField>
        )}
        {common?.startedAt && (
          <DetailField label="Started">
            <TimeAgo rfc3339={common.startedAt} />
          </DetailField>
        )}
        {common?.endedAt && (
          <DetailField label="Ended">
            <TimeAgo rfc3339={common.endedAt} />
          </DetailField>
        )}
        {durationBetween(common?.startedAt, common?.endedAt) && (
          <DetailField label="Duration">
            {durationBetween(common?.startedAt, common?.endedAt)}
          </DetailField>
        )}
        {common && common.sequence > 0 && (
          <DetailField label="Connection sequence">
            {common.sequence.toLocaleString()}
          </DetailField>
        )}
        {(common?.isPublic || common?.isAnonymous) && (
          <div className="flex min-w-0 flex-wrap items-end gap-1.5">
            {common.isPublic && <ContextChip label="Public" />}
            {common.isAnonymous && <ContextChip label="Anonymous" />}
          </div>
        )}

        {common?.sessionRef && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Session
            </span>
            <CardSession itemRef={common.sessionRef} />
          </div>
        )}

        {common?.serviceRef && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Service
            </span>
            <CardService itemRef={common.serviceRef} />
          </div>
        )}

        {(common?.namespaceRef || common?.regionRef) && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Scope
            </span>
            <div className="mt-1 flex flex-wrap items-center gap-1.5">
              {common.namespaceRef && (
                <ResourceListLabel
                  label="Namespace"
                  itemRef={common.namespaceRef}
                />
              )}
              {common.regionRef && (
                <ResourceListLabel label="Region" itemRef={common.regionRef} />
              )}
            </div>
          </div>
        )}

        {common?.reason?.details?.type.oneofKind === "policyMatch" && (
          <div className="min-w-0 border-t border-slate-200 pt-3 sm:col-span-2 lg:col-span-3 xl:col-span-4">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
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
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Protocol details
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {info.type.oneofKind === "tcp" && (
          <>
            <DetailField label="Event type">
              {AccessLog_Entry_Info_TCP_Type[info.type.tcp.type]}
            </DetailField>
            {info.type.tcp.receivedBytes > 0 && (
              <DetailField label="Received">
                {convertBytes(info.type.tcp.receivedBytes)}
              </DetailField>
            )}
            {info.type.tcp.sentBytes > 0 && (
              <DetailField label="Sent">
                {convertBytes(info.type.tcp.sentBytes)}
              </DetailField>
            )}
          </>
        )}

        {info.type.oneofKind === "udp" && (
          <DetailField label="Event type">
            {AccessLog_Entry_Info_UDP_Type[info.type.udp.type]}
          </DetailField>
        )}

        {info.type.oneofKind === "http" && (
          <>
            {info.type.http.request?.path && (
              <DetailField label="Path">
                {info.type.http.request.path}
              </DetailField>
            )}
            {info.type.http.request?.method && (
              <div className="flex flex-col gap-0.5">
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                  Method
                </span>
                <HttpMethodBadge method={info.type.http.request.method} />
              </div>
            )}
            {info.type.http.response?.code &&
              info.type.http.response.code > 0 && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
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
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
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
          <>
            <DetailField label="SSH type">
              {AccessLog_Entry_Info_SSH_Type[info.type.ssh.type]}
            </DetailField>
            {info.type.ssh.details.oneofKind === "start" && (
              <>
                <DetailField label="Requested user">
                  {info.type.ssh.details.start.requestedUser}
                </DetailField>
                <DetailField label="Effective user">
                  {info.type.ssh.details.start.user}
                </DetailField>
              </>
            )}
            {info.type.ssh.details.oneofKind === "directTCPIPStart" && (
              <DetailField label="Destination">
                {info.type.ssh.details.directTCPIPStart.host}:
                {info.type.ssh.details.directTCPIPStart.port}
              </DetailField>
            )}
            {info.type.ssh.details.oneofKind === "sessionRequestExec" && (
              <DetailField label="Command">
                {info.type.ssh.details.sessionRequestExec.command}
              </DetailField>
            )}
            {info.type.ssh.details.oneofKind === "sessionRequestSubsystem" && (
              <DetailField label="Subsystem">
                {info.type.ssh.details.sessionRequestSubsystem.name}
              </DetailField>
            )}
          </>
        )}

        {info.type.oneofKind === "socks5" && (
          <>
            <DetailField label="Event type">
              {AccessLog_Entry_Info_SOCKS5_Type[info.type.socks5.type]}
            </DetailField>
            {(info.type.socks5.host || info.type.socks5.port > 0) && (
              <DetailField label="Destination">
                {info.type.socks5.host || "—"}
                {info.type.socks5.port > 0 ? `:${info.type.socks5.port}` : ""}
              </DetailField>
            )}
            {info.type.socks5.addressType > 0 && (
              <DetailField label="Address type">
                {AccessLog_Entry_Info_SOCKS5_AddressType[
                  info.type.socks5.addressType
                ]}
              </DetailField>
            )}
            {(info.type.socks5.upstreamHost ||
              info.type.socks5.upstreamPort > 0) && (
              <DetailField label="Upstream">
                {info.type.socks5.upstreamHost || "—"}
                {info.type.socks5.upstreamPort > 0
                  ? `:${info.type.socks5.upstreamPort}`
                  : ""}
              </DetailField>
            )}
            {info.type.socks5.receivedBytes > 0 && (
              <DetailField label="Received">
                {convertBytes(info.type.socks5.receivedBytes)}
              </DetailField>
            )}
            {info.type.socks5.sentBytes > 0 && (
              <DetailField label="Sent">
                {convertBytes(info.type.socks5.sentBytes)}
              </DetailField>
            )}
          </>
        )}

        {info.type.oneofKind === "mcp" && (
          <>
            <DetailField label="Event type">
              {AccessLog_Entry_Info_MCP_Type[info.type.mcp.type]}
            </DetailField>
            {info.type.mcp.method && (
              <DetailField label="Method">{info.type.mcp.method}</DetailField>
            )}
            {info.type.mcp.name && (
              <DetailField label="Target">{info.type.mcp.name}</DetailField>
            )}
            {info.type.mcp.protocolVersion && (
              <DetailField label="Protocol version">
                {info.type.mcp.protocolVersion}
              </DetailField>
            )}
            {info.type.mcp.requestID && (
              <DetailField label="Request ID">
                {info.type.mcp.requestID}
              </DetailField>
            )}
            {info.type.mcp.resultType && (
              <DetailField label="Result">{info.type.mcp.resultType}</DetailField>
            )}
            {info.type.mcp.isProtocolError && (
              <DetailField label="Protocol error">
                {info.type.mcp.errorCode}: {info.type.mcp.errorMessage}
              </DetailField>
            )}
            {info.type.mcp.client && (
              <DetailField label="Client">
                {[info.type.mcp.client.title || info.type.mcp.client.name,
                  info.type.mcp.client.version]
                  .filter(Boolean)
                  .join(" ")}
              </DetailField>
            )}
          </>
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
  const serviceName = common.serviceRef?.name ?? common.serviceRef?.uid;
  const duration = durationBetween(common.startedAt, common.endedAt);
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
        "mb-1.5 overflow-hidden rounded-lg border bg-white",
        "transition-[border-color,background-color,box-shadow] duration-200 ease-out",
        "hover:border-slate-300 hover:bg-slate-50/50 hover:shadow-card",
        isAllowed ? "border-slate-200" : "border-red-200/80 shadow-card",
        expanded && "border-slate-300 shadow-raised",
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
          className={twMerge(
            "h-9 w-1 shrink-0 rounded-full",
            isAllowed ? "bg-emerald-500" : "bg-red-500",
          )}
        />

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span className="flex min-w-0 items-center gap-1.5 text-body font-semibold text-slate-800">
              <span className="max-w-44 truncate" title={sourceName}>
                {sourceName ?? (common.isAnonymous ? "Anonymous" : "Unknown source")}
              </span>
              <ArrowRight size={12} className="shrink-0 text-slate-300" />
              <span className="max-w-48 truncate" title={serviceName}>
                {serviceName ?? "Unknown service"}
              </span>
            </span>
            <span
              className={twMerge(
                "inline-flex items-center gap-1 rounded-md border px-1.5 py-px text-[10px] font-semibold leading-4",
                isAllowed
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                  : "border-red-200 bg-red-50 text-red-700",
              )}
            >
              {isAllowed ? (
                <ShieldCheck size={9} strokeWidth={2.5} />
              ) : (
                <ShieldX size={9} strokeWidth={2.5} />
              )}
              {isAllowed ? "Allowed" : "Denied"}
            </span>
            {protoLabel && (
              <span className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-px font-mono text-[10px] font-semibold leading-4 text-slate-600">
                {protoLabel}
              </span>
            )}
            {operation && <HttpMethodBadge method={operation} />}
            {common.isPublic && <ContextChip label="Public" />}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-[10px] font-medium text-slate-500">
            {target && (
              <span className="max-w-72 truncate font-mono" title={target}>
                {target}
              </span>
            )}
            {hasReason && (
              <span className="max-w-56 truncate" title={reason}>
                {reason}
              </span>
            )}
            {common.namespaceRef?.name && (
              <span className="truncate">{common.namespaceRef.name}</span>
            )}
            {x.metadata?.createdAt && (
              <TimeAgo rfc3339={x.metadata.createdAt} />
            )}
          </span>
        </span>

        {(response || duration) && (
          <span className="hidden shrink-0 items-center gap-4 sm:flex">
            {response && (
              <span className="text-right">
                <span className="block text-[9px] font-semibold uppercase tracking-[0.06em] text-slate-500">
                  Response
                </span>
                <span className="block text-xs font-semibold tabular-nums text-slate-700">
                  {response}
                </span>
              </span>
            )}
            {duration && (
              <span className="text-right">
                <span className="block text-[9px] font-semibold uppercase tracking-[0.06em] text-slate-500">
                  Duration
                </span>
                <span className="block text-xs font-semibold tabular-nums text-slate-700">
                  {duration}
                </span>
              </span>
            )}
          </span>
        )}

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
  query?: string;
  page?: number;
  onPageChange?: (page: number) => void;
}) => {
  const [page, setPage] = React.useState(props.page ?? 0);

  React.useEffect(() => {
    setPage(props.page ?? 0);
  }, [props.page]);

  const changePage = (nextPage: number) => {
    setPage(nextPage);
    props.onPageChange?.(nextPage);
  };

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
        common: {
          page,
          itemsPerPage: props.itemsPerPage ?? 25,
          query: props.query,
        },
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
        <span className="text-xs font-normal text-slate-500 tabular-nums">
          {totalCount ? `${totalCount.toLocaleString()} entries` : "No entries"}
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
            ? "Refresh failed. Showing the last loaded access logs."
            : "Access logs could not be loaded. Refresh to try again."}
        </div>
      )}

      {qry.isLoading && !qry.data ? (
        <ListLoading label="access logs" />
      ) : qry.data ? (
        <div className="w-full">
          {qry.data?.items.map((x) => (
            <AccessLogC key={x.metadata!.id} accessLog={x} />
          ))}
          {qry.data?.items.length === 0 && (
            <div className="flex items-center justify-center py-16">
              <span className="text-body font-semibold uppercase tracking-[0.08em] text-slate-500">
                No log entries found
              </span>
            </div>
          )}
        </div>
      ) : null}

      {qry.data?.listResponseMeta && (
        <div className="mt-4">
          <Paginator
            meta={qry.data.listResponseMeta}
            onPageChange={changePage}
            showFilters={false}
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
  query?: string;
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
          <span className="shrink-0 text-xs font-semibold uppercase tracking-[0.05em] text-slate-500">
            Since
          </span>
          <SelectFromTimestamp
            initialValue="6 hour"
            onUpdate={setLocalFrom}
          />
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
  onPageChange?: (page: number) => void;
  status?: AccessLogStatusFilter;
  query?: string;
}) => {
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <span className="text-xs font-semibold uppercase tracking-[0.05em] text-slate-500 shrink-0">
          Since
        </span>
        <SelectFromTimestamp
          initialValue="6 hour"
          onUpdate={(value) => {
            setFrom(value);
            props.onPageChange?.(0);
          }}
        />
      </div>

      {!props.query && (
        <CompactSummary>
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
        </CompactSummary>
      )}

      <DoAccessLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        serviceRef={props.serviceRef}
        namespaceRef={props.namespaceRef}
        regionRef={props.regionRef}
        policyRef={props.policyRef}
        deviceRef={props.deviceRef}
        from={from}
        status={props.status}
        query={props.query}
        itemsPerPage={props.itemsPerPage}
        page={props.page}
        onPageChange={props.onPageChange}
      />
    </div>
  );
};

export default AccessLogViewer;
