import {
  parseResourceURI,
  RESOURCE_URI_PREFIX,
  type APIRisk,
  type Conversation,
  type Message,
  type ResourceRef,
  type TableBlock,
  type ToolBlock,
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

export const readStorage = (key: string): string | undefined => {
  try {
    return localStorage.getItem(key) ?? undefined;
  } catch {
    return undefined;
  }
};

export const writeStorage = (key: string, value: string) => {
  try {
    localStorage.setItem(key, value);
  } catch {
    return;
  }
};

export const searchTerms = (query: string): string[] => [
  ...new Set(
    query
      .toLowerCase()
      .split(/\s+/)
      .filter((term) => term !== ""),
  ),
];

export const matchesTerms = (text: string, terms: string[]): boolean => {
  const lower = text.toLowerCase();
  return terms.every((term) => lower.includes(term));
};

export interface ConversationGroup {
  label: string;
  items: Conversation[];
}

const dayMs = 24 * 60 * 60 * 1000;

export const conversationGroupLabel = (date: Date, now: Date): string => {
  const today = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
  ).getTime();
  const time = date.getTime();
  if (time >= today) return "Today";
  if (time >= today - dayMs) return "Yesterday";
  if (time >= today - 7 * dayMs) return "Previous 7 days";
  if (time >= today - 30 * dayMs) return "Previous 30 days";
  return date.toLocaleDateString(undefined, { month: "long", year: "numeric" });
};

export const groupConversations = (
  conversations: Conversation[],
  now = new Date(),
): ConversationGroup[] => {
  const ret: ConversationGroup[] = [];
  for (const conversation of conversations) {
    const label = conversationGroupLabel(new Date(conversation.updatedAt), now);
    const last = ret.at(-1);
    if (last?.label === label) {
      last.items.push(conversation);
    } else {
      ret.push({ label, items: [conversation] });
    }
  }
  return ret;
};

export const messageText = (message: Message): string =>
  message.blocks
    .map((block) => (block.type === "markdown" ? block.text.trim() : ""))
    .filter((text) => text !== "")
    .join("\n\n");

const tableMarkdown = (block: TableBlock): string => {
  const cell = (value: string) =>
    value.replace(/\|/g, "\\|").replace(/\s*\n\s*/g, " ");
  return [
    `| ${block.columns.map((c) => cell(c.label ?? c.key)).join(" | ")} |`,
    `| ${block.columns.map(() => "---").join(" | ")} |`,
    ...block.rows.map(
      (row) =>
        `| ${block.columns.map((c) => cell(formatValue(row[c.key]))).join(" | ")} |`,
    ),
  ].join("\n");
};

export const conversationMarkdown = (
  title: string,
  messages: Message[],
): string => {
  const ret: string[] = [`# ${title}`];

  for (const message of messages) {
    ret.push(message.role === "user" ? "## You" : "## Agent");
    if (message.attachments && message.attachments.length > 0) {
      ret.push(
        `Attachments: ${message.attachments.map((a) => a.name).join(", ")}`,
      );
    }
    for (const block of message.blocks) {
      switch (block.type) {
        case "markdown":
          if (block.text.trim() !== "") ret.push(block.text.trim());
          break;
        case "tool":
          ret.push(`- ${block.title || block.name} (${block.status})`);
          break;
        case "approval":
          ret.push(`> Approval ${block.status}: ${block.title}`);
          break;
        case "table":
          if (block.title) ret.push(`**${block.title}**`);
          ret.push(tableMarkdown(block));
          break;
        case "chart":
          ret.push(`_Chart: ${block.chart.title ?? block.chart.chartType}_`);
          break;
        case "resources":
          ret.push(
            block.resources
              .map((item) => `- ${item.ref.kind} ${item.ref.name ?? ""}`)
              .join("\n"),
          );
          break;
        case "artifact":
          ret.push(`_Artifact: ${block.artifact.name}_`);
          break;
        case "notice":
          ret.push(`> ${block.text}`);
          break;
        case "error":
          ret.push(`> Error: ${block.message}`);
          break;
      }
    }
  }

  return `${ret.join("\n\n")}\n`;
};

export const formatDuration = (ms: number): string => {
  if (!Number.isFinite(ms) || ms < 0) return "";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 10000) return `${(ms / 1000).toFixed(1)}s`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
};

export const formatElapsed = (ms: number): string => {
  const seconds = Math.max(0, Math.floor(ms / 1000));
  if (seconds < 60) return `${seconds}s`;
  return `${Math.floor(seconds / 60)}m ${String(seconds % 60).padStart(2, "0")}s`;
};

export const formatCount = (n: number): string => {
  if (n < 1000) return String(n);
  if (n < 1000000) return `${(n / 1000).toFixed(n < 10000 ? 1 : 0)}k`;
  return `${(n / 1000000).toFixed(1)}M`;
};

export const toolDuration = (block: ToolBlock): number | undefined =>
  block.startedAt && block.completedAt
    ? new Date(block.completedAt).getTime() -
      new Date(block.startedAt).getTime()
    : undefined;

export const greeting = (date = new Date()): string => {
  const hour = date.getHours();
  if (hour >= 5 && hour < 12) return "Good morning";
  if (hour >= 12 && hour < 18) return "Good afternoon";
  return "Good evening";
};

export const toFileName = (value: string, ext: string): string =>
  `${
    value
      .trim()
      .replace(/[^\w.-]+/g, "_")
      .slice(0, 80) || "conversation"
  }.${ext}`;
