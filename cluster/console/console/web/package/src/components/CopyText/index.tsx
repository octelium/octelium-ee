import { AnimatePresence, motion } from "framer-motion";
import { CheckCheck, Copy, X } from "lucide-react";
import * as React from "react";
import truncate from "truncate-utf8-bytes";

type CopyState = "idle" | "copied" | "failed";

const writeToClipboard = async (value: string) => {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }

  const area = document.createElement("textarea");
  area.value = value;
  area.setAttribute("readonly", "");
  area.style.position = "fixed";
  area.style.opacity = "0";
  document.body.appendChild(area);
  area.select();
  try {
    if (!document.execCommand("copy")) throw new Error("copy rejected");
  } finally {
    document.body.removeChild(area);
  }
};

const CopyText = (props: {
  value?: string;
  truncate?: number;
  hide?: boolean;
}) => {
  const [state, setState] = React.useState<CopyState>("idle");
  const timer = React.useRef<number | undefined>(undefined);
  const { value, hide } = props;

  React.useEffect(
    () => () => {
      if (timer.current) window.clearTimeout(timer.current);
    },
    [],
  );

  if (!value) return null;

  const displayValue =
    props.truncate && props.truncate < value.length
      ? `${truncate(value, props.truncate)}...`
      : value;

  const flash = (next: CopyState) => {
    setState(next);
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setState("idle"), 1200);
  };

  return (
    <span className="inline-flex items-center gap-1">
      {!hide && <span className="leading-none">{displayValue}</span>}
      <button
        type="button"
        className="inline-flex shrink-0 cursor-pointer items-center justify-center text-slate-500 transition-colors duration-150 hover:text-slate-800"
        aria-label={state === "failed" ? "Copy failed" : "Copy to clipboard"}
        title={state === "failed" ? "Copy failed" : "Copy to clipboard"}
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          writeToClipboard(value).then(
            () => flash("copied"),
            () => flash("failed"),
          );
        }}
      >
        <AnimatePresence initial={false} mode="popLayout">
          <motion.span
            key={state}
            initial={{ y: 6, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -6, opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="inline-flex"
          >
            {state === "copied" ? (
              <CheckCheck
                size={12}
                strokeWidth={2.5}
                className="text-emerald-500"
              />
            ) : state === "failed" ? (
              <X size={12} strokeWidth={2.5} className="text-red-500" />
            ) : (
              <Copy size={12} strokeWidth={2.5} />
            )}
          </motion.span>
        </AnimatePresence>
      </button>
    </span>
  );
};

export default CopyText;
