import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Gavel, Info, X } from "lucide-react";
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

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
  CopyValue,
  ErrorState,
  Field,
  Loading,
  Note,
  NotFoundState,
  PageHeader,
  SectionCard,
  StatusBadge,
  UrgencyBadge,
} from "@/ui";
import {
  currentReviewStepIndex,
  isPendingRequest,
  approvalRequirementLabel,
  requestResourceLabel,
  reviewSteps,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";

const ReviewRequest = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [justification, setJustification] = React.useState("");
  const [decisionToSubmit, setDecisionToSubmit] =
    React.useState<AccessP.Review_Spec_Decision>();

  const qry = useQuery({
    queryKey: ["reviewer", "getRequest", name],
    enabled: !!name,
    queryFn: async () => {
      const { response } = await getReviewerClient().getRequest(
        MetaP.GetOptions.create({ name: name! }),
      );
      return response;
    },
  });

  const decideMutation = useMutation({
    mutationFn: async (decision: AccessP.Review_Spec_Decision) => {
      const review = AccessP.Review.create({
        apiVersion: "access/v1",
        kind: "Review",
        metadata: {},
        spec: { decision, justification },
        status: {
          requestRef: MetaP.ObjectReference.create({ name: name! }),
        },
      });
      const { response } = await getReviewerClient().createReview(review);
      return response;
    },
    onSuccess: () => {
      toast.success("Review submitted");
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listRequest"] });
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listReview"] });
      queryClient.invalidateQueries({
        queryKey: ["reviewer", "getRequest", name],
      });
      navigate("/reviewer/requests");
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to submit review");
    },
  });

  if (qry.isLoading) return <Loading label="Loading request..." />;
  if (qry.isError) {
    return (
      <ErrorState
        title="Could not load this request"
        description="It may have been decided already or it is no longer assigned to you."
        onRetry={() => qry.refetch()}
      />
    );
  }
  if (!qry.data) {
    return (
      <NotFoundState
        title="Request not found"
        description="This request either does not exist or you can no longer review it."
      />
    );
  }

  const item = qry.data;
  const resource = requestResourceLabel(item);
  const pending = isPendingRequest(item);
  const steps = reviewSteps(item);
  const currentStep = steps[currentReviewStepIndex(item)];

  return (
    <div className="w-full">
      <BackLink to="/reviewer/requests">Review Queue</BackLink>

      <PageHeader
        eyebrow={`${resource.kind} access request`}
        title={resource.name || item.metadata!.name}
        meta={
          <>
            <StatusBadge status={item.status?.state?.status} withHint />
            <UrgencyBadge urgency={item.spec?.urgency} />
            <CopyValue value={item.metadata!.name} />
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
          />
          <PeopleCard request={item} />
          <ResourceCard request={item} />
          <AuthorizationCard request={item} />
          <ReviewWorkflow request={item} />
          <RequestTimeline request={item} />
        </div>

        <div className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-20">
          <SectionCard
            title="Your decision"
            description={
              pending
                ? approvalRequirementLabel(currentStep)
                : "This request is no longer awaiting review"
            }
            icon={<Gavel size={14} strokeWidth={2.4} />}
            tone={pending ? "amber" : "slate"}
          >
            {pending ? (
              <div className="flex flex-col gap-4">
                <Field
                  label="Justification"
                  description="Required when rejecting, recommended when approving"
                >
                  <Textarea
                    autosize
                    minRows={3}
                    maxRows={7}
                    placeholder="Explain your decision..."
                    value={justification}
                    onChange={(e) => setJustification(e.currentTarget.value)}
                  />
                </Field>

                <div className="flex flex-col gap-2">
                  <Button
                    fullWidth
                    variant="filled"
                    color="teal"
                    leftSection={<Check size={15} strokeWidth={3} />}
                    disabled={decideMutation.isPending}
                    onClick={() =>
                      setDecisionToSubmit(AccessP.Review_Spec_Decision.APPROVE)
                    }
                  >
                    Approve
                  </Button>
                  <Button
                    fullWidth
                    variant="light"
                    color="red"
                    leftSection={<X size={15} strokeWidth={3} />}
                    disabled={decideMutation.isPending}
                    onClick={() => {
                      if (!justification.trim()) {
                        toast.error("Add a reason before rejecting this request");
                        return;
                      }
                      setDecisionToSubmit(AccessP.Review_Spec_Decision.REJECT);
                    }}
                  >
                    Reject
                  </Button>
                </div>

                <Note tone="slate" icon={<Info size={13} strokeWidth={2.4} />}>
                  A single rejection ends the request immediately. An approval
                  advances it to the next step of the workflow.
                </Note>
              </div>
            ) : (
              <Note tone="slate">
                A decision has already been reached for this request.
              </Note>
            )}
          </SectionCard>

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
        title={
          decisionToSubmit === AccessP.Review_Spec_Decision.REJECT
            ? "Reject this request?"
            : "Approve this request?"
        }
        description={
          decisionToSubmit === AccessP.Review_Spec_Decision.REJECT
            ? "The request is rejected immediately and no access is granted."
            : "Your approval is recorded for the current step. Access is granted once every step of the workflow is satisfied."
        }
        details={
          <div className="flex flex-col gap-1.5 text-[0.74rem] font-semibold text-slate-600">
            <div className="flex items-center justify-between gap-3">
              <span className="text-slate-400">Resource</span>
              <span className="truncate font-mono">
                {resource.name || item.metadata!.name}
              </span>
            </div>
            {justification.trim() && (
              <div className="flex items-start justify-between gap-3">
                <span className="shrink-0 text-slate-400">Your reason</span>
                <span className="min-w-0 text-right">{justification.trim()}</span>
              </div>
            )}
          </div>
        }
        confirmLabel={
          decisionToSubmit === AccessP.Review_Spec_Decision.REJECT
            ? "Reject request"
            : "Approve request"
        }
        danger={decisionToSubmit === AccessP.Review_Spec_Decision.REJECT}
        loading={decideMutation.isPending}
      />
    </div>
  );
};

export default ReviewRequest;
