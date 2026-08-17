import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";

import Cond from "@/components/Condition";
import ItemMessage from "@/components/ItemMessage";
import PriorityPicker from "@/components/PriorityPicker";
import { Group, Select, Switch, TextInput } from "@mantine/core";

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
