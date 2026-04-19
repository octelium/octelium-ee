import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";

import Cond from "@/components/Condition";
import ItemMessage from "@/components/ItemMessage";
import { Group, Select, Switch, TextInput } from "@mantine/core";

const Edit = (props: {
  item: CoreP.Policy;
  onUpdate: (item: CoreP.Policy) => void;
}) => {
  let [req, setReq] = React.useState(props.item);
  const updateReq = () => {
    setReq(CoreP.Policy.clone(req));
    props.onUpdate(req);
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
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.rules.push(CoreP.Policy_Spec_Rule.create({}));
          updateReq();
        }}
      >
        {req.spec!.rules &&
          req.spec!.rules.map((rule: any, ruleIdx: number) => (
            <EditItem
              obj={req.spec!.rules[ruleIdx]}
              onUnset={() => {
                req.spec!.rules.splice(ruleIdx, 1);
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
                  defaultValue={
                    CoreP.Policy_Spec_Rule_Effect[
                      req.spec!.rules[ruleIdx].effect
                    ]
                  }
                  onChange={(v) => {
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
                  description="Lower values execute first. Default is 0."
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
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.enforcementRules.push(
            CoreP.Policy_Spec_EnforcementRule.create({}),
          );
          updateReq();
        }}
      >
        {req.spec!.enforcementRules &&
          req.spec!.enforcementRules.map((rule: any, ruleIdx: number) => (
            <EditItem
              obj={req.spec!.enforcementRules[ruleIdx]}
              onUnset={() => {
                req.spec!.enforcementRules.splice(ruleIdx, 1);
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
                  defaultValue={
                    CoreP.Policy_Spec_Rule_Effect[
                      req.spec!.enforcementRules[ruleIdx].effect
                    ]
                  }
                  onChange={(v) => {
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
      label: `${p}`,
      hint: "Highest",
      color: "bg-slate-900 text-white border-slate-900",
    };
  if (p <= -1)
    return {
      label: `${p}`,
      hint: "High",
      color: "bg-slate-700 text-white border-slate-700",
    };
  if (p === 0)
    return {
      label: "0",
      hint: "Default",
      color: "bg-blue-600 text-white border-blue-600",
    };
  if (p <= 2)
    return {
      label: `+${p}`,
      hint: "Low",
      color: "bg-slate-300 text-slate-700 border-slate-300",
    };
  return {
    label: `+${p}`,
    hint: "Lowest",
    color: "bg-slate-200 text-slate-500 border-slate-200",
  };
};

const PriorityPicker = (props: {
  value: number;
  onChange: (v: number) => void;
  label?: string;
  description?: string;
}) => {
  const value = Math.min(4, Math.max(-4, props.value)) as Priority;

  return (
    <div className="flex flex-col gap-1.5">
      {props.label && (
        <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-600">
          {props.label}
        </span>
      )}
      {props.description && (
        <span className="text-[0.7rem] font-semibold text-slate-400">
          {props.description}
        </span>
      )}

      <div className="flex items-stretch gap-0 rounded-lg overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)] bg-white">
        {PRIORITY_STEPS.map((step) => {
          const isActive = step === value;
          const meta = priorityMeta(step);
          return (
            <button
              key={step}
              onClick={() => props.onChange(step)}
              title={meta.hint}
              className={twMerge(
                "flex-1 flex flex-col items-center justify-center py-2 gap-0.5",
                "text-[0.7rem] font-bold cursor-pointer",
                "transition-colors duration-150",
                "border-r border-slate-100 last:border-r-0",
                isActive
                  ? meta.color
                  : "text-slate-500 hover:bg-slate-50 hover:text-slate-800",
              )}
            >
              <span className="leading-none">
                {step > 0 ? `+${step}` : step}
              </span>
            </button>
          );
        })}
      </div>

      <div className="flex items-center justify-between mt-0.5">
        <span className="text-[0.62rem] font-bold uppercase tracking-[0.06em] text-slate-400">
          ← Highest priority
        </span>
        <span className="text-[0.7rem] font-semibold text-slate-600">
          {priorityMeta(value).hint}{" "}
          <span className="font-bold text-slate-800">
            ({value > 0 ? `+${value}` : value})
          </span>
        </span>
        <span className="text-[0.62rem] font-bold uppercase tracking-[0.06em] text-slate-400">
          Lowest priority →
        </span>
      </div>
    </div>
  );
};
