export type AuthenticationLogStatusFilter = "all" | "success" | "failure";

export const FAILURE_REASONS = [
  "INVALID_CREDENTIAL",
  "EXPIRED_CREDENTIAL",
  "REVOKED_CREDENTIAL",
  "USER_NOT_FOUND",
  "USER_DEACTIVATED",
  "USER_LOCKED",
  "DENIED_BY_RULE",
  "NOT_ALLOWED",
  "LIMIT_EXCEEDED",
  "IDENTITY_PROVIDER_ERROR",
  "INTERNAL_ERROR",
] as const;

const FAILURE_REASON_LABELS: Record<string, string> = {
  INVALID_CREDENTIAL: "Invalid credential",
  EXPIRED_CREDENTIAL: "Expired credential",
  REVOKED_CREDENTIAL: "Revoked credential",
  USER_NOT_FOUND: "Unknown user",
  USER_DEACTIVATED: "Deactivated user",
  USER_LOCKED: "Locked user",
  DENIED_BY_RULE: "Denied by rule",
  NOT_ALLOWED: "Not allowed",
  LIMIT_EXCEEDED: "Limit exceeded",
  IDENTITY_PROVIDER_ERROR: "Identity provider error",
  INTERNAL_ERROR: "Internal error",
};

export const getFailureReasonLabel = (reason?: string) =>
  (reason && FAILURE_REASON_LABELS[reason]) || "Unknown reason";

export const parseAuthenticationLogStatus = (
  arg?: string,
): AuthenticationLogStatusFilter | undefined => {
  switch (arg?.toLowerCase()) {
    case "all":
      return "all";
    case "success":
    case "successful":
      return "success";
    case "failure":
    case "failed":
      return "failure";
    default:
      return undefined;
  }
};

export const parseFailureReason = (arg?: string) => {
  const reason = arg?.toUpperCase();
  return FAILURE_REASONS.find((item) => item === reason);
};

export const getFailedAuthenticationLogsPath = (reason?: string) =>
  reason
    ? `/visibility/authenticationlogs?status=failure&reason=${reason}`
    : "/visibility/authenticationlogs?status=failure";

export const getFailureReasonSegments = (
  byReason?: { [key: string]: number },
  limit = 4,
) =>
  Object.entries(byReason ?? {})
    .map(([reason, value]) => ({
      reason,
      label: getFailureReasonLabel(reason),
      value: Number(value ?? 0),
    }))
    .filter((item) => item.value > 0)
    .sort((a, b) => b.value - a.value)
    .slice(0, limit);
