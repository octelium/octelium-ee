import { ActionIcon, Button, Menu, Tooltip } from "@mantine/core";
import { Sparkles } from "lucide-react";
import { twMerge } from "@/lib/utils";
import {
  useAgentAsk,
  useOptionalAgent,
  type AgentActivity,
  type PinnedContext,
} from "./context";

const isMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

const statusLabels: Record<string, string> = {
  setup: "Set up",
  stopped: "Stopped",
  starting: "Starting",
};

const ActivityDot = (props: { activity?: AgentActivity }) =>
  props.activity ? (
    <span className="absolute -right-1 -top-1 flex h-2.5 w-2.5">
      <span
        className={twMerge(
          "absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 motion-reduce:animate-none",
          props.activity === "approval" ? "bg-amber-500" : "bg-blue-500",
        )}
      />
      <span
        className={twMerge(
          "relative inline-flex h-2.5 w-2.5 rounded-full ring-2 ring-white",
          props.activity === "approval" ? "bg-amber-500" : "bg-blue-500",
        )}
      />
    </span>
  ) : null;

export const AgentHeaderButton = () => {
  const agent = useOptionalAgent();
  if (
    !agent ||
    agent.hidden ||
    agent.status === "loading" ||
    agent.status === "unavailable"
  ) {
    return null;
  }

  const ready = agent.status === "ready";
  const statusLabel = statusLabels[agent.status];
  const label = ready
    ? "Ask the agent about this page"
    : `Agent: ${statusLabel}`;

  return (
    <Tooltip
      label={`${agent.opened ? "Close the agent" : label} (${isMac ? "⌘" : "Ctrl+"}J)`}
      withArrow
      openDelay={400}
    >
      <button
        type="button"
        aria-label={label}
        aria-pressed={agent.opened}
        aria-keyshortcuts={isMac ? "Meta+J" : "Control+J"}
        onClick={agent.toggle}
        className={twMerge(
          "group flex h-[30px] shrink-0 cursor-pointer items-center gap-2 rounded-lg border pl-1.5 pr-1.5 outline-none transition-[border-color,background-color,box-shadow,color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 sm:pr-2",
          agent.opened
            ? "border-slate-300 bg-slate-100 text-slate-900"
            : ready
              ? "border-slate-200 bg-white text-slate-700 shadow-card hover:border-slate-300 hover:text-slate-900 hover:shadow-raised"
              : "border-dashed border-slate-300 bg-transparent text-slate-500 hover:border-slate-400 hover:text-slate-700",
        )}
      >
        <span
          className={twMerge(
            "relative flex h-5 w-5 shrink-0 items-center justify-center rounded-md transition-colors duration-150",
            ready
              ? "bg-slate-900 text-white"
              : "bg-slate-200 text-slate-500 group-hover:text-slate-700",
          )}
        >
          <Sparkles size={11} strokeWidth={2.4} />
          {!agent.opened && <ActivityDot activity={agent.activity} />}
        </span>
        <span className="hidden whitespace-nowrap text-xs font-semibold sm:inline">
          {ready ? "Ask agent" : "Agent"}
        </span>
        {statusLabel && (
          <span className="hidden whitespace-nowrap text-micro font-semibold text-slate-400 md:inline">
            {statusLabel}
          </span>
        )}
        {ready && (
          <kbd className="text-micro hidden h-[18px] items-center rounded border border-slate-200 bg-slate-50 px-1 font-semibold text-slate-400 lg:inline-flex">
            {isMac ? "⌘J" : "Ctrl J"}
          </kbd>
        )}
      </button>
    </Tooltip>
  );
};

export const AskAgentMenuItem = (props: {
  pinned: PinnedContext;
  label?: string;
}) => {
  const { available, ask } = useAgentAsk();
  if (!available) return null;
  return (
    <Menu.Item
      leftSection={<Sparkles size={14} />}
      onClick={() => ask({ pinned: props.pinned, fresh: true })}
    >
      {props.label ?? "Ask the agent"}
    </Menu.Item>
  );
};

export const AskAgentButton = (props: {
  pinned?: () => PinnedContext;
  label?: string;
  variant?: "outline" | "icon" | "white";
}) => {
  const { available, ask } = useAgentAsk();
  if (!available) return null;
  const onClick = () => ask({ pinned: props.pinned?.(), fresh: true });
  const label = props.label ?? "Ask agent";

  if (props.variant === "icon") {
    return (
      <Tooltip label={label} withArrow openDelay={300}>
        <ActionIcon
          variant="subtle"
          color="gray"
          aria-label={label}
          onClick={onClick}
        >
          <Sparkles size={15} />
        </ActionIcon>
      </Tooltip>
    );
  }

  return (
    <Button
      type="button"
      size="compact-xs"
      variant={props.variant === "white" ? "white" : "outline"}
      color={props.variant === "white" ? "dark" : undefined}
      leftSection={<Sparkles size={11} strokeWidth={2.5} />}
      onClick={onClick}
      styles={{ root: { fontSize: "0.68rem", fontWeight: 700 } }}
    >
      {label}
    </Button>
  );
};
