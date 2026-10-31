import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { randomStringLowerCase, slugify } from "@/utils";
import { getClientCore } from "@/utils/client";
import { getResourcePath, getResourceRef } from "@/utils/pb";
import { Button, Drawer, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  Ban,
  Bot,
  Fingerprint,
  KeyRound,
  Link2,
  Lock,
  Mail,
  Plus,
  ShieldCheck,
  Tags,
  Timer,
  UserRound,
  Users,
  X,
} from "lucide-react";
import {
  formatDuration,
  MutedValue,
  OverviewNotice,
  ResourceActionButton,
} from "@/components/ResourceLayout/overview";
import {
  attrsItems,
  authorizationItems,
} from "@/components/ResourceLayout/overviewItems";
import { CountryFlag } from "@/components/SelectCountry";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { notifications } from "@mantine/notifications";
import { twMerge } from "@/lib/utils";
import { match } from "ts-pattern";
import Edit from "../Credential/Edit";
import { useUserRelatedResources } from "./related";

export const AccessLog = (props: { item: CoreC.User }) => {
  return <AccessLogViewer userRef={getResourceRef(props.item)} />;
};

export const ResourceItemInfo = (props: { item: CoreC.User }) => {
  let { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">
        <Label>
          {match(item.spec!.type)
            .with(CoreC.User_Spec_Type.HUMAN, () => "Human")
            .with(CoreC.User_Spec_Type.WORKLOAD, () => "Workload")
            .otherwise(() => "")}
        </Label>
      </InfoItem>

      {item.spec?.email && (
        <InfoItem title="Email">
          <CopyText value={item.spec.email} />
        </InfoItem>
      )}

      {item.spec!.groups.length > 0 && (
        <InfoItem title="Groups">
          <div className="flex items-center">
            {item.spec!.groups.map((x) => (
              <Label key={x}>{x}</Label>
            ))}
          </div>
        </InfoItem>
      )}

      {item.spec!.authorization &&
        item.spec!.authorization.policies.length > 0 && (
          <InfoItem title="Policies">
            <div className="flex items-center">
              {item.spec!.authorization.policies.map((x) => (
                <Label key={x}>{x}</Label>
              ))}
            </div>
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
              item.spec!.isDisabled = !v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>
    </>
  );
};

export default (props: { item: CoreC.User }) => {
  let { item } = props;

  return (
    <div>
      <div className="w-full mb-8">
        <ResourceItemInfo item={item} />
      </div>
    </div>
  );
};

const CreateUserCredential = (props: { item: CoreC.User }) => {
  const { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const navigate = useNavigate();
  const getCredentialNamePrefix = (
    userName: string,
    type: CoreC.Credential_Spec_Type,
  ) =>
    `${userName}-${slugify(CoreC.Credential_Spec_Type[type]).toLowerCase()}-`;
  const getCredentialName = (
    userName: string,
    type: CoreC.Credential_Spec_Type,
  ) => `${getCredentialNamePrefix(userName, type)}${randomStringLowerCase(6)}`;
  const [credential, setCredential] = useState(() => {
    const next = CoreC.Credential.create({
      kind: "Credential",
      apiVersion: "core/v1",
      metadata: {},
      spec: {
        user: item.metadata?.name,
        type: CoreC.Credential_Spec_Type.AUTH_TOKEN,
        sessionType: CoreC.Session_Status_Type.CLIENT,
      },
    });
    next.metadata!.name = getCredentialName(
      item.metadata?.name ?? "",
      CoreC.Credential_Spec_Type.AUTH_TOKEN,
    );
    return next;
  });

  const mutationCredential = useMutation({
    mutationFn: async () => {
      return await getClientCore().createCredential(credential).response;
    },
    onSuccess: (response) => {
      navigate(getResourcePath(response));
    },
    onError: (err: unknown) => {
      if (err instanceof Error) {
        notifications.show({ color: "red", message: err.message });
      }
    },
  });

  return (
    <div className="flex items-center">
      <ResourceActionButton tone="primary" icon={Plus} onClick={open}>
        Create credential
      </ResourceActionButton>
      <Drawer
        opened={opened}
        onClose={close}
        position="right"
        size="min(760px, 100vw)"
        closeOnClickOutside={!mutationCredential.isPending}
        closeOnEscape={!mutationCredential.isPending}
        title={
          <div className="flex items-center gap-3">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-600">
              <KeyRound size={16} strokeWidth={2.25} />
            </span>
            <div className="flex flex-col">
              <span className="text-sm font-bold text-slate-800">
                Create credential
              </span>
              <span className="text-xs font-normal text-slate-500">
                Issue a new credential for {item.metadata?.name}
              </span>
            </div>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: {
            borderBottomWidth: "1px",
            borderBottomStyle: "solid",
            borderBottomColor: "var(--color-slate-200)",
            minHeight: "56px",
          },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: "16px",
            backgroundColor: "var(--color-slate-50)",
          },
          content: {
            borderLeftWidth: "1px",
            borderLeftStyle: "solid",
            borderLeftColor: "var(--color-slate-200)",
          },
        }}
      >
        <div className="flex min-h-[calc(100dvh-96px)] flex-col gap-3">
          <div className="flex-1 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <Edit
              hideUser
              onUpdate={(v) => {
                const prefix = getCredentialNamePrefix(
                  v.spec!.user,
                  v.spec!.type,
                );
                if (!v.metadata!.name.startsWith(prefix)) {
                  v.metadata!.name = getCredentialName(
                    v.spec!.user,
                    v.spec!.type,
                  );
                }
                setCredential(CoreC.Credential.clone(v));
              }}
              item={credential}
            />
          </div>

          <div className="flex items-center justify-end gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm sm:px-5">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              disabled={mutationCredential.isPending}
              onClick={close}
            >
              Cancel
            </Button>
            <Button
              color="ink"
              size="sm"
              leftSection={<KeyRound size={13} strokeWidth={2.5} />}
              loading={mutationCredential.isPending}
              disabled={
                mutationCredential.isPending ||
                credential.spec?.type ===
                  CoreC.Credential_Spec_Type.TYPE_UNKNOWN
              }
              onClick={() => mutationCredential.mutate()}
            >
              {mutationCredential.isPending ? "Creating…" : "Create credential"}
            </Button>
          </div>
        </div>
      </Drawer>
    </div>
  );
};

const UserNotices = (props: { item: CoreC.User }) => {
  const { item } = props;
  if (!item.spec?.isDisabled && !item.status?.isLocked) return null;
  return (
    <div className="flex flex-col gap-2 border-b border-slate-200 bg-white px-4 py-3">
      {item.spec?.isDisabled && (
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          The User cannot authenticate and its Sessions cannot access any
          Service.
        </OverviewNotice>
      )}
      {item.status?.isLocked && (
        <OverviewNotice tone="danger" icon={Lock} title="Locked by the Cluster.">
          Every request of this User is rejected until it is unlocked.
        </OverviewNotice>
      )}
    </div>
  );
};

export const MainInfo = (props: { item: CoreC.User }): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec!;
  const related = useUserRelatedResources(item);
  const isWorkload = spec.type === CoreC.User_Spec_Type.WORKLOAD;
  const info = spec.info;
  const fullName = [info?.firstName, info?.middleName, info?.lastName]
    .filter(Boolean)
    .join(" ");
  const identities = spec.authentication?.identities ?? [];
  const session = spec.session;

  return {
    header: <UserNotices item={item} />,
    actions: <CreateUserCredential item={item} />,
    groupOrder: [
      { title: "Profile", icon: UserRound },
      { title: "Identities", icon: Fingerprint },
      { title: "Session policy", icon: Timer },
      { title: "Authorization", icon: ShieldCheck },
      { title: "Related resources", icon: Link2 },
      { title: "Attributes", icon: Tags },
    ],
    items: [
      {
        label: "Type",
        primary: true,
        icon: isWorkload ? Bot : UserRound,
        value: (
          <span className="flex flex-col">
            <span className="font-semibold">
              {match(spec.type)
                .with(CoreC.User_Spec_Type.HUMAN, () => "Human")
                .with(CoreC.User_Spec_Type.WORKLOAD, () => "Workload")
                .otherwise(() => "Unknown")}
            </span>
            <span className="text-xs font-normal text-slate-500">
              {isWorkload ? "Machine identity" : "Person signing in"}
            </span>
          </span>
        ),
      },
      {
        label: "Email",
        primary: true,
        icon: Mail,
        value: spec.email ? (
          <CopyText value={spec.email} />
        ) : (
          <MutedValue>Not set</MutedValue>
        ),
      },
      {
        label: "Groups",
        primary: true,
        icon: Users,
        value: (
          <span className="font-semibold tabular-nums">
            {spec.groups.length}
          </span>
        ),
      },
      {
        label: "Identities",
        primary: true,
        icon: Fingerprint,
        value: (
          <span className="font-semibold tabular-nums">{identities.length}</span>
        ),
      },

      ...(fullName
        ? [{ label: "Name", group: "Profile", value: fullName }]
        : []),
      ...(info?.country
        ? [
            {
              label: "Country",
              group: "Profile",
              value: (
                <span className="inline-flex items-center gap-1.5">
                  <CountryFlag code={info.country} />
                  {info.country}
                </span>
              ),
            },
          ]
        : []),
      ...(info?.locale
        ? [{ label: "Locale", group: "Profile", value: info.locale }]
        : []),
      ...(info?.phone
        ? [{ label: "Phone", group: "Profile", value: <CopyText value={info.phone} /> }]
        : []),
      ...(info?.website
        ? [
            {
              label: "Website",
              group: "Profile",
              value: <CopyText value={info.website} />,
            },
          ]
        : []),

      ...(identities.length > 0
        ? [
            {
              label: "Linked identities",
              hint: "Accounts that sign in as this User",
              group: "Identities",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {identities.map((identity) => (
                    <ResourceListLabel
                      key={`${identity.identityProvider}-${identity.identifier}`}
                      label={identity.identityProvider || "Any provider"}
                    >
                      {identity.identifier}
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(session
        ? [
            ...(session.clientDuration
              ? [
                  {
                    label: "Client Sessions",
                    group: "Session policy",
                    value: formatDuration(session.clientDuration),
                  },
                ]
              : []),
            ...(session.clientlessDuration
              ? [
                  {
                    label: "Clientless Sessions",
                    group: "Session policy",
                    value: formatDuration(session.clientlessDuration),
                  },
                ]
              : []),
            ...(session.accessTokenDuration
              ? [
                  {
                    label: "Access tokens",
                    group: "Session policy",
                    value: formatDuration(session.accessTokenDuration),
                  },
                ]
              : []),
            ...(session.refreshTokenDuration
              ? [
                  {
                    label: "Refresh tokens",
                    group: "Session policy",
                    value: formatDuration(session.refreshTokenDuration),
                  },
                ]
              : []),
            ...(session.maxPerUser > 0
              ? [
                  {
                    label: "Max Sessions",
                    group: "Session policy",
                    value: (
                      <span className="font-semibold tabular-nums">
                        {session.maxPerUser}
                      </span>
                    ),
                  },
                ]
              : []),
            ...(session.defaultState !== CoreC.Session_Spec_State.STATE_UNKNOWN
              ? [
                  {
                    label: "New Sessions",
                    group: "Session policy",
                    value: match(session.defaultState)
                      .with(CoreC.Session_Spec_State.ACTIVE, () => "Active")
                      .with(CoreC.Session_Spec_State.PENDING, () => "Pending approval")
                      .with(CoreC.Session_Spec_State.REJECTED, () => "Rejected")
                      .otherwise(() => "Default"),
                  },
                ]
              : []),
          ]
        : []),

      ...(spec.groups.length > 0
        ? [
            {
              label: "Groups",
              group: "Authorization",
              value: (
                <div className="flex flex-wrap gap-1">
                  {spec.groups.map((x) => (
                    <ResourceListLabel
                      key={x}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Group",
                        name: x,
                      })}
                    />
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
      ...authorizationItems(spec.authorization),

      {
        label: "Linked resources",
        group: "Related resources",
        value:
          related.length > 0 ? (
            <div className="flex flex-wrap gap-1">
              {related.map(({ label, count, path, icon: Icon }) => (
                <ResourceListLabel key={label} label={label} to={path}>
                  <Icon size={12} strokeWidth={2.5} />
                  {count === undefined ? "…" : count.toLocaleString()}
                </ResourceListLabel>
              ))}
            </div>
          ) : (
            <MutedValue>No Sessions, Devices or Credentials yet</MutedValue>
          ),
        span: "full" as const,
      },
      ...attrsItems(spec.attrs),
    ],
  };
};
