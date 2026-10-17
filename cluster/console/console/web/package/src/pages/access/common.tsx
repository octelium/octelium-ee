import * as AccessP from "@/apis/accessv1/accessv1";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import { twMerge } from "@/lib/utils";
import {
  getAPIFromAPIVersion,
  getResourcePathFromAPIKind,
  ResourceName,
} from "@/utils/pb/paths";
import { CopyButton, Tooltip } from "@mantine/core";
import {
  Ban,
  Check,
  CircleCheck,
  Copy,
  LucideIcon,
  ShieldQuestion,
  Users,
  Webhook,
} from "lucide-react";
import * as React from "react";
import { Link, useLocation } from "react-router-dom";

export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

export const TONE_SOFT: Record<Tone, string> = {
  neutral: "border-slate-200 bg-slate-50 text-slate-600",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
  info: "border-blue-200 bg-blue-50 text-blue-700",
};

export const TONE_TEXT: Record<Tone, string> = {
  neutral: "text-slate-600",
  success: "text-emerald-700",
  warning: "text-amber-700",
  danger: "text-red-700",
  info: "text-blue-700",
};

export const TONE_DOT: Record<Tone, string> = {
  neutral: "bg-slate-400",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  info: "bg-blue-500",
};

export type EffectKey = "AUTO_APPROVE" | "REVIEW" | "DENY";

export const EFFECT_META: Record<
  EffectKey,
  {
    value: AccessP.Policy_Spec_Rule_Effect;
    label: string;
    short: string;
    description: string;
    useWhen: string;
    tone: Tone;
    icon: LucideIcon;
  }
> = {
  AUTO_APPROVE: {
    value: AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE,
    label: "Auto-approve",
    short: "Auto-approved",
    description: "Grant the access immediately, without any reviewer.",
    useWhen:
      "Low-risk access, or trusted people such as the on-call engineer during their shift.",
    tone: "success",
    icon: CircleCheck,
  },
  REVIEW: {
    value: AccessP.Policy_Spec_Rule_Effect.REVIEW,
    label: "Require review",
    short: "Reviewed",
    description:
      "Route the request through one or more review steps before the access is granted.",
    useWhen:
      "Sensitive resources that need a human decision, optionally from several teams in order.",
    tone: "warning",
    icon: Users,
  },
  DENY: {
    value: AccessP.Policy_Spec_Rule_Effect.DENY,
    label: "Deny",
    short: "Denied",
    description: "Reject the request immediately. Nobody can approve it.",
    useWhen:
      "Guardrails that must always win, such as contractors and production data. Give them the highest priority.",
    tone: "danger",
    icon: Ban,
  },
};

export const getEffectKey = (
  effect?: AccessP.Policy_Spec_Rule_Effect,
): EffectKey | undefined => {
  switch (effect) {
    case AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE:
      return "AUTO_APPROVE";
    case AccessP.Policy_Spec_Rule_Effect.REVIEW:
      return "REVIEW";
    case AccessP.Policy_Spec_Rule_Effect.DENY:
      return "DENY";
    default:
      return undefined;
  }
};

export const EffectBadge = (props: {
  effect?: AccessP.Policy_Spec_Rule_Effect;
  className?: string;
}) => {
  const key = getEffectKey(props.effect);
  const meta = key ? EFFECT_META[key] : undefined;
  const Icon = meta?.icon ?? ShieldQuestion;

  return (
    <span
      className={twMerge(
        "inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-micro font-semibold",
        TONE_SOFT[meta?.tone ?? "neutral"],
        props.className,
      )}
    >
      <Icon size={11} strokeWidth={2.4} aria-hidden="true" />
      {meta?.label ?? "No effect"}
    </span>
  );
};

export const ToneBadge = (props: {
  tone: Tone;
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-micro font-semibold",
      TONE_SOFT[props.tone],
      props.className,
    )}
  >
    <span
      aria-hidden="true"
      className={twMerge("h-1.5 w-1.5 rounded-full", TONE_DOT[props.tone])}
    />
    {props.children}
  </span>
);

export type ProviderKind = "slack" | "jira" | "webhook";

export const ProviderMark = (props: {
  provider?: ProviderKind;
  size?: number;
  className?: string;
}) => {
  const size = props.size ?? 18;

  if (props.provider === "slack") {
    return (
      <svg
        viewBox="0 0 24 24"
        width={size}
        height={size}
        aria-hidden="true"
        className={props.className}
      >
        <path
          fill="#E01E5A"
          d="M5.04 15.17a2.52 2.52 0 1 1-2.52-2.52h2.52v2.52Zm1.27 0a2.52 2.52 0 0 1 5.04 0v6.31a2.52 2.52 0 1 1-5.04 0v-6.31Z"
        />
        <path
          fill="#36C5F0"
          d="M8.83 5.04a2.52 2.52 0 1 1 2.52-2.52v2.52H8.83Zm0 1.27a2.52 2.52 0 0 1 0 5.04H2.52a2.52 2.52 0 0 1 0-5.04h6.31Z"
        />
        <path
          fill="#2EB67D"
          d="M18.96 8.83a2.52 2.52 0 1 1 2.52 2.52h-2.52V8.83Zm-1.27 0a2.52 2.52 0 0 1-5.04 0V2.52a2.52 2.52 0 1 1 5.04 0v6.31Z"
        />
        <path
          fill="#ECB22E"
          d="M15.17 18.96a2.52 2.52 0 1 1-2.52 2.52v-2.52h2.52Zm0-1.27a2.52 2.52 0 0 1 0-5.04h6.31a2.52 2.52 0 1 1 0 5.04h-6.31Z"
        />
      </svg>
    );
  }

  if (props.provider === "jira") {
    return (
      <svg
        viewBox="0 0 24 24"
        width={size}
        height={size}
        aria-hidden="true"
        className={props.className}
      >
        <path
          fill="#2684FF"
          d="M22.16 11.1 13.7 2.64 12.89 1.8 6.51 8.2l-2.92 2.9a.78.78 0 0 0 0 1.1l5.82 5.82 3.47 3.46 6.37-6.37.1-.1 2.81-2.81a.78.78 0 0 0 0-1.1ZM12.89 14.6l-2.9-2.9 2.9-2.9 2.9 2.9-2.9 2.9Z"
        />
        <path
          fill="#0052CC"
          d="M12.89 8.8A4.88 4.88 0 0 1 12.87 1.9L6.5 8.26l3.46 3.46 2.93-2.92Z"
          opacity="0.9"
        />
        <path
          fill="#2684FF"
          d="m15.8 11.69-2.91 2.91a4.88 4.88 0 0 1 0 6.9l6.38-6.37-3.47-3.44Z"
          opacity="0.9"
        />
      </svg>
    );
  }

  return (
    <Webhook
      size={size}
      strokeWidth={2.1}
      aria-hidden="true"
      className={twMerge("text-slate-600", props.className)}
    />
  );
};

export const ProviderTile = (props: {
  provider?: ProviderKind;
  size?: "sm" | "md" | "lg";
  className?: string;
}) => {
  const size = props.size ?? "md";
  return (
    <span
      className={twMerge(
        "flex shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white shadow-card",
        size === "sm" && "h-7 w-7",
        size === "md" && "h-9 w-9",
        size === "lg" && "h-11 w-11 rounded-xl",
        props.className,
      )}
    >
      <ProviderMark
        provider={props.provider}
        size={size === "sm" ? 14 : size === "md" ? 18 : 22}
      />
    </span>
  );
};

export const getProviderKind = (
  item?: AccessP.Integration,
): ProviderKind | undefined => {
  switch (item?.spec?.type?.oneofKind) {
    case "slack":
      return "slack";
    case "jira":
      return "jira";
    case "webhook":
      return "webhook";
    default:
      return undefined;
  }
};

export const getProviderKindFromStatus = (
  type?: AccessP.Integration_Status_Type,
): ProviderKind | undefined => {
  switch (type) {
    case AccessP.Integration_Status_Type.SLACK:
      return "slack";
    case AccessP.Integration_Status_Type.JIRA:
      return "jira";
    case AccessP.Integration_Status_Type.WEBHOOK:
      return "webhook";
    default:
      return undefined;
  }
};

export const PROVIDER_LABEL: Record<ProviderKind, string> = {
  slack: "Slack",
  jira: "Jira",
  webhook: "Webhook",
};

export const normalizeRef = (
  ref: ObjectReference | undefined,
  apiVersion: string,
  kind: string,
): ObjectReference | undefined =>
  ref && (ref.name || ref.uid)
    ? ObjectReference.create({
        ...ref,
        apiVersion: ref.apiVersion || apiVersion,
        kind: ref.kind || kind,
      })
    : undefined;

export const getRefPath = (ref?: ObjectReference) => {
  if (!ref?.name || !ref.apiVersion || !ref.kind) return undefined;
  const api = getAPIFromAPIVersion(ref.apiVersion);
  if (!api) return undefined;
  const kindPath = getResourcePathFromAPIKind({
    api,
    kind: ref.kind as ResourceName,
  });
  return kindPath ? `/${api}/${kindPath}/${ref.name}` : undefined;
};

const initials = (value: string) =>
  value
    .split(/[\s._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join("") || "?";

export const Avatar = (props: {
  name: string;
  src?: string;
  size?: "xs" | "sm" | "md";
  className?: string;
}) => {
  const [failed, setFailed] = React.useState(false);
  const size = props.size ?? "sm";
  const dimension =
    size === "xs" ? "h-5 w-5 text-[9px]" : size === "sm" ? "h-6 w-6 text-micro" : "h-8 w-8 text-xs";

  if (props.src && !failed) {
    return (
      <img
        src={props.src}
        alt=""
        referrerPolicy="no-referrer"
        loading="lazy"
        onError={() => setFailed(true)}
        className={twMerge(
          "shrink-0 rounded-full border border-slate-200 bg-slate-100 object-cover",
          dimension,
          props.className,
        )}
      />
    );
  }

  return (
    <span
      aria-hidden="true"
      className={twMerge(
        "flex shrink-0 items-center justify-center rounded-full border border-slate-200 bg-slate-100 font-semibold text-slate-600",
        dimension,
        props.className,
      )}
    >
      {initials(props.name)}
    </span>
  );
};

export const RefChip = (props: {
  itemRef?: ObjectReference;
  label?: React.ReactNode;
  icon?: LucideIcon;
  avatar?: { name: string; src?: string };
  muted?: boolean;
  className?: string;
}) => {
  const location = useLocation();
  const path = getRefPath(props.itemRef);
  const Icon = props.icon;
  const text = props.label ?? props.itemRef?.name ?? props.itemRef?.uid;
  if (!text) return null;

  const content = (
    <>
      {props.avatar ? (
        <Avatar name={props.avatar.name} src={props.avatar.src} size="xs" />
      ) : (
        Icon && (
          <Icon
            size={12}
            strokeWidth={2.3}
            aria-hidden="true"
            className="shrink-0 text-slate-500"
          />
        )
      )}
      <span className="min-w-0 truncate">{text}</span>
    </>
  );

  const className = twMerge(
    "inline-flex max-w-full items-center gap-1.5 rounded-full border border-slate-200 bg-white py-0.5 text-xs font-medium text-slate-700",
    props.avatar ? "pl-0.5 pr-2" : "px-2",
    props.muted && "bg-slate-50 text-slate-600",
    props.className,
  );

  if (!path) return <span className={className}>{content}</span>;

  return (
    <Link
      to={path}
      state={{ returnTo: `${location.pathname}${location.search}` }}
      preventScrollReset
      className={twMerge(
        className,
        "outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {content}
    </Link>
  );
};

export const durationSeconds = (duration?: Duration): number => {
  const type = duration?.type;
  switch (type?.oneofKind) {
    case "milliseconds":
      return type.milliseconds / 1000;
    case "seconds":
      return type.seconds;
    case "minutes":
      return type.minutes * 60;
    case "hours":
      return type.hours * 3600;
    case "days":
      return type.days * 86400;
    case "weeks":
      return type.weeks * 7 * 86400;
    case "months":
      return type.months * 30 * 86400;
    default:
      return 0;
  }
};

export const formatSeconds = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) return "0s";
  if (value < 60) return `${Math.round(value)}s`;

  const minutes = Math.round(value / 60);
  if (minutes < 60) return `${minutes}m`;

  const hours = Math.floor(minutes / 60);
  const restMinutes = minutes % 60;
  if (hours < 24) return restMinutes ? `${hours}h ${restMinutes}m` : `${hours}h`;

  const days = Math.floor(hours / 24);
  const restHours = hours % 24;
  return restHours ? `${days}d ${restHours}h` : `${days}d`;
};

export const formatDurationShort = (duration?: Duration, fallback = "—") => {
  const seconds = durationSeconds(duration);
  return seconds > 0 ? formatSeconds(seconds) : fallback;
};

export const CopyField = (props: {
  label: string;
  value: string;
  hint?: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2",
      props.className,
    )}
  >
    <div className="flex items-center justify-between gap-2">
      <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
        {props.label}
      </span>
      <CopyButton value={props.value}>
        {({ copied, copy }) => (
          <Tooltip label={copied ? "Copied" : "Copy"} withArrow>
            <button
              type="button"
              onClick={copy}
              aria-label={`Copy ${props.label}`}
              className="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-md text-slate-500 outline-none transition-colors duration-150 hover:bg-white hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              {copied ? (
                <Check size={13} strokeWidth={2.5} />
              ) : (
                <Copy size={13} strokeWidth={2.2} />
              )}
            </button>
          </Tooltip>
        )}
      </CopyButton>
    </div>
    <code className="break-all font-mono text-xs leading-5 text-slate-800">
      {props.value}
    </code>
    {props.hint && (
      <span className="text-micro font-normal leading-4 text-slate-500">
        {props.hint}
      </span>
    )}
  </div>
);

export const InlineCopy = (props: { value?: string; className?: string }) =>
  props.value ? (
    <span
      className={twMerge(
        "inline-flex max-w-full items-center gap-1.5 break-all font-mono text-xs text-slate-700",
        props.className,
      )}
    >
      <CopyText value={props.value} />
    </span>
  ) : null;

export const EmptyNote = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <p
    className={twMerge(
      "rounded-lg border border-dashed border-slate-200 bg-slate-50/60 px-3 py-2.5 text-xs font-normal text-slate-500",
      props.className,
    )}
  >
    {props.children}
  </p>
);

export const Callout = (props: {
  tone?: Tone;
  icon?: LucideIcon;
  title?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
}) => {
  const tone = props.tone ?? "info";
  const Icon = props.icon;
  return (
    <div
      className={twMerge(
        "flex items-start gap-2.5 rounded-lg border px-3 py-2.5",
        TONE_SOFT[tone],
        props.className,
      )}
    >
      {Icon && (
        <Icon
          size={14}
          strokeWidth={2.3}
          aria-hidden="true"
          className="mt-0.5 shrink-0"
        />
      )}
      <div className="min-w-0 text-xs leading-5">
        {props.title && <div className="font-semibold">{props.title}</div>}
        {props.children && (
          <div className="font-normal text-slate-700">{props.children}</div>
        )}
      </div>
    </div>
  );
};
