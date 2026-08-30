import * as AccessP from "@/apis/accessv1/accessv1";
import { FileCheck2, ShieldCheck } from "lucide-react";

import { Eyebrow, InfoGrid, KeyValue, MonoValue, Note, SectionCard } from "@/ui";
import { durationToSeconds, formatDuration } from "@/utils";

const AuthorizationCard = (props: { request: AccessP.Request }) => {
  const authorization = props.request.status?.rule?.authorization;
  if (!authorization) return null;

  const policies = authorization.policies ?? [];
  const inlinePolicies = authorization.inlinePolicies ?? [];
  const maxSeconds = durationToSeconds(authorization.maxAccessDuration);
  const requestedSeconds = durationToSeconds(props.request.spec?.duration);
  const capped = maxSeconds > 0 && requestedSeconds > maxSeconds;

  return (
    <SectionCard
      title="Granted authorization"
      description="The access that applies once this request is approved"
      icon={<ShieldCheck size={14} strokeWidth={2.4} />}
      tone="emerald"
    >
      <div className="flex flex-col gap-4">
        <InfoGrid>
          <KeyValue label="Requested duration">
            {formatDuration(props.request.spec?.duration)}
          </KeyValue>
          <KeyValue label="Maximum allowed">
            {authorization.maxAccessDuration
              ? formatDuration(authorization.maxAccessDuration)
              : "24 hours (default)"}
          </KeyValue>
        </InfoGrid>

        {(policies.length > 0 || inlinePolicies.length > 0) && (
          <div className="flex flex-col gap-2 border-t border-slate-100 pt-4">
            <Eyebrow>Applied policies</Eyebrow>
            <div className="flex flex-wrap gap-1.5">
              {policies.map((policy) => (
                <MonoValue key={policy}>{policy}</MonoValue>
              ))}
              {inlinePolicies.map((policy, index) => (
                <MonoValue
                  key={`${policy.name}-${index}`}
                  className="bg-violet-50 text-violet-700"
                >
                  {policy.name || `inline-${index + 1}`}
                </MonoValue>
              ))}
            </div>
          </div>
        )}

        {capped && (
          <Note tone="amber" icon={<FileCheck2 size={13} strokeWidth={2.4} />}>
            The requested duration is longer than the maximum allowed by the
            matched rule and will be capped.
          </Note>
        )}
      </div>
    </SectionCard>
  );
};

export default AuthorizationCard;
