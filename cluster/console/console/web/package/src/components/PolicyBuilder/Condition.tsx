import * as CoreP from "@/apis/corev1/corev1";
import {
  Condition,
  Condition_Expression as Expression,
} from "@/apis/enterprisev1/enterprisev1";
import { Drawer, ScrollArea, Select, Tooltip } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { Edit2, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import itemList from "./ItemList";
import { ExprChip } from "./PrintCondition";

const getNewExpression = () =>
  Expression.create({
    type: {
      oneofKind: "serviceMode",
      serviceMode: { mode: CoreP.Service_Spec_Mode.HTTP },
    },
  });

type itemT = (typeof itemList)[0];

const TYPE_OPTIONS = [
  { value: "expression", label: "Expression" },
  { value: "matchAny", label: "Match everything" },
  { value: "all", label: "All of (AND)" },
  { value: "any", label: "Any of (OR)" },
  { value: "none", label: "None of (NOR)" },
  { value: "not", label: "Not" },
];

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
    if (props.item) setLocalItem(props.item);
  }, [props.item]);

  const currentItem = props.item ?? localItem;

  const updateItem = (newItem: Condition) => {
    setLocalItem(newItem);
    props.onChange(newItem);
  };

  const handleTypeChange = (newType: string) => {
    const next = Condition.clone(currentItem);

    let existingChildren: Condition[] = [];
    if (currentItem.type.oneofKind === "all")
      existingChildren = currentItem.type.all.of;
    else if (currentItem.type.oneofKind === "any")
      existingChildren = currentItem.type.any.of;
    else if (currentItem.type.oneofKind === "none")
      existingChildren = currentItem.type.none.of;

    const fallbackChild = Condition.create({
      type: { oneofKind: "expression", expression: getNewExpression() },
    });
    const childrenToKeep =
      existingChildren.length > 0 ? existingChildren : [fallbackChild];

    switch (newType) {
      case "all":
        next.type = { oneofKind: "all", all: { of: childrenToKeep } };
        break;
      case "any":
        next.type = { oneofKind: "any", any: { of: childrenToKeep } };
        break;
      case "none":
        next.type = { oneofKind: "none", none: { of: childrenToKeep } };
        break;
      case "not":
        next.type = {
          oneofKind: "not",
          not: { condition: childrenToKeep[0] },
        };
        break;
      case "expression":
        next.type = {
          oneofKind: "expression",
          expression:
            currentItem.type.oneofKind === "expression"
              ? currentItem.type.expression
              : getNewExpression(),
        };
        break;
      case "matchAny":
        next.type = { oneofKind: "matchAny", matchAny: true };
        break;
    }

    updateItem(next);
  };

  const accentColor = match(currentItem.type.oneofKind)
    .with("all", () => "#3b82f6")
    .with("any", () => "#22c55e")
    .with("none", () => "#f97316")
    .with("not", () => "#ef4444")
    .otherwise(() => "#e2e8f0");

  return (
    <div
      className="w-full bg-white rounded-lg border border-slate-200 mb-3 overflow-hidden transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-[0_2px_8px_rgba(15,23,42,0.06)]"
      style={{ borderLeftColor: accentColor, borderLeftWidth: "3px" }}
    >
      <div className="flex items-center justify-between px-3 py-2 bg-slate-50/80 border-b border-slate-100">
        <Select
          size="xs"
          w={180}
          value={currentItem.type.oneofKind ?? "expression"}
          data={TYPE_OPTIONS}
          onChange={(v) => v && handleTypeChange(v)}
          styles={{
            input: {
              fontWeight: 700,
              fontSize: "0.72rem",
              backgroundColor: "#ffffff",
              border: "1px solid #e2e8f0",
              height: "26px",
              minHeight: "26px",
            },
          }}
        />

        {props.onDelete && (
          <Tooltip label="Remove condition" position="left" withArrow>
            <button
              onClick={props.onDelete}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-red-500 hover:bg-red-50 transition-colors duration-150 cursor-pointer ml-2"
            >
              <Trash2 size={13} strokeWidth={2} />
            </button>
          </Tooltip>
        )}
      </div>

      <div className="p-3">
        {match(currentItem.type)
          .with({ oneofKind: "all" }, (t) => (
            <LogicalGroup
              kind="all"
              depth={depth}
              items={t.all.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "all") next.type.all.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "any" }, (t) => (
            <LogicalGroup
              kind="any"
              depth={depth}
              items={t.any.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "any") next.type.any.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "none" }, (t) => (
            <LogicalGroup
              kind="none"
              depth={depth}
              items={t.none.of}
              onUpdate={(items) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "none") next.type.none.of = items;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "not" }, (t) => (
            <div className="pl-3 border-l-2 border-red-200">
              <Cond
                item={t.not.condition}
                depth={depth + 1}
                onChange={(v) => {
                  if (!v) return;
                  const next = Condition.clone(currentItem);
                  if (next.type.oneofKind === "not")
                    next.type.not.condition = v;
                  updateItem(next);
                }}
              />
            </div>
          ))
          .with({ oneofKind: "expression" }, (t) => (
            <ExpressionC
              item={t.expression}
              onUpdate={(v) => {
                const next = Condition.clone(currentItem);
                next.type = v
                  ? { oneofKind: "expression", expression: v }
                  : { oneofKind: undefined };
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "matchAny" }, () => (
            <p className="text-[0.75rem] font-semibold text-slate-400 italic py-1">
              Matches all resources automatically.
            </p>
          ))
          .otherwise(() => null)}
      </div>
    </div>
  );
};

const groupAccent = {
  all: { border: "border-blue-200", bg: "bg-blue-50/40" },
  any: { border: "border-green-200", bg: "bg-green-50/40" },
  none: { border: "border-orange-200", bg: "bg-orange-50/40" },
};

const LogicalGroup = (props: {
  kind: "all" | "any" | "none";
  items: Condition[];
  depth: number;
  onUpdate: (items: Condition[]) => void;
}) => {
  const accent = groupAccent[props.kind];

  const handleAdd = () => {
    props.onUpdate([
      ...props.items,
      Condition.create({
        type: { oneofKind: "expression", expression: getNewExpression() },
      }),
    ]);
  };

  return (
    <div className={twMerge("pl-3 border-l-2", accent.border)}>
      <div className="flex flex-col gap-0">
        {props.items.map((item, idx) => (
          <Cond
            key={idx}
            item={item}
            depth={props.depth + 1}
            onChange={(v) => {
              const next = [...props.items];
              if (v) next[idx] = v;
              props.onUpdate(next);
            }}
            onDelete={
              props.items.length > 1
                ? () => {
                    const next = [...props.items];
                    next.splice(idx, 1);
                    props.onUpdate(next);
                  }
                : undefined
            }
          />
        ))}
      </div>

      <button
        onClick={handleAdd}
        className="flex items-center gap-1.5 mt-1 mb-1 px-2.5 py-1 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
      >
        <Plus size={11} strokeWidth={2.5} />
        Add condition
      </button>
    </div>
  );
};

const ExpressionC = (props: {
  item?: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  const [opened, { open, close }] = useDisclosure(false);
  if (!props.item) return null;

  return (
    <>
      <div className="flex items-center justify-between gap-2 p-2 rounded-lg border border-slate-200 bg-slate-50/50">
        <ExprChip item={props.item} />
        <button
          onClick={open}
          className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.7rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] shrink-0"
        >
          <Edit2 size={11} strokeWidth={2.5} />
          Edit
        </button>
      </div>

      <Drawer
        opened={opened}
        onClose={close}
        size="lg"
        title={
          <span className="text-[0.78rem] font-bold uppercase tracking-[0.05em] text-slate-700">
            Edit expression
          </span>
        }
        styles={{
          header: {
            borderBottom: "1px solid #e2e8f0",
            paddingBottom: "12px",
          },
          body: { padding: "16px" },
        }}
      >
        <ScrollArea type="auto" offsetScrollbars>
          <ExpressionEditC item={props.item} onUpdate={props.onUpdate} />
        </ScrollArea>
      </Drawer>
    </>
  );
};

const ExpressionEditC = (props: {
  item?: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  return (
    <div className="w-full">
      <p className="text-[0.68rem] font-bold uppercase tracking-[0.08em] text-slate-400 mb-4">
        Choose a rule type
      </p>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        {itemList.map((x) => (
          <ExpressionItemC
            key={x.type}
            item={props.item}
            itemI={x}
            onUpdate={props.onUpdate}
          />
        ))}
      </div>
    </div>
  );
};

const ExpressionItemC = (props: {
  item?: Expression;
  itemI: itemT;
  onUpdate: (item?: Expression) => void;
}) => {
  const isSelected = props.itemI.type === props.item?.type.oneofKind;

  return (
    <div
      className={twMerge(
        "flex flex-col rounded-lg border bg-white overflow-hidden transition-[border-color,box-shadow] duration-150",
        isSelected
          ? "border-slate-800 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
          : "border-slate-200 hover:border-slate-300 hover:shadow-[0_1px_4px_rgba(15,23,42,0.06)]",
      )}
    >
      <div
        className={twMerge(
          "text-[0.72rem] font-bold uppercase tracking-[0.04em] px-3 py-2 border-b",
          isSelected
            ? "bg-slate-900 text-white border-slate-900"
            : "bg-slate-50 border-slate-100 text-slate-600",
        )}
      >
        {props.itemI.title}
      </div>

      <div className="p-3 flex-1">
        <props.itemI.components.Edit
          item={props.item}
          onUpdate={props.onUpdate}
        />
      </div>
    </div>
  );
};

export default Cond;
