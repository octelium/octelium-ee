import * as React from "react";
import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Check, CircleDashed, Clock3, FilePlus2, History, XCircle } from "lucide-react";

import TimeAgo from "@/components/TimeAgo";
import { SectionCard, Timeline, TimelineItem } from "@/ui";
import { Tone, shortName, statusMeta, tsToMillis } from "@/utils";

type Event = {
  key: string;
  at?: Timestamp;
  millis: number;
  title: string;
  tone: Tone;
  icon: React.ReactNode;
  description?: string;
};

const statusIcon = (tone: Tone, group: string) => {
  if (group === "active") return <Check size={13} strokeWidth={3} />;
  if (tone === "red") return <XCircle size={13} strokeWidth={2.6} />;
  if (group === "pending") return <Clock3 size={12} strokeWidth={2.8} />;
  return <CircleDashed size={12} strokeWidth={2.8} />;
};

const RequestTimeline = (props: { request: AccessP.Request }) => {
  const status = props.request.status;
  const events: Event[] = [];

  const pushState = (
    state: AccessP.Request_Status_State | undefined,
    key: string,
  ) => {
    if (!state) return;
    const meta = statusMeta(state.status);
    events.push({
      key,
      at: state.createdAt,
      millis: tsToMillis(state.createdAt) ?? 0,
      title: meta.label,
      tone: meta.tone,
      icon: statusIcon(meta.tone, meta.group),
      description: meta.hint,
    });
  };

  pushState(status?.state, "current-state");
  (status?.lastStates ?? []).forEach((state, index) =>
    pushState(state, `state-${index}`),
  );

  (status?.review?.lastSteps ?? []).forEach((step, index) => {
    events.push({
      key: `review-${index}`,
      at: step.setAt,
      millis: tsToMillis(step.setAt) ?? 0,
      title: `Review decision on step ${step.stepIndex + 1}`,
      tone: "blue",
      icon: <Check size={12} strokeWidth={2.8} />,
      description: shortName(step.reviewRef?.name)
        ? `Review ${shortName(step.reviewRef?.name)}`
        : undefined,
    });
  });

  const createdAt = props.request.metadata?.createdAt;
  if (createdAt) {
    events.push({
      key: "created",
      at: createdAt,
      millis: tsToMillis(createdAt) ?? 0,
      title: "Request created",
      tone: "slate",
      icon: <FilePlus2 size={12} strokeWidth={2.6} />,
    });
  }

  const sorted = events
    .filter((event) => event.millis > 0)
    .sort((a, b) => b.millis - a.millis)
    .filter(
      (event, index, list) =>
        index === 0 ||
        !(event.title === list[index - 1].title && event.millis === list[index - 1].millis),
    );

  if (sorted.length <= 1) return null;

  return (
    <SectionCard
      title="Activity"
      description="Everything that happened to this request"
      icon={<History size={14} strokeWidth={2.4} />}
    >
      <Timeline>
        {sorted.map((event, index) => (
          <TimelineItem
            key={event.key}
            tone={event.tone}
            icon={event.icon}
            title={event.title}
            meta={<TimeAgo rfc3339={event.at} />}
            last={index === sorted.length - 1}
          >
            {event.description}
          </TimelineItem>
        ))}
      </Timeline>
    </SectionCard>
  );
};

export default RequestTimeline;
