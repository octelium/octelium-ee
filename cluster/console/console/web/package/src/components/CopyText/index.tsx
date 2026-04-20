import { AnimatePresence, motion } from "framer-motion";
import { CheckCheck, Copy } from "lucide-react";
import { useState } from "react";
import truncate from "truncate-utf8-bytes";

const CopyText = (props: {
  value?: string;
  truncate?: number;
  hide?: boolean;
}) => {
  const [copied, setCopied] = useState(false);
  const { value, hide } = props;

  if (!value) return null;

  const displayValue =
    props.truncate && props.truncate < value.length
      ? `${truncate(value, props.truncate)}...`
      : value;

  return (
    <span className="inline-flex items-center gap-1">
      {!hide && <span className="leading-none">{displayValue}</span>}
      <button
        className="inline-flex items-center justify-center text-slate-400 hover:text-slate-800 transition-colors duration-150 cursor-pointer shrink-0"
        aria-label="Copy to clipboard"
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          navigator.clipboard.writeText(value);
          setCopied(true);
          setTimeout(() => setCopied(false), 1000);
        }}
      >
        <AnimatePresence initial={false} mode="popLayout">
          <motion.span
            key={copied ? "copied" : "copy"}
            initial={{ y: 6, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -6, opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="inline-flex"
          >
            {copied ? (
              <CheckCheck
                size={12}
                strokeWidth={2.5}
                className="text-emerald-500"
              />
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
