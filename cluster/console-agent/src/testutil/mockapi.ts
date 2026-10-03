import * as fs from "node:fs";
import * as http2 from "node:http2";
import * as os from "node:os";
import * as path from "node:path";
import {
  fromJson,
  toJson,
  type DescMessage,
  type JsonValue,
} from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import type { APICatalog } from "../octelium/catalog.ts";

export type MockHandler = (
  request: JsonValue,
  headers: Headers,
) => JsonValue | JsonValue[] | Promise<JsonValue | JsonValue[]>;

export interface MockCall {
  method: string;
  request: JsonValue;
  headers: Headers;
}

export interface MockAPI {
  socketPath: string;
  calls: MockCall[];
  close(): Promise<void>;
}

export const startMockAPI = async (
  catalog: APICatalog,
  handlers: Record<string, MockHandler>,
): Promise<MockAPI> => {
  const calls: MockCall[] = [];
  const registry = catalog.registry;
  const byService = new Map<string, Record<string, MockHandler>>();

  for (const [id, handler] of Object.entries(handlers)) {
    const method = catalog.resolveMethod(id);
    if (!method) {
      throw new Error(`Unknown method: ${id}`);
    }
    const impls = byService.get(method.serviceName) ?? {};
    impls[method.name] = handler;
    byService.set(method.serviceName, impls);
  }

  const toOutput = (desc: DescMessage, json: JsonValue) =>
    fromJson(desc, json, { registry });

  const handler = connectNodeAdapter({
    routes: (router) => {
      for (const [serviceName, impls] of byService) {
        const service = registry.getService(serviceName)!;
        const impl: Record<string, unknown> = {};
        for (const method of service.methods) {
          const fn = impls[method.name];
          if (!fn) {
            continue;
          }
          const id = `${serviceName}/${method.name}`;
          if (method.methodKind === "unary") {
            impl[method.localName] = async (
              req: never,
              ctx: { requestHeader: Headers },
            ) => {
              const request = toJson(method.input, req, { registry });
              calls.push({ method: id, request, headers: ctx.requestHeader });
              const res = await fn(request, ctx.requestHeader);
              return toOutput(method.output, res as JsonValue);
            };
          } else if (method.methodKind === "server_streaming") {
            impl[method.localName] = async function* (
              req: never,
              ctx: { requestHeader: Headers },
            ) {
              const request = toJson(method.input, req, { registry });
              calls.push({ method: id, request, headers: ctx.requestHeader });
              const res = await fn(request, ctx.requestHeader);
              for (const item of res as JsonValue[]) {
                yield toOutput(method.output, item);
              }
            };
          }
        }
        router.service(service, impl as ServiceImpl<never>);
      }
    },
  });

  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "octelium-mockapi-"));
  const socketPath = path.join(dir, "api.sock");
  const server = http2.createServer(handler);
  const sessions = new Set<http2.ServerHttp2Session>();
  server.on("session", (session) => {
    sessions.add(session);
    session.on("close", () => sessions.delete(session));
  });
  await new Promise<void>((resolve) => server.listen(socketPath, resolve));

  return {
    socketPath,
    calls,
    close: async () => {
      for (const session of sessions) {
        session.destroy();
      }
      await new Promise<void>((resolve) => server.close(() => resolve()));
      fs.rmSync(dir, { recursive: true, force: true });
    },
  };
};
