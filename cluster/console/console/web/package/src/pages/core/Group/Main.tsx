import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ListUserOptions } from "@/apis/visibilityv1/core/vcorev1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { Shield, Users } from "lucide-react";

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

  return {
    items: [
      {
        label: "Related resources",
        value: <GroupUsersLabel item={item} />,
        span: "full",
      },
      ...(item.spec?.authorization?.policies.length
        ? [
            {
              label: "Policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.policies.map((policy) => (
                    <ResourceListLabel
                      key={policy}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Policy",
                        name: policy,
                      })}
                    />
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
      ...(item.spec?.authorization?.inlinePolicies.length
        ? [
            {
              label: "Inline policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.inlinePolicies.map(
                    (policy, index) => (
                      <ResourceListLabel
                        key={`${policy.name}-${index}`}
                        label="Inline policy"
                      >
                        <Shield size={12} strokeWidth={2.5} />
                        {policy.name || `Inline policy ${index + 1}`}
                      </ResourceListLabel>
                    ),
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
