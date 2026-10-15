import * as AccessP from "@/apis/accessv1/accessv1";
import { Info, ShieldCheck } from "lucide-react";

import { Eyebrow, InfoGrid, KeyValue, MonoValue, Note, SectionCard } from "@/ui";
import { durationToSeconds, formatDuration, wasApproved } from "@/utils";

const AuthorizationCard = (props: { request: AccessP.Request }) => {
  const authorization = props.request.status?.rule?.authorization;
  if (!authorization) return null;

  const policies = authorization.policies ?? [];
  const inlinePolicies = authorization.inlinePolicies ?? [];
  const effective = props.request.status?.effectiveDuration;
  const requestedSeconds = durationToSeconds(props.request.spec?.duration);
  const effectiveSeconds = durationToSeconds(effective);
  const capped =
    effectiveSeconds > 0 && requestedSeconds > 0 && effectiveSeconds < requestedSeconds;
  const granted = wasApproved(props.request);

  return (
    <SectionCard
      title={granted ? "Granted access" : "Access if approved"}
      description={
        granted
          ? "What the approval of this request granted"
          : "What the matched rule grants once this request is approved"
      }
      icon={<ShieldCheck size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-col gap-4">
        <InfoGrid>
          <KeyValue label="Requested">
            {formatDuration(props.request.spec?.duration)}
          </KeyValue>
          {effective && (
            <KeyValue label="Effective">{formatDuration(effective)}</KeyValue>
          )}
          <KeyValue label="Maximum allowed">
            {authorization.maxAccessDuration
              ? formatDuration(authorization.maxAccessDuration)
              : "24 hours (default)"}
          </KeyValue>
        </InfoGrid>

        {capped && (
          <Note tone="amber" icon={<Info size={13} strokeWidth={2.25} />}>
            The requested duration is longer than the matched rule allows, so
            the access lasts {formatDuration(effective)}.
          </Note>
        )}

        {(policies.length > 0 || inlinePolicies.length > 0) && (
          <div className="flex flex-col gap-2 border-t border-slate-100 pt-4">
            <Eyebrow>Applied policies</Eyebrow>
            <div className="flex flex-wrap gap-1.5">
              {policies.map((policy) => (
                <MonoValue key={policy} title={policy}>
                  {policy}
                </MonoValue>
              ))}
              {inlinePolicies.map((policy, index) => (
                <MonoValue
                  key={`${policy.name}-${index}`}
                  title="Inline policy"
                  className="border-dashed"
                >
                  {policy.name || `inline-${index + 1}`}
                </MonoValue>
              ))}
            </div>
          </div>
        )}
      </div>
    </SectionCard>
  );
};

export default AuthorizationCard;
