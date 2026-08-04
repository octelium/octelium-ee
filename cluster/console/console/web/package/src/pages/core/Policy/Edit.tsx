import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";

import Cond from "@/components/Condition";
import ItemMessage from "@/components/ItemMessage";
import { Group, Select, Slider, Switch, TextInput } from "@mantine/core";

const Edit = (props: {
  item: CoreP.Policy;
  onUpdate: (item: CoreP.Policy) => void;
}) => {
  const [req, setReq] = React.useState(CoreP.Policy.clone(props.item));
  const ruleKeys = React.useRef(
    props.item.spec!.rules.map(() => crypto.randomUUID()),
  );
  const enforcementRuleKeys = React.useRef(
    props.item.spec!.enforcementRules.map(() => crypto.randomUUID()),
  );
  const itemKey = props.item.metadata?.uid || props.item.metadata?.name;

  React.useEffect(() => {
    setReq(CoreP.Policy.clone(props.item));
    ruleKeys.current = props.item.spec!.rules.map(() => crypto.randomUUID());
    enforcementRuleKeys.current = props.item.spec!.enforcementRules.map(() =>
      crypto.randomUUID(),
    );
  }, [itemKey]);

  const updateReq = () => {
    const next = CoreP.Policy.clone(req);
    setReq(next);
    props.onUpdate(CoreP.Policy.clone(next));
  };

  return (
    <div className="w-full">
      <Group grow>
        <Switch
          label="Disabled"
          description="Disable/deactivate the Policy"
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>
      <ItemMessage
        title="Rules"
        obj={req.spec?.rules}
        isList
        onSet={() => {
          req.spec!.rules = [CoreP.Policy_Spec_Rule.create({})];
          ruleKeys.current = [crypto.randomUUID()];
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.rules.push(CoreP.Policy_Spec_Rule.create({}));
          ruleKeys.current.push(crypto.randomUUID());
          updateReq();
        }}
      >
        {req.spec!.rules &&
          req.spec!.rules.map((rule: any, ruleIdx: number) => (
            <EditItem
              key={ruleKeys.current[ruleIdx]}
              obj={req.spec!.rules[ruleIdx]}
              onUnset={() => {
                req.spec!.rules.splice(ruleIdx, 1);
                ruleKeys.current.splice(ruleIdx, 1);
                updateReq();
              }}
            >
              <Group grow>
                <Select
                  label="Effect"
                  required
                  description="Set the effect to either ALLOW or DENY"
                  data={[
                    {
                      label: "Allow",
                      value:
                        CoreP.Policy_Spec_Rule_Effect[
                          CoreP.Policy_Spec_Rule_Effect.ALLOW
                        ],
                    },
                    {
                      label: "Deny",
                      value:
                        CoreP.Policy_Spec_Rule_Effect[
                          CoreP.Policy_Spec_Rule_Effect.DENY
                        ],
                    },
                  ]}
                  value={
                    CoreP.Policy_Spec_Rule_Effect[
                      req.spec!.rules[ruleIdx].effect
                    ]
                  }
                  onChange={(v) => {
                    if (!v) return;
                    req.spec!.rules[ruleIdx].effect =
                      CoreP.Policy_Spec_Rule_Effect[v as "ALLOW"];
                    updateReq();
                  }}
                />
                <TextInput
                  label="Name"
                  // required
                  description="Set an optional, descriptive name for the rule"
                  placeholder="my-rule"
                  value={req.spec!.rules[ruleIdx].name}
                  onChange={(v) => {
                    req.spec!.rules[ruleIdx].name = v.target.value;
                    updateReq();
                  }}
                />

                <PriorityPicker
                  label="Priority"
                  value={req.spec!.rules[ruleIdx].priority}
                  onChange={(v) => {
                    req.spec!.rules[ruleIdx].priority = v;
                    updateReq();
                  }}
                />
              </Group>

              <Cond
                item={req.spec!.rules[ruleIdx].condition}
                onChange={(v) => {
                  req.spec!.rules[ruleIdx].condition = v;
                  updateReq();
                }}
              />
            </EditItem>
          ))}
      </ItemMessage>

      <ItemMessage
        title="Enforcement Rules"
        obj={req.spec?.enforcementRules}
        isList
        onSet={() => {
          req.spec!.enforcementRules = [
            CoreP.Policy_Spec_EnforcementRule.create({}),
          ];
          enforcementRuleKeys.current = [crypto.randomUUID()];
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.enforcementRules.push(
            CoreP.Policy_Spec_EnforcementRule.create({}),
          );
          enforcementRuleKeys.current.push(crypto.randomUUID());
          updateReq();
        }}
      >
        {req.spec!.enforcementRules &&
          req.spec!.enforcementRules.map((rule: any, ruleIdx: number) => (
            <EditItem
              key={enforcementRuleKeys.current[ruleIdx]}
              obj={req.spec!.enforcementRules[ruleIdx]}
              onUnset={() => {
                req.spec!.enforcementRules.splice(ruleIdx, 1);
                enforcementRuleKeys.current.splice(ruleIdx, 1);
                updateReq();
              }}
            >
              <Group>
                <Select
                  label="Effect"
                  required
                  description="Set the effect to either ALLOW or DENY"
                  data={[
                    {
                      label: "Ignore",
                      value:
                        CoreP.Policy_Spec_EnforcementRule_Effect[
                          CoreP.Policy_Spec_EnforcementRule_Effect.IGNORE
                        ],
                    },
                    {
                      label: "Enforce",
                      value:
                        CoreP.Policy_Spec_EnforcementRule_Effect[
                          CoreP.Policy_Spec_EnforcementRule_Effect.ENFORCE
                        ],
                    },
                  ]}
                  value={
                    CoreP.Policy_Spec_EnforcementRule_Effect[
                      req.spec!.enforcementRules[ruleIdx].effect
                    ]
                  }
                  onChange={(v) => {
                    if (!v) return;
                    req.spec!.enforcementRules[ruleIdx].effect =
                      CoreP.Policy_Spec_EnforcementRule_Effect[v as "ENFORCE"];
                    updateReq();
                  }}
                />
              </Group>

              <Cond
                item={req.spec!.enforcementRules[ruleIdx].condition}
                onChange={(v) => {
                  req.spec!.enforcementRules[ruleIdx].condition = v;
                  updateReq();
                }}
              />
            </EditItem>
          ))}
      </ItemMessage>
    </div>
  );
};

export default Edit;

import { twMerge } from "tailwind-merge";

const PRIORITY_STEPS = [-4, -3, -2, -1, 0, 1, 2, 3, 4] as const;
type Priority = (typeof PRIORITY_STEPS)[number];

const priorityMeta = (p: Priority) => {
  if (p <= -3)
    return {
      hint: "Highest",
      color: "border-slate-700 bg-slate-800 text-white",
    };
  if (p <= -1)
    return {
      hint: "High",
      color: "border-slate-300 bg-slate-100 text-slate-700",
    };
  if (p === 0)
    return {
      hint: "Default",
      color: "border-blue-200 bg-blue-50 text-blue-700",
    };
  if (p <= 2)
    return {
      hint: "Low",
      color: "border-amber-200 bg-amber-50 text-amber-700",
    };
  return {
      hint: "Lowest",
      color: "border-orange-200 bg-orange-50 text-orange-700",
  };
};

const PriorityPicker = (props: {
  value: number;
  onChange: (v: number) => void;
  label?: string;
  description?: string;
}) => {
  const value = Math.min(4, Math.max(-4, props.value)) as Priority;
  const valueLabel = value > 0 ? `+${value}` : `${value}`;
  const meta = priorityMeta(value);

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 shadow-[0_1px_3px_rgba(15,23,42,0.035)]">
      <div className="flex items-start justify-between gap-3 border-b border-slate-100 bg-white px-3 py-2.5">
        <div className="min-w-0">
          {props.label && (
            <div className="text-[0.72rem] font-bold text-slate-700">
              {props.label}
            </div>
          )}
          {props.description && (
            <div className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
              {props.description}
            </div>
          )}
        </div>
        <span
          className={twMerge(
            "inline-flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-[0.65rem] font-bold",
            meta.color,
          )}
        >
          {meta.hint}
          <span className="opacity-70">{valueLabel}</span>
        </span>
      </div>

      <div className="px-4 pb-5 pt-4">
        <Slider
          aria-label={props.label ?? "Rule priority"}
          min={-4}
          max={4}
          step={1}
          value={value}
          onChange={props.onChange}
          color="dark"
          size="sm"
          label={(next) => {
            const priority = next as Priority;
            const formatted = priority > 0 ? `+${priority}` : `${priority}`;
            return `${priorityMeta(priority).hint} · ${formatted}`;
          }}
          marks={[
            { value: -4, label: "−4 · Earlier" },
            { value: 0, label: "0 · Default" },
            { value: 4, label: "+4 · Later" },
          ]}
          styles={{
            markLabel: {
              fontSize: "0.61rem",
              fontWeight: 700,
              color: "#94a3b8",
              whiteSpace: "nowrap",
            },
          }}
        />
      </div>
    </div>
  );
};
