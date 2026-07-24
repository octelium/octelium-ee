import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "tailwind-merge";
import { getStatusMeta, getUrgencyLabel } from "./utils";

export const ItemInfo = (props: { item: AccessC.Request }) => {
  const { item } = props;
  const meta = getStatusMeta(item.status?.state?.status);
  return (
    <>
      <InfoItem title="State">
        <span className={meta.className}>{meta.label}</span>
      </InfoItem>
      <InfoItem title="Urgency">
        <span>{getUrgencyLabel(item.spec!.urgency)}</span>
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
              <span className="text-[0.68rem] font-semibold text-slate-400">
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
            {getUrgencyLabel(item.spec!.urgency)}
          </span>
        ),
      },

      ...(status?.userRef
        ? [
            {
              label: "Requester",
              value: (
                <ResourceListLabel
                  itemRef={item.status!.userRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      ...(status?.policyRef
        ? [
            {
              label: "Policy",
              value: (
                <ResourceListLabel
                  itemRef={item.status!.policyRef}
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
                <span className="text-[0.78rem] font-semibold text-slate-700">
                  {item.spec.justification}
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
                </span>
              ),
            },
          ]
        : []),
    ],
  };
};
