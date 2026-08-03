import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import AccessLogViewer from "@/components/AccessLogViewer";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { PanelTop, Shield } from "lucide-react";

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

  return {
    items: [
      {
        label: "Related resources",
        value: <NamespaceServicesLabel item={item} />,
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
