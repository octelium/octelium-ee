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
import {
  ActionIcon,
  Button,
  Modal,
  ScrollArea,
  SegmentedControl,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import {
  AlertTriangle,
  Braces,
  Check,
  Copy,
  Maximize2,
  Search,
} from "lucide-react";
import { useMemo, useRef, useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { twMerge } from "@/lib/utils";
import { looksComplete } from "./cel";
import {
  CEL_FUNCTIONS,
  schemaAutocomplete,
  schemaFields,
} from "./celCompletion";
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
  Prec.high(
    keymap.of([
      { key: "Tab", run: acceptCompletion },
      { key: "Mod-z", run: undo, shift: redo },
      ...completionKeymap,
    ]),
  ),
  keymap.of([...defaultKeymap, ...historyKeymap]),
];

const floatingTooltips = () =>
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
          top +
          (viewport?.height ?? document.documentElement.clientHeight) -
          12,
      };
    },
  });

const editorExtensions = (args: {
  context: CELContext;
  label: string;
  invalid: boolean;
  hint: string;
  room: boolean;
}) => [
  ...baseExtensions,
  placeholder(args.hint),
  completions[args.context],
  floatingTooltips(),
  ...(args.room ? [roomForActions] : []),
  EditorView.contentAttributes.of({
    "aria-label": args.label,
    "aria-multiline": "true",
    "aria-invalid": args.invalid ? "true" : "false",
  }),
];

const roomForActions = EditorView.theme({
  ".cm-content": { paddingRight: "30px" },
});

const EXAMPLES: Record<CELContext, { title: string; expr: string }[]> = {
  core: [
    { title: "User is in a Group", expr: '"admins" in ctx.user.spec.groups' },
    {
      title: "Read-only HTTP methods",
      expr: 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]',
    },
    {
      title: "HTTP path prefix",
      expr: 'ctx.request.http.path.startsWith("/api/")',
    },
    {
      title: "Business hours in a timezone",
      expr: 'now().getDayOfWeek("Europe/Berlin") in [1, 2, 3, 4, 5] && now().getHours("Europe/Berlin") >= 9 && now().getHours("Europe/Berlin") < 17',
    },
    {
      title: "Source IP range",
      expr: 'net.isIPInRange(ctx.request.ip, "10.0.0.0/8")',
    },
    {
      title: "Multi-factor Session",
      expr: 'ctx.session.status.authentication.info.aal in ["AAL2", "AAL3"]',
    },
    {
      title: "User label",
      expr: 'ctx.user.metadata.labels["team"] == "payments"',
    },
    {
      title: "User email domain",
      expr: 'ctx.user.spec.email.endsWith("@example.com")',
    },
    {
      title: "MCP tool call",
      expr: 'ctx.request.mcp.method == "tools/call" && ctx.request.mcp.name == "delete_repository"',
    },
    {
      title: "Healthy Device",
      expr: 'has(ctx.device.status.posture) && ctx.device.status.posture.threatFree == "PASS"',
    },
    {
      title: "Policy attribute",
      expr: "ctx.request.http.path in attrs.allowedPaths",
    },
    {
      title: "JSON body field",
      expr: 'ctx.request.http.bodyMap.action != "delete"',
    },
  ],
  access: [
    { title: "Requester is in a Group", expr: '"sre" in ctx.user.spec.groups' },
    { title: "Urgent request", expr: 'ctx.request.spec.urgency == "HIGH"' },
    {
      title: "Short access",
      expr: 'ctx.request.spec.duration <= duration("4h")',
    },
  ],
};

const fieldsCache = new Map<CELContext, ReturnType<typeof schemaFields>>();

const fieldsOf = (context: CELContext) => {
  let fields = fieldsCache.get(context);
  if (!fields) {
    fields = schemaFields(celSchemas[context]);
    fieldsCache.set(context, fields);
  }
  return fields;
};

const ReferenceItem = (props: {
  title: string;
  code: string;
  meta?: string;
  description?: string;
  values?: string[];
  onInsert: () => void;
}) => (
  <button
    type="button"
    onMouseDown={(event) => event.preventDefault()}
    onClick={props.onInsert}
    className="flex w-full min-w-0 cursor-pointer flex-col gap-0.5 rounded-lg px-2.5 py-2 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400"
  >
    <span className="flex min-w-0 items-baseline gap-2">
      <span className="min-w-0 truncate font-mono text-xs font-semibold text-slate-800">
        {props.title}
      </span>
      {props.meta && (
        <span className="shrink-0 font-mono text-micro text-slate-400">
          {props.meta}
        </span>
      )}
    </span>
    {props.code !== props.title && (
      <code className="line-clamp-2 break-all font-mono text-micro text-slate-500">
        {props.code}
      </code>
    )}
    {props.description && (
      <span className="line-clamp-2 text-micro leading-4 text-slate-500">
        {props.description}
      </span>
    )}
    {props.values && props.values.length > 0 && (
      <span className="mt-0.5 flex flex-wrap gap-1">
        {props.values.slice(0, 6).map((value) => (
          <span
            key={value}
            className="rounded bg-slate-100 px-1 font-mono text-micro text-slate-600"
          >
            {value}
          </span>
        ))}
      </span>
    )}
  </button>
);

const Reference = (props: {
  context: CELContext;
  onInsert: (text: string) => void;
}) => {
  const [tab, setTab] = useState<"fields" | "functions" | "examples">("fields");
  const [query, setQuery] = useState("");
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  const hit = (...parts: (string | undefined)[]) => {
    const text = parts.filter(Boolean).join(" ").toLowerCase();
    return words.every((word) => text.includes(word));
  };

  const items =
    tab === "fields"
      ? fieldsOf(props.context)
          .filter((field) => hit(field.path, field.description))
          .slice(0, 150)
          .map((field) => (
            <ReferenceItem
              key={field.path}
              title={field.path}
              code={field.path}
              meta={field.type}
              description={field.description}
              values={field.values}
              onInsert={() => props.onInsert(field.path)}
            />
          ))
      : tab === "functions"
        ? CEL_FUNCTIONS.filter((item) => hit(item.name, item.description)).map(
            (item) => (
              <ReferenceItem
                key={item.name}
                title={item.name}
                code={item.signature}
                description={item.description}
                onInsert={() => props.onInsert(item.insert)}
              />
            ),
          )
        : EXAMPLES[props.context]
            .filter((item) => hit(item.title, item.expr))
            .map((item) => (
              <ReferenceItem
                key={item.expr}
                title={item.title}
                code={item.expr}
                onInsert={() => props.onInsert(item.expr)}
              />
            ));

  return (
    <aside className="flex min-h-0 flex-col border-t border-slate-200 bg-slate-50/60 lg:border-l lg:border-t-0">
      <div className="flex shrink-0 flex-col gap-2 border-b border-slate-200 p-3">
        <SegmentedControl
          size="xs"
          fullWidth
          value={tab}
          onChange={(value) => setTab(value as typeof tab)}
          data={[
            { value: "fields", label: "Fields" },
            { value: "functions", label: "Functions" },
            { value: "examples", label: "Examples" },
          ]}
        />
        <TextInput
          size="xs"
          aria-label="Search the reference"
          placeholder={
            tab === "fields" ? "Search fields, e.g. email" : "Search"
          }
          leftSection={<Search size={12} />}
          value={query}
          onChange={(event) => setQuery(event.currentTarget.value)}
        />
      </div>
      <ScrollArea className="min-h-0 flex-1" type="auto">
        <div className="flex flex-col p-1.5">
          {items.length > 0 ? (
            items
          ) : (
            <p className="px-3 py-6 text-center text-xs text-slate-500">
              Nothing matches.
            </p>
          )}
        </div>
      </ScrollArea>
      <p className="shrink-0 border-t border-slate-200 px-3 py-2 text-micro leading-4 text-slate-500">
        Click an item to insert it at the cursor.
      </p>
    </aside>
  );
};

const SyntaxStatus = (props: { exp: string }) => {
  const empty = props.exp.trim() === "";
  const incomplete = !empty && !looksComplete(props.exp);
  return (
    <span aria-live="polite" className="flex min-w-0 items-center gap-1">
      {incomplete && (
        <span className="inline-flex items-center gap-1 truncate text-micro font-semibold text-amber-700">
          <AlertTriangle size={11} className="shrink-0" />
          Unfinished expression
        </span>
      )}
    </span>
  );
};

const ExpandedEditor = (props: {
  opened: boolean;
  exp: string;
  label: string;
  context: CELContext;
  extensions: Extension[];
  onClose: () => void;
  onChange: (value: string) => void;
}) => {
  const viewRef = useRef<EditorView | null>(null);
  const mobile = useMediaQuery("(max-width: 40em)");
  const insert = (text: string) => {
    const view = viewRef.current;
    if (!view) return;
    view.focus();
    view.dispatch(view.state.replaceSelection(text));
  };
  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      fullScreen={!!mobile}
      size="min(1180px, calc(100vw - 32px))"
      padding={0}
      radius={mobile ? 0 : "lg"}
      withCloseButton={false}
      styles={{ body: { padding: 0 } }}
    >
      <div className="flex h-[100dvh] flex-col sm:h-[min(760px,calc(100dvh-64px))]">
        <header className="flex shrink-0 items-center gap-3 border-b border-slate-200 px-4 py-3 sm:px-5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-600">
            <Braces size={15} />
          </span>
          <div className="min-w-0 flex-1">
            <Modal.Title className="truncate text-sm font-bold tracking-tight text-slate-900">
              {props.label}
            </Modal.Title>
            <p className="hidden text-xs text-slate-500 sm:block">
              Common Expression Language · press Ctrl+Space for suggestions
            </p>
          </div>
          <Button
            size="compact-sm"
            color="ink"
            leftSection={<Check size={13} />}
            onClick={props.onClose}
          >
            Done
          </Button>
        </header>
        <div className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_minmax(0,40%)] lg:grid-cols-[minmax(0,1fr)_360px] lg:grid-rows-1">
          <div className="flex min-h-0 flex-col p-3 sm:p-4">
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card focus-within:border-slate-400">
              <CodeMirror
                className="min-h-0 flex-1 overflow-auto"
                value={props.exp}
                theme="none"
                extensions={props.extensions}
                minHeight="240px"
                basicSetup={false}
                indentWithTab={false}
                autoFocus
                onCreateEditor={(view) => {
                  viewRef.current = view;
                }}
                onChange={props.onChange}
              />
              <div className="flex items-center justify-between gap-3 border-t border-slate-100 px-3 py-1.5">
                <SyntaxStatus exp={props.exp} />
                <span className="text-micro tabular-nums text-slate-400">
                  {props.exp.length} characters
                </span>
              </div>
            </div>
          </div>
          <Reference context={props.context} onInsert={insert} />
        </div>
      </div>
    </Modal>
  );
};

export const CELEditor = (props: {
  exp: string;
  label?: string;
  invalid?: boolean;
  context?: CELContext;
  placeholder?: string;
  expandable?: boolean;
  onChange: (value: string) => void;
}) => {
  const viewRef = useRef<EditorView | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const context = props.context ?? "core";
  const label = props.label ?? "CEL expression";
  const expandable = props.expandable ?? true;

  const invalid = !!props.invalid;
  const hint =
    props.placeholder ?? "Write an expression, or type ctx. to browse fields";

  const inlineExtensions = useMemo(
    () => editorExtensions({ context, label, invalid, hint, room: expandable }),
    [context, label, invalid, hint, expandable],
  );
  const largeExtensions = useMemo(
    () =>
      editorExtensions({
        context,
        label: `${label} (large editor)`,
        invalid,
        hint,
        room: false,
      }),
    [context, label, invalid, hint],
  );

  const suggest = () => {
    const view = viewRef.current;
    if (!view) return;
    view.focus();
    startCompletion(view);
  };

  const copy = async () => {
    await navigator.clipboard?.writeText(props.exp);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  return (
    <div
      className={twMerge(
        "group/cel relative mt-2 min-w-0 rounded-xl border border-slate-200 bg-white shadow-card transition-[border-color,box-shadow] focus-within:border-slate-400 focus-within:shadow-[0_0_0_3px_color-mix(in_oklab,var(--color-slate-900)_6%,transparent)]",
        props.invalid &&
          "border-red-500 focus-within:border-red-500 focus-within:shadow-[0_0_0_3px_color-mix(in_oklab,var(--color-red-500)_12%,transparent)]",
      )}
    >
      {expandable && (
        <div className="absolute right-1 top-1 z-[1] flex items-center gap-0.5 rounded-md bg-white/90 opacity-0 transition-opacity duration-150 focus-within:opacity-100 group-hover/cel:opacity-100 [@media(hover:none)]:opacity-100">
          {props.exp && (
            <Tooltip
              label={copied ? "Copied" : "Copy"}
              withArrow
              openDelay={300}
            >
              <ActionIcon
                size="sm"
                variant="subtle"
                color="gray"
                aria-label="Copy the CEL expression"
                onClick={copy}
              >
                {copied ? (
                  <Check size={12} className="text-emerald-600" />
                ) : (
                  <Copy size={12} />
                )}
              </ActionIcon>
            </Tooltip>
          )}
          <Tooltip
            label="Open the large editor with a field reference"
            withArrow
            openDelay={300}
          >
            <ActionIcon
              size="sm"
              variant="subtle"
              color="gray"
              aria-label="Open the large CEL editor"
              onClick={() => setExpanded(true)}
            >
              <Maximize2 size={12} />
            </ActionIcon>
          </Tooltip>
        </div>
      )}
      <CodeMirror
        value={props.exp}
        theme="none"
        extensions={inlineExtensions}
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
        <div className="flex min-w-0 items-center gap-3">
          <button
            type="button"
            aria-label="Show CEL suggestions"
            aria-keyshortcuts="Control+Space"
            title="Show suggestions (Ctrl+Space)"
            onMouseDown={(event) => event.preventDefault()}
            onClick={suggest}
            className="inline-flex shrink-0 items-center gap-1.5 rounded px-1 py-0.5 text-xs font-normal text-slate-500 transition-colors hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            <Braces size={13} strokeWidth={1.75} />
            Suggestions
          </button>
          <SyntaxStatus exp={props.exp} />
        </div>
        <span
          className="hidden shrink-0 items-center gap-1 text-micro text-slate-500 sm:flex"
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
      {expandable && expanded && (
        <ExpandedEditor
          opened={expanded}
          exp={props.exp}
          label={label}
          context={context}
          extensions={largeExtensions}
          onClose={() => setExpanded(false)}
          onChange={props.onChange}
        />
      )}
    </div>
  );
};
