import * as AccessP from "@/apis/accessv1/accessv1";
import { Button } from "@mantine/core";
import { KeyRound, RotateCcw } from "lucide-react";
import { Link } from "react-router-dom";

import { AbsoluteTime, Countdown } from "@/components/TimeAgo";
import {
  Eyebrow,
  InfoGrid,
  KeyValue,
  Note,
  ProgressBar,
  SectionCard,
} from "@/ui";
import {
  accessWindow,
  formatDuration,
  grantedDuration,
  newRequestHref,
  tsToDate,
} from "@/utils";
import { absoluteLabel, useNow } from "@/utils/time";

const AccessWindowCard = (props: {
  request: AccessP.Request;
  canRequestAgain?: boolean;
}) => {
  const now = useNow();
  const endsAt = tsToDate(props.request.status?.accessEndsAt);
  if (!endsAt) return null;

  const grant = accessWindow(props.request, now);
  const active = endsAt.getTime() > now;
  const endingSoon = active && grant.progress > 0.8;

  return (
    <SectionCard
      title="Access window"
      description={active ? "The granted access is live" : "The granted access has ended"}
      icon={<KeyRound size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-end justify-between gap-4">
          <div>
            <Eyebrow>{active ? "Remaining" : "Ended"}</Eyebrow>
            <p className="mt-1 text-2xl font-semibold leading-none tracking-tight text-slate-900">
              <Countdown date={endsAt} suffix="" endedLabel="Access ended" />
            </p>
          </div>
          <div className="text-right">
            <Eyebrow>Granted for</Eyebrow>
            <p className="mt-1 text-body font-semibold text-slate-800">
              {formatDuration(grantedDuration(props.request))}
            </p>
          </div>
        </div>

        <ProgressBar
          value={active ? grant.progress : 1}
          tone={!active ? "slate" : endingSoon ? "amber" : "emerald"}
          label="Elapsed share of the access window"
        />

        <InfoGrid className="border-t border-slate-100 pt-3">
          {grant.startMs && (
            <KeyValue label="Started">
              <time dateTime={new Date(grant.startMs).toISOString()}>
                {absoluteLabel(new Date(grant.startMs))}
              </time>
            </KeyValue>
          )}
          <KeyValue label="Ends">
            <AbsoluteTime rfc3339={props.request.status?.accessEndsAt} />
          </KeyValue>
        </InfoGrid>

        {(endingSoon || !active) && props.canRequestAgain && (
          <Note tone={endingSoon ? "amber" : "slate"}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span>
                {endingSoon
                  ? "This access window is almost over."
                  : "Raise a new request if you still need this access."}
              </span>
              <Button
                component={Link}
                to={newRequestHref(props.request)}
                size="compact-xs"
                variant="default"
                leftSection={<RotateCcw size={11} strokeWidth={2.5} />}
              >
                Request again
              </Button>
            </div>
          </Note>
        )}
      </div>
    </SectionCard>
  );
};

export default AccessWindowCard;
