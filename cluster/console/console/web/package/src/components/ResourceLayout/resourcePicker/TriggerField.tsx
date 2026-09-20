import { Resource } from "@/utils/pb";
import { Input } from "@mantine/core";
import { ChevronDown, Search, X } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { resourceFacts, resourceIcon } from "./facts";

export const SelectedChip = (props: {
  name: string;
  item?: Resource;
  api: string;
  kind: string;
  onRemove?: () => void;
}) => {
  const Icon = resourceIcon(props.api, props.kind);
  const metadata = props.item?.metadata;
  const facts = props.item ? resourceFacts(props.item) : undefined;

  return (
    <span className="inline-flex min-w-0 max-w-full items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 py-1 pl-1 pr-1.5">
      <span className="flex h-5 w-5 shrink-0 items-center justify-center overflow-hidden rounded border border-slate-200 bg-white text-slate-500">
        {metadata?.picURL ? (
          <img
            src={metadata.picURL}
            alt=""
            loading="lazy"
            className="h-full w-full object-cover"
          />
        ) : (
          <Icon size={11} strokeWidth={2.2} />
        )}
      </span>

      <span className="truncate text-xs font-semibold text-slate-800">
        {props.name}
      </span>

      {facts?.primary && (
        <span className="hidden truncate text-micro font-normal text-slate-500 sm:inline">
          {facts.primary}
        </span>
      )}

      {props.onRemove && (
        <button
          type="button"
          aria-label={`Remove ${props.name}`}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            props.onRemove?.();
          }}
          className="ml-0.5 shrink-0 cursor-pointer rounded text-slate-400 outline-none transition-colors duration-150 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <X size={12} strokeWidth={2.5} />
        </button>
      )}
    </span>
  );
};

const TriggerField = (props: {
  label?: string;
  description?: string;
  required?: boolean;
  placeholder: string;
  empty: boolean;
  onOpen: () => void;
  onClear?: () => void;
  children: React.ReactNode;
}) => (
  <Input.Wrapper
    label={props.label}
    description={props.description}
    required={props.required}
  >
    <div
      className={twMerge(
        "mt-1 flex min-h-9 w-full items-center gap-2 rounded-md border border-slate-300 bg-white px-2 py-1.5 transition-[border-color,box-shadow] duration-150",
        "focus-within:border-slate-500 hover:border-slate-400",
      )}
    >
      <button
        type="button"
        onClick={props.onOpen}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        {props.empty ? (
          <span className="flex min-w-0 items-center gap-2 text-slate-500">
            <Search size={13} strokeWidth={2.1} className="shrink-0" />
            <span className="truncate text-body font-normal">
              {props.placeholder}
            </span>
          </span>
        ) : (
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            {props.children}
          </span>
        )}
      </button>

      {props.onClear && !props.empty && (
        <button
          type="button"
          aria-label="Clear the selection"
          onClick={props.onClear}
          className="shrink-0 cursor-pointer rounded text-slate-400 outline-none transition-colors duration-150 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <X size={14} strokeWidth={2.4} />
        </button>
      )}

      <ChevronDown
        size={14}
        strokeWidth={2.2}
        aria-hidden="true"
        className="shrink-0 text-slate-400"
      />
    </div>
  </Input.Wrapper>
);

export default TriggerField;
