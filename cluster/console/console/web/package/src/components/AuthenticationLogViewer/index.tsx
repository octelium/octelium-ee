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
    <div className="border-t border-slate-200 bg-slate-50/70 px-4 py-4 sm:px-5">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h4 className="text-[0.75rem] font-bold text-slate-700">
          Log details
        </h4>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">

        {entry?.sessionRef && (
          <div className="col-span-full flex min-h-14 min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
              Session
            </span>
            <CardSession itemRef={entry.sessionRef} />
          </div>
        )}

        {(entry?.userRef || entry?.deviceRef) && (
          <div className="col-span-full flex min-h-14 min-w-0 flex-col gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
            <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-500">
              Identity context
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

        {info && (
          <div className="col-span-full flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-100/60 p-3">
            <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-600">
              Authentication details
            </span>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
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
            </div>
          </div>
        )}
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
  const detailsID = React.useId();
  const entry = x.entry;
  const info = entry?.authentication?.info;

  if (!entry) return null;

  const authType = info?.type;
  const aal =
    info?.aal != null &&
    info.aal !== Session_Status_Authentication_Info_AAL.AAL_UNSET
      ? getAALName(info.aal)
      : null;
  const userName = entry.userRef?.name ?? entry.userRef?.uid;
  const sessionName = entry.sessionRef?.name ?? entry.sessionRef?.uid;

  return (
    <div
      className={twMerge(
        "mb-2 overflow-hidden rounded-xl border border-slate-200 bg-white",
        "shadow-[0_1px_3px_rgba(15,23,42,0.04)] transition-[border-color,box-shadow] duration-500 ease-out",
        "hover:border-slate-300 hover:shadow-[0_4px_14px_rgba(15,23,42,0.065)]",
        expanded &&
          "border-slate-300 shadow-[0_4px_16px_rgba(15,23,42,0.07)]",
      )}
    >
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={detailsID}
        className="group flex w-full cursor-pointer items-start gap-3 px-3.5 py-3 text-left outline-none transition-colors duration-500 hover:bg-slate-50/50 focus-visible:bg-blue-50/40 sm:px-4"
        onClick={() => setExpanded((v) => !v)}
      >
        <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-sky-200 bg-sky-50 text-sky-600">
          <ShieldUser size={16} strokeWidth={2.4} />
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-2">
          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5">
            <span className="rounded-md border border-sky-200 bg-sky-50 px-2 py-1 text-[0.65rem] font-bold text-sky-700">
              Authentication
            </span>

            {userName ? (
              <span className="flex min-w-0 items-center gap-1.5">
                <span className="shrink-0 text-[0.62rem] font-bold uppercase tracking-[0.05em] text-slate-400">
                  User
                </span>
                <span className="max-w-48 truncate font-mono text-[0.7rem] font-semibold text-slate-700">
                  {userName}
                </span>
              </span>
            ) : (
              <span className="text-[0.7rem] font-semibold text-slate-400">
                Unknown user
              </span>
            )}

            {sessionName && (
              <span className="hidden min-w-0 items-center gap-1.5 sm:flex">
                <span className="text-slate-300">·</span>
                <span className="shrink-0 text-[0.62rem] font-bold uppercase tracking-[0.05em] text-slate-400">
                  Session
                </span>
                <span className="max-w-36 truncate font-mono text-[0.67rem] font-medium text-slate-500">
                  {sessionName}
                </span>
              </span>
            )}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1.5">
            {authType != null &&
              authType !==
                Session_Status_Authentication_Info_Type.TYPE_UNSET && (
                <AuthTypeBadge type={authType} />
              )}

            {aal && (
              <span className="shrink-0 rounded-md bg-slate-800 px-1.5 py-0.5 font-mono text-[0.62rem] font-bold text-slate-100">
                {aal}
              </span>
            )}

            {info?.downstream?.ipAddress && (
              <span className="font-mono text-[0.67rem] font-semibold text-slate-500">
                {info.downstream.ipAddress}
              </span>
            )}

            {info?.downstream?.clientVersion && (
              <span className="text-[0.67rem] font-medium text-slate-400">
                Client {info.downstream.clientVersion}
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

  React.useEffect(() => {
    setPage(0);
  }, [
    props.userRef?.uid,
    props.userRef?.name,
    props.sessionRef?.uid,
    props.sessionRef?.name,
    props.deviceRef?.uid,
    props.deviceRef?.name,
    props.identityProviderRef?.uid,
    props.identityProviderRef?.name,
    props.from?.seconds,
    props.from?.nanos,
  ]);

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
            identityProviderRef: props.identityProviderRef,
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
        <Paginator
          meta={qry.data.listResponseMeta}
          onPageChange={setPage}
        />
      )}
    </div>
  );
};

export const AuthenticationLogList = (props: {
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
    <div className="flex w-full flex-col gap-4">
      <div className="flex items-center gap-3">
        <span className="shrink-0 text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
          Since
        </span>
        <SelectFromTimestamp onUpdate={setFrom} />
      </div>
      <DoAuthenticationLogViewer {...props} from={from} />
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
