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
import { useEffect, useState } from "react";
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
    { tag: t.keyword, color: "var(--color-violet-600)", fontWeight: "bold" },
    { tag: t.string, color: "var(--color-emerald-600)" },
    { tag: t.number, color: "var(--color-sky-600)" },
    { tag: t.variableName, color: "var(--color-slate-800)" },
    { tag: t.comment, color: "var(--color-slate-400)", fontStyle: "italic" },
    { tag: t.operator, color: "var(--color-slate-600)", fontWeight: "bold" },
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
    setTimeout(() => {
      if (view.dom.isConnected) startCompletion(view);
    }, 0);
  }
  return false;
});

const celKeymap = keymap.of([
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
    run: (view) =>
      completionStatus(view.state) === "active"
        ? acceptCompletion(view)
        : false,
  },
  ...completionKeymap.filter((binding) => binding.key !== "Enter" && binding.key !== "Tab"),
  ...historyKeymap,
]);

const celBaseTheme = EditorView.theme({
  "&": {
    backgroundColor: "var(--color-white)",
    border: "1px solid var(--color-slate-200)",
    borderRadius: "6px",
    boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
  },
  "&.cm-focused": {
    outline: "none",
    borderColor: "var(--color-slate-400)",
    boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
  },
  ".cm-content": {
    fontSize: "13px",
    padding: "8px 10px",
    caretColor: "var(--color-slate-900)",
  },
  ".cm-line": { padding: "0" },
  ".cm-tooltip": {
    border: "none !important",
    backgroundColor: "transparent !important",
    boxShadow: "none !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete": {
    backgroundColor: "var(--color-white) !important",
    border: "1px solid var(--color-slate-200) !important",
    borderRadius: "10px !important",
    boxShadow:
      "0 12px 32px rgba(15,23,42,0.14), 0 2px 8px rgba(15,23,42,0.08) !important",
    zIndex: "1000 !important",
    minWidth: "240px",
    overflow: "hidden",
    padding: "4px",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul": {
    fontSize: "13px !important",
    maxHeight: "280px !important",
    overflowY: "auto",
    margin: "0",
    padding: "0",
    backgroundColor: "var(--color-white) !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
    padding: "6px 10px !important",
    color: "var(--color-slate-700) !important",
    borderRadius: "6px",
    margin: "1px 0",
    display: "flex",
    alignItems: "center",
    lineHeight: "1.4",
    cursor: "pointer",
    backgroundColor: "var(--color-white) !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "var(--color-slate-900) !important",
    color: "var(--color-white) !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail":
    {
      color: "var(--color-slate-400) !important",
    },
  ".cm-completionIcon": { display: "none" },
  ".cm-completionLabel": {
    fontWeight: "700",
    fontSize: "13px",
    flex: "1",
  },
  ".cm-completionDetail": {
    color: "var(--color-slate-400)",
    fontSize: "11px",
    marginLeft: "10px",
    fontStyle: "normal",
    whiteSpace: "nowrap",
  },
  ".cm-completionMatchedText": {
    textDecoration: "none",
    color: "var(--color-blue-700)",
    fontWeight: "800",
  },
  "li[aria-selected] .cm-completionMatchedText": {
    color: "var(--color-blue-300)",
  },
});

let resolvedSchemaPromise: Promise<any> | undefined;

const getResolvedSchema = () =>
  (resolvedSchemaPromise ??= $RefParser.dereference(RequestContext));

const celBaseExtensions = [
  celLanguage,
  celHighlight,
  celBaseTheme,
  history(),
  drawSelection(),
  closeBrackets(),
  inputDotTrigger,
  singleLineFilter,
  celKeymap,
];

const regoExtensions = [regoLanguage, regoHighlight, regoTheme];

export const CELEditor = (props: {
  exp: string;
  label?: string;
  invalid?: boolean;
  onChange: (val: string) => void;
}) => {
  const [extensions, setExtensions] = useState<any[]>(celBaseExtensions);
  const [completionError, setCompletionError] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function init() {
      try {
        const resolvedSchema = await getResolvedSchema();
        if (cancelled) return;
        setExtensions([
          ...celBaseExtensions,
          schemaAutocomplete({
            type: "object",
            properties: { ctx: resolvedSchema },
          }),
        ]);
      } catch {
        if (!cancelled) setCompletionError(true);
      }
    }

    void init();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="mt-3 w-full">
      <CodeMirror
        value={props.exp}
        aria-label={props.label ?? "CEL expression"}
        aria-invalid={props.invalid || undefined}
        tabIndex={0}
        extensions={extensions}
        height="38px"
        basicSetup={false}
        onChange={(val) => props.onChange(val)}
      />
      {completionError && (
        <p className="mt-1 text-micro font-semibold text-amber-700" role="status">
          Context completion is unavailable, but the expression editor remains usable.
        </p>
      )}
    </div>
  );
};

export const OPAEditor = (props: {
  exp: string;
  label?: string;
  invalid?: boolean;
  onChange: (val: string) => void;
}) => {
  return (
    <div className="mt-3 w-full rounded-lg overflow-hidden border border-slate-700">
      <CodeMirror
        value={props.exp}
        aria-label={props.label ?? "OPA/Rego policy"}
        aria-invalid={props.invalid || undefined}
        tabIndex={0}
        extensions={regoExtensions}
        minHeight="160px"
        maxHeight="420px"
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
