import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Clock, X } from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

import TimeAgo from "@/components/TimeAgo";
import RequestContext from "@/components/Access/RequestContext";
import {
  Badge,
  Card,
  ConfirmDialog,
  ErrorState,
  Eyebrow,
  Field,
  KeyValue,
  Loading,
  PageHeader,
  SectionTitle,
} from "../../../ui";
import {
  requestResourceLabel,
  durationToParts,
  shortName,
  statusMeta,
  urgencyMeta,
} from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

const ReviewRequest = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [justification, setJustification] = React.useState("");
  const [decisionToSubmit, setDecisionToSubmit] = React.useState<AccessP.Review_Spec_Decision>();

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
    return <ErrorState title="Could not load this request" onRetry={() => qry.refetch()} />;
  }
  if (!qry.data) {
    return (
      <Card className="p-8 text-center text-[0.82rem] font-semibold text-slate-500">
        Request not found.
      </Card>
    );
  }

  const item = qry.data;
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const urgency = urgencyMeta(item.spec?.urgency);
  const duration = durationToParts(item.spec?.duration);
  const requester = shortName(item.status?.userRef?.name);
  const isPending =
    item.status?.state?.status === AccessP.Request_Status_State_Status.PENDING;
  const review = item.status?.review;

  return (
    <div className="w-full">
      <Link
        to="/reviewer/requests"
        className="inline-flex items-center gap-1.5 text-[0.75rem] font-bold text-slate-400 hover:text-slate-700 transition-colors duration-150 mb-4"
      >
        <ArrowLeft size={13} strokeWidth={2.5} />
        Review Queue
      </Link>

      <PageHeader
        eyebrow={resource.kind}
        title={resource.name || item.metadata!.name}
        actions={
          <div className="flex items-center gap-2">
            <Badge tone={urgency.tone}>{urgency.label}</Badge>
            <Badge tone={status.tone}>{status.label}</Badge>
          </div>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-5 items-start">
        <div className="flex flex-col gap-4">
          <Card className="p-5">
            <SectionTitle>Request</SectionTitle>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <KeyValue label="Requester">
                <span className="font-mono">{requester || "—"}</span>
              </KeyValue>
              <KeyValue label="Resource">
                <span className="font-mono">{resource.name || "—"}</span>
              </KeyValue>
              <KeyValue label="Type">{resource.kind}</KeyValue>
              <KeyValue label="Urgency">{urgency.label}</KeyValue>
              <KeyValue label="Duration">{duration.amount} {duration.unit}</KeyValue>
              <KeyValue label="Requested">
                {item.status?.state?.createdAt ? (
                  <TimeAgo rfc3339={item.status.state.createdAt} />
                ) : (
                  "—"
                )}
              </KeyValue>
              {item.spec?.justification && (
                <KeyValue label="Justification" full>
                  <span className="font-normal text-slate-600">
                    {item.spec.justification}
                  </span>
                </KeyValue>
              )}
              {item.spec?.deadline && (
                <KeyValue label="Deadline"><TimeAgo rfc3339={item.spec.deadline} /></KeyValue>
              )}
              {item.status?.policyRef?.name && (
                <KeyValue label="Policy"><span className="font-mono">{item.status.policyRef.name}</span></KeyValue>
              )}
              {item.status?.rule?.name && (
                <KeyValue label="Matched rule"><span className="font-mono">{item.status.rule.name}</span></KeyValue>
              )}
            </div>
          </Card>

          <RequestContext request={item} />

          {item.status?.lastStates && item.status.lastStates.length > 1 && (
            <Card className="p-5">
              <SectionTitle>Status history</SectionTitle>
              <div className="flex flex-col gap-2">
                {item.status.lastStates.map((state, index) => {
                  const stateMeta = statusMeta(state.status);
                  return (
                    <div key={`${state.status}-${index}`} className="flex items-center gap-3 px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/60">
                      <span className={`w-2 h-2 rounded-full ${stateMeta.tone === "emerald" ? "bg-emerald-500" : stateMeta.tone === "red" ? "bg-red-500" : stateMeta.tone === "amber" ? "bg-amber-500" : "bg-slate-400"}`} />
                      <span className="text-[0.78rem] font-bold text-slate-700">{stateMeta.label}</span>
                      {state.createdAt && <span className="ml-auto text-[0.7rem] font-semibold text-slate-400"><TimeAgo rfc3339={state.createdAt} /></span>}
                    </div>
                  );
                })}
              </div>
            </Card>
          )}

          {review && review.lastSteps.length > 0 && (
            <Card className="p-5">
              <SectionTitle>Review progress</SectionTitle>
              <div className="flex flex-col gap-2">
                {review.lastSteps.map((step, idx) => {
                const isCurrent = step.stepIndex === review.currentStep;
                const isComplete = step.stepIndex < review.currentStep;
                  return (
                    <div
                      key={idx}
                      className="flex items-center gap-3 px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/60"
                    >
                      <div
                        className={
                        isCurrent
                          ? "flex items-center justify-center w-6 h-6 rounded-full bg-amber-100 text-amber-600"
                          : isComplete
                            ? "flex items-center justify-center w-6 h-6 rounded-full bg-emerald-100 text-emerald-600"
                            : "flex items-center justify-center w-6 h-6 rounded-full bg-slate-100 text-slate-400"
                        }
                      >
                      {isCurrent ? (
                        <Clock size={12} strokeWidth={2.5} />
                      ) : isComplete ? (
                        <Check size={12} strokeWidth={2.5} />
                      ) : (
                        <span className="text-[0.65rem] font-bold">{step.stepIndex + 1}</span>
                        )}
                      </div>
                      <span className="text-[0.78rem] font-bold text-slate-700">
                        Step {step.stepIndex + 1}
                      </span>
                      {step.setAt && (
                        <span className="text-[0.7rem] font-semibold text-slate-400 ml-auto">
                          <TimeAgo rfc3339={step.setAt} />
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
              {review.currentStepStartedAt && (
                <p className="text-[0.7rem] font-medium text-slate-400 mt-3">
                  Current step started <TimeAgo rfc3339={review.currentStepStartedAt} />
                </p>
              )}
            </Card>
          )}
        </div>

        <Card className="p-5 lg:sticky lg:top-20">
          <Eyebrow>Your decision</Eyebrow>
          {isPending ? (
            <div className="flex flex-col gap-4 mt-3">
              <Field
                label="Justification"
                description="Optionally explain your decision"
              >
                <Textarea
                  autosize
                  minRows={3}
                  maxRows={7}
                  placeholder="Reason for approving or rejecting..."
                  value={justification}
                  onChange={(e) => setJustification(e.currentTarget.value)}
                />
              </Field>

              <div className="flex flex-col gap-2">
                <Button
                  fullWidth
                  variant="filled"
                  color="dark"
                  leftSection={<Check size={14} strokeWidth={2.5} />}
                  loading={
                    decideMutation.isPending &&
                    decideMutation.variables ===
                      AccessP.Review_Spec_Decision.APPROVE
                  }
                  disabled={decideMutation.isPending}
                  onClick={() => setDecisionToSubmit(AccessP.Review_Spec_Decision.APPROVE)}
                >
                  Approve
                </Button>
                <Button
                  fullWidth
                  variant="outline"
                  color="red"
                  leftSection={<X size={14} strokeWidth={2.5} />}
                  loading={
                    decideMutation.isPending &&
                    decideMutation.variables ===
                      AccessP.Review_Spec_Decision.REJECT
                  }
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
            </div>
          ) : (
            <p className="mt-3 text-[0.76rem] font-semibold text-slate-400">
              This request is no longer awaiting review.
            </p>
          )}
        </Card>
      </div>

      <ConfirmDialog
        opened={decisionToSubmit !== undefined}
        onClose={() => setDecisionToSubmit(undefined)}
        onConfirm={() => {
          if (decisionToSubmit !== undefined) decideMutation.mutate(decisionToSubmit);
        }}
        title={decisionToSubmit === AccessP.Review_Spec_Decision.REJECT ? "Reject request?" : "Approve request?"}
        description={
          decisionToSubmit === AccessP.Review_Spec_Decision.REJECT
            ? "This decision will reject the access request and notify the requester."
            : "This decision will approve this review step and may grant access if all required steps are complete."
        }
        confirmLabel={decisionToSubmit === AccessP.Review_Spec_Decision.REJECT ? "Reject request" : "Approve request"}
        danger={decisionToSubmit === AccessP.Review_Spec_Decision.REJECT}
        loading={decideMutation.isPending}
      />
    </div>
  );
};

export default ReviewRequest;
