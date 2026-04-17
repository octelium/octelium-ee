import $RefParser from "@apidevtools/json-schema-ref-parser";
import {
  acceptCompletion,
  closeBrackets,
  completionKeymap,
  completionStatus,
  startCompletion,
} from "@codemirror/autocomplete";
import { history, historyKeymap } from "@codemirror/commands";
import {
  HighlightStyle,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { drawSelection, EditorView, keymap } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror from "@uiw/react-codemirror";
import { useEffect, useRef, useState } from "react";
import RequestContext from "../../jsonschema/core/RequestContext.json";
import { schemaAutocomplete } from "./celCompletion";
import { regoHighlight, regoLanguage, regoTheme } from "./opa";

const celLanguage = StreamLanguage.define({
  token(stream) {
    if (stream.eatSpace()) return null;
    if (stream.match("//")) {
      stream.skipToEnd();
      return "comment";
    }
    if (stream.match(/"(?:[^"\\]|\\.)*"/)) return "string";
    if (stream.match(/(?:\d+\.\d*|\d*\.\d+|\d+)/)) return "number";
    if (stream.match(/\b(true|false|null|in|exists|has|map|list|size|type)\b/))
      return "keyword";
    if (stream.match(/==|!=|<=|>=|&&|\|\||[<>+\-*/%!]/)) return "operator";
    if (stream.match(/[A-Za-z_][A-Za-z0-9_.]*/)) return "variableName";
    stream.next();
    return null;
  },
});

const celHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: t.keyword, color: "#7c3aed", fontWeight: "bold" },
    { tag: t.string, color: "#059669" },
    { tag: t.number, color: "#0284c7" },
    { tag: t.variableName, color: "#1e293b" },
    { tag: t.comment, color: "#94a3b8", fontStyle: "italic" },
    { tag: t.operator, color: "#475569", fontWeight: "bold" },
  ]),
);

const singleLineFilter = EditorState.transactionFilter.of((tr) => {
  if (tr.newDoc.lines > 1) {
    if (tr.isUserEvent("input.paste")) {
      return [
        tr,
        {
          changes: {
            from: 0,
            to: tr.newDoc.length,
            insert: tr.newDoc.sliceString(0, undefined, " "),
          },
          sequential: true,
        },
      ];
    }
    return [];
  }
  return tr;
});

const inputDotTrigger = EditorView.inputHandler.of((view, _from, _to, text) => {
  if (text === ".") {
    setTimeout(() => startCompletion(view), 0);
  }
  return false;
});

const celBaseTheme = EditorView.theme({
  "&": {
    backgroundColor: "#ffffff",
    border: "1px solid #e2e8f0",
    borderRadius: "6px",
    boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
  },
  "&.cm-focused": {
    outline: "none",
    borderColor: "#94a3b8",
    boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
  },
  ".cm-content": {
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "13px",
    padding: "8px 10px",
    caretColor: "#0f172a",
  },
  ".cm-line": { padding: "0" },
  ".cm-tooltip": {
    border: "none !important",
    backgroundColor: "transparent !important",
    boxShadow: "none !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete": {
    backgroundColor: "#ffffff !important",
    border: "1px solid #e2e8f0 !important",
    borderRadius: "10px !important",
    boxShadow:
      "0 12px 32px rgba(15,23,42,0.14), 0 2px 8px rgba(15,23,42,0.08) !important",
    zIndex: "1000 !important",
    minWidth: "240px",
    overflow: "hidden",
    padding: "4px",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul": {
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important",
    fontSize: "13px !important",
    maxHeight: "280px !important",
    overflowY: "auto",
    margin: "0",
    padding: "0",
    backgroundColor: "#ffffff !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
    padding: "6px 10px !important",
    color: "#334155 !important",
    borderRadius: "6px",
    margin: "1px 0",
    display: "flex",
    alignItems: "center",
    lineHeight: "1.4",
    cursor: "pointer",
    backgroundColor: "#ffffff !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "#0f172a !important",
    color: "#ffffff !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail":
    {
      color: "#94a3b8 !important",
    },
  ".cm-completionIcon": { display: "none" },
  ".cm-completionLabel": {
    fontWeight: "700",
    fontSize: "13px",
    flex: "1",
  },
  ".cm-completionDetail": {
    color: "#94a3b8",
    fontSize: "11px",
    marginLeft: "10px",
    fontStyle: "normal",
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    whiteSpace: "nowrap",
  },
  ".cm-completionMatchedText": {
    textDecoration: "none",
    color: "#1d4ed8",
    fontWeight: "800",
  },
  "li[aria-selected] .cm-completionMatchedText": {
    color: "#93c5fd",
  },
});

let resolvedSchema: any = null;

export const CELEditor = (props: {
  exp: string;
  onChange: (val: string) => void;
}) => {
  const [extensions, setExtensions] = useState<any[]>([]);
  const initialized = useRef(false);

  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;

    async function init() {
      if (!resolvedSchema) {
        resolvedSchema = await $RefParser.dereference(RequestContext);
      }

      const completionExtension = schemaAutocomplete({
        type: "object",
        properties: { ctx: resolvedSchema },
      });

      setExtensions([
        celLanguage,
        celHighlight,
        celBaseTheme,
        history(),
        drawSelection(),
        closeBrackets(),
        completionExtension,
        inputDotTrigger,
        singleLineFilter,
        keymap.of([
          {
            key: "Enter",
            run: (view) => {
              if (completionStatus(view.state) === "active") {
                return acceptCompletion(view);
              }
              return true;
            },
            preventDefault: true,
          },
          {
            key: "Tab",
            run: (view) => {
              if (completionStatus(view.state) !== null) {
                return acceptCompletion(view);
              }
              return true;
            },
            preventDefault: true,
          },
          ...completionKeymap.filter(
            (b) => b.key !== "Enter" && b.key !== "Tab",
          ),
          ...historyKeymap,
        ]),
      ]);
    }

    init();
  }, []);

  return (
    <div className="mt-3 w-full">
      <CodeMirror
        value={props.exp}
        autoFocus
        tabIndex={-1}
        extensions={extensions}
        height="38px"
        basicSetup={false}
        onChange={(val) => props.onChange(val)}
      />
    </div>
  );
};

export const OPAEditor = (props: {
  exp: string;
  onChange: (val: string) => void;
}) => {
  return (
    <div className="mt-3 w-full rounded-lg overflow-hidden border border-slate-700">
      <CodeMirror
        value={props.exp}
        autoFocus
        extensions={[regoLanguage, regoHighlight, regoTheme]}
        minHeight="160px"
        basicSetup={{
          lineNumbers: true,
          foldGutter: false,
          autocompletion: false,
          highlightActiveLine: true,
        }}
        onChange={(val) => props.onChange(val)}
      />
    </div>
  );
};
