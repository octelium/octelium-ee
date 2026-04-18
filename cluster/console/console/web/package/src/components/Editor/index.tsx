import { Resource, resourceFromYAML } from "@/utils/pb";
import { closeBrackets } from "@codemirror/autocomplete";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { json } from "@codemirror/lang-json";
import { yaml, yamlLanguage } from "@codemirror/lang-yaml";
import {
  foldGutter,
  HighlightStyle,
  syntaxHighlighting,
} from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import {
  drawSelection,
  EditorView,
  keymap,
  lineNumbers,
} from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror from "@uiw/react-codemirror";
import { jsonSchema } from "codemirror-json-schema";
import { yamlCompletion, yamlSchema } from "codemirror-json-schema/yaml";
import { JSONSchema7 } from "json-schema";
import { match } from "ts-pattern";
import resourceJSONSchema from "../../jsonschema";

const darkTheme = EditorView.theme(
  {
    "&": { backgroundColor: "#0d1117", color: "#c9d1d9" },
    "&.cm-editor": { minHeight: "300px" },
    ".cm-scroller": {
      backgroundColor: "#0d1117 !important",
      minHeight: "300px",
    },
    ".cm-content": {
      fontSize: "13px",
      padding: "12px 0",
      caretColor: "#c9d1d9",
    },
    ".cm-line": { padding: "0 16px" },
    ".cm-gutters": {
      backgroundColor: "#0d1117 !important",
      color: "#484f58",
      border: "none",
      borderRight: "1px solid #21262d",

      fontSize: "12px",
    },
    ".cm-gutterElement": {
      padding: "0 8px 0 4px",
      backgroundColor: "#0d1117 !important",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "#161b22 !important",
      color: "#8b949e",
    },
    ".cm-activeLine": { backgroundColor: "#161b22 !important" },
    ".cm-selectionBackground, ::selection": {
      backgroundColor: "#264f78 !important",
    },
    ".cm-cursor": { borderLeftColor: "#c9d1d9" },
    ".cm-foldGutter": { padding: "0 4px" },
    ".cm-tooltip": {
      backgroundColor: "#161b22 !important",
      border: "1px solid #30363d !important",
      borderRadius: "8px !important",
      boxShadow: "0 8px 24px rgba(1,4,9,0.5) !important",
      color: "#c9d1d9",
    },
    ".cm-tooltip-hover": {
      backgroundColor: "#161b22 !important",
      border: "1px solid #30363d !important",
      borderRadius: "8px !important",
      boxShadow: "0 8px 24px rgba(1,4,9,0.5) !important",
      padding: "8px 12px",
      maxWidth: "360px",
      fontSize: "12px",
      lineHeight: "1.6",
      color: "#c9d1d9",
    },
    ".cm-tooltip.cm-tooltip-autocomplete": {
      backgroundColor: "#161b22 !important",
      border: "1px solid #30363d !important",
      borderRadius: "10px !important",
      boxShadow: "0 12px 32px rgba(1,4,9,0.5) !important",
      overflow: "hidden",
      padding: "4px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul": {
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important",
      fontSize: "12px !important",
      maxHeight: "260px !important",
      overflowY: "auto",
      backgroundColor: "#161b22 !important",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
      padding: "5px 10px !important",
      color: "#c9d1d9 !important",
      borderRadius: "6px",
      margin: "1px 0",
      backgroundColor: "#161b22 !important",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
      backgroundColor: "#1f6feb !important",
      color: "#ffffff !important",
    },
    ".cm-completionIcon": { display: "none" },
    ".cm-completionLabel": { fontWeight: "600", fontSize: "12px" },
    ".cm-completionDetail": {
      color: "#8b949e",
      fontSize: "11px",
      marginLeft: "8px",
      fontStyle: "normal",
    },
    ".cm-lintRange-error": {
      backgroundImage: "none",
      borderBottom: "2px solid #f85149",
    },
    ".cm-lintRange-warning": {
      backgroundImage: "none",
      borderBottom: "2px solid #d29922",
    },
    ".cm-diagnostic-error": {
      borderLeft: "3px solid #f85149",
      paddingLeft: "8px",
      backgroundColor: "rgba(248,81,73,0.1)",
      borderRadius: "0 4px 4px 0",
      fontSize: "12px",
      color: "#ffa198",
    },
    ".cm-diagnostic-warning": {
      borderLeft: "3px solid #d29922",
      paddingLeft: "8px",
      backgroundColor: "rgba(210,153,34,0.1)",
      borderRadius: "0 4px 4px 0",
      fontSize: "12px",
      color: "#e3b341",
    },
  },
  { dark: true },
);

const lightTheme = EditorView.theme({
  "&": { backgroundColor: "#ffffff", color: "#1e293b" },
  ".cm-content": {
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "13px",
    padding: "12px 0",
    caretColor: "#0f172a",
  },
  ".cm-line": { padding: "0 16px" },
  ".cm-gutters": {
    backgroundColor: "#f8fafc !important",
    color: "#94a3b8",
    border: "none",
    borderRight: "1px solid #f1f5f9",
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize: "12px",
  },
  ".cm-gutterElement": { padding: "0 8px 0 4px" },
  ".cm-activeLineGutter": {
    backgroundColor: "#f1f5f9 !important",
    color: "#475569",
  },
  ".cm-activeLine": { backgroundColor: "#f8fafc !important" },
  ".cm-selectionBackground, ::selection": {
    backgroundColor: "#bfdbfe !important",
  },
  ".cm-cursor": { borderLeftColor: "#0f172a" },
  ".cm-foldGutter": { padding: "0 4px" },
  ".cm-tooltip": {
    backgroundColor: "#ffffff !important",
    border: "1px solid #e2e8f0 !important",
    borderRadius: "8px !important",
    boxShadow: "0 8px 24px rgba(15,23,42,0.12) !important",
    color: "#1e293b",
  },
  ".cm-tooltip-hover": {
    backgroundColor: "#ffffff !important",
    border: "1px solid #e2e8f0 !important",
    borderRadius: "8px !important",
    boxShadow: "0 8px 24px rgba(15,23,42,0.12) !important",
    padding: "8px 12px",
    maxWidth: "360px",
    fontSize: "12px",
    lineHeight: "1.6",
    color: "#1e293b",
  },
  ".cm-tooltip.cm-tooltip-autocomplete": {
    backgroundColor: "#ffffff !important",
    border: "1px solid #e2e8f0 !important",
    borderRadius: "10px !important",
    boxShadow: "0 12px 32px rgba(15,23,42,0.12) !important",
    overflow: "hidden",
    padding: "4px",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul": {
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important",
    fontSize: "12px !important",
    maxHeight: "260px !important",
    overflowY: "auto",
    backgroundColor: "#ffffff !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
    padding: "5px 10px !important",
    color: "#334155 !important",
    borderRadius: "6px",
    margin: "1px 0",
    backgroundColor: "#ffffff !important",
  },
  ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "#0f172a !important",
    color: "#ffffff !important",
  },
  ".cm-completionIcon": { display: "none" },
  ".cm-completionLabel": { fontWeight: "600", fontSize: "12px" },
  ".cm-completionDetail": {
    color: "#94a3b8",
    fontSize: "11px",
    marginLeft: "8px",
    fontStyle: "normal",
  },
  ".cm-lintRange-error": {
    backgroundImage: "none",
    borderBottom: "2px solid #ef4444",
  },
  ".cm-lintRange-warning": {
    backgroundImage: "none",
    borderBottom: "2px solid #f59e0b",
  },
  ".cm-diagnostic-error": {
    borderLeft: "3px solid #ef4444",
    paddingLeft: "8px",
    backgroundColor: "#fef2f2",
    borderRadius: "0 4px 4px 0",
    fontSize: "12px",
    color: "#991b1b",
  },
  ".cm-diagnostic-warning": {
    borderLeft: "3px solid #f59e0b",
    paddingLeft: "8px",
    backgroundColor: "#fffbeb",
    borderRadius: "0 4px 4px 0",
    fontSize: "12px",
    color: "#92400e",
  },
});

const darkHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: t.keyword, color: "#ff7b72", fontWeight: "bold" },
    { tag: t.string, color: "#a5d6ff" },
    { tag: t.number, color: "#ffa657" },
    { tag: t.bool, color: "#79c0ff", fontWeight: "bold" },
    { tag: t.null, color: "#8b949e", fontStyle: "italic" },
    { tag: t.comment, color: "#8b949e", fontStyle: "italic" },
    { tag: t.propertyName, color: "#7ee787", fontWeight: "700" },
    { tag: t.variableName, color: "#c9d1d9" },
    { tag: t.operator, color: "#ff7b72" },
    { tag: t.punctuation, color: "#8b949e" },
    { tag: t.meta, color: "#d2a8ff" },
    { tag: t.atom, color: "#79c0ff" },
  ]),
);

const lightHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: t.keyword, color: "#7c3aed", fontWeight: "bold" },
    { tag: t.string, color: "#059669" },
    { tag: t.number, color: "#0284c7" },
    { tag: t.bool, color: "#d97706", fontWeight: "bold" },
    { tag: t.null, color: "#94a3b8", fontStyle: "italic" },
    { tag: t.comment, color: "#94a3b8", fontStyle: "italic" },
    { tag: t.propertyName, color: "#0f172a", fontWeight: "700" },
    { tag: t.variableName, color: "#334155" },
    { tag: t.operator, color: "#475569" },
    { tag: t.punctuation, color: "#64748b" },
    { tag: t.meta, color: "#7c3aed" },
    { tag: t.atom, color: "#0284c7" },
  ]),
);

const Editor = (props: {
  value: string;
  mode: "yaml" | "dockerfile" | "shell" | "json" | undefined;
  onChange?: (val: string) => void;
  readOnly?: boolean;
  onResourceChange?: (arg: Resource) => void;
  item: Resource;
  colorScheme?: "dark" | "light";
  schemaMode?: "full" | "spec" | "status" | "metadata";
  minHeight?: string;
  maxHeight?: string;
}) => {
  const isDark = (props.colorScheme ?? "dark") === "dark";
  const minHeight = props.minHeight ?? "300px";
  const maxHeight = props.maxHeight ?? "600px";
  const schema = resourceJSONSchema(props.item) as JSONSchema7;

  const getSchemaForMode = (): JSONSchema7 | null => {
    if (!props.schemaMode || props.schemaMode === "full") return schema;
    return null;
  };

  const activeSchema = getSchemaForMode();

  const fillTheme = EditorView.theme({
    "&.cm-editor": { minHeight },
    ".cm-scroller": { minHeight },
  });

  const extensions = [
    EditorView.lineWrapping,
    EditorState.tabSize.of(2),
    isDark ? darkTheme : lightTheme,
    isDark ? darkHighlight : lightHighlight,
    fillTheme,
    history(),
    drawSelection(),
    closeBrackets(),
    lineNumbers(),
    foldGutter(),
    keymap.of([...defaultKeymap, ...historyKeymap]),
    ...(props.mode === "json"
      ? [json(), ...(activeSchema ? [jsonSchema(activeSchema)] : [])]
      : [
          yaml(),
          yamlLanguage.data.of({ autocomplete: yamlCompletion() }),
          ...(activeSchema ? [yamlSchema(activeSchema)] : []),
        ]),
  ];

  return (
    <div
      className={`w-full rounded-lg overflow-hidden border shadow-[0_1px_4px_rgba(15,23,42,0.06)] ${
        isDark ? "border-slate-700" : "border-slate-200"
      }`}
    >
      <CodeMirror
        value={props.value}
        autoFocus
        readOnly={props.readOnly}
        className="w-full"
        maxHeight={maxHeight}
        minHeight={minHeight}
        basicSetup={false}
        extensions={extensions}
        onChange={(val) => {
          props.onChange?.(val);
          if (props.onResourceChange) {
            match(props.mode)
              .with("yaml", () => {
                const res = resourceFromYAML(val);
                if (res) props.onResourceChange!(res);
              })
              .otherwise(() => {});
          }
        }}
      />
    </div>
  );
};

export default Editor;
