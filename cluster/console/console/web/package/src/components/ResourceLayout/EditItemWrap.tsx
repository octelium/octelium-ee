import { AnimatePresence, motion } from "framer-motion";
import { Check, LoaderCircle, Pencil, X } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

const EditItemWrap = (props: {
  showComponent: React.ReactNode;
  editComponent: React.ReactNode;
  label?: string;
  mutation?: {
    isPending: boolean;
    isError: boolean;
    isSuccess: boolean;
    error: unknown;
    reset: () => void;
  };
}) => {
  const [enabled, setEnabled] = React.useState(false);
  const errorMessage =
    props.mutation?.error instanceof Error
      ? props.mutation.error.message
      : "The update could not be saved";

  return (
    <div className="flex flex-col items-start gap-1 group/edit">
      <AnimatePresence mode="wait" initial={false}>
        {enabled ? (
          <motion.div
            key="edit"
            initial={{ opacity: 0, x: -4 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -4 }}
            transition={{ duration: 0.15 }}
            className="flex items-center gap-1.5"
          >
            <div
              className={twMerge(
                "flex items-center",
                props.mutation?.isPending &&
                  "pointer-events-none opacity-60",
              )}
              aria-busy={props.mutation?.isPending}
            >
              {props.editComponent}
            </div>
            <button
              type="button"
              disabled={props.mutation?.isPending}
              onClick={() => {
                props.mutation?.reset();
                setEnabled(false);
              }}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 disabled:cursor-not-allowed disabled:opacity-50"
              title="Cancel"
              aria-label="Close editor"
            >
              <X size={12} strokeWidth={2.5} />
            </button>
          </motion.div>
        ) : (
          <motion.div
            key="show"
            initial={{ opacity: 0, x: 4 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 4 }}
            transition={{ duration: 0.15 }}
            className="flex items-center gap-1.5"
          >
            {props.showComponent}
            <button
              type="button"
              onClick={() => setEnabled(true)}
              className={twMerge(
                "flex items-center justify-center w-5 h-5 rounded cursor-pointer shrink-0",
                "text-slate-400 hover:text-slate-600 hover:bg-slate-100",
                "transition-colors duration-150",
                "opacity-0 group-hover/edit:opacity-100 group-focus-within/edit:opacity-100 focus-visible:opacity-100",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500",
              )}
              title={props.label ? `Edit ${props.label}` : "Edit"}
              aria-label={props.label ? `Edit ${props.label}` : "Edit"}
            >
              <Pencil size={11} strokeWidth={2.5} />
            </button>
          </motion.div>
        )}
      </AnimatePresence>
      {enabled && props.mutation?.isPending && (
        <span className="flex items-center gap-1 text-[0.68rem] font-semibold text-slate-500" role="status">
          <LoaderCircle size={11} className="animate-spin" /> Saving…
        </span>
      )}
      {enabled && props.mutation?.isError && (
        <span className="text-[0.68rem] font-semibold text-red-600" role="alert">
          {errorMessage}
        </span>
      )}
      {enabled && props.mutation?.isSuccess && !props.mutation.isPending && (
        <span className="flex items-center gap-1 text-[0.68rem] font-semibold text-emerald-600" role="status">
          <Check size={11} /> Saved
        </span>
      )}
    </div>
  );
};

export default EditItemWrap;
