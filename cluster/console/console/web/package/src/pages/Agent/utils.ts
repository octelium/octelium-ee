import {
  parseResourceURI,
  RESOURCE_URI_PREFIX,
  type APIRisk,
  type ResourceRef,
} from "@/apis/consoleagent/protocol";
import { Workspace, Workspace_Status_State } from "@/apis/cordiumv1/cordiumv1";
import { getDomain } from "@/utils";
import {
  getResourcePathFromAPIKind,
  type API,
  type ResourceName,
} from "@/utils/pb";
import { defaultUrlTransform } from "react-markdown";

export const formatBytes = (n: number): string => {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(1)} GB`;
};

const consoleAPIs: API[] = ["core", "enterprise", "access"];

export const getResourceRoute = (ref: ResourceRef): string | undefined => {
  const api = ref.apiVersion.split("/")[0] as API;
  if (!ref.name || !consoleAPIs.includes(api)) {
    return undefined;
  }

  const path = getResourcePathFromAPIKind({
    api,
    kind: ref.kind as ResourceName,
  });
  if (!path) {
    return undefined;
  }

  return `/${api}/${path}/${encodeURIComponent(ref.name)}`;
};

export const getResourceURIRoute = (uri: string): string | undefined => {
  const ref = parseResourceURI(uri);
  return ref ? getResourceRoute(ref) : undefined;
};

export const urlTransform = (url: string): string =>
  url.startsWith(RESOURCE_URI_PREFIX) ? url : defaultUrlTransform(url);

export const riskColor = (risk: APIRisk): string => {
  switch (risk) {
    case "read":
      return "green";
    case "write":
      return "blue";
    case "destructive":
      return "red";
    case "sensitive":
      return "orange";
  }
};

export const workspaceStateLabel = (state: Workspace_Status_State): string => {
  switch (state) {
    case Workspace_Status_State.INIT_REQUEST:
      return "Requested";
    case Workspace_Status_State.INITIALIZING:
      return "Initializing";
    case Workspace_Status_State.PULLING_IMAGE:
      return "Pulling the image";
    case Workspace_Status_State.BUILDING_IMAGE:
      return "Building the image";
    case Workspace_Status_State.STARTING_RUNTIME:
      return "Starting the runtime";
    case Workspace_Status_State.PREPARING:
      return "Preparing";
    case Workspace_Status_State.RUNNING:
      return "Running";
    case Workspace_Status_State.STOPPING_REQUEST:
    case Workspace_Status_State.STOPPING:
      return "Stopping";
    case Workspace_Status_State.STOPPED:
      return "Stopped";
    default:
      return "Unknown";
  }
};

export const isWorkspaceStarting = (ws?: Workspace): boolean => {
  switch (ws?.status?.state) {
    case Workspace_Status_State.INIT_REQUEST:
    case Workspace_Status_State.INITIALIZING:
    case Workspace_Status_State.PULLING_IMAGE:
    case Workspace_Status_State.BUILDING_IMAGE:
    case Workspace_Status_State.STARTING_RUNTIME:
    case Workspace_Status_State.PREPARING:
      return true;
    default:
      return false;
  }
};

export const isWorkspaceStopping = (ws?: Workspace): boolean =>
  ws?.status?.state === Workspace_Status_State.STOPPING_REQUEST ||
  ws?.status?.state === Workspace_Status_State.STOPPING;

export const getCordiumURL = (path = ""): string =>
  `https://cordium.${getDomain()}${path}`;

export const getWorkspaceCordiumURL = (ws: Workspace, page = ""): string =>
  getCordiumURL(
    `/workspaces/${encodeURIComponent(ws.metadata?.name ?? "")}${page ? `/${page}` : ""}`,
  );

export const getWorkspaceDisplayName = (ws?: Workspace): string =>
  ws?.metadata?.displayName || ws?.metadata?.name || "Workspace";

export const formatValue = (value: unknown): string => {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
};

export const toJSON = (value: unknown): string => {
  if (value === undefined) return "";
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
};

export const saveBlob = (blob: Blob, name: string) => {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 10000);
};

export const toCSV = (
  columns: { key: string; label?: string }[],
  rows: Record<string, unknown>[],
): string => {
  const escape = (value: string) =>
    /[",\n\r]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;

  return [
    columns.map((c) => escape(c.label ?? c.key)).join(","),
    ...rows.map((row) =>
      columns.map((c) => escape(formatValue(row[c.key]))).join(","),
    ),
  ].join("\n");
};
