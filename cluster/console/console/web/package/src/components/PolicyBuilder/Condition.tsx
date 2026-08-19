import * as CoreP from "@/apis/corev1/corev1";
import {
  Condition,
  Condition_Expression as Expression,
} from "@/apis/enterprisev1/enterprisev1";
import {
  ActionIcon,
  Button,
  Drawer,
  Select,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  Braces,
  Check,
  Edit2,
  GitBranch,
  Plus,
  Search,
  Sparkles,
  Trash2,
  X,
} from "lucide-react";
import * as React from "react";
import { useMemo, useState } from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import itemList, { ItemDef } from "./ItemList";
import { ExprChip } from "./PrintCondition";

const getNewExpression = () =>
  Expression.create({
    type: {
      oneofKind: "serviceMode",
      serviceMode: { mode: CoreP.Service_Spec_Mode.HTTP },
    },
  });

const getNewCondition = () =>
  Condition.create({
    type: { oneofKind: "expression", expression: getNewExpression() },
  });

const TYPE_OPTIONS = [
  { value: "expression", label: "Expression" },
  { value: "matchAny", label: "Match everything" },
  { value: "all", label: "All conditions · AND" },
  { value: "any", label: "Any condition · OR" },
  { value: "none", label: "No conditions · NOR" },
  { value: "not", label: "Negated expression · NOT" },
];

const TYPE_META = {
  all: {
    label: "All conditions",
    description: "Every child condition must match",
    border: "border-l-blue-500",
    rail: "border-blue-200",
    badge: "border-blue-200 bg-blue-50 text-blue-700",
  },
  any: {
    label: "Any condition",
    description: "At least one child condition must match",
    border: "border-l-emerald-500",
    rail: "border-emerald-200",
    badge: "border-emerald-200 bg-emerald-50 text-emerald-700",
  },
  none: {
    label: "No conditions",
    description: "None of the child conditions may match",
    border: "border-l-amber-500",
    rail: "border-amber-200",
    badge: "border-amber-200 bg-amber-50 text-amber-700",
  },
};

let conditionID = 0;
const nextConditionID = () => `policy-condition-${++conditionID}`;

const Cond = (props: {
  item?: Condition;
  onChange: (condition?: Condition) => void;
  onDelete?: () => void;
  depth?: number;
}) => {
  const depth = props.depth ?? 0;
  const [localItem, setLocalItem] = React.useState<Condition>(
    props.item ??
      Condition.create({ type: { oneofKind: "matchAny", matchAny: true } }),
  );

  React.useEffect(() => {
    setLocalItem(
      props.item
        ? Condition.clone(props.item)
        : Condition.create({ type: { oneofKind: "matchAny", matchAny: true } }),
    );
  }, [props.item]);

  const currentItem = props.item ?? localItem;

  const updateItem = (next: Condition) => {
    setLocalItem(next);
    props.onChange(next);
  };

  const handleTypeChange = (newType: string) => {
    const next = Condition.clone(currentItem);
    const existingChildren =
      currentItem.type.oneofKind === "all"
        ? currentItem.type.all.of
        : currentItem.type.oneofKind === "any"
          ? currentItem.type.any.of
          : currentItem.type.oneofKind === "none"
            ? currentItem.type.none.of
            : [];
    const children =
      existingChildren.length > 0 ? existingChildren : [getNewCondition()];
    const expression =
      (currentItem.type.oneofKind === "expression"
        ? currentItem.type.expression
        : currentItem.type.oneofKind === "not"
          ? currentItem.type.not.expression
          : getNewExpression()) ?? getNewExpression();

    switch (newType) {
      case "all":
        next.type = { oneofKind: "all", all: { of: children } };
        break;
      case "any":
        next.type = { oneofKind: "any", any: { of: children } };
        break;
      case "none":
        next.type = { oneofKind: "none", none: { of: children } };
        break;
      case "not":
        next.type = { oneofKind: "not", not: { expression } };
        break;
      case "expression":
        next.type = { oneofKind: "expression", expression };
        break;
      case "matchAny":
        next.type = { oneofKind: "matchAny", matchAny: true };
        break;
    }

    updateItem(next);
  };

  const kind = currentItem.type.oneofKind;
  const groupMeta =
    kind === "all" || kind === "any" || kind === "none"
      ? TYPE_META[kind]
      : undefined;
  const accent = groupMeta
    ? groupMeta.border
    : kind === "not"
      ? "border-l-red-500"
      : kind === "matchAny"
        ? "border-l-slate-400"
        : "border-l-indigo-500";

  return (
    <section
      className={twMerge(
        "w-full overflow-hidden rounded-xl border border-l-[3px] border-slate-200 bg-white shadow-[0_1px_3px_rgba(15,23,42,0.04)] transition-[border-color,box-shadow] duration-500 hover:border-slate-300 hover:shadow-[0_4px_14px_rgba(15,23,42,0.06)]",
        accent,
      )}
    >
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-slate-50/70 px-3 py-2.5 sm:px-4">
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500">
            {groupMeta ? (
              <GitBranch size={13} strokeWidth={2.25} />
            ) : kind === "matchAny" ? (
              <Sparkles size={13} strokeWidth={2.25} />
            ) : (
              <Braces size={13} strokeWidth={2.25} />
            )}
          </span>
          <Select
            value={kind ?? "expression"}
            data={TYPE_OPTIONS}
            allowDeselect={false}
            onChange={(value) => value && handleTypeChange(value)}
            className="min-w-0 max-w-[250px] flex-1"
            styles={{ input: { minHeight: "32px", height: "32px" } }}
          />
          {depth > 0 && (
            <span className="hidden rounded-md bg-slate-100 px-1.5 py-0.5 text-[0.58rem] font-bold uppercase tracking-[0.06em] text-slate-400 sm:inline">
              Level {depth + 1}
            </span>
          )}
        </div>

        {props.onDelete && (
          <Tooltip label="Remove condition" position="left" withArrow>
            <ActionIcon
              type="button"
              variant="subtle"
              color="red"
              size="sm"
              aria-label="Remove condition"
              onClick={props.onDelete}
            >
              <Trash2 size={13} strokeWidth={2.25} />
            </ActionIcon>
          </Tooltip>
        )}
      </header>

      <div className="p-3 sm:p-4">
        {match(currentItem.type)
          .with({ oneofKind: "all" }, (type) => (
            <LogicalGroup
              kind="all"
              depth={depth}
              items={type.all.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "all") next.type.all.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "any" }, (type) => (
            <LogicalGroup
              kind="any"
              depth={depth}
              items={type.any.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "any") next.type.any.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "none" }, (type) => (
            <LogicalGroup
              kind="none"
              depth={depth}
              items={type.none.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "none") next.type.none.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "not" }, (type) => (
            <div className="rounded-xl border border-red-100 bg-red-50/40 p-3">
              <div className="mb-2.5 flex items-center gap-2">
                <span className="rounded-md border border-red-200 bg-white px-2 py-0.5 text-[0.6rem] font-bold uppercase tracking-[0.07em] text-red-600">
                  NOT
                </span>
                <span className="text-[0.68rem] font-semibold text-red-700/70">
                  The expression result is inverted
                </span>
              </div>
              <ExpressionC
                item={type.not.expression}
                onUpdate={(expression) => {
                  const next = Condition.clone(currentItem);
                  if (next.type.oneofKind === "not") {
                    next.type.not.expression = expression ?? getNewExpression();
                  }
                  updateItem(next);
                }}
              />
            </div>
          ))
          .with({ oneofKind: "expression" }, (type) => (
            <ExpressionC
              item={type.expression}
              onUpdate={(expression) => {
                const next = Condition.clone(currentItem);
                next.type = expression
                  ? { oneofKind: "expression", expression }
                  : { oneofKind: undefined };
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "matchAny" }, () => (
            <div className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50/70 px-3.5 py-3">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Check size={13} strokeWidth={2.5} />
              </span>
              <div>
                <p className="text-[0.75rem] font-bold text-slate-700">
                  No restrictions
                </p>
                <p className="mt-0.5 text-[0.68rem] font-semibold text-slate-400">
                  This condition matches every request automatically.
                </p>
              </div>
            </div>
          ))
          .otherwise(() => null)}
      </div>
    </section>
  );
};

const LogicalGroup = (props: {
  kind: "all" | "any" | "none";
  items: Condition[];
  depth: number;
  onUpdate: (items: Condition[]) => void;
}) => {
  const meta = TYPE_META[props.kind];
  const ids = React.useRef(props.items.map(nextConditionID));
  while (ids.current.length < props.items.length) {
    ids.current.push(nextConditionID());
  }
  if (ids.current.length > props.items.length) {
    ids.current = ids.current.slice(0, props.items.length);
  }

  const addCondition = () => {
    ids.current.push(nextConditionID());
    props.onUpdate([...props.items, getNewCondition()]);
  };

  const removeCondition = (index: number) => {
    ids.current.splice(index, 1);
    props.onUpdate(props.items.filter((_, itemIndex) => itemIndex !== index));
  };

  return (
    <div className={twMerge("border-l-2 pl-3 sm:pl-4", meta.rail)}>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span
          className={twMerge(
            "rounded-md border px-2 py-0.5 text-[0.6rem] font-bold uppercase tracking-[0.07em]",
            meta.badge,
          )}
        >
          {props.kind === "all" ? "AND" : props.kind === "any" ? "OR" : "NOR"}
        </span>
        <div>
          <span className="text-[0.72rem] font-bold text-slate-700">
            {meta.label}
          </span>
          <span className="ml-2 text-[0.65rem] font-semibold text-slate-400">
            {meta.description}
          </span>
        </div>
      </div>

      <div className="space-y-3">
        {props.items.map((item, index) => (
          <Cond
            key={ids.current[index]}
            item={item}
            depth={props.depth + 1}
            onChange={(updated) => {
              if (!updated) return;
              props.onUpdate(
                props.items.map((existing, itemIndex) =>
                  itemIndex === index ? updated : existing,
                ),
              );
            }}
            onDelete={() => removeCondition(index)}
          />
        ))}
      </div>

      <Button
        type="button"
        variant="default"
        size="compact-sm"
        leftSection={<Plus size={12} strokeWidth={2.5} />}
        onClick={addCondition}
        className="mt-3"
      >
        Add condition
      </Button>
    </div>
  );
};

const ExpressionC = (props: {
  item?: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [draft, setDraft] = useState<Expression>();
  const [changed, setChanged] = useState(false);
  if (!props.item) return null;

  const openEditor = () => {
    setDraft(Expression.clone(props.item!));
    setChanged(false);
    open();
  };

  return (
    <>
      <div className="flex min-w-0 flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/60 p-3">
        <div className="min-w-0 flex-1">
          <ExprChip item={props.item} />
        </div>
        <Button
          type="button"
          variant="default"
          size="compact-sm"
          leftSection={<Edit2 size={12} strokeWidth={2.5} />}
          onClick={openEditor}
        >
          Configure
        </Button>
      </div>

      <Drawer
        opened={opened}
        onClose={close}
        position="right"
        size="min(960px, 100vw)"
        padding="md"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <Braces size={15} className="text-slate-400" strokeWidth={2.25} />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              Expression
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {itemList.find((item) => item.type === draft?.type.oneofKind)
                ?.title ?? "Configure rule"}
            </span>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: 0,
            display: "flex",
            flexDirection: "column",
            backgroundColor: "#f8fafc",
          },
          content: {
            display: "flex",
            flexDirection: "column",
            borderLeft: "1px solid #e2e8f0",
          },
        }}
      >
        {draft && (
          <div className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-hidden p-4">
              <ExpressionEditC
                item={draft}
                onUpdate={(updated) => {
                  if (!updated) return;
                  setDraft(Expression.clone(updated));
                  setChanged(true);
                }}
              />
            </div>
            <div className="flex shrink-0 items-center justify-between gap-3 border-t border-slate-200 bg-white px-4 py-3">
              <span className="text-[0.68rem] font-semibold text-slate-400">
                {changed ? "Unsaved expression changes" : "No changes yet"}
              </span>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="default"
                  leftSection={<X size={12} />}
                  onClick={close}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  color="dark"
                  leftSection={<Check size={12} />}
                  disabled={!changed}
                  onClick={() => {
                    props.onUpdate(Expression.clone(draft));
                    close();
                  }}
                >
                  Apply expression
                </Button>
              </div>
            </div>
          </div>
        )}
      </Drawer>
    </>
  );
};

const ALL_TAGS = Array.from(new Set(itemList.flatMap((item) => item.tags))).sort();

const ExpressionEditC = (props: {
  item: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  const [query, setQuery] = useState("");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const selectedType = props.item.type.oneofKind;
  const selectedDef = itemList.find((item) => item.type === selectedType);

  const filtered = useMemo(() => {
    const normalizedQuery = query.toLowerCase().trim();
    return itemList.filter((item) => {
      const matchesTag = !activeTag || item.tags.includes(activeTag);
      const matchesQuery =
        !normalizedQuery ||
        item.title.toLowerCase().includes(normalizedQuery) ||
        item.tags.some((tag) => tag.includes(normalizedQuery));
      return matchesTag && matchesQuery;
    });
  }, [query, activeTag]);

  const selectType = (definition: ItemDef) => {
    if (selectedType === definition.type) return;
    props.onUpdate(definition.makeDefault());
  };

  return (
    <div className="grid h-full min-h-0 gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
      <aside className="flex min-h-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="space-y-3 border-b border-slate-100 bg-slate-50/60 p-3">
          <TextInput
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search expression types"
            leftSection={<Search size={13} strokeWidth={2.25} />}
          />
          <Select
            value={activeTag}
            onChange={setActiveTag}
            data={ALL_TAGS.map((tag) => ({ value: tag, label: tag }))}
            searchable
            clearable
            placeholder="Filter by category"
          />
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto p-2">
          {filtered.length === 0 ? (
            <div className="px-3 py-10 text-center">
              <p className="text-[0.75rem] font-bold text-slate-600">
                No expressions found
              </p>
              <p className="mt-1 text-[0.68rem] font-semibold text-slate-400">
                Try another search or category.
              </p>
            </div>
          ) : (
            <div className="space-y-1">
              {filtered.map((definition) => {
                const selected = selectedType === definition.type;
                return (
                  <button
                    key={definition.type}
                    type="button"
                    onClick={() => selectType(definition)}
                    className={twMerge(
                      "flex w-full items-start gap-2.5 rounded-lg px-3 py-2.5 text-left transition-colors duration-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-inset",
                      selected
                        ? "bg-slate-900 text-white"
                        : "text-slate-700 hover:bg-slate-100",
                    )}
                  >
                    <span
                      className={twMerge(
                        "mt-0.5 h-2 w-2 shrink-0 rounded-full",
                        selected ? "bg-blue-400" : "bg-slate-300",
                      )}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block text-[0.74rem] font-bold">
                        {definition.title}
                      </span>
                      <span
                        className={twMerge(
                          "mt-1 block truncate text-[0.6rem] font-semibold",
                          selected ? "text-slate-300" : "text-slate-400",
                        )}
                      >
                        {definition.tags.join(" · ")}
                      </span>
                    </span>
                    {selected && <Check size={13} className="shrink-0" />}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </aside>

      <section className="min-h-0 overflow-y-auto rounded-xl border border-slate-200 bg-white shadow-sm">
        {selectedDef ? (
          <>
            <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50/60 px-4 py-3.5">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Braces size={14} strokeWidth={2.25} />
              </span>
              <div>
                <p className="text-[0.78rem] font-bold text-slate-800">
                  {selectedDef.title}
                </p>
                <p className="mt-0.5 text-[0.63rem] font-semibold text-slate-400">
                  Configure the values used during policy evaluation
                </p>
              </div>
            </div>
            <div className="p-4 sm:p-5">
              <selectedDef.components.Edit
                item={props.item}
                onUpdate={props.onUpdate}
              />
            </div>
          </>
        ) : (
          <div className="flex h-full min-h-64 items-center justify-center px-6 text-center">
            <div>
              <Braces size={22} className="mx-auto text-slate-300" />
              <p className="mt-3 text-[0.78rem] font-bold text-slate-600">
                Choose an expression
              </p>
              <p className="mt-1 text-[0.68rem] font-semibold text-slate-400">
                Select an expression type from the catalog to configure it.
              </p>
            </div>
          </div>
        )}
      </section>
    </div>
  );
};

export default Cond;
