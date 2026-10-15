import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Gavel, Info, X } from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "@/lib/utils";

import AuthorizationCard from "@/components/Access/AuthorizationCard";
import JustificationCard from "@/components/Access/JustificationCard";
import PeopleCard from "@/components/Access/PeopleCard";
import RequestFacts from "@/components/Access/RequestFacts";
import RequestTimeline from "@/components/Access/RequestTimeline";
import ResourceCard from "@/components/Access/ResourceCard";
import ReviewWorkflow from "@/components/Access/ReviewWorkflow";
import StatusHero from "@/components/Access/StatusHero";
import {
  BackLink,
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
  isNotFoundError,
} from "@/ui";
import {
  MAX_JUSTIFICATION_BYTES,
  approvalRequirementLabel,
  currentReviewStepIndex,
  formatDuration,
  grantedDuration,
  isPendingRequest,
  requestResourceLabel,
  requestSubjectName,
  reviewStepLabel,
  reviewStepName,
  reviewSteps,
  utf8Length,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";

const DecisionPanel = (props: {
  className?: string;
  pending: boolean;
  requirement: string;
  justification: string;
  onJustificationChange: (value: string) => void;
  error?: string;
  busy: boolean;
  onApprove: () => void;
  onReject: () => void;
}) => {
  const bytes = utf8Length(props.justification);

  return (
    <SectionCard
      title="Your decision"
      description={
        props.pending ? props.requirement : "This request is no longer awaiting review"
      }
      icon={<Gavel size={14} strokeWidth={2.25} />}
      className={props.className}
    >
      {props.pending ? (
        <div className="flex flex-col gap-4">
          <Textarea
            label={
              <span className="flex w-full items-baseline justify-between gap-3">
                Reason
                <LengthCounter length={bytes} max={MAX_JUSTIFICATION_BYTES} />
              </span>
            }
            description="Required when rejecting, recommended when approving"
            error={props.error}
            autosize
            minRows={3}
            maxRows={7}
            placeholder="Explain your decision…"
            value={props.justification}
            onChange={(e) => props.onJustificationChange(e.currentTarget.value)}
            styles={{ label: { display: "flex", width: "100%" } }}
          />

          <div className="grid grid-cols-2 gap-2">
            <Button
              variant="default"
              leftSection={<X size={14} strokeWidth={2.5} />}
              disabled={props.busy}
              onClick={props.onReject}
            >
              Reject
            </Button>
            <Button
              variant="filled"
              color="ink"
              leftSection={<Check size={14} strokeWidth={2.5} />}
              disabled={props.busy}
              onClick={props.onApprove}
            >
              Approve
            </Button>
          </div>

          <Note icon={<Info size={13} strokeWidth={2.25} />}>
            A single rejection ends the request immediately. An approval counts
            towards the current step of the workflow.
          </Note>
        </div>
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

  const qry = useQuery({
    queryKey: ["reviewer", "getRequest", name],
    enabled: !!name,
    queryFn: async ({ signal }) => {
      const { response } = await getReviewerClient().getRequest(
        MetaP.GetOptions.create({ name: name! }),
        { abort: signal },
      );
      return response;
    },
    refetchInterval: (query) =>
      isPendingRequest(query.state.data) ? 15000 : false,
  });

  const item = qry.data;
  const steps = reviewSteps(item);
  const currentIndex = currentReviewStepIndex(item);
  const currentStep = steps[currentIndex];

  const decideMutation = useMutation({
    mutationFn: async (decision: AccessP.Review_Spec_Decision) => {
      const { response } = await getReviewerClient().setReviewDecision(
        AccessP.SetReviewDecisionRequest.create({
          requestRef: MetaP.ObjectReference.create({ name: name! }),
          decision,
          justification,
          expectedStepName: reviewStepName(currentStep, currentIndex),
        }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Review submitted");
      setDecisionToSubmit(undefined);
      queryClient.invalidateQueries({ queryKey: ["reviewer"] });
      navigate("/reviewer/requests");
    },
    onError: (e) => {
      setDecisionToSubmit(undefined);
      toast.error(e instanceof Error ? e.message : "Failed to submit review");
      void qry.refetch();
    },
  });

  if (qry.isLoading) return <Loading label="Loading request…" />;
  if (!item) {
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
  const tooLong = utf8Length(justification) > MAX_JUSTIFICATION_BYTES;

  const requestDecision = (decision: AccessP.Review_Spec_Decision) => {
    if (tooLong) {
      setJustificationError("The reason is too long");
      return;
    }
    if (decision === AccessP.Review_Spec_Decision.REJECT && !justification.trim()) {
      setJustificationError("Add a reason before rejecting this request");
      return;
    }
    setJustificationError(undefined);
    setDecisionToSubmit(decision);
  };

  const panelProps = {
    pending,
    requirement: [stepLabel, approvalRequirementLabel(currentStep)]
      .filter(Boolean)
      .join(" · "),
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
      <DocumentTitle title={`Review ${resource.name || item.metadata!.name}`} />
      <BackLink to="/reviewer/requests">Review Queue</BackLink>

      <PageHeader
        eyebrow={`${resource.kind} access request`}
        title={resource.name || item.metadata!.name}
        meta={
          <>
            <StatusBadge status={item.status?.state?.status} withHint />
            <UrgencyBadge urgency={item.spec?.urgency} />
          </>
        }
      />

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <StatusHero request={item} />
          <JustificationCard
            text={item.spec?.justification}
            description="Why the requester says the access is needed"
            emptyLabel="The requester did not explain why this access is needed. Consider asking before approving."
            emptyTone="amber"
          />
          <PeopleCard request={item} />
          <ResourceCard request={item} reviewer />
          <DecisionPanel {...panelProps} className="lg:hidden" />
          <AuthorizationCard request={item} />
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
              {formatDuration(grantedDuration(item))}
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
    </div>
  );
};

export default ReviewRequest;
