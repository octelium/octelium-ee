import { AccessLog, ComponentLog } from "@/apis/corev1/corev1";
import { AuditLog, AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { json } from "@codemirror/lang-json";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { Button, CopyButton, Drawer } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { AnimatePresence, motion } from "framer-motion";
import { Braces, Check, Copy, FileJson } from "lucide-react";
import * as React from "react";
import { match } from "ts-pattern";

type LogItem = AccessLog | AuditLog | AuthenticationLog | ComponentLog;

const editorTheme = EditorView.theme(
  {
    "&": {
      height: "100%",
      backgroundColor: "#282c34",
      color: "#abb2bf",
    },
    ".cm-content": {
      padding: "14px 0",
      fontSize: "13px",
      fontWeight: "600",
      lineHeight: "1.65",
    },
    ".cm-gutters": {
      borderRight: "1px solid rgba(148,163,184,0.12)",
      backgroundColor: "#282c34",
      color: "#64748b",
      fontSize: "12px",
    },
    ".cm-scroller": {
      minHeight: "100%",
      overflow: "auto",
      backgroundColor: "#282c34",
    },
    ".cm-activeLine": {
      backgroundColor: "rgba(148,163,184,0.06)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "rgba(148,163,184,0.06)",
      color: "#cbd5e1",
    },
  },
  { dark: true },
);

const editorExtensions = [json(), editorTheme, EditorView.lineWrapping];

const serializeLog = (item: LogItem) => {
  try {
    const compact = match(item.kind)
      .with("AccessLog", () => AccessLog.toJsonString(item as AccessLog))
      .with("AuditLog", () => AuditLog.toJsonString(item as AuditLog))
      .with("AuthenticationLog", () =>
        AuthenticationLog.toJsonString(item as AuthenticationLog),
      )
      .with("ComponentLog", () =>
        ComponentLog.toJsonString(item as ComponentLog),
      )
      .otherwise(() => "{}");

    return JSON.stringify(JSON.parse(compact), null, 2);
  } catch {
    return "{}";
  }
};

const Editor = ({ item }: { item: LogItem }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const value = React.useMemo(() => serializeLog(item), [item]);
  const statistics = React.useMemo(
    () => ({
      lines: value.split("\n").length,
      size: new TextEncoder().encode(value).length,
    }),
    [value],
  );

  return (
    <>
      <Button
        type="button"
        size="compact-xs"
        variant="outline"
        leftSection={<Braces size={11} strokeWidth={2.5} />}
        onClick={open}
        styles={{ root: { fontSize: "0.68rem", fontWeight: 700 } }}
      >
        JSON
      </Button>

      <Drawer
        opened={opened}
        onClose={close}
        position="right"
        size="min(900px, 100vw)"
        padding="md"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <FileJson
              size={15}
              className="shrink-0 text-slate-400"
              strokeWidth={2.25}
            />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              JSON
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {item.kind}
            </span>
            <span className="hidden items-center rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-[0.6rem] font-bold uppercase tracking-wider text-slate-500 sm:inline-flex">
              Read only
            </span>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: "16px",
            display: "flex",
            flexDirection: "column",
            backgroundColor: "#f8fafc",
          },
          content: {
            display: "flex",
            flexDirection: "column",
            borderLeft: "1px solid #e2e8f0",
          },
        }}
      >
        <div className="flex min-h-0 flex-1 flex-col gap-3">
          <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white p-2 shadow-sm sm:px-3">
            <span className="text-[0.65rem] font-semibold text-slate-400">
              {statistics.lines.toLocaleString()} lines ·{" "}
              {statistics.size.toLocaleString()} bytes
            </span>

            <CopyButton value={value} timeout={2_000}>
              {({ copied, copy }) => (
                <Button
                  type="button"
                  onClick={copy}
                  variant={copied ? "light" : "default"}
                  color={copied ? "teal" : undefined}
                  size="compact-sm"
                  leftSection={
                    <AnimatePresence initial={false} mode="popLayout">
                      <motion.span
                        key={copied ? "check" : "copy"}
                        initial={{ y: 6, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        exit={{ y: -6, opacity: 0 }}
                        transition={{ duration: 0.15 }}
                        className="flex items-center"
                      >
                        {copied ? (
                          <Check size={12} strokeWidth={2.5} />
                        ) : (
                          <Copy size={12} strokeWidth={2.5} />
                        )}
                      </motion.span>
                    </AnimatePresence>
                  }
                  styles={{ root: { fontSize: "0.7rem", fontWeight: 700 } }}
                >
                  {copied ? "Copied" : "Copy JSON"}
                </Button>
              )}
            </CopyButton>
          </div>

          <div className="min-h-0 flex-1 overflow-hidden rounded-xl border border-slate-700 bg-[#282c34] shadow-sm">
            <CodeMirror
              value={value}
              autoFocus
              readOnly
              theme={oneDark}
              basicSetup={{
                autocompletion: false,
                closeBrackets: false,
                highlightSelectionMatches: false,
              }}
              className="h-full w-full"
              minHeight="calc(100dvh - 160px)"
              maxHeight="calc(100dvh - 160px)"
              extensions={editorExtensions}
            />
          </div>
        </div>
      </Drawer>
    </>
  );
};

export default Editor;
