import {
  AccessLog,
  AccessLog_Entry_Common_Reason_Type,
  AccessLog_Entry_Common_Status,
  AccessLog_Entry_Info_DNS_Type,
  AccessLog_Entry_Info_MySQL_Type,
  AccessLog_Entry_Info_Postgres_Type,
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
import { ChevronDown, RefreshCw, ShieldCheck, ShieldX } from "lucide-react";
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
import { SelectFromTimestamp } from "./utils";

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
    <div className="px-4 py-3 border-t border-slate-100 bg-slate-50/40">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-3">
        <DetailField label="Log ID" mono>
          <CopyText value={x.metadata!.id} />
        </DetailField>

        {common?.connectionID && (
          <DetailField label="Connection ID">{common.connectionID}</DetailField>
        )}

        {common?.sessionRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Session
            </span>
            <CardSession itemRef={common.sessionRef} />
          </div>
        )}

        {common?.serviceRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Service
            </span>
            <CardService itemRef={common.serviceRef} />
          </div>
        )}

        {common?.reason?.details?.type.oneofKind === "policyMatch" && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
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

        {info?.type.oneofKind === "http" && (
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

        {info?.type.oneofKind === "kubernetes" && (
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

        {info?.type.oneofKind === "grpc" && (
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

        {info?.type.oneofKind === "postgres" && (
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

        {info?.type.oneofKind === "mysql" && (
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

        {info?.type.oneofKind === "dns" && (
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

        {info?.type.oneofKind === "ssh" && info.type.ssh.type && (
          <DetailField label="SSH type">{info.type.ssh.type}</DetailField>
        )}

        <div className="col-span-full flex justify-end pt-1">
          <Editor item={x} />
        </div>
      </div>
    </div>
  );
};

export const AccessLogC = ({ accessLog }: { accessLog: AccessLog }) => {
  const x = accessLog;
  const [expanded, setExpanded] = React.useState(false);

  if (!x.entry?.common) return null;

  const common = x.entry.common;
  const isAllowed = common.status === AccessLog_Entry_Common_Status.ALLOWED;
  const protoLabel = getProtoLabel(common.mode);
  const reason = getPolicyReason(common.reason?.type);
  const hasReason =
    common.reason?.type != null &&
    (common.reason.type as number) !==
      (AccessLog_Entry_Common_Reason_Type.TYPE_UNKNOWN_REASON as number);

  return (
    <div
      className={twMerge(
        "bg-white border rounded-lg overflow-hidden mb-1.5",
        "transition-[border-color,box-shadow] duration-150",
        "hover:border-slate-300 hover:shadow-[0_2px_8px_rgba(15,23,42,0.06)]",
        isAllowed
          ? "border-l-[3px] border-l-emerald-500 border-slate-200"
          : "border-l-[3px] border-l-red-500 border-slate-200",
      )}
    >
      <button
        className="w-full flex items-center gap-2 px-3.5 py-2 text-left cursor-pointer"
        onClick={() => setExpanded((v) => !v)}
      >
        {isAllowed ? (
          <ShieldCheck
            size={13}
            className="text-emerald-500 shrink-0"
            strokeWidth={2.5}
          />
        ) : (
          <ShieldX
            size={13}
            className="text-red-500 shrink-0"
            strokeWidth={2.5}
          />
        )}

        <span
          className={twMerge(
            "text-[0.65rem] font-bold px-1.5 py-px rounded border shrink-0",
            isAllowed
              ? "bg-emerald-50 text-emerald-700 border-emerald-200"
              : "bg-red-50 text-red-700 border-red-200",
          )}
        >
          {isAllowed ? "Allowed" : "Denied"}
        </span>

        <span className="text-[0.68rem] font-semibold text-slate-400 font-mono shrink-0">
          <TimeAgo rfc3339={x.metadata!.createdAt} />
        </span>

        <div className="flex items-center gap-1.5 flex-1 min-w-0 overflow-hidden">
          {common.sessionRef && (
            <span className="text-[0.72rem] font-semibold text-slate-600 bg-slate-100 border border-slate-200 rounded px-1.5 py-px truncate max-w-[160px]">
              {common.sessionRef.name ?? common.sessionRef.uid}
            </span>
          )}
          <span className="text-slate-300 text-[0.7rem] shrink-0">→</span>
          {common.serviceRef && (
            <span className="text-[0.72rem] font-semibold text-blue-700 bg-blue-50 border border-blue-200 rounded px-1.5 py-px truncate max-w-[160px]">
              {common.serviceRef.name ?? common.serviceRef.uid}
            </span>
          )}
          {common.namespaceRef && (
            <span className="text-[0.68rem] font-semibold text-slate-400 truncate hidden sm:block">
              · {common.namespaceRef.name}
            </span>
          )}
        </div>

        {protoLabel && (
          <span className="text-[0.62rem] font-bold px-1.5 py-px rounded bg-slate-800 text-slate-200 font-mono shrink-0">
            {protoLabel}
          </span>
        )}

        {hasReason && (
          <span className="text-[0.68rem] font-semibold text-slate-500 shrink-0 hidden md:block">
            {reason}
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
  return ListAccessLogResponse.create({
    items: [
      AccessLog.create({
        kind: "AccessLog",
        metadata: {
          createdAt: Timestamp.now(),
          id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
          actorRef: getResourceRef(sess!),
        },
        entry: {
          common: {
            sessionRef: getResourceRef(sess!),
            userRef: sess?.status?.userRef,
            deviceRef: sess?.status?.deviceRef,
            mode: Service_Spec_Mode.HTTP,
            serviceRef: getResourceRef(svc!),
            namespaceRef: svc?.status?.namespaceRef,
            status: AccessLog_Entry_Common_Status.ALLOWED,
            reason: { type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH },
          },
        },
      }),
    ],
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
}) => {
  const [page, setPage] = React.useState(0);

  const qry = useQuery({
    queryKey: ["visibility", "listAccessLog", { ...props }],
    queryFn: async () => {
      if (isDev()) return getListAccessLogResponseTest();
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
      });
      const { response } =
        await getClientVisibilityAccessLog().listAccessLog(req);
      return response;
    },
    refetchInterval: 60000,
  });

  return (
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
          <Paginator meta={qry.data.listResponseMeta} />
        </div>
      )}
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
