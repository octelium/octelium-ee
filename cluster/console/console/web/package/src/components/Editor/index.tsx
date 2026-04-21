import { Resource, resourceFromYAML } from "@/utils/pb";
import { closeBrackets } from "@codemirror/autocomplete";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { json } from "@codemirror/lang-json";
import { yaml, yamlLanguage } from "@codemirror/lang-yaml";
import { foldGutter } from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { oneDark } from "@codemirror/theme-one-dark";
import {
  drawSelection,
  EditorView,
  keymap,
  lineNumbers,
} from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { jsonSchema } from "codemirror-json-schema";
import { yamlCompletion, yamlSchema } from "codemirror-json-schema/yaml";
import { JSONSchema7 } from "json-schema";
import { match } from "ts-pattern";
import resourceJSONSchema from "../../jsonschema";

const fontTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "#282c34",
      height: "100%",
    },
    ".cm-content": {
      fontSize: "14px",
      fontWeight: `700`,
      lineHeight: "1.6",
    },
    ".cm-gutters": {
      fontSize: "13px",
    },
    ".cm-scroller": {
      overflow: "auto",
      backgroundColor: "#282c34",
    },
  },
  { dark: true },
);

const Editor = (props: {
  value: string;
  mode: "yaml" | "dockerfile" | "shell" | "json" | undefined;
  onChange?: (val: string) => void;
  readOnly?: boolean;
  onResourceChange?: (arg: Resource) => void;
  item: Resource;
  schemaMode?: "full" | "spec" | "status" | "metadata";
  minHeight?: string;
  maxHeight?: string;
}) => {
  const minHeight = props.minHeight ?? "300px";
  const maxHeight = props.maxHeight ?? "600px";
  const schema = resourceJSONSchema(props.item) as JSONSchema7;

  const activeSchema =
    !props.schemaMode || props.schemaMode === "full" ? schema : null;

  const fillTheme = EditorView.theme({
    "&.cm-editor": { minHeight, height: "100%" },
    ".cm-scroller": { minHeight, overflow: "auto" },
  });

  const extensions = [
    oneDark,
    fontTheme,
    fillTheme,
    EditorView.lineWrapping,
    EditorState.tabSize.of(2),
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
    <div className="w-full rounded-lg overflow-hidden border border-slate-700 bg-[#282c34]">
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
