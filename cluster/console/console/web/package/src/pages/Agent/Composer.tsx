import type { FileInfo } from "@/apis/consoleagent/protocol";
import { ActionIcon, Loader, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowUp, Paperclip, Square, X } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { fileIcon } from "./Blocks";
import { Kbd } from "./ui";
import { formatBytes } from "./utils";

export interface Upload {
  key: string;
  name: string;
}

const isCoarsePointer = () =>
  typeof window !== "undefined" &&
  window.matchMedia?.("(pointer: coarse)").matches;

const chipMotion = {
  initial: { opacity: 0, scale: 0.92 },
  animate: { opacity: 1, scale: 1 },
  exit: { opacity: 0, scale: 0.92 },
  transition: { duration: 0.16 },
};

const AttachmentPreview = (props: {
  file: FileInfo;
  preview?: string;
  onRemove: () => void;
}) => {
  const { file } = props;
  const Icon = fileIcon(file.mimeType, file.name);

  const remove = (
    <button
      type="button"
      aria-label={`Remove ${file.name}`}
      onClick={props.onRemove}
      className="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center rounded-full bg-slate-900 text-white opacity-90 shadow-sm transition-opacity hover:opacity-100"
    >
      <X size={11} strokeWidth={3} />
    </button>
  );

  if (props.preview) {
    return (
      <motion.div {...chipMotion} layout className="relative shrink-0">
        <Tooltip label={`${file.name} · ${formatBytes(file.size)}`} withArrow>
          <img
            src={props.preview}
            alt={file.name}
            className="h-14 w-14 rounded-xl border border-slate-200 object-cover"
          />
        </Tooltip>
        <span className="absolute -right-1.5 -top-1.5">{remove}</span>
      </motion.div>
    );
  }

  return (
    <motion.div
      {...chipMotion}
      layout
      className="relative flex h-14 w-[200px] shrink-0 items-center gap-2.5 rounded-xl border border-slate-200 bg-slate-50 pl-2 pr-3"
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500">
        <Icon size={15} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-xs font-semibold text-slate-700">
          {file.name}
        </span>
        <span className="block text-micro text-slate-400">
          {formatBytes(file.size)}
        </span>
      </span>
      <span className="absolute -right-1.5 -top-1.5">{remove}</span>
    </motion.div>
  );
};

const Composer = (props: {
  value: string;
  onChange: (value: string) => void;
  attachments: FileInfo[];
  uploads: Upload[];
  previews: Record<string, string>;
  onAddFiles: (files: File[]) => void;
  onRemoveAttachment: (id: string) => void;
  onSend: () => void;
  onStop: () => void;
  isActive: boolean;
  sending: boolean;
  placeholder: string;
  modelPicker: React.ReactNode;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  className?: string;
}) => {
  const fileRef = React.useRef<HTMLInputElement>(null);
  const { textareaRef } = props;
  const [focused, setFocused] = React.useState(false);

  React.useLayoutEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = `${el.scrollHeight}px`;
  }, [props.value, textareaRef]);

  React.useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    let width = el.clientWidth;
    const observer = new ResizeObserver(() => {
      if (el.clientWidth === width) return;
      width = el.clientWidth;
      el.style.height = "0px";
      el.style.height = `${el.scrollHeight}px`;
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [textareaRef]);

  const canSend =
    !props.sending &&
    props.uploads.length === 0 &&
    (props.value.trim() !== "" || props.attachments.length > 0);

  const hasFiles = props.attachments.length > 0 || props.uploads.length > 0;

  return (
    <div
      className={`text-base leading-6 sm:text-body ${twMerge(
        "relative flex cursor-text flex-col rounded-2xl border border-slate-200 bg-white shadow-raised transition-[border-color,box-shadow] duration-200",
        focused &&
          "border-slate-300 shadow-[0_0_0_4px_color-mix(in_oklab,var(--color-slate-900)_6%,transparent),var(--shadow-raised)]",
        props.className,
      )}`}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          event.preventDefault();
          textareaRef.current?.focus();
        }
      }}
    >
      <AnimatePresence initial={false}>
        {hasFiles && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden"
          >
            <div className="flex gap-2.5 overflow-x-auto px-3 pb-1 pt-3.5">
              <AnimatePresence initial={false}>
                {props.attachments.map((file) => (
                  <AttachmentPreview
                    key={file.id}
                    file={file}
                    preview={props.previews[file.id]}
                    onRemove={() => props.onRemoveAttachment(file.id)}
                  />
                ))}
                {props.uploads.map((upload) => (
                  <motion.div
                    key={upload.key}
                    {...chipMotion}
                    layout
                    className="flex h-14 w-[200px] shrink-0 items-center gap-2.5 rounded-xl border border-dashed border-slate-300 bg-white pl-2 pr-3"
                  >
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-50">
                      <Loader size={14} color="gray" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs font-semibold text-slate-600">
                        {upload.name}
                      </span>
                      <span className="block text-micro text-slate-400">
                        Uploading…
                      </span>
                    </span>
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <textarea
        ref={textareaRef}
        rows={1}
        aria-label="Message"
        placeholder={props.placeholder}
        value={props.value}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onChange={(event) => props.onChange(event.currentTarget.value)}
        onPaste={(event) => {
          const files = Array.from(event.clipboardData.files);
          if (files.length > 0) {
            event.preventDefault();
            props.onAddFiles(files);
          }
        }}
        onKeyDown={(event) => {
          if (
            event.key === "Enter" &&
            !event.shiftKey &&
            !event.altKey &&
            !event.nativeEvent.isComposing &&
            !isCoarsePointer()
          ) {
            event.preventDefault();
            if (canSend && !props.isActive) props.onSend();
          }
        }}
        className="block max-h-[min(40dvh,320px)] min-h-[52px] w-full resize-none overflow-y-auto bg-transparent px-4 pb-1 pt-3.5 text-slate-900 outline-none placeholder:text-slate-400"
      />

      <div className="flex items-center gap-1 px-2 pb-2 pt-1">
        <input
          ref={fileRef}
          type="file"
          multiple
          hidden
          onChange={(event) => {
            props.onAddFiles(Array.from(event.currentTarget.files ?? []));
            event.currentTarget.value = "";
          }}
        />
        <Tooltip label="Attach files" withArrow>
          <ActionIcon
            size={32}
            radius="xl"
            variant="subtle"
            color="gray"
            aria-label="Attach files"
            onClick={() => fileRef.current?.click()}
          >
            <Paperclip size={15} />
          </ActionIcon>
        </Tooltip>
        <div className="min-w-0">{props.modelPicker}</div>
        <div className="ml-auto flex items-center gap-2">
          <span
            className={`text-micro ${twMerge(
              "hidden items-center gap-1 text-slate-400 transition-opacity duration-200 md:flex",
              focused && props.value.trim() !== "" && !props.isActive
                ? "opacity-100"
                : "opacity-0",
            )}`}
            aria-hidden
          >
            <Kbd>↵</Kbd> send
            <Kbd className="ml-1">⇧ ↵</Kbd> new line
          </span>
          <Tooltip label={props.isActive ? "Stop the agent" : "Send"} withArrow>
            <button
              type="button"
              aria-label={props.isActive ? "Stop" : "Send"}
              disabled={!props.isActive && !canSend}
              onClick={props.isActive ? props.onStop : props.onSend}
              className={twMerge(
                "relative flex h-8 w-8 shrink-0 cursor-pointer items-center justify-center rounded-full transition-[background-color,color,transform] duration-150 active:scale-95",
                props.isActive || canSend
                  ? "bg-slate-900 text-white hover:bg-slate-700"
                  : "cursor-not-allowed bg-slate-100 text-slate-400",
              )}
            >
              <AnimatePresence initial={false} mode="popLayout">
                <motion.span
                  key={
                    props.sending ? "sending" : props.isActive ? "stop" : "send"
                  }
                  initial={{ opacity: 0, scale: 0.6 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.6 }}
                  transition={{ duration: 0.15 }}
                  className="flex items-center justify-center"
                >
                  {props.sending ? (
                    <Loader size={13} color="currentColor" />
                  ) : props.isActive ? (
                    <Square size={11} fill="currentColor" strokeWidth={0} />
                  ) : (
                    <ArrowUp size={16} strokeWidth={2.5} />
                  )}
                </motion.span>
              </AnimatePresence>
            </button>
          </Tooltip>
        </div>
      </div>
    </div>
  );
};

export default Composer;
