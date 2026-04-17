import Yaml from "js-yaml";

import * as CoreP from "../../apis/corev1/corev1";
import * as EnterpriseP from "../../apis/enterprisev1/enterprisev1";
import * as MetaPB from "../../apis/metav1/metav1";
import * as VisibilityCoreP from "../../apis/visibilityv1/core/vcorev1";

import { queryClient } from "@/utils";
import { match } from "ts-pattern";
import {
  getClientCore,
  getClientEnterprise,
  getClientVisibilityCore,
} from "../client";

export type Resource = ResourceCore | ResourceEnterprise;
export type ResourceList = ResourceCoreList | ResourceEnterpriseList;
export type ResourceName = ResourceCoreName | ResourceEnterpriseName;

export type API = "core" | "enterprise";

export type ResourceCore =
  | CoreP.Service
  | CoreP.Namespace
  | CoreP.Policy
  | CoreP.Session
  | CoreP.User
  | CoreP.Group
  | CoreP.Device
  | CoreP.Secret
  | CoreP.IdentityProvider
  | CoreP.Gateway
  | CoreP.Region
  | CoreP.Credential
  | CoreP.ClusterConfig
  | CoreP.Authenticator;

export type ResourceEnterprise =
  | EnterpriseP.ClusterConfig
  | EnterpriseP.Secret
  | EnterpriseP.Certificate
  | EnterpriseP.CertificateIssuer
  | EnterpriseP.CollectorExporter
  | EnterpriseP.DNSProvider
  | EnterpriseP.SecretStore
  | EnterpriseP.DeviceManager
  | EnterpriseP.DirectoryProvider;

export type ResourceCoreList =
  | CoreP.ServiceList
  | CoreP.NamespaceList
  | CoreP.PolicyList
  | CoreP.SessionList
  | CoreP.UserList
  | CoreP.GroupList
  | CoreP.DeviceList
  | CoreP.SecretList
  | CoreP.IdentityProviderList
  | CoreP.GatewayList
  | CoreP.RegionList
  | CoreP.CredentialList;

export type ResourceEnterpriseList =
  | EnterpriseP.SecretList
  | EnterpriseP.CertificateList
  | EnterpriseP.CertificateIssuerList
  | EnterpriseP.CollectorExporterList
  | EnterpriseP.DNSProviderList
  | EnterpriseP.SecretStoreList
  | EnterpriseP.DeviceManagerList
  | EnterpriseP.DirectoryProviderList;

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
  | "DeviceManager"
  | "DirectoryProvider";

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

const coreKindToPathMap = new Map<ResourceCoreName, string>(
  [...coreResourcePathMap].map(([path, kind]) => [kind, path]),
);

const enterpriseKindToPathMap = new Map<ResourceEnterpriseName, string>(
  [...enterpriseResourcePathMap].map(([path, kind]) => [kind, path]),
);

export const getAPIFromAPIVersion = (arg: string): API | undefined => {
  if (!arg) return undefined;
  switch (arg.split("/").at(0)) {
    case "core":
      return "core";
    case "enterprise":
      return "enterprise";
    default:
      return undefined;
  }
};

export const getAPI = (arg: Resource): API | undefined =>
  getAPIFromAPIVersion(arg.apiVersion);

export const getAPIFromResourceList = (arg: ResourceList): API | undefined =>
  getAPIFromAPIVersion(arg.apiVersion);

export const cloneResource = (arg: Resource): Resource => {
  switch (getAPI(arg)) {
    case "core":
      return CoreP[arg.kind as ResourceCoreName].clone(arg as any) as Resource;
    case "enterprise":
      return EnterpriseP[arg.kind as ResourceEnterpriseName].clone(
        arg as any,
      ) as Resource;
    default:
      return CoreP[arg.kind as ResourceCoreName].clone(arg as any) as Resource;
  }
};

const safeJSONStringify = (obj: unknown): string =>
  JSON.stringify(obj, null, 2);

export const resourceToJSON = (arg: Resource): string => {
  try {
    switch (getAPI(arg)) {
      case "core":
        return safeJSONStringify(
          JSON.parse(
            CoreP[arg.kind as ResourceCoreName].toJsonString(arg as any),
          ),
        );
      case "enterprise":
        return safeJSONStringify(
          JSON.parse(
            EnterpriseP[arg.kind as ResourceEnterpriseName].toJsonString(
              arg as any,
            ),
          ),
        );
      default:
        return "";
    }
  } catch (err) {
    console.error("resourceToJSON err:", err);
    return "";
  }
};

export const resourceToJSONObject = (arg: Resource): any =>
  JSON.parse(resourceToJSON(arg));

export const resourceSpecToJSON = (arg: Resource): string =>
  safeJSONStringify(resourceToJSONObject(arg)["spec"]);

export const resourceSpecToYAML = (arg: Resource): string =>
  jsonToYAML(resourceSpecToJSON(arg));

export const resourceStatusToJSON = (arg: Resource): string =>
  safeJSONStringify(resourceToJSONObject(arg)["status"]);

export const resourceStatusToYAML = (arg: Resource): string =>
  jsonToYAML(resourceStatusToJSON(arg));

export const resourceMetadataToJSON = (arg: Resource): string =>
  safeJSONStringify(resourceToJSONObject(arg)["metadata"]);

export const resourceMetadataToYAML = (arg: Resource): string =>
  jsonToYAML(resourceMetadataToJSON(arg));

export const resourceToYAML = (arg: Resource): string => {
  try {
    return Yaml.dump(JSON.parse(resourceToJSON(arg)));
  } catch (err) {
    console.error("resourceToYAML err:", err);
    return "";
  }
};

const jsonToYAML = (arg: string): string => {
  try {
    return Yaml.dump(JSON.parse(arg));
  } catch (err) {
    console.error("jsonToYAML err:", err);
    return "";
  }
};

export const resourceFromYAML = (arg: string): Resource | undefined => {
  try {
    const obj = Yaml.load(arg) as any;
    if (!obj) return undefined;
    return resourceFromJSON(JSON.stringify(obj));
  } catch (err) {
    console.error("resourceFromYAML err:", err);
    return undefined;
  }
};

export const resourceFromJSON = (arg: string): Resource | undefined => {
  try {
    const obj = JSON.parse(arg);
    if (!obj) return undefined;

    const kind = obj["kind"] as string | undefined;
    const apiVersion = obj["apiVersion"] as string | undefined;
    if (!kind || !apiVersion) return undefined;

    switch (getAPIFromAPIVersion(apiVersion)) {
      case "core":
        return CoreP[kind as ResourceCoreName].fromJsonString(arg);
      case "enterprise":
        return EnterpriseP[kind as ResourceEnterpriseName].fromJsonString(arg);
      default:
        return undefined;
    }
  } catch (err) {
    console.error("resourceFromJSON err:", err);
    return undefined;
  }
};

export const getResourceRef = (arg: Resource): MetaPB.ObjectReference =>
  MetaPB.ObjectReference.create({
    apiVersion: arg.apiVersion,
    kind: arg.kind,
    uid: arg.metadata?.uid,
    name: arg.metadata?.name,
  });

export const getShortName = (arg: Resource): string =>
  getShortNameFromStr(arg.metadata!.name);

export const getShortNameFromRef = (arg: MetaPB.ObjectReference): string =>
  getShortNameFromStr(arg.name);

export const getShortNameFromStr = (arg: string): string =>
  arg.split(".").at(0) ?? "";

export const getResourcePathFromAPIKind = (arg: APIKind): string => {
  switch (arg.api) {
    case "core":
      return coreKindToPathMap.get(arg.kind as ResourceCoreName) ?? "";
    case "enterprise":
      return (
        enterpriseKindToPathMap.get(arg.kind as ResourceEnterpriseName) ?? ""
      );
    default:
      return "";
  }
};

export const getResourcePath = (arg: Resource): string =>
  `/${getAPI(arg)}/${getResourcePathFromAPIKind({ api: getAPI(arg) as API, kind: arg.kind as ResourceName })}/${arg.metadata!.name}`;

export const getResourceListPath = (arg: ResourceList): string => {
  const api = getAPIFromAPIVersion(arg.apiVersion) as API;
  return `/${api}/${getResourcePathFromAPIKind({ api, kind: getKindFromResourceList(arg) })}`;
};

export const getResourceListPathFromResource = (arg: Resource): string =>
  `/${getAPI(arg)}/${getResourcePathFromAPIKind({ api: getAPI(arg) as API, kind: arg.kind as ResourceName })}`;

export const invalidateResource = (arg: Resource) => {
  queryClient.invalidateQueries({
    queryKey: [getGetKey(arg), arg.metadata?.uid],
  });
  queryClient.invalidateQueries({
    queryKey: [getGetKey(arg), arg.metadata?.name],
  });
  queryClient.invalidateQueries({
    queryKey: ["visibility", getAPI(arg), "summary", arg.kind],
  });
};

export const invalidateResourceList = (arg: Resource) => {
  queryClient.invalidateQueries({ queryKey: [getListKey(arg)] });
  queryClient.invalidateQueries({
    queryKey: ["listSelectComponent", getAPI(arg), arg.kind],
  });
  queryClient.invalidateQueries({
    queryKey: ["visibility", getAPI(arg), "summary", arg.kind],
  });
};

export const getKindFromResourceList = (arg: ResourceList): ResourceName =>
  arg.kind.replace(/List$/, "") as ResourceName;

export const invalidateResourceListFromList = (arg: ResourceList) => {
  const kind = getKindFromResourceList(arg);
  const api = getAPIFromAPIVersion(arg.apiVersion);
  queryClient.invalidateQueries({ queryKey: [`${api}.list${kind}`] });
};

export const invalidateKey = (key: string[]) => {
  queryClient.invalidateQueries({ queryKey: key });
};

export const getListKey = (arg: Resource): string =>
  `${getAPI(arg)}.list${arg.kind}`;

export const getGetKey = (arg: Resource): string =>
  `${getAPI(arg)}.get${arg.kind}`;

export const getGetKeyFromRef = (arg: MetaPB.ObjectReference): string =>
  `${getAPIFromAPIVersion(arg.apiVersion)}.get${arg.kind}`;

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
    .otherwise(() => undefined);

  if (!api) return undefined;

  const kind = match(api)
    .with("core", () => coreResourcePathMap.get(args[2]))
    .with("enterprise", () => enterpriseResourcePathMap.get(args[2]))
    .otherwise(() => undefined) as ResourceName | undefined;

  if (!kind) return undefined;

  return { api, kind };
};

export const getListKeyFromResource = (arg: Resource): string =>
  `${getAPI(arg)}.list${arg.kind}`;

export const getListKeyFromPath = (path: string): string => {
  const apiKind = getAPIKindFromPath(path);
  return `${apiKind?.api}.list${apiKind?.kind}`;
};

export const getGetKeyFromPath = (path: string): string => {
  const apiKind = getAPIKindFromPath(path);
  return `${apiKind?.api}.get${apiKind?.kind}`;
};

export const getPB = (arg: Resource) => getPBFromAPI(getAPI(arg) as API);

export const getPBFromAPI = (api: API) =>
  match(api)
    .with("core", () => CoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);

export const getPBResourceListFromAPI = (api: API) =>
  match(api)
    .with("core", () => VisibilityCoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);

export const getClient = (api: API) =>
  match(api)
    .with("core", () => getClientCore())
    .with("enterprise", () => getClientEnterprise())
    .otherwise(() => undefined);

export const getResourceClient = (arg: Resource) => {
  const api = getAPI(arg);
  if (!api) return undefined;
  return getClient(api);
};

export const getPathClient = (path: string) => {
  const apiKind = getAPIKindFromPath(path);
  if (!apiKind) return undefined;
  return getClient(apiKind.api);
};

export const getClientResourceList = (api: API) =>
  match(api)
    .with("core", () => getClientVisibilityCore())
    .with("enterprise", () => getClientEnterprise())
    .otherwise(() => undefined);

export const getClientResourceListP = (api: API) =>
  match(api)
    .with("core", () => VisibilityCoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);

export const printResourceNameWithDisplay = (arg: Resource): string =>
  arg.metadata?.displayName
    ? `${arg.metadata.name} (${arg.metadata.displayName})`
    : (arg.metadata?.name ?? "");

export const printUserWithEmail = (arg: CoreP.User): string => {
  if (arg.spec?.email) {
    return arg.metadata?.displayName
      ? `${arg.metadata.displayName} (${arg.spec.email})`
      : arg.spec.email;
  }
  return arg.metadata?.displayName
    ? `${arg.metadata.name} (${arg.metadata.displayName})`
    : (arg.metadata?.name ?? "");
};

export const printServiceMode = (mode: CoreP.Service_Spec_Mode): string =>
  match(mode)
    .with(CoreP.Service_Spec_Mode.GRPC, () => "gRPC")
    .with(CoreP.Service_Spec_Mode.HTTP, () => "HTTP")
    .with(CoreP.Service_Spec_Mode.KUBERNETES, () => "Kubernetes")
    .with(CoreP.Service_Spec_Mode.MYSQL, () => "MySQL")
    .with(CoreP.Service_Spec_Mode.POSTGRES, () => "PostgreSQL")
    .with(CoreP.Service_Spec_Mode.SSH, () => "SSH")
    .with(CoreP.Service_Spec_Mode.TCP, () => "TCP")
    .with(CoreP.Service_Spec_Mode.UDP, () => "UDP")
    .with(CoreP.Service_Spec_Mode.WEB, () => "Web")
    .with(CoreP.Service_Spec_Mode.DNS, () => "DNS")
    .otherwise((v) => v.toString());

export const hasAuthenticationLog = (resource: Resource): boolean =>
  resource.apiVersion === "core/v1" &&
  match(resource.kind)
    .with(
      "User",
      "Session",
      "IdentityProvider",
      "Credential",
      "Authenticator",
      () => true,
    )
    .otherwise(() => false);

export const hasAccessLog = (resource: Resource): boolean =>
  resource.apiVersion === "core/v1" &&
  match(resource.kind)
    .with(
      "User",
      "Session",
      "Device",
      "Service",
      "Namespace",
      "Region",
      "Policy",
      () => true,
    )
    .otherwise(() => false);

export const hasAuditLog = (resource: Resource): boolean =>
  resource.apiVersion === "core/v1" &&
  match(resource.kind)
    .with("User", "Session", "Device", () => true)
    .otherwise(() => false);

export const hasSSHSessionLog = (resource: Resource): boolean => {
  if (resource.apiVersion !== "core/v1") return false;
  return match(resource.kind)
    .with("User", "Session", "Device", "Namespace", "Region", () => true)
    .with(
      "Service",
      () =>
        (resource as CoreP.Service).spec?.mode === CoreP.Service_Spec_Mode.SSH,
    )
    .otherwise(() => false);
};

export const getRefName = (arg: Resource): string => {
  const kind = arg.kind;
  return `${kind.at(0)?.toLowerCase() ?? ""}${kind.slice(1)}Ref`;
};

export const getRefNameQueryArg = (arg: Resource) => ({
  key: getRefName(arg),
  name: arg.metadata?.name,
});

export const getRefNameQueryArgStr = (arg: Resource): string => {
  const r = getRefNameQueryArg(arg);
  return `${r.key}.name=${r.name}`;
};

export const getVisibilityAPIKindFromPath = (
  pth: string,
): string | undefined => {
  switch (pth.split("/").at(2)) {
    case "accesslogs":
      return "AccessLog";
    case "auditlogs":
      return "AuditLog";
    case "authenticationlogs":
      return "AuthenticationLog";
    case "componentlogs":
      return "ComponentLog";
    case "ssh":
      return "SSHSession";
    default:
      return undefined;
  }
};
