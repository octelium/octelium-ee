import * as AccessP from "@/apis/accessv1/accessv1";
import { ActionIcon, Button, Modal, Tooltip } from "@mantine/core";
import {
  AlertCircle,
  ArrowLeft,
  Ban,
  Check,
  ChevronDown,
  ChevronUp,
  ChevronsUp,
  CircleDashed,
  Clock3,
  Copy,
  Flame,
  Loader2,
  Minus,
  RefreshCw,
  Search,
  UserRound,
  X,
  XCircle,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";

import {
  Tone,
  decisionMeta,
  statusMeta,
  toneClasses,
  urgencyMeta,
} from "../utils";

export const Eyebrow = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "text-[0.62rem] font-bold uppercase tracking-[0.09em] text-slate-400",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const BackLink = (props: { to: string; children: React.ReactNode }) => (
  <Link
    to={props.to}
    className="group inline-flex items-center gap-1.5 mb-4 text-[0.72rem] font-bold text-slate-400 transition-colors duration-150 hover:text-slate-700"
  >
    <ArrowLeft
      size={13}
      strokeWidth={2.5}
      className="transition-transform duration-150 group-hover:-translate-x-0.5"
    />
    {props.children}
  </Link>
);

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
      "w-full flex flex-col gap-3 mb-5 sm:flex-row sm:items-start sm:justify-between sm:gap-6",
      props.className,
    )}
  >
    <div className="flex min-w-0 flex-col gap-1">
      {props.eyebrow && <Eyebrow>{props.eyebrow}</Eyebrow>}
      <h1 className="text-[1.45rem] font-bold leading-tight tracking-[-0.015em] text-slate-900 break-words">
        {props.title}
      </h1>
      {props.description && (
        <p className="max-w-2xl text-[0.8rem] font-medium leading-relaxed text-slate-500">
          {props.description}
        </p>
      )}
      {props.meta && (
        <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1.5">
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
  onClick?: () => void;
  interactive?: boolean;
}) => (
  <div
    onClick={props.onClick}
    className={twMerge(
      "rounded-xl border border-slate-200 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.04)]",
      props.interactive &&
        "cursor-pointer transition-[border-color,box-shadow,transform] duration-150 hover:border-slate-300 hover:shadow-[0_4px_16px_rgba(15,23,42,0.08)]",
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
    className={twMerge(
      "flex shrink-0 items-center justify-center rounded-lg",
      toneClasses[props.tone ?? "slate"].icon,
      props.size === "lg"
        ? "h-11 w-11 rounded-xl"
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
  tone?: Tone;
  actions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  bodyClassName?: string;
}) => (
  <Card className={twMerge("overflow-hidden", props.className)}>
    <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
      <div className="flex min-w-0 items-center gap-2.5">
        {props.icon && (
          <IconTile size="sm" tone={props.tone}>
            {props.icon}
          </IconTile>
        )}
        <div className="min-w-0">
          <h2 className="text-[0.83rem] font-bold leading-snug text-slate-800">
            {props.title}
          </h2>
          {props.description && (
            <p className="text-[0.7rem] font-medium leading-snug text-slate-400">
              {props.description}
            </p>
          )}
        </div>
      </div>
      {props.actions && (
        <div className="flex shrink-0 items-center gap-2">{props.actions}</div>
      )}
    </div>
    <div className={twMerge("p-4", props.bodyClassName)}>{props.children}</div>
  </Card>
);

export const SectionTitle = (props: { children: React.ReactNode }) => (
  <h2 className="mb-3 text-[0.8rem] font-bold text-slate-800">
    {props.children}
  </h2>
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
        "inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[0.65rem] font-bold leading-4 tracking-[0.02em]",
        mono ? "font-mono" : "uppercase",
        toneClasses[tone ?? "slate"].badge,
        className,
      )}
    >
      {dot && (
        <span
          className={twMerge(
            "h-1.5 w-1.5 rounded-full",
            toneClasses[tone ?? "slate"].dot,
          )}
        />
      )}
      {icon}
      {children}
    </span>
  ),
);

Badge.displayName = "Badge";

export const StatusBadge = (props: {
  status?: AccessP.Request_Status_State_Status;
  withHint?: boolean;
}) => {
  const meta = statusMeta(props.status);
  const icon =
    meta.group === "pending" ? (
      <Clock3 size={11} strokeWidth={2.6} />
    ) : meta.group === "active" ? (
      <Check size={11} strokeWidth={3} />
    ) : meta.label === "Rejected" || meta.label === "Revoked" ? (
      <XCircle size={11} strokeWidth={2.6} />
    ) : (
      <CircleDashed size={11} strokeWidth={2.6} />
    );

  const badge = (
    <Badge tone={meta.tone} icon={icon}>
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
    <Badge
      tone={meta.tone}
      icon={
        meta.tone === "emerald" ? (
          <Check size={11} strokeWidth={3} />
        ) : meta.tone === "red" ? (
          <X size={11} strokeWidth={3} />
        ) : (
          <CircleDashed size={11} strokeWidth={2.6} />
        )
      }
    >
      {meta.label}
    </Badge>
  );
};

const urgencyIcon = (level: number, size = 11) => {
  if (level >= 6) return <Flame size={size} strokeWidth={2.6} />;
  if (level >= 5) return <ChevronsUp size={size} strokeWidth={3} />;
  if (level >= 4) return <ChevronUp size={size} strokeWidth={3} />;
  if (level === 3) return <Minus size={size} strokeWidth={3} />;
  return <ChevronDown size={size} strokeWidth={3} />;
};

export const UrgencyBadge = (props: {
  urgency?: AccessP.Request_Spec_Urgency;
  short?: boolean;
}) => {
  const meta = urgencyMeta(props.urgency);
  return (
    <Tooltip label={`Urgency: ${meta.label}`}>
      <Badge tone={meta.tone} icon={urgencyIcon(meta.level)}>
        {props.short ? meta.short : meta.label}
      </Badge>
    </Tooltip>
  );
};

export const UrgencyMeter = (props: {
  urgency?: AccessP.Request_Spec_Urgency;
}) => {
  const meta = urgencyMeta(props.urgency);
  return (
    <Tooltip label={`Urgency: ${meta.label}`}>
      <div className="flex items-end gap-[2px]" aria-label={`Urgency ${meta.label}`}>
        {[1, 2, 3, 4, 5, 6].map((step) => (
          <span
            key={step}
            className={twMerge(
              "w-[3px] rounded-full transition-colors",
              step <= meta.level
                ? toneClasses[meta.tone].bar
                : "bg-slate-200",
            )}
            style={{ height: `${4 + step * 1.6}px` }}
          />
        ))}
      </div>
    </Tooltip>
  );
};

export const Avatar = (props: {
  src?: string;
  name?: string;
  size?: "xs" | "sm" | "md" | "lg";
  className?: string;
}) => {
  const initials = (props.name ?? "")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();

  const size =
    props.size === "xs"
      ? "w-6 h-6 text-[0.52rem]"
      : props.size === "sm"
        ? "w-8 h-8 text-[0.6rem]"
        : props.size === "lg"
          ? "w-12 h-12 text-[0.78rem]"
          : "w-10 h-10 text-[0.68rem]";

  return (
    <div
      className={twMerge(
        "flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-slate-800 font-bold text-white",
        size,
        props.className,
      )}
    >
      {props.src ? (
        <img
          src={props.src}
          alt={props.name ?? "User"}
          className="h-full w-full object-cover"
        />
      ) : initials ? (
        <span>{initials}</span>
      ) : (
        <UserRound size={14} strokeWidth={2} />
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
      <div className="flex items-center gap-1.5">
        <span className="truncate text-[0.8rem] font-bold text-slate-800">
          {props.name}
        </span>
        {props.badge}
      </div>
      {props.secondary && (
        <p className="truncate text-[0.68rem] font-medium text-slate-400">
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
}) => (
  <div className="flex flex-col gap-1.5">
    <div className="flex items-baseline justify-between gap-3">
      <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500">
        {props.label}
      </span>
      {props.hint}
    </div>
    {props.description && (
      <span className="-mt-1 text-[0.68rem] font-medium leading-relaxed text-slate-400">
        {props.description}
      </span>
    )}
    {props.children}
  </div>
);

export const InfoGrid = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-x-5 gap-y-4 sm:grid-cols-2",
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
  icon?: React.ReactNode;
  mono?: boolean;
}) => (
  <div className={twMerge("flex min-w-0 flex-col gap-1", props.full && "sm:col-span-2")}>
    <Eyebrow>{props.label}</Eyebrow>
    <div
      className={twMerge(
        "flex min-w-0 items-center gap-1.5 text-[0.8rem] font-semibold text-slate-700",
        props.mono && "font-mono text-[0.76rem]",
      )}
    >
      {props.icon}
      {props.children}
    </div>
  </div>
);

export const MonoValue = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "truncate rounded bg-slate-50 px-1.5 py-0.5 font-mono text-[0.72rem] font-semibold text-slate-600",
      props.className,
    )}
  >
    {props.children}
  </span>
);

export const CopyValue = (props: { value: string; label?: string }) => {
  const [copied, setCopied] = React.useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(props.value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      toast.error("Could not copy to the clipboard");
    }
  };

  return (
    <div className="flex min-w-0 items-center gap-1">
      <span className="truncate font-mono text-[0.72rem] font-semibold text-slate-600">
        {props.label ?? props.value}
      </span>
      <Tooltip label={copied ? "Copied" : "Copy"}>
        <ActionIcon
          size="sm"
          variant="subtle"
          color="gray"
          aria-label="Copy value"
          onClick={copy}
        >
          {copied ? (
            <Check size={12} strokeWidth={2.8} className="text-emerald-600" />
          ) : (
            <Copy size={12} strokeWidth={2.4} />
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
      "flex items-start gap-2 rounded-lg border px-3 py-2.5 text-[0.74rem] font-medium leading-relaxed",
      toneClasses[props.tone ?? "slate"].soft,
      toneClasses[props.tone ?? "slate"].text,
      props.className,
    )}
  >
    {props.icon && <span className="mt-[1px] shrink-0">{props.icon}</span>}
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
  const tone = toneClasses[props.tone ?? "slate"];
  const content = (
    <>
      <div className="flex items-center gap-2">
        {props.icon && (
          <span
            className={twMerge(
              "flex h-6 w-6 items-center justify-center rounded-md",
              tone.icon,
            )}
          >
            {props.icon}
          </span>
        )}
        <Eyebrow>{props.label}</Eyebrow>
      </div>
      <p className="mt-2 text-[1.3rem] font-bold leading-none tracking-tight text-slate-900">
        {props.value}
      </p>
      {props.hint && (
        <p className="mt-1.5 text-[0.68rem] font-semibold text-slate-400">
          {props.hint}
        </p>
      )}
    </>
  );

  const className = twMerge(
    "rounded-xl border bg-white px-3.5 py-3 text-left shadow-[0_1px_2px_rgba(15,23,42,0.04)]",
    props.active ? "border-slate-900" : "border-slate-200",
    props.onClick &&
      "cursor-pointer transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-[0_2px_10px_rgba(15,23,42,0.07)]",
  );

  return props.onClick ? (
    <button type="button" onClick={props.onClick} className={className}>
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
}) => (
  <div
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
      style={{ width: `${Math.min(100, Math.max(0, props.value * 100))}%` }}
    />
  </div>
);

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
}) => {
  const tone = toneClasses[props.tone ?? "slate"];
  return (
    <li className="relative flex gap-3 pb-4 last:pb-0">
      {!props.last && (
        <span className="absolute left-[13px] top-7 bottom-0 w-px bg-slate-200" />
      )}
      <span
        className={twMerge(
          "relative z-10 flex h-[27px] w-[27px] shrink-0 items-center justify-center rounded-full border-2 border-white",
          tone.icon,
        )}
      >
        {props.icon ?? <span className={twMerge("h-2 w-2 rounded-full", tone.dot)} />}
      </span>
      <div className="min-w-0 flex-1 pt-0.5">
        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
          <span className="text-[0.78rem] font-bold text-slate-800">
            {props.title}
          </span>
          {props.meta && (
            <span className="text-[0.68rem] font-semibold text-slate-400">
              {props.meta}
            </span>
          )}
        </div>
        {props.children && (
          <div className="mt-1 text-[0.74rem] font-medium leading-relaxed text-slate-500">
            {props.children}
          </div>
        )}
      </div>
    </li>
  );
};

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
      strokeWidth={2.5}
      className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
    />
    <input
      value={props.value}
      onChange={(event) => props.onChange(event.target.value)}
      placeholder={props.placeholder}
      aria-label={props.ariaLabel ?? props.placeholder}
      className="h-9 w-full rounded-md border border-slate-200 bg-white pl-9 pr-8 text-[0.78rem] font-semibold text-slate-700 shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none transition-[border-color,box-shadow] duration-150 placeholder:font-medium placeholder:text-slate-400 focus:border-slate-400 focus:shadow-[0_0_0_2px_rgba(148,163,184,0.18)]"
    />
    {props.value && (
      <button
        type="button"
        aria-label="Clear search"
        onClick={() => props.onChange("")}
        className="absolute right-2.5 top-1/2 flex h-5 w-5 -translate-y-1/2 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700"
      >
        <X size={12} strokeWidth={2.8} />
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
      onClick={props.onClick}
    >
      <RefreshCw
        size={14}
        strokeWidth={2.5}
        className={props.loading ? "animate-spin text-slate-500" : "text-slate-500"}
      />
    </ActionIcon>
  </Tooltip>
);

export const Loading = (props: { label?: string }) => (
  <div className="flex w-full items-center justify-center py-16">
    <div className="flex items-center gap-2 text-slate-400">
      <Loader2 size={16} strokeWidth={2.5} className="animate-spin" />
      <span className="text-[0.78rem] font-semibold">
        {props.label ?? "Loading..."}
      </span>
    </div>
  </div>
);

export const SkeletonRows = (props: { rows?: number }) => (
  <div className="flex flex-col gap-2">
    {Array.from({ length: props.rows ?? 4 }).map((_, index) => (
      <div
        key={index}
        className="flex items-center gap-4 rounded-xl border border-slate-200 bg-white px-4 py-3.5"
      >
        <div className="h-9 w-9 shrink-0 animate-pulse rounded-lg bg-slate-100" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-1/3 animate-pulse rounded bg-slate-100" />
          <div className="h-2.5 w-1/2 animate-pulse rounded bg-slate-50" />
        </div>
        <div className="h-5 w-16 animate-pulse rounded bg-slate-100" />
      </div>
    ))}
  </div>
);

export const EmptyState = (props: {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) => (
  <div className="flex w-full flex-col items-center justify-center px-6 py-14 text-center">
    {props.icon && (
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-slate-50 text-slate-400 ring-1 ring-slate-100">
        {props.icon}
      </div>
    )}
    <p className="text-[0.88rem] font-bold text-slate-700">{props.title}</p>
    {props.description && (
      <p className="mt-1 max-w-sm text-[0.78rem] font-medium leading-relaxed text-slate-400">
        {props.description}
      </p>
    )}
    {props.action && <div className="mt-4">{props.action}</div>}
  </div>
);

export const ErrorState = (props: {
  title?: string;
  description?: string;
  onRetry?: () => void;
}) => (
  <Card className="p-8">
    <div className="flex w-full flex-col items-center justify-center text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-red-50 text-red-500">
        <AlertCircle size={20} strokeWidth={2} />
      </div>
      <p className="text-[0.88rem] font-bold text-slate-700">
        {props.title ?? "Something went wrong"}
      </p>
      <p className="mt-1 max-w-sm text-[0.78rem] font-medium leading-relaxed text-slate-400">
        {props.description ?? "We could not load this data. Please try again."}
      </p>
      {props.onRetry && (
        <Button
          variant="default"
          className="mt-4"
          leftSection={<RefreshCw size={13} strokeWidth={2.5} />}
          onClick={props.onRetry}
        >
          Try again
        </Button>
      )}
    </div>
  </Card>
);

export const NotFoundState = (props: { title: string; description: string }) => (
  <Card className="p-8">
    <EmptyState
      icon={<Ban size={20} strokeWidth={2} />}
      title={props.title}
      description={props.description}
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
    radius="md"
    title={props.title}
    styles={{ title: { fontWeight: 700, fontSize: "0.9rem" } }}
  >
    <div className="text-[0.8rem] font-medium leading-relaxed text-slate-600">
      {props.description}
    </div>
    {props.details && (
      <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2.5">
        {props.details}
      </div>
    )}
    <div className="mt-6 flex items-center justify-end gap-2">
      <Button variant="default" onClick={props.onClose} disabled={props.loading}>
        Cancel
      </Button>
      <Button
        variant="filled"
        color={props.danger === false ? "dark" : "red"}
        loading={props.loading}
        onClick={props.onConfirm}
      >
        {props.confirmLabel}
      </Button>
    </div>
  </Modal>
);
