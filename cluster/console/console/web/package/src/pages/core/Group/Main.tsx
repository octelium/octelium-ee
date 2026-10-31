import * as CoreC from "@/apis/corev1/corev1";
import { ListUserOptions } from "@/apis/visibilityv1/core/vcorev1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ShieldCheck, Tags, Users } from "lucide-react";
import {
  attrsItems,
  authorizationItems,
} from "@/components/ResourceLayout/overviewItems";

export default (props: { item: CoreC.Group }) => {
  const { item } = props;
  return <></>;
};

export const GroupUsersLabel = (props: { item: CoreC.Group }) => {
  const itemName = props.item.metadata!.name;
  const qryUsers = useQuery({
    queryKey: ["selectGroupComponent", itemName],
    queryFn: () =>
      getClientVisibilityCore().listUser(
        ListUserOptions.create({ groupRef: getResourceRef(props.item) }),
      ).response,
  });

  return (
    <ResourceListLabel
      label="Users"
      to={`/core/users?groupRef.name=${encodeURIComponent(itemName)}`}
    >
      <Users size={12} strokeWidth={2.5} />
      {qryUsers.data?.listResponseMeta?.totalCount?.toLocaleString() ?? "…"}
    </ResourceListLabel>
  );
};

export const MainInfo = (props: { item: CoreC.Group }): ResourceMainInfo => {
  const { item } = props;
  const authorization = item.spec?.authorization;

  return {
    groupOrder: [
      { title: "Authorization", icon: ShieldCheck },
      { title: "Attributes", icon: Tags },
    ],
    items: [
      {
        label: "Members",
        primary: true,
        icon: Users,
        value: <GroupUsersLabel item={item} />,
      },
      {
        label: "Policies",
        primary: true,
        icon: ShieldCheck,
        value: (
          <span className="font-semibold tabular-nums">
            {(authorization?.policies.length ?? 0) +
              (authorization?.inlinePolicies.length ?? 0)}
          </span>
        ),
      },
      ...authorizationItems(authorization),
      ...attrsItems(item.spec?.attrs),
    ],
  };
};
