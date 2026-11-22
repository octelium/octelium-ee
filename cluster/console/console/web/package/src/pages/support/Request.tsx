import {
  AddSupportRequestMessageRequest,
  EscalateSupportRequestRequest,
  ListSupportRequestEventOptions,
  ResolveSupportRequestRequest,
  SupportInfo_State,
  SupportRequest,
  SupportRequest_Actor_Type,
  SupportRequest_Attachment_Purpose,
  SupportRequest_Event,
  SupportRequest_Event_Type,
  SupportRequest_Status,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import Meta from "@/components/Meta";
import { PageActions, PageTitleAddon } from "@/components/PageHeader/context";
import RefreshButton from "@/components/RefreshButton";
import RpcError from "@/components/RpcError";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { onError } from "@/utils";
import { getClientSupport } from "@/utils/client";
import { rpcErrorCode } from "@/utils/rpcError";
import {
  Alert,
  Button,
  Collapse,
  Modal,
  Skeleton,
  Textarea,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  ArrowRightLeft,
  ChevronDown,
  CircleCheck,
  CircleDot,
  ExternalLink,
  FileSearch,
  Flag,
  LifeBuoy,
  Paperclip,
  Send,
  Server,
  Tag,
  UserCheck,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useParams } from "react-router";
import {
  AttachmentChip,
  DiagnosticsButton,
  EnvironmentList,
  FileButton,
  PendingFiles,
  PriorityBadge,
  StatusBadge,
  ToneBadge,
} from "./components";
import Diagnostics from "./Diagnostics";
import {
  PendingFile,
  supportKeys,
  toPendingFile,
  uploadFiles,
  useSupportInfo,
  useSupportRequest,
} from "./queries";
import {
  ACTOR_LABEL,
  checkFile,
  getPriority,
  initials,
  isOpenStatus,
  LEVEL_LABEL,
  newIdempotencyKey,
  STATUS_WORDS,
} from "./utils";

const formatTime = (value?: Timestamp) =>
  value ? dayjs(Timestamp.toDate(value)).format("MMM D, YYYY HH:mm") : "—";

const mergeEvents = (...pages: SupportRequest_Event[][]) => {
  const events = new Map<string, SupportRequest_Event>();
  for (const page of pages) {
    for (const event of page)
      if (!events.has(event.id)) events.set(event.id, event);
  }
  return [...events.values()].sort((a, b) => a.sequence - b.sequence);
};

const actorName = (event: SupportRequest_Event) => {
  if (event.actor?.type === SupportRequest_Actor_Type.AGENT) {
    return event.actor.name || "Octelium Support";
  }
  return event.actor?.name || ACTOR_LABEL[event.actor?.type ?? 0] || "Someone";
};

const Avatar = (props: { event: SupportRequest_Event }) => {
  const agent = props.event.actor?.type === SupportRequest_Actor_Type.AGENT;
  return (
    <span
      aria-hidden="true"
      className={twMerge(
        "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border text-micro font-bold",
        agent
          ? "border-blue-200 bg-blue-50 text-blue-700"
          : "border-slate-200 bg-white text-slate-600",
      )}
    >
      {agent ? (
        <LifeBuoy size={14} strokeWidth={2.3} />
      ) : (
        initials(actorName(props.event)) || "?"
      )}
    </span>
  );
};

const describe = (event: SupportRequest_Event) => {
  switch (event.type) {
    case SupportRequest_Event_Type.STATUS:
      return `set the request to ${STATUS_WORDS[event.to] ?? event.to.toLowerCase()}`;
    case SupportRequest_Event_Type.PRIORITY:
      return `changed the priority from ${event.from} to ${event.to}`;
    case SupportRequest_Event_Type.CATEGORY:
      return `changed the category to ${event.to.toLowerCase()}`;
    case SupportRequest_Event_Type.ASSIGNMENT:
      return event.to
        ? `assigned the request to ${event.to}`
        : "unassigned the request";
    case SupportRequest_Event_Type.ESCALATION:
      return event.to === "CLEARED"
        ? "cleared the escalation"
        : "escalated the request";
    case SupportRequest_Event_Type.ATTACHMENT:
      return `attached ${event.body || "a file"}`;
    case SupportRequest_Event_Type.ORGANIZATION:
      return "linked the request to the organization";
    default:
      return "updated the request";
  }
};

const eventIcon = (event: SupportRequest_Event) => {
  switch (event.type) {
    case SupportRequest_Event_Type.STATUS:
      return event.to === "RESOLVED" ? CircleCheck : CircleDot;
    case SupportRequest_Event_Type.PRIORITY:
      return ArrowRightLeft;
    case SupportRequest_Event_Type.ASSIGNMENT:
      return UserCheck;
    case SupportRequest_Event_Type.ESCALATION:
      return Flag;
    case SupportRequest_Event_Type.ATTACHMENT:
      return Paperclip;
    default:
      return Tag;
  }
};

const isMessage = (event: SupportRequest_Event) =>
  event.type === SupportRequest_Event_Type.CREATED ||
  event.type === SupportRequest_Event_Type.MESSAGE;

const TimelineItem = (props: {
  event: SupportRequest_Event;
  request: SupportRequest;
  last: boolean;
}) => {
  const { event, request } = props;
  const agent = event.actor?.type === SupportRequest_Actor_Type.AGENT;
  const attachments = request.attachments.filter((item) =>
    event.attachmentIDs.includes(item.id),
  );

  return (
    <li className="relative flex gap-3 pb-5 last:pb-0">
      {!props.last && (
        <span
          aria-hidden="true"
          className="absolute bottom-0 left-[15px] top-8 w-px bg-slate-200"
        />
      )}

      {isMessage(event) ? (
        <>
          <Avatar event={event} />
          <article
            className={twMerge(
              "min-w-0 flex-1 overflow-hidden rounded-xl border",
              agent
                ? "border-blue-200 bg-blue-50/40"
                : "border-slate-200 bg-white",
            )}
          >
            <header className="flex flex-wrap items-center gap-x-2 gap-y-0.5 border-b border-inherit px-3.5 py-2">
              <span className="text-xs font-semibold text-slate-900">
                {actorName(event)}
              </span>
              <span className="text-micro font-normal text-slate-500">
                {agent
                  ? "Octelium Support"
                  : event.type === SupportRequest_Event_Type.CREATED
                    ? "opened the request"
                    : `via ${ACTOR_LABEL[event.actor?.type ?? 0]?.toLowerCase() ?? "unknown"}`}
              </span>
              {event.createdAt && (
                <span
                  className="ml-auto text-micro font-normal text-slate-500"
                  title={formatTime(event.createdAt)}
                >
                  <TimeAgo rfc3339={event.createdAt} />
                </span>
              )}
            </header>
            <div className="px-3.5 py-3">
              <p className="whitespace-pre-wrap break-words text-body font-normal leading-6 text-slate-800">
                {event.body}
              </p>
              {attachments.length > 0 && (
                <div className="mt-3 grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                  {attachments.map((attachment) => (
                    <AttachmentChip
                      key={attachment.id}
                      request={request}
                      attachment={attachment}
                    />
                  ))}
                </div>
              )}
              {event.skippedAttachments.length > 0 && (
                <p className="mt-2 text-micro font-normal text-amber-700">
                  Not accepted: {event.skippedAttachments.join(", ")}
                </p>
              )}
            </div>
          </article>
        </>
      ) : (
        <>
          <span
            aria-hidden="true"
            className="relative z-10 flex h-8 w-8 shrink-0 items-center justify-center"
          >
            <span className="flex h-6 w-6 items-center justify-center rounded-full border border-slate-200 bg-slate-50 text-slate-500">
              {React.createElement(eventIcon(event), {
                size: 11,
                strokeWidth: 2.5,
              })}
            </span>
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-1 pt-1.5">
            <p className="text-xs font-normal leading-5 text-slate-600">
              <span className="font-semibold text-slate-800">
                {actorName(event)}
              </span>{" "}
              {describe(event)}
              {event.createdAt && (
                <span
                  className="text-micro text-slate-400"
                  title={formatTime(event.createdAt)}
                >
                  {" · "}
                  <TimeAgo rfc3339={event.createdAt} />
                </span>
              )}
            </p>
            {event.reason && (
              <blockquote className="whitespace-pre-wrap break-words rounded-lg border-l-2 border-slate-300 bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-700">
                {event.reason}
              </blockquote>
            )}
          </div>
        </>
      )}
    </li>
  );
};

const Detail = (props: { label: string; children: React.ReactNode }) => (
  <div className="flex min-w-0 items-start justify-between gap-3 py-2">
    <dt className="shrink-0 text-xs font-normal text-slate-500">
      {props.label}
    </dt>
    <dd className="min-w-0 text-right text-xs font-semibold text-slate-800">
      {props.children}
    </dd>
  </div>
);

const SidePanel = (props: {
  title: string;
  icon: React.ElementType<{ size?: number; strokeWidth?: number }>;
  children: React.ReactNode;
  className?: string;
}) => {
  const Icon = props.icon;
  return (
    <section
      className={twMerge(
        "overflow-hidden rounded-xl border border-slate-200 bg-white",
        props.className,
      )}
    >
      <header className="flex items-center gap-2 border-b border-slate-100 px-4 py-3">
        <Icon size={13} strokeWidth={2.3} className="text-slate-400" />
        <h2 className="text-xs font-semibold text-slate-800">{props.title}</h2>
      </header>
      <div className="px-4 py-2">{props.children}</div>
    </section>
  );
};

const Notice = (props: {
  tone: "warning" | "success" | "info";
  title: string;
  children?: React.ReactNode;
}) => (
  <div
    role="status"
    className={twMerge(
      "rounded-xl border px-4 py-3",
      props.tone === "warning"
        ? "border-amber-200 bg-amber-50/60"
        : props.tone === "success"
          ? "border-emerald-200 bg-emerald-50/60"
          : "border-blue-200 bg-blue-50/50",
    )}
  >
    <div className="text-xs font-semibold text-slate-900">{props.title}</div>
    {props.children && (
      <p className="mt-0.5 text-xs font-normal leading-5 text-slate-600">
        {props.children}
      </p>
    )}
  </div>
);

const Escalate = (props: {
  request: SupportRequest;
  opened: boolean;
  onClose: () => void;
}) => {
  const queryClient = useQueryClient();
  const [reason, setReason] = React.useState("");

  const close = () => {
    setReason("");
    props.onClose();
  };

  const mutation = useMutation({
    mutationFn: async () => {
      const { response } = await getClientSupport().escalateSupportRequest(
        EscalateSupportRequestRequest.create({
          id: props.request.id,
          reason: reason.trim(),
        }),
      );
      return response;
    },
    onSuccess: (response) => {
      queryClient.setQueryData(supportKeys.request(props.request.id), response);
      queryClient.invalidateQueries({ queryKey: supportKeys.requests });
      notifications.show({
        color: "green",
        message: "The support management was notified",
      });
      close();
    },
    onError,
  });

  return (
    <Modal
      opened={props.opened}
      onClose={() => !mutation.isPending && close()}
      centered
      size="md"
      withCloseButton={false}
      padding={0}
    >
      <form
        noValidate
        className="flex flex-col"
        onSubmit={(event) => {
          event.preventDefault();
          if (reason.trim().length >= 10 && !mutation.isPending)
            mutation.mutate();
        }}
      >
        <div className="flex items-start gap-3 px-5 py-4">
          <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-amber-200 bg-amber-50 text-amber-600">
            <Flag size={16} strokeWidth={2.3} />
          </span>
          <div className="min-w-0">
            <Modal.Title className="text-sm font-bold text-slate-900">
              Escalate request #{props.request.number}
            </Modal.Title>
            <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
              The Octelium support management is notified and reviews the
              request. A request can only be escalated once.
            </p>
          </div>
        </div>
        <div className="px-5 pb-4">
          <Textarea
            label="Why does this request need more urgency?"
            autosize
            minRows={3}
            maxRows={8}
            maxLength={2000}
            value={reason}
            disabled={mutation.isPending}
            onChange={(event) => setReason(event.currentTarget.value)}
            description="At least 10 characters"
          />
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
          <Button
            variant="default"
            size="sm"
            leftSection={<X size={13} strokeWidth={2.5} />}
            disabled={mutation.isPending}
            onClick={close}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="filled"
            color="ink"
            size="sm"
            leftSection={<Flag size={13} strokeWidth={2.5} />}
            disabled={reason.trim().length < 10}
            loading={mutation.isPending}
          >
            Escalate
          </Button>
        </div>
      </form>
    </Modal>
  );
};

const Composer = (props: {
  request: SupportRequest;
  onDiagnostics: () => void;
  files: PendingFile[];
  setFiles: (files: PendingFile[]) => void;
  disabled: boolean;
}) => {
  const queryClient = useQueryClient();
  const info = useSupportInfo();
  const [body, setBody] = React.useState("");
  const [progress, setProgress] = React.useState<
    Record<string, "uploading" | "done" | "failed">
  >({});
  const [idempotencyKey, setIdempotencyKey] = React.useState(newIdempotencyKey);
  const uploaded = React.useRef(new Map<string, string>());
  const limits = info.data?.limits;
  const maxAttachments = limits?.maxAttachmentsPerMessage ?? 10;
  const draftKey = `${body.trim()}|${props.files.map((item) => item.key).join(",")}`;
  const lastDraft = React.useRef(draftKey);

  React.useEffect(() => {
    if (lastDraft.current === draftKey) return;
    lastDraft.current = draftKey;
    setIdempotencyKey(newIdempotencyKey());
  }, [draftKey]);

  const mutation = useMutation({
    mutationFn: async () => {
      setProgress(
        Object.fromEntries(
          props.files
            .filter((item) => !uploaded.current.has(item.key))
            .map((item) => [item.key, "uploading" as const]),
        ),
      );
      const result = await uploadFiles(
        props.request.id,
        props.files,
        uploaded.current,
      );
      if (result.failed.length > 0) {
        setProgress(
          Object.fromEntries(
            result.failed.map(({ item }) => [item.key, "failed" as const]),
          ),
        );
        throw result.failed[0].error;
      }
      setProgress({});
      const { response } = await getClientSupport().addSupportRequestMessage(
        AddSupportRequestMessageRequest.create({
          id: props.request.id,
          body: body.trim(),
          attachmentIDs: result.ids,
          idempotencyKey,
        }),
      );
      return response;
    },
    onSuccess: (response) => {
      uploaded.current.clear();
      setBody("");
      props.setFiles([]);
      queryClient.setQueryData(supportKeys.request(props.request.id), response);
      queryClient.invalidateQueries({ queryKey: supportKeys.requests });
    },
    onError,
  });

  const addFiles = (items: PendingFile[]) => {
    const next = [...props.files];
    const errors: string[] = [];
    for (const item of items) {
      const error = checkFile(limits, item.file);
      if (error) {
        errors.push(error);
        continue;
      }
      if (next.some((existing) => existing.key === item.key)) continue;
      if (next.length >= maxAttachments) {
        errors.push(`A reply can include at most ${maxAttachments} files`);
        break;
      }
      next.push(item);
    }
    props.setFiles(next);
    for (const error of new Set(errors)) {
      notifications.show({ color: "red", message: error });
    }
  };

  const pending = mutation.isPending;
  const canSend = !!body.trim() && !pending && !props.disabled;
  const shortcut =
    typeof navigator !== "undefined" &&
    /Mac|iPhone|iPad/.test(navigator.userAgent)
      ? "⌘"
      : "Ctrl";

  return (
    <form
      noValidate
      aria-label="Reply"
      className="overflow-hidden rounded-xl border border-slate-200 bg-white"
      onSubmit={(event) => {
        event.preventDefault();
        if (canSend) mutation.mutate();
      }}
    >
      <Textarea
        aria-label="Message"
        placeholder={
          props.request.status === SupportRequest_Status.RESOLVED
            ? "Replying reopens the request"
            : "Add information or answer the support engineer"
        }
        autosize
        minRows={4}
        maxRows={16}
        maxLength={limits?.maxBodyLength ?? 16000}
        value={body}
        disabled={pending || props.disabled}
        variant="unstyled"
        onChange={(event) => setBody(event.currentTarget.value)}
        onKeyDown={(event) => {
          if (
            event.key === "Enter" &&
            (event.metaKey || event.ctrlKey) &&
            canSend
          ) {
            event.preventDefault();
            mutation.mutate();
          }
        }}
        styles={{
          input: {
            paddingInline: "14px",
            paddingBlock: "12px",
            lineHeight: 1.6,
          },
        }}
      />
      {props.files.length > 0 && (
        <div className="px-3 pb-3">
          <PendingFiles
            files={props.files}
            progress={progress}
            disabled={pending}
            onRemove={(key) =>
              props.setFiles(props.files.filter((item) => item.key !== key))
            }
          />
        </div>
      )}
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 bg-slate-50/60 px-3 py-2.5">
        <div className="flex flex-wrap items-center gap-2">
          <DiagnosticsButton
            disabled={pending || props.disabled}
            onClick={props.onDiagnostics}
          />
          <FileButton
            disabled={pending || props.disabled}
            accept={limits?.attachmentExtensions}
            onFiles={(items) =>
              addFiles(items.map((file) => toPendingFile(file)))
            }
          />
        </div>
        <div className="ml-auto flex items-center gap-2">
          <span className="hidden text-micro font-normal text-slate-400 sm:inline">
            {shortcut} + Enter
          </span>
          <Button
            type="submit"
            variant="filled"
            color="ink"
            size="xs"
            leftSection={<Send size={13} strokeWidth={2.5} />}
            disabled={!canSend}
            loading={pending}
          >
            Send
          </Button>
        </div>
      </div>
    </form>
  );
};

const RequestView = (props: { id: string }) => {
  const queryClient = useQueryClient();
  const info = useSupportInfo();
  const query = useSupportRequest(props.id);
  const [escalateOpened, escalate] = useDisclosure(false);
  const [resolveOpened, resolveModal] = useDisclosure(false);
  const [diagnosticsOpened, setDiagnosticsOpened] = React.useState(false);
  const [environmentOpened, setEnvironmentOpened] = React.useState(false);
  const [files, setFiles] = React.useState<PendingFile[]>([]);
  const [earlier, setEarlier] = React.useState<{
    events: SupportRequest_Event[];
    cursor: string;
  } | null>(null);
  const [seen, setSeen] = React.useState<{
    latest?: SupportRequest_Event[];
    events: SupportRequest_Event[];
  }>({ events: [] });
  if (query.data && seen.latest !== query.data.events) {
    setSeen({
      latest: query.data.events,
      events: mergeEvents(seen.events, query.data.events),
    });
  }

  const resolve = useMutation({
    mutationFn: async () => {
      const { response } = await getClientSupport().resolveSupportRequest(
        ResolveSupportRequestRequest.create({ id: props.id }),
      );
      return response;
    },
    onSuccess: (response) => {
      queryClient.setQueryData(supportKeys.request(props.id), response);
      queryClient.invalidateQueries({ queryKey: supportKeys.requests });
      resolveModal.close();
      notifications.show({
        color: "green",
        message: "The request was marked as resolved",
      });
    },
    onError,
  });

  const loadEarlier = useMutation({
    mutationFn: async (cursor: string) => {
      const { response } = await getClientSupport().listSupportRequestEvent(
        ListSupportRequestEventOptions.create({ id: props.id, cursor }),
      );
      return response;
    },
    onSuccess: (response) =>
      setEarlier((current) => ({
        events: [...response.items, ...(current?.events ?? [])],
        cursor: response.nextCursor,
      })),
    onError,
  });

  if (query.isLoading) {
    return (
      <div className="flex flex-col gap-4 py-4">
        <Skeleton height={120} radius="lg" />
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,320px)]">
          <Skeleton height={420} radius="lg" />
          <Skeleton height={300} radius="lg" />
        </div>
      </div>
    );
  }

  if (query.isError || !query.data) {
    const notFound = rpcErrorCode(query.error) === "NOT_FOUND";
    return (
      <div className="flex flex-col gap-4 py-4">
        <Meta title="Support request" />
        {notFound ? (
          <div className="flex min-h-64 flex-col items-center justify-center gap-2.5 rounded-xl border border-dashed border-slate-200 bg-white px-6 py-10 text-center">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
              <FileSearch size={17} strokeWidth={2.2} />
            </span>
            <span className="text-body font-semibold text-slate-800">
              This request is not available from this Cluster
            </span>
            <span className="max-w-md text-micro font-normal text-slate-500">
              Only the requests opened from this Cluster can be followed here.
              Requests that were reclassified or opened elsewhere are available
              in the customer portal.
            </span>
            <Button
              component={Link}
              to="/support"
              variant="default"
              size="xs"
              leftSection={<ArrowLeft size={13} strokeWidth={2.5} />}
            >
              Back to Support
            </Button>
          </div>
        ) : (
          <Alert color="red" title="Could not load the request">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <RpcError error={query.error} />
              <Button
                size="compact-xs"
                variant="outline"
                onClick={() => query.refetch()}
              >
                Try again
              </Button>
            </div>
          </Alert>
        )}
      </div>
    );
  }

  const request = query.data;
  const events = mergeEvents(
    earlier?.events ?? [],
    seen.events,
    request.events,
  );
  const olderCursor = earlier ? earlier.cursor : request.eventsCursor;
  const priority = getPriority(request.priority);
  const requestedPriority = getPriority(request.requestedPriority);
  const available = info.data?.state === SupportInfo_State.AVAILABLE;
  const sla = request.sla;

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex w-full flex-col gap-4 py-4"
    >
      <Meta title={`#${request.number} ${request.subject}`} />

      <PageTitleAddon>
        <StatusBadge status={request.status} />
      </PageTitleAddon>

      <PageActions>
        <RefreshButton
          label="Refresh the request"
          loading={query.isFetching}
          onClick={() => query.refetch()}
        />
        {request.portalURL && (
          <Button
            component="a"
            href={request.portalURL}
            target="_blank"
            rel="noopener noreferrer"
            variant="default"
            size="xs"
            leftSection={<ExternalLink size={13} strokeWidth={2.5} />}
          >
            <span className="hidden sm:inline">Customer portal</span>
            <span className="sm:hidden">Portal</span>
          </Button>
        )}
      </PageActions>

      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        <div className="flex flex-col gap-3 px-4 py-4 sm:px-5 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-mono text-micro font-semibold text-slate-600">
                #{request.number}
              </span>
              <StatusBadge status={request.status} />
              <PriorityBadge priority={request.priority} label />
              {request.isEscalated && (
                <ToneBadge tone="warning">
                  <Flag size={9} strokeWidth={2.5} />
                  Escalated
                </ToneBadge>
              )}
            </div>
            <h1 className="mt-2 break-words text-lg font-bold leading-snug text-slate-900">
              {request.subject}
            </h1>
            <p className="mt-1 text-xs font-normal text-slate-500">
              Opened{" "}
              {request.createdAt && <TimeAgo rfc3339={request.createdAt} />}
              {request.reporter?.name ? ` by ${request.reporter.name}` : ""}
              {request.clusterDomain ? ` from ${request.clusterDomain}` : ""}
            </p>
          </div>
          {(request.actions?.canEscalate || request.actions?.canResolve) && (
            <div className="flex shrink-0 flex-wrap items-center gap-2">
              {request.actions?.canEscalate && (
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<Flag size={13} strokeWidth={2.5} />}
                  disabled={!available}
                  onClick={escalate.open}
                >
                  Escalate
                </Button>
              )}
              {request.actions?.canResolve && (
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<CircleCheck size={13} strokeWidth={2.5} />}
                  disabled={!available}
                  onClick={resolveModal.open}
                >
                  Mark as resolved
                </Button>
              )}
            </div>
          )}
        </div>
      </section>

      {request.status === SupportRequest_Status.WAITING_ON_CUSTOMER && (
        <Notice tone="warning" title="Octelium is waiting for your reply">
          The response time is paused until you answer.
        </Notice>
      )}
      {request.status === SupportRequest_Status.RESOLVED && (
        <Notice tone="success" title="This request is resolved">
          Reply within {info.data?.limits?.autoCloseDays ?? 7} days if the issue
          persists; the request is closed automatically afterwards.
        </Notice>
      )}
      {request.status === SupportRequest_Status.CLOSED && (
        <Notice tone="info" title="This request is closed">
          Open a new request to continue, and mention request #{request.number}.
        </Notice>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,320px)]">
        <div className="flex min-w-0 flex-col gap-4">
          <section
            aria-label="Timeline"
            className="rounded-xl border border-slate-200 bg-slate-50/50 px-3 py-4 sm:px-4"
          >
            {olderCursor && (
              <div className="mb-4 flex justify-center">
                <Button
                  variant="default"
                  size="compact-xs"
                  loading={loadEarlier.isPending}
                  onClick={() => loadEarlier.mutate(olderCursor)}
                >
                  Show earlier activity
                </Button>
              </div>
            )}
            <ol>
              {events.map((event, index) => (
                <TimelineItem
                  key={event.id}
                  event={event}
                  request={request}
                  last={index === events.length - 1}
                />
              ))}
            </ol>
          </section>

          {request.actions?.canReply && (
            <Composer
              request={request}
              files={files}
              setFiles={setFiles}
              disabled={!available}
              onDiagnostics={() => setDiagnosticsOpened(true)}
            />
          )}
        </div>

        <aside className="flex min-w-0 flex-col gap-4">
          <SidePanel title="Details" icon={Tag}>
            <dl className="divide-y divide-slate-100">
              <Detail label="Status">
                <StatusBadge status={request.status} />
              </Detail>
              <Detail label="Priority">
                {priority ? `${priority.code} · ${priority.label}` : "—"}
              </Detail>
              {requestedPriority &&
                request.requestedPriority !== request.priority && (
                  <Detail label="Requested">{requestedPriority.code}</Detail>
                )}
              <Detail label="Support">
                {LEVEL_LABEL[request.supportLevel] ?? "Unknown"}
              </Detail>
              <Detail label="Engineer">
                {request.assignee || (
                  <span className="font-normal text-slate-500">
                    Not assigned yet
                  </span>
                )}
              </Detail>
              {sla && (
                <Detail label="First response">
                  {sla.firstRespondedAt
                    ? formatTime(sla.firstRespondedAt)
                    : sla.isPaused
                      ? "Paused"
                      : sla.firstResponseDueAt
                        ? `Due ${formatTime(sla.firstResponseDueAt)}`
                        : "—"}
                </Detail>
              )}
              {sla?.nextUpdateDueAt && isOpenStatus(request.status) && (
                <Detail label="Next update">
                  Due {formatTime(sla.nextUpdateDueAt)}
                </Detail>
              )}
              <Detail label="Opened">{formatTime(request.createdAt)}</Detail>
              <Detail label="Last activity">
                {request.lastActivityAt ? (
                  <TimeAgo rfc3339={request.lastActivityAt} />
                ) : (
                  "—"
                )}
              </Detail>
            </dl>
          </SidePanel>

          {request.attachments.length > 0 && (
            <SidePanel title="Files" icon={Paperclip}>
              <div className="flex flex-col gap-1.5 py-2">
                {request.attachments.map((attachment) => (
                  <AttachmentChip
                    key={attachment.id}
                    request={request}
                    attachment={attachment}
                  />
                ))}
              </div>
            </SidePanel>
          )}

          {Object.keys(request.environment).length > 0 && (
            <section className="@container overflow-hidden rounded-xl border border-slate-200 bg-white">
              <UnstyledButton
                onClick={() => setEnvironmentOpened((value) => !value)}
                aria-expanded={environmentOpened}
                className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400"
              >
                <span className="flex items-center gap-2 text-xs font-semibold text-slate-800">
                  <Server
                    size={13}
                    strokeWidth={2.3}
                    className="text-slate-400"
                  />
                  Cluster details
                </span>
                <ChevronDown
                  size={14}
                  strokeWidth={2.4}
                  className={twMerge(
                    "text-slate-400 transition-transform duration-150",
                    environmentOpened && "rotate-180",
                  )}
                />
              </UnstyledButton>
              <Collapse expanded={environmentOpened}>
                <div className="border-t border-slate-100 px-4 py-3">
                  <EnvironmentList environment={request.environment} />
                </div>
              </Collapse>
            </section>
          )}
        </aside>
      </div>

      <Escalate
        request={request}
        opened={escalateOpened}
        onClose={escalate.close}
      />

      <Modal
        opened={resolveOpened}
        onClose={() => !resolve.isPending && resolveModal.close()}
        centered
        size="sm"
        withCloseButton={false}
        padding={0}
      >
        <div className="flex flex-col">
          <div className="flex items-start gap-3 px-5 py-4">
            <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 text-emerald-600">
              <CircleCheck size={16} strokeWidth={2.3} />
            </span>
            <div className="min-w-0">
              <Modal.Title className="text-sm font-bold text-slate-900">
                Mark request #{request.number} as resolved
              </Modal.Title>
              <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
                Octelium Support is notified. You can still reply within{" "}
                {info.data?.limits?.autoCloseDays ?? 7} days to reopen it.
              </p>
            </div>
          </div>
          <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              disabled={resolve.isPending}
              onClick={resolveModal.close}
            >
              Cancel
            </Button>
            <Button
              variant="filled"
              color="ink"
              size="sm"
              leftSection={<CircleCheck size={13} strokeWidth={2.5} />}
              loading={resolve.isPending}
              onClick={() => resolve.mutate()}
            >
              Mark as resolved
            </Button>
          </div>
        </div>
      </Modal>

      <Diagnostics
        opened={diagnosticsOpened}
        onClose={() => setDiagnosticsOpened(false)}
        onAttach={(file) =>
          setFiles((current) =>
            current.some((item) => item.file.name === file.name)
              ? current
              : [
                  ...current,
                  toPendingFile(
                    file,
                    SupportRequest_Attachment_Purpose.DIAGNOSTIC,
                  ),
                ],
          )
        }
        maxBytes={Number(info.data?.limits?.maxAttachmentSize ?? 3 * 1048576)}
        clusterDomain={info.data?.clusterDomain}
      />
    </motion.div>
  );
};

const Request = () => {
  const { id } = useParams();
  return <RequestView key={id} id={id ?? ""} />;
};

export default Request;
