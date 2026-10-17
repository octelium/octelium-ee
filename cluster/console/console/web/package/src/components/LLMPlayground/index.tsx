import type {
  FileInfo,
  Message as AgentMessage,
} from "@/apis/consoleagent/protocol";
import { Service } from "@/apis/corev1/corev1";
import Composer, { type Upload } from "@/pages/Agent/Composer";
import { readStorage, writeStorage } from "@/pages/Agent/environment";
import { saveBlob, toFileName } from "@/pages/Agent/files";
import { AgentMark, copyText, StatusDot } from "@/pages/Agent/ui";
import { useTranscript } from "@/pages/Agent/useTranscript";
import { getDomain } from "@/utils";
import { getServicePublicURL } from "@/utils/octelium";
import { ActionIcon, Menu, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowDown,
  Bot,
  Brain,
  ChevronDown,
  Code,
  Copy,
  Download,
  Ellipsis,
  ImageUp,
  MessageCircle,
  SlidersHorizontal,
  Sparkles,
  SquarePen,
  Table,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "@/lib/utils";
import Message from "./Message";
import { defaultEndpoint, messageText, supportsStreaming } from "./protocols";
import { runCompletion } from "./runner";
import Settings from "./Settings";
import {
  Attachment,
  ChatMessage,
  Part,
  Protocol,
  PROTOCOL_LABELS,
  protocolFromEnum,
  RequestOptions,
  Usage,
} from "./types";

type Session = {
  protocol: Protocol;
  messages: ChatMessage[];
  input: string;
  endpoint: string;
  options: RequestOptions;
};

const sessions = new Map<string, Session>();

const settingsStorageKey = "octelium-console-llm-playground-settings";
const settingsWidth = 300;
const wideWidth = 720;
const ease = [0.22, 1, 0.36, 1] as const;
const noMessages: AgentMessage[] = [];

const suggestions = [
  {
    icon: MessageCircle,
    title: "Say hello",
    prompt:
      "Introduce yourself in two sentences and tell me which model you are.",
  },
  {
    icon: Brain,
    title: "Reason it out",
    prompt:
      "A bat and a ball cost $1.10 in total. The bat costs $1.00 more than the ball. How much does the ball cost?",
  },
  {
    icon: Table,
    title: "Format a table",
    prompt: "Compare TCP and UDP in a short Markdown table.",
  },
  {
    icon: Code,
    title: "Write code",
    prompt:
      "Write a Go function that reverses a UTF-8 string, with a short test.",
  },
];

const getLLMConfig = (service: Service) => {
  const type = service.spec?.config?.type;
  return type?.oneofKind === "llm" ? type.llm : undefined;
};

const getConfiguredModel = (service: Service) => {
  const model = getLLMConfig(service)?.model;
  return model?.type.oneofKind === "value" ? model.type.value : "";
};

const defaultOptions = (
  service: Service,
  protocol: Protocol,
): RequestOptions => ({
  model: getConfiguredModel(service),
  systemPrompt: "",
  temperature: "",
  topP: "",
  maxTokens: "",
  reasoning: "",
  stream: supportsStreaming(protocol),
  apiShape: "chat",
});

const newMessage = (
  role: ChatMessage["role"],
  parts: Part[] = [],
  attachments: Attachment[] = [],
): ChatMessage => ({
  id: crypto.randomUUID(),
  role,
  parts,
  attachments,
  createdAt: Date.now(),
});

const readFile = (file: File) =>
  new Promise<Attachment>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () =>
      resolve({
        id: crypto.randomUUID(),
        name: file.name,
        mime: file.type || "application/octet-stream",
        dataURL: String(reader.result ?? ""),
      });
    reader.onerror = () => reject(new Error("The file could not be read."));
    reader.readAsDataURL(file);
  });

const attachmentSize = (attachment: Attachment) =>
  Math.floor(
    ((attachment.dataURL.length - attachment.dataURL.indexOf(",") - 1) * 3) / 4,
  );

const getErrorMessage = (error: unknown) => {
  if (error instanceof Error) return error.message;
  return "The inference request failed.";
};

const isAbort = (error: unknown) =>
  error instanceof DOMException && error.name === "AbortError";

const hasFiles = (event: React.DragEvent) =>
  Array.from(event.dataTransfer.types).includes("Files");

const findLastUser = (messages: ChatMessage[]) => {
  for (let idx = messages.length - 1; idx >= 0; idx--) {
    if (messages[idx].role === "user") return idx;
  }
  return -1;
};

const mergeUsage = (current: Usage | undefined, next: Usage): Usage => {
  const ret: Usage = { ...current };
  for (const [name, value] of Object.entries(next) as [keyof Usage, number][]) {
    if (value !== undefined) ret[name] = value;
  }
  if (next.totalTokens === undefined && ret.inputTokens !== undefined) {
    ret.totalTokens = ret.inputTokens + (ret.outputTokens ?? 0);
  }
  return ret;
};

const stopLast = (messages: ChatMessage[]) =>
  messages.map((message, idx) =>
    idx === messages.length - 1 &&
    message.role === "assistant" &&
    message.completedAt === undefined
      ? { ...message, stopped: true, completedAt: Date.now() }
      : message,
  );

const transcriptMarkdown = (arg: {
  title: string;
  endpoint: string;
  options: RequestOptions;
  messages: ChatMessage[];
}) => {
  const ret = [
    `# ${arg.title}`,
    [
      `- Endpoint: ${arg.endpoint}`,
      `- Model: ${arg.options.model || "unset"}`,
      ...(arg.options.systemPrompt.trim()
        ? [`- System prompt: ${arg.options.systemPrompt.trim()}`]
        : []),
    ].join("\n"),
  ];
  for (const message of arg.messages) {
    const text = messageText(message);
    ret.push(
      `## ${message.role === "user" ? "You" : (message.model ?? "Assistant")}\n\n${text || (message.error ? `_${message.error}_` : "")}`,
    );
  }
  return `${ret.join("\n\n")}\n`;
};

const useContainerWidth = () => {
  const ref = React.useRef<HTMLDivElement>(null);
  const [width, setWidth] = React.useState(0);

  React.useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    setWidth(el.clientWidth);
    const observer = new ResizeObserver((entries) =>
      setWidth(entries[0].contentRect.width),
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return { ref, width };
};

const ToolbarButton = (props: {
  label: string;
  onClick: () => void;
  active?: boolean;
  children: React.ReactNode;
}) => (
  <Tooltip label={props.label} withArrow>
    <ActionIcon
      variant={props.active ? "light" : "subtle"}
      color="gray"
      aria-label={props.label}
      aria-pressed={props.active}
      onClick={props.onClick}
      className="relative"
    >
      {props.children}
    </ActionIcon>
  </Tooltip>
);

const LLMPlayground = (props: { service: Service }) => {
  const { service } = props;
  const key = service.metadata?.uid || service.metadata?.name || "";
  const protocol = protocolFromEnum(getLLMConfig(service)?.protocol);
  const serviceEndpoint = defaultEndpoint(
    getServicePublicURL(service, getDomain()),
    protocol,
  );
  const configuredModel = getConfiguredModel(service);
  const title = service.metadata?.displayName || service.metadata?.name || "";

  const [initial] = React.useState<Session>(() => {
    const cached = sessions.get(key);
    if (cached?.protocol === protocol) return cached;
    return {
      protocol,
      messages: [],
      input: "",
      endpoint: serviceEndpoint,
      options: defaultOptions(service, protocol),
    };
  });

  const [messages, setMessages] = React.useState(initial.messages);
  const [input, setInput] = React.useState(initial.input);
  const [endpoint, setEndpoint] = React.useState(initial.endpoint);
  const [options, setOptions] = React.useState(initial.options);
  const [attachments, setAttachments] = React.useState<Attachment[]>([]);
  const [uploads, setUploads] = React.useState<Upload[]>([]);
  const [pending, setPending] = React.useState(false);
  const [modelError, setModelError] = React.useState<string>();
  const [pinned, setPinned] = React.useState(
    () => readStorage(settingsStorageKey) !== "hidden",
  );
  const [overlay, setOverlay] = React.useState(false);
  const [dragging, setDragging] = React.useState(false);

  const { ref: rootRef, width } = useContainerWidth();
  const abortRef = React.useRef<AbortController | null>(null);
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);
  const modelRef = React.useRef<HTMLInputElement>(null);
  const overlayRef = React.useRef<HTMLElement>(null);
  const dragDepthRef = React.useRef(0);

  const wide = width >= wideWidth;
  const settingsOpen = wide ? pinned : overlay;
  const isEmpty = messages.length === 0;
  const defaults = defaultOptions(service, protocol);
  const customized =
    endpoint !== serviceEndpoint ||
    (Object.keys(defaults) as (keyof RequestOptions)[]).some(
      (option) => option !== "model" && options[option] !== defaults[option],
    );

  const {
    scrollRef,
    contentRef,
    stickRef,
    atBottom,
    scrolled,
    onScroll,
    scrollToBottom,
    reset: resetTranscript,
  } = useTranscript({
    mounted: !isEmpty,
    awaitingApproval: false,
    messages: noMessages,
  });

  React.useEffect(() => {
    sessions.set(key, { protocol, messages, input, endpoint, options });
  }, [key, protocol, messages, input, endpoint, options]);

  React.useEffect(
    () => () => {
      const controller = abortRef.current;
      if (!controller) return;
      controller.abort();
      const snapshot = sessions.get(key);
      if (snapshot) {
        sessions.set(key, {
          ...snapshot,
          messages: stopLast(snapshot.messages),
        });
      }
    },
    [key],
  );

  React.useEffect(() => {
    if (overlay) overlayRef.current?.focus();
  }, [overlay]);

  const focusComposer = () =>
    requestAnimationFrame(() => textareaRef.current?.focus());

  const openSettings = (focusModel = false) => {
    if (wide) {
      setPinned(true);
      writeStorage(settingsStorageKey, "visible");
    } else {
      setOverlay(true);
    }
    if (focusModel) {
      window.setTimeout(() => modelRef.current?.focus(), 320);
    }
  };

  const closeSettings = () => {
    if (wide) {
      setPinned(false);
      writeStorage(settingsStorageKey, "hidden");
    } else {
      setOverlay(false);
    }
  };

  const setOption = <K extends keyof RequestOptions>(
    option: K,
    value: RequestOptions[K],
  ) => {
    if (option === "model") setModelError(undefined);
    setOptions((current) => ({ ...current, [option]: value }));
  };

  const appendPart = (
    id: string,
    kind: Part["kind"],
    text: string,
    name?: string,
  ) =>
    setMessages((items) =>
      items.map((item) => {
        if (item.id !== id) return item;
        const parts = [...item.parts];
        const last = parts.at(-1);
        const at = Date.now();

        if (kind === "tool") {
          if (last?.kind === "tool" && !name) {
            parts[parts.length - 1] = { ...last, input: last.input + text };
          } else {
            parts.push({ kind: "tool", name: name ?? "tool", input: text, at });
          }
          return { ...item, parts };
        }

        if (last?.kind === kind) {
          parts[parts.length - 1] = { ...last, text: last.text + text };
        } else {
          parts.push({ kind, text, at });
        }
        return { ...item, parts };
      }),
    );

  const patchMessage = (id: string, patch: Partial<ChatMessage>) =>
    setMessages((items) =>
      items.map((item) => (item.id === id ? { ...item, ...patch } : item)),
    );

  const ensureModel = () => {
    if (options.model.trim()) return true;
    setModelError("Enter the model to send the request to");
    openSettings(true);
    return false;
  };

  const run = async (history: ChatMessage[]) => {
    const assistant = newMessage("assistant");
    setMessages([...history, assistant]);
    setPending(true);
    stickRef.current = true;
    requestAnimationFrame(() => scrollToBottom("auto"));

    const controller = new AbortController();
    abortRef.current = controller;
    const startedAt = performance.now();
    let firstTokenAt: number | undefined;
    let stopped = false;
    let error: string | undefined;

    try {
      await runCompletion({
        endpoint,
        protocol,
        options,
        messages: history,
        signal: controller.signal,
        onEvent: (event) => {
          switch (event.type) {
            case "text":
              firstTokenAt ??= performance.now();
              appendPart(assistant.id, "text", event.text);
              break;
            case "thinking":
              firstTokenAt ??= performance.now();
              appendPart(assistant.id, "thinking", event.text);
              break;
            case "tool":
              firstTokenAt ??= performance.now();
              appendPart(
                assistant.id,
                "tool",
                event.input,
                event.name || undefined,
              );
              break;
            case "usage":
              setMessages((items) =>
                items.map((item) =>
                  item.id === assistant.id
                    ? { ...item, usage: mergeUsage(item.usage, event.usage) }
                    : item,
                ),
              );
              break;
            case "model":
              patchMessage(assistant.id, { model: event.model });
              break;
            case "finish":
              patchMessage(assistant.id, { finishReason: event.reason });
              break;
          }
        },
      });
    } catch (requestError) {
      if (isAbort(requestError)) {
        stopped = true;
      } else {
        error = getErrorMessage(requestError);
      }
    } finally {
      patchMessage(assistant.id, {
        completedAt: Date.now(),
        latencyMs: performance.now() - startedAt,
        ...(firstTokenAt !== undefined
          ? { ttftMs: firstTokenAt - startedAt }
          : {}),
        ...(stopped ? { stopped } : {}),
        ...(error ? { error } : {}),
      });
      if (abortRef.current === controller) {
        abortRef.current = null;
        setPending(false);
      }
    }
  };

  const send = async () => {
    const value = input.trim();
    if ((!value && attachments.length === 0) || pending) return;
    if (uploads.length > 0 || !ensureModel()) return;

    const user = newMessage(
      "user",
      [{ kind: "text", text: value }],
      attachments,
    );
    setInput("");
    setAttachments([]);
    await run([...messages, user]);
  };

  const regenerate = async () => {
    if (pending || !ensureModel()) return;
    const index = findLastUser(messages);
    if (index === -1) return;
    await run(messages.slice(0, index + 1));
  };

  const stop = () => abortRef.current?.abort();

  const newChat = () => {
    abortRef.current?.abort();
    setMessages([]);
    resetTranscript();
    focusComposer();
  };

  const copyTranscript = async () => {
    const text = messages
      .map(
        (item) =>
          `${item.role === "user" ? "You" : "Assistant"}:\n${messageText(item)}`,
      )
      .join("\n\n");
    if (await copyText(text)) toast.success("The transcript is copied");
  };

  const exportTranscript = () =>
    saveBlob(
      new Blob([transcriptMarkdown({ title, endpoint, options, messages })], {
        type: "text/markdown",
      }),
      toFileName(`${title}-playground`, "md"),
    );

  const addFiles = async (files: File[]) => {
    const images = files.filter((file) => file.type.startsWith("image/"));
    if (images.length < files.length) {
      toast.error("Only images can be attached to a prompt");
    }
    for (const file of images) {
      const uploadKey = `${file.name}-${file.size}-${Math.random()}`;
      setUploads((current) => [
        ...current,
        { key: uploadKey, name: file.name },
      ]);
      try {
        const attachment = await readFile(file);
        setAttachments((current) => [...current, attachment]);
      } catch (err) {
        toast.error(`Could not read ${file.name}: ${getErrorMessage(err)}`);
      } finally {
        setUploads((current) => current.filter((itm) => itm.key !== uploadKey));
      }
    }
  };

  const fileInfos = React.useMemo<FileInfo[]>(
    () =>
      attachments.map((attachment) => ({
        id: attachment.id,
        name: attachment.name,
        mimeType: attachment.mime,
        size: attachmentSize(attachment),
        path: "",
        createdAt: "",
      })),
    [attachments],
  );

  const previews = React.useMemo(
    () =>
      Object.fromEntries(
        attachments.map((attachment) => [attachment.id, attachment.dataURL]),
      ),
    [attachments],
  );

  const lastUserIndex = findLastUser(messages);

  const modelButton = (
    <button
      type="button"
      aria-label="Change the model"
      onClick={() => openSettings(true)}
      className={twMerge(
        "flex h-8 min-w-0 max-w-[220px] cursor-pointer items-center gap-1.5 rounded-full px-2.5 transition-colors duration-150 hover:bg-slate-100 hover:text-slate-900",
        options.model.trim() ? "text-slate-600" : "text-amber-700",
      )}
    >
      <Sparkles size={13} className="shrink-0" />
      <span className="truncate text-xs font-semibold">
        {options.model.trim() || "Set the model"}
      </span>
      <ChevronDown size={12} strokeWidth={2.5} className="shrink-0" />
    </button>
  );

  const settings = (
    <Settings
      protocol={protocol}
      options={options}
      onOption={setOption}
      endpoint={endpoint}
      defaultEndpoint={serviceEndpoint}
      onEndpoint={setEndpoint}
      configuredModel={configuredModel}
      modelError={modelError}
      modelRef={modelRef}
      onReset={() => {
        setOptions(defaults);
        setEndpoint(serviceEndpoint);
        setModelError(undefined);
      }}
      onClose={closeSettings}
    />
  );

  return (
    <div
      ref={rootRef}
      className="relative flex h-[calc(100dvh-11.125rem)] min-h-[30rem] w-full overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card"
    >
      <section
        aria-label="Conversation"
        className="relative flex min-w-0 flex-1 flex-col"
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
            "relative z-10 flex h-12 shrink-0 items-center gap-1 border-b pl-3 pr-2 transition-colors duration-200",
            scrolled && !isEmpty ? "border-slate-200/80" : "border-transparent",
          )}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <span className="shrink-0 rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-micro font-semibold text-slate-600">
              {PROTOCOL_LABELS[protocol]}
              {protocol === "openai" &&
                (options.apiShape === "responses"
                  ? " · Responses"
                  : " · Chat Completions")}
            </span>
            <span
              className="hidden min-w-0 truncate font-mono text-micro text-slate-500 sm:block"
              title={endpoint}
            >
              {endpoint.replace(/^https?:\/\//, "")}
            </span>
            {pending && (
              <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600">
                <StatusDot tone="blue" pulse />
                <span className="hidden sm:inline">Generating</span>
              </span>
            )}
          </div>
          {!isEmpty && (
            <>
              <ToolbarButton label="New conversation" onClick={newChat}>
                <SquarePen size={15} />
              </ToolbarButton>
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
                    leftSection={<Copy size={13} />}
                    onClick={() => void copyTranscript()}
                  >
                    Copy the transcript
                  </Menu.Item>
                  <Menu.Item
                    leftSection={<Download size={13} />}
                    onClick={exportTranscript}
                  >
                    Export as Markdown
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </>
          )}
          <ToolbarButton
            label={
              settingsOpen ? "Hide the request settings" : "Request settings"
            }
            onClick={() => (settingsOpen ? closeSettings() : openSettings())}
            active={settingsOpen}
          >
            <SlidersHorizontal size={15} />
            {customized && !settingsOpen && (
              <span className="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-blue-500" />
            )}
          </ToolbarButton>
        </header>

        <div
          className={twMerge(
            "relative flex min-h-0 flex-1 flex-col",
            isEmpty &&
              "justify-center-safe overflow-y-auto overscroll-contain px-3 py-6 sm:px-6",
          )}
        >
          {isEmpty ? (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, ease }}
              className="mx-auto mb-6 flex w-full max-w-3xl flex-col items-center text-center"
            >
              <AgentMark size="lg" icon={Bot} />
              <h2 className="mt-4 max-w-full break-words text-xl font-bold tracking-tight text-slate-900 sm:text-2xl">
                Chat with {title}
              </h2>
              <p className="mt-1.5 max-w-md text-body leading-6 text-slate-500">
                Requests go through Octelium with your identity, so the
                Policies, plugins and guardrails of this Service apply. The
                provider credentials stay on the Cluster.
              </p>
            </motion.div>
          ) : (
            <div
              ref={scrollRef}
              onScroll={onScroll}
              className="min-h-0 flex-1 overflow-y-auto overscroll-contain [scrollbar-gutter:stable]"
            >
              <div
                ref={contentRef}
                className="mx-auto flex w-full max-w-3xl flex-col gap-5 px-4 pb-10 pt-4 sm:px-6"
              >
                {messages.map((message, idx) => (
                  <Message
                    key={message.id}
                    message={message}
                    streaming={pending && idx === messages.length - 1}
                    isLast={idx === messages.length - 1}
                    onRegenerate={
                      !pending &&
                      message.role === "assistant" &&
                      idx === messages.length - 1 &&
                      lastUserIndex !== -1
                        ? () => void regenerate()
                        : undefined
                    }
                  />
                ))}
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
                  {pending && (
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
            <Composer
              value={input}
              onChange={setInput}
              attachments={fileInfos}
              uploads={uploads}
              previews={previews}
              onAddFiles={(files) => void addFiles(files)}
              onRemoveAttachment={(id) =>
                setAttachments((current) =>
                  current.filter((itm) => itm.id !== id),
                )
              }
              onSend={() => void send()}
              onStop={stop}
              isActive={pending}
              sending={false}
              placeholder={
                isEmpty ? "Send a prompt to the model…" : "Reply to the model…"
              }
              modelPicker={modelButton}
              textareaRef={textareaRef}
              accept="image/*"
              attachLabel="Attach images"
              className="shadow-card"
            />
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
                    <span className="mt-0.5 line-clamp-2 block text-xs leading-5 text-slate-500">
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
                <ImageUp size={20} />
              </span>
              <p className="mt-3 text-body font-semibold text-slate-800">
                Drop images to attach them
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </section>

      {wide ? (
        <motion.aside
          initial={false}
          animate={{ width: pinned ? settingsWidth : 0 }}
          transition={{ duration: 0.3, ease }}
          className="h-full shrink-0 overflow-hidden"
          aria-label="Request settings"
          aria-hidden={!pinned}
          inert={!pinned}
        >
          <div
            className="h-full border-l border-slate-200 bg-slate-50/70"
            style={{ width: settingsWidth }}
          >
            {settings}
          </div>
        </motion.aside>
      ) : (
        <AnimatePresence>
          {overlay && (
            <>
              <motion.div
                key="backdrop"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
                onClick={() => setOverlay(false)}
                className="absolute inset-0 z-30 bg-[rgba(15,23,42,0.28)] backdrop-blur-[1px]"
              />
              <motion.aside
                key="settings"
                ref={overlayRef}
                tabIndex={-1}
                aria-label="Request settings"
                initial={{ x: "100%" }}
                animate={{ x: 0 }}
                exit={{ x: "100%" }}
                transition={{ duration: 0.3, ease }}
                onKeyDown={(event) => {
                  if (
                    event.key !== "Escape" ||
                    event.defaultPrevented ||
                    (event.target as HTMLElement).dataset
                      .mantineStopPropagation === "true"
                  ) {
                    return;
                  }
                  event.preventDefault();
                  setOverlay(false);
                }}
                className="absolute inset-y-0 right-0 z-40 w-[min(320px,92%)] border-l border-slate-200 bg-slate-50 shadow-overlay outline-none"
              >
                {settings}
              </motion.aside>
            </>
          )}
        </AnimatePresence>
      )}
    </div>
  );
};

export default LLMPlayground;
