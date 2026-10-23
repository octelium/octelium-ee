import type { Conversation, FileInfo } from "@/apis/consoleagent/protocol";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  usePageConditionsValue,
  usePageDetailsValue,
} from "@/components/PageHeader/context";
import TimeAgo from "@/components/TimeAgo";
import { AgentSurface, type ReadyChatProps } from "@/pages/Agent/Agent";
import { type BlockContext } from "@/pages/Agent/Blocks";
import Composer, { type Upload } from "@/pages/Agent/Composer";
import { ContextChips, type ContextChip } from "@/pages/Agent/Context";
import {
  MessageView,
  PendingAssistant,
  TranscriptSkeleton,
} from "@/pages/Agent/Messages";
import ModelsDrawer, { ModelPicker } from "@/pages/Agent/Models";
import { AgentMark, StatusDot } from "@/pages/Agent/ui";
import { useConversation } from "@/pages/Agent/useConversation";
import { useTranscript } from "@/pages/Agent/useTranscript";
import { formatBytes } from "@/pages/Agent/utils";
import { ActionIcon, Button, Menu, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowDown,
  Box,
  Check,
  History,
  LayoutDashboard,
  Maximize2,
  MessagesSquare,
  PanelsTopLeft,
  ScrollText,
  ShieldAlert,
  SquarePen,
  SquareTerminal,
  X,
} from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "@/lib/utils";
import { useAgent, type PinnedContext } from "./context";
import {
  buildPageContext,
  buildRunContext,
  contextAsText,
  routeResources,
  withConditions,
} from "./pageContext";
import { getSuggestions } from "./suggestions";

const ease = [0.22, 1, 0.36, 1] as const;

const DockFrameContext = React.createContext<(mounted: boolean) => void>(
  () => {},
);

const HeaderButton = (props: {
  label: string;
  onClick?: () => void;
  children: React.ReactNode;
}) => (
  <Tooltip label={props.label} withArrow openDelay={300}>
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

const DockHeader = (props: {
  title: string;
  status?: React.ReactNode;
  children?: React.ReactNode;
  onExpand: () => void;
  onClose: () => void;
}) => (
  <header className="flex h-12 shrink-0 items-center gap-0.5 border-b border-slate-200 pl-3 pr-1.5">
    <AgentMark />
    <div className="flex min-w-0 flex-1 items-center gap-2 px-2">
      <AnimatePresence mode="wait" initial={false}>
        <motion.h2
          key={props.title}
          initial={{ opacity: 0, y: 3 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -3 }}
          transition={{ duration: 0.15 }}
          className="min-w-0 truncate text-body font-semibold text-slate-800"
        >
          {props.title}
        </motion.h2>
      </AnimatePresence>
      {props.status}
    </div>
    {props.children}
    <HeaderButton label="Open in the Agent page" onClick={props.onExpand}>
      <Maximize2 size={14} />
    </HeaderButton>
    <HeaderButton label="Close" onClick={props.onClose}>
      <X size={16} />
    </HeaderButton>
  </header>
);

const HistoryMenu = (props: {
  conversations: Conversation[];
  selected?: string;
  onSelect: (id: string) => void;
  onShowAll: () => void;
}) => (
  <Menu position="bottom-end" width={280} withinPortal>
    <Menu.Target>
      <Tooltip label="Recent conversations" withArrow openDelay={300}>
        <ActionIcon
          variant="subtle"
          color="gray"
          aria-label="Recent conversations"
        >
          <History size={15} />
        </ActionIcon>
      </Tooltip>
    </Menu.Target>
    <Menu.Dropdown>
      <Menu.Label>Recent conversations</Menu.Label>
      {props.conversations.length === 0 && (
        <p className="px-3 pb-2 text-xs text-slate-500">No conversations yet</p>
      )}
      {props.conversations.slice(0, 8).map((conversation) => (
        <Menu.Item
          key={conversation.id}
          onClick={() => props.onSelect(conversation.id)}
          rightSection={
            conversation.id === props.selected ? (
              <Check size={13} strokeWidth={2.75} />
            ) : undefined
          }
        >
          <span className="block truncate">{conversation.title}</span>
          <span className="block text-micro font-normal text-slate-500">
            <TimeAgo
              rfc3339={Timestamp.fromDate(new Date(conversation.updatedAt))}
            />
          </span>
        </Menu.Item>
      ))}
      <Menu.Divider />
      <Menu.Item
        leftSection={<MessagesSquare size={13} />}
        onClick={props.onShowAll}
      >
        All conversations
      </Menu.Item>
    </Menu.Dropdown>
  </Menu>
);

const DockChat = (props: ReadyChatProps) => {
  const { client, info } = props;
  const agent = useAgent();
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const setChatMounted = React.useContext(DockFrameContext);
  const details = usePageDetailsValue();
  const conditions = usePageConditionsValue();

  const [input, setInput] = React.useState("");
  const [attachments, setAttachments] = React.useState<FileInfo[]>([]);
  const [uploads, setUploads] = React.useState<Upload[]>([]);
  const [previews, setPreviews] = React.useState<Record<string, string>>({});
  const [pinned, setPinned] = React.useState<PinnedContext>();
  const [pageIncluded, setPageIncluded] = React.useState(true);
  const [pending, setPending] = React.useState<string>();
  const [modelsOpened, setModelsOpened] = React.useState(false);
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);
  const sentPageRef = React.useRef<{ key?: string; path: string }>(undefined);
  const previewURLsRef = React.useRef(new Map<string, string>());
  const uploadVersionRef = React.useRef(0);

  const clearAttachments = React.useCallback(() => {
    uploadVersionRef.current += 1;
    previewURLsRef.current.forEach((url) => URL.revokeObjectURL(url));
    previewURLsRef.current.clear();
    setAttachments([]);
    setUploads([]);
    setPreviews({});
  }, []);

  React.useEffect(
    () => () => {
      uploadVersionRef.current += 1;
      previewURLsRef.current.forEach((url) => URL.revokeObjectURL(url));
      previewURLsRef.current.clear();
    },
    [],
  );

  const { trail } = agent;
  const page = React.useMemo(
    () =>
      buildPageContext({
        pathname,
        search,
        trail,
        details: withConditions(details, conditions),
      }),
    [pathname, search, trail, details, conditions],
  );
  const resources = React.useMemo(() => routeResources(trail), [trail]);
  const current = trail.at(-1);
  const resourceCrumb = trail.find((crumb) => crumb.resource);

  React.useEffect(() => {
    setChatMounted(true);
    return () => setChatMounted(false);
  }, [setChatMounted]);

  React.useEffect(() => {
    setPageIncluded(true);
  }, [pathname]);

  const conv = useConversation({
    client,
    conversationId: agent.conversationId,
    onCreated: (created) => {
      if (sentPageRef.current && sentPageRef.current.key === undefined) {
        sentPageRef.current = { ...sentPageRef.current, key: created.id };
      }
      agent.setConversationId(created.id);
    },
  });
  const {
    chat,
    conversation,
    isActive,
    sending,
    busy,
    switching,
    isEmpty,
    run,
    awaitingApproval,
  } = conv;

  const transcript = useTranscript({
    mounted: !isEmpty,
    awaitingApproval,
    messages: chat.messages,
  });

  const { setActivity } = agent;
  React.useEffect(() => {
    setActivity(
      awaitingApproval ? "approval" : isActive ? "working" : undefined,
    );
  }, [awaitingApproval, isActive, setActivity]);

  React.useEffect(() => () => setActivity(undefined), [setActivity]);

  const focusComposer = React.useCallback(
    () => requestAnimationFrame(() => textareaRef.current?.focus()),
    [],
  );

  React.useEffect(() => {
    if (agent.opened && !agent.hidden) focusComposer();
  }, [agent.opened, agent.hidden, focusComposer]);

  const addFiles = async (files: File[]) => {
    const version = uploadVersionRef.current;
    for (const file of files) {
      if (version !== uploadVersionRef.current) return;
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
        if (version !== uploadVersionRef.current) return;
        setAttachments((current) => [...current, uploaded]);
        if (file.type.startsWith("image/")) {
          const url = URL.createObjectURL(file);
          previewURLsRef.current.set(uploaded.id, url);
          setPreviews((current) => ({ ...current, [uploaded.id]: url }));
        }
      } catch (err) {
        if (version === uploadVersionRef.current) {
          toast.error(`Could not upload ${file.name}: ${(err as Error).message}`);
        }
      } finally {
        if (version === uploadVersionRef.current) {
          setUploads((current) => current.filter((itm) => itm.key !== key));
        }
      }
    }
  };

  const removeAttachment = (id: string) => {
    setAttachments((current) => current.filter((itm) => itm.id !== id));
    const url = previewURLsRef.current.get(id);
    if (!url) return;
    URL.revokeObjectURL(url);
    previewURLsRef.current.delete(id);
    setPreviews(Object.fromEntries(previewURLsRef.current));
  };

  const submit = async (value: string) => {
    const text = value.trim();
    const files = attachments;
    if ((!text && files.length === 0) || isActive || sending) return;
    if (uploads.length > 0) return;

    const key = agent.conversationId;
    const previous = sentPageRef.current;
    const includePage =
      pageIncluded &&
      !(
        key !== undefined &&
        previous?.key === key &&
        previous.path === page.path
      );
    let context = buildRunContext({
      page: includePage ? page : undefined,
      routeResources: resources,
      pinned,
    });
    let sendText = text;
    if (context && info.protocolVersion < 2) {
      sendText = `${contextAsText(context)}\n\n${text}`;
      context = undefined;
    }

    if (includePage) sentPageRef.current = { key, path: page.path };
    transcript.stickRef.current = true;
    requestAnimationFrame(() => transcript.scrollToBottom("auto"));

    if (await conv.send({ text: sendText, files, context })) {
      files.forEach((file) => removeAttachment(file.id));
      setPinned(undefined);
      if (value === input) setInput("");
    } else {
      sentPageRef.current = previous;
    }
  };

  const { request, consume } = agent;
  React.useEffect(() => {
    if (!request) return;
    consume(request.id);
    if (request.fresh || request.pinned) clearAttachments();
    if ((request.fresh || request.pinned) && agent.conversationId) {
      agent.setConversationId(undefined);
      sentPageRef.current = undefined;
    }
    setPinned(request.pinned);
    setPageIncluded(true);
    if (request.send && request.text) {
      setPending(request.text);
    } else {
      setInput(request.text ?? "");
      focusComposer();
    }
  }, [request, clearAttachments]);

  React.useEffect(() => {
    if (
      pending === undefined ||
      busy ||
      switching ||
      sending ||
      isActive ||
      uploads.length > 0
    ) {
      return;
    }
    setPending(undefined);
    void submit(pending);
  }, [pending, busy, switching, sending, isActive, uploads.length]);

  const newChat = () => {
    agent.setConversationId(undefined);
    sentPageRef.current = undefined;
    setPinned(undefined);
    setPageIncluded(true);
    setInput("");
    clearAttachments();
    focusComposer();
  };

  const expand = () => {
    agent.close();
    navigate(
      agent.conversationId
        ? `/agent?c=${encodeURIComponent(agent.conversationId)}`
        : "/agent",
    );
  };

  const blockCtx = React.useMemo<Omit<BlockContext, "runId" | "streaming">>(
    () => ({ client, onDecide: conv.decide }),
    [client, conv.decide],
  );

  const chips: ContextChip[] = [];
  if (pageIncluded && current) {
    chips.push({
      key: "page",
      icon: resourceCrumb?.icon ?? current.icon ?? PanelsTopLeft,
      label:
        resourceCrumb && resourceCrumb !== current
          ? `${resourceCrumb.label} › ${current.label}`
          : current.label,
      title: page.title,
      onRemove: () => setPageIncluded(false),
    });
  }
  if (pinned) {
    chips.push({
      key: "pinned",
      icon:
        pinned.items?.[0]?.kind === "DashboardSection"
          ? LayoutDashboard
          : pinned.items?.[0]?.kind === "TerminalSession"
            ? SquareTerminal
            : pinned.items && pinned.items.length > 0
              ? ScrollText
              : Box,
      label: pinned.label,
      title: pinned.label,
      onRemove: () => setPinned(undefined),
    });
  }

  const subject =
    pinned?.label ?? resourceCrumb?.label ?? current?.label ?? "this page";
  const suggestions = React.useMemo(
    () => getSuggestions({ trail, pathname, pinned, details, conditions }),
    [trail, pathname, pinned, details, conditions],
  );

  const onRetry = conv.retryInput
    ? () => {
        transcript.stickRef.current = true;
        void conv.send(conv.retryInput!);
      }
    : undefined;

  return (
    <>
      <DockHeader
        title={conversation?.title ?? "New conversation"}
        onExpand={expand}
        onClose={agent.close}
        status={
          isActive && (
            <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600">
              <StatusDot tone={awaitingApproval ? "amber" : "blue"} pulse />
              {awaitingApproval ? "Needs approval" : "Working"}
            </span>
          )
        }
      >
        <HistoryMenu
          conversations={conv.conversations ?? []}
          selected={agent.conversationId}
          onSelect={(id) => {
            if (id !== agent.conversationId) clearAttachments();
            agent.setConversationId(id);
            sentPageRef.current = undefined;
            setPinned(undefined);
          }}
          onShowAll={() => {
            agent.close();
            navigate("/agent");
          }}
        />
        <HeaderButton label="New conversation" onClick={newChat}>
          <SquarePen size={15} />
        </HeaderButton>
      </DockHeader>

      {props.banner && <div className="px-3 pt-2">{props.banner}</div>}

      <div className="relative flex min-h-0 flex-1 flex-col">
        {isEmpty ? (
          <div className="flex min-h-0 flex-1 flex-col justify-end overflow-y-auto overscroll-contain px-4 pb-3 pt-6">
            <motion.div
              key={subject}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, ease }}
            >
              <h3 className="text-base font-bold tracking-tight text-slate-900">
                Ask about{" "}
                <span className="break-words text-slate-600">{subject}</span>
              </h3>
              <p className="mt-1 text-xs leading-5 text-slate-500">
                The agent sees what you are looking at, acts with your identity
                and asks before changing anything.
              </p>
              <div className="mt-4 flex flex-col gap-1.5">
                {suggestions.map((suggestion, idx) => (
                  <motion.button
                    key={suggestion.prompt}
                    type="button"
                    initial={{ opacity: 0, y: 6 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: 0.04 * idx, ease }}
                    onClick={() => void submit(suggestion.prompt)}
                    disabled={sending}
                    className="group flex w-full cursor-pointer items-start gap-2.5 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-left outline-none transition-[border-color,background-color,box-shadow] duration-200 hover:border-slate-300 hover:bg-slate-50/70 hover:shadow-card focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-wait disabled:opacity-60"
                  >
                    <span className="mt-px flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-500 transition-colors duration-200 group-hover:text-slate-800">
                      <suggestion.icon size={12} strokeWidth={2.25} />
                    </span>
                    <span className="min-w-0">
                      <span className="block text-xs font-semibold text-slate-800">
                        {suggestion.title}
                      </span>
                      <span className="mt-0.5 line-clamp-2 block text-xs leading-5 text-slate-500">
                        {suggestion.prompt}
                      </span>
                    </span>
                  </motion.button>
                ))}
              </div>
            </motion.div>
          </div>
        ) : (
          <div
            ref={transcript.scrollRef}
            onScroll={transcript.onScroll}
            className="min-h-0 flex-1 overflow-y-auto overscroll-contain [scrollbar-gutter:stable]"
          >
            <div
              ref={transcript.contentRef}
              className="flex w-full flex-col gap-5 px-4 pb-8 pt-4"
            >
              {busy && (switching || chat.messages.length === 0) ? (
                <TranscriptSkeleton />
              ) : (
                chat.messages.map((message) => {
                  const streaming =
                    isActive &&
                    message.id === conv.lastAssistantID &&
                    message.status === "streaming";
                  return (
                    <MessageView
                      key={message.id}
                      message={message}
                      ctx={blockCtx}
                      streaming={streaming}
                      run={streaming ? run : undefined}
                      isLast={message.id === conv.lastMessage?.id}
                      flash={false}
                      onRetry={
                        message.id === conv.retryMessageID ? onRetry : undefined
                      }
                    />
                  );
                })
              )}
              {conv.showOptimistic && conv.optimisticMessage && (
                <MessageView
                  key="optimistic"
                  message={conv.optimisticMessage}
                  ctx={blockCtx}
                  streaming={false}
                  isLast={false}
                  flash={false}
                  pending
                />
              )}
              {conv.showPendingAssistant && !busy && <PendingAssistant />}
            </div>
          </div>
        )}

        <div className="relative px-3 pb-3">
          <AnimatePresence>
            {!isEmpty && !transcript.atBottom && (
              <motion.button
                key="scroll"
                type="button"
                aria-label="Scroll to the latest message"
                initial={{ opacity: 0, y: 6, scale: 0.9 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: 6, scale: 0.9 }}
                transition={{ duration: 0.18 }}
                onClick={() => transcript.scrollToBottom()}
                className="absolute -top-11 left-1/2 z-10 flex h-8 w-8 -translate-x-1/2 cursor-pointer items-center justify-center rounded-full border border-slate-200 bg-white text-slate-600 shadow-raised transition-colors hover:text-slate-900"
              >
                <ArrowDown size={15} strokeWidth={2.25} />
              </motion.button>
            )}
          </AnimatePresence>
          {!isEmpty && (
            <div className="pointer-events-none absolute inset-x-0 -top-6 h-6 bg-gradient-to-t from-white to-transparent" />
          )}

          <AnimatePresence initial={false}>
            {awaitingApproval && !transcript.approvalVisible && (
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
                    Waiting for your approval
                  </span>
                  <Button
                    size="compact-xs"
                    variant="default"
                    onClick={transcript.scrollToApproval}
                  >
                    Review
                  </Button>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          <Composer
            value={input}
            onChange={setInput}
            attachments={attachments}
            uploads={uploads}
            previews={previews}
            onAddFiles={(files) => void addFiles(files)}
            onRemoveAttachment={removeAttachment}
            onSend={() => void submit(input)}
            onStop={() => void conv.stop()}
            isActive={isActive}
            sending={sending}
            placeholder={
              isEmpty ? `Ask about ${subject}…` : "Reply to the agent…"
            }
            header={chips.length > 0 ? <ContextChips chips={chips} /> : null}
            modelPicker={
              <ModelPicker
                client={client}
                info={info}
                onManage={() => setModelsOpened(true)}
              />
            }
            textareaRef={textareaRef}
            className="shadow-card"
          />
        </div>
      </div>

      <ModelsDrawer
        client={client}
        opened={modelsOpened}
        onClose={() => setModelsOpened(false)}
      />
    </>
  );
};

const renderChat = (props: ReadyChatProps) => <DockChat {...props} />;

const AgentDock = () => {
  const agent = useAgent();
  const navigate = useNavigate();
  const [chatMounted, setChatMounted] = React.useState(false);
  const visible = agent.opened && !agent.hidden;

  return (
    <aside
      ref={agent.setDockElement}
      aria-label="Agent"
      aria-hidden={!visible}
      inert={!visible}
      data-open={visible || undefined}
      onKeyDown={(event) => {
        if (event.key === "Escape") event.stopPropagation();
      }}
      className={twMerge(
        "agent-dock fixed bottom-0 right-0 top-[60px] z-[210] flex w-[var(--agent-dock-width)] flex-col border-l border-slate-200 bg-white",
      )}
    >
      <DockFrameContext.Provider value={setChatMounted}>
        {!chatMounted && (
          <DockHeader
            title="Agent"
            onExpand={() => {
              agent.close();
              navigate("/agent");
            }}
            onClose={agent.close}
          />
        )}
        {(visible || chatMounted) && (
          <AgentSurface compact renderChat={renderChat} />
        )}
      </DockFrameContext.Provider>
    </aside>
  );
};

export default AgentDock;
