import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import {
  formatCount,
  formatDuration,
  formatElapsed,
} from "@/pages/Agent/format";
import { AgentMark, CopyButton, Shimmer, useNow } from "@/pages/Agent/ui";
import { ActionIcon, Tooltip } from "@mantine/core";
import { motion } from "framer-motion";
import {
  Bot,
  CircleAlert,
  DatabaseZap,
  Paperclip,
  RotateCcw,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";
import Markdown from "./Markdown";
import { messageText } from "./protocols";
import Steps, { Step } from "./Steps";
import { ChatMessage, Usage } from "./types";

type Group =
  | { type: "steps"; id: string; steps: Step[] }
  | { type: "text"; id: string; text: string };

const longUserText = 700;

const truncatedReasons = new Set([
  "length",
  "max_tokens",
  "max_output_tokens",
  "MAX_TOKENS",
  "incomplete",
]);

const filteredReasons = new Set([
  "content_filter",
  "SAFETY",
  "RECITATION",
  "PROHIBITED_CONTENT",
  "refusal",
  "guardrail_intervened",
  "content_filtered",
]);

const finishNote = (reason?: string): string | undefined => {
  if (!reason) return undefined;
  if (truncatedReasons.has(reason)) return "Truncated at the output limit";
  if (filteredReasons.has(reason)) return "Filtered";
  return undefined;
};

const groupParts = (message: ChatMessage) => {
  const groups: Group[] = [];

  message.parts.forEach((part, idx) => {
    const id = `${message.id}-${idx}`;
    if (part.kind === "text") {
      groups.push({ type: "text", id, text: part.text });
      return;
    }

    const end = message.parts[idx + 1]?.at ?? message.completedAt;
    const duration =
      end !== undefined ? end - (part.at ?? message.createdAt) : undefined;
    const step: Step =
      part.kind === "thinking"
        ? { kind: "thinking", id, text: part.text, duration }
        : { kind: "tool", id, name: part.name, input: part.input, duration };

    const last = groups.at(-1);
    if (last?.type === "steps") {
      last.steps.push(step);
    } else {
      groups.push({ type: "steps", id, steps: [step] });
    }
  });

  return groups;
};

const totalTokens = (usage: Usage) =>
  usage.totalTokens ?? (usage.inputTokens ?? 0) + (usage.outputTokens ?? 0);

const Typing = () => (
  <span
    className="inline-flex items-center gap-1 py-1 text-slate-400"
    aria-label="Writing"
  >
    <i className="h-1.5 w-1.5 rounded-full bg-current motion-safe:animate-pulse" />
    <i className="h-1.5 w-1.5 rounded-full bg-current motion-safe:animate-pulse [animation-delay:150ms]" />
    <i className="h-1.5 w-1.5 rounded-full bg-current motion-safe:animate-pulse [animation-delay:300ms]" />
  </span>
);

const Waiting = (props: { since: number }) => {
  const now = useNow(true);

  return (
    <p
      className="flex h-7 min-w-0 items-center gap-2 text-xs font-semibold"
      aria-live="polite"
    >
      <Shimmer className="min-w-0 truncate">Waiting for the model</Shimmer>
      <span className="shrink-0 font-mono text-micro font-normal text-slate-400">
        {formatElapsed(now - props.since)}
      </span>
    </p>
  );
};

const UsageInfo = (props: { usage: Usage }) => {
  const { usage } = props;
  const rows = (
    [
      ["Input", usage.inputTokens],
      ["Output", usage.outputTokens],
      ["Reasoning", usage.reasoningTokens],
      ["Cache read", usage.cacheReadTokens],
      ["Cache write", usage.cacheWriteTokens],
    ] as [string, number | undefined][]
  ).filter((row): row is [string, number] => !!row[1]);

  return (
    <Tooltip
      withArrow
      multiline
      label={
        <span className="block text-micro leading-5">
          {rows.map(([label, value]) => (
            <span key={label} className="block">
              {label}: {value.toLocaleString()}
            </span>
          ))}
        </span>
      }
    >
      <span className="cursor-default tabular-nums">
        {formatCount(totalTokens(usage))} tokens
      </span>
    </Tooltip>
  );
};

const LatencyInfo = (props: { latencyMs: number; ttftMs?: number }) => (
  <Tooltip
    withArrow
    multiline
    label={
      <span className="block text-micro leading-5">
        {props.ttftMs !== undefined && (
          <span className="block">
            First token: {formatDuration(props.ttftMs)}
          </span>
        )}
        <span className="block">Total: {formatDuration(props.latencyMs)}</span>
      </span>
    }
  >
    <span className="cursor-default tabular-nums">
      {formatDuration(props.latencyMs)}
    </span>
  </Tooltip>
);

const UserMessage = (props: { message: ChatMessage }) => {
  const { message } = props;
  const [expanded, setExpanded] = React.useState(false);
  const text = messageText(message);
  const isLong = text.length > longUserText || text.split("\n").length > 12;

  return (
    <div className="group/msg flex w-full flex-col items-end gap-1.5">
      {message.attachments.length > 0 && (
        <div className="flex max-w-[85%] flex-wrap justify-end gap-1.5">
          {message.attachments.map((attachment) =>
            attachment.mime.startsWith("image/") ? (
              <img
                key={attachment.id}
                src={attachment.dataURL}
                alt={attachment.name}
                className="h-20 w-20 rounded-xl border border-slate-200 object-cover shadow-card sm:h-24 sm:w-24"
              />
            ) : (
              <span
                key={attachment.id}
                className="inline-flex max-w-[240px] items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-2.5 py-1.5 text-micro font-semibold text-slate-700 shadow-card"
              >
                <Paperclip size={12} className="shrink-0 text-slate-500" />
                <span className="truncate">{attachment.name}</span>
              </span>
            ),
          )}
        </div>
      )}
      {text && (
        <div className="relative max-w-[min(85%,40rem)] rounded-2xl rounded-br-md bg-slate-100 px-4 py-2.5">
          <p
            className={`text-body ${twMerge(
              "whitespace-pre-wrap break-words leading-6 text-slate-800",
              isLong && !expanded && "max-h-60 overflow-hidden",
            )}`}
          >
            {text}
          </p>
          {isLong && (
            <div
              className={twMerge(
                "flex justify-end",
                !expanded &&
                  "absolute inset-x-0 bottom-0 h-16 items-end rounded-b-2xl bg-gradient-to-t from-slate-100 via-slate-100/90 to-transparent px-4 pb-2",
              )}
            >
              <button
                type="button"
                onClick={() => setExpanded((value) => !value)}
                className="cursor-pointer text-slate-600 hover:text-slate-900"
              >
                <span className="text-micro font-semibold">
                  {expanded ? "Show less" : "Show more"}
                </span>
              </button>
            </div>
          )}
        </div>
      )}
      <div className="flex h-6 items-center gap-1 text-micro text-slate-400 opacity-0 transition-opacity duration-150 group-hover/msg:opacity-100 focus-within:opacity-100 [@media(hover:none)]:opacity-100">
        <TimeAgo rfc3339={Timestamp.fromDate(new Date(message.createdAt))} />
        {text && <CopyButton value={() => text} label="Copy message" />}
      </div>
    </div>
  );
};

const AssistantMessage = (props: {
  message: ChatMessage;
  streaming: boolean;
  isLast: boolean;
  onRegenerate?: () => void;
}) => {
  const { message, streaming } = props;
  const groups = React.useMemo(() => groupParts(message), [message]);
  const text = streaming ? "" : messageText(message);
  const last = message.parts.at(-1);
  const note = finishNote(message.finishReason);

  return (
    <div className="group/msg flex w-full gap-3">
      <AgentMark
        icon={Bot}
        live={streaming}
        className="mt-0.5 hidden sm:flex"
      />
      <div className="min-w-0 flex-1">
        {groups.map((group, idx) =>
          group.type === "steps" ? (
            <Steps
              key={group.id}
              steps={group.steps}
              live={streaming && idx === groups.length - 1}
            />
          ) : (
            <div key={group.id} className="text-body break-words">
              <Markdown>{group.text}</Markdown>
            </div>
          ),
        )}
        {streaming && !last && <Waiting since={message.createdAt} />}
        {streaming && last?.kind === "text" && <Typing />}
        {message.error && (
          <div
            role="alert"
            className="my-2 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold leading-5 text-red-700"
          >
            <CircleAlert size={14} className="mt-0.5 shrink-0" />
            <span className="min-w-0 break-words">{message.error}</span>
          </div>
        )}
        {!streaming && message.stopped && (
          <p className="mt-2 flex items-start gap-1.5 text-xs font-semibold text-slate-500">
            <CircleAlert size={13} className="mt-0.5 shrink-0" />
            Stopped
          </p>
        )}
        {!streaming && (
          <div
            className={`text-micro ${twMerge(
              "-ml-1 mt-1.5 flex h-7 items-center gap-0.5 text-slate-400 transition-opacity duration-150 focus-within:opacity-100 [@media(hover:none)]:opacity-100",
              props.isLast
                ? "opacity-100"
                : "opacity-0 group-hover/msg:opacity-100",
            )}`}
          >
            {text && <CopyButton value={() => text} label="Copy response" />}
            {props.onRegenerate && (
              <Tooltip label="Regenerate" withArrow>
                <ActionIcon
                  size="sm"
                  variant="subtle"
                  color="gray"
                  aria-label="Regenerate"
                  onClick={props.onRegenerate}
                >
                  <RotateCcw size={13} />
                </ActionIcon>
              </Tooltip>
            )}
            <span className="ml-1.5 flex min-w-0 items-center gap-1.5 truncate">
              {message.model && (
                <>
                  <span className="truncate">{message.model}</span>
                  <span aria-hidden>·</span>
                </>
              )}
              {message.cached && (
                <>
                  <span className="inline-flex shrink-0 items-center gap-1 text-emerald-700">
                    <DatabaseZap size={11} />
                    Cached
                  </span>
                  <span aria-hidden>·</span>
                </>
              )}
              {message.latencyMs !== undefined ? (
                <LatencyInfo
                  latencyMs={message.latencyMs}
                  ttftMs={message.ttftMs}
                />
              ) : (
                <TimeAgo
                  rfc3339={Timestamp.fromDate(new Date(message.createdAt))}
                />
              )}
              {message.usage && totalTokens(message.usage) > 0 && (
                <>
                  <span aria-hidden>·</span>
                  <UsageInfo usage={message.usage} />
                </>
              )}
              {note && (
                <>
                  <span aria-hidden>·</span>
                  <span
                    className="shrink-0 font-semibold text-amber-700"
                    title={message.finishReason}
                  >
                    {note}
                  </span>
                </>
              )}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

const Message = React.memo(
  (props: {
    message: ChatMessage;
    streaming: boolean;
    isLast: boolean;
    onRegenerate?: () => void;
  }) => (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
      data-message-id={props.message.id}
    >
      {props.message.role === "user" ? (
        <UserMessage message={props.message} />
      ) : (
        <AssistantMessage
          message={props.message}
          streaming={props.streaming}
          isLast={props.isLast}
          onRegenerate={props.onRegenerate}
        />
      )}
    </motion.div>
  ),
);

export default Message;
