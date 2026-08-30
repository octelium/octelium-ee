import * as AccessP from "@/apis/accessv1/accessv1";
import { Tooltip } from "@mantine/core";
import { Check, Circle, Clock3, Timer, UserRound, Users, Workflow } from "lucide-react";
import { twMerge } from "tailwind-merge";

import TimeAgo, { Countdown } from "@/components/TimeAgo";
import { Avatar, Badge, Note, SectionCard } from "@/ui";
import {
  ReviewerRef,
  approvalRequirementLabel,
  currentReviewStepIndex,
  formatDuration,
  isPendingRequest,
  onTimeoutMeta,
  reviewStepTimeoutAt,
  reviewSteps,
  shortName,
  stepReviewers,
  toneClasses,
} from "@/utils";

import { useSubjectUser } from "./hooks";

const ReviewerChip = (props: { reviewer: ReviewerRef }) => {
  const isUser = props.reviewer.kind === "user";
  const query = useSubjectUser(isUser ? props.reviewer.name : undefined);
  const label = isUser
    ? query.data?.displayName || shortName(props.reviewer.name)
    : props.reviewer.name;

  return (
    <Tooltip label={`${isUser ? "User" : "Group"}: ${props.reviewer.name}`}>
      <span className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-slate-200 bg-white py-0.5 pl-0.5 pr-2.5">
        {isUser ? (
          <Avatar src={query.data?.picURL} name={label} size="xs" />
        ) : (
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-50 text-violet-600">
            <Users size={11} strokeWidth={2.6} />
          </span>
        )}
        <span className="truncate text-[0.7rem] font-bold text-slate-600">
          {label}
        </span>
      </span>
    </Tooltip>
  );
};

const StepRow = (props: {
  step: AccessP.Policy_Spec_Rule_Action_Review_Step;
  index: number;
  total: number;
  state: "complete" | "current" | "upcoming";
  decisions: AccessP.Request_Status_Review_Step[];
  timeoutAt?: Date;
}) => {
  const { state } = props;
  const tone =
    state === "complete" ? "emerald" : state === "current" ? "amber" : "slate";
  const reviewers = stepReviewers(props.step);
  const timeout = onTimeoutMeta(props.step);

  return (
    <li className="relative flex gap-3 pb-4 last:pb-0">
      {props.index < props.total - 1 && (
        <span
          className={twMerge(
            "absolute left-[13px] top-7 bottom-0 w-px",
            state === "complete" ? "bg-emerald-200" : "bg-slate-200",
          )}
        />
      )}
      <span
        className={twMerge(
          "relative z-10 flex h-[27px] w-[27px] shrink-0 items-center justify-center rounded-full text-[0.66rem] font-bold",
          toneClasses[tone].icon,
          state === "current" && "ring-2 ring-amber-200",
        )}
      >
        {state === "complete" ? (
          <Check size={13} strokeWidth={3} />
        ) : state === "current" ? (
          <Clock3 size={12} strokeWidth={2.8} />
        ) : (
          props.index + 1
        )}
      </span>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-[0.78rem] font-bold text-slate-800">
            Step {props.index + 1}
          </span>
          <Badge tone={tone}>
            {state === "complete"
              ? "Cleared"
              : state === "current"
                ? "In review"
                : "Upcoming"}
          </Badge>
          <span className="text-[0.7rem] font-semibold text-slate-400">
            {approvalRequirementLabel(props.step)}
          </span>
        </div>

        {reviewers.length > 0 && (
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            {reviewers.map((reviewer) => (
              <ReviewerChip
                key={`${reviewer.kind}-${reviewer.name}`}
                reviewer={reviewer}
              />
            ))}
          </div>
        )}

        {props.step.timeout && state !== "complete" && (
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[0.7rem] font-semibold text-slate-400">
            <span className="inline-flex items-center gap-1">
              <Timer size={11} strokeWidth={2.6} />
              Times out after {formatDuration(props.step.timeout)}
            </span>
            <span className={toneClasses[timeout.tone].text}>
              {timeout.label}
            </span>
            {state === "current" && props.timeoutAt && (
              <Countdown
                date={props.timeoutAt}
                suffix=" until timeout"
                endedLabel="Timeout reached"
                tone="amber"
              />
            )}
          </div>
        )}

        {props.decisions.length > 0 && (
          <div className="mt-2 flex flex-col gap-1">
            {props.decisions.map((decision, index) => (
              <div
                key={`${decision.reviewRef?.name}-${index}`}
                className="flex items-center gap-2 text-[0.7rem] font-semibold text-slate-500"
              >
                <UserRound size={11} strokeWidth={2.6} className="text-slate-300" />
                <span className="truncate font-mono">
                  {shortName(decision.reviewRef?.name) || "Review"}
                </span>
                <span className="ml-auto text-slate-400">
                  <TimeAgo rfc3339={decision.setAt} />
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </li>
  );
};

const ReviewWorkflow = (props: {
  request: AccessP.Request;
  className?: string;
}) => {
  const steps = reviewSteps(props.request);
  if (steps.length === 0) {
    return null;
  }

  const current = currentReviewStepIndex(props.request);
  const pending = isPendingRequest(props.request);
  const approved =
    props.request.status?.state?.status ===
    AccessP.Request_Status_State_Status.APPROVED;
  const applied = props.request.status?.review?.lastSteps ?? [];
  const timeoutAt = reviewStepTimeoutAt(props.request);

  return (
    <SectionCard
      title="Review workflow"
      description={
        pending
          ? `Currently on step ${Math.min(current + 1, steps.length)} of ${steps.length}`
          : `${steps.length} step${steps.length === 1 ? "" : "s"} defined by the matched policy rule`
      }
      icon={<Workflow size={14} strokeWidth={2.4} />}
      tone={pending ? "amber" : "slate"}
      className={props.className}
      actions={
        pending && timeoutAt ? (
          <Badge tone="amber" icon={<Timer size={10} strokeWidth={2.8} />}>
            <Countdown date={timeoutAt} suffix=" left" endedLabel="Timed out" />
          </Badge>
        ) : undefined
      }
    >
      <ol className="flex flex-col">
        {steps.map((step, index) => (
          <StepRow
            key={index}
            step={step}
            index={index}
            total={steps.length}
            state={
              approved || index < current
                ? "complete"
                : index === current && pending
                  ? "current"
                  : "upcoming"
            }
            decisions={applied.filter((item) => item.stepIndex === index)}
            timeoutAt={timeoutAt}
          />
        ))}
      </ol>

      {!pending && !approved && (
        <Note tone="slate" className="mt-3" icon={<Circle size={11} strokeWidth={2.8} />}>
          The workflow stopped before completing every step.
        </Note>
      )}
    </SectionCard>
  );
};

export default ReviewWorkflow;
