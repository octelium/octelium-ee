import { Namespace } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { ListServiceOptions } from "@/apis/corev1/corev1";
import { GetNamespaceSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { SummaryItemCount, SummaryItemCountWrap } from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientCore, getClientVisibilityCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { PanelTop, Shield } from "lucide-react";

const ItemDetails = (props: { item: Namespace; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Namespace }) => {
  const { item } = props;

  const { isLoading, isSuccess, data } = useQuery({
    queryKey: ["selectServiceComponent", item.metadata!.name],
    queryFn: async () => {
      return await getClientCore().listService(
        ListServiceOptions.create({ namespaceRef: getResourceRef(item) }),
      );
    },
  });

  return (
    <ResourceListLabelWrap>
      {isSuccess &&
        data &&
        data.response.listResponseMeta &&
        data.response.listResponseMeta.totalCount > 0 && (
          <ResourceListLabel
            to={toURLWithQry(`/core/services`, {
              "namespaceRef.name": item.metadata!.name,
            })}
          >
            <PanelTop size={14} className="mr-1" />
            {data.response.listResponseMeta.totalCount} Services
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

export const ExtraComponent = (props: { item: Namespace }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetNamespaceSummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/namespaces`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = () => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Namespace"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getNamespaceSummary(
        {},
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return <DoSummary resp={qry.data} />;
};
