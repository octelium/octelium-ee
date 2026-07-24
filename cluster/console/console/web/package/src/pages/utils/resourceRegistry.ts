import {
  API,
  getAPI,
  getAPIKindFromPath,
  Resource,
  ResourceName,
} from "@/utils/pb";
import { ResourceComponentInfo } from "./types";

import { resourceList as resourceListAccess } from "../access/router";
import { resourceList as resourceListCore } from "../core/router";
import { resourceList as resourceListEnterprise } from "../enterprise/router";

export type ResourceComponentKey = `${API}/${ResourceName}`;

export const getResourceComponentKey = (
  api: API,
  kind: ResourceName,
): ResourceComponentKey => `${api}/${kind}`;

let byAPICache: Record<API, ResourceComponentInfo[]> | undefined;
let listCache: ResourceComponentInfo[] | undefined;
let mapCache: Map<ResourceComponentKey, ResourceComponentInfo> | undefined;

export const getResourceListByAPI = (): Record<
  API,
  ResourceComponentInfo[]
> => {
  if (!byAPICache) {
    byAPICache = {
      core: resourceListCore,
      enterprise: resourceListEnterprise,
      access: resourceListAccess,
    };
  }
  return byAPICache;
};

export const getAllResourceComponents = (): ResourceComponentInfo[] => {
  if (!listCache) {
    const byAPI = getResourceListByAPI();
    listCache = [...byAPI.core, ...byAPI.enterprise, ...byAPI.access];
  }
  return listCache;
};

export const getResourceComponentMap = (): ReadonlyMap<
  ResourceComponentKey,
  ResourceComponentInfo
> => {
  if (!mapCache) {
    const ret = new Map<ResourceComponentKey, ResourceComponentInfo>();

    for (const info of getAllResourceComponents()) {
      const key = getResourceComponentKey(info.API, info.Kind);
      if (ret.has(key)) {
        console.warn(`Duplicate ResourceComponentInfo for ${key}`);
      }
      ret.set(key, info);
    }

    mapCache = ret;
  }
  return mapCache;
};

export const getResourceComponentInfo = (
  api?: API,
  kind?: ResourceName,
): ResourceComponentInfo | undefined => {
  if (!api || !kind) {
    return undefined;
  }
  return getResourceComponentMap().get(getResourceComponentKey(api, kind));
};

export const getResourceComponentInfoFromResource = (
  arg: Resource,
): ResourceComponentInfo | undefined =>
  getResourceComponentInfo(getAPI(arg), arg.kind as ResourceName);

export const getResourceComponentInfoFromPath = (
  path: string,
): ResourceComponentInfo | undefined => {
  const apiKind = getAPIKindFromPath(path);
  if (!apiKind) {
    return undefined;
  }
  return getResourceComponentInfo(apiKind.api, apiKind.kind);
};

export const getResourceComponentList = (api: API): ResourceComponentInfo[] =>
  getResourceListByAPI()[api] ?? [];
