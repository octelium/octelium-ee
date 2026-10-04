import { Space } from "@/apis/cordiumv1/cordiumv1";
import { GetSpaceSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientVisibilityCordium } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  Building2,
  Container,
  LayoutTemplate,
  ShieldOff,
  UserRound,
  UsersRound,
} from "lucide-react";
import { getLimitItems, getSpaceTypeLabel, userRefOf } from "../utils";

export const LabelComponent = (props: { item: Space }) => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);
  const maxLimits = getLimitItems(item.spec?.limit?.maxLimit);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">
        {getSpaceTypeLabel(item.status?.type)}
      </ResourceListLabel>
      {userRef && <ResourceListLabel label="Owner" itemRef={userRef} />}
      {item.spec?.authorization?.disableSSH && (
        <ResourceListLabel>SSH disabled</ResourceListLabel>
      )}
      {maxLimits.map((limit) => (
        <ResourceListLabel key={limit.label} label={`Max ${limit.label}`}>
          {limit.value}
        </ResourceListLabel>
      ))}
      <ResourceListLabel
        label="Workspaces"
        to={`/cordium/workspaces?spaceRef.name=${item.metadata!.name}`}
      >
        View
      </ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetSpaceSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/spaces">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalUserSpace}
          to="/cordium/spaces?type=USER"
          icon={UserRound}
        >
          Personal
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOrganizationSpace}
          to="/cordium/spaces?type=ORGANIZATION"
          icon={Building2}
        >
          Organization
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSSHDisabled}
          to="/cordium/spaces?isSSHDisabled=true"
          icon={ShieldOff}
        >
          SSH disabled
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalMembership}
          to="/cordium/memberships"
          icon={UsersRound}
        >
          Memberships
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWorkspace}
          to="/cordium/workspaces"
          icon={Container}
        >
          Workspaces
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalTemplate}
          to="/cordium/templates"
          icon={LayoutTemplate}
        >
          Templates
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "Space"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCordium().getSpaceSummary(
        {},
      );
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
