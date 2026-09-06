import {
  Attachment,
  ChatMessage,
  Part,
  Protocol,
  RequestOptions,
  StreamEvent,
  Usage,
} from "./types";

const COOKIE_API_KEY = "octelium-cookie";

export const fetchThroughOctelium = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => {
  const headers = new Headers(init?.headers);
  headers.delete("authorization");
  headers.delete("x-api-key");
  headers.delete("api-key");
  return fetch(input, { ...init, credentials: "include", headers });
};

export const supportsStreaming = (protocol: Protocol) => protocol !== "bedrock";

export const messageText = (message: ChatMessage): string =>
  message.parts
    .filter((part): part is Extract<Part, { kind: "text" }> =>
      part.kind === "text",
    )
    .map((part) => part.text)
    .join("");

const numberOrUndefined = (value: string): number | undefined => {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
};

const base64Of = (attachment: Attachment): string =>
  attachment.dataURL.slice(attachment.dataURL.indexOf(",") + 1);

const bedrockFormat = (mime: string): string => {
  const subtype = mime.split("/")[1] ?? "png";
  return subtype === "jpg" ? "jpeg" : subtype;
};

const trimEnd = (value: string) => value.replace(/\/+$/, "");

export const defaultEndpoint = (origin: string, protocol: Protocol): string => {
  switch (protocol) {
    case "openai":
      return `${trimEnd(origin)}/v1`;
    case "anthropic":
      return `${trimEnd(origin)}/v1`;
    case "gemini":
      return `${trimEnd(origin)}/v1beta`;
    case "bedrock":
      return trimEnd(origin);
  }
};

export const requestPath = (
  protocol: Protocol,
  options: RequestOptions,
): string => {
  const model = encodeURIComponent(options.model.trim());
  switch (protocol) {
    case "openai":
      return options.apiShape === "responses" ? "/responses" : "/chat/completions";
    case "anthropic":
      return "/messages";
    case "gemini":
      return options.stream
        ? `/models/${model}:streamGenerateContent?alt=sse`
        : `/models/${model}:generateContent`;
    case "bedrock":
      return `/model/${model}/converse`;
  }
};

export const buildBody = (
  protocol: Protocol,
  messages: ChatMessage[],
  options: RequestOptions,
): Record<string, unknown> => {
  const temperature = numberOrUndefined(options.temperature);
  const topP = numberOrUndefined(options.topP);
  const maxTokens = numberOrUndefined(options.maxTokens);
  const system = options.systemPrompt.trim();
  const reasoning = options.reasoning.trim();

  switch (protocol) {
    case "openai": {
      if (options.apiShape === "responses") {
        return {
          model: options.model.trim(),
          input: [
            ...(system ? [{ role: "system", content: system }] : []),
            ...messages.map((message) => ({
              role: message.role,
              content: [
                { type: "input_text", text: messageText(message) },
                ...message.attachments.map((attachment) => ({
                  type: "input_image",
                  image_url: attachment.dataURL,
                })),
              ],
            })),
          ],
          ...(options.stream
            ? { stream: true }
            : {}),
          ...(temperature !== undefined ? { temperature } : {}),
          ...(topP !== undefined ? { top_p: topP } : {}),
          ...(maxTokens !== undefined ? { max_output_tokens: maxTokens } : {}),
          ...(reasoning ? { reasoning: { effort: reasoning } } : {}),
        };
      }
      return {
        model: options.model.trim(),
        messages: [
          ...(system ? [{ role: "system", content: system }] : []),
          ...messages.map((message) => ({
            role: message.role,
            content: message.attachments.length
              ? [
                  { type: "text", text: messageText(message) },
                  ...message.attachments.map((attachment) => ({
                    type: "image_url",
                    image_url: { url: attachment.dataURL },
                  })),
                ]
              : messageText(message),
          })),
        ],
        ...(options.stream
          ? { stream: true, stream_options: { include_usage: true } }
          : {}),
        ...(temperature !== undefined ? { temperature } : {}),
        ...(topP !== undefined ? { top_p: topP } : {}),
        ...(maxTokens !== undefined ? { max_completion_tokens: maxTokens } : {}),
        ...(reasoning ? { reasoning_effort: reasoning } : {}),
      };
    }

    case "anthropic": {
      const budget = numberOrUndefined(reasoning);
      return {
        model: options.model.trim(),
        max_tokens: maxTokens ?? 4096,
        ...(system ? { system } : {}),
        messages: messages.map((message) => ({
          role: message.role,
          content: [
            ...message.attachments.map((attachment) => ({
              type: "image",
              source: {
                type: "base64",
                media_type: attachment.mime,
                data: base64Of(attachment),
              },
            })),
            { type: "text", text: messageText(message) },
          ],
        })),
        ...(temperature !== undefined ? { temperature } : {}),
        ...(topP !== undefined ? { top_p: topP } : {}),
        ...(budget !== undefined
          ? { thinking: { type: "enabled", budget_tokens: budget } }
          : {}),
        ...(options.stream ? { stream: true } : {}),
      };
    }

    case "gemini": {
      const budget = numberOrUndefined(reasoning);
      return {
        contents: messages.map((message) => ({
          role: message.role === "assistant" ? "model" : "user",
          parts: [
            { text: messageText(message) },
            ...message.attachments.map((attachment) => ({
              inline_data: {
                mime_type: attachment.mime,
                data: base64Of(attachment),
              },
            })),
          ],
        })),
        ...(system
          ? { systemInstruction: { parts: [{ text: system }] } }
          : {}),
        generationConfig: {
          ...(temperature !== undefined ? { temperature } : {}),
          ...(topP !== undefined ? { topP } : {}),
          ...(maxTokens !== undefined ? { maxOutputTokens: maxTokens } : {}),
          ...(budget !== undefined
            ? { thinkingConfig: { thinkingBudget: budget, includeThoughts: true } }
            : {}),
        },
      };
    }

    case "bedrock": {
      return {
        messages: messages.map((message) => ({
          role: message.role,
          content: [
            { text: messageText(message) },
            ...message.attachments.map((attachment) => ({
              image: {
                format: bedrockFormat(attachment.mime),
                source: { bytes: base64Of(attachment) },
              },
            })),
          ],
        })),
        ...(system ? { system: [{ text: system }] } : {}),
        inferenceConfig: {
          ...(temperature !== undefined ? { temperature } : {}),
          ...(topP !== undefined ? { topP } : {}),
          ...(maxTokens !== undefined ? { maxTokens } : {}),
        },
      };
    }
  }
};

export const requestHeaders = (protocol: Protocol): Record<string, string> => {
  const base: Record<string, string> = { "content-type": "application/json" };
  switch (protocol) {
    case "openai":
      return { ...base, authorization: `Bearer ${COOKIE_API_KEY}` };
    case "anthropic":
      return {
        ...base,
        "x-api-key": COOKIE_API_KEY,
        "anthropic-version": "2023-06-01",
      };
    case "gemini":
      return { ...base, "x-goog-api-key": COOKIE_API_KEY };
    case "bedrock":
      return base;
  }
};

const openAIUsage = (usage: any): Usage => ({
  inputTokens: usage?.prompt_tokens ?? usage?.input_tokens,
  outputTokens: usage?.completion_tokens ?? usage?.output_tokens,
  totalTokens: usage?.total_tokens,
  reasoningTokens:
    usage?.completion_tokens_details?.reasoning_tokens ??
    usage?.output_tokens_details?.reasoning_tokens,
  cacheReadTokens:
    usage?.prompt_tokens_details?.cached_tokens ??
    usage?.input_tokens_details?.cached_tokens,
});

const anthropicUsage = (usage: any): Usage => ({
  inputTokens: usage?.input_tokens,
  outputTokens: usage?.output_tokens,
  totalTokens:
    usage?.input_tokens !== undefined && usage?.output_tokens !== undefined
      ? usage.input_tokens + usage.output_tokens
      : undefined,
  cacheReadTokens: usage?.cache_read_input_tokens,
  cacheWriteTokens: usage?.cache_creation_input_tokens,
});

const geminiUsage = (usage: any): Usage => ({
  inputTokens: usage?.promptTokenCount,
  outputTokens: usage?.candidatesTokenCount,
  totalTokens: usage?.totalTokenCount,
  reasoningTokens: usage?.thoughtsTokenCount,
  cacheReadTokens: usage?.cachedContentTokenCount,
});

const bedrockUsage = (usage: any): Usage => ({
  inputTokens: usage?.inputTokens,
  outputTokens: usage?.outputTokens,
  totalTokens: usage?.totalTokens,
  cacheReadTokens: usage?.cacheReadInputTokens,
  cacheWriteTokens: usage?.cacheWriteInputTokens,
});

export const parseSSEChunk = (
  protocol: Protocol,
  apiShape: RequestOptions["apiShape"],
  payload: string,
): StreamEvent[] => {
  if (payload === "[DONE]") return [];

  let event: any;
  try {
    event = JSON.parse(payload);
  } catch {
    return [];
  }

  const ret: StreamEvent[] = [];

  switch (protocol) {
    case "openai": {
      if (apiShape === "responses") {
        if (event.type === "response.output_text.delta" && event.delta) {
          ret.push({ type: "text", text: String(event.delta) });
        }
        if (
          (event.type === "response.reasoning_summary_text.delta" ||
            event.type === "response.reasoning_text.delta") &&
          event.delta
        ) {
          ret.push({ type: "thinking", text: String(event.delta) });
        }
        if (event.type === "response.completed" && event.response) {
          if (event.response.model) {
            ret.push({ type: "model", model: event.response.model });
          }
          if (event.response.usage) {
            ret.push({ type: "usage", usage: openAIUsage(event.response.usage) });
          }
          ret.push({ type: "finish", reason: event.response.status ?? "completed" });
        }
        break;
      }

      if (event.model) ret.push({ type: "model", model: event.model });
      const choice = event.choices?.[0];
      const delta = choice?.delta;
      if (typeof delta?.content === "string" && delta.content) {
        ret.push({ type: "text", text: delta.content });
      }
      const reasoning = delta?.reasoning_content ?? delta?.reasoning;
      if (typeof reasoning === "string" && reasoning) {
        ret.push({ type: "thinking", text: reasoning });
      }
      (delta?.tool_calls ?? []).forEach((call: any) => {
        if (call?.function?.name) {
          ret.push({
            type: "tool",
            name: call.function.name,
            input: call.function.arguments ?? "",
          });
        }
      });
      if (event.usage) ret.push({ type: "usage", usage: openAIUsage(event.usage) });
      if (choice?.finish_reason) {
        ret.push({ type: "finish", reason: choice.finish_reason });
      }
      break;
    }

    case "anthropic": {
      if (event.type === "message_start" && event.message) {
        if (event.message.model) {
          ret.push({ type: "model", model: event.message.model });
        }
        if (event.message.usage) {
          ret.push({ type: "usage", usage: anthropicUsage(event.message.usage) });
        }
      }
      if (event.type === "content_block_start" && event.content_block) {
        if (event.content_block.type === "tool_use") {
          ret.push({
            type: "tool",
            name: event.content_block.name ?? "tool",
            input: "",
          });
        }
      }
      if (event.type === "content_block_delta") {
        const delta = event.delta ?? {};
        if (delta.type === "text_delta" && delta.text) {
          ret.push({ type: "text", text: delta.text });
        }
        if (delta.type === "thinking_delta" && delta.thinking) {
          ret.push({ type: "thinking", text: delta.thinking });
        }
        if (delta.type === "input_json_delta" && delta.partial_json) {
          ret.push({ type: "tool", name: "", input: delta.partial_json });
        }
      }
      if (event.type === "message_delta") {
        if (event.usage) {
          ret.push({ type: "usage", usage: anthropicUsage(event.usage) });
        }
        if (event.delta?.stop_reason) {
          ret.push({ type: "finish", reason: event.delta.stop_reason });
        }
      }
      break;
    }

    case "gemini": {
      if (event.modelVersion) {
        ret.push({ type: "model", model: event.modelVersion });
      }
      const candidate = event.candidates?.[0];
      (candidate?.content?.parts ?? []).forEach((part: any) => {
        if (typeof part.text === "string" && part.text) {
          ret.push({
            type: part.thought ? "thinking" : "text",
            text: part.text,
          });
        }
        if (part.functionCall?.name) {
          ret.push({
            type: "tool",
            name: part.functionCall.name,
            input: JSON.stringify(part.functionCall.args ?? {}, null, 2),
          });
        }
      });
      if (event.usageMetadata) {
        ret.push({ type: "usage", usage: geminiUsage(event.usageMetadata) });
      }
      if (candidate?.finishReason) {
        ret.push({ type: "finish", reason: candidate.finishReason });
      }
      break;
    }

    case "bedrock":
      break;
  }

  return ret;
};

export const parseCompleteBody = (
  protocol: Protocol,
  apiShape: RequestOptions["apiShape"],
  body: any,
): StreamEvent[] => {
  const ret: StreamEvent[] = [];

  switch (protocol) {
    case "openai": {
      if (body.model) ret.push({ type: "model", model: body.model });
      if (apiShape === "responses") {
        (body.output ?? []).forEach((item: any) => {
          if (item.type === "reasoning") {
            (item.summary ?? []).forEach((entry: any) => {
              if (entry.text) ret.push({ type: "thinking", text: entry.text });
            });
          }
          (item.content ?? []).forEach((entry: any) => {
            if (entry.type === "output_text" && entry.text) {
              ret.push({ type: "text", text: entry.text });
            }
          });
          if (item.type === "function_call") {
            ret.push({
              type: "tool",
              name: item.name ?? "tool",
              input: item.arguments ?? "",
            });
          }
        });
        if (body.usage) ret.push({ type: "usage", usage: openAIUsage(body.usage) });
        if (body.status) ret.push({ type: "finish", reason: body.status });
        break;
      }
      const choice = body.choices?.[0];
      const reasoning = choice?.message?.reasoning_content;
      if (typeof reasoning === "string" && reasoning) {
        ret.push({ type: "thinking", text: reasoning });
      }
      if (typeof choice?.message?.content === "string") {
        ret.push({ type: "text", text: choice.message.content });
      }
      (choice?.message?.tool_calls ?? []).forEach((call: any) => {
        ret.push({
          type: "tool",
          name: call?.function?.name ?? "tool",
          input: call?.function?.arguments ?? "",
        });
      });
      if (body.usage) ret.push({ type: "usage", usage: openAIUsage(body.usage) });
      if (choice?.finish_reason) {
        ret.push({ type: "finish", reason: choice.finish_reason });
      }
      break;
    }

    case "anthropic": {
      if (body.model) ret.push({ type: "model", model: body.model });
      (body.content ?? []).forEach((block: any) => {
        if (block.type === "text" && block.text) {
          ret.push({ type: "text", text: block.text });
        }
        if (block.type === "thinking" && block.thinking) {
          ret.push({ type: "thinking", text: block.thinking });
        }
        if (block.type === "tool_use") {
          ret.push({
            type: "tool",
            name: block.name ?? "tool",
            input: JSON.stringify(block.input ?? {}, null, 2),
          });
        }
      });
      if (body.usage) ret.push({ type: "usage", usage: anthropicUsage(body.usage) });
      if (body.stop_reason) ret.push({ type: "finish", reason: body.stop_reason });
      break;
    }

    case "gemini": {
      if (body.modelVersion) ret.push({ type: "model", model: body.modelVersion });
      const candidate = body.candidates?.[0];
      (candidate?.content?.parts ?? []).forEach((part: any) => {
        if (typeof part.text === "string" && part.text) {
          ret.push({
            type: part.thought ? "thinking" : "text",
            text: part.text,
          });
        }
        if (part.functionCall?.name) {
          ret.push({
            type: "tool",
            name: part.functionCall.name,
            input: JSON.stringify(part.functionCall.args ?? {}, null, 2),
          });
        }
      });
      if (body.usageMetadata) {
        ret.push({ type: "usage", usage: geminiUsage(body.usageMetadata) });
      }
      if (candidate?.finishReason) {
        ret.push({ type: "finish", reason: candidate.finishReason });
      }
      break;
    }

    case "bedrock": {
      (body.output?.message?.content ?? []).forEach((block: any) => {
        if (typeof block.text === "string" && block.text) {
          ret.push({ type: "text", text: block.text });
        }
        if (block.reasoningContent?.reasoningText?.text) {
          ret.push({
            type: "thinking",
            text: block.reasoningContent.reasoningText.text,
          });
        }
        if (block.toolUse?.name) {
          ret.push({
            type: "tool",
            name: block.toolUse.name,
            input: JSON.stringify(block.toolUse.input ?? {}, null, 2),
          });
        }
      });
      if (body.usage) ret.push({ type: "usage", usage: bedrockUsage(body.usage) });
      if (body.stopReason) ret.push({ type: "finish", reason: body.stopReason });
      break;
    }
  }

  return ret;
};

export const errorMessage = (protocol: Protocol, status: number, body: string) => {
  try {
    const parsed = JSON.parse(body);
    const detail =
      parsed?.error?.message ??
      parsed?.error ??
      parsed?.message ??
      parsed?.Message ??
      parsed?.detail;
    if (typeof detail === "string" && detail) return `${status}: ${detail}`;
  } catch {
    if (body.trim()) return `${status}: ${body.slice(0, 400)}`;
  }
  return `The ${protocol} request failed with status ${status}.`;
};
