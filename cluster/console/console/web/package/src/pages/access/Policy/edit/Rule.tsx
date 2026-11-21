import * as AccessP from "@/apis/accessv1/accessv1";
import DurationPicker from "@/components/DurationPicker";
import PriorityPicker from "@/components/PriorityPicker";
import { twMerge } from "@/lib/utils";
import { useListKeys } from "@/utils/forms";
import { ActionIcon, Collapse, Textarea, TextInput, Tooltip } from "@mantine/core";
import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  BellRing,
  ChevronDown,
  Copy,
  Filter,
  KeyRound,
  ListOrdered,
  LucideIcon,
  MessageSquareText,
  Trash2,
  Zap,
} from "lucide-react";
import * as React from "react";
import { EFFECT_META, EffectBadge, EffectKey, getEffectKey, TONE_SOFT } from "../../common";
import { ConditionView } from "../../rules";
import { ConditionEdit, newCondition } from "./Condition";
import { newReviewStep, StepsEdit } from "./Step";
import { SurfacesEdit } from "./Surface";
import { ruleIssues } from "./validate";

const RuleSection = (props: {
  icon: LucideIcon;
  title: string;
  hint?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) => {
  const Icon = props.icon;
  return (
    <section className={twMerge("flex flex-col gap-3", props.className)}>
      <header className="flex items-start gap-2.5">
        <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-500">
          <Icon size={12} strokeWidth={2.3} />
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="text-body font-semibold text-slate-800">{props.title}</span>
          {props.hint && (
            <span className="text-xs font-normal leading-5 text-slate-500">{props.hint}</span>
          )}
        </span>
      </header>
      <div className="flex flex-col gap-3 pl-0 @2xl:pl-8">{props.children}</div>
    </section>
  );
};

const EffectCards = (props: {
  value?: AccessP.Policy_Spec_Rule_Effect;
  onChange: (key: EffectKey) => void;
}) => {
  const current = getEffectKey(props.value);
  const order: EffectKey[] = ["AUTO_APPROVE", "REVIEW", "DENY"];
  return (
    <div role="radiogroup" aria-label="Effect" className="grid grid-cols-1 gap-2 @2xl:grid-cols-3">
      {order.map((key) => {
        const meta = EFFECT_META[key];
        const Icon = meta.icon;
        const active = current === key;
        return (
          <button
            key={key}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => !active && props.onChange(key)}
            className={twMerge(
              "flex min-w-0 cursor-pointer flex-col gap-1.5 rounded-xl border px-3 py-2.5 text-left outline-none transition-[border-color,box-shadow,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
              active
                ? twMerge(TONE_SOFT[meta.tone], "shadow-sm ring-1 ring-current/20")
                : "border-slate-200 bg-white hover:border-slate-300",
            )}
          >
            <span className="flex items-center gap-2">
              <span
                className={twMerge(
                  "flex h-6 w-6 shrink-0 items-center justify-center rounded-md border",
                  active ? "border-current/30 bg-white/60" : TONE_SOFT[meta.tone],
                )}
              >
                <Icon size={13} strokeWidth={2.4} />
              </span>
              <span className={twMerge("text-xs font-bold", active ? "" : "text-slate-800")}>
                {meta.label}
              </span>
            </span>
            <span className={twMerge("text-micro leading-4", active ? "opacity-90" : "text-slate-600")}>
              {meta.description}
            </span>
            <span className={twMerge("text-micro leading-4", active ? "opacity-75" : "text-slate-500")}>
              {meta.useWhen}
            </span>
          </button>
        );
      })}
    </div>
  );
};

const MAX_RULE_MESSAGE_LENGTH = 500;

const LimitsEdit = (props: {
  rule: AccessP.Policy_Spec_Rule;
  onUpdate: () => void;
}) => {
  const { rule, onUpdate } = props;

  return (
    <>
      <div className="max-w-sm">
        <DurationPicker
          title="Longest access"
          description="Requesters may ask for less, never more. The access level of the Catalog can only lower it."
          placeholder="Access level maximum"
          value={rule.maxDuration}
          onChange={(value) => {
            rule.maxDuration = value;
            onUpdate();
          }}
        />
      </div>
      <p className="text-xs leading-5 text-slate-600">
        What an approval grants is defined by the access level that is
        requested from the Catalog. This rule only decides whether and how the
        access is approved.
      </p>
    </>
  );
};

type Stash = {
  action?: AccessP.Policy_Spec_Rule_Action;
  maxDuration?: AccessP.Policy_Spec_Rule["maxDuration"];
};

const setEffect = (rule: AccessP.Policy_Spec_Rule, key: EffectKey, stash: Stash) => {
  if (rule.action) stash.action = rule.action;
  if (rule.maxDuration) stash.maxDuration = rule.maxDuration;

  rule.effect = EFFECT_META[key].value;
  if (key === "REVIEW") {
    rule.action =
      stash.action ??
      AccessP.Policy_Spec_Rule_Action.create({
        type: {
          oneofKind: "review",
          review: AccessP.Policy_Spec_Rule_Action_Review.create({
            steps: [newReviewStep("approval")],
          }),
        },
      });
  } else {
    rule.action = undefined;
  }
  rule.maxDuration = key === "DENY" ? undefined : stash.maxDuration;
};

export const RuleCard = (props: {
  rule: AccessP.Policy_Spec_Rule;
  index: number;
  rules: AccessP.Policy_Spec_Rule[];
  defaultOpen: boolean;
  onUpdate: () => void;
  onMove: (delta: number) => void;
  onDuplicate: () => void;
  onRemove: () => void;
}) => {
  const rows = useListKeys();
  const { rule, index, rules, onUpdate } = props;
  const [open, setOpen] = React.useState(props.defaultOpen);
  const stash = React.useRef<Stash>({});
  const issues = ruleIssues(rule, rules);
  const effectKey = getEffectKey(rule.effect);
  const review = rule.action?.type.oneofKind === "review" ? rule.action.type.review : undefined;
  const bodyId = React.useId();

  return (
    <article
      className={twMerge(
        "overflow-hidden rounded-xl border bg-white shadow-card transition-colors duration-150",
        issues.length > 0 && !open ? "border-amber-200" : "border-slate-200",
      )}
    >
      <header className="flex items-start gap-3 px-4 py-3">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={bodyId}
          onClick={() => setOpen((value) => !value)}
          className="flex min-w-0 flex-1 cursor-pointer items-start gap-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-slate-100 text-micro font-bold text-slate-600">
            {index + 1}
          </span>
          <span className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="flex flex-wrap items-center gap-2">
              <span className="truncate text-body font-semibold text-slate-900">
                {rule.name || <span className="italic text-slate-400">Unnamed rule</span>}
              </span>
              <EffectBadge effect={rule.effect} />
              <span className="rounded-md border border-slate-200 px-1.5 py-px text-micro font-semibold tabular-nums text-slate-500">
                priority {rule.priority > 0 ? `+${rule.priority}` : rule.priority}
              </span>
              {review && (
                <span className="text-micro text-slate-500">
                  {review.steps.length} review step{review.steps.length === 1 ? "" : "s"}
                </span>
              )}
              {issues.length > 0 && (
                <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-1.5 py-px text-micro font-semibold text-amber-700">
                  <AlertTriangle size={10} strokeWidth={2.5} />
                  {issues.length} to fix
                </span>
              )}
            </span>
            {!open && (
              <span className="text-xs text-slate-600">
                <ConditionView condition={rule.condition} />
              </span>
            )}
          </span>
        </button>
        <div className="flex shrink-0 items-center gap-0.5">
          <Tooltip label="Move up" withArrow>
            <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move rule up" disabled={index === 0} onClick={() => props.onMove(-1)}>
              <ArrowUp size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Move down" withArrow>
            <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move rule down" disabled={index === rules.length - 1} onClick={() => props.onMove(1)}>
              <ArrowDown size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Duplicate" withArrow>
            <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Duplicate rule" onClick={props.onDuplicate}>
              <Copy size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Remove rule" withArrow>
            <ActionIcon variant="subtle" color="red" size="sm" aria-label="Remove rule" onClick={props.onRemove}>
              <Trash2 size={13} />
            </ActionIcon>
          </Tooltip>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={open ? "Collapse rule" : "Expand rule"}
            onClick={() => setOpen((value) => !value)}
          >
            <ChevronDown size={14} className={twMerge("transition-transform duration-150", !open && "-rotate-90")} />
          </ActionIcon>
        </div>
      </header>

      <Collapse expanded={open} id={bodyId}>
        <div className="@container flex flex-col gap-6 border-t border-slate-100 px-4 py-5">
          <RuleSection icon={ListOrdered} title="Name and order">
            <div className="grid grid-cols-1 gap-3 @2xl:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
              <TextInput
                label="Rule name"
                description="Unique within the Policy. Shown on every request it decides."
                placeholder="sre-review"
                value={rule.name}
                error={
                  rule.name && rules.filter((other) => other.name === rule.name).length > 1
                    ? "Another rule already has this name"
                    : undefined
                }
                onChange={(event) => {
                  rule.name = event.target.value;
                  onUpdate();
                }}
              />
              <PriorityPicker
                label="Priority"
                description="Rules of every enabled Policy are checked together, highest priority first. The first rule that matches decides."
                value={rule.priority}
                onChange={(value) => {
                  rule.priority = value;
                  onUpdate();
                }}
              />
            </div>
          </RuleSection>

          <RuleSection
            icon={Filter}
            title="When a request matches"
            hint="Describe the requests this rule decides: who the access is for, what it is for, or both."
          >
            {rule.condition ? (
              <ConditionEdit condition={rule.condition} onUpdate={onUpdate} />
            ) : (
              <button
                type="button"
                className="w-fit cursor-pointer text-xs font-semibold text-blue-700 hover:text-blue-800"
                onClick={() => {
                  rule.condition = newCondition();
                  onUpdate();
                }}
              >
                Set a condition
              </button>
            )}
          </RuleSection>

          <RuleSection icon={Zap} title="Then" hint="What happens to the requests that match.">
            <EffectCards
              value={rule.effect}
              onChange={(key) => {
                setEffect(rule, key, stash.current);
                onUpdate();
              }}
            />
          </RuleSection>

          {effectKey === "REVIEW" && review && (
            <RuleSection
              icon={ListOrdered}
              title="Review steps"
              hint="Who must approve, in what order, and what happens when nobody answers."
            >
              <StepsEdit review={review} onUpdate={onUpdate} />
            </RuleSection>
          )}

          {(effectKey === "REVIEW" || effectKey === "AUTO_APPROVE") && (
            <RuleSection
              icon={KeyRound}
              title="Access limits"
              hint={
                effectKey === "REVIEW"
                  ? "Applies once the last step is cleared."
                  : "Applies immediately to every matching request."
              }
            >
              <LimitsEdit rule={rule} onUpdate={onUpdate} />
            </RuleSection>
          )}

          {effectKey === "DENY" && (
            <p className="rounded-lg border border-red-100 bg-red-50/60 px-3 py-2 text-xs leading-5 text-red-800">
              Matching requests are rejected as soon as they are created.
              Rules checked after this one are never reached for them, so give
              guardrails the highest priority.
            </p>
          )}

          <RuleSection
            icon={MessageSquareText}
            title="Message to the requester"
            hint="Shown to the requester before submitting and on every request this rule decides."
          >
            <Textarea
              label="Message"
              description={
                effectKey === "DENY"
                  ? "Explain why the access is denied and where to ask instead."
                  : "Optional. For instance, what reviewers expect in the justification."
              }
              autosize
              minRows={2}
              maxRows={5}
              value={rule.message}
              error={
                Array.from(rule.message).length > MAX_RULE_MESSAGE_LENGTH
                  ? `Use at most ${MAX_RULE_MESSAGE_LENGTH} characters`
                  : undefined
              }
              onChange={(event) => {
                rule.message = event.target.value;
                onUpdate();
              }}
            />
          </RuleSection>

          <RuleSection
            icon={BellRing}
            title="Announce the outcome"
            hint="Optionally post the final decision to a channel or message the people involved. These messages never carry any action."
          >
            <SurfacesEdit
              surfaces={rule.notifications}
              isReview={false}
              keyAt={(i) => rows.keyAt("notifications", i)}
              removeKeyAt={(i) => rows.removeAt("notifications", i)}
              onUpdate={onUpdate}
              addLabel="Add notification"
              emptyLabel="No notification: the outcome is only visible in the access portal."
            />
          </RuleSection>

          {issues.length > 0 && (
            <div className="flex flex-col gap-1 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2">
              <span className="flex items-center gap-1.5 text-xs font-semibold text-amber-800">
                <AlertTriangle size={12} /> Before saving
              </span>
              <ul className="list-disc pl-5 text-micro leading-5 text-amber-800">
                {issues.map((issue) => (
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
