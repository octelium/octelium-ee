import {
  isTerminalRunStatus,
  type AgentEvent,
  type AgentInfo,
  type ApprovalDecision,
  type Conversation,
  type FileInfo,
  type ListConversationsResponse,
  type Message,
  type RunInput,
} from "@/apis/consoleagent/protocol";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import { ActionIcon, Button, Menu, Textarea, Tooltip } from "@mantine/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  Bot,
  CircleAlert,
  EllipsisVertical,
  MessageSquarePlus,
  Paperclip,
  Pencil,
  Send,
  Sparkles,
  Square,
  Trash2,
  X,
} from "lucide-react";
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { BlockView, type BlockContext } from "./Blocks";
import { AgentClient } from "./client";
import ModelsDrawer, { modelLabel } from "./Models";
import { applyEvent, emptyChatState, type ChatState } from "./reducer";
import { formatBytes } from "./utils";

const Typing = () => (
  <span className="inline-flex items-center gap-1 py-1 text-slate-500">
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:150ms]" />
    <i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:300ms]" />
  </span>
);

const messageStatusLabel: Partial<Record<Message["status"], string>> = {
  failed: "Failed",
  cancelled: "Stopped",
  interrupted: "Interrupted",
};

const MessageView = (props: {
  message: Message;
  ctx: Omit<BlockContext, "runId" | "streaming">;
  streaming: boolean;
}) => {
  const { message } = props;
  const isUser = message.role === "user";

  if (isUser) {
    const text = message.blocks
      .map((b) => (b.type === "markdown" ? b.text : ""))
      .join("\n");
    return (
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="flex w-full justify-end"
      >
        <div className="flex max-w-[85%] flex-col items-end gap-1">
          {message.attachments && message.attachments.length > 0 && (
            <div className="flex flex-wrap justify-end gap-1">
              {message.attachments.map((file) => (
                <span
                  key={file.id}
                  className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-0.5 text-micro text-slate-600"
                >
                  <Paperclip size={10} />
                  {file.name}
                </span>
              ))}
            </div>
          )}
          {text && (
            <div className="rounded-2xl rounded-br-md bg-slate-900 px-3.5 py-2.5 text-white">
              <p className="whitespace-pre-wrap break-words text-body leading-6">
                {text}
              </p>
            </div>
          )}
        </div>
      </motion.div>
    );
  }

  const ctx: BlockContext = {
    ...props.ctx,
    runId: message.runId,
    streaming: props.streaming,
  };
  const status = messageStatusLabel[message.status];

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex w-full gap-2.5"
    >
      <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
        <Bot size={14} strokeWidth={2.2} />
      </span>
      <div className="min-w-0 flex-1">
        {message.blocks.length === 0 && props.streaming ? (
          <Typing />
        ) : (
          message.blocks.map((block, idx) => (
            <BlockView
              key={block.id}
              block={block}
              ctx={ctx}
              isLast={idx === message.blocks.length - 1}
            />
          ))
        )}
        {props.streaming && message.blocks.length > 0 && <Typing />}
        {(status || message.error) && (
          <p className="mt-1.5 flex items-center gap-1.5 text-micro font-semibold text-slate-500">
            <CircleAlert size={12} />
            {status}
            {message.error && !message.blocks.some((b) => b.type === "error")
              ? `: ${message.error.message}`
              : ""}
          </p>
        )}
      </div>
    </motion.div>
  );
};

const ConversationList = (props: {
  conversations: Conversation[];
  selected?: string;
  onSelect: (id?: string) => void;
  onRename: (conversation: Conversation) => void;
  onDelete: (conversation: Conversation) => void;
}) => (
  <div className="flex min-h-0 flex-1 flex-col">
    <div className="p-2">
      <Button
        fullWidth
        size="xs"
        color="dark"
        leftSection={<MessageSquarePlus size={14} />}
        onClick={() => props.onSelect(undefined)}
      >
        New chat
      </Button>
    </div>
    <ul className="min-h-0 flex-1 overflow-y-auto px-2 pb-2">
      {props.conversations.map((conversation) => {
        const isSelected = conversation.id === props.selected;
        return (
          <li key={conversation.id}>
            <div
              className={twMerge(
                "group flex items-center gap-1 rounded-lg pr-1 transition-colors duration-150",
                isSelected
                  ? "bg-slate-900 text-white"
                  : "text-slate-700 hover:bg-slate-100",
              )}
            >
              <button
                type="button"
                onClick={() => props.onSelect(conversation.id)}
                className="min-w-0 flex-1 cursor-pointer px-2.5 py-2 text-left"
              >
                <span className="block truncate text-xs font-semibold">
                  {conversation.title}
                </span>
                <span
                  className={twMerge(
                    "block truncate text-micro",
                    isSelected ? "text-slate-300" : "text-slate-500",
                  )}
                >
                  {conversation.activeRunId ? (
                    "Working…"
                  ) : (
                    <TimeAgo
                      rfc3339={Timestamp.fromDate(
                        new Date(conversation.updatedAt),
                      )}
                    />
                  )}
                </span>
              </button>
              <Menu position="bottom-end" withinPortal>
                <Menu.Target>
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color={isSelected ? "gray.0" : "gray"}
                    aria-label="Conversation actions"
                    className="opacity-0 transition-opacity group-hover:opacity-100 focus:opacity-100"
                  >
                    <EllipsisVertical size={14} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item
                    leftSection={<Pencil size={13} />}
                    onClick={() => props.onRename(conversation)}
                  >
                    Rename
                  </Menu.Item>
                  <Menu.Item
                    color="red"
                    leftSection={<Trash2 size={13} />}
                    onClick={() => props.onDelete(conversation)}
                  >
                    Delete
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </div>
          </li>
        );
      })}
      {props.conversations.length === 0 && (
        <li className="px-2.5 py-6 text-center text-micro text-slate-500">
          No conversations yet
        </li>
      )}
    </ul>
  </div>
);

const suggestions = [
  "Give me an overview of the Cluster's Services and their health",
  "Which Users had denied access requests in the last 24 hours?",
  "Chart the access logs per Service over the last 7 days",
  "List the Policies that apply to the Group admins",
];

const Chat = (props: { client: AgentClient; info: AgentInfo }) => {
  const { client, info } = props;
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const selected = searchParams.get("c") ?? undefined;

  const [chat, setChat] = React.useState<ChatState>(emptyChatState);
  const [loading, setLoading] = React.useState(false);
  const [input, setInput] = React.useState("");
  const [attachments, setAttachments] = React.useState<FileInfo[]>([]);
  const [uploading, setUploading] = React.useState(0);
  const [sending, setSending] = React.useState(false);
  const [modelsOpened, setModelsOpened] = React.useState(false);
  const fileRef = React.useRef<HTMLInputElement>(null);
  const transcriptRef = React.useRef<HTMLDivElement>(null);
  const stickToBottomRef = React.useRef(true);
  const loadedRef = React.useRef<string | undefined>(undefined);

  const conversationsKey = React.useMemo(
    () => ["agent", client.baseUrl, "conversations"],
    [client.baseUrl],
  );
  const conversationsQuery = useQuery({
    queryKey: conversationsKey,
    queryFn: () => client.listConversations(),
  });

  const modelsQuery = useQuery({
    queryKey: ["agent", client.baseUrl, "models"],
    queryFn: () => client.listModels(),
    enabled: info.capabilities.models,
  });
  const currentModel = modelsQuery.data?.current ?? info.model;

  const setSelected = React.useCallback(
    (id?: string) => {
      setSearchParams(
        (params) => {
          const ret = new URLSearchParams(params);
          if (id) {
            ret.set("c", id);
          } else {
            ret.delete("c");
          }
          return ret;
        },
        { replace: false },
      );
    },
    [setSearchParams],
  );

  const onConversationUpdated = React.useCallback(
    (conversation: Conversation) => {
      queryClient.setQueryData<ListConversationsResponse>(
        conversationsKey,
        (data) => {
          const items = (data?.items ?? []).filter(
            (itm) => itm.id !== conversation.id,
          );
          return { items: [conversation, ...items] };
        },
      );
    },
    [queryClient, conversationsKey],
  );

  const load = React.useCallback(
    async (id: string) => {
      setLoading(true);
      try {
        const detail = await client.getConversation(id);
        if (loadedRef.current !== id) return;
        const messages = [...detail.messages];
        const active = detail.activeRun;
        if (
          active?.message &&
          !messages.some((m) => m.id === active.message!.id)
        ) {
          messages.push(active.message);
        }
        setChat({
          messages,
          run: active?.run,
          lastSeq: active?.run.lastSeq ?? 0,
        });
      } catch (err) {
        if (loadedRef.current === id) {
          toast.error((err as Error).message);
          setChat(emptyChatState);
        }
      } finally {
        if (loadedRef.current === id) setLoading(false);
      }
    },
    [client],
  );

  React.useEffect(() => {
    if (loadedRef.current === selected) return;
    loadedRef.current = selected;
    stickToBottomRef.current = true;
    setChat(emptyChatState);
    if (selected) {
      void load(selected);
    } else {
      setLoading(false);
    }
  }, [selected, load]);

  const run = chat.run;
  const isActive = !!run && !isTerminalRunStatus(run.status);
  const runId = run?.id;
  const conversationId = run?.conversationId;

  const lastSeqRef = React.useRef(chat.lastSeq);
  React.useEffect(() => {
    lastSeqRef.current = chat.lastSeq;
  }, [chat.lastSeq]);

  React.useEffect(() => {
    if (!isActive || !runId || !conversationId) return;
    const afterSeq = lastSeqRef.current;

    const unsubscribe = client.subscribe(
      runId,
      afterSeq,
      (event: AgentEvent) => {
        if (event.type === "conversation.updated") {
          onConversationUpdated(event.conversation);
        }
        setChat((state) =>
          loadedRef.current === event.conversationId
            ? applyEvent(state, event)
            : state,
        );
      },
      () => {
        void queryClient.invalidateQueries({ queryKey: conversationsKey });
        if (loadedRef.current === conversationId) {
          void load(conversationId);
        }
      },
    );

    return unsubscribe;
  }, [
    client,
    isActive,
    runId,
    conversationId,
    onConversationUpdated,
    queryClient,
    conversationsKey,
    load,
  ]);

  React.useEffect(() => {
    const el = transcriptRef.current;
    if (el && stickToBottomRef.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [chat.messages]);

  const addFiles = async (files: FileList | File[] | null) => {
    if (!files) return;
    for (const file of Array.from(files)) {
      if (file.size > info.capabilities.uploads.maxBytes) {
        toast.error(
          `${file.name} exceeds the maximum size of ${formatBytes(info.capabilities.uploads.maxBytes)}`,
        );
        continue;
      }
      setUploading((n) => n + 1);
      try {
        const uploaded = await client.uploadFile(file);
        setAttachments((current) => [...current, uploaded]);
      } catch (err) {
        toast.error(`Could not upload ${file.name}: ${(err as Error).message}`);
      } finally {
        setUploading((n) => n - 1);
      }
    }
  };

  const send = async (textArg?: string) => {
    const text = (textArg ?? input).trim();
    if ((!text && attachments.length === 0) || isActive || sending) return;

    const runInput: RunInput = {
      text,
      attachments:
        attachments.length > 0 ? attachments.map((a) => a.id) : undefined,
    };

    setSending(true);
    stickToBottomRef.current = true;
    try {
      if (selected) {
        const next = await client.startRun(selected, runInput);
        setChat((state) => ({ ...state, run: next, lastSeq: 0 }));
      } else {
        const res = await client.createConversation(runInput);
        loadedRef.current = res.conversation.id;
        setChat({ messages: [], run: res.run, lastSeq: 0 });
        onConversationUpdated(res.conversation);
        setSelected(res.conversation.id);
      }
      setInput("");
      setAttachments([]);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSending(false);
    }
  };

  const stop = async () => {
    if (!run) return;
    try {
      await client.cancelRun(run.id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const onDecide = React.useCallback(
    async (
      decisionRunId: string,
      approvalId: string,
      decision: ApprovalDecision,
    ) => {
      try {
        await client.decideApproval(decisionRunId, approvalId, decision);
      } catch (err) {
        toast.error((err as Error).message);
      }
    },
    [client],
  );

  const rename = async (conversation: Conversation) => {
    const title = window.prompt("Rename the conversation", conversation.title);
    if (!title || title.trim() === "" || title === conversation.title) return;
    try {
      onConversationUpdated(
        await client.renameConversation(conversation.id, title.trim()),
      );
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const remove = async (conversation: Conversation) => {
    if (!window.confirm(`Delete the conversation "${conversation.title}"?`)) {
      return;
    }
    try {
      await client.deleteConversation(conversation.id);
      queryClient.setQueryData<ListConversationsResponse>(
        conversationsKey,
        (data) => ({
          items: (data?.items ?? []).filter(
            (itm) => itm.id !== conversation.id,
          ),
        }),
      );
      if (selected === conversation.id) {
        setSelected(undefined);
      }
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const blockCtx = React.useMemo(
    () => ({ client, onDecide }),
    [client, onDecide],
  );
  const lastAssistantID = [...chat.messages]
    .reverse()
    .find((m) => m.role === "assistant")?.id;

  return (
    <div className="flex min-h-0 flex-1 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      <aside className="hidden w-64 shrink-0 flex-col border-r border-slate-100 bg-slate-50/50 md:flex">
        <ConversationList
          conversations={conversationsQuery.data?.items ?? []}
          selected={selected}
          onSelect={setSelected}
          onRename={(conversation) => void rename(conversation)}
          onDelete={(conversation) => void remove(conversation)}
        />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-100 px-4 py-2">
          <span className="min-w-0 truncate text-xs font-semibold text-slate-700">
            {conversationsQuery.data?.items.find((c) => c.id === selected)
              ?.title ?? "New conversation"}
          </span>
          <div className="flex items-center gap-1">
            <Button
              size="compact-xs"
              variant="subtle"
              color="gray"
              hiddenFrom="md"
              leftSection={<MessageSquarePlus size={13} />}
              onClick={() => setSelected(undefined)}
            >
              New
            </Button>
            <Tooltip label="Model and accounts" withArrow>
              <Button
                size="compact-xs"
                variant="light"
                color="gray"
                leftSection={<Sparkles size={13} />}
                onClick={() => setModelsOpened(true)}
              >
                {modelLabel(currentModel)}
              </Button>
            </Tooltip>
          </div>
        </div>

        <div
          ref={transcriptRef}
          onScroll={(event) => {
            const el = event.currentTarget;
            stickToBottomRef.current =
              el.scrollHeight - el.scrollTop - el.clientHeight < 80;
          }}
          className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4 py-5"
        >
          {loading && chat.messages.length === 0 ? (
            <div className="flex flex-1 items-center justify-center">
              <Typing />
            </div>
          ) : chat.messages.length === 0 ? (
            <div className="mx-auto flex max-w-xl flex-1 flex-col items-center justify-center text-center">
              <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-900 text-white">
                <Bot size={23} />
              </span>
              <p className="mt-3 text-sm font-bold text-slate-800">
                How can I help you manage the Cluster?
              </p>
              <p className="mt-1 text-xs leading-5 text-slate-500">
                The agent runs inside your own Workspace and uses your Octelium
                identity. Changes to the Cluster require your approval.
              </p>
              {!currentModel && (
                <Button
                  className="mt-3"
                  size="xs"
                  variant="light"
                  color="orange"
                  leftSection={<Sparkles size={13} />}
                  onClick={() => setModelsOpened(true)}
                >
                  Choose a model or sign in to get started
                </Button>
              )}
              <div className="mt-5 grid w-full gap-2 sm:grid-cols-2">
                {suggestions.map((suggestion) => (
                  <button
                    key={suggestion}
                    type="button"
                    disabled={sending}
                    onClick={() => void send(suggestion)}
                    className="cursor-pointer rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-left text-xs text-slate-600 transition-colors duration-200 hover:border-slate-300 hover:bg-slate-50"
                  >
                    {suggestion}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="mx-auto flex w-full max-w-4xl flex-col gap-5">
              {chat.messages.map((message) => (
                <MessageView
                  key={message.id}
                  message={message}
                  ctx={blockCtx}
                  streaming={
                    isActive &&
                    message.id === lastAssistantID &&
                    message.status === "streaming"
                  }
                />
              ))}
            </div>
          )}
        </div>

        {run?.activity && isActive && (
          <p className="mx-4 mb-1 text-micro font-semibold text-slate-500">
            {run.activity.message}
          </p>
        )}
        {run?.status === "awaiting_approval" && (
          <p className="mx-4 mb-1 text-micro font-semibold text-amber-700">
            The agent is waiting for your approval
          </p>
        )}

        <div className="shrink-0 border-t border-slate-100 p-3">
          <div className="mx-auto w-full max-w-4xl">
            {(attachments.length > 0 || uploading > 0) && (
              <div className="mb-2 flex flex-wrap gap-1.5">
                {attachments.map((file) => (
                  <span
                    key={file.id}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 py-1 pl-2 pr-1 text-micro text-slate-600"
                  >
                    <Paperclip size={11} />
                    <span className="max-w-[160px] truncate">{file.name}</span>
                    <span className="text-slate-400">
                      {formatBytes(file.size)}
                    </span>
                    <button
                      type="button"
                      aria-label={`Remove ${file.name}`}
                      onClick={() =>
                        setAttachments((current) =>
                          current.filter((itm) => itm.id !== file.id),
                        )
                      }
                      className="flex h-4 w-4 cursor-pointer items-center justify-center rounded text-slate-500 hover:bg-slate-200"
                    >
                      <X size={10} strokeWidth={3} />
                    </button>
                  </span>
                ))}
                {uploading > 0 && (
                  <span className="rounded-lg border border-dashed border-slate-300 px-2 py-1 text-micro text-slate-500">
                    Uploading…
                  </span>
                )}
              </div>
            )}

            <Textarea
              aria-label="Message"
              placeholder="Ask the agent to inspect, analyze or change the Cluster…"
              autosize
              minRows={2}
              maxRows={10}
              value={input}
              onChange={(event) => setInput(event.currentTarget.value)}
              onPaste={(event) => {
                const files = Array.from(event.clipboardData.files);
                if (files.length > 0) {
                  event.preventDefault();
                  void addFiles(files);
                }
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  void send();
                }
              }}
            />

            <div className="mt-2 flex items-center justify-between gap-2">
              <div className="flex items-center gap-1">
                <input
                  ref={fileRef}
                  type="file"
                  multiple
                  hidden
                  onChange={(event) => {
                    void addFiles(event.currentTarget.files);
                    event.currentTarget.value = "";
                  }}
                />
                <Tooltip label="Attach files" withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    aria-label="Attach files"
                    onClick={() => fileRef.current?.click()}
                  >
                    <Paperclip size={15} />
                  </ActionIcon>
                </Tooltip>
              </div>
              <div className="flex items-center gap-2">
                <span className="hidden text-micro text-slate-500 sm:inline">
                  Enter to send · Shift + Enter for a new line
                </span>
                {isActive ? (
                  <Button
                    size="sm"
                    color="red"
                    leftSection={<Square size={14} />}
                    onClick={() => void stop()}
                  >
                    Stop
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    color="dark"
                    leftSection={<Send size={14} />}
                    loading={sending}
                    disabled={
                      uploading > 0 ||
                      (input.trim() === "" && attachments.length === 0)
                    }
                    onClick={() => void send()}
                  >
                    Send
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      <ModelsDrawer
        client={client}
        opened={modelsOpened}
        onClose={() => setModelsOpened(false)}
      />
    </div>
  );
};

export default Chat;
