import * as AccessP from "@/apis/accessv1/accessv1";
import { Tooltip } from "@mantine/core";
import {
  Check,
  Clock3,
  Minus,
  ShieldAlert,
  Users,
  Workflow,
} from "lucide-react";
import { twMerge } from "@/lib/utils";

import TimeAgo, { Countdown } from "@/components/TimeAgo";
import { Avatar, Badge, Note, SectionCard } from "@/ui";
import {
  ReviewerRef,
  Tone,
  WorkflowStep,
  approvalRequirementLabel,
  currentReviewStepIndex,
  formatDuration,
  isPendingRequest,
  onTimeoutLabel,
  reasonRequirementLabel,
  reviewStepLabel,
  reviewStepState,
  reviewStepTimeoutAt,
  reviewSteps,
  shortName,
  stepReviewers,
  wasApproved,
} from "@/utils";

type StepState = "complete" | "current" | "stopped" | "upcoming" | "unreached";

const STEP_BADGE: Record<StepState, { label: string; tone: Tone }> = {
  complete: { label: "Cleared", tone: "emerald" },
  current: { label: "In review", tone: "amber" },
  stopped: { label: "Stopped here", tone: "slate" },
  upcoming: { label: "Upcoming", tone: "slate" },
  unreached: { label: "Not reached", tone: "slate" },
};

const REVIEWER_KIND: Record<ReviewerRef["kind"], string> = {
  user: "User",
  group: "Group",
  owners: "Catalog owners",
};

const ReviewerChip = (props: { reviewer: ReviewerRef }) => {
  const isUser = props.reviewer.kind === "user";
  const label = props.reviewer.name;

  return (
    <Tooltip
      label={`${REVIEWER_KIND[props.reviewer.kind]}: ${props.reviewer.name}`}
    >
      <span className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-slate-200 bg-white py-0.5 pl-0.5 pr-2.5">
        {isUser ? (
          <Avatar name={label} size="xs" />
        ) : (
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-slate-100 text-slate-500">
            <Users size={11} strokeWidth={2.25} aria-hidden />
          </span>
        )}
        <span className="truncate text-xs font-medium text-slate-700">
          {props.reviewer.kind === "group" ? `${label} group` : label}
        </span>
      </span>
    </Tooltip>
  );
};

const DecisionLine = (props: { decision: AccessP.Request_Status_Review_Step }) => {
  const { decision } = props;
  const reviewer = decision.userRef?.name;
  const label = reviewer ? shortName(reviewer) : undefined;
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
          <Avatar name={label} size="xs" />
          {label}
        </span>
      ) : (
        <span>Decision recorded</span>
      )}
      <TimeAgo rfc3339={decision.setAt} />
      {decision.justification && (
        <span className="basis-full pl-1 text-slate-600 [overflow-wrap:anywhere]">
          “{decision.justification}”
        </span>
      )}
    </li>
  );
};

const OUTCOME_BADGE: Partial<
  Record<
    AccessP.Request_Status_Review_StepState_Outcome,
    { label: string; tone: Tone }
  >
> = {
  [AccessP.Request_Status_Review_StepState_Outcome.TIMED_OUT]: {
    label: "Timed out",
    tone: "slate",
  },
  [AccessP.Request_Status_Review_StepState_Outcome.REJECTED]: {
    label: "Rejected",
    tone: "red",
  },
};

const StepRow = (props: {
  step: WorkflowStep;
  stepState?: AccessP.Request_Status_Review_StepState;
  index: number;
  total: number;
  state: StepState;
  decisions: AccessP.Request_Status_Review_Step[];
  timeoutAt?: Date;
}) => {
  const { state, stepState } = props;
  const reviewers = stepReviewers(props.step);
  const badge =
    (stepState?.outcome !== undefined && OUTCOME_BADGE[stepState.outcome]) ||
    STEP_BADGE[state];
  const reasonLabel = reasonRequirementLabel(props.step);

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
          {stepState?.isEscalated && <Badge tone="amber">Escalated</Badge>}
        </div>
        <p className="mt-0.5 text-xs font-normal text-slate-500">
          {approvalRequirementLabel(props.step)}
          {state === "current" &&
            stepState &&
            stepState.requiredApprovals > 0 && (
              <>
                {" · "}
                <span className="font-medium text-slate-700">
                  {stepState.approvals} of {stepState.requiredApprovals}{" "}
                  approval
                  {stepState.requiredApprovals === 1 ? "" : "s"}
                </span>
              </>
            )}
          {reasonLabel && <> · {reasonLabel}</>}
        </p>

        {state === "current" && stepState?.isUnroutable && (
          <Note
            tone="amber"
            icon={<ShieldAlert size={13} strokeWidth={2.25} />}
            className="mt-2"
          >
            No one can currently review this step. An administrator has been
            notified.
          </Note>
        )}

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
            stepState={reviewStepState(props.request, index)}
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
