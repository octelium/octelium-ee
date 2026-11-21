import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Checkbox, Modal, Textarea } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  Gavel,
  History,
  Info,
  MessageCircleQuestionMark,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { twMerge } from "@/lib/utils";

import GrantCard from "@/components/Access/GrantCard";
import InformationRequestCard from "@/components/Access/InformationRequestCard";
import JustificationCard from "@/components/Access/JustificationCard";
import PeopleCard from "@/components/Access/PeopleCard";
import RequestFacts from "@/components/Access/RequestFacts";
import RequestTimeline from "@/components/Access/RequestTimeline";
import ResourceCard from "@/components/Access/ResourceCard";
import ReviewWorkflow from "@/components/Access/ReviewWorkflow";
import StatusHero from "@/components/Access/StatusHero";
import DurationInput from "@/components/DurationInput";
import TimeAgo from "@/components/TimeAgo";
import UnsavedChangesDialog from "@/components/UnsavedChangesDialog";
import {
  BackLink,
  Badge,
  ConfirmDialog,
  DocumentTitle,
  ErrorState,
  LengthCounter,
  Loading,
  Note,
  NotFoundState,
  PageHeader,
  SectionCard,
  StatusBadge,
  SummaryRow,
  UrgencyBadge,
} from "@/ui";
import {
  MAX_JUSTIFICATION_LENGTH,
  MAX_MESSAGE_LENGTH,
  approvalRequirementLabel,
  currentReviewStepIndex,
  durationToSeconds,
  formatDuration,
  grantedDuration,
  isPendingRequest,
  isReasonRequired,
  normalizeDuration,
  requestResourceLabel,
  requestSubjectName,
  reviewStepLabel,
  reviewStepName,
  reviewSteps,
  statusMeta,
  textLength,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";
import { isNotFoundError } from "@/utils/rpcError";
import { useUnsavedChanges } from "@/utils/hooks/useUnsavedChanges";

const A = AccessP.ReviewCase_Action;

const FLAG_LABELS: {
  key: keyof AccessP.ReviewCase_Flags;
  label: string;
  tone?: "amber" | "blue";
}[] = [
  { key: "isSpecChanged", label: "Changed since your decision", tone: "amber" },
  { key: "isAwaitingRequester", label: "Awaiting the requester", tone: "blue" },
  { key: "overridesGuardrails", label: "Overrides guardrails", tone: "amber" },
  { key: "isFullAccess", label: "Full access" },
  { key: "isOnBehalf", label: "On behalf of another user" },
  {
    key: "isFirstRequest",
    label: "First request for this access",
    tone: "blue",
  },
  { key: "isExtension", label: "Extension" },
];

const RequesterHistoryCard = (props: { reviewCase: AccessP.ReviewCase }) => {
  const items = [
    ...props.reviewCase.activeRequests.map((item) => ({ item, active: true })),
    ...props.reviewCase.recentRequests.map((item) => ({ item, active: false })),
  ];
  const groups = props.reviewCase.subjectGroups;
  if (items.length === 0 && groups.length === 0) return null;

  return (
    <SectionCard
      title="Context"
      description="The recipient's groups and their other requests for this access"
      icon={<History size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-col gap-3">
        {groups.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {groups.map((group) => (
              <Badge key={group}>{group}</Badge>
            ))}
          </div>
        )}
        {items.length > 0 && (
          <ul className="flex flex-col gap-1.5">
            {items.map(({ item, active }) => (
              <li
                key={item.metadata?.name}
                className="flex flex-wrap items-center gap-1.5 text-xs font-normal text-slate-600"
              >
                <Badge
                  tone={
                    active
                      ? "emerald"
                      : statusMeta(item.status?.state?.status).tone
                  }
                >
                  {active
                    ? "Active"
                    : statusMeta(item.status?.state?.status).label}
                </Badge>
                <span className="font-mono text-slate-700">
                  {item.metadata?.name}
                </span>
                <TimeAgo rfc3339={item.metadata?.createdAt} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </SectionCard>
  );
};

const DecisionPanel = (props: {
  className?: string;
  pending: boolean;
  actions: AccessP.ReviewCase_Action[];
  requirement: string;
  reasonHint: string;
  justification: string;
  onJustificationChange: (value: string) => void;
  error?: string;
  busy: boolean;
  shorten: boolean;
  onShortenChange: (value: boolean) => void;
  duration: MetaP.Duration;
  onDurationChange: (value: MetaP.Duration) => void;
  onApprove: () => void;
  onReject: () => void;
  onAskInformation: () => void;
}) => {
  const length = textLength(props.justification);
  const canApprove = props.actions.includes(A.APPROVE);
  const canReject = props.actions.includes(A.REJECT);
  const canAsk = props.actions.includes(A.REQUEST_INFORMATION);
  const canDecide = canApprove || canReject;

  return (
    <SectionCard
      title="Your decision"
      description={
        props.pending ? props.requirement : "This request is no longer awaiting review"
      }
      icon={<Gavel size={14} strokeWidth={2.25} />}
      className={props.className}
    >
      {props.pending && canDecide ? (
        <div className="flex flex-col gap-4">
          <Textarea
            label={
              <span className="flex w-full items-baseline justify-between gap-3">
                Reason
                <LengthCounter length={length} max={MAX_JUSTIFICATION_LENGTH} />
              </span>
            }
            description={props.reasonHint}
            error={props.error}
            autosize
            minRows={3}
            maxRows={7}
            placeholder="Explain your decision…"
            value={props.justification}
            onChange={(e) => props.onJustificationChange(e.currentTarget.value)}
            styles={{ label: { display: "flex", width: "100%" } }}
          />

          {canApprove && (
            <div className="flex flex-col gap-2">
              <Checkbox
                label="Approve a shorter duration"
                checked={props.shorten}
                onChange={(event) =>
                  props.onShortenChange(event.currentTarget.checked)
                }
              />
              {props.shorten && (
                <DurationInput
                  value={props.duration}
                  onChange={props.onDurationChange}
                />
              )}
            </div>
          )}

          <div className="grid grid-cols-2 gap-2">
            <Button
              variant="default"
              leftSection={<X size={14} strokeWidth={2.5} />}
              disabled={props.busy || !canReject}
              onClick={props.onReject}
            >
              Reject
            </Button>
            <Button
              variant="filled"
              color="ink"
              leftSection={<Check size={14} strokeWidth={2.5} />}
              disabled={props.busy || !canApprove}
              onClick={props.onApprove}
            >
              Approve
            </Button>
          </div>

          {canAsk && (
            <Button
              variant="subtle"
              color="gray"
              leftSection={
                <MessageCircleQuestionMark size={14} strokeWidth={2.25} />
              }
              disabled={props.busy}
              onClick={props.onAskInformation}
            >
              Ask the requester a question
            </Button>
          )}

          <Note icon={<Info size={13} strokeWidth={2.25} />}>
            A single rejection ends the request immediately. An approval counts
            towards the current step of the workflow.
          </Note>
        </div>
      ) : props.pending ? (
        <Note>
          You have already decided on the current step, or you are not one of
          its reviewers.
        </Note>
      ) : (
        <Note>A decision has already been reached for this request.</Note>
      )}
    </SectionCard>
  );
};

const ReviewRequest = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [justification, setJustification] = React.useState("");
  const [justificationError, setJustificationError] = React.useState<string>();
  const [decisionToSubmit, setDecisionToSubmit] =
    React.useState<AccessP.Review_Spec_Decision>();
  const [shorten, setShorten] = React.useState(false);
  const [approvedDuration, setApprovedDuration] =
    React.useState<MetaP.Duration>();
  const [askOpen, setAskOpen] = React.useState(false);
  const [question, setQuestion] = React.useState("");
  const unsavedChanges = useUnsavedChanges(
    justification !== "" || question !== "",
  );

  const qry = useQuery({
    queryKey: ["reviewer", "getReviewCase", name],
    enabled: !!name,
    queryFn: async ({ signal }) => {
      const { response } = await getReviewerClient().getReviewCase(
        AccessP.GetReviewCaseRequest.create({
          requestRef: MetaP.ObjectReference.create({ name: name! }),
        }),
        { abort: signal },
      );
      return response;
    },
    refetchInterval: (query) =>
      isPendingRequest(query.state.data?.request) ? 15000 : false,
  });

  const reviewCase = qry.data;
  const item = reviewCase?.request;
  const steps = reviewSteps(item);
  const currentIndex = currentReviewStepIndex(item);
  const currentStep = steps[currentIndex];
  const duration = approvedDuration ?? normalizeDuration(grantedDuration(item));

  const decideMutation = useMutation({
    mutationFn: async (decision: AccessP.Review_Spec_Decision) => {
      const { response } = await getReviewerClient().setReviewDecision(
        AccessP.SetReviewDecisionRequest.create({
          requestRef: MetaP.ObjectReference.create({ name: name! }),
          decision,
          justification,
          expectedStepName: reviewStepName(currentStep, currentIndex),
          expectedSpecRevision: item?.status?.specRevision ?? 0,
          duration:
            shorten && decision === AccessP.Review_Spec_Decision.APPROVE
              ? duration
              : undefined,
        }),
      );
      return response;
    },
    onSuccess: () => {
      notifications.show({ color: "green", message: "Review submitted" });
      setDecisionToSubmit(undefined);
      queryClient.invalidateQueries({ queryKey: ["reviewer"] });
      unsavedChanges.allowNavigation(() => navigate("/reviewer/requests"));
    },
    onError: (e) => {
      setDecisionToSubmit(undefined);
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to submit review",
      });
      void qry.refetch();
    },
  });

  const askMutation = useMutation({
    mutationFn: async () => {
      const { response } = await getReviewerClient().requestInformation(
        AccessP.RequestInformationRequest.create({
          requestRef: MetaP.ObjectReference.create({ name: name! }),
          message: question.trim(),
        }),
      );
      return response;
    },
    onSuccess: () => {
      notifications.show({
        color: "green",
        message: "Question sent to the requester",
      });
      setAskOpen(false);
      setQuestion("");
      queryClient.invalidateQueries({ queryKey: ["reviewer"] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to send the question",
      });
    },
  });

  if (qry.isLoading) return <Loading label="Loading request…" />;
  if (!item || !reviewCase) {
    return (
      <div className="w-full">
        <DocumentTitle title="Review" />
        <BackLink to="/reviewer/requests">Review Queue</BackLink>
        {qry.isError && !isNotFoundError(qry.error) ? (
          <ErrorState
            title="Could not load this request"
            error={qry.error}
            onRetry={() => qry.refetch()}
          />
        ) : (
          <NotFoundState
            title="This request is not waiting for you"
            description="It may have been decided already, moved to a step you do not review, or it does not exist."
            action={
              <Button component={Link} to="/reviewer/requests" variant="default">
                Back to the Review Queue
              </Button>
            }
          />
        )}
      </div>
    );
  }

  const resource = requestResourceLabel(item);
  const pending = isPendingRequest(item);
  const stepLabel = currentStep ? reviewStepLabel(currentStep, currentIndex) : undefined;
  const tooLong = textLength(justification) > MAX_JUSTIFICATION_LENGTH;
  const questionTooLong = textLength(question) > MAX_MESSAGE_LENGTH;
  const reasonAlways = isReasonRequired(
    currentStep,
    AccessP.Review_Spec_Decision.APPROVE,
  );
  const flags = FLAG_LABELS.filter((flag) => reviewCase.flags?.[flag.key]);
  const shorterThanRequested =
    !shorten ||
    durationToSeconds(duration) < durationToSeconds(grantedDuration(item));

  const requestDecision = (decision: AccessP.Review_Spec_Decision) => {
    if (tooLong) {
      setJustificationError("The reason is too long");
      return;
    }
    if (
      (decision === AccessP.Review_Spec_Decision.REJECT ||
        isReasonRequired(currentStep, decision)) &&
      !justification.trim()
    ) {
      setJustificationError(
        decision === AccessP.Review_Spec_Decision.REJECT
          ? "Add a reason before rejecting this request"
          : "Add a reason before approving this request",
      );
      return;
    }
    if (
      decision === AccessP.Review_Spec_Decision.APPROVE &&
      !shorterThanRequested
    ) {
      setJustificationError(
        "The approved duration must be shorter than the requested one",
      );
      return;
    }
    setJustificationError(undefined);
    setDecisionToSubmit(decision);
  };

  const panelProps = {
    pending,
    actions: reviewCase.allowedActions,
    requirement: [stepLabel, approvalRequirementLabel(currentStep)]
      .filter(Boolean)
      .join(" · "),
    reasonHint: reasonAlways
      ? "Required for every decision on this step"
      : "Required when rejecting, recommended when approving",
    shorten,
    onShortenChange: setShorten,
    duration,
    onDurationChange: setApprovedDuration,
    onAskInformation: () => setAskOpen(true),
    justification,
    onJustificationChange: (value: string) => {
      setJustification(value);
      if (justificationError) setJustificationError(undefined);
    },
    error: justificationError,
    busy: decideMutation.isPending,
    onApprove: () => requestDecision(AccessP.Review_Spec_Decision.APPROVE),
    onReject: () => requestDecision(AccessP.Review_Spec_Decision.REJECT),
  };

  const rejecting = decisionToSubmit === AccessP.Review_Spec_Decision.REJECT;

  return (
    <div className="w-full">
      <DocumentTitle
        title={`Review ${resource.displayName || resource.name || item.metadata!.name}`}
      />
      <UnsavedChangesDialog guard={unsavedChanges} />
      <BackLink to="/reviewer/requests">Review Queue</BackLink>

      <PageHeader
        eyebrow={
          resource.accessLevel
            ? `${resource.accessLevel} access request`
            : `${resource.kind} access request`
        }
        title={resource.displayName || resource.name || item.metadata!.name}
        meta={
          <>
            <StatusBadge status={item.status?.state?.status} withHint />
            <UrgencyBadge urgency={item.spec?.urgency} />
            {flags.map((flag) => (
              <Badge key={flag.key} tone={flag.tone}>
                {flag.label}
              </Badge>
            ))}
          </>
        }
      />

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <StatusHero request={item} />
          <InformationRequestCard request={item} />
          <JustificationCard
            text={item.spec?.justification}
            description="Why the requester says the access is needed"
            emptyLabel="The requester did not explain why this access is needed. Consider asking before approving."
            emptyTone="amber"
          />
          <PeopleCard request={item} />
          <ResourceCard request={item} reviewer />
          <DecisionPanel {...panelProps} className="lg:hidden" />
          <GrantCard request={item} />
          <RequesterHistoryCard reviewCase={reviewCase} />
          <ReviewWorkflow request={item} />
          <RequestTimeline request={item} />
        </div>

        <div className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-20">
          <DecisionPanel {...panelProps} className="hidden lg:block" />
          <SectionCard title="Request details" bodyClassName="px-4 py-1">
            <RequestFacts request={item} showPolicy />
          </SectionCard>
        </div>
      </div>

      <ConfirmDialog
        opened={decisionToSubmit !== undefined}
        onClose={() => setDecisionToSubmit(undefined)}
        onConfirm={() => {
          if (decisionToSubmit !== undefined) decideMutation.mutate(decisionToSubmit);
        }}
        title={rejecting ? "Reject this request?" : "Approve this request?"}
        description={
          rejecting
            ? "The request is rejected immediately and no access is granted."
            : "Your approval is recorded for the current step. Access is granted once every step of the workflow is satisfied."
        }
        details={
          <div className="flex flex-col">
            <SummaryRow label={resource.kind}>
              {resource.name || item.metadata!.name}
            </SummaryRow>
            <SummaryRow label="For">{requestSubjectName(item) || "—"}</SummaryRow>
            <SummaryRow label="Duration">
              {formatDuration(
                shorten && !rejecting ? duration : grantedDuration(item),
              )}
            </SummaryRow>
            {stepLabel && <SummaryRow label="Step">{stepLabel}</SummaryRow>}
            <SummaryRow label="Your reason">
              <span className={twMerge(!justification.trim() && "text-slate-500")}>
                {justification.trim() || "None"}
              </span>
            </SummaryRow>
          </div>
        }
        confirmLabel={rejecting ? "Reject request" : "Approve request"}
        danger={rejecting}
        loading={decideMutation.isPending}
      />

      <Modal
        opened={askOpen}
        onClose={() => setAskOpen(false)}
        centered
        radius="lg"
        title="Ask the requester a question"
        styles={{
          title: {
            fontSize: "0.9375rem",
            fontWeight: 600,
            color: "var(--color-slate-900)",
          },
        }}
      >
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            if (question.trim() && !questionTooLong) askMutation.mutate();
          }}
        >
          <p className="text-xs font-normal leading-5 text-slate-500">
            The requester is notified and the request stays pending. Their
            answer appears on this page.
          </p>
          <Textarea
            label={
              <span className="flex w-full items-baseline justify-between gap-3">
                Question
                <LengthCounter
                  length={textLength(question)}
                  max={MAX_MESSAGE_LENGTH}
                />
              </span>
            }
            error={questionTooLong ? "The question is too long" : undefined}
            autosize
            minRows={3}
            maxRows={6}
            value={question}
            onChange={(event) => setQuestion(event.currentTarget.value)}
            styles={{ label: { display: "flex", width: "100%" } }}
          />
          <div className="flex justify-end gap-2">
            <Button variant="default" onClick={() => setAskOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="filled"
              color="ink"
              loading={askMutation.isPending}
              disabled={!question.trim() || questionTooLong}
            >
              Send question
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default ReviewRequest;
