import {
  acceptCompletion,
  closeBrackets,
  completionKeymap,
  startCompletion,
} from "@codemirror/autocomplete";
import {
  defaultKeymap,
  history,
  historyKeymap,
  redo,
  undo,
} from "@codemirror/commands";
import { Prec, type Extension } from "@codemirror/state";
import {
  drawSelection,
  EditorView,
  keymap,
  placeholder,
  tooltips,
} from "@codemirror/view";
import { Braces } from "lucide-react";
import { useMemo, useRef } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { twMerge } from "@/lib/utils";
import { schemaAutocomplete } from "./celCompletion";
import { celSchemas, type CELContext } from "./celSchema";
import { celHighlight, celLanguage, celTheme } from "./celTheme";

const completions: Record<CELContext, Extension> = {
  core: schemaAutocomplete(celSchemas.core),
  access: schemaAutocomplete(celSchemas.access),
};

const baseExtensions: Extension[] = [
  celLanguage,
  celHighlight,
  celTheme,
  history(),
  drawSelection(),
  closeBrackets(),
  EditorView.lineWrapping,
  placeholder("Write an expression, or type ctx. to browse fields"),
  Prec.high(
    keymap.of([
      { key: "Tab", run: acceptCompletion },
      { key: "Mod-z", run: undo, shift: redo },
      ...completionKeymap,
    ]),
  ),
  keymap.of([...defaultKeymap, ...historyKeymap]),
];

export const CELEditor = (props: {
  exp: string;
  label?: string;
  invalid?: boolean;
  context?: CELContext;
  onChange: (value: string) => void;
}) => {
  const viewRef = useRef<EditorView | null>(null);
  const context = props.context ?? "core";
  const extensions = useMemo(
    () => [
      ...baseExtensions,
      completions[context],
      tooltips({
        parent: document.body,
        position: "fixed",
        tooltipSpace: (view) => {
          const document = view.dom.ownerDocument;
          const viewport = document.defaultView?.visualViewport;
          const left = viewport?.offsetLeft ?? 0;
          const top = viewport?.offsetTop ?? 0;
          return {
            left: left + 12,
            top: top + 12,
            right:
              left + (viewport?.width ?? document.documentElement.clientWidth) - 12,
            bottom:
              top + (viewport?.height ?? document.documentElement.clientHeight) - 12,
          };
        },
      }),
      EditorView.contentAttributes.of({
        "aria-label": props.label ?? "CEL expression",
        "aria-multiline": "true",
        "aria-invalid": props.invalid ? "true" : "false",
      }),
    ],
    [context, props.label, props.invalid],
  );

  const suggest = () => {
    const view = viewRef.current;
    if (!view) return;
    view.focus();
    startCompletion(view);
  };

  return (
    <div
      className={twMerge(
        "mt-2 min-w-0 rounded-xl border border-slate-200 bg-white shadow-card transition-[border-color,box-shadow] focus-within:border-slate-400 focus-within:shadow-[0_0_0_3px_color-mix(in_oklab,var(--color-slate-900)_6%,transparent)]",
        props.invalid &&
          "border-red-500 focus-within:border-red-500 focus-within:shadow-[0_0_0_3px_color-mix(in_oklab,var(--color-red-500)_12%,transparent)]",
      )}
    >
      <CodeMirror
        value={props.exp}
        theme="none"
        extensions={extensions}
        minHeight="80px"
        maxHeight="240px"
        basicSetup={false}
        indentWithTab={false}
        onCreateEditor={(view) => {
          viewRef.current = view;
        }}
        onChange={props.onChange}
      />
      <div className="flex items-center justify-between gap-3 border-t border-slate-100 px-3 py-1.5">
        <button
          type="button"
          aria-label="Show CEL suggestions"
          aria-keyshortcuts="Control+Space"
          title="Show suggestions (Ctrl+Space)"
          onMouseDown={(event) => event.preventDefault()}
          onClick={suggest}
          className="inline-flex items-center gap-1.5 rounded px-1 py-0.5 text-xs font-normal text-slate-500 transition-colors hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <Braces size={13} strokeWidth={1.75} />
          Suggestions
        </button>
        <span
          className="flex items-center gap-1 text-micro text-slate-500"
          aria-hidden
        >
          <kbd className="rounded border border-slate-200 px-1 font-sans">
            Ctrl
          </kbd>
          <kbd className="rounded border border-slate-200 px-1 font-sans">
            Space
          </kbd>
        </span>
      </div>
    </div>
  );
};
