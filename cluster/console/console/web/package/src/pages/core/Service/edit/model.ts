import * as CoreP from "@/apis/corev1/corev1";
import { secretName } from "@/pages/enterprise/CollectorExporter/fields";
import { isConditionComplete } from "@/components/Condition";
import { ConfigKind, getModeInfo, ModeInfo } from "./modes";

export type Config = CoreP.Service_Spec_Config;

export type Spec = CoreP.Service_Spec;

export type SectionID =
  | "upstream"
  | "protocol"
  | "access"
  | "dynamic"
  | "settings";

export type Issues = Record<SectionID, string[]>;

export const createConfigType = (kind: ConfigKind): Config["type"] => {
  switch (kind) {
    case "http":
      return { oneofKind: "http", http: CoreP.Service_Spec_Config_HTTP.create() };
    case "ssh":
      return { oneofKind: "ssh", ssh: CoreP.Service_Spec_Config_SSH.create() };
    case "postgres":
      return {
        oneofKind: "postgres",
        postgres: CoreP.Service_Spec_Config_Postgres.create(),
      };
    case "mysql":
      return {
        oneofKind: "mysql",
        mysql: CoreP.Service_Spec_Config_MySQL.create(),
      };
    case "kubernetes":
      return {
        oneofKind: "kubernetes",
        kubernetes: CoreP.Service_Spec_Config_Kubernetes.create({
          type: {
            oneofKind: "kubeconfig",
            kubeconfig: CoreP.Service_Spec_Config_Kubernetes_Kubeconfig.create({
              type: { oneofKind: "fromSecret", fromSecret: "" },
            }),
          },
        }),
      };
    case "socks5":
      return {
        oneofKind: "socks5",
        socks5: CoreP.Service_Spec_Config_SOCKS5.create(),
      };
    case "rdp":
      return { oneofKind: "rdp", rdp: CoreP.Service_Spec_Config_RDP.create() };
    case "mcp":
      return { oneofKind: "mcp", mcp: CoreP.Service_Spec_Config_MCP.create() };
    case "llm":
      return { oneofKind: "llm", llm: CoreP.Service_Spec_Config_LLM.create() };
    default:
      return { oneofKind: undefined };
  }
};

export const createConfig = (info: ModeInfo) =>
  CoreP.Service_Spec_Config.create({
    upstream: CoreP.Service_Spec_Config_Upstream.create({
      type: { oneofKind: "url", url: "" },
    }),
    type: createConfigType(info.config),
  });

export const cloneForEdit = (item: CoreP.Service) => {
  const next = CoreP.Service.clone(item);
  if (!next.spec) next.spec = CoreP.Service_Spec.create();
  const kind = getModeInfo(next.spec.mode)?.config;
  if (kind && next.spec.config && next.spec.config.type.oneofKind !== kind) {
    next.spec.config.type = createConfigType(kind);
  }
  return next;
};

export type ModeStash = Partial<
  Record<
    number,
    { config?: Config; dynamicConfig?: CoreP.Service_Spec_DynamicConfig }
  >
>;

const pruneConfigForMode = (config: Config, info: ModeInfo) => {
  if (info.upstreamTLS === "none") {
    config.tls = undefined;
    config.clientCertificate = undefined;
  }
  if (info.upstreamTLS === "trust" && config.tls) {
    config.tls.insecureSkipVerify = false;
  }
};

export const pruneAuthorization = (spec: Spec) => {
  const auth = spec.authorization;
  if (!auth) return;
  if (!spec.isAnonymous) auth.enableAnonymous = false;
  if (
    auth.policies.length === 0 &&
    auth.inlinePolicies.length === 0 &&
    !auth.enableAnonymous
  ) {
    spec.authorization = undefined;
  }
};

export const applyExposure = (
  spec: Spec,
  exposure: "private" | "public" | "anonymous",
) => {
  spec.isPublic = exposure !== "private";
  spec.isAnonymous = exposure === "anonymous";
  if (spec.isAnonymous && spec.authorization) {
    spec.authorization.enableAnonymous = true;
  }
  pruneAuthorization(spec);
};

export const switchMode = (
  spec: Spec,
  next: CoreP.Service_Spec_Mode,
  stash: ModeStash,
) => {
  const info = getModeInfo(next);
  if (!info) return;
  const previous = getModeInfo(spec.mode);
  stash[spec.mode] = {
    config: spec.config && CoreP.Service_Spec_Config.clone(spec.config),
    dynamicConfig:
      spec.dynamicConfig &&
      CoreP.Service_Spec_DynamicConfig.clone(spec.dynamicConfig),
  };
  spec.mode = next;

  if (!spec.isPublic || !info.isPublic) applyExposure(spec, "private");
  else if (spec.isAnonymous && !info.isAnonymous) {
    applyExposure(spec, "public");
  }
  if (!info.listenerTLS) spec.isTLS = false;

  if (!previous || previous.config !== info.config) {
    const saved = stash[next];
    spec.config = saved?.config
      ? CoreP.Service_Spec_Config.clone(saved.config)
      : createConfig(info);
    if (saved?.dynamicConfig) {
      spec.dynamicConfig = CoreP.Service_Spec_DynamicConfig.clone(
        saved.dynamicConfig,
      );
    } else if (spec.dynamicConfig) {
      for (const config of spec.dynamicConfig.configs) {
        if (config.type.oneofKind !== info.config) {
          config.type = { oneofKind: undefined };
        }
      }
    }
  }

  if (spec.config) pruneConfigForMode(spec.config, info);
  spec.dynamicConfig?.configs.forEach((config) =>
    pruneConfigForMode(config, info),
  );
};

const KNOWN_SCHEMES = [
  "http",
  "ws",
  "h2c",
  "https",
  "wss",
  "ssh",
  "dns",
  "postgres",
  "postgresql",
  "redis",
  "mysql",
  "mongodb",
  "rdp",
  "amqp",
  "ftp",
  "dot",
  "socks5",
];

export const parseUpstreamURL = (value: string) => {
  const match = /^([a-zA-Z][a-zA-Z0-9+.-]*):\/\/([^/?#]+)/.exec(value.trim());
  if (!match) return undefined;
  const port = /:(\d+)$/.exec(match[2])?.[1];
  return { scheme: match[1].toLowerCase(), port };
};

export const upstreamURLError = (value: string) => {
  if (!value) return undefined;
  const url = parseUpstreamURL(value);
  if (!url) return "Use a full URL with a scheme, like https://example.com";
  if (!url.port && !KNOWN_SCHEMES.includes(url.scheme)) {
    return `Add a port, ${url.scheme}:// has no default port`;
  }
  if (url.port && (Number(url.port) < 1 || Number(url.port) > 65535)) {
    return "The port must be between 1 and 65535";
  }
  return undefined;
};

export const isNameValid = (value: string) =>
  /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(value);

export const nameError = (value: string) =>
  isNameValid(value)
    ? undefined
    : "Use lowercase letters, digits and dashes only";

export const headerNameError = (value: string) =>
  /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(value)
    ? undefined
    : "Invalid header name";

export const pathPrefixError = (value: string) =>
  value.startsWith("/") ? undefined : "Start the path with /";

export const hostPortError = (value: string) =>
  /^[^\s:]+:\d{1,5}$/.test(value) ? undefined : "Use the host:port format";

export const corsOriginError = (value: string) => {
  if (value === "*") return undefined;
  try {
    const url = new URL(value);
    return url.origin === value
      ? undefined
      : `${value} must look like https://example.com`;
  } catch {
    return `${value} must look like https://example.com`;
  }
};

const secretsMissing = (node: unknown): number => {
  if (!node || typeof node !== "object") return 0;
  if (node instanceof Uint8Array) return 0;
  if (Array.isArray(node)) {
    return node.reduce<number>((sum, item) => sum + secretsMissing(item), 0);
  }
  const record = node as Record<string, unknown>;
  if (record.oneofKind === "fromSecret") return record.fromSecret ? 0 : 1;
  return Object.values(record).reduce<number>(
    (sum, item) => sum + secretsMissing(item),
    0,
  );
};

const upstreamIssues = (upstream?: Config["upstream"]) => {
  const ret: string[] = [];
  const type = upstream?.type;
  if (!type) return ret;
  if (type.oneofKind === "url") {
    if (!type.url) ret.push("Set the upstream URL");
    else if (upstreamURLError(type.url)) ret.push("Fix the upstream URL");
  }
  if (type.oneofKind === "loadbalance") {
    const endpoints = type.loadbalance.endpoints;
    if (endpoints.length === 0) ret.push("Add at least one endpoint");
    if (endpoints.some((item) => !item.url || upstreamURLError(item.url))) {
      ret.push("Fix the endpoint URLs");
    }
    const schemes = new Set(
      endpoints.map((item) => parseUpstreamURL(item.url)?.scheme),
    );
    if (schemes.size > 1) ret.push("Use the same URL scheme in all endpoints");
  }
  if (type.oneofKind === "container") {
    const container = type.container;
    if (!container.image) ret.push("Set the container image");
    if (!container.port) ret.push("Set the container port");
    const volumes = container.volumes.map((item) => item.name);
    if (container.volumes.some((item) => !isNameValid(item.name))) {
      ret.push("Name every volume");
    }
    if (new Set(volumes).size !== volumes.length) {
      ret.push("Use unique volume names");
    }
    if (
      container.volumeMounts.some(
        (item) => !volumes.includes(item.name) || !item.mountPath,
      )
    ) {
      ret.push("Complete the volume mounts");
    }
    if (container.env.some((item) => !item.name)) {
      ret.push("Name every environment variable");
    }
  }
  return ret;
};

const pluginIssues = (plugins: any[]) => {
  const ret: string[] = [];
  const names = plugins.map((item) => item.name);
  if (names.some((item) => !isNameValid(item))) ret.push("Name every Plugin");
  if (new Set(names).size !== names.length) ret.push("Use unique Plugin names");
  if (
    plugins.some(
      (item) => !item.condition || !isConditionComplete(item.condition),
    )
  ) {
    ret.push("Complete every Plugin Condition");
  }
  for (const plugin of plugins) {
    const type = plugin.type;
    if (type.oneofKind === "rateLimit") {
      if (!type.rateLimit.limit || !type.rateLimit.window) {
        ret.push(`Set the limit and window of ${plugin.name}`);
      }
    }
    if (type.oneofKind === "tokenRateLimit") {
      if (!type.tokenRateLimit.limit || !type.tokenRateLimit.window) {
        ret.push(`Set the limit and window of ${plugin.name}`);
      }
    }
    if (type.oneofKind === "guardrail") {
      if (type.guardrail.patterns.length === 0) {
        ret.push(`Add a pattern to ${plugin.name}`);
      }
    }
    if (type.oneofKind === "semanticRouter") {
      if (type.semanticRouter.routes.length === 0) {
        ret.push(`Add a route to ${plugin.name}`);
      }
    }
  }
  return ret;
};

const protocolIssues = (config: Config) => {
  const ret: string[] = [];
  const { upstream: _upstream, ...rest } = config;
  if (secretsMissing(rest) > 0) ret.push("Select the missing Secrets");
  const type = config.type;
  if (
    type.oneofKind === "http" ||
    type.oneofKind === "mcp" ||
    type.oneofKind === "llm"
  ) {
    const plugins =
      type.oneofKind === "http"
        ? type.http.plugins
        : type.oneofKind === "mcp"
          ? type.mcp.plugins
          : type.llm.plugins;
    ret.push(...pluginIssues(plugins));
  }
  if (type.oneofKind === "llm" && type.llm.embedding) {
    if (!type.llm.embedding.model) ret.push("Set the embedding model");
  }
  return ret;
};

export const collectIssues = (spec: Spec): Issues => {
  const issues: Issues = {
    upstream: [],
    protocol: [],
    access: [],
    dynamic: [],
    settings: [],
  };
  const config = spec.config;
  const dynamic = spec.dynamicConfig;

  if (!config && !dynamic) {
    issues.upstream.push("Add a Configuration");
  }
  if (config) {
    issues.upstream.push(...upstreamIssues(config.upstream));
    if (secretsMissing(config.upstream) > 0) {
      issues.upstream.push("Select the missing Secrets");
    }
    issues.protocol.push(...protocolIssues(config));
  }
  if (dynamic) {
    const names = dynamic.configs.map((item) => item.name);
    if (names.some((item) => !isNameValid(item) || item === "default")) {
      issues.dynamic.push("Name every Configuration");
    }
    if (new Set(names).size !== names.length) {
      issues.dynamic.push("Use unique Configuration names");
    }
    for (const item of dynamic.configs) {
      issues.dynamic.push(
        ...upstreamIssues(item.upstream),
        ...protocolIssues(item),
      );
    }
    for (const rule of dynamic.rules) {
      if (!rule.condition || !isConditionComplete(rule.condition)) {
        issues.dynamic.push("Complete every rule Condition");
      }
      if (rule.type.oneofKind === "configName") {
        const name = rule.type.configName;
        if (!name || (name !== "default" && !names.includes(name))) {
          issues.dynamic.push("Pick a Configuration in every rule");
        }
      }
    }
    if (
      dynamic.configs.length === 0 &&
      !dynamic.rules.some((item) => item.type.oneofKind !== "configName") &&
      !config
    ) {
      issues.dynamic.push("Add a named Configuration");
    }
  }
  return issues;
};

export { secretName };
