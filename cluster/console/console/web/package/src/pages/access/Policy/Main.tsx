import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";

const effectMeta = (effect: AccessC.Policy_Spec_Rule_Effect) => {
  switch (effect) {
    case AccessC.Policy_Spec_Rule_Effect.DENY:
      return { label: "Deny", className: "border-red-200 bg-red-50 text-red-700" };
    case AccessC.Policy_Spec_Rule_Effect.REVIEW:
      return { label: "Review", className: "border-blue-200 bg-blue-50 text-blue-700" };
    case AccessC.Policy_Spec_Rule_Effect.AUTO_APPROVE:
      return {
        label: "Auto approve",
        className: "border-emerald-200 bg-emerald-50 text-emerald-700",
      };
    default:
      return { label: "Unset", className: "border-slate-200 bg-slate-50 text-slate-500" };
  }
};

const conditionLabel = (condition?: AccessC.Policy_Spec_Rule_Condition) => {
  if (!condition) return "No condition";
  if (condition.type.oneofKind === "subject") {
    return condition.type.subject.type.oneofKind === "groupRef"
      ? "Subject group"
      : "Subject user";
  }
  if (condition.type.oneofKind === "resource") {
    return condition.type.resource.type.oneofKind === "catalogRef"
      ? "Catalog"
      : "Service";
  }
  if (condition.type.oneofKind === "matchAny") return "Any request";
  if (condition.type.oneofKind === "userRef") return "Requester";
  if (condition.type.oneofKind === "all") return "All conditions";
  if (condition.type.oneofKind === "any") return "Any condition";
  if (condition.type.oneofKind === "match") return "CEL expression";
  return "No condition";
};

const RuleOverview = (props: { rules: AccessC.Policy_Spec_Rule[] }) => (
  <div className="flex w-full flex-col gap-2">
    {props.rules.map((rule, index) => {
      const effect = effectMeta(rule.effect);
      const reviewSteps =
        rule.action?.type.oneofKind === "review"
          ? rule.action.type.review.steps.length
          : 0;
      return (
        <div
          key={`${rule.name || "rule"}-${index}`}
          className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="truncate text-[0.75rem] font-bold text-slate-700">
                {rule.name || `Rule ${index + 1}`}
              </span>
              <span
                className={twMerge(
                  "rounded-md border px-1.5 py-0.5 text-[0.61rem] font-bold",
                  effect.className,
                )}
              >
                {effect.label}
              </span>
            </div>
            <p className="mt-1 text-[0.66rem] font-semibold text-slate-400">
              {conditionLabel(rule.condition)}
              {reviewSteps > 0
                ? ` · ${reviewSteps} review ${reviewSteps === 1 ? "step" : "steps"}`
                : ""}
            </p>
          </div>
          <span className="shrink-0 rounded-md border border-slate-200 bg-white px-2 py-1 text-[0.64rem] font-bold text-slate-500">
            Priority {rule.priority > 0 ? `+${rule.priority}` : rule.priority}
          </span>
        </div>
      );
    })}
  </div>
);

export const ItemInfo = (props: { item: AccessC.Policy }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <>
      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={!item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = !v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>

      <InfoItem title="Rules">
        <span>{item.spec!.rules.length}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Policy }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: { item: AccessC.Policy }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "Active",
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  item.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },
      {
        label: "Rules",
        value: (
          <span className="text-[0.75rem] font-semibold text-slate-700">
            {item.spec!.rules.length}
          </span>
        ),
      },
      ...(item.spec!.rules.length > 0
        ? [
            {
              label: "Rule evaluation",
              value: <RuleOverview rules={item.spec!.rules} />,
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
