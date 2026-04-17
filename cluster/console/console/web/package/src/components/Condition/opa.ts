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
    { tag: t.keyword, color: "#ff7b72", fontWeight: "bold" },
    { tag: t.variableName, color: "#d2a8ff" },
    { tag: t.atom, color: "#79c0ff" },
    { tag: t.number, color: "#ffa657" },
    { tag: t.string, color: "#a5d6ff" },
    { tag: t.comment, color: "#8b949e", fontStyle: "italic" },
    { tag: t.operator, color: "#ff7b72" },
    { tag: t.punctuation, color: "#c9d1d9" },
  ]),
);

export const regoTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "#0d1117 !important",
      color: "#c9d1d9",
    },
    "&.cm-editor": {
      backgroundColor: "#0d1117 !important",
    },
    ".cm-scroller": {
      backgroundColor: "#0d1117 !important",
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    },
    ".cm-content": {
      backgroundColor: "#0d1117 !important",
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize: "13px",
      caretColor: "#c9d1d9",
      padding: "8px 0",
    },
    ".cm-line": {
      backgroundColor: "#0d1117 !important",
      padding: "0 12px",
      color: "#c9d1d9",
    },
    ".cm-gutters": {
      backgroundColor: "#0d1117 !important",
      color: "#484f58",
      border: "none",
      borderRight: "1px solid #21262d",
    },
    ".cm-gutter": {
      backgroundColor: "#0d1117 !important",
    },
    ".cm-gutterElement": {
      backgroundColor: "#0d1117 !important",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "#161b22 !important",
      color: "#8b949e",
    },
    ".cm-activeLine": {
      backgroundColor: "#161b22 !important",
    },
    ".cm-selectionBackground, ::selection": {
      backgroundColor: "#264f78 !important",
    },
    ".cm-cursor": {
      borderLeftColor: "#c9d1d9",
    },
    ".cm-tooltip-autocomplete": {
      backgroundColor: "#161b22 !important",
      border: "1px solid #30363d",
      borderRadius: "6px",
      boxShadow: "0 8px 24px rgba(1,4,9,0.4)",
    },
    ".cm-tooltip-autocomplete ul li": {
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      fontSize: "12px",
      padding: "3px 10px",
      color: "#c9d1d9",
    },
    ".cm-tooltip-autocomplete ul li[aria-selected]": {
      backgroundColor: "#1f6feb !important",
      color: "#ffffff",
    },
  },
  { dark: true },
);
