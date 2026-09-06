import { ActionIcon, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  Bot,
  Brain,
  Check,
  ChevronDown,
  CircleAlert,
  Copy,
  DatabaseZap,
  RefreshCw,
  User,
  Wrench,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import Markdown from "./Markdown";
import { messageText } from "./protocols";
import { ChatMessage, Usage } from "./types";

const Typing = () => (
  <span className="inline-flex items-center gap-1 py-1 text-slate-400">
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:150ms]" />
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:300ms]" />
  </span>
);

const Meta = (props: { children: React.ReactNode; title?: string }) => (
  <span
    title={props.title}
    className="inline-flex items-center gap-1 rounded border border-slate-200 bg-white px-1.5 py-0.5 text-[0.6rem] font-bold text-slate-500"
  >
    {props.children}
  </span>
);

const usageSummary = (usage?: Usage): string[] => {
  if (!usage) return [];
  const ret: string[] = [];
  if (usage.inputTokens !== undefined) ret.push(`${usage.inputTokens} in`);
  if (usage.outputTokens !== undefined) ret.push(`${usage.outputTokens} out`);
  if (usage.reasoningTokens) ret.push(`${usage.reasoningTokens} reasoning`);
  if (usage.cacheReadTokens) ret.push(`${usage.cacheReadTokens} cache read`);
  if (usage.cacheWriteTokens) ret.push(`${usage.cacheWriteTokens} cache write`);
  return ret;
};

const Thinking = (props: { text: string; streaming: boolean }) => {
  const [opened, setOpened] = React.useState(false);

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-violet-200 bg-violet-50/60">
      <button
        type="button"
        onClick={() => setOpened((value) => !value)}
        aria-expanded={opened}
        className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left outline-none transition-colors duration-300 hover:bg-violet-100/50"
      >
        <Brain size={13} strokeWidth={2.4} className="shrink-0 text-violet-600" />
        <span className="flex-1 text-[0.68rem] font-bold uppercase tracking-[0.06em] text-violet-700">
          {props.streaming ? "Thinking…" : "Reasoning"}
        </span>
        <span className="text-[0.62rem] font-semibold text-violet-500">
          {props.text.length.toLocaleString()} chars
        </span>
        <ChevronDown
          size={13}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-violet-500 transition-transform duration-400",
            opened && "rotate-180",
          )}
        />
      </button>
      <AnimatePresence initial={false}>
        {opened && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden border-t border-violet-200"
          >
            <div className="max-h-[320px] overflow-y-auto px-3 py-2.5">
              <p className="whitespace-pre-wrap text-[0.76rem] leading-6 text-violet-900/80">
                {props.text}
              </p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const ToolCall = (props: { name: string; input: string }) => (
  <div className="my-2 overflow-hidden rounded-xl border border-amber-200 bg-amber-50/60">
    <div className="flex items-center gap-2 border-b border-amber-200 px-3 py-1.5">
      <Wrench size={12} strokeWidth={2.5} className="shrink-0 text-amber-600" />
      <span className="truncate font-mono text-[0.7rem] font-bold text-amber-800">
        {props.name || "tool call"}
      </span>
    </div>
    <pre className="max-h-[240px] overflow-auto px-3 py-2">
      <code className="font-mono text-[0.72rem] leading-5 text-amber-900/80">
        {props.input || "{}"}
      </code>
    </pre>
  </div>
);

const Message = (props: {
  message: ChatMessage;
  streaming: boolean;
  onRegenerate?: () => void;
}) => {
  const { message } = props;
  const [copied, setCopied] = React.useState(false);
  const isUser = message.role === "user";
  const text = messageText(message);
  const isEmpty = message.parts.length === 0 && !message.error;

  const copy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className={twMerge(
        "flex w-full gap-2.5",
        isUser ? "justify-end" : "justify-start",
      )}
    >
      {!isUser && (
        <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
          <Bot size={14} strokeWidth={2.2} />
        </span>
      )}

      <div
        className={twMerge(
          "flex min-w-0 max-w-[88%] flex-col gap-1",
          isUser && "items-end",
        )}
      >
        <div
          className={twMerge(
            "min-w-0 overflow-hidden rounded-2xl px-3.5 py-2.5",
            isUser
              ? "rounded-br-md bg-slate-900 text-white"
              : "rounded-bl-md border border-slate-200 bg-white",
          )}
        >
          {message.attachments.length > 0 && (
            <div className="mb-2 flex flex-wrap gap-1.5">
              {message.attachments.map((attachment) =>
                attachment.mime.startsWith("image/") ? (
                  <img
                    key={attachment.id}
                    src={attachment.dataURL}
                    alt={attachment.name}
                    className="h-20 w-20 rounded-lg border border-white/20 object-cover"
                  />
                ) : (
                  <span
                    key={attachment.id}
                    className="rounded-lg bg-white/15 px-2 py-1 text-[0.65rem] font-semibold"
                  >
                    {attachment.name}
                  </span>
                ),
              )}
            </div>
          )}

          {isUser ? (
            <p className="whitespace-pre-wrap text-[0.82rem] leading-6">
              {text}
            </p>
          ) : isEmpty && props.streaming ? (
            <Typing />
          ) : (
            <div className="min-w-0">
              {message.parts.map((part, index) => {
                if (part.kind === "thinking") {
                  return (
                    <Thinking
                      key={`thinking-${index}`}
                      text={part.text}
                      streaming={
                        props.streaming && index === message.parts.length - 1
                      }
                    />
                  );
                }
                if (part.kind === "tool") {
                  return (
                    <ToolCall
                      key={`tool-${index}`}
                      name={part.name}
                      input={part.input}
                    />
                  );
                }
                return (
                  <Markdown key={`text-${index}`}>{part.text}</Markdown>
                );
              })}
              {message.error && (
                <div className="mt-2 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-2.5 py-2 text-[0.72rem] font-semibold text-red-700">
                  <CircleAlert size={14} className="mt-0.5 shrink-0" />
                  <span className="min-w-0 break-words">{message.error}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {!isUser && !props.streaming && !isEmpty && (
          <div className="flex flex-wrap items-center gap-1.5 px-1">
            {message.model && <Meta title="Model that answered">{message.model}</Meta>}
            {message.cached && (
              <Meta title="Served from the semantic cache">
                <DatabaseZap size={9} strokeWidth={2.8} />
                cached
              </Meta>
            )}
            {usageSummary(message.usage).map((entry) => (
              <Meta key={entry}>{entry}</Meta>
            ))}
            {message.latencyMs !== undefined && (
              <Meta title="Round trip latency">
                {Math.round(message.latencyMs)} ms
              </Meta>
            )}
            {message.ttftMs !== undefined && (
              <Meta title="Time to first token">
                TTFT {Math.round(message.ttftMs)} ms
              </Meta>
            )}
            {message.finishReason && <Meta>{message.finishReason}</Meta>}
            <Tooltip label={copied ? "Copied" : "Copy response"} withArrow>
              <ActionIcon
                size="xs"
                variant="subtle"
                color="gray"
                aria-label="Copy response"
                onClick={copy}
              >
                {copied ? <Check size={12} /> : <Copy size={12} />}
              </ActionIcon>
            </Tooltip>
            {props.onRegenerate && (
              <Tooltip label="Regenerate" withArrow>
                <ActionIcon
                  size="xs"
                  variant="subtle"
                  color="gray"
                  aria-label="Regenerate"
                  onClick={props.onRegenerate}
                >
                  <RefreshCw size={12} />
                </ActionIcon>
              </Tooltip>
            )}
          </div>
        )}
      </div>

      {isUser && (
        <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-100 text-slate-500">
          <User size={14} strokeWidth={2.2} />
        </span>
      )}
    </motion.div>
  );
};

export default Message;
