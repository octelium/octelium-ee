import * as CoreP from "@/apis/corev1/corev1";
import { Struct } from "@/apis/google/protobuf/struct";
import {
  clearAgentApplyTarget,
  setAgentApplyTarget,
} from "@/components/AgentDock/applyTarget";
import { useAgentAsk, useOptionalAgent } from "@/components/AgentDock/context";
import { RECIPE_ICONS } from "@/components/Condition/meta";
import {
  cloneNode,
  compileNode,
  conditionIssues,
  isNodeComplete,
  nodeCEL,
  parseCondition,
  splitCEL,
  type ConditionNode,
} from "@/components/Condition/model";
import {
  STARTERS,
  type Recipe,
  type Starter,
} from "@/components/Condition/recipes";
import { twMerge } from "@/lib/utils";
import { useFormIssues } from "@/utils/forms";
import { Button, Collapse, JsonInput, Menu, Switch } from "@mantine/core";
import {
  ChevronDown,
  FilePlus2,
  FlaskConical,
  Info,
  LayoutTemplate,
  ListOrdered,
  Plus,
  Power,
  Sparkles,
  Undo2,
  Variable,
  X,
} from "lucide-react";
import * as React from "react";
import {
  extractCEL,
  parseSpec,
  policyContext,
  policyPrompt,
  POLICY_ACTIONS,
  RULE_ACTIONS,
  rulePrompt,
  specToYAML,
  type PolicyAIAction,
  type RuleAIAction,
} from "./edit/ai";
import AskAIDialog from "./edit/AskAI";
import { RuleCard } from "./edit/RuleCard";

const TryPolicy = React.lazy(() => import("./edit/Try"));

type Group = "rules" | "enforcement";
type Row = { key: string; node?: ConditionNode };
type Rows = Record<Group, Row[]>;

const Effect = CoreP.Policy_Spec_Rule_Effect;
const EnforcementEffect = CoreP.Policy_Spec_EnforcementRule_Effect;

const newKey = () =>
  globalThis.crypto?.randomUUID?.() ?? `rule-${Date.now()}-${Math.random()}`;

const rowsOf = (spec?: CoreP.Policy_Spec): Rows => ({
  rules: (spec?.rules ?? []).map((rule) => ({
    key: newKey(),
    node: parseCondition(rule.condition),
  })),
  enforcement: (spec?.enforcementRules ?? []).map((rule) => ({
    key: newKey(),
    node: parseCondition(rule.condition),
  })),
});

const uniqueName = (rules: CoreP.Policy_Spec_Rule[], base: string) => {
  const names = new Set(rules.map((rule) => rule.name));
  if (!names.has(base)) return base;
  for (let i = 2; ; i++) if (!names.has(`${base}-${i}`)) return `${base}-${i}`;
};

const attrsText = (attrs?: Struct) =>
  attrs && Object.keys(attrs.fields).length > 0
    ? JSON.stringify(Struct.toJson(attrs), null, 2)
    : "";

const effectName = (kind: Group, value: number) =>
  kind === "rules"
    ? value === Effect.ALLOW
      ? "ALLOW"
      : value === Effect.DENY
        ? "DENY"
        : ""
    : value === EnforcementEffect.ENFORCE
      ? "ENFORCE"
      : value === EnforcementEffect.IGNORE
        ? "IGNORE"
        : "";

const RULE_EXAMPLES = [
  "Members of the sre group, from a managed device",
  "Only GET requests under /api/reports",
  "Weekdays from 08:00 to 18:00 in Europe/Paris",
  "Sessions authenticated with a passkey from Germany or France",
];

const POLICY_EXAMPLES = [
  "Let the developers group use every Service of the staging Namespace, read-only on production",
  "Auditors can read every Octelium API resource but change nothing",
  "Contractors may reach the billing Service until the end of the year, on business hours only",
  "Deny MCP tools that delete anything, unless the User is in the platform-admins group",
];

const StarterGrid = (props: {
  prominent?: boolean;
  onPick: (starter: Starter) => void;
}) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-2 @lg:grid-cols-2",
      props.prominent && "@4xl:grid-cols-3",
    )}
  >
    {STARTERS.map((starter) => {
      const Icon = RECIPE_ICONS[starter.icon];
      return (
        <button
          key={starter.id}
          type="button"
          onClick={() => props.onPick(starter)}
          className="group flex min-w-0 cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-left shadow-card outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span
            className={twMerge(
              "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border",
              starter.tone === "deny"
                ? "border-red-200 bg-red-50 text-red-700"
                : starter.tone === "allow"
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                  : "border-slate-200 bg-slate-100 text-slate-700",
            )}
          >
            <Icon size={15} strokeWidth={2.2} />
          </span>
          <span className="flex min-w-0 flex-col gap-0.5">
            <span className="text-xs font-semibold text-slate-800 group-hover:text-slate-950">
              {starter.title}
            </span>
            <span className="text-micro leading-4 text-slate-500">
              {starter.description}
            </span>
          </span>
        </button>
      );
    })}
  </div>
);

const SectionHeader = (props: {
  icon: typeof ListOrdered;
  title: string;
  count?: number;
  badge?: string;
  hint?: React.ReactNode;
  actions?: React.ReactNode;
}) => (
  <header className="flex flex-wrap items-start gap-x-3 gap-y-2">
    <div className="flex min-w-0 flex-1 items-start gap-2">
      <props.icon size={14} className="mt-0.5 shrink-0 text-slate-500" />
      <div className="flex min-w-0 flex-col">
        <h3 className="text-body font-semibold text-slate-900">
          <span>{props.title}</span>
          {props.count !== undefined && (
            <span className="ml-1.5 font-normal text-slate-500">
              {props.count}
            </span>
          )}
          {props.badge && (
            <span className="ml-2 rounded-md bg-slate-100 px-1.5 py-px align-middle text-micro font-semibold text-slate-500">
              {props.badge}
            </span>
          )}
        </h3>
        {props.hint && (
          <p className="text-xs leading-5 text-slate-500">{props.hint}</p>
        )}
      </div>
    </div>
    {props.actions && (
      <div className="flex shrink-0 flex-wrap items-center gap-1.5">
        {props.actions}
      </div>
    )}
  </header>
);

type Draft = {
  policy: CoreP.Policy;
  rows: Rows;
  attrs: string;
  attrsError?: string;
};

type State = Draft & {
  version: number;
  undo?: { label: string; previous: Draft };
};

const draftOf = (state: Draft): Draft => ({
  policy: state.policy,
  rows: state.rows,
  attrs: state.attrs,
  attrsError: state.attrsError,
});

const initialState = (item: CoreP.Policy): State => ({
  policy: CoreP.Policy.clone(item),
  rows: rowsOf(item.spec),
  attrs: attrsText(item.spec?.attrs),
  version: 0,
});

const specOf = (draft: Draft) => {
  if (!draft.policy.spec) draft.policy.spec = CoreP.Policy_Spec.create();
  return draft.policy.spec;
};

const listOf = (draft: Draft, group: Group) =>
  group === "rules" ? specOf(draft).rules : specOf(draft).enforcementRules;

const indexOf = (draft: Draft, group: Group, key: string) =>
  draft.rows[group].findIndex((row) => row.key === key);

const setNodeIn = (
  draft: Draft,
  group: Group,
  key: string,
  node?: ConditionNode,
) => {
  const index = indexOf(draft, group, key);
  if (index < 0) return;
  draft.rows[group][index] = { key, node };
  const item = listOf(draft, group)[index];
  if (!node) item.condition = undefined;
  else if (isNodeComplete(node)) item.condition = compileNode(node);
};

const ruleLabel = (rule: CoreP.Policy_Spec_Rule, index: number) =>
  rule.name ? `Rule “${rule.name}”` : `Rule ${index + 1}`;

const openKeys = (rows: Rows) =>
  new Set(rows.rules.length <= 1 ? rows.rules.map((row) => row.key) : []);

const Edit = (props: {
  item: CoreP.Policy;
  onUpdate: (item: CoreP.Policy) => void;
}) => {
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;
  const [state, setState] = React.useState(() => initialState(props.item));
  const [seen, setSeen] = React.useState(itemKey);
  const [open, setOpen] = React.useState(() => openKeys(state.rows));
  const [highlight, setHighlight] = React.useState<Set<string>>(new Set());
  const [dialog, setDialog] = React.useState<
    { group: Group; key: string } | "policy"
  >();
  const [showAttrs, setShowAttrs] = React.useState(() => state.attrs !== "");
  const [showEnforcement, setShowEnforcement] = React.useState(
    () => state.rows.enforcement.length > 0,
  );
  const [showTry, setShowTry] = React.useState(false);
  const [showHelp, setShowHelp] = React.useState(false);
  const targets = React.useRef(new Set<string>());
  const onUpdate = React.useRef(props.onUpdate);
  const latest = React.useRef(state);

  if (seen !== itemKey) {
    const next = initialState(props.item);
    setSeen(itemKey);
    setState(next);
    setOpen(openKeys(next.rows));
  }

  React.useEffect(() => {
    onUpdate.current = props.onUpdate;
    latest.current = state;
  });

  React.useEffect(() => {
    if (state.version > 0) onUpdate.current(CoreP.Policy.clone(state.policy));
  }, [state.policy, state.version]);

  React.useEffect(() => {
    const ids = targets.current;
    return () => ids.forEach((id) => clearAgentApplyTarget(id));
  }, []);

  const agent = useOptionalAgent();
  const { available, ask } = useAgentAsk();
  const aiUsable =
    !!agent &&
    !agent.hidden &&
    agent.status !== "unavailable" &&
    agent.status !== "loading";

  const change = (update: (draft: Draft) => void, undoLabel?: string) =>
    setState((prev) => {
      const next: Draft = {
        policy: CoreP.Policy.clone(prev.policy),
        rows: {
          rules: [...prev.rows.rules],
          enforcement: [...prev.rows.enforcement],
        },
        attrs: prev.attrs,
        attrsError: prev.attrsError,
      };
      update(next);
      return {
        ...next,
        version: prev.version + 1,
        undo: undoLabel
          ? { label: undoLabel, previous: draftOf(prev) }
          : prev.undo,
      };
    });

  const restore = () =>
    setState((prev) =>
      prev.undo ? { ...prev.undo.previous, version: prev.version + 1 } : prev,
    );

  const dismissUndo = () => setState((prev) => ({ ...prev, undo: undefined }));

  const expand = (...keys: string[]) =>
    setOpen((current) => new Set([...current, ...keys]));

  const flash = (keys: string[]) => {
    setHighlight(new Set(keys));
    window.setTimeout(() => setHighlight(new Set()), 2400);
  };

  const setNode = (group: Group, key: string, node?: ConditionNode) =>
    change((draft) => setNodeIn(draft, group, key, node));

  const updateItem = (
    group: Group,
    key: string,
    update: (
      item: CoreP.Policy_Spec_Rule | CoreP.Policy_Spec_EnforcementRule,
      draft: Draft,
    ) => void,
  ) =>
    change((draft) => {
      const index = indexOf(draft, group, key);
      if (index >= 0) update(listOf(draft, group)[index], draft);
    });

  const addRule = (rule: CoreP.Policy_Spec_Rule, node?: ConditionNode) => {
    const key = newKey();
    change((draft) => {
      if (node && isNodeComplete(node)) rule.condition = compileNode(node);
      specOf(draft).rules.push(rule);
      draft.rows.rules.push({ key, node });
    });
    expand(key);
  };

  const addEnforcement = (
    rule: CoreP.Policy_Spec_EnforcementRule,
    node?: ConditionNode,
  ) => {
    const key = newKey();
    change((draft) => {
      if (node && isNodeComplete(node)) rule.condition = compileNode(node);
      specOf(draft).enforcementRules.push(rule);
      draft.rows.enforcement.push({ key, node });
    });
    expand(key);
    setShowEnforcement(true);
  };

  const applyStarter = (starter: Starter) => {
    const built = starter.build();
    const rules = built.rules.map((item) => ({ key: newKey(), item }));
    const enforcement = built.enforcementRules.map((item) => ({
      key: newKey(),
      item,
    }));
    change((draft) => {
      for (const { key, item } of enforcement) {
        specOf(draft).enforcementRules.push(
          CoreP.Policy_Spec_EnforcementRule.create({
            effect:
              item.effect === "ENFORCE"
                ? EnforcementEffect.ENFORCE
                : EnforcementEffect.IGNORE,
            condition:
              item.node && isNodeComplete(item.node)
                ? compileNode(item.node)
                : undefined,
          }),
        );
        draft.rows.enforcement.push({ key, node: item.node });
      }
      for (const { key, item } of rules) {
        specOf(draft).rules.push(
          CoreP.Policy_Spec_Rule.create({
            name: uniqueName(specOf(draft).rules, item.name),
            effect: item.effect === "DENY" ? Effect.DENY : Effect.ALLOW,
            priority: item.priority,
            condition:
              item.node && isNodeComplete(item.node)
                ? compileNode(item.node)
                : undefined,
          }),
        );
        draft.rows.rules.push({ key, node: item.node });
      }
    });
    const keys = [...enforcement, ...rules].map(({ key }) => key);
    if (enforcement.length > 0) setShowEnforcement(true);
    expand(...keys);
    flash(keys);
  };

  const remove = (group: Group, key: string) =>
    change((draft) => {
      const index = indexOf(draft, group, key);
      if (index < 0) return;
      draft.rows[group].splice(index, 1);
      listOf(draft, group).splice(index, 1);
    });

  const duplicate = (group: Group, key: string) => {
    const copyKey = newKey();
    change((draft) => {
      const index = indexOf(draft, group, key);
      if (index < 0) return;
      const node = draft.rows[group][index].node;
      if (group === "rules") {
        const rules = specOf(draft).rules;
        const copy = CoreP.Policy_Spec_Rule.clone(rules[index]);
        copy.name = copy.name ? uniqueName(rules, `${copy.name}-copy`) : "";
        rules.splice(index + 1, 0, copy);
      } else {
        const rules = specOf(draft).enforcementRules;
        rules.splice(
          index + 1,
          0,
          CoreP.Policy_Spec_EnforcementRule.clone(rules[index]),
        );
      }
      draft.rows[group].splice(index + 1, 0, {
        key: copyKey,
        node: node ? cloneNode(node) : undefined,
      });
    });
    expand(copyKey);
  };

  const setAttrs = (text: string) =>
    change((draft) => {
      draft.attrs = text;
      if (!text.trim()) {
        specOf(draft).attrs = undefined;
        draft.attrsError = undefined;
        return;
      }
      try {
        const value = JSON.parse(text);
        if (!value || typeof value !== "object" || Array.isArray(value)) {
          draft.attrsError = "Use a JSON object";
          return;
        }
        specOf(draft).attrs = Struct.fromJson(value);
        draft.attrsError = undefined;
      } catch {
        draft.attrsError = "Invalid JSON";
      }
    });

  const replaceSpec = (next: CoreP.Policy_Spec) => {
    const rows = rowsOf(next);
    change((draft) => {
      const spec = specOf(draft);
      spec.rules = CoreP.Policy_Spec.clone(next).rules;
      spec.enforcementRules = CoreP.Policy_Spec.clone(next).enforcementRules;
      if (next.attrs) {
        spec.attrs = next.attrs;
        draft.attrs = attrsText(next.attrs);
        draft.attrsError = undefined;
      }
      draft.rows = rows;
    }, "the rules of this Policy");
    const keys = [...rows.rules, ...rows.enforcement].map((row) => row.key);
    setOpen(new Set(keys));
    if (rows.enforcement.length > 0) setShowEnforcement(true);
    if (next.attrs) setShowAttrs(true);
    flash(keys);
  };

  const spec = state.policy.spec ?? CoreP.Policy_Spec.create();
  const policyName = state.policy.metadata?.name || "";

  const send = (text: string) => {
    if (!available) {
      agent?.open();
      return;
    }
    ask({
      text,
      send: true,
      fresh: true,
      pinned: {
        label: policyName ? `Policy ${policyName}` : "Policy draft",
        items: [policyContext(state.policy)],
      },
    });
  };

  const askRule = (
    group: Group,
    key: string,
    action: RuleAIAction,
    text: string,
  ) => {
    const index = state.rows[group].findIndex((row) => row.key === key);
    if (index < 0) return;
    const item = (group === "rules" ? spec.rules : spec.enforcementRules)[
      index
    ];
    const node = state.rows[group][index].node;
    const name = group === "rules" ? (item as CoreP.Policy_Spec_Rule).name : "";
    const id = `policy-rule:${key}`;
    targets.current.add(id);
    setAgentApplyTarget({
      id,
      label:
        group === "rules"
          ? name
            ? `rule “${name}”`
            : `rule ${index + 1}`
          : `enforcement rule ${index + 1}`,
      languages: ["cel", ""],
      apply: (code) => {
        const expr = extractCEL(code);
        if (!expr) return "This code is not a complete CEL expression";
        if (!latest.current.rows[group].some((row) => row.key === key)) {
          return "The rule does not exist anymore";
        }
        change(
          (draft) => setNodeIn(draft, group, key, splitCEL(expr)),
          name ? `the condition of “${name}”` : "the rule's condition",
        );
        expand(key);
        flash([key]);
        return undefined;
      },
    });
    send(
      rulePrompt({
        action,
        text,
        kind: group === "rules" ? "rule" : "enforcement",
        name,
        index,
        effect: effectName(group, item.effect),
        priority:
          group === "rules"
            ? (item as CoreP.Policy_Spec_Rule).priority
            : undefined,
        cel: node ? nodeCEL(node) : undefined,
      }),
    );
  };

  const askPolicy = (action: PolicyAIAction, text: string) => {
    const id = "policy-spec";
    targets.current.add(id);
    setAgentApplyTarget({
      id,
      label: "this Policy",
      languages: ["yaml", "yml", "json", ""],
      apply: (code) => {
        const parsed = parseSpec(code);
        if ("error" in parsed) return parsed.error;
        replaceSpec(parsed.spec);
        return undefined;
      },
    });
    send(
      policyPrompt({
        action,
        text,
        name: policyName,
        yaml: specToYAML(state.policy.spec),
        empty: spec.rules.length === 0 && spec.enforcementRules.length === 0,
      }),
    );
  };

  const ruleIssues = (group: Group, index: number) => {
    const item = (group === "rules" ? spec.rules : spec.enforcementRules)[
      index
    ];
    const ret: string[] = [];
    if (!effectName(group, item.effect)) ret.push("Choose the effect");
    ret.push(...conditionIssues(state.rows[group][index]?.node));
    return ret;
  };

  const allIssues = [
    ...spec.rules.flatMap((rule, index) =>
      ruleIssues("rules", index).map(
        (issue) => `${ruleLabel(rule, index)}: ${issue}`,
      ),
    ),
    ...spec.enforcementRules.flatMap((_, index) =>
      ruleIssues("enforcement", index).map(
        (issue) => `Enforcement rule ${index + 1}: ${issue}`,
      ),
    ),
    ...(state.attrsError ? [`Attributes: ${state.attrsError}`] : []),
  ];
  useFormIssues(allIssues);

  const toggle = (key: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });

  const dialogRow =
    dialog && dialog !== "policy"
      ? {
          index: state.rows[dialog.group].findIndex(
            (row) => row.key === dialog.key,
          ),
          ...dialog,
        }
      : undefined;
  const dialogName =
    dialogRow && dialogRow.group === "rules" && dialogRow.index >= 0
      ? spec.rules[dialogRow.index]?.name
      : "";

  const renderCard = (group: Group, row: Row, index: number) => {
    const item = (group === "rules" ? spec.rules : spec.enforcementRules)[
      index
    ];
    const rule =
      group === "rules" ? (item as CoreP.Policy_Spec_Rule) : undefined;
    return (
      <RuleCard
        key={row.key}
        kind={group === "rules" ? "rule" : "enforcement"}
        index={index}
        name={rule?.name ?? ""}
        effect={effectName(group, item.effect)}
        priority={rule?.priority}
        node={row.node}
        issues={ruleIssues(group, index)}
        open={open.has(row.key)}
        highlight={highlight.has(row.key)}
        duplicateName={
          !!rule?.name &&
          spec.rules.filter((other) => other.name === rule.name).length > 1
        }
        onToggle={() => toggle(row.key)}
        onEffect={(effect) =>
          updateItem(group, row.key, (current) => {
            current.effect =
              group === "rules"
                ? Effect[effect as "ALLOW" | "DENY"]
                : EnforcementEffect[effect as "ENFORCE" | "IGNORE"];
          })
        }
        onName={
          rule
            ? (name) =>
                updateItem(group, row.key, (current) => {
                  (current as CoreP.Policy_Spec_Rule).name = name;
                })
            : undefined
        }
        onPriority={
          rule
            ? (priority) =>
                updateItem(group, row.key, (current) => {
                  (current as CoreP.Policy_Spec_Rule).priority = priority;
                })
            : undefined
        }
        onNode={(node) => setNode(group, row.key, node)}
        onRecipe={
          rule
            ? (recipe: Recipe) =>
                updateItem(group, row.key, (current, draft) => {
                  const target = current as CoreP.Policy_Spec_Rule;
                  if (target.name) return;
                  target.name = uniqueName(specOf(draft).rules, recipe.name);
                  if (recipe.effect) target.effect = Effect[recipe.effect];
                })
            : undefined
        }
        onAskAI={
          aiUsable ? () => setDialog({ group, key: row.key }) : undefined
        }
        onDuplicate={() => duplicate(group, row.key)}
        onRemove={() => remove(group, row.key)}
      />
    );
  };

  const empty = spec.rules.length === 0;

  return (
    <div className="@container flex w-full min-w-0 flex-col gap-5">
      <div
        className={twMerge(
          "flex items-start gap-3 rounded-xl border px-4 py-3",
          spec.isDisabled
            ? "border-amber-200 bg-amber-50/60"
            : "border-slate-200 bg-white",
        )}
      >
        <Switch
          checked={!spec.isDisabled}
          aria-label="Policy enabled"
          onChange={(event) => {
            const enabled = event.currentTarget.checked;
            change((draft) => {
              specOf(draft).isDisabled = !enabled;
            });
          }}
        />
        <span className="flex min-w-0 flex-col">
          <span className="text-body font-semibold text-slate-800">
            {spec.isDisabled ? "Disabled" : "Enabled"}
          </span>
          <span className="text-xs leading-5 text-slate-500">
            {spec.isDisabled
              ? "Its rules are skipped everywhere it is attached, without having to detach it."
              : "Its rules take part in every decision where the Policy is attached."}
          </span>
        </span>
      </div>

      {aiUsable && (
        <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 @2xl:flex-row @2xl:items-center">
          <div className="flex min-w-0 flex-1 items-start gap-3">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
              <Sparkles size={15} strokeWidth={2.25} />
            </span>
            <span className="flex min-w-0 flex-col">
              <span className="text-body font-semibold text-slate-800">
                Build it with AI
              </span>
              <span className="text-xs leading-5 text-slate-500">
                Describe the access you need in plain words. The AI drafts,
                explains or reviews the rules, and you apply its proposal here.
              </span>
            </span>
          </div>
          <div className="flex shrink-0 flex-wrap gap-1.5">
            <Button
              size="compact-sm"
              color="ink"
              leftSection={<Sparkles size={12} />}
              onClick={() => setDialog("policy")}
            >
              {empty ? "Draft with AI" : "Ask AI"}
            </Button>
            {!empty && (
              <Button
                size="compact-sm"
                variant="default"
                onClick={() => askPolicy("review", "")}
              >
                Review
              </Button>
            )}
          </div>
        </div>
      )}

      {state.undo && (
        <div
          role="status"
          className="flex flex-wrap items-center gap-2 rounded-xl border border-blue-200 bg-blue-50/70 px-4 py-2.5"
        >
          <Sparkles size={13} className="shrink-0 text-blue-700" />
          <span className="min-w-0 flex-1 text-xs text-slate-700">
            The AI's proposal replaced {state.undo.label}. Review it before
            saving.
          </span>
          <Button
            size="compact-xs"
            variant="default"
            leftSection={<Undo2 size={12} />}
            onClick={restore}
          >
            Undo
          </Button>
          <Button
            size="compact-xs"
            variant="subtle"
            color="gray"
            aria-label="Dismiss"
            onClick={dismissUndo}
          >
            <X size={12} />
          </Button>
        </div>
      )}

      <section className="flex flex-col gap-3">
        <SectionHeader
          icon={ListOrdered}
          title="Rules"
          count={spec.rules.length}
          actions={
            !empty && (
              <Button
                size="compact-xs"
                variant="subtle"
                color="gray"
                leftSection={<Info size={12} />}
                onClick={() => setShowHelp((value) => !value)}
                aria-expanded={showHelp}
              >
                How rules decide
              </Button>
            )
          }
        />

        <Collapse expanded={showHelp || empty}>
          <div className="flex items-start gap-2.5 rounded-xl border border-blue-100 bg-blue-50/50 px-4 py-3 text-xs leading-5 text-slate-700">
            <Info size={14} className="mt-0.5 shrink-0 text-blue-700" />
            <span>
              For every request, the rules of all the Policies that apply
              (attached to the User, its Groups, the Service, its Namespace…)
              are checked together from priority −4 to 4, Deny before Allow at
              the same priority. <b>The first rule that matches decides</b>; a
              request that matches no rule is denied.
            </span>
          </div>
        </Collapse>

        {empty && (
          <div className="flex flex-col gap-3 rounded-xl border border-dashed border-slate-300 bg-slate-50/50 p-4">
            <span className="flex flex-col">
              <span className="text-body font-semibold text-slate-800">
                Start with a template
              </span>
              <span className="text-xs leading-5 text-slate-500">
                Each template adds ready-made rules. Every value can be changed
                afterwards, and the rules can be edited visually or as CEL.
              </span>
            </span>
            <StarterGrid prominent onPick={applyStarter} />
            <div className="flex flex-wrap gap-2">
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<FilePlus2 size={13} />}
                onClick={() =>
                  addRule(
                    CoreP.Policy_Spec_Rule.create({ effect: Effect.ALLOW }),
                  )
                }
              >
                Blank rule
              </Button>
              {aiUsable && (
                <Button
                  variant="subtle"
                  color="gray"
                  size="compact-sm"
                  leftSection={<Sparkles size={13} />}
                  onClick={() => setDialog("policy")}
                >
                  Describe it to the AI
                </Button>
              )}
            </div>
          </div>
        )}

        {state.rows.rules.map((row, index) => renderCard("rules", row, index))}

        {!empty && (
          <div className="flex flex-wrap gap-1.5">
            <Button
              variant="default"
              size="compact-sm"
              leftSection={<Plus size={13} strokeWidth={2.5} />}
              onClick={() =>
                addRule(CoreP.Policy_Spec_Rule.create({ effect: Effect.ALLOW }))
              }
            >
              Add rule
            </Button>
            <Menu position="bottom-start" withinPortal shadow="md" width={300}>
              <Menu.Target>
                <Button
                  variant="subtle"
                  color="gray"
                  size="compact-sm"
                  leftSection={<LayoutTemplate size={13} />}
                >
                  Add from a template
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                {STARTERS.map((starter) => {
                  const Icon = RECIPE_ICONS[starter.icon];
                  return (
                    <Menu.Item
                      key={starter.id}
                      leftSection={<Icon size={14} />}
                      onClick={() => applyStarter(starter)}
                    >
                      <span className="flex flex-col">
                        <span className="text-xs font-semibold">
                          {starter.title}
                        </span>
                        <span className="text-micro text-slate-500">
                          {starter.description}
                        </span>
                      </span>
                    </Menu.Item>
                  );
                })}
              </Menu.Dropdown>
            </Menu>
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <SectionHeader
          icon={Power}
          title="Enforcement rules"
          count={spec.enforcementRules.length || undefined}
          badge="Advanced"
          hint="Decide when this Policy takes part at all, e.g. only for HTTP Services. Without enforcement rules it always does."
          actions={
            !showEnforcement && (
              <Button
                size="compact-xs"
                variant="subtle"
                color="gray"
                rightSection={<ChevronDown size={12} />}
                onClick={() => setShowEnforcement(true)}
              >
                Show
              </Button>
            )
          }
        />
        <Collapse expanded={showEnforcement}>
          <div className="flex flex-col gap-3">
            {spec.enforcementRules.length > 0 && (
              <p className="text-micro leading-4 text-slate-500">
                If an Enforce rule matches, the Policy is evaluated. Otherwise,
                if an Ignore rule matches, it is skipped.
              </p>
            )}
            {state.rows.enforcement.map((row, index) =>
              renderCard("enforcement", row, index),
            )}
            <Button
              className="w-fit"
              variant="default"
              size="compact-sm"
              leftSection={<Plus size={13} strokeWidth={2.5} />}
              onClick={() =>
                addEnforcement(
                  CoreP.Policy_Spec_EnforcementRule.create({
                    effect: EnforcementEffect.ENFORCE,
                  }),
                )
              }
            >
              Add enforcement rule
            </Button>
          </div>
        </Collapse>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHeader
          icon={Variable}
          title="Attributes"
          badge="Advanced"
          hint={
            <>
              Values the rules can read as{" "}
              <code className="font-mono">attrs</code>, e.g.{" "}
              <code className="font-mono">
                ctx.request.http.path in attrs.allowedPaths
              </code>
              .
            </>
          }
          actions={
            !showAttrs && (
              <Button
                size="compact-xs"
                variant="subtle"
                color="gray"
                rightSection={<ChevronDown size={12} />}
                onClick={() => setShowAttrs(true)}
              >
                Show
              </Button>
            )
          }
        />
        <Collapse expanded={showAttrs}>
          <JsonInput
            aria-label="Attributes"
            placeholder={'{\n  "allowedPaths": ["/api/reports"]\n}'}
            autosize
            minRows={4}
            maxRows={16}
            formatOnBlur
            error={state.attrsError}
            value={state.attrs}
            onChange={setAttrs}
          />
        </Collapse>
      </section>

      <section className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3">
        <button
          type="button"
          aria-expanded={showTry}
          onClick={() => setShowTry((value) => !value)}
          className="flex cursor-pointer items-start gap-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-600">
            <FlaskConical size={15} />
          </span>
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="text-body font-semibold text-slate-800">
              Try this draft
            </span>
            <span className="text-xs leading-5 text-slate-500">
              Check whether a User, Session or Device would reach a Service with
              these rules, before saving.
            </span>
          </span>
          <ChevronDown
            size={14}
            className={twMerge(
              "mt-2 shrink-0 text-slate-500 transition-transform",
              !showTry && "-rotate-90",
            )}
          />
        </button>
        {showTry && (
          <div className="border-t border-slate-100 pt-3">
            <React.Suspense
              fallback={
                <div className="h-24 rounded-lg bg-slate-100 motion-safe:animate-pulse" />
              }
            >
              <TryPolicy
                policy={state.policy}
                blocked={
                  allIssues.length > 0
                    ? "Fix the issues of the draft first."
                    : undefined
                }
              />
            </React.Suspense>
          </div>
        )}
      </section>

      {dialog === "policy" && (
        <AskAIDialog
          opened
          title={
            empty ? "Draft this Policy with AI" : "Ask AI about this Policy"
          }
          subject={policyName ? `Policy ${policyName}` : "New Policy"}
          actions={
            empty
              ? POLICY_ACTIONS.filter((item) => item.id === "draft")
              : POLICY_ACTIONS
          }
          placeholder="Who should access what, how and when? Mention Groups, Services, Namespaces, times, devices…"
          examples={POLICY_EXAMPLES}
          applyLabel="this Policy"
          onClose={() => setDialog(undefined)}
          onSend={(action, text) => {
            setDialog(undefined);
            askPolicy(action as PolicyAIAction, text);
          }}
        />
      )}
      {dialogRow && dialogRow.index >= 0 && (
        <AskAIDialog
          opened
          title={
            dialogRow.group === "rules"
              ? `Ask AI about ${dialogName ? `rule “${dialogName}”` : `rule ${dialogRow.index + 1}`}`
              : `Ask AI about enforcement rule ${dialogRow.index + 1}`
          }
          subject={policyName ? `Policy ${policyName}` : "New Policy"}
          actions={
            state.rows[dialogRow.group][dialogRow.index]?.node
              ? RULE_ACTIONS
              : RULE_ACTIONS.filter((item) => item.id === "write")
          }
          placeholder="Which requests should this rule match? e.g. members of sre using a passkey, on weekdays"
          examples={RULE_EXAMPLES}
          applyLabel="the rule"
          onClose={() => setDialog(undefined)}
          onSend={(action, text) => {
            setDialog(undefined);
            askRule(
              dialogRow.group,
              dialogRow.key,
              action as RuleAIAction,
              text,
            );
          }}
        />
      )}
    </div>
  );
};

export default Edit;
