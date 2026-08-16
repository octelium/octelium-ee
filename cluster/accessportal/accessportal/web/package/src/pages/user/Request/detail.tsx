import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Clock, X } from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";

import TimeAgo, { TimeRemaining } from "@/components/TimeAgo";
import {
  Badge,
  Card,
  ConfirmDialog,
  ErrorState,
  KeyValue,
  Loading,
  PageHeader,
  SectionTitle,
} from "../../../ui";
import {
  durationToParts,
  requestResourceLabel,
  requestSubjectName,
  statusMeta,
  urgencyMeta,
} from "../../../utils";
import { getUserClient } from "../../../utils/client";

const RequestDetail = () => {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const [cancelOpen, setCancelOpen] = React.useState(false);

  const qry = useQuery({
    queryKey: ["user", "getRequest", name],
    enabled: !!name,
    queryFn: async () => {
      const { response } = await getUserClient().getRequest(
        MetaP.GetOptions.create({ name: name! }),
      );
      return response;
    },
  });

  const subjectName = qry.data ? requestSubjectName(qry.data) : "";
  const subjectQry = useQuery({
    queryKey: ["user", "getSubjectUser", subjectName],
    enabled: !!subjectName,
    queryFn: async () => {
      const { response } = await getUserClient().getSubjectUser({
        userRef: MetaP.ObjectReference.create({ name: subjectName }),
      });
      return response;
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async () => {
      const { response } = await getUserClient().cancelRequest(
        AccessP.CancelRequestRequest.create({ requestRef: { name: name! } }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Request cancelled");
      setCancelOpen(false);
      queryClient.invalidateQueries({ queryKey: ["user", "getRequest", name] });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to cancel request");
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
  const isPending =
    item.status?.state?.status === AccessP.Request_Status_State_Status.PENDING;
  const review = item.status?.review;

  return (
    <div className="w-full">
      <Link
        to="/user/requests"
        className="inline-flex items-center gap-1.5 text-[0.75rem] font-bold text-slate-400 hover:text-slate-700 transition-colors duration-150 mb-4"
      >
        <ArrowLeft size={13} strokeWidth={2.5} />
        My Requests
      </Link>

      <PageHeader
        eyebrow={resource.kind}
        title={resource.name || item.metadata!.name}
        actions={
          <div className="flex items-center gap-2">
            <Badge tone={urgency.tone}>{urgency.label}</Badge>
            <Badge tone={status.tone}>{status.label}</Badge>
            {isPending && (
              <Button
                variant="outline"
                color="red"
                leftSection={<X size={13} strokeWidth={2.5} />}
                loading={cancelMutation.isPending}
                onClick={() => setCancelOpen(true)}
              >
                Cancel
              </Button>
            )}
          </div>
        }
      />

      <div className="flex flex-col gap-4">
        <Card className="p-5">
          <SectionTitle>Overview</SectionTitle>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <KeyValue label="Resource">
              <span className="font-mono">{resource.name || "—"}</span>
            </KeyValue>
            <KeyValue label="Type">{resource.kind}</KeyValue>
            <KeyValue label="Urgency">{urgency.label}</KeyValue>
            <KeyValue label="Duration">
              {duration.amount} {duration.unit}
            </KeyValue>
            {subjectName && (
              <KeyValue label="Access for">
                <span className="font-mono">{subjectQry.data?.displayName || subjectName}</span>
                {subjectQry.data?.email && (
                  <span className="block text-[0.7rem] font-medium text-slate-400 mt-0.5">
                    {subjectQry.data.email}
                  </span>
                )}
              </KeyValue>
            )}
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
              <KeyValue label="Deadline">
                <TimeAgo rfc3339={item.spec.deadline} />
              </KeyValue>
            )}
          </div>
        </Card>

        {(item.status?.approvalStartAt ||
          item.status?.approvalEndAt ||
          item.status?.accessEndsAt) && (
          <Card className="p-5">
            <SectionTitle>Access window</SectionTitle>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {item.status?.approvalStartAt && (
                <KeyValue label="Approved at">
                  <TimeAgo rfc3339={item.status.approvalStartAt} />
                </KeyValue>
              )}
              {item.status?.approvalEndAt && (
                <KeyValue label="Approval ends">
                  <TimeRemaining rfc3339={item.status.approvalEndAt} />
                </KeyValue>
              )}
              {item.status?.accessEndsAt && (
                <KeyValue label="Access ends">
                  <TimeRemaining rfc3339={item.status.accessEndsAt} />
                </KeyValue>
              )}
            </div>
          </Card>
        )}

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
          </Card>
        )}
      </div>

      <ConfirmDialog
        opened={cancelOpen}
        onClose={() => setCancelOpen(false)}
        onConfirm={() => cancelMutation.mutate()}
        title="Cancel request?"
        description="This request will stop progressing and cannot be restored."
        confirmLabel="Cancel request"
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default RequestDetail;
