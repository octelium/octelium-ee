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
import Meta from "@/components/Meta";
import { ActionIcon, Button, Menu, Popover, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowDown,
  ChartColumn,
  Download,
  Ellipsis,
  FileUp,
  KeyRound,
  PanelLeftOpen,
  Pencil,
  ShieldAlert,
  ShieldCheck,
  ShieldEllipsis,
  SquarePen,
  Trash2,
  TriangleAlert,
  Waypoints,
} from "lucide-react";
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { type BlockContext } from "./Blocks";
import { AgentClient } from "./client";
import Composer, { type Upload } from "./Composer";
import { MessageView, PendingAssistant, TranscriptSkeleton } from "./Messages";
import ModelsDrawer, { ModelPicker } from "./Models";
import { applyEvent, emptyChatState, type ChatState } from "./reducer";
import Sidebar from "./Sidebar";
import { AgentMark, ConfirmModal, isMac, StatusDot } from "./ui";
import {
  conversationMarkdown,
  formatBytes,
  greeting,
  messageText,
  readStorage,
  saveBlob,
  toFileName,
  writeStorage,
} from "./utils";

const sidebarStorageKey = "octelium-console-agent-sidebar";
const sidebarWidth = 272;
const ease = [0.22, 1, 0.36, 1] as const;

const suggestions = [
  {
    icon: Waypoints,
    title: "Cluster overview",
    prompt: "Give me an overview of the Cluster's Services and their health",
  },
  {
    icon: ShieldEllipsis,
    title: "Denied access",
    prompt: "Which Users had denied access requests in the last 24 hours?",
  },
  {
    icon: ChartColumn,
    title: "Traffic trends",
    prompt: "Chart the access logs per Service over the last 7 days",
  },
  {
    icon: ShieldCheck,
    title: "Policy review",
    prompt: "List the Policies that apply to the Group admins",
  },
];

interface Optimistic {
  text: string;
  attachments: FileInfo[];
  createdAt: string;
}

const hasFiles = (event: React.DragEvent) =>
  Array.from(event.dataTransfer.types).includes("Files");

const ToolbarButton = (props: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) => (
  <Tooltip label={props.label} withArrow>
    <ActionIcon
      variant="subtle"
      color="gray"
      aria-label={props.label}
      onClick={props.onClick}
    >
      {props.children}
    </ActionIcon>
  </Tooltip>
);

const Chat = (props: {
  client: AgentClient;
  info: AgentInfo;
  footer?: React.ReactNode;
  banner?: React.ReactNode;
}) => {
  const { client, info } = props;
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const selected = searchParams.get("c") ?? undefined;
  const wide = useMediaQuery("(min-width: 64em)", true, {
    getInitialValueInEffect: false,
  });

  const [chat, setChat] = React.useState<ChatState>(emptyChatState);
  const [loading, setLoading] = React.useState(() => !!selected);
  const [input, setInput] = React.useState("");
  const [attachments, setAttachments] = React.useState<FileInfo[]>([]);
  const [uploads, setUploads] = React.useState<Upload[]>([]);
  const [previews, setPreviews] = React.useState<Record<string, string>>({});
  const [sending, setSending] = React.useState(false);
  const [optimistic, setOptimistic] = React.useState<Optimistic>();
  const [modelsOpened, setModelsOpened] = React.useState(false);
  const [collapsed, setCollapsed] = React.useState(
    () => readStorage(sidebarStorageKey) === "collapsed",
  );
  const [drawerOpened, setDrawerOpened] = React.useState(false);
  const [dragging, setDragging] = React.useState(false);
  const [atBottom, setAtBottom] = React.useState(true);
  const [scrolled, setScrolled] = React.useState(false);
  const [focusTarget, setFocusTarget] = React.useState<string>();
  const [flashId, setFlashId] = React.useState<string>();
  const [deleting, setDeleting] = React.useState<Conversation>();
  const [editingTitle, setEditingTitle] = React.useState(false);
  const [approvalVisible, setApprovalVisible] = React.useState(false);

  const textareaRef = React.useRef<HTMLTextAreaElement>(null);
  const searchRef = React.useRef<HTMLInputElement>(null);
  const transcriptRef = React.useRef<HTMLDivElement>(null);
  const contentRef = React.useRef<HTMLDivElement>(null);
  const stickToBottomRef = React.useRef(true);
  const loadedRef = React.useRef<string | undefined>(undefined);
  const dragDepthRef = React.useRef(0);
  const draftsRef = React.useRef(new Map<string, string>());
  const inputRef = React.useRef(input);
  const draftKeyRef = React.useRef(selected ?? "");

  const sidebarOpen = wide ? !collapsed : drawerOpened;

  const conversationsKey = React.useMemo(
    () => ["agent", client.baseUrl, "conversations"],
    [client.baseUrl],
  );
  const conversationsQuery = useQuery({
    queryKey: conversationsKey,
    queryFn: () => client.listConversations(),
  });
  const conversations = conversationsQuery.data?.items;
  const conversation = conversations?.find((c) => c.id === selected);

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
    (updated: Conversation) => {
      queryClient.setQueryData<ListConversationsResponse>(
        conversationsKey,
        (data) => {
          const items = (data?.items ?? []).filter(
            (itm) => itm.id !== updated.id,
          );
          return { items: [updated, ...items] };
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
        setOptimistic(undefined);
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
    inputRef.current = input;
  }, [input]);

  React.useEffect(() => {
    const key = selected ?? "";
    if (draftKeyRef.current === key) return;
    draftsRef.current.set(draftKeyRef.current, inputRef.current);
    draftKeyRef.current = key;
    setInput(draftsRef.current.get(key) ?? "");
  }, [selected]);

  React.useEffect(() => {
    if (loadedRef.current === selected) return;
    loadedRef.current = selected;
    stickToBottomRef.current = true;
    setAtBottom(true);
    setScrolled(false);
    setOptimistic(undefined);
    setEditingTitle(false);
    setChat(emptyChatState);
    if (selected) {
      void load(selected);
    } else {
      setLoading(false);
    }
  }, [selected, load]);

  const switching = loadedRef.current !== selected;
  const run = switching ? undefined : chat.run;
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
    let queue: AgentEvent[] = [];
    let frame: number | undefined;

    const flush = () => {
      if (frame !== undefined) {
        cancelAnimationFrame(frame);
        frame = undefined;
      }
      if (queue.length === 0) return;
      const events = queue;
      queue = [];
      setChat((state) =>
        events.reduce(
          (current, event) =>
            loadedRef.current === event.conversationId
              ? applyEvent(current, event)
              : current,
          state,
        ),
      );
    };

    const unsubscribe = client.subscribe(
      runId,
      afterSeq,
      (event: AgentEvent) => {
        if (event.type === "conversation.updated") {
          onConversationUpdated(event.conversation);
        }
        queue.push(event);
        frame ??= requestAnimationFrame(flush);
      },
      () => {
        flush();
        void queryClient.invalidateQueries({ queryKey: conversationsKey });
        if (loadedRef.current === conversationId) {
          void load(conversationId);
        }
      },
    );

    return () => {
      unsubscribe();
      flush();
    };
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

  const userMessageArrived =
    !!run && chat.messages.some((m) => m.id === run.userMessageId);
  const showOptimistic = !!optimistic && !userMessageArrived;

  React.useEffect(() => {
    if (userMessageArrived) setOptimistic(undefined);
  }, [userMessageArrived]);
  const showPendingAssistant =
    (sending || isActive) &&
    !chat.messages.some(
      (m) => m.role === "assistant" && m.id === run?.assistantMessageId,
    );
  const busy = loading || switching;
  const isEmpty =
    chat.messages.length === 0 && !showOptimistic && !busy && !sending;

  React.useEffect(() => {
    const el = transcriptRef.current;
    const content = contentRef.current;
    if (!el || !content) return;
    const observer = new ResizeObserver(() => {
      if (stickToBottomRef.current) {
        el.scrollTop = el.scrollHeight;
      }
    });
    observer.observe(content);
    return () => observer.disconnect();
  }, [isEmpty]);

  const awaitingApproval = run?.status === "awaiting_approval";

  React.useEffect(() => {
    const root = transcriptRef.current;
    if (!awaitingApproval || !root) return;
    const targets = root.querySelectorAll("[data-approval-pending]");
    if (targets.length === 0) {
      setApprovalVisible(false);
      return;
    }
    const visible = new Set<Element>();
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            visible.add(entry.target);
          } else {
            visible.delete(entry.target);
          }
        }
        setApprovalVisible(visible.size > 0);
      },
      { root, threshold: 0.35 },
    );
    targets.forEach((target) => observer.observe(target));
    return () => observer.disconnect();
  }, [awaitingApproval, chat.messages]);

  React.useLayoutEffect(() => {
    if (!focusTarget || loading) return;
    const el = transcriptRef.current?.querySelector(
      `[data-message-id="${CSS.escape(focusTarget)}"]`,
    );
    if (!el) return;
    setFocusTarget(undefined);
    stickToBottomRef.current = false;
    el.scrollIntoView({ block: "center", behavior: "smooth" });
    setFlashId(focusTarget);
  }, [focusTarget, loading, chat.messages]);

  React.useEffect(() => {
    if (!flashId) return;
    const timer = window.setTimeout(() => setFlashId(undefined), 1600);
    return () => window.clearTimeout(timer);
  }, [flashId]);

  const scrollToBottom = (behavior: ScrollBehavior = "smooth") => {
    const el = transcriptRef.current;
    if (!el) return;
    stickToBottomRef.current = true;
    el.scrollTo({ top: el.scrollHeight, behavior });
  };

  const focusComposer = () =>
    requestAnimationFrame(() => textareaRef.current?.focus());

  const toggleSidebar = React.useCallback(() => {
    if (wide) {
      setCollapsed((value) => {
        writeStorage(sidebarStorageKey, value ? "expanded" : "collapsed");
        return !value;
      });
    } else {
      setDrawerOpened((value) => !value);
    }
  }, [wide]);

  const newChat = React.useCallback(() => {
    setSelected(undefined);
    setDrawerOpened(false);
    focusComposer();
  }, [setSelected]);

  const openConversation = React.useCallback(
    (id: string, messageId?: string) => {
      setSelected(id);
      setDrawerOpened(false);
      setFocusTarget(messageId);
    },
    [setSelected],
  );

  const openSearch = React.useCallback(() => {
    if (wide) {
      setCollapsed(false);
      writeStorage(sidebarStorageKey, "expanded");
    } else {
      setDrawerOpened(true);
    }
    window.setTimeout(() => searchRef.current?.focus(), 60);
  }, [wide]);

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const mod = isMac ? event.metaKey : event.ctrlKey;
      const key = event.key.toLowerCase();
      if (mod && !event.shiftKey && !event.altKey && key === "k") {
        event.preventDefault();
        openSearch();
        return;
      }
      if (mod && event.shiftKey && !event.altKey && key === "o") {
        event.preventDefault();
        newChat();
        return;
      }
      if (
        event.target === document.body &&
        event.key.length === 1 &&
        event.key !== " " &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey
      ) {
        textareaRef.current?.focus();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [openSearch, newChat]);

  React.useEffect(() => {
    if (!drawerOpened) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setDrawerOpened(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [drawerOpened]);

  const addFiles = async (files: File[]) => {
    for (const file of files) {
      if (file.size > info.capabilities.uploads.maxBytes) {
        toast.error(
          `${file.name} exceeds the maximum size of ${formatBytes(info.capabilities.uploads.maxBytes)}`,
        );
        continue;
      }
      const key = `${file.name}-${file.size}-${Math.random()}`;
      setUploads((current) => [...current, { key, name: file.name }]);
      try {
        const uploaded = await client.uploadFile(file);
        setAttachments((current) => [...current, uploaded]);
        if (file.type.startsWith("image/")) {
          const url = URL.createObjectURL(file);
          setPreviews((current) => ({ ...current, [uploaded.id]: url }));
        }
      } catch (err) {
        toast.error(`Could not upload ${file.name}: ${(err as Error).message}`);
      } finally {
        setUploads((current) => current.filter((itm) => itm.key !== key));
      }
    }
  };

  const removeAttachment = (id: string) => {
    setAttachments((current) => current.filter((itm) => itm.id !== id));
    setPreviews((current) => {
      if (!current[id]) return current;
      URL.revokeObjectURL(current[id]);
      const { [id]: _, ...rest } = current;
      return rest;
    });
  };

  const send = async (override?: { text: string; files: FileInfo[] }) => {
    const text = (override?.text ?? input).trim();
    const files = override?.files ?? attachments;
    if ((!text && files.length === 0) || isActive || sending) return;
    if (!override && uploads.length > 0) return;

    const runInput: RunInput = {
      text,
      attachments: files.length > 0 ? files.map((a) => a.id) : undefined,
    };

    setSending(true);
    setOptimistic({
      text,
      attachments: files,
      createdAt: new Date().toISOString(),
    });
    if (!override) {
      setInput("");
      setAttachments([]);
    }
    stickToBottomRef.current = true;
    requestAnimationFrame(() => scrollToBottom("auto"));

    try {
      if (selected) {
        const next = await client.startRun(selected, runInput);
        if (loadedRef.current === selected) {
          setChat((state) => ({ ...state, run: next, lastSeq: 0 }));
        }
      } else {
        const res = await client.createConversation(runInput);
        loadedRef.current = res.conversation.id;
        draftKeyRef.current = res.conversation.id;
        draftsRef.current.delete("");
        setChat({ messages: [], run: res.run, lastSeq: 0 });
        onConversationUpdated(res.conversation);
        setSelected(res.conversation.id);
      }
      if (!override) {
        setPreviews((current) => {
          Object.values(current).forEach((url) => URL.revokeObjectURL(url));
          return {};
        });
      }
    } catch (err) {
      toast.error((err as Error).message);
      setOptimistic(undefined);
      if (!override) {
        setInput((current) => current || text);
        setAttachments((current) => (current.length > 0 ? current : files));
      }
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
      reason?: string,
    ) => {
      try {
        await client.decideApproval(
          decisionRunId,
          approvalId,
          decision,
          reason,
        );
      } catch (err) {
        toast.error((err as Error).message);
      }
    },
    [client],
  );

  const rename = React.useCallback(
    async (target: Conversation, title: string) => {
      try {
        onConversationUpdated(
          await client.renameConversation(target.id, title),
        );
      } catch (err) {
        toast.error((err as Error).message);
      }
    },
    [client, onConversationUpdated],
  );

  const remove = async (target: Conversation) => {
    await client.deleteConversation(target.id);
    queryClient.setQueryData<ListConversationsResponse>(
      conversationsKey,
      (data) => ({
        items: (data?.items ?? []).filter((itm) => itm.id !== target.id),
      }),
    );
    if (selected === target.id) {
      setSelected(undefined);
    }
    toast.success("The conversation is deleted");
  };

  const exportConversation = () => {
    const title = conversation?.title ?? "Conversation";
    saveBlob(
      new Blob([conversationMarkdown(title, chat.messages)], {
        type: "text/markdown",
      }),
      toFileName(title, "md"),
    );
  };

  const scrollToApproval = () => {
    const items = transcriptRef.current?.querySelectorAll(
      "[data-approval-pending]",
    );
    items?.[items.length - 1]?.scrollIntoView({
      block: "center",
      behavior: "smooth",
    });
  };

  const blockCtx = React.useMemo<Omit<BlockContext, "runId" | "streaming">>(
    () => ({ client, onDecide }),
    [client, onDecide],
  );

  const lastMessage = chat.messages.at(-1);
  const lastAssistantID = [...chat.messages]
    .reverse()
    .find((m) => m.role === "assistant")?.id;

  const retryMessageID =
    !isActive &&
    !sending &&
    lastMessage?.role === "assistant" &&
    lastMessage.status !== "completed" &&
    lastMessage.status !== "streaming"
      ? lastMessage.id
      : undefined;
  const retryInput = React.useMemo(() => {
    if (!retryMessageID) return undefined;
    const user = [...chat.messages].reverse().find((m) => m.role === "user");
    if (!user) return undefined;
    return { text: messageText(user), files: user.attachments ?? [] };
  }, [retryMessageID, chat.messages]);
  const onRetry = retryInput ? () => void send(retryInput) : undefined;

  const optimisticMessage: Message | undefined = optimistic && {
    id: "optimistic",
    conversationId: selected ?? "",
    role: "user",
    status: "completed",
    createdAt: optimistic.createdAt,
    attachments: optimistic.attachments,
    blocks: [
      {
        id: "optimistic",
        type: "markdown",
        text: optimistic.text,
        createdAt: optimistic.createdAt,
      },
    ],
  };

  const firstName =
    info.octelium.user?.displayName?.split(" ")[0] || info.octelium.user?.name;
  const issues = info.status === "degraded" ? info.issues : [];

  const modelPicker = (
    <ModelPicker
      client={client}
      info={info}
      onManage={() => setModelsOpened(true)}
    />
  );

  const composer = (
    <Composer
      value={input}
      onChange={setInput}
      attachments={attachments}
      uploads={uploads}
      previews={previews}
      onAddFiles={(files) => void addFiles(files)}
      onRemoveAttachment={removeAttachment}
      onSend={() => void send()}
      onStop={() => void stop()}
      isActive={isActive}
      sending={sending}
      placeholder={
        isEmpty
          ? "Ask the agent to inspect, analyze or change the Cluster…"
          : "Reply to the agent…"
      }
      modelPicker={modelPicker}
      textareaRef={textareaRef}
    />
  );

  const sidebar = (
    <Sidebar
      client={client}
      info={info}
      conversations={conversations ?? []}
      loading={conversationsQuery.isPending}
      selected={selected}
      onSelect={openConversation}
      onNew={newChat}
      onRename={rename}
      onDelete={setDeleting}
      onClose={toggleSidebar}
      closeLabel={wide ? "Hide the sidebar" : "Close"}
      searchRef={searchRef}
      footer={props.footer}
    />
  );

  return (
    <div className="relative flex h-full min-h-0 w-full overflow-hidden">
      <Meta title={conversation ? `${conversation.title} - Agent` : "Agent"} />

      {wide ? (
        <motion.aside
          initial={false}
          animate={{ width: sidebarOpen ? sidebarWidth : 0 }}
          transition={{ duration: 0.3, ease }}
          className="h-full shrink-0 overflow-hidden"
          aria-hidden={!sidebarOpen}
          inert={!sidebarOpen}
        >
          <div
            className="h-full border-r border-slate-200/80 bg-slate-50/70"
            style={{ width: sidebarWidth }}
          >
            {sidebar}
          </div>
        </motion.aside>
      ) : (
        <AnimatePresence>
          {sidebarOpen && (
            <>
              <motion.div
                key="backdrop"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
                onClick={() => setDrawerOpened(false)}
                className="absolute inset-0 z-30 bg-[rgba(15,23,42,0.28)] backdrop-blur-[1px]"
              />
              <motion.aside
                key="drawer"
                initial={{ x: "-100%" }}
                animate={{ x: 0 }}
                exit={{ x: "-100%" }}
                transition={{ duration: 0.3, ease }}
                className="absolute inset-y-0 left-0 z-40 w-[min(300px,88%)] border-r border-slate-200 bg-slate-50 shadow-overlay"
              >
                {sidebar}
              </motion.aside>
            </>
          )}
        </AnimatePresence>
      )}

      <main
        className="relative flex min-w-0 flex-1 flex-col bg-white"
        onDragEnter={(event) => {
          if (!hasFiles(event)) return;
          event.preventDefault();
          dragDepthRef.current++;
          setDragging(true);
        }}
        onDragOver={(event) => {
          if (hasFiles(event)) event.preventDefault();
        }}
        onDragLeave={(event) => {
          if (!hasFiles(event)) return;
          dragDepthRef.current = Math.max(0, dragDepthRef.current - 1);
          if (dragDepthRef.current === 0) setDragging(false);
        }}
        onDrop={(event) => {
          if (!hasFiles(event)) return;
          event.preventDefault();
          dragDepthRef.current = 0;
          setDragging(false);
          void addFiles(Array.from(event.dataTransfer.files));
        }}
      >
        <header
          className={twMerge(
            "relative z-10 flex h-12 shrink-0 items-center gap-1 border-b px-2 transition-colors duration-200 sm:px-3",
            scrolled && !isEmpty ? "border-slate-200/80" : "border-transparent",
          )}
        >
          {!sidebarOpen && (
            <>
              <ToolbarButton label="Show conversations" onClick={toggleSidebar}>
                <PanelLeftOpen size={16} />
              </ToolbarButton>
              <ToolbarButton label="New chat" onClick={newChat}>
                <SquarePen size={15} />
              </ToolbarButton>
            </>
          )}
          <div className="flex min-w-0 flex-1 items-center gap-2 px-1.5 text-body font-semibold">
            {editingTitle && conversation ? (
              <input
                autoFocus
                aria-label="Conversation title"
                defaultValue={conversation.title}
                maxLength={200}
                onFocus={(event) => event.currentTarget.select()}
                onBlur={(event) => {
                  setEditingTitle(false);
                  const value = event.currentTarget.value.trim();
                  if (value && value !== conversation.title) {
                    void rename(conversation, value);
                  }
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter") event.currentTarget.blur();
                  if (event.key === "Escape") {
                    event.currentTarget.value = conversation.title;
                    event.currentTarget.blur();
                  }
                }}
                className="h-8 w-full max-w-md rounded-md border border-slate-300 bg-white px-2 text-slate-900 outline-none ring-2 ring-slate-900/10"
              />
            ) : (
              <AnimatePresence mode="wait" initial={false}>
                <motion.button
                  key={conversation?.id ?? "new"}
                  type="button"
                  initial={{ opacity: 0, y: 3 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -3 }}
                  transition={{ duration: 0.15 }}
                  disabled={!conversation}
                  onClick={() => setEditingTitle(true)}
                  title={conversation ? "Rename" : undefined}
                  className="min-w-0 truncate rounded-md px-1.5 py-1 text-left text-slate-800 transition-colors duration-150 enabled:cursor-pointer enabled:hover:bg-slate-100 disabled:text-slate-500"
                >
                  {conversation?.title ?? (selected ? "" : "New conversation")}
                </motion.button>
              </AnimatePresence>
            )}
            {isActive && (
              <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600">
                <StatusDot
                  tone={run?.status === "awaiting_approval" ? "amber" : "blue"}
                  pulse
                />
                <span className="hidden sm:inline">
                  {run?.status === "awaiting_approval"
                    ? "Needs approval"
                    : "Working"}
                </span>
              </span>
            )}
          </div>
          {issues.length > 0 && (
            <Popover position="bottom-end" withArrow shadow="md" width={320}>
              <Popover.Target>
                <Button
                  size="compact-xs"
                  variant="light"
                  color="orange"
                  leftSection={<TriangleAlert size={12} />}
                >
                  <span className="hidden sm:inline">Degraded</span>
                  <span className="sm:hidden">{issues.length}</span>
                </Button>
              </Popover.Target>
              <Popover.Dropdown>
                <p className="text-xs font-semibold text-slate-800">
                  The agent reports issues
                </p>
                <ul className="mt-2 space-y-1.5">
                  {issues.map((issue) => (
                    <li
                      key={issue}
                      className="flex items-start gap-1.5 text-xs leading-5 text-slate-600"
                    >
                      <TriangleAlert
                        size={12}
                        className="mt-1 shrink-0 text-amber-600"
                      />
                      {issue}
                    </li>
                  ))}
                </ul>
              </Popover.Dropdown>
            </Popover>
          )}
          {conversation && (
            <Menu position="bottom-end" withinPortal>
              <Menu.Target>
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  aria-label="Conversation actions"
                >
                  <Ellipsis size={16} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item
                  leftSection={<Pencil size={13} />}
                  onClick={() => setEditingTitle(true)}
                >
                  Rename
                </Menu.Item>
                <Menu.Item
                  leftSection={<Download size={13} />}
                  disabled={chat.messages.length === 0}
                  onClick={exportConversation}
                >
                  Export as Markdown
                </Menu.Item>
                {info.capabilities.login && (
                  <Menu.Item
                    leftSection={<KeyRound size={13} />}
                    onClick={() => setModelsOpened(true)}
                  >
                    Subscriptions
                  </Menu.Item>
                )}
                <Menu.Divider />
                <Menu.Item
                  color="red"
                  leftSection={<Trash2 size={13} />}
                  onClick={() => setDeleting(conversation)}
                >
                  Delete
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          )}
        </header>

        {props.banner && (
          <div className="mx-auto w-full max-w-3xl px-3 sm:px-6">
            {props.banner}
          </div>
        )}

        <div
          className={twMerge(
            "relative flex min-h-0 flex-1 flex-col",
            isEmpty &&
              "justify-center-safe overflow-y-auto overscroll-contain px-3 py-8 sm:px-6",
          )}
        >
          {isEmpty ? (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, ease }}
              className="mx-auto mb-6 flex w-full max-w-3xl flex-col items-center text-center"
            >
              <AgentMark size="lg" />
              <h1 className="mt-4 text-xl font-bold tracking-tight text-slate-900 sm:text-2xl">
                {greeting()}
                {firstName ? `, ${firstName}` : ""}
              </h1>
              <p className="mt-1.5 max-w-md text-body leading-6 text-slate-500">
                Inspect, troubleshoot and manage the Cluster. The agent acts
                with your own identity and asks before changing anything.
              </p>
            </motion.div>
          ) : (
            <div
              ref={transcriptRef}
              onScroll={(event) => {
                const el = event.currentTarget;
                const distance =
                  el.scrollHeight - el.scrollTop - el.clientHeight;
                stickToBottomRef.current = distance < 80;
                setAtBottom(distance < 80);
                setScrolled(el.scrollTop > 4);
              }}
              className="min-h-0 flex-1 overflow-y-auto overscroll-contain [scrollbar-gutter:stable]"
            >
              <div
                ref={contentRef}
                className="mx-auto flex w-full max-w-3xl flex-col gap-5 px-4 pb-10 pt-4 sm:px-6"
              >
                {busy && (switching || chat.messages.length === 0) ? (
                  <TranscriptSkeleton />
                ) : (
                  chat.messages.map((message) => {
                    const streaming =
                      isActive &&
                      message.id === lastAssistantID &&
                      message.status === "streaming";
                    return (
                      <MessageView
                        key={message.id}
                        message={message}
                        ctx={blockCtx}
                        streaming={streaming}
                        run={streaming ? run : undefined}
                        isLast={message.id === lastMessage?.id}
                        flash={message.id === flashId}
                        onRetry={
                          message.id === retryMessageID ? onRetry : undefined
                        }
                      />
                    );
                  })
                )}
                {showOptimistic && optimisticMessage && (
                  <MessageView
                    key="optimistic"
                    message={optimisticMessage}
                    ctx={blockCtx}
                    streaming={false}
                    isLast={false}
                    flash={false}
                    pending
                  />
                )}
                {showPendingAssistant && !busy && <PendingAssistant />}
              </div>
            </div>
          )}

          <motion.div
            layout="position"
            layoutDependency={isEmpty}
            transition={{ duration: 0.4, ease }}
            className={twMerge(
              "relative mx-auto w-full max-w-3xl",
              !isEmpty && "px-3 pb-3 sm:px-6 sm:pb-4",
            )}
          >
            <AnimatePresence>
              {!isEmpty && !atBottom && (
                <motion.button
                  key="scroll"
                  type="button"
                  aria-label="Scroll to the latest message"
                  initial={{ opacity: 0, y: 6, scale: 0.9 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: 6, scale: 0.9 }}
                  transition={{ duration: 0.18 }}
                  onClick={() => scrollToBottom()}
                  className="absolute -top-11 left-1/2 z-10 flex h-8 w-8 -translate-x-1/2 cursor-pointer items-center justify-center rounded-full border border-slate-200 bg-white text-slate-600 shadow-raised transition-colors hover:text-slate-900"
                >
                  <ArrowDown size={15} strokeWidth={2.25} />
                  {isActive && (
                    <span className="absolute -right-0.5 -top-0.5">
                      <StatusDot tone="blue" pulse />
                    </span>
                  )}
                </motion.button>
              )}
            </AnimatePresence>
            {!isEmpty && (
              <div className="pointer-events-none absolute inset-x-0 -top-6 h-6 bg-gradient-to-t from-white to-transparent" />
            )}

            <AnimatePresence initial={false}>
              {awaitingApproval && !approvalVisible && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.2, ease }}
                  className="overflow-hidden"
                >
                  <div className="mb-2 flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 py-1.5 pl-3 pr-1.5 text-xs font-semibold text-amber-800">
                    <ShieldAlert size={14} className="shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      The agent is waiting for your approval
                    </span>
                    <Button
                      size="compact-xs"
                      variant="default"
                      onClick={scrollToApproval}
                    >
                      Review
                    </Button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            {composer}

            {!isEmpty && (
              <p className="mt-2 hidden text-center text-micro text-slate-400 sm:block">
                The agent can make mistakes. Changes to the Cluster always
                require your approval.
              </p>
            )}
          </motion.div>

          {isEmpty && (
            <div className="mx-auto mt-4 grid w-full max-w-3xl gap-2 sm:grid-cols-2">
              {suggestions.map((suggestion, idx) => (
                <motion.button
                  key={suggestion.title}
                  type="button"
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: 0.35,
                    delay: 0.08 + idx * 0.05,
                    ease,
                  }}
                  onClick={() => {
                    setInput(suggestion.prompt);
                    focusComposer();
                  }}
                  className="group flex cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-white px-3.5 py-3 text-left outline-none transition-[border-color,box-shadow,background-color] duration-200 hover:border-slate-300 hover:bg-slate-50/60 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
                >
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500 transition-colors duration-200 group-hover:text-slate-800">
                    <suggestion.icon size={14} strokeWidth={2.25} />
                  </span>
                  <span className="min-w-0">
                    <span className="block text-xs font-semibold text-slate-800">
                      {suggestion.title}
                    </span>
                    <span className="mt-0.5 block text-xs leading-5 text-slate-500">
                      {suggestion.prompt}
                    </span>
                  </span>
                </motion.button>
              ))}
            </div>
          )}
        </div>

        <AnimatePresence>
          {dragging && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              className="pointer-events-none absolute inset-2 z-20 flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-slate-300 bg-white/85 backdrop-blur-sm"
            >
              <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-slate-900 text-white shadow-raised">
                <FileUp size={20} />
              </span>
              <p className="mt-3 text-body font-semibold text-slate-800">
                Drop files to attach them
              </p>
              <p className="mt-0.5 text-micro text-slate-500">
                Up to {formatBytes(info.capabilities.uploads.maxBytes)} each
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </main>

      <ModelsDrawer
        client={client}
        opened={modelsOpened}
        onClose={() => setModelsOpened(false)}
      />

      <ConfirmModal
        opened={!!deleting}
        onClose={() => setDeleting(undefined)}
        title="Delete the conversation"
        confirmLabel="Delete"
        onConfirm={() => deleting && remove(deleting)}
      >
        <span className="font-semibold text-slate-800">{deleting?.title}</span>{" "}
        and its messages will be permanently deleted. Any run in progress is
        stopped.
      </ConfirmModal>
    </div>
  );
};

export default Chat;
