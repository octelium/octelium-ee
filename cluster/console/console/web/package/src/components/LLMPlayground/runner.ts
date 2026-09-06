import {
  buildBody,
  errorMessage,
  fetchThroughOctelium,
  parseCompleteBody,
  parseSSEChunk,
  requestHeaders,
  requestPath,
  supportsStreaming,
} from "./protocols";
import { ChatMessage, Protocol, RequestOptions, StreamEvent } from "./types";

const readSSE = async (
  body: ReadableStream<Uint8Array>,
  onPayload: (payload: string) => void,
) => {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let index = buffer.indexOf("\n\n");
    while (index !== -1) {
      const block = buffer.slice(0, index);
      buffer = buffer.slice(index + 2);
      const payload = block
        .split("\n")
        .filter((line) => line.startsWith("data:"))
        .map((line) => line.slice(5).trimStart())
        .join("\n");
      if (payload) onPayload(payload);
      index = buffer.indexOf("\n\n");
    }
  }

  if (buffer.trim()) {
    const payload = buffer
      .split("\n")
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice(5).trimStart())
      .join("\n");
    if (payload) onPayload(payload);
  }
};

export const runCompletion = async (arg: {
  endpoint: string;
  protocol: Protocol;
  options: RequestOptions;
  messages: ChatMessage[];
  signal: AbortSignal;
  onEvent: (event: StreamEvent) => void;
}) => {
  const stream = arg.options.stream && supportsStreaming(arg.protocol);
  const options = { ...arg.options, stream };
  const url = `${arg.endpoint.replace(/\/+$/, "")}${requestPath(arg.protocol, options)}`;

  const response = await fetchThroughOctelium(url, {
    method: "POST",
    signal: arg.signal,
    headers: requestHeaders(arg.protocol),
    body: JSON.stringify(buildBody(arg.protocol, arg.messages, options)),
  });

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(errorMessage(arg.protocol, response.status, text));
  }

  const contentType = response.headers.get("content-type") ?? "";
  const isEventStream = contentType.includes("text/event-stream");

  if (stream && isEventStream && response.body) {
    await readSSE(response.body, (payload) => {
      parseSSEChunk(arg.protocol, options.apiShape, payload).forEach(
        arg.onEvent,
      );
    });
    return;
  }

  const body = await response.json().catch(() => ({}));
  parseCompleteBody(arg.protocol, options.apiShape, body).forEach(arg.onEvent);
};
