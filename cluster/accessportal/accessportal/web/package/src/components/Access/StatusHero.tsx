import * as AccessP from "@/apis/accessv1/accessv1";
import { Button } from "@mantine/core";
import { CircleDashed, Clock3, Loader2, RotateCcw } from "lucide-react";
import { Link } from "react-router-dom";
import { twMerge } from "@/lib/utils";

import TimeAgo, { Countdown, Elapsed } from "@/components/TimeAgo";
import { Eyebrow, SectionCard, StatusBadge } from "@/ui";
import {
  currentReviewStepIndex,
  isEvaluatingRequest,
  isPendingRequest,
  newRequestHref,
  reviewStepLabel,
  reviewStepTimeoutAt,
  reviewSteps,
  statusMeta,
  tsToDate,
} from "@/utils";

import AccessWindowCard from "./AccessWindowCard";

const StatusHero = (props: {
  request: AccessP.Request;
  canRequestAgain?: boolean;
}) => {
  const { request } = props;
  const status = statusMeta(request.status?.state?.status);

  if (status.group === "active" && request.status?.accessEndsAt) {
    return (
      <AccessWindowCard
        request={request}
        canRequestAgain={props.canRequestAgain}
      />
    );
  }

  if (isEvaluatingRequest(request)) {
    return (
      <SectionCard
        title="Evaluating"
        description="The Cluster is matching this request against the access policies"
        icon={<Loader2 size={14} strokeWidth={2.25} className="motion-safe:animate-spin" />}
      >
        <p className="text-body font-normal text-slate-600">
          This usually takes a few seconds. The page updates on its own once
          the request is approved, rejected or routed to its reviewers.
        </p>
      </SectionCard>
    );
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
            ? `Step ${Math.min(current + 1, steps.length)} of ${steps.length}: ${reviewStepLabel(steps[current], current)}`
            : "The request is waiting for a decision"
        }
        icon={<Clock3 size={14} strokeWidth={2.25} />}
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <Eyebrow>Waiting for</Eyebrow>
              <p className="mt-1 text-2xl font-semibold leading-none tracking-tight text-slate-900">
                <Elapsed
                  rfc3339={request.status?.approvalStartAt ?? request.metadata?.createdAt}
                  suffix=""
                />
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-5">
              {timeoutAt && (
                <div className="text-right">
                  <Eyebrow>Step timeout</Eyebrow>
                  <p className="mt-1 text-body font-semibold text-slate-800">
                    <Countdown date={timeoutAt} suffix=" left" endedLabel="Timed out" />
                  </p>
                </div>
              )}
              {deadline && (
                <div className="text-right">
                  <Eyebrow>Deadline</Eyebrow>
                  <p className="mt-1 text-body font-semibold text-slate-800">
                    <Countdown date={deadline} suffix=" left" endedLabel="Passed" />
                  </p>
                </div>
              )}
            </div>
          </div>

          {steps.length > 1 && (
            <div className="flex items-center gap-1.5" aria-hidden>
              {steps.map((_, index) => (
                <span
                  key={index}
                  className={twMerge(
                    "h-1.5 flex-1 rounded-full",
                    index < current
                      ? "bg-emerald-500"
                      : index === current
                        ? "bg-slate-500"
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
      icon={<CircleDashed size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2 text-xs font-normal text-slate-500">
          <StatusBadge status={request.status?.state?.status} />
          {request.status?.approvalEndAt ? (
            <span>
              Decided <TimeAgo rfc3339={request.status.approvalEndAt} />
            </span>
          ) : (
            <span>
              Updated <TimeAgo rfc3339={request.status?.state?.createdAt} />
            </span>
          )}
        </div>
        {props.canRequestAgain && (
          <Button
            component={Link}
            to={newRequestHref(request)}
            size="compact-sm"
            variant="default"
            leftSection={<RotateCcw size={12} strokeWidth={2.25} />}
          >
            Request again
          </Button>
        )}
      </div>
    </SectionCard>
  );
};

export default StatusHero;
