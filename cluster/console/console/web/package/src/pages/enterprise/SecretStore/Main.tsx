import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Alert } from "@mantine/core";
import { AlertTriangle, Loader2 } from "lucide-react";
import { twMerge } from "tailwind-merge";
import {
  getSecretStorePresentation,
  getSecretStoreSyncState,
} from "./List";

const copied = (value?: string) =>
  value ? (
    <CopyText value={value} />
  ) : (
    <span className="text-slate-400">Not configured</span>
  );

export default (_props: { item: EnterpriseP.SecretStore }) => <></>;

export const MainInfo = (props: {
  item: EnterpriseP.SecretStore;
}): ResourceMainInfo => {
  const { item } = props;
  const presentation = getSecretStorePresentation(item);
  const type = item.spec?.type;
  const items: NonNullable<ResourceMainInfo["items"]> = [
    {
      label: "Store type",
      value: <Label>{presentation.type}</Label>,
    },
    {
      label: "Runtime state",
      value: (
        <span
          className={twMerge(
            "inline-flex items-center gap-1.5 font-semibold",
            presentation.runtimeState.tone === "success"
              ? "text-emerald-600"
              : presentation.runtimeState.tone === "info"
                ? "text-blue-600"
                : "text-slate-500",
          )}
        >
          {item.status?.state === EnterpriseP.SecretStore_Status_State.LOADING && (
            <Loader2 size={13} className="animate-spin" />
          )}
          {presentation.runtimeState.label}
        </span>
      ),
    },
  ];

  if (
    item.status?.type &&
    presentation.runtimeType !== "Unknown" &&
    presentation.runtimeType !== presentation.type
  ) {
    items.push({
      label: "Runtime backend",
      value: <Label>{presentation.runtimeType}</Label>,
    });
  }

  if (type?.oneofKind === "kubernetes") {
    items.push({
      label: "Storage",
      value: "Cluster-native Kubernetes key storage",
      span: "full",
    });
  }
  if (type?.oneofKind === "hashicorpVault") {
    items.push(
      {
        label: "Vault address",
        value: copied(type.hashicorpVault.address),
        span: "full",
      },
      { label: "Role", value: copied(type.hashicorpVault.role) },
      { label: "Key", value: copied(type.hashicorpVault.key) },
    );
  }
  if (type?.oneofKind === "awsKeyManagementService") {
    items.push(
      { label: "Key ID", value: copied(type.awsKeyManagementService.keyID) },
      { label: "Region", value: type.awsKeyManagementService.region || "Not configured" },
      {
        label: "Role ARN",
        value: copied(type.awsKeyManagementService.roleARN),
        span: "full",
      },
    );
  }
  if (type?.oneofKind === "googleCloudKeyManagementService") {
    items.push(
      { label: "Project", value: copied(type.googleCloudKeyManagementService.project) },
      { label: "Location", value: type.googleCloudKeyManagementService.location || "Not configured" },
      { label: "Key ring", value: copied(type.googleCloudKeyManagementService.keyRing) },
      { label: "Key", value: copied(type.googleCloudKeyManagementService.key) },
    );
  }
  if (type?.oneofKind === "azureKeyVault") {
    items.push(
      { label: "Client ID", value: copied(type.azureKeyVault.clientID) },
      { label: "Tenant ID", value: copied(type.azureKeyVault.tenantID) },
      {
        label: "Vault URL",
        value: copied(type.azureKeyVault.vaultURL),
        span: "full",
      },
      { label: "Key", value: copied(type.azureKeyVault.key) },
    );
  }

  if (presentation.currentSync) {
    const sync = presentation.currentSync;
    items.push({
      label: "Current synchronization",
      span: "full",
      value: (
        <div className="flex flex-wrap items-center gap-2">
          {(sync.state ===
            EnterpriseP.SecretStore_Status_Synchronization_State.SYNC_REQUESTED ||
            sync.state ===
              EnterpriseP.SecretStore_Status_Synchronization_State.SYNCING) && (
            <Loader2 size={13} className="animate-spin text-blue-500" />
          )}
          <span
            className={twMerge(
              "font-semibold",
              presentation.syncState.tone === "success"
                ? "text-emerald-600"
                : presentation.syncState.tone === "danger"
                  ? "text-red-600"
                  : "text-blue-600",
            )}
          >
            {presentation.syncState.label}
          </span>
          {sync.createdAt && (
            <span className="text-[0.68rem] font-semibold text-slate-400">
              Started <TimeAgo rfc3339={sync.createdAt} />
            </span>
          )}
          {sync.completedAt && (
            <span className="text-[0.68rem] font-semibold text-slate-400">
              Completed <TimeAgo rfc3339={sync.completedAt} />
            </span>
          )}
        </div>
      ),
    });
  }

  if (presentation.allSynchronizations.length) {
    items.push({
      label: "Synchronization history",
      span: "full",
      value: (
        <div className="w-full overflow-hidden rounded-lg border border-slate-200">
          {presentation.allSynchronizations.slice(0, 6).map((entry, index) => {
            const state = getSecretStoreSyncState(entry.state);
            return (
              <div
                key={`${entry.createdAt?.seconds ?? 0}-${entry.completedAt?.seconds ?? 0}-${index}`}
                className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0"
              >
                <span
                  className={twMerge(
                    "text-[0.7rem] font-bold",
                    state.tone === "success"
                      ? "text-emerald-600"
                      : state.tone === "danger"
                        ? "text-red-600"
                        : "text-blue-600",
                  )}
                >
                  {index === 0 && presentation.currentSync === entry
                    ? "Current · "
                    : ""}
                  {state.label}
                </span>
                <div className="flex gap-3 text-[0.67rem] font-semibold text-slate-400">
                  {entry.createdAt && (
                    <span>Started <TimeAgo rfc3339={entry.createdAt} /></span>
                  )}
                  {entry.completedAt && (
                    <span>Completed <TimeAgo rfc3339={entry.completedAt} /></span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ),
    });
  }

  if (
    presentation.currentSync?.state ===
    EnterpriseP.SecretStore_Status_Synchronization_State.FAILED
  ) {
    items.push({
      label: "Synchronization failure",
      span: "full",
      value: (
        <Alert color="red" icon={<AlertTriangle size={14} />}>
          The API does not expose a failure reason. Inspect component logs for
          more information.
        </Alert>
      ),
    });
  }

  return { items };
};
