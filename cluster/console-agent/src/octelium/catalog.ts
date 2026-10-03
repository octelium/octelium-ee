import * as fs from "node:fs";
import {
  createFileRegistry,
  fromBinary,
  ScalarType,
  type DescEnum,
  type DescEnumValue,
  type DescField,
  type DescFile,
  type DescMessage,
  type DescMethod,
  type DescOneof,
  type DescService,
  type FileRegistry,
} from "@bufbuild/protobuf";
import { FileDescriptorSetSchema } from "@bufbuild/protobuf/wkt";
import type { APIRisk } from "../protocol/index.ts";
import { SearchIndex, splitCamelCase, type SearchHit } from "./search.ts";

export const TYPE_PREFIX = "octelium.api.main.";

export const DEFAULT_DESCRIPTOR_PATH = new URL(
  "../../apis/octelium.binpb",
  import.meta.url,
);

const packageDescriptions: Record<string, string> = {
  "core.v1":
    "Core Cluster management: Services, Namespaces, Users, Groups, Policies, Sessions, Devices, Credentials, Secrets, IdentityProviders, Gateways, Regions and the ClusterConfig",
  "user.v1":
    "The current User's own view: their status and the Services and Namespaces they can access",
  "enterprise.v1":
    "Enterprise resources (e.g. CollectorExporters, Certificates, DNSProviders, DirectoryProviders, SecretStores, DeviceManagers), Policy evaluation and the Cluster lifecycle (version, upgrades, license)",
  "access.v1":
    "Just-in-time access: Catalogs, access Policies, access Requests and their Reviews",
  "cordium.v1":
    "Cordium sandboxes: Spaces, Templates, Workspaces, WorkspaceSnapshots, Volumes, Memberships, Secrets, UserSecrets and GitProviders",
  "visibility.v1":
    "Visibility and analytics: access, authentication, audit and component logs, top-N analytics, Cluster overview and health, metrics",
  "visibility.core.v1":
    "Summaries and searchable/filterable listings of the core resources",
  "visibility.enterprise.v1":
    "Summaries and searchable/filterable listings of the enterprise resources",
  "visibility.access.v1":
    "Summaries and searchable/filterable listings of the access resources",
  "visibility.cordium.v1":
    "Summaries and searchable/filterable listings of the Cordium resources",
  "visibility.metrics.v1": "Metrics time series and data points",
  "visibility.llm.v1":
    "LLM gateway analytics (requests, tokens, models, costs, latencies)",
};

const packagePriors: Record<string, number> = {
  "core.v1": 1,
  "cordium.v1": 1,
  "enterprise.v1": 0.98,
  "visibility.v1": 0.98,
  "access.v1": 0.96,
  "user.v1": 0.94,
};

const excludedServices = new Set([
  "octelium.api.main.cordium.v1.WorkspaceService",
]);

const excludedMethods = new Set([
  "octelium.api.main.user.v1.MainService/Connect",
  "octelium.api.main.user.v1.MainService/Disconnect",
  "octelium.api.main.user.v1.MainService/SetServiceConfigs",
]);

const readVerbs = [
  "Get",
  "List",
  "Watch",
  "Listen",
  "Search",
  "Query",
  "Describe",
  "Count",
  "Check",
  "Validate",
  "Evaluate",
  "Is",
  "Has",
];

const destructiveVerbs = [
  "Delete",
  "Remove",
  "Leave",
  "Revoke",
  "Purge",
  "Reset",
  "Terminate",
  "Kill",
];

const knownVerbs = [
  ...readVerbs,
  ...destructiveVerbs,
  "Create",
  "Update",
  "Set",
  "Start",
  "Stop",
  "Restart",
  "Build",
  "Cancel",
  "Share",
  "Unshare",
  "Generate",
  "Upgrade",
  "Join",
  "Approve",
  "Reject",
  "Review",
];

export type MethodKind = DescMethod["methodKind"];

export interface APIMethod {
  id: string;
  grpcMethod: string;
  packageName: string;
  serviceName: string;
  name: string;
  verb?: string;
  noun?: string;
  kind: MethodKind;
  risk: APIRisk;
  invocable: boolean;
  description: string;
  requestType: string;
  responseType: string;
  desc: DescMethod;
}

export const shortTypeName = (typeName: string): string =>
  typeName.startsWith(TYPE_PREFIX)
    ? typeName.slice(TYPE_PREFIX.length)
    : typeName;

const getVerb = (name: string): { verb?: string; noun?: string } => {
  const verb = knownVerbs
    .filter(
      (v) =>
        name.startsWith(v) &&
        (name.length === v.length || /[A-Z]/.test(name[v.length])),
    )
    .sort((a, b) => b.length - a.length)[0];
  if (!verb) {
    return {};
  }
  return { verb, noun: name.slice(verb.length) || undefined };
};

export const classifyRisk = (
  serviceTypeName: string,
  methodName: string,
): APIRisk => {
  if (
    serviceTypeName.startsWith(`${TYPE_PREFIX}visibility.`) ||
    serviceTypeName.endsWith(".PolicyPortalService")
  ) {
    return "read";
  }

  if (/Token|Password|Credential.*Secret/.test(methodName)) {
    return "sensitive";
  }

  const { verb } = getVerb(methodName);
  if (verb && readVerbs.includes(verb)) {
    return "read";
  }
  if (verb && destructiveVerbs.includes(verb)) {
    return "destructive";
  }
  return "write";
};

type AnyDesc =
  | DescMessage
  | DescEnum
  | DescEnumValue
  | DescField
  | DescOneof
  | DescService
  | DescMethod;

const getSourcePath = (desc: AnyDesc): number[] => {
  switch (desc.kind) {
    case "message":
      return desc.parent
        ? [
            ...getSourcePath(desc.parent),
            3,
            desc.parent.proto.nestedType.indexOf(desc.proto),
          ]
        : [4, desc.file.proto.messageType.indexOf(desc.proto)];
    case "enum":
      return desc.parent
        ? [
            ...getSourcePath(desc.parent),
            4,
            desc.parent.proto.enumType.indexOf(desc.proto),
          ]
        : [5, desc.file.proto.enumType.indexOf(desc.proto)];
    case "enum_value":
      return [
        ...getSourcePath(desc.parent),
        2,
        desc.parent.proto.value.indexOf(desc.proto),
      ];
    case "field":
      return [
        ...getSourcePath(desc.parent),
        2,
        desc.parent.proto.field.indexOf(desc.proto),
      ];
    case "oneof":
      return [
        ...getSourcePath(desc.parent),
        8,
        desc.parent.proto.oneofDecl.indexOf(desc.proto),
      ];
    case "service":
      return [6, desc.file.proto.service.indexOf(desc.proto)];
    case "rpc":
      return [
        ...getSourcePath(desc.parent),
        2,
        desc.parent.proto.method.indexOf(desc.proto),
      ];
  }
};

const getDescFile = (desc: AnyDesc): DescFile => {
  switch (desc.kind) {
    case "message":
    case "enum":
    case "service":
      return desc.file;
    default:
      return getDescFile(desc.parent);
  }
};

export const normalizeComment = (comment: string): string =>
  comment
    .split("\n")
    .map((line) => line.trim())
    .join("\n")
    .replace(/\n{2,}/g, "\n\n")
    .trim();

export const firstSentences = (text: string, max = 200): string => {
  const flat = text.replace(/\s+/g, " ").trim();
  if (flat.length <= max) {
    return flat;
  }
  const cut = flat.slice(0, max);
  let end = -1;
  for (const match of cut.matchAll(/\. /g)) {
    const before = cut.slice(0, match.index);
    if (!/(\b(i\.e|e\.g|etc|vs)|\.\.)$/.test(before)) {
      end = match.index;
    }
  }
  if (end > max / 3) {
    return cut.slice(0, end + 1);
  }
  return `${cut.trimEnd()}…`;
};

export interface DescribeOptions {
  maxChars?: number;
  maxDepth?: number;
  verbose?: boolean;
}

export class APICatalog {
  readonly registry: FileRegistry;
  private methodsById = new Map<string, APIMethod>();
  private methodsByLowerId = new Map<string, APIMethod>();
  private comments = new Map<string, Map<string, string>>();
  private index = new SearchIndex<APIMethod>();
  private services: DescService[] = [];

  constructor(registry: FileRegistry) {
    this.registry = registry;
    this.build();
  }

  static fromBinary(data: Uint8Array): APICatalog {
    return new APICatalog(
      createFileRegistry(fromBinary(FileDescriptorSetSchema, data)),
    );
  }

  static load(filePath: string | URL = DEFAULT_DESCRIPTOR_PATH): APICatalog {
    return APICatalog.fromBinary(fs.readFileSync(filePath));
  }

  getComment(desc: AnyDesc): string {
    const file = getDescFile(desc);
    let fileComments = this.comments.get(file.name);
    if (!fileComments) {
      fileComments = new Map();
      for (const location of file.proto.sourceCodeInfo?.location ?? []) {
        const text = [location.leadingComments, location.trailingComments]
          .filter((c): c is string => !!c && c.trim() !== "")
          .map(normalizeComment)
          .join("\n");
        if (text) {
          fileComments.set(location.path.join(","), text);
        }
      }
      this.comments.set(file.name, fileComments);
    }
    return fileComments.get(getSourcePath(desc).join(",")) ?? "";
  }

  private build() {
    for (const type of this.registry) {
      if (type.kind !== "service") {
        continue;
      }
      if (
        !type.typeName.startsWith(TYPE_PREFIX) ||
        excludedServices.has(type.typeName)
      ) {
        continue;
      }
      this.services.push(type);

      const packageName = shortTypeName(type.file.proto.package);
      const serviceComment = this.getComment(type);

      for (const method of type.methods) {
        const grpcMethod = `/${type.typeName}/${method.name}`;
        if (excludedMethods.has(`${type.typeName}/${method.name}`)) {
          continue;
        }

        const { verb, noun } = getVerb(method.name);
        const entry: APIMethod = {
          id: `${shortTypeName(type.typeName)}/${method.name}`,
          grpcMethod,
          packageName,
          serviceName: type.typeName,
          name: method.name,
          verb,
          noun,
          kind: method.methodKind,
          risk: classifyRisk(type.typeName, method.name),
          invocable:
            method.methodKind === "unary" ||
            method.methodKind === "server_streaming",
          description: this.getComment(method),
          requestType: shortTypeName(method.input.typeName),
          responseType: shortTypeName(method.output.typeName),
          desc: method,
        };

        this.methodsById.set(entry.id, entry);
        this.methodsByLowerId.set(entry.id.toLowerCase(), entry);

        this.index.add(
          entry,
          [
            { text: method.name, weight: 3, primary: true },
            { text: noun ?? "", weight: 2 },
            { text: `${packageName} ${type.name}`, weight: 1.5 },
            { text: entry.description, weight: 1 },
            {
              text: `${method.input.name} ${method.output.name}`,
              weight: 1,
            },
            {
              text: firstSentences(this.getComment(method.output), 160),
              weight: 0.5,
            },
            {
              text: packageDescriptions[packageName] ?? "",
              weight: 0.3,
            },
            { text: firstSentences(serviceComment, 200), weight: 0.3 },
          ],
          packagePriors[packageName] ?? 0.95,
        );
      }
    }
  }

  get methods(): APIMethod[] {
    return [...this.methodsById.values()];
  }

  resolveMethod(input: string): APIMethod | undefined {
    let id = input.trim();
    if (id.startsWith("/")) {
      id = id.slice(1);
    }
    if (id.startsWith(TYPE_PREFIX)) {
      id = id.slice(TYPE_PREFIX.length);
    }
    if (!id.includes("/")) {
      const idx = id.lastIndexOf(".");
      if (idx > 0) {
        id = `${id.slice(0, idx)}/${id.slice(idx + 1)}`;
      }
    }
    return (
      this.methodsById.get(id) ?? this.methodsByLowerId.get(id.toLowerCase())
    );
  }

  resolveType(input: string): DescMessage | DescEnum | undefined {
    const name = input.trim().replace(/^\./, "");
    const candidates = [name, `${TYPE_PREFIX}${name}`];
    for (const candidate of candidates) {
      const desc =
        this.registry.getMessage(candidate) ?? this.registry.getEnum(candidate);
      if (desc) {
        return desc;
      }
    }
    return undefined;
  }

  search(query: string, limit = 10): SearchHit<APIMethod>[] {
    return this.index.search(query, limit);
  }

  formatSearchResults(hits: SearchHit<APIMethod>[]): string {
    if (hits.length === 0) {
      return "No matching methods found. Try other keywords (e.g. the resource kind such as Service, User, Workspace, or an action such as list, create, delete).";
    }

    return hits
      .map((hit, i) => {
        const m = hit.item;
        const description = m.description
          ? ` — ${firstSentences(m.description, 160)}`
          : "";
        const flags = [m.risk, m.kind !== "unary" ? m.kind : undefined]
          .filter(Boolean)
          .join(", ");
        const invocable = m.invocable ? "" : " (not invocable)";
        return `${i + 1}. ${m.id}${description} [${flags}]${invocable}\n   request: ${m.requestType} → response: ${m.responseType}`;
      })
      .join("\n");
  }

  formatIndex(): string {
    const byPackage = new Map<string, DescService[]>();
    for (const service of this.services) {
      const pkg = shortTypeName(service.file.proto.package);
      byPackage.set(pkg, [...(byPackage.get(pkg) ?? []), service]);
    }

    const lines: string[] = [];
    for (const [pkg, services] of byPackage) {
      const description = packageDescriptions[pkg];
      lines.push(`${pkg}${description ? ` — ${description}` : ""}`);
      for (const service of services) {
        const methods = this.methods.filter(
          (m) => m.serviceName === service.typeName,
        );
        if (methods.length === 0) {
          continue;
        }
        const groups = new Map<string, string[]>();
        const singles: string[] = [];
        for (const m of methods) {
          if (m.noun && m.verb) {
            groups.set(m.noun, [...(groups.get(m.noun) ?? []), m.verb]);
          } else {
            singles.push(m.name);
          }
        }
        const parts = [...groups.entries()].map(([noun, verbs]) =>
          verbs.length === 1
            ? `${verbs[0]}${noun}`
            : `${noun}[${verbs.join("|")}]`,
        );
        lines.push(`  ${service.name}: ${[...parts, ...singles].join(", ")}`);
      }
    }
    return lines.join("\n");
  }

  describeMethod(method: APIMethod, opts: DescribeOptions = {}): string {
    const lines = [
      `Method: ${method.id}`,
      `gRPC method: ${method.grpcMethod}`,
      `Kind: ${method.kind}${method.invocable ? "" : " (not invocable by this agent)"}`,
      `Risk: ${method.risk}`,
    ];
    if (method.description) {
      lines.push(`Description: ${method.description.replace(/\n+/g, " ")}`);
    }
    lines.push(
      `Request type: ${method.requestType}`,
      `Response type: ${method.responseType}`,
      "",
      "Request schema (proto3 JSON mapping, every field is optional unless its description says otherwise):",
      this.renderTypes(method.desc.input, opts),
      "",
      "Response summary:",
      this.renderTypes(method.desc.output, {
        maxChars: 2500,
        maxDepth: 0,
        verbose: false,
      }),
    );
    return lines.join("\n");
  }

  describeType(
    desc: DescMessage | DescEnum,
    opts: DescribeOptions = {},
  ): string {
    if (desc.kind === "enum") {
      return this.renderEnumBlock(desc);
    }
    return [
      "Schema (proto3 JSON mapping, every field is optional unless its description says otherwise):",
      this.renderTypes(desc, opts),
    ].join("\n");
  }

  private renderEnumBlock(desc: DescEnum): string {
    const comment = this.getComment(desc);
    const lines = [`enum ${shortTypeName(desc.typeName)} {`];
    if (comment) {
      lines.unshift(`// ${firstSentences(comment, 300)}`);
    }
    for (const value of desc.values) {
      const c = this.getComment(value);
      lines.push(
        `  "${value.name}"${c ? `  // ${firstSentences(c, 160)}` : ""}`,
      );
    }
    lines.push("}");
    return lines.join("\n");
  }

  private renderTypes(root: DescMessage, opts: DescribeOptions): string {
    const maxChars = opts.maxChars ?? 14000;
    const maxDepth = opts.maxDepth ?? 3;
    const verbose = opts.verbose ?? false;

    const blocks: string[] = [];
    const seen = new Set<string>([root.typeName]);
    const queue: { desc: DescMessage; depth: number }[] = [
      { desc: root, depth: 0 },
    ];
    const skipped: string[] = [];
    let used = 0;

    while (queue.length > 0) {
      const { desc, depth } = queue.shift()!;
      const nested: DescMessage[] = [];
      const block = this.renderMessageBlock(
        desc,
        depth === 0 || verbose,
        nested,
      );
      if (used + block.length > maxChars && blocks.length > 0) {
        skipped.push(shortTypeName(desc.typeName));
        continue;
      }
      blocks.push(block);
      used += block.length;

      for (const child of nested) {
        if (seen.has(child.typeName)) {
          continue;
        }
        seen.add(child.typeName);
        if (depth + 1 > maxDepth) {
          skipped.push(shortTypeName(child.typeName));
          continue;
        }
        queue.push({ desc: child, depth: depth + 1 });
      }
    }

    if (skipped.length > 0) {
      blocks.push(
        `// Not expanded (use octelium_api_describe with {"type": "<name>"} to see them): ${skipped.join(", ")}`,
      );
    }
    return blocks.join("\n\n");
  }

  private renderMessageBlock(
    desc: DescMessage,
    verbose: boolean,
    nested: DescMessage[],
  ): string {
    const lines: string[] = [];
    const comment = this.getComment(desc);
    if (comment) {
      lines.push(
        `// ${verbose ? firstSentences(comment, 600) : firstSentences(comment, 240)}`,
      );
    }

    const wkt = describeWellKnown(desc.typeName);
    if (wkt) {
      lines.push(`${shortTypeName(desc.typeName)} = ${wkt}`);
      return lines.join("\n");
    }

    lines.push(`${shortTypeName(desc.typeName)} {`);
    const renderField = (field: DescField, indent: string) => {
      const fieldComment = this.getComment(field);
      const type = this.renderFieldType(field, nested);
      const c = fieldComment
        ? `  // ${firstSentences(fieldComment, verbose ? 400 : 200)}`
        : "";
      lines.push(`${indent}${field.jsonName}: ${type}${c}`);
    };

    for (const member of desc.members) {
      if (member.kind === "oneof") {
        const oneofComment = this.getComment(member);
        lines.push(
          `  oneof ${member.name} (set at most one)${oneofComment ? `  // ${firstSentences(oneofComment, 160)}` : ""} {`,
        );
        for (const field of member.fields) {
          renderField(field, "    ");
        }
        lines.push("  }");
      } else {
        renderField(member, "  ");
      }
    }
    lines.push("}");
    return lines.join("\n");
  }

  private renderFieldType(field: DescField, nested: DescMessage[]): string {
    const messageType = (desc: DescMessage) => {
      const wkt = describeWellKnown(desc.typeName);
      if (wkt) {
        return wkt;
      }
      nested.push(desc);
      return shortTypeName(desc.typeName);
    };

    const enumType = (desc: DescEnum) => {
      const values = desc.values.map((v) => `"${v.name}"`);
      const shown =
        values.length > 16
          ? `${values.slice(0, 16).join(" | ")} | …`
          : values.join(" | ");
      return `enum ${shortTypeName(desc.typeName)} (${shown})`;
    };

    switch (field.fieldKind) {
      case "scalar":
        return scalarTypeName(field.scalar);
      case "enum":
        return enumType(field.enum);
      case "message":
        return messageType(field.message);
      case "list":
        switch (field.listKind) {
          case "scalar":
            return `${scalarTypeName(field.scalar)}[]`;
          case "enum":
            return `${enumType(field.enum)}[]`;
          case "message":
            return `${messageType(field.message)}[]`;
        }
        break;
      case "map": {
        const key = scalarTypeName(field.mapKey);
        switch (field.mapKind) {
          case "scalar":
            return `map<${key}, ${scalarTypeName(field.scalar)}>`;
          case "enum":
            return `map<${key}, ${enumType(field.enum)}>`;
          case "message":
            return `map<${key}, ${messageType(field.message)}>`;
        }
      }
    }
    return "unknown";
  }
}

export const scalarTypeName = (scalar: ScalarType): string => {
  switch (scalar) {
    case ScalarType.STRING:
      return "string";
    case ScalarType.BOOL:
      return "bool";
    case ScalarType.BYTES:
      return "bytes (base64 string)";
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      return "number";
    case ScalarType.INT32:
    case ScalarType.SINT32:
    case ScalarType.SFIXED32:
      return "int32";
    case ScalarType.UINT32:
    case ScalarType.FIXED32:
      return "uint32";
    case ScalarType.INT64:
    case ScalarType.SINT64:
    case ScalarType.SFIXED64:
      return "int64 (number or decimal string)";
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
      return "uint64 (number or decimal string)";
  }
  return "unknown";
};

export const describeWellKnown = (typeName: string): string | undefined => {
  switch (typeName) {
    case "google.protobuf.Timestamp":
      return 'timestamp (RFC 3339 string, e.g. "2026-01-02T15:04:05Z")';
    case "google.protobuf.Duration":
      return 'duration (string with "s" suffix, e.g. "3.5s")';
    case "google.protobuf.Struct":
      return "object (arbitrary JSON object)";
    case "google.protobuf.Value":
      return "any JSON value";
    case "google.protobuf.ListValue":
      return "array (arbitrary JSON values)";
    case "google.protobuf.Empty":
      return "{}";
    case "google.protobuf.FieldMask":
      return 'string (comma-separated field paths, e.g. "spec.port,metadata.labels")';
    case "google.protobuf.Any":
      return 'object with "@type" (type URL) and the fields of that type';
    case "google.protobuf.StringValue":
      return "string | null";
    case "google.protobuf.BoolValue":
      return "bool | null";
    case "google.protobuf.Int32Value":
    case "google.protobuf.UInt32Value":
    case "google.protobuf.DoubleValue":
    case "google.protobuf.FloatValue":
      return "number | null";
    case "google.protobuf.Int64Value":
    case "google.protobuf.UInt64Value":
      return "int64 (number or decimal string) | null";
    case "google.protobuf.BytesValue":
      return "bytes (base64 string) | null";
  }
  return undefined;
};

export const methodTitle = (method: APIMethod): string =>
  `${splitCamelCase(method.name)} (${method.packageName})`;
