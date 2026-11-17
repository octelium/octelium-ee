import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import Cond from "@/components/Condition";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import { strToNum } from "@/utils/convert";
import { ActionIcon, NumberInput, Tooltip } from "@mantine/core";
import { ArrowDown, ArrowUp, Trash2 } from "lucide-react";
import * as React from "react";

const newCondition = () =>
  CordiumP.Condition.create({ type: { oneofKind: "match", match: "" } });

export const ComputeLimit = (props: {
  title: string;
  description: string;
  value?: CordiumP.Workspace_Spec_Limit;
  onChange: (value?: CordiumP.Workspace_Spec_Limit) => void;
}) => (
  <EditItem
    title={props.title}
    description={props.description}
    obj={props.value}
    onSet={() => props.onChange(CordiumP.Workspace_Spec_Limit.create())}
    onUnset={() => props.onChange(undefined)}
  >
    {props.value && (
      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <NumberInput
          label="CPU (millicores)"
          description="1000 millicores = 1 vCPU."
          placeholder="Not set"
          min={0}
          max={4294967295}
          allowDecimal={false}
          value={props.value.cpu?.millicores ?? ""}
          onChange={(value) => {
            const next = CordiumP.Workspace_Spec_Limit.clone(props.value!);
            next.cpu = value === "" ? undefined : { millicores: strToNum(value) };
            props.onChange(next);
          }}
        />
        <NumberInput
          label="Memory (MB)"
          description="Memory in megabytes."
          placeholder="Not set"
          min={0}
          max={4294967295}
          allowDecimal={false}
          value={props.value.memory?.megabytes ?? ""}
          onChange={(value) => {
            const next = CordiumP.Workspace_Spec_Limit.clone(props.value!);
            next.memory = value === "" ? undefined : { megabytes: strToNum(value) };
            props.onChange(next);
          }}
        />
        <NumberInput
          label="Storage (MB)"
          description="Persistent disk in megabytes."
          placeholder="Not set"
          min={0}
          max={4294967295}
          allowDecimal={false}
          value={props.value.storage?.megabytes ?? ""}
          onChange={(value) => {
            const next = CordiumP.Workspace_Spec_Limit.clone(props.value!);
            next.storage = value === "" ? undefined : { megabytes: strToNum(value) };
            props.onChange(next);
          }}
        />
      </div>
    )}
  </EditItem>
);

export const RuleList = <T extends { condition?: CordiumP.Condition }>(props: {
  title: string;
  description: string;
  items: T[];
  create: () => T;
  onChange: (items: T[]) => void;
  children: (item: T, onChange: (item: T) => void) => React.ReactNode;
}) => {
  const [ids, setIDs] = React.useState(() => props.items.map(() => crypto.randomUUID()));

  const move = (index: number, direction: number) => {
    const next = [...props.items];
    const target = index + direction;
    [next[index], next[target]] = [next[target], next[index]];
    const nextIDs = [...ids];
    [nextIDs[index], nextIDs[target]] = [nextIDs[target], nextIDs[index]];
    setIDs(nextIDs);
    props.onChange(next);
  };

  return (
    <ItemMessage
      title={props.title}
      description={props.description}
      obj={props.items}
      isList
      onAddListItem={() => {
        setIDs([...ids, crypto.randomUUID()]);
        props.onChange([...props.items, props.create()]);
      }}
    >
      <div className="space-y-3">
        {props.items.map((item, index) => {
          const update = (next: T) =>
            props.onChange(props.items.map((value, i) => i === index ? next : value));
          return (
            <div key={ids[index]} className="min-w-0 rounded-lg border border-slate-200 bg-slate-50/60 p-3 sm:p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <span className="text-xs font-semibold text-slate-600">Rule {index + 1}</span>
                <div className="flex items-center gap-1">
                  <Tooltip label="Move up" withArrow>
                    <ActionIcon type="button" variant="subtle" color="gray" size="sm" aria-label={`Move rule ${index + 1} up`} disabled={index === 0} onClick={() => move(index, -1)}>
                      <ArrowUp size={13} />
                    </ActionIcon>
                  </Tooltip>
                  <Tooltip label="Move down" withArrow>
                    <ActionIcon type="button" variant="subtle" color="gray" size="sm" aria-label={`Move rule ${index + 1} down`} disabled={index === props.items.length - 1} onClick={() => move(index, 1)}>
                      <ArrowDown size={13} />
                    </ActionIcon>
                  </Tooltip>
                  <Tooltip label="Remove rule" withArrow>
                    <ActionIcon type="button" variant="subtle" color="red" size="sm" aria-label={`Remove rule ${index + 1}`} onClick={() => {
                      setIDs(ids.filter((_, i) => i !== index));
                      props.onChange(props.items.filter((_, i) => i !== index));
                    }}>
                      <Trash2 size={13} />
                    </ActionIcon>
                  </Tooltip>
                </div>
              </div>
              {props.children(item, update)}
              <Cond item={item.condition ?? newCondition()} onChange={(condition) => update({ ...item, condition })} />
            </div>
          );
        })}
      </div>
    </ItemMessage>
  );
};
