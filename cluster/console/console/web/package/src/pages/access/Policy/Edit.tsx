import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { twMerge } from "@/lib/utils";
import { Switch } from "@mantine/core";
import { Ban, CircleCheck, FilePlus2, Info, Layers, ListOrdered, Plus, Users } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import * as React from "react";
import { newCondition } from "./edit/Condition";
import { RuleCard } from "./edit/Rule";
import { newReviewStep } from "./edit/Step";

const Effect = AccessP.Policy_Spec_Rule_Effect;

const uniqueName = (rules: AccessP.Policy_Spec_Rule[], base: string) => {
  const names = new Set(rules.map((rule) => rule.name));
  if (!names.has(base)) return base;
  for (let i = 2; ; i++) if (!names.has(`${base}-${i}`)) return `${base}-${i}`;
};

type Template = {
  id: string;
  label: string;
  description: string;
  icon: LucideIcon;
  build: (rules: AccessP.Policy_Spec_Rule[]) => AccessP.Policy_Spec_Rule;
};

const reviewAction = (names: string[]) =>
  AccessP.Policy_Spec_Rule_Action.create({
    type: {
      oneofKind: "review",
      review: AccessP.Policy_Spec_Rule_Action_Review.create({
        steps: names.map((name) => newReviewStep(name)),
      }),
    },
  });

const groupCondition = () =>
  AccessP.Policy_Spec_Rule_Condition.create({
    type: {
      oneofKind: "subject",
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: { oneofKind: "groupRef", groupRef: MetaP.ObjectReference.create() },
      }),
    },
  });

const catalogCondition = () =>
  AccessP.Policy_Spec_Rule_Condition.create({
    type: {
      oneofKind: "resource",
      resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
        type: { oneofKind: "catalogRef", catalogRef: MetaP.ObjectReference.create() },
      }),
    },
  });

const TEMPLATES: Template[] = [
  {
    id: "review",
    label: "Require a review",
    description: "Requests for a Catalog need one approval",
    icon: Users,
    build: (rules) =>
      AccessP.Policy_Spec_Rule.create({
        name: uniqueName(rules, "review"),
        condition: catalogCondition(),
        effect: Effect.REVIEW,
        action: reviewAction(["approval"]),
      }),
  },
  {
    id: "two-steps",
    label: "Two-step review",
    description: "A team lead, then security",
    icon: Layers,
    build: (rules) =>
      AccessP.Policy_Spec_Rule.create({
        name: uniqueName(rules, "two-step-review"),
        condition: catalogCondition(),
        effect: Effect.REVIEW,
        action: reviewAction(["team-lead", "security"]),
      }),
  },
  {
    id: "auto",
    label: "Auto-approve a Group",
    description: "Trusted people get access at once",
    icon: CircleCheck,
    build: (rules) =>
      AccessP.Policy_Spec_Rule.create({
        name: uniqueName(rules, "auto-approve"),
        condition: groupCondition(),
        effect: Effect.AUTO_APPROVE,
        priority: -1,
      }),
  },
  {
    id: "deny",
    label: "Deny guardrail",
    description: "Always reject, checked first",
    icon: Ban,
    build: (rules) =>
      AccessP.Policy_Spec_Rule.create({
        name: uniqueName(rules, "deny"),
        condition: groupCondition(),
        effect: Effect.DENY,
        priority: -4,
      }),
  },
  {
    id: "blank",
    label: "Blank rule",
    description: "Start from scratch",
    icon: FilePlus2,
    build: (rules) =>
      AccessP.Policy_Spec_Rule.create({
        name: uniqueName(rules, "rule"),
        condition: newCondition(),
      }),
  },
];

const TemplateGrid = (props: { onPick: (template: Template) => void; prominent?: boolean }) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-2",
      props.prominent ? "sm:grid-cols-2 lg:grid-cols-5" : "sm:grid-cols-3 lg:grid-cols-5",
    )}
  >
    {TEMPLATES.map((template) => {
      const Icon = template.icon;
      return (
        <button
          key={template.id}
          type="button"
          onClick={() => props.onPick(template)}
          className="group flex min-w-0 cursor-pointer items-start gap-2 rounded-lg border border-dashed border-slate-300 bg-white px-3 py-2 text-left outline-none transition-colors duration-150 hover:border-slate-400 hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <Icon size={14} className="mt-0.5 shrink-0 text-slate-500 group-hover:text-slate-800" />
          <span className="flex min-w-0 flex-col">
            <span className="text-xs font-semibold text-slate-800">{template.label}</span>
            <span className="text-micro leading-4 text-slate-500">{template.description}</span>
          </span>
        </button>
      );
    })}
  </div>
);

const Edit = (props: {
  item: AccessP.Policy;
  onUpdate: (item: AccessP.Policy) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Policy.clone(item));
  const ruleKeys = React.useRef(item.spec?.rules.map(() => crypto.randomUUID()) ?? []);
  const openKeys = React.useRef(
    new Set<string>((item.spec?.rules.length ?? 0) <= 1 ? ruleKeys.current : []),
  );
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(AccessP.Policy.clone(item));
    ruleKeys.current = item.spec?.rules.map(() => crypto.randomUUID()) ?? [];
    openKeys.current = new Set((item.spec?.rules.length ?? 0) <= 1 ? ruleKeys.current : []);
  }, [itemKey]);

  const updateReq = () => {
    const next = AccessP.Policy.clone(req);
    setReq(next);
    onUpdate(AccessP.Policy.clone(next));
  };

  if (!req.spec) req.spec = AccessP.Policy_Spec.create();
  const rules = req.spec.rules;

  const addRule = (rule: AccessP.Policy_Spec_Rule) => {
    const key = crypto.randomUUID();
    rules.push(rule);
    ruleKeys.current.push(key);
    openKeys.current.add(key);
    updateReq();
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <div
        className={twMerge(
          "flex items-start gap-3 rounded-xl border px-4 py-3",
          req.spec.isDisabled ? "border-amber-200 bg-amber-50/60" : "border-slate-200 bg-white",
        )}
      >
        <Switch
          checked={!req.spec.isDisabled}
          aria-label="Policy enabled"
          onChange={(event) => {
            req.spec!.isDisabled = !event.target.checked;
            updateReq();
          }}
        />
        <span className="flex min-w-0 flex-col">
          <span className="text-body font-semibold text-slate-800">
            {req.spec.isDisabled ? "Disabled" : "Enabled"}
          </span>
          <span className="text-xs leading-5 text-slate-500">
            {req.spec.isDisabled
              ? "Its rules are skipped: matching requests fall through to the rules of other Policies, or are rejected when nothing matches."
              : "Its rules take part in deciding every new access request."}
          </span>
        </span>
      </div>

      <div className="flex items-start gap-2.5 rounded-xl border border-blue-100 bg-blue-50/50 px-4 py-3 text-xs leading-5 text-slate-700">
        <Info size={14} className="mt-0.5 shrink-0 text-blue-700" />
        <span>
          When a request is created, the rules of every enabled Policy are
          checked from the highest priority to the lowest (then by Policy name,
          then in the order below). <b>The first rule that matches decides</b>.
          A request that matches no rule is rejected.
        </span>
      </div>

      <section className="flex flex-col gap-3">
        <header className="flex items-center gap-2">
          <ListOrdered size={14} className="text-slate-500" />
          <h3 className="text-body font-semibold text-slate-900">
            Rules
            <span className="ml-1.5 font-normal text-slate-500">{rules.length}</span>
          </h3>
        </header>

        {rules.length === 0 && (
          <div className="flex flex-col gap-3 rounded-xl border border-dashed border-slate-300 bg-slate-50/50 px-4 py-5">
            <span className="flex flex-col">
              <span className="text-body font-semibold text-slate-800">Start with a rule</span>
              <span className="text-xs leading-5 text-slate-500">
                A Policy without rules decides nothing. Pick a starting point;
                every field can be changed afterwards.
              </span>
            </span>
            <TemplateGrid prominent onPick={(template) => addRule(template.build(rules))} />
          </div>
        )}

        {rules.map((rule, index) => (
          <RuleCard
            key={ruleKeys.current[index]}
            rule={rule}
            index={index}
            rules={rules}
            defaultOpen={openKeys.current.has(ruleKeys.current[index])}
            onUpdate={updateReq}
            onMove={(delta) => {
              const target = index + delta;
              if (target < 0 || target >= rules.length) return;
              const [moved] = rules.splice(index, 1);
              rules.splice(target, 0, moved);
              const [key] = ruleKeys.current.splice(index, 1);
              ruleKeys.current.splice(target, 0, key);
              updateReq();
            }}
            onDuplicate={() => {
              const copy = AccessP.Policy_Spec_Rule.clone(rule);
              copy.name = uniqueName(rules, `${rule.name || "rule"}-copy`);
              const key = crypto.randomUUID();
              rules.splice(index + 1, 0, copy);
              ruleKeys.current.splice(index + 1, 0, key);
              openKeys.current.add(key);
              updateReq();
            }}
            onRemove={() => {
              rules.splice(index, 1);
              ruleKeys.current.splice(index, 1);
              updateReq();
            }}
          />
        ))}

        {rules.length > 0 && (
          <div className="flex flex-col gap-2">
            <span className="flex items-center gap-1.5 text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
              <Plus size={11} strokeWidth={2.6} /> Add a rule
            </span>
            <TemplateGrid onPick={(template) => addRule(template.build(rules))} />
          </div>
        )}
      </section>
    </div>
  );
};

export default Edit;
