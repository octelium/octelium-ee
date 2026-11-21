import * as AccessP from "@/apis/accessv1/accessv1";
import { Info, ShieldAlert, ShieldCheck } from "lucide-react";

import { AbsoluteTime } from "@/components/TimeAgo";
import { Badge, InfoGrid, KeyValue, Note, SectionCard } from "@/ui";
import {
  Tone,
  accessLevelOf,
  durationToSeconds,
  formatDuration,
  isFullAccessLevel,
  wasApproved,
} from "@/utils";

const GRANT_STATE: Partial<
  Record<AccessP.Request_Status_Grant_State, { label: string; tone: Tone }>
> = {
  [AccessP.Request_Status_Grant_State.SCHEDULED]: {
    label: "Scheduled",
    tone: "blue",
  },
  [AccessP.Request_Status_Grant_State.PROVISIONING]: {
    label: "Provisioning",
    tone: "amber",
  },
  [AccessP.Request_Status_Grant_State.ACTIVE]: {
    label: "Active",
    tone: "emerald",
  },
  [AccessP.Request_Status_Grant_State.FAILED]: { label: "Failed", tone: "red" },
  [AccessP.Request_Status_Grant_State.ENDED]: { label: "Ended", tone: "slate" },
};

const GrantCard = (props: { request: AccessP.Request }) => {
  const level = accessLevelOf(props.request);
  if (!level) return null;

  const scope = props.request.status?.resolvedScope;
  const grant = props.request.status?.grant;
  const grantState = grant ? GRANT_STATE[grant.state] : undefined;
  const effective = props.request.status?.effectiveDuration;
  const requestedSeconds = durationToSeconds(props.request.spec?.duration);
  const effectiveSeconds = durationToSeconds(effective);
  const capped =
    effectiveSeconds > 0 &&
    requestedSeconds > 0 &&
    effectiveSeconds < requestedSeconds;
  const granted = wasApproved(props.request);
  const full = isFullAccessLevel(level);

  return (
    <SectionCard
      title={granted ? "Granted access" : "Access if approved"}
      description={
        granted
          ? "What the approval of this request granted"
          : "What this request grants once it is approved"
      }
      icon={<ShieldCheck size={14} strokeWidth={2.25} />}
      actions={
        grantState && <Badge tone={grantState.tone}>{grantState.label}</Badge>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-body font-semibold text-slate-800">
              {level.displayName || level.name}
            </span>
            <Badge>{full ? "Full access" : "Restricted"}</Badge>
            {scope?.overridesGuardrails && (
              <Badge tone="amber">Overrides guardrails</Badge>
            )}
          </div>
          {level.description && (
            <p className="text-xs font-normal leading-5 text-slate-600">
              {level.description}
            </p>
          )}
        </div>

        <InfoGrid className="border-t border-slate-100 pt-4">
          <KeyValue label="Requested">
            {formatDuration(props.request.spec?.duration)}
          </KeyValue>
          {effective && (
            <KeyValue label="Effective">{formatDuration(effective)}</KeyValue>
          )}
          <KeyValue label="Maximum allowed">
            {level.maxDuration
              ? formatDuration(level.maxDuration)
              : "24 hours (default)"}
          </KeyValue>
          {props.request.status?.accessStartsAt ? (
            <KeyValue label="Starts">
              <AbsoluteTime rfc3339={props.request.status.accessStartsAt} />
            </KeyValue>
          ) : props.request.spec?.startAt ? (
            <KeyValue label="Requested start">
              <AbsoluteTime rfc3339={props.request.spec.startAt} />
            </KeyValue>
          ) : null}
          {props.request.spec?.ticket && (
            <KeyValue label="Ticket" mono>
              {props.request.spec.ticket}
            </KeyValue>
          )}
        </InfoGrid>

        {capped && (
          <Note tone="amber" icon={<Info size={13} strokeWidth={2.25} />}>
            The requested duration is longer than allowed, so the access lasts{" "}
            {formatDuration(effective)}.
          </Note>
        )}

        {scope?.overridesGuardrails && (
          <Note
            tone="amber"
            icon={<ShieldAlert size={13} strokeWidth={2.25} />}
          >
            This access level takes precedence over the policies that would
            otherwise deny this access.
          </Note>
        )}

        {grant?.state === AccessP.Request_Status_Grant_State.FAILED && (
          <Note tone="red" icon={<ShieldAlert size={13} strokeWidth={2.25} />}>
            The access could not be provisioned. Contact an administrator.
          </Note>
        )}
      </div>
    </SectionCard>
  );
};

export default GrantCard;
