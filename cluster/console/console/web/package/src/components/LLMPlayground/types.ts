import { Service_Spec_Config_LLM_Protocol } from "@/apis/corev1/corev1";

export type Protocol = "openai" | "anthropic" | "gemini" | "bedrock";

export type Role = "user" | "assistant";

export type Attachment = {
  id: string;
  name: string;
  mime: string;
  dataURL: string;
};

export type Part =
  | { kind: "text"; text: string }
  | { kind: "thinking"; text: string }
  | { kind: "tool"; name: string; input: string };

export type Usage = {
  inputTokens?: number;
  outputTokens?: number;
  totalTokens?: number;
  reasoningTokens?: number;
  cacheReadTokens?: number;
  cacheWriteTokens?: number;
};

export type ChatMessage = {
  id: string;
  role: Role;
  parts: Part[];
  attachments: Attachment[];
  usage?: Usage;
  model?: string;
  finishReason?: string;
  latencyMs?: number;
  ttftMs?: number;
  error?: string;
  cached?: boolean;
};

export type RequestOptions = {
  model: string;
  systemPrompt: string;
  temperature: string;
  topP: string;
  maxTokens: string;
  reasoning: string;
  stream: boolean;
  apiShape: "chat" | "responses";
};

export type StreamEvent =
  | { type: "text"; text: string }
  | { type: "thinking"; text: string }
  | { type: "tool"; name: string; input: string }
  | { type: "usage"; usage: Usage }
  | { type: "model"; model: string }
  | { type: "finish"; reason: string };

export const PROTOCOL_LABELS: Record<Protocol, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  gemini: "Gemini",
  bedrock: "Bedrock",
};

export const PROTOCOL_MODEL_PLACEHOLDER: Record<Protocol, string> = {
  openai: "gpt-5-mini",
  anthropic: "claude-sonnet-5",
  gemini: "gemini-3-pro",
  bedrock: "anthropic.claude-sonnet-4-20250514-v1:0",
};

export const protocolFromEnum = (
  value?: Service_Spec_Config_LLM_Protocol,
): Protocol => {
  switch (value) {
    case Service_Spec_Config_LLM_Protocol.ANTHROPIC:
      return "anthropic";
    case Service_Spec_Config_LLM_Protocol.GEMINI:
      return "gemini";
    case Service_Spec_Config_LLM_Protocol.BEDROCK:
      return "bedrock";
    default:
      return "openai";
  }
};
