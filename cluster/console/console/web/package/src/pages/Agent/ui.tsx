import { ActionIcon, Button, Modal, Tooltip } from "@mantine/core";
import { Check, Copy, Sparkles, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";

export const isMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

export const modKey = isMac ? "⌘" : "Ctrl";

export const Kbd = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <kbd
    className={`text-micro ${twMerge(
      "inline-flex h-5 min-w-5 items-center justify-center rounded-md border border-slate-200 bg-white px-1 font-semibold text-slate-500",
      props.className,
    )}`}
  >
    {props.children}
  </kbd>
);

export const Shimmer = (props: {
  children: React.ReactNode;
  className?: string;
}) => (
  <span
    className={twMerge(
      "animate-shimmer bg-[linear-gradient(90deg,var(--color-slate-500)_40%,var(--color-slate-900)_50%,var(--color-slate-500)_60%)] bg-[length:250%_100%] bg-clip-text text-transparent motion-reduce:animate-none",
      props.className,
    )}
  >
    {props.children}
  </span>
);

const dotTones = {
  green: "bg-emerald-500",
  blue: "bg-blue-500",
  amber: "bg-amber-500",
  red: "bg-red-500",
  gray: "bg-slate-400",
};

export type DotTone = keyof typeof dotTones;

export const StatusDot = (props: {
  tone: DotTone;
  pulse?: boolean;
  className?: string;
}) => (
  <span
    className={twMerge(
      "relative inline-flex h-2 w-2 shrink-0",
      props.className,
    )}
  >
    {props.pulse && (
      <span
        className={twMerge(
          "absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 motion-reduce:animate-none",
          dotTones[props.tone],
        )}
      />
    )}
    <span
      className={twMerge(
        "relative inline-flex h-2 w-2 rounded-full",
        dotTones[props.tone],
      )}
    />
  </span>
);

const escapeRegExp = (value: string) =>
  value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

export const Highlight = (props: { text: string; terms: string[] }) => {
  const parts = React.useMemo(() => {
    if (props.terms.length === 0) return [props.text];
    return props.text.split(
      new RegExp(`(${props.terms.map(escapeRegExp).join("|")})`, "gi"),
    );
  }, [props.text, props.terms]);

  return (
    <>
      {parts.map((part, idx) =>
        idx % 2 === 1 ? (
          <mark
            key={idx}
            className="rounded-[3px] bg-amber-200/70 px-px text-inherit"
          >
            {part}
          </mark>
        ) : (
          part
        ),
      )}
    </>
  );
};

export const copyText = async (value: string): Promise<boolean> => {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    toast.error("Could not copy to the clipboard");
    return false;
  }
};

export const CopyButton = (props: { value: () => string; label?: string }) => {
  const [copied, setCopied] = React.useState(false);
  const label = props.label ?? "Copy";

  React.useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1400);
    return () => window.clearTimeout(timer);
  }, [copied]);

  return (
    <Tooltip label={copied ? "Copied" : label} withArrow>
      <ActionIcon
        size="sm"
        variant="subtle"
        color="gray"
        aria-label={label}
        onClick={async () => setCopied(await copyText(props.value()))}
      >
        {copied ? (
          <Check size={13} strokeWidth={2.5} className="text-emerald-600" />
        ) : (
          <Copy size={13} />
        )}
      </ActionIcon>
    </Tooltip>
  );
};

export const ConfirmModal = (props: {
  opened: boolean;
  onClose: () => void;
  title: string;
  confirmLabel: string;
  onConfirm: () => Promise<unknown> | void;
  icon?: React.ReactNode;
  children?: React.ReactNode;
}) => {
  const [pending, setPending] = React.useState(false);

  const confirm = async () => {
    setPending(true);
    try {
      await props.onConfirm();
      props.onClose();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setPending(false);
    }
  };

  return (
    <Modal
      opened={props.opened}
      onClose={() => !pending && props.onClose()}
      centered
      size="sm"
      withCloseButton={false}
      padding={0}
      overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
      transitionProps={{ transition: "pop", duration: 200 }}
      styles={{
        content: {
          border: "1px solid var(--color-slate-200)",
          borderRadius: "14px",
          boxShadow: "var(--shadow-modal)",
          overflow: "hidden",
        },
      }}
    >
      <div className="flex flex-col">
        <div className="flex items-center gap-3 border-b border-slate-200 bg-slate-50/70 px-5 py-3.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-600 text-white shadow-sm">
            {props.icon ?? <Trash2 size={15} strokeWidth={2.25} />}
          </span>
          <span className="min-w-0 truncate text-body font-semibold text-slate-900">
            {props.title}
          </span>
        </div>
        <div className="px-5 py-4 text-body font-normal leading-relaxed text-slate-600">
          {props.children}
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-5 py-3">
          <Button
            variant="default"
            size="sm"
            disabled={pending}
            onClick={props.onClose}
          >
            Cancel
          </Button>
          <Button
            color="red"
            size="sm"
            loading={pending}
            data-autofocus
            onClick={() => void confirm()}
          >
            {props.confirmLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export const useNow = (active: boolean, intervalMs = 1000): number => {
  const [now, setNow] = React.useState(() => Date.now());

  React.useEffect(() => {
    if (!active) return;
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(timer);
  }, [active, intervalMs]);

  return now;
};

export const AgentMark = (props: {
  live?: boolean;
  size?: "sm" | "lg";
  className?: string;
}) => {
  const large = props.size === "lg";
  return (
    <span
      className={twMerge(
        "relative flex shrink-0 items-center justify-center bg-slate-900 text-white shadow-sm",
        large ? "h-11 w-11 rounded-2xl" : "h-7 w-7 rounded-lg",
        props.className,
      )}
    >
      {props.live && (
        <span className="absolute -inset-1 rounded-[11px] border border-slate-300 motion-safe:animate-pulse" />
      )}
      <Sparkles size={large ? 20 : 14} strokeWidth={2.2} />
    </span>
  );
};
