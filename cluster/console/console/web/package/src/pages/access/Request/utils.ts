import * as AccessP from "@/apis/accessv1/accessv1";
import { match } from "ts-pattern";

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
    .otherwise(() => ({ label: "Unknown", className: "text-slate-400" }));

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
