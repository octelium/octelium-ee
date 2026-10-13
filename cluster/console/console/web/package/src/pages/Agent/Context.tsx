import type { RunContext } from "@/apis/consoleagent/protocol";
import {
  Box,
  PanelsTopLeft,
  ScrollText,
  X,
  type LucideIcon,
} from "lucide-react";
import { twMerge } from "@/lib/utils";

export interface ContextChip {
  key: string;
  icon: LucideIcon;
  label: string;
  title?: string;
  onRemove?: () => void;
}

const lastSegment = (title: string) => title.split(" › ").at(-1) ?? title;

export const contextChips = (ctx?: RunContext): ContextChip[] => {
  if (!ctx) return [];
  const resources: ContextChip[] = (ctx.resources ?? []).map((ref, idx) => ({
    key: `r-${idx}-${ref.kind}-${ref.name ?? ref.uid}`,
    icon: Box,
    label: ref.name ?? ref.uid ?? ref.kind,
    title: `${ref.kind} ${ref.name ?? ref.uid ?? ""}`.trim(),
  }));
  const items: ContextChip[] = (ctx.items ?? []).map((item, idx) => ({
    key: `i-${idx}-${item.kind}-${item.id ?? ""}`,
    icon: ScrollText,
    label: item.title ?? item.kind,
    title: item.title ? `${item.kind}: ${item.title}` : item.kind,
  }));
  const page: ContextChip[] = [];
  if (ctx.page) {
    const label = lastSegment(ctx.page.title ?? ctx.page.path);
    if (!resources.some((chip) => chip.label === label)) {
      page.push({
        key: "page",
        icon: PanelsTopLeft,
        label,
        title: ctx.page.title ?? ctx.page.path,
      });
    }
  }
  return [...page, ...resources, ...items];
};

export const ContextChips = (props: {
  chips: ContextChip[];
  className?: string;
  muted?: boolean;
}) => {
  if (props.chips.length === 0) return null;
  return (
    <div className={twMerge("flex min-w-0 flex-wrap gap-1.5", props.className)}>
      {props.chips.map((chip) => (
        <span
          key={chip.key}
          title={chip.title}
          className={`text-micro ${twMerge(
            "inline-flex h-6 min-w-0 max-w-full items-center gap-1 rounded-md border pl-1.5 font-semibold",
            props.muted
              ? "border-slate-200/80 bg-white text-slate-500"
              : "border-slate-200 bg-slate-50 text-slate-600",
            chip.onRemove ? "pr-0.5" : "pr-2",
          )}`}
        >
          <chip.icon size={11} strokeWidth={2.25} className="shrink-0" />
          <span className="max-w-[200px] truncate">{chip.label}</span>
          {chip.onRemove && (
            <button
              type="button"
              aria-label={`Remove ${chip.title ?? chip.label} from the context`}
              onClick={chip.onRemove}
              className="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center rounded text-slate-400 transition-colors duration-150 hover:bg-slate-200/70 hover:text-slate-700"
            >
              <X size={11} strokeWidth={2.5} />
            </button>
          )}
        </span>
      ))}
    </div>
  );
};
