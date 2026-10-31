import * as AccessC from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { CommonSummaryOptions } from "@/apis/visibilityv1/meta/vmetav1";
import InfoItem from "@/components/InfoItem";
import { twMerge } from "@/lib/utils";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowUpRight,
  BellRing,
  Filter,
  KeyRound,
  ListOrdered,
  Scale,
  Zap,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { EFFECT_META, EffectBadge, EffectKey, getEffectKey, normalizeRef, RefChip } from "../common";
import { ConditionView, GrantSummary, StepTimeline, SurfaceChip } from "../rules";

const WINDOW_DAYS = 30;

const usePolicyActivity = (item: AccessC.Policy) =>
  useQuery({
    queryKey: ["access", "policy", item.metadata?.uid, "activity", WINDOW_DAYS],
    enabled: !!item.metadata?.uid,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) => {
      const to = new Date();
      const from = new Date(to.getTime() - WINDOW_DAYS * 24 * 3600 * 1000);
      return (
        await getClientVisibilityAccess().getRequestSummary(
          {
            common: CommonSummaryOptions.create({
              from: Timestamp.fromDate(from),
              to: Timestamp.fromDate(to),
            }),
            policyRef: getResourceRef(item),
          },
          { abort: signal },
        )
      ).response;
    },
  });

const Stat = (props: {
  label: string;
  value?: number | string;
  to?: string;
  tone?: "success" | "danger" | "warning";
  loading?: boolean;
}) => {
  const body = (
    <>
      <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
        {props.label}
      </span>
      <span
        className={twMerge(
          "text-lg font-bold tabular-nums text-slate-900",
          props.tone === "success" && "text-emerald-700",
          props.tone === "danger" && "text-red-700",
          props.tone === "warning" && "text-amber-700",
        )}
      >
        {props.loading ? (
          <span className="block h-6 w-10 motion-safe:animate-pulse rounded bg-slate-100" />
        ) : (
          (props.value ?? "—")
        )}
      </span>
    </>
  );
  const className =
    "flex min-w-0 flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2";
  return props.to ? (
    <Link
      to={props.to}
      className={twMerge(
        className,
        "outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {body}
    </Link>
  ) : (
    <div className={className}>{body}</div>
  );
};

const Activity = (props: { item: AccessC.Policy }) => {
  const query = usePolicyActivity(props.item);
  const data = query.data;
  const name = props.item.metadata?.name ?? "";
  const base = `/access/requests?policyRef.name=${encodeURIComponent(name)}`;
  const median = data?.decisionTime?.p50Seconds;

  return (
    <div className="flex flex-col gap-2 border-b border-slate-200 bg-white px-4 py-4">
      <span className="flex items-center justify-between gap-2">
        <span className="text-xs font-semibold text-slate-700">
          Requests decided by this Policy · last {WINDOW_DAYS} days
        </span>
        <Link
          to={base}
          className="inline-flex items-center gap-0.5 text-micro font-semibold text-blue-700 hover:text-blue-800"
        >
          All its requests <ArrowUpRight size={11} strokeWidth={2.5} />
        </Link>
      </span>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
        <Stat label="Requests" value={data ? Number(data.totalNumber) : undefined} loading={query.isLoading} to={base} />
        <Stat label="Auto-approved" value={data ? Number(data.totalAutoApproved) : undefined} loading={query.isLoading} tone="success" />
        <Stat label="Reviewed" value={data ? Number(data.totalReviewed) : undefined} loading={query.isLoading} />
        <Stat label="Denied by a rule" value={data ? Number(data.totalDenied) : undefined} loading={query.isLoading} tone={data && data.totalDenied > 0 ? "danger" : undefined} />
        <Stat
          label="Pending now"
          value={data ? Number(data.totalPending) : undefined}
          loading={query.isLoading}
          tone={data && data.totalPending > 0 ? "warning" : undefined}
          to={`${base}&state=PENDING`}
        />
        <Stat
          label="Median decision"
          value={median ? formatMedian(Number(median)) : undefined}
          loading={query.isLoading}
        />
      </div>
    </div>
  );
};

const formatMedian = (seconds: number) => {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  if (seconds < 86400) return `${(seconds / 3600).toFixed(1).replace(/\.0$/, "")}h`;
  return `${(seconds / 86400).toFixed(1).replace(/\.0$/, "")}d`;
};

const RuleBlock = (props: {
  icon: typeof Filter;
  label: string;
  children: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-[110px_minmax(0,1fr)] sm:gap-3">
      <span className="flex items-center gap-1.5 self-start pt-0.5 text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
        <Icon size={11} strokeWidth={2.4} />
        {props.label}
      </span>
      <div className="min-w-0 text-xs text-slate-700">{props.children}</div>
    </div>
  );
};

const RuleCard = (props: {
  rule: AccessC.Policy_Spec_Rule;
  index: number;
}) => {
  const { rule } = props;
  const key = getEffectKey(rule.effect);
  const steps =
    rule.action?.type.oneofKind === "review" ? rule.action.type.review.steps : [];

  return (
    <li className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="flex h-6 w-6 items-center justify-center rounded-md bg-slate-100 text-micro font-bold text-slate-600">
          {props.index + 1}
        </span>
        <span className="text-body font-semibold text-slate-900">
          {rule.name || `Rule ${props.index + 1}`}
        </span>
        <EffectBadge effect={rule.effect} />
        <span className="ml-auto rounded-md border border-slate-200 px-1.5 py-px text-micro font-semibold tabular-nums text-slate-500">
          priority {rule.priority > 0 ? `+${rule.priority}` : rule.priority}
        </span>
      </div>

      <RuleBlock icon={Filter} label="When">
        <ConditionView condition={rule.condition} />
      </RuleBlock>

      <RuleBlock icon={Zap} label="Then">
        {key ? (
          <span className="text-slate-600">{EFFECT_META[key].description}</span>
        ) : (
          <span className="text-amber-700">No effect is set: the rule cannot be saved like this.</span>
        )}
      </RuleBlock>

      {key === "REVIEW" && steps.length > 0 && (
        <RuleBlock icon={ListOrdered} label="Review">
          <StepTimeline steps={steps} showSurfaces />
        </RuleBlock>
      )}

      {(key === "REVIEW" || key === "AUTO_APPROVE") && (
        <RuleBlock icon={KeyRound} label="Grants">
          <GrantSummary authorization={rule.authorization} />
        </RuleBlock>
      )}

      {rule.notifications.length > 0 && (
        <RuleBlock icon={BellRing} label="Announces">
          <span className="flex flex-wrap gap-1.5">
            {rule.notifications.map((surface, index) => (
              <SurfaceChip key={index} surface={surface} isNotification />
            ))}
          </span>
        </RuleBlock>
      )}
    </li>
  );
};

const Integrations = (props: { rules: AccessC.Policy_Spec_Rule[] }) => {
  const refs = new Map<string, ReturnType<typeof normalizeRef>>();
  props.rules.forEach((rule) => {
    const surfaces = [
      ...rule.notifications,
      ...(rule.action?.type.oneofKind === "review"
        ? rule.action.type.review.steps.flatMap((step) => step.surfaces)
        : []),
    ];
    surfaces.forEach((surface) => {
      const ref = normalizeRef(surface.destination?.integrationRef, "access/v1", "Integration");
      if (ref) refs.set(ref.name || ref.uid, ref);
    });
  });

  if (refs.size === 0) return <span className="text-slate-500">Access portal only</span>;

  return (
    <span className="flex flex-wrap gap-1.5">
      {[...refs.values()].map((ref) => ref && <RefChip key={ref.name || ref.uid} itemRef={ref} />)}
    </span>
  );
};

const effectMix = (rules: AccessC.Policy_Spec_Rule[]) => {
  const counts = new Map<EffectKey, number>();
  rules.forEach((rule) => {
    const key = getEffectKey(rule.effect);
    if (key) counts.set(key, (counts.get(key) ?? 0) + 1);
  });
  return (["AUTO_APPROVE", "REVIEW", "DENY"] as EffectKey[])
    .filter((key) => counts.has(key))
    .map((key) => ({ key, count: counts.get(key)! }));
};

export const ItemInfo = (props: { item: AccessC.Policy }) => (
  <>
    <InfoItem title="Rules">
      <span>{props.item.spec?.rules.length ?? 0}</span>
    </InfoItem>
  </>
);

export default (props: { item: AccessC.Policy }) => (
  <div className="w-full">
    <ItemInfo item={props.item} />
  </div>
);

export const MainInfo = (props: { item: AccessC.Policy }): ResourceMainInfo => {
  const { item } = props;
  const rules = item.spec?.rules ?? [];
  const sorted = rules
    .map((rule, index) => ({ rule, index }))
    .sort((a, b) => a.rule.priority - b.rule.priority || a.index - b.index);
  const mix = effectMix(rules);

  return {
    header: <Activity item={item} />,
    groupOrder: [
      { title: "Rules", icon: ListOrdered },
    ],
    items: [
      {
        label: "Rules",
        primary: true,
        icon: ListOrdered,
        value: <span className="font-semibold">{rules.length}</span>,
      },
      {
        label: "Effects",
        primary: true,
        icon: Scale,
        value:
          mix.length === 0 ? (
            "None"
          ) : (
            <span className="flex flex-wrap gap-1">
              {mix.map(({ key, count }) => (
                <span key={key} className="inline-flex items-center gap-1">
                  <EffectBadge effect={EFFECT_META[key].value} />
                  {count > 1 && <span className="text-micro text-slate-500">×{count}</span>}
                </span>
              ))}
            </span>
          ),
      },
      {
        label: "Presented on",
        primary: true,
        icon: BellRing,
        value: <Integrations rules={rules} />,
      },
      {
        label: "",
        group: "Rules",
        layout: "stacked" as const,
        value:
          rules.length === 0 ? (
            <span className="text-xs text-slate-500">
              This Policy has no rule yet, so it decides nothing.
            </span>
          ) : (
            <div className="flex flex-col gap-2">
              <span className="text-micro leading-4 text-slate-500">
                In evaluation order. Rules of other Policies are interleaved by
                priority; the first rule that matches decides.
              </span>
              <ol className="flex flex-col gap-2">
                {sorted.map(({ rule, index }, position) => (
                  <RuleCard key={`${rule.name}-${index}`} rule={rule} index={position} />
                ))}
              </ol>
            </div>
          ),
      },
    ],
  };
};
