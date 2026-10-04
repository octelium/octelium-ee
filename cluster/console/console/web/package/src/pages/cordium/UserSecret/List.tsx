import { UserSecret } from "@/apis/cordiumv1/cordiumv1";
import { GetUserSecretSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
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
import { KeyRound, KeySquare, UserRound } from "lucide-react";
import { getUserSecretTypeLabel, userRefOf } from "../utils";

export const LabelComponent = (props: { item: UserSecret }) => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">
        {getUserSecretTypeLabel(item.spec?.type)}
      </ResourceListLabel>
      {userRef && <ResourceListLabel label="User" itemRef={userRef} />}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetUserSecretSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/usersecrets">
          Total
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalDefault} icon={KeyRound}>
          Default
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSSHKey}
          to="/cordium/usersecrets?type=SSH_KEY"
          icon={KeySquare}
        >
          SSH keys
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
    queryKey: ["visibility", "cordium", "summary", "UserSecret"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCordium().getUserSecretSummary({});
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
