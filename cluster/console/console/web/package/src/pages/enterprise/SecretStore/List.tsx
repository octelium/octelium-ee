import {
  SecretStore,
  SecretStore_Status_State,
  SecretStore_Status_Synchronization,
  SecretStore_Status_Synchronization_State,
  SecretStore_Status_Type,
} from "@/apis/enterprisev1/enterprisev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { match } from "ts-pattern";

export const getSecretStoreType = (item: SecretStore): string =>
  match(item.spec?.type.oneofKind)
    .with("kubernetes", () => "Kubernetes")
    .with("hashicorpVault", () => "HashiCorp Vault")
    .with("awsKeyManagementService", () => "AWS KMS")
    .with("azureKeyVault", () => "Azure Key Vault")
    .with("googleCloudKeyManagementService", () => "Google Cloud KMS")
    .otherwise(() => "Not configured");

export const getSecretStoreRuntimeType = (type?: SecretStore_Status_Type) =>
  match(type)
    .with(SecretStore_Status_Type.TYPE_AZURE_KEY_VAULT, () => "Azure Key Vault")
    .with(SecretStore_Status_Type.TYPE_HASHICORP_VAULT, () => "HashiCorp Vault")
    .with(SecretStore_Status_Type.TYPE_GCP_KMS, () => "Google Cloud KMS")
    .with(SecretStore_Status_Type.TYPE_AWS_KMS, () => "AWS KMS")
    .with(SecretStore_Status_Type.KUBERNETES, () => "Kubernetes")
    .otherwise(() => "Unknown");

export const getSecretStoreState = (state?: SecretStore_Status_State) =>
  match(state)
    .with(SecretStore_Status_State.OK, () => ({
      label: "Ready",
      tone: "success" as const,
    }))
    .with(SecretStore_Status_State.LOADING, () => ({
      label: "Loading",
      tone: "info" as const,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "neutral" as const }));

export const getSecretStoreSyncState = (
  state?: SecretStore_Status_Synchronization_State,
) =>
  match(state)
    .with(SecretStore_Status_Synchronization_State.SYNC_REQUESTED, () => ({
      label: "Sync requested",
      tone: "info" as const,
    }))
    .with(SecretStore_Status_Synchronization_State.SYNCING, () => ({
      label: "Synchronizing",
      tone: "info" as const,
    }))
    .with(SecretStore_Status_Synchronization_State.SUCCESS, () => ({
      label: "Successful",
      tone: "success" as const,
    }))
    .with(SecretStore_Status_Synchronization_State.FAILED, () => ({
      label: "Failed",
      tone: "danger" as const,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "neutral" as const }));

const syncTime = (entry: SecretStore_Status_Synchronization) =>
  Number((entry.completedAt ?? entry.createdAt)?.seconds ?? 0);

export const getSecretStorePresentation = (item: SecretStore) => {
  const currentSync = item.status?.synchronization;
  const previousSyncs = item.status?.lastSynchronizations ?? [];
  const allSynchronizations = [
    ...(currentSync ? [currentSync] : []),
    ...previousSyncs,
  ];
  const lastSuccessfulSync = allSynchronizations
    .filter(
      (entry) =>
        entry.state === SecretStore_Status_Synchronization_State.SUCCESS,
    )
    .sort((a, b) => syncTime(b) - syncTime(a))[0];

  return {
    type: getSecretStoreType(item),
    runtimeType: getSecretStoreRuntimeType(item.status?.type),
    runtimeState: getSecretStoreState(item.status?.state),
    currentSync,
    previousSyncs,
    allSynchronizations,
    lastSuccessfulSync,
    syncState: getSecretStoreSyncState(currentSync?.state),
  };
};

export const LabelComponent = (props: { item: SecretStore }) => {
  const presentation = getSecretStorePresentation(props.item);
  const isRunning =
    presentation.currentSync?.state ===
      SecretStore_Status_Synchronization_State.SYNC_REQUESTED ||
    presentation.currentSync?.state ===
      SecretStore_Status_Synchronization_State.SYNCING;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">{presentation.type}</ResourceListLabel>
      <ResourceListLabel label="State">
        {props.item.status?.state === SecretStore_Status_State.LOADING ? (
          <Loader2 size={11} className="animate-spin text-blue-500" />
        ) : props.item.status?.state === SecretStore_Status_State.OK ? (
          <CheckCircle2 size={11} className="text-emerald-500" />
        ) : null}
        {presentation.runtimeState.label}
      </ResourceListLabel>
      {presentation.currentSync && (
        <ResourceListLabel label="Synchronization">
          {isRunning ? (
            <Loader2 size={11} className="animate-spin text-blue-500" />
          ) : presentation.currentSync.state ===
            SecretStore_Status_Synchronization_State.FAILED ? (
            <AlertTriangle size={11} className="text-red-500" />
          ) : presentation.currentSync.state ===
            SecretStore_Status_Synchronization_State.SUCCESS ? (
            <CheckCircle2 size={11} className="text-emerald-500" />
          ) : null}
          {presentation.syncState.label}
        </ResourceListLabel>
      )}
      {presentation.lastSuccessfulSync?.completedAt && (
        <ResourceListLabel label="Last synchronized">
          <TimeAgo rfc3339={presentation.lastSuccessfulSync.completedAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};
