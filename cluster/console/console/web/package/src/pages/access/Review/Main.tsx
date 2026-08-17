import * as AccessC from "@/apis/accessv1/accessv1";
import * as MetaC from "@/apis/metav1/metav1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { getUrgencyLabel } from "../Request/utils";
import { getDecisionMeta } from "./utils";

const normalizeRef = (
  ref: MetaC.ObjectReference | undefined,
  apiVersion: string,
  kind: string,
) =>
  ref
    ? MetaC.ObjectReference.create({
        ...ref,
        apiVersion: ref.apiVersion || apiVersion,
        kind: ref.kind || kind,
      })
    : undefined;

export const ItemInfo = (props: { item: AccessC.Review }) => {
  const { item } = props;
  const meta = getDecisionMeta(item.spec?.decision);
  return (
    <>
      <InfoItem title="Decision">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Review }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: { item: AccessC.Review }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const meta = getDecisionMeta(item.spec?.decision);
  const requestRef = normalizeRef(status?.requestRef, "access/v1", "Request");
  const requestQry = useQuery({
    queryKey: [
      "access.getRequest",
      requestRef?.uid || requestRef?.name || "missing",
    ],
    enabled: Boolean(requestRef?.uid || requestRef?.name),
    queryFn: async () => {
      const { response } = await getClientAccess().getRequest(
        MetaC.GetOptions.create({
          uid: requestRef?.uid,
          name: requestRef?.name,
        }),
      );
      return response;
    },
  });
  const request = requestQry.data;
  const reviewerRef = normalizeRef(status?.userRef, "core/v1", "User");
  const requesterRef = normalizeRef(request?.status?.userRef, "core/v1", "User");
  const subjectRef =
    request?.spec?.subject?.type.oneofKind === "userRef"
      ? normalizeRef(request.spec.subject.type.userRef, "core/v1", "User")
      : undefined;
  const resourceRef =
    request?.spec?.resource?.type.oneofKind === "serviceRef"
      ? normalizeRef(
          request.spec.resource.type.serviceRef,
          "core/v1",
          "Service",
        )
      : request?.spec?.resource?.type.oneofKind === "catalog"
        ? normalizeRef(
            request.spec.resource.type.catalog.catalogRef,
            "access/v1",
            "Catalog",
          )
        : undefined;

  return {
    items: [
      {
        label: "Decision",
        value: (
          <span className={twMerge("text-sm font-semibold", meta.className)}>
            {meta.label}
          </span>
        ),
      },

      ...(item.spec?.justification
        ? [
            {
              label: "Justification",
              span: "full" as const,
              value: (
                <span className="text-[0.78rem] font-semibold text-slate-700">
                  {item.spec.justification}
                </span>
              ),
            },
          ]
        : []),

      ...(status?.requestRef
        ? [
            {
              label: "Request",
              value: (
                <ResourceListLabel
                  itemRef={requestRef!}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      ...(requestQry.isLoading
        ? [
            {
              label: "Request details",
              value: (
                <span className="inline-flex items-center gap-1.5 text-[0.72rem] font-semibold text-slate-400">
                  <Loader2 size={12} className="animate-spin" />
                  Loading request context…
                </span>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(requestQry.isError
        ? [
            {
              label: "Request details",
              value: (
                <span className="text-[0.72rem] font-semibold text-red-600">
                  The linked request details could not be loaded.
                </span>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(requesterRef?.name || requesterRef?.uid
        ? [
            {
              label: "Requester",
              value: (
                <ResourceListLabel label="User" itemRef={requesterRef} />
              ),
            },
          ]
        : []),

      ...(subjectRef?.name || subjectRef?.uid
        ? [
            {
              label: "Subject",
              value: <ResourceListLabel label="User" itemRef={subjectRef} />,
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

      ...(request?.status?.state
        ? [
            {
              label: "Request state",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  {AccessC.Request_Status_State_Status[
                    request.status.state.status
                  ]
                    .replace("STATUS_", "")
                    .replaceAll("_", " ")}
                </span>
              ),
            },
          ]
        : []),

      ...(request?.spec
        ? [
            {
              label: "Request urgency",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  {getUrgencyLabel(request.spec.urgency)}
                </span>
              ),
            },
          ]
        : []),

      ...(request?.spec?.deadline
        ? [
            {
              label: "Request deadline",
              value: <TimeAgo rfc3339={request.spec.deadline} />,
            },
          ]
        : []),

      ...(request?.spec?.justification
        ? [
            {
              label: "Request justification",
              value: (
                <span className="text-[0.78rem] font-semibold text-slate-700">
                  {request.spec.justification}
                </span>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(reviewerRef?.name || reviewerRef?.uid
        ? [
            {
              label: "Reviewer",
              value: (
                <ResourceListLabel
                  label="User"
                  itemRef={reviewerRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      {
        label: "Step index",
        value: (
          <span className="text-sm font-semibold text-slate-700">
            {status?.stepIndex ?? 0}
          </span>
        ),
      },

      ...(status?.setAt
        ? [
            {
              label: "Set at",
              value: (
                <span className="text-sm font-semibold text-slate-700">
                  <TimeAgo rfc3339={status.setAt} />
                </span>
              ),
            },
          ]
        : []),
    ],
  };
};
