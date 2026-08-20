import { Service, Service_Spec_Mode, Service_Spec_Config_LLM_Protocol } from "@/apis/corev1/corev1";
import { getServicePublicURL } from "@/utils/octelium";
import { getDomain } from "@/utils";
import {
  ActionIcon,
  Badge,
  Button,
  Drawer,
  Group,
  SegmentedControl,
  Text,
  TextInput,
  Textarea,
  Tooltip,
} from "@mantine/core";
import {
  Bot,
  Check,
  CircleAlert,
  Copy,
  RotateCcw,
  Send,
  Settings2,
  Sparkles,
  Square,
  User,
} from "lucide-react";
import * as React from "react";

type Role = "user" | "assistant";
type ChatMessage = { id: string; role: Role; content: string };
type APIShape = "chat" | "responses";

const COOKIE_API_KEY = "octelium-cookie";

const fetchThroughOctelium = (input: RequestInfo | URL, init?: RequestInit) => {
  const headers = new Headers(
    init?.headers ?? (input instanceof Request ? input.headers : undefined),
  );
  headers.delete("authorization");
  headers.delete("x-api-key");
  headers.delete("api-key");
  return fetch(input, { ...init, credentials: "include", headers });
};

const getLLMConfig = (service: Service) => {
  const type = service.spec?.config?.type;
  return type?.oneofKind === "llm" ? type.llm : undefined;
};

const getProtocol = (service: Service) =>
  getLLMConfig(service)?.protocol === Service_Spec_Config_LLM_Protocol.ANTHROPIC
    ? "anthropic"
    : "openai";

const getConfiguredModel = (service: Service) => {
  const model = getLLMConfig(service)?.model;
  return model?.type.oneofKind === "value" ? model.type.value : "";
};

const getDefaultEndpoint = (service: Service, protocol: "openai" | "anthropic") => {
  const origin = getServicePublicURL(service, getDomain());
  return protocol === "openai" ? `${origin}/v1` : origin;
};

const getErrorMessage = (error: unknown) => {
  if (error instanceof DOMException && error.name === "AbortError") {
    return "The request was stopped.";
  }
  if (error instanceof Error) return error.message;
  return "The LLM request failed.";
};

const LLMPlayground = (props: { service: Service }) => {
  const { service } = props;
  const protocol = getProtocol(service);
  const isPublic = service.spec?.isPublic === true;
  const configuredModel = getConfiguredModel(service);
  const [opened, setOpened] = React.useState(false);
  const [messages, setMessages] = React.useState<ChatMessage[]>([]);
  const [input, setInput] = React.useState("");
  const [systemPrompt, setSystemPrompt] = React.useState("");
  const [model, setModel] = React.useState(configuredModel);
  const [temperature, setTemperature] = React.useState("");
  const [topP, setTopP] = React.useState("");
  const [maxTokens, setMaxTokens] = React.useState("");
  const [reasoning, setReasoning] = React.useState("");
  const [apiShape, setApiShape] = React.useState<APIShape>("chat");
  const [endpoint, setEndpoint] = React.useState(() =>
    getDefaultEndpoint(service, protocol),
  );
  const [advanced, setAdvanced] = React.useState(false);
  const [pending, setPending] = React.useState(false);
  const [error, setError] = React.useState<string>();
  const [copied, setCopied] = React.useState(false);
  const abortRef = React.useRef<AbortController | null>(null);
  const transcriptRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const nextProtocol = getProtocol(service);
    setModel(getConfiguredModel(service));
    setEndpoint(getDefaultEndpoint(service, nextProtocol));
    setApiShape("chat");
    setMessages([]);
    setInput("");
    setError(undefined);
  }, [service]);

  React.useEffect(() => {
    transcriptRef.current?.scrollTo({
      top: transcriptRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages, pending]);

  const updateAssistant = (id: string, content: string) => {
    setMessages((items) =>
      items.map((item) => (item.id === id ? { ...item, content } : item)),
    );
  };

  const streamOpenAI = async (
    requestMessages: ChatMessage[],
    assistantID: string,
    signal: AbortSignal,
  ) => {
    const { default: OpenAI } = await import("openai");
    const client = new OpenAI({
      apiKey: COOKIE_API_KEY,
      baseURL: endpoint.replace(/\/$/, ""),
      dangerouslyAllowBrowser: true,
      maxRetries: 0,
      fetch: fetchThroughOctelium,
      fetchOptions: { credentials: "include" },
    });
    const common = {
      model: model.trim(),
      stream: true as const,
      messages: [
        ...(systemPrompt.trim()
          ? [{ role: "system" as const, content: systemPrompt.trim() }]
          : []),
        ...requestMessages.map((item) => ({ role: item.role, content: item.content })),
      ],
      ...(temperature.trim() ? { temperature: Number(temperature) } : {}),
      ...(topP.trim() ? { top_p: Number(topP) } : {}),
      ...(maxTokens.trim() ? { max_tokens: Number(maxTokens) } : {}),
      ...(reasoning.trim() ? { reasoning_effort: reasoning.trim() } : {}),
    };
    let text = "";
    if (apiShape === "responses") {
      const stream = await (client.responses.create as any)({
        model: common.model,
        input: [
          ...(systemPrompt.trim()
            ? [{ role: "system", content: systemPrompt.trim() }]
            : []),
          ...requestMessages.map((item) => ({
            role: item.role,
            content: [{ type: "input_text", text: item.content }],
          })),
        ],
        stream: true,
        ...(temperature.trim() ? { temperature: Number(temperature) } : {}),
        ...(maxTokens.trim() ? { max_output_tokens: Number(maxTokens) } : {}),
        ...(reasoning.trim() ? { reasoning: { effort: reasoning.trim() } } : {}),
      }, { signal });
      for await (const event of stream as AsyncIterable<any>) {
        if (event?.type === "response.output_text.delta") {
          text += event.delta || "";
          updateAssistant(assistantID, text);
        }
      }
      return;
    }
    const stream = await (client.chat.completions.create as any)(common, { signal });
    for await (const chunk of stream as AsyncIterable<any>) {
      const delta = chunk?.choices?.[0]?.delta?.content;
      if (typeof delta === "string") {
        text += delta;
        updateAssistant(assistantID, text);
      }
    }
  };

  const streamAnthropic = async (
    requestMessages: ChatMessage[],
    assistantID: string,
    signal: AbortSignal,
  ) => {
    const { default: Anthropic } = await import("@anthropic-ai/sdk");
    const client = new Anthropic({
      apiKey: COOKIE_API_KEY,
      baseURL: endpoint.replace(/\/$/, ""),
      dangerouslyAllowBrowser: true,
      maxRetries: 0,
      fetch: fetchThroughOctelium,
      fetchOptions: { credentials: "include" },
    });
    const stream = await (client.messages.create as any)({
      model: model.trim(),
      max_tokens: Number(maxTokens) || 1024,
      ...(systemPrompt.trim() ? { system: systemPrompt.trim() } : {}),
      messages: requestMessages.map((item) => ({ role: item.role, content: item.content })),
      ...(temperature.trim() ? { temperature: Number(temperature) } : {}),
      ...(topP.trim() ? { top_p: Number(topP) } : {}),
      ...(reasoning.trim()
        ? { thinking: { type: "enabled", budget_tokens: Number(reasoning) || 1024 } }
        : {}),
      stream: true,
    }, { signal });
    let text = "";
    for await (const event of stream as AsyncIterable<any>) {
      if (event?.type === "content_block_delta" && event?.delta?.type === "text_delta") {
        text += event.delta.text || "";
        updateAssistant(assistantID, text);
      }
    }
  };

  const send = async () => {
    const value = input.trim();
    if (!value || pending) return;
    if (!model.trim()) {
      setError("Enter a model name before sending a request.");
      setAdvanced(true);
      return;
    }
    setError(undefined);
    setInput("");
    const userMessage: ChatMessage = {
      id: crypto.randomUUID(),
      role: "user",
      content: value,
    };
    const assistantMessage: ChatMessage = {
      id: crypto.randomUUID(),
      role: "assistant",
      content: "",
    };
    const nextMessages = [...messages, userMessage];
    setMessages([...nextMessages, assistantMessage]);
    setPending(true);
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      if (protocol === "anthropic") {
        await streamAnthropic(nextMessages, assistantMessage.id, controller.signal);
      } else {
        await streamOpenAI(nextMessages, assistantMessage.id, controller.signal);
      }
    } catch (requestError) {
      const message = getErrorMessage(requestError);
      setError(message);
      updateAssistant(assistantMessage.id, "");
    } finally {
      abortRef.current = null;
      setPending(false);
    }
  };

  const stop = () => abortRef.current?.abort();

  const clear = () => {
    if (pending) return;
    setMessages([]);
    setError(undefined);
  };

  const copyTranscript = async () => {
    const text = messages
      .map((item) => `${item.role === "user" ? "You" : "Assistant"}:\n${item.content}`)
      .join("\n\n");
    await navigator.clipboard.writeText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
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
        size="min(820px, 100vw)"
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
        transitionProps={{ transition: "slide-left", duration: 500, exitDuration: 500 }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: { minHeight: "calc(100dvh - 56px)", padding: "16px", backgroundColor: "#f8fafc" },
          content: { borderLeft: "1px solid #e2e8f0" },
        }}
      >
        <div className="flex min-h-[calc(100dvh-96px)] flex-col gap-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-start gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Sparkles size={16} />
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-bold text-slate-800">Talk to this LLM Service</p>
                <p className="mt-1 text-xs font-semibold leading-5 text-slate-500">
                  Requests are sent through Octelium. Provider credentials remain on the cluster.
                </p>
              </div>
              <Badge variant="light" color={protocol === "anthropic" ? "orange" : "indigo"}>
                {protocol === "anthropic" ? "Anthropic" : "OpenAI-compatible"}
              </Badge>
            </div>
            <div className="mt-4 grid gap-3 sm:grid-cols-[1fr_auto]">
              <TextInput label="Service endpoint" value={endpoint} onChange={(e) => setEndpoint(e.currentTarget.value)} />
              {protocol === "openai" && (
                <SegmentedControl
                  className="self-end"
                  value={apiShape}
                  onChange={(value) => setApiShape(value as APIShape)}
                  data={[
                    { label: "Chat", value: "chat" },
                    { label: "Responses", value: "responses" },
                  ]}
                />
              )}
            </div>
          </div>

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
            <div className="flex items-center justify-between gap-2 border-b border-slate-100 px-4 py-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
                  Conversation
                </span>
                {messages.length > 0 && <Badge size="sm" variant="light">{messages.length}</Badge>}
              </div>
              <Group gap={4}>
                <Tooltip label={copied ? "Copied" : "Copy transcript"}>
                  <ActionIcon variant="subtle" color="gray" onClick={copyTranscript} disabled={!messages.length}>
                    {copied ? <Check size={15} /> : <Copy size={15} />}
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Clear conversation">
                  <ActionIcon variant="subtle" color="gray" onClick={clear} disabled={!messages.length || pending}>
                    <RotateCcw size={15} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            </div>
            <div ref={transcriptRef} className="min-h-[260px] flex-1 space-y-4 overflow-y-auto p-4">
              {messages.length === 0 ? (
                <div className="flex h-full min-h-[240px] flex-col items-center justify-center px-8 text-center">
                  <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-500">
                    <Bot size={23} />
                  </span>
                  <p className="mt-3 text-sm font-bold text-slate-700">Start a conversation</p>
                  <p className="mt-1 max-w-sm text-xs font-semibold leading-5 text-slate-500">
                    Ask a question or describe a task. Use advanced options to choose the model and sampling behavior.
                  </p>
                </div>
              ) : (
                messages.map((message) => (
                  <div key={message.id} className={`flex gap-3 ${message.role === "user" ? "justify-end" : "justify-start"}`}>
                    {message.role === "assistant" && (
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                        <Bot size={14} />
                      </span>
                    )}
                    <div className={`max-w-[88%] rounded-2xl px-3 py-2.5 text-sm leading-6 ${message.role === "user" ? "rounded-br-md bg-slate-900 text-white" : "rounded-bl-md border border-slate-200 bg-slate-50 text-slate-700"}`}>
                      {message.content || (pending && message.role === "assistant" ? <span className="inline-flex gap-1 text-slate-400"><i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" /><i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:150ms]" /><i className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:300ms]" /></span> : "No response content")}
                    </div>
                    {message.role === "user" && (
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
                        <User size={14} />
                      </span>
                    )}
                  </div>
                ))
              )}
            </div>
            {error && (
              <div className="mx-4 mb-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
                <CircleAlert size={15} className="mt-0.5 shrink-0" />
                <span className="min-w-0 break-words">{error}</span>
              </div>
            )}
            <div className="border-t border-slate-100 p-3">
              <Textarea
                aria-label="Message"
                placeholder="Ask the model something…"
                autosize
                minRows={2}
                maxRows={6}
                value={input}
                onChange={(event) => setInput(event.currentTarget.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
                    event.preventDefault();
                    void send();
                  }
                }}
                rightSection={pending ? <ActionIcon variant="subtle" color="red" onClick={stop}><Square size={15} /></ActionIcon> : undefined}
              />
              <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
                <Button type="button" variant="subtle" size="compact-xs" color="gray" leftSection={<Settings2 size={14} />} onClick={() => setAdvanced((value) => !value)}>
                  {advanced ? "Hide options" : "Request options"}
                </Button>
                <div className="flex items-center gap-2">
                  <Text size="xs" c="dimmed" fw={600}>Ctrl/Cmd + Enter</Text>
                  <Button type="button" size="sm" leftSection={pending ? <Square size={14} /> : <Send size={14} />} color={pending ? "red" : "dark"} onClick={pending ? stop : () => void send()}>
                    {pending ? "Stop" : "Send"}
                  </Button>
                </div>
              </div>
            </div>
          </div>

          {advanced && (
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2">
                <Settings2 size={15} className="text-slate-500" />
                <p className="text-xs font-bold uppercase tracking-[0.06em] text-slate-600">Request options</p>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="Model" required value={model} onChange={(event) => setModel(event.currentTarget.value)} placeholder={protocol === "anthropic" ? "claude-sonnet-4-5" : "gpt-4.1-mini"} />
                <TextInput label="System prompt" value={systemPrompt} onChange={(event) => setSystemPrompt(event.currentTarget.value)} placeholder="Optional instructions" />
                <TextInput label="Temperature" value={temperature} onChange={(event) => setTemperature(event.currentTarget.value)} placeholder="Provider default" inputMode="decimal" />
                <TextInput label="Top P" value={topP} onChange={(event) => setTopP(event.currentTarget.value)} placeholder="Provider default" inputMode="decimal" />
                <TextInput label={protocol === "anthropic" ? "Max output tokens" : "Max output tokens (optional)"} value={maxTokens} onChange={(event) => setMaxTokens(event.currentTarget.value)} placeholder={protocol === "anthropic" ? "1024" : "Provider default"} inputMode="numeric" />
                <TextInput label={protocol === "anthropic" ? "Thinking budget" : "Reasoning effort"} value={reasoning} onChange={(event) => setReasoning(event.currentTarget.value)} placeholder={protocol === "anthropic" ? "Optional token budget" : "low / medium / high"} />
              </div>
              <Text size="xs" c="dimmed" fw={600} mt="sm">
                Service limits and policy checks still apply. Do not paste secrets into prompts.
              </Text>
            </div>
          )}
        </div>
      </Drawer>
    </>
  );
};

export default LLMPlayground;
