import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import * as React from "react";

import { CELEditor } from "@/components/Condition/Editor";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { strToNum } from "@/utils/convert";
import { getResourceRef } from "@/utils/pb";
import {
  Group,
  Input,
  NumberInput,
  Select,
  Switch,
  TextInput,
} from "@mantine/core";
import { match } from "ts-pattern";

const newCondition = (): AccessP.Policy_Spec_Rule_Condition =>
  AccessP.Policy_Spec_Rule_Condition.create({
    type: {
      oneofKind: "subject",
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: {
          oneofKind: "userRef",
          userRef: MetaP.ObjectReference.create(),
        },
      }),
    },
  });

const ConditionEdit = (props: {
  condition: AccessP.Policy_Spec_Rule_Condition;
  onUpdate: () => void;
}) => {
  const { condition, onUpdate } = props;

  return (
    <div className="w-full">
      <Select
        label="Condition type"
        required
        description="Set the type of the condition"
        data={[
          { label: "Match Any (always matches)", value: "matchAny" },
          { label: "Subject", value: "subject" },
          { label: "Resource", value: "resource" },
          { label: "All of", value: "all" },
          { label: "Any of", value: "any" },
          { label: "Requester", value: "userRef" },
          { label: "CEL Match", value: "match" },
        ]}
        value={condition.type.oneofKind}
        onChange={(v) => {
          condition.type = match(v)
            .with("matchAny", () => ({
              oneofKind: "matchAny" as const,
              matchAny: true,
            }))
            .with("subject", () => ({
              oneofKind: "subject" as const,
              subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
                type: {
                  oneofKind: "userRef",
                  userRef: MetaP.ObjectReference.create(),
                },
              }),
            }))
            .with("resource", () => ({
              oneofKind: "resource" as const,
              resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
                type: {
                  oneofKind: "serviceRef",
                  serviceRef: MetaP.ObjectReference.create(),
                },
              }),
            }))
            .with("all", () => ({
              oneofKind: "all" as const,
              all: AccessP.Policy_Spec_Rule_Condition_All.create(),
            }))
            .with("any", () => ({
              oneofKind: "any" as const,
              any: AccessP.Policy_Spec_Rule_Condition_Any.create(),
            }))
            .with("userRef", () => ({
              oneofKind: "userRef" as const,
              userRef: MetaP.ObjectReference.create(),
            }))
            .otherwise(() => ({
              oneofKind: "match" as const,
              match: "",
            }));
          onUpdate();
        }}
      />

      <div className="mt-3">
        {match(condition.type)
          .when(
            (x) => x.oneofKind === "matchAny",
            (matchAny) => (
              <Switch
                label="Match any"
                description="When enabled, this condition always matches"
                checked={matchAny.matchAny}
                onChange={(v) => {
                  matchAny.matchAny = v.target.checked;
                  onUpdate();
                }}
              />
            ),
          )
          .when(
            (x) => x.oneofKind === "subject",
            (subject) => (
              <Group grow align="flex-start">
                <Select
                  label="Subject type"
                  required
                  description="Match a User or a Group"
                  data={[
                    { label: "User", value: "userRef" },
                    { label: "Group", value: "groupRef" },
                  ]}
                  value={subject.subject.type.oneofKind}
                  onChange={(v) => {
                    subject.subject.type = match(v)
                      .with("groupRef", () => ({
                        oneofKind: "groupRef" as const,
                        groupRef: MetaP.ObjectReference.create(),
                      }))
                      .otherwise(() => ({
                        oneofKind: "userRef" as const,
                        userRef: MetaP.ObjectReference.create(),
                      }));
                    onUpdate();
                  }}
                />
                {subject.subject.type.oneofKind === "userRef" && (
                  <SelectResource
                    api="core"
                    kind="User"
                    label="User"
                    description="Select the User"
                    defaultValue={subject.subject.type.userRef.name}
                    onChange={(v) => {
                      if (subject.subject.type.oneofKind === "userRef") {
                        v
                          ? (subject.subject.type.userRef = getResourceRef(v))
                          : (subject.subject.type.userRef =
                              MetaP.ObjectReference.create());
                        onUpdate();
                      }
                    }}
                  />
                )}
                {subject.subject.type.oneofKind === "groupRef" && (
                  <SelectResource
                    api="core"
                    kind="Group"
                    label="Group"
                    description="Select the Group"
                    defaultValue={subject.subject.type.groupRef.name}
                    onChange={(v) => {
                      if (subject.subject.type.oneofKind === "groupRef") {
                        subject.subject.type.groupRef = v
                          ? getResourceRef(v)
                          : MetaP.ObjectReference.create();
                        onUpdate();
                      }
                    }}
                  />
                )}
              </Group>
            ),
          )
          .when(
            (x) => x.oneofKind === "resource",
            (resource) => (
              <Group grow align="flex-start">
                <Select
                  label="Resource type"
                  required
                  description="Match a Service or a Catalog"
                  data={[
                    { label: "Service", value: "serviceRef" },
                    { label: "Catalog", value: "catalogRef" },
                  ]}
                  value={resource.resource.type.oneofKind}
                  onChange={(v) => {
                    resource.resource.type = match(v)
                      .with("catalogRef", () => ({
                        oneofKind: "catalogRef" as const,
                        catalogRef: MetaP.ObjectReference.create(),
                      }))
                      .otherwise(() => ({
                        oneofKind: "serviceRef" as const,
                        serviceRef: MetaP.ObjectReference.create(),
                      }));
                    onUpdate();
                  }}
                />
                {resource.resource.type.oneofKind === "serviceRef" && (
                  <SelectResource
                    api="core"
                    kind="Service"
                    label="Service"
                    description="Select the Service"
                    defaultValue={resource.resource.type.serviceRef.name}
                    onChange={(v) => {
                      if (resource.resource.type.oneofKind === "serviceRef") {
                        resource.resource.type.serviceRef = v
                          ? getResourceRef(v)
                          : MetaP.ObjectReference.create();
                        onUpdate();
                      }
                    }}
                  />
                )}
                {resource.resource.type.oneofKind === "catalogRef" && (
                  <SelectResource
                    api="access"
                    kind="Catalog"
                    label="Catalog"
                    description="Select the Catalog"
                    defaultValue={resource.resource.type.catalogRef.name}
                    onChange={(v) => {
                      if (resource.resource.type.oneofKind === "catalogRef") {
                        resource.resource.type.catalogRef = v
                          ? getResourceRef(v)
                          : MetaP.ObjectReference.create();
                        onUpdate();
                      }
                    }}
                  />
                )}
              </Group>
            ),
          )
          .when(
            (x) => x.oneofKind === "all",
            (all) => (
              <ItemMessage
                title="All of"
                obj={all.all.of}
                isList
                onSet={() => {
                  all.all.of = [newCondition()];
                  onUpdate();
                }}
                onAddListItem={() => {
                  all.all.of.push(newCondition());
                  onUpdate();
                }}
              >
                {all.all.of.map((sub, idx) => (
                  <EditItem
                    key={idx}
                    obj={sub}
                    onUnset={() => {
                      all.all.of.splice(idx, 1);
                      onUpdate();
                    }}
                  >
                    <ConditionEdit condition={sub} onUpdate={onUpdate} />
                  </EditItem>
                ))}
              </ItemMessage>
            ),
          )
          .when(
            (x) => x.oneofKind === "any",
            (any) => (
              <ItemMessage
                title="Any of"
                obj={any.any.of}
                isList
                onSet={() => {
                  any.any.of = [newCondition()];
                  onUpdate();
                }}
                onAddListItem={() => {
                  any.any.of.push(newCondition());
                  onUpdate();
                }}
              >
                {any.any.of.map((sub, idx) => (
                  <EditItem
                    key={idx}
                    obj={sub}
                    onUnset={() => {
                      any.any.of.splice(idx, 1);
                      onUpdate();
                    }}
                  >
                    <ConditionEdit condition={sub} onUpdate={onUpdate} />
                  </EditItem>
                ))}
              </ItemMessage>
            ),
          )
          .when(
            (x) => x.oneofKind === "userRef",
            (userRef) => (
              <SelectResource
                api="core"
                kind="User"
                label="Requester User"
                description="Matches the requester against this User"
                defaultValue={userRef.userRef.name}
                onChange={(v) => {
                  userRef.userRef = v
                    ? getResourceRef(v)
                    : MetaP.ObjectReference.create();
                  onUpdate();
                }}
              />
            ),
          )
          .when(
            (x) => x.oneofKind === "match",
            (matchType) => (
              <Input.Wrapper
                label="CEL Expression"
                description="Write a CEL expression that must evaluate to true"
              >
                <CELEditor
                  exp={matchType.match}
                  onChange={(v: string) => {
                    matchType.match = v ?? "";
                    onUpdate();
                  }}
                />
              </Input.Wrapper>
            ),
          )
          .otherwise(() => (
            <></>
          ))}
      </div>
    </div>
  );
};

const ReviewStepEdit = (props: {
  step: AccessP.Policy_Spec_Rule_Action_Review_Step;
  onUpdate: () => void;
}) => {
  const { step, onUpdate } = props;

  return (
    <div className="w-full">
      <ItemMessage
        title="Reviewers"
        obj={step.reviewers}
        isList
        onSet={() => {
          step.reviewers = [
            AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer.create({
              type: {
                oneofKind: "user",
                user: { userRef: MetaP.ObjectReference.create() },
              },
            }),
          ];
          onUpdate();
        }}
        onAddListItem={() => {
          step.reviewers.push(
            AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer.create({
              type: {
                oneofKind: "user",
                user: { userRef: MetaP.ObjectReference.create() },
              },
            }),
          );
          onUpdate();
        }}
      >
        {step.reviewers.map((reviewer, idx) => (
          <EditItem
            key={idx}
            obj={reviewer}
            onUnset={() => {
              step.reviewers.splice(idx, 1);
              onUpdate();
            }}
          >
            <Group grow align="flex-start">
              <Select
                label="Reviewer type"
                required
                description="Set the reviewer as a User or a Group"
                data={[
                  { label: "User", value: "user" },
                  { label: "Group", value: "group" },
                ]}
                value={reviewer.type.oneofKind}
                onChange={(v) => {
                  reviewer.type = match(v)
                    .with("group", () => ({
                      oneofKind: "group" as const,
                      group: { groupRef: MetaP.ObjectReference.create() },
                    }))
                    .otherwise(() => ({
                      oneofKind: "user" as const,
                      user: { userRef: MetaP.ObjectReference.create() },
                    }));
                  onUpdate();
                }}
              />
              {reviewer.type.oneofKind === "user" && (
                <SelectResource
                  api="core"
                  kind="User"
                  label="User"
                  description="Select the reviewer User"
                  defaultValue={reviewer.type.user.userRef?.name}
                  onChange={(v) => {
                    if (reviewer.type.oneofKind === "user") {
                      v
                        ? (reviewer.type.user.userRef = getResourceRef(v))
                        : (reviewer.type.user.userRef = undefined);
                      onUpdate();
                    }
                  }}
                />
              )}
              {reviewer.type.oneofKind === "group" && (
                <SelectResource
                  api="core"
                  kind="Group"
                  label="Group"
                  description="Select the reviewer Group"
                  defaultValue={reviewer.type.group.groupRef?.name}
                  onChange={(v) => {
                    if (reviewer.type.oneofKind === "group") {
                      v
                        ? (reviewer.type.group.groupRef = getResourceRef(v))
                        : (reviewer.type.group.groupRef = undefined);
                      onUpdate();
                    }
                  }}
                />
              )}
            </Group>
          </EditItem>
        ))}
      </ItemMessage>

      <Group grow>
        <Select
          label="Approval Requirement"
          required
          description="Set how many reviewers must approve"
          data={[
            {
              label: "Any",
              value:
                AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
                  AccessP
                    .Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY
                ],
            },
            {
              label: "All",
              value:
                AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
                  AccessP
                    .Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ALL
                ],
            },
            {
              label: "Count",
              value:
                AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
                  AccessP
                    .Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
                    .COUNT
                ],
            },
          ]}
          value={
            AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
              step.approvalRequirement
            ]
          }
          onChange={(v) => {
            if (!v) return;
            step.approvalRequirement =
              AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
                v as "ANY"
              ];
            onUpdate();
          }}
        />

        {step.approvalRequirement ===
          AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
            .COUNT && (
          <NumberInput
            label="Approval Count"
            description="Set the number of required approvals"
            min={0}
            value={step.approvalCount}
            onChange={(v) => {
              step.approvalCount = strToNum(v);
              onUpdate();
            }}
          />
        )}

        <Select
          label="On Timeout"
          description="Set the behavior when the step times out"
          data={[
            {
              label: "Go to next step",
              value:
                AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout[
                  AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout
                    .GOTO_NEXT_STEP
                ],
            },
            {
              label: "Reject",
              value:
                AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout[
                  AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT
                ],
            },
          ]}
          value={
            AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout[
              step.onTimeout
            ]
          }
          onChange={(v) => {
            if (!v) return;
            step.onTimeout =
              AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout[
                v as "REJECT"
              ];
            onUpdate();
          }}
        />
      </Group>

      <DurationPicker
        value={step.timeout}
        title="Timeout"
        onChange={(v) => {
          step.timeout = v;
          onUpdate();
        }}
      />
    </div>
  );
};

const RuleEdit = (props: {
  rule: AccessP.Policy_Spec_Rule;
  onUpdate: () => void;
}) => {
  const { rule, onUpdate } = props;

  return (
    <div className="w-full">
      <Group grow>
        <TextInput
          label="Name"
          description="Set an optional name for the rule"
          placeholder="my-rule"
          value={rule.name}
          onChange={(v) => {
            rule.name = v.target.value;
            onUpdate();
          }}
        />

        <NumberInput
          label="Priority"
          description="Set the rule priority"
          value={rule.priority}
          onChange={(v) => {
            rule.priority = strToNum(v);
            onUpdate();
          }}
        />

        <Select
          label="Effect"
          required
          description="Set the effect when the rule matches"
          data={[
            {
              label: "Deny",
              value:
                AccessP.Policy_Spec_Rule_Effect[
                  AccessP.Policy_Spec_Rule_Effect.DENY
                ],
            },
            {
              label: "Review",
              value:
                AccessP.Policy_Spec_Rule_Effect[
                  AccessP.Policy_Spec_Rule_Effect.REVIEW
                ],
            },
            {
              label: "Auto Approve",
              value:
                AccessP.Policy_Spec_Rule_Effect[
                  AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE
                ],
            },
          ]}
          value={AccessP.Policy_Spec_Rule_Effect[rule.effect]}
          onChange={(v) => {
            if (!v) return;
            rule.effect = AccessP.Policy_Spec_Rule_Effect[v as "DENY"];
            onUpdate();
          }}
        />
      </Group>

      <EditItem
        title="Condition"
        description="Set the rule's matching condition"
        onUnset={() => {
          rule.condition = undefined;
          onUpdate();
        }}
        obj={rule.condition}
        onSet={() => {
          rule.condition = newCondition();
          onUpdate();
        }}
      >
        {rule.condition && (
          <ConditionEdit condition={rule.condition} onUpdate={onUpdate} />
        )}
      </EditItem>

      <EditItem
        title="Action"
        description="Set the review action triggered when the rule matches"
        onUnset={() => {
          rule.action = undefined;
          onUpdate();
        }}
        obj={rule.action}
        onSet={() => {
          rule.action = AccessP.Policy_Spec_Rule_Action.create({
            type: {
              oneofKind: "review",
              review: AccessP.Policy_Spec_Rule_Action_Review.create(),
            },
          });
          onUpdate();
        }}
      >
        {rule.action &&
          match(rule.action.type)
            .when(
              (x) => x.oneofKind === "review",
              (review) => (
                <ItemMessage
                  title="Review Steps"
                  obj={review.review.steps}
                  isList
                  onSet={() => {
                    review.review.steps = [
                      AccessP.Policy_Spec_Rule_Action_Review_Step.create(),
                    ];
                    onUpdate();
                  }}
                  onAddListItem={() => {
                    review.review.steps.push(
                      AccessP.Policy_Spec_Rule_Action_Review_Step.create(),
                    );
                    onUpdate();
                  }}
                >
                  {review.review.steps.map((step, idx) => (
                    <EditItem
                      key={idx}
                      obj={step}
                      onUnset={() => {
                        review.review.steps.splice(idx, 1);
                        onUpdate();
                      }}
                    >
                      <ReviewStepEdit step={step} onUpdate={onUpdate} />
                    </EditItem>
                  ))}
                </ItemMessage>
              ),
            )
            .otherwise(() => <></>)}
      </EditItem>

      <EditItem
        title="Authorization"
        description="Set the authorization granted when access is approved"
        onUnset={() => {
          rule.authorization = undefined;
          onUpdate();
        }}
        obj={rule.authorization}
        onSet={() => {
          rule.authorization = AccessP.Policy_Spec_Rule_Authorization.create({
            policies: [],
          });
          onUpdate();
        }}
      >
        {rule.authorization && (
          <div>
            <SelectPolicies
              policies={rule.authorization.policies}
              onUpdate={(v) => {
                rule.authorization!.policies = v ?? [];
                onUpdate();
              }}
            />

            <SelectInlinePolicies
              inlinePolicies={rule.authorization.inlinePolicies}
              onUpdate={(v) => {
                rule.authorization!.inlinePolicies = v;
                onUpdate();
              }}
            />

            <DurationPicker
              value={rule.authorization.maxAccessDuration}
              title="Max Access Duration"
              onChange={(v) => {
                rule.authorization!.maxAccessDuration = v;
                onUpdate();
              }}
            />
          </div>
        )}
      </EditItem>
    </div>
  );
};

const Edit = (props: {
  item: AccessP.Policy;
  onUpdate: (item: AccessP.Policy) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Policy.clone(item));
  const updateReq = () => {
    setReq(AccessP.Policy.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <Group grow>
        <Switch
          label="Disabled"
          description="Disable the Policy so it stops being evaluated"
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>

      <ItemMessage
        title="Rules"
        obj={req.spec!.rules}
        isList
        onSet={() => {
          req.spec!.rules = [
            AccessP.Policy_Spec_Rule.create({
              condition: newCondition(),
            }),
          ];
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.rules.push(
            AccessP.Policy_Spec_Rule.create({
              condition: newCondition(),
            }),
          );
          updateReq();
        }}
      >
        {req.spec!.rules.map((rule, idx) => (
          <EditItem
            key={idx}
            obj={rule}
            onUnset={() => {
              req.spec!.rules.splice(idx, 1);
              updateReq();
            }}
          >
            <RuleEdit rule={rule} onUpdate={updateReq} />
          </EditItem>
        ))}
      </ItemMessage>
    </div>
  );
};

export default Edit;
