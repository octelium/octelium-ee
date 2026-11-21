import * as React from "react";
import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  FilePlus2,
  Gavel,
  History,
  MessageCircleQuestionMark,
  MessageSquareReply,
} from "lucide-react";

import TimeAgo from "@/components/TimeAgo";
import { SectionCard, Timeline, TimelineItem } from "@/ui";
import {
  Tone,
  requestCreatedAt,
  reviewStepLabel,
  reviewSteps,
  shortName,
  stateDescription,
  statusMeta,
  tsToMillis,
} from "@/utils";

type Event = {
  key: string;
  at?: Timestamp;
  millis: number;
  title: string;
  tone?: Tone;
  icon?: React.ReactNode;
  description?: string;
};

const reviewDecisionTitle = (step: AccessP.Request_Status_Review_Step) => {
  const reviewer = step.userRef?.name ? shortName(step.userRef.name) : "a reviewer";
  switch (step.decision) {
    case AccessP.Review_Spec_Decision.APPROVE:
      return `Approved by ${reviewer}`;
    case AccessP.Review_Spec_Decision.REJECT:
      return `Rejected by ${reviewer}`;
    default:
      return "Reviewer decision recorded";
  }
};

const RequestTimeline = (props: { request: AccessP.Request }) => {
  const status = props.request.status;
  const steps = reviewSteps(props.request);
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
      description: stateDescription(state),
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
      title: reviewDecisionTitle(step),
      tone:
        step.decision === AccessP.Review_Spec_Decision.REJECT
          ? "red"
          : step.decision === AccessP.Review_Spec_Decision.APPROVE
            ? "emerald"
            : undefined,
      icon: <Gavel size={11} strokeWidth={2.25} />,
      description: step.justification
        ? `${step.stepName || reviewStepLabel(steps[step.stepIndex], step.stepIndex)}: “${step.justification}”`
        : step.stepName || reviewStepLabel(steps[step.stepIndex], step.stepIndex),
    });
  });

  (status?.review?.informationRequests ?? []).forEach((info, index) => {
    const asker = info.userRef?.name
      ? shortName(info.userRef.name)
      : "A reviewer";
    events.push({
      key: `info-${index}`,
      at: info.requestedAt,
      millis: tsToMillis(info.requestedAt) ?? 0,
      title: `${asker} asked for more information`,
      icon: <MessageCircleQuestionMark size={11} strokeWidth={2.25} />,
      description: info.message,
    });
    if (info.respondedAt) {
      events.push({
        key: `info-response-${index}`,
        at: info.respondedAt,
        millis: tsToMillis(info.respondedAt) ?? 0,
        title: `${info.responderRef?.name ? shortName(info.responderRef.name) : "The requester"} responded`,
        icon: <MessageSquareReply size={11} strokeWidth={2.25} />,
        description: info.response,
      });
    }
  });

  const createdAt = requestCreatedAt(props.request);
  if (createdAt) {
    events.push({
      key: "created",
      at: createdAt,
      millis: tsToMillis(createdAt) ?? 0,
      title: "Request created",
      icon: <FilePlus2 size={11} strokeWidth={2.25} />,
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
      icon={<History size={14} strokeWidth={2.25} />}
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
