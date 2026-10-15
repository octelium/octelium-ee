import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, SegmentedControl, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Gavel, History, Lock, Pencil, RotateCcw, X } from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";

import JustificationCard from "@/components/Access/JustificationCard";
import RequestFacts from "@/components/Access/RequestFacts";
import ResourceCard from "@/components/Access/ResourceCard";
import TimeAgo from "@/components/TimeAgo";
import {
  BackLink,
  ConfirmDialog,
  DecisionBadge,
  DocumentTitle,
  ErrorState,
  Field,
  InfoGrid,
  KeyValue,
  LengthCounter,
  Loading,
  Note,
  NotFoundState,
  PageHeader,
  SectionCard,
  Timeline,
  TimelineItem,
  isNotFoundError,
} from "@/ui";
import {
  MAX_JUSTIFICATION_BYTES,
  currentReviewStepIndex,
  decisionMeta,
  shortName,
  utf8Length,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";
import { describeRpcError } from "@/utils/rpcError";

const ReviewDetail = () => {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const [resetOpen, setResetOpen] = React.useState(false);
  const [editing, setEditing] = React.useState(false);
  const [decision, setDecision] = React.useState<AccessP.Review_Spec_Decision>(
    AccessP.Review_Spec_Decision.UNSET,
  );
  const [justification, setJustification] = React.useState("");
  const [justificationError, setJustificationError] = React.useState<string>();

  const qry = useQuery({
    queryKey: ["reviewer", "getReview", name],
    enabled: !!name,
    queryFn: async ({ signal }) => {
      const { response } = await getReviewerClient().getReview(
        MetaP.GetOptions.create({ name: name! }),
        { abort: signal },
      );
      return response;
    },
  });

  const requestName = qry.data?.status?.requestRef?.name ?? "";
  const requestQry = useQuery({
    queryKey: ["reviewer", "getRequest", requestName],
    enabled: !!requestName,
    queryFn: async ({ signal }) => {
      const { response } = await getReviewerClient().getRequest(
        MetaP.GetOptions.create({ name: requestName }),
        { abort: signal },
      );
      return response;
    },
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["reviewer"] });
  };

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
      invalidate();
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to update review");
      invalidate();
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
      invalidate();
    },
    onError: (e) => {
      setResetOpen(false);
      toast.error(e instanceof Error ? e.message : "Failed to reset the decision");
      invalidate();
    },
  });

  if (qry.isLoading) return <Loading label="Loading review…" />;
  if (!qry.data) {
    return (
      <div className="w-full">
        <DocumentTitle title="Review" />
        <BackLink to="/reviewer/reviews">My Reviews</BackLink>
        {qry.isError && !isNotFoundError(qry.error) ? (
          <ErrorState
            title="Could not load this review"
            error={qry.error}
            onRetry={() => qry.refetch()}
          />
        ) : (
          <NotFoundState
            title="Review not found"
            description="This review either does not exist or is no longer available to you."
            action={
              <Button component={Link} to="/reviewer/reviews" variant="default">
                Back to My Reviews
              </Button>
            }
          />
        )}
      </div>
    );
  }

  const item = qry.data;
  const request = shortName(item.status?.requestRef?.name);
  const revisions = item.status?.lastRevisions ?? [];
  const reviewable = requestQry.data;
  const editable =
    !!reviewable &&
    currentReviewStepIndex(reviewable) === (item.status?.stepIndex ?? 0);
  const hasDecision = item.spec?.decision !== AccessP.Review_Spec_Decision.UNSET;
  const stepLabel = item.status?.stepName || `Step ${(item.status?.stepIndex ?? 0) + 1}`;
  const bytes = utf8Length(justification);

  const startEditing = () => {
    setDecision(item.spec?.decision ?? AccessP.Review_Spec_Decision.UNSET);
    setJustification(item.spec?.justification ?? "");
    setJustificationError(undefined);
    setEditing(true);
  };

  const save = () => {
    if (bytes > MAX_JUSTIFICATION_BYTES) {
      setJustificationError("The reason is too long");
      return;
    }
    if (decision === AccessP.Review_Spec_Decision.REJECT && !justification.trim()) {
      setJustificationError("Add a reason before saving a rejection");
      return;
    }
    updateMutation.mutate();
  };

  const readOnlyReason = requestQry.isLoading
    ? undefined
    : reviewable
      ? editable
        ? undefined
        : "This decision was applied to an earlier step of the request and can no longer be changed."
      : requestQry.isError && !isNotFoundError(requestQry.error)
        ? `The reviewed request could not be loaded: ${describeRpcError(requestQry.error).message}`
        : "The reviewed request is no longer waiting for your review, so this decision can no longer be changed.";

  return (
    <div className="w-full">
      <DocumentTitle title={`Review of ${request || item.metadata!.name}`} />
      <BackLink to="/reviewer/reviews">My Reviews</BackLink>

      <PageHeader
        eyebrow="Review"
        title={
          <>
            Request <span className="font-mono">{request || item.metadata!.name}</span>
          </>
        }
        meta={<DecisionBadge decision={item.spec?.decision} />}
        actions={
          editable && !editing ? (
            <>
              <Button
                variant="default"
                leftSection={<Pencil size={13} strokeWidth={2.25} />}
                onClick={startEditing}
              >
                {hasDecision ? "Change decision" : "Set decision"}
              </Button>
              {hasDecision && (
                <Button
                  variant="default"
                  leftSection={<RotateCcw size={13} strokeWidth={2.25} />}
                  onClick={() => setResetOpen(true)}
                >
                  Reset decision
                </Button>
              )}
            </>
          ) : undefined
        }
      />

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <SectionCard
            title="Your decision"
            description={editing ? "The change applies until the review is applied to its request" : undefined}
            icon={<Gavel size={14} strokeWidth={2.25} />}
          >
            {editing ? (
              <form
                className="flex flex-col gap-4"
                onSubmit={(event) => {
                  event.preventDefault();
                  save();
                }}
              >
                <Field label="Decision">
                  <SegmentedControl
                    fullWidth
                    aria-label="Decision"
                    value={
                      decision === AccessP.Review_Spec_Decision.REJECT
                        ? "reject"
                        : decision === AccessP.Review_Spec_Decision.APPROVE
                          ? "approve"
                          : ""
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

                <Textarea
                  label={
                    <span className="flex w-full items-baseline justify-between gap-3">
                      Reason
                      <LengthCounter length={bytes} max={MAX_JUSTIFICATION_BYTES} />
                    </span>
                  }
                  description="Required when rejecting the request"
                  error={justificationError}
                  autosize
                  minRows={3}
                  maxRows={7}
                  value={justification}
                  onChange={(e) => {
                    setJustification(e.currentTarget.value);
                    if (justificationError) setJustificationError(undefined);
                  }}
                  styles={{ label: { display: "flex", width: "100%" } }}
                />

                {decision === AccessP.Review_Spec_Decision.UNSET && (
                  <Note>Choose Approve or Reject to save a decision.</Note>
                )}

                <div className="flex items-center gap-2">
                  <Button
                    type="submit"
                    variant="filled"
                    color="ink"
                    leftSection={<Check size={14} strokeWidth={2.5} />}
                    loading={updateMutation.isPending}
                    disabled={decision === AccessP.Review_Spec_Decision.UNSET}
                  >
                    Save
                  </Button>
                  <Button
                    type="button"
                    variant="default"
                    leftSection={<X size={14} strokeWidth={2.5} />}
                    disabled={updateMutation.isPending}
                    onClick={() => setEditing(false)}
                  >
                    Discard
                  </Button>
                </div>
              </form>
            ) : (
              <div className="flex flex-col gap-4">
                <InfoGrid>
                  <KeyValue label="Decision">
                    <DecisionBadge decision={item.spec?.decision} />
                  </KeyValue>
                  <KeyValue label="Set">
                    <TimeAgo rfc3339={item.status?.setAt} />
                  </KeyValue>
                  <KeyValue label="Reviewed request" mono>
                    {reviewable ? (
                      <Link
                        to={`/reviewer/requests/${item.status?.requestRef?.name ?? ""}`}
                        className="truncate text-slate-800 underline decoration-slate-300 underline-offset-2 hover:decoration-slate-600"
                      >
                        {request || "—"}
                      </Link>
                    ) : (
                      request || "—"
                    )}
                  </KeyValue>
                  <KeyValue label="Workflow step">{stepLabel}</KeyValue>
                </InfoGrid>
                {readOnlyReason && (
                  <Note icon={<Lock size={12} strokeWidth={2.25} />}>
                    {readOnlyReason}
                  </Note>
                )}
              </div>
            )}
          </SectionCard>

          {!editing && (
            <JustificationCard
              text={item.spec?.justification}
              title="Your reason"
              description="The reason you recorded for this decision"
              emptyLabel="You did not record a reason for this decision."
            />
          )}

          {reviewable && <ResourceCard request={reviewable} reviewer />}

          {revisions.length > 0 && (
            <SectionCard
              title="Decision history"
              description={`${revisions.length} earlier version${revisions.length === 1 ? "" : "s"} of this review`}
              icon={<History size={14} strokeWidth={2.25} />}
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
          <SectionCard title="Request under review" bodyClassName={reviewable ? "px-4 py-1" : undefined}>
            {reviewable ? (
              <RequestFacts request={reviewable} showPolicy />
            ) : requestQry.isLoading ? (
              <p className="text-xs font-normal text-slate-500">
                Loading the reviewed request…
              </p>
            ) : requestQry.isError && !isNotFoundError(requestQry.error) ? (
              <ErrorState
                title="Could not load the reviewed request"
                error={requestQry.error}
                onRetry={() => requestQry.refetch()}
              />
            ) : (
              <Note>
                The reviewed request is no longer waiting for your review, so its
                details are not available to you.
              </Note>
            )}
          </SectionCard>
        </div>
      </div>

      <ConfirmDialog
        opened={resetOpen}
        onClose={() => setResetOpen(false)}
        onConfirm={() => resetMutation.mutate()}
        title="Reset this decision?"
        description="The review keeps its history but no longer counts as an approval or a rejection. You can set a new decision afterwards while the request is still waiting for this step."
        confirmLabel="Reset decision"
        danger
        loading={resetMutation.isPending}
      />
    </div>
  );
};

export default ReviewDetail;
