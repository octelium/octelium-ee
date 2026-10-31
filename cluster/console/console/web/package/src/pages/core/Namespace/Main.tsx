import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { PanelTop, ShieldCheck, Tags } from "lucide-react";
import {
  attrsItems,
  authorizationItems,
} from "@/components/ResourceLayout/overviewItems";

export const AccessLog = (props: { item: CoreC.Namespace }) => {
  return <AccessLogViewer namespaceRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Namespace }) => {
  const { item } = props;
  return <div className="w-full"></div>;
};

export const NamespaceServicesLabel = (props: { item: CoreC.Namespace }) => {
  const itemName = props.item.metadata!.name;
  const qryServices = useQuery({
    queryKey: ["selectServiceComponent", itemName],
    queryFn: () =>
      getClientCore().listService(
        CoreC.ListServiceOptions.create({
          namespaceRef: getResourceRef(props.item),
        }),
      ).response,
  });

  return (
    <ResourceListLabel
      label="Services"
      to={`/core/services?namespaceRef.name=${encodeURIComponent(itemName)}`}
    >
      <PanelTop size={12} strokeWidth={2.5} />
      {qryServices.data?.listResponseMeta?.totalCount?.toLocaleString() ?? "…"}
    </ResourceListLabel>
  );
};

export const MainInfo = (
  props: { item: CoreC.Namespace },
): ResourceMainInfo => {
  const { item } = props;
  const authorization = item.spec?.authorization;

  return {
    groupOrder: [
      { title: "Authorization", icon: ShieldCheck },
      { title: "Attributes", icon: Tags },
    ],
    items: [
      {
        label: "Services",
        primary: true,
        icon: PanelTop,
        value: <NamespaceServicesLabel item={item} />,
      },
      {
        label: "Policies",
        primary: true,
        icon: ShieldCheck,
        hint: "Applied to every Service of the Namespace",
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
