import * as AccessP from "@/apis/accessv1/accessv1";
import { ResourceStatusTone } from "@/pages/utils/types";
import { match } from "ts-pattern";

export const getStateMeta = (
  state?: AccessP.IntegrationBinding_Status_State,
): { label: string; tone: ResourceStatusTone; className: string } =>
  match(state)
    .with(AccessP.IntegrationBinding_Status_State.PENDING, () => ({
      label: "Pending",
      tone: "warning" as const,
      className: "text-amber-600",
    }))
    .with(AccessP.IntegrationBinding_Status_State.READY, () => ({
      label: "Ready",
      tone: "success" as const,
      className: "text-emerald-600",
    }))
    .with(AccessP.IntegrationBinding_Status_State.DEGRADED, () => ({
      label: "Degraded",
      tone: "danger" as const,
      className: "text-red-600",
    }))
    .with(AccessP.IntegrationBinding_Status_State.CLOSED, () => ({
      label: "Closed",
      tone: "neutral" as const,
      className: "text-slate-500",
    }))
    .otherwise(() => ({
      label: "Unknown",
      tone: "neutral" as const,
      className: "text-slate-500",
    }));

export const getPurposeLabel = (
  purpose?: AccessP.IntegrationBinding_Status_Purpose,
): string =>
  match(purpose)
    .with(
      AccessP.IntegrationBinding_Status_Purpose.REVIEW_SURFACE,
      () => "Review surface",
    )
    .with(
      AccessP.IntegrationBinding_Status_Purpose.NOTIFICATION,
      () => "Notification",
    )
    .otherwise(() => "Unset");

export const isOutOfDate = (item: AccessP.IntegrationBinding): boolean =>
  !!item.status &&
  item.status.desiredRevision !== item.status.appliedRevision;

export const isFailing = (item: AccessP.IntegrationBinding): boolean =>
  (item.status?.attempts ?? 0) > 0;
