import {
  formatResourceURI,
  type ContextItem,
  type PageContext,
  type ResourceRef,
  type RunContext,
} from "../protocol/index.ts";
import { badRequest } from "../errors.ts";

export const CONTEXT_LIMITS = {
  path: 2048,
  title: 200,
  details: 12,
  detailKey: 64,
  detailValue: 256,
  resources: 20,
  items: 10,
  kind: 64,
  apiVersion: 64,
  name: 256,
  uid: 128,
  itemData: 8000,
} as const;

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const optionalString = (
  value: unknown,
  field: string,
  max: number,
): string | undefined => {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }
  if (typeof value !== "string") {
    throw badRequest(`The context field "${field}" must be a string`);
  }
  const trimmed = value.trim();
  if (trimmed.length > max) {
    throw badRequest(`The context field "${field}" exceeds ${max} characters`);
  }
  return trimmed === "" ? undefined : trimmed;
};

const requiredString = (value: unknown, field: string, max: number): string => {
  const ret = optionalString(value, field, max);
  if (ret === undefined) {
    throw badRequest(`The context field "${field}" is required`);
  }
  return ret;
};

const sanitizePage = (raw: unknown): PageContext | undefined => {
  if (raw === undefined || raw === null) {
    return undefined;
  }
  if (!isObject(raw)) {
    throw badRequest('The context field "page" must be an object');
  }
  const path = requiredString(raw.path, "page.path", CONTEXT_LIMITS.path);
  if (!path.startsWith("/") || path.startsWith("//")) {
    throw badRequest('The context field "page.path" must be a console path');
  }
  const page: PageContext = { path };
  const title = optionalString(raw.title, "page.title", CONTEXT_LIMITS.title);
  if (title) {
    page.title = title;
  }
  if (raw.details !== undefined && raw.details !== null) {
    if (!isObject(raw.details)) {
      throw badRequest('The context field "page.details" must be an object');
    }
    const entries = Object.entries(raw.details);
    if (entries.length > CONTEXT_LIMITS.details) {
      throw badRequest(
        `The context field "page.details" exceeds ${CONTEXT_LIMITS.details} entries`,
      );
    }
    const details: Record<string, string> = {};
    for (const [key, value] of entries) {
      const k = requiredString(key, "page.details", CONTEXT_LIMITS.detailKey);
      const v = optionalString(
        value,
        `page.details.${k}`,
        CONTEXT_LIMITS.detailValue,
      );
      if (v !== undefined) {
        details[k] = v;
      }
    }
    if (Object.keys(details).length > 0) {
      page.details = details;
    }
  }
  return page;
};

const sanitizeResources = (raw: unknown): ResourceRef[] => {
  if (raw === undefined || raw === null) {
    return [];
  }
  if (!Array.isArray(raw)) {
    throw badRequest('The context field "resources" must be an array');
  }
  if (raw.length > CONTEXT_LIMITS.resources) {
    throw badRequest(
      `The context field "resources" exceeds ${CONTEXT_LIMITS.resources} entries`,
    );
  }
  return raw.map((itm, idx) => {
    if (!isObject(itm)) {
      throw badRequest(
        `The context field "resources[${idx}]" must be an object`,
      );
    }
    const ref: ResourceRef = {
      apiVersion: requiredString(
        itm.apiVersion,
        `resources[${idx}].apiVersion`,
        CONTEXT_LIMITS.apiVersion,
      ),
      kind: requiredString(
        itm.kind,
        `resources[${idx}].kind`,
        CONTEXT_LIMITS.kind,
      ),
    };
    const name = optionalString(
      itm.name,
      `resources[${idx}].name`,
      CONTEXT_LIMITS.name,
    );
    const uid = optionalString(
      itm.uid,
      `resources[${idx}].uid`,
      CONTEXT_LIMITS.uid,
    );
    if (!name && !uid) {
      throw badRequest(
        `The context field "resources[${idx}]" requires a name or a uid`,
      );
    }
    if (name) {
      ref.name = name;
    }
    if (uid) {
      ref.uid = uid;
    }
    return ref;
  });
};

const truncateData = (data: unknown): unknown => {
  let serialized: string | undefined;
  try {
    serialized = JSON.stringify(data);
  } catch {
    throw badRequest("The context item data must be JSON");
  }
  if (serialized === undefined) {
    return undefined;
  }
  if (serialized.length <= CONTEXT_LIMITS.itemData) {
    return JSON.parse(serialized) as unknown;
  }
  return `${serialized.slice(0, CONTEXT_LIMITS.itemData)}… (truncated)`;
};

const sanitizeItems = (raw: unknown): ContextItem[] => {
  if (raw === undefined || raw === null) {
    return [];
  }
  if (!Array.isArray(raw)) {
    throw badRequest('The context field "items" must be an array');
  }
  if (raw.length > CONTEXT_LIMITS.items) {
    throw badRequest(
      `The context field "items" exceeds ${CONTEXT_LIMITS.items} entries`,
    );
  }
  return raw.map((itm, idx) => {
    if (!isObject(itm)) {
      throw badRequest(`The context field "items[${idx}]" must be an object`);
    }
    const item: ContextItem = {
      kind: requiredString(itm.kind, `items[${idx}].kind`, CONTEXT_LIMITS.kind),
    };
    const id = optionalString(itm.id, `items[${idx}].id`, CONTEXT_LIMITS.uid);
    const title = optionalString(
      itm.title,
      `items[${idx}].title`,
      CONTEXT_LIMITS.title,
    );
    if (id) {
      item.id = id;
    }
    if (title) {
      item.title = title;
    }
    if (itm.data !== undefined && itm.data !== null) {
      const data = truncateData(itm.data);
      if (data !== undefined) {
        item.data = data;
      }
    }
    return item;
  });
};

export const sanitizeContext = (raw: unknown): RunContext | undefined => {
  if (raw === undefined || raw === null) {
    return undefined;
  }
  if (!isObject(raw)) {
    throw badRequest('The "context" field must be an object');
  }
  const ret: RunContext = {};
  const page = sanitizePage(raw.page);
  const resources = sanitizeResources(raw.resources);
  const items = sanitizeItems(raw.items);
  if (page) {
    ret.page = page;
  }
  if (resources.length > 0) {
    ret.resources = resources;
  }
  if (items.length > 0) {
    ret.items = items;
  }
  return Object.keys(ret).length > 0 ? ret : undefined;
};

const quote = (value: string) => JSON.stringify(value);

const formatResource = (ref: ResourceRef): string => {
  const ret = [`${ref.apiVersion} ${ref.kind}`];
  if (ref.name) {
    ret.push(quote(ref.name));
  }
  if (ref.uid) {
    ret.push(`(UID: ${ref.uid})`);
  }
  if (ref.name) {
    ret.push(formatResourceURI(ref));
  }
  return ret.join(" ");
};

const formatItem = (item: ContextItem): string[] => {
  const head = [item.kind];
  if (item.title) {
    head.push(quote(item.title));
  }
  if (item.id) {
    head.push(`(ID: ${item.id})`);
  }
  const ret = [`- ${head.join(" ")}`];
  if (item.data !== undefined) {
    ret.push(
      `  ${typeof item.data === "string" ? item.data : JSON.stringify(item.data)}`,
    );
  }
  return ret;
};

export const formatContext = (ctx: RunContext): string => {
  const lines: string[] = [];
  if (ctx.page) {
    lines.push(
      ctx.page.title
        ? `Page: ${ctx.page.title} (${ctx.page.path})`
        : `Page: ${ctx.page.path}`,
    );
    for (const [key, value] of Object.entries(ctx.page.details ?? {})) {
      lines.push(`- ${key}: ${value}`);
    }
  }
  if (ctx.resources && ctx.resources.length > 0) {
    lines.push("Resources:");
    for (const ref of ctx.resources) {
      lines.push(`- ${formatResource(ref)}`);
    }
  }
  if (ctx.items && ctx.items.length > 0) {
    lines.push("Items:");
    for (const item of ctx.items) {
      lines.push(...formatItem(item));
    }
  }
  return `<console_context>\n${lines.join("\n")}\n</console_context>`;
};

export const contextSummary = (ctx: RunContext): string => {
  const labels = [
    ...(ctx.resources ?? []).map((ref) =>
      `${ref.kind} ${ref.name ?? ref.uid ?? ""}`.trim(),
    ),
    ...(ctx.items ?? []).map((item) =>
      item.title ? `${item.kind} ${item.title}` : item.kind,
    ),
  ];
  if (labels.length === 0 && ctx.page) {
    labels.push(ctx.page.title ?? ctx.page.path);
  }
  return labels.slice(0, 3).join(", ");
};
