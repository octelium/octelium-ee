import CodeMirror from "@uiw/react-codemirror";

import {
  HighlightStyle,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { tags as t } from "@lezer/highlight";

import $RefParser from "@apidevtools/json-schema-ref-parser";
import { acceptCompletion, startCompletion } from "@codemirror/autocomplete";
import { defaultKeymap } from "@codemirror/commands";
import { EditorState, Extension } from "@codemirror/state";
import RequestContext from "../../jsonschema/core/RequestContext.json";
import { schemaAutocomplete } from "./celCompletion";

import { completionStatus } from "@codemirror/autocomplete";
import { indentMore } from "@codemirror/commands";

import { EditorView, keymap, ViewUpdate } from "@codemirror/view";
import { useEffect, useState } from "react";
import { regoLanguage, regoTheme } from "./opa";

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
    { tag: t.keyword, color: "#c792ea" },
    { tag: t.string, color: "#555" },
    { tag: t.number, color: "#333" },
    { tag: t.variableName, color: "#333" },
    { tag: t.comment, color: "#546e7a" },
    { tag: t.operator, color: "#000" },
  ]),
);

const triggerOnDot = EditorView.inputHandler.of((view, from, to, text) => {
  if (text === ".") setTimeout(() => startCompletion(view), 0);
  return false;
});

const tabAcceptCompletion = keymap.of([
  {
    key: "Tab",
    run: acceptCompletion,
  },
]);

const singleLineFilter: Extension = EditorState.transactionFilter.of((tr) => {
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

const smartTab = keymap.of([
  {
    key: "Tab",
    run: (view) => {
      const status = completionStatus(view.state);

      if (status === "active") {
        acceptCompletion(view);
        return true;
      }

      if (status === null) {
        startCompletion(view);
        return true;
      }

      return indentMore(view);
    },
    preventDefault: true,
  },
]);

const filteredKeymap = defaultKeymap.filter((k) => k.key !== "Tab");

const tabAccept = keymap.of([
  { key: "Tab", run: acceptCompletion, preventDefault: true },
  {
    key: "Shift-Tab",
    run: (view) => {
      const list = document.querySelector(".cm-tooltip-autocomplete ul");
      if (list)
        (list as HTMLElement).dispatchEvent(
          new KeyboardEvent("keydown", { key: "ArrowUp" }),
        );
      return true;
    },
  },
]);

const noTabKeymap = defaultKeymap.filter((k) => k.key !== "Tab");

const tabAcceptOnly = keymap.of([
  {
    key: "Tab",
    run: acceptCompletion,
    preventDefault: true,
  },
]);
const autoDotTrigger = EditorView.updateListener.of((update: ViewUpdate) => {
  if (update.docChanged) {
    const last = update.state.doc.sliceString(
      Math.max(0, update.state.selection.main.from - 1),
      update.state.selection.main.from,
    );
    if (last === ".") {
      startCompletion(update.view);
    }
  }
});

export const inputDotTrigger = EditorView.inputHandler.of(
  (view, from, to, text) => {
    if (text === ".") {
      setTimeout(() => startCompletion(view), 0);
    }
    return false;
  },
);

export const CELEditor = (props: {
  exp: string;
  onChange: (val: string) => void;
}) => {
  const [extensions, setExtensions] = useState<any[]>([]);

  const oneLineKeymap = keymap.of([
    {
      key: "Enter",
      run: () => true,
      preventDefault: true,
    },

    ...defaultKeymap,
  ]);

  useEffect(() => {
    async function init() {
      const resolved = await $RefParser.dereference(RequestContext);

      setExtensions([
        celLanguage,
        celHighlight,
        keymap.of(noTabKeymap),
        schemaAutocomplete({
          type: "object",
          properties: {
            ctx: resolved,
          },
        }),

        inputDotTrigger,
        singleLineFilter,
        tabAcceptOnly,
      ]);
    }
    init();
  }, []);

  return (
    <CodeMirror
      value={props.exp}
      autoFocus={true}
      className="mt-4 w-full !rounded !outline-none"
      tabIndex={-1}
      extensions={extensions}
      height="50px"
      basicSetup={{
        lineNumbers: false,
        autocompletion: true,
        indentOnInput: false,
      }}
      onChange={(val) => {
        props.onChange(val);
      }}
    />
  );
};

export const OPAEditor = (props: {
  exp: string;
  onChange: (val: string) => void;
}) => {
  return (
    <CodeMirror
      value={props.exp}
      autoFocus={true}
      className="mt-4 w-full !rounded !outline-none"
      theme={"dark"}
      extensions={[regoLanguage, regoTheme]}
      minHeight="150px"
      lang="rego"
      basicSetup={{
        lineNumbers: true,
      }}
      onChange={(val) => {
        props.onChange(val);
      }}
    />
  );
};
