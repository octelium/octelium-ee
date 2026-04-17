import { Collapse } from "@mantine/core";
import { motion } from "framer-motion";
import { ChevronDown, Plus } from "lucide-react";
import * as React from "react";
import { twJoin } from "tailwind-merge";

interface Props {
  children?: React.ReactNode;
  title: string;
  obj?: object | Array<any> | undefined;
  onSet?: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
}

const ItemMessage = (props: Props) => {
  const arr = props.obj as Array<any> | undefined;
  const arrLen = props.isList ? (arr?.length ?? 0) : 0;
  const isObjNull = !props.obj;

  const [isExpanded, setIsExpanded] = React.useState(arrLen > 0);

  const handleAddItem = (e: React.MouseEvent) => {
    e.stopPropagation();
    props.onAddListItem?.();
    setIsExpanded(true);
  };

  const handleHeaderClick = () => {
    if (isObjNull) {
      props.onSet?.();
      setIsExpanded(true);
    } else {
      setIsExpanded((v) => !v);
    }
    if (props.isList && arrLen < 1) {
      props.onAddListItem?.();
    }
  };

  const itemLabel = arrLen === 1 ? "1 item" : `${arrLen} items`;

  return (
    <div className="mt-5 mb-10">
      <button
        className={twJoin(
          "w-full flex items-center gap-2 cursor-pointer",
          "pb-2 border-b border-slate-200",
          "transition-[border-color] duration-150",
          "hover:border-slate-300 group",
        )}
        onClick={handleHeaderClick}
      >
        <motion.span
          animate={{ rotate: isExpanded ? 180 : 0 }}
          transition={{ duration: 0.2, ease: "easeInOut" }}
          className="flex items-center shrink-0 text-slate-400 group-hover:text-slate-600 transition-colors duration-150"
        >
          <ChevronDown size={14} strokeWidth={2.5} />
        </motion.span>

        <span
          className={twJoin(
            "text-[0.78rem] font-bold uppercase tracking-[0.05em] transition-colors duration-150 flex-1 text-left",
            isExpanded
              ? "text-slate-800"
              : "text-slate-500 group-hover:text-slate-700",
          )}
        >
          {props.title}
        </span>

        {props.isList && (
          <span className="text-[0.68rem] font-semibold text-slate-400 shrink-0">
            {itemLabel}
          </span>
        )}

        {props.isList && (
          <button
            onClick={handleAddItem}
            className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
          >
            <Plus size={11} strokeWidth={2.5} />
            Add
          </button>
        )}
      </button>

      {props.obj && (
        <Collapse in={isExpanded} transitionDuration={180}>
          <div className="ml-3 mt-4 mb-2">{props.children}</div>

          {props.isList && (
            <div className="mt-4 mb-1">
              <button
                onClick={handleAddItem}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.72rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
              >
                <Plus size={12} strokeWidth={2.5} />
                Add another item
                <span className="text-slate-400">({itemLabel})</span>
              </button>
            </div>
          )}
        </Collapse>
      )}
    </div>
  );
};

export default ItemMessage;
