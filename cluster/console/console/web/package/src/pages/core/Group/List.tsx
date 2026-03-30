import { Group } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  GetGroupSummaryResponse,
  ListUserOptions,
} from "@/apis/visibilityv1/core/vcorev1";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { Shield } from "lucide-react";

const ItemDetails = (props: { item: Group; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Group }) => {
  const { item } = props;

  const qryGroup = useQuery({
    queryKey: ["selectGroupComponent", item.metadata!.name],
    queryFn: async () => {
      return await getClientVisibilityCore().listUser(
        ListUserOptions.create({
          groupRef: getResourceRef(item),
        }),
      );
    },
  });

  return (
    <ResourceListLabelWrap>
      {qryGroup.isSuccess &&
        qryGroup.data.response.listResponseMeta &&
        qryGroup.data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/users`, {
              "groupRef.name": item.metadata!.name,
            })}
          >
            {qryGroup.data.response.listResponseMeta.totalCount} Users
          </ResourceListLabel>
        )}
      {item.spec?.authorization &&
        item.spec?.authorization.policies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.policies.length} Policies
          </ResourceListLabel>
        )}
      {item.spec?.authorization &&
        item.spec?.authorization.inlinePolicies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.inlinePolicies.length} Inline Policies
          </ResourceListLabel>
        )}
    </ResourceListLabelWrap>
  );
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
