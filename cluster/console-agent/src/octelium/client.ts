import * as fs from "node:fs";
import * as net from "node:net";
import {
  fromJson,
  toJson,
  type DescMethodStreaming,
  type DescMethodUnary,
  type JsonValue,
} from "@bufbuild/protobuf";
import { Code, ConnectError, type Transport } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import type { Config } from "../config.ts";
import type { APICatalog, APIMethod } from "./catalog.ts";

export type OcteliumMode = "proxy" | "direct" | "disabled";

export class OcteliumAPIError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

export class InvalidRequestError extends Error {}

export interface InvokeOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
  maxMessages?: number;
  streamTimeoutMs?: number;
}

export interface InvokeResult {
  response: JsonValue;
  messageCount?: number;
}

export interface OcteliumClientOptions {
  mode: OcteliumMode;
  domain?: string;
  authProxySocket?: string;
  accessToken?: string;
  insecureTLS?: boolean;
  timeoutSeconds?: number;
  userAgent?: string;
}

export const resolveOcteliumMode = (
  cfg: Config["octelium"],
): { mode: OcteliumMode; reason?: string } => {
  const proxyAvailable =
    !!cfg.authProxySocket && fs.existsSync(cfg.authProxySocket);

  switch (cfg.mode) {
    case "disabled":
      return {
        mode: "disabled",
        reason: "The Octelium API access is disabled",
      };
    case "proxy":
      return proxyAvailable
        ? { mode: "proxy" }
        : {
            mode: "disabled",
            reason: `The Octelium auth proxy socket is not available: ${cfg.authProxySocket ?? "(unset)"}`,
          };
    case "direct":
      return cfg.domain && cfg.accessToken
        ? { mode: "direct" }
        : {
            mode: "disabled",
            reason:
              "The direct mode requires both the Octelium domain and an access token",
          };
    default:
      if (proxyAvailable) {
        return { mode: "proxy" };
      }
      if (cfg.domain && cfg.accessToken) {
        return { mode: "direct" };
      }
      return {
        mode: "disabled",
        reason:
          "Could not find neither OCTELIUM_AUTH_PROXY_SOCKET nor OCTELIUM_DOMAIN and OCTELIUM_ACCESS_TOKEN",
      };
  }
};

export const errorCodeName = (code: Code): string =>
  (Code[code] ?? "unknown").replace(/([a-z])([A-Z])/g, "$1_$2").toLowerCase();

export const toOcteliumAPIError = (err: unknown): OcteliumAPIError => {
  if (err instanceof OcteliumAPIError) {
    return err;
  }
  const connectErr = ConnectError.from(err);
  return new OcteliumAPIError(
    errorCodeName(connectErr.code),
    connectErr.rawMessage || connectErr.message,
  );
};

const isRecord = (arg: unknown): arg is Record<string, unknown> =>
  typeof arg === "object" && arg !== null && !Array.isArray(arg);

export class OcteliumClient {
  readonly mode: OcteliumMode;
  readonly domain?: string;
  readonly catalog: APICatalog;
  private transport?: Transport;
  private timeoutMs: number;

  constructor(catalog: APICatalog, opts: OcteliumClientOptions) {
    this.catalog = catalog;
    this.mode = opts.mode;
    this.domain = opts.domain;
    this.timeoutMs = (opts.timeoutSeconds ?? 30) * 1000;
    this.transport = this.createTransport(opts);
  }

  static fromConfig(
    catalog: APICatalog,
    cfg: Config["octelium"],
    userAgent?: string,
  ): { client: OcteliumClient; reason?: string } {
    const { mode, reason } = resolveOcteliumMode(cfg);
    return {
      client: new OcteliumClient(catalog, {
        mode,
        domain: cfg.domain,
        authProxySocket: cfg.authProxySocket,
        accessToken: cfg.accessToken,
        insecureTLS: cfg.insecureTLS,
        timeoutSeconds: cfg.timeoutSeconds,
        userAgent,
      }),
      reason,
    };
  }

  private createTransport(opts: OcteliumClientOptions): Transport | undefined {
    const userAgent = opts.userAgent ?? "octelium-console-agent";

    switch (opts.mode) {
      case "proxy": {
        const socketPath = opts.authProxySocket!;
        return createGrpcTransport({
          baseUrl: "http://localhost",
          nodeOptions: {
            createConnection: () => net.connect(socketPath),
          },
          interceptors: [
            (next) => async (req) => {
              req.header.set("user-agent", userAgent);
              return next(req);
            },
          ],
        });
      }
      case "direct": {
        const accessToken = opts.accessToken!;
        return createGrpcTransport({
          baseUrl: `https://octelium-api.${opts.domain}`,
          nodeOptions: {
            rejectUnauthorized: !opts.insecureTLS,
          },
          interceptors: [
            (next) => async (req) => {
              req.header.set("user-agent", userAgent);
              req.header.set("x-octelium-auth", accessToken);
              return next(req);
            },
          ],
        });
      }
      default:
        return undefined;
    }
  }

  get enabled(): boolean {
    return this.transport !== undefined;
  }

  parseRequest(method: APIMethod, request: unknown) {
    const input = request === undefined || request === null ? {} : request;
    if (!isRecord(input)) {
      throw new InvalidRequestError(
        `The request of ${method.id} must be a JSON object of type ${method.requestType}`,
      );
    }
    try {
      return fromJson(method.desc.input, input as JsonValue, {
        registry: this.catalog.registry,
      });
    } catch (err) {
      throw new InvalidRequestError(
        `Invalid request for ${method.id} (type ${method.requestType}): ${(err as Error).message}`,
      );
    }
  }

  async invoke(
    method: APIMethod,
    request: unknown,
    opts: InvokeOptions = {},
  ): Promise<InvokeResult> {
    if (!this.transport) {
      throw new OcteliumAPIError(
        "unavailable",
        "The Octelium API is not available to this agent",
      );
    }
    if (!method.invocable) {
      throw new InvalidRequestError(
        `${method.id} is a ${method.kind} method which cannot be invoked by this agent`,
      );
    }

    const input = this.parseRequest(method, request);
    const registry = this.catalog.registry;
    const toJsonOpts = { registry };

    try {
      if (method.kind === "unary") {
        const res = await this.transport.unary(
          method.desc as DescMethodUnary,
          opts.signal,
          opts.timeoutMs ?? this.timeoutMs,
          undefined,
          input,
        );
        return {
          response: toJson(method.desc.output, res.message, toJsonOpts),
        };
      }

      const maxMessages = opts.maxMessages ?? 50;
      const controller = new AbortController();
      const onAbort = () => controller.abort();
      opts.signal?.addEventListener("abort", onAbort, { once: true });
      const timer = setTimeout(
        () => controller.abort(),
        opts.streamTimeoutMs ?? 10000,
      );

      const messages: JsonValue[] = [];
      try {
        const res = await this.transport.stream(
          method.desc as DescMethodStreaming,
          controller.signal,
          undefined,
          undefined,
          (async function* () {
            yield input;
          })(),
        );
        for await (const message of res.message) {
          messages.push(toJson(method.desc.output, message, toJsonOpts));
          if (messages.length >= maxMessages) {
            controller.abort();
            break;
          }
        }
      } catch (err) {
        if (!controller.signal.aborted || opts.signal?.aborted) {
          throw err;
        }
      } finally {
        clearTimeout(timer);
        opts.signal?.removeEventListener("abort", onAbort);
      }

      return { response: messages, messageCount: messages.length };
    } catch (err) {
      throw toOcteliumAPIError(err);
    }
  }

  async call<T = Record<string, unknown>>(
    methodId: string,
    request: unknown = {},
    opts: InvokeOptions = {},
  ): Promise<T> {
    const method = this.catalog.resolveMethod(methodId);
    if (!method) {
      throw new InvalidRequestError(`Unknown method: ${methodId}`);
    }
    const res = await this.invoke(method, request, opts);
    return res.response as T;
  }
}
