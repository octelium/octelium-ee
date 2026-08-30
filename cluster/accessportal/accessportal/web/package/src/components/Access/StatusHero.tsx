import * as AccessP from "@/apis/accessv1/accessv1";
import { CircleDashed, Clock3, Timer, XCircle } from "lucide-react";
import { twMerge } from "tailwind-merge";

import TimeAgo, { Countdown, Elapsed } from "@/components/TimeAgo";
import { Eyebrow, SectionCard, StatusBadge } from "@/ui";
import {
  currentReviewStepIndex,
  isPendingRequest,
  reviewStepTimeoutAt,
  reviewSteps,
  statusMeta,
  toneClasses,
  tsToDate,
} from "@/utils";

import AccessWindowCard from "./AccessWindowCard";

const StatusHero = (props: { request: AccessP.Request }) => {
  const { request } = props;
  const status = statusMeta(request.status?.state?.status);

  if (status.group === "active" && request.status?.accessEndsAt) {
    return <AccessWindowCard request={request} />;
  }

  if (isPendingRequest(request)) {
    const steps = reviewSteps(request);
    const current = currentReviewStepIndex(request);
    const timeoutAt = reviewStepTimeoutAt(request);
    const deadline = tsToDate(request.spec?.deadline);

    return (
      <SectionCard
        title="Awaiting review"
        description={
          steps.length > 0
            ? `Step ${Math.min(current + 1, steps.length)} of ${steps.length} in the review workflow`
            : "The request is waiting for a decision"
        }
        icon={<Clock3 size={14} strokeWidth={2.4} />}
        tone="amber"
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <Eyebrow>Waiting for</Eyebrow>
              <p className="mt-1 text-[1.5rem] font-bold leading-none tracking-tight text-slate-900">
                <Elapsed
                  rfc3339={request.status?.approvalStartAt ?? request.metadata?.createdAt}
                  suffix=""
                />
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-4">
              {timeoutAt && (
                <div className="text-right">
                  <Eyebrow>Step timeout</Eyebrow>
                  <p className="mt-1 text-[0.82rem] font-bold text-amber-600">
                    <Countdown date={timeoutAt} suffix=" left" endedLabel="Timed out" />
                  </p>
                </div>
              )}
              {deadline && (
                <div className="text-right">
                  <Eyebrow>Deadline</Eyebrow>
                  <p className="mt-1 text-[0.82rem] font-bold text-slate-700">
                    <Countdown date={deadline} suffix=" left" endedLabel="Passed" />
                  </p>
                </div>
              )}
            </div>
          </div>

          {steps.length > 1 && (
            <div className="flex items-center gap-1.5">
              {steps.map((_, index) => (
                <span
                  key={index}
                  className={twMerge(
                    "h-1.5 flex-1 rounded-full",
                    index < current
                      ? "bg-emerald-500"
                      : index === current
                        ? "bg-amber-500"
                        : "bg-slate-200",
                  )}
                />
              ))}
            </div>
          )}
        </div>
      </SectionCard>
    );
  }

  return (
    <SectionCard
      title={status.label}
      description={status.hint}
      icon={
        status.tone === "red" ? (
          <XCircle size={14} strokeWidth={2.4} />
        ) : (
          <CircleDashed size={14} strokeWidth={2.4} />
        )
      }
      tone={status.tone}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <StatusBadge status={request.status?.state?.status} />
        <div className="flex items-center gap-1.5 text-[0.74rem] font-semibold text-slate-400">
          <Timer size={12} className={toneClasses[status.tone].text} />
          {request.status?.approvalEndAt ? (
            <>
              Decided <TimeAgo rfc3339={request.status.approvalEndAt} />
            </>
          ) : (
            <>
              Created <TimeAgo rfc3339={request.status?.state?.createdAt} />
            </>
          )}
        </div>
      </div>
    </SectionCard>
  );
};

export default StatusHero;
