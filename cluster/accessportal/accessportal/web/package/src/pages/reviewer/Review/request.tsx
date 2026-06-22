import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Clock, X } from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

import TimeAgo from "@/components/TimeAgo";
import {
  Badge,
  Card,
  Eyebrow,
  Field,
  KeyValue,
  Loading,
  PageHeader,
  SectionTitle,
} from "../../../ui";
import {
  requestResourceLabel,
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

  const qry = useQuery({
    queryKey: ["reviewer", "getRequest", name],
    enabled: !!name,
    queryFn: async () => {
      const { response } = await getReviewerClient().getRequest({
        name: name!,
      } as any);
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
      navigate("/reviewer/requests");
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to submit review");
    },
  });

  if (qry.isLoading) return <Loading label="Loading request..." />;
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
            <div className="grid grid-cols-2 gap-4">
              <KeyValue label="Requester">
                <span className="font-mono">{requester || "—"}</span>
              </KeyValue>
              <KeyValue label="Resource">
                <span className="font-mono">{resource.name || "—"}</span>
              </KeyValue>
              <KeyValue label="Type">{resource.kind}</KeyValue>
              <KeyValue label="Urgency">{urgency.label}</KeyValue>
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
            </div>
          </Card>

          {review && review.lastSteps.length > 0 && (
            <Card className="p-5">
              <SectionTitle>Review progress</SectionTitle>
              <div className="flex flex-col gap-2">
                {review.lastSteps.map((step, idx) => {
                  const isCurrent = step.stepIndex === review.currentStep;
                  return (
                    <div
                      key={idx}
                      className="flex items-center gap-3 px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/60"
                    >
                      <div
                        className={
                          isCurrent
                            ? "flex items-center justify-center w-6 h-6 rounded-full bg-amber-100 text-amber-600"
                            : "flex items-center justify-center w-6 h-6 rounded-full bg-emerald-100 text-emerald-600"
                        }
                      >
                        {isCurrent ? (
                          <Clock size={12} strokeWidth={2.5} />
                        ) : (
                          <Check size={12} strokeWidth={2.5} />
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
                  onClick={() =>
                    decideMutation.mutate(AccessP.Review_Spec_Decision.APPROVE)
                  }
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
                  onClick={() =>
                    decideMutation.mutate(AccessP.Review_Spec_Decision.REJECT)
                  }
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
    </div>
  );
};

export default ReviewRequest;
