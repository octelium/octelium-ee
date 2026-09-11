import {
  Authenticator_Status_Type,
  Credential_Spec_Type,
  IdentityProvider_Status_Type,
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
import { ListLoading } from "@/components/Loading";
import { isDev } from "@/utils";
import {
  getClientCore,
  getClientVisibilityAuthenticationLog,
} from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, RefreshCw } from "lucide-react";
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
        "shrink-0 rounded-md border px-1.5 py-px text-[10px] font-semibold leading-4",
        className,
      )}
    >
      {label}
    </span>
  );
};

const BoolChip = ({ label }: { label: string }) => (
  <span className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-px text-[10px] font-semibold leading-4 text-slate-600">
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
    <div className="border-t border-slate-200 bg-slate-50/70">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-3.5 py-2.5 sm:px-4">
        <h4 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
          Event details
        </h4>
        <Editor item={x} />
      </div>

      <div className="grid grid-cols-1 gap-3 px-3.5 py-3 sm:grid-cols-2 sm:px-4 lg:grid-cols-3 xl:grid-cols-4">
        {x.metadata?.id && (
          <DetailField label="Log ID" mono>
            <CopyText value={x.metadata.id} />
          </DetailField>
        )}
        {entry && (
          <DetailField label="Authentication number">
            #{entry.authenticationIndex + 1}
          </DetailField>
        )}

        {entry?.sessionRef && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Session
            </span>
            <CardSession itemRef={entry.sessionRef} />
          </div>
        )}

        {(entry?.userRef || entry?.deviceRef) && (
          <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
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
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Authentication details
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
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
            {info.details.identityProvider.type > 0 && (
              <DetailField label="Provider type">
                {
                  IdentityProvider_Status_Type[
                    info.details.identityProvider.type
                  ]
                }
              </DetailField>
            )}
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
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
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
            {info.details.credential.type > 0 && (
              <DetailField label="Credential type">
                {Credential_Spec_Type[info.details.credential.type]}
              </DetailField>
            )}
            {info.details.credential.credentialRef && (
              <div className="flex flex-col gap-0.5">
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
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
                <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                  Authenticator
                </span>
                <ResourceListLabel
                  itemRef={info.details.authenticator.authenticatorRef}
                />
              </div>
            )}
            {info.details.authenticator.type > 0 && (
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

        {info?.details.oneofKind === "external" &&
          info.details.external.ownerRef && (
            <div className="col-span-full min-w-0 border-t border-slate-200 pt-3">
              <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
                External owner
              </span>
              <ResourceListLabel itemRef={info.details.external.ownerRef} />
            </div>
          )}

        {info?.geoip && (
          <div className="col-span-full border-t border-slate-200 pt-3">
            <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              Network location
            </span>
            <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {info.geoip.ip && (
                <DetailField label="IP address" mono>
                  {info.geoip.ip}
                </DetailField>
              )}
              {(info.geoip.city?.name || info.geoip.country?.name) && (
                <DetailField label="Location">
                  {[info.geoip.city?.name, info.geoip.country?.name]
                    .filter(Boolean)
                    .join(", ")}
                </DetailField>
              )}
              {(info.geoip.region?.name || info.geoip.region?.code) && (
                <DetailField label="Region">
                  {info.geoip.region.name || info.geoip.region.code}
                </DetailField>
              )}
              {(info.geoip.network?.asn ||
                info.geoip.network?.organization ||
                info.geoip.network?.isp) && (
                <DetailField label="Network">
                  {[
                    info.geoip.network.asn
                      ? `AS${info.geoip.network.asn}`
                      : undefined,
                    info.geoip.network.organization,
                    info.geoip.network.isp,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </DetailField>
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
  const userName =
    entry.userRef?.name ??
    entry.userRef?.uid ??
    x.metadata?.actorRef?.name ??
    x.metadata?.actorRef?.uid;
  const sessionName = entry.sessionRef?.name ?? entry.sessionRef?.uid;
  const detailName =
    info?.details.oneofKind === "identityProvider"
      ? info.details.identityProvider.email ||
        info.details.identityProvider.identityProviderRef?.name
      : info?.details.oneofKind === "credential"
        ? info.details.credential.credentialRef?.name ||
          info.details.credential.tokenID
        : info?.details.oneofKind === "authenticator"
          ? info.details.authenticator.authenticatorRef?.name
          : undefined;
  const location = [info?.geoip?.city?.name, info?.geoip?.country?.code]
    .filter(Boolean)
    .join(", ");

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
          className="h-9 w-1 shrink-0 rounded-full bg-sky-500"
        />

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span
              className="max-w-56 truncate text-body font-semibold text-slate-800"
              title={userName}
            >
              {userName ?? "Unknown user"}
            </span>
            {authType != null &&
              authType !==
                Session_Status_Authentication_Info_Type.TYPE_UNSET && (
                <AuthTypeBadge type={authType} />
              )}

            {aal && (
              <span className="shrink-0 rounded-md border border-slate-300 bg-slate-100 px-1.5 py-px font-mono text-[10px] font-semibold leading-4 text-slate-700">
                {aal}
              </span>
            )}
          </span>

          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-[10px] font-medium text-slate-500">
            <span>Authentication #{entry.authenticationIndex + 1}</span>
            {sessionName && (
              <span className="max-w-44 truncate" title={sessionName}>
                Session {sessionName}
              </span>
            )}
            {info?.downstream?.ipAddress && (
              <span className="font-mono">
                {info.downstream.ipAddress}
              </span>
            )}
            {info?.downstream?.clientVersion && (
              <span>Client {info.downstream.clientVersion}</span>
            )}
            {location && <span>{location}</span>}
            {x.metadata?.createdAt && (
              <TimeAgo rfc3339={x.metadata.createdAt} />
            )}
          </span>
        </span>

        {detailName && (
          <span className="hidden max-w-48 shrink-0 text-right sm:block">
            <span className="block text-[9px] font-semibold uppercase tracking-[0.06em] text-slate-500">
              Authentication source
            </span>
            <span className="block truncate text-xs font-semibold text-slate-700">
              {detailName}
            </span>
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
            <AuthenticationLogDetails authLog={x} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const getListAuthenticationLogResponseTest = async () => {
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
  const identityProviderRef = ObjectReference.create({
    uid: "dev-idp",
    name: "google-workspace",
  });
  const authenticatorRef = ObjectReference.create({
    uid: "dev-authenticator",
    name: "alice-passkey",
  });
  const credentialRef = ObjectReference.create({
    uid: "dev-credential",
    name: "ci-access-token",
  });
  const createdAt = (minutesAgo: number) =>
    Timestamp.fromDate(dayjs().subtract(minutesAgo, "minute").toDate());
  const makeLog = (
    id: string,
    minutesAgo: number,
    info: Record<string, unknown>,
  ) =>
    AuthenticationLog.create({
      kind: "AuthenticationLog",
      metadata: { id, createdAt: createdAt(minutesAgo), actorRef: userRef },
      entry: {
        sessionRef,
        userRef,
        deviceRef,
        authentication: { info: info as any },
      },
    });
  const items = [
    makeLog("dev-auth-idp", 1, {
      type: Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
      aal: Session_Status_Authentication_Info_AAL.AAL2,
      downstream: {
        ipAddress: "203.0.113.24",
        userAgent: "Mozilla/5.0 Chrome/126.0",
        clientVersion: "0.15.2",
      },
      details: {
        oneofKind: "identityProvider",
        identityProvider: {
          email: "alice@example.com",
          identifier: "00u-dev-alice",
          identityProviderRef,
        },
      },
    }),
    makeLog("dev-auth-passkey", 4, {
      type: Session_Status_Authentication_Info_Type.AUTHENTICATOR,
      aal: Session_Status_Authentication_Info_AAL.AAL3,
      downstream: {
        ipAddress: "198.51.100.18",
        userAgent: "Mozilla/5.0 Safari/17.5",
        clientVersion: "0.15.2",
      },
      details: {
        oneofKind: "authenticator",
        authenticator: {
          authenticatorRef,
          type: Authenticator_Status_Type.FIDO,
          mode: 1,
          info: {
            type: {
              oneofKind: "fido",
              fido: {
                isPasskey: true,
                isHardware: true,
                isSoftware: false,
                isAttestationVerified: true,
                userVerified: true,
                userPresent: true,
                aaguid: "adce0002-35bc-c60a-648b-0b25f1f05503",
              },
            },
          },
        },
      },
    }),
    makeLog("dev-auth-credential", 9, {
      type: Session_Status_Authentication_Info_Type.CREDENTIAL,
      aal: Session_Status_Authentication_Info_AAL.AAL1,
      downstream: {
        ipAddress: "10.20.4.17",
        userAgent: "octelium-cli/0.15.2",
        clientVersion: "0.15.2",
      },
      details: {
        oneofKind: "credential",
        credential: {
          credentialRef,
          tokenID: "tok_dev_7e2f9c4d",
        },
      },
    }),
    makeLog("dev-auth-refresh", 16, {
      type: Session_Status_Authentication_Info_Type.REFRESH_TOKEN,
      aal: Session_Status_Authentication_Info_AAL.AAL2,
      downstream: {
        ipAddress: "203.0.113.24",
        userAgent: "octelium-client/0.15.2 linux/amd64",
        clientVersion: "0.15.2",
      },
    }),
    makeLog("dev-auth-external", 28, {
      type: Session_Status_Authentication_Info_Type.EXTERNAL,
      aal: Session_Status_Authentication_Info_AAL.AAL1,
      downstream: {
        ipAddress: "192.0.2.55",
        userAgent: "External-OIDC-Bridge/2.4",
        clientVersion: "2.4.0",
      },
    }),
  ];

  return ListAuthenticationLogResponse.create({
    items,
    listResponseMeta: {
      totalCount: items.length,
      page: 0,
      itemsPerPage: items.length,
      hasMore: false,
    },
  });
};

const DoAuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  itemsPerPage?: number;
  from?: Timestamp;
  page?: number;
  onPageChange?: (page: number) => void;
  query?: string;
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
    queryKey: [
      "visibility",
      "listAuthenticationLog",
      {
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        deviceRef: props.deviceRef,
        identityProviderRef: props.identityProviderRef,
        credentialRef: props.credentialRef,
        authenticatorRef: props.authenticatorRef,
        page,
        from: props.from,
        query: props.query,
      },
    ],
    queryFn: async () => {
      if (isDev()) {
        const response = await getListAuthenticationLogResponseTest();
        const matchesRef = (candidate?: ObjectReference, wanted?: ObjectReference) =>
          !wanted ||
          Boolean(
            (candidate?.uid && wanted.uid && candidate.uid === wanted.uid) ||
              (candidate?.name && wanted.name && candidate.name === wanted.name),
          );
        const items = response.items.filter(
          (item) => {
            const details = (item.entry?.authentication?.info as any)?.details;
            const identityProviderRef =
              details?.oneofKind === "identityProvider"
                ? details.identityProvider?.identityProviderRef
                : undefined;
            const credentialRef =
              details?.oneofKind === "credential"
                ? details.credential?.credentialRef
                : undefined;
            const authenticatorRef =
              details?.oneofKind === "authenticator"
                ? details.authenticator?.authenticatorRef
                : undefined;
            return (
              matchesRef(item.entry?.userRef, props.userRef) &&
              matchesRef(item.entry?.sessionRef, props.sessionRef) &&
              matchesRef(item.entry?.deviceRef, props.deviceRef) &&
              matchesRef(identityProviderRef, props.identityProviderRef) &&
              matchesRef(credentialRef, props.credentialRef) &&
              matchesRef(authenticatorRef, props.authenticatorRef)
            );
          },
        );
        return ListAuthenticationLogResponse.create({
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

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLog(
          ListAuthenticationLogRequest.create({
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
            common: {
              page,
              itemsPerPage: props.itemsPerPage ?? 25,
              query: props.query,
            },
            from: props.from,
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
            ? "Refresh failed. Showing the last loaded authentication logs."
            : "Authentication logs could not be loaded. Refresh to try again."}
        </div>
      )}

      {qry.isLoading && !qry.data ? (
        <ListLoading label="authentication logs" />
      ) : qry.data ? (
        <>
          {qry.data?.items.map((x) => (
            <AuthenticationLogC key={x.metadata!.id} authLog={x} />
          ))}

          {qry.isSuccess && qry.data?.items.length === 0 && (
            <div className="flex items-center justify-center py-16">
              <span className="text-body font-semibold uppercase tracking-[0.08em] text-slate-500">
                No authentication log entries found
              </span>
            </div>
          )}
        </>
      ) : null}

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

export const AuthenticationLogList = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
  onPageChange?: (page: number) => void;
  periodMinutes?: number;
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
  const from = controlledFrom ?? localFrom;

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
      <DoAuthenticationLogViewer {...props} from={from} />
    </div>
  );
};

const AuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
  onPageChange?: (page: number) => void;
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
          <AuthenticationLogSummary
            userRef={props.userRef}
            sessionRef={props.sessionRef}
            deviceRef={props.deviceRef}
            identityProviderRef={props.identityProviderRef}
            credentialRef={props.credentialRef}
            authenticatorRef={props.authenticatorRef}
            from={from}
          />
        </CompactSummary>
      )}

      <DoAuthenticationLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        deviceRef={props.deviceRef}
        identityProviderRef={props.identityProviderRef}
        credentialRef={props.credentialRef}
        authenticatorRef={props.authenticatorRef}
        from={from}
        itemsPerPage={props.itemsPerPage}
        page={props.page}
        onPageChange={props.onPageChange}
        query={props.query}
      />
    </div>
  );
};

export default AuthenticationLogViewer;
