import { ActionIcon, Button, Collapse, Tooltip } from "@mantine/core";
import { ChevronRight, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

const DepthContext = React.createContext(0);

export type SectionLevel = 1 | 2 | 3;

export interface SectionProps {
  children?: React.ReactNode;
  title?: string;
  description?: string;
  level?: SectionLevel;
  obj?: object | Array<any>;
  onSet?: () => void;
  onUnset?: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
  noDelete?: boolean;
}

const WRAPPER: Record<SectionLevel, string> = {
  1: "rounded-lg border bg-white",
  2: "border-l-2 border-slate-200 pl-4",
  3: "border-l border-slate-200 pl-3",
};

const HEADER: Record<SectionLevel, string> = {
  1: "px-3.5 py-2.5",
  2: "py-1.5",
  3: "py-1",
};

const BODY: Record<SectionLevel, string> = {
  1: "border-t border-slate-100 px-3.5 py-3.5",
  2: "pt-3 pb-1",
  3: "pt-2 pb-1",
};

const TITLE: Record<SectionLevel, string> = {
  1: "text-body font-semibold text-slate-900",
  2: "text-xs font-semibold text-slate-800",
  3: "text-xs font-medium text-slate-700",
};

const Section = (props: SectionProps) => {
  const depth = React.useContext(DepthContext);
  const level = props.level ?? (Math.min(depth + 1, 3) as SectionLevel);

  const arrLen =
    props.isList && Array.isArray(props.obj) ? props.obj.length : 0;
  const isSet = props.isList ? arrLen > 0 : props.obj !== undefined;
  const canSet = !!props.onSet || !!props.onAddListItem;

  const [open, setOpen] = React.useState(isSet);
  const wasSet = React.useRef(isSet);

  React.useEffect(() => {
    if (isSet !== wasSet.current) {
      wasSet.current = isSet;
      setOpen(isSet);
    }
  }, [isSet]);

  const addItem = () => {
    props.onAddListItem?.();
    setOpen(true);
  };

  const toggle = () => {
    if (isSet) {
      setOpen((value) => !value);
      return;
    }
    if (props.onSet) props.onSet();
    else props.onAddListItem?.();
    setOpen(true);
  };

  return (
    <section
      className={twMerge(
        "[&:not(:first-child)]:mt-4",
        WRAPPER[level],
        level === 1 && (isSet ? "border-slate-200" : "border-slate-200/70"),
      )}
    >
      <div className={twMerge("flex items-start gap-2", HEADER[level])}>
        <button
          type="button"
          onClick={toggle}
          aria-expanded={isSet ? open : undefined}
          disabled={!isSet && !canSet}
          className="group flex min-w-0 flex-1 items-start gap-2 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-default"
        >
          <ChevronRight
            size={14}
            strokeWidth={2.25}
            className={twMerge(
              "mt-0.5 shrink-0 text-slate-500 transition-transform duration-150",
              isSet && open && "rotate-90",
              !isSet && "opacity-60",
            )}
          />

          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              {props.title && (
                <span className={TITLE[level]}>{props.title}</span>
              )}

              {props.isList && isSet && (
                <span className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600">
                  {arrLen} {arrLen === 1 ? "item" : "items"}
                </span>
              )}

              {!isSet && canSet && (
                <span className="text-micro font-normal text-slate-500">
                  Not configured
                </span>
              )}
            </span>

            {props.description && (
              <span className="max-w-2xl text-xs font-normal leading-5 text-slate-500">
                {props.description}
              </span>
            )}
          </span>
        </button>

        {props.isList && props.onAddListItem && (
          <Button
            type="button"
            variant="default"
            size="compact-xs"
            leftSection={<Plus size={11} strokeWidth={2.5} />}
            onClick={addItem}
          >
            Add
          </Button>
        )}

        {isSet && props.onUnset && !props.noDelete && (
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
        )}
      </div>

      <Collapse expanded={isSet && open} transitionDuration={150}>
        <div className={BODY[level]}>
          <DepthContext.Provider value={depth + 1}>
            {props.children}
          </DepthContext.Provider>
        </div>
      </Collapse>
    </section>
  );
};

export default Section;
