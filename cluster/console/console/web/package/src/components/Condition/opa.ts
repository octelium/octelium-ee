import {
  HighlightStyle,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";

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
});

export const regoHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: t.keyword, color: "var(--color-violet-600)", fontWeight: "bold" },
    { tag: t.variableName, color: "var(--color-slate-800)" },
    { tag: t.atom, color: "var(--color-sky-700)" },
    { tag: t.number, color: "var(--color-sky-600)" },
    { tag: t.string, color: "var(--color-emerald-700)" },
    { tag: t.comment, color: "var(--color-slate-500)", fontStyle: "italic" },
    { tag: t.operator, color: "var(--color-slate-600)", fontWeight: "bold" },
    { tag: t.punctuation, color: "var(--color-slate-600)" },
  ]),
);

export const regoTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "var(--color-white) !important",
      color: "var(--color-slate-800)",
      border: "1px solid var(--color-slate-200)",
      borderRadius: "8px",
      boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
    },
    "&.cm-editor": {
      backgroundColor: "var(--color-white) !important",
    },
    ".cm-scroller": {
      backgroundColor: "var(--color-white) !important",
    },
    ".cm-content": {
      backgroundColor: "var(--color-white) !important",

      fontSize: "13px",
      caretColor: "var(--color-slate-900)",
      padding: "8px 0",
    },
    ".cm-line": {
      backgroundColor: "var(--color-white) !important",
      padding: "0 12px",
      color: "var(--color-slate-800)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--color-slate-50) !important",
      color: "var(--color-slate-400)",
      border: "none",
      borderRight: "1px solid var(--color-slate-200)",
    },
    ".cm-gutter": {
      backgroundColor: "var(--color-slate-50) !important",
    },
    ".cm-gutterElement": {
      backgroundColor: "var(--color-slate-50) !important",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "var(--color-slate-200) !important",
      color: "var(--color-slate-600)",
    },
    ".cm-activeLine": {
      backgroundColor: "var(--color-slate-50) !important",
    },
    ".cm-selectionBackground, ::selection": {
      backgroundColor: "var(--color-blue-200) !important",
    },
    ".cm-cursor": {
      borderLeftColor: "var(--color-slate-900)",
    },
    ".cm-tooltip-autocomplete": {
      backgroundColor: "var(--color-white) !important",
      border: "1px solid var(--color-slate-200)",
      borderRadius: "6px",
      boxShadow: "0 8px 24px rgba(15,23,42,0.14)",
    },
    ".cm-tooltip-autocomplete ul li": {
      fontSize: "12px",
      padding: "3px 10px",
      color: "var(--color-slate-700)",
    },
    ".cm-tooltip-autocomplete ul li[aria-selected]": {
      backgroundColor: "var(--color-slate-900) !important",
      color: "var(--color-white)",
    },
  },
  { dark: false },
);
