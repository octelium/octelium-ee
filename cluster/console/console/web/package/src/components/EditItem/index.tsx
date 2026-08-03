import { ActionIcon, Button, Collapse, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { ChevronRight, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

interface Props {
  children?: React.ReactNode;
  title?: string;
  description?: string;
  obj?: object | Array<any>;
  onSet?: () => void;
  onUnset: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
  noDelete?: boolean;
}

const EditItem = (props: Props) => {
  const arrLen = props.isList && Array.isArray(props.obj) ? props.obj.length : 0;
  const isExpanded = props.isList ? arrLen > 0 : props.obj !== undefined;
  const canSet = !isExpanded && !!props.onSet;

  const heading = (
    <div className="flex min-w-0 flex-1 items-center gap-2.5 text-left">
      <span
        className={twMerge(
          "flex h-6 w-6 shrink-0 items-center justify-center rounded-md border transition-[background-color,border-color,color] duration-500",
          isExpanded
            ? "border-slate-700 bg-slate-800 text-white"
            : "border-slate-200 bg-white text-slate-400 group-hover:border-slate-300 group-hover:text-slate-600",
        )}
      >
        {isExpanded ? (
          <span className="h-1.5 w-1.5 rounded-full bg-white" />
        ) : (
          <ChevronRight size={13} strokeWidth={2.4} />
        )}
      </span>

      <span className="flex min-w-0 flex-1 flex-col">
        <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
          {props.title && (
            <span
              className={twMerge(
                "text-[0.76rem] font-bold tracking-[0.015em] transition-colors duration-500",
                isExpanded
                  ? "text-slate-800"
                  : "text-slate-600 group-hover:text-slate-800",
              )}
            >
              {props.title}
            </span>
          )}
          {props.isList && isExpanded && (
            <span className="rounded-full border border-slate-200 bg-white px-1.5 py-px text-[0.61rem] font-bold text-slate-500">
              {arrLen} {arrLen === 1 ? "item" : "items"}
            </span>
          )}
          {canSet && (
            <span className="text-[0.65rem] font-semibold text-slate-400">
              Click to configure
            </span>
          )}
        </span>
        {props.description && (
          <span className="mt-0.5 truncate text-[0.69rem] font-medium text-slate-400">
            {props.description}
          </span>
        )}
      </span>
    </div>
  );

  return (
    <section
      className={twMerge(
        "mt-4 overflow-hidden rounded-xl border bg-transparent transition-[border-color,box-shadow] duration-500",
        isExpanded
          ? "border-slate-200 shadow-[0_1px_4px_rgba(15,23,42,0.04)]"
          : "border-slate-200/80 hover:border-slate-300",
      )}
    >
      <div className="flex min-h-11 items-center gap-2 px-3 py-2">
        {canSet ? (
          <button
            type="button"
            onClick={props.onSet}
            className="group flex min-w-0 flex-1 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-blue-500/30"
          >
            {heading}
          </button>
        ) : (
          <div className="group flex min-w-0 flex-1">{heading}</div>
        )}

        {props.isList && props.onAddListItem && (
          <Button
            type="button"
            variant="default"
            size="compact-xs"
            leftSection={<Plus size={11} strokeWidth={2.5} />}
            onClick={props.onAddListItem}
          >
            Add item
          </Button>
        )}

        {!props.noDelete && (
          <AnimatePresence initial={false}>
            {isExpanded && (
              <motion.div
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                transition={{ duration: 0.18 }}
              >
                <Tooltip label="Remove section" withArrow>
                  <ActionIcon
                    type="button"
                    variant="subtle"
                    color="red"
                    size="sm"
                    aria-label={`Remove ${props.title || "section"}`}
                    onClick={props.onUnset}
                  >
                    <Trash2 size={13} strokeWidth={2.1} />
                  </ActionIcon>
                </Tooltip>
              </motion.div>
            )}
          </AnimatePresence>
        )}
      </div>

      <Collapse expanded={isExpanded} transitionDuration={250}>
        <div className="border-t border-slate-100 px-3 pb-3 pt-3">
          {props.children}

          {props.isList && props.onAddListItem && (
            <div className="mt-3 flex items-center border-t border-slate-100 pt-3">
              <Button
                type="button"
                variant="subtle"
                color="gray"
                size="compact-xs"
                leftSection={<Plus size={11} strokeWidth={2.5} />}
                onClick={props.onAddListItem}
              >
                Add another item
              </Button>
            </div>
          )}
        </div>
      </Collapse>
    </section>
  );
};

export default EditItem;
