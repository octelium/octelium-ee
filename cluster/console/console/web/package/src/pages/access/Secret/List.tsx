import { Integration, Secret } from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  GetSecretSummaryResponse,
  ListIntegrationOptions,
} from "@/apis/visibilityv1/access/vaccessv1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { KeyRound } from "lucide-react";
import { integrationSecretNames } from "../Integration/utils";

const USAGE_ITEMS = 200;

export const useSecretUsage = (name?: string) => {
  const query = useQuery({
    queryKey: ["access", "integration", "secretUsage"],
    queryFn: async () =>
      (
        await getClientVisibilityAccess().listIntegration(
          ListIntegrationOptions.create({
            common: CommonListOptions.create({ itemsPerPage: USAGE_ITEMS }),
          }),
        )
      ).response,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  const items = (query.data?.items ?? []).filter(
    (integration: Integration) =>
      !!name && integrationSecretNames(integration).includes(name),
  );

  return {
    isError: query.isError,
    isLoading: query.isLoading,
    refs: items.map((integration) => getResourceRef(integration)),
  };
};

export const SecretUsageLabels = (props: { item: Secret }) => {
  const usage = useSecretUsage(props.item.metadata?.name);

  if (usage.isError) {
    return <ResourceListLabel label="Used by">Unavailable</ResourceListLabel>;
  }
  if (usage.isLoading) {
    return <ResourceListLabel label="Used by">…</ResourceListLabel>;
  }
  if (usage.refs.length === 0) {
    return <ResourceListLabel label="Used by">No Integration</ResourceListLabel>;
  }

  return (
    <>
      {usage.refs.map((itemRef: ObjectReference) => (
        <ResourceListLabel key={itemRef.uid} label="Used by" itemRef={itemRef} />
      ))}
    </>
  );
};

export const LabelComponent = (props: { item: Secret }) => (
  <ResourceListLabelWrap>
    <SecretUsageLabels item={props.item} />
  </ResourceListLabelWrap>
);

export const ExtraComponent = (props: { item: Secret }) => {
  return <div></div>;
};

const DoSummary = ({ resp }: { resp: GetSecretSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/secrets" icon={KeyRound}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Secret"],
    queryFn: async () => {
      const { response } = await getClientVisibilityAccess().getSecretSummary({});
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
