import * as CoreP from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import {
  formatDuration,
  formatSeconds,
  HistoryList,
  MutedValue,
  OverviewTone,
  toDate,
  TonePill,
} from "@/components/ResourceLayout/overview";
import { hasRef, RefLabel } from "@/components/ResourceLayout/overviewItems";
import { ResourceListLabel } from "@/components/ResourceList";
import { CountryFlag } from "@/components/SelectCountry";
import TimeAgo from "@/components/TimeAgo";
import { ResourceInfoMainItem } from "@/pages/utils/types";
import { Globe, SquareTerminal } from "lucide-react";
import { IconType } from "react-icons";
import {
  FaAndroid,
  FaApple,
  FaChrome,
  FaEdge,
  FaFirefoxBrowser,
  FaInternetExplorer,
  FaLinux,
  FaOpera,
  FaSafari,
  FaWindows,
} from "react-icons/fa6";
import { IoBrowsersOutline } from "react-icons/io5";

const humanize = (value?: string): string =>
  value
    ? value
        .replace(/^(TYPE|MODE|L3|L4|AUTHENTICATOR_ACTION)_/, "")
        .toLowerCase()
        .replaceAll("_", " ")
        .replace(/^\w/, (char) => char.toUpperCase())
    : "Not available";

const enumName = (values: any, value: number | undefined): string =>
  humanize(value === undefined ? undefined : values[value]);

const AuthType = CoreP.Session_Status_Authentication_Info_Type;
const AAL = CoreP.Session_Status_Authentication_Info_AAL;
const Action = CoreP.Session_Status_AuthenticatorAction;

export type Authentication = CoreP.Session_Status_Authentication;

interface Client {
  name: string;
  version?: string;
  icon: IconType | typeof SquareTerminal;
}

interface Platform {
  name: string;
  icon?: IconType;
}

const browsers: [RegExp, string, IconType][] = [
  [/Edg(?:e|A|iOS)?\/([\d.]+)/, "Edge", FaEdge],
  [/(?:OPR|Opera)\/([\d.]+)/, "Opera", FaOpera],
  [/(?:Firefox|FxiOS)\/([\d.]+)/, "Firefox", FaFirefoxBrowser],
  [/(?:Chrome|CriOS|Chromium)\/([\d.]+)/, "Chrome", FaChrome],
  [/Version\/([\d.]+).*Safari\//, "Safari", FaSafari],
  [/(?:MSIE |Trident\/.*rv:)([\d.]+)/, "Internet Explorer", FaInternetExplorer],
];

const platforms: [RegExp, string, IconType][] = [
  [/iPhone|iPad|iPod/, "iOS", FaApple],
  [/Android/, "Android", FaAndroid],
  [/CrOS/, "ChromeOS", FaChrome],
  [/Windows/, "Windows", FaWindows],
  [/Mac OS X|Macintosh|darwin/i, "macOS", FaApple],
  [/Linux/i, "Linux", FaLinux],
];

export const parseUserAgent = (
  userAgent?: string,
  clientVersion?: string,
): { client?: Client; platform?: Platform } => {
  const value = userAgent ?? "";
  const platform = platforms.find(([pattern]) => pattern.test(value));

  if (clientVersion || /^octelium/i.test(value)) {
    return {
      client: {
        name: "Octelium client",
        version: clientVersion || /\/([\w.-]+)/.exec(value)?.[1],
        icon: SquareTerminal,
      },
      platform: platform && { name: platform[1], icon: platform[2] },
    };
  }

  const browser = browsers.find(([pattern]) => pattern.test(value));
  return {
    client: browser
      ? {
          name: browser[1],
          version: browser[0].exec(value)?.[1]?.split(".")[0],
          icon: browser[2],
        }
      : undefined,
    platform: platform && { name: platform[1], icon: platform[2] },
  };
};

export const ClientIcon = (props: { item: CoreP.Session; size?: number }) => {
  const status = props.item.status;
  const downstream = status?.authentication?.info?.downstream;
  const size = props.size ?? 20;
  if (status?.type === CoreP.Session_Status_Type.CLIENT)
    return <SquareTerminal size={size} strokeWidth={2.1} />;
  const { client } = parseUserAgent(
    downstream?.userAgent,
    downstream?.clientVersion,
  );
  if (client) {
    const Icon = client.icon;
    return <Icon size={size} />;
  }
  return status?.isBrowser ? (
    <IoBrowsersOutline size={size} />
  ) : (
    <Globe size={size} strokeWidth={2.1} />
  );
};

export const describeClient = (item: CoreP.Session) => {
  const downstream = item.status?.authentication?.info?.downstream;
  const { client, platform } = parseUserAgent(
    downstream?.userAgent,
    downstream?.clientVersion,
  );
  const name = client
    ? `${client.name}${client.version ? ` ${client.version}` : ""}`
    : item.status?.type === CoreP.Session_Status_Type.CLIENT
      ? "Octelium client"
      : item.status?.isBrowser
        ? "Web browser"
        : "Clientless access";
  return platform ? `${name} on ${platform.name}` : name;
};

export const getMethodLabel = (type?: CoreP.Session_Status_Authentication_Info_Type) => {
  switch (type) {
    case AuthType.IDENTITY_PROVIDER:
      return "Identity provider";
    case AuthType.CREDENTIAL:
      return "Credential";
    case AuthType.AUTHENTICATOR:
      return "Authenticator";
    case AuthType.REFRESH_TOKEN:
      return "Refresh token";
    case AuthType.EXTERNAL:
      return "External owner";
    case AuthType.INTERNAL:
      return "Internal";
    default:
      return "Not available";
  }
};

export const getAALMeta = (aal?: CoreP.Session_Status_Authentication_Info_AAL) => {
  switch (aal) {
    case AAL.AAL1:
      return { label: "AAL1", hint: "Single factor", tone: "neutral" as const };
    case AAL.AAL2:
      return { label: "AAL2", hint: "Two factors", tone: "success" as const };
    case AAL.AAL3:
      return {
        label: "AAL3",
        hint: "Hardware-backed factors",
        tone: "success" as const,
      };
    default:
      return undefined;
  }
};

export const getAuthenticationSource = (
  auth?: Authentication,
): ObjectReference | undefined => {
  const details = auth?.info?.details;
  switch (details?.oneofKind) {
    case "identityProvider":
      return details.identityProvider.identityProviderRef;
    case "credential":
      return details.credential.credentialRef;
    case "authenticator":
      return details.authenticator.authenticatorRef;
    case "external":
      return details.external.ownerRef;
    default:
      return undefined;
  }
};

export const getSignIn = (item: CoreP.Session) =>
  [item.status?.authentication, item.status?.initialAuthentication].find(
    (auth) => hasRef(getAuthenticationSource(auth)),
  );

export const formatLocation = (geoip?: CoreP.GeoIP) => {
  if (!geoip) return undefined;
  const city = geoip.city?.name?.trim();
  const region = geoip.region?.name?.trim();
  const country = geoip.country?.name?.trim() || geoip.country?.code?.trim();
  const parts = [city, region !== city ? region : undefined, country].filter(
    Boolean,
  );
  return parts.length ? parts.join(", ") : undefined;
};

export const LocationValue = (props: { info?: CoreP.Session_Status_Authentication_Info }) => {
  const geoip = props.info?.geoip;
  const location = formatLocation(geoip);
  const ip = props.info?.downstream?.ipAddress || geoip?.ip;

  if (!location && !ip) return <MutedValue>Unknown</MutedValue>;

  return (
    <span className="flex min-w-0 flex-col">
      {location && (
        <span className="inline-flex min-w-0 items-center gap-1.5">
          {geoip?.country?.code && <CountryFlag code={geoip.country.code} />}
          <span className="min-w-0">{location}</span>
        </span>
      )}
      {ip && (
        <span className="text-xs font-normal tabular-nums text-slate-500">
          {ip}
        </span>
      )}
    </span>
  );
};

export const getExpiry = (item: CoreP.Session, now: number) => {
  const expiresAt = toDate(item.spec?.expiresAt);
  const left = expiresAt ? (expiresAt.getTime() - now) / 1000 : undefined;
  const tone: OverviewTone =
    left === undefined
      ? "neutral"
      : left <= 0
        ? "danger"
        : left <= 3600
          ? "warning"
          : "neutral";
  return { expiresAt, left, tone, isExpired: left !== undefined && left <= 0 };
};

export const getAuthenticatorAction = (item: CoreP.Session) => {
  switch (item.status?.authenticatorAction) {
    case Action.AUTHENTICATION_REQUIRED:
      return {
        tone: "warning" as const,
        title: "Authenticator authentication required.",
        detail:
          "The Session must authenticate with an Authenticator before it can access any Service.",
      };
    case Action.REGISTRATION_REQUIRED:
      return {
        tone: "warning" as const,
        title: "Authenticator registration required.",
        detail:
          "The User must register an Authenticator before this Session can access any Service.",
      };
    case Action.AUTHENTICATION_RECOMMENDED:
      return {
        tone: "info" as const,
        title: "Authenticator authentication recommended.",
        detail: "MFA is recommended for this Session but not enforced.",
      };
    case Action.REGISTRATION_RECOMMENDED:
      return {
        tone: "info" as const,
        title: "Authenticator registration recommended.",
        detail: "Registering an Authenticator is recommended but not enforced.",
      };
    default:
      return undefined;
  }
};

const FidoPosture = (props: {
  fido: CoreP.Session_Status_Authentication_Info_Authenticator_Info_FIDO;
}) => {
  const { fido } = props;
  const flags: [boolean, string, OverviewTone][] = [
    [fido.userVerified, "User verified", "success"],
    [fido.userPresent, "User present", "neutral"],
    [fido.isPasskey, "Passkey", "neutral"],
    [fido.isHardware, "Hardware", "neutral"],
    [fido.isSoftware, "Software", "neutral"],
    [fido.isAttestationVerified, "Attestation verified", "success"],
  ];
  const visible = flags.filter(([shown]) => shown);
  if (visible.length === 0) return <MutedValue>No posture signals</MutedValue>;
  return (
    <div className="flex flex-wrap gap-1">
      {visible.map(([, label, tone]) => (
        <TonePill key={label} tone={tone}>
          {label}
        </TonePill>
      ))}
    </div>
  );
};

const AuthenticationRow = (props: { auth: Authentication }) => {
  const { auth } = props;
  const source = getAuthenticationSource(auth);
  const aal = getAALMeta(auth.info?.aal);
  return (
    <>
      {getMethodLabel(auth.info?.type)}
      {hasRef(source) && (
        <span className="font-normal"> · {source.name || source.uid}</span>
      )}
      {aal && <span className="font-normal text-slate-500"> · {aal.label}</span>}
    </>
  );
};

export const authenticationItems = (
  item: CoreP.Session,
  group: string,
): ResourceInfoMainItem[] => {
  const status = item.status;
  const auth = status?.authentication;
  const info = auth?.info;
  const details = info?.details;
  const aal = getAALMeta(info?.aal);
  const source = getAuthenticationSource(auth);
  const fido =
    details?.oneofKind === "authenticator" &&
    details.authenticator.info?.type.oneofKind === "fido"
      ? details.authenticator.info.type.fido
      : undefined;
  const history = [...(status?.lastAuthentications ?? [])].sort(
    (a, b) => (toDate(b.setAt)?.getTime() ?? 0) - (toDate(a.setAt)?.getTime() ?? 0),
  );

  return [
    {
      label: "Method",
      group,
      value: (
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="font-semibold">{getMethodLabel(info?.type)}</span>
          {hasRef(source) && <ResourceListLabel itemRef={source} />}
        </span>
      ),
    },
    {
      label: "Assurance level",
      group,
      value: aal ? (
        <span className="flex flex-wrap items-center gap-1.5">
          <TonePill tone={aal.tone}>{aal.label}</TonePill>
          <span className="text-xs text-slate-500">{aal.hint}</span>
        </span>
      ) : (
        <MutedValue>Not set</MutedValue>
      ),
    },
    ...(auth?.setAt
      ? [
          {
            label: "Last authenticated",
            group,
            value: <TimeAgo rfc3339={auth.setAt} />,
          },
        ]
      : []),
    ...(status?.initialAuthentication?.setAt
      ? [
          {
            label: "First authenticated",
            group,
            value: <TimeAgo rfc3339={status.initialAuthentication.setAt} />,
          },
        ]
      : []),
    ...(details?.oneofKind === "identityProvider" &&
    details.identityProvider.identifier
      ? [
          {
            label: "Identity",
            group,
            value: (
              <span className="flex min-w-0 flex-col">
                <span className="font-semibold">
                  {details.identityProvider.identifier}
                </span>
                {details.identityProvider.email && (
                  <span className="text-xs text-slate-500">
                    {details.identityProvider.email}
                  </span>
                )}
              </span>
            ),
          },
        ]
      : []),
    ...(details?.oneofKind === "authenticator"
      ? [
          {
            label: "Authenticator",
            group,
            value: (
              <span className="flex flex-wrap items-center gap-1.5">
                <Label size="sm">
                  {enumName(CoreP.Authenticator_Status_Type, details.authenticator.type)}
                </Label>
                <span className="text-xs text-slate-500">
                  {enumName(
                    CoreP.Session_Status_Authentication_Info_Authenticator_Mode,
                    details.authenticator.mode,
                  )}{" "}
                  mode
                </span>
              </span>
            ),
          },
        ]
      : []),
    ...(fido
      ? [
          {
            label: "FIDO posture",
            group,
            span: "full" as const,
            value: <FidoPosture fido={fido} />,
          },
        ]
      : []),
    {
      label: "Token lifetimes",
      group,
      value: (
        <span className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5">
          <span>
            <span className="text-xs text-slate-500">Access </span>
            <span className="font-semibold">
              {formatDuration(auth?.accessTokenDuration)}
            </span>
          </span>
          <span>
            <span className="text-xs text-slate-500">Refresh </span>
            <span className="font-semibold">
              {formatDuration(auth?.refreshTokenDuration)}
            </span>
          </span>
        </span>
      ),
    },
    {
      label: "Authentications",
      group,
      value: (
        <span className="font-semibold tabular-nums">
          {(status?.totalAuthentications ?? 0).toLocaleString()}
        </span>
      ),
    },
    ...(hasRef(status?.credentialRef) &&
    status!.credentialRef!.uid !== source?.uid &&
    status!.credentialRef!.name !== source?.name
      ? [
          {
            label: "Created by",
            group,
            value: <RefLabel itemRef={status!.credentialRef} />,
          },
        ]
      : []),
    ...(history.length > 0
      ? [
          {
            label: "Previous authentications",
            group,
            layout: "stacked" as const,
            value: (
              <HistoryList
                entries={history.slice(0, 5).map((entry, index) => ({
                  key: `${entry.tokenID}-${entry.setAt?.seconds ?? index}`,
                  tone: "neutral" as const,
                  label: <AuthenticationRow auth={entry} />,
                  meta: (
                    <>
                      {entry.info?.downstream?.ipAddress && (
                        <span className="tabular-nums">
                          {entry.info.downstream.ipAddress}
                        </span>
                      )}
                      {formatLocation(entry.info?.geoip) && (
                        <span>{formatLocation(entry.info?.geoip)}</span>
                      )}
                      {entry.setAt && <TimeAgo rfc3339={entry.setAt} />}
                    </>
                  ),
                }))}
              />
            ),
          },
        ]
      : []),
  ];
};

export const clientItems = (
  item: CoreP.Session,
  group: string,
): ResourceInfoMainItem[] => {
  const info = item.status?.authentication?.info;
  const downstream = info?.downstream;
  const geoip = info?.geoip;
  const { client, platform } = parseUserAgent(
    downstream?.userAgent,
    downstream?.clientVersion,
  );
  const network = geoip?.network;
  const networkLabel = [
    network?.asn ? `AS${network.asn}` : undefined,
    ...new Set(
      [network?.organization?.trim(), network?.isp?.trim()].filter(Boolean),
    ),
    network?.domain?.trim(),
  ].filter(Boolean);
  const offset = geoip?.timezone
    ? `UTC${geoip.timezone.offset >= 0 ? "+" : ""}${geoip.timezone.offset}`
    : undefined;
  const ClientGlyph = client?.icon;
  const PlatformGlyph = platform?.icon;

  return [
    {
      label: "Client",
      group,
      value: (
        <span className="inline-flex items-center gap-1.5 font-semibold">
          {ClientGlyph && (
            <ClientGlyph size={14} className="shrink-0 text-slate-500" />
          )}
          {client
            ? `${client.name}${client.version ? ` ${client.version}` : ""}`
            : describeClient(item)}
        </span>
      ),
    },
    ...(platform
      ? [
          {
            label: "Operating system",
            group,
            value: (
              <span className="inline-flex items-center gap-1.5 font-semibold">
                {PlatformGlyph && (
                  <PlatformGlyph size={14} className="shrink-0 text-slate-500" />
                )}
                {platform.name}
              </span>
            ),
          },
        ]
      : []),
    ...(downstream?.ipAddress
      ? [
          {
            label: "Source IP",
            group,
            value: (
              <span className="tabular-nums">
                <CopyText value={downstream.ipAddress} />
              </span>
            ),
          },
        ]
      : []),
    ...(formatLocation(geoip)
      ? [
          {
            label: "Location",
            group,
            value: <LocationValue info={info} />,
          },
        ]
      : []),
    ...(networkLabel.length
      ? [{ label: "Network", group, value: networkLabel.join(" · ") }]
      : []),
    ...(geoip?.timezone?.id || offset
      ? [
          {
            label: "Timezone",
            group,
            value: (
              <span>
                {geoip?.timezone?.id || offset}
                {geoip?.timezone?.id && offset && (
                  <span className="ml-1 text-xs text-slate-500">{offset}</span>
                )}
              </span>
            ),
          },
        ]
      : []),
    ...(geoip?.coordinates &&
    (geoip.coordinates.latitude !== 0 || geoip.coordinates.longitude !== 0)
      ? [
          {
            label: "Coordinates",
            group,
            value: (
              <span className="tabular-nums">
                {geoip.coordinates.latitude}, {geoip.coordinates.longitude}
                {!!geoip.coordinates.accuracyRadius && (
                  <span className="ml-1 text-xs text-slate-500">
                    ±{geoip.coordinates.accuracyRadius} km
                  </span>
                )}
              </span>
            ),
          },
        ]
      : []),
    ...(downstream?.userAgent
      ? [
          {
            label: "User agent",
            group,
            span: "full" as const,
            value: (
              <span className="text-xs leading-5 text-slate-600">
                {downstream.userAgent}
              </span>
            ),
          },
        ]
      : []),
  ];
};

const toBase64 = (value: Uint8Array) =>
  btoa(String.fromCharCode(...Array.from(value)));

const l3ModeLabel = (mode?: CoreP.Session_Status_Connection_L3Mode) =>
  mode === CoreP.Session_Status_Connection_L3Mode.V4
    ? "IPv4 only"
    : mode === CoreP.Session_Status_Connection_L3Mode.V6
      ? "IPv6 only"
      : "Dual-stack";

const transportLabel = (type?: CoreP.Session_Status_Connection_Type) =>
  type === CoreP.Session_Status_Connection_Type.WIREGUARD
    ? "WireGuard"
    : type === CoreP.Session_Status_Connection_Type.QUICV0
      ? "QUIC"
      : "Unknown";

export const getTransportLabel = transportLabel;

export const connectionItems = (
  item: CoreP.Session,
  group: string,
): ResourceInfoMainItem[] => {
  const status = item.status;
  if (status?.type !== CoreP.Session_Status_Type.CLIENT) return [];
  const connection = status.isConnected ? status.connection : undefined;
  const history = status.lastConnections ?? [];

  return [
    {
      label: "Status",
      group,
      value: connection ? (
        <TonePill tone="success">Connected</TonePill>
      ) : (
        <TonePill>Not connected</TonePill>
      ),
    },
    ...(connection
      ? [
          {
            label: "Transport",
            group,
            value: (
              <span className="font-semibold">
                {transportLabel(connection.type)}
                <span className="ml-1.5 text-xs font-normal text-slate-500">
                  {l3ModeLabel(connection.l3Mode)}
                </span>
              </span>
            ),
          },
          ...(connection.startedAt
            ? [
                {
                  label: "Connected",
                  group,
                  value: <TimeAgo rfc3339={connection.startedAt} />,
                },
              ]
            : []),
          ...(connection.lastSeenAt
            ? [
                {
                  label: "Last seen",
                  group,
                  value: <TimeAgo rfc3339={connection.lastSeenAt} />,
                },
              ]
            : []),
          ...(connection.addresses.length > 0
            ? [
                {
                  label: "Private addresses",
                  group,
                  span: "full" as const,
                  value: (
                    <div className="flex flex-wrap gap-1">
                      {connection.addresses
                        .flatMap((address) => [address.v4, address.v6])
                        .filter(Boolean)
                        .map((address) => (
                          <ResourceListLabel key={address} label="Address">
                            <CopyText value={address} />
                          </ResourceListLabel>
                        ))}
                    </div>
                  ),
                },
              ]
            : []),
          {
            label: "Cluster DNS",
            group,
            value: connection.ignoreDNS ? "Ignored" : "Enabled",
          },
          {
            label: "Embedded servers",
            group,
            value: (
              <div className="flex flex-wrap gap-1">
                <TonePill tone={connection.eSSHEnable ? "info" : "neutral"}>
                  SSH{" "}
                  {connection.eSSHEnable ? `:${connection.eSSHPort}` : "off"}
                </TonePill>
                <TonePill tone={connection.eSOCKS5Enable ? "info" : "neutral"}>
                  SOCKS5{" "}
                  {connection.eSOCKS5Enable
                    ? `:${connection.eSOCKS5Port}`
                    : "off"}
                </TonePill>
              </div>
            ),
          },
          ...(connection.x25519PublicKey.length > 0
            ? [
                {
                  label: "WireGuard public key",
                  group,
                  span: "full" as const,
                  value: (
                    <span className="font-mono text-xs">
                      <CopyText value={toBase64(connection.x25519PublicKey)} />
                    </span>
                  ),
                },
              ]
            : []),
          ...(connection.ed25519PublicKey.length > 0
            ? [
                {
                  label: "Ed25519 public key",
                  group,
                  span: "full" as const,
                  value: (
                    <span className="font-mono text-xs">
                      <CopyText value={toBase64(connection.ed25519PublicKey)} />
                    </span>
                  ),
                },
              ]
            : []),
        ]
      : []),
    {
      label: "Connections",
      group,
      value: (
        <span className="font-semibold tabular-nums">
          {(status.totalConnections ?? 0).toLocaleString()}
        </span>
      ),
    },
    ...(history.length > 0
      ? [
          {
            label: "Previous connections",
            group,
            layout: "stacked" as const,
            value: (
              <HistoryList
                entries={[...history]
                  .sort(
                    (a, b) =>
                      (toDate(b.startedAt)?.getTime() ?? 0) -
                      (toDate(a.startedAt)?.getTime() ?? 0),
                  )
                  .slice(0, 5)
                  .map((entry, index) => {
                    const start = toDate(entry.startedAt);
                    const end = toDate(entry.endedAt);
                    return {
                      key: `${entry.startedAt?.seconds ?? index}`,
                      tone: "neutral" as const,
                      label:
                        start && end
                          ? `Lasted ${formatSeconds((end.getTime() - start.getTime()) / 1000)}`
                          : "Connection",
                      meta: (
                        <>
                          {entry.startedAt && (
                            <span>
                              Started <TimeAgo rfc3339={entry.startedAt} />
                            </span>
                          )}
                          {entry.endedAt && (
                            <span>
                              Ended <TimeAgo rfc3339={entry.endedAt} />
                            </span>
                          )}
                        </>
                      ),
                    };
                  })}
              />
            ),
          },
        ]
      : []),
  ];
};

export const servedItems = (
  item: CoreP.Session,
  group: string,
): ResourceInfoMainItem[] => {
  const connection = item.status?.isConnected
    ? item.status.connection
    : undefined;
  if (!connection) return [];
  const requested = connection.serviceOptions?.requestedServices ?? [];

  return [
    ...(connection.upstreams.length > 0
      ? [
          {
            label: "Served upstreams",
            hint: "Services reached through this client",
            group,
            span: "full" as const,
            value: (
              <ul className="flex flex-col gap-1.5">
                {connection.upstreams.map((upstream, index) => (
                  <li
                    key={`${upstream.serviceRef?.name}-${index}`}
                    className="flex flex-wrap items-center gap-1.5"
                  >
                    <RefLabel itemRef={upstream.serviceRef} />
                    <Label size="sm" outlined>
                      {enumName(
                        CoreP.Session_Status_Connection_Upstream_L4Type,
                        upstream.l4Type,
                      ).toUpperCase()}{" "}
                      :{upstream.port}
                    </Label>
                    {upstream.backend?.host && (
                      <span className="text-xs tabular-nums text-slate-500">
                        → {upstream.backend.host}:{upstream.backend.port}
                      </span>
                    )}
                    {upstream.mode ===
                      CoreP.Session_Status_Connection_Upstream_Mode.ESSH && (
                      <TonePill tone="info">Embedded SSH</TonePill>
                    )}
                  </li>
                ))}
              </ul>
            ),
          },
        ]
      : []),
    ...(connection.publishedServices.length > 0
      ? [
          {
            label: "Published locally",
            hint: "Services exposed on the client host",
            group,
            span: "full" as const,
            value: (
              <ul className="flex flex-col gap-1.5">
                {connection.publishedServices.map((service, index) => (
                  <li
                    key={`${service.serviceRef?.name}-${index}`}
                    className="flex flex-wrap items-center gap-1.5"
                  >
                    <RefLabel itemRef={service.serviceRef} />
                    <span className="text-xs tabular-nums text-slate-500">
                      {service.address || "localhost"}:{service.port}
                    </span>
                  </li>
                ))}
              </ul>
            ),
          },
        ]
      : []),
    ...(connection.serviceOptions?.serveAll || requested.length > 0
      ? [
          {
            label: "Requested to serve",
            group,
            span: "full" as const,
            value: connection.serviceOptions?.serveAll ? (
              <span className="font-semibold">All servable Services</span>
            ) : (
              <div className="flex flex-wrap gap-1">
                {requested.map((entry, index) => (
                  <RefLabel key={index} itemRef={entry.serviceRef} />
                ))}
              </div>
            ),
          },
        ]
      : []),
  ];
};

const scopeLabel = (scope: CoreP.Scope) => {
  const type = scope.type;
  if (type.oneofKind === "service") {
    const service = type.service.type;
    if (service.oneofKind === "all") return ["Services", "All Services"];
    if (service.oneofKind === "filter") {
      const parts = [
        service.filter.names.length ? service.filter.names.join(", ") : "",
        service.filter.namespaces.length
          ? `Namespaces ${service.filter.namespaces.join(", ")}`
          : "",
      ].filter(Boolean);
      return ["Services", parts.join(" · ") || "Filtered"];
    }
    return ["Services", "Unknown"];
  }
  if (type.oneofKind === "api") {
    const api = type.api.type;
    if (api.oneofKind === "all") return ["API", "All operations"];
    if (api.oneofKind === "filter") {
      const parts = [
        ...api.filter.packages,
        ...api.filter.services,
        ...api.filter.methods,
      ];
      return ["API", parts.join(", ") || "Filtered"];
    }
    return ["API", "Unknown"];
  }
  return ["Scope", "Unknown"];
};

export const scopeItems = (
  item: CoreP.Session,
  group: string,
): ResourceInfoMainItem[] => {
  const scopes = item.status?.scopes ?? [];
  if (scopes.length === 0) return [];
  return [
    {
      label: "Scopes",
      hint: "Restrict what the Session can reach",
      group,
      span: "full",
      value: (
        <div className="flex flex-wrap gap-1">
          {scopes.map((scope, index) => {
            const [label, value] = scopeLabel(scope);
            return (
              <ResourceListLabel key={index} label={label}>
                {value}
              </ResourceListLabel>
            );
          })}
        </div>
      ),
    },
  ];
};
