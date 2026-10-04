import { Membership } from "@/apis/cordiumv1/cordiumv1";
import { GetMembershipSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
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
  Crown,
  Layers,
  ShieldCheck,
  UserRound,
  UsersRound,
} from "lucide-react";
import { getRoleMeta, spaceRefOf, userRefOf } from "../utils";

export const LabelComponent = (props: { item: Membership }) => {
  const { item } = props;
  const meta = getRoleMeta(item.spec?.role);
  const userRef = userRefOf(item.status?.userRef);
  const spaceRef = spaceRefOf(item.status?.spaceRef);
  const gitProviders = Object.keys(item.status?.gitProviderStateMap ?? {});

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Role">
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      {userRef && (
        <ResourceListLabel label="User" itemRef={userRef}>
          {item.status?.userInfo?.displayName || undefined}
        </ResourceListLabel>
      )}
      {spaceRef && <ResourceListLabel itemRef={spaceRef} />}
      {gitProviders.length > 0 && (
        <ResourceListLabel label="Git providers">
          {gitProviders.length}
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetMembershipSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/memberships">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRoleOwner}
          to="/cordium/memberships?role=OWNER"
          icon={Crown}
        >
          Owners
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRoleAdmin}
          to="/cordium/memberships?role=ADMIN"
          icon={ShieldCheck}
        >
          Admins
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRoleUser}
          to="/cordium/memberships?role=USER"
          icon={UserRound}
        >
          Users
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser} icon={UsersRound}>
          Members
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalSpace} icon={Layers}>
          Spaces
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "Membership"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCordium().getMembershipSummary({});
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
