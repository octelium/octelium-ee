import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import * as React from "react";

import { CELEditor } from "@/components/Condition/Editor";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import PriorityPicker from "@/components/PriorityPicker";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { strToNum } from "@/utils/convert";
import { getResourceRef } from "@/utils/pb";
import {
  Group,
  Input,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  TextInput,
} from "@mantine/core";
import {
  Braces,
  Check,
  Combine,
  Hash,
  Library,
  ListChecks,
  PanelTop,
  User,
  UserRoundSearch,
  Users,
  WandSparkles,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { match } from "ts-pattern";
import { twMerge } from "tailwind-merge";

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

const newReviewStep = () =>
  AccessP.Policy_Spec_Rule_Action_Review_Step.create({
    approvalRequirement:
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
    onTimeout:
      AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT,
  });

type ConditionChoice =
  | "matchAny"
  | "subjectUser"
  | "subjectGroup"
  | "service"
  | "catalog"
  | "all"
  | "any"
  | "requester"
  | "match";

const conditionChoice = (
  condition: AccessP.Policy_Spec_Rule_Condition,
): ConditionChoice => {
  if (condition.type.oneofKind === "subject") {
    return condition.type.subject.type.oneofKind === "groupRef"
      ? "subjectGroup"
      : "subjectUser";
  }
  if (condition.type.oneofKind === "resource") {
    return condition.type.resource.type.oneofKind === "catalogRef"
      ? "catalog"
      : "service";
  }
  if (condition.type.oneofKind === "userRef") return "requester";
  return condition.type.oneofKind || "subjectUser";
};

const CONDITION_CHOICES: Array<{
  value: ConditionChoice;
  label: string;
  description: string;
  icon: LucideIcon;
}> = [
  {
    value: "matchAny",
    label: "Any request",
    description: "Match without restrictions",
    icon: WandSparkles,
  },
  {
    value: "subjectUser",
    label: "Subject user",
    description: "Access is for this user",
    icon: User,
  },
  {
    value: "subjectGroup",
    label: "Subject group",
    description: "Subject belongs to this group",
    icon: Users,
  },
  {
    value: "service",
    label: "Service",
    description: "Request targets this service",
    icon: PanelTop,
  },
  {
    value: "catalog",
    label: "Catalog",
    description: "Request targets this catalog",
    icon: Library,
  },
  {
    value: "requester",
    label: "Requester",
    description: "Request was made by this user",
    icon: UserRoundSearch,
  },
  {
    value: "all",
    label: "All conditions",
    description: "Every nested condition matches",
    icon: ListChecks,
  },
  {
    value: "any",
    label: "Any condition",
    description: "One nested condition matches",
    icon: Combine,
  },
  {
    value: "match",
    label: "CEL expression",
    description: "Use an advanced expression",
    icon: Braces,
  },
];

const setConditionChoice = (
  condition: AccessP.Policy_Spec_Rule_Condition,
  choice: ConditionChoice,
) => {
  condition.type = match(choice)
    .with("matchAny", () => ({
      oneofKind: "matchAny" as const,
      matchAny: true,
    }))
    .with("subjectUser", () => ({
      oneofKind: "subject" as const,
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: {
          oneofKind: "userRef",
          userRef: MetaP.ObjectReference.create(),
        },
      }),
    }))
    .with("subjectGroup", () => ({
      oneofKind: "subject" as const,
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: {
          oneofKind: "groupRef",
          groupRef: MetaP.ObjectReference.create(),
        },
      }),
    }))
    .with("service", () => ({
      oneofKind: "resource" as const,
      resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
        type: {
          oneofKind: "serviceRef",
          serviceRef: MetaP.ObjectReference.create(),
        },
      }),
    }))
    .with("catalog", () => ({
      oneofKind: "resource" as const,
      resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
        type: {
          oneofKind: "catalogRef",
          catalogRef: MetaP.ObjectReference.create(),
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
    .with("requester", () => ({
      oneofKind: "userRef" as const,
      userRef: MetaP.ObjectReference.create(),
    }))
    .otherwise(() => ({
      oneofKind: "match" as const,
      match: "",
    }));
};

const ConditionTypePicker = (props: {
  condition: AccessP.Policy_Spec_Rule_Condition;
  onChange: (choice: ConditionChoice) => void;
}) => {
  const selected = conditionChoice(props.condition);

  return (
    <div>
      <div className="mb-2">
        <p className="text-[0.72rem] font-bold text-slate-700">
          What should this rule match?
        </p>
        <p className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
          Choose a condition and configure its value directly below.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {CONDITION_CHOICES.map(({ value, label, description, icon: Icon }) => {
          const active = selected === value;
          return (
            <button
              key={value}
              type="button"
              aria-pressed={active}
              onClick={() => {
                if (!active) props.onChange(value);
              }}
              className={twMerge(
                "flex min-w-0 items-start gap-2.5 rounded-xl border px-3 py-2.5 text-left outline-none transition-[border-color,background-color,box-shadow] duration-500 focus-visible:ring-2 focus-visible:ring-blue-500/30",
                active
                  ? "border-slate-700 bg-slate-800 text-white shadow-sm"
                  : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50/70",
              )}
            >
              <Icon
                size={15}
                strokeWidth={2.2}
                className={twMerge(
                  "mt-0.5 shrink-0",
                  active ? "text-slate-200" : "text-slate-400",
                )}
              />
              <span className="min-w-0">
                <span className="block text-[0.72rem] font-bold">{label}</span>
                <span
                  className={twMerge(
                    "mt-0.5 block text-[0.63rem] font-semibold leading-4",
                    active ? "text-slate-300" : "text-slate-400",
                  )}
                >
                  {description}
                </span>
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
};

const ConditionEdit = (props: {
  condition: AccessP.Policy_Spec_Rule_Condition;
  onUpdate: () => void;
}) => {
  const { condition, onUpdate } = props;

  return (
    <div className="w-full">
      <ConditionTypePicker
        condition={condition}
        onChange={(choice) => {
          setConditionChoice(condition, choice);
          onUpdate();
        }}
      />

      <div className="mt-3">
        {match(condition.type)
          .when(
            (x) => x.oneofKind === "matchAny",
            () => (
              <div className="rounded-xl border border-blue-100 bg-blue-50/60 px-3 py-2.5 text-[0.69rem] font-semibold text-blue-700">
                This rule applies to every access request that reaches it.
              </div>
            ),
          )
          .when(
            (x) => x.oneofKind === "subject",
            (subject) => (
              <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
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
              </div>
            ),
          )
          .when(
            (x) => x.oneofKind === "resource",
            (resource) => (
              <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
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
              </div>
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

const ChoiceButtonGrid = <T extends string,>(props: {
  label: string;
  description: string;
  value?: T;
  columns: 2 | 3;
  options: Array<{
    value: T;
    label: string;
    description: string;
    icon: LucideIcon;
  }>;
  onChange: (value: T) => void;
}) => (
  <div>
    <div className="mb-2">
      <p className="text-[0.72rem] font-bold text-slate-700">{props.label}</p>
      <p className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
        {props.description}
      </p>
    </div>
    <div
      role="radiogroup"
      aria-label={props.label}
      className={twMerge(
        "grid grid-cols-1 gap-2",
        props.columns === 2 ? "sm:grid-cols-2" : "sm:grid-cols-3",
      )}
    >
      {props.options.map(({ value, label, description, icon: Icon }) => {
        const active = props.value === value;
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => {
              if (!active) props.onChange(value);
            }}
            className={twMerge(
              "flex min-w-0 items-start gap-2.5 rounded-xl border px-3 py-2.5 text-left outline-none transition-[border-color,background-color,box-shadow] duration-500 focus-visible:ring-2 focus-visible:ring-blue-500/30",
              active
                ? "border-slate-700 bg-slate-800 text-white shadow-sm"
                : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50/70",
            )}
          >
            <Icon
              size={15}
              strokeWidth={2.2}
              className={twMerge(
                "mt-0.5 shrink-0",
                active ? "text-slate-200" : "text-slate-400",
              )}
            />
            <span className="min-w-0">
              <span className="block text-[0.72rem] font-bold">{label}</span>
              <span
                className={twMerge(
                  "mt-0.5 block text-[0.63rem] font-semibold leading-4",
                  active ? "text-slate-300" : "text-slate-400",
                )}
              >
                {description}
              </span>
            </span>
          </button>
        );
      })}
    </div>
  </div>
);

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
            <ChoiceButtonGrid
              label="Reviewer type"
              description="Choose whether a User or any member of a Group can review"
              columns={2}
              value={reviewer.type.oneofKind || undefined}
              options={[
                {
                  value: "user",
                  label: "User",
                  description: "Assign one specific reviewer",
                  icon: User,
                },
                {
                  value: "group",
                  label: "Group",
                  description: "Allow a reviewer group",
                  icon: Users,
                },
              ]}
              onChange={(value) => {
                reviewer.type =
                  value === "group"
                    ? {
                        oneofKind: "group",
                        group: { groupRef: MetaP.ObjectReference.create() },
                      }
                    : {
                        oneofKind: "user",
                        user: { userRef: MetaP.ObjectReference.create() },
                      };
                onUpdate();
              }}
            />
            <div className="mt-3 rounded-xl border border-slate-200 bg-slate-50/50 p-3">
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
            </div>
          </EditItem>
        ))}
      </ItemMessage>

      <div className="mt-4">
        <ChoiceButtonGrid
          label="Approval requirement"
          description="Choose how many assigned reviewers must approve this step"
          columns={3}
          value={
            AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement[
              step.approvalRequirement
            ]
          }
          options={[
            {
              value: "ANY",
              label: "Any reviewer",
              description: "One approval completes the step",
              icon: Check,
            },
            {
              value: "ALL",
              label: "All reviewers",
              description: "Every reviewer must approve",
              icon: Users,
            },
            {
              value: "COUNT",
              label: "Required count",
              description: "Use a specific approval threshold",
              icon: Hash,
            },
          ]}
          onChange={(value) => {
            step.approvalRequirement =
              value === "ALL"
                ? AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
                    .ALL
                : value === "COUNT"
                  ? AccessP
                      .Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
                      .COUNT
                  : AccessP
                      .Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
                      .ANY;
            if (value !== "COUNT") step.approvalCount = 0;
            onUpdate();
          }}
        />

        {step.approvalRequirement ===
          AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement
            .COUNT && (
          <div className="mt-3 max-w-sm rounded-xl border border-slate-200 bg-slate-50/50 p-3">
            <NumberInput
              label="Required approvals"
              description="Set the number of approvals needed to complete this step"
              min={1}
              max={Math.max(1, step.reviewers.length)}
              value={step.approvalCount}
              onChange={(v) => {
                step.approvalCount = strToNum(v);
                onUpdate();
              }}
            />
          </div>
        )}

        <div className="mt-4 grid grid-cols-1 gap-3 lg:grid-cols-2">
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

          <DurationPicker
            value={step.timeout}
            title="Timeout"
            onChange={(v) => {
              step.timeout = v;
              onUpdate();
            }}
          />
        </div>
      </div>

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
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(320px,1.25fr)]">
        <TextInput
          label="Name"
          description="Give this rule a recognizable name"
          placeholder="my-rule"
          value={rule.name}
          onChange={(v) => {
            rule.name = v.target.value;
            onUpdate();
          }}
        />
        <Input.Wrapper
          label="Effect"
          required
          description="Choose what happens when this rule matches"
        >
          <SegmentedControl
            fullWidth
            value={AccessP.Policy_Spec_Rule_Effect[rule.effect]}
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
                label: "Auto approve",
                value:
                  AccessP.Policy_Spec_Rule_Effect[
                    AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE
                  ],
              },
            ]}
            onChange={(value) => {
              const effect = match(value)
                .with(
                  "REVIEW",
                  () => AccessP.Policy_Spec_Rule_Effect.REVIEW,
                )
                .with(
                  "AUTO_APPROVE",
                  () => AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE,
                )
                .otherwise(() => AccessP.Policy_Spec_Rule_Effect.DENY);
              rule.effect = effect;
              if (effect !== AccessP.Policy_Spec_Rule_Effect.REVIEW) {
                rule.action = undefined;
              }
              if (effect === AccessP.Policy_Spec_Rule_Effect.DENY) {
                rule.authorization = undefined;
              }
              onUpdate();
            }}
          />
        </Input.Wrapper>
      </div>

      <div className="mt-4">
        <PriorityPicker
          label="Priority"
          description="Control where this rule is evaluated relative to other rules"
          value={rule.priority}
          onChange={(value) => {
            rule.priority = value;
            onUpdate();
          }}
        />
      </div>

      <EditItem
        title="Condition"
        description="Choose which requests this rule applies to"
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

      {rule.effect === AccessP.Policy_Spec_Rule_Effect.REVIEW && (
        <EditItem
          title="Review workflow"
          description="Configure who reviews this request and how approval proceeds"
          onUnset={() => {
            rule.action = undefined;
            onUpdate();
          }}
          obj={rule.action}
          onSet={() => {
            rule.action = AccessP.Policy_Spec_Rule_Action.create({
              type: {
                oneofKind: "review",
                review: AccessP.Policy_Spec_Rule_Action_Review.create({
                  steps: [newReviewStep()],
                }),
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
                    title="Review steps"
                    obj={review.review.steps}
                    isList
                    onSet={() => {
                      review.review.steps = [
                        newReviewStep(),
                      ];
                      onUpdate();
                    }}
                    onAddListItem={() => {
                      review.review.steps.push(
                        newReviewStep(),
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
              .otherwise(() => null)}
        </EditItem>
      )}

      {(rule.effect === AccessP.Policy_Spec_Rule_Effect.REVIEW ||
        rule.effect === AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE) && (
        <EditItem
          title="Granted access"
          description="Set the authorization and maximum duration granted after approval"
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
                title="Maximum access duration"
                onChange={(v) => {
                  rule.authorization!.maxAccessDuration = v;
                  onUpdate();
                }}
              />
            </div>
          )}
        </EditItem>
      )}
    </div>
  );
};

const Edit = (props: {
  item: AccessP.Policy;
  onUpdate: (item: AccessP.Policy) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Policy.clone(item));
  const ruleKeys = React.useRef(
    item.spec?.rules.map(() => crypto.randomUUID()) ?? [],
  );
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(AccessP.Policy.clone(item));
    ruleKeys.current =
      item.spec?.rules.map(() => crypto.randomUUID()) ?? [];
  }, [itemKey]);

  const updateReq = () => {
    const next = AccessP.Policy.clone(req);
    setReq(next);
    onUpdate(AccessP.Policy.clone(next));
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
          ruleKeys.current = [crypto.randomUUID()];
          updateReq();
        }}
        onAddListItem={() => {
          req.spec!.rules.push(
            AccessP.Policy_Spec_Rule.create({
              condition: newCondition(),
            }),
          );
          ruleKeys.current.push(crypto.randomUUID());
          updateReq();
        }}
      >
        {req.spec!.rules.map((rule, idx) => (
          <EditItem
            key={ruleKeys.current[idx]}
            obj={rule}
            onUnset={() => {
              req.spec!.rules.splice(idx, 1);
              ruleKeys.current.splice(idx, 1);
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
