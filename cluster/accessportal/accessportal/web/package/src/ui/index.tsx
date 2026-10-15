import * as AccessP from "@/apis/accessv1/accessv1";
import { ActionIcon, Button, Modal, Tooltip } from "@mantine/core";
import {
  AlertCircle,
  ArrowLeft,
  Check,
  Copy,
  Inbox,
  Loader2,
  RefreshCw,
  Search,
  SearchX,
  UserRound,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useLocation } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "@/lib/utils";

import { Tone, decisionMeta, statusMeta, toneClasses, urgencyMeta } from "../utils";
import { describeRpcError, rpcErrorCode } from "../utils/rpcError";

export const DocumentTitle = (props: { title: string }) => (
  <title>{`${props.title} · Octelium Access Portal`}</title>
);

export const Eyebrow = (props: {
  children: React.ReactNode;
  className?: string;
  id?: string;
}) => (
  <span
    id={props.id}
    className={twMerge(
      "text-micro font-semibold uppercase tracking-[0.07em] text-slate-500",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const BackLink = (props: { to: string; children: React.ReactNode }) => {
  const location = useLocation();
  const returnTo = (location.state as { returnTo?: unknown } | null)?.returnTo;
  const target =
    typeof returnTo === "string" && returnTo.startsWith(props.to)
      ? returnTo
      : props.to;

  return (
    <Link
      to={target}
      className="group mb-3 inline-flex items-center gap-1.5 rounded text-xs font-medium text-slate-500 outline-none transition-colors duration-150 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <ArrowLeft
        size={13}
        strokeWidth={2.25}
        className="transition-transform duration-150 group-hover:-translate-x-0.5"
      />
      {props.children}
    </Link>
  );
};

export const PageHeader = (props: {
  eyebrow?: string;
  title: React.ReactNode;
  description?: string;
  actions?: React.ReactNode;
  meta?: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "mb-5 flex w-full flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-6",
      props.className,
    )}
  >
    <div className="flex min-w-0 flex-col gap-1">
      {props.eyebrow && <Eyebrow>{props.eyebrow}</Eyebrow>}
      <h1 className="text-xl font-semibold leading-tight tracking-[-0.015em] text-slate-900 [overflow-wrap:anywhere] sm:text-2xl">
        {props.title}
      </h1>
      {props.description && (
        <p className="max-w-2xl text-body font-normal text-slate-500">
          {props.description}
        </p>
      )}
      {props.meta && (
        <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1.5">
          {props.meta}
        </div>
      )}
    </div>
    {props.actions && (
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        {props.actions}
      </div>
    )}
  </div>
);

export const Card = (props: {
  children: React.ReactNode;
  className?: string;
  interactive?: boolean;
}) => (
  <div
    className={twMerge(
      "rounded-xl border border-slate-200 bg-white",
      props.interactive &&
        "transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const IconTile = (props: {
  children: React.ReactNode;
  tone?: Tone;
  size?: "sm" | "md" | "lg";
  className?: string;
}) => (
  <div
    aria-hidden
    className={twMerge(
      "flex shrink-0 items-center justify-center rounded-lg border",
      toneClasses[props.tone ?? "slate"].icon,
      props.size === "lg"
        ? "h-10 w-10"
        : props.size === "sm"
          ? "h-7 w-7 rounded-md"
          : "h-9 w-9",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const SectionCard = (props: {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  actions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  bodyClassName?: string;
}) => (
  <section
    className={twMerge(
      "min-w-0 rounded-xl border border-slate-200 bg-white",
      props.className,
    )}
  >
    <header className="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
      <div className="flex min-w-0 items-center gap-2.5">
        {props.icon && <IconTile size="sm">{props.icon}</IconTile>}
        <div className="min-w-0">
          <h2 className="text-body font-semibold text-slate-800">
            {props.title}
          </h2>
          {props.description && (
            <p className="text-micro font-normal text-slate-500">
              {props.description}
            </p>
          )}
        </div>
      </div>
      {props.actions && (
        <div className="flex shrink-0 items-center gap-2">{props.actions}</div>
      )}
    </header>
    <div className={twMerge("@container p-4", props.bodyClassName)}>
      {props.children}
    </div>
  </section>
);

type BadgeProps = {
  tone?: Tone;
  children: React.ReactNode;
  className?: string;
  icon?: React.ReactNode;
  dot?: boolean;
  mono?: boolean;
} & Omit<React.HTMLAttributes<HTMLSpanElement>, "children">;

export const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(
  ({ tone, children, className, icon, dot, mono, ...rest }, ref) => (
    <span
      ref={ref}
      {...rest}
      className={twMerge(
        "inline-flex max-w-full shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-micro font-semibold leading-4",
        mono && "rounded-md px-1.5 font-mono font-medium",
        toneClasses[tone ?? "slate"].badge,
        className,
      )}
    >
      {dot && (
        <span
          aria-hidden
          className={twMerge(
            "h-1.5 w-1.5 shrink-0 rounded-full",
            toneClasses[tone ?? "slate"].dot,
          )}
        />
      )}
      {icon}
      <span className="truncate">{children}</span>
    </span>
  ),
);

Badge.displayName = "Badge";

export const StatusBadge = (props: {
  status?: AccessP.Request_Status_State_Status;
  withHint?: boolean;
}) => {
  const meta = statusMeta(props.status);
  const badge = (
    <Badge tone={meta.tone} dot>
      {meta.label}
    </Badge>
  );

  return props.withHint ? (
    <Tooltip label={meta.hint}>{badge}</Tooltip>
  ) : (
    badge
  );
};

export const DecisionBadge = (props: {
  decision?: AccessP.Review_Spec_Decision;
}) => {
  const meta = decisionMeta(props.decision);
  return (
    <Badge tone={meta.tone} dot>
      {meta.label}
    </Badge>
  );
};

export const UrgencyBadge = (props: {
  urgency?: AccessP.Request_Spec_Urgency;
}) => {
  const meta = urgencyMeta(props.urgency);
  return <Badge tone={meta.tone}>{meta.label} urgency</Badge>;
};

const initialsOf = (name?: string) =>
  (name ?? "")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();

export const Avatar = (props: {
  src?: string;
  name?: string;
  size?: "xs" | "sm" | "md" | "lg";
  className?: string;
}) => {
  const [failedSrc, setFailedSrc] = React.useState<string>();
  const initials = initialsOf(props.name);
  const showImage = !!props.src && failedSrc !== props.src;

  const size =
    props.size === "xs"
      ? "h-6 w-6 text-[0.55rem]"
      : props.size === "sm"
        ? "h-8 w-8 text-[0.625rem]"
        : props.size === "lg"
          ? "h-12 w-12 text-sm"
          : "h-10 w-10 text-xs";

  return (
    <div
      className={twMerge(
        "flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-slate-700 font-semibold text-white",
        size,
        props.className,
      )}
    >
      {showImage ? (
        <img
          src={props.src}
          alt=""
          onError={() => setFailedSrc(props.src)}
          className="h-full w-full object-cover"
        />
      ) : initials ? (
        <span aria-hidden>{initials}</span>
      ) : (
        <UserRound size={14} strokeWidth={2} aria-hidden />
      )}
    </div>
  );
};

export const UserChip = (props: {
  name: string;
  secondary?: string;
  src?: string;
  badge?: React.ReactNode;
  size?: "sm" | "md";
  className?: string;
}) => (
  <div className={twMerge("flex min-w-0 items-center gap-2.5", props.className)}>
    <Avatar src={props.src} name={props.name} size={props.size ?? "sm"} />
    <div className="min-w-0">
      <div className="flex min-w-0 items-center gap-1.5">
        <span className="truncate text-body font-semibold text-slate-800">
          {props.name}
        </span>
        {props.badge}
      </div>
      {props.secondary && (
        <p className="truncate text-xs font-normal text-slate-500">
          {props.secondary}
        </p>
      )}
    </div>
  </div>
);

export const Field = (props: {
  label: string;
  description?: string;
  children: React.ReactNode;
  hint?: React.ReactNode;
  error?: string;
}) => {
  const id = React.useId();
  const describedBy =
    [props.description && `${id}-description`, props.error && `${id}-error`]
      .filter(Boolean)
      .join(" ") || undefined;

  return (
    <div
      role="group"
      aria-labelledby={`${id}-label`}
      aria-describedby={describedBy}
      className="flex flex-col gap-1"
    >
      <div className="flex items-baseline justify-between gap-3">
        <span id={`${id}-label`} className="text-xs font-semibold text-slate-700">
          {props.label}
        </span>
        {props.hint}
      </div>
      {props.description && (
        <span
          id={`${id}-description`}
          className="mb-0.5 text-xs font-normal leading-[1.45] text-slate-500"
        >
          {props.description}
        </span>
      )}
      {props.children}
      {props.error && (
        <span id={`${id}-error`} className="text-xs font-medium text-red-600">
          {props.error}
        </span>
      )}
    </div>
  );
};

export const LengthCounter = (props: { length: number; max: number }) => (
  <span
    className={twMerge(
      "text-micro font-medium tabular-nums",
      props.length > props.max ? "text-red-600" : "text-slate-500",
    )}
  >
    {props.length}/{props.max}
  </span>
);

export const InfoGrid = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-x-5 gap-y-4 @md:grid-cols-2",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const KeyValue = (props: {
  label: string;
  children: React.ReactNode;
  full?: boolean;
  mono?: boolean;
}) => (
  <div className={twMerge("flex min-w-0 flex-col gap-1", props.full && "@md:col-span-2")}>
    <Eyebrow>{props.label}</Eyebrow>
    <div
      className={twMerge(
        "flex min-w-0 items-center gap-1.5 text-body font-medium text-slate-800",
        props.mono && "font-mono text-xs [overflow-wrap:anywhere]",
      )}
    >
      {props.children}
    </div>
  </div>
);

export const MonoValue = (props: {
  children: React.ReactNode;
  className?: string;
  title?: string;
}) => (
  <span
    title={props.title}
    className={twMerge(
      "inline-block max-w-full truncate rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-mono text-xs font-medium text-slate-700",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const CopyValue = (props: {
  value: string;
  label?: string;
  copyLabel?: string;
}) => {
  const [copied, setCopied] = React.useState(false);
  const timer = React.useRef<number>(undefined);

  React.useEffect(() => () => window.clearTimeout(timer.current), []);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(props.value);
      setCopied(true);
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setCopied(false), 1600);
    } catch {
      toast.error("Could not copy to the clipboard");
    }
  };

  return (
    <div className="flex min-w-0 items-center gap-1">
      <span className="truncate font-mono text-xs font-medium text-slate-600">
        {props.label ?? props.value}
      </span>
      <Tooltip label={copied ? "Copied" : "Copy"}>
        <ActionIcon
          size="sm"
          variant="subtle"
          color="gray"
          aria-label={props.copyLabel ?? "Copy value"}
          onClick={copy}
        >
          {copied ? (
            <Check size={12} strokeWidth={2.5} className="text-emerald-600" />
          ) : (
            <Copy size={12} strokeWidth={2.25} />
          )}
        </ActionIcon>
      </Tooltip>
    </div>
  );
};

export const Note = (props: {
  tone?: Tone;
  icon?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "flex items-start gap-2 rounded-lg border px-3 py-2.5 text-xs font-normal leading-5",
      toneClasses[props.tone ?? "slate"].soft,
      props.className,
    )}
  >
    {props.icon && <span className="mt-[3px] shrink-0">{props.icon}</span>}
    <div className="min-w-0">{props.children}</div>
  </div>
);

export const StatTile = (props: {
  label: string;
  value: React.ReactNode;
  hint?: string;
  tone?: Tone;
  icon?: React.ReactNode;
  active?: boolean;
  onClick?: () => void;
}) => {
  const content = (
    <>
      <div className="flex items-center gap-2 text-slate-500">
        {props.icon}
        <Eyebrow>{props.label}</Eyebrow>
      </div>
      <p
        className={twMerge(
          "mt-2 text-xl font-semibold leading-none tracking-tight text-slate-900",
          props.tone && toneClasses[props.tone].text,
        )}
      >
        {props.value}
      </p>
      {props.hint && (
        <p className="mt-1.5 text-micro font-normal text-slate-500">
          {props.hint}
        </p>
      )}
    </>
  );

  const className = twMerge(
    "rounded-xl border bg-white px-3.5 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
    props.active ? "border-slate-900 ring-1 ring-slate-900" : "border-slate-200",
    props.onClick &&
      "cursor-pointer transition-[border-color,box-shadow] duration-150 hover:border-slate-400",
  );

  return props.onClick ? (
    <button
      type="button"
      onClick={props.onClick}
      aria-pressed={!!props.active}
      className={className}
    >
      {content}
    </button>
  ) : (
    <div className={className}>{content}</div>
  );
};

export const ProgressBar = (props: {
  value: number;
  tone?: Tone;
  className?: string;
  label?: string;
}) => {
  const percent = Math.round(Math.min(100, Math.max(0, props.value * 100)));
  return (
    <div
      role="progressbar"
      aria-label={props.label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={percent}
      className={twMerge(
        "h-1.5 w-full overflow-hidden rounded-full bg-slate-100",
        props.className,
      )}
    >
      <div
        className={twMerge(
          "h-full rounded-full transition-[width] duration-500",
          toneClasses[props.tone ?? "slate"].bar,
        )}
        style={{ width: `${percent}%` }}
      />
    </div>
  );
};

export const Timeline = (props: { children: React.ReactNode }) => (
  <ol className="relative flex flex-col">{props.children}</ol>
);

export const TimelineItem = (props: {
  tone?: Tone;
  icon?: React.ReactNode;
  title: React.ReactNode;
  meta?: React.ReactNode;
  children?: React.ReactNode;
  last?: boolean;
}) => (
  <li className="relative flex gap-3 pb-4 last:pb-0">
    {!props.last && (
      <span
        aria-hidden
        className="absolute bottom-0 left-[11px] top-6 w-px bg-slate-200"
      />
    )}
    <span
      aria-hidden
      className="relative z-10 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500"
    >
      {props.icon ?? (
        <span
          className={twMerge(
            "h-2 w-2 rounded-full",
            toneClasses[props.tone ?? "slate"].dot,
          )}
        />
      )}
    </span>
    <div className="min-w-0 flex-1 pt-0.5">
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-0.5">
        <span className="text-xs font-semibold text-slate-800">
          {props.title}
        </span>
        {props.meta && (
          <span className="text-micro font-normal text-slate-500">
            {props.meta}
          </span>
        )}
      </div>
      {props.children && (
        <div className="mt-0.5 text-xs font-normal leading-5 text-slate-600 [overflow-wrap:anywhere]">
          {props.children}
        </div>
      )}
    </div>
  </li>
);

export const SearchInput = (props: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  ariaLabel?: string;
  className?: string;
}) => (
  <div className={twMerge("relative min-w-0 flex-1", props.className)}>
    <Search
      size={14}
      strokeWidth={2.25}
      aria-hidden
      className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
    />
    <input
      value={props.value}
      onChange={(event) => props.onChange(event.target.value)}
      placeholder={props.placeholder}
      aria-label={props.ariaLabel ?? props.placeholder}
      className="h-[38px] w-full rounded-lg border border-slate-200 bg-white pl-9 pr-10 text-body font-normal text-slate-900 outline-none transition-[border-color,box-shadow] duration-150 placeholder:font-normal placeholder:text-slate-400 focus:border-slate-900 focus:shadow-[0_0_0_2px_color-mix(in_oklab,var(--color-slate-900)_12%,transparent)]"
    />
    {props.value && (
      <button
        type="button"
        aria-label="Clear search"
        onClick={() => props.onChange("")}
        className="absolute right-1.5 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-md text-slate-500 outline-none transition-colors hover:bg-slate-100 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <X size={13} strokeWidth={2.5} />
      </button>
    )}
  </div>
);

export const Toolbar = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "mb-4 flex flex-col gap-2 sm:flex-row sm:items-center",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const RefreshButton = (props: {
  onClick: () => void;
  loading?: boolean;
}) => (
  <Tooltip label="Refresh">
    <ActionIcon
      variant="default"
      size={36}
      radius="md"
      aria-label="Refresh"
      aria-busy={props.loading || undefined}
      onClick={props.onClick}
    >
      <RefreshCw
        size={14}
        strokeWidth={2.25}
        className={twMerge(
          "text-slate-600",
          props.loading && "motion-safe:animate-spin",
        )}
      />
    </ActionIcon>
  </Tooltip>
);

export const Loading = (props: { label?: string }) => (
  <div role="status" className="flex w-full items-center justify-center py-16">
    <div className="flex items-center gap-2 text-slate-500">
      <Loader2 size={15} strokeWidth={2.25} className="motion-safe:animate-spin" />
      <span className="text-xs font-medium">{props.label ?? "Loading…"}</span>
    </div>
  </div>
);

export const SkeletonRows = (props: { rows?: number }) => (
  <div aria-hidden className="flex flex-col gap-2">
    {Array.from({ length: props.rows ?? 4 }).map((_, index) => (
      <div
        key={index}
        className="flex items-center gap-4 rounded-xl border border-slate-200 bg-white px-4 py-3.5"
      >
        <div className="h-9 w-9 shrink-0 rounded-lg bg-slate-100 motion-safe:animate-pulse" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-1/3 rounded bg-slate-100 motion-safe:animate-pulse" />
          <div className="h-2.5 w-1/2 rounded bg-slate-50 motion-safe:animate-pulse" />
        </div>
        <div className="h-5 w-16 rounded-full bg-slate-100 motion-safe:animate-pulse" />
      </div>
    ))}
  </div>
);

export const EmptyState = (props: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
  heading?: boolean;
}) => (
  <div className="flex w-full flex-col items-center justify-center px-6 py-12 text-center">
    <div
      aria-hidden
      className="mb-3 flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500"
    >
      {props.icon ?? <Inbox size={18} strokeWidth={2} />}
    </div>
    {props.heading ? (
      <h1 className="text-base font-semibold text-slate-900">{props.title}</h1>
    ) : (
      <p className="text-body font-semibold text-slate-800">{props.title}</p>
    )}
    {props.description && (
      <p className="mt-1 max-w-sm text-xs font-normal leading-5 text-slate-500">
        {props.description}
      </p>
    )}
    {props.action && <div className="mt-4">{props.action}</div>}
  </div>
);

export const isNotFoundError = (error: unknown) => {
  const code = rpcErrorCode(error);
  return code === "NOT_FOUND" || code === "PERMISSION_DENIED";
};

export const ErrorState = (props: {
  title?: string;
  description?: string;
  error?: unknown;
  onRetry?: () => void;
}) => (
  <Card className="p-8">
    <div
      role="alert"
      className="flex w-full flex-col items-center justify-center text-center"
    >
      <div
        aria-hidden
        className="mb-3 flex h-10 w-10 items-center justify-center rounded-xl border border-red-200 bg-red-50 text-red-600"
      >
        <AlertCircle size={18} strokeWidth={2} />
      </div>
      <p className="text-body font-semibold text-slate-800">
        {props.title ?? "Something went wrong"}
      </p>
      <p className="mt-1 max-w-sm text-xs font-normal leading-5 text-slate-500">
        {props.description ??
          (props.error !== undefined
            ? describeRpcError(props.error).message
            : "This information could not be loaded. Try again.")}
      </p>
      {props.onRetry && (
        <Button
          variant="default"
          className="mt-4"
          leftSection={<RefreshCw size={13} strokeWidth={2.25} />}
          onClick={props.onRetry}
        >
          Try again
        </Button>
      )}
    </div>
  </Card>
);

export const NotFoundState = (props: {
  title: string;
  description: string;
  action?: React.ReactNode;
}) => (
  <Card>
    <EmptyState
      icon={<SearchX size={18} strokeWidth={2} />}
      title={props.title}
      description={props.description}
      action={props.action}
      heading
    />
  </Card>
);

export const ConfirmDialog = (props: {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: React.ReactNode;
  confirmLabel: string;
  loading?: boolean;
  danger?: boolean;
  details?: React.ReactNode;
}) => (
  <Modal
    opened={props.opened}
    onClose={props.onClose}
    centered
    radius="lg"
    title={props.title}
    closeOnEscape={!props.loading}
    closeOnClickOutside={!props.loading}
    withCloseButton={!props.loading}
    styles={{
      title: {
        fontSize: "0.9375rem",
        fontWeight: 600,
        color: "var(--color-slate-900)",
      },
    }}
  >
    <div className="text-body font-normal text-slate-600">
      {props.description}
    </div>
    {props.details && (
      <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2">
        {props.details}
      </div>
    )}
    <div className="mt-6 flex items-center justify-end gap-2">
      <Button variant="default" onClick={props.onClose} disabled={props.loading}>
        Cancel
      </Button>
      <Button
        variant="filled"
        color={props.danger ? "red" : "ink"}
        loading={props.loading}
        onClick={props.onConfirm}
      >
        {props.confirmLabel}
      </Button>
    </div>
  </Modal>
);

export const SummaryRow = (props: {
  label: string;
  children: React.ReactNode;
}) => (
  <div className="flex items-start justify-between gap-3 py-1">
    <span className="shrink-0 text-xs font-normal text-slate-500">
      {props.label}
    </span>
    <span className="min-w-0 text-right text-xs font-medium text-slate-800 [overflow-wrap:anywhere]">
      {props.children}
    </span>
  </div>
);
