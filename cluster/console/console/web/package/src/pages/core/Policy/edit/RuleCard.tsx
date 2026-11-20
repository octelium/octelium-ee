import ConditionBuilder from "@/components/Condition/Builder";
import { summarize, type ConditionNode } from "@/components/Condition/model";
import type { Recipe } from "@/components/Condition/recipes";
import PriorityPicker from "@/components/PriorityPicker";
import { twMerge } from "@/lib/utils";
import { ActionIcon, Collapse, TextInput, Tooltip } from "@mantine/core";
import {
  AlertTriangle,
  ChevronDown,
  CircleOff,
  Copy,
  Filter,
  ListOrdered,
  Power,
  ShieldCheck,
  ShieldX,
  Sparkles,
  Trash2,
  Zap,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";

export type RuleKind = "rule" | "enforcement";

type EffectMeta = {
  label: string;
  description: string;
  icon: LucideIcon;
  active: string;
  badge: string;
};

const EFFECTS: Record<RuleKind, Record<string, EffectMeta>> = {
  rule: {
    ALLOW: {
      label: "Allow",
      description: "Grant access to the matching requests",
      icon: ShieldCheck,
      active:
        "border-emerald-300 bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200",
      badge: "border-emerald-200 bg-emerald-50 text-emerald-700",
    },
    DENY: {
      label: "Deny",
      description: "Block them, before any Allow rule of the same priority",
      icon: ShieldX,
      active: "border-red-300 bg-red-50 text-red-800 ring-1 ring-red-200",
      badge: "border-red-200 bg-red-50 text-red-700",
    },
  },
  enforcement: {
    ENFORCE: {
      label: "Enforce",
      description: "Evaluate this Policy's rules for the matching requests",
      icon: Power,
      active:
        "border-indigo-300 bg-indigo-50 text-indigo-800 ring-1 ring-indigo-200",
      badge: "border-indigo-200 bg-indigo-50 text-indigo-700",
    },
    IGNORE: {
      label: "Ignore",
      description: "Skip this Policy entirely for the matching requests",
      icon: CircleOff,
      active:
        "border-amber-300 bg-amber-50 text-amber-800 ring-1 ring-amber-200",
      badge: "border-amber-200 bg-amber-50 text-amber-700",
    },
  },
};

export const EffectBadge = (props: { kind: RuleKind; effect: string }) => {
  const meta = EFFECTS[props.kind][props.effect];
  if (!meta) {
    return (
      <span className="rounded-md border border-dashed border-slate-300 px-1.5 py-px text-micro font-semibold text-slate-500">
        No effect
      </span>
    );
  }
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-1 rounded-md border px-1.5 py-px text-micro font-bold uppercase tracking-[0.05em]",
        meta.badge,
      )}
    >
      <meta.icon size={10} strokeWidth={2.6} />
      {meta.label}
    </span>
  );
};

const Section = (props: {
  icon: LucideIcon;
  title: string;
  hint?: React.ReactNode;
  children: React.ReactNode;
}) => (
  <section className="flex min-w-0 flex-col gap-3">
    <header className="flex items-start gap-2.5">
      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-500">
        <props.icon size={12} strokeWidth={2.3} />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="text-body font-semibold text-slate-800">
          {props.title}
        </span>
        {props.hint && (
          <span className="text-xs leading-5 text-slate-500">{props.hint}</span>
        )}
      </span>
    </header>
    <div className="flex min-w-0 flex-col gap-3 @2xl:pl-8">
      {props.children}
    </div>
  </section>
);

const EffectCards = (props: {
  kind: RuleKind;
  value: string;
  onChange: (effect: string) => void;
}) => (
  <div
    role="radiogroup"
    aria-label="Effect"
    className="grid grid-cols-1 gap-2 @xl:grid-cols-2"
  >
    {Object.entries(EFFECTS[props.kind]).map(([effect, meta]) => {
      const active = props.value === effect;
      return (
        <button
          key={effect}
          type="button"
          role="radio"
          aria-checked={active}
          onClick={() => !active && props.onChange(effect)}
          className={twMerge(
            "flex min-w-0 cursor-pointer items-start gap-2.5 rounded-xl border px-3 py-2.5 text-left outline-none transition-[border-color,box-shadow,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
            active
              ? meta.active
              : "border-slate-200 bg-white hover:border-slate-300",
          )}
        >
          <span
            className={twMerge(
              "mt-px flex h-6 w-6 shrink-0 items-center justify-center rounded-md border",
              active ? "border-current/30 bg-white/60" : meta.badge,
            )}
          >
            <meta.icon size={13} strokeWidth={2.4} />
          </span>
          <span className="flex min-w-0 flex-col">
            <span
              className={twMerge(
                "text-xs font-bold",
                !active && "text-slate-800",
              )}
            >
              {meta.label}
            </span>
            <span
              className={twMerge(
                "text-micro leading-4",
                active ? "opacity-90" : "text-slate-500",
              )}
            >
              {meta.description}
            </span>
          </span>
        </button>
      );
    })}
  </div>
);

export const RuleCard = (props: {
  kind: RuleKind;
  index: number;
  name: string;
  effect: string;
  priority?: number;
  node?: ConditionNode;
  issues: string[];
  open: boolean;
  highlight?: boolean;
  onToggle: () => void;
  onEffect: (effect: string) => void;
  onName?: (name: string) => void;
  onPriority?: (priority: number) => void;
  onNode: (node?: ConditionNode) => void;
  onRecipe?: (recipe: Recipe) => void;
  onAskAI?: () => void;
  onDuplicate: () => void;
  onRemove: () => void;
  duplicateName?: boolean;
}) => {
  const bodyId = React.useId();
  const rule = props.kind === "rule";
  const title = rule
    ? props.name || "Unnamed rule"
    : `Enforcement rule ${props.index + 1}`;

  return (
    <article
      data-rule-card={props.kind}
      className={twMerge(
        "min-w-0 rounded-xl border bg-white shadow-card transition-[border-color,box-shadow] duration-300",
        props.issues.length > 0 && !props.open
          ? "border-amber-200"
          : "border-slate-200",
        props.highlight && "border-blue-300 ring-2 ring-blue-100",
      )}
    >
      <header className="flex items-start gap-3 px-3 py-3 sm:px-4">
        <button
          type="button"
          aria-expanded={props.open}
          aria-controls={bodyId}
          onClick={props.onToggle}
          className="flex min-w-0 flex-1 cursor-pointer items-start gap-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-slate-100 text-micro font-bold text-slate-600">
            {props.index + 1}
          </span>
          <span className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span
                className={twMerge(
                  "min-w-0 truncate text-body font-semibold text-slate-900",
                  rule && !props.name && "font-medium italic text-slate-400",
                )}
              >
                {title}
              </span>
              <EffectBadge kind={props.kind} effect={props.effect} />
              {rule && (
                <span className="rounded-md border border-slate-200 px-1.5 py-px text-micro font-semibold tabular-nums text-slate-500">
                  priority{" "}
                  {(props.priority ?? 0) > 0
                    ? `+${props.priority}`
                    : (props.priority ?? 0)}
                </span>
              )}
              {props.issues.length > 0 && (
                <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-1.5 py-px text-micro font-semibold text-amber-700">
                  <AlertTriangle size={10} strokeWidth={2.5} />
                  {props.issues.length} to fix
                </span>
              )}
            </span>
            {!props.open && (
              <span className="line-clamp-2 break-words text-xs text-slate-600">
                <span className="font-semibold text-slate-500">When </span>
                {props.node
                  ? summarize(props.node)
                  : "no condition is set (never matches)"}
              </span>
            )}
          </span>
        </button>
        <div className="flex shrink-0 items-center gap-0.5">
          {props.onAskAI && (
            <Tooltip label="Ask AI about this rule" withArrow>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                aria-label="Ask AI about this rule"
                onClick={props.onAskAI}
              >
                <Sparkles size={13} />
              </ActionIcon>
            </Tooltip>
          )}
          <Tooltip label="Duplicate" withArrow>
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              aria-label="Duplicate rule"
              onClick={props.onDuplicate}
            >
              <Copy size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Remove" withArrow>
            <ActionIcon
              variant="subtle"
              color="red"
              size="sm"
              aria-label="Remove rule"
              onClick={props.onRemove}
            >
              <Trash2 size={13} />
            </ActionIcon>
          </Tooltip>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={props.open ? "Collapse rule" : "Expand rule"}
            onClick={props.onToggle}
          >
            <ChevronDown
              size={14}
              className={twMerge(
                "transition-transform duration-150",
                !props.open && "-rotate-90",
              )}
            />
          </ActionIcon>
        </div>
      </header>

      <Collapse expanded={props.open} id={bodyId}>
        <div className="@container flex flex-col gap-6 border-t border-slate-100 px-3 py-5 sm:px-4">
          <Section
            icon={Zap}
            title={rule ? "Effect" : "Effect on this Policy"}
            hint={
              rule
                ? "What happens to the requests this rule matches."
                : "Whether this Policy takes part in the decision for the matching requests."
            }
          >
            <EffectCards
              kind={props.kind}
              value={props.effect}
              onChange={props.onEffect}
            />
          </Section>

          <Section
            icon={Filter}
            title="When the request matches"
            hint="Combine conditions on who makes the request, what it targets, how and when. Switch to CEL at any time."
          >
            <ConditionBuilder
              value={props.node}
              onChange={props.onNode}
              scope="request"
              emptyText={
                rule
                  ? "A rule without a condition never matches. Start from a common use case, browse every condition or write CEL."
                  : "Start from a common use case, browse every condition or write CEL."
              }
              featured={
                rule
                  ? undefined
                  : [
                      "api-full",
                      "http-read-only",
                      "business-hours",
                      "group-members",
                      "every",
                    ]
              }
              onRecipe={props.onRecipe}
              onAskAI={props.onAskAI}
            />
          </Section>

          {rule && props.onName && props.onPriority && (
            <Section icon={ListOrdered} title="Name and priority">
              <div className="grid grid-cols-1 gap-3 @2xl:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
                <TextInput
                  label="Rule name"
                  description="Optional. Shown in access logs as the rule that decided."
                  placeholder="allow-sre"
                  value={props.name}
                  error={
                    props.duplicateName
                      ? "Another rule already has this name"
                      : undefined
                  }
                  onChange={(event) =>
                    props.onName?.(event.currentTarget.value)
                  }
                />
                <PriorityPicker
                  label="Priority"
                  description="Rules of every applicable Policy are checked together, from -4 to 4. The first rule that matches decides."
                  value={props.priority ?? 0}
                  onChange={(value) => props.onPriority?.(value)}
                />
              </div>
            </Section>
          )}

          {props.issues.length > 0 && (
            <div className="flex flex-col gap-1 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2">
              <span className="flex items-center gap-1.5 text-xs font-semibold text-amber-800">
                <AlertTriangle size={12} /> Before saving
              </span>
              <ul className="list-disc pl-5 text-micro leading-5 text-amber-800">
                {props.issues.map((issue) => (
                  <li key={issue}>{issue}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </Collapse>
    </article>
  );
};
