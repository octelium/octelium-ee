import TimeAgo from "@/components/TimeAgo";
import { Resource } from "@/utils/pb";
import { ActionIcon, Tooltip } from "@mantine/core";
import { Check, FileText } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { resourceFacts, resourceIcon } from "./facts";

const Badge = (props: {
  children: React.ReactNode;
  tone?: "muted" | "warning";
}) => (
  <span
    className={twMerge(
      "inline-flex shrink-0 items-center rounded border px-1.5 py-px text-micro font-semibold",
      props.tone === "warning"
        ? "border-amber-200 bg-amber-50 text-amber-700"
        : "border-slate-200 bg-slate-50 text-slate-600",
    )}
  >
    {props.children}
  </span>
);

const ResourceRow = (props: {
  item: Resource;
  api: string;
  kind: string;
  selected: boolean;
  active: boolean;
  onPick: () => void;
  onPeek: () => void;
}) => {
  const { item } = props;
  const metadata = item.metadata!;
  const facts = resourceFacts(item);
  const Icon = resourceIcon(props.api, props.kind);

  return (
    <li>
      <div
        role="option"
        aria-selected={props.selected}
        tabIndex={-1}
        onClick={props.onPick}
        className={twMerge(
          "group flex w-full cursor-pointer items-start gap-3 rounded-xl border px-3 py-2.5 text-left transition-[border-color,background-color,box-shadow] duration-150",
          props.selected
            ? "border-slate-400 bg-white shadow-card"
            : "border-slate-200 bg-white hover:border-slate-300",
          props.active && !props.selected && "border-slate-400 bg-slate-50/70",
        )}
      >
        <span
          className={twMerge(
            "mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border",
            props.selected
              ? "border-slate-900 bg-slate-900 text-white"
              : "border-slate-200 bg-slate-50 text-slate-500",
          )}
        >
          {props.selected ? (
            <Check size={13} strokeWidth={3} />
          ) : metadata.picURL ? (
            <img
              src={metadata.picURL}
              alt=""
              loading="lazy"
              className="h-full w-full rounded-[7px] object-cover"
            />
          ) : (
            <Icon size={13} strokeWidth={2.2} />
          )}
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <span className="truncate text-body font-semibold text-slate-800">
              {metadata.name}
            </span>
            {metadata.displayName && (
              <span className="truncate text-xs font-normal text-slate-500">
                {metadata.displayName}
              </span>
            )}
            {metadata.isSystem && <Badge>System</Badge>}
            {facts.isDisabled && <Badge tone="warning">Disabled</Badge>}
          </span>

          {(facts.primary || metadata.description) && (
            <span className="truncate text-xs font-normal text-slate-600">
              {facts.primary ?? metadata.description}
            </span>
          )}

          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            {facts.chips.map((chip) => (
              <Badge key={chip}>{chip}</Badge>
            ))}
            <span className="truncate text-micro font-normal text-slate-500">
              Created <TimeAgo rfc3339={metadata.createdAt} />
            </span>
          </span>
        </span>

        <Tooltip label="View YAML" withArrow>
          <ActionIcon
            type="button"
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={`View the YAML of ${metadata.name}`}
            className="shrink-0"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              props.onPeek();
            }}
          >
            <FileText size={13} strokeWidth={2.1} />
          </ActionIcon>
        </Tooltip>
      </div>
    </li>
  );
};

export default ResourceRow;
