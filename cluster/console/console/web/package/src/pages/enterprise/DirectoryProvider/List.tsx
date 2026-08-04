import {
  DirectoryProvider,
  DirectoryProvider_Status_Synchronization,
  DirectoryProvider_Status_Synchronization_State,
  ListDirectoryProviderGroupOptions,
  ListDirectoryProviderUserOptions,
} from "@/apis/enterprisev1/enterprisev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { getDomain } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { match } from "ts-pattern";

export const getType = (item: DirectoryProvider): string =>
  match(item.spec?.type.oneofKind)
    .with("scim", () => "SCIM")
    .with("googleWorkspace", () => "Google Workspace")
    .with("keycloak", () => "Keycloak")
    .otherwise(() => "Not configured");

export const isSyncable = (item: DirectoryProvider): boolean =>
  item.spec?.type.oneofKind === "googleWorkspace" ||
  item.spec?.type.oneofKind === "keycloak";

export const getSyncStateMeta = (
  state?: DirectoryProvider_Status_Synchronization_State,
) =>
  match(state)
    .with(
      DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED,
      () => ({ label: "Sync requested", tone: "warning" as const }),
    )
    .with(DirectoryProvider_Status_Synchronization_State.SYNCING, () => ({
      label: "Synchronizing",
      tone: "info" as const,
    }))
    .with(DirectoryProvider_Status_Synchronization_State.SUCCESS, () => ({
      label: "Successful",
      tone: "success" as const,
    }))
    .with(DirectoryProvider_Status_Synchronization_State.FAILED, () => ({
      label: "Failed",
      tone: "danger" as const,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "neutral" as const }));

const syncTime = (item: DirectoryProvider_Status_Synchronization) => {
  const time = item.completedAt ?? item.createdAt;
  return time ? Number(time.seconds) : 0;
};

export const getDirectoryProviderPresentation = (item: DirectoryProvider) => {
  const currentSync = item.status?.synchronization;
  const previousSyncs = item.status?.lastSynchronizations ?? [];
  const allSynchronizations = [
    ...(currentSync ? [currentSync] : []),
    ...previousSyncs,
  ];
  const lastSuccessfulSync = allSynchronizations
    .filter(
      (entry) =>
        entry.state ===
        DirectoryProvider_Status_Synchronization_State.SUCCESS,
    )
    .sort((a, b) => syncTime(b) - syncTime(a))[0];

  return {
    type: getType(item),
    isScim: item.spec?.type.oneofKind === "scim",
    isSyncable: isSyncable(item),
    currentSync,
    previousSyncs,
    allSynchronizations,
    lastSuccessfulSync,
    syncMeta: getSyncStateMeta(currentSync?.state),
  };
};

export const DirectoryProviderInventoryLabels = (props: {
  item: DirectoryProvider;
}) => {
  const common = CommonListOptions.create({ itemsPerPage: 1 });
  const identity = props.item.metadata?.uid || props.item.metadata?.name;
  const isSynchronizing =
    props.item.status?.synchronization?.state ===
      DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED ||
    props.item.status?.synchronization?.state ===
      DirectoryProvider_Status_Synchronization_State.SYNCING;
  const users = useQuery({
    queryKey: ["enterprise.directoryProviderUser.count", identity],
    queryFn: async () =>
      (
        await getClientEnterprise().listDirectoryProviderUser(
          ListDirectoryProviderUserOptions.create({
            common,
            directoryProviderRef: getResourceRef(props.item),
          }),
        )
      ).response,
    enabled: !!identity,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchInterval: isSynchronizing ? 5_000 : false,
  });
  const groups = useQuery({
    queryKey: ["enterprise.directoryProviderGroup.count", identity],
    queryFn: async () =>
      (
        await getClientEnterprise().listDirectoryProviderGroup(
          ListDirectoryProviderGroupOptions.create({
            common,
            directoryProviderRef: getResourceRef(props.item),
          }),
        )
      ).response,
    enabled: !!identity,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchInterval: isSynchronizing ? 5_000 : false,
  });

  return (
    <>
      <ResourceListLabel label="Synced users">
        {users.isError
          ? "Unavailable"
          : users.data?.listResponseMeta?.totalCount?.toLocaleString() ?? "…"}
      </ResourceListLabel>
      <ResourceListLabel label="Synced groups">
        {groups.isError
          ? "Unavailable"
          : groups.data?.listResponseMeta?.totalCount?.toLocaleString() ?? "…"}
      </ResourceListLabel>
    </>
  );
};

const ItemDetails = (props: { item: DirectoryProvider; domain: string }) => {
  const { item } = props;
  return <div></div>;
};

export const LabelComponent = (props: { item: DirectoryProvider }) => {
  const p = getDirectoryProviderPresentation(props.item);
  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">{p.type}</ResourceListLabel>
      {props.item.spec?.isDisabled && (
        <ResourceListLabel>Disabled</ResourceListLabel>
      )}
      {p.isScim ? (
        <ResourceListLabel>SCIM endpoint</ResourceListLabel>
      ) : p.currentSync ? (
        <ResourceListLabel label="Synchronization">
          {p.currentSync.state ===
          DirectoryProvider_Status_Synchronization_State.SYNCING ||
          p.currentSync.state ===
            DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED ? (
            <Loader2 size={11} className="animate-spin text-blue-500" />
          ) : p.currentSync.state ===
            DirectoryProvider_Status_Synchronization_State.FAILED ? (
            <AlertTriangle size={11} className="text-red-500" />
          ) : p.currentSync.state ===
            DirectoryProvider_Status_Synchronization_State.SUCCESS ? (
            <CheckCircle2 size={11} className="text-emerald-500" />
          ) : null}
          {p.syncMeta.label}
        </ResourceListLabel>
      ) : null}
      {p.lastSuccessfulSync?.completedAt && (
        <ResourceListLabel label="Last synchronized">
          <TimeAgo rfc3339={p.lastSuccessfulSync.completedAt} />
        </ResourceListLabel>
      )}
      <DirectoryProviderInventoryLabels item={props.item} />
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: DirectoryProvider }) => {
  const domain = getDomain();
  return <ItemDetails item={props.item} domain={domain} />;
};
