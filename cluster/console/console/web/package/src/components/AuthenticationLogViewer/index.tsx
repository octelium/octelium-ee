import {
  Authenticator_Status_Type,
  Session_Status_Authentication_Info_AAL,
  Session_Status_Authentication_Info_Authenticator_Mode,
  Session_Status_Authentication_Info_Type,
} from "@/apis/corev1/corev1";
import { AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuthenticationLogRequest,
  ListAuthenticationLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { isDev } from "@/utils";
import {
  getClientCore,
  getClientVisibilityAuthenticationLog,
} from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, RefreshCw, ShieldUser } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import AuthenticationLogSummary from "../LogSummary/AuthenticationLogSummary";
import { ResourceListLabel } from "../ResourceList";
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

const getAuthTypeName = (type: Session_Status_Authentication_Info_Type) =>
  match(type)
    .with(
      Session_Status_Authentication_Info_Type.AUTHENTICATOR,
      () => "Authenticator",
    )
    .with(
      Session_Status_Authentication_Info_Type.CREDENTIAL,
      () => "Credential",
    )
    .with(
      Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
      () => "Identity Provider",
    )
    .with(
      Session_Status_Authentication_Info_Type.REFRESH_TOKEN,
      () => "Refresh Token",
    )
    .with(Session_Status_Authentication_Info_Type.INTERNAL, () => "Internal")
    .with(Session_Status_Authentication_Info_Type.EXTERNAL, () => "External")
    .otherwise((o) => Session_Status_Authentication_Info_Type[o]);

const getAALName = (aal: Session_Status_Authentication_Info_AAL) =>
  match(aal)
    .with(Session_Status_Authentication_Info_AAL.AAL1, () => "AAL1")
    .with(Session_Status_Authentication_Info_AAL.AAL2, () => "AAL2")
    .with(Session_Status_Authentication_Info_AAL.AAL3, () => "AAL3")
    .otherwise(() => "");

const AuthTypeBadge = ({
  type,
}: {
  type: Session_Status_Authentication_Info_Type;
}) => {
  const { label, className } = match(type)
    .with(Session_Status_Authentication_Info_Type.AUTHENTICATOR, () => ({
      label: "Authenticator",
      className: "bg-blue-50 text-blue-700 border-blue-200",
    }))
    .with(Session_Status_Authentication_Info_Type.CREDENTIAL, () => ({
      label: "Credential",
      className: "bg-amber-50 text-amber-700 border-amber-200",
    }))
    .with(Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER, () => ({
      label: "Identity Provider",
      className: "bg-violet-50 text-violet-700 border-violet-200",
    }))
    .with(Session_Status_Authentication_Info_Type.REFRESH_TOKEN, () => ({
      label: "Refresh Token",
      className: "bg-slate-50 text-slate-600 border-slate-200",
    }))
    .with(Session_Status_Authentication_Info_Type.INTERNAL, () => ({
      label: "Internal",
      className: "bg-slate-50 text-slate-500 border-slate-200",
    }))
    .with(Session_Status_Authentication_Info_Type.EXTERNAL, () => ({
      label: "External",
      className: "bg-teal-50 text-teal-700 border-teal-200",
    }))
    .otherwise(() => ({
      label: Session_Status_Authentication_Info_Type[type],
      className: "bg-slate-50 text-slate-600 border-slate-200",
    }));

  return (
    <span
      className={twMerge(
        "text-[0.65rem] font-bold px-1.5 py-px rounded border shrink-0",
        className,
      )}
    >
      {label}
    </span>
  );
};

const BoolChip = ({ label }: { label: string }) => (
  <span className="text-[0.65rem] font-bold px-1.5 py-px rounded border bg-emerald-50 text-emerald-700 border-emerald-200">
    {label}
  </span>
);

const AuthenticationLogDetails = ({
  authLog,
}: {
  authLog: AuthenticationLog;
}) => {
  const x = authLog;
  const entry = x.entry;
  const info = entry?.authentication?.info;

  return (
    <div className="px-4 py-3 border-t border-slate-100 bg-slate-50/40">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-x-6 gap-y-3">
        <DetailField label="Log ID" mono>
          <CopyText value={x.metadata!.id} />
        </DetailField>

        {entry?.sessionRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Session
            </span>
            <CardSession itemRef={entry.sessionRef} />
          </div>
        )}

        {entry?.userRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              User
            </span>
            <ResourceListLabel itemRef={entry.userRef} />
          </div>
        )}

        {entry?.deviceRef && (
          <div className="flex flex-col gap-0.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
              Device
            </span>
            <ResourceListLabel itemRef={entry.deviceRef} />
          </div>
        )}

        {info?.aal != null &&
          info.aal !== Session_Status_Authentication_Info_AAL.AAL_UNSET && (
            <DetailField label="AAL">{getAALName(info.aal)}</DetailField>
          )}

        {info?.downstream?.ipAddress && (
          <DetailField label="IP address" mono>
            {info.downstream.ipAddress}
          </DetailField>
        )}

        {info?.downstream?.userAgent && (
          <DetailField label="User agent">
            {info.downstream.userAgent}
          </DetailField>
        )}

        {info?.downstream?.clientVersion && (
          <DetailField label="Client version" mono>
            {info.downstream.clientVersion}
          </DetailField>
        )}

        {info?.details.oneofKind === "identityProvider" && (
          <>
            {info.details.identityProvider.email && (
              <DetailField label="Email">
                {info.details.identityProvider.email}
              </DetailField>
            )}
            {info.details.identityProvider.identifier && (
              <DetailField label="Identifier" mono>
                {info.details.identityProvider.identifier}
              </DetailField>
            )}
            {info.details.identityProvider.identityProviderRef && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Identity Provider
                </span>
                <ResourceListLabel
                  itemRef={info.details.identityProvider.identityProviderRef}
                />
              </div>
            )}
          </>
        )}

        {info?.details.oneofKind === "credential" && (
          <>
            {info.details.credential.credentialRef && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Credential
                </span>
                <ResourceListLabel
                  itemRef={info.details.credential.credentialRef}
                />
              </div>
            )}
            {info.details.credential.tokenID && (
              <DetailField label="Token ID" mono>
                {info.details.credential.tokenID}
              </DetailField>
            )}
          </>
        )}

        {info?.details.oneofKind === "authenticator" && (
          <>
            {info.details.authenticator.authenticatorRef && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                  Authenticator
                </span>
                <ResourceListLabel
                  itemRef={info.details.authenticator.authenticatorRef}
                />
              </div>
            )}
            {info.details.authenticator.type != null && (
              <DetailField label="Authenticator type">
                {Authenticator_Status_Type[info.details.authenticator.type]}
              </DetailField>
            )}
            {info.details.authenticator.mode != null &&
              info.details.authenticator.mode !==
                Session_Status_Authentication_Info_Authenticator_Mode.MODE_UNSET && (
                <DetailField label="Mode">
                  {
                    Session_Status_Authentication_Info_Authenticator_Mode[
                      info.details.authenticator.mode
                    ]
                  }
                </DetailField>
              )}
            {info.details.authenticator.info?.type.oneofKind === "fido" && (
              <div className="col-span-full flex flex-wrap gap-1.5">
                {info.details.authenticator.info.type.fido.isPasskey && (
                  <BoolChip label="Passkey" />
                )}
                {info.details.authenticator.info.type.fido.isHardware && (
                  <BoolChip label="Hardware-based" />
                )}
                {info.details.authenticator.info.type.fido.isSoftware && (
                  <BoolChip label="Software-based" />
                )}
                {info.details.authenticator.info.type.fido
                  .isAttestationVerified && (
                  <BoolChip label="Attestation verified" />
                )}
                {info.details.authenticator.info.type.fido.userVerified && (
                  <BoolChip label="User verified" />
                )}
                {info.details.authenticator.info.type.fido.userPresent && (
                  <BoolChip label="User present" />
                )}
                {info.details.authenticator.info.type.fido.aaguid && (
                  <DetailField label="AAGUID" mono>
                    {info.details.authenticator.info.type.fido.aaguid}
                  </DetailField>
                )}
              </div>
            )}
          </>
        )}

        <div className="col-span-full flex justify-end pt-1">
          <Editor item={x} />
        </div>
      </div>
    </div>
  );
};

export const AuthenticationLogC = ({
  authLog,
}: {
  authLog: AuthenticationLog;
}) => {
  const x = authLog;
  const [expanded, setExpanded] = React.useState(false);
  const entry = x.entry;
  const info = entry?.authentication?.info;

  if (!entry) return null;

  const authType = info?.type;
  const aal =
    info?.aal != null &&
    info.aal !== Session_Status_Authentication_Info_AAL.AAL_UNSET
      ? getAALName(info.aal)
      : null;

  return (
    <div
      className={twMerge(
        "bg-white border border-slate-200 border-l-[3px] border-l-sky-500 rounded-lg overflow-hidden mb-1.5",
        "transition-[border-color,box-shadow] duration-150",
        "hover:border-slate-300 hover:shadow-[0_2px_8px_rgba(15,23,42,0.06)]",
      )}
    >
      <button
        className="w-full flex items-center gap-2 px-3.5 py-2 text-left cursor-pointer"
        onClick={() => setExpanded((v) => !v)}
      >
        <ShieldUser
          size={13}
          className="text-sky-500 shrink-0"
          strokeWidth={2.5}
        />

        <span className="text-[0.68rem] font-semibold text-slate-400 font-mono shrink-0">
          <TimeAgo rfc3339={x.metadata!.createdAt} />
        </span>

        <div className="flex items-center gap-1.5 flex-1 min-w-0 overflow-hidden">
          {entry.userRef && (
            <span className="text-[0.72rem] font-semibold text-slate-600 bg-slate-100 border border-slate-200 rounded px-1.5 py-px truncate max-w-[160px] font-mono">
              {entry.userRef.name ?? entry.userRef.uid}
            </span>
          )}
          {entry.sessionRef && (
            <>
              <span className="text-slate-300 text-[0.7rem] shrink-0">·</span>
              <span className="text-[0.68rem] font-semibold text-slate-400 truncate max-w-[120px] font-mono hidden sm:block">
                {entry.sessionRef.name ?? entry.sessionRef.uid}
              </span>
            </>
          )}
        </div>

        {authType != null &&
          authType !== Session_Status_Authentication_Info_Type.TYPE_UNSET && (
            <AuthTypeBadge type={authType} />
          )}

        {aal && (
          <span className="text-[0.62rem] font-bold px-1.5 py-px rounded bg-slate-800 text-slate-200 font-mono shrink-0">
            {aal}
          </span>
        )}

        {info?.downstream?.ipAddress && (
          <span className="text-[0.68rem] font-semibold text-slate-400 font-mono shrink-0 hidden lg:block">
            {info.downstream.ipAddress}
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
            <AuthenticationLogDetails authLog={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const DoAuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  itemsPerPage?: number;
  from?: Timestamp;
}) => {
  const [page, setPage] = React.useState(0);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listAuthenticationLog",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.deviceRef?.uid,
      props.identityProviderRef?.uid,
      page,
      props.from?.seconds,
      props.from?.nanos,
    ],
    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        return ListAuthenticationLogResponse.create({
          items: [
            AuthenticationLog.create({
              kind: "AuthenticationLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                sessionRef: getResourceRef(sess!),
                userRef: sess?.status?.userRef,
                deviceRef: sess?.status?.deviceRef,
                authentication: {
                  info: {
                    type: Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
                    aal: Session_Status_Authentication_Info_AAL.AAL2,
                    downstream: {
                      ipAddress: "1.2.3.4",
                      userAgent: "Mozilla/5.0",
                    },
                  },
                },
              },
            }),
          ],
        });
      }

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLog(
          ListAuthenticationLogRequest.create({
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            common: {
              page,
              itemsPerPage: props.itemsPerPage ?? 100,
            },
            from: props.from,
          }),
        );
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

      {qry.data?.items.map((x) => (
        <AuthenticationLogC key={x.metadata!.id} authLog={x} />
      ))}

      {qry.isSuccess && qry.data?.items.length === 0 && (
        <div className="flex items-center justify-center py-16">
          <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            No authentication log entries found
          </span>
        </div>
      )}

      {qry.data?.listResponseMeta && (
        <Paginator meta={qry.data.listResponseMeta} />
      )}
    </div>
  );
};

const AuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  itemsPerPage?: number;
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

      <AuthenticationLogSummary
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        deviceRef={props.deviceRef}
        identityProviderRef={props.identityProviderRef}
        from={from}
      />

      <DoAuthenticationLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        deviceRef={props.deviceRef}
        identityProviderRef={props.identityProviderRef}
        from={from}
      />
    </div>
  );
};

export default AuthenticationLogViewer;
