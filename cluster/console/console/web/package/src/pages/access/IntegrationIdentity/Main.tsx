import * as AccessC from "@/apis/accessv1/accessv1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getIntegrationRef, getUserRef } from "./List";
import { getSourceLabel } from "./utils";

export const ItemInfo = (props: { item: AccessC.IntegrationIdentity }) => {
  const { item } = props;
  return (
    <>
      <InfoItem title="External ID">
        <span>{item.status?.externalID}</span>
      </InfoItem>
      <InfoItem title="Source">
        <span>{getSourceLabel(item.status?.source)}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.IntegrationIdentity }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.IntegrationIdentity;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const integrationRef = getIntegrationRef(item);
  const userRef = getUserRef(item);

  return {
    status: {
      label: getSourceLabel(status?.source),
      tone: "info",
      hint: "IntegrationIdentities are entirely managed by the Cluster.",
    },
    items: [
      ...(integrationRef?.name || integrationRef?.uid
        ? [
            {
              label: "Integration",
              primary: true,
              value: <ResourceListLabel itemRef={integrationRef} />,
            },
          ]
        : []),
      ...(userRef?.name || userRef?.uid
        ? [
            {
              label: "Cluster User",
              primary: true,
              value: <ResourceListLabel itemRef={userRef} />,
            },
          ]
        : []),
      ...(status?.externalID
        ? [
            {
              label: "External actor ID",
              value: <CopyText value={status.externalID} />,
              hint: "Stable identifier of the external actor within the provider. It is unique within the Integration.",
            },
          ]
        : []),
      ...(status?.externalUsername
        ? [
            {
              label: "External username",
              value: <CopyText value={status.externalUsername} />,
            },
          ]
        : []),
      ...(status?.externalEmail
        ? [
            {
              label: "External email",
              value: <CopyText value={status.externalEmail} />,
              hint: "Email that the provider reports for the external actor. It is informational.",
            },
          ]
        : []),
      {
        label: "Source",
        value: <Label>{getSourceLabel(status?.source)}</Label>,
      },
      ...(status?.verifiedAt
        ? [
            {
              label: "Last verified",
              value: <TimeAgo rfc3339={status.verifiedAt} />,
              hint: "Time at which the link was last confirmed against the provider.",
            },
          ]
        : []),
    ],
  };
};
