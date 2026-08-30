import * as AccessP from "@/apis/accessv1/accessv1";
import { CalendarClock, KeyRound } from "lucide-react";

import { AbsoluteTime, Countdown, absoluteLabel, useNow } from "@/components/TimeAgo";
import { Eyebrow, InfoGrid, KeyValue, Note, ProgressBar, SectionCard } from "@/ui";
import { accessWindow, formatDuration, tsToDate } from "@/utils";

const AccessWindowCard = (props: { request: AccessP.Request }) => {
  const now = useNow(15000);
  const endsAt = tsToDate(props.request.status?.accessEndsAt);
  if (!endsAt) return null;

  const window = accessWindow(props.request);
  const active = endsAt.getTime() > now;
  const tone = !active ? "slate" : window.progress > 0.8 ? "amber" : "emerald";

  return (
    <SectionCard
      title="Access window"
      description={active ? "The granted access is live" : "The granted access has ended"}
      icon={<KeyRound size={14} strokeWidth={2.4} />}
      tone={tone}
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-end justify-between gap-4">
          <div>
            <Eyebrow>{active ? "Remaining" : "Ended"}</Eyebrow>
            <p className="mt-1 text-[1.5rem] font-bold leading-none tracking-tight text-slate-900">
              <Countdown date={endsAt} suffix="" endedLabel="Access ended" tone={tone} />
            </p>
          </div>
          <div className="text-right">
            <Eyebrow>Granted for</Eyebrow>
            <p className="mt-1 text-[0.82rem] font-bold text-slate-700">
              {formatDuration(props.request.spec?.duration)}
            </p>
          </div>
        </div>

        <ProgressBar value={active ? window.progress : 1} tone={tone} />

        <InfoGrid className="border-t border-slate-100 pt-3">
          {window.startMs && (
            <KeyValue label="Started" icon={<CalendarClock size={12} className="text-slate-300" />}>
              <span className="text-[0.78rem] font-semibold text-slate-600">
                {absoluteLabel(new Date(window.startMs))}
              </span>
            </KeyValue>
          )}
          <KeyValue label="Ends" icon={<CalendarClock size={12} className="text-slate-300" />}>
            <AbsoluteTime rfc3339={props.request.status?.accessEndsAt} />
          </KeyValue>
        </InfoGrid>

        {active && window.progress > 0.8 && (
          <Note tone="amber">
            This access window is almost over. Raise a new request if you still
            need the access.
          </Note>
        )}
      </div>
    </SectionCard>
  );
};

export default AccessWindowCard;
