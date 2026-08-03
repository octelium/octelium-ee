import * as CoreP from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import {
  FaChrome,
  FaEdge,
  FaFirefoxBrowser,
  FaInternetExplorer,
  FaOpera,
  FaSafari,
} from "react-icons/fa6";
import { IoBrowsersOutline } from "react-icons/io5";

const humanize = (value?: string): string =>
  value
    ? value
        .replace(/^(TYPE|MODE|L3|L4|AUTHENTICATOR_ACTION)_/, "")
        .toLowerCase()
        .replaceAll("_", " ")
        .replace(/\b\w/g, (char) => char.toUpperCase())
    : "Not available";

const enumName = (values: any, value: number | undefined): string =>
  humanize(value === undefined ? undefined : values[value]);

const hasRef = (ref: any): boolean => !!(ref?.name || ref?.uid);

const bytesToHex = (value: Uint8Array): string =>
  Array.from(value, (byte) => byte.toString(16).padStart(2, "0")).join("");

const formatDuration = (duration: any): string => {
  const kind = duration?.type?.oneofKind;
  if (!kind) return "Not set";
  return `${duration.type[kind]} ${kind}`;
};

const formatLocation = (item: CoreP.Session): string | undefined => {
  const geo = item.status?.authentication?.info?.geoip;
  if (!geo) return undefined;
  return [geo.city?.name, geo.region?.name, geo.country?.name]
    .filter(Boolean)
    .join(", ");
};

const BrowserIcon = (props: { userAgent: string }) => {
  const value = props.userAgent.toLowerCase();
  const Icon = value.includes("edg/")
    ? FaEdge
    : value.includes("opr/") || value.includes("opera")
      ? FaOpera
      : value.includes("firefox/")
        ? FaFirefoxBrowser
        : value.includes("chromium/")
          ? FaChrome
          : value.includes("chrome/") || value.includes("crios/")
            ? FaChrome
            : value.includes("safari/")
              ? FaSafari
              : value.includes("msie") || value.includes("trident/")
                ? FaInternetExplorer
                : IoBrowsersOutline;

  return <Icon className="h-4 w-4 shrink-0 text-slate-500" aria-hidden />;
};

export const getSessionPresentation = (item: CoreP.Session) => {
  const status = item.status;
  const authentication = status?.authentication;
  const info = authentication?.info;
  const connection = status?.connection;
  const isConnectedClient =
    status?.type === CoreP.Session_Status_Type.CLIENT && status.isConnected;

  return {
    status,
    authentication,
    info,
    connection,
    isConnectedClient,
    authenticationMethod: enumName(
      CoreP.Session_Status_Authentication_Info_Type,
      info?.type,
    ),
    aal:
      info?.aal === undefined
        ? undefined
        : CoreP.Session_Status_Authentication_Info_AAL[info.aal]?.toUpperCase(),
    authenticatorAction: enumName(
      CoreP.Session_Status_AuthenticatorAction,
      status?.authenticatorAction,
    ),
    connectionType: enumName(
      CoreP.Session_Status_Connection_Type,
      connection?.type,
    ),
    l3Mode: enumName(
      CoreP.Session_Status_Connection_L3Mode,
      connection?.l3Mode,
    ),
    location: formatLocation(item),
  };
};

const Signal = (props: {
  children: React.ReactNode;
  tone?: "neutral" | "success" | "warning" | "danger";
}) => {
  const colors = {
    neutral: "border-slate-200 bg-slate-50 text-slate-600",
    success: "border-emerald-200 bg-emerald-50 text-emerald-700",
    warning: "border-amber-200 bg-amber-50 text-amber-700",
    danger: "border-red-200 bg-red-50 text-red-700",
  };
  return (
    <span
      className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[0.68rem] font-bold ${colors[props.tone ?? "neutral"]}`}
    >
      {props.children}
    </span>
  );
};

export const SessionCompactSecurityInfo = (props: { item: CoreP.Session }) => {
  const { item } = props;
  const p = getSessionPresentation(item);
  const downstream = p.info?.downstream;
  const action = item.status?.authenticatorAction;
  const expiresAt = item.spec?.expiresAt
    ? Timestamp.toDate(item.spec.expiresAt)
    : undefined;
  const expiresSoon =
    expiresAt &&
    expiresAt.getTime() > Date.now() &&
    expiresAt.getTime() - Date.now() <= 2 * 60 * 60 * 1000;

  return (
    <>
      {item.status?.isLocked && <Signal tone="danger">Locked</Signal>}
      {expiresAt && expiresAt.getTime() <= Date.now() && (
        <Signal tone="danger">Expired</Signal>
      )}
      {expiresSoon && <Signal tone="warning">Expires soon</Signal>}
      {action !== undefined &&
        action !==
          CoreP.Session_Status_AuthenticatorAction
            .AUTHENTICATOR_ACTION_UNSET && (
          <Signal
            tone={
              humanize(
                CoreP.Session_Status_AuthenticatorAction[action],
              ).includes("Required")
                ? "danger"
                : "warning"
            }
          >
            {p.authenticatorAction}
          </Signal>
        )}
      {p.info?.aal !== undefined &&
        p.info.aal !==
          CoreP.Session_Status_Authentication_Info_AAL.AAL_UNSET && (
          <Signal>{p.aal}</Signal>
        )}
      {p.info?.type !== undefined &&
        p.info.type !==
          CoreP.Session_Status_Authentication_Info_Type.TYPE_UNSET && (
          <Signal>{p.authenticationMethod}</Signal>
        )}
      {p.isConnectedClient && p.connection?.lastSeenAt && (
        <Signal tone="success">
          Last seen&nbsp;
          <TimeAgo rfc3339={p.connection.lastSeenAt} />
        </Signal>
      )}
      {p.isConnectedClient &&
        p.connection?.type !== undefined &&
        p.connection.type !==
          CoreP.Session_Status_Connection_Type.TYPE_UNKNOWN && (
          <Signal>{p.connectionType}</Signal>
        )}
      {downstream?.ipAddress && <Signal>{downstream.ipAddress}</Signal>}
      {p.location && <Signal>{p.location}</Signal>}
      {downstream?.clientVersion && (
        <Signal>Client {downstream.clientVersion}</Signal>
      )}
    </>
  );
};

const Section = (props: { title: string; children: React.ReactNode }) => (
  <section className="rounded-lg border border-slate-200 bg-white overflow-hidden">
    <div className="border-b border-slate-100 bg-slate-50/70 px-3 py-2 text-[0.65rem] font-bold uppercase tracking-[0.07em] text-slate-500">
      {props.title}
    </div>
    <div className="grid grid-cols-1 gap-px bg-slate-100 sm:grid-cols-2">
      {props.children}
    </div>
  </section>
);

const Field = (props: {
  label: string;
  children: React.ReactNode;
  full?: boolean;
}) => (
  <div
    className={`${props.full ? "sm:col-span-2" : ""} min-w-0 bg-white px-3 py-2`}
  >
    <div className="text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {props.label}
    </div>
    <div className="mt-0.5 break-words text-[0.75rem] font-semibold text-slate-700">
      {props.children}
    </div>
  </div>
);

const ReferenceField = (props: { label: string; ref: any }) =>
  hasRef(props.ref) ? (
    <Field label={props.label}>
      <ResourceListLabel itemRef={props.ref} />
    </Field>
  ) : null;

const AuthenticationDetails = (props: { item: CoreP.Session }) => {
  const p = getSessionPresentation(props.item);
  const auth = p.authentication;
  const info = p.info;
  const network = info?.geoip?.network;
  const details = info?.details;
  const fido =
    details?.oneofKind === "authenticator" &&
    details.authenticator.info?.type.oneofKind === "fido"
      ? details.authenticator.info.type.fido
      : undefined;

  return (
    <Section title="Current authentication">
      {auth?.setAt && (
        <Field label="Authenticated">
          <TimeAgo rfc3339={auth.setAt} />
        </Field>
      )}
      <Field label="Method">{p.authenticationMethod}</Field>
      {info?.aal !== undefined &&
        info.aal !==
          CoreP.Session_Status_Authentication_Info_AAL.AAL_UNSET && (
          <Field label="Assurance level">{p.aal}</Field>
        )}
      <Field label="Access token duration">
        {formatDuration(auth?.accessTokenDuration)}
      </Field>
      <Field label="Refresh token duration">
        {formatDuration(auth?.refreshTokenDuration)}
      </Field>
      {details?.oneofKind === "identityProvider" && (
        <>
          <ReferenceField
            label="Identity provider"
            ref={details.identityProvider.identityProviderRef}
          />
          {details.identityProvider.identifier && (
            <Field label="Identifier">
              {details.identityProvider.identifier}
            </Field>
          )}
          {details.identityProvider.email && (
            <Field label="Email">{details.identityProvider.email}</Field>
          )}
        </>
      )}
      {details?.oneofKind === "credential" && (
        <>
          <ReferenceField
            label="Credential"
            ref={details.credential.credentialRef}
          />
        </>
      )}
      {details?.oneofKind === "authenticator" && (
        <>
          <ReferenceField
            label="Authenticator"
            ref={details.authenticator.authenticatorRef}
          />
          <Field label="Authenticator type">
            {enumName(
              CoreP.Authenticator_Status_Type,
              details.authenticator.type,
            )}
          </Field>
          <Field label="Authenticator mode">
            {enumName(
              CoreP.Session_Status_Authentication_Info_Authenticator_Mode,
              details.authenticator.mode,
            )}
          </Field>
        </>
      )}
      {fido && (
        <Field label="FIDO posture" full>
          <div className="flex flex-wrap gap-1">
            {fido.userPresent && <Signal>User present</Signal>}
            {fido.userVerified && <Signal tone="success">User verified</Signal>}
            {fido.isPasskey && <Signal>Passkey</Signal>}
            {fido.isHardware && <Signal>Hardware</Signal>}
            {fido.isSoftware && <Signal>Software</Signal>}
            {fido.isAttestationVerified && (
              <Signal tone="success">Attestation verified</Signal>
            )}
          </div>
        </Field>
      )}
      {info?.downstream?.ipAddress && (
        <Field label="Source IP">
          <span className="font-mono">{info.downstream.ipAddress}</span>
        </Field>
      )}
      {p.location && <Field label="Location">{p.location}</Field>}
      {!!network?.asn && (
        <Field label="Network">
          AS{network.asn} {network.organization || network.isp}
        </Field>
      )}
      {info?.downstream?.clientVersion && (
        <Field label="Client version">
          <span className="font-mono">{info.downstream.clientVersion}</span>
        </Field>
      )}
      {info?.downstream?.userAgent && (
        <Field label="User agent" full>
          <span className="flex items-start gap-2">
            <BrowserIcon userAgent={info.downstream.userAgent} />
            <span>{info.downstream.userAgent}</span>
          </span>
        </Field>
      )}
    </Section>
  );
};

const ConnectionDetails = (props: { item: CoreP.Session }) => {
  const p = getSessionPresentation(props.item);
  const connection = p.connection;
  if (!p.isConnectedClient || !connection) return null;
  const requestedServices = connection.serviceOptions?.requestedServices ?? [];

  return (
    <Section title="Connection">
      <Field label="Transport">{p.connectionType}</Field>
      <Field label="Network mode">{p.l3Mode}</Field>
      {connection.startedAt && (
        <Field label="Connected">
          <TimeAgo rfc3339={connection.startedAt} />
        </Field>
      )}
      {connection.lastSeenAt && (
        <Field label="Last seen">
          <TimeAgo rfc3339={connection.lastSeenAt} />
        </Field>
      )}
      {connection.addresses.length > 0 && (
        <Field label="Private networks" full>
          <div className="flex flex-wrap gap-1">
            {connection.addresses
              .flatMap((address) => [address.v4, address.v6])
              .filter(Boolean)
              .map((address) => (
                <Label key={address}>{address}</Label>
              ))}
          </div>
        </Field>
      )}
      <Field label="DNS">
        {connection.ignoreDNS ? "Cluster DNS ignored" : "Cluster DNS enabled"}
      </Field>
      <Field label="Embedded SSH">
        {connection.eSSHEnable
          ? `Enabled on port ${connection.eSSHPort}`
          : "Disabled"}
      </Field>
      <Field label="Embedded SOCKS5">
        {connection.eSOCKS5Enable
          ? `Enabled on port ${connection.eSOCKS5Port}`
          : "Disabled"}
      </Field>
      {connection.x25519PublicKey.length > 0 && (
        <Field label="X25519 key fingerprint">
          <CopyText
            value={bytesToHex(connection.x25519PublicKey)}
            truncate={20}
          />
        </Field>
      )}
      {connection.ed25519PublicKey.length > 0 && (
        <Field label="Ed25519 key fingerprint">
          <CopyText
            value={bytesToHex(connection.ed25519PublicKey)}
            truncate={20}
          />
        </Field>
      )}
      {connection.upstreams.length > 0 && (
        <Field label={`Served upstreams (${connection.upstreams.length})`} full>
          <div className="space-y-1">
            {connection.upstreams.map((upstream, index) => (
              <div key={index} className="flex flex-wrap gap-1">
                <ResourceListLabel itemRef={upstream.serviceRef} />
                <Label>
                  {enumName(
                    CoreP.Session_Status_Connection_Upstream_L4Type,
                    upstream.l4Type,
                  )}{" "}
                  :{upstream.port}
                </Label>
                {upstream.backend?.host && (
                  <Label>
                    {upstream.backend.host}:{upstream.backend.port}
                  </Label>
                )}
              </div>
            ))}
          </div>
        </Field>
      )}
      {connection.publishedServices.length > 0 && (
        <Field
          label={`Published services (${connection.publishedServices.length})`}
          full
        >
          <div className="space-y-1">
            {connection.publishedServices.map((service, index) => (
              <div key={index} className="flex flex-wrap gap-1">
                <ResourceListLabel itemRef={service.serviceRef} />
                <Label>
                  {service.address}:{service.port}
                </Label>
              </div>
            ))}
          </div>
        </Field>
      )}
      {requestedServices.length > 0 && (
        <Field label="Requested services" full>
          <div className="flex flex-wrap gap-1">
            {requestedServices.map((requested, index) => (
              <ResourceListLabel key={index} itemRef={requested.serviceRef} />
            ))}
          </div>
        </Field>
      )}
    </Section>
  );
};

const AuthorizationDetails = (props: { item: CoreP.Session }) => {
  const { item } = props;
  const scopes = item.status?.scopes ?? [];
  if (scopes.length === 0) return null;

  return (
    <Section title="Authorization scopes">
      <Field label="Scopes" full>
        <div className="space-y-1">
          {scopes.map((scope, index) => (
            <div key={index}>
              <Label>
                {scope.type.oneofKind === "service"
                  ? `Service: ${humanize(scope.type.service.type.oneofKind)}`
                  : scope.type.oneofKind === "api"
                    ? `API: ${humanize(scope.type.api.type.oneofKind)}`
                    : "Unknown scope"}
              </Label>
            </div>
          ))}
        </div>
      </Field>
    </Section>
  );
};

const HistoryDetails = (props: { item: CoreP.Session }) => {
  const status = props.item.status;

  return (
    <Section title="History">
      <Field label="Authentication count">
        {status?.totalAuthentications ?? 0}
      </Field>
      {!!status?.totalConnections && status.totalConnections > 0 && (
        <Field label="Connection count">{status.totalConnections}</Field>
      )}
    </Section>
  );
};

export const SessionOperationalDetails = (props: { item: CoreP.Session }) => {
  const { item } = props;
  const p = getSessionPresentation(item);
  const action = item.status?.authenticatorAction;

  return (
    <div className="flex flex-col gap-3">
      <AuthenticationDetails item={item} />
      <Section title="Security status">
        {item.status?.isLocked && (
          <Field label="Locked">
            <Signal tone="danger">Locked</Signal>
          </Field>
        )}
        {action !== undefined &&
          action !==
            CoreP.Session_Status_AuthenticatorAction
              .AUTHENTICATOR_ACTION_UNSET && (
            <Field label="Authenticator action">{p.authenticatorAction}</Field>
          )}
        <ReferenceField
          label="Required authenticator"
          ref={item.status?.requiredAuthenticatorRef}
        />
        <ReferenceField
          label="Session credential"
          ref={item.status?.credentialRef}
        />
      </Section>
      <ConnectionDetails item={item} />
      <AuthorizationDetails item={item} />
      <HistoryDetails item={item} />
      {item.status?.ext && Object.keys(item.status.ext).length > 0 && (
        <Section title="Advanced">
          <Field label="Extension keys" full>
            <div className="flex flex-wrap gap-1">
              {Object.keys(item.status.ext).map((key) => (
                <Label key={key}>{key}</Label>
              ))}
            </div>
          </Field>
        </Section>
      )}
    </div>
  );
};
