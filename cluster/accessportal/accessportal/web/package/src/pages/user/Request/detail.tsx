import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Modal, TextInput, Textarea } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarPlus, CircleStop, Pencil, Save, X } from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router-dom";

import GrantCard from "@/components/Access/GrantCard";
import InformationRequestCard from "@/components/Access/InformationRequestCard";
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
import UnsavedChangesDialog from "@/components/UnsavedChangesDialog";
import {
  BackLink,
  ConfirmDialog,
  DocumentTitle,
  ErrorState,
  Field,
  LengthCounter,
  Loading,
  Note,
  NotFoundState,
  PageHeader,
  SectionCard,
  StatusBadge,
  SummaryRow,
  UrgencyBadge,
} from "@/ui";
import {
  MAX_JUSTIFICATION_LENGTH,
  MAX_MESSAGE_LENGTH,
  MAX_TICKET_LENGTH,
  durationToSeconds,
  extendRequestHref,
  isAccessActive,
  isApprovedRequest,
  isEvaluatingRequest,
  isPendingRequest,
  normalizeDuration,
  requestRefetchInterval,
  requestResourceLabel,
  requestSubjectName,
  requesterName,
  textLength,
  tsToMillis,
} from "@/utils";
import { getUserClient } from "@/utils/client";
import { useAppSelector } from "@/utils/hooks";
import { isNotFoundError } from "@/utils/rpcError";
import { useUnsavedChanges } from "@/utils/hooks/useUnsavedChanges";
import { useNow } from "@/utils/time";

const requestDraft = (request: AccessP.Request) => ({
  urgency: request.spec?.urgency ?? AccessP.Request_Spec_Urgency.NORMAL,
  duration: normalizeDuration(request.spec?.duration),
  deadline: request.spec?.deadline,
  justification: request.spec?.justification ?? "",
  ticket: request.spec?.ticket ?? "",
});

const EditRequestModal = (props: {
  opened: boolean;
  onClose: () => void;
  request: AccessP.Request;
}) => {
  const queryClient = useQueryClient();
  const now = useNow();
  const { request } = props;
  const [initial, setInitial] = React.useState(() => requestDraft(request));
  const [urgency, setUrgency] = React.useState<AccessP.Request_Spec_Urgency>(
    initial.urgency,
  );
  const [duration, setDuration] = React.useState<MetaP.Duration>(
    initial.duration,
  );
  const [deadline, setDeadline] = React.useState<Timestamp | undefined>(
    initial.deadline,
  );
  const [justification, setJustification] = React.useState(
    initial.justification,
  );
  const [ticket, setTicket] = React.useState(initial.ticket);

  const [prevOpened, setPrevOpened] = React.useState(props.opened);
  if (props.opened !== prevOpened) {
    setPrevOpened(props.opened);
    if (props.opened) {
      const draft = requestDraft(request);
      setInitial(draft);
      setUrgency(draft.urgency);
      setDuration(draft.duration);
      setDeadline(draft.deadline);
      setJustification(draft.justification);
      setTicket(draft.ticket);
    }
  }

  const unsavedChanges = useUnsavedChanges(
    props.opened &&
      (urgency !== initial.urgency ||
        durationToSeconds(duration) !== durationToSeconds(initial.duration) ||
        deadline?.seconds !== initial.deadline?.seconds ||
        deadline?.nanos !== initial.deadline?.nanos ||
        justification !== initial.justification ||
        ticket !== initial.ticket),
  );
  const restartsReview =
    justification !== initial.justification ||
    ticket !== initial.ticket ||
    durationToSeconds(duration) > durationToSeconds(initial.duration);
  const close = () => unsavedChanges.confirmDiscard(props.onClose);

  const mutation = useMutation({
    mutationFn: async () => {
      const next = AccessP.Request.clone(request);
      next.spec!.urgency = urgency;
      next.spec!.justification = justification;
      next.spec!.duration = duration;
      next.spec!.deadline = deadline;
      next.spec!.ticket = ticket.trim();
      const { response } = await getUserClient().updateRequest(next);
      return response;
    },
    onSuccess: () => {
      notifications.show({ color: "green", message: "Request updated" });
      props.onClose();
      queryClient.invalidateQueries({
        queryKey: ["user", "getRequest", request.metadata!.name],
      });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to update request",
      });
    },
  });

  const justificationLength = textLength(justification);
  const tooLong = justificationLength > MAX_JUSTIFICATION_LENGTH;
  const ticketTooLong = textLength(ticket) > MAX_TICKET_LENGTH;
  const deadlinePassed = !!deadline && (tsToMillis(deadline) ?? 0) <= now;

  return (
    <>
      <Modal
        opened={props.opened}
        onClose={close}
        centered
        radius="lg"
        title="Edit pending request"
        closeOnEscape={!mutation.isPending && !unsavedChanges.opened}
        closeOnClickOutside={!mutation.isPending && !unsavedChanges.opened}
        trapFocus={!unsavedChanges.opened}
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
            if (!tooLong && !ticketTooLong && !deadlinePassed)
              mutation.mutate();
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
                <LengthCounter length={justificationLength} max={MAX_JUSTIFICATION_LENGTH} />
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

          <TextInput
            label="Ticket"
            description="Optional. A change ticket or incident reference"
            value={ticket}
            error={ticketTooLong ? "The ticket is too long" : undefined}
            onChange={(event) => setTicket(event.currentTarget.value)}
          />

          {restartsReview && (
            <Note tone="amber">
              Changing the justification or the ticket, or asking for a longer
              duration, restarts the review. Approvals already given no longer
              count.
            </Note>
          )}

          <div className="flex items-center justify-end gap-2">
            <Button
              type="button"
              variant="default"
              leftSection={<X size={13} strokeWidth={2.25} />}
              onClick={close}
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
              disabled={tooLong || ticketTooLong || deadlinePassed}
            >
              Save changes
            </Button>
          </div>
        </form>
      </Modal>
      <UnsavedChangesDialog guard={unsavedChanges} />
    </>
  );
};

const RequestDetail = () => {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const now = useNow();
  const status = useAppSelector((state) => state.settings.status);
  const [cancelOpen, setCancelOpen] = React.useState(false);
  const [editOpen, setEditOpen] = React.useState(false);
  const [endOpen, setEndOpen] = React.useState(false);
  const [endReason, setEndReason] = React.useState("");

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
      notifications.show({ color: "green", message: "Request cancelled" });
      setCancelOpen(false);
      queryClient.invalidateQueries({ queryKey: ["user", "getRequest", name] });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to cancel request",
      });
    },
  });

  const endMutation = useMutation({
    mutationFn: async () => {
      const { response } = await getUserClient().endAccess(
        AccessP.EndAccessRequest.create({
          requestRef: { name: name! },
          reason: endReason.trim(),
        }),
      );
      return response;
    },
    onSuccess: () => {
      notifications.show({ color: "green", message: "Access ended" });
      setEndOpen(false);
      setEndReason("");
      queryClient.invalidateQueries({ queryKey: ["user", "getRequest", name] });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
      queryClient.invalidateQueries({ queryKey: ["user", "getInbox"] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to end the access",
      });
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
  const isCreator =
    !status?.user?.metadata?.name ||
    requesterName(item) === status.user.metadata.name;
  const isSubject = requestSubjectName(item) === status?.user?.metadata?.name;
  const approved = isApprovedRequest(item);
  const active = isAccessActive(item, now);
  const supersededBy = item.status?.supersededByRef?.name;
  const title = resource.displayName || resource.name || item.metadata!.name;

  return (
    <div className="w-full">
      <DocumentTitle title={title} />
      <BackLink to="/user/requests">My Requests</BackLink>

      <PageHeader
        eyebrow={
          resource.accessLevel
            ? `${resource.accessLevel} access request`
            : `${resource.kind} access request`
        }
        title={title}
        meta={
          <>
            <StatusBadge status={item.status?.state?.status} withHint />
            <UrgencyBadge urgency={item.spec?.urgency} />
          </>
        }
        actions={
          pending ? (
            <>
              {isCreator && (
                <Button
                  variant="default"
                  leftSection={<Pencil size={13} strokeWidth={2.25} />}
                  onClick={() => setEditOpen(true)}
                >
                  Edit
                </Button>
              )}
              <Button
                variant="default"
                leftSection={<X size={13} strokeWidth={2.25} />}
                onClick={() => setCancelOpen(true)}
              >
                Cancel request
              </Button>
            </>
          ) : approved ? (
            <>
              {active && (isCreator || isSubject) && (
                <Button
                  component={Link}
                  to={extendRequestHref(item, status?.user?.metadata?.name)}
                  variant="default"
                  leftSection={<CalendarPlus size={13} strokeWidth={2.25} />}
                >
                  Extend
                </Button>
              )}
              <Button
                variant="default"
                leftSection={<CircleStop size={13} strokeWidth={2.25} />}
                onClick={() => setEndOpen(true)}
              >
                End access
              </Button>
            </>
          ) : undefined
        }
      />

      {supersededBy && (
        <Note tone="blue" className="mb-4">
          This access was extended by{" "}
          <Link
            to={`/user/requests/${supersededBy}`}
            className="font-medium underline underline-offset-2"
          >
            a newer request
          </Link>
          .
        </Note>
      )}

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="flex min-w-0 flex-col gap-4">
          <StatusHero request={item} canRequestAgain={!open} />
          <InformationRequestCard request={item} canRespond />
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
          <GrantCard request={item} />
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

      <ConfirmDialog
        opened={endOpen}
        onClose={() => setEndOpen(false)}
        onConfirm={() => endMutation.mutate()}
        title="End this access now?"
        description="The access is removed immediately. Raise a new request if you need it again."
        details={
          <div className="flex flex-col gap-3">
            <SummaryRow label={resource.kind}>{title}</SummaryRow>
            <Textarea
              label="Reason"
              description="Optional"
              autosize
              minRows={2}
              maxRows={4}
              value={endReason}
              error={
                textLength(endReason) > MAX_MESSAGE_LENGTH
                  ? "The reason is too long"
                  : undefined
              }
              onChange={(event) => setEndReason(event.currentTarget.value)}
            />
          </div>
        }
        confirmLabel="End access"
        danger
        loading={endMutation.isPending}
      />
    </div>
  );
};

export default RequestDetail;
