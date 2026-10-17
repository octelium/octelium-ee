import * as AccessP from "@/apis/accessv1/accessv1";
import { Tooltip } from "@mantine/core";
import { Check, Clock3, Minus, Users, Workflow } from "lucide-react";
import { twMerge } from "@/lib/utils";

import TimeAgo, { Countdown } from "@/components/TimeAgo";
import { Avatar, Badge, SectionCard } from "@/ui";
import {
  ReviewerRef,
  Tone,
  approvalRequirementLabel,
  currentReviewStepIndex,
  formatDuration,
  isPendingRequest,
  onTimeoutLabel,
  reviewStepLabel,
  reviewStepTimeoutAt,
  reviewSteps,
  shortName,
  stepReviewers,
  wasApproved,
} from "@/utils";

import { useSubjectUser } from "./hooks";

type StepState = "complete" | "current" | "stopped" | "upcoming" | "unreached";

const STEP_BADGE: Record<StepState, { label: string; tone: Tone }> = {
  complete: { label: "Cleared", tone: "emerald" },
  current: { label: "In review", tone: "amber" },
  stopped: { label: "Stopped here", tone: "slate" },
  upcoming: { label: "Upcoming", tone: "slate" },
  unreached: { label: "Not reached", tone: "slate" },
};

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
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-slate-100 text-slate-500">
            <Users size={11} strokeWidth={2.25} aria-hidden />
          </span>
        )}
        <span className="truncate text-xs font-medium text-slate-700">
          {isUser ? label : `${label} group`}
        </span>
      </span>
    </Tooltip>
  );
};

const DecisionLine = (props: { decision: AccessP.Request_Status_Review_Step }) => {
  const { decision } = props;
  const reviewer = decision.userRef?.name;
  const query = useSubjectUser(reviewer);
  const label = query.data?.displayName || (reviewer ? shortName(reviewer) : undefined);
  const verdict =
    decision.decision === AccessP.Review_Spec_Decision.APPROVE
      ? { label: "Approved", tone: "emerald" as Tone }
      : decision.decision === AccessP.Review_Spec_Decision.REJECT
        ? { label: "Rejected", tone: "red" as Tone }
        : undefined;

  return (
    <li className="flex flex-wrap items-center gap-1.5 text-xs font-normal text-slate-500">
      {verdict ? (
        <Badge tone={verdict.tone}>{verdict.label}</Badge>
      ) : (
        <span aria-hidden className="h-1 w-1 rounded-full bg-slate-400" />
      )}
      {label ? (
        <span className="inline-flex items-center gap-1 font-medium text-slate-700">
          <Avatar src={query.data?.picURL} name={label} size="xs" />
          {label}
        </span>
      ) : (
        <span>Decision recorded</span>
      )}
      <TimeAgo rfc3339={decision.setAt} />
    </li>
  );
};

const StepRow = (props: {
  step: AccessP.Policy_Spec_Rule_Action_Review_Step;
  index: number;
  total: number;
  state: StepState;
  decisions: AccessP.Request_Status_Review_Step[];
  timeoutAt?: Date;
}) => {
  const { state } = props;
  const reviewers = stepReviewers(props.step);
  const badge = STEP_BADGE[state];

  return (
    <li className="relative flex gap-3 pb-5 last:pb-0">
      {props.index < props.total - 1 && (
        <span
          aria-hidden
          className={twMerge(
            "absolute bottom-0 left-[13px] top-7 w-px",
            state === "complete" ? "bg-emerald-300" : "bg-slate-200",
          )}
        />
      )}
      <span
        aria-hidden
        className={twMerge(
          "relative z-10 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-micro font-semibold",
          state === "complete"
            ? "border-emerald-200 bg-emerald-50 text-emerald-700"
            : state === "current"
              ? "border-slate-900 bg-white text-slate-900"
              : "border-slate-200 bg-white text-slate-500",
        )}
      >
        {state === "complete" ? (
          <Check size={13} strokeWidth={2.75} />
        ) : state === "current" ? (
          <Clock3 size={12} strokeWidth={2.5} />
        ) : state === "stopped" || state === "unreached" ? (
          <Minus size={12} strokeWidth={2.5} />
        ) : (
          props.index + 1
        )}
      </span>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-body font-semibold text-slate-800">
            {reviewStepLabel(props.step, props.index)}
          </span>
          <Badge tone={badge.tone}>{badge.label}</Badge>
        </div>
        <p className="mt-0.5 text-xs font-normal text-slate-500">
          {approvalRequirementLabel(props.step)}
        </p>

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

        {props.step.timeout && (state === "current" || state === "upcoming") && (
          <p className="mt-2 text-xs font-normal text-slate-500">
            Times out after {formatDuration(props.step.timeout)},{" "}
            {onTimeoutLabel(props.step)}
            {state === "current" && props.timeoutAt && (
              <>
                {" · "}
                <Countdown
                  date={props.timeoutAt}
                  suffix=" left"
                  endedLabel="timeout reached"
                  className="font-medium text-slate-700"
                />
              </>
            )}
          </p>
        )}

        {props.decisions.length > 0 && (
          <ul className="mt-2 flex flex-col gap-1">
            {props.decisions.map((decision, index) => (
              <DecisionLine
                key={`${decision.reviewRef?.uid ?? decision.reviewRef?.name}-${index}`}
                decision={decision}
              />
            ))}
          </ul>
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
  const completed = wasApproved(props.request);
  const applied = props.request.status?.review?.lastSteps ?? [];
  const timeoutAt = reviewStepTimeoutAt(props.request);

  const stateOf = (index: number): StepState => {
    if (completed || index < current) return "complete";
    if (index === current) return pending ? "current" : "stopped";
    return pending ? "upcoming" : "unreached";
  };

  return (
    <SectionCard
      title="Review workflow"
      description={
        pending
          ? `Waiting on step ${Math.min(current + 1, steps.length)} of ${steps.length}`
          : completed
            ? `All ${steps.length} step${steps.length === 1 ? "" : "s"} cleared`
            : `Stopped at step ${Math.min(current + 1, steps.length)} of ${steps.length}`
      }
      icon={<Workflow size={14} strokeWidth={2.25} />}
      className={props.className}
    >
      <ol className="flex flex-col">
        {steps.map((step, index) => (
          <StepRow
            key={index}
            step={step}
            index={index}
            total={steps.length}
            state={stateOf(index)}
            decisions={applied.filter((item) => item.stepIndex === index)}
            timeoutAt={index === current ? timeoutAt : undefined}
          />
        ))}
      </ol>
    </SectionCard>
  );
};

export default ReviewWorkflow;
