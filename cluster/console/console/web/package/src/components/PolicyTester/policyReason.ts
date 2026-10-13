import { AccessLog_Entry_Common_Reason_Type } from "@/apis/corev1/corev1";
import { match } from "ts-pattern";

export const getPolicyReason = (arg?: AccessLog_Entry_Common_Reason_Type) => {
  return match(arg)
    .with(AccessLog_Entry_Common_Reason_Type.POLICY_MATCH, () => "Policy Match")
    .with(
      AccessLog_Entry_Common_Reason_Type.NO_POLICY_MATCH,
      () => "No Policy Match",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.USER_DEACTIVATED,
      () => "User Deactivated",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_NOT_ACTIVE,
      () => "Session Not Active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_EXPIRED,
      () => "Session Expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.ACCESS_TOKEN_EXPIRED,
      () => "Access Token Expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_AUTHENTICATION_REQUIRED,
      () => "Authenticator Authentication Required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_REGISTRATION_REQUIRED,
      () => "Authenticator Registration Required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SCOPE_UNAUTHORIZED,
      () => "Unauthorized Scope",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_NOT_ACTIVE,
      () => "Device not Active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_CLIENT_TYPE_INVALID,
      () => "Invalid Session type",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_LOCKED,
      () => "Locked Device",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_LOCKED,
      () => "Locked Session",
    )
    .with(AccessLog_Entry_Common_Reason_Type.USER_LOCKED, () => "Locked User")
    .otherwise((typ) => (typ ? AccessLog_Entry_Common_Reason_Type[typ] : ""));
};
