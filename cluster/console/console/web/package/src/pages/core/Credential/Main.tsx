import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { onError } from "@/utils";
import { getClientCore } from "@/utils/client";
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
  Select,
  Switch,
} from "@mantine/core";
import { useMutation } from "@tanstack/react-query";

import * as CoreP from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { useDisclosure } from "@mantine/hooks";
import {
  AlertTriangle,
  Check,
  Copy,
  KeyRound,
  RefreshCw,
  Shield,
  X,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

const TokenField = (props: { label: string; value: string }) => (
  <div className="flex items-end gap-2">
    <PasswordInput className="flex-1" label={props.label} value={props.value} readOnly />
    <CopyButton value={props.value}>
      {({ copied, copy }) => (
        <Button type="button" variant="default" leftSection={copied ? <Check size={13} /> : <Copy size={13} />} onClick={copy}>
          {copied ? "Copied" : "Copy"}
        </Button>
      )}
    </CopyButton>
  </div>
);

const GenerateC = (props: { item: CoreC.Credential; onClose: () => void }) => {
  const { item, onClose } = props;
  const [tkn, setTkn] = React.useState<CoreC.CredentialToken | undefined>();
  const [confirmed, setConfirmed] = React.useState(false);

  const mutationGenerate = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCore().generateCredentialToken(
        CoreC.GenerateCredentialTokenRequest.create({ credentialRef: getResourceRef(item) }),
      );
      return response;
    },
    onSuccess: (response) => {
      setTkn(response);
      setConfirmed(false);
      invalidateResource(item);
      invalidateResourceList(item);
    },
    onError,
  });

  const tokenFields = (() => {
    if (tkn?.type.oneofKind === "authenticationToken") return [{ label: "Authentication token", value: tkn.type.authenticationToken.authenticationToken }];
    if (tkn?.type.oneofKind === "accessToken") return [{ label: "Access token", value: tkn.type.accessToken.accessToken }];
    if (tkn?.type.oneofKind === "oauth2Credentials") return [
      { label: "Client ID", value: tkn.type.oauth2Credentials.clientID },
      { label: "Client secret", value: tkn.type.oauth2Credentials.clientSecret },
    ];
    return [];
  })();

  return (
    <div className="bg-white">
      <header className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
        <div className="flex items-center gap-2.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900 text-white"><KeyRound size={15} /></span>
          <div>
            <h2 className="text-sm font-bold text-slate-900">Credential token</h2>
            <p className="text-xs font-normal text-slate-500">{item.metadata?.displayName || item.metadata?.name}</p>
          </div>
        </div>
        <Button type="button" variant="subtle" color="gray" size="compact-xs" disabled={mutationGenerate.isPending} onClick={onClose}><X size={14} /></Button>
      </header>
      <div className="space-y-4 px-5 py-5">
        {tkn ? (
          <>
            <Alert color="green" title="Copy these values now">These values are shown only in this dialog. Store them securely before closing.</Alert>
            {tokenFields.map((field) => <TokenField key={field.label} {...field} />)}
          </>
        ) : (
          <>
            <Alert color="amber" icon={<AlertTriangle size={15} />} title="The current token will stop working">
              Generating a new token immediately invalidates the credential's current token.
            </Alert>
            <Switch checked={confirmed} label="I understand that the current token will be invalidated" onChange={(event) => setConfirmed(event.currentTarget.checked)} />
          </>
        )}
      </div>
      <footer className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
        <Button type="button" variant="default" size="sm" disabled={mutationGenerate.isPending} onClick={onClose}>{tkn ? "Done" : "Cancel"}</Button>
        {!tkn && <Button type="button" color="ink" size="sm" disabled={!confirmed} loading={mutationGenerate.isPending} leftSection={<KeyRound size={13} />} onClick={() => mutationGenerate.mutate()}>Generate token</Button>}
      </footer>
    </div>
  );
};

const GenerateTokenModal = (props: {
  item: CoreC.Credential;
  opened: boolean;
  onClose: () => void;
}) => {
  const { item, opened, onClose } = props;

  return (
    <Modal opened={opened} onClose={onClose} centered size="lg" withCloseButton={false} padding={0} styles={{ content: { borderRadius: 14, overflow: "hidden" } }}>
      {opened && <GenerateC key={item.metadata?.uid || item.metadata?.name} item={item} onClose={onClose} />}
    </Modal>
  );
};

export const ItemInfo = (props: { item: CoreC.Credential }) => {
  let { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">
        <EditItemWrap
          mutation={mutationUpdate}
          showComponent={
            <span>
              {match(item.spec!.type)
                .with(
                  CoreP.Credential_Spec_Type.AUTH_TOKEN,
                  () => "Authentication Token",
                )
                .with(
                  CoreP.Credential_Spec_Type.OAUTH2,
                  () => "OAuth2 Client Credential",
                )
                .with(
                  CoreP.Credential_Spec_Type.ACCESS_TOKEN,
                  () => "Access Token",
                )
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Authentication Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.AUTH_TOKEN
                    ],
                },
                {
                  label: "OAuth2 Client Credential",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.OAUTH2
                    ],
                },
                {
                  label: "Access Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.ACCESS_TOKEN
                    ],
                },
              ]}
              value={CoreP.Credential_Spec_Type[item.spec!.type]}
              onChange={(v) => {
                if (!v) return;
                const next = CoreC.Credential.clone(item);
                next.spec!.type = CoreP.Credential_Spec_Type[v as "OAUTH2"];
                mutationUpdate.mutate(next);
              }}
            />
          }
        />
      </InfoItem>
      {item.spec!.maxAuthentications > 0 && (
        <InfoItem title="Max Authentications">
          {item.spec!.maxAuthentications}
        </InfoItem>
      )}
      {item.status!.lastRotationAt && (
        <InfoItem title="Last Rotation">
          <TimeAgo rfc3339={item.status!.lastRotationAt} />
        </InfoItem>
      )}

      {item.spec!.expiresAt && (
        <InfoItem title="Expires at">
          <TimeAgo rfc3339={item.spec!.expiresAt} />
        </InfoItem>
      )}

      {item.status!.totalAuthentications > 0 && (
        <InfoItem title="Total Rotations">
          <span>{item.status!.totalRotations}</span>
          {item.spec!.maxAuthentications > 0 && (
            <span className="ml-1">
              (Max Authentications {item.spec!.maxAuthentications})
            </span>
          )}
        </InfoItem>
      )}

      {item.status!.totalRotations > 0 && (
        <InfoItem title="Total Authentications">
          {item.status!.totalAuthentications}
        </InfoItem>
      )}

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
            checked={!item.spec!.isDisabled}
            onChange={(v) => {
              const next = CoreC.Credential.clone(item);
              next.spec!.isDisabled = !v.currentTarget.checked;
              mutationUpdate.mutate(next);
            }}
          />
        </div>
      </InfoItem>

      <InfoItem title="Generate">
        <Button size={`xs`} onClick={open}>
          Generate/Rotate Token
        </Button>
      </InfoItem>
      <GenerateTokenModal item={item} opened={opened} onClose={close} />
    </>
  );
};

export default (props: { item: CoreC.Credential }) => {
  const { item } = props;

  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: CoreC.Credential;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const [opened, { open, close }] = useDisclosure(false);
  const userRef =
    item.status?.userRef?.name || item.status?.userRef?.uid
      ? item.status.userRef
      : ObjectReference.create({
          apiVersion: "core/v1",
          kind: "User",
          name: item.spec!.user,
        });

  return {
    groupOrder: ["Usage", "Security", "Authorization"],
    actions: (
      <>
        <Button
          variant="default"
          size="compact-sm"
          leftSection={<RefreshCw size={11} strokeWidth={2.5} />}
          onClick={open}
        >
          Generate / Rotate
        </Button>
        <GenerateTokenModal item={item} opened={opened} onClose={close} />
      </>
    ),
    items: [
      {
        label: "User",
        primary: true,
        value: <ResourceListLabel itemRef={userRef} />,
      },
      {
        label: "Type",
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="type"
            showComponent={
              <span className="text-body font-semibold text-slate-700">
                {match(item.spec!.type)
                  .with(
                    CoreP.Credential_Spec_Type.AUTH_TOKEN,
                    () => "Authentication Token",
                  )
                  .with(
                    CoreP.Credential_Spec_Type.OAUTH2,
                    () => "OAuth2 Client Credential",
                  )
                  .with(
                    CoreP.Credential_Spec_Type.ACCESS_TOKEN,
                    () => "Access Token",
                  )
                  .otherwise(() => "")}
              </span>
            }
            editComponent={
              <Select
                size="xs"
                data={[
                  {
                    label: "Authentication Token",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.AUTH_TOKEN
                      ],
                  },
                  {
                    label: "OAuth2 Client Credential",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.OAUTH2
                      ],
                  },
                  {
                    label: "Access Token",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.ACCESS_TOKEN
                      ],
                  },
                ]}
                value={CoreP.Credential_Spec_Type[item.spec!.type]}
                onChange={(v) => {
                  if (!v) return;
                  const next = CoreC.Credential.clone(item);
                  next.spec!.type =
                    CoreP.Credential_Spec_Type[v as "OAUTH2"];
                  mutationUpdate.mutate(next);
                }}
              />
            }
          />
        ),
      },

      ...(item.spec!.expiresAt
        ? [
            {
              label: "Expires",
              primary: true,
              value: (
                <span className="text-body font-normal text-slate-600">
                  <TimeAgo rfc3339={item.spec!.expiresAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.lastRotationAt
        ? [
            {
              label: "Last rotation",
              group: "Usage",
              value: (
                <span className="text-body font-normal text-slate-600">
                  <TimeAgo rfc3339={item.status!.lastRotationAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.totalRotations > 0
        ? [
            {
              label: "Total rotations",
              group: "Usage",
              value: (
                <span className="text-body font-semibold text-slate-700 tabular-nums">
                  {item.status!.totalRotations}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.totalAuthentications > 0
        ? [
            {
              label: "Total authentications",
              group: "Usage",
              value: (
                <span className="text-body font-semibold text-slate-700 tabular-nums">
                  {item.status!.totalAuthentications}
                  {item.spec!.maxAuthentications > 0 && (
                    <span className="text-slate-500 font-medium ml-1">
                      / {item.spec!.maxAuthentications} max
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),

      ...(item.spec!.maxAuthentications > 0
        ? [
            {
              label: "Max authentications",
              group: "Usage",
              value: (
                <span className="text-body font-semibold text-slate-700 tabular-nums">
                  {item.spec!.maxAuthentications}
                </span>
              ),
            },
          ]
        : []),

      {
        label: "Session type",
        group: "Usage",
        value:
          item.spec!.sessionType === CoreC.Session_Status_Type.CLIENT
            ? "Client"
            : item.spec!.sessionType === CoreC.Session_Status_Type.CLIENTLESS
              ? "Clientless"
              : "Not set",
      },

      ...(item.spec!.autoDelete
        ? [
            {
              label: "Lifecycle",
              group: "Usage",
              value: "Automatically delete after reaching the usage limit",
              span: "full" as const,
            },
          ]
        : []),

      ...(item.status?.isLocked
        ? [
            {
              label: "Security state",
              group: "Security",
              value: <span className="font-semibold text-red-600">Locked</span>,
            },
          ]
        : []),

      ...(item.spec?.authorization?.policies.length
        ? [
            {
              label: "Policies",
              group: "Authorization",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.policies.map((policy) => (
                    <ResourceListLabel
                      key={policy}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Policy",
                        name: policy,
                      })}
                    />
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.spec?.authorization?.inlinePolicies.length
        ? [
            {
              label: "Inline policies",
              group: "Authorization",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.inlinePolicies.map(
                    (policy, index) => (
                      <ResourceListLabel
                        key={`${policy.name}-${index}`}
                        label="Inline policy"
                      >
                        <Shield size={12} strokeWidth={2.5} />
                        {policy.name || `Inline policy ${index + 1}`}
                      </ResourceListLabel>
                    ),
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

    ],
  };
};
