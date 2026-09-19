import { IntegrationIdentity } from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { GetIntegrationIdentitySummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import TimeAgo from "@/components/TimeAgo";
import { getClientVisibilityAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Link2, Plug, UserRound } from "lucide-react";
import { getSourceLabel } from "./utils";

export const getIntegrationRef = (
  item: IntegrationIdentity,
): ObjectReference | undefined =>
  item.status?.integrationRef
    ? ObjectReference.create({
        ...item.status.integrationRef,
        apiVersion: item.status.integrationRef.apiVersion || "access/v1",
        kind: item.status.integrationRef.kind || "Integration",
      })
    : undefined;

export const getUserRef = (
  item: IntegrationIdentity,
): ObjectReference | undefined =>
  item.status?.userRef
    ? ObjectReference.create({
        ...item.status.userRef,
        apiVersion: item.status.userRef.apiVersion || "core/v1",
        kind: item.status.userRef.kind || "User",
      })
    : undefined;

export const LabelComponent = (props: { item: IntegrationIdentity }) => {
  const { item } = props;
  const integrationRef = getIntegrationRef(item);
  const userRef = getUserRef(item);

  return (
    <ResourceListLabelWrap>
      {(integrationRef?.name || integrationRef?.uid) && (
        <ResourceListLabel itemRef={integrationRef} />
      )}
      {(userRef?.name || userRef?.uid) && <ResourceListLabel itemRef={userRef} />}
      {item.status?.externalUsername && (
        <ResourceListLabel label="External user">
          {item.status.externalUsername}
        </ResourceListLabel>
      )}
      {item.status?.externalEmail && (
        <ResourceListLabel label="Email">
          {item.status.externalEmail}
        </ResourceListLabel>
      )}
      <ResourceListLabel label="Source">
        {getSourceLabel(item.status?.source)}
      </ResourceListLabel>
      {item.status?.verifiedAt && (
        <ResourceListLabel label="Verified">
          <TimeAgo rfc3339={item.status.verifiedAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: IntegrationIdentity }) => {
  return <div></div>;
};

const DoSummary = ({
  resp,
}: {
  resp: GetIntegrationIdentitySummaryResponse;
}) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount
          count={resp.totalNumber}
          to="/access/integrationidentities"
        >
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalEmailDiscovery}
          to="/access/integrationidentities?source=EMAIL_DISCOVERY"
          icon={Link2}
        >
          Email discovery
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalIntegration} icon={Plug}>
          Integrations
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={UserRound}>
          Users
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "IntegrationIdentity"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccess().getIntegrationIdentitySummary({});
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return (
    <div>
      {qry.data.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {qry.data.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
