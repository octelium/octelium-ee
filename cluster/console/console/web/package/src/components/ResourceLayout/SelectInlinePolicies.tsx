import * as CoreP from "@/apis/corev1/corev1";
import { InlinePolicy } from "@/apis/corev1/corev1";
import Edit from "@/pages/core/Policy/Edit";
import { CloseButton, TextInput } from "@mantine/core";
import * as React from "react";
import ItemMessage from "../ItemMessage";

type Entry = { id: string; policy: InlinePolicy };

let nextEntryID = 0;
const createEntryID = () => `inline-policy-${++nextEntryID}`;

const createPolicy = (index: number): InlinePolicy =>
  CoreP.InlinePolicy.create({
    name: `inline-policy-${index + 1}`,
    spec: { rules: [CoreP.Policy_Spec_Rule.create()] },
  });

const SelectInlinePolicies = (props: {
  inlinePolicies: InlinePolicy[];
  onUpdate: (inlinePolicies: InlinePolicy[]) => void;
}) => {
  const [entries, setEntries] = React.useState<Entry[]>(() =>
    props.inlinePolicies.map((policy) => ({
      id: createEntryID(),
      policy: CoreP.InlinePolicy.clone(policy),
    })),
  );

  React.useEffect(() => {
    setEntries((current) =>
      props.inlinePolicies.map((policy, index) => ({
        id: current[index]?.id ?? createEntryID(),
        policy: CoreP.InlinePolicy.clone(policy),
      })),
    );
  }, [props.inlinePolicies]);

  const commit = (next: Entry[]) => {
    setEntries(next);
    props.onUpdate(
      next.map(({ policy }) => CoreP.InlinePolicy.clone(policy)),
    );
  };

  const addPolicy = () => {
    commit([
      ...entries,
      { id: createEntryID(), policy: createPolicy(entries.length) },
    ]);
  };

  return (
    <div className="w-full">
      <ItemMessage
        title="Inline Policies"
        obj={entries}
        isList
        onSet={addPolicy}
        onAddListItem={addPolicy}
      >
        <div className="flex flex-col gap-6">
          {entries.map(({ id, policy }, index) => (
            <div
              className="w-full rounded-lg border border-slate-200 p-4"
              key={id}
            >
              <div className="mb-4 flex items-start gap-2">
                <TextInput
                  className="flex-1"
                  required
                  label="Name"
                  description="Set a unique name for this inline Policy"
                  value={policy.name}
                  onChange={(event) => {
                    const next = entries.map((entry, entryIndex) =>
                      entryIndex === index
                        ? {
                            ...entry,
                            policy: CoreP.InlinePolicy.create({
                              ...entry.policy,
                              name: event.currentTarget.value,
                            }),
                          }
                        : entry,
                    );
                    commit(next);
                  }}
                />
                <CloseButton
                  mt={22}
                  aria-label={`Remove inline Policy ${policy.name || index + 1}`}
                  onClick={() =>
                    commit(entries.filter((_, entryIndex) => entryIndex !== index))
                  }
                />
              </div>

              <Edit
                item={CoreP.Policy.create({ spec: policy.spec })}
                onUpdate={(updated) => {
                  const next = entries.map((entry, entryIndex) =>
                    entryIndex === index
                      ? {
                          ...entry,
                          policy: CoreP.InlinePolicy.create({
                            ...entry.policy,
                            spec: updated.spec,
                          }),
                        }
                      : entry,
                  );
                  commit(next);
                }}
              />
            </div>
          ))}
        </div>
      </ItemMessage>
    </div>
  );
};

export default SelectInlinePolicies;
