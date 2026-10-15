import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Modal, Textarea } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Save, X } from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router-dom";
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
  DocumentTitle,
  ErrorState,
  Field,
  LengthCounter,
  Loading,
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
  isEvaluatingRequest,
  isPendingRequest,
  normalizeDuration,
  requestRefetchInterval,
  requestResourceLabel,
  tsToMillis,
  utf8Length,
} from "@/utils";
import { getUserClient } from "@/utils/client";

const EditRequestModal = (props: {
  opened: boolean;
  onClose: () => void;
  request: AccessP.Request;
}) => {
  const queryClient = useQueryClient();
  const { request } = props;
  const [urgency, setUrgency] = React.useState<AccessP.Request_Spec_Urgency>(
    request.spec?.urgency || AccessP.Request_Spec_Urgency.NORMAL,
  );
  const [duration, setDuration] = React.useState<MetaP.Duration>(
    normalizeDuration(request.spec?.duration),
  );
  const [deadline, setDeadline] = React.useState<Timestamp | undefined>(
    request.spec?.deadline,
  );
  const [justification, setJustification] = React.useState(
    request.spec?.justification ?? "",
  );

  const [prevOpened, setPrevOpened] = React.useState(props.opened);
  if (props.opened !== prevOpened) {
    setPrevOpened(props.opened);
    if (props.opened) {
      setUrgency(request.spec?.urgency || AccessP.Request_Spec_Urgency.NORMAL);
      setDuration(normalizeDuration(request.spec?.duration));
      setDeadline(request.spec?.deadline);
      setJustification(request.spec?.justification ?? "");
    }
  }

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

  const justificationBytes = utf8Length(justification);
  const tooLong = justificationBytes > MAX_JUSTIFICATION_BYTES;
  const deadlinePassed = !!deadline && (tsToMillis(deadline) ?? 0) <= Date.now();

  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      centered
      radius="lg"
      title="Edit pending request"
      closeOnEscape={!mutation.isPending}
      closeOnClickOutside={!mutation.isPending}
      withCloseButton={!mutation.isPending}
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
          if (!tooLong && !deadlinePassed) mutation.mutate();
        }}
      >
        <p className="text-xs font-normal leading-5 text-slate-500">
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

        <Textarea
          label={
            <span className="flex w-full items-baseline justify-between gap-3">
              Justification
              <LengthCounter length={justificationBytes} max={MAX_JUSTIFICATION_BYTES} />
            </span>
          }
          error={
            tooLong
              ? "The justification is too long"
              : deadlinePassed
                ? "The deadline is in the past. Choose a later time or clear it."
                : undefined
          }
          autosize
          minRows={3}
          maxRows={7}
          value={justification}
          onChange={(event) => setJustification(event.currentTarget.value)}
          styles={{ label: { display: "flex", width: "100%" } }}
        />

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="default"
            leftSection={<X size={13} strokeWidth={2.25} />}
            onClick={props.onClose}
            disabled={mutation.isPending}
          >
            Discard
          </Button>
          <Button
            type="submit"
            variant="filled"
            color="ink"
            leftSection={<Save size={13} strokeWidth={2.25} />}
            loading={mutation.isPending}
            disabled={tooLong || deadlinePassed}
          >
            Save changes
          </Button>
        </div>
      </form>
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
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().getRequest(
        MetaP.GetOptions.create({ name: name! }),
        { abort: signal },
      );
      return response;
    },
    refetchInterval: (query) => requestRefetchInterval(query.state.data),
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

  if (qry.isLoading) return <Loading label="Loading request…" />;
  if (!qry.data) {
    return (
      <div className="w-full">
        <DocumentTitle title="Request" />
        <BackLink to="/user/requests">My Requests</BackLink>
        {qry.isError && !isNotFoundError(qry.error) ? (
          <ErrorState
            title="Could not load this request"
            error={qry.error}
            onRetry={() => qry.refetch()}
          />
        ) : (
          <NotFoundState
            title="Request not found"
            description="This request either does not exist or is no longer available to you."
            action={
              <Button component={Link} to="/user/requests" variant="default">
                Back to My Requests
              </Button>
            }
          />
        )}
      </div>
    );
  }

  const item = qry.data;
  const resource = requestResourceLabel(item);
  const pending = isPendingRequest(item);
  const open = pending || isEvaluatingRequest(item);

  return (
    <div className="w-full">
      <DocumentTitle title={resource.name || item.metadata!.name} />
      <BackLink to="/user/requests">My Requests</BackLink>

      <PageHeader
        eyebrow={`${resource.kind} access request`}
        title={resource.name || item.metadata!.name}
        meta={
          <>
            <StatusBadge status={item.status?.state?.status} withHint />
            <UrgencyBadge urgency={item.spec?.urgency} />
          </>
        }
        actions={
          pending ? (
            <>
              <Button
                variant="default"
                leftSection={<Pencil size={13} strokeWidth={2.25} />}
                onClick={() => setEditOpen(true)}
              >
                Edit
              </Button>
              <Button
                variant="default"
                leftSection={<X size={13} strokeWidth={2.25} />}
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
          <StatusHero request={item} canRequestAgain={!open} />
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
        details={
          <SummaryRow label={resource.kind}>
            {resource.name || item.metadata!.name}
          </SummaryRow>
        }
        confirmLabel="Cancel request"
        danger
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default RequestDetail;
