import { Secret } from "@/apis/cordiumv1/cordiumv1";
import { GetSecretSummaryResponse } from "@/apis/visibilityv1/cordium/vcordiumv1";
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
import { Layers, UserRound } from "lucide-react";
import { spaceRefOf, userRefOf } from "../utils";

export const LabelComponent = (props: { item: Secret }) => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);
  const spaceRef = spaceRefOf(item.status?.spaceRef);

  return (
    <ResourceListLabelWrap>
      {spaceRef && <ResourceListLabel itemRef={spaceRef} />}
      {userRef && <ResourceListLabel label="User" itemRef={userRef} />}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetSecretSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/secrets">
          Total
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalSpace} icon={Layers}>
          Spaces
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
    queryKey: ["visibility", "cordium", "summary", "Secret"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCordium().getSecretSummary(
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
