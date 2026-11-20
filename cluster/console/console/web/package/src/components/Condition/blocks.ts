import * as CoreP from "@/apis/corev1/corev1";
import {
  celString,
  celStringList,
  conjuncts,
  escapeRegExp,
  exact,
  LIST_PATTERN,
  parseIntList,
  parseStringList,
  parseStringLiteral,
  sameCEL,
  STRING_PATTERN,
} from "./cel";

export type CategoryId =
  | "identity"
  | "resource"
  | "session"
  | "device"
  | "network"
  | "time"
  | "http"
  | "api"
  | "protocol"
  | "mcp"
  | "llm";

export type Option = { value: string; label: string };

export type MatchOp = "is" | "in" | "prefix" | "suffix" | "contains" | "regex";
export type MatchValue = { op: MatchOp; values: string[] };
export type NumberOp = "==" | "<" | "<=" | ">" | ">=";
export type NumberValue = { op: NumberOp; value?: number };
export type Transform = "upper" | "lower" | "cidr";

export type FieldDef = FieldKind & { visible?: (values: Values) => boolean };

type FieldKind =
  | { kind: "refs"; label: string; resource: string }
  | { kind: "options"; label: string; options: Option[] }
  | {
      kind: "strings";
      label: string;
      placeholder?: string;
      suggestions?: string[];
      transform?: Transform;
    }
  | { kind: "text"; label: string; placeholder?: string; optional?: boolean }
  | {
      kind: "match";
      label: string;
      placeholder?: string;
      suggestions?: string[];
      transform?: Transform;
    }
  | { kind: "number"; label: string; placeholder?: string }
  | { kind: "timestamp"; label: string; optional?: boolean }
  | { kind: "timezone"; label: string }
  | { kind: "days"; label: string }
  | { kind: "hours"; label: string }
  | { kind: "switch"; label: string; description?: string }
  | { kind: "choice"; label: string; options: Option[] }
  | { kind: "countries"; label: string };

export type Values = Record<string, any>;

export type BlockDef = {
  id: string;
  category: CategoryId;
  group?: string;
  title: string;
  description: string;
  keywords?: string[];
  fields: [string, FieldDef][];
  defaults: () => Values;
  toCEL: (values: Values) => string;
  parse: (expr: string) => Values | undefined;
  summary: (values: Values) => string;
  issues: (values: Values) => string[];
};

export const CATEGORIES: {
  id: CategoryId;
  label: string;
  description: string;
}[] = [
  {
    id: "identity",
    label: "Users & Groups",
    description: "Who makes the request",
  },
  {
    id: "resource",
    label: "Services & Namespaces",
    description: "What the request is for",
  },
  {
    id: "session",
    label: "Sessions & Authentication",
    description: "How the User signed in",
  },
  {
    id: "device",
    label: "Devices",
    description: "The Device and its posture",
  },
  {
    id: "network",
    label: "Network & Location",
    description: "Where the request comes from",
  },
  { id: "time", label: "Time", description: "When the request happens" },
  { id: "http", label: "HTTP", description: "HTTP and web requests" },
  {
    id: "api",
    label: "Octelium API",
    description: "Access to the Cluster's own APIs",
  },
  {
    id: "protocol",
    label: "Protocols",
    description: "SSH, Kubernetes, gRPC, databases, DNS and SOCKS5",
  },
  { id: "mcp", label: "MCP", description: "Model Context Protocol requests" },
  { id: "llm", label: "LLM", description: "Requests to LLM Services" },
];

const SKIPPED_ENUM =
  /^(TYPE_UNSET|TYPE_UNKNOWN|MODE_UNSET|AAL_UNSET|OS_TYPE_UNKNOWN|ASSURANCE_UNKNOWN|RISK_LEVEL_UNKNOWN|PROTOCOL_UNSET|OPERATION_UNSET|ADDRESS_TYPE_UNSPECIFIED|ESTIMATE_QUALITY_UNSET|SIGNAL_STATE_UNKNOWN)$/;

const ACRONYMS = new Set([
  "HTTP",
  "TCP",
  "UDP",
  "SSH",
  "GRPC",
  "MYSQL",
  "AAL1",
  "AAL2",
  "AAL3",
  "IOS",
  "IPV4",
  "IPV6",
  "OAUTH2",
  "ID",
  "MAC",
  "LLM",
  "OPENAI",
  "DNS",
  "SOCKS5",
  "RDP",
  "MCP",
  "FIDO",
]);

const WORD_LABELS: Record<string, string> = {
  GRPC: "gRPC",
  MYSQL: "MySQL",
  POSTGRES: "PostgreSQL",
  IOS: "iOS",
  MAC: "macOS",
  CHROMEOS: "ChromeOS",
  OPENAI: "OpenAI",
  IPV4: "IPv4",
  IPV6: "IPv6",
  OAUTH2: "OAuth2",
  RDP_WEB: "RDP (web)",
};

export const enumLabel = (value: string) =>
  WORD_LABELS[value] ??
  value
    .split("_")
    .map((word, index) =>
      ACRONYMS.has(word)
        ? word
        : index === 0
          ? word.charAt(0) + word.slice(1).toLowerCase()
          : word.toLowerCase(),
    )
    .join(" ");

const enumOptions = (values: object): Option[] =>
  Object.keys(values)
    .filter((key) => Number.isNaN(Number(key)) && !SKIPPED_ENUM.test(key))
    .map((value) => ({ value, label: enumLabel(value) }));

const optionsOf = (values: string[], labels: Record<string, string> = {}) =>
  values.map((value) => ({ value, label: labels[value] ?? value }));

const orList = (values: string[], conjunction = "or") => {
  if (values.length === 0) return "…";
  if (values.length === 1) return values[0];
  if (values.length <= 3) {
    return `${values.slice(0, -1).join(", ")} ${conjunction} ${values.at(-1)}`;
  }
  return `${values.slice(0, 2).join(", ")} ${conjunction} ${values.length - 2} more`;
};

const quoted = (values: string[], conjunction = "or") =>
  orList(
    values.map((value) => `"${value}"`),
    conjunction,
  );

const labelsOf = (options: Option[], values: string[]) =>
  values.map(
    (value) => options.find((option) => option.value === value)?.label ?? value,
  );

const STR = STRING_PATTERN;
const LIST = LIST_PATTERN;

const eqOrIn = (field: string, values: string[]) =>
  values.length === 1
    ? `${field} == ${celString(values[0])}`
    : `${field} in ${celStringList(values)}`;

const parseEqOrInOp = (
  field: string,
  expr: string,
): { values: string[]; list: boolean } | undefined => {
  const f = escapeRegExp(field);
  const eq =
    exact(`${f}\\s*==\\s*(${STR})`).exec(expr) ??
    exact(`(${STR})\\s*==\\s*${f}`).exec(expr);
  if (eq) {
    const value = parseStringLiteral(eq[1]);
    return value === undefined ? undefined : { values: [value], list: false };
  }
  const list = exact(`${f}\\s+in\\s+(${LIST})`).exec(expr);
  if (list) {
    const values = parseStringList(list[1]);
    return values ? { values, list: true } : undefined;
  }
  return undefined;
};

const parseEqOrIn = (field: string, expr: string) =>
  parseEqOrInOp(field, expr)?.values;

const transformValue = (value: string, transform?: Transform) => {
  const trimmed = value.trim();
  switch (transform) {
    case "upper":
      return trimmed.toUpperCase();
    case "lower":
      return trimmed.toLowerCase();
    case "cidr":
      return trimmed && !trimmed.includes("/")
        ? `${trimmed}/${trimmed.includes(":") ? 128 : 32}`
        : trimmed;
    default:
      return value;
  }
};

const requiredIssues =
  (fields: [string, FieldDef][]) =>
  (values: Values): string[] => {
    const ret: string[] = [];
    for (const [key, field] of fields) {
      const value = values[key];
      switch (field.kind) {
        case "refs":
        case "strings":
        case "options":
        case "countries":
          if (!Array.isArray(value) || value.length === 0) {
            ret.push(`Choose at least one ${field.label.toLowerCase()}`);
          }
          break;
        case "match":
          if (
            !value ||
            !Array.isArray(value.values) ||
            value.values.filter((item: string) => item !== "").length === 0
          ) {
            ret.push(`Enter the ${field.label.toLowerCase()}`);
          }
          break;
        case "text":
          if (!field.optional && !String(value ?? "").trim()) {
            ret.push(`Enter the ${field.label.toLowerCase()}`);
          }
          break;
        case "number":
          if (
            !value ||
            typeof value.value !== "number" ||
            Number.isNaN(value.value)
          ) {
            ret.push(`Enter the ${field.label.toLowerCase()}`);
          }
          break;
        case "timestamp":
          if (!field.optional && !value) {
            ret.push(`Choose the ${field.label.toLowerCase()}`);
          }
          break;
      }
    }
    return ret;
  };

const MATCH_LABELS: Record<MatchOp, string> = {
  is: "is",
  in: "is one of",
  prefix: "starts with",
  suffix: "ends with",
  contains: "contains",
  regex: "matches",
};

export const MATCH_OPS: Option[] = (Object.keys(MATCH_LABELS) as MatchOp[]).map(
  (value) => ({ value, label: MATCH_LABELS[value] }),
);

const METHOD_OF: Record<string, MatchOp> = {
  startsWith: "prefix",
  endsWith: "suffix",
  contains: "contains",
  matches: "regex",
};

const FUNCTION_OF: Partial<Record<MatchOp, string>> = {
  prefix: "startsWith",
  suffix: "endsWith",
  contains: "contains",
  regex: "matches",
};

export const matchCEL = (
  field: string,
  match: MatchValue,
  transform?: Transform,
) => {
  const values = match.values
    .map((value) => transformValue(value, transform))
    .filter((value) => value !== "");
  const fn = FUNCTION_OF[match.op];
  if (!fn) return eqOrIn(field, values);
  if (values.length === 1) return `${field}.${fn}(${celString(values[0])})`;
  return `${celStringList(values)}.exists(x, ${field}.${fn}(x))`;
};

export const parseMatch = (
  field: string,
  expr: string,
): MatchValue | undefined => {
  const f = escapeRegExp(field);
  const eq = parseEqOrInOp(field, expr);
  if (eq) return { op: eq.list ? "in" : "is", values: eq.values };
  const call = exact(
    `${f}\\.(startsWith|endsWith|contains|matches)\\(\\s*(${STR})\\s*\\)`,
  ).exec(expr);
  if (call) {
    const value = parseStringLiteral(call[2]);
    if (value === undefined) return undefined;
    return { op: METHOD_OF[call[1]], values: [value] };
  }
  const list = exact(
    `(${LIST})\\.exists\\(\\s*(\\w+)\\s*,\\s*${f}\\.(startsWith|endsWith|contains|matches)\\(\\s*(\\w+)\\s*\\)\\s*\\)`,
  ).exec(expr);
  if (list && list[2] === list[4]) {
    const parsed = parseStringList(list[1]);
    if (!parsed) return undefined;
    return { op: METHOD_OF[list[3]], values: parsed };
  }
  return undefined;
};

export const matchSummary = (match?: MatchValue) =>
  match
    ? `${MATCH_LABELS[match.op]} ${quoted(match.values.filter(Boolean))}`
    : "…";

const NUMBER_OPS: NumberOp[] = ["==", "<", "<=", ">", ">="];

export const NUMBER_LABELS: Record<NumberOp, string> = {
  "==": "is",
  "<": "is less than",
  "<=": "is at most",
  ">": "is more than",
  ">=": "is at least",
};

type Numeric = "int" | "uint";

const numberLiteral = (numeric: Numeric, value: number) =>
  numeric === "uint"
    ? `uint(${Math.max(0, Math.trunc(value))})`
    : `${Math.trunc(value)}`;

const NUMBER_PATTERN = String.raw`(==|<=|>=|<|>)\s*(?:uint\(\s*(\d+)\s*\)|(-?\d+)u?)`;

type FactoryBase = {
  id: string;
  category: CategoryId;
  group?: string;
  title: string;
  description: string;
  keywords?: string[];
};

const build = (
  base: FactoryBase,
  rest: Omit<BlockDef, keyof FactoryBase | "issues"> & {
    issues?: (values: Values) => string[];
  },
): BlockDef => ({
  ...base,
  ...rest,
  issues: rest.issues ?? requiredIssues(rest.fields),
});

const refsBlock = (
  base: FactoryBase & { field: string; resource: string; subject: string },
): BlockDef =>
  build(base, {
    fields: [
      [
        "names",
        { kind: "refs", label: base.resource, resource: base.resource },
      ],
    ],
    defaults: () => ({ names: [] }),
    toCEL: (values) => eqOrIn(base.field, values.names),
    parse: (expr) => {
      const names = parseEqOrIn(base.field, expr);
      return names && names.length > 0 ? { names } : undefined;
    },
    summary: (values) => `${base.subject} is ${orList(values.names)}`,
  });

const optionsBlock = (
  base: FactoryBase & {
    field: string;
    options: Option[];
    subject: string;
    label?: string;
    initial?: string[];
  },
): BlockDef =>
  build(base, {
    fields: [
      [
        "values",
        {
          kind: "options",
          label: base.label ?? "Value",
          options: base.options,
        },
      ],
    ],
    defaults: () => ({ values: base.initial ?? [] }),
    toCEL: (values) => eqOrIn(base.field, values.values),
    parse: (expr) => {
      const values = parseEqOrIn(base.field, expr);
      return values && values.length > 0 ? { values } : undefined;
    },
    summary: (values) =>
      `${base.subject} is ${orList(labelsOf(base.options, values.values))}`,
  });

const flagBlock = (
  base: FactoryBase & { expr: string; aliases?: string[]; summary: string },
): BlockDef =>
  build(base, {
    fields: [],
    defaults: () => ({}),
    toCEL: () => base.expr,
    parse: (expr) =>
      [base.expr, ...(base.aliases ?? [])].some((item) => sameCEL(item, expr))
        ? {}
        : undefined,
    summary: () => base.summary,
  });

const matchBlock = (
  base: FactoryBase & {
    field: string;
    subject: string;
    label?: string;
    placeholder?: string;
    suggestions?: string[];
    transform?: Transform;
    guard?: string;
    initial?: MatchValue;
  },
): BlockDef => {
  const withGuard = (expr: string) =>
    base.guard ? `${base.guard} && ${expr}` : expr;
  return build(base, {
    fields: [
      [
        "match",
        {
          kind: "match",
          label: base.label ?? "Value",
          placeholder: base.placeholder,
          suggestions: base.suggestions,
          transform: base.transform,
        },
      ],
    ],
    defaults: () => ({ match: base.initial ?? { op: "is", values: [] } }),
    toCEL: (values) =>
      withGuard(matchCEL(base.field, values.match, base.transform)),
    parse: (expr) => {
      let source = expr;
      if (base.guard) {
        const parts = conjuncts(expr);
        if (parts.length !== 2 || !sameCEL(parts[0], base.guard)) {
          return undefined;
        }
        source = parts[1];
      }
      const match = parseMatch(base.field, source);
      return match ? { match } : undefined;
    },
    summary: (values) => `${base.subject} ${matchSummary(values.match)}`,
  });
};

const keyedMatchBlock = (
  base: FactoryBase & {
    map: string;
    subject: string;
    keyLabel: string;
    keyPlaceholder?: string;
    keyTransform?: Transform;
    placeholder?: string;
  },
): BlockDef => {
  const fieldOf = (key: string) =>
    `${base.map}[${celString(transformValue(key ?? "", base.keyTransform))}]`;
  return build(base, {
    fields: [
      [
        "key",
        {
          kind: "text",
          label: base.keyLabel,
          placeholder: base.keyPlaceholder,
        },
      ],
      [
        "match",
        { kind: "match", label: "Value", placeholder: base.placeholder },
      ],
    ],
    defaults: () => ({ key: "", match: { op: "is", values: [] } }),
    toCEL: (values) => matchCEL(fieldOf(values.key), values.match),
    parse: (expr) => {
      const head = new RegExp(
        `^\\s*${escapeRegExp(base.map)}\\[\\s*(${STR})\\s*\\]`,
      ).exec(expr);
      if (!head) return undefined;
      const key = parseStringLiteral(head[1]);
      if (key === undefined) return undefined;
      const match = parseMatch(`${base.map}[${head[1]}]`, expr);
      return match ? { key, match } : undefined;
    },
    summary: (values) =>
      `${base.subject} "${values.key || "…"}" ${matchSummary(values.match)}`,
  });
};

const presenceBlock = (
  base: FactoryBase & {
    map: string;
    subject: string;
    keyLabel: string;
    keyPlaceholder?: string;
    keyTransform?: Transform;
  },
): BlockDef =>
  build(base, {
    fields: [
      [
        "key",
        {
          kind: "text",
          label: base.keyLabel,
          placeholder: base.keyPlaceholder,
        },
      ],
    ],
    defaults: () => ({ key: "" }),
    toCEL: (values) =>
      `${celString(transformValue(values.key ?? "", base.keyTransform))} in ${base.map}`,
    parse: (expr) => {
      const m = exact(`(${STR})\\s+in\\s+${escapeRegExp(base.map)}`).exec(expr);
      const key = m ? parseStringLiteral(m[1]) : undefined;
      return key !== undefined ? { key } : undefined;
    },
    summary: (values) => `${base.subject} "${values.key || "…"}" is present`,
  });

const numberBlock = (
  base: FactoryBase & {
    field: string;
    numeric: Numeric;
    subject: string;
    label?: string;
    initial?: NumberValue;
  },
): BlockDef =>
  build(base, {
    fields: [["number", { kind: "number", label: base.label ?? "Value" }]],
    defaults: () => ({
      number: base.initial ?? { op: "<=", value: undefined },
    }),
    toCEL: (values) =>
      `${base.field} ${values.number.op} ${numberLiteral(base.numeric, values.number.value ?? 0)}`,
    parse: (expr) => {
      const m = exact(`${escapeRegExp(base.field)}\\s*${NUMBER_PATTERN}`).exec(
        expr,
      );
      if (!m) return undefined;
      const literal = m[2] ?? m[3];
      if (base.numeric === "uint" && m[3]?.startsWith("-")) return undefined;
      return { number: { op: m[1] as NumberOp, value: Number(literal) } };
    },
    summary: (values) =>
      `${base.subject} ${NUMBER_LABELS[values.number?.op as NumberOp] ?? "is"} ${values.number?.value ?? "…"}`,
  });

const DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export const DAYS: Option[] = DAY_LABELS.map((label, index) => ({
  value: String(index),
  label,
}));

const toMinutes = (value: string) => {
  const m = /^(\d{1,2}):(\d{2})$/.exec(value ?? "");
  if (!m) return undefined;
  const hours = Number(m[1]);
  const minutes = Number(m[2]);
  if (hours > 23 || minutes > 59) return undefined;
  return hours * 60 + minutes;
};

const fromMinutes = (value: number) =>
  `${String(Math.floor(value / 60)).padStart(2, "0")}:${String(value % 60).padStart(2, "0")}`;

const tzArg = (timezone?: string) => (timezone ? celString(timezone) : "");

export const timeWindowCEL = (values: Values) => {
  const tz = tzArg(values.timezone);
  const days: number[] = [...(values.days ?? [])]
    .map(Number)
    .sort((a, b) => a - b);
  const parts: string[] = [];
  if (days.length > 0 && days.length < 7) {
    parts.push(`now().getDayOfWeek(${tz}) in [${days.join(", ")}]`);
  }
  const start = toMinutes(values.from);
  const end = toMinutes(values.to);
  if (start !== undefined && end !== undefined && start !== end) {
    const hourly = start % 60 === 0 && end % 60 === 0;
    const clock = hourly
      ? `now().getHours(${tz})`
      : `now().getHours(${tz}) * 60 + now().getMinutes(${tz})`;
    const a = hourly ? start / 60 : start;
    const b = hourly ? end / 60 : end;
    parts.push(
      start < end
        ? `${clock} >= ${a} && ${clock} < ${b}`
        : `(${clock} >= ${a} || ${clock} < ${b})`,
    );
  }
  return parts.length > 0 ? parts.join(" && ") : "true";
};

const parseTimeWindow = (expr: string): Values | undefined => {
  if (!/now\(\)\.get(DayOfWeek|Hours)\(/.test(expr)) return undefined;
  const tzMatch = new RegExp(
    `now\\(\\)\\.get(?:DayOfWeek|Hours|Minutes)\\(\\s*(${STR})?\\s*\\)`,
  ).exec(expr);
  const timezone = tzMatch?.[1] ? parseStringLiteral(tzMatch[1]) : "";
  if (timezone === undefined) return undefined;
  const daysMatch = /getDayOfWeek\([^)]*\)\s+in\s+(\[[\d,\s]*\])/.exec(expr);
  const days = daysMatch ? parseIntList(daysMatch[1]) : [];
  if (!days) return undefined;
  const minutes = /getMinutes\(/.test(expr);
  const lower = />=\s*(\d+)/.exec(expr);
  const upper = /<\s*(\d+)/.exec(expr);
  const values: Values = {
    days: days.map(String),
    timezone,
    from: "",
    to: "",
  };
  if (lower && upper) {
    const scale = minutes ? 1 : 60;
    values.from = fromMinutes(Number(lower[1]) * scale);
    values.to = fromMinutes(Number(upper[1]) * scale);
  }
  return sameCEL(timeWindowCEL(values), expr) ? values : undefined;
};

const describeDays = (days: string[]) => {
  const sorted = [...days].map(Number).sort((a, b) => a - b);
  if (sorted.length === 0 || sorted.length === 7) return "Every day";
  if (sorted.join() === "1,2,3,4,5") return "Mon–Fri";
  if (sorted.join() === "0,6") return "Weekends";
  return sorted.map((day) => DAY_LABELS[day]).join(", ");
};

const timeWindow = build(
  {
    id: "time.window",
    category: "time",
    title: "Time window",
    description: "Days of the week and hours of the day, in a timezone",
    keywords: [
      "business hours",
      "schedule",
      "working hours",
      "weekday",
      "timezone",
      "shift",
    ],
  },
  {
    fields: [
      ["days", { kind: "days", label: "Days" }],
      ["from", { kind: "hours", label: "Hours" }],
      ["timezone", { kind: "timezone", label: "Timezone" }],
    ],
    defaults: () => ({
      days: ["1", "2", "3", "4", "5"],
      from: "09:00",
      to: "17:00",
      timezone: "",
    }),
    toCEL: timeWindowCEL,
    parse: parseTimeWindow,
    summary: (values) => {
      const hours =
        values.from && values.to ? `${values.from}–${values.to}` : "all day";
      return `${describeDays(values.days ?? [])}, ${hours} ${values.timezone || "UTC"}`;
    },
    issues: (values) => {
      const ret: string[] = [];
      const days = values.days ?? [];
      const hasHours = Boolean(values.from || values.to);
      if ((days.length === 0 || days.length === 7) && !hasHours) {
        ret.push("Choose the days or the hours of the window");
      }
      if (hasHours) {
        const start = toMinutes(values.from);
        const end = toMinutes(values.to);
        if (start === undefined || end === undefined) {
          ret.push("Set both the start and the end hour");
        } else if (start === end) {
          ret.push("The window must start and end at different times");
        }
      }
      return ret;
    },
  },
);

const dayType = build(
  {
    id: "time.dayType",
    category: "time",
    title: "Weekdays or weekends",
    description: "Monday to Friday, or Saturday and Sunday",
    keywords: ["weekday", "weekend", "timezone"],
  },
  {
    fields: [
      [
        "type",
        {
          kind: "choice",
          label: "Days",
          options: [
            { value: "weekday", label: "Weekdays" },
            { value: "weekend", label: "Weekends" },
          ],
        },
      ],
      ["timezone", { kind: "timezone", label: "Timezone" }],
    ],
    defaults: () => ({ type: "weekday", timezone: "" }),
    toCEL: (values) => {
      const name = values.type === "weekend" ? "Weekend" : "Weekday";
      return values.timezone
        ? `time.is${name}InTZ(now(), ${celString(values.timezone)})`
        : `time.is${name}(now())`;
    },
    parse: (expr) => {
      const m = exact(
        `time\\.is(Weekday|Weekend)(?:InTZ)?\\(\\s*now\\(\\)\\s*(?:,\\s*(${STR})\\s*)?\\)`,
      ).exec(expr);
      if (!m) return undefined;
      if (/InTZ/.test(expr) !== Boolean(m[2])) return undefined;
      const timezone = m[2] ? parseStringLiteral(m[2]) : "";
      if (timezone === undefined) return undefined;
      return { type: m[1].toLowerCase(), timezone };
    },
    summary: (values) =>
      `${values.type === "weekend" ? "Weekends" : "Weekdays"} ${values.timezone || "UTC"}`,
  },
);

const toTimestamp = (value?: string) => {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
};

const dateRange = build(
  {
    id: "time.range",
    category: "time",
    title: "Date range",
    description: "Only after a date, before a date, or between both",
    keywords: [
      "temporary",
      "expire",
      "until",
      "after",
      "before",
      "deadline",
      "date",
    ],
  },
  {
    fields: [
      ["after", { kind: "timestamp", label: "Starts", optional: true }],
      ["before", { kind: "timestamp", label: "Ends", optional: true }],
    ],
    defaults: () => ({ after: "", before: "" }),
    toCEL: (values) => {
      const parts: string[] = [];
      const after = toTimestamp(values.after);
      const before = toTimestamp(values.before);
      if (after) parts.push(`now() > timestamp(${celString(after)})`);
      if (before) parts.push(`now() < timestamp(${celString(before)})`);
      return parts.length > 0 ? parts.join(" && ") : "true";
    },
    parse: (expr) => {
      const parts = conjuncts(expr);
      if (parts.length > 2) return undefined;
      const values: Values = { after: "", before: "" };
      for (const part of parts) {
        const m = exact(
          `now\\(\\)\\s*(<|>)\\s*timestamp\\(\\s*(${STR})\\s*\\)`,
        ).exec(part);
        if (!m) return undefined;
        const value = parseStringLiteral(m[2]);
        if (!value || Number.isNaN(new Date(value).getTime())) return undefined;
        const key = m[1] === ">" ? "after" : "before";
        if (values[key]) return undefined;
        values[key] = value;
      }
      return values;
    },
    summary: (values) => {
      const format = (value: string) =>
        new Date(value).toLocaleString(undefined, {
          dateStyle: "medium",
          timeStyle: "short",
        });
      if (values.after && values.before) {
        return `Between ${format(values.after)} and ${format(values.before)}`;
      }
      if (values.after) return `After ${format(values.after)}`;
      if (values.before) return `Until ${format(values.before)}`;
      return "Any date";
    },
    issues: (values) => {
      const after = toTimestamp(values.after);
      const before = toTimestamp(values.before);
      if (!after && !before)
        return ["Choose a start date, an end date or both"];
      if (after && before && after >= before) {
        return ["The end date must come after the start date"];
      }
      return [];
    },
  },
);

const CIDR =
  /^(?:\d{1,3}(?:\.\d{1,3}){3}\/(?:3[0-2]|[12]?\d)|[0-9a-fA-F:.]*:[0-9a-fA-F:.]*\/(?:12[0-8]|1[01]\d|\d{1,2}))$/;

const ipRange = build(
  {
    id: "network.ipRange",
    category: "network",
    title: "Source IP address",
    description: "The request comes from an IP address or a CIDR range",
    keywords: ["ip", "cidr", "subnet", "office", "vpn", "network", "allowlist"],
  },
  {
    fields: [
      [
        "ranges",
        {
          kind: "strings",
          label: "IP ranges",
          placeholder: "10.0.0.0/8 or 203.0.113.7",
          transform: "cidr",
        },
      ],
    ],
    defaults: () => ({ ranges: [] }),
    toCEL: (values) => {
      const ranges: string[] = values.ranges;
      if (ranges.length === 1) {
        return `net.isIPInRange(ctx.request.ip, ${celString(ranges[0])})`;
      }
      return `${celStringList(ranges)}.exists(r, net.isIPInRange(ctx.request.ip, r))`;
    },
    parse: (expr) => {
      const single = exact(
        `net\\.isIPInRange\\(\\s*ctx\\.request\\.ip\\s*,\\s*(${STR})\\s*\\)`,
      ).exec(expr);
      if (single) {
        const value = parseStringLiteral(single[1]);
        return value ? { ranges: [value] } : undefined;
      }
      const list = exact(
        `(${LIST})\\.exists\\(\\s*(\\w+)\\s*,\\s*net\\.isIPInRange\\(\\s*ctx\\.request\\.ip\\s*,\\s*(\\w+)\\s*\\)\\s*\\)`,
      ).exec(expr);
      if (list && list[2] === list[3]) {
        const ranges = parseStringList(list[1]);
        return ranges && ranges.length > 0 ? { ranges } : undefined;
      }
      const ips = parseEqOrIn("ctx.request.ip", expr);
      if (ips && ips.length > 0) {
        return { ranges: ips.map((ip) => transformValue(ip, "cidr")) };
      }
      return undefined;
    },
    summary: (values) => `Source IP in ${orList(values.ranges)}`,
    issues: (values) => {
      const ranges: string[] = values.ranges ?? [];
      if (ranges.length === 0) return ["Add at least one IP range"];
      const invalid = ranges.filter((range) => !CIDR.test(range));
      return invalid.length > 0
        ? [`Fix the invalid range ${invalid.join(", ")}`]
        : [];
    },
  },
);

const CONTINENTS: Option[] = [
  { value: "AF", label: "Africa" },
  { value: "AN", label: "Antarctica" },
  { value: "AS", label: "Asia" },
  { value: "EU", label: "Europe" },
  { value: "NA", label: "North America" },
  { value: "OC", label: "Oceania" },
  { value: "SA", label: "South America" },
];

const countries = build(
  {
    id: "network.country",
    category: "network",
    title: "Country",
    description: "The country where the Session authenticated, from GeoIP",
    keywords: [
      "geo",
      "geoip",
      "location",
      "country",
      "region",
      "sanctions",
      "geofence",
    ],
  },
  {
    fields: [["codes", { kind: "countries", label: "Countries" }]],
    defaults: () => ({ codes: [] }),
    toCEL: (values) =>
      eqOrIn(
        "ctx.session.status.authentication.info.geoip.country.code",
        values.codes,
      ),
    parse: (expr) => {
      const codes = parseEqOrIn(
        "ctx.session.status.authentication.info.geoip.country.code",
        expr,
      );
      return codes && codes.length > 0 ? { codes } : undefined;
    },
    summary: (values) => `Country is ${orList(values.codes)}`,
  },
);

const group = build(
  {
    id: "identity.group",
    category: "identity",
    title: "Group membership",
    description: "The User belongs to one or more Groups",
    keywords: ["rbac", "role", "team", "group", "member"],
  },
  {
    fields: [
      ["groups", { kind: "refs", label: "Groups", resource: "Group" }],
      [
        "mode",
        {
          kind: "choice",
          label: "The User must be in",
          options: [
            { value: "any", label: "Any of them" },
            { value: "all", label: "All of them" },
          ],
          visible: (values) => (values.groups ?? []).length > 1,
        },
      ],
    ],
    defaults: () => ({ groups: [], mode: "any" }),
    toCEL: (values) => {
      const groups: string[] = values.groups;
      if (groups.length === 1) {
        return `${celString(groups[0])} in ctx.user.spec.groups`;
      }
      return `${celStringList(groups)}.${values.mode === "all" ? "all" : "exists"}(g, g in ctx.user.spec.groups)`;
    },
    parse: (expr) => {
      const single = exact(`(${STR})\\s+in\\s+ctx\\.user\\.spec\\.groups`).exec(
        expr,
      );
      if (single) {
        const value = parseStringLiteral(single[1]);
        return value !== undefined
          ? { groups: [value], mode: "any" }
          : undefined;
      }
      const list = exact(
        `(${LIST})\\.(exists|all)\\(\\s*(\\w+)\\s*,\\s*(\\w+)\\s+in\\s+ctx\\.user\\.spec\\.groups\\s*\\)`,
      ).exec(expr);
      if (list && list[3] === list[4]) {
        const groups = parseStringList(list[1]);
        if (!groups || groups.length === 0) return undefined;
        return { groups, mode: list[2] === "all" ? "all" : "any" };
      }
      return undefined;
    },
    summary: (values) =>
      values.groups?.length > 1 && values.mode === "all"
        ? `User is in all of ${orList(values.groups, "and")}`
        : `User is in ${orList(values.groups ?? [])}`,
    issues: (values) =>
      (values.groups ?? []).length === 0 ? ["Choose at least one Group"] : [],
  },
);

const API_PACKAGES: Option[] = [
  { value: "octelium.api.main.core.v1", label: "Core" },
  { value: "octelium.api.main.user.v1", label: "User" },
  { value: "octelium.api.main.auth.v1", label: "Auth" },
  { value: "octelium.api.main.enterprise.v1", label: "Enterprise" },
  { value: "octelium.api.main.access.v1", label: "Access" },
  { value: "octelium.api.main.cordium.v1", label: "Cordium" },
  { value: "octelium.api.main.visibility.v1", label: "Visibility" },
  { value: "octelium.api.main.visibility.core.v1", label: "Visibility · Core" },
  {
    value: "octelium.api.main.visibility.enterprise.v1",
    label: "Visibility · Enterprise",
  },
  {
    value: "octelium.api.main.visibility.access.v1",
    label: "Visibility · Access",
  },
  {
    value: "octelium.api.main.visibility.cordium.v1",
    label: "Visibility · Cordium",
  },
  { value: "octelium.api.main.visibility.llm.v1", label: "Visibility · LLM" },
  {
    value: "octelium.api.main.visibility.metrics.v1",
    label: "Visibility · Metrics",
  },
  { value: "octelium.api.main.clusterman.v1", label: "Cluster management" },
];

const API_SERVICES = [
  "MainService",
  "UserService",
  "ReviewerService",
  "ClusterService",
  "PolicyPortalService",
  "AgentService",
  "WorkspaceService",
  "ManagementService",
  "ResourceService",
  "AccessLogService",
  "AuthenticationLogService",
  "AuditLogService",
  "ComponentLogService",
  "MetricsService",
  "LLMService",
];

const API_NAMESPACE = `ctx.service.status.namespaceRef.name == "octelium-api"`;
const API_NAMESPACE_ALIAS = `ctx.namespace.metadata.name == "octelium-api"`;
const API_MODE = `ctx.service.spec.mode == "GRPC"`;
const API_READ_ONLY = `["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x))`;

const octeliumAPI = build(
  {
    id: "api.octelium",
    category: "api",
    title: "Octelium API",
    description: "Requests to the Cluster's own APIs, optionally read-only",
    keywords: [
      "api",
      "octelium",
      "admin",
      "read-only",
      "readonly",
      "grpc",
      "cli",
      "console",
      "octeliumctl",
    ],
  },
  {
    fields: [
      ["apis", { kind: "options", label: "APIs", options: API_PACKAGES }],
      [
        "services",
        {
          kind: "strings",
          label: "Services",
          placeholder: "MainService",
          suggestions: API_SERVICES,
        },
      ],
      [
        "methods",
        { kind: "strings", label: "Methods", placeholder: "GetUser" },
      ],
      [
        "readOnly",
        {
          kind: "switch",
          label: "Read-only methods",
          description: "Only methods starting with Get or List",
        },
      ],
    ],
    defaults: () => ({ apis: [], services: [], methods: [], readOnly: false }),
    toCEL: (values) => {
      const parts = [API_NAMESPACE, API_MODE];
      if (values.apis?.length)
        parts.push(eqOrIn("ctx.request.grpc.package", values.apis));
      if (values.services?.length)
        parts.push(eqOrIn("ctx.request.grpc.service", values.services));
      if (values.methods?.length)
        parts.push(eqOrIn("ctx.request.grpc.method", values.methods));
      if (values.readOnly) parts.push(API_READ_ONLY);
      return parts.join(" && ");
    },
    parse: (expr) => {
      const values: Values = {
        apis: [],
        services: [],
        methods: [],
        readOnly: false,
      };
      let namespace = false;
      let mode = false;
      for (const part of conjuncts(expr)) {
        if (
          !namespace &&
          (sameCEL(part, API_NAMESPACE) || sameCEL(part, API_NAMESPACE_ALIAS))
        ) {
          namespace = true;
          continue;
        }
        if (!mode && sameCEL(part, API_MODE)) {
          mode = true;
          continue;
        }
        if (!values.readOnly && sameCEL(part, API_READ_ONLY)) {
          values.readOnly = true;
          continue;
        }
        const apis = parseEqOrIn("ctx.request.grpc.package", part);
        if (apis && values.apis.length === 0) {
          values.apis = apis;
          continue;
        }
        const services = parseEqOrIn("ctx.request.grpc.service", part);
        if (services && values.services.length === 0) {
          values.services = services;
          continue;
        }
        const methods = parseEqOrIn("ctx.request.grpc.method", part);
        if (methods && values.methods.length === 0) {
          values.methods = methods;
          continue;
        }
        return undefined;
      }
      return namespace && mode ? values : undefined;
    },
    summary: (values) => {
      const apis = values.apis?.length
        ? `${orList(labelsOf(API_PACKAGES, values.apis))} API`
        : "Any Octelium API";
      const extra = [
        values.services?.length ? orList(values.services) : "",
        values.methods?.length ? `methods ${orList(values.methods)}` : "",
        values.readOnly ? "read-only" : "",
      ].filter(Boolean);
      return extra.length ? `${apis} · ${extra.join(" · ")}` : apis;
    },
    issues: () => [],
  },
);

const POSTURE_CHECKS: Option[] = [
  { value: "threatFree", label: "Threat-free" },
  { value: "compliant", label: "Compliant" },
  { value: "diskEncryption", label: "Disk encryption" },
  { value: "firewall", label: "Firewall" },
  { value: "screenLock", label: "Screen lock" },
  { value: "secureBoot", label: "Secure boot" },
  { value: "osUpToDate", label: "OS up to date" },
  { value: "osIntegrity", label: "OS integrity" },
  { value: "antivirus", label: "Antivirus" },
  { value: "tamperProtection", label: "Tamper protection" },
  { value: "agentHealthy", label: "Agent healthy" },
  { value: "agentUpToDate", label: "Agent up to date" },
  { value: "enrolled", label: "Enrolled" },
  { value: "notContained", label: "Not contained" },
  { value: "appControl", label: "App control" },
];

const posture = build(
  {
    id: "device.posture",
    category: "device",
    title: "Device posture",
    description: "Posture checks reported by the Device's manager pass",
    keywords: [
      "posture",
      "compliance",
      "edr",
      "mdm",
      "healthy",
      "zero trust",
      "encryption",
    ],
  },
  {
    fields: [
      ["checks", { kind: "options", label: "Checks", options: POSTURE_CHECKS }],
      [
        "grace",
        {
          kind: "switch",
          label: "Accept grace periods",
          description: "Also match checks that are in their grace period",
        },
      ],
    ],
    defaults: () => ({ checks: ["threatFree", "compliant"], grace: false }),
    toCEL: (values) =>
      (values.checks as string[])
        .map((check) =>
          values.grace
            ? `ctx.device.status.posture.${check} in ["PASS", "GRACE_PERIOD"]`
            : `ctx.device.status.posture.${check} == "PASS"`,
        )
        .join(" && "),
    parse: (expr) => {
      const checks: string[] = [];
      let grace: boolean | undefined;
      for (const part of conjuncts(expr)) {
        const m = exact(
          `ctx\\.device\\.status\\.posture\\.(\\w+)\\s*(==\\s*"PASS"|in\\s+\\[\\s*"PASS"\\s*,\\s*"GRACE_PERIOD"\\s*\\])`,
        ).exec(part);
        if (!m || !POSTURE_CHECKS.some((check) => check.value === m[1]))
          return undefined;
        const isGrace = m[2].startsWith("in");
        if (grace !== undefined && grace !== isGrace) return undefined;
        grace = isGrace;
        checks.push(m[1]);
      }
      return checks.length > 0 ? { checks, grace: grace ?? false } : undefined;
    },
    summary: (values) =>
      `Device passes ${orList(labelsOf(POSTURE_CHECKS, values.checks ?? []), "and").toLowerCase()}`,
  },
);

const HTTP_METHODS = [
  "GET",
  "HEAD",
  "OPTIONS",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "CONNECT",
  "TRACE",
];

const K8S_VERBS = [
  "get",
  "list",
  "watch",
  "create",
  "update",
  "patch",
  "delete",
  "deletecollection",
];

const MCP_METHODS = [
  "initialize",
  "tools/list",
  "tools/call",
  "prompts/list",
  "prompts/get",
  "resources/list",
  "resources/read",
  "resources/subscribe",
  "completion/complete",
  "logging/setLevel",
  "ping",
];

const llmToolName = build(
  {
    id: "llm.toolName",
    category: "llm",
    title: "LLM tool offered",
    description: "The request offers the model a tool with a matching name",
    keywords: ["llm", "tool", "function calling"],
  },
  {
    fields: [
      [
        "match",
        { kind: "match", label: "Tool name", placeholder: "web_search" },
      ],
    ],
    defaults: () => ({ match: { op: "is", values: [] } }),
    toCEL: (values) =>
      `ctx.request.llm.toolNames.exists(x, ${matchCEL("x", values.match)})`,
    parse: (expr) => {
      const m = exact(
        `ctx\\.request\\.llm\\.toolNames\\.exists\\(\\s*x\\s*,\\s*(.+)\\)`,
      ).exec(expr);
      if (!m) return undefined;
      const match = parseMatch("x", m[1]);
      return match ? { match } : undefined;
    },
    summary: (values) => `A tool name ${matchSummary(values.match)}`,
  },
);

export const BLOCKS: BlockDef[] = [
  group,
  refsBlock({
    id: "identity.user",
    category: "identity",
    title: "User",
    description: "The request is made by specific Users",
    keywords: ["user", "person", "identity", "who"],
    field: "ctx.user.metadata.name",
    resource: "User",
    subject: "User",
  }),
  optionsBlock({
    id: "identity.userType",
    category: "identity",
    title: "User type",
    description: "Human Users or workloads (services, CI jobs, containers)",
    keywords: ["human", "workload", "machine", "bot", "service account"],
    field: "ctx.user.spec.type",
    options: enumOptions(CoreP.User_Spec_Type),
    subject: "User type",
    label: "Types",
    initial: ["HUMAN"],
  }),
  matchBlock({
    id: "identity.email",
    category: "identity",
    title: "User email",
    description: "The User's email address, e.g. a company domain",
    keywords: ["email", "domain", "company"],
    field: "ctx.user.spec.email",
    subject: "Email",
    label: "Email",
    placeholder: "@example.com",
    initial: { op: "suffix", values: [] },
  }),
  keyedMatchBlock({
    id: "identity.userLabel",
    category: "identity",
    title: "User label",
    description: "A label set on the User, e.g. team or department",
    keywords: ["label", "attribute", "abac", "team", "department"],
    map: "ctx.user.metadata.labels",
    subject: "User label",
    keyLabel: "Label",
    keyPlaceholder: "team",
  }),

  refsBlock({
    id: "resource.service",
    category: "resource",
    title: "Service",
    description: "The request targets specific Services",
    keywords: ["service", "app", "upstream", "target", "resource"],
    field: "ctx.service.metadata.name",
    resource: "Service",
    subject: "Service",
  }),
  refsBlock({
    id: "resource.namespace",
    category: "resource",
    title: "Namespace",
    description: "The request targets any Service of specific Namespaces",
    keywords: ["namespace", "environment", "production", "staging", "project"],
    field: "ctx.namespace.metadata.name",
    resource: "Namespace",
    subject: "Namespace",
  }),
  optionsBlock({
    id: "resource.serviceMode",
    category: "resource",
    title: "Service mode",
    description: "The protocol of the Service, e.g. HTTP, SSH or Kubernetes",
    keywords: [
      "mode",
      "protocol",
      "http",
      "ssh",
      "tcp",
      "kubernetes",
      "postgres",
    ],
    field: "ctx.service.spec.mode",
    options: enumOptions(CoreP.Service_Spec_Mode),
    subject: "Service mode",
    label: "Modes",
  }),
  flagBlock({
    id: "resource.public",
    category: "resource",
    title: "Public Service",
    description: "The Service is publicly exposed to browsers without a client",
    keywords: ["public", "clientless", "browser"],
    expr: "ctx.service.spec.isPublic",
    summary: "Service is public",
  }),
  keyedMatchBlock({
    id: "resource.serviceLabel",
    category: "resource",
    title: "Service label",
    description: "A label set on the Service, e.g. its environment or owner",
    keywords: ["label", "attribute", "abac", "environment", "tier"],
    map: "ctx.service.metadata.labels",
    subject: "Service label",
    keyLabel: "Label",
    keyPlaceholder: "env",
  }),

  optionsBlock({
    id: "session.type",
    category: "session",
    title: "Session type",
    description:
      "Client Sessions (octelium CLI) or clientless browser and SDK access",
    keywords: ["client", "clientless", "cli", "browser"],
    field: "ctx.session.status.type",
    options: enumOptions(CoreP.Session_Status_Type),
    subject: "Session type",
    label: "Types",
  }),
  flagBlock({
    id: "session.browser",
    category: "session",
    title: "Browser Session",
    description: "The Session was created by signing in from a web browser",
    keywords: ["browser", "web", "portal"],
    expr: "ctx.session.status.isBrowser",
    summary: "Session is a browser Session",
  }),
  optionsBlock({
    id: "session.aal",
    category: "session",
    title: "Authentication assurance (AAL)",
    description:
      "NIST assurance level of the Session's authentication, e.g. MFA",
    keywords: ["mfa", "2fa", "aal", "assurance", "strong", "step-up"],
    field: "ctx.session.status.authentication.info.aal",
    options: enumOptions(CoreP.Session_Status_Authentication_Info_AAL),
    subject: "Assurance level",
    label: "Levels",
    initial: ["AAL2", "AAL3"],
  }),
  optionsBlock({
    id: "session.authType",
    category: "session",
    title: "Authentication method",
    description:
      "How the Session authenticated: identity provider, credential…",
    keywords: ["authentication", "sso", "idp", "credential", "token"],
    field: "ctx.session.status.authentication.info.type",
    options: enumOptions(CoreP.Session_Status_Authentication_Info_Type),
    subject: "Authentication method",
    label: "Methods",
  }),
  refsBlock({
    id: "session.identityProvider",
    category: "session",
    title: "Identity provider",
    description: "The Session authenticated with specific identity providers",
    keywords: ["sso", "idp", "oidc", "saml", "github", "okta", "google"],
    field:
      "ctx.session.status.authentication.info.identityProvider.identityProviderRef.name",
    resource: "IdentityProvider",
    subject: "Identity provider",
  }),
  refsBlock({
    id: "session.credential",
    category: "session",
    title: "Credential",
    description: "The Session authenticated with specific Credentials",
    keywords: ["credential", "token", "workload"],
    field:
      "ctx.session.status.authentication.info.credential.credentialRef.name",
    resource: "Credential",
    subject: "Credential",
  }),
  optionsBlock({
    id: "session.credentialType",
    category: "session",
    title: "Credential type",
    description:
      "Authentication tokens, OAuth2 client credentials or access tokens",
    keywords: ["credential", "token", "oauth2"],
    field: "ctx.session.status.authentication.info.credential.type",
    options: enumOptions(CoreP.Credential_Spec_Type),
    subject: "Credential type",
    label: "Types",
  }),
  flagBlock({
    id: "session.passkey",
    category: "session",
    title: "Passkey",
    description: "The Session authenticated with a FIDO passkey",
    keywords: ["passkey", "fido", "webauthn", "passwordless"],
    expr: "ctx.session.status.authentication.info.authenticator.info.fido.isPasskey",
    summary: "Authenticated with a passkey",
  }),
  flagBlock({
    id: "session.hardwareKey",
    category: "session",
    title: "Hardware security key",
    description: "The Session authenticated with a hardware FIDO authenticator",
    keywords: ["yubikey", "fido", "hardware", "security key"],
    expr: "ctx.session.status.authentication.info.authenticator.info.fido.isHardware",
    summary: "Authenticated with a hardware key",
  }),
  flagBlock({
    id: "session.userVerified",
    category: "session",
    title: "User verified",
    description: "The FIDO authenticator verified the User (PIN or biometrics)",
    keywords: ["fido", "biometric", "pin", "uv"],
    expr: "ctx.session.status.authentication.info.authenticator.info.fido.userVerified",
    summary: "FIDO user verification",
  }),
  flagBlock({
    id: "session.userPresent",
    category: "session",
    title: "User present",
    description: "The FIDO authenticator confirmed the User was present",
    keywords: ["fido", "presence", "up"],
    expr: "ctx.session.status.authentication.info.authenticator.info.fido.userPresent",
    summary: "FIDO user presence",
  }),
  flagBlock({
    id: "session.attestation",
    category: "session",
    title: "Verified attestation",
    description: "The FIDO authenticator's attestation was verified",
    keywords: ["fido", "attestation", "hardware"],
    expr: "ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified",
    summary: "FIDO attestation verified",
  }),
  matchBlock({
    id: "session.aaguid",
    category: "session",
    title: "Authenticator model (AAGUID)",
    description: "Specific FIDO authenticator models",
    keywords: ["aaguid", "fido", "yubikey", "model"],
    field:
      "ctx.session.status.authentication.info.authenticator.info.fido.aaguid",
    subject: "AAGUID",
    label: "AAGUID",
    placeholder: "cb69481e-8ff7-4039-93ec-0a2729a154a8",
  }),
  optionsBlock({
    id: "session.deviceProof",
    category: "session",
    title: "Device proof",
    description: "How the Session proved it runs on a bound Device",
    keywords: ["device", "proof", "hardware key", "attestation"],
    field: "ctx.session.status.deviceProof.type",
    options: enumOptions(CoreP.Session_Status_DeviceProof_Type),
    subject: "Device proof",
    label: "Proofs",
  }),
  refsBlock({
    id: "session.session",
    category: "session",
    title: "Session",
    description: "Specific Sessions",
    keywords: ["session"],
    field: "ctx.session.metadata.name",
    resource: "Session",
    subject: "Session",
  }),

  refsBlock({
    id: "device.device",
    category: "device",
    title: "Device",
    description: "Specific Devices",
    keywords: ["device", "laptop", "machine"],
    field: "ctx.device.metadata.name",
    resource: "Device",
    subject: "Device",
  }),
  optionsBlock({
    id: "device.os",
    category: "device",
    title: "Operating system",
    description: "The Device's operating system",
    keywords: ["os", "linux", "windows", "mac", "ios", "android", "platform"],
    field: "ctx.device.status.osType",
    options: enumOptions(CoreP.Device_Status_OSType),
    subject: "OS",
    label: "Systems",
  }),
  posture,
  optionsBlock({
    id: "device.risk",
    category: "device",
    title: "Device risk level",
    description: "The risk level reported by the Device's manager",
    keywords: ["risk", "edr", "threat", "posture"],
    field: "ctx.device.status.posture.riskLevel",
    options: enumOptions(CoreP.Device_Status_Posture_RiskLevel),
    subject: "Risk level",
    label: "Levels",
    initial: ["LOW"],
  }),
  optionsBlock({
    id: "device.assurance",
    category: "device",
    title: "Binding assurance",
    description: "How strongly the Device is bound to a managed device",
    keywords: ["binding", "managed", "assurance", "mdm", "hardware"],
    field: "ctx.device.status.binding.assurance",
    options: enumOptions(CoreP.Device_Status_Binding_Assurance),
    subject: "Binding assurance",
    label: "Assurance",
    initial: ["SIGNED", "HARDWARE"],
  }),
  flagBlock({
    id: "device.ownerMatched",
    category: "device",
    title: "Owner matched",
    description: "The managed device's owner is the requesting User",
    keywords: ["owner", "binding", "managed"],
    expr: "ctx.device.status.binding.isOwnerMatched",
    summary: "Device owner is the User",
  }),

  ipRange,
  flagBlock({
    id: "network.privateIP",
    category: "network",
    title: "Private IP address",
    description: "The request comes from a private (RFC 1918 or ULA) address",
    keywords: ["private", "internal", "lan", "ip"],
    expr: "net.isPrivateIP(ctx.request.ip)",
    summary: "Source IP is private",
  }),
  countries,
  optionsBlock({
    id: "network.continent",
    category: "network",
    title: "Continent",
    description: "The continent where the Session authenticated, from GeoIP",
    keywords: ["geo", "geoip", "continent", "region", "location"],
    field: "ctx.session.status.authentication.info.geoip.continent.code",
    options: CONTINENTS,
    subject: "Continent",
    label: "Continents",
  }),

  timeWindow,
  dayType,
  dateRange,

  optionsBlock({
    id: "http.method",
    category: "http",
    title: "HTTP method",
    description: "GET, POST, DELETE…",
    keywords: ["method", "verb", "read-only", "readonly", "get", "post"],
    field: "ctx.request.http.method",
    options: optionsOf(HTTP_METHODS),
    subject: "Method",
    label: "Methods",
  }),
  matchBlock({
    id: "http.path",
    category: "http",
    title: "HTTP path",
    description: "The path of the request, e.g. /admin",
    keywords: ["path", "url", "route", "endpoint", "prefix"],
    field: "ctx.request.http.path",
    subject: "Path",
    label: "Path",
    placeholder: "/admin",
    initial: { op: "prefix", values: [] },
  }),
  matchBlock({
    id: "http.host",
    category: "http",
    title: "HTTP host",
    description: "The Host header of the request",
    keywords: ["host", "domain", "hostname"],
    field: "ctx.request.http.host",
    subject: "Host",
    label: "Host",
  }),
  presenceBlock({
    id: "http.hasHeader",
    category: "http",
    title: "HTTP header present",
    description: "The request carries a header",
    keywords: ["header"],
    map: "ctx.request.http.headers",
    subject: "Header",
    keyLabel: "Header",
    keyPlaceholder: "x-api-key",
    keyTransform: "lower",
  }),
  keyedMatchBlock({
    id: "http.header",
    category: "http",
    title: "HTTP header value",
    description: "The value of a request header",
    keywords: ["header", "value"],
    map: "ctx.request.http.headers",
    subject: "Header",
    keyLabel: "Header",
    keyPlaceholder: "x-tenant",
    keyTransform: "lower",
  }),
  keyedMatchBlock({
    id: "http.query",
    category: "http",
    title: "Query parameter",
    description: "The value of a URL query parameter",
    keywords: ["query", "parameter", "param", "url"],
    map: "ctx.request.http.queryParams",
    subject: "Query parameter",
    keyLabel: "Parameter",
    keyPlaceholder: "tenant",
  }),
  matchBlock({
    id: "http.scheme",
    category: "http",
    title: "HTTP scheme",
    description: "http or https",
    keywords: ["scheme", "https", "tls"],
    field: "ctx.request.http.scheme",
    subject: "Scheme",
    label: "Scheme",
    suggestions: ["https", "http"],
    transform: "lower",
  }),
  numberBlock({
    id: "http.size",
    category: "http",
    title: "HTTP body size",
    description: "The size of the request body in bytes",
    keywords: ["size", "body", "upload", "bytes"],
    field: "ctx.request.http.size",
    numeric: "int",
    subject: "Body size",
    label: "Bytes",
  }),

  octeliumAPI,

  flagBlock({
    id: "protocol.ssh",
    category: "protocol",
    group: "SSH",
    title: "SSH request",
    description: "Any request to an SSH Service",
    keywords: ["ssh"],
    expr: "has(ctx.request.ssh)",
    summary: "Request is SSH",
  }),
  matchBlock({
    id: "protocol.sshUser",
    category: "protocol",
    group: "SSH",
    title: "SSH user",
    description: "The Linux user the SSH connection logs in as",
    keywords: ["ssh", "user", "root", "linux"],
    field: "ctx.request.ssh.connect.user",
    subject: "SSH user",
    label: "Users",
    placeholder: "ubuntu",
  }),
  flagBlock({
    id: "protocol.kubernetes",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes request",
    description: "Any request to a Kubernetes Service",
    keywords: ["kubernetes", "k8s", "kubectl"],
    expr: "has(ctx.request.kubernetes)",
    summary: "Request is Kubernetes",
  }),
  matchBlock({
    id: "protocol.k8sVerb",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes verb",
    description: "get, list, watch, create, delete…",
    keywords: ["kubernetes", "k8s", "verb", "read-only", "readonly"],
    field: "ctx.request.kubernetes.verb",
    subject: "Verb",
    label: "Verbs",
    suggestions: K8S_VERBS,
    initial: { op: "in", values: [] },
  }),
  matchBlock({
    id: "protocol.k8sNamespace",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes namespace",
    description: "The Kubernetes namespace of the request",
    keywords: ["kubernetes", "k8s", "namespace"],
    field: "ctx.request.kubernetes.namespace",
    subject: "Kubernetes namespace",
    label: "Namespaces",
    placeholder: "default",
  }),
  matchBlock({
    id: "protocol.k8sResource",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes resource",
    description: "pods, deployments, secrets…",
    keywords: ["kubernetes", "k8s", "resource", "pods", "secrets"],
    field: "ctx.request.kubernetes.resource",
    subject: "Kubernetes resource",
    label: "Resources",
    suggestions: [
      "pods",
      "deployments",
      "services",
      "secrets",
      "configmaps",
      "namespaces",
      "nodes",
    ],
  }),
  matchBlock({
    id: "protocol.k8sName",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes object name",
    description: "The name of the Kubernetes object",
    keywords: ["kubernetes", "k8s", "name"],
    field: "ctx.request.kubernetes.name",
    subject: "Kubernetes object",
    label: "Names",
  }),
  matchBlock({
    id: "protocol.k8sAPIGroup",
    category: "protocol",
    group: "Kubernetes",
    title: "Kubernetes API group",
    description: "apps, batch, rbac.authorization.k8s.io…",
    keywords: ["kubernetes", "k8s", "api group"],
    field: "ctx.request.kubernetes.apiGroup",
    subject: "API group",
    label: "Groups",
  }),
  flagBlock({
    id: "protocol.grpc",
    category: "protocol",
    group: "gRPC",
    title: "gRPC request",
    description: "Any gRPC request",
    keywords: ["grpc"],
    expr: "has(ctx.request.grpc)",
    summary: "Request is gRPC",
  }),
  matchBlock({
    id: "protocol.grpcService",
    category: "protocol",
    group: "gRPC",
    title: "gRPC service",
    description: "The full name of the gRPC service, e.g. pkg.v1.Service",
    keywords: ["grpc", "service"],
    field: "ctx.request.grpc.serviceFullName",
    subject: "gRPC service",
    label: "Services",
  }),
  matchBlock({
    id: "protocol.grpcMethod",
    category: "protocol",
    group: "gRPC",
    title: "gRPC method",
    description: "The name of the gRPC method",
    keywords: ["grpc", "method", "rpc"],
    field: "ctx.request.grpc.method",
    subject: "gRPC method",
    label: "Methods",
  }),
  matchBlock({
    id: "protocol.postgresUser",
    category: "protocol",
    group: "PostgreSQL",
    title: "PostgreSQL user",
    description: "The database role of the connection",
    keywords: ["postgres", "postgresql", "database", "db", "role", "user"],
    field: "ctx.request.postgres.connect.user",
    subject: "PostgreSQL user",
    label: "Users",
  }),
  matchBlock({
    id: "protocol.postgresDatabase",
    category: "protocol",
    group: "PostgreSQL",
    title: "PostgreSQL database",
    description: "The database of the connection",
    keywords: ["postgres", "postgresql", "database", "db"],
    field: "ctx.request.postgres.connect.database",
    subject: "PostgreSQL database",
    label: "Databases",
  }),
  matchBlock({
    id: "protocol.postgresQuery",
    category: "protocol",
    group: "PostgreSQL",
    title: "PostgreSQL query",
    description: "The text of a simple query, e.g. starting with SELECT",
    keywords: ["postgres", "postgresql", "sql", "query", "select", "read-only"],
    field: "ctx.request.postgres.query.query",
    subject: "Query",
    label: "Query",
    placeholder: "SELECT",
    initial: { op: "prefix", values: [] },
  }),
  matchBlock({
    id: "protocol.mysqlDatabase",
    category: "protocol",
    group: "MySQL",
    title: "MySQL database",
    description: "The database selected by the connection",
    keywords: ["mysql", "database", "db"],
    field: "ctx.request.mysql.initDB.database",
    subject: "MySQL database",
    label: "Databases",
  }),
  matchBlock({
    id: "protocol.mysqlQuery",
    category: "protocol",
    group: "MySQL",
    title: "MySQL query",
    description: "The text of a query, e.g. starting with SELECT",
    keywords: ["mysql", "sql", "query", "select"],
    field: "ctx.request.mysql.query.query",
    subject: "Query",
    label: "Query",
    placeholder: "SELECT",
    initial: { op: "prefix", values: [] },
  }),
  matchBlock({
    id: "protocol.dnsName",
    category: "protocol",
    group: "DNS",
    title: "DNS name",
    description: "The name being resolved",
    keywords: ["dns", "domain", "name"],
    field: "ctx.request.dns.name",
    subject: "DNS name",
    label: "Names",
    transform: "lower",
    initial: { op: "suffix", values: [] },
  }),
  matchBlock({
    id: "protocol.socks5Host",
    category: "protocol",
    group: "SOCKS5",
    title: "SOCKS5 host",
    description: "The destination host of the SOCKS5 connection",
    keywords: ["socks5", "proxy", "host", "egress", "destination"],
    field: "ctx.request.socks5.connect.host",
    subject: "SOCKS5 host",
    label: "Hosts",
    transform: "lower",
  }),
  numberBlock({
    id: "protocol.socks5Port",
    category: "protocol",
    group: "SOCKS5",
    title: "SOCKS5 port",
    description: "The destination port of the SOCKS5 connection",
    keywords: ["socks5", "proxy", "port"],
    field: "ctx.request.socks5.connect.port",
    numeric: "uint",
    subject: "SOCKS5 port",
    label: "Port",
    initial: { op: "==", value: 443 },
  }),

  matchBlock({
    id: "mcp.method",
    category: "mcp",
    title: "MCP method",
    description: "tools/call, resources/read, prompts/get…",
    keywords: ["mcp", "method", "json-rpc"],
    field: "ctx.request.mcp.method",
    subject: "MCP method",
    label: "Methods",
    suggestions: MCP_METHODS,
  }),
  matchBlock({
    id: "mcp.tool",
    category: "mcp",
    title: "MCP tool call",
    description: "Calls to specific MCP tools",
    keywords: ["mcp", "tool", "agent", "ai"],
    field: "ctx.request.mcp.name",
    guard: `ctx.request.mcp.method == "tools/call"`,
    subject: "MCP tool",
    label: "Tools",
    placeholder: "delete_repository",
  }),
  matchBlock({
    id: "mcp.prompt",
    category: "mcp",
    title: "MCP prompt",
    description: "Requests for specific MCP prompts",
    keywords: ["mcp", "prompt"],
    field: "ctx.request.mcp.name",
    guard: `ctx.request.mcp.method == "prompts/get"`,
    subject: "MCP prompt",
    label: "Prompts",
  }),
  matchBlock({
    id: "mcp.resource",
    category: "mcp",
    title: "MCP resource",
    description: "Reads of specific MCP resource URIs",
    keywords: ["mcp", "resource", "uri"],
    field: "ctx.request.mcp.name",
    guard: `ctx.request.mcp.method == "resources/read"`,
    subject: "MCP resource",
    label: "URIs",
    placeholder: "file:///",
  }),
  flagBlock({
    id: "mcp.notification",
    category: "mcp",
    title: "MCP notification",
    description: "JSON-RPC notifications, which expect no response",
    keywords: ["mcp", "notification"],
    expr: "ctx.request.mcp.isNotification",
    summary: "Request is an MCP notification",
  }),

  matchBlock({
    id: "llm.model",
    category: "llm",
    title: "LLM model",
    description: "The model requested, e.g. gpt-4o or claude-sonnet-4",
    keywords: ["llm", "model", "ai", "gpt", "claude", "gemini"],
    field: "ctx.request.llm.model",
    subject: "Model",
    label: "Models",
  }),
  optionsBlock({
    id: "llm.operation",
    category: "llm",
    title: "LLM operation",
    description: "Generation, embeddings, moderation…",
    keywords: ["llm", "operation", "embedding"],
    field: "ctx.request.llm.operation",
    options: enumOptions(CoreP.Service_Spec_Config_LLM_Operation),
    subject: "Operation",
    label: "Operations",
  }),
  optionsBlock({
    id: "llm.protocol",
    category: "llm",
    title: "LLM API",
    description: "The provider API of the request",
    keywords: ["llm", "openai", "anthropic", "gemini", "bedrock"],
    field: "ctx.request.llm.protocol",
    options: enumOptions(CoreP.Service_Spec_Config_LLM_Protocol),
    subject: "API",
    label: "APIs",
  }),
  numberBlock({
    id: "llm.maxOutputTokens",
    category: "llm",
    title: "Max output tokens",
    description: "The output token limit set by the request",
    keywords: ["llm", "tokens", "cost", "budget"],
    field: "ctx.request.llm.maxOutputTokens",
    numeric: "uint",
    subject: "Max output tokens",
    label: "Tokens",
  }),
  numberBlock({
    id: "llm.inputTokens",
    category: "llm",
    title: "Estimated input tokens",
    description: "The estimated size of the prompt",
    keywords: ["llm", "tokens", "prompt", "cost", "budget"],
    field: "ctx.request.llm.estimatedInputTokens",
    numeric: "uint",
    subject: "Input tokens",
    label: "Tokens",
  }),
  flagBlock({
    id: "llm.stream",
    category: "llm",
    title: "Streaming",
    description: "The request asks for a streamed response",
    keywords: ["llm", "stream", "sse"],
    expr: "ctx.request.llm.stream",
    summary: "Response is streamed",
  }),
  flagBlock({
    id: "llm.hasTools",
    category: "llm",
    title: "Tools offered",
    description: "The request offers tools to the model",
    keywords: ["llm", "tools", "function calling", "agent"],
    expr: "ctx.request.llm.hasTools",
    summary: "Request offers tools",
  }),
  llmToolName,
  flagBlock({
    id: "llm.image",
    category: "llm",
    title: "Image input",
    description: "The prompt includes images",
    keywords: ["llm", "image", "vision", "multimodal"],
    expr: "ctx.request.llm.hasImageInput",
    summary: "Prompt includes images",
  }),
  flagBlock({
    id: "llm.audio",
    category: "llm",
    title: "Audio input",
    description: "The prompt includes audio",
    keywords: ["llm", "audio", "multimodal"],
    expr: "ctx.request.llm.hasAudioInput",
    summary: "Prompt includes audio",
  }),
];

const BY_ID = new Map(BLOCKS.map((block) => [block.id, block]));

export const getBlock = (id: string) => BY_ID.get(id);

export const recognize = (
  expr: string,
): { block: BlockDef; values: Values } | undefined => {
  const source = expr.trim();
  if (!source) return undefined;
  for (const block of BLOCKS) {
    let values: Values | undefined;
    try {
      values = block.parse(source);
    } catch {
      values = undefined;
    }
    if (values) return { block, values };
  }
  return undefined;
};

export const NUMBER_OPTIONS: Option[] = NUMBER_OPS.map((value) => ({
  value,
  label: NUMBER_LABELS[value],
}));
