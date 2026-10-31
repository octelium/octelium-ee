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
import TimeAgo, { useNow } from "@/components/TimeAgo";
import {
  HeroStat,
  Meter,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  ResourceActionButton,
  toDate,
} from "@/components/ResourceLayout/overview";
import { authorizationItems } from "@/components/ResourceLayout/overviewItems";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { useDisclosure } from "@mantine/hooks";
import {
  Activity,
  AlertTriangle,
  Ban,
  CalendarClock,
  Check,
  Clock3,
  Copy,
  Fingerprint,
  KeyRound,
  Lock,
  MonitorSmartphone,
  RefreshCw,
  ShieldCheck,
  UserRound,
  X,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";
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

const typeLabel = (type: CoreP.Credential_Spec_Type) =>
  match(type)
    .with(CoreP.Credential_Spec_Type.AUTH_TOKEN, () => "Authentication Token")
    .with(CoreP.Credential_Spec_Type.OAUTH2, () => "OAuth2 Client Credential")
    .with(CoreP.Credential_Spec_Type.ACCESS_TOKEN, () => "Access Token")
    .otherwise(() => "Unknown type");

const sessionTypeLabel = (type?: CoreC.Session_Status_Type) =>
  type === CoreC.Session_Status_Type.CLIENT
    ? "Client"
    : type === CoreC.Session_Status_Type.CLIENTLESS
      ? "Clientless"
      : "Not set";

const CredentialHero = (props: { item: CoreC.Credential }) => {
  const { item } = props;
  const now = useNow();
  const spec = item.spec!;
  const status = item.status;
  const expiresAt = toDate(spec.expiresAt);
  const left = expiresAt ? (expiresAt.getTime() - now) / 1000 : undefined;
  const isExpired = left !== undefined && left <= 0;
  const max = spec.maxAuthentications;
  const used = status?.totalAuthentications ?? 0;
  const exhausted = max > 0 && used >= max;
  const neverIssued = !status?.lastRotationAt && !status?.totalRotations;
  const tone =
    spec.isDisabled || status?.isLocked || isExpired || exhausted
      ? "danger"
      : neverIssued || (left !== undefined && left <= 7 * 86400)
        ? "warning"
        : "success";

  return (
    <OverviewHero
      tone={tone}
      icon={<KeyRound size={18} strokeWidth={2.2} />}
      title={typeLabel(spec.type)}
      detail={
        <>
          For {status?.userRef?.name || spec.user || "an unknown User"} ·{" "}
          {sessionTypeLabel(spec.sessionType).toLowerCase()} Sessions ·{" "}
          {status?.lastRotationAt ? (
            <>
              token rotated <TimeAgo rfc3339={status.lastRotationAt} />
            </>
          ) : (
            "no token generated yet"
          )}
        </>
      }
      meter={
        max > 0 && (
          <Meter
            value={Math.min(100, (used / max) * 100)}
            label="Authentications used"
            tone={exhausted ? "danger" : used / max >= 0.8 ? "warning" : "success"}
            caption={`${used.toLocaleString()} of ${max.toLocaleString()} authentications used`}
          />
        )
      }
      aside={
        <HeroStat
          label="Expires"
          tone={isExpired ? "danger" : left !== undefined && left <= 7 * 86400 ? "warning" : "neutral"}
          value={spec.expiresAt ? <TimeAgo rfc3339={spec.expiresAt} /> : "Never"}
        />
      }
    >
      {spec.isDisabled && (
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          The Credential cannot be used to authenticate.
        </OverviewNotice>
      )}
      {status?.isLocked && (
        <OverviewNotice tone="danger" icon={Lock} title="Locked by the Cluster.">
          Authentications with this Credential are rejected.
        </OverviewNotice>
      )}
      {isExpired && (
        <OverviewNotice tone="danger" icon={Clock3} title="Expired.">
          Generate a new token or extend the expiration to use it again.
        </OverviewNotice>
      )}
      {exhausted && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="Usage limit reached.">
          {spec.autoDelete
            ? "The Credential is deleted automatically."
            : "Raise the maximum authentications to use it again."}
        </OverviewNotice>
      )}
      {neverIssued && !spec.isDisabled && (
        <OverviewNotice tone="warning" icon={KeyRound} title="No token generated yet.">
          Generate a token to start using this Credential.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

const GenerateTokenAction = (props: { item: CoreC.Credential }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const neverIssued =
    !props.item.status?.lastRotationAt && !props.item.status?.totalRotations;

  return (
    <>
      <ResourceActionButton
        tone="primary"
        icon={neverIssued ? KeyRound : RefreshCw}
        onClick={open}
      >
        {neverIssued ? "Generate token" : "Rotate token"}
      </ResourceActionButton>
      <GenerateTokenModal item={props.item} opened={opened} onClose={close} />
    </>
  );
};

export const MainInfo = (props: {
  item: CoreC.Credential;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const userRef =
    item.status?.userRef?.name || item.status?.userRef?.uid
      ? item.status.userRef
      : ObjectReference.create({
          apiVersion: "core/v1",
          kind: "User",
          name: item.spec!.user,
        });
  const max = item.spec!.maxAuthentications;

  return {
    header: <CredentialHero item={item} />,
    actions: <GenerateTokenAction item={item} />,
    groupOrder: [
      { title: "Token", icon: KeyRound },
      { title: "Lifecycle", icon: CalendarClock },
      { title: "Authorization", icon: ShieldCheck },
    ],
    items: [
      {
        label: "User",
        primary: true,
        icon: UserRound,
        value: <ResourceListLabel itemRef={userRef} />,
      },
      {
        label: "Type",
        primary: true,
        icon: Fingerprint,
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="type"
            showComponent={
              <span className="font-semibold text-slate-900">
                {typeLabel(item.spec!.type)}
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
      {
        label: "Session type",
        primary: true,
        icon: MonitorSmartphone,
        value: (
          <span className="font-semibold">
            {sessionTypeLabel(item.spec!.sessionType)}
          </span>
        ),
      },
      {
        label: "Authentications",
        primary: true,
        icon: Activity,
        value: (
          <span className="font-semibold tabular-nums">
            {item.status!.totalAuthentications.toLocaleString()}
            {max > 0 && (
              <span className="ml-1 text-xs font-normal text-slate-500">
                of {max.toLocaleString()}
              </span>
            )}
          </span>
        ),
      },

      ...(item.status?.tokenID
        ? [
            {
              label: "Token ID",
              group: "Token",
              value: <CopyText value={item.status.tokenID} />,
              span: "full" as const,
            },
          ]
        : []),
      {
        label: "Last rotation",
        group: "Token",
        value: item.status!.lastRotationAt ? (
          <TimeAgo rfc3339={item.status!.lastRotationAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },
      {
        label: "Rotations",
        group: "Token",
        value: (
          <span className="font-semibold tabular-nums">
            {item.status!.totalRotations.toLocaleString()}
          </span>
        ),
      },

      {
        label: "Expires",
        group: "Lifecycle",
        value: item.spec!.expiresAt ? (
          <TimeAgo rfc3339={item.spec!.expiresAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },
      {
        label: "Max authentications",
        group: "Lifecycle",
        value:
          max > 0 ? (
            <span className="font-semibold tabular-nums">
              {max.toLocaleString()}
            </span>
          ) : (
            <MutedValue>Unlimited</MutedValue>
          ),
      },
      {
        label: "Auto-delete",
        group: "Lifecycle",
        value: item.spec!.autoDelete ? (
          "After reaching the usage limit"
        ) : (
          <MutedValue>Disabled</MutedValue>
        ),
      },

      ...authorizationItems(item.spec?.authorization),
    ],
  };
};
