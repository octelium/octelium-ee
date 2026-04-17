import { AnimatePresence, motion } from "framer-motion";
import { Pencil, X } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

const EditItemWrap = (props: {
  showComponent: React.ReactNode;
  editComponent: React.ReactNode;
  label?: string;
}) => {
  const [enabled, setEnabled] = React.useState(false);

  return (
    <div className="flex items-center gap-1.5 group/edit">
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
            {props.editComponent}
            <button
              onClick={() => setEnabled(false)}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer shrink-0"
              title="Cancel"
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
              onClick={() => setEnabled(true)}
              className={twMerge(
                "flex items-center justify-center w-5 h-5 rounded cursor-pointer shrink-0",
                "text-slate-400 hover:text-slate-600 hover:bg-slate-100",
                "transition-colors duration-150",
                "opacity-0 group-hover/edit:opacity-100",
              )}
              title={props.label ? `Edit ${props.label}` : "Edit"}
            >
              <Pencil size={11} strokeWidth={2.5} />
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default EditItemWrap;
