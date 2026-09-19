import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ListIntegrationBindingOptions } from "@/apis/visibilityv1/access/vaccessv1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { twMerge } from "tailwind-merge";
import { getAudienceLabel } from "../Integration/utils";
import {
  getPurposeLabel,
  getStateMeta as getBindingStateMeta,
} from "../IntegrationBinding/utils";
import { OriginValue, hasOrigin } from "../Origin";
import { formatDuration, getStatusMeta, getUrgencyLabel } from "./utils";

const PRESENTATION_ITEMS = 25;

const effectLabel = (effect?: AccessC.Policy_Spec_Rule_Effect): string =>
  effect === AccessC.Policy_Spec_Rule_Effect.DENY
    ? "Deny"
    : effect === AccessC.Policy_Spec_Rule_Effect.REVIEW
      ? "Review"
      : effect === AccessC.Policy_Spec_Rule_Effect.AUTO_APPROVE
        ? "Auto approve"
        : "Unset";

const ruleSurfaceCount = (rule: AccessC.Policy_Spec_Rule): number =>
  rule.notifications.length +
  (rule.action?.type.oneofKind === "review"
    ? rule.action.type.review.steps.reduce(
        (total, step) => total + step.surfaces.length,
        0,
      )
    : 0);

const Presentations = (props: { item: AccessC.Request }) => {
  const identity = props.item.metadata?.uid || props.item.metadata?.name;
  const query = useQuery({
    queryKey: ["access.integrationBinding.request", identity],
    queryFn: async () =>
      (
        await getClientVisibilityAccess().listIntegrationBinding(
          ListIntegrationBindingOptions.create({
            common: CommonListOptions.create({
              itemsPerPage: PRESENTATION_ITEMS,
            }),
            requestRef: getResourceRef(props.item),
          }),
        )
      ).response,
    enabled: !!identity,
    staleTime: 15_000,
    refetchOnWindowFocus: false,
  });

  if (query.isError) {
    return (
      <span className="text-xs font-normal text-slate-500">
        The Integration presentations could not be loaded
      </span>
    );
  }
  if (query.isLoading) {
    return <span className="text-xs font-normal text-slate-500">Loading…</span>;
  }

  const items = query.data?.items ?? [];
  if (items.length === 0) {
    return (
      <span className="text-xs font-normal text-slate-500">
        This Request has not been presented on any Integration
      </span>
    );
  }

  return (
    <div className="w-full overflow-hidden rounded-lg border border-slate-200">
      {items.map((binding) => {
        const meta = getBindingStateMeta(binding.status?.state);
        const integrationRef = binding.status?.integrationRef;
        return (
          <div
            key={binding.metadata!.uid}
            className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0"
          >
            <div className="flex min-w-0 flex-wrap items-center gap-1.5">
              <ResourceListLabel itemRef={getResourceRef(binding)} />
              {(integrationRef?.name || integrationRef?.uid) && (
                <ResourceListLabel
                  itemRef={ObjectReference.create({
                    ...integrationRef,
                    apiVersion: integrationRef.apiVersion || "access/v1",
                    kind: integrationRef.kind || "Integration",
                  })}
                />
              )}
              <Label size="sm" outlined>
                {getPurposeLabel(binding.status?.purpose)}
              </Label>
              <Label size="sm" outlined>
                {getAudienceLabel(binding.status?.audience)}
              </Label>
            </div>
            <span className={twMerge("text-xs font-semibold", meta.className)}>
              {meta.label}
            </span>
          </div>
        );
      })}
    </div>
  );
};

export const ItemInfo = (props: { item: AccessC.Request }) => {
  const { item } = props;
  const meta = getStatusMeta(item.status?.state?.status);
  return (
    <>
      <InfoItem title="State">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
      <InfoItem title="Urgency">
        <span>{getUrgencyLabel(item.spec?.urgency)}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Request }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.Request;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const meta = getStatusMeta(status?.state?.status);
  const requesterRef = status?.userRef
    ? ObjectReference.create({
        ...status.userRef,
        apiVersion: status.userRef.apiVersion || "core/v1",
        kind: status.userRef.kind || "User",
      })
    : undefined;
  const policyRef = status?.policyRef
    ? ObjectReference.create({
        ...status.policyRef,
        apiVersion: status.policyRef.apiVersion || "access/v1",
        kind: status.policyRef.kind || "Policy",
      })
    : undefined;
  const subjectRef =
    item.spec?.subject?.type?.oneofKind === "userRef"
      ? item.spec.subject.type.userRef
      : undefined;
  const resourceRef =
    item.spec?.resource?.type?.oneofKind === "serviceRef"
      ? ObjectReference.create({
          ...item.spec.resource.type.serviceRef,
          apiVersion:
            item.spec.resource.type.serviceRef.apiVersion || "core/v1",
          kind: item.spec.resource.type.serviceRef.kind || "Service",
        })
      : item.spec?.resource?.type?.oneofKind === "catalog"
        ? ObjectReference.create({
            ...item.spec.resource.type.catalog.catalogRef,
            apiVersion:
              item.spec.resource.type.catalog.catalogRef?.apiVersion ||
              "access/v1",
            kind:
              item.spec.resource.type.catalog.catalogRef?.kind || "Catalog",
          })
        : undefined;

  return {
    items: [
      {
        label: "State",
        value: (
          <span className="flex items-center gap-2">
            <span className={twMerge("text-sm font-semibold", meta.className)}>
              {meta.label}
            </span>
            {status?.state?.createdAt && (
              <span className="text-xs font-normal text-slate-500">
                <TimeAgo rfc3339={status.state.createdAt} />
              </span>
            )}
          </span>
        ),
      },

      {
        label: "Urgency",
        value: (
          <span className="text-sm font-semibold text-slate-700">
            {getUrgencyLabel(item.spec?.urgency)}
          </span>
        ),
      },

      ...(requesterRef?.name || requesterRef?.uid
        ? [
            {
              label: "Requester",
              value: (
                <ResourceListLabel
                  label="User"
                  itemRef={requesterRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      ...(subjectRef?.name || subjectRef?.uid
        ? [
            {
              label: "Subject",
              value: (
                <ResourceListLabel
                  label="User"
                  itemRef={ObjectReference.create({
                    ...subjectRef,
                    apiVersion: subjectRef.apiVersion || "core/v1",
                    kind: subjectRef.kind || "User",
                  })}
                />
              ),
            },
          ]
        : []),

      ...(resourceRef?.name || resourceRef?.uid
        ? [
            {
              label: "Requested resource",
              value: <ResourceListLabel itemRef={resourceRef} />,
            },
          ]
        : []),

      ...(policyRef?.name || policyRef?.uid
        ? [
            {
              label: "Policy",
              value: (
                <ResourceListLabel
                  itemRef={policyRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      ...(item.spec?.justification
        ? [
            {
              label: "Justification",
              span: "full" as const,
              value: (
                <span className="text-body font-semibold text-slate-700">
                  {item.spec.justification}
                </span>
              ),
            },
          ]
        : []),

      ...(item.spec?.deadline
        ? [
            {
              label: "Deadline",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={item.spec.deadline} />
                </span>
              ),
            },
          ]
        : []),

      ...(status?.approvalStartAt
        ? [
            {
              label: "Approval start",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={status.approvalStartAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(status?.approvalEndAt
        ? [
            {
              label: "Approval end",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={status.approvalEndAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(status?.accessEndsAt
        ? [
            {
              label: "Access ends",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={status.accessEndsAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(status?.review
        ? [
            {
              label: "Current review step",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  {status.review.currentStep}
                  {status.review.lastSteps.length > 0 && (
                    <span className="ml-1.5 text-xs font-normal text-slate-500">
                      {status.review.lastSteps.length} decided
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),

      ...(status?.review?.currentStepStartedAt
        ? [
            {
              label: "Current step started",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={status.review.currentStepStartedAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(item.spec?.duration
        ? [
            {
              label: "Requested duration",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  {formatDuration(item.spec.duration)}
                </span>
              ),
            },
          ]
        : []),

      ...(status?.effectiveDuration
        ? [
            {
              label: "Effective duration",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  {formatDuration(status.effectiveDuration)}
                </span>
              ),
              hint: "The requested duration capped by the maximum access duration of the matched Rule.",
            },
          ]
        : []),

      ...(status?.policyTriggerRef?.name || status?.policyTriggerRef?.uid
        ? [
            {
              label: "Policy trigger",
              value: (
                <ResourceListLabel
                  label="PolicyTrigger"
                  itemRef={ObjectReference.create({
                    ...status.policyTriggerRef,
                    apiVersion:
                      status.policyTriggerRef.apiVersion || "core/v1",
                    kind: status.policyTriggerRef.kind || "PolicyTrigger",
                  })}
                />
              ),
              hint: "Hidden PolicyTrigger that the Cluster creates in order to grant the access of an approved Request.",
            },
          ]
        : []),

      ...(hasOrigin(status?.origin)
        ? [
            {
              label: "Created from",
              span: "full" as const,
              value: <OriginValue origin={status!.origin!} />,
              hint: "Where the Request was created from. It is entirely set by the Cluster from the authenticated transport of the creation itself.",
            },
          ]
        : []),

      ...(status?.rule
        ? [
            {
              label: "Matched rule",
              span: "full" as const,
              hint: "Copy of the Policy Rule that matched the Request when it was evaluated. The subsequent changes to the Policy do not affect it.",
              value: (
                <div className="flex flex-wrap items-center gap-1.5">
                  <Label size="sm" outlined>
                    {status.rule.name || "Unnamed rule"}
                  </Label>
                  <Label
                    size="sm"
                    tone={
                      status.rule.effect ===
                      AccessC.Policy_Spec_Rule_Effect.DENY
                        ? "danger"
                        : status.rule.effect ===
                            AccessC.Policy_Spec_Rule_Effect.AUTO_APPROVE
                          ? "success"
                          : "info"
                    }
                  >
                    {effectLabel(status.rule.effect)}
                  </Label>
                  {status.rule.action?.type.oneofKind === "review" && (
                    <Label size="sm" outlined>
                      {status.rule.action.type.review.steps.length} review
                      {status.rule.action.type.review.steps.length === 1
                        ? " step"
                        : " steps"}
                    </Label>
                  )}
                  {ruleSurfaceCount(status.rule) > 0 && (
                    <Label size="sm" outlined>
                      {ruleSurfaceCount(status.rule)} Integration
                      {ruleSurfaceCount(status.rule) === 1
                        ? " surface"
                        : " surfaces"}
                    </Label>
                  )}
                </div>
              ),
            },
          ]
        : []),

      ...((status?.lastStates ?? []).length > 0
        ? [
            {
              label: "State history",
              span: "full" as const,
              value: (
                <div className="w-full overflow-hidden rounded-lg border border-slate-200">
                  {status!.lastStates.map((state, index) => {
                    const stateMeta = getStatusMeta(state.status);
                    return (
                      <div
                        key={`${state.createdAt?.seconds ?? 0}-${index}`}
                        className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0"
                      >
                        <span
                          className={twMerge(
                            "text-xs font-semibold",
                            stateMeta.className,
                          )}
                        >
                          {stateMeta.label}
                        </span>
                        {state.createdAt && (
                          <span className="text-xs font-normal text-slate-500">
                            <TimeAgo rfc3339={state.createdAt} />
                          </span>
                        )}
                      </div>
                    );
                  })}
                </div>
              ),
            },
          ]
        : []),

      {
        label: "Integration presentations",
        span: "full" as const,
        value: <Presentations item={item} />,
      },
    ],
  };
};
