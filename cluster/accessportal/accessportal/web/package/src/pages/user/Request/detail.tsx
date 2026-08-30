import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Modal, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Save, X } from "lucide-react";
import * as React from "react";
import { useParams } from "react-router-dom";
import { toast } from "sonner";

import AuthorizationCard from "@/components/Access/AuthorizationCard";
import JustificationCard from "@/components/Access/JustificationCard";
import PeopleCard from "@/components/Access/PeopleCard";
import RequestFacts from "@/components/Access/RequestFacts";
import RequestTimeline from "@/components/Access/RequestTimeline";
import ResourceCard from "@/components/Access/ResourceCard";
import ReviewWorkflow from "@/components/Access/ReviewWorkflow";
import StatusHero from "@/components/Access/StatusHero";
import UrgencyPicker from "@/components/Access/UrgencyPicker";
import DurationInput from "@/components/DurationInput";
import TimestampPicker from "@/components/TimestampPicker";
import {
  BackLink,
  ConfirmDialog,
  CopyValue,
  ErrorState,
  Field,
  Loading,
  NotFoundState,
  PageHeader,
  SectionCard,
  StatusBadge,
  UrgencyBadge,
} from "@/ui";
import { isPendingRequest, requestResourceLabel } from "@/utils";
import { getUserClient } from "@/utils/client";

const MAX_JUSTIFICATION = 1500;

const EditRequestModal = (props: {
  opened: boolean;
  onClose: () => void;
  request: AccessP.Request;
}) => {
  const queryClient = useQueryClient();
  const { request } = props;
  const [urgency, setUrgency] = React.useState(
    request.spec?.urgency ?? AccessP.Request_Spec_Urgency.NORMAL,
  );
  const [duration, setDuration] = React.useState<MetaP.Duration | undefined>(
    request.spec?.duration,
  );
  const [deadline, setDeadline] = React.useState<Timestamp | undefined>(
    request.spec?.deadline,
  );
  const [justification, setJustification] = React.useState(
    request.spec?.justification ?? "",
  );

  React.useEffect(() => {
    if (!props.opened) return;
    setUrgency(request.spec?.urgency ?? AccessP.Request_Spec_Urgency.NORMAL);
    setDuration(request.spec?.duration);
    setDeadline(request.spec?.deadline);
    setJustification(request.spec?.justification ?? "");
  }, [props.opened]);

  const mutation = useMutation({
    mutationFn: async () => {
      const next = AccessP.Request.clone(request);
      next.spec!.urgency = urgency;
      next.spec!.justification = justification;
      next.spec!.duration = duration;
      next.spec!.deadline = deadline;
      const { response } = await getUserClient().updateRequest(next);
      return response;
    },
    onSuccess: () => {
      toast.success("Request updated");
      props.onClose();
      queryClient.invalidateQueries({
        queryKey: ["user", "getRequest", request.metadata!.name],
      });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to update request");
    },
  });

  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      centered
      radius="md"
      title="Edit pending request"
      styles={{ title: { fontWeight: 700, fontSize: "0.9rem" } }}
    >
      <div className="flex flex-col gap-4">
        <p className="text-[0.76rem] font-medium leading-relaxed text-slate-500">
          The resource and the recipient of a request cannot be changed. Create a
          new request if you need different access.
        </p>

        <Field label="Urgency" description="Higher urgency helps reviewers triage the queue">
          <UrgencyPicker value={urgency} onChange={setUrgency} />
        </Field>

        <Field label="Duration" description="How long the access should last once approved">
          <DurationInput value={duration} onChange={setDuration} />
        </Field>

        <TimestampPicker
          label="Deadline"
          description="Expire the request if it is not decided by this time"
          placeholder="No deadline"
          value={deadline}
          isFuture
          onChange={setDeadline}
        />

        <Field
          label="Justification"
          hint={
            <span
              className={
                justification.length > MAX_JUSTIFICATION
                  ? "text-[0.66rem] font-bold text-red-500"
                  : "text-[0.66rem] font-semibold text-slate-400"
              }
            >
              {justification.length}/{MAX_JUSTIFICATION}
            </span>
          }
        >
          <Textarea
            autosize
            minRows={3}
            maxRows={7}
            value={justification}
            onChange={(event) => setJustification(event.currentTarget.value)}
          />
        </Field>

        <div className="flex items-center justify-end gap-2">
          <Button
            variant="default"
            leftSection={<X size={13} strokeWidth={2.6} />}
            onClick={props.onClose}
            disabled={mutation.isPending}
          >
            Discard
          </Button>
          <Button
            variant="filled"
            color="dark"
            leftSection={<Save size={13} strokeWidth={2.6} />}
            loading={mutation.isPending}
            disabled={justification.length > MAX_JUSTIFICATION}
            onClick={() => mutation.mutate()}
          >
            Save changes
          </Button>
        </div>
      </div>
    </Modal>
  );
};

const RequestDetail = () => {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const [cancelOpen, setCancelOpen] = React.useState(false);
  const [editOpen, setEditOpen] = React.useState(false);

  const qry = useQuery({
    queryKey: ["user", "getRequest", name],
    enabled: !!name,
    queryFn: async () => {
      const { response } = await getUserClient().getRequest(
        MetaP.GetOptions.create({ name: name! }),
      );
      return response;
    },
    refetchInterval: (query) =>
      isPendingRequest(query.state.data) ? 15000 : false,
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
    return (
      <ErrorState
        title="Could not load this request"
        onRetry={() => qry.refetch()}
      />
    );
  }
  if (!qry.data) {
    return (
      <NotFoundState
        title="Request not found"
        description="This request either does not exist or is no longer available to you."
      />
    );
  }

  const item = qry.data;
  const resource = requestResourceLabel(item);
  const pending = isPendingRequest(item);

  return (
    <div className="w-full">
      <BackLink to="/user/requests">My Requests</BackLink>

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
        actions={
          pending ? (
            <>
              <Button
                variant="default"
                leftSection={<Pencil size={13} strokeWidth={2.6} />}
                onClick={() => setEditOpen(true)}
              >
                Edit
              </Button>
              <Button
                variant="light"
                color="red"
                leftSection={<X size={13} strokeWidth={2.8} />}
                loading={cancelMutation.isPending}
                onClick={() => setCancelOpen(true)}
              >
                Cancel request
              </Button>
            </>
          ) : undefined
        }
      />

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="flex min-w-0 flex-col gap-4">
          <StatusHero request={item} />
          <ResourceCard request={item} />
          <JustificationCard
            text={item.spec?.justification}
            description="Why you asked for this access"
            emptyLabel="You did not add a justification. Reviewers decide faster when they know why the access is needed."
          />
          <ReviewWorkflow request={item} />
          <RequestTimeline request={item} />
        </div>

        <div className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-20">
          <SectionCard title="Request details" bodyClassName="px-4 py-1">
            <RequestFacts request={item} />
          </SectionCard>
          <PeopleCard request={item} />
          <AuthorizationCard request={item} />
        </div>
      </div>

      <EditRequestModal
        opened={editOpen}
        onClose={() => setEditOpen(false)}
        request={item}
      />

      <ConfirmDialog
        opened={cancelOpen}
        onClose={() => setCancelOpen(false)}
        onConfirm={() => cancelMutation.mutate()}
        title="Cancel this request?"
        description="The request stops progressing through its review workflow and cannot be restored."
        confirmLabel="Cancel request"
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default RequestDetail;
