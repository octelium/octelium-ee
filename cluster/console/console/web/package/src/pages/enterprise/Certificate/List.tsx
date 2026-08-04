import {
  Certificate,
  Certificate_Spec_Mode,
  Certificate_Status_Issuance,
  Certificate_Status_Issuance_State,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { getDomain } from "@/utils";
import { AlertTriangle, CheckCircle2, Clock3, Loader2 } from "lucide-react";
import { match } from "ts-pattern";

const ItemDetails = (props: { item: Certificate; domain: string }) => {
  const { item } = props;
  return <div></div>;
};

export const getIssuanceState = (arg: Certificate_Status_Issuance_State) =>
  match(arg)
    .with(Certificate_Status_Issuance_State.FAILED, () => "Failed")
    .with(Certificate_Status_Issuance_State.ISSUING, () => "Issuing")
    .with(
      Certificate_Status_Issuance_State.ISSUANCE_REQUESTED,
      () => "Issuance requested",
    )
    .with(Certificate_Status_Issuance_State.SUCCESS, () => "Successful")
    .otherwise(() => "Unknown");

const issuanceTime = (issuance: Certificate_Status_Issuance) => {
  const value =
    issuance.issuanceCompletedAt ??
    issuance.issuanceStartedAt ??
    issuance.createdAt;
  return value ? Timestamp.toDate(value).getTime() : 0;
};

export const getCertificatePresentation = (item: Certificate) => {
  const issuance = item.status?.issuance;
  const successfulIssuance = [
    ...(issuance?.state === Certificate_Status_Issuance_State.SUCCESS
      ? [issuance]
      : []),
    ...(item.status?.lastIssuances ?? []).filter(
      (entry) => entry.state === Certificate_Status_Issuance_State.SUCCESS,
    ),
  ].sort((a, b) => issuanceTime(b) - issuanceTime(a))[0];
  const expiresAt = successfulIssuance?.expiresAt
    ? Timestamp.toDate(successfulIssuance.expiresAt)
    : item.status?.info?.notAfter
      ? Timestamp.toDate(item.status.info.notAfter)
      : undefined;
  const remaining = expiresAt ? expiresAt.getTime() - Date.now() : undefined;
  const expiryState =
    remaining === undefined
      ? "unknown"
      : remaining <= 0
        ? "expired"
        : remaining <= 21 * 24 * 60 * 60 * 1000
          ? "expiring"
          : "valid";

  return {
    issuance,
    successfulIssuance,
    expiresAt,
    expiryState,
    isInProgress:
      issuance?.state === Certificate_Status_Issuance_State.ISSUING ||
      issuance?.state ===
        Certificate_Status_Issuance_State.ISSUANCE_REQUESTED,
    mode:
      item.spec?.mode === Certificate_Spec_Mode.MANUAL ? "Manual" : "Managed",
  } as const;
};

export const LabelComponent = (props: { item: Certificate }) => {
  const { item } = props;
  const presentation = getCertificatePresentation(item);
  const issuance = presentation.issuance;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Mode">{presentation.mode}</ResourceListLabel>
      {issuance &&
        issuance.state !== Certificate_Status_Issuance_State.STATE_UNSET && (
          <ResourceListLabel label="Issuance">
            {presentation.isInProgress && (
              <Loader2 size={11} className="animate-spin text-blue-500" />
            )}
            {issuance.state === Certificate_Status_Issuance_State.FAILED && (
              <AlertTriangle size={11} className="text-red-500" />
            )}
            {getIssuanceState(issuance.state)}
          </ResourceListLabel>
        )}
      {presentation.expiryState !== "unknown" && (
        <ResourceListLabel label="Validity">
          {presentation.expiryState === "valid" ? (
            <CheckCircle2 size={11} className="text-emerald-500" />
          ) : presentation.expiryState === "expiring" ? (
            <Clock3 size={11} className="text-amber-500" />
          ) : (
            <AlertTriangle size={11} className="text-red-500" />
          )}
          {presentation.expiryState === "expiring"
            ? "Expiring soon"
            : presentation.expiryState === "expired"
              ? "Expired"
              : "Valid"}
        </ResourceListLabel>
      )}
      {presentation.successfulIssuance?.expiresAt && (
        <ResourceListLabel label="Expires">
          <TimeAgo rfc3339={presentation.successfulIssuance.expiresAt} />
        </ResourceListLabel>
      )}
      {item.status?.info?.commonName && (
        <ResourceListLabel label="Common name">
          {item.status.info.commonName}
        </ResourceListLabel>
      )}
      {(item.status?.failedIssuances ?? 0) > 0 && (
        <ResourceListLabel label="Failed issuances">
          {item.status!.failedIssuances}
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Certificate }) => {
  const domain = getDomain();
  return <ItemDetails item={props.item} domain={domain} />;
};
