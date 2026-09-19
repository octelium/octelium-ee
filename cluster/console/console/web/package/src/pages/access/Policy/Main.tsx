import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";
import {
  getAudienceLabel,
  getInteractionModeLabel,
} from "../Integration/utils";

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

const reviewSurfaces = (rule: AccessC.Policy_Spec_Rule) =>
  rule.action?.type.oneofKind === "review"
    ? rule.action.type.review.steps.reduce(
        (total, step) => total + step.surfaces.length,
        0,
      )
    : 0;

const surfaceCount = (rule: AccessC.Policy_Spec_Rule) =>
  reviewSurfaces(rule) + rule.notifications.length;

const integrationRefOf = (surface: AccessC.Policy_Spec_Rule_Surface) => {
  const itemRef = surface.destination?.integrationRef;
  if (!itemRef?.name && !itemRef?.uid) return undefined;
  return ObjectReference.create({
    ...itemRef,
    apiVersion: itemRef.apiVersion || "access/v1",
    kind: itemRef.kind || "Integration",
  });
};

const SurfaceChips = (props: {
  surfaces: AccessC.Policy_Spec_Rule_Surface[];
  isReviewSurface: boolean;
}) => (
  <div className="flex flex-col gap-1.5">
    {props.surfaces.map((surface, index) => {
      const itemRef = integrationRefOf(surface);
      return (
        <div
          key={`surface-${index}`}
          className="flex flex-wrap items-center gap-1.5"
        >
          {itemRef ? (
            <ResourceListLabel itemRef={itemRef} />
          ) : (
            <span className="text-micro font-normal text-slate-500">
              No Integration set
            </span>
          )}
          <Label size="sm" outlined>
            {getAudienceLabel(surface.destination?.audience)}
          </Label>
          {props.isReviewSurface && (
            <Label
              size="sm"
              tone={
                surface.interactionMode ===
                AccessC.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE
                  ? "warning"
                  : "neutral"
              }
            >
              {getInteractionModeLabel(surface.interactionMode)}
            </Label>
          )}
        </div>
      );
    })}
  </div>
);

const SurfaceOverview = (props: { rules: AccessC.Policy_Spec_Rule[] }) => {
  const entries: Array<{
    title: string;
    surfaces: AccessC.Policy_Spec_Rule_Surface[];
    isReviewSurface: boolean;
  }> = [];

  props.rules.forEach((rule, ruleIndex) => {
    const ruleName = rule.name || `Rule ${ruleIndex + 1}`;
    if (rule.action?.type.oneofKind === "review") {
      rule.action.type.review.steps.forEach((step, stepIndex) => {
        if (step.surfaces.length === 0) return;
        entries.push({
          title: `${ruleName} · ${step.name || `Step ${stepIndex + 1}`}`,
          surfaces: step.surfaces,
          isReviewSurface: true,
        });
      });
    }
    if (rule.notifications.length > 0) {
      entries.push({
        title: `${ruleName} · Outcome notifications`,
        surfaces: rule.notifications,
        isReviewSurface: false,
      });
    }
  });

  return (
    <div className="flex w-full flex-col gap-2">
      {entries.map((entry) => (
        <div
          key={entry.title}
          className="rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5"
        >
          <div className="text-body font-semibold text-slate-700">
            {entry.title}
          </div>
          <div className="mt-1.5">
            <SurfaceChips
              surfaces={entry.surfaces}
              isReviewSurface={entry.isReviewSurface}
            />
          </div>
        </div>
      ))}
    </div>
  );
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
              <span className="truncate text-body font-semibold text-slate-700">
                {rule.name || `Rule ${index + 1}`}
              </span>
              <span
                className={twMerge(
                  "rounded-md border px-1.5 py-0.5 text-micro font-semibold",
                  effect.className,
                )}
              >
                {effect.label}
              </span>
            </div>
            <p className="mt-1 text-micro font-normal text-slate-500">
              {conditionLabel(rule.condition)}
              {reviewSteps > 0
                ? ` · ${reviewSteps} review ${reviewSteps === 1 ? "step" : "steps"}`
                : ""}
              {surfaceCount(rule) > 0
                ? ` · ${surfaceCount(rule)} Integration ${surfaceCount(rule) === 1 ? "surface" : "surfaces"}`
                : ""}
            </p>
          </div>
          <span className="shrink-0 rounded-md border border-slate-200 bg-white px-2 py-1 text-micro font-normal text-slate-500">
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

  return {
    items: [
      {
        label: "Rules",
        value: (
          <span className="text-body font-semibold text-slate-700">
            {item.spec!.rules.length}
          </span>
        ),
      },
      {
        label: "Integration surfaces",
        value: (
          <span className="text-body font-semibold text-slate-700">
            {item.spec!.rules.reduce(
              (total, rule) => total + surfaceCount(rule),
              0,
            )}
          </span>
        ),
        hint: "The Cluster's own access portal is always available regardless of the configured surfaces.",
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
      ...(item.spec!.rules.some((rule) => surfaceCount(rule) > 0)
        ? [
            {
              label: "Surface delivery",
              value: <SurfaceOverview rules={item.spec!.rules} />,
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
