import type { Timestamp } from "@/apis/google/protobuf/timestamp";
import type { Duration } from "@/apis/metav1/metav1";
import { twMerge } from "@/lib/utils";
import type { ResourceInfoIcon } from "@/pages/utils/types";
import { Button, Modal } from "@mantine/core";
import { X } from "lucide-react";
import * as React from "react";

export type OverviewTone = "neutral" | "success" | "warning" | "danger" | "info";

export const TONE_SOFT: Record<OverviewTone, string> = {
  neutral: "border-slate-200 bg-slate-50 text-slate-600",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
  info: "border-blue-200 bg-blue-50 text-blue-700",
};

export const TONE_TEXT: Record<OverviewTone, string> = {
  neutral: "text-slate-700",
  success: "text-emerald-700",
  warning: "text-amber-700",
  danger: "text-red-700",
  info: "text-blue-700",
};

export const TONE_DOT: Record<OverviewTone, string> = {
  neutral: "bg-slate-400",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  info: "bg-blue-500",
};

export const toDate = (value?: Timestamp) =>
  value
    ? new Date(Number(value.seconds) * 1000 + Math.floor(value.nanos / 1e6))
    : undefined;

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

export const formatDuration = (duration?: Duration, fallback = "Not set") => {
  const seconds = durationSeconds(duration);
  return seconds > 0 ? formatSeconds(seconds) : fallback;
};

export const remainingPercent = (start?: Date, end?: Date, now = Date.now()) => {
  if (!start || !end) return undefined;
  const total = end.getTime() - start.getTime();
  if (total <= 0) return 0;
  return Math.max(0, Math.min(100, ((end.getTime() - now) / total) * 100));
};

export const ResourceActionButton = (props: {
  icon?: ResourceInfoIcon;
  tone?: "primary" | "default" | "danger";
  children: React.ReactNode;
  onClick?: () => void;
  href?: string;
  disabled?: boolean;
  loading?: boolean;
  title?: string;
}) => {
  const { icon: Icon, tone = "default", href, ...rest } = props;
  const common = {
    size: "compact-sm",
    variant:
      tone === "primary" ? "filled" : tone === "danger" ? "outline" : "default",
    color: tone === "primary" ? "ink" : tone === "danger" ? "red.7" : undefined,
    leftSection: Icon ? <Icon size={13} strokeWidth={2.25} /> : undefined,
    ...rest,
  } as const;

  return href ? (
    <Button
      component="a"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      {...common}
    />
  ) : (
    <Button type="button" {...common} />
  );
};

export const Meter = (props: {
  value: number;
  label: string;
  tone?: OverviewTone;
  caption?: React.ReactNode;
  className?: string;
}) => (
  <div className={twMerge("flex w-full max-w-md flex-col gap-1", props.className)}>
    <span
      role="progressbar"
      aria-label={props.label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(props.value)}
      className="block h-1.5 w-full overflow-hidden rounded-full bg-slate-100"
    >
      <span
        className={twMerge(
          "block h-full rounded-full",
          TONE_DOT[props.tone ?? "success"],
        )}
        style={{ width: `${Math.max(0, Math.min(100, props.value))}%` }}
      />
    </span>
    {props.caption && (
      <span className="text-micro font-normal text-slate-500">
        {props.caption}
      </span>
    )}
  </div>
);

export const OverviewHero = (props: {
  tone: OverviewTone;
  icon: React.ReactNode;
  title: React.ReactNode;
  detail?: React.ReactNode;
  meta?: React.ReactNode;
  meter?: React.ReactNode;
  aside?: React.ReactNode;
  children?: React.ReactNode;
}) => (
  <div className="flex flex-col gap-3 border-b border-slate-200 bg-white px-4 py-4">
    <div className="flex flex-col gap-3 @2xl:flex-row @2xl:items-start">
      <div className="flex min-w-0 flex-1 items-start gap-3">
        <span
          className={twMerge(
            "flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border",
            TONE_SOFT[props.tone],
          )}
        >
          {props.icon}
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-base font-bold leading-6 text-slate-900 [overflow-wrap:anywhere]">
            {props.title}
          </span>
          {props.detail && (
            <span className="text-xs font-normal leading-5 text-slate-600 [overflow-wrap:anywhere]">
              {props.detail}
            </span>
          )}
          {props.meta && (
            <div className="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
              {props.meta}
            </div>
          )}
          {props.meter && <div className="mt-1.5">{props.meter}</div>}
        </div>
      </div>
      {props.aside && (
        <div className="flex min-w-0 shrink-0 flex-col items-start gap-1 pl-14 @2xl:max-w-[45%] @2xl:items-end @2xl:pl-0 @2xl:text-right">
          {props.aside}
        </div>
      )}
    </div>
    {props.children && <div className="flex flex-col gap-2">{props.children}</div>}
  </div>
);

export const HeroStat = (props: {
  label: string;
  value: React.ReactNode;
  tone?: OverviewTone;
}) => (
  <>
    <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
      {props.label}
    </span>
    <div
      className={twMerge(
        "text-sm font-bold tabular-nums [overflow-wrap:anywhere]",
        TONE_TEXT[props.tone ?? "neutral"],
      )}
    >
      {props.value}
    </div>
  </>
);

export const OverviewNotice = (props: {
  tone?: OverviewTone;
  icon?: ResourceInfoIcon;
  title: React.ReactNode;
  children?: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <div
      role={props.tone === "danger" ? "alert" : "status"}
      className={twMerge(
        "flex items-start gap-2.5 rounded-lg border px-3 py-2",
        TONE_SOFT[props.tone ?? "warning"],
      )}
    >
      {Icon && (
        <Icon size={14} strokeWidth={2.3} className="mt-0.5 shrink-0" />
      )}
      <div className="min-w-0 text-xs leading-5 [overflow-wrap:anywhere]">
        <span className="font-semibold">{props.title}</span>
        {props.children && (
          <span className="font-normal text-slate-700"> {props.children}</span>
        )}
      </div>
    </div>
  );
};

export const TonePill = (props: {
  tone?: OverviewTone;
  icon?: ResourceInfoIcon;
  children: React.ReactNode;
  className?: string;
}) => {
  const Icon = props.icon;
  return (
    <span
      className={twMerge(
        "inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs font-semibold leading-4 [overflow-wrap:anywhere]",
        TONE_SOFT[props.tone ?? "neutral"],
        props.className,
      )}
    >
      {Icon && <Icon size={11} strokeWidth={2.4} className="shrink-0" />}
      {props.children}
    </span>
  );
};

export const HistoryList = (props: {
  entries: {
    key: string;
    tone: OverviewTone;
    label: React.ReactNode;
    meta?: React.ReactNode;
    detail?: React.ReactNode;
  }[];
}) => (
  <ol className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
    {props.entries.map((entry) => (
      <li
        key={entry.key}
        className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2"
      >
        <span
          aria-hidden="true"
          className={twMerge("h-2 w-2 shrink-0 rounded-full", TONE_DOT[entry.tone])}
        />
        <span
          className={twMerge("text-xs font-semibold", TONE_TEXT[entry.tone])}
        >
          {entry.label}
        </span>
        {entry.meta && (
          <span className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs font-normal text-slate-500">
            {entry.meta}
          </span>
        )}
        {entry.detail && (
          <span className="basis-full pl-5 text-xs font-normal text-slate-600 [overflow-wrap:anywhere]">
            {entry.detail}
          </span>
        )}
      </li>
    ))}
  </ol>
);

export const MutedValue = (props: { children: React.ReactNode }) => (
  <span className="text-xs font-normal text-slate-500">{props.children}</span>
);

export const ActionConfirmModal = (props: {
  opened: boolean;
  onClose: () => void;
  icon: ResourceInfoIcon;
  title: string;
  subtitle?: string;
  children: React.ReactNode;
  confirmLabel: string;
  danger?: boolean;
  loading?: boolean;
  disabled?: boolean;
  onConfirm: () => void;
}) => {
  const Icon = props.icon;
  const close = () => {
    if (!props.loading) props.onClose();
  };
  return (
    <Modal
      opened={props.opened}
      onClose={close}
      centered
      size="md"
      withCloseButton={false}
      padding={0}
      styles={{ content: { borderRadius: 14, overflow: "hidden" } }}
    >
      <div className="bg-white">
        <header className="flex items-center justify-between gap-3 border-b border-slate-200 px-5 py-4">
          <div className="flex min-w-0 items-center gap-2.5">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
              <Icon size={15} />
            </span>
            <div className="min-w-0">
              <h2 className="text-sm font-bold text-slate-900">{props.title}</h2>
              {props.subtitle && (
                <p className="truncate text-xs font-normal text-slate-500">
                  {props.subtitle}
                </p>
              )}
            </div>
          </div>
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-xs"
            aria-label="Close"
            disabled={props.loading}
            onClick={close}
          >
            <X size={14} />
          </Button>
        </header>
        <div className="space-y-4 px-5 py-5 text-sm text-slate-700">
          {props.children}
        </div>
        <footer className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
          <Button
            type="button"
            variant="default"
            size="sm"
            disabled={props.loading}
            onClick={close}
          >
            Cancel
          </Button>
          <Button
            type="button"
            color={props.danger ? "red.7" : "ink"}
            size="sm"
            loading={props.loading}
            disabled={props.disabled}
            leftSection={<Icon size={13} />}
            onClick={props.onConfirm}
          >
            {props.confirmLabel}
          </Button>
        </footer>
      </div>
    </Modal>
  );
};
