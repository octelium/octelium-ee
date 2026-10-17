import { match } from "ts-pattern";

export type ResourceName =
  | ResourceCoreName
  | ResourceEnterpriseName
  | ResourceAccessName
  | ResourceCordiumName;

export type API = "core" | "enterprise" | "access" | "cordium";

export type ResourceCoreName =
  | "Service"
  | "Namespace"
  | "Policy"
  | "Session"
  | "User"
  | "Group"
  | "Device"
  | "Secret"
  | "IdentityProvider"
  | "Gateway"
  | "Region"
  | "Credential"
  | "Config"
  | "ClusterConfig"
  | "Authenticator";

export type ResourceEnterpriseName =
  | "ClusterConfig"
  | "Secret"
  | "Certificate"
  | "CertificateIssuer"
  | "CollectorExporter"
  | "DNSProvider"
  | "SecretStore"
  | "DirectoryProvider";

export type ResourceAccessName =
  | "ClusterConfig"
  | "Catalog"
  | "Policy"
  | "Request"
  | "Review"
  | "Secret"
  | "Integration";

export type ResourceCordiumName =
  | "Workspace"
  | "Template"
  | "Space"
  | "Membership"
  | "GitProvider"
  | "Secret"
  | "UserSecret"
  | "Region";

const coreResourcePathMap = new Map<string, ResourceCoreName>([
  ["services", "Service"],
  ["identityproviders", "IdentityProvider"],
  ["clusterconfig", "ClusterConfig"],
  ["policies", "Policy"],
  ["devices", "Device"],
  ["namespaces", "Namespace"],
  ["users", "User"],
  ["groups", "Group"],
  ["secrets", "Secret"],
  ["sessions", "Session"],
  ["credentials", "Credential"],
  ["configs", "Config"],
  ["regions", "Region"],
  ["gateways", "Gateway"],
  ["authenticators", "Authenticator"],
]);

const enterpriseResourcePathMap = new Map<string, ResourceEnterpriseName>([
  ["clusterconfig", "ClusterConfig"],
  ["secrets", "Secret"],
  ["dnsproviders", "DNSProvider"],
  ["collectorexporters", "CollectorExporter"],
  ["certificates", "Certificate"],
  ["certificateissuers", "CertificateIssuer"],
  ["directoryproviders", "DirectoryProvider"],
  ["secretstores", "SecretStore"],
]);

const accessResourcePathMap = new Map<string, ResourceAccessName>([
  ["catalogs", "Catalog"],
  ["policies", "Policy"],
  ["requests", "Request"],
  ["reviews", "Review"],
  ["secrets", "Secret"],
  ["integrations", "Integration"],
]);

const cordiumResourcePathMap = new Map<string, ResourceCordiumName>([
  ["workspaces", "Workspace"],
  ["templates", "Template"],
  ["spaces", "Space"],
  ["memberships", "Membership"],
  ["gitproviders", "GitProvider"],
  ["secrets", "Secret"],
  ["usersecrets", "UserSecret"],
  ["regions", "Region"],
]);

const coreKindToPathMap = new Map<ResourceCoreName, string>(
  [...coreResourcePathMap].map(([path, kind]) => [kind, path]),
);

const enterpriseKindToPathMap = new Map<ResourceEnterpriseName, string>(
  [...enterpriseResourcePathMap].map(([path, kind]) => [kind, path]),
);

const accessKindToPathMap = new Map<ResourceAccessName, string>(
  [...accessResourcePathMap].map(([path, kind]) => [kind, path]),
);

const cordiumKindToPathMap = new Map<ResourceCordiumName, string>(
  [...cordiumResourcePathMap].map(([path, kind]) => [kind, path]),
);

export const getAPIFromAPIVersion = (arg: string): API | undefined => {
  if (!arg) return undefined;
  switch (arg.split("/").at(0)) {
    case "core":
      return "core";
    case "enterprise":
      return "enterprise";
    case "access":
      return "access";
    case "cordium":
      return "cordium";
    default:
      return undefined;
  }
};

export const getResourcePathFromAPIKind = (arg: APIKind): string => {
  switch (arg.api) {
    case "core":
      return coreKindToPathMap.get(arg.kind as ResourceCoreName) ?? "";
    case "enterprise":
      return (
        enterpriseKindToPathMap.get(arg.kind as ResourceEnterpriseName) ?? ""
      );
    case "access":
      return accessKindToPathMap.get(arg.kind as ResourceAccessName) ?? "";
    case "cordium":
      return cordiumKindToPathMap.get(arg.kind as ResourceCordiumName) ?? "";
    default:
      return "";
  }
};

export interface APIKind {
  api: API;
  kind: ResourceName;
}

export const getAPIKindFromPath = (path: string): APIKind | undefined => {
  const args = path.split("/");
  if (args.length < 3) return undefined;

  const api = match(args[1])
    .with("core", () => "core" as const)
    .with("enterprise", () => "enterprise" as const)
    .with("access", () => "access" as const)
    .with("cordium", () => "cordium" as const)
    .otherwise(() => undefined);

  if (!api) return undefined;

  const kind = match(api)
    .with("core", () => coreResourcePathMap.get(args[2]))
    .with("enterprise", () => enterpriseResourcePathMap.get(args[2]))
    .with("access", () => accessResourcePathMap.get(args[2]))
    .with("cordium", () => cordiumResourcePathMap.get(args[2]))
    .otherwise(() => undefined) as ResourceName | undefined;

  if (!kind) return undefined;

  return { api, kind };
};
