import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { getDomain } from "@/utils";
import { getServicePublicURL } from "@/utils/octelium";
import {
  ActionIcon,
  Badge,
  Button,
  Drawer,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  TextInput,
  Textarea,
  Tooltip,
} from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  Bot,
  Check,
  ChevronDown,
  CircleAlert,
  Copy,
  ImagePlus,
  Paperclip,
  RotateCcw,
  Send,
  Settings2,
  Sparkles,
  Square,
  X,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import Message from "./Message";
import { defaultEndpoint, messageText, supportsStreaming } from "./protocols";
import { runCompletion } from "./runner";
import {
  Attachment,
  ChatMessage,
  Part,
  Protocol,
  PROTOCOL_LABELS,
  PROTOCOL_MODEL_PLACEHOLDER,
  protocolFromEnum,
  RequestOptions,
} from "./types";

const PROTOCOL_COLORS: Record<Protocol, string> = {
  openai: "teal",
  anthropic: "orange",
  gemini: "blue",
  bedrock: "grape",
};

const REASONING_HINT: Record<Protocol, string> = {
  openai: "low / medium / high",
  anthropic: "Thinking token budget",
  gemini: "Thinking token budget",
  bedrock: "Not supported for Converse",
};

const getLLMConfig = (service: Service) => {
  const type = service.spec?.config?.type;
  return type?.oneofKind === "llm" ? type.llm : undefined;
};

const getConfiguredModel = (service: Service) => {
  const model = getLLMConfig(service)?.model;
  return model?.type.oneofKind === "value" ? model.type.value : "";
};

const newMessage = (
  role: ChatMessage["role"],
  parts: Part[] = [],
  attachments: Attachment[] = [],
): ChatMessage => ({
  id: crypto.randomUUID(),
  role,
  parts,
  attachments,
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

const getErrorMessage = (error: unknown) => {
  if (error instanceof DOMException && error.name === "AbortError") {
    return "The request was stopped.";
  }
  if (error instanceof Error) return error.message;
  return "The inference request failed.";
};

const LLMPlayground = (props: { service: Service }) => {
  const { service } = props;
  const protocol = protocolFromEnum(getLLMConfig(service)?.protocol);
  const isPublic = service.spec?.isPublic === true;
  const canStream = supportsStreaming(protocol);

  const [opened, setOpened] = React.useState(false);
  const [messages, setMessages] = React.useState<ChatMessage[]>([]);
  const [input, setInput] = React.useState("");
  const [attachments, setAttachments] = React.useState<Attachment[]>([]);
  const [pending, setPending] = React.useState(false);
  const [error, setError] = React.useState<string>();
  const [copied, setCopied] = React.useState(false);
  const [advanced, setAdvanced] = React.useState(false);
  const [endpoint, setEndpoint] = React.useState(() =>
    defaultEndpoint(getServicePublicURL(service, getDomain()), protocol),
  );
  const [options, setOptions] = React.useState<RequestOptions>(() => ({
    model: getConfiguredModel(service),
    systemPrompt: "",
    temperature: "",
    topP: "",
    maxTokens: "",
    reasoning: "",
    stream: supportsStreaming(protocol),
    apiShape: "chat",
  }));

  const abortRef = React.useRef<AbortController | null>(null);
  const transcriptRef = React.useRef<HTMLDivElement>(null);
  const fileRef = React.useRef<HTMLInputElement>(null);

  React.useEffect(() => {
    const nextProtocol = protocolFromEnum(getLLMConfig(service)?.protocol);
    setEndpoint(
      defaultEndpoint(getServicePublicURL(service, getDomain()), nextProtocol),
    );
    setOptions((current) => ({
      ...current,
      model: getConfiguredModel(service),
      apiShape: "chat",
      stream: supportsStreaming(nextProtocol),
    }));
    setMessages([]);
    setInput("");
    setAttachments([]);
    setError(undefined);
  }, [service]);

  React.useEffect(() => {
    transcriptRef.current?.scrollTo({
      top: transcriptRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages, pending]);

  const setOption = <K extends keyof RequestOptions>(
    key: K,
    value: RequestOptions[K],
  ) => setOptions((current) => ({ ...current, [key]: value }));

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

        if (kind === "tool") {
          if (last?.kind === "tool" && !name) {
            parts[parts.length - 1] = { ...last, input: last.input + text };
          } else {
            parts.push({ kind: "tool", name: name ?? "tool", input: text });
          }
          return { ...item, parts };
        }

        if (last?.kind === kind) {
          parts[parts.length - 1] = { ...last, text: last.text + text };
        } else {
          parts.push(
            kind === "thinking"
              ? { kind: "thinking", text }
              : { kind: "text", text },
          );
        }
        return { ...item, parts };
      }),
    );

  const patchMessage = (id: string, patch: Partial<ChatMessage>) =>
    setMessages((items) =>
      items.map((item) => (item.id === id ? { ...item, ...patch } : item)),
    );

  const run = async (history: ChatMessage[]) => {
    if (!options.model.trim()) {
      setError("Enter a model name before sending a request.");
      setAdvanced(true);
      return;
    }

    setError(undefined);
    const assistant = newMessage("assistant");
    setMessages([...history, assistant]);
    setPending(true);

    const controller = new AbortController();
    abortRef.current = controller;
    const startedAt = performance.now();
    let firstTokenAt: number | undefined;

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
              appendPart(assistant.id, "tool", event.input, event.name || undefined);
              break;
            case "usage":
              patchMessage(assistant.id, { usage: event.usage });
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
      patchMessage(assistant.id, { error: getErrorMessage(requestError) });
      setError(getErrorMessage(requestError));
    } finally {
      patchMessage(assistant.id, {
        latencyMs: performance.now() - startedAt,
        ...(firstTokenAt !== undefined
          ? { ttftMs: firstTokenAt - startedAt }
          : {}),
      });
      abortRef.current = null;
      setPending(false);
    }
  };

  const send = async () => {
    const value = input.trim();
    if ((!value && attachments.length === 0) || pending) return;

    const user = newMessage("user", [{ kind: "text", text: value }], attachments);
    setInput("");
    setAttachments([]);
    await run([...messages, user]);
  };

  const regenerate = async () => {
    if (pending) return;
    const lastUser = [...messages]
      .reverse()
      .find((item) => item.role === "user");
    if (!lastUser) return;
    const index = messages.findIndex((item) => item.id === lastUser.id);
    await run(messages.slice(0, index + 1));
  };

  const stop = () => abortRef.current?.abort();

  const clear = () => {
    if (pending) return;
    setMessages([]);
    setError(undefined);
  };

  const copyTranscript = async () => {
    const text = messages
      .map(
        (item) =>
          `${item.role === "user" ? "You" : "Assistant"}:\n${messageText(item)}`,
      )
      .join("\n\n");
    await navigator.clipboard.writeText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  const addFiles = async (files: FileList | File[] | null) => {
    if (!files) return;
    const loaded = await Promise.all(Array.from(files).map(readFile));
    setAttachments((current) => [...current, ...loaded]);
  };

  if (service.spec?.mode !== Service_Spec_Mode.LLM || !isPublic) return null;

  return (
    <>
      <Button
        type="button"
        size="compact-xs"
        variant="default"
        leftSection={<Sparkles size={12} strokeWidth={2.5} />}
        onClick={() => setOpened(true)}
      >
        LLM playground
      </Button>
      <Drawer
        opened={opened}
        onClose={() => setOpened(false)}
        position="right"
        size="min(880px, 100vw)"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <Bot size={15} className="shrink-0 text-slate-400" />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              LLM playground
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {service.metadata?.name}
            </span>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            height: "calc(100dvh - 56px)",
            padding: "16px",
            backgroundColor: "#f8fafc",
          },
          content: { borderLeft: "1px solid #e2e8f0" },
        }}
      >
        <div className="flex h-full flex-col gap-3">
          <div className="shrink-0 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-[0_1px_4px_rgba(15,23,42,0.04)]">
            <div className="flex flex-wrap items-center gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Sparkles size={16} />
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-[0.82rem] font-bold text-slate-800">
                  Talk to this LLM Service
                </p>
                <p className="mt-0.5 text-[0.7rem] font-semibold leading-5 text-slate-500">
                  Requests go through Octelium, so the Policies, Plugins and
                  guardrails of this Service apply. Provider credentials stay on
                  the Cluster.
                </p>
              </div>
              <Badge variant="light" color={PROTOCOL_COLORS[protocol]}>
                {PROTOCOL_LABELS[protocol]}
              </Badge>
            </div>

            <div className="mt-3 flex flex-wrap items-end gap-2">
              <TextInput
                size="xs"
                className="min-w-[240px] flex-1"
                label="Service endpoint"
                value={endpoint}
                onChange={(event) => setEndpoint(event.currentTarget.value)}
              />
              <TextInput
                size="xs"
                className="min-w-[190px]"
                label="Model"
                required
                placeholder={PROTOCOL_MODEL_PLACEHOLDER[protocol]}
                value={options.model}
                onChange={(event) => setOption("model", event.currentTarget.value)}
              />
              {protocol === "openai" && (
                <SegmentedControl
                  size="xs"
                  value={options.apiShape}
                  onChange={(value) =>
                    setOption("apiShape", value as RequestOptions["apiShape"])
                  }
                  data={[
                    { label: "Chat", value: "chat" },
                    { label: "Responses", value: "responses" },
                  ]}
                />
              )}
              <Tooltip
                label={
                  canStream
                    ? "Stream the response"
                    : "The Bedrock Converse route is served without streaming here"
                }
                withArrow
              >
                <div>
                  <Switch
                    size="xs"
                    label="Stream"
                    disabled={!canStream}
                    checked={options.stream && canStream}
                    onChange={(event) =>
                      setOption("stream", event.currentTarget.checked)
                    }
                  />
                </div>
              </Tooltip>
            </div>
          </div>

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_4px_rgba(15,23,42,0.04)]">
            <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-100 px-4 py-2.5">
              <div className="flex items-center gap-2">
                <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500">
                  Conversation
                </span>
                {messages.length > 0 && (
                  <Badge size="sm" variant="light" color="gray">
                    {messages.length}
                  </Badge>
                )}
              </div>
              <div className="flex items-center gap-1">
                <Tooltip label={copied ? "Copied" : "Copy transcript"} withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    aria-label="Copy transcript"
                    onClick={copyTranscript}
                    disabled={!messages.length}
                  >
                    {copied ? <Check size={15} /> : <Copy size={15} />}
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Clear conversation" withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    aria-label="Clear conversation"
                    onClick={clear}
                    disabled={!messages.length || pending}
                  >
                    <RotateCcw size={15} />
                  </ActionIcon>
                </Tooltip>
              </div>
            </div>

            <div
              ref={transcriptRef}
              className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4"
            >
              {messages.length === 0 ? (
                <div className="flex flex-1 flex-col items-center justify-center px-8 text-center">
                  <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-500">
                    <Bot size={23} />
                  </span>
                  <p className="mt-3 text-sm font-bold text-slate-700">
                    Start a conversation
                  </p>
                  <p className="mt-1 max-w-sm text-xs font-semibold leading-5 text-slate-500">
                    Ask a question or attach an image. Use the request options to
                    set the sampling behavior and the reasoning budget.
                  </p>
                </div>
              ) : (
                messages.map((message, index) => (
                  <Message
                    key={message.id}
                    message={message}
                    streaming={pending && index === messages.length - 1}
                    onRegenerate={
                      !pending &&
                      message.role === "assistant" &&
                      index === messages.length - 1
                        ? regenerate
                        : undefined
                    }
                  />
                ))
              )}
            </div>

            {error && (
              <div className="mx-4 mb-2 flex shrink-0 items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
                <CircleAlert size={15} className="mt-0.5 shrink-0" />
                <span className="min-w-0 break-words">{error}</span>
              </div>
            )}

            <div className="shrink-0 border-t border-slate-100 p-3">
              {attachments.length > 0 && (
                <div className="mb-2 flex flex-wrap gap-1.5">
                  {attachments.map((attachment) => (
                    <span
                      key={attachment.id}
                      className="group relative inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 py-1 pl-1.5 pr-1 text-[0.65rem] font-semibold text-slate-600"
                    >
                      {attachment.mime.startsWith("image/") ? (
                        <img
                          src={attachment.dataURL}
                          alt=""
                          className="h-6 w-6 rounded object-cover"
                        />
                      ) : (
                        <Paperclip size={11} />
                      )}
                      <span className="max-w-[140px] truncate">
                        {attachment.name}
                      </span>
                      <button
                        type="button"
                        aria-label={`Remove ${attachment.name}`}
                        onClick={() =>
                          setAttachments((current) =>
                            current.filter((item) => item.id !== attachment.id),
                          )
                        }
                        className="flex h-4 w-4 cursor-pointer items-center justify-center rounded text-slate-400 transition-colors duration-300 hover:bg-slate-200 hover:text-slate-700"
                      >
                        <X size={10} strokeWidth={3} />
                      </button>
                    </span>
                  ))}
                </div>
              )}

              <Textarea
                aria-label="Message"
                placeholder="Ask the model something…"
                autosize
                minRows={2}
                maxRows={8}
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

              <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-1">
                  <input
                    ref={fileRef}
                    type="file"
                    accept="image/*"
                    multiple
                    hidden
                    onChange={(event) => {
                      void addFiles(event.currentTarget.files);
                      event.currentTarget.value = "";
                    }}
                  />
                  <Tooltip label="Attach an image" withArrow>
                    <ActionIcon
                      variant="subtle"
                      color="gray"
                      aria-label="Attach an image"
                      onClick={() => fileRef.current?.click()}
                    >
                      <ImagePlus size={15} />
                    </ActionIcon>
                  </Tooltip>
                  <Button
                    type="button"
                    variant="subtle"
                    size="compact-xs"
                    color="gray"
                    leftSection={<Settings2 size={13} />}
                    rightSection={
                      <ChevronDown
                        size={12}
                        className={twMerge(
                          "transition-transform duration-400",
                          advanced && "rotate-180",
                        )}
                      />
                    }
                    onClick={() => setAdvanced((value) => !value)}
                  >
                    Request options
                  </Button>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-[0.65rem] font-semibold text-slate-400">
                    Enter to send · Shift + Enter for a new line
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    leftSection={
                      pending ? <Square size={14} /> : <Send size={14} />
                    }
                    color={pending ? "red" : "dark"}
                    onClick={pending ? stop : () => void send()}
                  >
                    {pending ? "Stop" : "Send"}
                  </Button>
                </div>
              </div>
            </div>
          </div>

          <AnimatePresence initial={false}>
            {advanced && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: "auto", opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
                className="shrink-0 overflow-hidden"
              >
                <div className="rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-[0_1px_4px_rgba(15,23,42,0.04)]">
                  <div className="mb-2.5 flex items-center gap-2">
                    <Settings2 size={14} className="text-slate-500" />
                    <p className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-600">
                      Request options
                    </p>
                  </div>
                  <div className="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
                    <Textarea
                      size="xs"
                      className="sm:col-span-2 lg:col-span-4"
                      label="System prompt"
                      placeholder="You are a concise assistant."
                      autosize
                      minRows={2}
                      maxRows={5}
                      value={options.systemPrompt}
                      onChange={(event) =>
                        setOption("systemPrompt", event.currentTarget.value)
                      }
                    />
                    <NumberInput
                      size="xs"
                      label="Temperature"
                      placeholder="Provider default"
                      min={0}
                      max={2}
                      step={0.1}
                      decimalScale={2}
                      value={options.temperature}
                      onChange={(value) => setOption("temperature", String(value ?? ""))}
                    />
                    <NumberInput
                      size="xs"
                      label="Top P"
                      placeholder="Provider default"
                      min={0}
                      max={1}
                      step={0.05}
                      decimalScale={2}
                      value={options.topP}
                      onChange={(value) => setOption("topP", String(value ?? ""))}
                    />
                    <NumberInput
                      size="xs"
                      label="Max output tokens"
                      placeholder={
                        protocol === "anthropic" ? "4096" : "Provider default"
                      }
                      min={1}
                      allowDecimal={false}
                      value={options.maxTokens}
                      onChange={(value) => setOption("maxTokens", String(value ?? ""))}
                    />
                    <Select
                      size="xs"
                      label="Reasoning effort"
                      placeholder={REASONING_HINT[protocol]}
                      disabled={protocol === "bedrock"}
                      clearable
                      searchable
                      data={
                        protocol === "openai"
                          ? ["minimal", "low", "medium", "high"]
                          : ["1024", "4096", "8192", "16384", "32768"]
                      }
                      value={options.reasoning || null}
                      onChange={(value) => setOption("reasoning", value ?? "")}
                    />
                  </div>
                  <p className="mt-2.5 text-[0.66rem] font-semibold text-slate-400">
                    The Service's limits, Policies and guardrails still apply. Do
                    not paste secrets into prompts.
                  </p>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </Drawer>
    </>
  );
};

export default LLMPlayground;
