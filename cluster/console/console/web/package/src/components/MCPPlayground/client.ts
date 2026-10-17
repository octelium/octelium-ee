export type JSONObject = Record<string, unknown>;

export const DEFAULT_PROTOCOL_VERSION = "2025-03-26";
export const PROTOCOL_VERSIONS = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];
const SERVER_DISCOVER_VERSION = "2026-07-28";
export const DEFAULT_CLIENT_NAME = "Octelium Console";
export const DEFAULT_CLIENT_VERSION = "1.0.0";
const REQUEST_TIMEOUT_MS = 30_000;

export type Exchange = {
  id: number;
  method: string;
  target?: string;
  request: JSONObject;
  response?: unknown;
  httpStatus?: number;
  error?: string;
  startedAt: number;
  durationMs?: number;
};

export type ClientConfig = {
  url: string;
  protocolVersion: string;
  clientName: string;
  clientVersion: string;
  capabilities: string;
  sessionID?: string;
};

export type Reply = {
  exchange: Exchange;
  result?: unknown;
  error?: string;
  sessionID?: string;
  expired?: boolean;
};

let nextID = 1;

export const usesServerDiscover = (version: string) => {
  const date = Date.parse(`${version}T00:00:00Z`);
  return (
    Number.isFinite(date) &&
    date >= Date.parse(`${SERVER_DISCOVER_VERSION}T00:00:00Z`)
  );
};

export const isObject = (value: unknown): value is JSONObject =>
  !!value && typeof value === "object" && !Array.isArray(value);

export const toJSONText = (value: unknown) =>
  typeof value === "string" ? value : JSON.stringify(value, null, 2);

export const parseCapabilities = (value: string): JSONObject => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value.trim() || "{}");
  } catch {
    throw new Error("The client capabilities must be valid JSON.");
  }
  if (!isObject(parsed)) {
    throw new Error("The client capabilities must be a JSON object.");
  }
  return parsed;
};

const parseEventStream = (body: string): unknown => {
  const data = body
    .split(/\r?\n/)
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trim())
    .filter((line) => line && line !== "[DONE]");
  const last = data.at(-1);
  if (!last) return body;
  try {
    return JSON.parse(last);
  } catch {
    return data.join("\n");
  }
};

const parseBody = (body: string, contentType: string): unknown => {
  if (!body.trim()) return undefined;
  if (contentType.includes("text/event-stream") || body.includes("data:")) {
    return parseEventStream(body);
  }
  try {
    return JSON.parse(body);
  } catch {
    return body;
  }
};

const rpcError = (body: unknown): string | undefined => {
  if (!isObject(body) || !isObject(body.error)) return undefined;
  const { code, message } = body.error;
  const text = typeof message === "string" && message ? message : "Error";
  return typeof code === "number" ? `${code}: ${text}` : text;
};

const encodeHeaderValue = (value: string) => {
  if (
    /^[\x20-\x7e]*$/.test(value) &&
    value === value.trim() &&
    !value.startsWith("=?base64?")
  ) {
    return value;
  }

  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return `=?base64?${btoa(binary)}?=`;
};

export const sendMCP = async (arg: {
  config: ClientConfig;
  method: string;
  params?: JSONObject;
  target?: string;
  notification?: boolean;
  signal?: AbortSignal;
}): Promise<Reply> => {
  const { config } = arg;
  const version = config.protocolVersion || DEFAULT_PROTOCOL_VERSION;
  const params: JSONObject = { ...arg.params };

  if (usesServerDiscover(version) && !arg.notification) {
    let capabilities: JSONObject;
    try {
      capabilities = parseCapabilities(config.capabilities);
    } catch (err) {
      const error = (err as Error).message;
      return {
        exchange: {
          id: nextID++,
          method: arg.method,
          target: arg.target,
          request: { jsonrpc: "2.0", method: arg.method, params },
          error,
          startedAt: Date.now(),
        },
        error,
      };
    }
    params._meta = {
      "io.modelcontextprotocol/protocolVersion": version,
      "io.modelcontextprotocol/clientInfo": {
        name: config.clientName.trim() || DEFAULT_CLIENT_NAME,
        version: config.clientVersion.trim() || DEFAULT_CLIENT_VERSION,
      },
      "io.modelcontextprotocol/clientCapabilities": capabilities,
    };
  }

  const request: JSONObject = {
    jsonrpc: "2.0",
    ...(arg.notification ? {} : { id: nextID }),
    method: arg.method,
    ...(Object.keys(params).length > 0 || !arg.notification ? { params } : {}),
  };
  const exchange: Exchange = {
    id: nextID++,
    method: arg.method,
    target: arg.target,
    request,
    startedAt: Date.now(),
  };

  const controller = new AbortController();
  const timeout = window.setTimeout(
    () => controller.abort("timeout"),
    REQUEST_TIMEOUT_MS,
  );
  const onAbort = () => controller.abort("cancelled");
  arg.signal?.addEventListener("abort", onAbort);
  const startedAt = performance.now();

  try {
    const response = await fetch(config.url, {
      method: "POST",
      credentials: "include",
      headers: {
        Accept: "application/json, text/event-stream",
        "Content-Type": "application/json",
        "MCP-Protocol-Version": version,
        "Mcp-Method": arg.method,
        ...(arg.target ? { "Mcp-Name": encodeHeaderValue(arg.target) } : {}),
        ...(config.sessionID ? { "Mcp-Session-Id": config.sessionID } : {}),
      },
      body: JSON.stringify(request),
      signal: controller.signal,
    });
    const text = await response.text();
    const body = parseBody(text, response.headers.get("content-type") ?? "");
    exchange.httpStatus = response.status;
    exchange.response = body;
    exchange.durationMs = performance.now() - startedAt;
    const sessionID = response.headers.get("Mcp-Session-Id") ?? undefined;

    if (!response.ok) {
      const detail =
        rpcError(body) ??
        (typeof body === "string" ? body.slice(0, 400) : undefined);
      exchange.error = `${response.status}${response.statusText ? ` ${response.statusText}` : ""}${detail ? `: ${detail}` : ""}`;
      return {
        exchange,
        error: exchange.error,
        sessionID,
        expired: response.status === 404 && !!config.sessionID,
      };
    }

    const error = rpcError(body);
    if (error) {
      exchange.error = error;
      return { exchange, error, sessionID };
    }

    return {
      exchange,
      result: isObject(body) ? body.result : body,
      sessionID,
    };
  } catch (err) {
    exchange.durationMs = performance.now() - startedAt;
    exchange.error =
      controller.signal.aborted && controller.signal.reason === "timeout"
        ? "The request timed out after 30 seconds."
        : controller.signal.aborted
          ? "The request was cancelled."
          : err instanceof Error
            ? err.message
            : "The MCP request failed.";
    return { exchange, error: exchange.error };
  } finally {
    window.clearTimeout(timeout);
    arg.signal?.removeEventListener("abort", onAbort);
  }
};
