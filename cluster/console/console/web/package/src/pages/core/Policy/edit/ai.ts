import * as CoreP from "@/apis/corev1/corev1";
import { looksComplete } from "@/components/Condition/cel";
import Yaml from "js-yaml";

export type RuleAIAction = "write" | "explain" | "review" | "stricter";
export type PolicyAIAction = "draft" | "review" | "explain" | "tests";

const EVALUATION = [
  "How Octelium evaluates core Policies: the rules of every Policy that applies to a request (attached to the User, its Groups, the Session, the Device, the Service, its Namespace or Cluster-wide) are sorted by priority from -4 (first) to 4 (last), DENY before ALLOW at equal priority, and the first rule whose condition matches decides. A request that matches no rule is denied.",
  "A Policy's enforcement rules decide whether its rules are evaluated at all: if an ENFORCE rule matches, the Policy is evaluated; otherwise if an IGNORE rule matches, the Policy is skipped; with no enforcement rules it is always evaluated.",
  'Conditions are CEL expressions over the request context `ctx` (ctx.user, ctx.groups, ctx.session, ctx.device, ctx.service, ctx.namespace and ctx.request with protocol-specific fields such as ctx.request.http, ctx.request.grpc, ctx.request.ssh, ctx.request.kubernetes, ctx.request.postgres, ctx.request.mcp or ctx.request.llm) and over the Policy attributes `attrs`. Besides standard CEL, the functions now(), net.isIPInRange(ip, cidr), net.isPrivateIP(ip), time.isWeekdayInTZ(ts, tz), json.parse(s) are available, and timestamps support getHours("Europe/Berlin"), getDayOfWeek(tz) (0 is Sunday). An expression that reads a missing field does not match.',
].join("\n");

const LOOKUP =
  "Use the names (metadata.name) of existing Users, Groups, Services and Namespaces; look them up with the Octelium API when the description refers to them, and do not invent names.";

export const specToYAML = (spec?: CoreP.Policy_Spec) =>
  Yaml.dump(
    CoreP.Policy_Spec.toJson(spec ?? CoreP.Policy_Spec.create(), {
      emitDefaultValues: false,
    }) as object,
    { lineWidth: 120, noRefs: true },
  );

export const policyContext = (policy: CoreP.Policy) => ({
  kind: "PolicyDraft",
  id: policy.metadata?.uid || undefined,
  title: policy.metadata?.name || "New Policy",
  data: {
    name: policy.metadata?.name || undefined,
    description: policy.metadata?.description || undefined,
    saved: !!policy.metadata?.uid,
    spec: CoreP.Policy_Spec.toJson(policy.spec ?? CoreP.Policy_Spec.create(), {
      emitDefaultValues: false,
    }),
  },
});

const ruleLabel = (name: string, index: number) =>
  name ? `the rule "${name}"` : `rule #${index + 1}`;

export const RULE_ACTIONS: {
  id: RuleAIAction;
  label: string;
  hint: string;
  needsText: boolean;
}[] = [
  {
    id: "write",
    label: "Write the condition",
    hint: "Describe the requests this rule should match",
    needsText: true,
  },
  {
    id: "explain",
    label: "Explain",
    hint: "In plain words, with examples",
    needsText: false,
  },
  {
    id: "review",
    label: "Review",
    hint: "Find mistakes, gaps and risky cases",
    needsText: false,
  },
  {
    id: "stricter",
    label: "Make it stricter",
    hint: "Narrow it to what is needed",
    needsText: false,
  },
];

export const POLICY_ACTIONS: {
  id: PolicyAIAction;
  label: string;
  hint: string;
  needsText: boolean;
}[] = [
  {
    id: "draft",
    label: "Draft the rules",
    hint: "Describe who may access what, and when",
    needsText: true,
  },
  {
    id: "review",
    label: "Review",
    hint: "Mistakes, unintended access, shadowed rules",
    needsText: false,
  },
  {
    id: "explain",
    label: "Explain",
    hint: "What this Policy allows and denies",
    needsText: false,
  },
  {
    id: "tests",
    label: "Suggest test cases",
    hint: "Requests to verify with the Policy tester",
    needsText: false,
  },
];

export const rulePrompt = (args: {
  action: RuleAIAction;
  text: string;
  kind: "rule" | "enforcement";
  name: string;
  index: number;
  effect: string;
  priority?: number;
  cel?: string;
}) => {
  const subject =
    args.kind === "rule"
      ? `${ruleLabel(args.name, args.index)} of the Policy draft in the console context (effect ${args.effect}, priority ${args.priority ?? 0})`
      : `enforcement rule #${args.index + 1} of the Policy draft in the console context (effect ${args.effect})`;
  const current = args.cel
    ? `Its current condition, as CEL:\n\`\`\`cel\n${args.cel}\n\`\`\``
    : "It has no condition yet.";
  const reply =
    "Reply with a short explanation, then exactly one ```cel code block holding the complete condition as a single CEL expression, so that it can be applied to the rule from the console.";
  const extra = args.text.trim()
    ? `\nAdditional notes: ${args.text.trim()}`
    : "";
  switch (args.action) {
    case "write":
      return `Write the condition of ${subject}. It must match: ${args.text.trim()}\n${current}\n${LOOKUP}\n${reply}\n\n${EVALUATION}`;
    case "explain":
      return `Explain in plain words what ${subject} matches, with a few concrete example requests that match and that do not.${extra}\n${current}\n\n${EVALUATION}`;
    case "review":
      return `Review ${subject}. Point out mistakes, requests it matches unintentionally or misses, fields that may be missing for some protocols, and interactions with its effect and priority.${extra}\n${current}\nIf you recommend a change, end with the corrected condition in exactly one \`\`\`cel code block so it can be applied from the console.\n\n${EVALUATION}`;
    case "stricter":
      return `Make the condition of ${subject} stricter so it only matches what is needed, following least privilege.${extra}\n${current}\n${LOOKUP}\n${reply}\n\n${EVALUATION}`;
  }
};

export const policyPrompt = (args: {
  action: PolicyAIAction;
  text: string;
  name: string;
  yaml: string;
  empty: boolean;
}) => {
  const subject = args.name
    ? `the core Policy "${args.name}"`
    : "the new core Policy";
  const current = args.empty
    ? "It has no rules yet."
    : `Its current spec:\n\`\`\`yaml\n${args.yaml}\`\`\``;
  const format =
    "Reply with a short explanation, then exactly one ```yaml code block holding only the Policy spec (the keys rules, enforcementRules and, if useful, attrs), so that it can be applied from the console. Each rule has a name, an effect (ALLOW or DENY), a priority (-4 to 4, 0 by default) and a condition, which is one of `match: <CEL>`, `not: <CEL>`, `matchAny: true`, or `all`, `any` or `none` with `of: [<conditions>]`. Prefer one simple condition per rule, split with all/any when it reads better.";
  const extra = args.text.trim()
    ? `\nAdditional notes: ${args.text.trim()}`
    : "";
  switch (args.action) {
    case "draft":
      return `Draft ${subject}. It must: ${args.text.trim()}\n${current}\n${LOOKUP}\n${format}\n\n${EVALUATION}`;
    case "review":
      return `Review ${subject}. Point out mistakes, unintended access, rules that can never decide because another rule shadows them, priority or ALLOW/DENY interplay issues, missing enforcement rules and conditions that read fields missing for some protocols.${extra}\n${current}\nYou can verify cases with the PolicyPortalService IsAuthorized method by passing this spec in additional.inlinePolicies. If you recommend changes, end with the full revised spec in exactly one \`\`\`yaml code block (only the spec keys) so it can be applied from the console.\n\n${EVALUATION}`;
    case "explain":
      return `Explain in plain words what ${subject} allows and denies, to whom, and when, with a few concrete examples.${extra}\n${current}\n\n${EVALUATION}`;
    case "tests":
      return `Suggest test cases for ${subject}: pairs of a User (or Session or Device) and a Service, with request details when relevant, and the expected decision. Run the most important ones with the PolicyPortalService IsAuthorized method by passing this spec in additional.inlinePolicies, and report the results in a table.${extra}\n${current}\n\n${EVALUATION}`;
  }
};

export const extractCEL = (code: string): string | undefined => {
  const text = code
    .trim()
    .replace(/^`+|`+$/g, "")
    .trim();
  if (!text) return undefined;
  return looksComplete(text) ? text : undefined;
};

export const parseSpec = (
  code: string,
): { spec: CoreP.Policy_Spec } | { error: string } => {
  let value: unknown;
  try {
    value = Yaml.load(code);
  } catch (err) {
    return {
      error: `The YAML cannot be parsed: ${(err as Error).message.split("\n")[0]}`,
    };
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return { error: "The code is not a Policy spec" };
  }
  let obj = value as Record<string, unknown>;
  if (obj.spec && typeof obj.spec === "object") {
    obj = obj.spec as Record<string, unknown>;
  }
  if (!Array.isArray(obj.rules) && !Array.isArray(obj.enforcementRules)) {
    return { error: "The spec has no rules" };
  }
  let spec: CoreP.Policy_Spec;
  try {
    spec = CoreP.Policy_Spec.fromJson(obj as never, {
      ignoreUnknownFields: true,
    });
  } catch (err) {
    return { error: `The spec is invalid: ${(err as Error).message}` };
  }
  const missing = spec.rules.findIndex(
    (rule) =>
      !rule.condition?.type.oneofKind ||
      rule.effect === CoreP.Policy_Spec_Rule_Effect.EFFECT_UNKNOWN,
  );
  if (missing >= 0) {
    return { error: `Rule #${missing + 1} has no condition or effect` };
  }
  return { spec };
};
