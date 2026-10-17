import * as AccessP from "@/apis/accessv1/accessv1";
import { Duration } from "@/apis/metav1/metav1";
import { match } from "ts-pattern";

export const formatDuration = (duration?: Duration): string => {
  const type = duration?.type;
  if (type?.oneofKind === "milliseconds")
    return `${type.milliseconds.toLocaleString()} milliseconds`;
  if (type?.oneofKind === "seconds")
    return `${type.seconds.toLocaleString()} seconds`;
  if (type?.oneofKind === "minutes")
    return `${type.minutes.toLocaleString()} minutes`;
  if (type?.oneofKind === "hours") return `${type.hours.toLocaleString()} hours`;
  if (type?.oneofKind === "days") return `${type.days.toLocaleString()} days`;
  if (type?.oneofKind === "weeks") return `${type.weeks.toLocaleString()} weeks`;
  if (type?.oneofKind === "months")
    return `${type.months.toLocaleString()} months`;
  return "Not set";
};

export const getStatusMeta = (
  status?: AccessP.Request_Status_State_Status,
): { label: string; className: string } =>
  match(status)
    .with(AccessP.Request_Status_State_Status.PENDING, () => ({
      label: "Pending",
      className: "text-amber-500",
    }))
    .with(AccessP.Request_Status_State_Status.APPROVED, () => ({
      label: "Approved",
      className: "text-emerald-600",
    }))
    .with(AccessP.Request_Status_State_Status.REJECTED, () => ({
      label: "Rejected",
      className: "text-red-500",
    }))
    .with(AccessP.Request_Status_State_Status.REVOKED, () => ({
      label: "Revoked",
      className: "text-red-500",
    }))
    .with(AccessP.Request_Status_State_Status.EXPIRED, () => ({
      label: "Expired",
      className: "text-slate-500",
    }))
    .with(AccessP.Request_Status_State_Status.CANCELLED, () => ({
      label: "Cancelled",
      className: "text-slate-500",
    }))
    .otherwise(() => ({ label: "Unknown", className: "text-slate-500" }));

export const getUrgencyLabel = (
  urgency?: AccessP.Request_Spec_Urgency,
): string =>
  match(urgency)
    .with(AccessP.Request_Spec_Urgency.VERY_LOW, () => "Very Low")
    .with(AccessP.Request_Spec_Urgency.LOW, () => "Low")
    .with(AccessP.Request_Spec_Urgency.NORMAL, () => "Normal")
    .with(AccessP.Request_Spec_Urgency.HIGH, () => "High")
    .with(AccessP.Request_Spec_Urgency.VERY_HIGH, () => "Very High")
    .with(AccessP.Request_Spec_Urgency.HIGHEST, () => "Highest")
    .otherwise(() => "Unset");

export const getUrgencyColor = (
  urgency?: AccessP.Request_Spec_Urgency,
): string =>
  match(urgency)
    .with(AccessP.Request_Spec_Urgency.VERY_LOW, () => "var(--color-slate-500)")
    .with(AccessP.Request_Spec_Urgency.LOW, () => "var(--color-emerald-600)")
    .with(AccessP.Request_Spec_Urgency.NORMAL, () => "var(--color-blue-600)")
    .with(AccessP.Request_Spec_Urgency.HIGH, () => "var(--color-amber-600)")
    .with(AccessP.Request_Spec_Urgency.VERY_HIGH, () => "var(--color-orange-600)")
    .with(AccessP.Request_Spec_Urgency.HIGHEST, () => "var(--color-red-600)")
    .otherwise(() => "var(--color-slate-400)");

export const getStatusTone = (
  status?: AccessP.Request_Status_State_Status,
): "neutral" | "success" | "warning" | "danger" | "info" =>
  match(status)
    .with(AccessP.Request_Status_State_Status.PENDING, () => "warning" as const)
    .with(AccessP.Request_Status_State_Status.APPROVED, () => "success" as const)
    .with(AccessP.Request_Status_State_Status.REJECTED, () => "danger" as const)
    .with(AccessP.Request_Status_State_Status.REVOKED, () => "danger" as const)
    .otherwise(() => "neutral" as const);

export const getApprovalRequirementLabel = (
  step: AccessP.Policy_Spec_Rule_Action_Review_Step,
): string =>
  match(step.approvalRequirement)
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
      () => "One approval needed",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ALL,
      () => "Every reviewer must approve",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.COUNT,
      () =>
        `${step.approvalCount} approval${step.approvalCount === 1 ? "" : "s"} needed`,
    )
    .otherwise(() => "No approval requirement");

export const getOnTimeoutLabel = (
  step: AccessP.Policy_Spec_Rule_Action_Review_Step,
  isLast?: boolean,
): string =>
  match(step.onTimeout)
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.GOTO_NEXT_STEP,
      () =>
        isLast
          ? "then the request expires"
          : "then moves to the next step",
    )
    .with(
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT,
      () => "then the request is rejected",
    )
    .otherwise(() => "then the request expires");
