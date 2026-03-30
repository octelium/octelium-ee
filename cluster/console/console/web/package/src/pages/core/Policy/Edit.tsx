import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";

import Cond from "@/components/Condition";
import ItemMessage from "@/components/ItemMessage";
import { Group, NumberInput, Select, Switch, TextInput } from "@mantine/core";

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
                <TextInput
                  label="Name"
                  // required
                  description="Set the rule name"
                  placeholder="my-rule"
                  value={req.spec!.rules[ruleIdx].name}
                  onChange={(v) => {
                    req.spec!.rules[ruleIdx].name = v.target.value;
                    updateReq();
                  }}
                />

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

                <NumberInput
                  label="Priority"
                  description="Set the rule priority (the lower the number the higher th priority)"
                  min={-4}
                  max={4}
                  value={req.spec!.rules[ruleIdx].priority}
                  onChange={(v) => {
                    req.spec!.rules[ruleIdx].priority = v as number;
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
