import * as AccessC from "@/apis/accessv1/accessv1";
import InfoItem from "@/components/InfoItem";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "tailwind-merge";
import { getDecisionMeta } from "./utils";

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
                  itemRef={item.status!.requestRef}
                ></ResourceListLabel>
              ),
            },
          ]
        : []),

      ...(status?.userRef
        ? [
            {
              label: "Reviewer",
              value: (
                <ResourceListLabel
                  itemRef={item.status!.userRef}
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
