import React from "react";
import CodeMirror from "@uiw/react-codemirror";
import { langs } from "@uiw/codemirror-extensions-langs";
import { match } from "ts-pattern";
import { Resource, resourceFromYAML } from "@/utils/pb";

import resourceJSONSchema from "../../jsonschema";

import { EditorState } from "@codemirror/state";
import {
  gutter,
  EditorView,
  lineNumbers,
  drawSelection,
  keymap,
  highlightActiveLineGutter,
  ViewUpdate,
} from "@codemirror/view";
import { lintKeymap, lintGutter } from "@codemirror/lint";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import {
  syntaxHighlighting,
  indentOnInput,
  bracketMatching,
  foldGutter,
  foldKeymap,
} from "@codemirror/language";
import { oneDarkHighlightStyle, oneDark } from "@codemirror/theme-one-dark";
import {
  autocompletion,
  completionKeymap,
  closeBrackets,
  closeBracketsKeymap,
} from "@codemirror/autocomplete";

import { JSONSchema7 } from "json-schema";
import {
  yamlSchema,
  yamlSchemaHover,
  yamlCompletion,
  yamlSchemaLinter,
} from "codemirror-json-schema/yaml";
import { hoverTooltip } from "@codemirror/view";

import { yaml, yamlFrontmatter, yamlLanguage } from "@codemirror/lang-yaml";
import { linter } from "@codemirror/lint";

const Editor = (props: {
  value: string;
  mode: "yaml" | "dockerfile" | "shell" | "json" | undefined;
  onChange?: (val: string) => void;

  readOnly?: boolean;
  onResourceChange?: (arg: Resource) => void;
  item: Resource;
}) => {
  let extensions = [
    // basicSetup,
    /*
    gutter({ class: "CodeMirror-lint-markers" }),
    bracketMatching(),
    highlightActiveLineGutter(),

    closeBrackets(),
    history(),
    autocompletion({ activateOnTyping: true }),
    lineNumbers(),
    lintGutter(),
    indentOnInput(),
    drawSelection(),
    foldGutter(),
    */

    EditorView.lineWrapping,
    EditorState.tabSize.of(2),
    yaml(),

    /*
    linter(jsonParseLinter(), {
      // default is 750ms
      delay: 300,
    }),
    linter(jsonSchemaLinter(), {
      needsRefresh: handleRefresh,
    }),
    */
    yamlLanguage.data.of({
      autocomplete: yamlCompletion(),
    }),
    // hoverTooltip(yamlSchemaHover()),

    /*
    keymap.of([
      ...closeBracketsKeymap,
      ...defaultKeymap,
      ...historyKeymap,
      ...foldKeymap,
      ...completionKeymap,
      ...lintKeymap,
    ]),
    */
  ];

  /*
  const ext = match(props.mode)
    .with("yaml", () => langs.yaml())
    .with("json", () => langs.json())
    .otherwise(() => undefined);
  if (ext) {
    extensions.push(ext);
  }
  */

  extensions.push(yamlSchema(resourceJSONSchema(props.item) as JSONSchema7));

  return (
    <div className="font-bold rounded-xl overflow-hidden w-full shadow-2xl text-xs">
      <CodeMirror
        value={props.value}
        autoFocus={true}
        readOnly={props.readOnly}
        className="w-full"
        theme={"dark"}
        maxHeight="600px"
        minHeight="300px"
        extensions={extensions}
        onChange={(val) => {
          if (props.onChange) {
            props.onChange(val);
          }

          if (props.onResourceChange) {
            match(props.mode)
              .with("yaml", () => {
                const res = resourceFromYAML(val);
                if (res && props.onResourceChange) {
                  props.onResourceChange(res);
                }
              })
              .otherwise(() => {
                return false;
              });
          }
        }}
      />
    </div>
  );
};

export default Editor;
