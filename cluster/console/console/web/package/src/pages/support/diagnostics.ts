export const REDACTION_VERSION = 2;

export const DIAGNOSTIC_LEVELS = ["WARN", "ERROR", "PANIC", "FATAL"] as const;

export const DEFAULT_DIAGNOSTIC_LEVELS = ["ERROR", "PANIC", "FATAL"];

export interface DiagnosticEntry {
  time?: string;
  level: string;
  component?: { namespace?: string; type?: string; uid?: string };
  function?: string;
  file?: string;
  line?: number;
  message: string;
  fields?: Record<string, unknown>;
}

export interface RedactionOptions {
  keepEmails: boolean;
}

export type RedactionRule =
  | "field"
  | "privateKey"
  | "jwt"
  | "header"
  | "authorization"
  | "urlCredentials"
  | "urlQuery"
  | "assignment"
  | "email";

export type RedactionStats = Record<RedactionRule, number>;

export const REDACTION_LABELS: Record<RedactionRule, string> = {
  field: "Secret fields",
  privateKey: "Private keys",
  jwt: "JWTs",
  header: "Authorization and cookie headers",
  authorization: "Authorization values",
  urlCredentials: "Credentials in URLs",
  urlQuery: "URL query strings",
  assignment: "Secret assignments",
  email: "Email addresses",
};

export interface DiagnosticsSelection {
  from: string;
  to: string;
  levels: string[];
  component?: { namespace?: string; type?: string; uid?: string };
  maxEntries: number;
  keepEmails: boolean;
}

export interface DiagnosticsManifest {
  type: "manifest";
  version: 1;
  generatedAt: string;
  clusterDomain?: string;
  window: { from: string; to: string };
  levels: string[];
  component?: { namespace?: string; type?: string; uid?: string };
  requested: number;
  matched: number;
  included: number;
  truncated: boolean;
  redaction: {
    version: number;
    keepEmails: boolean;
    counts: RedactionStats;
  };
}

export interface DiagnosticsBundle {
  name: string;
  manifest: DiagnosticsManifest;
  entries: DiagnosticEntry[];
  text: string;
  data: Uint8Array<ArrayBuffer>;
  compressed: boolean;
}

const SECRET_KEY_PARTS = [
  "password",
  "passwd",
  "passphrase",
  "secret",
  "authorization",
  "cookie",
  "credential",
  "jwt",
  "apikey",
  "privatekey",
  "accesskey",
  "signature",
  "bearer",
];

const SECRET_KEYS = new Set(["key", "token", "auth", "pwd"]);

const PATTERNS: {
  rule: Exclude<RedactionRule, "field" | "assignment">;
  regex: RegExp;
  replace: string;
}[] = [
  {
    rule: "privateKey",
    regex:
      /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?(-----END [A-Z0-9 ]*PRIVATE KEY-----|$)/g,
    replace: "[REDACTED PRIVATE KEY]",
  },
  {
    rule: "jwt",
    regex: /\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]*/g,
    replace: "[REDACTED JWT]",
  },
  {
    rule: "header",
    regex:
      /\b((?:proxy-)?authorization|(?:set-)?cookie)([ \t]*:[ \t]*)[^\r\n]+/gi,
    replace: "$1$2[REDACTED]",
  },
  {
    rule: "authorization",
    regex: /\b(Bearer|Basic|Token|Digest|Negotiate)\s+[A-Za-z0-9._~+/=-]{8,}/gi,
    replace: "$1 [REDACTED]",
  },
  {
    rule: "urlCredentials",
    regex: /(\b[a-z][a-z0-9+.-]*:\/\/)[^\s/@:"']+:[^\s/@"']+@/gi,
    replace: "$1[REDACTED]@",
  },
  {
    rule: "urlQuery",
    regex: /(\b[a-z][a-z0-9+.-]*:\/\/[^\s?#"'<>]+)\?[^\s#"'<>]+/gi,
    replace: "$1?[REDACTED]",
  },
  {
    rule: "email",
    regex:
      /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}\b/g,
    replace: "[REDACTED EMAIL]",
  },
];

const ASSIGNMENT =
  /(\\?["']?)([A-Za-z_][A-Za-z0-9_.-]{0,63})\1(\s*[=:]\s*)(\\"(?:(?!\\").)*\\"|"(?:[^"\\]|\\.)*"|'[^']*'|[^\s,;&)}\]"'{[]+)/g;

export const newRedactionStats = (): RedactionStats => ({
  field: 0,
  privateKey: 0,
  jwt: 0,
  header: 0,
  authorization: 0,
  urlCredentials: 0,
  urlQuery: 0,
  assignment: 0,
  email: 0,
});

export const countRedactions = (stats: RedactionStats) =>
  Object.values(stats).reduce((sum, value) => sum + value, 0);

export const isSecretKey = (key: string) => {
  const normalized = key.toLowerCase().replace(/[-_.\s]/g, "");
  return (
    SECRET_KEYS.has(normalized) ||
    normalized.endsWith("token") ||
    SECRET_KEY_PARTS.some((part) => normalized.includes(part))
  );
};

export const redactText = (
  text: string,
  options: RedactionOptions,
  stats: RedactionStats,
) => {
  let ret = text;
  for (const pattern of PATTERNS) {
    if (pattern.rule === "email") {
      ret = redactAssignments(ret, stats);
      if (options.keepEmails) continue;
    }
    ret = ret.replace(pattern.regex, (...args) => {
      stats[pattern.rule] += 1;
      return pattern.replace.replace(/\$(\d)/g, (_, group: string) =>
        String(args[Number(group)] ?? ""),
      );
    });
  }
  return ret;
};

const redactAssignments = (text: string, stats: RedactionStats) => {
  const regex = new RegExp(ASSIGNMENT);
  let ret = "";
  let last = 0;
  for (let match = regex.exec(text); match; match = regex.exec(text)) {
    const [all, quote, key, separator, value] = match;
    if (!isSecretKey(key)) {
      regex.lastIndex = match.index + quote.length + key.length;
      continue;
    }
    const wrapper = /^(\\"|"|')/.exec(value)?.[1] ?? "";
    const inner = wrapper
      ? value.slice(wrapper.length, value.length - wrapper.length)
      : value;
    if (!inner || inner.includes("[REDACTED")) continue;
    stats.assignment += 1;
    ret += `${text.slice(last, match.index)}${quote}${key}${quote}${separator}${wrapper}[REDACTED]${wrapper}`;
    last = match.index + all.length;
  }
  return ret + text.slice(last);
};

export const redactValue = (
  value: unknown,
  options: RedactionOptions,
  stats: RedactionStats,
): unknown => {
  if (typeof value === "string") return redactText(value, options, stats);
  if (Array.isArray(value)) {
    return value.map((item) => redactValue(item, options, stats));
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([key, item]) => {
        if (isSecretKey(key) && item !== null && item !== "") {
          stats.field += 1;
          return [key, "[REDACTED]"];
        }
        return [key, redactValue(item, options, stats)];
      }),
    );
  }
  return value;
};

export const redactEntry = (
  entry: DiagnosticEntry,
  options: RedactionOptions,
  stats: RedactionStats,
): DiagnosticEntry => ({
  ...entry,
  message: redactText(entry.message, options, stats),
  ...(entry.fields
    ? {
        fields: redactValue(entry.fields, options, stats) as Record<
          string,
          unknown
        >,
      }
    : {}),
});

const compactTime = (value: string) =>
  value
    .replace(/\.\d+/, "")
    .replace(/[-:]/g, "")
    .replace(/(T\d{4})\d{2}Z$/, "$1Z");

export const getDiagnosticsName = (
  selection: Pick<DiagnosticsSelection, "from" | "to">,
  compressed: boolean,
) =>
  `component-logs-${compactTime(selection.from)}-${compactTime(selection.to)}.ndjson${compressed ? ".gz" : ""}`;

export const gzip = async (
  text: string,
): Promise<Uint8Array<ArrayBuffer> | null> => {
  if (typeof CompressionStream === "undefined") return null;
  const stream = new Blob([text])
    .stream()
    .pipeThrough(new CompressionStream("gzip"));
  return new Uint8Array(await new Response(stream).arrayBuffer());
};

const byTime = (a: DiagnosticEntry, b: DiagnosticEntry) =>
  (a.time ?? "").localeCompare(b.time ?? "");

export const buildDiagnostics = async (input: {
  entries: DiagnosticEntry[];
  selection: DiagnosticsSelection;
  matched: number;
  maxBytes: number;
  clusterDomain?: string;
  now?: Date;
}): Promise<DiagnosticsBundle> => {
  const options = { keepEmails: input.selection.keepEmails };
  const sorted = [...input.entries].sort(byTime).reverse();
  let count = Math.min(sorted.length, input.selection.maxEntries);

  for (let attempt = 0; ; attempt++) {
    const stats = newRedactionStats();
    const entries = sorted
      .slice(0, count)
      .map((entry) => redactEntry(entry, options, stats))
      .sort(byTime);
    const manifest: DiagnosticsManifest = {
      type: "manifest",
      version: 1,
      generatedAt: (input.now ?? new Date()).toISOString(),
      ...(input.clusterDomain ? { clusterDomain: input.clusterDomain } : {}),
      window: { from: input.selection.from, to: input.selection.to },
      levels: input.selection.levels,
      ...(input.selection.component &&
      Object.values(input.selection.component).some(Boolean)
        ? { component: input.selection.component }
        : {}),
      requested: input.selection.maxEntries,
      matched: Math.max(input.matched, input.entries.length),
      included: entries.length,
      truncated: entries.length < Math.max(input.matched, input.entries.length),
      redaction: {
        version: REDACTION_VERSION,
        keepEmails: input.selection.keepEmails,
        counts: stats,
      },
    };
    const text =
      [
        JSON.stringify(manifest),
        ...entries.map((entry) => JSON.stringify({ type: "log", ...entry })),
      ].join("\n") + "\n";
    const compressed = await gzip(text);
    const data = compressed ?? new TextEncoder().encode(text);

    if (data.byteLength <= input.maxBytes || count === 0 || attempt >= 4) {
      return {
        name: getDiagnosticsName(input.selection, !!compressed),
        manifest,
        entries,
        text,
        data,
        compressed: !!compressed,
      };
    }

    count = Math.floor((count * input.maxBytes * 0.9) / data.byteLength);
  }
};

export const getWindow = (
  preset: string,
  anchor?: string,
  now = new Date(),
) => {
  const minutes: Record<string, number> = {
    "15m": 15,
    "1h": 60,
    "6h": 360,
    "24h": 1440,
  };
  if (preset === "around" && anchor) {
    const time = new Date(anchor).getTime();
    if (Number.isFinite(time)) {
      return {
        from: new Date(time - 15 * 60_000).toISOString(),
        to: new Date(Math.min(time + 15 * 60_000, now.getTime())).toISOString(),
      };
    }
  }
  return {
    from: new Date(
      now.getTime() - (minutes[preset] ?? 60) * 60_000,
    ).toISOString(),
    to: now.toISOString(),
  };
};
