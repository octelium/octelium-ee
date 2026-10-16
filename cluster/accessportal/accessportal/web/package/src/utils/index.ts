const isDevVal = import.meta.env.MODE === "development";

import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { onQueryError, queryRetryDelay, retryQuery } from "./queryPolicy";

export function isDev(): boolean {
  return isDevVal;
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: onQueryError }),
  mutationCache: new MutationCache({ onError: onQueryError }),
  defaultOptions: {
    queries: {
      staleTime: 30000,
      retry: retryQuery,
      retryDelay: queryRetryDelay,
    },
  },
});

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

import * as AccessP from "@/apis/accessv1/accessv1";
import * as CoreP from "@/apis/corev1/corev1";
import * as MetaP from "@/apis/metav1/metav1";
import * as UserP from "@/apis/userv1/userv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { match } from "ts-pattern";

export type Tone = "amber" | "emerald" | "red" | "slate" | "blue";

export type ToneClasses = {
  text: string;
  badge: string;
  soft: string;
  dot: string;
  bar: string;
  icon: string;
};

export const toneClasses: Record<Tone, ToneClasses> = {
  amber: {
    text: "text-amber-700",
    badge: "border-amber-200 bg-amber-50 text-amber-700",
    soft: "border-amber-200 bg-amber-50 text-amber-800",
    dot: "bg-amber-500",
    bar: "bg-amber-500",
    icon: "border-amber-200 bg-amber-50 text-amber-700",
  },
  emerald: {
    text: "text-emerald-700",
    badge: "border-emerald-200 bg-emerald-50 text-emerald-700",
    soft: "border-emerald-200 bg-emerald-50 text-emerald-800",
    dot: "bg-emerald-500",
    bar: "bg-emerald-500",
    icon: "border-emerald-200 bg-emerald-50 text-emerald-700",
  },
  red: {
    text: "text-red-700",
    badge: "border-red-200 bg-red-50 text-red-700",
    soft: "border-red-200 bg-red-50 text-red-800",
    dot: "bg-red-500",
    bar: "bg-red-500",
    icon: "border-red-200 bg-red-50 text-red-700",
  },
  slate: {
    text: "text-slate-600",
    badge: "border-slate-200 bg-slate-50 text-slate-600",
    soft: "border-slate-200 bg-slate-50 text-slate-600",
    dot: "bg-slate-400",
    bar: "bg-slate-500",
    icon: "border-slate-200 bg-slate-50 text-slate-500",
  },
  blue: {
    text: "text-blue-700",
    badge: "border-blue-200 bg-blue-50 text-blue-700",
    soft: "border-blue-200 bg-blue-50 text-blue-800",
    dot: "bg-blue-500",
    bar: "bg-blue-500",
    icon: "border-blue-200 bg-blue-50 text-blue-700",
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
): { label: string; tone: Tone; level: number } =>
  match(urgency)
    .with(AccessP.Request_Spec_Urgency.VERY_LOW, () => ({
      label: "Very Low",
      tone: "slate" as Tone,
      level: 1,
    }))
    .with(AccessP.Request_Spec_Urgency.LOW, () => ({
      label: "Low",
      tone: "slate" as Tone,
      level: 2,
    }))
    .with(AccessP.Request_Spec_Urgency.NORMAL, () => ({
      label: "Normal",
      tone: "slate" as Tone,
      level: 3,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGH, () => ({
      label: "High",
      tone: "amber" as Tone,
      level: 4,
    }))
    .with(AccessP.Request_Spec_Urgency.VERY_HIGH, () => ({
      label: "Very High",
      tone: "red" as Tone,
      level: 5,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGHEST, () => ({
      label: "Highest",
      tone: "red" as Tone,
      level: 6,
    }))
    .otherwise(() => ({
      label: "Normal",
      tone: "slate" as Tone,
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

export const serviceModeLabel = (mode?: UserP.Service_Spec_Type): string =>
  match(mode)
    .with(UserP.Service_Spec_Type.HTTP, () => "HTTP")
    .with(UserP.Service_Spec_Type.WEB, () => "Web")
    .with(UserP.Service_Spec_Type.GRPC, () => "gRPC")
    .with(UserP.Service_Spec_Type.TCP, () => "TCP")
    .with(UserP.Service_Spec_Type.UDP, () => "UDP")
    .with(UserP.Service_Spec_Type.SSH, () => "SSH")
    .with(UserP.Service_Spec_Type.KUBERNETES, () => "Kubernetes")
    .with(UserP.Service_Spec_Type.POSTGRES, () => "PostgreSQL")
    .with(UserP.Service_Spec_Type.MYSQL, () => "MySQL")
    .with(UserP.Service_Spec_Type.DNS, () => "DNS")
    .with(UserP.Service_Spec_Type.SOCKS5, () => "SOCKS5")
    .with(UserP.Service_Spec_Type.RDP_WEB, () => "RDP Web")
    .with(UserP.Service_Spec_Type.MCP, () => "MCP")
    .with(UserP.Service_Spec_Type.LLM, () => "LLM")
    .otherwise(() => "Service");

export const isWorkloadUser = (type?: CoreP.User_Spec_Type): boolean =>
  type === CoreP.User_Spec_Type.WORKLOAD;

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

export const newRequestHref = (item: AccessP.Request): string => {
  const resource = requestResourceLabel(item);
  if (!resource.name) return "/user/new";
  const params = new URLSearchParams({
    [resource.kind === "Catalog" ? "catalogRef.name" : "serviceRef.name"]:
      resource.name,
  });
  return `/user/new?${params}`;
};

export const requestCreatedAt = (item?: AccessP.Request): Timestamp | undefined =>
  item?.metadata?.createdAt ?? item?.status?.approvalStartAt;

export const MAX_JUSTIFICATION_BYTES = 1500;

const textEncoder = new TextEncoder();

export const utf8Length = (value: string): number =>
  textEncoder.encode(value).length;

const DURATION_UNITS = ["minutes", "hours", "days"] as const;
export type DurationUnit = (typeof DURATION_UNITS)[number];

export const DURATION_UNIT_OPTIONS = [
  { value: "minutes", label: "Minutes" },
  { value: "hours", label: "Hours" },
  { value: "days", label: "Days" },
];

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
  const amount = Number((d!.type as Record<string, unknown>)[kind]) || 0;
  return amount * (DURATION_UNIT_SECONDS[kind] ?? 0);
};

export const durationToParts = (
  d?: MetaP.Duration,
): { unit: DurationUnit; amount: number } => {
  const kind = d?.type.oneofKind;
  if (kind && (DURATION_UNITS as readonly string[]).includes(kind)) {
    const amount = Number((d!.type as Record<string, unknown>)[kind]) || 0;
    if (amount > 0) return { unit: kind as DurationUnit, amount };
  }

  const seconds = durationToSeconds(d);
  if (seconds <= 0) return { unit: "hours", amount: 1 };
  if (seconds % 86400 === 0) return { unit: "days", amount: seconds / 86400 };
  if (seconds % 3600 === 0) return { unit: "hours", amount: seconds / 3600 };
  return { unit: "minutes", amount: Math.ceil(seconds / 60) };
};

export const partsToDuration = (
  unit: DurationUnit,
  amount: number,
): MetaP.Duration => {
  const value = Math.max(1, Math.trunc(amount) || 1);
  switch (unit) {
    case "minutes":
      return MetaP.Duration.create({ type: { oneofKind: unit, minutes: value } });
    case "hours":
      return MetaP.Duration.create({ type: { oneofKind: unit, hours: value } });
    case "days":
      return MetaP.Duration.create({ type: { oneofKind: unit, days: value } });
  }
};

export const normalizeDuration = (d?: MetaP.Duration): MetaP.Duration => {
  const { unit, amount } = durationToParts(d);
  return partsToDuration(unit, amount);
};

const plural = (amount: number, unit: string) =>
  `${amount.toLocaleString()} ${amount === 1 ? unit : `${unit}s`}`;

export const formatDuration = (d?: MetaP.Duration): string => {
  const kind = d?.type.oneofKind;
  if (!kind) return "—";
  if (kind === "seconds" || kind === "milliseconds") {
    const seconds = durationToSeconds(d);
    if (seconds > 0 && seconds % 86400 === 0)
      return plural(seconds / 86400, "day");
    if (seconds > 0 && seconds % 3600 === 0)
      return plural(seconds / 3600, "hour");
    if (seconds > 0 && seconds % 60 === 0) return plural(seconds / 60, "minute");
    return plural(Math.round(seconds), "second");
  }
  const amount = Number((d!.type as Record<string, unknown>)[kind]) || 0;
  return plural(amount, kind.replace(/s$/, ""));
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

export const reviewStepName = (
  step: AccessP.Policy_Spec_Rule_Action_Review_Step | undefined,
  index: number,
): string => step?.name || `step-${index + 1}`;

export const reviewStepLabel = (
  step: AccessP.Policy_Spec_Rule_Action_Review_Step | undefined,
  index: number,
): string => step?.name || `Step ${index + 1}`;

export const reviewStepTimeoutAt = (
  item?: AccessP.Request,
): Date | undefined => {
  const step = reviewSteps(item)[currentReviewStepIndex(item)];
  const startedAt = tsToMillis(item?.status?.review?.currentStepStartedAt);
  const seconds = durationToSeconds(step?.timeout);
  if (!startedAt || !seconds) return undefined;
  return new Date(startedAt + seconds * 1000);
};

export const wasApproved = (item?: AccessP.Request): boolean =>
  isApprovedRequest(item) ||
  (item?.status?.lastStates ?? []).some(
    (state) => state.status === AccessP.Request_Status_State_Status.APPROVED,
  );

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

export const onTimeoutLabel = (
  step?: AccessP.Policy_Spec_Rule_Action_Review_Step,
): string =>
  match(step?.onTimeout)
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.GOTO_NEXT_STEP,
      () => "then moves to the next step",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT,
      () => "then the request is rejected",
    )
    .otherwise(() => "then the request expires");

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

export const grantedDuration = (item?: AccessP.Request): MetaP.Duration | undefined =>
  item?.status?.effectiveDuration ?? item?.spec?.duration;

export const accessWindow = (
  item?: AccessP.Request,
  now = Date.now(),
): { startMs?: number; endMs?: number; progress: number } => {
  const endMs = tsToMillis(item?.status?.accessEndsAt);
  if (!endMs) return { progress: 0 };
  const seconds = durationToSeconds(grantedDuration(item));
  const startMs =
    tsToMillis(item?.status?.approvalEndAt) ??
    (seconds ? endMs - seconds * 1000 : undefined);
  if (!startMs || startMs >= endMs) return { startMs, endMs, progress: 0 };
  const ratio = (now - startMs) / (endMs - startMs);
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

export const isEvaluatingRequest = (item?: AccessP.Request): boolean =>
  !!item &&
  (item.status?.state?.status ?? AccessP.Request_Status_State_Status.STATUS_UNKNOWN) ===
    AccessP.Request_Status_State_Status.STATUS_UNKNOWN;

export const requestRefetchInterval = (
  item?: AccessP.Request,
): number | false => {
  if (isPendingRequest(item) || isEvaluatingRequest(item)) return 15000;
  if (isApprovedRequest(item)) return 60000;
  return false;
};

export const waitingMillis = (item?: AccessP.Request, now = Date.now()): number => {
  const startedAt =
    tsToMillis(item?.status?.approvalStartAt) ??
    tsToMillis(item?.metadata?.createdAt);
  return startedAt ? now - startedAt : 0;
};

export const isPastDeadline = (item?: AccessP.Request, now = Date.now()): boolean => {
  const deadline = tsToMillis(item?.spec?.deadline);
  return !!deadline && deadline <= now;
};
