import {
  DirectoryProvider,
  ListDirectoryProviderGroupOptions,
  ListDirectoryProviderUserOptions,
} from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel } from "@/components/ResourceList";

import { getDomain } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { match } from "ts-pattern";

export const getType = (svc: DirectoryProvider): string => {
  return match(svc.spec?.type.oneofKind)
    .with("scim", () => "SCIM")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: DirectoryProvider; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: DirectoryProvider }) => {
  const { item } = props;

  const client = getClientEnterprise();

  const qryUser = useQuery({
    queryKey: ["enterprise.directoryProviderUser.lst", item.metadata?.uid],
    queryFn: async () => {
      return await client.listDirectoryProviderUser(
        ListDirectoryProviderUserOptions.create({
          directoryProviderRef: getResourceRef(item),
        }),
      );
    },
  });

  const qryGroup = useQuery({
    queryKey: ["enterprise.directoryProviderGroup.lst", item.metadata?.uid],
    queryFn: async () => {
      return await client.listDirectoryProviderGroup(
        ListDirectoryProviderGroupOptions.create({
          directoryProviderRef: getResourceRef(item),
        }),
      );
    },
  });

  return (
    <div className="w-full mt-1 flex flex-row">
      <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>

      {qryUser.data && qryUser.data.response.listResponseMeta && (
        <ResourceListLabel label="Synced Users">{`${qryUser.data.response.listResponseMeta.totalCount}`}</ResourceListLabel>
      )}

      {qryGroup.data && qryGroup.data.response.listResponseMeta && (
        <ResourceListLabel label="Synced Groups">{`${qryGroup.data.response.listResponseMeta.totalCount}`}</ResourceListLabel>
      )}
    </div>
  );
};

export const ExtraComponent = (props: { item: DirectoryProvider }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};
