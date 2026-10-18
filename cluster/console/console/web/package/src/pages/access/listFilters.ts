import * as AccessP from "@/apis/accessv1/accessv1";
import {
  getAudienceLabel,
  getDeliveryPurposeLabel,
  getDeliveryStateMeta,
  getIdentitySourceLabel,
  getInteractionModeLabel,
} from "./Integration/utils";
import { getOriginLabel } from "./Origin";
import { getStatusMeta, getUrgencyLabel } from "./Request/utils";
import { getDecisionMeta } from "./Review/utils";

export type AccessListKind =
  | "Request"
  | "Review"
  | "Policy"
  | "Catalog"
  | "IntegrationIdentity"
  | "IntegrationBinding";

export type AccessListFilter = {
  key: string;
  label: string;
  options?: { value: string; label: string }[];
};

const enumOptions = (
  values: Record<string, string | number>,
  label?: (value: number) => string,
) =>
  Object.entries(values)
    .filter(
      (entry): entry is [string, number] =>
        typeof entry[1] === "number" && entry[1] > 0,
    )
    .map(([key, value]) => ({
      value: key,
      label: label
        ? label(value)
        : key.charAt(0) + key.slice(1).toLowerCase().replaceAll("_", " "),
    }));

const effect: AccessListFilter = {
  key: "effect",
  label: "Effect",
  options: [
    { value: "AUTO_APPROVE", label: "Auto-approve" },
    { value: "REVIEW", label: "Require review" },
    { value: "DENY", label: "Deny" },
  ],
};

const origin: AccessListFilter = {
  key: "originType",
  label: "Origin",
  options: enumOptions(AccessP.Origin_Type, getOriginLabel),
};

const filters: Partial<Record<AccessListKind, AccessListFilter[]>> = {
  Request: [
    {
      key: "state",
      label: "State",
      options: enumOptions(
        AccessP.Request_Status_State_Status,
        (value) => getStatusMeta(value).label,
      ),
    },
    {
      key: "urgency",
      label: "Urgency",
      options: enumOptions(AccessP.Request_Spec_Urgency, getUrgencyLabel),
    },
    effect,
    origin,
    { key: "isActive", label: "Active access" },
    { key: "hasDeadline", label: "With deadline" },
    { key: "isDeadlinePassed", label: "Past deadline" },
    { key: "isUnmatched", label: "No matching rule" },
    { key: "isExpiringSoon", label: "Expiring within an hour" },
    { key: "isPendingStale", label: "Pending for over 24 hours" },
    { key: "isOnBehalf", label: "Requested on behalf of another user" },
  ],
  Review: [
    {
      key: "decision",
      label: "Decision",
      options: enumOptions(
        AccessP.Review_Spec_Decision,
        (value) => getDecisionMeta(value).label,
      ),
    },
    origin,
    { key: "isDecided", label: "Decided reviews" },
  ],
  Policy: [effect, { key: "isDisabled", label: "Disabled policies" }],
  IntegrationIdentity: [
    {
      key: "source",
      label: "Source",
      options: enumOptions(
        AccessP.IntegrationIdentity_Status_Source,
        getIdentitySourceLabel,
      ),
    },
  ],
  IntegrationBinding: [
    {
      key: "state",
      label: "State",
      options: enumOptions(
        AccessP.IntegrationBinding_Status_State,
        (value) => getDeliveryStateMeta(value).label,
      ),
    },
    {
      key: "purpose",
      label: "Purpose",
      options: enumOptions(
        AccessP.IntegrationBinding_Status_Purpose,
        getDeliveryPurposeLabel,
      ),
    },
    {
      key: "audience",
      label: "Audience",
      options: enumOptions(
        AccessP.Policy_Spec_Rule_Surface_Destination_Audience,
        getAudienceLabel,
      ),
    },
    {
      key: "interactionMode",
      label: "Interaction",
      options: enumOptions(
        AccessP.Policy_Spec_Rule_Surface_InteractionMode,
        getInteractionModeLabel,
      ),
    },
    { key: "isOutOfDate", label: "Out of date" },
    { key: "isFailing", label: "Failing deliveries" },
  ],
};

export const getAccessListFilters = (kind: string): AccessListFilter[] =>
  filters[kind as AccessListKind] ?? [];
