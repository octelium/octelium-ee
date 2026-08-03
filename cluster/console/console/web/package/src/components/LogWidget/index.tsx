import { ActionIcon, Tooltip } from "@mantine/core";
import { LucideIcon, RefreshCw } from "lucide-react";
import * as React from "react";

export const LogWidgetHeader = (props: {
  icon: LucideIcon;
  title: string;
  description: string;
  isLoading?: boolean;
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
          <h2 className="text-[0.78rem] font-bold text-slate-800">
            {props.title}
          </h2>
          <p className="truncate text-[0.65rem] font-semibold text-slate-400">
            {props.description}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-2 sm:justify-end">
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
