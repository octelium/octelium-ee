import * as EnterpriseC from "@/apis/enterprisev1/enterprisev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
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
import { Button, Modal, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { RefreshCcw, RefreshCw } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getType, isSyncable } from "./List";

const getSyncStateMeta = (
  state?: EnterpriseC.DirectoryProvider_Status_Synchronization_State,
): { label: string; className: string } =>
  match(state)
    .with(
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED,
      () => ({ label: "Sync Requested", className: "text-amber-500" }),
    )
    .with(
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNCING,
      () => ({ label: "Syncing", className: "text-amber-500" }),
    )
    .with(
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SUCCESS,
      () => ({ label: "Success", className: "text-emerald-600" }),
    )
    .with(
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.FAILED,
      () => ({ label: "Failed", className: "text-red-500" }),
    )
    .otherwise(() => ({ label: "", className: "text-slate-500" }));

const GenerateC = (props: { item: EnterpriseC.DirectoryProvider }) => {
  const { item } = props;
  let [tkn, setTkn] = React.useState<
    EnterpriseC.GenerateDirectoryProviderCredentialResponse | undefined
  >(undefined);

  let [enabled, setEnabled] = React.useState(false);

  const mutationGenerate = useMutation({
    mutationFn: async () => {
      const { response } =
        await getClientEnterprise().generateDirectoryProviderCredential({
          directoryProviderRef: getResourceRef(item),
          mode: EnterpriseC.GenerateDirectoryProviderCredentialRequest_Mode
            .BEARER,
        });
      return response;
    },

    onSuccess: (response) => {
      setTkn(response);
      invalidateResource(item);
      invalidateResourceList(item);
    },
    onError: onError,
  });

  return (
    <div className="w-full">
      <div className="flex flex-col items-center justify-center my-8 w-full">
        <div className="w-full flex items-center justify-center">
          <Button
            onClick={() => {
              mutationGenerate.mutate();
            }}
            loading={mutationGenerate.isPending}
            leftSection={<RefreshCcw />}
            disabled={!enabled}
          >
            Generate Access Token
          </Button>
        </div>

        <div className="w-full flex items-center my-4 justify-center">
          <Switch
            checked={enabled}
            label={
              <div className="font-bold text-sm text-center text-slate-600">
                Generating a new token immediately invalidates the previous one
              </div>
            }
            onChange={(event) => setEnabled(event.currentTarget.checked)}
          />
        </div>
      </div>
      {tkn && (
        <div className="w-full">
          {tkn.type.oneofKind === `bearer` && (
            <div className="w-full flex items-center justify-center">
              <InfoItem title="Access Token">
                <CopyText value={tkn.type.bearer.accessToken} truncate={20} />
              </InfoItem>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const SynchronizeButton = (props: {
  item: EnterpriseC.DirectoryProvider;
  variant?: string;
}) => {
  const { item } = props;

  const mutationSync = useMutation({
    mutationFn: async () => {
      const { response } =
        await getClientEnterprise().synchronizeDirectoryProvider({
          directoryProviderRef: getResourceRef(item),
        });
      return response;
    },
    onSuccess: () => {
      toast.success("Synchronization requested");
      invalidateResource(item);
      invalidateResourceList(item);
    },
    onError: onError,
  });

  const isSyncing =
    item.status?.synchronization?.state ===
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNCING ||
    item.status?.synchronization?.state ===
      EnterpriseC.DirectoryProvider_Status_Synchronization_State.SYNC_REQUESTED;

  return (
    <Button
      variant={props.variant ?? "default"}
      size="sm"
      leftSection={<RefreshCw size={13} strokeWidth={2.5} />}
      loading={mutationSync.isPending || isSyncing}
      onClick={() => mutationSync.mutate()}
    >
      Synchronize
    </Button>
  );
};

export const ItemInfo = (props: { item: EnterpriseC.DirectoryProvider }) => {
  let { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const mutationUpdate = useUpdateResource();

  const isScim = item.spec?.type.oneofKind === "scim";

  return (
    <>
      <InfoItem title="Type">
        <Label>{getType(item)}</Label>
      </InfoItem>
      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>
      <InfoItem title="Short ID">
        <CopyText value={item.status?.id} />
      </InfoItem>
      {isScim && (
        <InfoItem title="SCIM Endpoint">
          <CopyText
            value={`https://dirsync.octelium.${getDomain()}/scim/${item.status?.id}`}
          />
        </InfoItem>
      )}

      {item.status?.synchronization?.state && (
        <InfoItem title="Synchronization">
          <span
            className={
              getSyncStateMeta(item.status.synchronization.state).className
            }
          >
            {getSyncStateMeta(item.status.synchronization.state).label}
          </span>
        </InfoItem>
      )}

      {isScim ? (
        <InfoItem title="Generate">
          <Button size={`xs`} onClick={open}>
            Generate Access Token
          </Button>
        </InfoItem>
      ) : isSyncable(item) ? (
        <InfoItem title="Synchronize">
          <SynchronizeButton item={item} />
        </InfoItem>
      ) : null}

      <Modal opened={opened} onClose={close} size={"xl"} centered>
        <GenerateC item={item} />
      </Modal>
    </>
  );
};

export default (props: { item: EnterpriseC.DirectoryProvider }) => {
  const { item } = props;

  return (
    <>
      <ItemInfo item={item} />
    </>
  );
};

export const MainInfo = (props: {
  item: EnterpriseC.DirectoryProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const [opened, { open, close }] = useDisclosure(false);

  const isScim = item.spec?.type.oneofKind === "scim";
  const sync = item.status?.synchronization;

  return {
    items: [
      {
        label: "Type",
        value: <Label>{getType(item)}</Label>,
      },

      {
        label: "Active",
        value: (
          <EditItemWrap
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-sm font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  item.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },

      ...(item.status?.id
        ? [
            {
              label: "Short ID",
              value: <CopyText value={item.status.id} />,
            },
          ]
        : []),

      ...(isScim && item.status?.id
        ? [
            {
              label: "SCIM endpoint",
              value: (
                <CopyText
                  value={`https://dirsync.octelium.${getDomain()}/scim/${item.status.id}`}
                />
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(sync?.state
        ? [
            {
              label: "Synchronization",
              value: (
                <span className="flex items-center gap-2">
                  <span
                    className={twMerge(
                      "text-sm font-semibold",
                      getSyncStateMeta(sync.state).className,
                    )}
                  >
                    {getSyncStateMeta(sync.state).label}
                  </span>
                  {sync.completedAt && (
                    <span className="text-[0.68rem] font-semibold text-slate-400">
                      <TimeAgo rfc3339={sync.completedAt} />
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),

      isScim
        ? {
            label: "Access token",
            value: (
              <>
                <Button
                  variant="default"
                  size="sm"
                  leftSection={<RefreshCw size={13} strokeWidth={2.5} />}
                  onClick={open}
                >
                  Generate access token
                </Button>
                <Modal opened={opened} onClose={close} size="xl" centered>
                  <GenerateC item={item} />
                </Modal>
              </>
            ),
          }
        : {
            label: "Synchronize",
            value: <SynchronizeButton item={item} />,
          },
    ],
  };
};
