const isDevVal = import.meta.env.MODE === "development";
import type { RpcError } from "@protobuf-ts/runtime-rpc";

import { QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export function isDev(): boolean {
  return isDevVal;
}

const isWebgl2SupportedFn = (() => {
  let isSupported = window.WebGL2RenderingContext ? undefined : false;
  return () => {
    if (isSupported === undefined) {
      const canvas = document.createElement("canvas");
      const gl = canvas.getContext("webgl2", {
        depth: false,
        antialias: false,
      });
      isSupported = gl instanceof window.WebGL2RenderingContext;
    }
    return isSupported;
  };
})();

export const isWebgl2Supported = isWebgl2SupportedFn();

export const onError = (err: RpcError) => {
  toast.error(err.message);
};

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
    },
  },
});

export const toNumOrZero = (arg: string | null | undefined): number => {
  if (!arg) {
    return 0;
  }

  try {
    return parseInt(arg, 10);
  } catch {
    return 0;
  }
};

let __domain: string | undefined;

export const getDomain = (): string => {
  if (isDev()) {
    return window.location.host;
  }

  if (__domain) {
    return __domain;
  }

  __domain =
    ("; " + window.document.cookie)
      .split("; octelium_domain=")
      .pop()
      ?.split(";")
      .shift() ?? "";

  return __domain;
};

export function randomStringLowerCase(n: number): string {
  const characters = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  const charactersLength = characters.length;

  for (let i = 0; i < n; i++) {
    const randomIndex = Math.floor(Math.random() * charactersLength);
    result += characters.charAt(randomIndex);
  }

  return result;
}

export function formatNumber(num: number): string {
  if (num >= 1_000_000_000) {
    return (num / 1_000_000_000).toFixed(2) + "B";
  } else if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(2) + "M";
  } else if (num >= 1_000) {
    return (num / 1_000).toFixed(2) + "K";
  } else {
    return num.toString();
  }
}

export function formatBytes(
  bytes: number,
  options: { useBinaryUnits?: boolean; decimals?: number } = {},
): string {
  const { useBinaryUnits = false, decimals = 2 } = options;

  if (decimals < 0) {
    throw new Error(`Invalid decimals ${decimals}`);
  }

  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.floor(Math.log(bytes) / Math.log(base));

  return `${(bytes / Math.pow(base, i)).toFixed(decimals)} ${units[i]}`;
}

export function slugify(input: string): string {
  return input
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-+|-+$/g, "");
}

import * as AccessP from "@/apis/accessv1/accessv1";
import * as CoreP from "@/apis/corev1/corev1";
import * as MetaP from "@/apis/metav1/metav1";
import * as UserP from "@/apis/userv1/userv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { match } from "ts-pattern";

export type Tone =
  | "amber"
  | "emerald"
  | "red"
  | "slate"
  | "blue"
  | "violet"
  | "sky";

export type ToneClasses = {
  text: string;
  badge: string;
  solid: string;
  soft: string;
  dot: string;
  bar: string;
  icon: string;
  border: string;
};

export const toneClasses: Record<Tone, ToneClasses> = {
  amber: {
    text: "text-amber-600",
    badge: "bg-amber-50 text-amber-700 border-amber-200",
    solid: "bg-amber-500 text-white",
    soft: "bg-amber-50/70 border-amber-100",
    dot: "bg-amber-500",
    bar: "bg-amber-500",
    icon: "bg-amber-50 text-amber-600",
    border: "border-amber-200",
  },
  emerald: {
    text: "text-emerald-600",
    badge: "bg-emerald-50 text-emerald-700 border-emerald-200",
    solid: "bg-emerald-600 text-white",
    soft: "bg-emerald-50/70 border-emerald-100",
    dot: "bg-emerald-500",
    bar: "bg-emerald-500",
    icon: "bg-emerald-50 text-emerald-600",
    border: "border-emerald-200",
  },
  red: {
    text: "text-red-600",
    badge: "bg-red-50 text-red-700 border-red-200",
    solid: "bg-red-600 text-white",
    soft: "bg-red-50/70 border-red-100",
    dot: "bg-red-500",
    bar: "bg-red-500",
    icon: "bg-red-50 text-red-600",
    border: "border-red-200",
  },
  slate: {
    text: "text-slate-600",
    badge: "bg-slate-100 text-slate-600 border-slate-200",
    solid: "bg-slate-800 text-white",
    soft: "bg-slate-50 border-slate-200",
    dot: "bg-slate-400",
    bar: "bg-slate-400",
    icon: "bg-slate-100 text-slate-500",
    border: "border-slate-200",
  },
  blue: {
    text: "text-blue-600",
    badge: "bg-blue-50 text-blue-700 border-blue-200",
    solid: "bg-blue-600 text-white",
    soft: "bg-blue-50/70 border-blue-100",
    dot: "bg-blue-500",
    bar: "bg-blue-500",
    icon: "bg-blue-50 text-blue-600",
    border: "border-blue-200",
  },
  violet: {
    text: "text-violet-600",
    badge: "bg-violet-50 text-violet-700 border-violet-200",
    solid: "bg-violet-600 text-white",
    soft: "bg-violet-50/70 border-violet-100",
    dot: "bg-violet-500",
    bar: "bg-violet-500",
    icon: "bg-violet-50 text-violet-600",
    border: "border-violet-200",
  },
  sky: {
    text: "text-sky-600",
    badge: "bg-sky-50 text-sky-700 border-sky-200",
    solid: "bg-sky-600 text-white",
    soft: "bg-sky-50/70 border-sky-100",
    dot: "bg-sky-500",
    bar: "bg-sky-500",
    icon: "bg-sky-50 text-sky-600",
    border: "border-sky-200",
  },
};

export type StatusGroup = "pending" | "active" | "closed";

export const statusMeta = (
  status?: AccessP.Request_Status_State_Status,
): { label: string; tone: Tone; group: StatusGroup; hint: string } =>
  match(status)
    .with(AccessP.Request_Status_State_Status.PENDING, () => ({
      label: "Pending",
      tone: "amber" as Tone,
      group: "pending" as StatusGroup,
      hint: "Waiting for a reviewer decision",
    }))
    .with(AccessP.Request_Status_State_Status.APPROVED, () => ({
      label: "Approved",
      tone: "emerald" as Tone,
      group: "active" as StatusGroup,
      hint: "Access is granted until the window ends",
    }))
    .with(AccessP.Request_Status_State_Status.REJECTED, () => ({
      label: "Rejected",
      tone: "red" as Tone,
      group: "closed" as StatusGroup,
      hint: "Denied by a policy rule or a reviewer",
    }))
    .with(AccessP.Request_Status_State_Status.REVOKED, () => ({
      label: "Revoked",
      tone: "red" as Tone,
      group: "closed" as StatusGroup,
      hint: "Access was terminated by an administrator",
    }))
    .with(AccessP.Request_Status_State_Status.EXPIRED, () => ({
      label: "Expired",
      tone: "slate" as Tone,
      group: "closed" as StatusGroup,
      hint: "The deadline, review step or access window passed",
    }))
    .with(AccessP.Request_Status_State_Status.CANCELLED, () => ({
      label: "Cancelled",
      tone: "slate" as Tone,
      group: "closed" as StatusGroup,
      hint: "Cancelled by the requester",
    }))
    .otherwise(() => ({
      label: "Evaluating",
      tone: "slate" as Tone,
      group: "pending" as StatusGroup,
      hint: "The Cluster has not evaluated this request yet",
    }));

export const urgencyMeta = (
  urgency?: AccessP.Request_Spec_Urgency,
): { label: string; short: string; tone: Tone; level: number } =>
  match(urgency)
    .with(AccessP.Request_Spec_Urgency.VERY_LOW, () => ({
      label: "Very Low",
      short: "V.Low",
      tone: "slate" as Tone,
      level: 1,
    }))
    .with(AccessP.Request_Spec_Urgency.LOW, () => ({
      label: "Low",
      short: "Low",
      tone: "slate" as Tone,
      level: 2,
    }))
    .with(AccessP.Request_Spec_Urgency.NORMAL, () => ({
      label: "Normal",
      short: "Normal",
      tone: "blue" as Tone,
      level: 3,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGH, () => ({
      label: "High",
      short: "High",
      tone: "amber" as Tone,
      level: 4,
    }))
    .with(AccessP.Request_Spec_Urgency.VERY_HIGH, () => ({
      label: "Very High",
      short: "V.High",
      tone: "red" as Tone,
      level: 5,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGHEST, () => ({
      label: "Highest",
      short: "Highest",
      tone: "red" as Tone,
      level: 6,
    }))
    .otherwise(() => ({
      label: "Normal",
      short: "Normal",
      tone: "blue" as Tone,
      level: 3,
    }));

export const URGENCY_OPTIONS: {
  value: AccessP.Request_Spec_Urgency;
  label: string;
}[] = [
  { value: AccessP.Request_Spec_Urgency.VERY_LOW, label: "Very Low" },
  { value: AccessP.Request_Spec_Urgency.LOW, label: "Low" },
  { value: AccessP.Request_Spec_Urgency.NORMAL, label: "Normal" },
  { value: AccessP.Request_Spec_Urgency.HIGH, label: "High" },
  { value: AccessP.Request_Spec_Urgency.VERY_HIGH, label: "Very High" },
  { value: AccessP.Request_Spec_Urgency.HIGHEST, label: "Highest" },
];

export const decisionMeta = (
  decision?: AccessP.Review_Spec_Decision,
): { label: string; tone: Tone } =>
  match(decision)
    .with(AccessP.Review_Spec_Decision.APPROVE, () => ({
      label: "Approved",
      tone: "emerald" as Tone,
    }))
    .with(AccessP.Review_Spec_Decision.REJECT, () => ({
      label: "Rejected",
      tone: "red" as Tone,
    }))
    .otherwise(() => ({ label: "No decision", tone: "slate" as Tone }));

export const requestResourceLabel = (
  item: AccessP.Request,
): { kind: "Service" | "Catalog" | "Unknown"; name: string } => {
  const r = item.spec?.resource?.type;
  if (!r) return { kind: "Unknown", name: "" };
  if (r.oneofKind === "serviceRef")
    return { kind: "Service", name: r.serviceRef.name };
  if (r.oneofKind === "catalog")
    return { kind: "Catalog", name: r.catalog.catalogRef?.name ?? "" };
  return { kind: "Unknown", name: "" };
};

export const shortName = (name?: string): string =>
  name?.split(".").at(0) ?? "";

export const namespaceFromName = (name?: string): string => {
  if (!name) return "";
  const parts = name.split(".");
  return parts.length > 1 ? parts.slice(1).join(".") : "";
};

export const serviceModeMeta = (
  mode?: UserP.Service_Spec_Type,
): { label: string; tone: Tone } =>
  match(mode)
    .with(UserP.Service_Spec_Type.HTTP, () => ({
      label: "HTTP",
      tone: "blue" as Tone,
    }))
    .with(UserP.Service_Spec_Type.WEB, () => ({
      label: "Web",
      tone: "blue" as Tone,
    }))
    .with(UserP.Service_Spec_Type.GRPC, () => ({
      label: "gRPC",
      tone: "violet" as Tone,
    }))
    .with(UserP.Service_Spec_Type.TCP, () => ({
      label: "TCP",
      tone: "slate" as Tone,
    }))
    .with(UserP.Service_Spec_Type.UDP, () => ({
      label: "UDP",
      tone: "slate" as Tone,
    }))
    .with(UserP.Service_Spec_Type.SSH, () => ({
      label: "SSH",
      tone: "violet" as Tone,
    }))
    .with(UserP.Service_Spec_Type.KUBERNETES, () => ({
      label: "Kubernetes",
      tone: "blue" as Tone,
    }))
    .with(UserP.Service_Spec_Type.POSTGRES, () => ({
      label: "PostgreSQL",
      tone: "emerald" as Tone,
    }))
    .with(UserP.Service_Spec_Type.MYSQL, () => ({
      label: "MySQL",
      tone: "emerald" as Tone,
    }))
    .with(UserP.Service_Spec_Type.DNS, () => ({
      label: "DNS",
      tone: "amber" as Tone,
    }))
    .with(UserP.Service_Spec_Type.SOCKS5, () => ({
      label: "SOCKS5",
      tone: "violet" as Tone,
    }))
    .with(UserP.Service_Spec_Type.RDP_WEB, () => ({
      label: "RDP Web",
      tone: "blue" as Tone,
    }))
    .with(UserP.Service_Spec_Type.MCP, () => ({
      label: "MCP",
      tone: "violet" as Tone,
    }))
    .with(UserP.Service_Spec_Type.LLM, () => ({
      label: "LLM",
      tone: "violet" as Tone,
    }))
    .otherwise(() => ({ label: "Service", tone: "slate" as Tone }));

export const userTypeMeta = (
  type?: CoreP.User_Spec_Type,
): { label: string; tone: Tone } =>
  match(type)
    .with(CoreP.User_Spec_Type.HUMAN, () => ({
      label: "Human",
      tone: "blue" as Tone,
    }))
    .with(CoreP.User_Spec_Type.WORKLOAD, () => ({
      label: "Workload",
      tone: "violet" as Tone,
    }))
    .otherwise(() => ({ label: "User", tone: "slate" as Tone }));

export const requestSubjectName = (item: AccessP.Request): string => {
  const subject = item.spec?.subject?.type;
  return subject?.oneofKind === "userRef" ? subject.userRef.name : "";
};

export const requesterName = (item: AccessP.Request): string =>
  item.status?.userRef?.name ?? "";

export const isOnBehalfOf = (item: AccessP.Request): boolean => {
  const subject = requestSubjectName(item);
  return !!subject && subject !== requesterName(item);
};

const DURATION_UNITS = ["minutes", "hours", "days"] as const;
export type DurationUnit = (typeof DURATION_UNITS)[number];

export const DURATION_UNIT_OPTIONS = [
  { value: "minutes", label: "Minutes" },
  { value: "hours", label: "Hours" },
  { value: "days", label: "Days" },
];

export const durationToParts = (
  d?: MetaP.Duration,
): { unit: DurationUnit; amount: number } => {
  const kind = d?.type.oneofKind;
  if (kind && (DURATION_UNITS as readonly string[]).includes(kind)) {
    return {
      unit: kind as DurationUnit,
      amount: Number((d!.type as any)[kind]) || 0,
    };
  }
  return { unit: "hours", amount: 1 };
};

export const partsToDuration = (
  unit: DurationUnit,
  amount: number,
): MetaP.Duration =>
  MetaP.Duration.create({
    type: { oneofKind: unit, [unit]: amount } as any,
  });

const DURATION_UNIT_SECONDS: Record<string, number> = {
  milliseconds: 0.001,
  seconds: 1,
  minutes: 60,
  hours: 3600,
  days: 86400,
  weeks: 604800,
  months: 2592000,
};

export const durationToSeconds = (d?: MetaP.Duration): number => {
  const kind = d?.type.oneofKind;
  if (!kind) return 0;
  const amount = Number((d!.type as any)[kind]) || 0;
  return amount * (DURATION_UNIT_SECONDS[kind] ?? 0);
};

export const formatDuration = (d?: MetaP.Duration): string => {
  const kind = d?.type.oneofKind;
  if (!kind) return "—";
  const amount = Number((d!.type as any)[kind]) || 0;
  const unit = kind.replace(/s$/, "");
  return `${amount} ${amount === 1 ? unit : `${unit}s`}`;
};

export const formatSeconds = (seconds: number): string => {
  const total = Math.max(0, Math.floor(seconds));
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);

  if (days > 0) return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
  if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  if (minutes > 0) return `${minutes}m`;
  return "under a minute";
};

export const tsToDate = (ts?: Timestamp): Date | undefined =>
  ts ? Timestamp.toDate(ts) : undefined;

export const tsToMillis = (ts?: Timestamp): number | undefined =>
  ts ? Timestamp.toDate(ts).getTime() : undefined;

export const reviewSteps = (
  item?: AccessP.Request,
): AccessP.Policy_Spec_Rule_Action_Review_Step[] => {
  const action = item?.status?.rule?.action?.type;
  return action?.oneofKind === "review" ? action.review.steps : [];
};

export const currentReviewStepIndex = (item?: AccessP.Request): number =>
  item?.status?.review?.currentStep ?? 0;

export const reviewStepTimeoutAt = (
  item?: AccessP.Request,
): Date | undefined => {
  const step = reviewSteps(item)[currentReviewStepIndex(item)];
  const startedAt = tsToMillis(item?.status?.review?.currentStepStartedAt);
  const seconds = durationToSeconds(step?.timeout);
  if (!startedAt || !seconds) return undefined;
  return new Date(startedAt + seconds * 1000);
};

export const approvalRequirementLabel = (
  step?: AccessP.Policy_Spec_Rule_Action_Review_Step,
): string =>
  match(step?.approvalRequirement)
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
      () => "Any one reviewer approves",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ALL,
      () => "Every reviewer must approve",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.COUNT,
      () =>
        `${step?.approvalCount ?? 0} approval${
          (step?.approvalCount ?? 0) === 1 ? "" : "s"
        } required`,
    )
    .otherwise(() => "Approval requirement unset");

export const onTimeoutMeta = (
  step?: AccessP.Policy_Spec_Rule_Action_Review_Step,
): { label: string; tone: Tone } =>
  match(step?.onTimeout)
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.GOTO_NEXT_STEP,
      () => ({ label: "Moves to the next step", tone: "blue" as Tone }),
    )
    .with(AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT, () => ({
      label: "Rejected on timeout",
      tone: "red" as Tone,
    }))
    .otherwise(() => ({ label: "Expires on timeout", tone: "slate" as Tone }));

export type ReviewerRef = { kind: "user" | "group"; name: string };

export const stepReviewers = (
  step?: AccessP.Policy_Spec_Rule_Action_Review_Step,
): ReviewerRef[] =>
  (step?.reviewers ?? []).flatMap<ReviewerRef>((reviewer) => {
    if (reviewer.type.oneofKind === "user") {
      const name = reviewer.type.user.userRef?.name;
      return name ? [{ kind: "user", name }] : [];
    }
    if (reviewer.type.oneofKind === "group") {
      const name = reviewer.type.group.groupRef?.name;
      return name ? [{ kind: "group", name }] : [];
    }
    return [];
  });

export const accessWindow = (
  item?: AccessP.Request,
): { startMs?: number; endMs?: number; progress: number } => {
  const endMs = tsToMillis(item?.status?.accessEndsAt);
  if (!endMs) return { progress: 0 };
  const seconds = durationToSeconds(item?.spec?.duration);
  const startMs = seconds
    ? endMs - seconds * 1000
    : tsToMillis(item?.status?.approvalEndAt);
  if (!startMs || startMs >= endMs) return { startMs, endMs, progress: 0 };
  const ratio = (Date.now() - startMs) / (endMs - startMs);
  return {
    startMs,
    endMs,
    progress: Math.min(1, Math.max(0, ratio)),
  };
};

export const isPendingRequest = (item?: AccessP.Request): boolean =>
  item?.status?.state?.status === AccessP.Request_Status_State_Status.PENDING;

export const isApprovedRequest = (item?: AccessP.Request): boolean =>
  item?.status?.state?.status === AccessP.Request_Status_State_Status.APPROVED;

export const waitingMillis = (item?: AccessP.Request): number => {
  const startedAt =
    tsToMillis(item?.status?.approvalStartAt) ??
    tsToMillis(item?.status?.state?.createdAt) ??
    tsToMillis(item?.metadata?.createdAt);
  return startedAt ? Date.now() - startedAt : 0;
};

export const waitingTone = (millis: number): Tone => {
  if (millis > 24 * 3600 * 1000) return "red";
  if (millis > 4 * 3600 * 1000) return "amber";
  return "slate";
};
