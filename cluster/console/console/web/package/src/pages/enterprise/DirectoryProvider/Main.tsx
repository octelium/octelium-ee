import * as EnterpriseC from "@/apis/enterprisev1/enterprisev1";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import TimeAgo from "@/components/TimeAgo";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getDomain, onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import {
  Alert,
  Button,
  CopyButton,
  Modal,
  PasswordInput,
  Switch,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  AlertTriangle,
  Check,
  Copy,
  KeyRound,
  Loader2,
  RefreshCw,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import {
  DirectoryProviderInventoryLabels,
  getDirectoryProviderPresentation,
  getSyncStateMeta,
} from "./List";

const formatDuration = (duration?: Duration) => {
  const type = duration?.type;
  if (type?.oneofKind === "milliseconds") return `${type.milliseconds.toLocaleString()} milliseconds`;
  if (type?.oneofKind === "seconds") return `${type.seconds.toLocaleString()} seconds`;
  if (type?.oneofKind === "minutes") return `${type.minutes.toLocaleString()} minutes`;
  if (type?.oneofKind === "hours") return `${type.hours.toLocaleString()} hours`;
  if (type?.oneofKind === "days") return `${type.days.toLocaleString()} days`;
  if (type?.oneofKind === "weeks") return `${type.weeks.toLocaleString()} weeks`;
  if (type?.oneofKind === "months") return `${type.months.toLocaleString()} months`;
  return "Not configured";
};

const SynchronizeButton = (props: { item: EnterpriseC.DirectoryProvider }) => {
  const p = getDirectoryProviderPresentation(props.item);
  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientEnterprise().synchronizeDirectoryProvider({
          directoryProviderRef: getResourceRef(props.item),
        })
      ).response,
    onSuccess: () => {
      toast.success("Synchronization requested");
      invalidateResource(props.item);
      invalidateResourceList(props.item);
    },
    onError,
  });
  const isRunning =
    p.currentSync?.state ===
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNCING ||
    p.currentSync?.state ===
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED;

  return (
    <Button
      type="button"
      color="dark"
      size="sm"
      leftSection={<RefreshCw size={13} />}
      disabled={props.item.spec?.isDisabled || !p.isSyncable}
      loading={mutation.isPending || isRunning}
      title={
        props.item.spec?.isDisabled
          ? "Enable the provider before synchronizing"
          : undefined
      }
      onClick={() => mutation.mutate()}
    >
      Synchronize now
    </Button>
  );
};

const ScimCredential = (props: { item: EnterpriseC.DirectoryProvider }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [confirmed, setConfirmed] = React.useState(false);
  const [token, setToken] = React.useState("");
  const handleClose = () => {
    setConfirmed(false);
    setToken("");
    close();
  };
  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientEnterprise().generateDirectoryProviderCredential({
          directoryProviderRef: getResourceRef(props.item),
          mode: EnterpriseC.GenerateDirectoryProviderCredentialRequest_Mode
            .BEARER,
        })
      ).response,
    onSuccess: (response) => {
      if (response.type.oneofKind === "bearer") {
        setToken(response.type.bearer.accessToken);
      }
      invalidateResource(props.item);
      invalidateResourceList(props.item);
    },
    onError,
  });

  return (
    <>
      <Button
        type="button"
        variant="default"
        size="sm"
        leftSection={<KeyRound size={13} />}
        disabled={props.item.spec?.isDisabled}
        onClick={open}
      >
        Generate access token
      </Button>
      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        size="lg"
        withCloseButton={false}
        padding={0}
        styles={{ content: { borderRadius: 14, overflow: "hidden" } }}
      >
        <div className="bg-white">
          <header className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <div className="flex items-center gap-2.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900 text-white">
                <KeyRound size={15} />
              </span>
              <div>
                <h2 className="text-[0.84rem] font-bold text-slate-900">
                  SCIM bearer token
                </h2>
                <p className="text-[0.67rem] font-semibold text-slate-400">
                  {props.item.metadata?.displayName || props.item.metadata?.name}
                </p>
              </div>
            </div>
            <Button type="button" variant="subtle" color="gray" size="compact-xs" onClick={handleClose}>
              <X size={14} />
            </Button>
          </header>
          <div className="space-y-4 px-5 py-5">
            {token ? (
              <>
                <Alert color="green" title="Copy this token now">
                  This credential is shown only in this dialog. Store it in
                  your identity provider before closing.
                </Alert>
                <div className="flex items-end gap-2">
                  <PasswordInput
                    className="flex-1"
                    label="Access token"
                    value={token}
                    readOnly
                  />
                  <CopyButton value={token}>
                    {({ copied, copy }) => (
                      <Button
                        type="button"
                        variant="default"
                        leftSection={copied ? <Check size={13} /> : <Copy size={13} />}
                        onClick={copy}
                      >
                        {copied ? "Copied" : "Copy"}
                      </Button>
                    )}
                  </CopyButton>
                </div>
              </>
            ) : (
              <>
                <Alert color="amber" icon={<AlertTriangle size={15} />} title="The previous token will stop working">
                  Generating a new bearer token immediately invalidates the
                  current SCIM credential.
                </Alert>
                <Switch
                  checked={confirmed}
                  label="I understand that the previous token will be invalidated"
                  onChange={(event) => setConfirmed(event.currentTarget.checked)}
                />
              </>
            )}
          </div>
          <footer className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
            <Button type="button" variant="default" size="sm" disabled={mutation.isPending} onClick={handleClose}>
              {token ? "Done" : "Cancel"}
            </Button>
            {!token && (
              <Button
                type="button"
                color="dark"
                size="sm"
                disabled={!confirmed}
                loading={mutation.isPending}
                leftSection={<KeyRound size={13} />}
                onClick={() => mutation.mutate()}
              >
                Generate token
              </Button>
            )}
          </footer>
        </div>
      </Modal>
    </>
  );
};

export default (_props: { item: EnterpriseC.DirectoryProvider }) => <></>;

export const MainInfo = (props: {
  item: EnterpriseC.DirectoryProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getDirectoryProviderPresentation(item);
  const mutationUpdate = useUpdateResource();
  const type = item.spec?.type;

  return {
    items: [
      { label: "Type", value: <Label>{p.type}</Label> },
      {
        label: "Active",
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="active"
            showComponent={
              <span className={twMerge("text-sm font-semibold", item.spec?.isDisabled ? "text-red-500" : "text-emerald-600")}>
                {item.spec?.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec?.isDisabled}
                onChange={(event) => {
                  item.spec!.isDisabled = !event.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },
      ...(item.status?.id
        ? [{ label: "Provider ID", value: <CopyText value={item.status.id} /> }]
        : []),
      ...(p.isScim && item.status?.id
        ? [
            {
              label: "SCIM endpoint",
              value: <CopyText value={`https://dirsync.octelium.${getDomain()}/scim/${item.status.id}`} />,
              span: "full" as const,
            },
          ]
        : []),
      ...(item.status?.userRef?.name
        ? [{ label: "Runtime user", value: <ResourceListLabel itemRef={item.status.userRef} /> }]
        : []),
      ...(item.status?.sessionRef?.name
        ? [{ label: "Runtime session", value: <ResourceListLabel itemRef={item.status.sessionRef} /> }]
        : []),
      ...(p.currentSync
        ? [
            {
              label: "Current synchronization",
              value: (
                <div className="flex flex-wrap items-center gap-2">
                  {(p.currentSync.state === EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNCING ||
                    p.currentSync.state === EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED) && (
                    <Loader2 size={13} className="animate-spin text-blue-500" />
                  )}
                  <span className={twMerge("font-semibold", p.syncMeta.tone === "success" ? "text-emerald-600" : p.syncMeta.tone === "danger" ? "text-red-600" : "text-blue-600")}>
                    {p.syncMeta.label}
                  </span>
                  {p.currentSync.createdAt && <span className="text-[0.68rem] font-semibold text-slate-400">Started <TimeAgo rfc3339={p.currentSync.createdAt} /></span>}
                  {p.currentSync.completedAt && <span className="text-[0.68rem] font-semibold text-slate-400">Completed <TimeAgo rfc3339={p.currentSync.completedAt} /></span>}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
      {
        label: "Directory inventory",
        value: <div className="flex flex-wrap gap-1"><DirectoryProviderInventoryLabels item={item} /></div>,
        span: "full" as const,
      },
      ...(type?.oneofKind === "googleWorkspace"
        ? [
            { label: "Customer ID", value: <CopyText value={type.googleWorkspace.customer} /> },
            { label: "Impersonated administrator", value: <CopyText value={type.googleWorkspace.impersonateSubject} /> },
            ...(type.googleWorkspace.serviceAccount?.type.oneofKind === "fromSecret" && type.googleWorkspace.serviceAccount.type.fromSecret
              ? [{ label: "Service account Secret", value: <ResourceListLabel itemRef={ObjectReference.create({ apiVersion: "enterprise/v1", kind: "Secret", name: type.googleWorkspace.serviceAccount.type.fromSecret })} /> }]
              : []),
            { label: "Polling interval", value: formatDuration(type.googleWorkspace.polling?.interval) },
          ]
        : []),
      ...(type?.oneofKind === "keycloak"
        ? [
            { label: "Keycloak URL", value: <CopyText value={type.keycloak.url} />, span: "full" as const },
            { label: "Realm", value: type.keycloak.realm },
            { label: "Client ID", value: <CopyText value={type.keycloak.clientID} /> },
            ...(type.keycloak.clientSecret?.type.oneofKind === "fromSecret" && type.keycloak.clientSecret.type.fromSecret
              ? [{ label: "Client Secret", value: <ResourceListLabel itemRef={ObjectReference.create({ apiVersion: "enterprise/v1", kind: "Secret", name: type.keycloak.clientSecret.type.fromSecret })} /> }]
              : []),
            { label: "Polling interval", value: formatDuration(type.keycloak.polling?.interval) },
            ...(type.keycloak.insecureSkipVerify
              ? [{ label: "TLS security", value: <span className="font-semibold text-red-600">Certificate verification disabled</span> }]
              : []),
          ]
        : []),
      ...(p.allSynchronizations.length
        ? [
            {
              label: "Synchronization history",
              span: "full" as const,
              value: (
                <div className="w-full overflow-hidden rounded-lg border border-slate-200">
                  {p.allSynchronizations.slice(0, 6).map((entry, index) => {
                    const meta = getSyncStateMeta(entry.state);
                    return (
                      <div key={`${entry.createdAt?.seconds ?? 0}-${index}`} className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0">
                        <span className={twMerge("text-[0.7rem] font-bold", meta.tone === "success" ? "text-emerald-600" : meta.tone === "danger" ? "text-red-600" : "text-blue-600")}>
                          {index === 0 && p.currentSync === entry ? "Current · " : ""}{meta.label}
                        </span>
                        <div className="flex gap-3 text-[0.67rem] font-semibold text-slate-400">
                          {entry.createdAt && <span>Started <TimeAgo rfc3339={entry.createdAt} /></span>}
                          {entry.completedAt && <span>Completed <TimeAgo rfc3339={entry.completedAt} /></span>}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ),
            },
          ]
        : []),
      ...(p.currentSync?.state === EnterpriseC.DirectoryProvider_Status_Synchronization_State.FAILED
        ? [{ label: "Synchronization failure", value: <div className="flex items-start gap-2 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-[0.7rem] font-semibold text-red-700"><AlertTriangle size={13} className="mt-0.5 shrink-0" />The API does not expose a failure reason. Inspect component logs for details.</div>, span: "full" as const }]
        : []),
      {
        label: "Actions",
        value: p.isScim ? <ScimCredential item={item} /> : p.isSyncable ? <SynchronizeButton item={item} /> : <span className="text-slate-400">No actions available</span>,
        span: "full" as const,
      },
    ],
  };
};
