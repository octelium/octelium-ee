import { StreamLanguage } from "@codemirror/language";

const keywords = new Set([
  "package",
  "import",
  "default",
  "true",
  "false",
  "if",
  "else",
]);

const builtins = new Set([
  "input",
  "data",
  "count",
  "contains",
  "startswith",
  "endswith",
]);

export const regoLanguage = StreamLanguage.define({
  token(stream) {
    if (stream.eatSpace()) return null;

    if (stream.match(/^#.*/)) return "comment";

    if (stream.match(/"(?:[^"\\]|\\.)*"/)) return "string";

    if (stream.match(/^-?\d+(\.\d+)?/)) return "number";

    if (stream.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current();
      if (keywords.has(word)) return "keyword";
      if (builtins.has(word)) return "atom";
      return "variableName";
    }

    if (stream.match(/:=|==|!=|<=|>=|=/)) return "operator";

    if (stream.match(/[{}()[\].,:]/)) return "punctuation";

    stream.next();
    return null;
  },

  indent(state, textAfter) {
    const s = state as any;

    const base = s.indentation ?? 0;

    if (/^\s*}/.test(textAfter)) return base - 2;

    if (s.current && typeof s.current === "function") {
      const line = s.current().trimEnd();
      if (line.endsWith("{")) return base + 2;
    }

    return base;
  },
});

import { EditorView } from "@codemirror/view";

export const regoTheme = EditorView.theme(
  {
    ".cm-keyword": { color: "#ff7b72", fontWeight: "bold" },
    ".cm-variableName": { color: "#d2a8ff" },
    ".cm-atom": { color: "#79c0ff" },
    ".cm-number": { color: "#ffa657" },
    ".cm-string": { color: "#a5d6ff" },
    ".cm-comment": { color: "#8b949e", fontStyle: "italic" },
  },
  { dark: true },
);
