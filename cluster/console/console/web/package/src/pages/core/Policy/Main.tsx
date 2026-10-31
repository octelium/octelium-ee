import * as CoreC from "@/apis/corev1/corev1";
import InfoItem from "@/components/InfoItem";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Switch } from "@mantine/core";
import { twMerge } from "@/lib/utils";
import {
  MutedValue,
  OverviewNotice,
  TonePill,
} from "@/components/ResourceLayout/overview";
import { attrsItems } from "@/components/ResourceLayout/overviewItems";
import {
  Ban,
  ListChecks,
  ShieldAlert,
  ShieldCheck,
  ShieldX,
  Tags,
} from "lucide-react";

export const ItemInfo = (props: { item: CoreC.Policy }) => {
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
    </>
  );
};

export default (props: { item: CoreC.Policy }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

const conditionSummary = (condition?: CoreC.Condition): string => {
  const type = condition?.type;
  switch (type?.oneofKind) {
    case "match":
      return type.match;
    case "matchAny":
      return type.matchAny ? "Matches every request" : "Matches no request";
    case "not":
      return `not (${type.not})`;
    case "all":
      return `All of ${type.all.of.length} conditions`;
    case "any":
      return `Any of ${type.any.of.length} conditions`;
    case "none":
      return `None of ${type.none.of.length} conditions`;
    case "opa":
      return "OPA (Rego) policy";
    default:
      return "No condition";
  }
};

const RuleRow = (props: {
  name?: string;
  effect: React.ReactNode;
  priority?: number;
  condition?: CoreC.Condition;
}) => (
  <li className="flex min-w-0 flex-col gap-1 px-3 py-2">
    <span className="flex flex-wrap items-center gap-2">
      {props.effect}
      {props.name && (
        <span className="text-xs font-semibold text-slate-800">
          {props.name}
        </span>
      )}
      {props.priority !== undefined && props.priority !== 0 && (
        <span className="ml-auto text-micro font-normal text-slate-500">
          Priority {props.priority}
        </span>
      )}
    </span>
    <code className="text-xs leading-5 text-slate-600 [overflow-wrap:anywhere]">
      {conditionSummary(props.condition)}
    </code>
  </li>
);

export const MainInfo = (props: { item: CoreC.Policy }): ResourceMainInfo => {
  const { item } = props;
  const rules = [...(item.spec?.rules ?? [])].sort(
    (a, b) => a.priority - b.priority,
  );
  const enforcementRules = item.spec?.enforcementRules ?? [];
  const allow = rules.filter(
    (rule) => rule.effect === CoreC.Policy_Spec_Rule_Effect.ALLOW,
  ).length;
  const deny = rules.filter(
    (rule) => rule.effect === CoreC.Policy_Spec_Rule_Effect.DENY,
  ).length;

  return {
    header: item.spec?.isDisabled ? (
      <div className="border-b border-slate-200 bg-white px-4 py-3">
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          None of the rules of this Policy is evaluated.
        </OverviewNotice>
      </div>
    ) : undefined,
    groupOrder: [
      { title: "Rules", icon: ListChecks },
      { title: "Enforcement rules", icon: ShieldAlert },
      { title: "Attributes", icon: Tags },
    ],
    items: [
      {
        label: "Rules",
        primary: true,
        icon: ListChecks,
        value: <span className="font-semibold tabular-nums">{rules.length}</span>,
      },
      {
        label: "Allow",
        primary: true,
        icon: ShieldCheck,
        value: (
          <span className="font-semibold tabular-nums text-emerald-700">
            {allow}
          </span>
        ),
      },
      {
        label: "Deny",
        primary: true,
        icon: ShieldX,
        value: (
          <span className="font-semibold tabular-nums text-red-700">{deny}</span>
        ),
      },
      {
        label: "Enforcement rules",
        primary: true,
        icon: ShieldAlert,
        value: (
          <span className="font-semibold tabular-nums">
            {enforcementRules.length}
          </span>
        ),
      },
      {
        label: "",
        group: "Rules",
        layout: "stacked",
        value:
          rules.length > 0 ? (
            <ol className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
              {rules.map((rule, index) => (
                <RuleRow
                  key={`${rule.name}-${index}`}
                  name={rule.name || `Rule ${index + 1}`}
                  priority={rule.priority}
                  condition={rule.condition}
                  effect={
                    rule.effect === CoreC.Policy_Spec_Rule_Effect.DENY ? (
                      <TonePill tone="danger" icon={ShieldX}>
                        Deny
                      </TonePill>
                    ) : (
                      <TonePill tone="success" icon={ShieldCheck}>
                        Allow
                      </TonePill>
                    )
                  }
                />
              ))}
            </ol>
          ) : (
            <MutedValue>The Policy has no rules.</MutedValue>
          ),
      },
      ...(enforcementRules.length > 0
        ? [
            {
              label: "",
              group: "Enforcement rules",
              layout: "stacked" as const,
              value: (
                <ol className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
                  {enforcementRules.map((rule, index) => (
                    <RuleRow
                      key={index}
                      condition={rule.condition}
                      effect={
                        rule.effect ===
                        CoreC.Policy_Spec_EnforcementRule_Effect.IGNORE ? (
                          <TonePill tone="neutral">Ignore</TonePill>
                        ) : (
                          <TonePill tone="info">Enforce</TonePill>
                        )
                      }
                    />
                  ))}
                </ol>
              ),
            },
          ]
        : []),
      ...attrsItems(item.spec?.attrs),
    ],
  };
};
