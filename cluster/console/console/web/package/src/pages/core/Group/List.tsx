import { Group } from "@/apis/corev1/corev1";
import { GetGroupSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { GroupUsersLabel } from "./Main";

const ItemDetails = (props: { item: Group; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Group }) => {
  return <GroupUsersLabel item={props.item} />;
};

export const ExtraComponent = (props: { item: Group }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetGroupSummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/groups`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Group"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getGroupSummary({});

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }
  const d = qry.data;
  return (
    <div>
      {d.totalNumber > 0 && (
        <div>
          <DoSummary resp={qry.data} />
        </div>
      )}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
