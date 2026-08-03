import { Button, Collapse } from "@mantine/core";
import { motion } from "framer-motion";
import { ChevronDown, Plus } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

interface Props {
  children?: React.ReactNode;
  title: string;
  obj?: object | Array<any>;
  onSet?: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
}

const ItemMessage = (props: Props) => {
  const arrLen = props.isList && Array.isArray(props.obj) ? props.obj.length : 0;
  const hasValue = props.isList ? arrLen > 0 : props.obj != null;
  const [isExpanded, setIsExpanded] = React.useState(hasValue);

  React.useEffect(() => {
    if (!hasValue) setIsExpanded(false);
  }, [hasValue]);

  const handleAddItem = () => {
    props.onAddListItem?.();
    setIsExpanded(true);
  };

  const handleHeaderClick = () => {
    if (hasValue) {
      setIsExpanded((value) => !value);
      return;
    }

    if (props.isList) {
      if (props.onSet) props.onSet();
      else props.onAddListItem?.();
    } else {
      props.onSet?.();
    }
    setIsExpanded(true);
  };

  return (
    <section className="mb-7 mt-4 bg-transparent">
      <div
        className={twMerge(
          "flex min-h-10 items-center gap-2 border-b transition-colors duration-500",
          isExpanded ? "border-slate-300" : "border-slate-200",
        )}
      >
        <button
          type="button"
          onClick={handleHeaderClick}
          aria-expanded={isExpanded}
          className="group flex min-w-0 flex-1 items-center gap-2.5 rounded-md py-2 text-left outline-none focus-visible:ring-2 focus-visible:ring-blue-500/30"
        >
          <motion.span
            animate={{ rotate: isExpanded ? 180 : 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
            className={twMerge(
              "flex h-6 w-6 shrink-0 items-center justify-center rounded-md border bg-white transition-[border-color,color,box-shadow] duration-500",
              isExpanded
                ? "border-slate-300 text-slate-700 shadow-[0_1px_3px_rgba(15,23,42,0.06)]"
                : "border-slate-200 text-slate-400 group-hover:border-slate-300 group-hover:text-slate-600",
            )}
          >
            <ChevronDown size={13} strokeWidth={2.4} />
          </motion.span>

          <span
            className={twMerge(
              "truncate text-[0.77rem] font-bold tracking-[0.015em] transition-colors duration-500",
              isExpanded
                ? "text-slate-800"
                : "text-slate-600 group-hover:text-slate-800",
            )}
          >
            {props.title}
          </span>

          {props.isList && (
            <span className="shrink-0 rounded-full border border-slate-200 bg-white px-1.5 py-px text-[0.61rem] font-bold text-slate-500">
              {arrLen} {arrLen === 1 ? "item" : "items"}
            </span>
          )}

          {!hasValue && (props.onSet || props.onAddListItem) && (
            <span className="hidden text-[0.65rem] font-semibold text-slate-400 sm:inline">
              Click to configure
            </span>
          )}
        </button>

        {props.isList && props.onAddListItem && (
          <Button
            type="button"
            variant="default"
            size="compact-xs"
            leftSection={<Plus size={11} strokeWidth={2.5} />}
            onClick={handleAddItem}
          >
            Add
          </Button>
        )}
      </div>

      {hasValue && (
        <Collapse expanded={isExpanded} transitionDuration={250}>
          <div className="ml-3 mt-3 border-l border-slate-200 pl-4">
            {props.children}

            {props.isList && props.onAddListItem && (
              <div className="mt-4 border-t border-slate-100 pt-3">
                <Button
                  type="button"
                  variant="subtle"
                  color="gray"
                  size="compact-xs"
                  leftSection={<Plus size={11} strokeWidth={2.5} />}
                  onClick={handleAddItem}
                >
                  Add another item
                </Button>
              </div>
            )}
          </div>
        </Collapse>
      )}
    </section>
  );
};

export default ItemMessage;
