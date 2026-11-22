import {
  SupportInfo,
  SupportInfo_State,
  SupportRequest,
  SupportRequest_Attachment,
  SupportRequest_Attachment_Purpose,
  SupportRequest_Priority,
  SupportRequest_Status,
} from "@/apis/enterprisev1/enterprisev1";
import { PILL } from "@/components/Dashboard/components";
import { twMerge } from "@/lib/utils";
import { formatBytes } from "@/pages/Agent/format";
import { onError } from "@/utils";
import { ActionIcon, Button, Loader, Tooltip } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import {
  AlertTriangle,
  BadgeCheck,
  CircleSlash,
  Download,
  ExternalLink,
  FileArchive,
  FileText,
  Mail,
  Paperclip,
  Plus,
  ShieldOff,
  Stethoscope,
  Unplug,
  X,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router";
import { match } from "ts-pattern";
import { downloadAttachment, PendingFile } from "./queries";
import {
  ENVIRONMENT_LABELS,
  environmentEntries,
  getPriority,
  getState,
  getStatus,
  Tone,
  TONE_DOT,
  TONE_SOFT,
} from "./utils";

export const ToneBadge = (props: {
  tone: Tone;
  children: React.ReactNode;
  dot?: boolean;
  className?: string;
}) => (
  <span className={twMerge(PILL, TONE_SOFT[props.tone], props.className)}>
    {props.dot && (
      <span
        aria-hidden="true"
        className={twMerge("h-1.5 w-1.5 rounded-full", TONE_DOT[props.tone])}
      />
    )}
    {props.children}
  </span>
);

export const StatusBadge = (props: { status?: SupportRequest_Status }) => {
  const meta = getStatus(props.status);
  return (
    <ToneBadge tone={meta.tone} dot>
      {meta.label}
    </ToneBadge>
  );
};

export const PriorityBadge = (props: {
  priority?: SupportRequest_Priority;
  label?: boolean;
}) => {
  const meta = getPriority(props.priority);
  if (!meta) return null;
  return (
    <ToneBadge tone={meta.tone} className="tabular-nums">
      {meta.code}
      {props.label && <span className="font-normal">· {meta.label}</span>}
    </ToneBadge>
  );
};

export const StateBadge = (props: { state?: SupportInfo_State }) => {
  const meta = getState(props.state);
  return (
    <ToneBadge tone={meta.tone} dot>
      {meta.label}
    </ToneBadge>
  );
};

export const PortalLink = (props: {
  href: string;
  children: React.ReactNode;
  className?: string;
}) => (
  <a
    href={props.href}
    target="_blank"
    rel="noopener noreferrer"
    className={twMerge(
      "inline-flex items-center gap-1 font-semibold text-slate-700 underline decoration-slate-300 underline-offset-2 transition-colors hover:text-slate-900 hover:decoration-slate-500",
      props.className,
    )}
  >
    {props.children}
    <ExternalLink size={11} strokeWidth={2.4} aria-hidden="true" />
  </a>
);

const STATE_ICONS: Record<number, typeof AlertTriangle> = {
  [SupportInfo_State.NO_LICENSE]: ShieldOff,
  [SupportInfo_State.NOT_ENTITLED]: CircleSlash,
  [SupportInfo_State.UNREACHABLE]: Unplug,
};

export const StateNotice = (props: { info: SupportInfo }) => {
  const { info } = props;
  if (
    info.state === SupportInfo_State.AVAILABLE ||
    info.state === SupportInfo_State.STATE_UNKNOWN
  ) {
    return null;
  }

  const meta = getState(info.state);
  const Icon = STATE_ICONS[info.state] ?? AlertTriangle;

  const hint = match(info.state)
    .with(
      SupportInfo_State.NO_LICENSE,
      () =>
        "Support requests are authenticated with the license of the Cluster. Set a valid license to open requests from the console.",
    )
    .with(
      SupportInfo_State.NOT_ENTITLED,
      () =>
        "Licensing, billing and account requests can still be opened from the customer portal.",
    )
    .with(
      SupportInfo_State.UNREACHABLE,
      () =>
        "The Cluster needs outbound HTTPS access to the customer portal. You can still contact Octelium Support from the customer portal or by email.",
    )
    .otherwise(
      () =>
        "You can still contact Octelium Support from the customer portal or by email.",
    );

  return (
    <div
      role="status"
      className={twMerge(
        "flex flex-col gap-3 rounded-xl border px-4 py-3.5 sm:flex-row sm:items-start sm:px-5",
        meta.tone === "danger"
          ? "border-red-200 bg-red-50/60"
          : meta.tone === "warning"
            ? "border-amber-200 bg-amber-50/60"
            : "border-slate-200 bg-slate-50/70",
      )}
    >
      <span
        className={twMerge(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border bg-white",
          meta.tone === "danger"
            ? "border-red-200 text-red-600"
            : meta.tone === "warning"
              ? "border-amber-200 text-amber-600"
              : "border-slate-200 text-slate-500",
        )}
      >
        <Icon size={16} strokeWidth={2.3} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="text-body font-semibold text-slate-900">
          {meta.title}
        </div>
        {info.message && (
          <p className="mt-0.5 break-words text-xs font-normal leading-5 text-slate-700">
            {info.message}
          </p>
        )}
        <p className="mt-0.5 text-xs font-normal leading-5 text-slate-600">
          {hint}
        </p>
        <div className="mt-2.5 flex flex-wrap items-center gap-2">
          {info.state === SupportInfo_State.NO_LICENSE && (
            <Button
              component={Link}
              to="/clusterman"
              variant="default"
              size="compact-xs"
              leftSection={<BadgeCheck size={11} strokeWidth={2.5} />}
            >
              Manage the license
            </Button>
          )}
          {info.portalURL && (
            <Button
              component="a"
              href={info.portalURL}
              target="_blank"
              rel="noopener noreferrer"
              variant="default"
              size="compact-xs"
              leftSection={<ExternalLink size={11} strokeWidth={2.5} />}
            >
              Customer portal
            </Button>
          )}
          {info.supportEmail && (
            <Button
              component="a"
              href={`mailto:${info.supportEmail}`}
              variant="default"
              size="compact-xs"
              leftSection={<Mail size={11} strokeWidth={2.5} />}
            >
              {info.supportEmail}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};

const FileIcon = (props: {
  name: string;
  diagnostic: boolean;
  size: number;
  className?: string;
}) =>
  props.diagnostic ? (
    <Stethoscope
      size={props.size}
      strokeWidth={2.3}
      className={props.className}
    />
  ) : /\.(gz|zip)$/i.test(props.name) ? (
    <FileArchive
      size={props.size}
      strokeWidth={2.3}
      className={props.className}
    />
  ) : (
    <FileText size={props.size} strokeWidth={2.3} className={props.className} />
  );

export const PendingFiles = (props: {
  files: PendingFile[];
  disabled?: boolean;
  progress?: Record<string, "uploading" | "done" | "failed">;
  onRemove: (key: string) => void;
}) => {
  if (props.files.length === 0) return null;

  return (
    <ul className="flex flex-col gap-1.5" aria-label="Files to send">
      {props.files.map((item) => {
        const diagnostic =
          item.purpose === SupportRequest_Attachment_Purpose.DIAGNOSTIC;
        const state = props.progress?.[item.key];
        return (
          <li
            key={item.key}
            className="flex min-w-0 items-center gap-2.5 rounded-lg border border-slate-200 bg-white px-2.5 py-2"
          >
            <span
              className={twMerge(
                "flex h-7 w-7 shrink-0 items-center justify-center rounded-md border",
                diagnostic
                  ? "border-blue-200 bg-blue-50 text-blue-700"
                  : "border-slate-200 bg-slate-50 text-slate-500",
              )}
            >
              <FileIcon
                name={item.file.name}
                diagnostic={diagnostic}
                size={13}
              />
            </span>
            <div className="min-w-0 flex-1">
              <div className="truncate text-xs font-semibold text-slate-800">
                {item.file.name}
              </div>
              <div className="flex flex-wrap items-center gap-x-2 text-micro font-normal text-slate-500">
                <span className="tabular-nums">
                  {formatBytes(item.file.size)}
                </span>
                {diagnostic && <span>Diagnostics</span>}
                {state === "failed" && (
                  <span className="font-semibold text-red-600">
                    Upload failed
                  </span>
                )}
              </div>
            </div>
            {state === "uploading" ? (
              <Loader size={14} aria-label={`Uploading ${item.file.name}`} />
            ) : (
              <Tooltip label="Remove" withArrow>
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  size="sm"
                  aria-label={`Remove ${item.file.name}`}
                  disabled={props.disabled}
                  onClick={() => props.onRemove(item.key)}
                >
                  <X size={13} strokeWidth={2.5} />
                </ActionIcon>
              </Tooltip>
            )}
          </li>
        );
      })}
    </ul>
  );
};

export const FileButton = (props: {
  disabled?: boolean;
  accept?: string[];
  onFiles: (files: File[]) => void;
}) => {
  const ref = React.useRef<HTMLInputElement>(null);

  return (
    <>
      <input
        ref={ref}
        type="file"
        multiple
        hidden
        accept={props.accept?.map((ext) => `.${ext}`).join(",")}
        onChange={(event) => {
          const files = Array.from(event.currentTarget.files ?? []);
          event.currentTarget.value = "";
          if (files.length > 0) props.onFiles(files);
        }}
      />
      <Button
        variant="default"
        size="xs"
        disabled={props.disabled}
        leftSection={<Paperclip size={13} strokeWidth={2.4} />}
        onClick={() => ref.current?.click()}
      >
        Attach files
      </Button>
    </>
  );
};

export const DiagnosticsButton = (props: {
  disabled?: boolean;
  onClick: () => void;
}) => (
  <Button
    variant="default"
    size="xs"
    disabled={props.disabled}
    leftSection={<Stethoscope size={13} strokeWidth={2.4} />}
    onClick={props.onClick}
  >
    Collect diagnostics
  </Button>
);

export const AttachmentChip = (props: {
  request: SupportRequest;
  attachment: SupportRequest_Attachment;
}) => {
  const { attachment } = props;
  const diagnostic =
    attachment.purpose === SupportRequest_Attachment_Purpose.DIAGNOSTIC;
  const mutation = useMutation({
    mutationFn: () => downloadAttachment(props.request, attachment),
    onError,
  });

  return (
    <div className="flex min-w-0 items-center gap-2 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5">
      <FileIcon
        name={attachment.name}
        diagnostic={diagnostic}
        size={13}
        className={twMerge(
          "shrink-0",
          diagnostic ? "text-blue-600" : "text-slate-500",
        )}
      />
      <div className="min-w-0 flex-1">
        <div className="truncate text-xs font-semibold text-slate-800">
          {attachment.name}
        </div>
        <div className="text-micro font-normal tabular-nums text-slate-500">
          {formatBytes(Number(attachment.size))}
          {diagnostic && " · Diagnostics"}
          {!attachment.isAvailable && " · Deleted after retention"}
        </div>
      </div>
      {attachment.isAvailable && (
        <Tooltip label="Download" withArrow>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={`Download ${attachment.name}`}
            loading={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            <Download size={13} strokeWidth={2.4} />
          </ActionIcon>
        </Tooltip>
      )}
    </div>
  );
};

export const EnvironmentList = (props: {
  environment?: Record<string, string>;
  className?: string;
}) => {
  const entries = environmentEntries(props.environment);
  if (entries.length === 0) {
    return (
      <p className="text-xs font-normal text-slate-500">
        No Cluster details are available.
      </p>
    );
  }

  return (
    <dl
      className={twMerge(
        "grid grid-cols-1 gap-x-4 gap-y-2 @md:grid-cols-2",
        props.className,
      )}
    >
      {entries.map(([key, value]) => (
        <div key={key} className="min-w-0">
          <dt className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            {ENVIRONMENT_LABELS[key] ?? key}
          </dt>
          <dd className="break-words font-mono text-xs font-semibold text-slate-700">
            {value}
          </dd>
        </div>
      ))}
    </dl>
  );
};

export const NewRequestButton = (props: {
  disabled?: boolean;
  size?: "xs" | "sm";
}) => (
  <Button
    component={Link}
    to="/support/new"
    variant="filled"
    color="ink"
    size={props.size ?? "xs"}
    disabled={props.disabled}
    leftSection={<Plus size={14} strokeWidth={2.5} />}
    onClick={(event: React.MouseEvent) => {
      if (props.disabled) event.preventDefault();
    }}
  >
    New request
  </Button>
);
