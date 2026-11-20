import * as CoreP from "@/apis/corev1/corev1";
import { getBlock, recognize, type Values } from "./blocks";
import {
  looksComplete,
  negate,
  negatedInner,
  splitTopLevel,
  unwrapParens,
  wrapFor,
} from "./cel";

export type GroupOp = "all" | "any" | "none";
export type Negation = "condition" | "expression";

type NodeBase = { key: string; raw?: CoreP.Condition };

export type BlockNode = NodeBase & {
  kind: "block";
  block: string;
  values: Values;
  negate?: Negation;
};

export type CELNode = NodeBase & {
  kind: "cel";
  expr: string;
  negate?: boolean;
};

export type OPANode = NodeBase & { kind: "opa"; rego: string };

export type GroupNode = NodeBase & {
  kind: "group";
  op: GroupOp;
  children: ConditionNode[];
};

export type EveryNode = NodeBase & { kind: "every" };

export type ConditionNode =
  BlockNode | CELNode | OPANode | GroupNode | EveryNode;

export type LeafNode = BlockNode | CELNode;

let keyCounter = 0;

export const newKey = () =>
  globalThis.crypto?.randomUUID?.() ??
  `condition-${Date.now()}-${++keyCounter}`;

export const blockNode = (
  block: string,
  values?: Values,
  negateKind?: Negation,
): BlockNode => ({
  kind: "block",
  key: newKey(),
  block,
  values: values ?? getBlock(block)?.defaults() ?? {},
  negate: negateKind,
});

export const celNode = (expr = "", negated = false): CELNode => ({
  kind: "cel",
  key: newKey(),
  expr,
  negate: negated || undefined,
});

export const opaNode = (rego = ""): OPANode => ({
  kind: "opa",
  key: newKey(),
  rego,
});

export const everyNode = (): EveryNode => ({ kind: "every", key: newKey() });

export const groupNode = (
  op: GroupOp,
  children: ConditionNode[],
): GroupNode => ({
  kind: "group",
  key: newKey(),
  op,
  children,
});

export const leafFromExpr = (
  expr: string,
  coreNot = false,
  raw?: CoreP.Condition,
): LeafNode => {
  const direct = recognize(expr);
  if (direct) {
    return {
      kind: "block",
      key: newKey(),
      block: direct.block.id,
      values: direct.values,
      negate: coreNot ? "condition" : undefined,
      raw,
    };
  }
  if (!coreNot) {
    const inner = negatedInner(expr);
    const negated = inner !== undefined ? recognize(inner) : undefined;
    if (negated) {
      return {
        kind: "block",
        key: newKey(),
        block: negated.block.id,
        values: negated.values,
        negate: "expression",
        raw,
      };
    }
  }
  return {
    kind: "cel",
    key: newKey(),
    expr,
    negate: coreNot || undefined,
    raw,
  };
};

export const parseCondition = (
  condition?: CoreP.Condition,
): ConditionNode | undefined => {
  if (!condition) return undefined;
  const type = condition.type;
  switch (type.oneofKind) {
    case "matchAny":
      return {
        kind: "every",
        key: newKey(),
        raw: type.matchAny ? undefined : CoreP.Condition.clone(condition),
      };
    case "match":
      return leafFromExpr(type.match, false, CoreP.Condition.clone(condition));
    case "not":
      return leafFromExpr(type.not, true, CoreP.Condition.clone(condition));
    case "all":
      return groupNode(
        "all",
        type.all.of.map((item) => parseCondition(item) ?? emptyLeaf(item)),
      );
    case "any":
      return groupNode(
        "any",
        type.any.of.map((item) => parseCondition(item) ?? emptyLeaf(item)),
      );
    case "none":
      return groupNode(
        "none",
        type.none.of.map((item) => parseCondition(item) ?? emptyLeaf(item)),
      );
    case "opa":
      return {
        kind: "opa",
        key: newKey(),
        rego: type.opa.type.oneofKind === "inline" ? type.opa.type.inline : "",
        raw: CoreP.Condition.clone(condition),
      };
    default:
      return undefined;
  }
};

const emptyLeaf = (raw: CoreP.Condition): CELNode => ({
  kind: "cel",
  key: newKey(),
  expr: "",
  raw: CoreP.Condition.clone(raw),
});

export const leafExpression = (
  node: BlockNode | CELNode,
): { expr: string; negated?: Negation } => {
  if (node.kind === "cel") {
    return { expr: node.expr, negated: node.negate ? "condition" : undefined };
  }
  const block = getBlock(node.block);
  return {
    expr: block ? block.toCEL(node.values) : "false",
    negated: node.negate,
  };
};

export const compileNode = (node: ConditionNode): CoreP.Condition => {
  if (node.raw) return CoreP.Condition.clone(node.raw);
  switch (node.kind) {
    case "every":
      return CoreP.Condition.create({
        type: { oneofKind: "matchAny", matchAny: true },
      });
    case "opa":
      return CoreP.Condition.create({
        type: {
          oneofKind: "opa",
          opa: { type: { oneofKind: "inline", inline: node.rego } },
        },
      });
    case "group": {
      const of = node.children.map(compileNode);
      switch (node.op) {
        case "all":
          return CoreP.Condition.create({
            type: { oneofKind: "all", all: { of } },
          });
        case "any":
          return CoreP.Condition.create({
            type: { oneofKind: "any", any: { of } },
          });
        default:
          return CoreP.Condition.create({
            type: { oneofKind: "none", none: { of } },
          });
      }
    }
    default: {
      const { expr, negated } = leafExpression(node);
      if (negated === "condition") {
        return CoreP.Condition.create({
          type: { oneofKind: "not", not: expr },
        });
      }
      return CoreP.Condition.create({
        type: {
          oneofKind: "match",
          match: negated === "expression" ? negate(expr) : expr,
        },
      });
    }
  }
};

export const compileCondition = (node?: ConditionNode) =>
  node ? compileNode(node) : undefined;

const rawCEL = (raw: CoreP.Condition): string | undefined => {
  switch (raw.type.oneofKind) {
    case "match":
      return raw.type.match;
    case "not":
      return negate(raw.type.not);
    case "matchAny":
      return raw.type.matchAny ? "true" : "false";
    default:
      return undefined;
  }
};

export const nodeCEL = (
  node: ConditionNode,
  pretty = false,
): string | undefined => {
  if (node.raw && node.kind !== "opa") {
    const text = rawCEL(node.raw);
    if (text !== undefined) return text;
  }
  switch (node.kind) {
    case "every":
      return "true";
    case "opa":
      return undefined;
    case "group": {
      const parts = node.children.map((child) => nodeCEL(child));
      if (parts.some((part) => part === undefined)) return undefined;
      if (parts.length === 0) return node.op === "none" ? "true" : "false";
      const operator = node.op === "all" ? "&&" : "||";
      const joined = (parts as string[])
        .map((part) => wrapFor(operator, part))
        .join(
          pretty && node.op !== "none" ? `\n${operator} ` : ` ${operator} `,
        );
      return node.op === "none" ? negate(joined) : joined;
    }
    default: {
      const { expr, negated } = leafExpression(node);
      return negated ? negate(expr) : expr;
    }
  }
};

const flatten = (node: ConditionNode): ConditionNode => {
  if (node.kind !== "group") return node;
  const children = node.children.flatMap((child) => {
    const flat = flatten(child);
    return flat.kind === "group" && flat.op === node.op && node.op !== "none"
      ? flat.children
      : [flat];
  });
  return { ...node, children };
};

const splitInto = (expr: string): ConditionNode => {
  const source = unwrapParens(expr);
  const any = splitTopLevel(source, "||");
  if (any) return groupNode("any", any.map(splitInto));
  const all = splitTopLevel(source, "&&");
  if (all) return groupNode("all", all.map(splitInto));
  return leafFromExpr(
    source,
    false,
    CoreP.Condition.create({ type: { oneofKind: "match", match: source } }),
  );
};

export const splitCEL = (expr: string): ConditionNode =>
  flatten(splitInto(expr));

export const canSplit = (expr: string) =>
  splitTopLevel(expr, "||") !== undefined ||
  splitTopLevel(expr, "&&") !== undefined;

export const hasOPA = (node?: ConditionNode): boolean =>
  !!node &&
  (node.kind === "opa" ||
    (node.kind === "group" && node.children.some(hasOPA)));

export const hasCoreNegation = (node?: ConditionNode): boolean => {
  if (!node) return false;
  switch (node.kind) {
    case "group":
      return node.op === "none" || node.children.some(hasCoreNegation);
    case "block":
      return node.negate === "condition";
    case "cel":
      return !!node.negate;
    default:
      return false;
  }
};

export const nodeIssues = (node?: ConditionNode): string[] => {
  if (!node || node.raw) return [];
  switch (node.kind) {
    case "block": {
      const block = getBlock(node.block);
      return block ? block.issues(node.values) : ["Unknown condition"];
    }
    case "cel":
      if (!node.expr.trim()) return ["Write the CEL expression"];
      return looksComplete(node.expr) ? [] : ["Finish the CEL expression"];
    case "opa":
      return node.rego.trim() ? [] : ["Write the Rego policy"];
    case "group":
      return [
        ...(node.children.length === 0
          ? ["Add a condition to the empty group"]
          : []),
        ...node.children.flatMap(nodeIssues),
      ];
    default:
      return [];
  }
};

export const conditionIssues = (node?: ConditionNode) =>
  node ? Array.from(new Set(nodeIssues(node))) : ["Add a condition"];

export const isNodeComplete = (node?: ConditionNode) =>
  !!node && nodeIssues(node).length === 0;

const OP_WORD: Record<GroupOp, string> = { all: "and", any: "or", none: "or" };

export const summarize = (node?: ConditionNode, nested = false): string => {
  if (!node) return "No condition";
  switch (node.kind) {
    case "every":
      return node.raw ? "Never matches" : "Every request";
    case "opa":
      return "OPA policy";
    case "cel": {
      const text =
        node.expr.trim().replace(/\s+/g, " ") || "Empty CEL expression";
      return node.negate ? `not (${text})` : text;
    }
    case "block": {
      const block = getBlock(node.block);
      const text = block ? block.summary(node.values) : node.block;
      return node.negate ? `not (${text})` : text;
    }
    case "group": {
      if (node.children.length === 0) return "Empty group";
      const parts = node.children.map((child) => summarize(child, true));
      const joined = parts.join(` ${OP_WORD[node.op]} `);
      if (node.op === "none") return `none of (${parts.join(", ")})`;
      return nested && parts.length > 1 ? `(${joined})` : joined;
    }
  }
};

export const touch = <T extends ConditionNode>(
  node: T,
  patch: Partial<T> = {},
): T => ({
  ...node,
  ...patch,
  raw: undefined,
});

export const findNode = (
  root: ConditionNode | undefined,
  key: string,
): ConditionNode | undefined => {
  if (!root) return undefined;
  if (root.key === key) return root;
  if (root.kind !== "group") return undefined;
  for (const child of root.children) {
    const found = findNode(child, key);
    if (found) return found;
  }
  return undefined;
};

export const replaceNode = (
  root: ConditionNode | undefined,
  key: string,
  next: ConditionNode | undefined,
): ConditionNode | undefined => {
  if (!root) return undefined;
  if (root.key === key) return next;
  if (root.kind !== "group") return root;
  let changed = false;
  const children: ConditionNode[] = [];
  for (const child of root.children) {
    const replaced = replaceNode(child, key, next);
    if (replaced !== child) changed = true;
    if (replaced) children.push(replaced);
  }
  return changed ? { ...root, children } : root;
};

export const appendNodes = (
  root: ConditionNode | undefined,
  nodes: ConditionNode[],
  groupKey?: string,
): ConditionNode | undefined => {
  if (nodes.length === 0) return root;
  if (groupKey) {
    const target = findNode(root, groupKey);
    if (target?.kind === "group") {
      return replaceNode(root, groupKey, {
        ...target,
        children: [...target.children, ...nodes],
      });
    }
  }
  if (!root || root.kind === "every") {
    return nodes.length === 1 ? nodes[0] : groupNode("all", nodes);
  }
  if (root.kind === "group" && root.op === "all") {
    return { ...root, children: [...root.children, ...nodes] };
  }
  return groupNode("all", [root, ...nodes]);
};

export const cloneNode = (node: ConditionNode): ConditionNode => {
  const copy = {
    ...node,
    key: newKey(),
    raw: node.raw ? CoreP.Condition.clone(node.raw) : undefined,
  } as ConditionNode;
  if (copy.kind === "group") copy.children = copy.children.map(cloneNode);
  if (copy.kind === "block") copy.values = structuredClone(copy.values);
  return copy;
};
