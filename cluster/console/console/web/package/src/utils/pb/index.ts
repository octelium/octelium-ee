import Yaml from "js-yaml";

import * as CoreP from "../../apis/corev1/corev1";
import * as EnterpriseP from "../../apis/enterprisev1/enterprisev1";
import * as MetaPB from "../../apis/metav1/metav1";
import * as VisibilityCoreP from "../../apis/visibilityv1/core/vcorev1";

import { queryClient } from "@/utils";
import { js_beautify } from "js-beautify";
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

const getAPIType = (arg: Resource) => {
  return getAPITypeFromAPIVersion(arg.apiVersion);
};

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

const getAPITypeFromAPIVersion = (arg: string) => {
  switch (arg) {
    case "core/v1":
      return CoreP;
    case "enterprise/v1":
      return EnterpriseP;
    default:
      return undefined;
  }
};

export const cloneResource = (arg: Resource): Resource => {
  const ap = getAPIType(arg)!;

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

const jsonFix = (arg: string): string => {
  return JSON.stringify(JSON.parse(arg), null, 2);
};

export const resourceToJSON = (arg: Resource): string => {
  try {
    switch (getAPI(arg)) {
      case "core":
        return jsonFix(
          CoreP[arg.kind as ResourceCoreName].toJsonString(arg as any),
        );
      case "enterprise":
        return jsonFix(
          EnterpriseP[arg.kind as ResourceEnterpriseName].toJsonString(
            arg as any,
          ),
        );

      default:
        return "";
    }
  } catch (err) {
    console.log("resourceToJSON err: ", err);
    return "";
  }
};

export const resourceToJSONObject = (arg: Resource): any => {
  return JSON.parse(resourceToJSON(arg));
};

export const resourceSpecToJSON = (arg: Resource): string => {
  return js_beautify(JSON.stringify(resourceToJSONObject(arg)["spec"]));
};

export const resourceSpecToYAML = (arg: Resource): string => {
  return jsonToYAML(resourceSpecToJSON(arg));
};

export const resourceStatusToJSON = (arg: Resource): string => {
  return js_beautify(JSON.stringify(resourceToJSONObject(arg)["status"]));
};

export const resourceStatusToYAML = (arg: Resource): string => {
  return jsonToYAML(resourceStatusToJSON(arg));
};

export const resourceMetadataToJSON = (arg: Resource): string => {
  return js_beautify(JSON.stringify(resourceToJSONObject(arg)["metadata"]));
};

export const resourceMetadataToYAML = (arg: Resource): string => {
  return jsonToYAML(resourceMetadataToJSON(arg));
};

export const resourceToYAML = (arg: Resource): string => {
  try {
    return Yaml.dump(JSON.parse(resourceToJSON(arg)));
  } catch (err) {
    return "";
  }
};

const jsonToYAML = (arg: string): string => {
  return Yaml.dump(JSON.parse(arg));
};

export const resourceFromYAML = (arg: string): Resource | undefined => {
  try {
    const obj = Yaml.load(arg) as any | undefined;
    if (!obj) {
      return undefined;
    }

    return resourceFromJSON(JSON.stringify(obj));
  } catch (err) {
    console.log("resourceFromYAML err", err);
    return undefined;
  }
};

export const resourceFromJSON = (arg: string): Resource | undefined => {
  const obj = JSON.parse(arg);
  if (!obj) {
    return undefined;
  }

  const kind = obj["kind"] as string | undefined;
  const apiVersion = obj["apiVersion"] as string | undefined;
  if (!kind) {
    return undefined;
  }
  if (!apiVersion) {
    return undefined;
  }

  const api = getAPIFromAPIVersion(apiVersion);

  switch (api) {
    case "core":
      try {
        return CoreP[kind as ResourceCoreName].fromJsonString(arg);
      } catch (err) {
        return undefined;
      }

    case "enterprise":
      try {
        return EnterpriseP[kind as ResourceEnterpriseName].fromJsonString(arg);
      } catch (err) {
        return undefined;
      }

    default:
  }
  return undefined;
};

export const getResourceRef = (arg: Resource): MetaPB.ObjectReference => {
  return MetaPB.ObjectReference.create({
    apiVersion: arg.apiVersion,
    kind: arg.kind,
    uid: arg.metadata?.uid,
    name: arg.metadata?.name,
  });
};

export const getShortName = (arg: Resource): string => {
  return getShortNameFromStr(arg.metadata!.name);
};

export const getShortNameFromRef = (arg: MetaPB.ObjectReference): string => {
  return getShortNameFromStr(arg.name);
};

export const getShortNameFromStr = (arg: string): string => {
  return arg.split(".").at(0) ?? "";
};

export const getAPI = (arg: Resource) => {
  return getAPIFromAPIVersion(arg.apiVersion);
};

export const getAPIFromResourceList = (arg: ResourceList) => {
  return getAPIFromAPIVersion(arg.apiVersion);
};

export const getAPIFromAPIVersion = (arg: string) => {
  if (!arg) {
    return undefined;
  }
  switch (arg.split("/").at(0)) {
    case "core":
      return "core";
    case "enterprise":
      return "enterprise";
    default:
      return undefined;
  }
};

export const getResourcePathKind = (arg: Resource): string => {
  return getResourcePathFromResource(arg);
};

export const getResourcePathFromResource = (arg: Resource): string => {
  const api = getAPI(arg);
  return getResourcePathFromAPIKind({
    api: api as API,
    kind: arg.kind as ResourceName,
  });
};

export const getResourcePathFromAPIKind = (arg: APIKind): string => {
  return match(arg.api)
    .with(
      "core",
      () =>
        [...coreResourcePathMap]
          .find(([key, value]) => arg.kind === value)
          ?.at(0) ?? "",
    )
    .with(
      "enterprise",
      () =>
        [...enterpriseResourcePathMap]
          .find(([key, value]) => arg.kind === value)
          ?.at(0) ?? "",
    )
    .otherwise(() => "");
};

export const getResourcePath = (arg: Resource): string => {
  return `/${getAPI(arg)}/${getResourcePathKind(arg)}/${arg.metadata!.name}`;
};

export const getResourceListPath = (arg: ResourceList): string => {
  const api = getAPIFromAPIVersion(arg.apiVersion) as API;
  return `/${getAPIFromResourceList(arg)}/${getResourcePathFromAPIKind({
    api,
    kind: getKindFromResourceList(arg),
  })}`;
};

export const getResourceListPathFromResource = (arg: Resource): string => {
  return `/${getAPI(arg)}/${getResourcePathFromResource(arg)}`;
};

export const invalidateResource = (arg: Resource) => {
  const api = arg.apiVersion.split(`/`).at(0);
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
  queryClient.invalidateQueries({
    queryKey: [getListKey(arg)],
  });

  queryClient.invalidateQueries({
    queryKey: [`listSelectComponent`, getAPI(arg), arg.kind],
  });

  queryClient.invalidateQueries({
    queryKey: ["visibility", getAPI(arg), "summary", arg.kind],
  });
};

export const getKindFromResourceList = (arg: ResourceList) => {
  return arg.kind.replace(/List$/, "") as ResourceName;
};

export const invalidateResourceListFromList = (arg: ResourceList) => {
  const kind = getKindFromResourceList(arg);
  const api = getAPIFromAPIVersion(arg.apiVersion);
  queryClient.invalidateQueries({
    queryKey: [`${api}.list${kind}`],
  });
};

export const invalidateKey = (key: string[]) => {
  queryClient.invalidateQueries({
    queryKey: key,
  });
};

export const getListKey = (arg: Resource) => {
  const api = getAPI(arg);
  return `${api}.list${arg.kind}`;
};

export const getGetKey = (arg: Resource) => {
  const api = getAPI(arg);
  return `${api}.get${arg.kind}`;
};

export const getGetKeyFromRef = (arg: MetaPB.ObjectReference) => {
  const api = getAPIFromAPIVersion(arg.apiVersion);
  return `${api}.get${arg.kind}`;
};

export interface APIKind {
  api: API;
  kind: ResourceName;
}

export const getAPIKindFromPath = (path: string): APIKind | undefined => {
  const args = path.split("/");
  if (args.length < 3) {
    return undefined;
  }

  const api = match(args[1])
    .when(
      (v) => {
        return v === `core` || v === `enterprise`;
      },
      (v) => v,
    )
    .otherwise(() => undefined) as API | undefined;
  if (!api) {
    return undefined;
  }

  const kind = match(api)
    .with("core", () => coreResourcePathMap.get(args[2]))
    .with("enterprise", () => enterpriseResourcePathMap.get(args[2]))
    .otherwise(() => undefined) as ResourceName | undefined;
  if (!kind) {
    return undefined;
  }

  return {
    api,
    kind,
  };
};

export const getListKeyFromResource = (arg: Resource) => {
  return `${getAPI(arg)}.list${arg.kind}`;
};

export const getListKeyFromPath = (path: string) => {
  const apiKind = getAPIKindFromPath(path);
  return `${apiKind?.api}.list${apiKind?.kind}`;
};

export const getGetKeyFromPath = (path: string) => {
  const apiKind = getAPIKindFromPath(path);
  return `${apiKind?.api}.get${apiKind?.kind}`;
};

export const getPB = (arg: Resource) => {
  const api = getAPI(arg);
  if (!api) {
    return undefined;
  }

  return getPBFromAPI(api);
};

export const getPBFromAPI = (api: API) => {
  return match(api)
    .with("core", () => CoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);
};

export const getPBResourceListFromAPI = (api: API) => {
  return match(api)
    .with("core", () => VisibilityCoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);
};

export const getClient = (api: API) => {
  return match(api)
    .with("core", () => getClientCore())
    .with("enterprise", () => getClientEnterprise())
    .otherwise(() => undefined);
};

export const getResourceClient = (arg: Resource) => {
  const api = getAPI(arg);
  if (!api) {
    return undefined;
  }

  return getClient(api);
};

export const getPathClient = (path: string) => {
  const apiKind = getAPIKindFromPath(path);
  if (!apiKind) {
    return undefined;
  }

  return getClient(apiKind.api);
};

export const getClientResourceList = (api: API) => {
  return match(api)
    .with("core", () => getClientVisibilityCore())
    .with("enterprise", () => getClientEnterprise())
    .otherwise(() => undefined);
};

export const getClientResourceListP = (api: API) => {
  return match(api)
    .with("core", () => VisibilityCoreP)
    .with("enterprise", () => EnterpriseP)
    .otherwise(() => undefined);
};

export const printResourceNameWithDisplay = (arg: Resource) => {
  return arg.metadata?.displayName
    ? `${arg.metadata?.name} (${arg.metadata?.displayName})`
    : arg.metadata!.name;
};

export const printUserWithEmail = (arg: CoreP.User) => {
  if (arg.spec!.email) {
    return arg.metadata?.displayName
      ? `${arg.metadata!.displayName} (${arg.spec!.email})`
      : arg.spec!.email;
  }
  return arg.metadata?.displayName
    ? `${arg.metadata?.name} (${arg.metadata?.displayName})`
    : arg.metadata!.name;
};

export const printServiceMode = (mode: CoreP.Service_Spec_Mode): string => {
  return match(mode)
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
};

export const hasAuthenticationLog = (resource: Resource) => {
  return (
    resource.apiVersion === `core/v1` &&
    match(resource.kind)
      .with(
        "User",
        "Session",
        "IdentityProvider",
        "Credential",
        "Authenticator",
        () => true,
      )
      .otherwise(() => false)
  );
};

export const hasAccessLog = (resource: Resource) => {
  return (
    resource.apiVersion === `core/v1` &&
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
      .otherwise(() => false)
  );
};

export const hasAuditLog = (resource: Resource) => {
  return (
    resource.apiVersion === `core/v1` &&
    match(resource.kind)
      .with("User", "Session", "Device", () => true)
      .otherwise(() => false)
  );
};

export const hasSSHSessionLog = (resource: Resource) => {
  return (
    resource.apiVersion === `core/v1` &&
    match(resource.kind)
      .with("User", "Session", "Device", "Namespace", "Region", () => true)
      .with("Service", () => {
        return (
          (resource as CoreP.Service).spec?.mode === CoreP.Service_Spec_Mode.SSH
        );
      })
      .otherwise(() => false)
  );
};

export const getRefName = (arg: Resource): string => {
  const kind = arg.kind as string;
  return `${kind.charAt(0).toLowerCase()}${kind.slice(1)}Ref`;
};

export const getRefNameQueryArg = (arg: Resource) => {
  return {
    key: getRefName(arg),
    name: arg.metadata?.name,
  };
};

export const getRefNameQueryArgStr = (arg: Resource) => {
  const r = getRefNameQueryArg(arg);
  return `${r.key}.name=${r.name}`;
};

export const getVisibilityAPIKindFromPath = (pth: string) => {
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
