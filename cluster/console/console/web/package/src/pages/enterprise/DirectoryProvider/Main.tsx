import * as EnterpriseC from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import TimeAgo from "@/components/TimeAgo";
import { ResourceInfoIcon, ResourceMainInfo } from "@/pages/utils/types";
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
  ArrowDownToLine,
  Ban,
  Check,
  Copy,
  Cpu,
  History,
  KeyRound,
  Loader2,
  RefreshCw,
  Timer,
  Users,
  X,
} from "lucide-react";
import { FaGoogle } from "react-icons/fa6";
import {
  formatDuration,
  HeroStat,
  HistoryList,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  ResourceActionButton,
  TonePill,
} from "@/components/ResourceLayout/overview";
import * as React from "react";
import { notifications } from "@mantine/notifications";
import {
  DirectoryProviderInventoryLabels,
  getDirectoryProviderPresentation,
  getSyncStateMeta,
} from "./List";

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
      notifications.show({
        color: "green",
        message: "Synchronization requested",
      });
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
    <ResourceActionButton
      tone="primary"
      icon={RefreshCw}
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
    </ResourceActionButton>
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
      <ResourceActionButton
        tone="primary"
        icon={KeyRound}
        disabled={props.item.spec?.isDisabled}
        onClick={open}
      >
        Generate token
      </ResourceActionButton>
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
                <h2 className="text-sm font-bold text-slate-900">
                  SCIM bearer token
                </h2>
                <p className="text-xs font-normal text-slate-500">
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
                color="ink"
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

const Sync = EnterpriseC.DirectoryProvider_Status_Synchronization_State;

const typeIcon = (item: EnterpriseC.DirectoryProvider): ResourceInfoIcon =>
  item.spec?.type.oneofKind === "googleWorkspace"
    ? (FaGoogle as ResourceInfoIcon)
    : item.spec?.type.oneofKind === "keycloak"
      ? KeyRound
      : ArrowDownToLine;

const pollingOf = (item: EnterpriseC.DirectoryProvider) => {
  const type = item.spec?.type;
  if (type?.oneofKind === "googleWorkspace") return type.googleWorkspace.polling?.interval;
  if (type?.oneofKind === "keycloak") return type.keycloak.polling?.interval;
  return undefined;
};

const DirectoryHero = (props: { item: EnterpriseC.DirectoryProvider }) => {
  const { item } = props;
  const p = getDirectoryProviderPresentation(item);
  const syncing =
    p.currentSync?.state === Sync.SYNCING ||
    p.currentSync?.state === Sync.SYNC_REQUESTED;
  const failed = p.currentSync?.state === Sync.FAILED;
  const polling = pollingOf(item);

  return (
    <OverviewHero
      tone={
        item.spec?.isDisabled || failed ? "danger" : syncing ? "info" : "success"
      }
      icon={
        syncing ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : (
          React.createElement(typeIcon(item), { size: 18, strokeWidth: 2.2 })
        )
      }
      title={`${p.type} directory`}
      detail={
        p.isScim ? (
          "Your identity provider pushes Users and Groups over SCIM 2.0."
        ) : (
          <>
            {polling
              ? `Synchronizes Users and Groups every ${formatDuration(polling)}`
              : "Synchronizes Users and Groups"}
            {p.lastSuccessfulSync?.completedAt && (
              <>
                {" · last synchronized "}
                <TimeAgo rfc3339={p.lastSuccessfulSync.completedAt} />
              </>
            )}
          </>
        )
      }
      meta={
        p.isScim && item.status?.id ? (
          <span className="inline-flex max-w-full items-center gap-1.5 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs leading-4">
            <span className="shrink-0 font-semibold text-slate-500">
              SCIM endpoint
            </span>
            <span aria-hidden="true" className="h-3 w-px shrink-0 bg-slate-200" />
            <span className="min-w-0 font-medium text-slate-800 [overflow-wrap:anywhere]">
              <CopyText
                value={`https://dirsync.octelium.${getDomain()}/scim/${item.status.id}`}
              />
            </span>
          </span>
        ) : undefined
      }
      aside={
        p.currentSync && (
          <HeroStat
            label="Synchronization"
            tone={p.syncMeta.tone}
            value={p.syncMeta.label}
          />
        )
      }
    >
      {item.spec?.isDisabled && (
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          {p.isScim
            ? "SCIM requests from the identity provider are rejected."
            : "The directory is not synchronized."}
        </OverviewNotice>
      )}
      {failed && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="The last synchronization failed.">
          The API does not expose a failure reason; inspect the component logs
          for details.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: EnterpriseC.DirectoryProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getDirectoryProviderPresentation(item);
  const type = item.spec?.type;
  const Icon = typeIcon(item);
  const secret = (name?: string) =>
    name ? (
      <ResourceListLabel
        itemRef={ObjectReference.create({
          apiVersion: "enterprise/v1",
          kind: "Secret",
          name,
        })}
      />
    ) : (
      <MutedValue>Not set</MutedValue>
    );

  return {
    header: <DirectoryHero item={item} />,
    actions: p.isScim ? (
      <ScimCredential item={item} />
    ) : p.isSyncable ? (
      <SynchronizeButton item={item} />
    ) : undefined,
    groupOrder: [
      { title: "Connection", icon: Icon },
      { title: "Synchronization", icon: RefreshCw },
      { title: "Runtime", icon: Cpu },
    ],
    items: [
      {
        label: "Type",
        primary: true,
        icon: Icon,
        value: <Label>{p.type}</Label>,
      },
      {
        label: "Inventory",
        primary: true,
        icon: Users,
        value: (
          <div className="flex flex-wrap gap-1">
            <DirectoryProviderInventoryLabels item={item} />
          </div>
        ),
      },
      {
        label: p.isScim ? "Delivery" : "Polling",
        primary: true,
        icon: Timer,
        value: p.isScim ? (
          <span className="font-semibold">Pushed by the provider</span>
        ) : (
          <span className="font-semibold">
            {formatDuration(pollingOf(item), "Default interval")}
          </span>
        ),
      },
      p.isScim
        ? {
            label: "Authentication",
            primary: true,
            icon: KeyRound,
            value: <span className="font-semibold">Bearer token</span>,
          }
        : {
            label: "Last synchronized",
            primary: true,
            icon: History,
            value: p.lastSuccessfulSync?.completedAt ? (
              <TimeAgo rfc3339={p.lastSuccessfulSync.completedAt} />
            ) : (
              <MutedValue>Never</MutedValue>
            ),
          },

      ...(type?.oneofKind === "googleWorkspace"
        ? [
            {
              label: "Customer ID",
              group: "Connection",
              value: <CopyText value={type.googleWorkspace.customer} />,
            },
            {
              label: "Administrator",
              group: "Connection",
              value: <CopyText value={type.googleWorkspace.impersonateSubject} />,
            },
            {
              label: "Service account",
              group: "Connection",
              value: secret(
                type.googleWorkspace.serviceAccount?.type.oneofKind === "fromSecret"
                  ? type.googleWorkspace.serviceAccount.type.fromSecret
                  : undefined,
              ),
            },
          ]
        : []),
      ...(type?.oneofKind === "keycloak"
        ? [
            {
              label: "Keycloak URL",
              group: "Connection",
              value: <CopyText value={type.keycloak.url} />,
              span: "full" as const,
            },
            { label: "Realm", group: "Connection", value: type.keycloak.realm },
            {
              label: "Client ID",
              group: "Connection",
              value: <CopyText value={type.keycloak.clientID} />,
            },
            {
              label: "Client secret",
              group: "Connection",
              value: secret(
                type.keycloak.clientSecret?.type.oneofKind === "fromSecret"
                  ? type.keycloak.clientSecret.type.fromSecret
                  : undefined,
              ),
            },
            ...(type.keycloak.insecureSkipVerify
              ? [
                  {
                    label: "TLS",
                    group: "Connection",
                    value: (
                      <TonePill tone="warning">
                        Certificate verification disabled
                      </TonePill>
                    ),
                  },
                ]
              : []),
          ]
        : []),
      ...(item.status?.id && !p.isScim
        ? [
            {
              label: "Provider ID",
              group: "Connection",
              value: <CopyText value={item.status.id} />,
            },
          ]
        : []),

      ...(!p.isScim
        ? [
            p.allSynchronizations.length
              ? {
                  label: "History",
                  group: "Synchronization",
                  layout: "stacked" as const,
                  value: (
                    <HistoryList
                      entries={p.allSynchronizations
                        .slice(0, 6)
                        .map((entry, index) => {
                          const meta = getSyncStateMeta(entry.state);
                          return {
                            key: `${entry.createdAt?.seconds ?? 0}-${index}`,
                            tone: meta.tone,
                            label: `${index === 0 && p.currentSync === entry ? "Current · " : ""}${meta.label}`,
                            meta: (
                              <>
                                {entry.createdAt && (
                                  <span>
                                    Started <TimeAgo rfc3339={entry.createdAt} />
                                  </span>
                                )}
                                {entry.completedAt && (
                                  <span>
                                    Completed{" "}
                                    <TimeAgo rfc3339={entry.completedAt} />
                                  </span>
                                )}
                              </>
                            ),
                          };
                        })}
                    />
                  ),
                }
              : {
                  label: "History",
                  group: "Synchronization",
                  value: <MutedValue>Never synchronized</MutedValue>,
                },
          ]
        : []),

      ...(item.status?.userRef?.name
        ? [
            {
              label: "Runtime User",
              group: "Runtime",
              value: <ResourceListLabel itemRef={item.status.userRef} />,
            },
          ]
        : []),
      ...(item.status?.sessionRef?.name
        ? [
            {
              label: "Runtime Session",
              group: "Runtime",
              value: <ResourceListLabel itemRef={item.status.sessionRef} />,
            },
          ]
        : []),
    ],
  };
};
