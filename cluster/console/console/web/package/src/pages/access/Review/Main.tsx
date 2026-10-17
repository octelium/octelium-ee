import * as AccessC from "@/apis/accessv1/accessv1";
import * as MetaC from "@/apis/metav1/metav1";
import InfoItem from "@/components/InfoItem";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Check, Layers, Server, User, X } from "lucide-react";
import { normalizeRef, RefChip, TONE_SOFT, ToneBadge } from "../common";
import { OriginValue, hasOrigin } from "../Origin";
import { getStatusMeta, getStatusTone, getUrgencyColor, getUrgencyLabel } from "../Request/utils";
import { getDecisionMeta } from "./utils";

const useReviewRequest = (item: AccessC.Review) => {
  const requestRef = normalizeRef(item.status?.requestRef, "access/v1", "Request");
  return useQuery({
    queryKey: ["access.getRequest", requestRef?.uid || requestRef?.name || "missing"],
    enabled: Boolean(requestRef?.uid || requestRef?.name),
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    queryFn: async () =>
      (
        await getClientAccess().getRequest(
          MetaC.GetOptions.create({ uid: requestRef?.uid, name: requestRef?.name }),
        )
      ).response,
  });
};

const decisionTone = (decision?: AccessC.Review_Spec_Decision) =>
  decision === AccessC.Review_Spec_Decision.APPROVE
    ? "success"
    : decision === AccessC.Review_Spec_Decision.REJECT
      ? "danger"
      : "neutral";

const DecisionHero = (props: { item: AccessC.Review }) => {
  const { item } = props;
  const status = item.status;
  const decision = item.spec?.decision;
  const meta = getDecisionMeta(decision);
  const tone = decisionTone(decision);
  const reviewer = status?.userRef?.name ?? "A reviewer";
  const request = useReviewRequest(item);
  const steps =
    request.data?.status?.rule?.action?.type.oneofKind === "review"
      ? request.data.status.rule.action.type.review.steps
      : [];

  return (
    <div className="flex items-start gap-3 border-b border-slate-200 bg-white px-4 py-4">
      <span
        className={twMerge(
          "flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border",
          TONE_SOFT[tone],
        )}
      >
        {decision === AccessC.Review_Spec_Decision.REJECT ? (
          <X size={18} strokeWidth={2.4} />
        ) : (
          <Check size={18} strokeWidth={2.4} />
        )}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="text-base font-bold text-slate-900">
          {meta.label} by {reviewer}
        </span>
        <span className="text-xs leading-5 text-slate-600">
          {status?.stepName ? `At the step ${status.stepName}` : `At step ${(status?.stepIndex ?? 0) + 1}`}
          {steps.length > 1 && ` (${(status?.stepIndex ?? 0) + 1} of ${steps.length})`}
          {status?.setAt && (
            <>
              {" · "}
              <TimeAgo rfc3339={status.setAt} />
            </>
          )}
          {(status?.lastRevisions.length ?? 0) > 0 &&
            ` · changed ${status!.lastRevisions.length} time${status!.lastRevisions.length === 1 ? "" : "s"}`}
        </span>
        {item.spec?.justification && (
          <span className="mt-1 w-fit max-w-2xl whitespace-pre-wrap break-words rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs italic leading-5 text-slate-700">
            “{item.spec.justification}”
          </span>
        )}
      </div>
    </div>
  );
};

const RequestContext = (props: { item: AccessC.Review }) => {
  const query = useReviewRequest(props.item);
  const request = query.data;
  const requestRef = normalizeRef(props.item.status?.requestRef, "access/v1", "Request");

  if (query.isLoading)
    return <span className="block h-16 motion-safe:animate-pulse rounded-lg bg-slate-100" />;

  if (!request)
    return (
      <span className="flex flex-wrap items-center gap-1.5 text-xs text-slate-500">
        {requestRef && <RefChip itemRef={requestRef} />}
        The request could not be loaded; it may have been deleted.
      </span>
    );

  const requester = normalizeRef(request.status?.userRef, "core/v1", "User");
  const subject =
    request.spec?.subject?.type.oneofKind === "userRef"
      ? normalizeRef(request.spec.subject.type.userRef, "core/v1", "User")
      : undefined;
  const resource =
    request.spec?.resource?.type.oneofKind === "serviceRef"
      ? normalizeRef(request.spec.resource.type.serviceRef, "core/v1", "Service")
      : request.spec?.resource?.type.oneofKind === "catalog"
        ? normalizeRef(request.spec.resource.type.catalog.catalogRef, "access/v1", "Catalog")
        : undefined;
  const state = request.status?.state?.status;
  const urgency = request.spec?.urgency ?? AccessC.Request_Spec_Urgency.URGENCY_UNSET;

  return (
    <div className="flex flex-col gap-2 rounded-xl border border-slate-200 bg-white px-3 py-3">
      <span className="flex flex-wrap items-center gap-2">
        {requestRef && <RefChip itemRef={requestRef} />}
        <ToneBadge tone={getStatusTone(state)}>{getStatusMeta(state).label}</ToneBadge>
        <span className="inline-flex items-center gap-1 text-micro font-semibold" style={{ color: getUrgencyColor(urgency) }}>
          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: getUrgencyColor(urgency) }} />
          {getUrgencyLabel(urgency)}
        </span>
      </span>
      <span className="flex flex-wrap items-center gap-1.5 text-xs text-slate-600">
        {requester && <RefChip itemRef={requester} icon={User} />}
        {subject && requester && subject.name !== requester.name && (
          <>
            for <RefChip itemRef={subject} icon={User} />
          </>
        )}
        <span>asked for</span>
        {resource && <RefChip itemRef={resource} icon={resource.kind === "Catalog" ? Layers : Server} />}
        {request.metadata?.createdAt && (
          <span className="text-micro text-slate-500">
            <TimeAgo rfc3339={request.metadata.createdAt} />
          </span>
        )}
      </span>
      {request.spec?.justification && (
        <span className="whitespace-pre-wrap break-words text-xs leading-5 text-slate-700">
          {request.spec.justification}
        </span>
      )}
    </div>
  );
};

const History = (props: { item: AccessC.Review }) => {
  const status = props.item.status;
  const entries = [
    {
      decision: props.item.spec?.decision,
      justification: props.item.spec?.justification,
      setAt: status?.setAt,
      origin: status?.origin,
      current: true,
    },
    ...(status?.lastRevisions ?? []).map((revision) => ({
      decision: revision.spec?.decision,
      justification: revision.spec?.justification,
      setAt: revision.setAt,
      origin: revision.origin,
      current: false,
    })),
  ];

  return (
    <ol className="flex flex-col gap-2">
      {entries.map((entry, index) => (
        <li key={`${entry.setAt?.seconds ?? 0}-${index}`} className="flex flex-col gap-1">
          <span className="flex flex-wrap items-center gap-2">
            <ToneBadge tone={decisionTone(entry.decision)}>{getDecisionMeta(entry.decision).label}</ToneBadge>
            {entry.current && <span className="text-micro font-semibold text-slate-500">current</span>}
            {entry.setAt && (
              <span className="text-micro text-slate-500">
                <TimeAgo rfc3339={entry.setAt} />
              </span>
            )}
            {hasOrigin(entry.origin) && <OriginValue origin={entry.origin!} />}
          </span>
          {entry.justification && !entry.current && (
            <span className="pl-1 text-micro italic text-slate-500">“{entry.justification}”</span>
          )}
        </li>
      ))}
    </ol>
  );
};

export const ItemInfo = (props: { item: AccessC.Review }) => {
  const meta = getDecisionMeta(props.item.spec?.decision);
  return (
    <>
      <InfoItem title="Decision">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Review }) => (
  <div className="w-full">
    <ItemInfo item={props.item} />
  </div>
);

export const MainInfo = (props: { item: AccessC.Review }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const meta = getDecisionMeta(item.spec?.decision);
  const reviewerRef = normalizeRef(status?.userRef, "core/v1", "User");

  return {
    status: { label: meta.label, tone: decisionTone(item.spec?.decision) },
    header: <DecisionHero item={item} />,
    groupOrder: ["Request", "History"],
    items: [
      {
        label: "Reviewer",
        primary: true,
        value: reviewerRef ? <RefChip itemRef={reviewerRef} icon={User} /> : "Unknown",
      },
      {
        label: "Step",
        primary: true,
        value: (
          <span className="font-semibold">
            {status?.stepName || `Step ${(status?.stepIndex ?? 0) + 1}`}
          </span>
        ),
      },
      {
        label: "Decided from",
        primary: true,
        value: hasOrigin(status?.origin) ? (
          <OriginValue origin={status!.origin!} />
        ) : (
          <span className="text-slate-500">Unknown</span>
        ),
      },
      {
        label: "",
        group: "Request",
        layout: "stacked" as const,
        value: <RequestContext item={item} />,
      },
      ...((status?.lastRevisions.length ?? 0) > 0
        ? [
            {
              label: "",
              group: "History",
              layout: "stacked" as const,
              value: <History item={item} />,
            },
          ]
        : []),
    ],
  };
};
