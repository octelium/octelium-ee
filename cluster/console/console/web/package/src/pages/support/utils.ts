import {
  SupportInfo_Limits,
  SupportInfo_State,
  SupportRequest_Actor_Type,
  SupportRequest_Priority,
  SupportRequest_Status,
  SupportRequest_SupportLevel,
} from "@/apis/enterprisev1/enterprisev1";
import { formatBytes } from "@/pages/Agent/format";

export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

export const TONE_SOFT: Record<Tone, string> = {
  neutral: "border-slate-200 bg-slate-50 text-slate-600",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
  info: "border-blue-200 bg-blue-50 text-blue-700",
};

export const TONE_DOT: Record<Tone, string> = {
  neutral: "bg-slate-400",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  info: "bg-blue-500",
};

export const PRIORITIES = [
  {
    value: SupportRequest_Priority.P1,
    code: "P1",
    label: "Critical",
    tone: "danger" as Tone,
    description:
      "Octelium is inoperative or a core function is unavailable with no workaround",
  },
  {
    value: SupportRequest_Priority.P2,
    code: "P2",
    label: "High",
    tone: "warning" as Tone,
    description: "A significant function is impaired or degraded",
  },
  {
    value: SupportRequest_Priority.P3,
    code: "P3",
    label: "Medium",
    tone: "info" as Tone,
    description: "A partial, non-critical loss of functionality",
  },
  {
    value: SupportRequest_Priority.P4,
    code: "P4",
    label: "Low",
    tone: "neutral" as Tone,
    description: "A question, a minor issue or a feature request",
  },
];

export const getPriority = (value?: SupportRequest_Priority) =>
  PRIORITIES.find((item) => item.value === value);

export const STATUS_META: Record<number, { label: string; tone: Tone }> = {
  [SupportRequest_Status.NEW]: { label: "Awaiting Octelium", tone: "info" },
  [SupportRequest_Status.OPEN]: { label: "In progress", tone: "info" },
  [SupportRequest_Status.WAITING_ON_CUSTOMER]: {
    label: "Awaiting your reply",
    tone: "warning",
  },
  [SupportRequest_Status.ON_HOLD]: { label: "On hold", tone: "neutral" },
  [SupportRequest_Status.RESOLVED]: { label: "Resolved", tone: "success" },
  [SupportRequest_Status.CLOSED]: { label: "Closed", tone: "neutral" },
};

export const getStatus = (value?: SupportRequest_Status) =>
  STATUS_META[value ?? SupportRequest_Status.STATUS_UNKNOWN] ?? {
    label: "Unknown",
    tone: "neutral" as Tone,
  };

export const STATUS_WORDS: Record<string, string> = {
  NEW: "new",
  OPEN: "in progress",
  WAITING_ON_CUSTOMER: "awaiting your reply",
  ON_HOLD: "on hold",
  RESOLVED: "resolved",
  CLOSED: "closed",
};

export const isOpenStatus = (value?: SupportRequest_Status) =>
  value !== SupportRequest_Status.RESOLVED &&
  value !== SupportRequest_Status.CLOSED;

export const LEVEL_LABEL: Record<number, string> = {
  [SupportRequest_SupportLevel.NONE]: "No technical support",
  [SupportRequest_SupportLevel.STANDARD]: "Standard support",
  [SupportRequest_SupportLevel.PREMIUM]: "Premium support",
  [SupportRequest_SupportLevel.SUPPORT_LEVEL_UNKNOWN]: "Unknown",
};

export const ACTOR_LABEL: Record<number, string> = {
  [SupportRequest_Actor_Type.CUSTOMER]: "Customer portal",
  [SupportRequest_Actor_Type.CLUSTER]: "Cluster",
  [SupportRequest_Actor_Type.EMAIL]: "Email",
  [SupportRequest_Actor_Type.AGENT]: "Octelium Support",
  [SupportRequest_Actor_Type.SYSTEM]: "Octelium",
  [SupportRequest_Actor_Type.TYPE_UNKNOWN]: "Unknown",
};

export const STATE_META: Record<
  number,
  { label: string; tone: Tone; title: string }
> = {
  [SupportInfo_State.AVAILABLE]: {
    label: "Available",
    tone: "success",
    title: "Support is available",
  },
  [SupportInfo_State.NO_LICENSE]: {
    label: "No license",
    tone: "warning",
    title: "This Cluster needs a valid license",
  },
  [SupportInfo_State.NOT_ENTITLED]: {
    label: "Not included",
    tone: "warning",
    title: "Technical support is not included",
  },
  [SupportInfo_State.UNAUTHORIZED]: {
    label: "License rejected",
    tone: "danger",
    title: "Octelium rejected the license of this Cluster",
  },
  [SupportInfo_State.UNREACHABLE]: {
    label: "Unreachable",
    tone: "danger",
    title: "The customer portal cannot be reached",
  },
  [SupportInfo_State.DISABLED]: {
    label: "Unavailable",
    tone: "neutral",
    title: "Support requests are temporarily unavailable",
  },
  [SupportInfo_State.STATE_UNKNOWN]: {
    label: "Checking",
    tone: "neutral",
    title: "Checking support",
  },
};

export const getState = (value?: SupportInfo_State) =>
  STATE_META[value ?? SupportInfo_State.STATE_UNKNOWN] ??
  STATE_META[SupportInfo_State.STATE_UNKNOWN];

export const DESCRIPTION_TEMPLATE = `What happened?


What did you expect to happen?


How can we reproduce it?


What is the impact (affected Users, Services and Regions)?


What changed recently (upgrades, configuration, infrastructure)?
`;

export const ENVIRONMENT_LABELS: Record<string, string> = {
  "version.core": "Core version",
  "version.enterprise": "Enterprise version",
  "version.cordium": "Cordium version",
  regions: "Regions",
  gateways: "Gateways",
  "license.uid": "License ID",
  "license.type": "License type",
  "license.plan": "Plan",
  "license.expiresAt": "License expiry",
};

export const environmentEntries = (environment?: Record<string, string>) =>
  Object.entries(environment ?? {}).sort(([a], [b]) => {
    const keys = Object.keys(ENVIRONMENT_LABELS);
    const ia = keys.indexOf(a);
    const ib = keys.indexOf(b);
    return (
      (ia < 0 ? keys.length : ia) - (ib < 0 ? keys.length : ib) ||
      a.localeCompare(b)
    );
  });

export const newIdempotencyKey = () => `console-${crypto.randomUUID()}`;

export const supportRequestPath = (id: string) =>
  `/support/requests/${encodeURIComponent(id)}`;

export const fileKey = (file: File) =>
  `${file.name}:${file.size}:${file.lastModified}`;

export const checkFile = (
  limits: SupportInfo_Limits | undefined,
  file: File,
): string | null => {
  const extension = /\.([A-Za-z0-9]{1,8})$/.exec(file.name)?.[1]?.toLowerCase();
  if (
    !extension ||
    (limits && !limits.attachmentExtensions.includes(extension))
  ) {
    return `${file.name}: this file type is not accepted`;
  }
  if (file.size === 0) return `${file.name} is empty`;
  const max = Number(limits?.maxAttachmentSize ?? 0);
  if (max > 0 && file.size > max) {
    return `${file.name} is larger than ${formatBytes(max)}`;
  }
  return null;
};

export const initials = (name: string) =>
  name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.at(0))
    .join("")
    .toUpperCase();
