import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import {
  Activity,
  AlertTriangle,
  History,
  Loader2,
  RefreshCw,
  LockKeyhole,
} from "lucide-react";
import {
  HeroStat,
  HistoryList,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  ResourceActionButton,
  TonePill,
} from "@/components/ResourceLayout/overview";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { notifications } from "@mantine/notifications";
import { useMutation } from "@tanstack/react-query";
import {
  getSecretStorePresentation,
  getSecretStoreSyncState,
} from "./List";

const copied = (value?: string) =>
  value ? (
    <CopyText value={value} />
  ) : (
    <span className="text-slate-500">Not configured</span>
  );

export default (_props: { item: EnterpriseP.SecretStore }) => <></>;

const Sync = EnterpriseP.SecretStore_Status_Synchronization_State;

const isSyncing = (state?: EnterpriseP.SecretStore_Status_Synchronization_State) =>
  state === Sync.SYNC_REQUESTED || state === Sync.SYNCING;

const SynchronizeButton = (props: { item: EnterpriseP.SecretStore }) => {
  const presentation = getSecretStorePresentation(props.item);
  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientEnterprise().synchronizeSecretStore({
          secretStoreRef: getResourceRef(props.item),
        })
      ).response,
    onSuccess: () => {
      notifications.show({
        color: "green",
        message: "Synchronization requested",
      });
      invalidateResource(props.item);
      invalidateResourceList(props.item);
    },
    onError,
  });

  return (
    <ResourceActionButton
      tone="primary"
      icon={RefreshCw}
      loading={mutation.isPending || isSyncing(presentation.currentSync?.state)}
      onClick={() => mutation.mutate()}
    >
      Synchronize now
    </ResourceActionButton>
  );
};

const SecretStoreHero = (props: { item: EnterpriseP.SecretStore }) => {
  const { item } = props;
  const p = getSecretStorePresentation(item);
  const failed = p.currentSync?.state === Sync.FAILED;
  const syncing = isSyncing(p.currentSync?.state);
  const lastSync = p.lastSuccessfulSync;

  return (
    <OverviewHero
      tone={failed ? "danger" : syncing ? "info" : p.runtimeState.tone}
      icon={
        syncing || item.status?.state === EnterpriseP.SecretStore_Status_State.LOADING ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : (
          <LockKeyhole size={18} strokeWidth={2.2} />
        )
      }
      title={p.type}
      detail={
        <>
          {p.runtimeState.label}
          {syncing && " · synchronizing"}
          {lastSync?.completedAt && (
            <>
              {" · last synchronized "}
              <TimeAgo rfc3339={lastSync.completedAt} />
            </>
          )}
        </>
      }
      aside={
        p.currentSync && (
          <HeroStat
            label="Synchronization"
            tone={p.syncState.tone}
            value={p.syncState.label}
          />
        )
      }
    >
      {failed && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="The last synchronization failed.">
          The API does not expose a failure reason; inspect the component logs
          and synchronize again.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: EnterpriseP.SecretStore;
}): ResourceMainInfo => {
  const { item } = props;
  const presentation = getSecretStorePresentation(item);
  const type = item.spec?.type;
  const config: { label: string; value: React.ReactNode; span?: "full" }[] = [];

  if (type?.oneofKind === "kubernetes") {
    config.push({
      label: "Storage",
      value: "Cluster-native Kubernetes key storage",
      span: "full",
    });
  }
  if (type?.oneofKind === "hashicorpVault") {
    config.push(
      { label: "Vault address", value: copied(type.hashicorpVault.address), span: "full" },
      { label: "Role", value: copied(type.hashicorpVault.role) },
      { label: "Key", value: copied(type.hashicorpVault.key) },
    );
  }
  if (type?.oneofKind === "awsKeyManagementService") {
    config.push(
      { label: "Key ID", value: copied(type.awsKeyManagementService.keyID) },
      { label: "Region", value: type.awsKeyManagementService.region || "Not configured" },
      { label: "Role ARN", value: copied(type.awsKeyManagementService.roleARN), span: "full" },
    );
  }
  if (type?.oneofKind === "googleCloudKeyManagementService") {
    config.push(
      { label: "Project", value: copied(type.googleCloudKeyManagementService.project) },
      { label: "Location", value: type.googleCloudKeyManagementService.location || "Not configured" },
      { label: "Key ring", value: copied(type.googleCloudKeyManagementService.keyRing) },
      { label: "Key", value: copied(type.googleCloudKeyManagementService.key) },
    );
  }
  if (type?.oneofKind === "azureKeyVault") {
    config.push(
      { label: "Vault URL", value: copied(type.azureKeyVault.vaultURL), span: "full" },
      { label: "Client ID", value: copied(type.azureKeyVault.clientID) },
      { label: "Tenant ID", value: copied(type.azureKeyVault.tenantID) },
      { label: "Key", value: copied(type.azureKeyVault.key) },
    );
  }

  return {
    header: <SecretStoreHero item={item} />,
    actions: <SynchronizeButton item={item} />,
    groupOrder: [
      { title: "Backend", icon: LockKeyhole },
      { title: "Synchronization", icon: RefreshCw },
    ],
    items: [
      {
        label: "Type",
        primary: true,
        icon: LockKeyhole,
        value: <Label>{presentation.type}</Label>,
      },
      {
        label: "Runtime",
        primary: true,
        icon: Activity,
        value: (
          <TonePill tone={presentation.runtimeState.tone}>
            {presentation.runtimeState.label}
          </TonePill>
        ),
        hint:
          item.status?.type &&
          presentation.runtimeType !== "Unknown" &&
          presentation.runtimeType !== presentation.type
            ? `Backend ${presentation.runtimeType}`
            : undefined,
      },
      {
        label: "Last synchronized",
        primary: true,
        icon: History,
        value: presentation.lastSuccessfulSync?.completedAt ? (
          <TimeAgo rfc3339={presentation.lastSuccessfulSync.completedAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },
      {
        label: "Synchronizations",
        primary: true,
        icon: RefreshCw,
        value: (
          <span className="font-semibold tabular-nums">
            {presentation.allSynchronizations.length}
          </span>
        ),
      },
      ...config.map((entry) => ({ ...entry, group: "Backend" })),
      ...(presentation.allSynchronizations.length
        ? [
            {
              label: "History",
              group: "Synchronization",
              layout: "stacked" as const,
              value: (
                <HistoryList
                  entries={presentation.allSynchronizations
                    .slice(0, 6)
                    .map((entry, index) => {
                      const state = getSecretStoreSyncState(entry.state);
                      return {
                        key: `${entry.createdAt?.seconds ?? 0}-${entry.completedAt?.seconds ?? 0}-${index}`,
                        tone: state.tone,
                        label: `${index === 0 && presentation.currentSync === entry ? "Current · " : ""}${state.label}`,
                        meta: (
                          <>
                            {entry.createdAt && (
                              <span>
                                Started <TimeAgo rfc3339={entry.createdAt} />
                              </span>
                            )}
                            {entry.completedAt && (
                              <span>
                                Completed <TimeAgo rfc3339={entry.completedAt} />
                              </span>
                            )}
                          </>
                        ),
                      };
                    })}
                />
              ),
            },
          ]
        : [
            {
              label: "History",
              group: "Synchronization",
              value: <MutedValue>Never synchronized</MutedValue>,
            },
          ]),
    ],
  };
};
