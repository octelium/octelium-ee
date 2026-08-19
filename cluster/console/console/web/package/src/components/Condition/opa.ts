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
    { tag: t.keyword, color: "#7c3aed", fontWeight: "bold" },
    { tag: t.variableName, color: "#1e293b" },
    { tag: t.atom, color: "#0369a1" },
    { tag: t.number, color: "#0284c7" },
    { tag: t.string, color: "#047857" },
    { tag: t.comment, color: "#64748b", fontStyle: "italic" },
    { tag: t.operator, color: "#475569", fontWeight: "bold" },
    { tag: t.punctuation, color: "#475569" },
  ]),
);

export const regoTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "#ffffff !important",
      color: "#1e293b",
      border: "1px solid #e2e8f0",
      borderRadius: "8px",
      boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
    },
    "&.cm-editor": {
      backgroundColor: "#ffffff !important",
    },
    ".cm-scroller": {
      backgroundColor: "#ffffff !important",
    },
    ".cm-content": {
      backgroundColor: "#ffffff !important",

      fontSize: "13px",
      caretColor: "#0f172a",
      padding: "8px 0",
    },
    ".cm-line": {
      backgroundColor: "#ffffff !important",
      padding: "0 12px",
      color: "#1e293b",
    },
    ".cm-gutters": {
      backgroundColor: "#f8fafc !important",
      color: "#94a3b8",
      border: "none",
      borderRight: "1px solid #e2e8f0",
    },
    ".cm-gutter": {
      backgroundColor: "#f8fafc !important",
    },
    ".cm-gutterElement": {
      backgroundColor: "#f8fafc !important",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "#e2e8f0 !important",
      color: "#475569",
    },
    ".cm-activeLine": {
      backgroundColor: "#f8fafc !important",
    },
    ".cm-selectionBackground, ::selection": {
      backgroundColor: "#bfdbfe !important",
    },
    ".cm-cursor": {
      borderLeftColor: "#0f172a",
    },
    ".cm-tooltip-autocomplete": {
      backgroundColor: "#ffffff !important",
      border: "1px solid #e2e8f0",
      borderRadius: "6px",
      boxShadow: "0 8px 24px rgba(15,23,42,0.14)",
    },
    ".cm-tooltip-autocomplete ul li": {
      fontSize: "12px",
      padding: "3px 10px",
      color: "#334155",
    },
    ".cm-tooltip-autocomplete ul li[aria-selected]": {
      backgroundColor: "#0f172a !important",
      color: "#ffffff",
    },
  },
  { dark: false },
);
