import { Collapse } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { twJoin, twMerge } from "tailwind-merge";

interface Props {
  children?: React.ReactNode;
  title?: string;
  description?: string;
  obj?: object | Array<any> | undefined;
  onSet?: () => void;
  onUnset: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
  noDelete?: boolean;
}

const EditItem = (props: Props) => {
  const arrLen = props.isList ? (props.obj as Array<any>).length : 0;
  const isExpanded = props.isList ? arrLen > 0 : props.obj !== undefined;

  return (
    <div
      className={twJoin(
        "mt-5 pl-3 border-l-[3px] transition-[border-color] duration-150",
        isExpanded ? "border-l-slate-800" : "border-l-slate-300",
        !isExpanded && "hover:border-l-slate-500",
      )}
    >
      <div className="w-full flex items-center gap-2 min-h-[28px]">
        <div
          className={twJoin(
            "flex-1 flex items-center gap-2 min-w-0",
            !isExpanded && "cursor-pointer group",
          )}
          onClick={() => {
            if (!isExpanded && props.onSet) props.onSet();
          }}
        >
          <div className="flex items-center gap-2 min-w-0 flex-wrap">
            {props.title && (
              <span
                className={twMerge(
                  "text-[0.78rem] font-bold uppercase tracking-[0.05em] transition-colors duration-150 shrink-0",
                  isExpanded
                    ? "text-slate-800"
                    : "text-slate-500 group-hover:text-slate-700",
                )}
              >
                {props.title}
              </span>
            )}

            {props.description && (
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold transition-colors duration-150 truncate",
                  isExpanded
                    ? "text-slate-500"
                    : "text-slate-400 group-hover:text-slate-500",
                )}
              >
                {props.description}
              </span>
            )}

            {!isExpanded && props.onSet && (
              <span className="text-[0.68rem] font-bold text-slate-400 group-hover:text-slate-600 transition-colors duration-150">
                Click to set
              </span>
            )}
          </div>

          {props.isList && props.onAddListItem && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                props.onAddListItem!();
              }}
              className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shrink-0 shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
            >
              <Plus size={11} strokeWidth={2.5} />
              Add item
              <span className="text-slate-400">({arrLen})</span>
            </button>
          )}
        </div>

        {!props.noDelete && (
          <AnimatePresence initial={false}>
            {isExpanded && (
              <motion.button
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.15 }}
                onClick={() => props.onUnset()}
                className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-red-500 hover:bg-red-50 transition-colors duration-150 cursor-pointer shrink-0"
                title="Remove"
              >
                <Trash2 size={13} strokeWidth={2} />
              </motion.button>
            )}
          </AnimatePresence>
        )}
      </div>

      <Collapse in={isExpanded} transitionDuration={180}>
        <div className="mt-3 mb-1">
          <div className="ml-1">{props.children}</div>

          {props.isList && props.onAddListItem && (
            <div className="flex items-center mt-4 mb-1">
              <button
                onClick={() => props.onAddListItem!()}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.72rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
              >
                <Plus size={12} strokeWidth={2.5} />
                Add another item
                <span className="text-slate-400">({arrLen})</span>
              </button>
            </div>
          )}
        </div>
      </Collapse>
    </div>
  );
};

export default EditItem;
