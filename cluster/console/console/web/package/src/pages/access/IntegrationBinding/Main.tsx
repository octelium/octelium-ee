import * as AccessC from "@/apis/accessv1/accessv1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Button } from "@mantine/core";
import { AlertTriangle, ExternalLink } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { getAudienceLabel, getInteractionModeLabel } from "../Integration/utils";
import { getIntegrationRef, getRequestRef, getUserRef } from "./List";
import { getPurposeLabel, getStateMeta, isFailing, isOutOfDate } from "./utils";

export const ItemInfo = (props: { item: AccessC.IntegrationBinding }) => {
  const { item } = props;
  const meta = getStateMeta(item.status?.state);
  return (
    <>
      <InfoItem title="State">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
      <InfoItem title="Purpose">
        <span>{getPurposeLabel(item.status?.purpose)}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.IntegrationBinding }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.IntegrationBinding;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const meta = getStateMeta(status?.state);
  const integrationRef = getIntegrationRef(item);
  const requestRef = getRequestRef(item);
  const userRef = getUserRef(item);
  const outOfDate = isOutOfDate(item);

  return {
    status: { label: meta.label, tone: meta.tone },
    actions: status?.externalURL ? (
      <Button
        component="a"
        href={status.externalURL}
        target="_blank"
        rel="noopener noreferrer"
        variant="default"
        size="sm"
        leftSection={<ExternalLink size={13} />}
      >
        Open external object
      </Button>
    ) : undefined,
    groupOrder: ["IntegrationBinding details", "External object", "Delivery"],
    items: [
      {
        label: "State",
        primary: true,
        value: (
          <span className={twMerge("font-semibold", meta.className)}>
            {meta.label}
          </span>
        ),
      },
      {
        label: "Purpose",
        primary: true,
        value: <Label>{getPurposeLabel(status?.purpose)}</Label>,
      },
      ...(integrationRef?.name || integrationRef?.uid
        ? [
            {
              label: "Integration",
              value: <ResourceListLabel itemRef={integrationRef} />,
            },
          ]
        : []),
      ...(requestRef?.name || requestRef?.uid
        ? [
            {
              label: "Request",
              value: <ResourceListLabel itemRef={requestRef} />,
            },
          ]
        : []),
      {
        label: "Audience",
        value: <Label>{getAudienceLabel(status?.audience)}</Label>,
      },
      ...(userRef?.name || userRef?.uid
        ? [
            {
              label: "Delivered to",
              value: <ResourceListLabel itemRef={userRef} />,
              hint: "Cluster User that the external object is directly delivered to. It is unset for the shared destinations.",
            },
          ]
        : []),
      {
        label: "Interaction mode",
        value: <Label>{getInteractionModeLabel(status?.interactionMode)}</Label>,
        hint: "How much the external provider is trusted to act on the presented Request.",
      },
      ...(status?.purpose ===
      AccessC.IntegrationBinding_Status_Purpose.REVIEW_SURFACE
        ? [
            {
              label: "Review step",
              value: (
                <span className="font-semibold text-slate-700">
                  {status.stepName || `Step ${status.stepIndex}`}
                  <span className="ml-1.5 text-xs font-normal text-slate-500">
                    index {status.stepIndex}
                  </span>
                </span>
              ),
            },
          ]
        : []),

      ...(status?.externalID
        ? [
            {
              label: "External object ID",
              group: "External object",
              value: <CopyText value={status.externalID} />,
              hint: "Provider's identifier of the external object (e.g. a Slack message timestamp or a Jira issue key).",
            },
          ]
        : []),
      ...(status?.externalRecipientID
        ? [
            {
              label: "External recipient ID",
              group: "External object",
              value: <CopyText value={status.externalRecipientID} />,
              hint: "Provider's identifier of the destination that the external object was delivered to.",
            },
          ]
        : []),
      ...(status?.externalURL
        ? [
            {
              label: "External URL",
              group: "External object",
              span: "full" as const,
              value: <CopyText value={status.externalURL} />,
            },
          ]
        : []),
      ...((status?.lastExternalEventIDs ?? []).length > 0
        ? [
            {
              label: "Recent provider events",
              group: "External object",
              span: "full" as const,
              hint: "Identifiers of the most recent provider events that were accepted through the external object. They are used to discard the duplicate deliveries of the same external action.",
              value: (
                <div className="flex flex-wrap gap-1.5">
                  {status!.lastExternalEventIDs.map((eventID) => (
                    <Label key={eventID} size="sm" outlined>
                      {eventID}
                    </Label>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      {
        label: "Presentation revision",
        group: "Delivery",
        span: "full" as const,
        value: (
          <div className="flex flex-wrap items-center gap-2">
            <Label size="sm" outlined>
              Desired {status?.desiredRevision || "unset"}
            </Label>
            <Label size="sm" outlined>
              Applied {status?.appliedRevision || "unset"}
            </Label>
            <span
              className={twMerge(
                "text-xs font-semibold",
                outOfDate ? "text-amber-600" : "text-emerald-600",
              )}
            >
              {outOfDate ? "Out of date" : "Up to date"}
            </span>
          </div>
        ),
      },
      ...(status?.lastSuccessAt
        ? [
            {
              label: "Last successful delivery",
              group: "Delivery",
              value: <TimeAgo rfc3339={status.lastSuccessAt} />,
            },
          ]
        : []),
      ...(isFailing(item)
        ? [
            {
              label: "Failed attempts",
              group: "Delivery",
              value: (
                <span className="font-semibold text-red-600">
                  {status!.attempts}
                </span>
              ),
            },
          ]
        : []),
      ...(status?.nextAttemptAt
        ? [
            {
              label: "Next attempt",
              group: "Delivery",
              value: <TimeAgo rfc3339={status.nextAttemptAt} />,
              hint: "Time before which no further delivery is attempted. It is set by the Cluster's backoff upon the failures.",
            },
          ]
        : []),
      ...(status?.lastError
        ? [
            {
              label: "Last error",
              group: "Delivery",
              span: "full" as const,
              value: (
                <div className="flex items-start gap-2 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
                  <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                  {status.lastError}
                </div>
              ),
            },
          ]
        : []),
    ],
  };
};
