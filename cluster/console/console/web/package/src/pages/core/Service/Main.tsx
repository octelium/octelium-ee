import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import {
  MutedValue,
  OverviewHero,
  OverviewNotice,
  OverviewTone,
  ResourceActionButton,
  TonePill,
} from "@/components/ResourceLayout/overview";
import {
  attrsItems,
  authorizationItems,
  RefLabel,
} from "@/components/ResourceLayout/overviewItems";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceInfoIcon, ResourceMainInfo } from "@/pages/utils/types";
import { getDomain } from "@/utils";
import {
  getServicePrivateFQDN,
  getServicePublicFQDN,
} from "@/utils/octelium";
import { getResourceRef, printServiceMode } from "@/utils/pb";
import {
  AppWindow,
  ArrowLeftRight,
  Ban,
  Boxes,
  Braces,
  BrainCircuit,
  Cable,
  Container,
  Earth,
  ExternalLink,
  Globe2,
  Layers,
  Lock,
  MapPinned,
  Monitor,
  Radio,
  Search,
  Shield,
  ShieldCheck,
  SquareTerminal,
  Tags,
  UserRoundX,
  Waypoints,
} from "lucide-react";
import * as React from "react";
import { GrMysql } from "react-icons/gr";
import { SiPostgresql } from "react-icons/si";
import { VscMcp } from "react-icons/vsc";

const Mode = CoreC.Service_Spec_Mode;

const MODE_ICONS: Partial<Record<CoreC.Service_Spec_Mode, ResourceInfoIcon>> = {
  [Mode.HTTP]: Globe2,
  [Mode.WEB]: AppWindow,
  [Mode.TCP]: Cable,
  [Mode.UDP]: Radio,
  [Mode.SSH]: SquareTerminal,
  [Mode.KUBERNETES]: Boxes,
  [Mode.POSTGRES]: SiPostgresql as ResourceInfoIcon,
  [Mode.MYSQL]: GrMysql as ResourceInfoIcon,
  [Mode.GRPC]: Braces,
  [Mode.DNS]: Search,
  [Mode.SOCKS5]: Shield,
  [Mode.RDP]: Monitor,
  [Mode.RDP_WEB]: Monitor,
  [Mode.MCP]: VscMcp as ResourceInfoIcon,
  [Mode.LLM]: BrainCircuit,
};

export const getModeIcon = (mode?: CoreC.Service_Spec_Mode): ResourceInfoIcon =>
  (mode !== undefined && MODE_ICONS[mode]) || Waypoints;

const enumLabel = (values: Record<number, string>, value: number | undefined) =>
  value === undefined
    ? "Unknown"
    : String(values[value] ?? "Unknown")
        .replace(/^(MODE|PROTOCOL|SSL_MODE)_/, "")
        .toLowerCase()
        .replaceAll("_", " ")
        .replace(/^\w/, (char) => char.toUpperCase());

const getAccess = (item: CoreC.Service) => {
  const spec = item.spec;
  if (spec?.isPublic && spec.isAnonymous)
    return {
      label: "Anonymous",
      hint: "Public without authentication",
      tone: "warning" as OverviewTone,
      icon: UserRoundX,
    };
  if (spec?.isPublic)
    return {
      label: "Public",
      hint: "Clientless access from browsers",
      tone: "info" as OverviewTone,
      icon: Earth,
    };
  return {
    label: "Private",
    hint: "Only reachable by connected clients",
    tone: "neutral" as OverviewTone,
    icon: Lock,
  };
};

const getUpstream = (item: CoreC.Service) => {
  const upstream = item.spec?.config?.upstream;
  const type = upstream?.type;
  if (type?.oneofKind === "url")
    return { kind: "URL", summary: type.url, url: type.url };
  if (type?.oneofKind === "loadbalance") {
    const count = type.loadbalance.endpoints.length;
    return {
      kind: "Load balancer",
      summary: `${count} endpoint${count === 1 ? "" : "s"}`,
      endpoints: type.loadbalance.endpoints,
    };
  }
  if (type?.oneofKind === "container")
    return {
      kind: "Managed container",
      summary: type.container.image,
      container: type.container,
    };
  if (item.status?.managedService?.image)
    return {
      kind: "Cluster-managed",
      summary: item.status.managedService.image,
    };
  return undefined;
};

const isOpenable = (item: CoreC.Service) =>
  !!item.spec?.isPublic &&
  (item.spec.mode === Mode.WEB || item.spec.mode === Mode.HTTP);

const OpenService = (props: { item: CoreC.Service }) => (
  <ResourceActionButton
    tone="primary"
    icon={ExternalLink}
    href={`https://${getServicePublicFQDN(props.item, getDomain())}`}
  >
    Open
  </ResourceActionButton>
);

const Endpoint = (props: { label: string; value: string }) => (
  <span className="inline-flex max-w-full items-center gap-1.5 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs leading-4">
    <span className="shrink-0 font-semibold text-slate-500">{props.label}</span>
    <span aria-hidden="true" className="h-3 w-px shrink-0 bg-slate-200" />
    <span className="min-w-0 font-medium text-slate-800 [overflow-wrap:anywhere]">
      <CopyText value={props.value} />
    </span>
  </span>
);

const ModeGlyph = (props: { mode?: CoreC.Service_Spec_Mode }) => {
  const icon = getModeIcon(props.mode);
  return React.createElement(icon, { size: 20, strokeWidth: 2.1 });
};

const ServiceHero = (props: { item: CoreC.Service }) => {
  const { item } = props;
  const domain = getDomain();
  const spec = item.spec;
  const access = getAccess(item);
  const upstream = getUpstream(item);
  const port = item.status?.port || spec?.port;
  const modeLabel = printServiceMode(spec?.mode ?? Mode.MODE_UNSET);

  return (
    <OverviewHero
      tone={spec?.isDisabled ? "danger" : access.tone === "warning" ? "warning" : "success"}
      icon={<ModeGlyph mode={spec?.mode} />}
      title={`${access.label} ${modeLabel} Service`}
      detail={
        upstream ? (
          <>
            {upstream.kind === "Load balancer"
              ? "Load-balances "
              : upstream.kind === "URL"
                ? "Proxies to "
                : "Runs "}
            <span className="font-semibold text-slate-700">
              {upstream.summary}
            </span>
            {spec?.config?.upstream?.user && (
              <> as {spec.config.upstream.user}</>
            )}
          </>
        ) : (
          access.hint
        )
      }
      meta={
        <>
          <Endpoint
            label="Private"
            value={`${getServicePrivateFQDN(item, domain)}${port ? `:${port}` : ""}`}
          />
          {spec?.isPublic && (
            <Endpoint
              label="Public"
              value={`https://${getServicePublicFQDN(item, domain)}`}
            />
          )}
        </>
      }
    >
      {spec?.isDisabled && (
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          The Service rejects every request until it is enabled again.
        </OverviewNotice>
      )}
      {spec?.isPublic && spec.isAnonymous && (
        <OverviewNotice tone="warning" icon={UserRoundX} title="Anonymous access.">
          Anyone who can reach the public address can use this Service without
          authentication or authorization.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

const secretRef = (name?: string) =>
  name ? (
    <ResourceListLabel
      itemRef={ObjectReference.create({
        apiVersion: "core/v1",
        kind: "Secret",
        name,
      })}
    />
  ) : undefined;

const configItems = (item: CoreC.Service, group: string) => {
  const config = item.spec?.config;
  const type = config?.type;
  const items: { label: string; value: React.ReactNode }[] = [];

  if (type?.oneofKind === "ssh") {
    if (type.ssh.user) items.push({ label: "SSH user", value: type.ssh.user });
    items.push({
      label: "SSH features",
      value: (
        <div className="flex flex-wrap gap-1">
          {type.ssh.eSSHMode && <TonePill tone="info">Embedded SSH</TonePill>}
          <TonePill tone={type.ssh.enableLocalPortForwarding ? "info" : "neutral"}>
            Port forwarding {type.ssh.enableLocalPortForwarding ? "on" : "off"}
          </TonePill>
          <TonePill tone={type.ssh.enableSubsystem ? "info" : "neutral"}>
            Subsystems {type.ssh.enableSubsystem ? "on" : "off"}
          </TonePill>
        </div>
      ),
    });
  }
  if (type?.oneofKind === "postgres") {
    if (type.postgres.user)
      items.push({ label: "Database user", value: type.postgres.user });
    if (type.postgres.database)
      items.push({ label: "Database", value: type.postgres.database });
    items.push({
      label: "SSL mode",
      value: enumLabel(
        CoreC.Service_Spec_Config_Postgres_SSLMode,
        type.postgres.sslMode,
      ),
    });
  }
  if (type?.oneofKind === "mysql") {
    if (type.mysql.user)
      items.push({ label: "Database user", value: type.mysql.user });
    if (type.mysql.database)
      items.push({ label: "Database", value: type.mysql.database });
    items.push({ label: "Upstream TLS", value: type.mysql.isTLS ? "Enabled" : "Disabled" });
  }
  if (type?.oneofKind === "kubernetes") {
    const auth = type.kubernetes.type;
    items.push({
      label: "Kubernetes auth",
      value:
        auth.oneofKind === "kubeconfig"
          ? "Kubeconfig"
          : auth.oneofKind === "bearerToken"
            ? "Bearer token"
            : auth.oneofKind === "clientCertificate"
              ? "Client certificate"
              : "Not set",
    });
  }
  if (type?.oneofKind === "llm") {
    items.push({
      label: "LLM provider",
      value: enumLabel(CoreC.Service_Spec_Config_LLM_Protocol, type.llm.protocol),
    });
    if (type.llm.model?.type.oneofKind === "value")
      items.push({ label: "Model", value: type.llm.model.type.value });
  }
  if (type?.oneofKind === "mcp" && type.mcp.endpoint) {
    items.push({ label: "MCP endpoint", value: type.mcp.endpoint });
  }
  if (type?.oneofKind === "socks5" && type.socks5.isEmbeddedMode) {
    items.push({ label: "SOCKS5", value: "Embedded mode" });
  }
  if (type?.oneofKind === "http") {
    const flags = [
      type.http.isUpstreamHTTP2 && "HTTP/2 upstream",
      type.http.listenHTTP2 && "HTTP/2 listener",
      type.http.enableRequestBuffering && "Request buffering",
      type.http.plugins.length > 0 &&
        `${type.http.plugins.length} plugin${type.http.plugins.length === 1 ? "" : "s"}`,
    ].filter((flag): flag is string => !!flag);
    if (flags.length)
      items.push({
        label: "HTTP features",
        value: (
          <div className="flex flex-wrap gap-1">
            {flags.map((flag) => (
              <TonePill key={flag} tone="info">
                {flag}
              </TonePill>
            ))}
          </div>
        ),
      });
  }
  if (config?.tls?.insecureSkipVerify) {
    items.push({
      label: "Upstream TLS",
      value: <TonePill tone="warning">Certificate verification disabled</TonePill>,
    });
  } else if ((config?.tls?.trustedCAs.length ?? 0) > 0) {
    items.push({
      label: "Upstream TLS",
      value: `${config!.tls!.trustedCAs.length} trusted CA${config!.tls!.trustedCAs.length === 1 ? "" : "s"}`,
    });
  }
  const clientCertificate = config?.clientCertificate?.type;
  if (clientCertificate?.oneofKind === "fromSecret") {
    items.push({
      label: "Client certificate",
      value: secretRef(clientCertificate.fromSecret),
    });
  }

  return items.map((entry) => ({ ...entry, group }));
};

export const AccessLog = (props: { item: CoreC.Service }) => {
  return <AccessLogViewer serviceRef={getResourceRef(props.item)} />;
};

export const MainInfo = (props: { item: CoreC.Service }): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec;
  const status = item.status;
  const ModeIcon = getModeIcon(spec?.mode);
  const access = getAccess(item);
  const upstream = getUpstream(item);
  const dynamic = spec?.dynamicConfig;
  const managed = status?.managedService;
  const replicas =
    upstream?.container?.replicas || spec?.deployment?.replicas || 0;

  return {
    header: <ServiceHero item={item} />,
    actions: isOpenable(item) ? <OpenService item={item} /> : undefined,
    groupOrder: [
      { title: "Endpoints", icon: Globe2 },
      { title: "Upstream", icon: ArrowLeftRight },
      { title: "Deployment", icon: Container },
      { title: "Authorization", icon: ShieldCheck },
      { title: "Attributes", icon: Tags },
    ],
    items: [
      {
        label: "Mode",
        primary: true,
        icon: ModeIcon,
        value: <Label>{printServiceMode(spec?.mode ?? Mode.MODE_UNSET)}</Label>,
      },
      {
        label: "Access",
        primary: true,
        icon: access.icon,
        value: (
          <span className="flex flex-col">
            <span className="font-semibold">{access.label}</span>
            <span className="text-xs font-normal text-slate-500">
              {access.hint}
            </span>
          </span>
        ),
      },
      {
        label: "Namespace",
        primary: true,
        icon: Layers,
        value: status?.namespaceRef?.name || status?.namespaceRef?.uid ? (
          <RefLabel itemRef={status.namespaceRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Region",
        primary: true,
        icon: MapPinned,
        value: status?.regionRef?.name || status?.regionRef?.uid ? (
          <RefLabel itemRef={status.regionRef} />
        ) : (
          <MutedValue>{spec?.region || "default"}</MutedValue>
        ),
      },

      ...(status?.port || spec?.port
        ? [
            {
              label: "Port",
              group: "Endpoints",
              value: (
                <span className="font-semibold tabular-nums">
                  {status?.port || spec?.port}
                </span>
              ),
            },
          ]
        : []),
      {
        label: "Listener TLS",
        group: "Endpoints",
        value: spec?.isTLS ? (
          <TonePill tone="success" icon={Lock}>
            Enabled
          </TonePill>
        ) : (
          <MutedValue>Disabled</MutedValue>
        ),
      },
      ...(status?.primaryHostname || status?.additionalHostnames.length
        ? [
            {
              label: "Hostnames",
              group: "Endpoints",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {[status!.primaryHostname, ...status!.additionalHostnames]
                    .filter(Boolean)
                    .map((hostname, index) => (
                      <ResourceListLabel
                        key={hostname}
                        label={index === 0 ? "Primary" : "Additional"}
                      >
                        <CopyText value={hostname} />
                      </ResourceListLabel>
                    ))}
                </div>
              ),
            },
          ]
        : []),
      ...(status?.addresses.length
        ? [
            {
              label: "Private addresses",
              group: "Endpoints",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {status.addresses.flatMap((address) =>
                    [address.dualStackIP?.ipv4, address.dualStackIP?.ipv6]
                      .filter((value): value is string => !!value)
                      .map((value) => (
                        <ResourceListLabel
                          key={value}
                          label={value.includes(":") ? "IPv6" : "IPv4"}
                        >
                          <CopyText value={value} />
                        </ResourceListLabel>
                      )),
                  )}
                </div>
              ),
            },
          ]
        : []),

      ...(upstream
        ? [
            {
              label: "Type",
              group: "Upstream",
              value: <span className="font-semibold">{upstream.kind}</span>,
            },
          ]
        : [
            {
              label: "Type",
              group: "Upstream",
              value: <MutedValue>Not configured</MutedValue>,
            },
          ]),
      ...(upstream?.url
        ? [
            {
              label: "URL",
              group: "Upstream",
              span: "full" as const,
              value: <CopyText value={upstream.url} />,
            },
          ]
        : []),
      ...(upstream?.endpoints?.length
        ? [
            {
              label: "Endpoints",
              group: "Upstream",
              span: "full" as const,
              value: (
                <div className="flex flex-col gap-1">
                  {upstream.endpoints.map((endpoint, index) => (
                    <span
                      key={`${endpoint.url}-${index}`}
                      className="flex flex-wrap items-center gap-1.5"
                    >
                      <CopyText value={endpoint.url} />
                      {endpoint.user && (
                        <span className="text-xs text-slate-500">
                          as {endpoint.user}
                        </span>
                      )}
                    </span>
                  ))}
                </div>
              ),
            },
          ]
        : []),
      ...(upstream?.container
        ? [
            {
              label: "Image",
              group: "Upstream",
              span: "full" as const,
              value: <CopyText value={upstream.container.image} />,
            },
            ...(upstream.container.port
              ? [
                  {
                    label: "Container port",
                    group: "Upstream",
                    value: upstream.container.port,
                  },
                ]
              : []),
          ]
        : []),
      ...(spec?.config?.upstream?.user
        ? [
            {
              label: "Upstream user",
              group: "Upstream",
              value: spec.config.upstream.user,
            },
          ]
        : []),
      ...configItems(item, "Upstream"),
      ...(dynamic && (dynamic.rules.length > 0 || dynamic.configs.length > 0)
        ? [
            {
              label: "Dynamic configuration",
              hint: "Overrides the config per request",
              group: "Upstream",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap items-center gap-1">
                  <TonePill tone="info">
                    {dynamic.rules.length} rule
                    {dynamic.rules.length === 1 ? "" : "s"}
                  </TonePill>
                  {dynamic.configs.map((config, index) => (
                    <ResourceListLabel key={`${config.name}-${index}`} label="Config">
                      {config.name || `#${index + 1}`}
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(replicas > 0
        ? [
            {
              label: "Replicas",
              group: "Deployment",
              value: <span className="font-semibold tabular-nums">{replicas}</span>,
            },
          ]
        : []),
      ...(managed
        ? [
            {
              label: "Managed by",
              group: "Deployment",
              value: (
                <span className="font-semibold">
                  The Cluster{managed.type ? ` (${managed.type})` : ""}
                </span>
              ),
            },
            ...(managed.image
              ? [
                  {
                    label: "Image",
                    group: "Deployment",
                    span: "full" as const,
                    value: <CopyText value={managed.image} />,
                  },
                ]
              : []),
          ]
        : []),

      ...authorizationItems(spec?.authorization),
      ...(spec?.authorization?.enableAnonymous
        ? [
            {
              label: "Anonymous access",
              group: "Authorization",
              value: <TonePill tone="warning">Enabled</TonePill>,
            },
          ]
        : []),
      ...attrsItems(spec?.attrs),
    ],
  };
};
