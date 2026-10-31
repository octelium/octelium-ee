import * as AccessC from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  ListIntegrationBindingOptions,
  ListReviewOptions,
} from "@/apis/visibilityv1/access/vaccessv1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import InfoItem from "@/components/InfoItem";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  ArrowUpRight,
  Ban,
  BellRing,
  CalendarClock,
  Check,
  CircleSlash,
  Clock3,
  FileText,
  Hourglass,
  Inbox,
  KeyRound,
  Layers,
  ListOrdered,
  Scale,
  ScrollText,
  Send,
  Server,
  Shield,
  Siren,
  Timer,
  User,
  Users,
  X,
} from "lucide-react";
import * as React from "react";
import {
  EffectBadge,
  EmptyNote,
  formatDurationShort,
  formatSeconds,
  normalizeRef,
  RefChip,
  TONE_SOFT,
  ToneBadge,
} from "../common";
import {
  getAudienceLabel,
  getDeliveryPurposeLabel,
  getDeliveryStateMeta,
  isDeliveryFailing,
} from "../Integration/utils";
import { OriginValue, hasOrigin } from "../Origin";
import { GrantSummary, StepDecision, StepState, StepTimeline } from "../rules";
import {
  getApprovalRequirementLabel,
  getStatusMeta,
  getStatusTone,
  getUrgencyColor,
  getUrgencyLabel,
} from "./utils";

const toDate = (ts?: Timestamp) => (ts ? Timestamp.toDate(ts) : undefined);

const useNow = () => {
  const [now, setNow] = React.useState(() => Date.now());
  React.useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
};

const reviewSteps = (item: AccessC.Request) =>
  item.status?.rule?.action?.type.oneofKind === "review"
    ? item.status.rule.action.type.review.steps
    : [];

const wasApproved = (item: AccessC.Request) =>
  item.status?.state?.status === AccessC.Request_Status_State_Status.APPROVED ||
  (item.status?.lastStates ?? []).some(
    (state) => state.status === AccessC.Request_Status_State_Status.APPROVED,
  );

const resourceRefOf = (item: AccessC.Request) => {
  const type = item.spec?.resource?.type;
  if (type?.oneofKind === "serviceRef")
    return normalizeRef(type.serviceRef, "core/v1", "Service");
  if (type?.oneofKind === "catalog")
    return normalizeRef(type.catalog.catalogRef, "access/v1", "Catalog");
  return undefined;
};

const subjectRefOf = (item: AccessC.Request) =>
  item.spec?.subject?.type.oneofKind === "userRef"
    ? normalizeRef(item.spec.subject.type.userRef, "core/v1", "User")
    : undefined;

const useRequestReviews = (item: AccessC.Request) =>
  useQuery({
    queryKey: ["access", "request", item.metadata?.uid, "reviews"],
    enabled: !!item.metadata?.uid,
    staleTime: 10_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listReview(
          ListReviewOptions.create({
            common: CommonListOptions.create({ itemsPerPage: 100 }),
            requestRef: getResourceRef(item),
          }),
          { abort: signal },
        )
      ).response,
  });

const useRequestDeliveries = (item: AccessC.Request) =>
  useQuery({
    queryKey: ["access", "request", item.metadata?.uid, "deliveries"],
    enabled: !!item.metadata?.uid,
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listIntegrationBinding(
          ListIntegrationBindingOptions.create({
            common: CommonListOptions.create({ itemsPerPage: 50 }),
            requestRef: getResourceRef(item),
          }),
          { abort: signal },
        )
      ).response,
  });

const StatusHero = (props: { item: AccessC.Request }) => {
  const { item } = props;
  const now = useNow();
  const status = item.status;
  const state = status?.state?.status;
  const tone = getStatusTone(state);
  const meta = getStatusMeta(state);
  const steps = reviewSteps(item);
  const currentIndex = Math.min(status?.review?.currentStep ?? 0, Math.max(steps.length - 1, 0));
  const current = steps[currentIndex];
  const lastSteps = status?.review?.lastSteps ?? [];

  let icon = <Clock3 size={18} strokeWidth={2.2} />;
  let title: React.ReactNode = meta.label;
  let detail: React.ReactNode = null;
  let progress: number | undefined;

  switch (state) {
    case AccessC.Request_Status_State_Status.PENDING: {
      icon = <Hourglass size={18} strokeWidth={2.2} />;
      if (current) {
        const startedAt = toDate(status?.review?.currentStepStartedAt);
        const timeoutSeconds = current.timeout
          ? formatDurationShort(current.timeout)
          : undefined;
        title = (
          <>
            Waiting on {current.name || `step ${currentIndex + 1}`}
            {steps.length > 1 && (
              <span className="font-normal text-slate-500">
                {" "}
                · step {currentIndex + 1} of {steps.length}
              </span>
            )}
          </>
        );
        detail = (
          <>
            {getApprovalRequirementLabel(current)}
            {startedAt && (
              <>
                {" · "}waiting for{" "}
                {formatSeconds((now - startedAt.getTime()) / 1000)}
              </>
            )}
            {timeoutSeconds && ` · times out after ${timeoutSeconds}`}
          </>
        );
      } else {
        title = "Being evaluated";
        detail = "The Cluster is matching the request against the access Policies.";
      }
      break;
    }
    case AccessC.Request_Status_State_Status.APPROVED: {
      icon = <KeyRound size={18} strokeWidth={2.2} />;
      const endsAt = toDate(status?.accessEndsAt);
      const startAt = toDate(status?.approvalEndAt ?? status?.state?.createdAt);
      title = "Access granted";
      if (endsAt) {
        const left = (endsAt.getTime() - now) / 1000;
        detail = (
          <>
            {left > 0 ? `${formatSeconds(left)} left` : "Ending now"} · until{" "}
            {endsAt.toLocaleString()}
          </>
        );
        if (startAt) {
          const total = endsAt.getTime() - startAt.getTime();
          progress =
            total > 0
              ? Math.max(0, Math.min(100, ((endsAt.getTime() - now) / total) * 100))
              : 0;
        }
      } else {
        detail = "No end is set for this access.";
      }
      break;
    }
    case AccessC.Request_Status_State_Status.REJECTED: {
      icon = <X size={18} strokeWidth={2.4} />;
      const rejected = lastSteps.find(
        (step) => step.decision === AccessC.Review_Spec_Decision.REJECT,
      );
      if (!status?.rule) {
        title = "Rejected · no matching rule";
        detail = "No rule of any enabled access Policy matched this request.";
      } else if (status.rule.effect === AccessC.Policy_Spec_Rule_Effect.DENY) {
        title = "Denied by a rule";
        detail = `The rule ${status.rule.name} rejects such requests.`;
      } else if (rejected) {
        title = (
          <>
            Rejected by {rejected.userRef?.name ?? "a reviewer"}
            {rejected.stepName && (
              <span className="font-normal text-slate-500">
                {" "}
                · step {rejected.stepName}
              </span>
            )}
          </>
        );
      } else {
        title = "Rejected";
        detail = "The review step timed out or was rejected.";
      }
      break;
    }
    case AccessC.Request_Status_State_Status.EXPIRED:
      icon = <CalendarClock size={18} strokeWidth={2.2} />;
      title = wasApproved(item) ? "Access ended" : "Expired before a decision";
      detail = wasApproved(item)
        ? "The granted access reached its end."
        : "The deadline or a review step timeout passed.";
      break;
    case AccessC.Request_Status_State_Status.REVOKED:
      icon = <Ban size={18} strokeWidth={2.2} />;
      title = "Revoked";
      detail = "An administrator revoked this request; any granted access was removed.";
      break;
    case AccessC.Request_Status_State_Status.CANCELLED:
      icon = <CircleSlash size={18} strokeWidth={2.2} />;
      title = "Cancelled by the requester";
      break;
  }

  return (
    <div className="flex flex-col gap-3 border-b border-slate-200 bg-white px-4 py-4">
      <div className="flex items-start gap-3">
        <span
          className={twMerge(
            "flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border",
            TONE_SOFT[tone],
          )}
        >
          {icon}
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-base font-bold text-slate-900">{title}</span>
          {detail && (
            <span className="text-xs font-normal leading-5 text-slate-600">
              {detail}
            </span>
          )}
          {progress !== undefined && (
            <span
              role="progressbar"
              aria-label="Access time left"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(progress)}
              className="mt-1 h-1.5 w-full max-w-md overflow-hidden rounded-full bg-slate-100"
            >
              <span
                className="block h-full rounded-full bg-emerald-500"
                style={{ width: `${progress}%` }}
              />
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

const PeopleValue = (props: { item: AccessC.Request }) => {
  const requester = normalizeRef(props.item.status?.userRef, "core/v1", "User");
  const subject = subjectRefOf(props.item);
  const onBehalf = subject && requester && subject.uid !== requester.uid && subject.name !== requester.name;
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      {requester ? <RefChip itemRef={requester} icon={User} /> : <span>Unknown</span>}
      {onBehalf && (
        <>
          <span className="text-xs text-slate-500">for</span>
          <RefChip itemRef={subject} icon={User} />
        </>
      )}
    </span>
  );
};

const Workflow = (props: { item: AccessC.Request }) => {
  const { item } = props;
  const reviews = useRequestReviews(item);
  const steps = reviewSteps(item);
  const state = item.status?.state?.status;
  const pending = state === AccessC.Request_Status_State_Status.PENDING;
  const approved = wasApproved(item);
  const current = item.status?.review?.currentStep ?? 0;

  const stateOf = (index: number): StepState => {
    if (approved || index < current) return "done";
    if (index === current) return pending ? "current" : "stopped";
    return pending ? "upcoming" : "unreached";
  };

  const decisionsOf = (index: number): StepDecision[] => {
    const items = (reviews.data?.items ?? []).filter(
      (review) => review.status?.stepIndex === index,
    );
    if (items.length > 0) {
      return items.map((review) => ({
        reviewerRef: normalizeRef(review.status?.userRef, "core/v1", "User"),
        decision: review.spec?.decision ?? AccessC.Review_Spec_Decision.UNSET,
        setAt: review.status?.setAt,
        justification: review.spec?.justification,
        viaIntegration:
          review.status?.origin?.type === AccessC.Origin_Type.INTEGRATION
            ? review.status.origin.integrationRef?.name
            : undefined,
      }));
    }
    return (item.status?.review?.lastSteps ?? [])
      .filter((step) => step.stepIndex === index)
      .map((step) => ({
        reviewerRef: normalizeRef(step.userRef, "core/v1", "User"),
        decision: step.decision,
        setAt: step.setAt,
      }));
  };

  return (
    <StepTimeline
      steps={steps}
      stateOf={stateOf}
      decisionsOf={decisionsOf}
      showSurfaces
    />
  );
};

const Deliveries = (props: { item: AccessC.Request }) => {
  const query = useRequestDeliveries(props.item);

  if (query.isLoading)
    return <span className="block h-10 motion-safe:animate-pulse rounded-lg bg-slate-100" />;
  if (query.isError)
    return <EmptyNote>The deliveries could not be loaded.</EmptyNote>;

  const items = query.data?.items ?? [];
  if (items.length === 0)
    return (
      <EmptyNote>
        This request was not presented on any Integration. The access portal
        always shows it to its reviewers.
      </EmptyNote>
    );

  return (
    <ul className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
      {items.map((binding) => {
        const status = binding.status;
        const meta = getDeliveryStateMeta(status?.state);
        const integrationRef = normalizeRef(status?.integrationRef, "access/v1", "Integration");
        const failing = isDeliveryFailing(binding);
        return (
          <li
            key={binding.metadata?.uid}
            className={twMerge("flex flex-col gap-1 px-3 py-2", failing && "bg-red-50/40")}
          >
            <div className="flex flex-wrap items-center gap-2">
              {status?.purpose === AccessC.IntegrationBinding_Status_Purpose.NOTIFICATION ? (
                <BellRing size={12} className="text-slate-500" />
              ) : (
                <Inbox size={12} className="text-slate-500" />
              )}
              {integrationRef && <RefChip itemRef={integrationRef} />}
              <span className="text-xs text-slate-600">
                {getDeliveryPurposeLabel(status?.purpose)} · {getAudienceLabel(status?.audience)}
                {status?.stepName && ` · step ${status.stepName}`}
              </span>
              <span className="ml-auto flex items-center gap-2">
                {status?.externalURL && (
                  <a
                    href={status.externalURL}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="inline-flex items-center gap-0.5 text-micro font-semibold text-blue-700 hover:text-blue-800"
                  >
                    Open {status.externalID}
                    <ArrowUpRight size={11} strokeWidth={2.5} />
                  </a>
                )}
                <ToneBadge tone={meta.tone}>{meta.label}</ToneBadge>
              </span>
            </div>
            {failing && (
              <span className="flex items-start gap-1.5 pl-5 text-micro text-red-700">
                <AlertTriangle size={11} className="mt-px shrink-0" />
                {status?.lastError || "The last delivery failed"} · {status?.attempts} attempts
              </span>
            )}
          </li>
        );
      })}
    </ul>
  );
};

const History = (props: { item: AccessC.Request }) => {
  const status = props.item.status;
  const states = [...(status?.lastStates ?? [])].reverse();
  if (status?.state) states.push(status.state);
  const created = props.item.metadata?.createdAt;

  return (
    <ol className="flex flex-col gap-1.5">
      {created && (
        <li className="flex items-center gap-2 text-xs text-slate-600">
          <span className="h-2 w-2 rounded-full bg-slate-300" />
          Created
          <span className="text-micro text-slate-500">
            <TimeAgo rfc3339={created} />
          </span>
        </li>
      )}
      {states.map((state, index) => {
        const tone = getStatusTone(state.status);
        return (
          <li key={`${state.createdAt?.seconds ?? 0}-${index}`} className="flex items-center gap-2 text-xs text-slate-600">
            <span
              className={twMerge(
                "h-2 w-2 rounded-full",
                tone === "success"
                  ? "bg-emerald-500"
                  : tone === "danger"
                    ? "bg-red-500"
                    : tone === "warning"
                      ? "bg-amber-500"
                      : "bg-slate-400",
              )}
            />
            {getStatusMeta(state.status).label}
            {state.createdAt && (
              <span className="text-micro text-slate-500">
                <TimeAgo rfc3339={state.createdAt} />
              </span>
            )}
          </li>
        );
      })}
    </ol>
  );
};

export const ItemInfo = (props: { item: AccessC.Request }) => {
  const meta = getStatusMeta(props.item.status?.state?.status);
  return (
    <>
      <InfoItem title="State">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
      <InfoItem title="Urgency">
        <span>{getUrgencyLabel(props.item.spec?.urgency)}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Request }) => (
  <div className="w-full">
    <ItemInfo item={props.item} />
  </div>
);

export const MainInfo = (props: {
  item: AccessC.Request;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const resourceRef = resourceRefOf(item);
  const policyRef = normalizeRef(status?.policyRef, "access/v1", "Policy");
  const urgency = item.spec?.urgency ?? AccessC.Request_Spec_Urgency.URGENCY_UNSET;
  const steps = reviewSteps(item);
  const deadline = toDate(item.spec?.deadline);
  const overdue =
    !!deadline &&
    deadline.getTime() < Date.now() &&
    status?.state?.status === AccessC.Request_Status_State_Status.PENDING;
  const policyTriggerRef = normalizeRef(status?.policyTriggerRef, "core/v1", "PolicyTrigger");

  return {
    status: {
      label: getStatusMeta(status?.state?.status).label,
      tone: getStatusTone(status?.state?.status),
    },
    header: <StatusHero item={item} />,
    groupOrder: [
      { title: "Request", icon: FileText },
      { title: "Review workflow", icon: ListOrdered },
      { title: "Decision", icon: Scale },
      { title: "Delivered to", icon: Send },
      { title: "History", icon: ScrollText },
    ],
    items: [
      {
        label: "Who",
        primary: true,
        icon: Users,
        value: <PeopleValue item={item} />,
      },
      {
        label: "Resource",
        primary: true,
        icon: Server,
        value: resourceRef ? (
          <RefChip
            itemRef={resourceRef}
            icon={resourceRef.kind === "Catalog" ? Layers : Server}
          />
        ) : (
          "Unknown"
        ),
      },
      {
        label: "Duration",
        primary: true,
        icon: Timer,
        value: (
          <span className="flex flex-wrap items-baseline gap-1.5">
            <span className="font-semibold">
              {formatDurationShort(status?.effectiveDuration ?? item.spec?.duration, "Default")}
            </span>
            {item.spec?.duration &&
              status?.effectiveDuration &&
              formatDurationShort(item.spec.duration) !==
                formatDurationShort(status.effectiveDuration) && (
                <span className="text-xs text-slate-500">
                  capped from {formatDurationShort(item.spec.duration)}
                </span>
              )}
          </span>
        ),
      },
      {
        label: "Urgency",
        primary: true,
        icon: Siren,
        value: (
          <span className="inline-flex items-center gap-1.5 font-semibold" style={{ color: getUrgencyColor(urgency) }}>
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: getUrgencyColor(urgency) }}
            />
            {getUrgencyLabel(urgency)}
          </span>
        ),
      },

      ...(item.spec?.justification
        ? [
            {
              label: "Justification",
              group: "Request",
              span: "full" as const,
              value: (
                <span className="whitespace-pre-wrap break-words text-slate-700">
                  {item.spec.justification}
                </span>
              ),
            },
          ]
        : []),
      ...(deadline
        ? [
            {
              label: "Deadline",
              group: "Request",
              value: (
                <span className={twMerge("inline-flex items-center gap-1.5", overdue && "font-semibold text-red-700")}>
                  {overdue && <AlertTriangle size={13} />}
                  <TimeAgo rfc3339={item.spec!.deadline!} />
                </span>
              ),
            },
          ]
        : []),
      ...(hasOrigin(status?.origin)
        ? [
            {
              label: "Created from",
              group: "Request",
              value: <OriginValue origin={status!.origin!} />,
            },
          ]
        : []),

      ...(steps.length > 0
        ? [
            {
              label: "",
              group: "Review workflow",
              layout: "stacked" as const,
              value: <Workflow item={item} />,
            },
          ]
        : []),

      {
        label: "Matched rule",
        group: "Decision",
        span: "full" as const,
        value: status?.rule ? (
          <span className="flex flex-wrap items-center gap-1.5">
            {policyRef && <RefChip itemRef={policyRef} icon={Shield} />}
            <span className="text-xs font-semibold text-slate-700">{status.rule.name}</span>
            <EffectBadge effect={status.rule.effect} />
          </span>
        ) : (
          <span className="text-xs text-slate-500">
            {status?.state?.status === AccessC.Request_Status_State_Status.REJECTED
              ? "None: no rule of an enabled Policy matched"
              : "Not evaluated yet"}
          </span>
        ),
      },
      ...(status?.rule && status.rule.effect !== AccessC.Policy_Spec_Rule_Effect.DENY
        ? [
            {
              label: "Grants",
              group: "Decision",
              span: "full" as const,
              value: <GrantSummary authorization={status.rule.authorization} />,
            },
          ]
        : []),
      ...(policyTriggerRef
        ? [
            {
              label: "Enforced by",
              group: "Decision",
              value: (
                <span className="flex flex-wrap items-center gap-1.5">
                  <RefChip itemRef={policyTriggerRef} icon={Check} />
                  <span className="text-micro text-slate-500">hidden PolicyTrigger</span>
                </span>
              ),
            },
          ]
        : []),

      {
        label: "",
        group: "Delivered to",
        layout: "stacked" as const,
        value: <Deliveries item={item} />,
      },
      {
        label: "",
        group: "History",
        layout: "stacked" as const,
        value: <History item={item} />,
      },
    ],
  };
};
