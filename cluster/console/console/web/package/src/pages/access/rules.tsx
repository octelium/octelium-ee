import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import {
  Ban,
  BellRing,
  Braces,
  Check,
  Clock3,
  FileKey2,
  Hash,
  Layers,
  Minus,
  Server,
  Shield,
  Timer,
  User,
  UserRoundSearch,
  Users,
  X,
} from "lucide-react";
import * as React from "react";
import {
  formatDurationShort,
  getProviderKindFromStatus,
  normalizeRef,
  ProviderMark,
  RefChip,
  ToneBadge,
} from "./common";
import {
  getApprovalRequirementLabel,
  getOnTimeoutLabel,
} from "./Request/utils";

const Token = (props: { children: React.ReactNode; className?: string }) => (
  <span
    className={twMerge(
      "inline-flex items-center rounded-md bg-slate-100 px-1.5 py-0.5 text-micro font-semibold uppercase tracking-[0.04em] text-slate-500",
      props.className,
    )}
  >
    {props.children}
  </span>
);

const refChip = (
  ref: ObjectReference | undefined,
  apiVersion: string,
  kind: string,
  icon: typeof User,
) => {
  const normalized = normalizeRef(ref, apiVersion, kind);
  return normalized ? (
    <RefChip itemRef={normalized} icon={icon} />
  ) : (
    <span className="text-xs italic text-slate-400">not set</span>
  );
};

export const ConditionView = (props: {
  condition?: AccessP.Policy_Spec_Rule_Condition;
  depth?: number;
}): React.ReactElement => {
  const condition = props.condition;
  const depth = props.depth ?? 0;
  const type = condition?.type;

  if (!type || !type.oneofKind) {
    return <span className="text-xs italic text-slate-400">no condition</span>;
  }

  const line = (label: string, value: React.ReactNode) => (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <span className="text-xs text-slate-600">{label}</span>
      {value}
    </span>
  );

  switch (type.oneofKind) {
    case "matchAny":
      return line("Any request", null);
    case "subject":
      return type.subject.type.oneofKind === "groupRef"
        ? line(
            "Subject is a member of",
            refChip(type.subject.type.groupRef, "core/v1", "Group", Users),
          )
        : line(
            "Subject is",
            refChip(
              type.subject.type.oneofKind === "userRef"
                ? type.subject.type.userRef
                : undefined,
              "core/v1",
              "User",
              User,
            ),
          );
    case "resource":
      return type.resource.type.oneofKind === "catalogRef"
        ? line(
            "Requested Catalog is",
            refChip(type.resource.type.catalogRef, "access/v1", "Catalog", Layers),
          )
        : line(
            "Requested Service is",
            refChip(
              type.resource.type.oneofKind === "serviceRef"
                ? type.resource.type.serviceRef
                : undefined,
              "core/v1",
              "Service",
              Server,
            ),
          );
    case "userRef":
      return line(
        "Requested by",
        refChip(type.userRef, "core/v1", "User", UserRoundSearch),
      );
    case "match":
      return (
        <span className="inline-flex min-w-0 max-w-full items-center gap-1.5">
          <Braces size={12} className="shrink-0 text-slate-500" />
          <code className="min-w-0 break-all rounded bg-slate-100 px-1.5 py-0.5 font-mono text-micro text-slate-700">
            {type.match || "empty expression"}
          </code>
        </span>
      );
    case "all":
    case "any": {
      const children = type.oneofKind === "all" ? type.all.of : type.any.of;
      const joiner = type.oneofKind === "all" ? "and" : "or";
      if (children.length === 0) {
        return <span className="text-xs italic text-slate-400">empty group</span>;
      }
      return (
        <span
          className={twMerge(
            "flex min-w-0 flex-col gap-1.5",
            depth > 0 && "rounded-md border border-dashed border-slate-200 px-2 py-1.5",
          )}
        >
          {children.map((child, index) => (
            <span key={index} className="flex min-w-0 flex-wrap items-center gap-1.5">
              {index > 0 && <Token>{joiner}</Token>}
              <ConditionView condition={child} depth={depth + 1} />
            </span>
          ))}
        </span>
      );
    }
    default:
      return <span className="text-xs italic text-slate-400">unknown condition</span>;
  }
};

export const ReviewerChips = (props: {
  reviewers: AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer[];
}) => (
  <span className="flex flex-wrap items-center gap-1.5">
    {props.reviewers.length === 0 && (
      <span className="text-xs italic text-amber-700">no reviewer</span>
    )}
    {props.reviewers.map((reviewer, index) =>
      reviewer.type.oneofKind === "group" ? (
        <React.Fragment key={index}>
          {refChip(reviewer.type.group.groupRef, "core/v1", "Group", Users)}
        </React.Fragment>
      ) : reviewer.type.oneofKind === "user" ? (
        <React.Fragment key={index}>
          {refChip(reviewer.type.user.userRef, "core/v1", "User", User)}
        </React.Fragment>
      ) : null,
    )}
  </span>
);

export const SurfaceChip = (props: {
  surface: AccessP.Policy_Spec_Rule_Surface;
  isNotification?: boolean;
}) => {
  const ref = normalizeRef(
    props.surface.destination?.integrationRef,
    "access/v1",
    "Integration",
  );
  const audience = props.surface.destination?.audience;
  const audienceLabel =
    audience === AccessP.Policy_Spec_Rule_Surface_Destination_Audience.REVIEWERS
      ? "reviewers' DMs"
      : audience === AccessP.Policy_Spec_Rule_Surface_Destination_Audience.REQUESTER
        ? "requester's DM"
        : audience === AccessP.Policy_Spec_Rule_Surface_Destination_Audience.SUBJECT
          ? "subject's DM"
          : "shared";

  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-white py-0.5 pl-1 pr-2 text-xs text-slate-700">
      <span className="flex h-4 w-4 items-center justify-center">
        {props.isNotification ? (
          <BellRing size={11} className="text-slate-500" />
        ) : (
          <Hash size={11} className="text-slate-500" />
        )}
      </span>
      {ref ? <RefChip itemRef={ref} className="border-0 bg-transparent p-0" /> : "No Integration"}
      <span className="text-micro text-slate-500">· {audienceLabel}</span>
      {!props.isNotification &&
        props.surface.interactionMode ===
          AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE && (
          <span className="text-micro font-semibold text-blue-700">
            · interactive
          </span>
        )}
    </span>
  );
};

export type StepState = "done" | "current" | "stopped" | "upcoming" | "unreached" | "plan";

export type StepDecision = {
  reviewerRef?: ObjectReference;
  decision: AccessP.Review_Spec_Decision;
  setAt?: Timestamp;
  justification?: string;
  viaIntegration?: string;
};

const STEP_DOT: Record<StepState, string> = {
  done: "border-emerald-200 bg-emerald-50 text-emerald-700",
  current: "border-amber-300 bg-amber-50 text-amber-700",
  stopped: "border-red-200 bg-red-50 text-red-700",
  upcoming: "border-slate-200 bg-white text-slate-500",
  unreached: "border-slate-200 bg-slate-50 text-slate-400",
  plan: "border-slate-200 bg-white text-slate-600",
};

const STEP_LABEL: Partial<Record<StepState, { label: string; tone: "success" | "warning" | "danger" | "neutral" }>> = {
  done: { label: "Cleared", tone: "success" },
  current: { label: "Waiting", tone: "warning" },
  stopped: { label: "Stopped here", tone: "danger" },
  upcoming: { label: "Next", tone: "neutral" },
  unreached: { label: "Not reached", tone: "neutral" },
};

export const StepTimeline = (props: {
  steps: AccessP.Policy_Spec_Rule_Action_Review_Step[];
  stateOf?: (index: number) => StepState;
  decisionsOf?: (index: number) => StepDecision[];
  footerOf?: (index: number) => React.ReactNode;
  showSurfaces?: boolean;
}) => (
  <ol className="flex flex-col">
    {props.steps.map((step, index) => {
      const state = props.stateOf?.(index) ?? "plan";
      const label = STEP_LABEL[state];
      const decisions = props.decisionsOf?.(index) ?? [];
      const sod = step.separationOfDuties;
      return (
        <li key={`${step.name}-${index}`} className="relative flex gap-3 pb-4 last:pb-0">
          {index < props.steps.length - 1 && (
            <span
              aria-hidden="true"
              className={twMerge(
                "absolute bottom-0 left-[13px] top-7 w-px",
                state === "done" ? "bg-emerald-200" : "bg-slate-200",
              )}
            />
          )}
          <span
            aria-hidden="true"
            className={twMerge(
              "relative z-10 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-micro font-bold",
              STEP_DOT[state],
            )}
          >
            {state === "done" ? (
              <Check size={13} strokeWidth={2.75} />
            ) : state === "current" ? (
              <Clock3 size={12} strokeWidth={2.5} />
            ) : state === "stopped" ? (
              <X size={12} strokeWidth={2.75} />
            ) : state === "unreached" ? (
              <Minus size={12} strokeWidth={2.5} />
            ) : (
              index + 1
            )}
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-1.5 pt-0.5">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span className="text-body font-semibold text-slate-800">
                {step.name || `Step ${index + 1}`}
              </span>
              {label && <ToneBadge tone={label.tone}>{label.label}</ToneBadge>}
              <span className="text-xs text-slate-500">
                {getApprovalRequirementLabel(step)}
              </span>
            </div>
            <ReviewerChips reviewers={step.reviewers} />
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-micro font-normal text-slate-500">
              <span className="inline-flex items-center gap-1">
                <Timer size={11} strokeWidth={2.4} />
                {step.timeout
                  ? `Times out after ${formatDurationShort(step.timeout)}, ${getOnTimeoutLabel(step, index === props.steps.length - 1)}`
                  : "No timeout"}
              </span>
              {(sod?.allowRequesterReview || sod?.allowSubjectReview) && (
                <span>
                  {sod.allowRequesterReview && sod.allowSubjectReview
                    ? "Requester and subject may review"
                    : sod.allowRequesterReview
                      ? "Requester may review"
                      : "Subject may review"}
                </span>
              )}
            </div>
            {props.showSurfaces && step.surfaces.length > 0 && (
              <span className="flex flex-wrap gap-1.5">
                {step.surfaces.map((surface, surfaceIndex) => (
                  <SurfaceChip key={surfaceIndex} surface={surface} />
                ))}
              </span>
            )}
            {decisions.length > 0 && (
              <ul className="mt-0.5 flex flex-col gap-1">
                {decisions.map((decision, decisionIndex) => (
                  <li
                    key={decisionIndex}
                    className="flex flex-wrap items-center gap-1.5 text-xs text-slate-600"
                  >
                    {decision.decision === AccessP.Review_Spec_Decision.APPROVE ? (
                      <ToneBadge tone="success">Approved</ToneBadge>
                    ) : decision.decision === AccessP.Review_Spec_Decision.REJECT ? (
                      <ToneBadge tone="danger">Rejected</ToneBadge>
                    ) : (
                      <ToneBadge tone="neutral">Withdrawn</ToneBadge>
                    )}
                    {decision.reviewerRef ? (
                      <RefChip itemRef={decision.reviewerRef} icon={User} />
                    ) : (
                      <span className="text-slate-500">A reviewer</span>
                    )}
                    {decision.setAt && (
                      <span className="text-micro text-slate-500">
                        <TimeAgo rfc3339={decision.setAt} />
                      </span>
                    )}
                    {decision.viaIntegration && (
                      <span className="text-micro text-slate-500">
                        via {decision.viaIntegration}
                      </span>
                    )}
                    {decision.justification && (
                      <span className="w-full pl-1 text-micro italic text-slate-500">
                        “{decision.justification}”
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            )}
            {props.footerOf?.(index)}
          </div>
        </li>
      );
    })}
  </ol>
);

export const GrantSummary = (props: {
  authorization?: AccessP.Policy_Spec_Rule_Authorization;
  compact?: boolean;
}) => {
  const authz = props.authorization;
  const policies = authz?.policies ?? [];
  const inline = authz?.inlinePolicies ?? [];
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-xs text-slate-700">
        <Timer size={11} className="text-slate-500" />
        up to {formatDurationShort(authz?.maxAccessDuration, "24h")}
      </span>
      {policies.map((name) => (
        <RefChip
          key={name}
          itemRef={ObjectReference.create({
            apiVersion: "core/v1",
            kind: "Policy",
            name,
          })}
          icon={Shield}
        />
      ))}
      {inline.map((policy, index) => (
        <span
          key={`${policy.name}-${index}`}
          className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs text-slate-600"
        >
          <FileKey2 size={11} className="text-slate-500" />
          {policy.name || "inline Policy"}
        </span>
      ))}
      {policies.length === 0 && inline.length === 0 && !props.compact && (
        <span className="text-micro text-slate-500">
          access to the requested resource, with no extra core Policy
        </span>
      )}
    </span>
  );
};

export const providerIconForType = (type?: AccessP.Integration_Status_Type) => (
  <ProviderMark provider={getProviderKindFromStatus(type)} size={12} />
);

export const DenyNote = () => (
  <span className="inline-flex items-center gap-1 text-xs text-red-700">
    <Ban size={12} /> Rejected immediately
  </span>
);
