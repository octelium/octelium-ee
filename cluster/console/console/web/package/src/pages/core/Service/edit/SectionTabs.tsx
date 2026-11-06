import { twMerge } from "@/lib/utils";
import { LucideIcon } from "lucide-react";
import * as React from "react";
import { ModeIcon } from "./modes";

export type SectionTab<K extends string> = {
  id: K;
  label: string;
  icon: LucideIcon | ModeIcon;
  count?: number;
  issues: string[];
};

const SectionTabs = <K extends string>(props: {
  tabs: SectionTab<K>[];
  value: K;
  onChange: (id: K) => void;
}) => {
  const refs = React.useRef<Partial<Record<K, HTMLDivElement | null>>>({});

  const select = (id: K) => {
    props.onChange(id);
    refs.current[id]?.focus();
    refs.current[id]?.scrollIntoView({ block: "nearest", inline: "nearest" });
  };

  return (
    <div
      role="tablist"
      aria-label="Service sections"
      className="sticky top-0 z-20 -mx-5 flex gap-1 overflow-x-auto border-b border-slate-200 bg-white px-5 [scrollbar-width:none] [#resource-item-drawer_&]:top-[56px]"
    >
      {props.tabs.map((tab, index) => {
        const Icon = tab.icon;
        const active = tab.id === props.value;
        const issues = tab.issues.length;
        return (
          <div
            key={tab.id}
            ref={(element) => {
              refs.current[tab.id] = element;
            }}
            id={`service-tab-${tab.id}`}
            role="tab"
            aria-selected={active}
            aria-controls={`service-panel-${tab.id}`}
            tabIndex={active ? 0 : -1}
            title={issues ? tab.issues.join(". ") : undefined}
            onClick={() => props.onChange(tab.id)}
            onKeyDown={(event) => {
              if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
                event.preventDefault();
                const step = event.key === "ArrowRight" ? 1 : -1;
                const next =
                  props.tabs[
                    (index + step + props.tabs.length) % props.tabs.length
                  ];
                select(next.id);
              }
              if (event.key === "Home" || event.key === "End") {
                event.preventDefault();
                select(
                  props.tabs[event.key === "Home" ? 0 : props.tabs.length - 1]
                    .id,
                );
              }
            }}
            className={twMerge(
              "-mb-px flex shrink-0 cursor-pointer select-none items-center gap-1.5 whitespace-nowrap border-b-2 px-2.5 py-2.5 text-body font-semibold outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400",
              active
                ? "border-slate-800 text-slate-900"
                : "border-transparent text-slate-500 hover:text-slate-800",
            )}
          >
            <Icon size={13} strokeWidth={2.25} className="hidden shrink-0 @xl:block" />
            {tab.label}
            {!!tab.count && !issues && (
              <span className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600">
                {tab.count}
              </span>
            )}
            {issues > 0 && (
              <span
                aria-label={`${issues} to complete`}
                className="rounded-full border border-amber-200 bg-amber-50 px-1.5 py-px text-micro font-semibold text-amber-700"
              >
                {issues}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default SectionTabs;
