import * as path from "node:path";
import { defineTool } from "@earendil-works/pi-coding-agent";
import { Type } from "@earendil-works/pi-ai";
import type { JsonValue } from "@bufbuild/protobuf";
import type { Config } from "../../config.ts";
import type { APIRisk, OcteliumAPICallDetails } from "../../protocol/index.ts";
import type { APICatalog, APIMethod } from "../../octelium/catalog.ts";
import { firstSentences } from "../../octelium/catalog.ts";
import {
  InvalidRequestError,
  OcteliumAPIError,
  type OcteliumClient,
} from "../../octelium/client.ts";
import { splitCamelCase } from "../../octelium/search.ts";
import { extractResources } from "../resources.ts";
import {
  errorResult,
  requireRunContext,
  resolvePath,
  textResult,
  writeJSONFile,
  type ToolDeps,
} from "./util.ts";

export interface OcteliumToolDeps extends ToolDeps {
  catalog: APICatalog;
  client: OcteliumClient;
  approvalPolicy: Config["approvals"]["octeliumAPI"];
  maxInlineChars?: number;
}

export const needsApproval = (
  risk: APIRisk,
  policy: Config["approvals"]["octeliumAPI"],
): boolean => {
  switch (policy) {
    case "never":
      return false;
    case "destructive":
      return risk === "destructive";
    default:
      return risk !== "read";
  }
};

const maxDetailsChars = 64 * 1024;

const summarizeResponse = (response: JsonValue): string => {
  if (Array.isArray(response)) {
    return `The method streamed ${response.length} message(s).`;
  }
  if (response === null || typeof response !== "object") {
    return "";
  }

  const lines: string[] = [];
  const record = response as Record<string, JsonValue>;
  if (Array.isArray(record.items)) {
    lines.push(`items: ${record.items.length}`);
    const names = record.items
      .map((item) =>
        item && typeof item === "object" && !Array.isArray(item)
          ? (item as Record<string, JsonValue>).metadata
          : undefined,
      )
      .map((metadata) =>
        metadata && typeof metadata === "object" && !Array.isArray(metadata)
          ? (metadata as Record<string, JsonValue>).name
          : undefined,
      )
      .filter((name): name is string => typeof name === "string");
    if (names.length > 0) {
      lines.push(
        `names: ${names.slice(0, 200).join(", ")}${names.length > 200 ? ", …" : ""}`,
      );
    }
  }
  if (record.listResponseMeta) {
    lines.push(`listResponseMeta: ${JSON.stringify(record.listResponseMeta)}`);
  }
  lines.push(`top-level fields: ${Object.keys(record).join(", ")}`);
  return lines.join("\n");
};

const notFoundText = (catalog: APICatalog, input: string) => {
  const hits = catalog.search(input.replace(/[/.]/g, " "), 5);
  return `Unknown method "${input}". Use octelium_api_search to find methods.${
    hits.length > 0
      ? `\nClosest matches:\n${catalog.formatSearchResults(hits)}`
      : ""
  }`;
};

const approvalTitle = (method: APIMethod): string =>
  `${splitCamelCase(method.name)} (${method.packageName})`;

export const createOcteliumTools = (deps: OcteliumToolDeps) => {
  const { catalog, client } = deps;
  const maxInlineChars = deps.maxInlineChars ?? 24000;

  const search = defineTool({
    name: "octelium_api_search",
    label: "Search Octelium APIs",
    description:
      "Search the catalog of the Octelium Cluster's gRPC APIs (core, enterprise, access, Cordium, visibility/analytics, user) by keywords. Returns matching method IDs with their descriptions, request/response types and risk. Use it when you are not sure which method to call.",
    parameters: Type.Object({
      query: Type.String({
        description:
          'Keywords describing the operation, e.g. "list services", "workspace snapshot", "denied access logs"',
      }),
      limit: Type.Optional(
        Type.Integer({
          minimum: 1,
          maximum: 25,
          description: "Maximum number of results (default 8)",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const hits = catalog.search(params.query, params.limit ?? 8);
      return textResult(catalog.formatSearchResults(hits));
    },
  });

  const describe = defineTool({
    name: "octelium_api_describe",
    label: "Describe Octelium API",
    description:
      "Describe an Octelium API method (its documentation, risk, request JSON schema and response summary) or a message/enum type. Always describe a method before calling it for the first time in a conversation, unless you already know its exact request schema.",
    parameters: Type.Object({
      method: Type.Optional(
        Type.String({
          description:
            'Method ID as returned by octelium_api_search, e.g. "core.v1.MainService/ListService"',
        }),
      ),
      type: Type.Optional(
        Type.String({
          description:
            'Fully qualified message or enum type, e.g. "core.v1.Service.Spec.Config.HTTP"',
        }),
      ),
      verbose: Type.Optional(
        Type.Boolean({
          description: "Include longer field documentation (default false)",
        }),
      ),
      maxDepth: Type.Optional(
        Type.Integer({
          minimum: 0,
          maximum: 8,
          description: "Depth of nested types to expand (default 3)",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const opts = { verbose: params.verbose, maxDepth: params.maxDepth };
      if (params.method) {
        const method = catalog.resolveMethod(params.method);
        if (!method) {
          return errorResult(notFoundText(catalog, params.method));
        }
        return textResult(catalog.describeMethod(method, opts));
      }
      if (params.type) {
        const desc = catalog.resolveType(params.type);
        if (!desc) {
          return errorResult(`Unknown type "${params.type}"`);
        }
        return textResult(catalog.describeType(desc, opts));
      }
      return errorResult('Either "method" or "type" must be set');
    },
  });

  const call = defineTool({
    name: "octelium_api_call",
    label: "Call Octelium API",
    description:
      "Invoke an Octelium API method on behalf of the current User. The request is a JSON object using the proto3 JSON mapping of the method's request type (see octelium_api_describe). Large responses are saved to a JSON file whose path is returned so that you can process it with shell tools (e.g. jq, python). Methods that modify the Cluster may require the User's approval which is requested automatically.",
    parameters: Type.Object({
      method: Type.String({
        description:
          'Method ID, e.g. "core.v1.MainService/ListService" or "/octelium.api.main.core.v1.MainService/ListService"',
      }),
      request: Type.Optional(
        Type.Object(
          {},
          {
            additionalProperties: true,
            description:
              "The request message in proto3 JSON. Omit or use {} for an empty request.",
          },
        ),
      ),
      outputPath: Type.Optional(
        Type.String({
          description:
            "Optional file path where the full JSON response is also written",
        }),
      ),
      maxMessages: Type.Optional(
        Type.Integer({
          minimum: 1,
          maximum: 1000,
          description:
            "Server-streaming methods only: maximum number of messages to collect (default 50)",
        }),
      ),
      timeoutSeconds: Type.Optional(
        Type.Integer({
          minimum: 1,
          maximum: 300,
          description:
            "Server-streaming methods only: how long to collect messages (default 10)",
        }),
      ),
    }),
    async execute(toolCallId, params, signal) {
      const method = catalog.resolveMethod(params.method);
      if (!method) {
        return errorResult(notFoundText(catalog, params.method));
      }

      const details: OcteliumAPICallDetails = {
        kind: "octelium_api",
        method: method.id,
        grpcMethod: method.grpcMethod,
        risk: method.risk,
        request: params.request ?? {},
      };

      try {
        client.parseRequest(method, params.request);
      } catch (err) {
        details.error = {
          code: "invalid_argument",
          message: (err as Error).message,
        };
        return errorResult(
          `${(err as Error).message}\nUse octelium_api_describe with {"method": "${method.id}"} to see the request schema.`,
          details,
        );
      }

      const ctx = requireRunContext(deps);

      if (needsApproval(method.risk, deps.approvalPolicy)) {
        const decision = await ctx.requestApproval(
          {
            toolCallId,
            title: approvalTitle(method),
            description: method.description
              ? firstSentences(method.description, 300)
              : undefined,
            risk: method.risk,
            preview: { method: method.id, request: params.request ?? {} },
          },
          signal,
        );
        if (!decision.approved) {
          details.error = {
            code: "rejected",
            message: decision.reason ?? "Rejected by the User",
          };
          return errorResult(
            `The User did not approve calling ${method.id}${decision.reason ? `. Reason: ${decision.reason}` : ""}. Do not retry the same action unless the User asks you to.`,
            details,
          );
        }
      }

      let response: JsonValue;
      try {
        const res = await client.invoke(method, params.request, {
          signal,
          maxMessages: params.maxMessages,
          streamTimeoutMs: params.timeoutSeconds
            ? params.timeoutSeconds * 1000
            : undefined,
        });
        response = res.response;
      } catch (err) {
        if (
          err instanceof OcteliumAPIError ||
          err instanceof InvalidRequestError
        ) {
          const code =
            err instanceof OcteliumAPIError ? err.code : "invalid_argument";
          details.error = { code, message: err.message };
          return errorResult(
            `The Octelium API returned an error (${code}): ${err.message}`,
            details,
          );
        }
        throw err;
      }

      const { refs, objects } = extractResources(response);
      ctx.resources.remember(objects);
      if (refs.length > 0) {
        details.resources = refs.slice(0, 100);
      }

      const lines: string[] = [];
      if (params.outputPath) {
        const outputPath = resolvePath(deps.workDir, params.outputPath);
        writeJSONFile(outputPath, response);
        lines.push(`The full response was written to ${outputPath}`);
      }

      const serialized = JSON.stringify(response);
      if (serialized.length <= maxDetailsChars) {
        details.response = response;
      } else {
        details.responseTruncated = true;
      }

      if (serialized.length <= maxInlineChars) {
        lines.push(serialized);
      } else {
        const resultPath = path.join(
          ctx.resultsDir,
          `${new Date().toISOString().replace(/[:.]/g, "-")}-${method.name}.json`,
        );
        writeJSONFile(resultPath, response);
        details.resultPath = resultPath;
        lines.push(
          `The response is too large to be shown inline (${serialized.length} characters). It was saved to ${resultPath}. Process it with shell tools (e.g. jq, python) instead of reading it whole.`,
          summarizeResponse(response),
          `Preview: ${serialized.slice(0, 3000)}…`,
        );
      }

      return textResult(lines.join("\n"), details);
    },
  });

  return [search, describe, call];
};
