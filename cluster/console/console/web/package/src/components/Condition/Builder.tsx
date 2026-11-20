import { twMerge } from "@/lib/utils";
import {
  ActionIcon,
  Button,
  Menu,
  SegmentedControl,
  Tooltip,
} from "@mantine/core";
import {
  AlertTriangle,
  Ban,
  Blocks,
  Braces,
  Check,
  ChevronDown,
  Copy,
  CopyPlus,
  Ellipsis,
  FileCode,
  Info,
  Layers,
  ListTree,
  Plus,
  Sparkles,
  Split,
  Trash2,
  Ungroup,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { getBlock, recognize } from "./blocks";
import { looksComplete } from "./cel";
import type { CELContext } from "./celSchema";
import { CELEditor } from "./CELEditor";
import { OPAEditor } from "./Editor";
import { BlockFields } from "./fields";
import {
  CATEGORY_META,
  RECIPE_ICONS,
  TONE_CHIP,
  blockIcon,
  type Tone,
} from "./meta";
import {
  appendNodes,
  canSplit,
  celNode,
  cloneNode,
  compileNode,
  everyNode,
  findNode,
  groupNode,
  hasCoreNegation,
  hasOPA,
  leafFromExpr,
  nodeCEL,
  nodeIssues,
  replaceNode,
  splitCEL,
  touch,
  type BlockNode,
  type CELNode,
  type ConditionNode,
  type GroupNode,
  type GroupOp,
} from "./model";
import ConditionPicker, { type PickerScope } from "./Picker";
import { featuredRecipes, type Recipe } from "./recipes";

type Ctx = {
  root?: ConditionNode;
  scope: PickerScope;
  context: CELContext;
  update: (key: string, next?: ConditionNode) => void;
  openPicker: (groupKey?: string) => void;
  setRoot: (next?: ConditionNode) => void;
};

const BuilderContext = React.createContext<Ctx | undefined>(undefined);

const useBuilder = () => React.useContext(BuilderContext)!;

const OP_META: Record<
  GroupOp,
  { label: string; word: string; hint: string; tone: Tone }
> = {
  all: {
    label: "All",
    word: "and",
    hint: "every condition must match",
    tone: "indigo",
  },
  any: {
    label: "Any",
    word: "or",
    hint: "at least one condition must match",
    tone: "emerald",
  },
  none: {
    label: "None",
    word: "nor",
    hint: "no condition may match",
    tone: "red",
  },
};

const parentOf = (
  root: ConditionNode | undefined,
  key: string,
): GroupNode | undefined => {
  if (!root || root.kind !== "group") return undefined;
  if (root.children.some((child) => child.key === key)) return root;
  for (const child of root.children) {
    const found = parentOf(child, key);
    if (found) return found;
  }
  return undefined;
};

const celFromLeaf = (node: BlockNode | CELNode): CELNode => {
  if (node.kind === "cel") return node;
  const compiled = compileNode(node);
  if (compiled.type.oneofKind === "not") {
    return {
      kind: "cel",
      key: node.key,
      expr: compiled.type.not,
      negate: true,
      raw: node.raw,
    };
  }
  return {
    kind: "cel",
    key: node.key,
    expr: compiled.type.oneofKind === "match" ? compiled.type.match : "",
    raw: node.raw,
  };
};

const IconChip = (props: {
  icon: LucideIcon;
  tone: Tone;
  className?: string;
}) => (
  <span
    className={twMerge(
      "flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border",
      TONE_CHIP[props.tone],
      props.className,
    )}
  >
    <props.icon size={14} strokeWidth={2.2} />
  </span>
);

const NotBadge = () => (
  <span className="rounded-md border border-red-200 bg-red-50 px-1.5 py-px text-micro font-bold uppercase tracking-[0.06em] text-red-700">
    Not
  </span>
);

const CELLine = (props: { text?: string }) => {
  const [copied, setCopied] = React.useState(false);
  if (!props.text) return null;
  return (
    <div className="flex min-w-0 items-center gap-2 border-t border-slate-100 bg-slate-50/60 px-3 py-1.5">
      <Braces size={11} className="shrink-0 text-slate-400" aria-hidden />
      <code
        className="min-w-0 flex-1 truncate font-mono text-micro text-slate-500"
        title={props.text}
      >
        {props.text}
      </code>
      <Tooltip label={copied ? "Copied" : "Copy CEL"} withArrow openDelay={300}>
        <ActionIcon
          size="xs"
          variant="subtle"
          color="gray"
          aria-label="Copy CEL"
          onClick={async () => {
            await navigator.clipboard?.writeText(props.text ?? "");
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1200);
          }}
        >
          {copied ? (
            <Check size={11} className="text-emerald-600" />
          ) : (
            <Copy size={11} />
          )}
        </ActionIcon>
      </Tooltip>
    </div>
  );
};

const Issues = (props: { issues: string[] }) =>
  props.issues.length > 0 ? (
    <ul className="mt-3 flex flex-col gap-1" aria-label="Missing values">
      {props.issues.map((issue) => (
        <li
          key={issue}
          className="flex items-center gap-1.5 text-micro font-medium text-amber-700"
        >
          <AlertTriangle size={11} className="shrink-0" />
          {issue}
        </li>
      ))}
    </ul>
  ) : null;

const NodeMenu = (props: { node: ConditionNode }) => {
  const { node } = props;
  const builder = useBuilder();
  const leaf = node.kind === "block" || node.kind === "cel" ? node : undefined;
  const negated = leaf ? !!leaf.negate : false;
  const recognized =
    node.kind === "cel" &&
    !node.negate &&
    looksComplete(node.expr) &&
    leafFromExpr(node.expr).kind === "block";

  const alternative = () => {
    const parent = parentOf(builder.root, node.key);
    if (parent?.op === "any") {
      builder.openPicker(parent.key);
      return;
    }
    const wrapper = groupNode("any", [node]);
    builder.update(node.key, wrapper);
    builder.openPicker(wrapper.key);
  };

  return (
    <Menu position="bottom-end" withinPortal shadow="md" width={230}>
      <Menu.Target>
        <ActionIcon
          variant="subtle"
          color="gray"
          size="sm"
          aria-label="Condition actions"
        >
          <Ellipsis size={15} />
        </ActionIcon>
      </Menu.Target>
      <Menu.Dropdown>
        {leaf && (
          <Menu.Item
            leftSection={<Ban size={14} />}
            onClick={() =>
              builder.update(
                node.key,
                leaf.kind === "block"
                  ? touch(leaf, {
                      negate: leaf.negate ? undefined : "condition",
                    })
                  : touch(leaf, { negate: leaf.negate ? undefined : true }),
              )
            }
          >
            {negated ? "Remove the negation" : "Negate (NOT)"}
          </Menu.Item>
        )}
        {builder.scope === "request" && (
          <Menu.Item leftSection={<Split size={14} />} onClick={alternative}>
            Add an alternative (OR)
          </Menu.Item>
        )}
        {node.kind === "block" && (
          <Menu.Item
            leftSection={<Braces size={14} />}
            onClick={() => builder.update(node.key, celFromLeaf(node))}
          >
            Edit as CEL
          </Menu.Item>
        )}
        {node.kind === "cel" && recognized && builder.scope === "request" && (
          <Menu.Item
            leftSection={<Blocks size={14} />}
            onClick={() =>
              builder.update(
                node.key,
                leafFromExpr(node.expr, false, compileNode(node)),
              )
            }
          >
            Show as a visual block
          </Menu.Item>
        )}
        <Menu.Item
          leftSection={<CopyPlus size={14} />}
          onClick={() => {
            const parent = parentOf(builder.root, node.key);
            builder.setRoot(
              appendNodes(builder.root, [cloneNode(node)], parent?.key),
            );
          }}
        >
          Duplicate
        </Menu.Item>
        <Menu.Divider />
        <Menu.Item
          color="red"
          leftSection={<Trash2 size={14} />}
          onClick={() => builder.update(node.key, undefined)}
        >
          Remove
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
};

const Card = (props: {
  node: ConditionNode;
  icon: LucideIcon;
  tone: Tone;
  title: string;
  subtitle?: string;
  summary?: string;
  issues: string[];
  expandable: boolean;
  footer?: React.ReactNode;
  children?: React.ReactNode;
}) => {
  const negated =
    (props.node.kind === "block" || props.node.kind === "cel") &&
    !!props.node.negate;
  const [open, setOpen] = React.useState(
    () => !props.node.raw || props.issues.length > 0,
  );
  const bodyId = React.useId();
  const showBody = props.expandable && open;

  return (
    <article
      className={twMerge(
        "min-w-0 overflow-hidden rounded-xl border bg-white shadow-card transition-[border-color,box-shadow] duration-150 hover:border-slate-300",
        props.issues.length > 0 && !open
          ? "border-amber-300"
          : "border-slate-200",
      )}
    >
      <header className="flex items-start gap-2 px-3 py-2.5">
        <button
          type="button"
          disabled={!props.expandable}
          aria-expanded={props.expandable ? open : undefined}
          aria-controls={props.expandable ? bodyId : undefined}
          onClick={() => setOpen((value) => !value)}
          className="flex min-w-0 flex-1 cursor-pointer items-start gap-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-default"
        >
          <IconChip icon={props.icon} tone={props.tone} />
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
              {negated && <NotBadge />}
              <span className="text-xs font-semibold text-slate-800">
                {props.title}
              </span>
              {props.subtitle && (
                <span className="text-micro text-slate-400">
                  {props.subtitle}
                </span>
              )}
              {props.issues.length > 0 && !open && (
                <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-1.5 py-px text-micro font-semibold text-amber-700">
                  <AlertTriangle size={10} strokeWidth={2.5} />
                  Needs input
                </span>
              )}
            </span>
            {props.summary && (!showBody || !props.children) && (
              <span className="line-clamp-2 break-words text-xs text-slate-600">
                {props.summary}
              </span>
            )}
          </span>
        </button>
        <div className="flex shrink-0 items-center">
          <NodeMenu node={props.node} />
          {props.expandable && (
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              aria-label={open ? "Collapse" : "Expand"}
              onClick={() => setOpen((value) => !value)}
            >
              <ChevronDown
                size={14}
                className={twMerge(
                  "transition-transform duration-150",
                  !open && "-rotate-90",
                )}
              />
            </ActionIcon>
          )}
        </div>
      </header>
      {showBody && props.children && (
        <div
          id={bodyId}
          className="@container border-t border-slate-100 px-3 pb-3 pt-3"
        >
          {props.children}
          <Issues issues={props.issues} />
        </div>
      )}
      {showBody && props.footer}
    </article>
  );
};

const BlockCard = (props: { node: BlockNode }) => {
  const { node } = props;
  const builder = useBuilder();
  const block = getBlock(node.block);
  if (!block) return null;
  const issues = nodeIssues(node);
  const cel = issues.length === 0 ? nodeCEL(node) : undefined;
  return (
    <Card
      node={node}
      icon={blockIcon(block.id, block.category)}
      tone={CATEGORY_META[block.category].tone}
      title={block.title}
      subtitle={block.group}
      summary={
        issues.length > 0 && block.fields.length > 0
          ? block.description
          : block.summary(node.values)
      }
      issues={issues}
      expandable={block.fields.length > 0}
      footer={<CELLine text={cel} />}
    >
      <BlockFields
        fields={block.fields}
        values={node.values}
        onChange={(values) => builder.update(node.key, touch(node, { values }))}
      />
    </Card>
  );
};

const CELCard = (props: { node: CELNode }) => {
  const { node } = props;
  const builder = useBuilder();
  const issues = nodeIssues(node);
  const complete = looksComplete(node.expr);
  const recognized =
    builder.scope === "request" && complete && !node.negate
      ? recognize(node.expr)
      : undefined;
  const splittable = complete && canSplit(node.expr);
  return (
    <Card
      node={node}
      icon={Braces}
      tone="slate"
      title="CEL expression"
      summary={
        node.expr.trim()
          ? node.expr.trim().replace(/\s+/g, " ")
          : "Empty expression"
      }
      issues={issues}
      expandable
    >
      <CELEditor
        label="CEL expression"
        context={builder.context}
        invalid={node.expr.trim() !== "" && !complete}
        exp={node.expr}
        onChange={(expr) => builder.update(node.key, touch(node, { expr }))}
      />
      {(recognized || splittable) && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {recognized && (
            <Button
              size="compact-xs"
              variant="light"
              color="gray"
              leftSection={<Blocks size={12} />}
              onClick={() =>
                builder.update(
                  node.key,
                  leafFromExpr(node.expr, false, compileNode(node)),
                )
              }
            >
              Show as “{recognized.block.title}”
            </Button>
          )}
          {splittable && (
            <Button
              size="compact-xs"
              variant="light"
              color="gray"
              leftSection={<ListTree size={12} />}
              onClick={() => {
                const split = splitCEL(node.expr);
                builder.update(
                  node.key,
                  node.negate ? groupNode("none", [split]) : split,
                );
              }}
            >
              Split into separate conditions
            </Button>
          )}
        </div>
      )}
    </Card>
  );
};

const NodeView = (props: { node: ConditionNode; depth: number }) => {
  const { node } = props;
  const builder = useBuilder();
  switch (node.kind) {
    case "group":
      return <GroupView node={node} depth={props.depth} />;
    case "block":
      return <BlockCard node={node} />;
    case "cel":
      return <CELCard node={node} />;
    case "opa":
      return (
        <Card
          node={node}
          icon={FileCode}
          tone="slate"
          title="OPA policy"
          summary="Inline Rego policy"
          issues={nodeIssues(node)}
          expandable
        >
          <OPAEditor
            label="OPA/Rego policy"
            invalid={!node.rego.trim()}
            exp={node.rego}
            onChange={(rego) => builder.update(node.key, touch(node, { rego }))}
          />
        </Card>
      );
    case "every":
      return (
        <Card
          node={node}
          icon={Check}
          tone="slate"
          title={node.raw ? "Never" : "Every request"}
          summary={node.raw ? "This condition never matches" : "Always matches"}
          issues={[]}
          expandable={false}
        />
      );
  }
};

const Connector = (props: { op: GroupOp }) => (
  <div className="relative h-7" aria-hidden>
    <span className="absolute bottom-0 left-[26px] top-0 w-px bg-slate-300" />
    <span
      className={twMerge(
        "absolute left-[26px] top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border px-1.5 text-[10px] font-bold uppercase leading-4 tracking-[0.08em]",
        TONE_CHIP[OP_META[props.op].tone],
      )}
    >
      {OP_META[props.op].word}
    </span>
  </div>
);

const AddBar = (props: { groupKey?: string; compact?: boolean }) => {
  const builder = useBuilder();
  return (
    <div className="flex flex-wrap items-center gap-1.5 pt-2">
      <Button
        type="button"
        variant="default"
        size="compact-sm"
        leftSection={<Plus size={13} strokeWidth={2.5} />}
        onClick={() => builder.openPicker(props.groupKey)}
      >
        Add condition
      </Button>
      <Button
        type="button"
        variant="subtle"
        color="gray"
        size="compact-sm"
        leftSection={<Layers size={13} />}
        onClick={() =>
          builder.setRoot(
            appendNodes(builder.root, [groupNode("any", [])], props.groupKey),
          )
        }
      >
        Add group
      </Button>
    </div>
  );
};

const GroupView = (props: { node: GroupNode; depth: number }) => {
  const { node } = props;
  const builder = useBuilder();
  const root = props.depth === 0;
  const meta = OP_META[node.op];

  return (
    <section
      aria-label={`${meta.label} of these conditions`}
      className={twMerge(
        "flex min-w-0 flex-col",
        !root &&
          "rounded-xl border border-dashed border-slate-300 bg-slate-50/50 p-2.5 sm:p-3",
      )}
    >
      <header className="mb-2 flex flex-wrap items-center gap-x-2 gap-y-1.5">
        {!root && <Layers size={13} className="text-slate-400" />}
        <span className="text-xs font-semibold text-slate-700">Match</span>
        <SegmentedControl
          size="xs"
          aria-label="How the conditions combine"
          value={node.op}
          data={(["all", "any", "none"] as GroupOp[]).map((op) => ({
            value: op,
            label: OP_META[op].label,
          }))}
          onChange={(op) =>
            builder.update(node.key, touch(node, { op: op as GroupOp }))
          }
        />
        <span className="text-xs text-slate-500">
          of these · <span className="text-slate-600">{meta.hint}</span>
        </span>
        <span className="ml-auto flex items-center">
          {node.children.length === 1 && (
            <Tooltip label="Ungroup" withArrow>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                aria-label="Ungroup"
                onClick={() => builder.update(node.key, node.children[0])}
              >
                <Ungroup size={14} />
              </ActionIcon>
            </Tooltip>
          )}
          {!root && (
            <Tooltip label="Remove group" withArrow>
              <ActionIcon
                variant="subtle"
                color="red"
                size="sm"
                aria-label="Remove group"
                onClick={() => builder.update(node.key, undefined)}
              >
                <Trash2 size={13} />
              </ActionIcon>
            </Tooltip>
          )}
        </span>
      </header>
      {node.children.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-300 px-3 py-3 text-xs text-slate-500">
          This group is empty. Add the conditions to combine.
        </p>
      ) : (
        <ol className="flex min-w-0 flex-col">
          {node.children.map((child, index) => (
            <li key={child.key} className="min-w-0">
              {index > 0 && <Connector op={node.op} />}
              <NodeView node={child} depth={props.depth + 1} />
            </li>
          ))}
        </ol>
      )}
      <AddBar groupKey={node.key} />
    </section>
  );
};

const FEATURED = [
  "group-members",
  "group-namespace",
  "specific-users",
  "business-hours",
  "http-read-only",
  "api-read-only",
];

const EmptyState = (props: {
  title: string;
  text: string;
  featured: string[];
  onAskAI?: () => void;
  onRecipe: (recipe: Recipe) => void;
}) => {
  const builder = useBuilder();
  const recipes =
    builder.scope === "request" ? featuredRecipes(props.featured) : [];
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-dashed border-slate-300 bg-slate-50/50 p-3 sm:p-4">
      <div className="flex flex-col">
        <span className="text-body font-semibold text-slate-800">
          {props.title}
        </span>
        <span className="text-xs leading-5 text-slate-500">{props.text}</span>
      </div>
      {recipes.length > 0 && (
        <div className="grid grid-cols-1 gap-1.5 @lg:grid-cols-2 @4xl:grid-cols-3">
          {recipes.map((recipe) => (
            <button
              key={recipe.id}
              type="button"
              onClick={() => props.onRecipe(recipe)}
              className="group flex min-w-0 cursor-pointer items-start gap-2.5 rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-left outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-card focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              <IconChip
                icon={RECIPE_ICONS[recipe.icon]}
                tone={CATEGORY_META[recipe.category].tone}
                className="h-6 w-6 rounded-md"
              />
              <span className="flex min-w-0 flex-col">
                <span className="text-xs font-semibold text-slate-800">
                  {recipe.title}
                </span>
                <span className="text-micro leading-4 text-slate-500">
                  {recipe.description}
                </span>
              </span>
            </button>
          ))}
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="default"
          size="compact-sm"
          leftSection={<Plus size={13} strokeWidth={2.5} />}
          onClick={() => builder.openPicker()}
        >
          {builder.scope === "request"
            ? "Browse all conditions"
            : "Add condition"}
        </Button>
        <Button
          type="button"
          variant="subtle"
          color="gray"
          size="compact-sm"
          leftSection={<Braces size={13} />}
          onClick={() => builder.setRoot(celNode(""))}
        >
          Write CEL
        </Button>
        {builder.scope !== "request" && (
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-sm"
            leftSection={<Check size={13} />}
            onClick={() => builder.setRoot(everyNode())}
          >
            Match every request
          </Button>
        )}
        {props.onAskAI && (
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-sm"
            leftSection={<Sparkles size={13} />}
            onClick={props.onAskAI}
          >
            Describe it to the AI
          </Button>
        )}
      </div>
    </div>
  );
};

export type ConditionBuilderProps = {
  value?: ConditionNode;
  onChange: (node?: ConditionNode) => void;
  scope?: PickerScope;
  context?: CELContext;
  header?: React.ReactNode;
  emptyTitle?: string;
  emptyText?: string;
  featured?: string[];
  onRecipe?: (recipe: Recipe) => void;
  onAskAI?: () => void;
};

const ConditionBuilder = (props: ConditionBuilderProps) => {
  const scope = props.scope ?? "request";
  const context = props.context ?? "core";
  const [mode, setMode] = React.useState<"visual" | "cel">("visual");
  const [celDraft, setCelDraft] = React.useState("");
  const [picker, setPicker] = React.useState<
    { groupKey?: string } | undefined
  >();
  const [pickerSeq, setPickerSeq] = React.useState(0);

  const setRoot = (next?: ConditionNode) => props.onChange(next);

  const ctx: Ctx = {
    root: props.value,
    scope,
    context,
    setRoot,
    update: (key, next) => {
      const root = props.value;
      if (root?.key === key) {
        setRoot(next);
        return;
      }
      let updated = replaceNode(root, key, next);
      if (
        updated?.kind === "group" &&
        updated.children.length === 0 &&
        root?.kind === "group" &&
        root.children.length > 0 &&
        !next
      ) {
        updated = undefined;
      }
      setRoot(updated);
    },
    openPicker: (groupKey) => {
      setPicker({ groupKey });
      setPickerSeq((value) => value + 1);
    },
  };

  const opa = hasOPA(props.value);
  const switchMode = (next: string) => {
    if (next === mode) return;
    if (next === "cel") {
      setCelDraft(props.value ? (nodeCEL(props.value, true) ?? "") : "");
    } else {
      const value = props.value;
      if (value?.kind === "cel" && !value.negate && looksComplete(value.expr)) {
        setRoot(leafFromExpr(value.expr, false, compileNode(value)));
      }
    }
    setMode(next as "visual" | "cel");
  };

  const pick = (nodes: ConditionNode[], recipe?: Recipe) => {
    const target = picker?.groupKey;
    setPicker(undefined);
    const exists = target ? findNode(props.value, target) : undefined;
    setRoot(appendNodes(props.value, nodes, exists ? target : undefined));
    if (recipe && !props.value) props.onRecipe?.(recipe);
  };

  const emptyTitle = props.emptyTitle ?? "No condition yet";
  const emptyText =
    props.emptyText ??
    "Pick what the requests must look like, combine several conditions, or write CEL.";

  return (
    <BuilderContext.Provider value={ctx}>
      <div
        className="@container flex min-w-0 flex-col gap-3"
        data-condition-builder
      >
        <div className="flex flex-wrap items-center gap-2">
          <div className="min-w-0 flex-1">{props.header}</div>
          {props.onAskAI && props.value && (
            <Button
              type="button"
              size="compact-xs"
              variant="default"
              leftSection={<Sparkles size={12} />}
              onClick={props.onAskAI}
            >
              Ask AI
            </Button>
          )}
          <SegmentedControl
            size="xs"
            aria-label="Condition editor mode"
            value={mode}
            onChange={switchMode}
            data={[
              {
                value: "visual",
                label: (
                  <span className="flex items-center gap-1">
                    <Blocks size={12} /> Visual
                  </span>
                ),
              },
              {
                value: "cel",
                disabled: opa,
                label: (
                  <Tooltip
                    label="OPA policies cannot be written as CEL"
                    disabled={!opa}
                    withArrow
                  >
                    <span className="flex items-center gap-1">
                      <Braces size={12} /> CEL
                    </span>
                  </Tooltip>
                ),
              },
            ]}
          />
        </div>

        {mode === "cel" ? (
          <div className="flex flex-col gap-2">
            {hasCoreNegation(props.value) && (
              <p className="flex items-start gap-2 rounded-lg border border-blue-100 bg-blue-50/50 px-3 py-2 text-micro leading-4 text-slate-700">
                <Info size={12} className="mt-0.5 shrink-0 text-blue-700" />
                Editing here saves the condition as one CEL expression. Its NOT
                conditions and None groups become “!”, which does not match
                requests that lack the negated fields (e.g. HTTP fields on an
                SSH request), whereas a NOT condition does.
              </p>
            )}
            <CELEditor
              label="Condition as CEL"
              context={context}
              invalid={celDraft.trim() !== "" && !looksComplete(celDraft)}
              exp={celDraft}
              onChange={(text) => {
                setCelDraft(text);
                setRoot(celNode(text));
              }}
            />
            <p className="text-micro leading-4 text-slate-500">
              Saved as a single CEL expression. Switch back to Visual to see it
              as blocks or split it into separate conditions.
            </p>
          </div>
        ) : !props.value ? (
          <EmptyState
            title={emptyTitle}
            text={emptyText}
            featured={props.featured ?? FEATURED}
            onAskAI={props.onAskAI}
            onRecipe={(recipe) => pick([recipe.build()], recipe)}
          />
        ) : props.value.kind === "group" ? (
          <GroupView node={props.value} depth={0} />
        ) : (
          <div className="flex min-w-0 flex-col">
            <NodeView node={props.value} depth={0} />
            <AddBar />
          </div>
        )}

        <ConditionPicker
          key={pickerSeq}
          opened={!!picker}
          scope={scope}
          onClose={() => setPicker(undefined)}
          onPick={pick}
        />
      </div>
    </BuilderContext.Provider>
  );
};

export default ConditionBuilder;
