import { ActionIcon, Tooltip } from "@mantine/core";
import { AlertTriangle, LucideIcon, RefreshCw } from "lucide-react";
import * as React from "react";

const relativeLabel = (updatedAt: number, now: number) => {
  const seconds = Math.max(0, Math.round((now - updatedAt) / 1000));
  if (seconds < 10) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  return `${Math.round(minutes / 60)}h ago`;
};

const Freshness = (props: { updatedAt: number; isError?: boolean }) => {
  const [now, setNow] = React.useState(() => Date.now());

  React.useEffect(() => {
    const tick = () => {
      if (!document.hidden) setNow(Date.now());
    };
    const id = window.setInterval(tick, 10000);
    document.addEventListener("visibilitychange", tick);
    return () => {
      window.clearInterval(id);
      document.removeEventListener("visibilitychange", tick);
    };
  }, []);

  if (props.isError) {
    return (
      <span className="inline-flex items-center gap-1 text-micro font-semibold text-amber-700">
        <AlertTriangle size={11} strokeWidth={2.4} />
        Stale
      </span>
    );
  }

  if (!props.updatedAt) return null;

  return (
    <span
      className="text-micro font-normal tabular-nums text-slate-500"
      title={new Date(props.updatedAt).toLocaleString()}
    >
      Updated {relativeLabel(props.updatedAt, now)}
    </span>
  );
};

export const LogWidgetHeader = (props: {
  icon: LucideIcon;
  title: string;
  description: string;
  isLoading?: boolean;
  isError?: boolean;
  updatedAt?: number;
  onRefresh: () => void;
  children?: React.ReactNode;
}) => {
  const Icon = props.icon;

  return (
    <header className="flex flex-col gap-3 border-b border-slate-200 pb-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex min-w-0 items-center gap-2.5">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
          <Icon size={14} strokeWidth={2.2} />
        </span>
        <div className="min-w-0">
          <h2 className="text-body font-semibold text-slate-800">
            {props.title}
          </h2>
          <p className="truncate text-micro font-normal text-slate-500">
            {props.description}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-2 sm:justify-end">
        {(props.updatedAt || props.isError) && (
          <Freshness updatedAt={props.updatedAt ?? 0} isError={props.isError} />
        )}
        {props.children}
        <Tooltip label="Refresh data" withArrow>
          <ActionIcon
            type="button"
            onClick={props.onRefresh}
            disabled={props.isLoading}
            variant="default"
            size="sm"
            aria-label={`Refresh ${props.title.toLowerCase()}`}
          >
            <RefreshCw
              size={12}
              strokeWidth={2.5}
              className={props.isLoading ? "animate-spin" : ""}
            />
          </ActionIcon>
        </Tooltip>
      </div>
    </header>
  );
};
