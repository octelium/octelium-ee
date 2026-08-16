import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, SegmentedControl, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Trash2, X } from "lucide-react";
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
  Field,
  KeyValue,
  Loading,
  PageHeader,
  SectionTitle,
} from "../../../ui";
import { decisionMeta, shortName } from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

const ReviewDetail = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [cancelOpen, setCancelOpen] = React.useState(false);

  const [decision, setDecision] = React.useState<AccessP.Review_Spec_Decision>(
    AccessP.Review_Spec_Decision.APPROVE,
  );
  const [justification, setJustification] = React.useState("");
  const [editing, setEditing] = React.useState(false);

  const qry = useQuery({
    queryKey: ["reviewer", "getReview", name],
    enabled: !!name,
    queryFn: async () => {
      const { response } = await getReviewerClient().getReview(
        MetaP.GetOptions.create({ name: name! }),
      );
      return response;
    },
  });

  const requestName = qry.data?.status?.requestRef?.name ?? "";
  const requestQry = useQuery({
    queryKey: ["reviewer", "getRequestForReview", requestName],
    enabled: !!requestName,
    queryFn: async () => {
      const { response } = await getReviewerClient().getRequest(
        MetaP.GetOptions.create({ name: requestName }),
      );
      return response;
    },
  });

  React.useEffect(() => {
    if (qry.data?.spec) {
      setDecision(qry.data.spec.decision);
      setJustification(qry.data.spec.justification);
    }
  }, [qry.data]);

  const updateMutation = useMutation({
    mutationFn: async () => {
      const next = AccessP.Review.clone(qry.data!);
      next.spec!.decision = decision;
      next.spec!.justification = justification;
      const { response } = await getReviewerClient().updateReview(next);
      return response;
    },
    onSuccess: () => {
      toast.success("Review updated");
      setEditing(false);
      queryClient.invalidateQueries({
        queryKey: ["reviewer", "getReview", name],
      });
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listReview"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to update review");
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async () => {
      const { response } = await getReviewerClient().cancelReview(
        AccessP.CancelReviewRequest.create({ reviewRef: { name: name! } }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Review cancelled");
      setCancelOpen(false);
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listReview"] });
      navigate("/reviewer/reviews");
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to cancel review");
    },
  });

  if (qry.isLoading) return <Loading label="Loading review..." />;
  if (qry.isError) {
    return <ErrorState title="Could not load this review" onRetry={() => qry.refetch()} />;
  }
  if (!qry.data) {
    return (
      <Card className="p-8 text-center text-[0.82rem] font-semibold text-slate-500">
        Review not found.
      </Card>
    );
  }

  const item = qry.data;
  const meta = decisionMeta(item.spec?.decision);
  const request = shortName(item.status?.requestRef?.name);

  return (
    <div className="w-full">
      <Link
        to="/reviewer/reviews"
        className="inline-flex items-center gap-1.5 text-[0.75rem] font-bold text-slate-400 hover:text-slate-700 transition-colors duration-150 mb-4"
      >
        <ArrowLeft size={13} strokeWidth={2.5} />
        My Reviews
      </Link>

      <PageHeader
        eyebrow="Review"
        title={request || item.metadata!.name}
        actions={
          <div className="flex items-center gap-2">
            <Badge tone={meta.tone}>{meta.label}</Badge>
            <Button
              variant="outline"
              color="red"
              leftSection={<Trash2 size={13} strokeWidth={2.5} />}
              loading={cancelMutation.isPending}
              onClick={() => setCancelOpen(true)}
            >
              Cancel
            </Button>
          </div>
        }
      />

      <div className="flex flex-col gap-4">
        {requestQry.data && <RequestContext request={requestQry.data} heading="Request under review" />}
        <Card className="p-5">
          <div className="flex items-center justify-between mb-3">
            <SectionTitle>Decision</SectionTitle>
            {!editing && (
              <Button
                size="compact-xs"
                variant="default"
                onClick={() => setEditing(true)}
              >
                Edit
              </Button>
            )}
          </div>

          {editing ? (
            <div className="flex flex-col gap-4">
              <Field label="Decision">
                <SegmentedControl
                  value={
                    AccessP.Review_Spec_Decision[decision] === "DECISION_REJECT"
                      ? "reject"
                      : "approve"
                  }
                  onChange={(v) =>
                    setDecision(
                      v === "reject"
                        ? AccessP.Review_Spec_Decision.REJECT
                        : AccessP.Review_Spec_Decision.APPROVE,
                    )
                  }
                  data={[
                    { label: "Approve", value: "approve" },
                    { label: "Reject", value: "reject" },
                  ]}
                />
              </Field>

              <Field label="Justification">
                <Textarea
                  autosize
                  minRows={3}
                  maxRows={7}
                  value={justification}
                  onChange={(e) => setJustification(e.currentTarget.value)}
                />
              </Field>

              <div className="flex items-center gap-2">
                <Button
                  variant="filled"
                  color="dark"
                  leftSection={<Check size={14} strokeWidth={2.5} />}
                  loading={updateMutation.isPending}
                  onClick={() => {
                    if (
                      decision === AccessP.Review_Spec_Decision.REJECT &&
                      !justification.trim()
                    ) {
                      toast.error("Add a reason before saving a rejection");
                      return;
                    }
                    updateMutation.mutate();
                  }}
                >
                  Save
                </Button>
                <Button
                  variant="default"
                  leftSection={<X size={14} strokeWidth={2.5} />}
                  disabled={updateMutation.isPending}
                  onClick={() => {
                    setEditing(false);
                    setDecision(item.spec!.decision);
                    setJustification(item.spec!.justification);
                  }}
                >
                  Discard
                </Button>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-4">
              <KeyValue label="Decision">
                <Badge tone={meta.tone}>{meta.label}</Badge>
              </KeyValue>
              <KeyValue label="Request">
                <span className="font-mono">{request || "—"}</span>
              </KeyValue>
              {typeof item.status?.stepIndex === "number" && (
                <KeyValue label="Step">{item.status.stepIndex + 1}</KeyValue>
              )}
              {item.status?.setAt && (
                <KeyValue label="Set at">
                  <TimeAgo rfc3339={item.status.setAt} />
                </KeyValue>
              )}
              {item.spec?.justification && (
                <KeyValue label="Justification" full>
                  <span className="font-normal text-slate-600">
                    {item.spec.justification}
                  </span>
                </KeyValue>
              )}
            </div>
          )}

          {!editing && item.status?.lastRevisions && item.status.lastRevisions.length > 0 && (
            <div className="border-t border-slate-100 mt-5 pt-5">
              <SectionTitle>Decision history</SectionTitle>
              <div className="flex flex-col gap-2">
                {item.status.lastRevisions.map((revision, index) => {
                  const revisionMeta = decisionMeta(revision.spec?.decision);
                  return (
                    <div key={index} className="flex items-center gap-3 px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/60">
                      <Badge tone={revisionMeta.tone}>{revisionMeta.label}</Badge>
                      {revision.setAt && <span className="ml-auto text-[0.7rem] font-semibold text-slate-400"><TimeAgo rfc3339={revision.setAt} /></span>}
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </Card>
      </div>

      <ConfirmDialog
        opened={cancelOpen}
        onClose={() => setCancelOpen(false)}
        onConfirm={() => cancelMutation.mutate()}
        title="Cancel review?"
        description="This review will be removed and the request may no longer reflect your decision."
        confirmLabel="Cancel review"
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default ReviewDetail;
