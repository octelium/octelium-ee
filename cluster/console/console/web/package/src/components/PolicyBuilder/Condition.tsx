import * as CoreP from "@/apis/corev1/corev1";
import {
  Condition,
  Condition_Expression as Expression,
} from "@/apis/enterprisev1/enterprisev1";
import {
  ActionIcon,
  Button,
  Drawer,
  Group,
  ScrollArea,
  Select,
  Tooltip,
} from "@mantine/core";
import { ChevronUp, Edit2, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

import { useDisclosure } from "@mantine/hooks";
import itemList from "./ItemList";
import { ExprChip } from "./PrintCondition";

const getNewExpression = () => {
  return Expression.create({
    type: {
      oneofKind: `serviceMode`,
      serviceMode: {
        mode: CoreP.Service_Spec_Mode.HTTP,
      },
    },
  });
};

type itemT = (typeof itemList)[0];

const Cond = (props: {
  item?: Condition;
  onChange: (condition?: Condition) => void;
  onDelete?: () => void;
}) => {
  const [localItem, setLocalItem] = React.useState<Condition>(
    props.item ??
      Condition.create({
        type: {
          oneofKind: `matchAny`,
          matchAny: true,
        },
      }),
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
        next.type = { oneofKind: "not", not: { condition: childrenToKeep[0] } };
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

  return (
    <div className="w-full bg-white rounded-lg border border-gray-200 shadow-sm mb-4 transition-all">
      <div className="flex items-center justify-between p-3 border-b border-gray-100 bg-gray-50/80 rounded-t-lg">
        <Group gap="sm">
          <Select
            size="sm"
            className="w-[180px]"
            value={currentItem.type.oneofKind ?? "expression"}
            classNames={{ input: "font-semibold bg-white" }}
            data={[
              { value: "expression", label: "Expression" },
              { value: "matchAny", label: "Match Everything" },
              { value: "all", label: "All (AND)" },
              { value: "any", label: "Any (OR)" },
              { value: "none", label: "None (NOR)" },
              { value: "not", label: "Match Not" },
            ]}
            onChange={(v) => v && handleTypeChange(v)}
          />
        </Group>

        {props.onDelete && (
          <Tooltip label="Remove" position="left">
            <ActionIcon
              // color="red"
              variant="subtle"
              onClick={props.onDelete}
              className="hover:bg-red-50 transition-colors"
            >
              <Trash2 size={18} />
            </ActionIcon>
          </Tooltip>
        )}
      </div>

      <div className="p-4">
        {match(currentItem.type)
          .with({ oneofKind: "all" }, (t) => (
            <LogicalGroup
              items={t.all.of}
              onUpdate={(newItems) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "all") next.type.all.of = newItems;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "any" }, (t) => (
            <LogicalGroup
              items={t.any.of}
              onUpdate={(newItems) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "any") next.type.any.of = newItems;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "none" }, (t) => (
            <LogicalGroup
              items={t.none.of}
              onUpdate={(newItems) => {
                const next = Condition.clone(currentItem);
                if (next.type.oneofKind === "none")
                  next.type.none.of = newItems;
                updateItem(next);
              }}
            />
          ))
          .with({ oneofKind: "not" }, (t) => (
            <div className="pl-4 border-l-2 border-red-300 ml-2">
              <Cond
                item={t.not.condition}
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
            <div className="text-sm text-gray-500 italic py-2">
              This rule will automatically match all resources.
            </div>
          ))
          .otherwise(() => null)}
      </div>
    </div>
  );
};

const LogicalGroup = (props: {
  items: Condition[];
  onUpdate: (items: Condition[]) => void;
}) => {
  const handleItemChange = (index: number, val?: Condition) => {
    const next = [...props.items];
    if (val) next[index] = val;
    props.onUpdate(next);
  };

  const handleItemDelete = (index: number) => {
    const next = [...props.items];
    next.splice(index, 1);
    props.onUpdate(next);
  };

  const handleAdd = () => {
    const next = [...props.items];
    next.push(
      Condition.create({
        type: { oneofKind: "expression", expression: getNewExpression() },
      }),
    );
    props.onUpdate(next);
  };

  return (
    <div className="flex flex-col space-y-4 pl-4 border-l-2 border-indigo-200 ml-2">
      {props.items.map((item, idx) => (
        <Cond
          key={idx}
          item={item}
          onChange={(v) => handleItemChange(idx, v)}
          onDelete={() => handleItemDelete(idx)}
        />
      ))}
      <div>
        <Button
          size="sm"
          leftSection={<Plus size={16} />}
          onClick={handleAdd}
          className="shadow-sm"
        >
          Add Condition
        </Button>
      </div>
    </div>
  );
};

const ExpressionC = (props: {
  item?: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  const [isEditing, setIsEditing] = React.useState(false);
  const [opened, { open, close }] = useDisclosure(false);
  if (!props.item) return null;

  return (
    <div className="border border-gray-200 rounded-lg shadow-sm bg-white overflow-hidden">
      <div className="flex items-center justify-between p-3 bg-gray-50/50">
        <ExprChip item={props.item} />
        <Button
          size="xs"
          leftSection={
            isEditing ? <ChevronUp size={14} /> : <Edit2 size={14} />
          }
          onClick={open}
        >
          {"Edit"}
        </Button>
      </div>

      <Drawer opened={opened} onClose={close} size={"lg"}>
        <div className="border-t border-gray-100 p-3 bg-gray-50/30">
          <ExpressionEditC item={props.item} onUpdate={props.onUpdate} />
        </div>
      </Drawer>
    </div>
  );
};

const ExpressionEditC = (props: {
  item?: Expression;
  onUpdate: (item?: Expression) => void;
}) => {
  return (
    <ScrollArea type="auto" offsetScrollbars>
      <div className="my-8 font-bold text-2xl">Choose a Rule Type</div>
      <div className="w-full">
        <div className="w-full grid grid-cols-1 md:grid-cols-2 gap-3 p-1">
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
    </ScrollArea>
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
        "flex flex-col border rounded-md bg-white transition-all overflow-hidden",
        isSelected
          ? "border-blue-400 ring-1 ring-blue-400 shadow-sm"
          : "border-gray-200 hover:border-gray-300 hover:shadow-sm",
      )}
    >
      <div
        className={twMerge(
          "text-sm font-semibold px-4 py-2 border-b",
          isSelected
            ? "bg-blue-50 border-blue-100 text-blue-900"
            : "bg-gray-50 border-gray-100 text-gray-700",
        )}
      >
        {props.itemI.title}
      </div>

      <div className="p-4 flex-1">
        <props.itemI.components.Edit
          item={props.item}
          onUpdate={props.onUpdate}
        />
      </div>
    </div>
  );
};

export default Cond;
