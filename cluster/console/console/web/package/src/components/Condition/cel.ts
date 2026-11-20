export const celString = (value: string) => JSON.stringify(value);

export const celStringList = (values: string[]) =>
  `[${values.map(celString).join(", ")}]`;

const OPENING = "([{";
const CLOSING = ")]}";

const literalEnd = (expr: string, start: number): number => {
  let index = start;
  while (index < expr.length && /[rRbB]/.test(expr[index])) index++;
  const quote = expr[index];
  const raw = /[rR]/.test(expr.slice(start, index));
  const triple = expr.startsWith(quote.repeat(3), index);
  const delimiter = triple ? quote.repeat(3) : quote;
  index += delimiter.length;
  while (index < expr.length) {
    if (!raw && expr[index] === "\\") {
      index += 2;
      continue;
    }
    if (expr.startsWith(delimiter, index)) return index + delimiter.length;
    if (!triple && expr[index] === "\n") return -1;
    index++;
  }
  return -1;
};

const literalStart = (expr: string, index: number) => {
  const char = expr[index];
  if (char === '"' || char === "'") return true;
  if (!/[rRbB]/.test(char) || /[\w.]/.test(expr[index - 1] ?? "")) {
    return false;
  }
  let next = index;
  while (next < expr.length && next - index < 2 && /[rRbB]/.test(expr[next])) {
    next++;
  }
  return expr[next] === '"' || expr[next] === "'";
};

export const scanTopLevel = (
  expr: string,
  visit: (index: number, depth: number) => boolean | void,
): boolean => {
  const stack: string[] = [];
  let index = 0;
  while (index < expr.length) {
    const char = expr[index];
    if (literalStart(expr, index)) {
      const end = literalEnd(expr, index);
      if (end < 0) return false;
      index = end;
      continue;
    }
    if (char === "/" && expr[index + 1] === "/") {
      const end = expr.indexOf("\n", index);
      index = end < 0 ? expr.length : end + 1;
      continue;
    }
    if (OPENING.includes(char)) {
      if (visit(index, stack.length) === false) return true;
      stack.push(char);
    } else if (CLOSING.includes(char)) {
      const opening = stack.pop();
      if (opening !== OPENING[CLOSING.indexOf(char)]) return false;
      if (visit(index, stack.length) === false) return true;
    } else if (visit(index, stack.length) === false) {
      return true;
    }
    index++;
  }
  return stack.length === 0;
};

export const looksComplete = (expr: string) => {
  const trimmed = expr.trim();
  if (!trimmed) return false;
  if (!scanTopLevel(trimmed, () => {})) return false;
  return !/(&&|\|\||[=!<>+\-*/%?:,.]|\bin)\s*$/.test(
    trimmed.replace(/\/\/[^\n]*$/, "").trimEnd(),
  );
};

const topLevelIndexes = (expr: string, token: string) => {
  const ret: number[] = [];
  const ok = scanTopLevel(expr, (index, depth) => {
    if (depth === 0 && expr.startsWith(token, index)) ret.push(index);
  });
  return ok ? ret : [];
};

const hasTopLevel = (expr: string, tokens: string[]) => {
  let found = false;
  scanTopLevel(expr, (index, depth) => {
    if (depth === 0 && tokens.some((token) => expr.startsWith(token, index))) {
      found = true;
      return false;
    }
  });
  return found;
};

export const hasTopLevelTernary = (expr: string) => hasTopLevel(expr, ["?"]);

export const unwrapParens = (expr: string): string => {
  let current = expr.trim();
  for (;;) {
    if (!current.startsWith("(") || !current.endsWith(")")) return current;
    let closesAtEnd = true;
    const ok = scanTopLevel(current, (index, depth) => {
      if (depth === 0 && index > 0 && index < current.length - 1) {
        closesAtEnd = false;
        return false;
      }
    });
    if (!ok || !closesAtEnd) return current;
    current = current.slice(1, -1).trim();
  }
};

export const splitTopLevel = (
  expr: string,
  operator: "&&" | "||",
): string[] | undefined => {
  const source = unwrapParens(expr);
  if (hasTopLevelTernary(source)) return undefined;
  if (operator === "&&" && topLevelIndexes(source, "||").length > 0) {
    return undefined;
  }
  const indexes = topLevelIndexes(source, operator);
  if (indexes.length === 0) return undefined;
  const parts: string[] = [];
  let start = 0;
  for (const index of indexes) {
    parts.push(source.slice(start, index).trim());
    start = index + operator.length;
  }
  parts.push(source.slice(start).trim());
  return parts.every((part) => part.length > 0) ? parts : undefined;
};

const BINARY = ["==", "!=", "<", ">", "&&", "||", "?", "+", "-", "*", "/", "%"];

export const isSimpleTerm = (expr: string) => {
  const source = expr.trim();
  if (!source || hasTopLevel(source, BINARY)) return false;
  let keyword = false;
  scanTopLevel(source, (index, depth) => {
    if (depth === 0 && /\s/.test(source[index])) {
      keyword = true;
      return false;
    }
  });
  return !keyword;
};

export const negatedInner = (expr: string): string | undefined => {
  const source = expr.trim();
  if (!source.startsWith("!") || source.startsWith("!=")) return undefined;
  const rest = source.slice(1).trim();
  if (!rest) return undefined;
  const inner = unwrapParens(rest);
  if (inner !== rest) return inner;
  return isSimpleTerm(rest) ? rest : undefined;
};

export const wrapFor = (operator: "&&" | "||", expr: string) => {
  const source = expr.trim();
  if (unwrapParens(source) !== source) return source;
  const other = operator === "&&" ? "||" : "&&";
  return hasTopLevel(source, [other, "?"]) ? `(${source})` : source;
};

export const negate = (expr: string) =>
  isSimpleTerm(expr) ? `!${expr.trim()}` : `!(${unwrapParens(expr)})`;

export const parseStringLiteral = (text: string): string | undefined => {
  const source = text.trim();
  if (/^"(?:[^"\\\n]|\\.)*"$/.test(source)) {
    try {
      return JSON.parse(source);
    } catch {
      return undefined;
    }
  }
  if (/^'(?:[^'\\\n]|\\.)*'$/.test(source)) {
    const body = source
      .slice(1, -1)
      .replace(/\\'/g, "'")
      .replace(/(^|[^\\])"/g, '$1\\"');
    try {
      return JSON.parse(`"${body}"`);
    } catch {
      return undefined;
    }
  }
  return undefined;
};

export const splitList = (text: string): string[] | undefined => {
  const source = text.trim();
  if (!source.startsWith("[") || !source.endsWith("]")) return undefined;
  const body = source.slice(1, -1);
  if (!body.trim()) return [];
  const items: string[] = [];
  let start = 0;
  const ok = scanTopLevel(body, (index, depth) => {
    if (depth === 0 && body[index] === ",") {
      items.push(body.slice(start, index).trim());
      start = index + 1;
    }
  });
  if (!ok) return undefined;
  const last = body.slice(start).trim();
  if (last) items.push(last);
  return items.every((item) => item.length > 0) ? items : undefined;
};

export const parseStringList = (text: string): string[] | undefined => {
  const items = splitList(text);
  if (!items) return undefined;
  const ret: string[] = [];
  for (const item of items) {
    const value = parseStringLiteral(item);
    if (value === undefined) return undefined;
    ret.push(value);
  }
  return ret;
};

export const parseIntList = (text: string): number[] | undefined => {
  const items = splitList(text);
  if (!items) return undefined;
  const ret: number[] = [];
  for (const item of items) {
    if (!/^-?\d+$/.test(item)) return undefined;
    ret.push(Number(item));
  }
  return ret;
};

export const canonical = (expr: string) => {
  let ret = "";
  let index = 0;
  let pendingSpace = false;
  while (index < expr.length) {
    if (literalStart(expr, index)) {
      const end = literalEnd(expr, index);
      if (end < 0) return expr.trim();
      pendingSpace = false;
      ret += expr.slice(index, end);
      index = end;
      continue;
    }
    const char = expr[index];
    if (/\s/.test(char)) {
      pendingSpace = true;
      index++;
      continue;
    }
    if (pendingSpace && /\w$/.test(ret) && /\w/.test(char)) ret += " ";
    pendingSpace = false;
    ret += char;
    index++;
  }
  return ret;
};

export const sameCEL = (a: string, b: string) =>
  canonical(unwrapParens(a)) === canonical(unwrapParens(b));

export const conjuncts = (expr: string): string[] => {
  const parts = splitTopLevel(expr, "&&");
  if (!parts) return [unwrapParens(expr)];
  return parts.flatMap(conjuncts);
};

export const escapeRegExp = (value: string) =>
  value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

export const STRING_PATTERN = String.raw`(?:"(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*')`;

export const LIST_PATTERN = String.raw`\[\s*(?:${STRING_PATTERN}\s*(?:,\s*${STRING_PATTERN}\s*)*,?\s*)?\]`;

export const exact = (pattern: string) => new RegExp(`^\\s*${pattern}\\s*$`);
