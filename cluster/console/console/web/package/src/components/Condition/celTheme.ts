import {
  HighlightStyle,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags } from "@lezer/highlight";

export const celLanguage = StreamLanguage.define({
  startState: () => ({ quote: "", raw: false }),
  token(stream, state) {
    if (stream.sol() && state.quote.length === 1) state.quote = "";
    if (!state.quote) {
      if (stream.eatSpace()) return null;
      if (stream.match("//")) {
        stream.skipToEnd();
        return "comment";
      }
      const literal = stream.match(/[bB]?[rR]?(?:"""|'''|"|')/);
      if (literal && typeof literal !== "boolean") {
        const value = literal[0];
        state.quote = value.match(/"""|'''|"|'/)![0];
        state.raw = /[rR]/.test(value);
      } else {
        if (
          stream.match(/(?:0[xX][\da-fA-F]+|(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?)[uU]?/)
        ) return "number";
        if (stream.match(/\b(?:true|false|null|in)\b/)) return "keyword";
        if (stream.match(/==|!=|<=|>=|&&|\|\||[<>+\-*/%!?=:]/))
          return "operator";
        if (stream.match(/[A-Za-z_]\w*/)) {
          return stream.match(/\s*\(/, false)
            ? "variableName.function"
            : "variableName";
        }
        stream.next();
        return "punctuation";
      }
    }
    while (!stream.eol()) {
      if (stream.match(state.quote)) {
        state.quote = "";
        break;
      }
      if (stream.next() === "\\" && !state.raw) stream.next();
    }
    return "string";
  },
});

export const celHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: tags.keyword, color: "var(--color-violet-700)", fontWeight: "600" },
    { tag: tags.string, color: "var(--color-emerald-700)" },
    { tag: tags.number, color: "var(--color-sky-700)" },
    { tag: tags.variableName, color: "var(--color-slate-800)" },
    { tag: tags.function(tags.variableName), color: "var(--color-blue-700)" },
    { tag: tags.comment, color: "var(--color-slate-500)", fontStyle: "italic" },
    { tag: tags.operator, color: "var(--color-slate-600)" },
    { tag: tags.punctuation, color: "var(--color-slate-500)" },
  ]),
);

const mono = "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace";

export const celTheme = EditorView.theme({
  "&": {
    backgroundColor: "var(--color-white)",
    color: "var(--color-slate-800)",
    borderRadius: "11px 11px 0 0",
  },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": {
    fontFamily: mono,
    lineHeight: "1.7",
    overflowY: "auto",
  },
  ".cm-content": {
    fontSize: "13px",
    fontWeight: "500",
    padding: "12px 14px",
    caretColor: "var(--color-slate-900)",
    minHeight: "80px",
  },
  ".cm-line": { padding: "0" },
  ".cm-placeholder": { color: "var(--color-slate-500)" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--color-slate-900)" },
  "&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground": {
    backgroundColor: "var(--color-blue-200)",
  },
  ".cm-tooltip.cm-tooltip-autocomplete": {
    display: "flex",
    flexDirection: "column",
    backgroundColor: "var(--color-white)",
    color: "var(--color-slate-800)",
    border: "1px solid var(--color-slate-200)",
    borderRadius: "12px",
    boxShadow: "var(--shadow-overlay)",
    zIndex: "var(--mantine-z-index-tooltip, 1000)",
    width: "min(360px, calc(100vw - 24px))",
    maxWidth: "calc(100vw - 24px)",
    overflow: "hidden",
    fontFamily: "var(--mantine-font-family)",
    padding: "5px",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul": {
    flex: "1 1 auto",
    minHeight: "0",
    fontFamily: "inherit",
    fontSize: "12px",
    maxHeight: "min(320px, 45dvh)",
    overflowY: "auto",
    margin: "0",
    padding: "0",
    scrollbarWidth: "thin",
    scrollbarColor: "var(--color-slate-300) transparent",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
    display: "grid",
    gridTemplateColumns: "minmax(0, 1fr) auto",
    alignItems: "center",
    gap: "3px 12px",
    padding: "9px 10px",
    margin: "1px 0",
    borderRadius: "7px",
    color: "var(--color-slate-800)",
    lineHeight: "1.5",
    whiteSpace: "normal",
    cursor: "pointer",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li:hover": {
    backgroundColor: "var(--color-slate-50)",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "var(--color-slate-100)",
    color: "var(--color-slate-900)",
  },
  ".cm-completionLabel": {
    fontFamily: mono,
    fontWeight: "600",
    minWidth: "0",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  },
  ".cm-completionDetail": {
    fontSize: "11px",
    fontStyle: "normal",
    color: "var(--color-slate-500)",
    maxWidth: "160px",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  },
  ".cm-completionDescription": {
    gridColumn: "1 / -1",
    color: "var(--color-slate-600)",
    fontSize: "11px",
    display: "-webkit-box",
    WebkitBoxOrient: "vertical",
    WebkitLineClamp: "2",
    overflow: "hidden",
  },
  ".cm-completionMatchedText": {
    textDecoration: "none",
    color: "var(--color-blue-700)",
  },
});
