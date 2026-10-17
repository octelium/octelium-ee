import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { CELEditor } from "@/components/Condition/Editor";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { twMerge } from "@/lib/utils";
import { useListKeys } from "@/utils/forms";
import { getResourceRef } from "@/utils/pb";
import { ActionIcon, Button, Input, Tooltip } from "@mantine/core";
import {
  Braces,
  Combine,
  Layers,
  ListChecks,
  LucideIcon,
  PanelTop,
  Plus,
  Trash2,
  User,
  UserRoundSearch,
  Users,
  WandSparkles,
} from "lucide-react";
import { match } from "ts-pattern";

export const newCondition = (): AccessP.Policy_Spec_Rule_Condition =>
  AccessP.Policy_Spec_Rule_Condition.create({
    type: { oneofKind: "matchAny", matchAny: true },
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
): ConditionChoice | undefined => {
  switch (condition.type.oneofKind) {
    case "subject":
      return condition.type.subject.type.oneofKind === "groupRef"
        ? "subjectGroup"
        : "subjectUser";
    case "resource":
      return condition.type.resource.type.oneofKind === "catalogRef"
        ? "catalog"
        : "service";
    case "userRef":
      return "requester";
    case undefined:
      return undefined;
    default:
      return condition.type.oneofKind;
  }
};

const CHOICE_GROUPS: Array<{
  label: string;
  items: Array<{ value: ConditionChoice; label: string; hint: string; icon: LucideIcon }>;
}> = [
  {
    label: "Who gets the access",
    items: [
      { value: "subjectGroup", label: "Member of a Group", hint: "The subject belongs to a Group", icon: Users },
      { value: "subjectUser", label: "A specific User", hint: "The subject is this User", icon: User },
      { value: "requester", label: "Requested by", hint: "Who created the request, even on behalf of someone else", icon: UserRoundSearch },
    ],
  },
  {
    label: "What is requested",
    items: [
      { value: "catalog", label: "A Catalog", hint: "The request targets this Catalog", icon: Layers },
      { value: "service", label: "A Service", hint: "The request targets this Service", icon: PanelTop },
    ],
  },
  {
    label: "Combine or go further",
    items: [
      { value: "all", label: "All of", hint: "Every nested condition matches (AND)", icon: ListChecks },
      { value: "any", label: "Any of", hint: "At least one nested condition matches (OR)", icon: Combine },
      { value: "match", label: "CEL expression", hint: "Any logic over ctx.request and ctx.user", icon: Braces },
      { value: "matchAny", label: "Every request", hint: "Matches unconditionally", icon: WandSparkles },
    ],
  },
];

const setConditionChoice = (
  condition: AccessP.Policy_Spec_Rule_Condition,
  choice: ConditionChoice,
) => {
  condition.type = match(choice)
    .with("matchAny", () => ({ oneofKind: "matchAny" as const, matchAny: true }))
    .with("subjectUser", () => ({
      oneofKind: "subject" as const,
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: { oneofKind: "userRef", userRef: MetaP.ObjectReference.create() },
      }),
    }))
    .with("subjectGroup", () => ({
      oneofKind: "subject" as const,
      subject: AccessP.Policy_Spec_Rule_Condition_Subject.create({
        type: { oneofKind: "groupRef", groupRef: MetaP.ObjectReference.create() },
      }),
    }))
    .with("service", () => ({
      oneofKind: "resource" as const,
      resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
        type: { oneofKind: "serviceRef", serviceRef: MetaP.ObjectReference.create() },
      }),
    }))
    .with("catalog", () => ({
      oneofKind: "resource" as const,
      resource: AccessP.Policy_Spec_Rule_Condition_Resource.create({
        type: { oneofKind: "catalogRef", catalogRef: MetaP.ObjectReference.create() },
      }),
    }))
    .with("all", () => ({
      oneofKind: "all" as const,
      all: AccessP.Policy_Spec_Rule_Condition_All.create({ of: [newCondition()] }),
    }))
    .with("any", () => ({
      oneofKind: "any" as const,
      any: AccessP.Policy_Spec_Rule_Condition_Any.create({ of: [newCondition()] }),
    }))
    .with("requester", () => ({
      oneofKind: "userRef" as const,
      userRef: MetaP.ObjectReference.create(),
    }))
    .otherwise(() => ({ oneofKind: "match" as const, match: "" }));
};

const ChoicePicker = (props: {
  value?: ConditionChoice;
  onChange: (value: ConditionChoice) => void;
  compact?: boolean;
}) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-2",
      !props.compact && "@2xl:grid-cols-3",
    )}
  >
    {CHOICE_GROUPS.map((group) => (
      <div key={group.label} className="flex min-w-0 flex-col gap-1">
        <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
          {group.label}
        </span>
        <div className="flex flex-wrap gap-1">
          {group.items.map((item) => {
            const active = props.value === item.value;
            const Icon = item.icon;
            return (
              <Tooltip key={item.value} label={item.hint} withArrow openDelay={300}>
                <button
                  type="button"
                  aria-pressed={active}
                  onClick={() => !active && props.onChange(item.value)}
                  className={twMerge(
                    "inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                    active
                      ? "border-slate-800 bg-slate-800 text-white"
                      : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:text-slate-900",
                  )}
                >
                  <Icon size={12} strokeWidth={2.3} />
                  {item.label}
                </button>
              </Tooltip>
            );
          })}
        </div>
      </div>
    ))}
  </div>
);

const RefSelect = (props: {
  api: "core" | "access";
  kind: string;
  label: string;
  description: string;
  value?: MetaP.ObjectReference;
  onChange: (ref: MetaP.ObjectReference) => void;
}) => (
  <SelectResource
    api={props.api}
    kind={props.kind}
    label={props.label}
    description={props.description}
    defaultValue={props.value?.name}
    onChange={(resource) =>
      props.onChange(resource ? getResourceRef(resource) : MetaP.ObjectReference.create())
    }
  />
);

export const ConditionEdit = (props: {
  condition: AccessP.Policy_Spec_Rule_Condition;
  onUpdate: () => void;
  depth?: number;
}) => {
  const rows = useListKeys();
  const { condition, onUpdate } = props;
  const depth = props.depth ?? 0;
  const type = condition.type;

  const value = match(type)
    .when(
      (x) => x.oneofKind === "subject",
      (x) =>
        x.oneofKind === "subject" && x.subject.type.oneofKind === "groupRef" ? (
          <RefSelect
            api="core"
            kind="Group"
            label="Group"
            description="Members of this Group match, whoever created the request."
            value={x.subject.type.groupRef}
            onChange={(ref) => {
              if (x.subject.type.oneofKind === "groupRef") x.subject.type.groupRef = ref;
              onUpdate();
            }}
          />
        ) : x.oneofKind === "subject" && x.subject.type.oneofKind === "userRef" ? (
          <RefSelect
            api="core"
            kind="User"
            label="User"
            description="Requests that grant access to this User match."
            value={x.subject.type.userRef}
            onChange={(ref) => {
              if (x.subject.type.oneofKind === "userRef") x.subject.type.userRef = ref;
              onUpdate();
            }}
          />
        ) : null,
    )
    .when(
      (x) => x.oneofKind === "resource",
      (x) =>
        x.oneofKind === "resource" && x.resource.type.oneofKind === "catalogRef" ? (
          <RefSelect
            api="access"
            kind="Catalog"
            label="Catalog"
            description="Requests for this Catalog match."
            value={x.resource.type.catalogRef}
            onChange={(ref) => {
              if (x.resource.type.oneofKind === "catalogRef") x.resource.type.catalogRef = ref;
              onUpdate();
            }}
          />
        ) : x.oneofKind === "resource" && x.resource.type.oneofKind === "serviceRef" ? (
          <RefSelect
            api="core"
            kind="Service"
            label="Service"
            description="Requests for this Service match. Catalogs that contain it do not."
            value={x.resource.type.serviceRef}
            onChange={(ref) => {
              if (x.resource.type.oneofKind === "serviceRef") x.resource.type.serviceRef = ref;
              onUpdate();
            }}
          />
        ) : null,
    )
    .when(
      (x) => x.oneofKind === "userRef",
      (x) =>
        x.oneofKind === "userRef" ? (
          <RefSelect
            api="core"
            kind="User"
            label="Requester"
            description="Requests created by this User match, including the ones on behalf of someone else."
            value={x.userRef}
            onChange={(ref) => {
              if (condition.type.oneofKind === "userRef") condition.type.userRef = ref;
              onUpdate();
            }}
          />
        ) : null,
    )
    .when(
      (x) => x.oneofKind === "match",
      (x) =>
        x.oneofKind === "match" ? (
          <Input.Wrapper
            label="CEL expression"
            description="Must evaluate to true. ctx.request is the access Request and ctx.user the User who created it."
          >
            <CELEditor
              context="access"
              exp={x.match}
              onChange={(next: string) => {
                if (condition.type.oneofKind === "match") condition.type.match = next ?? "";
                onUpdate();
              }}
            />
          </Input.Wrapper>
        ) : null,
    )
    .when(
      (x) => x.oneofKind === "all" || x.oneofKind === "any",
      (x) => {
        const list =
          x.oneofKind === "all" ? x.all.of : x.oneofKind === "any" ? x.any.of : [];
        const joiner = x.oneofKind === "all" ? "AND" : "OR";
        return (
          <div className="flex flex-col gap-2">
            {list.map((child, index) => (
              <div key={rows.keyAt("children", index)} className="flex flex-col gap-2">
                {index > 0 && (
                  <span className="w-fit rounded-md bg-slate-100 px-1.5 py-0.5 text-micro font-bold text-slate-500">
                    {joiner}
                  </span>
                )}
                <div className="flex items-start gap-2 rounded-lg border border-slate-200 bg-slate-50/50 p-3">
                  <div className="min-w-0 flex-1">
                    <ConditionEdit condition={child} onUpdate={onUpdate} depth={depth + 1} />
                  </div>
                  <Tooltip label="Remove condition" withArrow>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      size="sm"
                      aria-label="Remove condition"
                      onClick={() => {
                        rows.removeAt("children", index);
                        list.splice(index, 1);
                        onUpdate();
                      }}
                    >
                      <Trash2 size={13} />
                    </ActionIcon>
                  </Tooltip>
                </div>
              </div>
            ))}
            <Button
              variant="default"
              size="compact-xs"
              className="w-fit"
              leftSection={<Plus size={12} />}
              onClick={() => {
                list.push(newCondition());
                onUpdate();
              }}
            >
              Add condition
            </Button>
          </div>
        );
      },
    )
    .when(
      (x) => x.oneofKind === "matchAny",
      () => (
        <p className="text-xs font-normal leading-5 text-slate-500">
          Every request that reaches this rule matches. Combine it with a
          lower priority so more specific rules are evaluated first.
        </p>
      ),
    )
    .otherwise(() => null);

  return (
    <div className="flex flex-col gap-3">
      <ChoicePicker
        value={conditionChoice(condition)}
        compact={depth > 0}
        onChange={(choice) => {
          setConditionChoice(condition, choice);
          onUpdate();
        }}
      />
      {value && <div className="max-w-2xl">{value}</div>}
    </div>
  );
};
