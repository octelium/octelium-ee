import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, SegmentedControl, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  ExternalLink,
  Gavel,
  History,
  Pencil,
  RotateCcw,
  Workflow,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";

import JustificationCard from "@/components/Access/JustificationCard";
import RequestFacts from "@/components/Access/RequestFacts";
import ResourceCard from "@/components/Access/ResourceCard";
import TimeAgo from "@/components/TimeAgo";
import {
  Badge,
  BackLink,
  ConfirmDialog,
  CopyValue,
  DecisionBadge,
  ErrorState,
  Field,
  InfoGrid,
  KeyValue,
  Loading,
  Note,
  NotFoundState,
  PageHeader,
  SectionCard,
  Timeline,
  TimelineItem,
} from "@/ui";
import { decisionMeta, shortName } from "@/utils";
import { getReviewerClient } from "@/utils/client";

const ReviewDetail = () => {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const [resetOpen, setResetOpen] = React.useState(false);
  const [editing, setEditing] = React.useState(false);
  const [decision, setDecision] = React.useState<AccessP.Review_Spec_Decision>(
    AccessP.Review_Spec_Decision.APPROVE,
  );
  const [justification, setJustification] = React.useState("");

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
    retry: false,
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
      queryClient.invalidateQueries({ queryKey: ["reviewer", "getReview", name] });
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listReview"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to update review");
    },
  });

  const resetMutation = useMutation({
    mutationFn: async () => {
      const { response } = await getReviewerClient().cancelReview(
        AccessP.CancelReviewRequest.create({ reviewRef: { name: name! } }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Decision reset");
      setResetOpen(false);
      queryClient.invalidateQueries({ queryKey: ["reviewer", "getReview", name] });
      queryClient.invalidateQueries({ queryKey: ["reviewer", "listReview"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to reset the decision");
    },
  });

  if (qry.isLoading) return <Loading label="Loading review..." />;
  if (qry.isError) {
    return (
      <ErrorState
        title="Could not load this review"
        onRetry={() => qry.refetch()}
      />
    );
  }
  if (!qry.data) {
    return (
      <NotFoundState
        title="Review not found"
        description="This review either does not exist or is no longer available to you."
      />
    );
  }

  const item = qry.data;
  const meta = decisionMeta(item.spec?.decision);
  const request = shortName(item.status?.requestRef?.name);
  const revisions = item.status?.lastRevisions ?? [];
  const stillPending = !!requestQry.data;

  return (
    <div className="w-full">
      <BackLink to="/reviewer/reviews">My Reviews</BackLink>

      <PageHeader
        eyebrow="Review"
        title={request || item.metadata!.name}
        meta={
          <>
            <DecisionBadge decision={item.spec?.decision} />
            {typeof item.status?.stepIndex === "number" && (
              <Badge tone="slate" icon={<Workflow size={10} strokeWidth={2.8} />}>
                Step {item.status.stepIndex + 1}
              </Badge>
            )}
            <CopyValue value={item.metadata!.name} />
          </>
        }
        actions={
          <>
            {!editing && (
              <Button
                variant="default"
                leftSection={<Pencil size={13} strokeWidth={2.6} />}
                onClick={() => setEditing(true)}
              >
                Edit
              </Button>
            )}
            <Button
              variant="light"
              color="red"
              leftSection={<RotateCcw size={13} strokeWidth={2.6} />}
              loading={resetMutation.isPending}
              onClick={() => setResetOpen(true)}
            >
              Reset decision
            </Button>
          </>
        }
      />

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <SectionCard
            title="Your decision"
            description={
              editing
                ? "Changes apply until the review is applied to its request"
                : undefined
            }
            icon={<Gavel size={14} strokeWidth={2.4} />}
            tone={meta.tone}
          >
            {editing ? (
              <div className="flex flex-col gap-4">
                <Field label="Decision">
                  <SegmentedControl
                    fullWidth
                    value={
                      decision === AccessP.Review_Spec_Decision.REJECT
                        ? "reject"
                        : "approve"
                    }
                    onChange={(value) =>
                      setDecision(
                        value === "reject"
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

                <Field
                  label="Justification"
                  description="Required when rejecting the request"
                >
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
                    leftSection={<Check size={14} strokeWidth={2.8} />}
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
                    leftSection={<X size={14} strokeWidth={2.8} />}
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
              <InfoGrid>
                <KeyValue label="Decision">
                  <DecisionBadge decision={item.spec?.decision} />
                </KeyValue>
                <KeyValue label="Set">
                  <TimeAgo rfc3339={item.status?.setAt} />
                </KeyValue>
                <KeyValue label="Reviewed request" mono>
                  <Link
                    to={`/reviewer/requests/${item.status?.requestRef?.name ?? ""}`}
                    className="inline-flex items-center gap-1 truncate text-slate-700 hover:text-slate-900"
                  >
                    {request || "—"}
                    <ExternalLink size={11} strokeWidth={2.6} className="text-slate-400" />
                  </Link>
                </KeyValue>
                {typeof item.status?.stepIndex === "number" && (
                  <KeyValue label="Workflow step">
                    Step {item.status.stepIndex + 1}
                  </KeyValue>
                )}
              </InfoGrid>
            )}
          </SectionCard>

          {!editing && (
            <JustificationCard
              text={item.spec?.justification}
              title="Your justification"
              description="The reason you recorded for this decision"
              emptyLabel="You did not record a reason for this decision."
            />
          )}

          {stillPending && <ResourceCard request={requestQry.data!} />}

          {revisions.length > 0 && (
            <SectionCard
              title="Decision history"
              description={`${revisions.length} earlier version${revisions.length === 1 ? "" : "s"} of this review`}
              icon={<History size={14} strokeWidth={2.4} />}
            >
              <Timeline>
                {revisions.map((revision, index) => {
                  const revisionMeta = decisionMeta(revision.spec?.decision);
                  return (
                    <TimelineItem
                      key={index}
                      tone={revisionMeta.tone}
                      title={revisionMeta.label}
                      meta={<TimeAgo rfc3339={revision.setAt} />}
                      last={index === revisions.length - 1}
                    >
                      {revision.spec?.justification}
                    </TimelineItem>
                  );
                })}
              </Timeline>
            </SectionCard>
          )}
        </div>

        <div className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-20">
          {stillPending ? (
            <SectionCard title="Request under review" bodyClassName="px-4 py-1">
              <RequestFacts request={requestQry.data!} showPolicy />
            </SectionCard>
          ) : (
            <SectionCard
              title="Request under review"
              icon={<Workflow size={14} strokeWidth={2.4} />}
            >
              <Note tone="slate">
                {requestQry.isLoading
                  ? "Loading the reviewed request..."
                  : "The reviewed request already reached a decision, so its details are no longer available to you."}
              </Note>
            </SectionCard>
          )}
        </div>
      </div>

      <ConfirmDialog
        opened={resetOpen}
        onClose={() => setResetOpen(false)}
        onConfirm={() => resetMutation.mutate()}
        title="Reset this decision?"
        description="The review keeps its history but no longer counts as an approval or a rejection. You can set a new decision afterwards."
        confirmLabel="Reset decision"
        loading={resetMutation.isPending}
      />
    </div>
  );
};

export default ReviewDetail;
